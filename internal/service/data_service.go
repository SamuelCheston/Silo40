package service

import (
	"fmt"
	"silo40/internal/cache"
	"time"

	"gorm.io/gorm"
)

type DataService struct {
	db *gorm.DB
}

func NewDataService(db *gorm.DB) *DataService {
	return &DataService{db: db}
}

// Database Operations

func (s *DataService) Create(model interface{}) error {
	return s.db.Create(model).Error
}

func (s *DataService) Save(model interface{}) error {
	return s.db.Save(model).Error
}

func (s *DataService) First(model interface{}, conds ...interface{}) error {
	return s.db.First(model, conds...).Error
}

func (s *DataService) Find(models interface{}, conds ...interface{}) error {
	return s.db.Find(models, conds...).Error
}

func (s *DataService) Delete(model interface{}, conds ...interface{}) error {
	return s.db.Delete(model, conds...).Error
}

// Cache Operations

func (s *DataService) SetCache(key string, value string, expirationSeconds int) error {
	if cache.GlobalCache == nil {
		return fmt.Errorf("cache not initialized")
	}
	return cache.GlobalCache.Set([]byte(key), []byte(value), expirationSeconds)
}

func (s *DataService) GetCache(key string) (string, error) {
	if cache.GlobalCache == nil {
		return "", fmt.Errorf("cache not initialized")
	}
	val, err := cache.GlobalCache.Get([]byte(key))
	if err != nil {
		return "", err
	}
	return string(val), nil
}

func (s *DataService) DelCache(key string) {
	if cache.GlobalCache != nil {
		cache.GlobalCache.Del([]byte(key))
	}
}

// Transaction support
func (s *DataService) Transaction(fc func(tx *gorm.DB) error) error {
	return s.db.Transaction(fc)
}

// Helper to get time
func (s *DataService) GetServerTime() time.Time {
	return time.Now()
}
