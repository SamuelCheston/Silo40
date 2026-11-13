package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"silo40/internal/api"
	"silo40/internal/cache"
	"silo40/internal/engine"
	"silo40/internal/llm"
	"silo40/internal/model"
	"silo40/internal/repository"
	"silo40/internal/service"
	"time"

	"gorm.io/gorm"
)

// dbFileName 存档数据库文件名（工作目录下）。
const dbFileName = "silo40.db"

// App struct
type App struct {
	ctx          context.Context
	db           *gorm.DB
	gameService  *service.GameService
	debugHTTPAPI *api.DebugHTTPServer
	eventsFS     embed.FS
	ready        chan struct{}
}

// NewApp creates a new App application struct
func NewApp(events embed.FS) *App {
	return &App{
		eventsFS: events,
		ready:    make(chan struct{}),
	}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	if a.ready != nil {
		defer close(a.ready)
	}
	a.ctx = ctx

	// Initialize Cache (100MB)
	cacheSize := 100 * 1024 * 1024
	cache.InitCache(cacheSize)
	log.Println("FreeCache initialized")

	// Initialize SQLite DB
	db, err := repository.InitDB(dbFileName)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	a.db = db

	// 无版本标记的旧档：启动时一次性显式升级到 v1（先做带时间戳备份，
	// 事务内补写标记并留痕）。失败则库保持原样、备份留存，直接终止启动。
	if _, err := service.UpgradeLegacySave(a.db, dbFileName); err != nil {
		log.Fatalf("legacy save upgrade failed: %v", err)
	}
	// v1 单堡档 → v2 世界模型：接力升级（同样的备份+事务+回读校验语义）。
	if _, err := service.UpgradeV1ToWorldSave(a.db, dbFileName); err != nil {
		log.Fatalf("v1 world upgrade failed: %v", err)
	}
	// v2 世界模型 → v3 双地堡：接力升级（同样的备份+事务+回读校验语义）。
	if _, err := service.UpgradeV2ToTwoSiloSave(a.db, dbFileName); err != nil {
		log.Fatalf("v2 two-silo upgrade failed: %v", err)
	}
	// v3 双地堡 → v4 派系持久身份：接力升级（同样的备份+事务+回读校验语义）。
	if _, err := service.UpgradeV3ToFactionIdentitySave(a.db, dbFileName); err != nil {
		log.Fatalf("v3 faction identity upgrade failed: %v", err)
	}
	// The legacy chain stops at v5. Resume then rejects that incompatible
	// identity model explicitly; upgrading fields never implies v6 compatibility.
	if _, err := service.UpgradeV4ToPlayableWorldSave(a.db, dbFileName); err != nil {
		log.Fatalf("v4 playable world upgrade failed: %v", err)
	}
	// v7 progression saves gain the independent authority state. v5/v6 remain
	// explicitly incompatible and therefore never enter this migration.
	if _, err := service.UpgradeV7ToAuthoritySave(a.db, dbFileName); err != nil {
		log.Fatalf("v7 authority upgrade failed: %v", err)
	}

	// 有状态游戏会话服务 (后端为唯一事实来源)
	a.gameService = service.NewGameService(db)
	if err := a.gameService.BootstrapContent(a.eventsFS); err != nil {
		log.Printf("failed to bootstrap content events: %v", err)
	}
	if err := a.gameService.Resume(); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Println("first start: no saved game found; starting fresh is normal")
		} else {
			log.Printf("saved game resume failed: %v", err)
		}
	}

	a.debugHTTPAPI = api.NewDebugHTTPServer(os.Getenv("SILO40_DEBUG_HTTP_ADDR"), a)
	a.debugHTTPAPI.SetLlmAPI(llmAPIAdapter{app: a})
	a.debugHTTPAPI.SetAgentAPI(agentAPIAdapter{app: a})
	if err := a.debugHTTPAPI.Start(); err != nil {
		log.Printf("failed to start debug HTTP server: %v", err)
	}
}

// shutdown is called when the app is closing.
func (a *App) shutdown(ctx context.Context) {
	if a.debugHTTPAPI == nil {
		return
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := a.debugHTTPAPI.Shutdown(shutdownCtx); err != nil {
		log.Printf("failed to stop debug HTTP server: %v", err)
	}
}

// ============ 游戏 API (后端驱动：数据存储 + 计算逻辑均在 Go) ============

// CreateGame 新建游戏：初始化地堡/特工 → 落库 SQLite → 缓存会话
func (a *App) CreateGame(req model.CreateGameRequest) (*model.GameState, error) {
	return a.gameService.CreateGame(req)
}

// GetGameState 获取当前游戏状态快照
func (a *App) GetGameState() (*model.GameState, error) {
	return a.gameService.GetState()
}

// SaveGame persists the current settled turn so it can be resumed after restart.
func (a *App) SaveGame() error {
	return a.gameService.SaveGame()
}

// GetEventHistory 获取事件历史，供 debug 前端或工具查询
func (a *App) GetEventHistory(limit int) (*model.EventHistoryResult, error) {
	return a.gameService.GetEventHistory(limit)
}

// PassTime 推进时间 months 个月 (tick 结算全部在 Go 完成)
func (a *App) PassTime(months int) (*model.TickResult, error) {
	return a.gameService.PassTime(months)
}

// BeginPassTime 入队一个时间推进操作。
func (a *App) BeginPassTime(months int) (*model.EventQueueState, error) {
	return a.gameService.BeginPassTime(months)
}

// ExecuteAction 执行玩家动作；耗时动作自动推进对应月份
func (a *App) ExecuteAction(action model.AgentAction) (*model.ActionOutcome, error) {
	return a.gameService.ExecuteAction(action)
}

// BeginAction 入队一个玩家动作。
func (a *App) BeginAction(action model.AgentAction) (*model.EventQueueState, error) {
	return a.gameService.BeginAction(action)
}

// ProcessNextEvent 处理队列中的下一个事件。
func (a *App) ProcessNextEvent() (*model.EventStepResult, error) {
	return a.gameService.ProcessNextEvent()
}

// GetEndingNarrative 获取结局叙事文案
func (a *App) GetEndingNarrative() (string, error) {
	return a.gameService.GetEndingNarrative(), nil
}

// HasActiveGame 是否有进行中的游戏会话
func (a *App) HasActiveGame() (bool, error) {
	if a.gameService == nil && a.ready != nil {
		select {
		case <-a.ready:
		case <-time.After(5 * time.Second):
			return false, fmt.Errorf("game service initialization timed out")
		}
	}
	if a.gameService == nil {
		return false, nil
	}
	return a.gameService.HasActiveGame(), nil
}

// GetAllFragments 返回全部信息碎片清单 (前端渲染共享选择器用；权威来源为后端 model，
// 避免前端维护静态副本)
func (a *App) GetAllFragments() []string {
	return model.ALL_FRAGMENTS
}

// GetProfessionActions 获取职业专属行动元数据 (前端渲染按钮用，不含执行逻辑)
func (a *App) GetProfessionActions(profession string) ([]model.ProfessionActionMeta, error) {
	return engine.GetProfessionActionMeta(profession), nil
}

// GetProfessionAction 获取单个职业行动元数据
func (a *App) GetProfessionAction(id string) (*model.ProfessionActionMeta, error) {
	def := engine.GetProfessionAction(id)
	if def == nil {
		return nil, nil
	}
	return &model.ProfessionActionMeta{
		ID:               def.ID,
		Profession:       def.Profession,
		Label:            def.Label,
		Description:      def.Description,
		APCost:           int(def.APCost),
		TargetType:       string(def.TargetType),
		SuspicionPenalty: def.SuspicionPenalty,
	}, nil
}

// ============ LLM 辅助 NPC 决策 (可选，默认关闭) ============

// GetLlmStatus LLM 顾问状态 (模式/参与度/日志规模/错误)。
func (a *App) GetLlmStatus() service.LlmStatus {
	return a.gameService.GetLlmStatus()
}

// GetLlmDecisions 决策审计日志 (新→旧；limit<=0 全部)。
func (a *App) GetLlmDecisions(limit int) ([]llm.DecisionLog, error) {
	return a.gameService.GetLlmDecisions(limit)
}

// SetLlmParticipation 中途切换参与度档位 (off|events|key|full)，切换记入日志。
func (a *App) SetLlmParticipation(level string) (string, error) {
	return a.gameService.SetLlmParticipation(level)
}

// llmAPIAdapter exposes the LLM surface on the debug HTTP server without
// extending the GameAPI interface.
type llmAPIAdapter struct{ app *App }

func (l llmAPIAdapter) LlmStatus() (any, error) { return l.app.GetLlmStatus(), nil }
func (l llmAPIAdapter) LlmDecisions(limit int) (any, error) {
	return l.app.GetLlmDecisions(limit)
}
func (l llmAPIAdapter) SetLlmParticipation(level string) (string, error) {
	return l.app.SetLlmParticipation(level)
}

// agentAPIAdapter exposes the agent-operator surface on the debug HTTP
// server: an external AI agent plays the game over these endpoints and its
// decisions are recorded for the silo40-npc-ml data pipeline.
type agentAPIAdapter struct{ app *App }

func (a agentAPIAdapter) AgentStart(body []byte) (any, error) {
	var req service.AgentStartRequest
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, fmt.Errorf("invalid start request: %w", err)
		}
	}
	return a.app.gameService.StartAgentGame(req)
}

func (a agentAPIAdapter) AgentStep() (any, error) { return a.app.gameService.AgentStep() }

func (a agentAPIAdapter) AgentChoose(body []byte) (any, error) {
	var req service.AgentChooseRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("invalid choose request: %w", err)
	}
	return a.app.gameService.AgentChoose(req)
}

func (a agentAPIAdapter) AgentStatus() (any, error) { return a.app.gameService.AgentStatus() }

func (a agentAPIAdapter) AgentExport() (any, error) { return a.app.gameService.AgentExport() }

func (a agentAPIAdapter) AgentExportJSONL() (string, error) {
	return a.app.gameService.AgentExportJSONL()
}

func (a agentAPIAdapter) AgentStop() (any, error) { return a.app.gameService.AgentStop() }
