package repository

import (
	"fmt"
	"log"
	"os"
	"silo40/internal/model"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func InitDB(dbPath string) (*gorm.DB, error) {
	// _busy_timeout: the LLM advisor's shadow workers write to the decision
	// log concurrently with the game thread; without a busy timeout SQLite
	// fails those writes instantly (SQLITE_BUSY).
	// Missing rows are expected during save detection. All other database
	// errors retain normal Error logging and are returned to callers.
	gormLog := gormlogger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), gormlogger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  gormlogger.Error,
		IgnoreRecordNotFoundError: true,
		Colorful:                  true,
	})
	db, err := gorm.Open(sqlite.Open(dbPath+"?_busy_timeout=10000"), &gorm.Config{Logger: gormLog})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Database connection established")

	// 自动迁移逻辑
	err = db.AutoMigrate(
		&model.User{},
		&model.World{},
		&model.Silo{},
		&model.Resource{},
		&model.Profession{},
		&model.Floor{},
		&model.PopulationCohort{},
		&model.Resident{},
		&model.Faction{},
		&model.Agent{},
		&model.Connection{},
		&model.StoryEventLog{},
		&model.ContentEventDefinition{},
		&model.ContentEventState{},
		&model.SaveMeta{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to run auto migration: %w", err)
	}

	log.Println("Database auto-migration completed")
	return db, nil
}
