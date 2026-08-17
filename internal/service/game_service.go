package service

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
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
	ActiveSiloID      = uint(1)
	ActiveAgentID     = uint(1)
	sessionKey        = "game_session_v1"
	timestampBaseYear = 100
	daysPerMonth      = 30
	monthsPerYear     = 12
)

type GameService struct {
	db                  *gorm.DB
	mu                  sync.Mutex
	engine              *engine.GameEngine
	silo                *model.Silo
	agent               *model.Agent
	eventLogs           []model.StoryEventLog
	contentDefinitions  []engine.ContentEventDefinition
	contentStates       map[string]model.ContentEventState
	contentUnsubscribes []func()
	delayCounter        int
	delayTasks          map[string]*delayTask
	pendingOperation    *pendingOperation
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
}

func NewGameService(db *gorm.DB) *GameService {
	svc := &GameService{
		db:            db,
		engine:        engine.NewGameEngine(),
		contentStates: map[string]model.ContentEventState{},
		delayTasks:    map[string]*delayTask{},
	}
	svc.bindEngineRuntime()
	return svc
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
	if err := s.loadContentDefinitions(); err != nil {
		return err
	}
	s.bindContentEventHandlers()
	return nil
}

func (s *GameService) ensureEventsExtracted(events embed.FS) error {
	exePath, _ := os.Executable()
	if exePath == "" {
		return nil // Fallback to embedded if executable path is unknown? No, user wants extraction.
	}
	exeDir := filepath.Dir(exePath)
	targetDir := filepath.Join(exeDir, "events")

	// 检查同目录有没有 events 目录
	if _, err := os.Stat(targetDir); err == nil {
		return nil // Already exists
	}

	// 没有则释放
	return fs.WalkDir(events, "events", func(path string, d fs.DirEntry, err error) error {
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
	})
}

// ---------- 内部辅助 ----------

// sessionState 缓存快照结构
type sessionState struct {
	Silo          model.Silo                `json:"silo"`
	Agent         model.Agent               `json:"agent"`
	EventLogs     []model.StoryEventLog     `json:"event_logs"`
	ContentStates []model.ContentEventState `json:"content_states"`
}

func (s *GameService) hasSession() bool {
	return s.silo != nil && s.agent != nil
}

func (s *GameService) gameOver() bool {
	return s.hasSession() && s.silo.VictoryStatus != nil
}

func (s *GameService) cacheSnapshot() {
	if cache.GlobalCache == nil || !s.hasSession() {
		return
	}
	contentStates := s.contentStateSlice()
	data, err := json.Marshal(sessionState{
		Silo:          *s.silo,
		Agent:         *s.agent,
		EventLogs:     s.eventLogs,
		ContentStates: contentStates,
	})
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
	eventLogs := make([]model.StoryEventLog, len(s.eventLogs))
	copy(eventLogs, s.eventLogs)
	contentStates := s.contentStateSlice()

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
	for i := range silo.Cohorts {
		silo.Cohorts[i].SiloID = ActiveSiloID
		if silo.Cohorts[i].ID == 0 {
			silo.Cohorts[i].ID = uint(i + 1)
		}
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
	for i := range eventLogs {
		eventLogs[i].ID = 0
		eventLogs[i].SiloID = ActiveSiloID
	}
	for i := range contentStates {
		contentStates[i].ID = 0
		contentStates[i].SiloID = ActiveSiloID
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

		var oldCohorts []model.PopulationCohort
		_ = tx.Where("silo_id = ?", ActiveSiloID).Find(&oldCohorts).Error
		for _, cohort := range oldCohorts {
			if err := tx.Delete(&model.PopulationCohort{}, cohort.ID).Error; err != nil {
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

		var oldEventLogs []model.StoryEventLog
		_ = tx.Where("silo_id = ?", ActiveSiloID).Find(&oldEventLogs).Error
		for _, eventLog := range oldEventLogs {
			if err := tx.Delete(&model.StoryEventLog{}, eventLog.ID).Error; err != nil {
				return err
			}
		}

		var oldContentStates []model.ContentEventState
		_ = tx.Where("silo_id = ?", ActiveSiloID).Find(&oldContentStates).Error
		for _, state := range oldContentStates {
			if err := tx.Delete(&model.ContentEventState{}, state.ID).Error; err != nil {
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
		for i := range silo.Cohorts {
			silo.Cohorts[i].SiloID = ActiveSiloID
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
		if err := tx.Create(&silo.Cohorts).Error; err != nil {
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
			if err := tx.Create(&agent.Connections).Error; err != nil {
				return err
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
				s.eventLogs = snap.EventLogs
				s.contentStates = map[string]model.ContentEventState{}
				for _, state := range snap.ContentStates {
					s.contentStates[state.EventKey] = state
				}
				s.engine = engine.NewGameEngine()
				s.pendingOperation = nil
				s.delayTasks = map[string]*delayTask{}
				s.bindEngineRuntime()
				return nil
			}
		}
	}

	var silo model.Silo
	if err := s.db.Preload("Resources").Preload("Professions").Preload("Floors").Preload("Cohorts").Preload("Residents").Preload("Factions").
		First(&silo, ActiveSiloID).Error; err != nil {
		return err
	}
	var agent model.Agent
	if err := s.db.Preload("Connections").First(&agent, ActiveAgentID).Error; err != nil {
		return err
	}
	var eventLogs []model.StoryEventLog
	if err := s.db.Order("id asc").Where("silo_id = ?", ActiveSiloID).Find(&eventLogs).Error; err != nil {
		return err
	}
	var contentStates []model.ContentEventState
	if err := s.db.Where("silo_id = ?", ActiveSiloID).Find(&contentStates).Error; err != nil {
		return err
	}
	s.silo = &silo
	s.agent = &agent
	s.eventLogs = eventLogs
	s.contentStates = map[string]model.ContentEventState{}
	for _, state := range contentStates {
		s.contentStates[state.EventKey] = state
	}
	s.engine = engine.NewGameEngine()
	s.pendingOperation = nil
	s.delayTasks = map[string]*delayTask{}
	s.bindEngineRuntime()
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
	s.bindEngineRuntime()
	s.silo = silo
	s.agent = agent
	s.eventLogs = nil
	s.contentStates = map[string]model.ContentEventState{}
	s.delayTasks = map[string]*delayTask{}
	s.pendingOperation = nil

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
	s.bindContentEventHandlers()
}

func (s *GameService) seedPendingOperation() {
	if s.pendingOperation == nil || s.engine.Bus.Pending() > 0 {
		return
	}
	op := s.pendingOperation
	switch op.Phase {
	case phaseActionRoot:
		actor := engine.CreateActorRefForAgent(s.agent, s.silo)
		s.engine.EnqueueAction(actor, s.silo, *op.Action, s.agent)
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
		if op.ActionDuration > 0 {
			s.enqueueMonthChains(op.ActionDuration, "ACTION:"+string(op.Action.Type)+":CONTENT")
			op.Phase = phaseQueued
			return
		}
		s.enqueueActionCompletion("ACTION:" + string(op.Action.Type) + ":CONTENT")
		op.Phase = phaseQueued
	case phaseQueued:
		s.pendingOperation = nil
	}
}

func (s *GameService) enqueueMonthChains(months int, source string) {
	if s.engine == nil || months <= 0 {
		return
	}
	for i := 0; i < months; i++ {
		s.engine.EnqueueMonthChain(s.silo, s.agent, source)
	}
}

func (s *GameService) enqueueActionCompletion(source string) {
	if s.engine == nil {
		return
	}
	s.engine.EnqueueInstantResolution(s.silo, s.agent)
	s.engine.Bus.Publish(s.engine.CreateTimeContentEvent(s.silo, s.agent, source))
}

func (s *GameService) nextDelayTaskID() string {
	s.delayCounter++
	return "delay#" + strconv.Itoa(s.delayCounter)
}

func (s *GameService) scheduleDelayedEvent(event *engine.GameEvent, delayMonths int, source *engine.GameEvent) {
	if s.engine == nil || s.engine.Bus == nil || event == nil || delayMonths <= 0 {
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
	}

	snapshot := s.engine.Bus.Snapshot()
	advanceCount := countQueuedEventType(snapshot, engine.EVENT_TIME_ADVANCE)
	if advanceCount < delayMonths {
		s.enqueueMonthChains(delayMonths-advanceCount, "DELAY:CONTENT")
		snapshot = s.engine.Bus.Snapshot()
	}

	insertIndex, anchorID := indexAfterNthEventType(snapshot, engine.EVENT_TIME_ADVANCE, delayMonths)
	if insertIndex < 0 {
		insertIndex = len(snapshot)
	}
	cloned.Data["delay_anchor_event_id"] = anchorID
	insertIndex = advancePastAnchoredDelayEvents(snapshot, insertIndex, anchorID)
	s.engine.Bus.InsertAt(insertIndex, cloned)
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
	s.enqueueMonthChains(months, "PASS_TIME:CONTENT")
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
		if op.ActionDuration > 0 {
			op.Phase = phaseQueued
			s.enqueueMonthChains(op.ActionDuration, "ACTION:"+string(action.Type)+":CONTENT")
		} else {
			op.Phase = phaseQueued
			s.enqueueActionCompletion("ACTION:" + string(action.Type) + ":CONTENT")
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
	processed, logs, stories := s.engine.Step(s.silo, s.agent)
	if processed == nil {
		return nil, fmt.Errorf("no pending event")
	}

	stepLogs := append([]string(nil), logs...)
	stepStories := make([]model.StoryEvent, 0, len(stories))
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
	s.advancePendingOperationAfterDrain()
	s.seedPendingOperation()
	complete := s.pendingOperation == nil && s.engine.Bus.Pending() == 0

	if s.gameOver() {
		if err := s.persist(); err != nil {
			return nil, err
		}
	}
	s.cacheSnapshot()
	return s.buildStepResult(processed, stepLogs, stepStories, complete), nil
}

// PassTime 推进 months 个月，返回 tick 结果
func (s *GameService) PassTime(months int) (*model.TickResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSession() {
		return nil, fmt.Errorf("no active game session")
	}

	s.enqueueMonthChains(months, "PASS_TIME:CONTENT")
	logs, stories := s.drainQueuedEvents()

	// 关键节点：游戏结束 → 落库
	if s.gameOver() {
		if err := s.persist(); err != nil {
			return nil, err
		}
	}
	s.cacheSnapshot()

	return &model.TickResult{
		Silo:             s.publicSiloSnapshot(),
		Agent:            *s.agent,
		AgentStats:       s.engine.BuildAgentStats(s.agent, s.silo),
		AvailableActions: s.availablePlayerActions(),
		Logs:             logs,
		Stories:          stories,
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

	var logs []string
	var stories []model.StoryEvent
	handledByContent, result, playerActionLogs, playerActionStories := s.executePlayerActionEvents(action)
	logs = append(logs, playerActionLogs...)
	stories = append(stories, playerActionStories...)
	if !handledByContent {
		actor := engine.CreateActorRefForAgent(s.agent, s.silo)
		s.engine.EnqueueAction(actor, s.silo, action, s.agent)
		actionLogs, actionStories := s.drainQueuedEvents()
		logs = append(logs, actionLogs...)
		stories = append(stories, actionStories...)
		if consumed := s.engine.ConsumeActionResult(); consumed != nil {
			result = *consumed
		} else {
			result = model.ActionResult{Executed: false, Message: "Unknown error."}
		}
	}

	if result.Executed {
		duration := s.actionDurationMonths(action)
		if duration > 0 {
			s.enqueueMonthChains(duration, "ACTION:"+string(action.Type)+":CONTENT")
			moreLogs, moreStories := s.drainQueuedEvents()
			logs = append(logs, moreLogs...)
			stories = append(stories, moreStories...)
		} else {
			s.enqueueActionCompletion("ACTION:" + string(action.Type) + ":CONTENT")
			moreLogs, moreStories := s.drainQueuedEvents()
			logs = append(logs, moreLogs...)
			stories = append(stories, moreStories...)
		}
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
		AgentStats:       s.engine.BuildAgentStats(s.agent, s.silo),
		AvailableActions: s.availablePlayerActions(),
		Result:           result,
		Logs:             logs,
		Stories:          stories,
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
		case "histories", "history", "special", "crisis", "player_actions", "player_action", "events":
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

		// Backward compatibility for the legacy flat layout.
		hasEventFiles, err := directoryHasContentEventFiles(root)
		if err == nil && hasEventFiles && dirs["events"] == "" {
			dirs["events"] = root
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
		if entry.IsDir() {
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

func (s *GameService) contentStateSlice() []model.ContentEventState {
	if len(s.contentStates) == 0 {
		return nil
	}
	states := make([]model.ContentEventState, 0, len(s.contentStates))
	for _, state := range s.contentStates {
		states = append(states, state)
	}
	sort.Slice(states, func(i, j int) bool {
		return states[i].EventKey < states[j].EventKey
	})
	return states
}

func (s *GameService) queueSystemContentEvents(source string) int {
	count, _, _ := s.queueContentEvents(source, nil, func(def engine.ContentEventDefinition) bool {
		return def.SourceGroup != "player_actions" &&
			def.SourceGroup != "player_action" &&
			!engine.HasEventTriggeredTrigger(def.Trigger)
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

func (s *GameService) queuePlayerActionEvents(action model.AgentAction) (bool, model.ActionResult, error) {
	if !s.hasSession() {
		return false, model.ActionResult{}, fmt.Errorf("no active game session")
	}
	if action.Type == model.ActionPlayerEvent {
		def, ok := s.playerActionDefinition(action.ActionID)
		if !ok {
			return true, model.ActionResult{Executed: false, Message: "Unknown player action."}, nil
		}
		if err := validatePlayerActionTarget(def, action); err != nil {
			return true, model.ActionResult{Executed: false, Message: err.Error()}, nil
		}
		action.Cost = float64(def.PlayerAction.APCost)
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
	count := 0
	for _, def := range s.contentDefinitions {
		if allow != nil && !allow(def) {
			continue
		}
		state, _ := s.contentStateForDefinition(def)
		ctx := engine.ContentEvaluationContext{
			Silo:   s.silo,
			States: s.contentRuntimeMap(),
			Action: action,
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
		processed, stepLogs, stepStories := s.engine.Step(s.silo, s.agent)
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
	if len(s.contentStates) == 0 {
		return nil
	}
	runtimes := make(map[string]engine.ContentEventRuntime, len(s.contentStates))
	for key, state := range s.contentStates {
		runtimes[key] = engine.ContentEventRuntime{
			Triggered:              state.Triggered,
			TriggerCount:           state.TriggerCount,
			LastTriggeredTimestamp: state.LastTriggeredTimestamp,
		}
	}
	return runtimes
}

func (s *GameService) contentStateForDefinition(def engine.ContentEventDefinition) (model.ContentEventState, string) {
	if state, ok := s.contentStates[def.Key]; ok {
		return state, def.Key
	}
	for key, state := range s.contentStates {
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
		"silo":            s.silo,
		"agent":           s.agent,
		"content_source":  source,
		"story_sink":      stories,
		"log_sink":        logs,
		"step_story_sink": &stepStories,
		"step_log_sink":   &stepLogs,
	}
	if action != nil {
		data["action"] = action
	}
	event := engine.CreateEvent("content#"+def.Key+"#"+strconv.Itoa(len(s.eventLogs)+len(*stories)+1), engine.ContentEventBusName(def), data)
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
	story, scheduled := engine.ApplyContentEvent(def, s.silo)

	state, legacyKey := s.contentStateForDefinition(def)
	state.SiloID = ActiveSiloID
	state.DefinitionID = def.ID
	state.EventKey = def.Key
	state.Triggered = true
	state.TriggerCount++
	state.LastTriggeredTimestamp = gameTimestamp(s.silo.CurrentYear, s.silo.CurrentMonth)
	if legacyKey != "" && legacyKey != def.Key {
		delete(s.contentStates, legacyKey)
	}
	s.contentStates[def.Key] = state

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
		Silo:   s.silo,
		States: s.contentRuntimeMap(),
		Action: contentActionFromEvent(event),
		Runtime: engine.ContentEventRuntime{
			Triggered:              state.Triggered,
			TriggerCount:           state.TriggerCount,
			LastTriggeredTimestamp: state.LastTriggeredTimestamp,
		},
	}
	scriptResult, err := engine.RunContentEventApplyScript(def, evalCtx)
	if err != nil {
		s.appendContentLog(event, fmt.Sprintf("[ContentEvent] Script apply failed for %s: %v", def.Key, err))
		return
	}
	scheduledFromScript := engine.ApplyContentEffects(scriptResult.Effects, s.silo)
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
			"silo":            s.silo,
			"agent":           s.agent,
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
	state, _ := s.contentStateForDefinition(def)
	evalCtx := engine.ContentEvaluationContext{
		Silo:   s.silo,
		States: s.contentRuntimeMap(),
		Action: contentActionFromEvent(sourceEvent),
		Runtime: engine.ContentEventRuntime{
			Triggered:              state.Triggered,
			TriggerCount:           state.TriggerCount,
			LastTriggeredTimestamp: state.LastTriggeredTimestamp,
		},
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
		AgentStats:        s.engine.BuildAgentStats(s.agent, s.silo),
		GameOver:          gameOver,
		EndingNarrative:   s.endingNarrative(),
		VictoryStatus:     s.silo.VictoryStatus,
		ProfessionActions: engine.GetProfessionActionMeta(s.agent.Profession),
		AvailableActions:  s.availablePlayerActions(),
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
		AgentStats:       s.engine.BuildAgentStats(s.agent, s.silo),
		AvailableActions: s.availablePlayerActions(),
		GameOver:         s.gameOver(),
		EndingNarrative:  s.endingNarrative(),
		PendingEvents:    s.engine.Bus.Pending(),
		PendingOperation: s.pendingOperationName(),
	}
}

func (s *GameService) buildStepResult(processed *engine.GameEvent, logs []string, stories []model.StoryEvent, complete bool) *model.EventStepResult {
	result := &model.EventStepResult{
		Silo:              s.publicSiloSnapshot(),
		Agent:             *s.agent,
		AgentStats:        s.engine.BuildAgentStats(s.agent, s.silo),
		AvailableActions:  s.availablePlayerActions(),
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
	if s.pendingOperation != nil && s.pendingOperation.ActionResult != nil {
		actionResult := *s.pendingOperation.ActionResult
		result.ActionResult = &actionResult
	}
	return result
}

func (s *GameService) actionDurationMonths(action model.AgentAction) int {
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
	stats := s.engine.BuildAgentStats(s.agent, s.silo)
	actions := append(s.staticPlayerActions(stats), s.contentPlayerActions(stats)...)
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
	var actions []model.PlayerActionMeta
	if s.engine != nil && s.engine.MechanicEngine != nil {
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
	} else {
		actions = []model.PlayerActionMeta{
			{ID: string(model.ActionGatherInfo), Source: "static", Label: "Gather Info", Description: "Collect information from a department.", Group: "common", ActionType: model.ActionGatherInfo, TargetType: "DEPT", APCost: int(model.ACTION_COSTS[model.ActionGatherInfo]), DurationMonths: model.ACTION_DURATIONS[model.ActionGatherInfo], Enabled: true},
			{ID: string(model.ActionShareInfo), Source: "static", Label: "Share Info", Description: "Share real or fake fragments with a department.", Group: "common", ActionType: model.ActionShareInfo, TargetType: "DEPT", APCost: int(model.ACTION_COSTS[model.ActionShareInfo]), DurationMonths: model.ACTION_DURATIONS[model.ActionShareInfo], Enabled: true},
			{ID: string(model.ActionBuildConnection), Source: "static", Label: "Build Network", Description: "Build influence with a department over time.", Group: "common", ActionType: model.ActionBuildConnection, TargetType: "DEPT", APCost: int(model.ACTION_COSTS[model.ActionBuildConnection]), DurationMonths: model.ACTION_DURATIONS[model.ActionBuildConnection], Enabled: true},
			{ID: string(model.ActionConductPropaganda), Source: "static", Label: "Propaganda", Description: "Increase your propaganda level.", Group: "common", ActionType: model.ActionConductPropaganda, TargetType: "NONE", APCost: int(model.ACTION_COSTS[model.ActionConductPropaganda]), DurationMonths: model.ACTION_DURATIONS[model.ActionConductPropaganda], Enabled: true},
			{ID: string(model.ActionInciteRebellion), Source: "static", Label: "Incite Rebellion", Description: "Agitate commoner departments.", Group: "common", ActionType: model.ActionInciteRebellion, TargetType: "NONE", APCost: int(model.ACTION_COSTS[model.ActionInciteRebellion]), DurationMonths: model.ACTION_DURATIONS[model.ActionInciteRebellion], Enabled: true},
		}
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
	if stats.IsFactionLeader && (s.engine == nil || s.engine.MechanicEngine == nil) {
		actions = append(actions, model.PlayerActionMeta{
			ID:             string(model.ActionPublicizeFaction),
			Source:         "static",
			Label:          "Publicize Faction",
			Description:    "Reveal your faction to the silo and gain prestige.",
			Group:          "faction_leader",
			ActionType:     model.ActionPublicizeFaction,
			TargetType:     "NONE",
			APCost:         int(model.ACTION_COSTS[model.ActionPublicizeFaction]),
			DurationMonths: model.ACTION_DURATIONS[model.ActionPublicizeFaction],
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
	for _, profession := range s.silo.Professions {
		if profession.Name == s.agent.Profession {
			return strings.ToUpper(strings.TrimSpace(profession.ClassType))
		}
	}
	return ""
}

func (s *GameService) recordStories(stories []*model.StoryEvent, source string) {
	if len(stories) == 0 || s.silo == nil {
		return
	}
	for _, story := range stories {
		if story == nil {
			continue
		}
		s.eventLogs = append(s.eventLogs, model.StoryEventLog{
			SiloID:      ActiveSiloID,
			Timestamp:   gameTimestamp(s.silo.CurrentYear, s.silo.CurrentMonth),
			Year:        s.silo.CurrentYear,
			Month:       s.silo.CurrentMonth,
			Source:      source,
			EventID:     story.ID,
			Category:    story.Category,
			Title:       story.Title,
			Description: story.Description,
			Type:        story.Type,
		})
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
