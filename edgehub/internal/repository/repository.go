package repository

import (
	"fmt"
	"time"

	"github.com/edgehub/edgehub/internal/config"
	"github.com/edgehub/edgehub/internal/models"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type PostgresDB struct {
	DB *gorm.DB
}

func NewPostgresDB(cfg config.DatabaseConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxConn)
	sqlDB.SetMaxIdleConns(cfg.MaxConn / 2)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.Tenant{},
		&models.Cluster{},
		&models.Node{},
		&models.NodePool{},
		&models.Job{},
		&models.MarketOffer{},
		&models.MarketOrder{},
		&models.Bill{},
		&models.Alert{},
		&models.BenchmarkResult{},
		&models.NodeMetrics{},
		&models.APIKey{},
	)
}

type RedisClient = redis.Client

func NewRedisClient(cfg config.RedisConfig) *RedisClient {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})
	return rdb
}
