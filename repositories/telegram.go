package repositories

import (
	"errors"
	"time"

	"github.com/ShowBaba/kagewallet/database"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TelegramRepository struct {
	DB *gorm.DB
}

func NewTelegramRepository(db *gorm.DB) *TelegramRepository {
	return &TelegramRepository{
		DB: db,
	}
}

func (t *TelegramRepository) Create(telegram *database.Telegram) error {
	return t.DB.Create(telegram).Error
}

func (t *TelegramRepository) UpdateField(telegramID uuid.UUID, fieldName string, fieldValue interface{}) error {
	updateData := map[string]interface{}{
		fieldName:    fieldValue,
		"updated_at": time.Now(),
	}

	return t.DB.Model(&database.Telegram{}).
		Where("id = ?", telegramID).
		Updates(updateData).Error
}

func (t *TelegramRepository) FindByUsername(username string) (*database.Telegram, error) {
	var telegram database.Telegram
	if err := t.DB.First(&telegram, "username = ?", username).Error; err != nil {
		return nil, err
	}
	return &telegram, nil
}

func (t *TelegramRepository) FindUserByTelegramID(telegramID int) (*database.User, error) {
	var telegram database.Telegram
	if err := t.DB.First(&telegram, "telegram_id = ?", telegramID).Error; err != nil {
		return nil, err
	}

	var user database.User
	if err := t.DB.First(&user, "id = ?", telegram.UserID).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (t *TelegramRepository) Upsert(username string, telegramID int) (*database.User, error) {
	var (
		telegram database.Telegram
		user     database.User
	)

	err := t.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Where("telegram_id = ?", telegramID).First(&telegram).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			user = database.User{
				ID:           uuid.New(),
				PasswordHash: "",
			}
			if err := tx.Create(&user).Error; err != nil {
				return err
			}

			telegram = database.Telegram{
				ID:         uuid.New(),
				Username:   username,
				TelegramID: telegramID,
				UserID:     user.ID,
			}
			if err := tx.Create(&telegram).Error; err != nil {
				return err
			}
		} else if err == nil {
			if telegram.Username != username {
				if err := tx.Model(&telegram).Update("username", username).Error; err != nil {
					return err
				}
			}

			if err := tx.First(&user, "id = ?", telegram.UserID).Error; err != nil {
				return err
			}
		} else {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &user, nil

}
