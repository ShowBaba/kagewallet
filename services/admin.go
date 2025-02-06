package services

import (
	"errors"
	"time"

	"github.com/ShowBaba/kagewallet/common"
	"github.com/ShowBaba/kagewallet/database"
	"github.com/ShowBaba/kagewallet/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdminService struct {
	RateRepo       *repositories.RateRepository
	AssetRepo      *repositories.AssetRepository
	MonnifyService *MonnifyService
}

func NewAdminService(rateRepo *repositories.RateRepository,
	assetRepo *repositories.AssetRepository, monnifyService *MonnifyService) *AdminService {
	return &AdminService{
		rateRepo,
		assetRepo,
		monnifyService,
	}
}

func (s *AdminService) CreateAsset(input common.CreateAssetInput) error {
	asset := database.Asset{
		ID:           uuid.New(),
		Name:         input.Name,
		Symbol:       input.Symbol,
		Standard:     input.Standard,
		LogoURL:      input.LogoURL,
		IsActive:     input.IsActive,
		Instructions: input.Instructions,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	return s.AssetRepo.AddNewAsset(asset)
}

func (s *AdminService) UpdateAsset(assetID uuid.UUID, updates map[string]interface{}) error {
	return s.AssetRepo.UpdateAsset(assetID, updates)
}

func (s *AdminService) CreateRate(rate float64, source string) error {
	return s.RateRepo.AddNewRate(rate, source)
}

func (s *AdminService) AssetExists(name, symbol, standard string) (bool, error) {
	var asset database.Asset
	if err := s.AssetRepo.DB.Where("name = ? AND symbol = ? AND standard = ?", name, symbol, standard).First(&asset).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *AdminService) GetAssets(active bool) ([]database.Asset, error) {
	var assets []database.Asset
	var err error

	if active {
		err = s.AssetRepo.DB.Where("is_active = ?", true).Order("is_active DESC").Find(&assets).Error
	} else {
		err = s.AssetRepo.DB.Order("is_active DESC").Find(&assets).Error
	}

	if err != nil {
		return nil, err
	}
	return assets, nil
}

func (s *AdminService) ValidateMonnifyTransferOTP(reference, otp string) error {
	return s.MonnifyService.ValidateTransferOTP(reference, otp)
}
