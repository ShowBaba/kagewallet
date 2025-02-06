package database

import (
	"context"
	"fmt"
	"log"
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
	User          string
	Password      string
	DBName        string
	DisableLogger bool
	SSLRootCert   string
}

func ConnectPg(config *Config, env string) (*gorm.DB, error) {
	var (
		err     error
		port, _ = strconv.ParseUint(config.Port, 10, 32)
		dsn     = fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=verify-full sslrootcert=%s",
			config.Host, port, config.User, config.Password, config.DBName, config.SSLRootCert)

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

	if env == "dev" {
		dsn = fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			config.Host, port, config.User, config.Password, config.DBName,
		)
	}

	if config.DisableLogger {
		options.Logger = logger.Default.LogMode(logger.Silent)
	}

	db, err = gorm.Open(postgres.Open(dsn), &options)
	if err != nil {
		return nil, fmt.Errorf("failed to connect database, err: %s", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying database connection, err: %s", err)
	}

	err = sqlDB.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping database, err: %s", err)
	}

	log.Printf("successfully connected to database")

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
