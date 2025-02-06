package repositories

import (
	"errors"
	"time"

	"github.com/ShowBaba/kagewallet/database"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WithdrawalRepository struct {
	DB *gorm.DB
}

func NewWithdrawalRepository(db *gorm.DB) *WithdrawalRepository {
	return &WithdrawalRepository{DB: db}
}

func (r *WithdrawalRepository) CreateWithdrawal(withdrawal *database.Withdrawal) error {
	withdrawal.ID = uuid.New()
	withdrawal.CreatedAt = time.Now()
	withdrawal.UpdatedAt = time.Now()

	return r.DB.Create(withdrawal).Error
}

func (r *WithdrawalRepository) GetWithdrawalByID(id uuid.UUID) (*database.Withdrawal, error) {
	var withdrawal database.Withdrawal
	err := r.DB.Where("id = ?", id).First(&withdrawal).Error
	if err != nil {
		return nil, err
	}
	return &withdrawal, nil
}

func (r *WithdrawalRepository) GetWithdrawalByTransactionID(id uuid.UUID) (*database.Withdrawal, error) {
	var withdrawal database.Withdrawal
	err := r.DB.Where("id = ?", id).First(&withdrawal).Error
	if err != nil {
		return nil, err
	}
	return &withdrawal, nil
}

func (r *WithdrawalRepository) GetWithdrawalsByUserID(userID uuid.UUID, limit, offset int) ([]database.Withdrawal, error) {
	var withdrawals []database.Withdrawal
	err := r.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&withdrawals).Error
	return withdrawals, err
}

func (r *WithdrawalRepository) UpdateWithdrawalStatus(withdrawalID uuid.UUID, status string) error {
	result := r.DB.Model(&database.Withdrawal{}).
		Where("id = ?", withdrawalID).
		Update("status", status)
	return result.Error
}

func (r *WithdrawalRepository) GetPendingWithdrawals() ([]database.Withdrawal, error) {
	var withdrawals []database.Withdrawal
	err := r.DB.Where("status = ?", "pending").
		Order("created_at ASC").
		Find(&withdrawals).Error
	return withdrawals, err
}

func (r *WithdrawalRepository) DeleteWithdrawal(id uuid.UUID) error {
	result := r.DB.Where("id = ?", id).Delete(&database.Withdrawal{})
	if result.RowsAffected == 0 {
		return errors.New("withdrawal not found")
	}
	return result.Error
}

func (r *WithdrawalRepository) CountUserWithdrawals(userID uuid.UUID) (int64, error) {
	var count int64
	err := r.DB.Model(&database.Withdrawal{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}

func (r *WithdrawalRepository) CreateTransactionAndWithdrawal(wallet *database.WalletWithDetails, transaction *database.Transaction, withdrawal *database.Withdrawal) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		transaction.ID = uuid.New()
		transaction.CreatedAt = time.Now()
		transaction.UpdatedAt = time.Now()

		if err := tx.Create(transaction).Error; err != nil {
			return err
		}

		withdrawal.ID = uuid.New()
		withdrawal.TransactionID = transaction.ID
		withdrawal.CreatedAt = time.Now()
		withdrawal.UpdatedAt = time.Now()

		if err := tx.Create(withdrawal).Error; err != nil {
			return err
		}

		wallet.UpdatedAt = time.Now()
		result := tx.Model(&database.Wallet{}).
			Where("id = ?", wallet.ID).
			Update("balance", wallet.Balance)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return errors.New("wallet update failed: no rows affected")
		}

		return nil
	})
}

func (r *WithdrawalRepository) GetWithdrawalsByAccountNumber(accountNumber string) ([]database.Withdrawal, error) {
	var withdrawals []database.Withdrawal
	err := r.DB.Where("account_number = ?", accountNumber).Find(&withdrawals).Error
	if err != nil {
		return nil, err
	}
	return withdrawals, nil
}
