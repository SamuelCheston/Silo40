package main

import (
	"context"
	"embed"
	"log"
	"os"
	"silo40/internal/api"
	"silo40/internal/cache"
	"silo40/internal/engine"
	"silo40/internal/model"
	"silo40/internal/repository"
	"silo40/internal/service"
	"time"

	"gorm.io/gorm"
)

// App struct
type App struct {
	ctx          context.Context
	db           *gorm.DB
	gameService  *service.GameService
	debugHTTPAPI *api.DebugHTTPServer
	eventsFS     embed.FS
}

// NewApp creates a new App application struct
func NewApp(events embed.FS) *App {
	return &App{
		eventsFS: events,
	}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Initialize Cache (100MB)
	cacheSize := 100 * 1024 * 1024
	cache.InitCache(cacheSize)
	log.Println("FreeCache initialized")

	// Initialize SQLite DB
	db, err := repository.InitDB("silo40.db")
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	a.db = db

	// 有状态游戏会话服务 (后端为唯一事实来源)
	a.gameService = service.NewGameService(db)
	if err := a.gameService.BootstrapContent(a.eventsFS); err != nil {
		log.Printf("failed to bootstrap content events: %v", err)
	}
	if err := a.gameService.Resume(); err != nil {
		log.Println("no saved game session to resume:", err)
	}

	a.debugHTTPAPI = api.NewDebugHTTPServer(os.Getenv("SILO40_DEBUG_HTTP_ADDR"), a)
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

// GetEventHistory 获取事件历史，供 debug 前端或工具查询
func (a *App) GetEventHistory(limit int) (*model.EventHistoryResult, error) {
	return a.gameService.GetEventHistory(limit)
}

// PassTime 推进时间 months 个月 (tick 结算全部在 Go 完成)
func (a *App) PassTime(months int) (*model.TickResult, error) {
	return a.gameService.PassTime(months)
}

// ExecuteAction 执行玩家动作；耗时动作自动推进对应月份
func (a *App) ExecuteAction(action model.AgentAction) (*model.ActionOutcome, error) {
	return a.gameService.ExecuteAction(action)
}

// GetEndingNarrative 获取结局叙事文案
func (a *App) GetEndingNarrative() (string, error) {
	return a.gameService.GetEndingNarrative(), nil
}

// HasActiveGame 是否有进行中的游戏会话
func (a *App) HasActiveGame() (bool, error) {
	return a.gameService.HasActiveGame(), nil
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
