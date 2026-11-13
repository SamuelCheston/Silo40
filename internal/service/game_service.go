package service

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"silo40/internal/cache"
	"silo40/internal/engine"
	"silo40/internal/llm"
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
	ActiveWorldID     = uint(1)
	ActiveSiloID      = uint(1)
	ActiveAgentID     = uint(1)
	sessionKey        = "game_session_v1"
	saveMetaKey       = "save"
	timestampBaseYear = 100
	daysPerMonth      = 30
	monthsPerYear     = 12
)

type GameService struct {
	db                  *gorm.DB
	mu                  sync.Mutex
	engine              *engine.GameEngine
	world               *model.World
	silo                *model.Silo  // 玩家堡 (= silos 中 PlayerSiloID 对应者)
	agent               *model.Agent // 玩家特工 (= agents[玩家堡])
	silos               []*model.Silo
	agents              map[uint]*model.Agent // siloID -> 该堡的操作者特工 (玩家堡 + AI 代理堡)
	eventLogs           []model.StoryEventLog
	contentDefinitions  []engine.ContentEventDefinition
	contentStates       map[uint]map[string]model.ContentEventState // siloID -> 事件触发状态
	contentUnsubscribes []func()
	actionGates         actionGateRuntime
	delayCounter        int
	delayTasks          map[string]*delayTask
	pendingOperation    *pendingOperation

	// currentSilo/currentAgent 世界 tick 的逐堡结算上下文：指向正在被推进的堡。
	// nil 时所有模拟路径回退到玩家堡 (单堡语义逐位不变)。
	currentSilo  *model.Silo
	currentAgent *model.Agent

	// receivedSignals[siloID] 本月该堡收到的跨堡信号 (signal_received 触发器输入)；
	// 每次世界 tick 结束时清空。monthPlayerLogs 本月记入玩家堡的事件日志
	// (LLM gating 输入，世界 tick 下与全局 eventLogs 分流)。
	receivedSignals map[uint][]string
	monthPlayerLogs []model.StoryEventLog

	// masterRng 非 nil 时表示设置了 SILO40_SEED：每局新游戏从它派生确定性种子。
	masterRng *rand.Rand

	// llmAdvisor 非 nil 时表示启用了 LLM 辅助 (SILO40_LLM_MODE=shadow|on)。
	// 决策日志与参与度切换记录在 llm_decision_logs 表 (append-only)。
	llmAdvisor   *llm.Advisor
	llmLogCursor int

	// agentSession 非 nil 时表示有外部 AI agent 正在以操作员模式玩当前这局
	// (agent_game.go)：月度 tick 在每个 NPC 决策点暂停，等待 agent 经
	// debug HTTP 提交决策；决策记录按数据管线契约收集并经 /debug/agent/export
	// 导出。会话活跃期间 engine.NpcPolicy 指向该会话而非 llmAdvisor。
	// 由 agentMu 保护：绝不能用 s.mu，runner 在 PassTime 内持有 s.mu
	// 等待决策时会与 HTTP 侧死锁。
	agentMu      sync.Mutex
	agentSession *AgentSession
}

type operationPhase string

const (
	phaseActionRoot operationPhase = "action_root"
	phaseQueued     operationPhase = "queued"
)

type pendingOperation struct {
	Kind            string
	Phase           operationPhase
	RemainingMonths int
	Action          *model.AgentAction
	ActionDuration  int
	ActionResult    *model.ActionResult
}

type delayTask struct {
	ID           string
	StartEventID string
	EndEventID   string
	DelayMonths  int
	Status       string
	// TargetSiloID 延迟事件归属的堡 (P0.5)。世界 tick 在目标堡的本地阶段
	// 开始前集中发布到期任务；单堡恒为玩家堡，行为不变。
	TargetSiloID uint
	Event        *engine.GameEvent
}

func NewGameService(db *gorm.DB) *GameService {
	svc := &GameService{
		db:            db,
		engine:        engine.NewGameEngine(),
		contentStates: map[uint]map[string]model.ContentEventState{},
		delayTasks:    map[string]*delayTask{},
	}
	if seedStr := os.Getenv("SILO40_SEED"); seedStr != "" {
		if seed, err := strconv.ParseInt(seedStr, 10, 64); err == nil {
			svc.masterRng = rand.New(rand.NewSource(seed))
			log.Printf("SILO40_SEED=%d: new games derive deterministic per-game seeds from this master seed", seed)
		} else {
			log.Printf("ignoring invalid SILO40_SEED %q: %v", seedStr, err)
		}
	}
	svc.setupLlmAdvisor()
	svc.bindEngineRuntime()
	return svc
}

// setupLlmAdvisor builds the optional LLM advisor from SILO40_LLM_* env
// vars. Without a key/endpoint the deterministic fake client is used so the
// shadow chain stays verifiable offline; parse errors fall back to off with
// an explicit log line (never a silent half-enabled state).
func (s *GameService) setupLlmAdvisor() {
	cfg := llm.LoadEnv()
	if cfg.ModeParseErr != nil {
		log.Printf("LLM disabled: %v", cfg.ModeParseErr)
		return
	}
	if cfg.LevelParseErr != nil {
		log.Printf("LLM participation defaulted to off: %v", cfg.LevelParseErr)
	}
	if cfg.Mode == llm.ModeOff {
		return
	}
	var client llm.Client
	if cfg.APIKey != "" || cfg.BaseURL != "" {
		client = llm.NewHTTPClient(cfg.BaseURL, cfg.APIKey, cfg.Model, cfg.ReasoningEffort)
		log.Printf("LLM advisor: mode=%s level=%s endpoint=%s model=%s reasoning_effort=%q",
			cfg.Mode, cfg.Level, cfg.BaseURL, cfg.Model, cfg.ReasoningEffort)
	} else {
		client = &llm.FakeClient{Name: "offline-default"}
		log.Printf("LLM advisor: mode=%s level=%s using deterministic fake client "+
			"(no SILO40_LLM_API_KEY / SILO40_LLM_BASE_URL set)", cfg.Mode, cfg.Level)
	}
	advisor, err := llm.NewAdvisor(llm.Config{
		DB:          s.db,
		Client:      client,
		Mode:        cfg.Mode,
		Level:       cfg.Level,
		EventWindow: cfg.EventWindow,
		TimeoutMS:   cfg.TimeoutMS,
		Prefetch:    cfg.Prefetch,
		Warmup:      cfg.Warmup,
	})
	if err != nil {
		log.Printf("LLM advisor disabled: %v", err)
		return
	}
	s.llmAdvisor = advisor
}

// llmError surfaces the advisor's sticky fatal error (replay divergence /
// log failure). Checked after every simulated month so contract violations
// fail the whole operation instead of degrading silently.
func (s *GameService) llmError() error {
	if s.llmAdvisor == nil {
		return nil
	}
	return s.llmAdvisor.Error()
}

// SetLlmParticipation switches the participation level mid-game (design
// §11): the switch is appended to the decision log so replays restore it.
func (s *GameService) SetLlmParticipation(level string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.llmAdvisor == nil {
		return "", fmt.Errorf("LLM is disabled (set SILO40_LLM_MODE=shadow|on)")
	}
	parsed, err := llm.ParseLevel(level)
	if err != nil {
		return "", err
	}
	if err := s.llmAdvisor.SetLevel(parsed, "api"); err != nil {
		return "", err
	}
	return string(parsed), nil
}

// LlmStatus is the author-visible LLM configuration snapshot.
type LlmStatus struct {
	Enabled    bool   `json:"enabled"`
	Mode       string `json:"mode"`
	Level      string `json:"level"`
	Model      string `json:"model,omitempty"`
	LogEntries int64  `json:"log_entries"`
	AdvisorErr string `json:"advisor_error,omitempty"`
}

// GetLlmStatus reports mode/level/log size for the debug endpoint.
func (s *GameService) GetLlmStatus() LlmStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.llmAdvisor == nil {
		return LlmStatus{Enabled: false, Mode: "off"}
	}
	st := LlmStatus{
		Enabled: true,
		Mode:    string(s.llmAdvisor.GetMode()),
		Level:   string(s.llmAdvisor.GetLevel()),
	}
	if entries, err := llm.LoadEntries(s.db); err == nil {
		st.LogEntries = int64(len(entries))
		if len(entries) > 0 {
			st.Model = entries[len(entries)-1].Model
		}
	}
	if err := s.llmAdvisor.Error(); err != nil {
		st.AdvisorErr = err.Error()
	}
	return st
}

// GetLlmDecisions returns recent decision/config log entries (newest first,
// limit <= 0 = all) for the "why did this NPC choose that" audit view.
func (s *GameService) GetLlmDecisions(limit int) ([]llm.DecisionLog, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.llmAdvisor == nil {
		return nil, fmt.Errorf("LLM is disabled (set SILO40_LLM_MODE=shadow|on)")
	}
	entries, err := llm.LoadEntries(s.db)
	if err != nil {
		return nil, err
	}
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}
	return entries, nil
}

// sessionSeed 返回存档上记录的 RNG 种子；没有种子的旧存档回退为新随机种子
// （保持引入种子机制之前的默认随机行为）。
func sessionSeed(silo *model.Silo) int64 {
	if silo != nil && silo.RNGSeed != 0 {
		return silo.RNGSeed
	}
	return rand.Int63()
}

// BootstrapContent 同步文件驱动事件定义到数据库并加载到内存。
func (s *GameService) BootstrapContent(events embed.FS) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 运行的时候检查同目录有没有 events 目录，没有则释放
	if err := s.ensureEventsExtracted(events); err != nil {
		return fmt.Errorf("failed to extract embedded events: %w", err)
	}

	dirs, err := resolveContentDirectories()
	if err != nil {
		return err
	}
	if err := s.syncContentDefinitions(dirs); err != nil {
		return err
	}
	if err := s.loadActionGates(dirs); err != nil {
		return err
	}
	if err := s.loadContentDefinitions(); err != nil {
		return err
	}
	s.bindContentEventHandlers()
	s.bindLlmEventFilter()
	return nil
}

// BootstrapContentFromDisk loads content definitions directly from the
// events/ tree on disk instead of extracting the embedded copy. Headless
// tools that run inside a source checkout (cmd/agenthost) use this; the
// production GUI binary keeps the fail-closed embedded path
// (BootstrapContent). The action-gate registry is loaded from disk with
// the same fail-closed guarantee (missing file = explicit error).
func (s *GameService) BootstrapContentFromDisk() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dirs, err := resolveContentDirectories()
	if err != nil {
		return err
	}
	if err := s.syncContentDefinitions(dirs); err != nil {
		return err
	}
	if err := s.loadActionGates(dirs); err != nil {
		return err
	}
	if err := s.loadContentDefinitions(); err != nil {
		return err
	}
	s.bindContentEventHandlers()
	s.bindLlmEventFilter()
	return nil
}

func (s *GameService) ensureEventsExtracted(events embed.FS) error {
	exePath, _ := os.Executable()
	if exePath == "" {
		return nil // Fallback to embedded if executable path is unknown? No, user wants extraction.
	}
	exeDir := filepath.Dir(exePath)
	targetDir := filepath.Join(exeDir, "events")

	// Existing packages may predate a required content file. Sync the gate
	// registry even when the rest of the events directory is already present.
	if _, err := os.Stat(targetDir); err == nil {
		return syncActionGateConfig(events, targetDir)
	}

	// 没有则释放
	if err := fs.WalkDir(events, "events", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel("events", path)
		if relPath == "." {
			return nil
		}

		targetPath := filepath.Join(targetDir, relPath)
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}

		// Copy file
		data, err := events.ReadFile(path)
		if err != nil {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}

		return os.WriteFile(targetPath, data, 0o644)
	}); err != nil {
		return err
	}
	return syncActionGateConfig(events, targetDir)
}

// syncActionGateConfig keeps the executable-local registry aligned with the
// embedded build artifact, preventing stale packages from silently opening every action.
func syncActionGateConfig(events fs.FS, targetDir string) error {
	data, err := fs.ReadFile(events, "events/action_gates.json")
	if err != nil {
		return fmt.Errorf("required embedded action gate config is missing: %w", err)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(targetDir, "action_gates.json"), data, 0o644)
}

// ---------- 内部辅助 ----------

// sessionState 缓存快照结构 (P0.5 世界化：全部堡 + 每堡操作者特工)。
// Silo/Agent 保留为玩家堡别名，供旧快照字段兼容读取。
type sessionState struct {
	World         *model.World              `json:"world,omitempty"`
	Silo          model.Silo                `json:"silo"`
	Agent         model.Agent               `json:"agent"`
	Silos         []model.Silo              `json:"silos,omitempty"`
	Agents        []model.Agent             `json:"agents,omitempty"`
	EventLogs     []model.StoryEventLog     `json:"event_logs"`
	ContentStates []model.ContentEventState `json:"content_states"`
	SaveVersion   int                       `json:"save_version"`
}

// ErrIncompatibleSave 存档版本不兼容：明确报错，拒绝静默加载。
var ErrIncompatibleSave = fmt.Errorf("save file is incompatible with this version")

func (s *GameService) hasSession() bool {
	return s.silo != nil && s.agent != nil
}

func (s *GameService) gameOver() bool {
	return s.hasSession() && s.silo.VictoryStatus != nil
}

// multiSilo 世界内是否有两个及以上完整模拟堡。单堡 (空邻堡) 配置下所有
// 多堡机制 (跨堡阶段/跨堡队列/Silo 1 裁决/AI 代操作/迷雾) 全部关闭，
// 行为与历史逐位一致——这是 P0.5 的奇偶校验锚点。
func (s *GameService) multiSilo() bool {
	return s.world != nil && len(s.silos) >= 2
}

// simSilo 当前模拟上下文指向的堡；世界 tick 外 (玩家交互路径) 回退到玩家堡。
func (s *GameService) simSilo() *model.Silo {
	if s.currentSilo != nil {
		return s.currentSilo
	}
	return s.playerActionSilo()
}

// simAgent 当前模拟上下文堡的操作者特工 (玩家堡 = 玩家特工；邻堡 = AI 代理)。
func (s *GameService) simAgent() *model.Agent {
	if s.currentAgent != nil {
		return s.currentAgent
	}
	return s.agent
}

// simSiloID 当前模拟上下文堡的 ID；无会话时返回玩家堡固定 ID。
func (s *GameService) simSiloID() uint {
	if silo := s.simSilo(); silo != nil && silo.ID != 0 {
		return silo.ID
	}
	return ActiveSiloID
}

// siloEnded 堡是否已结算出本地结局 (DEATH/AGENT_COMPROMISED/…)；终局堡
// 不再参与本地结算与内容事件评估，但仍作为世界事实存在 (迷雾显示覆灭)。
func siloEnded(silo *model.Silo) bool {
	return silo == nil || silo.VictoryStatus != nil
}

// siloByID 世界内按 ID 查堡 (确定性：调用方保证升序遍历)。
func (s *GameService) siloByID(id uint) *model.Silo {
	for _, silo := range s.silos {
		if silo != nil && silo.ID == id {
			return silo
		}
	}
	return nil
}

// statesMapFor 返回指定堡的内容状态子表 (惰性创建)。
func (s *GameService) statesMapFor(siloID uint) map[string]model.ContentEventState {
	states := s.contentStates[siloID]
	if states == nil {
		states = map[string]model.ContentEventState{}
		s.contentStates[siloID] = states
	}
	return states
}

func (s *GameService) cacheSnapshot() {
	if cache.GlobalCache == nil || !s.hasSession() {
		return
	}
	contentStates := s.contentStateSlice()
	silos := make([]model.Silo, 0, len(s.silos))
	for _, silo := range s.silos {
		if silo != nil {
			silos = append(silos, *silo)
		}
	}
	agents := make([]model.Agent, 0, len(s.agents))
	for _, siloID := range sortedSiloIDs(s.silos) {
		if agent, ok := s.agents[siloID]; ok && agent != nil {
			agents = append(agents, *agent)
		}
	}
	data, err := json.Marshal(sessionState{
		World:         s.world,
		Silo:          *s.silo,
		Agent:         *s.agent,
		Silos:         silos,
		Agents:        agents,
		EventLogs:     s.eventLogs,
		ContentStates: contentStates,
		SaveVersion:   model.CurrentSaveVersion,
	})
	if err != nil {
		return
	}
	_ = cache.GlobalCache.Set([]byte(sessionKey), data, 3600)
}

// sortedSiloIDs 返回堡指针切片的升序 ID 列表 (确定性遍历)。
func sortedSiloIDs(silos []*model.Silo) []uint {
	ids := make([]uint, 0, len(silos))
	for _, silo := range silos {
		if silo != nil {
			ids = append(ids, silo.ID)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// siloIDBaseStride 多堡落库时子表主键的堡间偏移步长。引擎在内存中按堡内
// 1 起始分配子表 (professions/cohorts/factions/residents) ID，两堡会在同一张
// 表主键冲突；落库时对第 siloID 堡的全部受影响子表 ID 与堡内外键统一加
// base = (siloID-1)*stride，读档时减去同一 base 还原。玩家堡 (siloID=1)
// base = 0，单堡存档逐位不变。resources/floors 无外键引用且 ID 由数据库自增
// 分配，无需偏移。
const siloIDBaseStride = 10000

func siloIDBase(siloID uint) uint {
	if siloID <= 1 {
		return 0
	}
	return (siloID - 1) * siloIDBaseStride
}

// persist 关键节点落库：整图替换式保存 (单活跃游戏；P0.5 起支持世界内多堡，
// 子表主键按堡偏移避免冲突)。
func (s *GameService) persist() error {
	if !s.hasSession() {
		return fmt.Errorf("no active session")
	}
	s.refreshActionProgress()
	s.applyHistoryInfluenceScore()
	world := s.world
	eventLogs := make([]model.StoryEventLog, len(s.eventLogs))
	copy(eventLogs, s.eventLogs)
	contentStates := s.contentStateSlice()
	if world != nil {
		world.ID = ActiveWorldID
	}
	for i := range eventLogs {
		eventLogs[i].ID = 0
		if eventLogs[i].SiloID == 0 {
			eventLogs[i].SiloID = ActiveSiloID
		}
	}
	for i := range contentStates {
		contentStates[i].ID = 0
		if contentStates[i].SiloID == 0 {
			contentStates[i].SiloID = ActiveSiloID
		}
	}

	// 固定堡/特工主键并归一化子表 (内存保持堡内 1 起始，落库时再加偏移)。
	silos := make([]*model.Silo, 0, len(s.silos))
	agents := make([]*model.Agent, 0, len(s.silos))
	for idx, silo := range s.silos {
		if silo == nil {
			continue
		}
		silo.ID = uint(idx + 1)
		silo.WorldID = ActiveWorldID
		normalizeSiloSubIDs(silo)
		silos = append(silos, silo)
		if agent := s.agents[silo.ID]; agent != nil {
			agent.ID = silo.ID
			agent.UserID = 0
			agent.SiloID = silo.ID
			agents = append(agents, agent)
		}
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		siloIDs := make([]uint, 0, len(silos))
		for _, silo := range silos {
			siloIDs = append(siloIDs, silo.ID)
		}
		if err := deleteWorldGraph(tx, siloIDs); err != nil {
			return err
		}

		if world != nil {
			if err := tx.Create(world).Error; err != nil {
				return err
			}
		}
		for _, silo := range silos {
			if err := tx.Omit(clause.Associations).Create(silo).Error; err != nil {
				return err
			}
			if err := createSiloSubRows(tx, silo, siloIDBase(silo.ID)); err != nil {
				return err
			}
		}
		for _, agent := range agents {
			if err := tx.Omit(clause.Associations).Create(agent).Error; err != nil {
				return err
			}
			if len(agent.Connections) > 0 {
				conns := offsetConnections(agent.Connections, agent.ID, siloIDBase(agent.SiloID))
				if err := tx.Create(&conns).Error; err != nil {
					return err
				}
			}
		}
		if len(eventLogs) > 0 {
			if err := tx.Create(&eventLogs).Error; err != nil {
				return err
			}
		}
		if len(contentStates) > 0 {
			if err := tx.Create(&contentStates).Error; err != nil {
				return err
			}
		}
		// 版本标记与存档同事务写入，保证"有存档必有版本"。
		return tx.Save(&model.SaveMeta{Key: saveMetaKey, SaveVersion: model.CurrentSaveVersion}).Error
	})
}

// normalizeSiloSubIDs 归一化堡子表外键与缺省主键 (内存中保持堡内 1 起始)。
func normalizeSiloSubIDs(silo *model.Silo) {
	siloID := silo.ID
	for i := range silo.Professions {
		silo.Professions[i].SiloID = siloID
		if silo.Professions[i].ID == 0 {
			silo.Professions[i].ID = uint(i + 1)
		}
	}
	for i := range silo.Resources {
		silo.Resources[i].SiloID = siloID
	}
	for i := range silo.Floors {
		silo.Floors[i].SiloID = siloID
	}
	for i := range silo.Cohorts {
		silo.Cohorts[i].SiloID = siloID
		if silo.Cohorts[i].ID == 0 {
			silo.Cohorts[i].ID = uint(i + 1)
		}
	}
	for i := range silo.Factions {
		silo.Factions[i].SiloID = siloID
		if silo.Factions[i].ID == 0 {
			silo.Factions[i].ID = uint(i + 1)
		}
	}
	for i := range silo.Residents {
		silo.Residents[i].SiloID = siloID
		if silo.Residents[i].ID == 0 {
			silo.Residents[i].ID = uint(i + 1)
		}
	}
}

// deleteWorldGraph 物理清除本世界全部堡及其子表、全部操作者特工与人脉。
func deleteWorldGraph(tx *gorm.DB, siloIDs []uint) error {
	for _, table := range []interface{}{
		&model.Floor{}, &model.PopulationCohort{}, &model.Resident{}, &model.Faction{},
		&model.Profession{}, &model.Resource{}, &model.StoryEventLog{}, &model.ContentEventState{},
	} {
		if err := tx.Where("silo_id IN ?", siloIDs).Delete(table).Error; err != nil {
			return err
		}
	}
	var agentIDs []uint
	if err := tx.Model(&model.Agent{}).Unscoped().Where("silo_id IN ?", siloIDs).Pluck("id", &agentIDs).Error; err != nil {
		return err
	}
	if len(agentIDs) > 0 {
		if err := tx.Where("agent_id IN ?", agentIDs).Delete(&model.Connection{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("id IN ?", agentIDs).Delete(&model.Agent{}).Error; err != nil {
			return err
		}
	}
	// Silo 含软删除字段，需 Unscoped 物理删除以复用固定主键。
	if err := tx.Unscoped().Where("id IN ?", siloIDs).Delete(&model.Silo{}).Error; err != nil {
		return err
	}
	// World 无软删除字段，普通 Delete 即可。
	return tx.Where("id = ?", ActiveWorldID).Delete(&model.World{}).Error
}

// createSiloSubRows 落库单个堡的子表 (受主键冲突影响的表加 base 偏移)。
func createSiloSubRows(tx *gorm.DB, silo *model.Silo, base uint) error {
	resources := make([]model.Resource, len(silo.Resources))
	for i, r := range silo.Resources {
		r.SiloID = silo.ID
		resources[i] = r
	}
	if err := tx.Create(&resources).Error; err != nil {
		return err
	}
	floors := make([]model.Floor, len(silo.Floors))
	for i, f := range silo.Floors {
		f.SiloID = silo.ID
		floors[i] = f
	}
	if err := tx.Create(&floors).Error; err != nil {
		return err
	}
	professions := offsetProfessions(silo.Professions, silo.ID, base)
	if err := tx.Create(&professions).Error; err != nil {
		return err
	}
	cohorts := offsetCohorts(silo.Cohorts, silo.ID, base)
	if err := tx.Create(&cohorts).Error; err != nil {
		return err
	}
	factions := offsetFactions(silo.Factions, silo.ID, base)
	if err := tx.Create(&factions).Error; err != nil {
		return err
	}
	residents := offsetResidents(silo.Residents, silo.ID, base)
	if err := tx.Create(&residents).Error; err != nil {
		return err
	}
	return nil
}

// 以下 offset* 函数生成子表的落库副本：主键与堡内外键统一加 base。对指针外键
// 分配新值，绝不改写内存中共享的底层数据 (内存维持堡内 1 起始)。
func offsetProfessions(in []model.Profession, siloID, base uint) []model.Profession {
	out := make([]model.Profession, len(in))
	for i, p := range in {
		p.ID += base
		p.SiloID = siloID
		out[i] = p
	}
	return out
}

func offsetCohorts(in []model.PopulationCohort, siloID, base uint) []model.PopulationCohort {
	out := make([]model.PopulationCohort, len(in))
	for i, c := range in {
		c.ID += base
		c.SiloID = siloID
		if c.ProfessionID != 0 {
			c.ProfessionID += base
		}
		if c.FactionID != nil && *c.FactionID != 0 {
			fid := *c.FactionID + base
			c.FactionID = &fid
		}
		out[i] = c
	}
	return out
}

func offsetFactions(in []model.Faction, siloID, base uint) []model.Faction {
	out := make([]model.Faction, len(in))
	for i, f := range in {
		f.ID += base
		f.SiloID = siloID
		if f.RepresentativeResidentID != 0 {
			f.RepresentativeResidentID += base
		}
		if f.RepresentativeCohortID != nil && *f.RepresentativeCohortID != 0 {
			cid := *f.RepresentativeCohortID + base
			f.RepresentativeCohortID = &cid
		}
		out[i] = f
	}
	return out
}

func offsetResidents(in []model.Resident, siloID, base uint) []model.Resident {
	out := make([]model.Resident, len(in))
	for i, r := range in {
		r.ID += base
		r.SiloID = siloID
		if r.ProfessionID != 0 {
			r.ProfessionID += base
		}
		if r.CohortID != nil && *r.CohortID != 0 {
			cid := *r.CohortID + base
			r.CohortID = &cid
		}
		if r.FactionID != nil && *r.FactionID != 0 {
			fid := *r.FactionID + base
			r.FactionID = &fid
		}
		out[i] = r
	}
	return out
}

func offsetConnections(in []model.Connection, agentID, base uint) []model.Connection {
	out := make([]model.Connection, len(in))
	for i, c := range in {
		c.ID = 0
		c.AgentID = agentID
		if c.ProfessionID != 0 {
			c.ProfessionID += base
		}
		out[i] = c
	}
	return out
}

// ---------- 对外 API ----------

// Resume 应用启动时恢复会话：优先 FreeCache 快照，其次 SQLite
func (s *GameService) Resume() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cache.GlobalCache != nil {
		if val, err := cache.GlobalCache.Get([]byte(sessionKey)); err == nil {
			var snap sessionState
			if json.Unmarshal(val, &snap) == nil && snap.SaveVersion == model.CurrentSaveVersion {
				s.world = snap.World
				s.silo = &snap.Silo
				s.agent = &snap.Agent
				s.eventLogs = snap.EventLogs
				s.contentStates = map[uint]map[string]model.ContentEventState{}
				for _, state := range snap.ContentStates {
					s.statesMapFor(state.SiloID)[state.EventKey] = state
				}
				// P0.5 世界化快照：全部堡 + 每堡操作者特工。
				if len(snap.Silos) > 0 {
					s.silos = make([]*model.Silo, len(snap.Silos))
					for i := range snap.Silos {
						s.silos[i] = &snap.Silos[i]
					}
					// 玩家堡指针与别名保持一致。
					if player := s.siloByID(s.world.PlayerSiloID); player != nil {
						s.silo = player
					}
					s.agents = map[uint]*model.Agent{}
					for i := range snap.Agents {
						s.agents[snap.Agents[i].SiloID] = &snap.Agents[i]
					}
					if agent, ok := s.agents[s.silo.ID]; ok {
						s.agent = agent
					}
				} else {
					s.silos = []*model.Silo{s.silo}
					s.agents = map[uint]*model.Agent{s.silo.ID: s.agent}
				}
				engine.SetRNG(rand.New(rand.NewSource(sessionSeed(s.silo))))
				s.engine = engine.NewGameEngine()
				s.pendingOperation = nil
				s.delayTasks = map[string]*delayTask{}
				s.receivedSignals = map[uint][]string{}
				s.bindEngineRuntime()
				if err := s.resumeLlmAdvisor(); err != nil {
					return err
				}
				return nil
			}
		}
	}

	// An entirely empty database is a normal first start, not an incompatible
	// save. Existing game rows without a version marker still fail explicitly.
	var meta model.SaveMeta
	err := s.db.First(&meta, "`key` = ?", saveMetaKey).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		var siloCount int64
		if countErr := s.db.Model(&model.Silo{}).Count(&siloCount).Error; countErr != nil {
			return fmt.Errorf("check empty save: %w", countErr)
		}
		if siloCount == 0 {
			return gorm.ErrRecordNotFound
		}
	} else if err != nil {
		return fmt.Errorf("read save meta: %w", err)
	}
	if meta.SaveVersion == 5 {
		return fmt.Errorf("%w: 旧档基于旧身份模型，无法继续，请开新局", ErrIncompatibleSave)
	}
	if meta.SaveVersion == 6 {
		return fmt.Errorf("%w: v6 存档缺少行动门控与多实体结局状态，无法继续，请开新局", ErrIncompatibleSave)
	}
	if meta.SaveVersion != model.CurrentSaveVersion {
		return fmt.Errorf("%w: 存档缺少版本标记或版本不符（存档=%d，当前要求=%d）。请备份后删除 silo40.db 重新开始，旧存档不会被静默改写",
			ErrIncompatibleSave, meta.SaveVersion, model.CurrentSaveVersion)
	}
	// v3 存档必须带世界行；缺失即半迁移状态，显式失败。
	var world model.World
	if err := s.db.First(&world, ActiveWorldID).Error; err != nil {
		return fmt.Errorf("%w: 存档缺少世界记录（world id=%d）。v3 存档必须包含世界层，请从备份还原",
			ErrIncompatibleSave, ActiveWorldID)
	}
	// 加载世界内全部堡 (按 ID 升序，确定性)。
	var silos []model.Silo
	if err := s.db.Order("id asc").Where("world_id = ?", ActiveWorldID).
		Preload("Resources").Preload("Professions").Preload("Floors").
		Preload("Cohorts").Preload("Residents").Preload("Factions").
		Find(&silos).Error; err != nil {
		return err
	}
	if len(silos) == 0 {
		return fmt.Errorf("%w: 世界内无地堡", ErrIncompatibleSave)
	}
	// 反向偏移：DB 中子表主键与外键统一加了 base，读档时减回堡内 1 起始。
	for i := range silos {
		unoffsetSiloSubIDs(&silos[i], siloIDBase(silos[i].ID))
	}
	// 加载每堡的操作者特工 (按 silo_id IN 堡 ID 集合)。
	siloIDs := make([]uint, 0, len(silos))
	for _, silo := range silos {
		siloIDs = append(siloIDs, silo.ID)
	}
	var agents []model.Agent
	if err := s.db.Order("silo_id asc").Preload("Connections").
		Where("silo_id IN ?", siloIDs).Find(&agents).Error; err != nil {
		return err
	}
	agentBySilo := map[uint]*model.Agent{}
	for i := range agents {
		unoffsetConnections(agents[i].Connections, siloIDBase(agents[i].SiloID))
		agentBySilo[agents[i].SiloID] = &agents[i]
	}
	var eventLogs []model.StoryEventLog
	if err := s.db.Order("id asc").Where("silo_id IN ?", siloIDs).Find(&eventLogs).Error; err != nil {
		return err
	}
	var contentStates []model.ContentEventState
	if err := s.db.Where("silo_id IN ?", siloIDs).Find(&contentStates).Error; err != nil {
		return err
	}
	world.Pacing = world.Pacing.Normalize()
	if err := validateStationAssignment(&world, silos, agentBySilo[world.PlayerSiloID]); err != nil {
		return fmt.Errorf("%w: %v。请保留备份并开新局，旧档不会被继续执行", ErrIncompatibleSave, err)
	}
	s.world = &world
	s.silos = make([]*model.Silo, len(silos))
	for i := range silos {
		s.silos[i] = &silos[i]
	}
	s.agents = agentBySilo
	if player := s.siloByID(world.PlayerSiloID); player != nil {
		s.silo = player
	} else {
		s.silo = s.silos[0]
	}
	if agent, ok := s.agents[s.silo.ID]; ok {
		s.agent = agent
	} else if len(agents) > 0 {
		s.agent = &agents[0]
	}
	s.eventLogs = eventLogs
	s.contentStates = map[uint]map[string]model.ContentEventState{}
	for _, state := range contentStates {
		s.statesMapFor(state.SiloID)[state.EventKey] = state
	}
	engine.SetRNG(rand.New(rand.NewSource(sessionSeed(s.silo))))
	s.engine = engine.NewGameEngine()
	s.pendingOperation = nil
	s.delayTasks = map[string]*delayTask{}
	s.receivedSignals = map[uint][]string{}
	s.bindEngineRuntime()
	if err := s.resumeLlmAdvisor(); err != nil {
		return err
	}
	return nil
}

// unoffsetSiloSubIDs 把 DB 中加过 base 的子表主键与堡内外键减回堡内 1 起始
// (与 persist 的 offset* 函数对称)。
func unoffsetSiloSubIDs(silo *model.Silo, base uint) {
	if base == 0 {
		return
	}
	for i := range silo.Professions {
		if silo.Professions[i].ID >= base {
			silo.Professions[i].ID -= base
		}
	}
	for i := range silo.Cohorts {
		c := &silo.Cohorts[i]
		if c.ID >= base {
			c.ID -= base
		}
		if c.ProfessionID >= base {
			c.ProfessionID -= base
		}
		if c.FactionID != nil && *c.FactionID >= base {
			fid := *c.FactionID - base
			c.FactionID = &fid
		}
	}
	for i := range silo.Factions {
		f := &silo.Factions[i]
		if f.ID >= base {
			f.ID -= base
		}
		if f.RepresentativeResidentID >= base {
			f.RepresentativeResidentID -= base
		}
		if f.RepresentativeCohortID != nil && *f.RepresentativeCohortID >= base {
			cid := *f.RepresentativeCohortID - base
			f.RepresentativeCohortID = &cid
		}
	}
	for i := range silo.Residents {
		r := &silo.Residents[i]
		if r.ID >= base {
			r.ID -= base
		}
		if r.ProfessionID >= base {
			r.ProfessionID -= base
		}
		if r.CohortID != nil && *r.CohortID >= base {
			cid := *r.CohortID - base
			r.CohortID = &cid
		}
		if r.FactionID != nil && *r.FactionID >= base {
			fid := *r.FactionID - base
			r.FactionID = &fid
		}
	}
}

func unoffsetConnections(conns []model.Connection, base uint) {
	if base == 0 {
		return
	}
	for i := range conns {
		if conns[i].ProfessionID >= base {
			conns[i].ProfessionID -= base
		}
	}
}

// resumeLlmAdvisor loads the decision log so LLM-influenced history
// replays from recorded choices (design §2.2/§11). A broken or gapped log
// fails the resume explicitly instead of silently diverging.
func (s *GameService) resumeLlmAdvisor() error {
	if s.llmAdvisor == nil {
		return nil
	}
	entries, err := llm.LoadEntries(s.db)
	if err != nil {
		return fmt.Errorf("load LLM decision log: %w", err)
	}
	s.llmAdvisor.ResumeFrom(entries)
	if err := s.llmAdvisor.Error(); err != nil {
		return err
	}
	s.llmAdvisor.ObserveMonth(s.silo, s.agent, nil)
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

	// 种子决议：请求显式种子 > SILO40_SEED 主种子派生 > 随机（默认行为不变）。
	// 种子随存档持久化 (Silo.RNGSeed)，读档时按它重建随机源。
	seed := req.Seed
	if seed == 0 {
		if s.masterRng != nil {
			seed = s.masterRng.Int63()
		} else {
			seed = rand.Int63()
		}
	}
	engine.SetRNG(rand.New(rand.NewSource(seed)))

	// The primary silo is the operative's assigned station. Silo 40 exists only
	// as the textual sender and is never created as a simulation entity.
	silo := createAssignedStation(seed, startYear, req.TraitIds)
	silo.RNGSeed = seed
	silo.WorldID = ActiveWorldID
	silo.ID = ActiveSiloID
	agent := engine.CreateInitialAgent(req.AgentName, req.Profession, req.TraitIds, silo)
	limitPlayerKnowledge(agent)
	agent.ID = ActiveAgentID
	agent.SiloID = ActiveSiloID
	agent.StationSiloID = ActiveSiloID

	// 世界层：主种子 = 玩家堡种子 (旧档升级兼容)；节奏旋钮缺省全 1.0。
	pacing := model.DefaultWorldPacing()
	if req.Pacing != nil {
		pacing = req.Pacing.Normalize()
	}
	world := &model.World{
		ID:           ActiveWorldID,
		Seed:         seed,
		CurrentYear:  silo.CurrentYear,
		CurrentMonth: silo.CurrentMonth,
		PlayerSiloID: ActiveSiloID,
		Pacing:       pacing,
		Silo1: model.Silo1State{
			Alive:      true,
			OrderPhase: "DORMANT",
			Attention:  map[uint]float64{},
		},
		Topology:        map[uint][]uint{},
		CrossQueue:      []model.CrossEvent{},
		Flags:           map[string]bool{},
		ProgressStage:   s.initialProgressStage(),
		UnlockedActions: []string{},
	}

	silos := []*model.Silo{silo}
	agents := map[uint]*model.Agent{ActiveSiloID: agent}
	nextSiloID := ActiveSiloID + 1
	linkSilos := func(a, b uint) {
		world.Topology[a] = append(world.Topology[a], b)
		world.Topology[b] = append(world.Topology[b], a)
	}

	// The playable scenario opts into a real Silo 1. It uses the same local
	// pipeline and AI operator as Silo 18; the legacy script remains untouched
	// for old two-silo requests and saves.
	if req.IncludeSilo1 {
		peer := createSilo1(startYear)
		peer.RNGSeed = seed
		peer.WorldID = ActiveWorldID
		peer.ID = nextSiloID
		nextSiloID++
		peerAgent := engine.CreateInitialAgent(silo1AgentName, "IT", nil, peer)
		peerAgent.ID = peer.ID
		peerAgent.SiloID = peer.ID
		silos = append(silos, peer)
		agents[peer.ID] = peerAgent
		linkSilos(ActiveSiloID, peer.ID)
		world.Silo1.OrderPhase = "LOCAL_SIM"
	}

	if req.Neighbor != nil {
		neighbor := createSilo18(startYear)
		neighbor.RNGSeed = seed
		neighbor.WorldID = ActiveWorldID
		neighbor.ID = nextSiloID
		nextSiloID++
		neighborAgent := engine.CreateInitialAgent(silo18AgentName, "IT", nil, neighbor)
		neighborAgent.ID = neighbor.ID
		neighborAgent.SiloID = neighbor.ID
		silos = append(silos, neighbor)
		agents[neighbor.ID] = neighborAgent
		linkSilos(ActiveSiloID, neighbor.ID)
		if peer := fullSilo1In(silos); peer != nil {
			linkSilos(peer.ID, neighbor.ID)
		} else {
			world.Silo1.OrderPhase = "WATCHING"
		}
	}

	// The station already occupies ActiveSiloID; no fourth silo and no Silo 40
	// entity are added. Silo 18 remains isolated from direct player action.

	// 初始化引擎 (清空旧调度器状态)
	s.engine = engine.NewGameEngine()
	s.bindEngineRuntime()
	s.world = world
	s.silos = silos
	s.agents = agents
	s.silo = silo
	s.agent = agent
	s.currentSilo = nil
	s.currentAgent = nil
	s.receivedSignals = map[uint][]string{}
	s.eventLogs = nil
	s.contentStates = map[uint]map[string]model.ContentEventState{}
	s.delayTasks = map[string]*delayTask{}
	s.pendingOperation = nil

	// LLM advisor: fresh timeline — wipe the previous game's log, record the
	// initial participation level, arm gating for month 1.
	if s.llmAdvisor != nil {
		if err := s.llmAdvisor.ResetForNewGame(); err != nil {
			return nil, err
		}
		if err := s.llmAdvisor.SetLevel(s.llmAdvisor.GetLevel(), "create"); err != nil {
			return nil, fmt.Errorf("record initial LLM participation: %w", err)
		}
		s.llmAdvisor.ObserveMonth(silo, agent, nil)
	}

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

// SaveGame persists a settled turn. Pending operations must finish first so the
// SQLite graph never captures a partially resolved action or event queue.
func (s *GameService) SaveGame() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSession() {
		return fmt.Errorf("no active game session")
	}
	if s.pendingOperation != nil || s.engine.Bus.Pending() > 0 {
		return fmt.Errorf("finish the pending operation before saving")
	}
	return s.persist()
}

func (s *GameService) ensureNoPendingOperation() error {
	if !s.hasSession() {
		return fmt.Errorf("no active game session")
	}
	if s.pendingOperation != nil || s.engine.Bus.Pending() > 0 {
		return fmt.Errorf("there is already a pending event operation")
	}
	return nil
}

func (s *GameService) bindEngineRuntime() {
	if s.engine == nil {
		return
	}
	s.engine.RuleEngine.OnSchedule = func(event *engine.GameEvent, delayMonths int, source *engine.GameEvent) {
		s.scheduleDelayedEvent(event, delayMonths, source)
	}
	if s.llmAdvisor != nil {
		s.engine.NpcPolicy = s.llmAdvisor
	}
	s.engine.WeightedEndingResolver = func(silo *model.Silo, agent *model.Agent) *model.EndingDecision {
		if s.world == nil || silo == nil || silo.ID != s.world.PlayerSiloID {
			return nil
		}
		return s.resolveWeightedEnding(silo, agent)
	}
	s.bindContentEventHandlers()
}

// bindLlmEventFilter wires the ONCE-event predicate used by the events
// participation level: category ∈ {special, crisis, history} with
// fire_mode=ONCE (existing fields, design §9.1).
func (s *GameService) bindLlmEventFilter() {
	if s.llmAdvisor == nil {
		return
	}
	once := make(map[string]bool, len(s.contentDefinitions))
	for _, def := range s.contentDefinitions {
		if def.FireMode == engine.ContentFireModeOnce {
			once[def.EventID] = true
		}
	}
	s.llmAdvisor.SetEventFilter(func(category, eventID string) bool {
		if category != "special" && category != "crisis" && category != "history" {
			return false
		}
		return once[eventID]
	})
}

func (s *GameService) seedPendingOperation() {
	if s.pendingOperation == nil || s.engine.Bus.Pending() > 0 {
		return
	}
	op := s.pendingOperation
	switch op.Phase {
	case phaseActionRoot:
		target := s.playerActionSilo()
		actor := engine.CreateActorRefForAgent(s.agent, target)
		s.engine.EnqueueAction(actor, target, *op.Action, s.agent)
	case phaseQueued:
		return
	}
}

func (s *GameService) advancePendingOperationAfterDrain() {
	if s.pendingOperation == nil || s.engine.Bus.Pending() > 0 {
		return
	}
	op := s.pendingOperation
	switch op.Phase {
	case phaseActionRoot:
		if op.ActionResult == nil {
			op.ActionResult = s.engine.ConsumeActionResult()
		}
		if op.ActionResult == nil {
			failed := model.ActionResult{Executed: false, Message: "Unknown error."}
			op.ActionResult = &failed
		}
		if !op.ActionResult.Executed {
			s.pendingOperation = nil
			return
		}

		duration := s.actionDurationMonths(*op.Action)
		if duration > 0 {
			op.RemainingMonths = duration
			op.Phase = phaseQueued
			return
		}

		// 瞬时动作：执行即时结算并触发内容
		s.engine.EnqueueInstantResolution(s.playerActionSilo(), s.agent)
		s.engine.Bus.Publish(s.engine.CreateTimeContentEvent(s.playerActionSilo(), s.agent, "ACTION:"+string(op.Action.Type)+":CONTENT"))
		op.Phase = phaseQueued
	case phaseQueued:
		if op.RemainingMonths > 0 {
			// 仍有剩余月份，保持 phaseQueued，等待下一轮 ProcessNextEvent 触发 runOneMonth
			return
		}
		if op.Action != nil {
			s.completePlayerAction(*op.Action)
		}
		s.pendingOperation = nil
	}
}

// completePlayerAction records a generic, content-addressable completion node.
// It is called only after the action and any duration have fully resolved.
func (s *GameService) completePlayerAction(action model.AgentAction) {
	if s.silo == nil {
		return
	}
	key := "player_action:" + s.actionIDForGate(action)
	state := s.statesMapFor(s.silo.ID)[key]
	state.SiloID = s.silo.ID
	state.EventKey = key
	state.Triggered = true
	state.TriggerCount++
	state.LastTriggeredTimestamp = gameTimestamp(s.silo.CurrentYear, s.silo.CurrentMonth)
	s.statesMapFor(s.silo.ID)[key] = state
	s.refreshActionProgress()
}

func (s *GameService) nextDelayTaskID() string {
	s.delayCounter++
	return "delay#" + strconv.Itoa(s.delayCounter)
}

func (s *GameService) scheduleDelayedEvent(event *engine.GameEvent, delayMonths int, source *engine.GameEvent) {
	if s.engine == nil || event == nil || delayMonths < 0 {
		return
	}

	cloned := cloneGameEvent(event)
	taskID := s.nextDelayTaskID()
	if cloned.Data == nil {
		cloned.Data = map[string]interface{}{}
	}
	cloned.Data["delay_task_id"] = taskID

	startEventID := ""
	if source != nil {
		startEventID = source.ID
	}
	s.delayTasks[taskID] = &delayTask{
		ID:           taskID,
		StartEventID: startEventID,
		EndEventID:   cloned.ID,
		DelayMonths:  delayMonths,
		Status:       "scheduled",
		TargetSiloID: s.simSiloID(),
		Event:        cloned,
	}

	// 如果延迟为 0，立即触发
	if delayMonths == 0 {
		s.engine.Bus.Publish(cloned)
	}
}

// checkDelayedEventsStaged 递减全部延迟任务一次，把本月到期的任务按目标堡分组
// 返回 (不发布)。调用方在各目标堡的本地阶段开始前按序发布，保证跨堡场景下任务
// 只在归属堡的结算上下文里生效。单堡时全部任务归属玩家堡，与历史行为一致。
//
// 确定性：delayTasks 是 map，直接 range 会随机化同月到期任务的触发顺序，
// 不同的顺序以不同方式消耗共享 RNG 并以不同序列变更状态，破坏同种子复现契约。
// 按创建序号 ("delay#N" 的数字后缀) 排序，保证先调度先触发。
func (s *GameService) checkDelayedEventsStaged() map[uint][]*engine.GameEvent {
	staged := map[uint][]*engine.GameEvent{}
	if len(s.delayTasks) == 0 {
		return staged
	}
	ids := make([]string, 0, len(s.delayTasks))
	for id := range s.delayTasks {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		si, sj := delayTaskSeq(ids[i]), delayTaskSeq(ids[j])
		if si != sj {
			return si < sj
		}
		return ids[i] < ids[j]
	})

	for _, id := range ids {
		task := s.delayTasks[id]
		if task.Status != "scheduled" {
			continue
		}
		task.DelayMonths--
		if task.DelayMonths <= 0 {
			task.Status = "triggered"
			target := task.TargetSiloID
			if target == 0 {
				target = ActiveSiloID
			}
			staged[target] = append(staged[target], task.Event)
		}
	}
	return staged
}

// publishStagedDelayedEvents 按序把某一堡的到期延迟事件发布到总线。
func (s *GameService) publishStagedDelayedEvents(staged map[uint][]*engine.GameEvent, siloID uint) {
	for _, event := range staged[siloID] {
		s.engine.Bus.Publish(event)
	}
}

// delayTaskSeq extracts the creation sequence from a delay task ID
// ("delay#N"). IDs that do not carry a numeric suffix sort last.
func delayTaskSeq(id string) int {
	if i := strings.LastIndex(id, "#"); i >= 0 {
		if n, err := strconv.Atoi(id[i+1:]); err == nil {
			return n
		}
	}
	return int(^uint(0) >> 1)
}

func (s *GameService) markDelayTaskIfCompleted(processed *engine.GameEvent) {
	if processed == nil || processed.Data == nil {
		return
	}
	taskID, _ := processed.Data["delay_task_id"].(string)
	if taskID == "" {
		return
	}
	if task, ok := s.delayTasks[taskID]; ok {
		task.Status = "completed"
	}
}

// BeginPassTime 入队一个月份推进操作，但不立即处理事件。
func (s *GameService) BeginPassTime(months int) (*model.EventQueueState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureNoPendingOperation(); err != nil {
		return nil, err
	}
	if months <= 0 {
		months = 1
	}
	s.pendingOperation = &pendingOperation{
		Kind:            "pass_time",
		Phase:           phaseQueued,
		RemainingMonths: months,
	}
	s.cacheSnapshot()
	return s.buildQueueState(), nil
}

// BeginAction 入队一个玩家动作，但不立即处理事件。
func (s *GameService) BeginAction(action model.AgentAction) (*model.EventQueueState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureNoPendingOperation(); err != nil {
		return nil, err
	}
	if err := s.resolveActionCost(&action); err != nil {
		return nil, err
	}
	if err := s.validateActionGate(action); err != nil {
		return nil, err
	}

	handledByContent, result, err := s.queuePlayerActionEvents(action)
	if err != nil {
		return nil, err
	}

	op := &pendingOperation{
		Kind:           "action",
		Action:         &action,
		ActionDuration: s.actionDurationMonths(action),
	}
	if handledByContent {
		if !result.Executed {
			return nil, fmt.Errorf("%s", result.Message)
		}
		op.ActionResult = &result
		duration := op.ActionDuration
		if duration > 0 {
			op.RemainingMonths = duration
			op.Phase = phaseQueued
		} else {
			op.Phase = phaseQueued
			// 瞬时动作内容触发
			s.engine.EnqueueInstantResolution(s.playerActionSilo(), s.agent)
			s.engine.Bus.Publish(s.engine.CreateTimeContentEvent(s.playerActionSilo(), s.agent, "ACTION:"+string(action.Type)+":CONTENT"))
		}
	} else {
		op.Phase = phaseActionRoot
	}
	s.pendingOperation = op
	s.seedPendingOperation()
	s.cacheSnapshot()
	return s.buildQueueState(), nil
}

// ProcessNextEvent 处理下一个排队事件，并返回完整快照。
func (s *GameService) ProcessNextEvent() (*model.EventStepResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSession() {
		return nil, fmt.Errorf("no active game session")
	}

	s.seedPendingOperation()

	// 如果队列为空但还有剩余月份，推进一个月
	var monthLogs []string
	var monthStories []model.StoryEvent
	advancedMonth := false
	if s.engine.Bus.Pending() == 0 && s.pendingOperation != nil && s.pendingOperation.RemainingMonths > 0 {
		source := "PASS_TIME:CONTENT"
		if s.pendingOperation.Kind == "action" {
			source = "ACTION:" + string(s.pendingOperation.Action.Type) + ":CONTENT"
		}
		monthLogs, monthStories = s.runOneMonth(source)
		s.applyHistoryInfluenceScore()
		if err := s.llmError(); err != nil {
			return nil, err
		}
		s.pendingOperation.RemainingMonths--
		advancedMonth = true
	}

	processed, logs, stories := s.engine.Step(s.playerActionSilo(), s.agent)
	if processed == nil {
		if !advancedMonth {
			return nil, fmt.Errorf("no pending event")
		}
		// 月度 tick 已在 runOneMonth 内整体排空（内容事件随 TIME_CONTENT 一并结算）。
		// 把该月的故事/日志回传给呈现端，并推进操作状态——否则纯推进月份的操作会
		// 永远卡在 phaseQueued（队列里没有可供 Step 的事件）。
		actionResult := s.pendingActionResult()
		s.advancePendingOperationAfterDrain()
		complete := s.pendingOperation == nil && s.engine.Bus.Pending() == 0
		if complete || s.gameOver() {
			if err := s.persist(); err != nil {
				return nil, err
			}
		}
		s.cacheSnapshot()
		return s.buildStepResult(nil, monthLogs, monthStories, complete, actionResult), nil
	}

	stepLogs := append(append([]string(nil), monthLogs...), logs...)
	stepStories := make([]model.StoryEvent, 0, len(monthStories)+len(stories))
	stepStories = append(stepStories, monthStories...)
	for _, story := range stories {
		if story != nil {
			stepStories = append(stepStories, *story)
		}
	}
	if moreLogs, ok := processed.Data["step_logs"].(*[]string); ok && moreLogs != nil {
		stepLogs = append(stepLogs, (*moreLogs)...)
	}
	if moreStories, ok := processed.Data["step_stories"].(*[]model.StoryEvent); ok && moreStories != nil {
		stepStories = append(stepStories, (*moreStories)...)
	}

	s.markDelayTaskIfCompleted(processed)
	actionResult := s.pendingActionResult()
	s.advancePendingOperationAfterDrain()
	s.seedPendingOperation()
	complete := s.pendingOperation == nil && s.engine.Bus.Pending() == 0

	if s.gameOver() {
		if err := s.persist(); err != nil {
			return nil, err
		}
	}
	s.cacheSnapshot()
	return s.buildStepResult(processed, stepLogs, stepStories, complete, actionResult), nil
}

func (s *GameService) runOneMonth(source string) ([]string, []model.StoryEvent) {
	if s.multiSilo() {
		return s.runOneMonthWorld(source)
	}
	return s.runOneMonthSingle(source)
}

// runOneMonthSingle 单堡 (空邻堡) 月度推进：与历史流水线逐位一致，是 P0.5 的
// 奇偶校验锚点。延迟任务全部归属玩家堡，在本地结算前发布。
func (s *GameService) runOneMonthSingle(source string) ([]string, []model.StoryEvent) {
	staged := s.checkDelayedEventsStaged()
	s.publishStagedDelayedEvents(staged, ActiveSiloID)
	logCursor := len(s.eventLogs)
	logs, stories := s.engine.UpdateSiloState(s.silo, 1.0/12.0, s.agent, source)
	// 世界时钟镜像玩家堡时钟。权威时钟仍在堡内流水线；多堡阶段 (P0.5+) 会把
	// 时钟推进上移到世界 tick 编排，此处改为世界时钟下发到各堡。
	if s.world != nil {
		s.world.CurrentYear = s.silo.CurrentYear
		s.world.CurrentMonth = s.silo.CurrentMonth
	}

	// LLM gating state for the next month: metric bands, faction structure
	// changes and ONCE special-event windows (design §9). Errors are sticky
	// and surfaced by callers via llmError().
	if s.llmAdvisor != nil {
		newLogs := s.eventLogs[logCursor:]
		s.llmAdvisor.ObserveMonth(s.silo, s.agent, newLogs)
	}

	storyList := make([]model.StoryEvent, 0, len(stories))
	for _, st := range stories {
		if st != nil {
			storyList = append(storyList, *st)
		}
	}
	return logs, storyList
}

// PassTime 推进 months 个月，返回 tick 结果
func (s *GameService) PassTime(months int) (*model.TickResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSession() {
		return nil, fmt.Errorf("no active game session")
	}

	var allLogs []string
	var allStories []model.StoryEvent

	for i := 0; i < months; i++ {
		logs, stories := s.runOneMonth("PASS_TIME:CONTENT")
		s.applyHistoryInfluenceScore()
		if err := s.llmError(); err != nil {
			return nil, err
		}
		allLogs = append(allLogs, logs...)
		allStories = append(allStories, stories...)
	}

	// Preserve the synchronous API's existing critical-node persistence.
	if s.gameOver() {
		if err := s.persist(); err != nil {
			return nil, err
		}
	}
	s.cacheSnapshot()

	return &model.TickResult{
		Silo:             s.publicSiloSnapshot(),
		Agent:            *s.agent,
		AgentStats:       s.engine.BuildAgentStats(s.agent, s.playerActionSilo()),
		AvailableActions: s.availablePlayerActions(),
		World:            s.publicWorldView(),
		Logs:             allLogs,
		Stories:          allStories,
		GameOver:         s.gameOver(),
		EndingNarrative:  s.endingNarrative(),
	}, nil
}

// ExecuteAction 执行玩家动作；时长>0 的动作自动推进时间
func (s *GameService) ExecuteAction(action model.AgentAction) (*model.ActionOutcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSession() {
		return nil, fmt.Errorf("no active game session")
	}
	if err := s.resolveActionCost(&action); err != nil {
		return nil, err
	}
	if err := s.validateActionGate(action); err != nil {
		return nil, err
	}

	var allLogs []string
	var allStories []model.StoryEvent
	handledByContent, result, playerActionLogs, playerActionStories := s.executePlayerActionEvents(action)
	allLogs = append(allLogs, playerActionLogs...)
	allStories = append(allStories, playerActionStories...)
	if !handledByContent {
		target := s.playerActionSilo()
		actor := engine.CreateActorRefForAgent(s.agent, target)
		s.engine.EnqueueAction(actor, target, action, s.agent)
		actionLogs, actionStories := s.drainQueuedEvents()
		allLogs = append(allLogs, actionLogs...)
		allStories = append(allStories, actionStories...)
		if consumed := s.engine.ConsumeActionResult(); consumed != nil {
			result = *consumed
		} else {
			result = model.ActionResult{Executed: false, Message: "Unknown error."}
		}
	}

	if result.Executed {
		duration := s.actionDurationMonths(action)
		source := "ACTION:" + string(action.Type) + ":CONTENT"
		if duration > 0 {
			for i := 0; i < duration; i++ {
				logs, stories := s.runOneMonth(source)
				s.applyHistoryInfluenceScore()
				if err := s.llmError(); err != nil {
					return nil, err
				}
				allLogs = append(allLogs, logs...)
				allStories = append(allStories, stories...)
			}
		} else {
			s.engine.EnqueueInstantResolution(s.playerActionSilo(), s.agent)
			s.engine.Bus.Publish(s.engine.CreateTimeContentEvent(s.playerActionSilo(), s.agent, source))
			moreLogs, moreStories := s.drainQueuedEvents()
			allLogs = append(allLogs, moreLogs...)
			allStories = append(allStories, moreStories...)
		}
	}

	if result.Executed {
		s.completePlayerAction(action)
	}
	if s.gameOver() {
		if err := s.persist(); err != nil {
			return nil, err
		}
	}
	s.cacheSnapshot()

	return &model.ActionOutcome{
		Silo:             s.publicSiloSnapshot(),
		Agent:            *s.agent,
		AgentStats:       s.engine.BuildAgentStats(s.agent, s.playerActionSilo()),
		AvailableActions: s.availablePlayerActions(),
		World:            s.publicWorldView(),
		Result:           result,
		Logs:             allLogs,
		Stories:          allStories,
		GameOver:         s.gameOver(),
		EndingNarrative:  s.endingNarrative(),
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

// GetEventHistory 获取事件历史，limit<=0 表示返回全部
func (s *GameService) GetEventHistory(limit int) (*model.EventHistoryResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSession() {
		return nil, fmt.Errorf("no active game session")
	}
	return &model.EventHistoryResult{Events: s.eventHistory(limit)}, nil
}

// ---------- 私有 ----------

func resolveContentDirectories() (map[string]string, error) {
	dirs := map[string]string{}

	cwd, _ := os.Getwd()
	exePath, _ := os.Executable()
	exeDir := ""
	if exePath != "" {
		exeDir = filepath.Dir(exePath)
	}

	// Build artifacts should prefer the events/ directory placed next to the binary.
	for _, base := range []string{exeDir, cwd} {
		for group, path := range discoverContentDirectoriesExact(base) {
			if dirs[group] == "" {
				dirs[group] = path
			}
		}
	}

	// Keep upward search as a compatibility fallback for dev and older layouts.
	for _, base := range []string{cwd, exeDir} {
		for group, path := range discoverContentDirectories(base) {
			if dirs[group] == "" {
				dirs[group] = path
			}
		}
	}

	if len(dirs) == 0 {
		return nil, fmt.Errorf("failed to locate content directories under events/")
	}
	return dirs, nil
}

func contentEventDirectories(dirs map[string]string) map[string]string {
	out := map[string]string{}
	for group, path := range dirs {
		switch group {
		case "histories", "history", "special", "crisis", "rules", "player_actions", "player_action", "events":
			out[group] = path
		}
	}
	return out
}

func discoverContentDirectoriesExact(baseDir string) map[string]string {
	if baseDir == "" {
		return map[string]string{}
	}
	return discoverContentDirectoriesFromRoot(filepath.Join(baseDir, "events"), "")
}

func discoverContentDirectories(baseDir string) map[string]string {
	root := findContentDirectory(baseDir, "events")
	return discoverContentDirectoriesFromRoot(root, baseDir)
}

func discoverContentDirectoriesFromRoot(root, baseDir string) map[string]string {
	dirs := map[string]string{}
	if root != "" {
		entries, err := os.ReadDir(root)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				group := strings.TrimSpace(entry.Name())
				if group == "" {
					continue
				}
				dirs[group] = filepath.Join(root, group)
			}
		}

		// Backward compatibility for the legacy flat layout: only when no
		// group subdirectories exist at all. With group dirs present,
		// root-level .json files are non-event config (e.g. action gates)
		// and must not be parsed as content event definitions.
		if len(dirs) == 0 {
			hasEventFiles, err := directoryHasContentEventFiles(root)
			if err == nil && hasEventFiles && dirs["events"] == "" {
				dirs["events"] = root
			}
		}
	}

	// Backward compatibility for the legacy top-level histories/ directory.
	if dirs["histories"] == "" {
		if legacy := findContentDirectory(baseDir, "histories"); legacy != "" {
			dirs["histories"] = legacy
		}
	}
	return dirs
}

func findContentDirectory(baseDir, group string) string {
	if baseDir == "" {
		return ""
	}
	current := baseDir
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(current, group)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return ""
}

func directoryHasContentEventFiles(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "action_gates.json" {
			continue
		}
		switch strings.ToLower(filepath.Ext(entry.Name())) {
		case ".json", ".js":
			return true, nil
		}
	}
	return false, nil
}

func (s *GameService) syncContentDefinitions(dirs map[string]string) error {
	defs, err := engine.LoadContentEventDefinitions(contentEventDirectories(dirs))
	if err != nil {
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		activeKeys := make([]string, 0, len(defs))
		for _, def := range defs {
			activeKeys = append(activeKeys, def.Key)
			triggerJSON, err := json.Marshal(def.Trigger)
			if err != nil {
				return fmt.Errorf("failed to marshal trigger for %s: %w", def.Key, err)
			}
			effectsJSON, err := json.Marshal(def.Effects)
			if err != nil {
				return fmt.Errorf("failed to marshal effects for %s: %w", def.Key, err)
			}
			playerActionJSON := ""
			if def.PlayerAction != nil {
				payload, err := json.Marshal(def.PlayerAction)
				if err != nil {
					return fmt.Errorf("failed to marshal player_action for %s: %w", def.Key, err)
				}
				playerActionJSON = string(payload)
			}

			record := model.ContentEventDefinition{
				Key:              def.Key,
				SourceGroup:      def.SourceGroup,
				SourceFile:       def.SourceFile,
				SourceFormat:     def.SourceFormat,
				EventID:          def.EventID,
				Title:            def.Title,
				Description:      def.Description,
				Type:             def.Type,
				FireMode:         def.FireMode,
				CooldownMonths:   def.CooldownMonths,
				Enabled:          true,
				TriggerSpec:      string(triggerJSON),
				EffectsSpec:      string(effectsJSON),
				PlayerActionSpec: playerActionJSON,
				ScriptSource:     def.ScriptSource,
				ScriptCanTrigger: def.ScriptHooks.CanTrigger,
				ScriptApply:      def.ScriptHooks.Apply,
			}

			err = tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "key"}},
				DoUpdates: clause.AssignmentColumns([]string{"source_group", "source_file", "source_format", "event_id", "title", "description", "type", "fire_mode", "cooldown_months", "enabled", "trigger_spec", "effects_spec", "player_action_spec", "script_source", "script_can_trigger", "script_apply", "updated_at"}),
			}).Create(&record).Error
			if err != nil {
				return fmt.Errorf("failed to upsert content event %s: %w", def.Key, err)
			}
		}

		query := tx.Model(&model.ContentEventDefinition{})
		if len(activeKeys) > 0 {
			query = query.Where("key NOT IN ?", activeKeys)
		} else {
			query = query.Session(&gorm.Session{AllowGlobalUpdate: true})
		}
		if err := query.Update("enabled", false).Error; err != nil {
			return fmt.Errorf("failed to disable stale content events: %w", err)
		}
		return nil
	})
}

func (s *GameService) loadContentDefinitions() error {
	var rows []model.ContentEventDefinition
	if err := s.db.Where("enabled = ?", true).Order("source_group asc, source_file asc, event_id asc").Find(&rows).Error; err != nil {
		return fmt.Errorf("failed to load content event definitions: %w", err)
	}

	defs := make([]engine.ContentEventDefinition, 0, len(rows))
	for _, row := range rows {
		def := engine.ContentEventDefinition{
			ID:             row.ID,
			Key:            row.Key,
			SourceGroup:    row.SourceGroup,
			SourceFile:     row.SourceFile,
			SourceFormat:   row.SourceFormat,
			EventID:        row.EventID,
			Title:          row.Title,
			Description:    row.Description,
			Type:           row.Type,
			FireMode:       row.FireMode,
			CooldownMonths: row.CooldownMonths,
			ScriptSource:   row.ScriptSource,
			ScriptHooks: engine.ContentScriptHooks{
				CanTrigger: row.ScriptCanTrigger,
				Apply:      row.ScriptApply,
			},
		}
		if err := json.Unmarshal([]byte(row.TriggerSpec), &def.Trigger); err != nil {
			return fmt.Errorf("failed to parse trigger for %s: %w", row.Key, err)
		}
		if err := json.Unmarshal([]byte(row.EffectsSpec), &def.Effects); err != nil {
			return fmt.Errorf("failed to parse effects for %s: %w", row.Key, err)
		}
		if strings.TrimSpace(row.PlayerActionSpec) != "" {
			var spec engine.ContentPlayerActionSpec
			if err := json.Unmarshal([]byte(row.PlayerActionSpec), &spec); err != nil {
				return fmt.Errorf("failed to parse player_action for %s: %w", row.Key, err)
			}
			def.PlayerAction = &spec
		}
		if err := engine.ValidateContentEventDefinition(def); err != nil {
			return fmt.Errorf("invalid stored content event %s: %w", row.Key, err)
		}
		defs = append(defs, def)
	}
	s.contentDefinitions = defs
	return nil
}

// contentStateSlice 全部堡的内容事件状态 (确定性排序：SiloID 升序 + EventKey)。
// 单堡时与历史排序 (仅 EventKey) 逐位一致。
func (s *GameService) contentStateSlice() []model.ContentEventState {
	if len(s.contentStates) == 0 {
		return nil
	}
	var states []model.ContentEventState
	siloIDs := make([]uint, 0, len(s.contentStates))
	for siloID := range s.contentStates {
		siloIDs = append(siloIDs, siloID)
	}
	sort.Slice(siloIDs, func(i, j int) bool { return siloIDs[i] < siloIDs[j] })
	for _, siloID := range siloIDs {
		for _, state := range s.contentStates[siloID] {
			states = append(states, state)
		}
	}
	sort.Slice(states, func(i, j int) bool {
		if states[i].SiloID != states[j].SiloID {
			return states[i].SiloID < states[j].SiloID
		}
		return states[i].EventKey < states[j].EventKey
	})
	return states
}

func (s *GameService) queueSystemContentEvents(source string) int {
	isPlayerSilo := s.world == nil || s.simSiloID() == s.world.PlayerSiloID
	count, _, _ := s.queueContentEvents(source, nil, func(def engine.ContentEventDefinition) bool {
		if def.SourceGroup == "player_actions" || def.SourceGroup == "player_action" {
			return false
		}
		if engine.HasEventTriggeredTrigger(def.Trigger) {
			return false
		}
		// P0.5 多堡：正史 (histories) 锚定玩家堡叙事，只在玩家堡评估；
		// 单堡时玩家堡即唯一堡，过滤恒为真，行为不变。
		if !isPlayerSilo && def.SourceGroup == "histories" {
			return false
		}
		return true
	})
	return count
}

func (s *GameService) triggerSystemContentEvents(source string) ([]string, []model.StoryEvent) {
	return s.triggerContentEvents(source, nil, func(def engine.ContentEventDefinition) bool {
		return def.SourceGroup != "player_actions" &&
			def.SourceGroup != "player_action" &&
			!engine.HasEventTriggeredTrigger(def.Trigger)
	})
}

// resolveActionCost overwrites action.Cost with the server-authoritative AP
// cost taken from the action definition itself. The client-submitted cost is
// never trusted (a stale or crafted client could otherwise send cost=0 and
// execute for free). The sources here are the same ones that feed
// PlayerActionMeta (mechanic-engine definitions / ACTION_COSTS after the JS
// override), so the charged cost always matches what the UI displayed.
func (s *GameService) resolveActionCost(action *model.AgentAction) error {
	switch action.Type {
	case model.ActionStationOperation:
		def, ok := stationOperationDefinition(action.ActionID)
		if !ok {
			return fmt.Errorf("unknown station operation %q", action.ActionID)
		}
		action.Cost = float64(def.APCost)
	case model.ActionSilo1Operation:
		def, ok := silo1OperationDefinition(action.ActionID)
		if !ok || !silo1OperationAllowed(action.ActionID) {
			return fmt.Errorf("Silo 1 channel permits only intelligence and deception")
		}
		action.Cost = float64(def.APCost)
	case model.ActionPlayerEvent:
		def, ok := s.playerActionDefinition(action.ActionID)
		if !ok || def.PlayerAction == nil {
			return fmt.Errorf("unknown player action %q", action.ActionID)
		}
		action.Cost = float64(def.PlayerAction.APCost)
	case model.ActionProfession:
		def := engine.GetProfessionAction(strings.TrimSpace(action.ProfessionAction))
		if def == nil {
			return fmt.Errorf("unknown profession action %q", action.ProfessionAction)
		}
		action.Cost = def.APCost
	default:
		cost, ok := model.ACTION_COSTS[action.Type]
		if !ok {
			return fmt.Errorf("unknown action type %q", action.Type)
		}
		action.Cost = cost
	}
	return nil
}

func (s *GameService) queuePlayerActionEvents(action model.AgentAction) (bool, model.ActionResult, error) {
	if !s.hasSession() {
		return false, model.ActionResult{}, fmt.Errorf("no active game session")
	}
	if action.Type == model.ActionStationOperation {
		return true, s.scheduleStationOperation(action), nil
	}
	if action.Type == model.ActionSilo1Operation {
		return true, s.scheduleSilo1Operation(action), nil
	}
	if action.Type == model.ActionPlayerEvent {
		def, ok := s.playerActionDefinition(action.ActionID)
		if !ok {
			return true, model.ActionResult{Executed: false, Message: "Unknown player action."}, nil
		}
		if err := validatePlayerActionTarget(def, action); err != nil {
			return true, model.ActionResult{Executed: false, Message: err.Error()}, nil
		}
		if s.agent.ActionPoints < action.Cost {
			return true, model.ActionResult{Executed: false, Message: "Not enough Action Points (AP)."}, nil
		}
	}
	if s.agent.ActionPoints < action.Cost {
		return false, model.ActionResult{Executed: false, Message: "Not enough Action Points (AP)."}, nil
	}

	count, logs, stories := s.queueContentEvents("ACTION:"+string(action.Type)+":PLAYER_EVENT", &action, func(def engine.ContentEventDefinition) bool {
		return def.SourceGroup == "player_actions" || def.SourceGroup == "player_action"
	})
	_ = logs
	_ = stories
	if count == 0 {
		if action.Type == model.ActionPlayerEvent {
			return true, model.ActionResult{Executed: false, Message: "This player action is currently unavailable."}, nil
		}
		return false, model.ActionResult{}, nil
	}

	s.agent.ActionPoints -= action.Cost
	if s.agent.ActionPoints < 0 {
		s.agent.ActionPoints = 0
	}
	message := "Triggered player action event."
	if count > 1 {
		message = fmt.Sprintf("Triggered %d player action events.", count)
	}
	return true, model.ActionResult{Executed: true, Message: message}, nil
}

func (s *GameService) executePlayerActionEvents(action model.AgentAction) (bool, model.ActionResult, []string, []model.StoryEvent) {
	handledByContent, result, err := s.queuePlayerActionEvents(action)
	if err != nil {
		return false, model.ActionResult{Executed: false, Message: err.Error()}, nil, nil
	}
	if !handledByContent {
		return false, result, nil, nil
	}
	logs, stories := s.drainQueuedEvents()
	return true, result, logs, stories
}

func (s *GameService) queueContentEvents(source string, action *model.AgentAction, allow func(engine.ContentEventDefinition) bool) (int, []string, []model.StoryEvent) {
	if !s.hasSession() || len(s.contentDefinitions) == 0 {
		return 0, nil, nil
	}

	var logs []string
	var stories []model.StoryEvent
	simSilo := s.simSilo()
	count := 0
	for _, def := range s.contentDefinitions {
		if allow != nil && !allow(def) {
			continue
		}
		state, _ := s.contentStateForDefinition(def)
		ctx := engine.ContentEvaluationContext{
			Silo:   simSilo,
			States: s.contentRuntimeMap(),
			Action: action,
		}
		// 多堡编排时注入世界事实与本月收到的跨堡信号；单堡保持 nil/空，
		// 世界化触发器按不可触发处理，行为不变。
		if s.multiSilo() {
			ctx.World = s.world
			ctx.SignalsReceived = s.receivedSignals[simSilo.ID]
		}
		ctx.Runtime = engine.ContentEventRuntime{
			Triggered:              state.Triggered,
			TriggerCount:           state.TriggerCount,
			LastTriggeredTimestamp: state.LastTriggeredTimestamp,
		}
		if !engine.CanTriggerContentEvent(def, ctx) {
			continue
		}
		if s.emitContentEvent(def, source, engine.NewEventContext(), action, &stories, &logs) {
			count++
		}
	}
	return count, logs, stories
}

func (s *GameService) triggerContentEvents(source string, action *model.AgentAction, allow func(engine.ContentEventDefinition) bool) ([]string, []model.StoryEvent) {
	_, logs, stories := s.queueContentEvents(source, action, allow)
	moreLogs, moreStories := s.drainQueuedEvents()
	logs = append(logs, moreLogs...)
	stories = append(stories, moreStories...)
	return logs, stories
}

func (s *GameService) drainQueuedEvents() ([]string, []model.StoryEvent) {
	if s.engine == nil {
		return nil, nil
	}
	var logs []string
	var stories []model.StoryEvent
	for s.engine.Bus.Pending() > 0 {
		processed, stepLogs, stepStories := s.engine.Step(s.simSilo(), s.simAgent())
		if processed == nil {
			break
		}
		logs = append(logs, stepLogs...)
		for _, story := range stepStories {
			if story != nil {
				stories = append(stories, *story)
			}
		}
		if moreLogs, ok := processed.Data["step_logs"].(*[]string); ok && moreLogs != nil {
			logs = append(logs, (*moreLogs)...)
		}
		if moreStories, ok := processed.Data["step_stories"].(*[]model.StoryEvent); ok && moreStories != nil {
			stories = append(stories, (*moreStories)...)
		}
	}
	return append([]string(nil), logs...), stories
}

func (s *GameService) contentRuntimeMap() map[string]engine.ContentEventRuntime {
	return s.contentRuntimeMapFor(s.simSiloID())
}

// contentRuntimeMapFor 指定堡的内容事件运行时状态视图。
func (s *GameService) contentRuntimeMapFor(siloID uint) map[string]engine.ContentEventRuntime {
	states := s.contentStates[siloID]
	if len(states) == 0 {
		return nil
	}
	runtimes := make(map[string]engine.ContentEventRuntime, len(states))
	for key, state := range states {
		runtimes[key] = engine.ContentEventRuntime{
			Triggered:              state.Triggered,
			TriggerCount:           state.TriggerCount,
			LastTriggeredTimestamp: state.LastTriggeredTimestamp,
		}
	}
	return runtimes
}

func (s *GameService) contentStateForDefinition(def engine.ContentEventDefinition) (model.ContentEventState, string) {
	return s.contentStateForDefinitionIn(s.simSiloID(), def)
}

// contentStateForDefinitionIn 指定堡内按定义查找触发状态 (含 legacy key 回退)。
func (s *GameService) contentStateForDefinitionIn(siloID uint, def engine.ContentEventDefinition) (model.ContentEventState, string) {
	states := s.contentStates[siloID]
	if state, ok := states[def.Key]; ok {
		return state, def.Key
	}
	for key, state := range states {
		if key == def.EventID || strings.HasSuffix(key, ":"+def.EventID) {
			return state, key
		}
	}
	return model.ContentEventState{}, ""
}

func (s *GameService) bindContentEventHandlers() {
	s.clearContentEventHandlers()
	if s.engine == nil || s.engine.Bus == nil {
		return
	}

	s.contentUnsubscribes = append(s.contentUnsubscribes, s.engine.Bus.Subscribe(engine.EVENT_TIME_CONTENT, func(event *engine.GameEvent, ctx *engine.EventContext) {
		s.handleTimeContentEvent(event)
	}))

	for _, def := range s.contentDefinitions {
		def := def
		s.contentUnsubscribes = append(s.contentUnsubscribes, s.engine.Bus.Subscribe(engine.ContentEventBusName(def), func(event *engine.GameEvent, ctx *engine.EventContext) {
			s.handleContentEvent(def, event, ctx)
		}))
	}

	for _, def := range s.contentDefinitions {
		def := def
		for _, parentEventName := range engine.ContentTriggerEventNames(def) {
			parentEventName := parentEventName
			s.contentUnsubscribes = append(s.contentUnsubscribes, s.engine.Bus.Subscribe(parentEventName, func(event *engine.GameEvent, ctx *engine.EventContext) {
				s.tryEmitFollowupContentEvent(def, event, ctx)
			}))
		}
	}
}

func (s *GameService) clearContentEventHandlers() {
	for _, unsubscribe := range s.contentUnsubscribes {
		if unsubscribe != nil {
			unsubscribe()
		}
	}
	s.contentUnsubscribes = nil
}

func (s *GameService) emitContentEvent(def engine.ContentEventDefinition, source string, ctx *engine.EventContext, action *model.AgentAction, stories *[]model.StoryEvent, logs *[]string) bool {
	if s.engine == nil || s.engine.Bus == nil {
		return false
	}
	stepStories := []model.StoryEvent{}
	stepLogs := []string{}
	data := map[string]interface{}{
		"silo":            s.simSilo(),
		"agent":           s.simAgent(),
		"content_source":  source,
		"story_sink":      stories,
		"log_sink":        logs,
		"step_story_sink": &stepStories,
		"step_log_sink":   &stepLogs,
	}
	if action != nil {
		data["action"] = action
	}
	// story_sink / log_sink may be nil when the emitter is a delay-queue
	// dispatch (scheduled events carry only step sinks). Guard the counter
	// dereference; handleContentEvent already nil-checks the sinks and still
	// records the story through the content_source path.
	storyCount := 0
	if stories != nil {
		storyCount = len(*stories)
	}
	event := engine.CreateEvent("content#"+def.Key+"#"+strconv.Itoa(len(s.eventLogs)+storyCount+1), engine.ContentEventBusName(def), data)
	s.engine.Bus.Emit(event, ctx)
	event.Data["step_stories"] = &stepStories
	event.Data["step_logs"] = &stepLogs
	return true
}

func (s *GameService) handleTimeContentEvent(event *engine.GameEvent) {
	source, _ := event.Data["content_source"].(string)
	if source == "" {
		source = "PASS_TIME:CONTENT"
	}
	count := s.queueSystemContentEvents(source)
	if count == 0 {
		return
	}
	if sink, ok := event.Data["step_logs"].(*[]string); ok && sink != nil {
		*sink = append(*sink, fmt.Sprintf("[TimeContent] Queued %d content events", count))
	}
}

func (s *GameService) handleContentEvent(def engine.ContentEventDefinition, event *engine.GameEvent, ctx *engine.EventContext) {
	simSilo := s.simSilo()
	scopeCtx := s.scopeContextFor(simSilo)
	story, scheduled := engine.ApplyContentEventScoped(def, scopeCtx)
	s.flushScopeResults(scopeCtx, event)

	state, legacyKey := s.contentStateForDefinition(def)
	siloID := s.simSiloID()
	state.SiloID = siloID
	state.DefinitionID = def.ID
	state.EventKey = def.Key
	state.Triggered = true
	state.TriggerCount++
	state.LastTriggeredTimestamp = gameTimestamp(simSilo.CurrentYear, simSilo.CurrentMonth)
	states := s.statesMapFor(siloID)
	if legacyKey != "" && legacyKey != def.Key {
		delete(states, legacyKey)
	}
	states[def.Key] = state

	if sink, ok := event.Data["story_sink"].(*[]model.StoryEvent); ok && sink != nil {
		*sink = append(*sink, story)
	}
	if sink, ok := event.Data["step_story_sink"].(*[]model.StoryEvent); ok && sink != nil {
		*sink = append(*sink, story)
	}
	if sink, ok := event.Data["log_sink"].(*[]string); ok && sink != nil {
		*sink = append(*sink, "[ContentEvent] "+story.Title)
	}
	if sink, ok := event.Data["step_log_sink"].(*[]string); ok && sink != nil {
		*sink = append(*sink, "[ContentEvent] "+story.Title)
	}
	if source, _ := event.Data["content_source"].(string); source != "" {
		storyCopy := story
		s.recordStories([]*model.StoryEvent{&storyCopy}, source)
	}
	event.Data["story_result"] = story

	s.scheduleContentEvents(scheduled, event, ctx)

	if !def.ScriptHooks.Apply {
		return
	}
	evalCtx := engine.ContentEvaluationContext{
		Silo:   simSilo,
		States: s.contentRuntimeMap(),
		Action: contentActionFromEvent(event),
		Runtime: engine.ContentEventRuntime{
			Triggered:              state.Triggered,
			TriggerCount:           state.TriggerCount,
			LastTriggeredTimestamp: state.LastTriggeredTimestamp,
		},
	}
	if s.multiSilo() {
		evalCtx.World = s.world
		evalCtx.SignalsReceived = s.receivedSignals[siloID]
	}
	scriptResult, err := engine.RunContentEventApplyScript(def, evalCtx)
	if err != nil {
		s.appendContentLog(event, fmt.Sprintf("[ContentEvent] Script apply failed for %s: %v", def.Key, err))
		return
	}
	scriptScopeCtx := s.scopeContextFor(simSilo)
	scheduledFromScript := engine.ApplyContentEffectsScoped(scriptResult.Effects, scriptScopeCtx)
	s.flushScopeResults(scriptScopeCtx, event)
	s.scheduleContentEvents(scheduledFromScript, event, ctx)
	s.scheduleContentEvents(scriptResult.Schedule, event, ctx)

	for _, name := range scriptResult.Emit {
		if !s.emitContentEventByName(name, event, ctx) {
			s.appendContentLog(event, fmt.Sprintf("[ContentEvent] Script emit target not found: %s", name))
		}
	}
}

func (s *GameService) scheduleContentEvents(scheduled []engine.ContentScheduledEvent, sourceEvent *engine.GameEvent, ctx *engine.EventContext) {
	if len(scheduled) == 0 || s.engine == nil {
		return
	}
	for _, sch := range scheduled {
		targetDef, ok := s.findContentDefinition(sch.EventID)
		if !ok {
			s.appendContentLog(sourceEvent, fmt.Sprintf("[ContentEvent] Scheduled target not found: %s", sch.EventID))
			continue
		}

		stepStories := []model.StoryEvent{}
		stepLogs := []string{}
		data := map[string]interface{}{
			"silo":            s.simSilo(),
			"agent":           s.simAgent(),
			"content_source":  "SCHEDULED",
			"step_story_sink": &stepStories,
			"step_log_sink":   &stepLogs,
		}
		event := engine.CreateEvent("scheduled#"+sch.EventID, engine.ContentEventBusName(targetDef), data)
		event.Data["step_stories"] = &stepStories
		event.Data["step_logs"] = &stepLogs
		s.scheduleDelayedEvent(event, sch.DelayMonths, sourceEvent)
		s.appendContentLog(sourceEvent, fmt.Sprintf("[ContentEvent] Scheduled event %s in %d months", sch.EventID, sch.DelayMonths))
	}
}

func (s *GameService) findContentDefinition(eventID string) (engine.ContentEventDefinition, bool) {
	for _, def := range s.contentDefinitions {
		if def.EventID == eventID || def.Key == eventID {
			return def, true
		}
	}
	return engine.ContentEventDefinition{}, false
}

func (s *GameService) emitContentEventByName(name string, sourceEvent *engine.GameEvent, ctx *engine.EventContext) bool {
	for _, def := range s.contentDefinitions {
		if engine.ContentEventBusName(def) != name {
			continue
		}
		stories, _ := sourceEvent.Data["story_sink"].(*[]model.StoryEvent)
		logs, _ := sourceEvent.Data["log_sink"].(*[]string)
		contentSource, _ := sourceEvent.Data["content_source"].(string)
		action := contentActionFromEvent(sourceEvent)
		return s.emitContentEvent(def, contentSource, ctx, action, stories, logs)
	}
	return false
}

func (s *GameService) appendContentLog(event *engine.GameEvent, message string) {
	if event == nil || message == "" {
		return
	}
	if sink, ok := event.Data["log_sink"].(*[]string); ok && sink != nil {
		*sink = append(*sink, message)
	}
}

func (s *GameService) tryEmitFollowupContentEvent(def engine.ContentEventDefinition, sourceEvent *engine.GameEvent, ctx *engine.EventContext) {
	simSilo := s.simSilo()
	state, _ := s.contentStateForDefinition(def)
	evalCtx := engine.ContentEvaluationContext{
		Silo:   simSilo,
		States: s.contentRuntimeMap(),
		Action: contentActionFromEvent(sourceEvent),
		Runtime: engine.ContentEventRuntime{
			Triggered:              state.Triggered,
			TriggerCount:           state.TriggerCount,
			LastTriggeredTimestamp: state.LastTriggeredTimestamp,
		},
	}
	if s.multiSilo() {
		evalCtx.World = s.world
		evalCtx.SignalsReceived = s.receivedSignals[simSilo.ID]
	}
	if !engine.CanTriggerContentEvent(def, evalCtx) {
		return
	}

	stories, _ := sourceEvent.Data["story_sink"].(*[]model.StoryEvent)
	logs, _ := sourceEvent.Data["log_sink"].(*[]string)
	contentSource, _ := sourceEvent.Data["content_source"].(string)
	action := contentActionFromEvent(sourceEvent)
	s.emitContentEvent(def, contentSource, ctx, action, stories, logs)
}

func contentActionFromEvent(event *engine.GameEvent) *model.AgentAction {
	if event == nil {
		return nil
	}
	action, _ := event.Data["action"].(*model.AgentAction)
	return action
}

func (s *GameService) endingNarrative() string {
	if !s.hasSession() {
		return ""
	}
	return s.engine.GetEndingNarrative(s.silo)
}

func (s *GameService) buildState() *model.GameState {
	gameOver := s.gameOver()
	return &model.GameState{
		Silo:              s.publicSiloSnapshot(),
		Agent:             *s.agent,
		AgentStats:        s.engine.BuildAgentStats(s.agent, s.playerActionSilo()),
		GameOver:          gameOver,
		EndingNarrative:   s.endingNarrative(),
		VictoryStatus:     s.silo.VictoryStatus,
		ProfessionActions: engine.GetProfessionActionMeta(s.agent.Profession),
		AvailableActions:  s.availablePlayerActions(),
		World:             s.publicWorldView(),
	}
}

// publicWorldView 世界层的公开快照。P0.5：列出世界内全部堡，非玩家堡以迷雾
// 视图 (RumorView) 下发——玩家只知道自己堡的真实值。单堡时仅含玩家堡、无迷雾。
func (s *GameService) publicWorldView() *model.WorldView {
	if s.world == nil || s.silo == nil {
		return nil
	}
	siloRefs := make([]model.WorldSiloRef, 0, len(s.silos))
	for _, silo := range s.silos {
		if silo == nil {
			continue
		}
		siloRefs = append(siloRefs, model.WorldSiloRef{
			ID:        silo.ID,
			Name:      silo.Name,
			IsPlayer:  silo.ID == s.world.PlayerSiloID,
			IsStation: s.agent != nil && silo.ID == s.agent.StationSiloID,
		})
	}
	if len(siloRefs) == 0 {
		siloRefs = []model.WorldSiloRef{
			{ID: s.silo.ID, Name: s.silo.Name, IsPlayer: s.silo.ID == s.world.PlayerSiloID},
		}
	}
	return &model.WorldView{
		ID:                       s.world.ID,
		Seed:                     s.world.Seed,
		CurrentYear:              s.world.CurrentYear,
		CurrentMonth:             s.world.CurrentMonth,
		PlayerSiloID:             s.world.PlayerSiloID,
		ProgressStage:            s.world.ProgressStage,
		Pacing:                   s.world.Pacing,
		Silo1Alive:               s.world.Silo1.Alive,
		Silo1Simulated:           s.fullSilo1() != nil,
		Silo1Departments:         s.silo1DepartmentNames(),
		Silo1LastReport:          s.world.Silo1.LastReport,
		Silo1Exposure:            s.world.Silo1.Exposure,
		Silo1CounterintelStage:   s.world.Silo1.CounterintelStage,
		Silo1Operations:          append([]model.Silo1Operation(nil), s.world.Silo1.Operations...),
		StationSiloID:            s.agent.StationSiloID,
		StationSilo:              s.stationSilo(),
		StationLastReport:        s.world.StationOps.LastReport,
		StationExposure:          s.world.StationOps.Exposure,
		StationCounterintelStage: s.world.StationOps.CounterintelStage,
		StationOperations:        append([]model.Silo1Operation(nil), s.world.StationOps.Operations...),
		Silos:                    siloRefs,
		Neighbors:                s.buildRumorViews(),
	}
}

func (s *GameService) pendingOperationName() string {
	if s.pendingOperation == nil {
		return ""
	}
	return s.pendingOperation.Kind
}

func (s *GameService) buildQueueState() *model.EventQueueState {
	return &model.EventQueueState{
		Silo:             s.publicSiloSnapshot(),
		Agent:            *s.agent,
		AgentStats:       s.engine.BuildAgentStats(s.agent, s.playerActionSilo()),
		AvailableActions: s.availablePlayerActions(),
		World:            s.publicWorldView(),
		GameOver:         s.gameOver(),
		EndingNarrative:  s.endingNarrative(),
		PendingEvents:    s.engine.Bus.Pending(),
		PendingOperation: s.pendingOperationName(),
	}
}

func (s *GameService) buildStepResult(processed *engine.GameEvent, logs []string, stories []model.StoryEvent, complete bool, actionResult *model.ActionResult) *model.EventStepResult {
	result := &model.EventStepResult{
		Silo:              s.publicSiloSnapshot(),
		Agent:             *s.agent,
		AgentStats:        s.engine.BuildAgentStats(s.agent, s.playerActionSilo()),
		AvailableActions:  s.availablePlayerActions(),
		World:             s.publicWorldView(),
		GameOver:          s.gameOver(),
		EndingNarrative:   s.endingNarrative(),
		PendingEvents:     s.engine.Bus.Pending(),
		PendingOperation:  s.pendingOperationName(),
		OperationComplete: complete,
		Logs:              logs,
		Stories:           stories,
	}
	if processed != nil {
		result.ProcessedEventID = processed.ID
		result.ProcessedEventType = processed.Type
	}
	result.ActionResult = actionResult
	return result
}

func (s *GameService) pendingActionResult() *model.ActionResult {
	if s.pendingOperation == nil || s.pendingOperation.ActionResult == nil {
		return nil
	}
	result := *s.pendingOperation.ActionResult
	return &result
}

func (s *GameService) actionDurationMonths(action model.AgentAction) int {
	if action.Type == model.ActionSilo1Operation || action.Type == model.ActionStationOperation {
		return 0
	}
	if action.Type == model.ActionPlayerEvent {
		if def, ok := s.playerActionDefinition(action.ActionID); ok && def.PlayerAction != nil {
			return def.PlayerAction.DurationMonths
		}
	}
	return model.ACTION_DURATIONS[action.Type]
}

func (s *GameService) playerActionDefinition(actionID string) (engine.ContentEventDefinition, bool) {
	matchID := strings.TrimSpace(actionID)
	if matchID == "" {
		return engine.ContentEventDefinition{}, false
	}
	for _, def := range s.contentDefinitions {
		if def.SourceGroup != "player_actions" && def.SourceGroup != "player_action" {
			continue
		}
		if def.PlayerAction == nil {
			continue
		}
		if engine.ContentPlayerActionID(def) == matchID {
			return def, true
		}
	}
	return engine.ContentEventDefinition{}, false
}

func validatePlayerActionTarget(def engine.ContentEventDefinition, action model.AgentAction) error {
	if def.PlayerAction == nil {
		return fmt.Errorf("player action metadata is missing")
	}
	switch def.PlayerAction.TargetType {
	case "DEPT":
		if strings.TrimSpace(action.TargetDept) == "" {
			return fmt.Errorf("Please select a target department.")
		}
	case "RESOURCE":
		if strings.TrimSpace(action.ResourceTarget) == "" {
			return fmt.Errorf("Please select a target resource.")
		}
	}
	return nil
}

func (s *GameService) availablePlayerActions() []model.PlayerActionMeta {
	if !s.hasSession() {
		return nil
	}
	stats := s.engine.BuildAgentStats(s.agent, s.playerActionSilo())
	actions := append(s.staticPlayerActions(stats), s.contentPlayerActions(stats)...)
	actions = append(actions, s.silo1OperationActions()...)
	actions = append(actions, s.stationOperationActions()...)
	s.refreshActionProgress()
	visible := make([]model.PlayerActionMeta, 0, len(actions))
	for _, action := range actions {
		if gated, ok := s.applyActionGate(action); ok {
			visible = append(visible, gated)
		}
	}
	actions = visible
	sort.Slice(actions, func(i, j int) bool {
		if actions[i].Group != actions[j].Group {
			return actions[i].Group < actions[j].Group
		}
		if actions[i].Enabled != actions[j].Enabled {
			return actions[i].Enabled && !actions[j].Enabled
		}
		if actions[i].APCost != actions[j].APCost {
			return actions[i].APCost < actions[j].APCost
		}
		return actions[i].Label < actions[j].Label
	})
	return actions
}

func (s *GameService) staticPlayerActions(stats model.AgentStats) []model.PlayerActionMeta {
	if s.engine == nil || s.engine.MechanicEngine == nil {
		// The mechanic engine is the single source of truth for action
		// metadata. Fail loudly instead of serving a stale hardcoded copy
		// (the removed fallback list drifted from the JS definitions).
		log.Printf("staticPlayerActions: mechanic engine unavailable; serving no static actions")
		return nil
	}
	var actions []model.PlayerActionMeta
	for _, def := range s.engine.MechanicEngine.CommonActionDefinitions() {
		actionType := model.AgentActionType(def.ActionType)
		group := "common"
		if actionType == model.ActionPublicizeFaction {
			if !stats.IsFactionLeader {
				continue
			}
			group = "faction_leader"
		}
		actions = append(actions, model.PlayerActionMeta{
			ID:             def.ID,
			Source:         "static",
			Label:          def.Label,
			Description:    def.Description,
			Group:          group,
			ActionType:     actionType,
			TargetType:     def.TargetType,
			APCost:         def.APCost,
			DurationMonths: def.DurationMonths,
			Enabled:        true,
		})
	}
	for _, def := range engine.GetProfessionActionMeta(s.agent.Profession) {
		actions = append(actions, model.PlayerActionMeta{
			ID:             def.ID,
			Source:         "static",
			Label:          def.Label,
			Description:    def.Description,
			Group:          "profession",
			ActionType:     model.ActionProfession,
			TargetType:     def.TargetType,
			APCost:         def.APCost,
			DurationMonths: model.ACTION_DURATIONS[model.ActionProfession],
			Enabled:        true,
		})
	}
	return actions
}

func (s *GameService) contentPlayerActions(stats model.AgentStats) []model.PlayerActionMeta {
	if len(s.contentDefinitions) == 0 {
		return nil
	}
	var actions []model.PlayerActionMeta
	for _, def := range s.contentDefinitions {
		if (def.SourceGroup != "player_actions" && def.SourceGroup != "player_action") || def.PlayerAction == nil {
			continue
		}
		if !s.matchesPlayerActionScope(def, stats) {
			continue
		}
		state, _ := s.contentStateForDefinition(def)
		ctx := engine.ContentEvaluationContext{
			Silo:   s.silo,
			States: s.contentRuntimeMap(),
			Runtime: engine.ContentEventRuntime{
				Triggered:              state.Triggered,
				TriggerCount:           state.TriggerCount,
				LastTriggeredTimestamp: state.LastTriggeredTimestamp,
			},
		}
		enabled := engine.CanDisplayPlayerAction(def, ctx)
		if !enabled && def.PlayerAction.UnavailableBehavior == "hide" {
			continue
		}
		actions = append(actions, model.PlayerActionMeta{
			ID:                  engine.ContentPlayerActionID(def),
			Source:              "content",
			EventID:             def.EventID,
			Label:               def.PlayerAction.Label,
			Description:         def.PlayerAction.Description,
			Group:               def.PlayerAction.Scope,
			ActionType:          model.AgentActionType(def.PlayerAction.ActionType),
			TargetType:          def.PlayerAction.TargetType,
			APCost:              def.PlayerAction.APCost,
			DurationMonths:      def.PlayerAction.DurationMonths,
			Enabled:             enabled,
			UnavailableBehavior: def.PlayerAction.UnavailableBehavior,
		})
	}
	return actions
}

func (s *GameService) matchesPlayerActionScope(def engine.ContentEventDefinition, stats model.AgentStats) bool {
	if def.PlayerAction == nil {
		return false
	}
	switch def.PlayerAction.Scope {
	case "common":
		return true
	case "profession":
		return def.PlayerAction.Profession == "" || strings.EqualFold(def.PlayerAction.Profession, s.agent.Profession)
	case "profession_group":
		return def.PlayerAction.ProfessionGroup != "" && def.PlayerAction.ProfessionGroup == s.agentProfessionGroup()
	case "faction_member":
		return false
	case "faction_leader":
		return stats.IsFactionLeader
	default:
		return false
	}
}

func (s *GameService) agentProfessionGroup() string {
	if !s.hasSession() || s.silo == nil {
		return ""
	}
	for _, profession := range s.playerActionSilo().Professions {
		if profession.Name == s.agent.Profession {
			return strings.ToUpper(strings.TrimSpace(profession.ClassType))
		}
	}
	return ""
}

func (s *GameService) recordStories(stories []*model.StoryEvent, source string) {
	simSilo := s.simSilo()
	if len(stories) == 0 || simSilo == nil {
		return
	}
	siloID := simSilo.ID
	if siloID == 0 {
		siloID = ActiveSiloID
	}
	for _, story := range stories {
		if story == nil {
			continue
		}
		entry := model.StoryEventLog{
			SiloID:      siloID,
			Timestamp:   gameTimestamp(simSilo.CurrentYear, simSilo.CurrentMonth),
			Year:        simSilo.CurrentYear,
			Month:       simSilo.CurrentMonth,
			Source:      source,
			EventID:     story.ID,
			Category:    story.Category,
			Title:       story.Title,
			Description: story.Description,
			Type:        story.Type,
		}
		s.eventLogs = append(s.eventLogs, entry)
		// 世界 tick 下为玩家堡分流本月日志 (LLM gating 输入)。
		if s.multiSilo() && siloID == s.world.PlayerSiloID {
			s.monthPlayerLogs = append(s.monthPlayerLogs, entry)
		}
	}
}

func gameTimestamp(year, month int) int {
	return ((year-timestampBaseYear)*monthsPerYear + (month - 1)) * daysPerMonth
}

func cloneGameEvent(event *engine.GameEvent) *engine.GameEvent {
	if event == nil {
		return nil
	}
	cloned := &engine.GameEvent{
		ID:          event.ID,
		Type:        event.Type,
		Source:      event.Source,
		Target:      event.Target,
		TriggerTime: event.TriggerTime,
	}
	if event.Data != nil {
		cloned.Data = make(map[string]interface{}, len(event.Data))
		for key, value := range event.Data {
			cloned.Data[key] = value
		}
	}
	return cloned
}

func countQueuedEventType(events []*engine.GameEvent, typ string) int {
	count := 0
	for _, event := range events {
		if event != nil && event.Type == typ {
			count++
		}
	}
	return count
}

func indexAfterNthEventType(events []*engine.GameEvent, typ string, n int) (int, string) {
	if n <= 0 {
		return 0, ""
	}
	seen := 0
	for idx, event := range events {
		if event == nil || event.Type != typ {
			continue
		}
		seen++
		if seen == n {
			return idx + 1, event.ID
		}
	}
	return -1, ""
}

func advancePastAnchoredDelayEvents(events []*engine.GameEvent, index int, anchorID string) int {
	if anchorID == "" {
		return index
	}
	for index < len(events) {
		event := events[index]
		if event == nil || event.Data == nil {
			break
		}
		nextAnchorID, _ := event.Data["delay_anchor_event_id"].(string)
		if nextAnchorID != anchorID {
			break
		}
		index++
	}
	return index
}

func (s *GameService) eventHistory(limit int) []model.StoryEventLog {
	if len(s.eventLogs) == 0 {
		return nil
	}
	max := len(s.eventLogs)
	if limit > 0 && limit < max {
		max = limit
	}
	events := make([]model.StoryEventLog, 0, max)
	for i := len(s.eventLogs) - 1; i >= 0 && len(events) < max; i-- {
		events = append(events, s.eventLogs[i])
	}
	return events
}

func (s *GameService) publicSiloSnapshot() model.Silo {
	if s.silo == nil {
		return model.Silo{}
	}
	snap := *s.silo

	// 找出“无阵营”的 ID
	var unaffiliatedID uint
	for _, f := range s.silo.Factions {
		if f.Signature == "special:unaffiliated" {
			unaffiliatedID = f.ID
			break
		}
	}

	// 转换特工关系为 map 以便复用引擎可见性逻辑
	agentRelations := make(map[string]float64)
	var agentProfID uint
	for _, p := range s.silo.Professions {
		if p.Name == s.agent.Profession {
			agentProfID = p.ID
		}
		for _, conn := range s.agent.Connections {
			if p.ID == conn.ProfessionID {
				agentRelations[p.Name] = conn.Value
			}
		}
	}

	// 阵营可见性过滤
	visibleFactions := make([]model.Faction, 0, len(s.silo.Factions))
	hiddenFactionIDs := make(map[uint]bool)

	for _, f := range s.silo.Factions {
		level := engine.GetFactionVisibilityLevel(s.silo, agentProfID, agentRelations, &f)
		switch level {
		case engine.FactionVisibilityVisible:
			visibleFactions = append(visibleFactions, f)
		case engine.FactionVisibilityAware:
			visibleFactions = append(visibleFactions, model.Faction{
				ID:        f.ID,
				SiloID:    f.SiloID,
				Name:      "Unknown Faction",
				Signature: "unknown",
				IsPublic:  false,
				TagStats:  make(map[string]int),
				Tags:      []string{"status:unknown"},
			})
		case engine.FactionVisibilityHidden:
			hiddenFactionIDs[f.ID] = true
		}
	}
	snap.Factions = visibleFactions

	// 同步人口单元（Cohort）的归属，并修正“无阵营”的人数统计
	// 如果所属阵营被隐藏，则在前端显示为“无阵营”
	if len(hiddenFactionIDs) > 0 {
		hiddenMembersCount := 0
		snap.Cohorts = make([]model.PopulationCohort, len(s.silo.Cohorts))
		for i, c := range s.silo.Cohorts {
			snap.Cohorts[i] = c
			if c.FactionID != nil && hiddenFactionIDs[*c.FactionID] {
				hiddenMembersCount += c.Count
				if unaffiliatedID > 0 {
					snap.Cohorts[i].FactionID = &unaffiliatedID
				} else {
					snap.Cohorts[i].FactionID = nil
				}
			}
		}

		// 修正“无阵营”显示的总人数
		if hiddenMembersCount > 0 && unaffiliatedID > 0 {
			for i := range snap.Factions {
				if snap.Factions[i].ID == unaffiliatedID {
					snap.Factions[i].MemberCount += hiddenMembersCount
					break
				}
			}
		}
	}

	// Residents stay in the backend/session snapshot for simulation, but we avoid
	// returning the full resident list on every UI round-trip.
	snap.Residents = nil
	return snap
}
