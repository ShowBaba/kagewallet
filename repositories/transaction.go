package repositories

import (
	"fmt"

	"gorm.io/gorm"
	"kagewallet/database"
)

type TransactionRepository struct {
	DB *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) *TransactionRepository {
	return &TransactionRepository{
		DB: db,
	}
}

func (r *TransactionRepository) CreateTransaction(transaction *database.Transaction) error {
	return r.DB.Create(transaction).Error
}

func (r *TransactionRepository) GetTransactionsByUser(userID string, limit, offset int) ([]database.TransactionWithAsset, error) {
	var transactions []database.TransactionWithAsset
	err := r.DB.Table("transaction").
		Select("transaction.id AS transaction_id, transaction.user_id, transaction.amount_usd, transaction.reference, transaction.confirmations, transaction.asset_id, transaction.type, transaction.amount, transaction.status, transaction.created_at, asset.name AS asset_name, asset.symbol AS asset_symbol, asset.standard AS asset_standard, COALESCE(rate.rate, 0) AS rate").
		Joins("JOIN asset ON transaction.asset_id = asset.id").
		Joins("LEFT JOIN rate ON transaction.rate_id = rate.id").
		Where("transaction.user_id = ?", userID).
		Order("transaction.created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&transactions).Error

	if err != nil {
		return nil, err
	}
	return transactions, nil
}

func (r *TransactionRepository) GetUserTransactionCount(userID string) (int, error) {
	var count int64
	err := r.DB.Model(&database.Transaction{}).
		Where("user_id = ?", userID).
		// Limit(limit).
		// Offset(offset).
		Count(&count).Error
	return int(count), err
}

func (r *TransactionRepository) GetTransactionByID(transactionID string) (*database.Transaction, error) {
	var transaction database.Transaction
	err := r.DB.Where("id = ?", transactionID).First(&transaction).Error
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

func (r *TransactionRepository) UpdateTransactionStatus(transactionID, status string) error {
	return r.DB.Model(&database.Transaction{}).
		Where("id = ?", transactionID).
		Update("status", status).Error
}

func (r *TransactionRepository) DeleteTransaction(transactionID string) error {
	return r.DB.Where("id = ?", transactionID).Delete(&database.Transaction{}).Error
}

func (r *TransactionRepository) GetTransactionsWithFilters(filters map[string]interface{}, limit, offset int) ([]database.Transaction, error) {
	var transactions []database.Transaction
	query := r.DB.Model(&database.Transaction{})

	for key, value := range filters {
		query = query.Where(fmt.Sprintf("%s = ?", key), value)
	}

	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&transactions).Error
	return transactions, err
}

func (r *TransactionRepository) GetTransactionByReference(reference string) (*database.Transaction, error) {
	var transaction database.Transaction
	err := r.DB.Where("reference = ?", reference).First(&transaction).Error
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

func (r *TransactionRepository) GetTransactionBySourceReference(reference string) (*database.Transaction, error) {
	var transaction database.Transaction
	err := r.DB.Where("source_reference = ?", reference).First(&transaction).Error
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

func (r *TransactionRepository) GetTotalTransactionCountByUser(userID string) (int64, error) {
	var count int64
	err := r.DB.Model(&database.Transaction{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}

func (r *TransactionRepository) GetTotalAmountByUser(userID string) (float64, error) {
	var totalAmount float64
	err := r.DB.Model(&database.Transaction{}).
		Where("user_id = ?", userID).
		Select("SUM(amount)").
		Scan(&totalAmount).Error
	return totalAmount, err
}

func (r *TransactionRepository) GetTransactionByColumn(column string, value interface{}) ([]database.Transaction, error) {
	var transactions []database.Transaction

	err := r.DB.
		Where(column+" = ?", value).
		Find(&transactions).Error

	if err != nil {
		return nil, err
	}

	return transactions, nil
}
