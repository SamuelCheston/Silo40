package service

import (
	"encoding/json"
	"fmt"
	"sync"

	"silo40/internal/cache"
	"silo40/internal/engine"
	"silo40/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ============ 有状态游戏会话服务 ============
// 后端是唯一事实来源：
// - 游戏状态驻留内存 (GameEngine + Silo + Agent)
// - FreeCache 缓存会话快照，应用重启可恢复
// - 关键节点 (新建游戏 / 游戏结束) 落库 SQLite
// 当前为单活跃游戏设计 (固定 Silo.ID=1 / Agent.ID=1)，与现有单窗口 UI 匹配。

const (
	ActiveSiloID  = uint(1)
	ActiveAgentID = uint(1)
	sessionKey    = "game_session_v1"
)

type GameService struct {
	db     *gorm.DB
	mu     sync.Mutex
	engine *engine.GameEngine
	silo   *model.Silo
	agent  *model.Agent
}

func NewGameService(db *gorm.DB) *GameService {
	return &GameService{
		db:     db,
		engine: engine.NewGameEngine(),
	}
}

// ---------- 内部辅助 ----------

// sessionState 缓存快照结构
type sessionState struct {
	Silo  model.Silo  `json:"silo"`
	Agent model.Agent `json:"agent"`
}

func (s *GameService) hasSession() bool {
	return s.silo != nil && s.agent != nil
}

func (s *GameService) organizedPopulation() int {
	if !s.hasSession() {
		return 0
	}
	return s.engine.GetOrganizedPopulation(s.silo, s.agent)
}

func (s *GameService) gameOver() bool {
	return s.hasSession() && s.silo.VictoryStatus != nil
}

func (s *GameService) cacheSnapshot() {
	if cache.GlobalCache == nil || !s.hasSession() {
		return
	}
	data, err := json.Marshal(sessionState{Silo: *s.silo, Agent: *s.agent})
	if err != nil {
		return
	}
	_ = cache.GlobalCache.Set([]byte(sessionKey), data, 3600)
}

// persist 关键节点落库：整图替换式保存 (单活跃游戏，固定 ID，避免 PK 冲突)
func (s *GameService) persist() error {
	if !s.hasSession() {
		return fmt.Errorf("no active session")
	}
	silo := s.silo
	agent := s.agent

	// 固定主键
	silo.ID = ActiveSiloID
	agent.ID = ActiveAgentID
	agent.UserID = 0
	for i := range silo.Professions {
		silo.Professions[i].SiloID = ActiveSiloID
		if silo.Professions[i].ID == 0 {
			silo.Professions[i].ID = uint(i + 1)
		}
	}
	for i := range silo.Resources {
		silo.Resources[i].SiloID = ActiveSiloID
	}
	for i := range silo.Floors {
		silo.Floors[i].SiloID = ActiveSiloID
	}
	for i := range silo.Factions {
		silo.Factions[i].SiloID = ActiveSiloID
		if silo.Factions[i].ID == 0 {
			silo.Factions[i].ID = uint(i + 1)
		}
	}
	for i := range silo.Residents {
		silo.Residents[i].SiloID = ActiveSiloID
		if silo.Residents[i].ID == 0 {
			silo.Residents[i].ID = uint(i + 1)
		}
	}
	for i := range agent.Connections {
		agent.Connections[i].AgentID = ActiveAgentID
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// 清除旧图 (含关联行)
		var oldConnections []model.Connection
		_ = tx.Where("agent_id = ?", ActiveAgentID).Find(&oldConnections).Error
		for _, c := range oldConnections {
			if err := tx.Delete(&model.Connection{}, c.ID).Error; err != nil {
				return err
			}
		}

		var oldFloors []model.Floor
		_ = tx.Where("silo_id = ?", ActiveSiloID).Find(&oldFloors).Error
		for _, f := range oldFloors {
			if err := tx.Delete(&model.Floor{}, f.ID).Error; err != nil {
				return err
			}
		}

		var oldResidents []model.Resident
		_ = tx.Where("silo_id = ?", ActiveSiloID).Find(&oldResidents).Error
		for _, resident := range oldResidents {
			if err := tx.Delete(&model.Resident{}, resident.ID).Error; err != nil {
				return err
			}
		}

		var oldFactions []model.Faction
		_ = tx.Where("silo_id = ?", ActiveSiloID).Find(&oldFactions).Error
		for _, faction := range oldFactions {
			if err := tx.Delete(&model.Faction{}, faction.ID).Error; err != nil {
				return err
			}
		}

		var oldProfessions []model.Profession
		_ = tx.Where("silo_id = ?", ActiveSiloID).Find(&oldProfessions).Error
		for _, p := range oldProfessions {
			if err := tx.Delete(&model.Profession{}, p.ID).Error; err != nil {
				return err
			}
		}

		var oldResources []model.Resource
		_ = tx.Where("silo_id = ?", ActiveSiloID).Find(&oldResources).Error
		for _, r := range oldResources {
			if err := tx.Delete(&model.Resource{}, r.ID).Error; err != nil {
				return err
			}
		}

		// Silo/Agent 含 gorm.DeletedAt，普通 Delete 为软删除，旧行仍占用固定主键；
		// 整图替换式保存需物理删除 (Unscoped) 才能用同 ID 重建。
		if err := tx.Unscoped().Delete(&model.Agent{}, ActiveAgentID).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Delete(&model.Silo{}, ActiveSiloID).Error; err != nil {
			return err
		}

		// 落库新图 (GORM 会填充关联外键)
		if err := tx.Omit(clause.Associations).Create(silo).Error; err != nil {
			return err
		}
		if err := tx.Omit(clause.Associations).Create(agent).Error; err != nil {
			return err
		}
		for i := range silo.Professions {
			silo.Professions[i].SiloID = ActiveSiloID
		}
		for i := range silo.Resources {
			silo.Resources[i].SiloID = ActiveSiloID
		}
		for i := range silo.Floors {
			silo.Floors[i].SiloID = ActiveSiloID
		}
		for i := range silo.Factions {
			silo.Factions[i].SiloID = ActiveSiloID
		}
		for i := range silo.Residents {
			silo.Residents[i].SiloID = ActiveSiloID
		}
		if err := tx.Create(&silo.Resources).Error; err != nil {
			return err
		}
		if err := tx.Create(&silo.Professions).Error; err != nil {
			return err
		}
		if err := tx.Create(&silo.Floors).Error; err != nil {
			return err
		}
		if err := tx.Create(&silo.Factions).Error; err != nil {
			return err
		}
		if err := tx.Create(&silo.Residents).Error; err != nil {
			return err
		}
		if len(agent.Connections) > 0 {
			agent.Connections[0].AgentID = ActiveAgentID
			return tx.Create(&agent.Connections).Error
		}
		return nil
	})
}

// ---------- 对外 API ----------

// Resume 应用启动时恢复会话：优先 FreeCache 快照，其次 SQLite
func (s *GameService) Resume() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cache.GlobalCache != nil {
		if val, err := cache.GlobalCache.Get([]byte(sessionKey)); err == nil {
			var snap sessionState
			if json.Unmarshal(val, &snap) == nil {
				s.silo = &snap.Silo
				s.agent = &snap.Agent
				s.engine = engine.NewGameEngine()
				return nil
			}
		}
	}

	var silo model.Silo
	if err := s.db.Preload("Resources").Preload("Professions").Preload("Floors").Preload("Residents").Preload("Factions").
		First(&silo, ActiveSiloID).Error; err != nil {
		return err
	}
	var agent model.Agent
	if err := s.db.Preload("Connections").First(&agent, ActiveAgentID).Error; err != nil {
		return err
	}
	s.silo = &silo
	s.agent = &agent
	s.engine = engine.NewGameEngine()
	return nil
}

// CreateGame 新建游戏：初始化 → 落库 → 缓存
func (s *GameService) CreateGame(req model.CreateGameRequest) (*model.GameState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	startYear := req.StartYear
	if startYear == 0 {
		startYear = 122
	}
	silo := engine.CreateInitialSilo(req.SiloName, startYear, req.TraitIds)
	agent := engine.CreateInitialAgent(req.AgentName, req.Profession, req.TraitIds, silo)

	// 初始化引擎 (清空旧调度器状态)
	s.engine = engine.NewGameEngine()
	s.silo = silo
	s.agent = agent

	if err := s.persist(); err != nil {
		return nil, fmt.Errorf("failed to persist new game: %w", err)
	}
	s.cacheSnapshot()

	return s.buildState(), nil
}

// GetState 当前游戏状态快照
func (s *GameService) GetState() (*model.GameState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSession() {
		return nil, fmt.Errorf("no active game session")
	}
	return s.buildState(), nil
}

// PassTime 推进 months 个月，返回 tick 结果
func (s *GameService) PassTime(months int) (*model.TickResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSession() {
		return nil, fmt.Errorf("no active game session")
	}

	var logs []string
	var stories []model.StoryEvent

	for i := 0; i < months; i++ {
		l, st := s.engine.UpdateSiloState(s.silo, 1.0/12.0, s.agent)
		logs = append(logs, l...)
		for _, story := range st {
			stories = append(stories, *story)
		}

		if s.silo.CurrentMonth == 12 {
			s.silo.CurrentMonth = 1
			s.silo.CurrentYear++
		} else {
			s.silo.CurrentMonth++
		}
	}

	// 关键节点：游戏结束 → 落库
	if s.gameOver() {
		if err := s.persist(); err != nil {
			return nil, err
		}
	}
	s.cacheSnapshot()

	return &model.TickResult{
		Silo:                s.publicSiloSnapshot(),
		Agent:               *s.agent,
		Logs:                logs,
		Stories:             stories,
		OrganizedPopulation: s.organizedPopulation(),
		GameOver:            s.gameOver(),
		EndingNarrative:     s.endingNarrative(),
	}, nil
}

// ExecuteAction 执行玩家动作；时长>0 的动作自动推进时间
func (s *GameService) ExecuteAction(action model.AgentAction) (*model.ActionOutcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSession() {
		return nil, fmt.Errorf("no active game session")
	}

	result := s.engine.ExecuteAgentAction(s.silo, s.agent, action)

	var logs []string
	var stories []model.StoryEvent

	if result.Executed {
		duration := model.ACTION_DURATIONS[action.Type]
		if duration > 0 {
			for i := 0; i < duration; i++ {
				l, st := s.engine.UpdateSiloState(s.silo, 1.0/12.0, s.agent)
				logs = append(logs, l...)
				for _, story := range st {
					stories = append(stories, *story)
				}
				if s.silo.CurrentMonth == 12 {
					s.silo.CurrentMonth = 1
					s.silo.CurrentYear++
				} else {
					s.silo.CurrentMonth++
				}
			}
		} else {
			var npcLogs []string
			s.engine.RunNpcTurn(s.silo, s.agent, 0, func(msg string) {
				npcLogs = append(npcLogs, msg)
			})
			l, st := s.engine.UpdateSiloState(s.silo, 0, s.agent)
			logs = append(npcLogs, l...)
			for _, story := range st {
				stories = append(stories, *story)
			}
		}
	}

	if s.gameOver() {
		if err := s.persist(); err != nil {
			return nil, err
		}
	}
	s.cacheSnapshot()

	return &model.ActionOutcome{
		Silo:                s.publicSiloSnapshot(),
		Agent:               *s.agent,
		Result:              result,
		Logs:                logs,
		Stories:             stories,
		OrganizedPopulation: s.organizedPopulation(),
		GameOver:            s.gameOver(),
		EndingNarrative:     s.endingNarrative(),
	}, nil
}

// GetEndingNarrative 结局叙事文案
func (s *GameService) GetEndingNarrative() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.endingNarrative()
}

// HasActiveGame 是否有进行中的游戏
func (s *GameService) HasActiveGame() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hasSession()
}

// ---------- 私有 ----------

func (s *GameService) endingNarrative() string {
	if !s.hasSession() {
		return ""
	}
	return s.engine.GetEndingNarrative(s.silo)
}

func (s *GameService) buildState() *model.GameState {
	gameOver := s.gameOver()
	return &model.GameState{
		Silo:                s.publicSiloSnapshot(),
		Agent:               *s.agent,
		OrganizedPopulation: s.organizedPopulation(),
		GameOver:            gameOver,
		EndingNarrative:     s.endingNarrative(),
		VictoryStatus:       s.silo.VictoryStatus,
		ProfessionActions:   engine.GetProfessionActionMeta(s.agent.Profession),
	}
}

func (s *GameService) publicSiloSnapshot() model.Silo {
	if s.silo == nil {
		return model.Silo{}
	}
	snap := *s.silo
	// Residents stay in the backend/session snapshot for simulation, but we avoid
	// returning the full population on every UI round-trip.
	snap.Residents = nil
	return snap
}
