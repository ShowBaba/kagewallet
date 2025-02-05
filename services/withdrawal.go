package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"kagewallet/common"
	"kagewallet/database"
	"kagewallet/helpers"
	"kagewallet/repositories"
)

type WithdrawalService struct {
	MonnifyService *MonnifyService
	WithdrawalRepo *repositories.WithdrawalRepository
	WalletRepo     *repositories.WalletRepository
}

func NewWithdrawalService(monnifyService *MonnifyService, withdrawalRepo *repositories.WithdrawalRepository,
	walletRepo *repositories.WalletRepository) *WithdrawalService {
	return &WithdrawalService{monnifyService,
		withdrawalRepo,
		walletRepo}
}

func (w *WithdrawalService) GetBanks(page, limit int) (paginatedBanks []Bank, totalPages int, err error) {
	return w.MonnifyService.GetBanks(page, limit)
}

func (w *WithdrawalService) GetBankByCode(code string) (bank Bank, err error) {
	return w.MonnifyService.GetBankByCode(code)
}

func (w *WithdrawalService) SearchBank(query string, page int, limit int) ([]Bank, int, error) {
	return w.MonnifyService.SearchBank(query, page, limit)
}

func (w *WithdrawalService) ValidateBankAccount(accountNumber, bankCode string) (*AccountDetails, error) {
	return w.MonnifyService.ValidateBankAccount(accountNumber, bankCode)
}

func (w *WithdrawalService) InitiateTransfer(accountNumber, bankCode, userId string, amount float64) error {
	amountDec := decimal.NewFromFloat(amount)
	withdrawalFeeDec := decimal.NewFromFloat(common.WithdrawalFee)
	finalAmount := amountDec.Sub(withdrawalFeeDec)
	wallet, err := w.WalletRepo.GetWalletsByUser(uuid.MustParse(userId))
	if err != nil {
		return err
	}
	if wallet.Balance < amount {
		return errors.New("insufficient wallet balance")
	}
	wallet.Balance -= amount
	wallet.UpdatedAt = time.Now()

	sourceRef, response, err := w.MonnifyService.InitiateTransfer(finalAmount, bankCode, accountNumber)
	if err != nil {
		return err
	}
	ref := helpers.GenerateTransactionReference()
	hash, err := helpers.GenerateRandomHash(ref)
	if err != nil {
		return err
	}
	fmt.Println("monnify initiate withdrawal response; ", response)
	transaction := database.Transaction{
		UserID:          uuid.MustParse(userId),
		AssetID:         uuid.MustParse(common.NairaAssetID),
		Type:            "withdrawal",
		Amount:          finalAmount,
		Status:          "pending",
		Reference:       helpers.GenerateTransactionReference(),
		SourceReference: sourceRef,
		Hash:            hash,
		RateID:          uuid.Nil,
		Confirmations:   0,
		AmountUSD:       0,
		Source:          "Monnify",
	}

	bank, err := w.GetBankByCode(bankCode)
	if err != nil {
		return err
	}
	withdrawal := database.Withdrawal{
		AccountNumber: accountNumber,
		BankCode:      bankCode,
		BankName:      bank.Name,
		UserID:        uuid.MustParse(userId),
		Status:        "pending",
		Amount:        finalAmount,
		Fee:           common.WithdrawalFee,
	}
	return w.WithdrawalRepo.CreateTransactionAndWithdrawal(wallet, &transaction, &withdrawal)
}
