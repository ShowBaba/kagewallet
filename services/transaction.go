package services

import (
	"kagewallet/database"
	"kagewallet/repositories"
)

type TransactionService struct {
	UserRepo        *repositories.UserRepository
	TransactionRepo *repositories.TransactionRepository
}

func NewTransactionService(userRepo *repositories.UserRepository, transactionRepo *repositories.TransactionRepository) *TransactionService {
	return &TransactionService{
		userRepo,
		transactionRepo,
	}
}

func (t *TransactionService) FetchUserTransactions(userID string, limit, offset int) ([]database.TransactionWithAsset, error) {
	return t.TransactionRepo.GetTransactionsByUser(userID, limit, offset)

}

func (t *TransactionService) FetchUserTransactionCount(userID string) (int, error) {
	return t.TransactionRepo.GetUserTransactionCount(userID)
}
