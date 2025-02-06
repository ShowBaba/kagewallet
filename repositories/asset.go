package repositories

import (
	"github.com/ShowBaba/kagewallet/database"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AssetRepository struct {
	DB *gorm.DB
}

func NewAssetRepository(db *gorm.DB) *AssetRepository {
	return &AssetRepository{
		DB: db,
	}
}

func (r *AssetRepository) AddNewAsset(asset database.Asset) error {
	return r.DB.Create(&asset).Error
}

func (r *AssetRepository) UpdateAsset(assetID uuid.UUID, updates map[string]interface{}) error {
	return r.DB.Model(&database.Asset{}).Where("id = ?", assetID).Updates(updates).Error
}

func (r *AssetRepository) FindAssetByID(assetID string) (*database.Asset, error) {
	var asset database.Asset
	if err := r.DB.First(&asset, "id = ?", assetID).Error; err != nil {
		return nil, err
	}
	return &asset, nil
}

func (r *AssetRepository) ListAllAssets() ([]database.Asset, error) {
	var assets []database.Asset
	if err := r.DB.Find(&assets).Error; err != nil {
		return nil, err
	}
	return assets, nil
}

func (a *AssetRepository) GetActiveAssets() ([]database.Asset, error) {
	var assets []database.Asset
	err := a.DB.Where("is_active = ?", true).Find(&assets).Error
	return assets, err
}
