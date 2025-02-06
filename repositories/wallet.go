package repositories

import (
	"errors"
	"fmt"

	"github.com/ShowBaba/kagewallet/database"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type WalletRepository struct {
	DB *gorm.DB
}

func NewWalletRepository(db *gorm.DB) *WalletRepository {
	return &WalletRepository{
		DB: db,
	}
}

func (r *WalletRepository) CreateWallet(wallet *database.Wallet) error {
	return r.DB.Create(wallet).Error
}

func (r *WalletRepository) GetWalletByUserAndAsset(userID, assetID uuid.UUID) (*database.Wallet, error) {
	var wallet database.Wallet
	err := r.DB.Where("user_id = ? AND asset_id = ?", userID, assetID).First(&wallet).Error
	if err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (r *WalletRepository) UpdateWalletBalance(userID uuid.UUID, amount decimal.Decimal) error {
	return r.DB.Model(&database.Wallet{}).
		Where("user_id = ?", userID).
		Update("balance", gorm.Expr("balance + ?", amount)).Error
}

func (r *WalletRepository) DeductWalletBalance(userID, assetID uuid.UUID, amount float64) error {
	return r.DB.Model(&database.Wallet{}).
		Where("user_id = ? AND asset_id = ? AND balance >= ?", userID, assetID, amount).
		Update("balance", gorm.Expr("balance - ?", amount)).Error
}

func (r *WalletRepository) GetWalletsByUser(userID uuid.UUID) (*database.WalletWithDetails, error) {
	var wallet database.WalletWithDetails
	err := r.DB.
		Table("wallet").
		Select("wallet.*, telegram.username as user_name, \"user\".email as user_email").
		Joins("JOIN \"user\" ON \"user\".id = wallet.user_id").
		Joins("JOIN telegram ON telegram.user_id = \"user\".id").
		Where("wallet.user_id = ?", userID).
		Scan(&wallet).Error
	if err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (r *WalletRepository) DeleteWallet(userID, assetID uuid.UUID) error {
	return r.DB.Where("user_id = ? AND asset_id = ?", userID, assetID).Delete(&database.Wallet{}).Error
}

func (r *WalletRepository) GetWalletByID(walletID uuid.UUID) (*database.Wallet, error) {
	var wallet database.Wallet
	err := r.DB.Where("id = ?", walletID).First(&wallet).Error
	if err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (r *WalletRepository) HandleTransactionAndUpdateBalance(userID string, transaction *database.Transaction, amount float64) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(transaction).Error; err != nil {
			return fmt.Errorf("error creating transaction: %v", err)
		}

		var wallet database.Wallet
		err := tx.Model(&database.Wallet{}).
			Where("user_id = ?", userID).
			First(&wallet).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				newWallet := &database.Wallet{
					UserID:  uuid.MustParse(userID),
					Balance: decimal.NewFromFloat(amount),
				}
				if createErr := tx.Create(newWallet).Error; createErr != nil {
					return fmt.Errorf("error creating wallet: %v", createErr)
				}
			} else {
				return fmt.Errorf("error checking wallet existence: %v", err)
			}
		} else {
			if updateErr := tx.Model(&database.Wallet{}).
				Where("user_id = ?", userID).
				Update("balance", gorm.Expr("balance + ?", amount)).Error; updateErr != nil {
				return fmt.Errorf("error updating wallet balance: %v", updateErr)
			}
		}

		return nil
	})
}
