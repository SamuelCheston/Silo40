package repository

import (
	"fmt"
	"log"
	"silo40/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func InitDB(dbPath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Database connection established")

	// 自动迁移逻辑
	err = db.AutoMigrate(
		&model.User{},
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
	)
	if err != nil {
		return nil, fmt.Errorf("failed to run auto migration: %w", err)
	}

	log.Println("Database auto-migration completed")
	return db, nil
}
