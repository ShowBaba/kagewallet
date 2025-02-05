package repositories

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"kagewallet/database"
)

type AddressRepository struct {
	DB *gorm.DB
}

func NewAddressRepository(db *gorm.DB) *AddressRepository {
	return &AddressRepository{
		DB: db,
	}
}

func (r *AddressRepository) CreateAddress(address *database.Address) error {
	return r.DB.Create(address).Error
}

func (r *AddressRepository) GetAddressByID(id uuid.UUID) (*database.Address, error) {
	var address database.Address
	err := r.DB.First(&address, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &address, err
}

func (r *AddressRepository) GetAddressesByUserID(userID uuid.UUID) ([]database.Address, error) {
	var addresses []database.Address
	err := r.DB.Where("user_id = ?", userID).Find(&addresses).Error
	return addresses, err
}

func (r *AddressRepository) GetAddressByUserAndAsset(userID, assetID, address string) (*database.Address, error) {
	var foundAddress database.Address
	err := r.DB.Where("user_id = ? AND asset_id = ? AND address = ?", userID, assetID, address).First(&foundAddress).Error
	return &foundAddress, err
}

func (r *AddressRepository) UpdateAddress(address database.Address) error {
	return r.DB.Save(&address).Error
}

func (r *AddressRepository) DeleteAddress(id uuid.UUID) error {
	return r.DB.Model(&database.Address{}).Where("id = ?", id).Update("is_active", false).Error
}

func (r *AddressRepository) GetLastActiveAddressByUser(userID uuid.UUID, assetID *string) (*database.Address, error) {
	var address database.Address

	query := r.DB.Where("user_id = ? AND is_active = ?", userID, true)

	if assetID != nil {
		query = query.Where("asset_id = ?", *assetID)
	}

	err := query.Order("updated_at DESC").First(&address).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &address, err
}

func (r *AddressRepository) SetAllAddressesInactiveByUserAndAsset(userID, assetID uuid.UUID) error {
	result := r.DB.Model(&database.Address{}).
		Where("user_id = ? AND asset_id = ? AND is_active = ?", userID, assetID, true).
		Update("is_active", false)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("no active addresses found for the given user and asset")
	}

	return nil
}

func (r *AddressRepository) GetAddresses(offset, limit int) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	err := r.DB.
		Model(&database.Address{}).
		Select("address, user_id").
		Offset(offset).
		Limit(limit).
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}

func (r *AddressRepository) GetAddressByColumn(column string, value interface{}) ([]database.Address, error) {
	var addresses []database.Address

	err := r.DB.
		Where(column+" = ?", value).
		Find(&addresses).Error

	if err != nil {
		return nil, err
	}

	return addresses, nil
}
