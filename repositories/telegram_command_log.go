package repositories

import (
	"gorm.io/gorm"
	"kagewallet/database"
)

type TelegramCommandLogRepository struct {
	DB *gorm.DB
}

func NewTelegramCommandLogRepository(db *gorm.DB) *TelegramCommandLogRepository {
	return &TelegramCommandLogRepository{
		DB: db,
	}
}

func (t *TelegramCommandLogRepository) Create(data *database.TelegramCommandLog) error {
	return t.DB.Create(data).Error
}
