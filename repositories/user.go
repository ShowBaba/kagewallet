package repositories

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"kagewallet/database"
)

type UserRepository struct {
	DB *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		DB: db,
	}
}

func (r *UserRepository) Create(user *database.User) error {
	return r.DB.Create(user).Error
}

func (r *UserRepository) UpdateField(id uuid.UUID, fieldName string, fieldValue interface{}) error {
	updateData := map[string]interface{}{
		fieldName:    fieldValue,
		"updated_at": time.Now(),
	}

	return r.DB.Model(&database.User{}).
		Where("id = ?", id).
		Updates(updateData).Error
}

func (r *UserRepository) FindOneByID(id string) (*database.User, error) {
	var user database.User
	if err := r.DB.First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) UpdatePassword(userID string, hashedPassword string) error {
	return r.DB.Model(&database.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"password_hash": hashedPassword,
		"updated_at":    time.Now(),
	}).Error
}

func (r *UserRepository) HasSetPassword(userID uuid.UUID) (bool, error) {
	var user database.User

	err := r.DB.Model(&database.User{}).
		Select("password_hash").
		Where("id = ?", userID).
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	return user.PasswordHash != "", nil
}
