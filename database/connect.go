package database

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type Config struct {
	Host          string
	Port          string
	Password      string
	User          string
	DBName        string
	DisableLogger bool
}

func ConnectPg(config *Config) (*gorm.DB, error) {
	var (
		err     error
		port, _ = strconv.ParseUint(config.Port, 10, 32)
		dsn     = fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			config.Host, port, config.User, config.Password, config.DBName,
		)

		db      *gorm.DB
		options = gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				SingularTable: true,
			},
			DisableForeignKeyConstraintWhenMigrating: true,
			SkipDefaultTransaction:                   true,
			Logger:                                   logger.Default.LogMode(logger.Info),
		}
	)

	if config.DisableLogger {
		options.Logger = logger.Default.LogMode(logger.Silent)
	}

	db, err = gorm.Open(postgres.Open(dsn), &options)
	if err != nil {
		return nil, fmt.Errorf("failed to connect database, err: %s", err)
	}
	return db, nil
}

func InitializeRedis(address, password string, db int) error {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     address,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := RedisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	fmt.Println("Connected to Redis successfully")
	return nil
}
