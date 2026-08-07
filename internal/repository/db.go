package repository

import (
	"fmt"
	"log"
	"silo40/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
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
	)
	if err != nil {
		return nil, fmt.Errorf("failed to run auto migration: %w", err)
	}

	log.Println("Database auto-migration completed")
	return db, nil
}
