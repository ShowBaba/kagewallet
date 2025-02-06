package services

import (
	"github.com/ShowBaba/kagewallet/database"
	"github.com/ShowBaba/kagewallet/repositories"
	"github.com/google/uuid"
)

type WalletService struct {
	WalletRepo *repositories.WalletRepository
	AssetRepo  *repositories.AssetRepository
}

func NewWalletService(walletRepo *repositories.WalletRepository,
	assetRepo *repositories.AssetRepository) *WalletService {
	return &WalletService{
		walletRepo,
		assetRepo,
	}
}

func (w *WalletService) GetUserWalletsData(userId string) (*database.WalletWithDetails, error) {
	wallet, err := w.WalletRepo.GetWalletsByUser(uuid.MustParse(userId))
	if err != nil {
		return nil, err
	}
	return wallet, nil
}
