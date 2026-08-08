package main

import (
	"context"
	"fmt"
	"log"
	"silo40/internal/cache"
	"silo40/internal/model"
	"silo40/internal/repository"
	"silo40/internal/service"

	"gorm.io/gorm"
)

// App struct
type App struct {
	ctx         context.Context
	db          *gorm.DB
	dataService *service.DataService
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
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
	a.dataService = service.NewDataService(db)
}

// --- Abstract Database API ---

// Silo Operations
func (a *App) GetSilo(id uint) (*model.Silo, error) {
	var silo model.Silo
	err := a.db.Preload("Resources").Preload("Professions").Preload("Floors").First(&silo, id).Error
	return &silo, err
}

func (a *App) SaveSilo(silo *model.Silo) error {
	return a.db.Save(silo).Error
}

func (a *App) CreateSilo(silo *model.Silo) (*model.Silo, error) {
	err := a.db.Create(silo).Error
	return silo, err
}

// Agent Operations
func (a *App) GetAgent(id uint) (*model.Agent, error) {
	var agent model.Agent
	err := a.db.Preload("Connections").First(&agent, id).Error
	return &agent, err
}

func (a *App) SaveAgent(agent *model.Agent) error {
	return a.db.Save(agent).Error
}

// User Operations
func (a *App) GetUser(id uint) (*model.User, error) {
	var user model.User
	err := a.db.Preload("Agent").First(&user, id).Error
	return &user, err
}

// Generic List (Example for Silos)
func (a *App) ListSilos() ([]model.Silo, error) {
	var silos []model.Silo
	err := a.db.Find(&silos).Error
	return silos, err
}

// --- Abstract Cache API ---

func (a *App) SetCache(key string, value string, ttl int) error {
	return a.dataService.SetCache(key, value, ttl)
}

func (a *App) GetCache(key string) (string, error) {
	return a.dataService.GetCache(key)
}

func (a *App) DelCache(key string) {
	a.dataService.DelCache(key)
}

// --- Legacy / Helper ---

func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}
