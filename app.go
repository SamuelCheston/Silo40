package main

import (
	"context"
	"fmt"
	"log"
	"silo40/internal/cache"
	"silo40/internal/repository"
	"silo40/internal/service"

	"gorm.io/gorm"
)

// App struct
type App struct {
	ctx         context.Context
	db          *gorm.DB
	siloService *service.SiloService
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
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
	a.siloService = service.NewSiloService(db)
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// InitGame initializes the game with a starting year
func (a *App) InitGame(name string, initialYear int) string {
	silo, err := a.siloService.InitSilo(name, initialYear)
	if err != nil {
		return fmt.Sprintf("Failed to initialize silo: %v", err)
	}
	return fmt.Sprintf("Welcome to %s. The year is %d. Juliette has just joined mechanical (or it's around that time).", silo.Name, silo.CurrentYear)
}

// SetData sets data in the cache
func (a *App) SetData(key string, value string) {
	err := cache.GlobalCache.Set([]byte(key), []byte(value), 60) // expire in 60s
	if err != nil {
		log.Printf("failed to set cache: %v", err)
	}
}

// GetData gets data from the cache
func (a *App) GetData(key string) string {
	val, err := cache.GlobalCache.Get([]byte(key))
	if err != nil {
		return ""
	}
	return string(val)
}
