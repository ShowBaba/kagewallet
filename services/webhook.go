package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ShowBaba/kagewallet/common"
	"github.com/ShowBaba/kagewallet/database"
	"github.com/ShowBaba/kagewallet/helpers"
	log "github.com/ShowBaba/kagewallet/logging"
	"github.com/ShowBaba/kagewallet/repositories"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type WebhookService struct {
	AddressRepo     *repositories.AddressRepository
	TransactionRepo *repositories.TransactionRepository
	WalletRepo      *repositories.WalletRepository
	AssetRepo       *repositories.AssetRepository
	WithdrawalRepo  *repositories.WithdrawalRepository
	RateService     *RateService
}

func NewWebhookService(addressRepo *repositories.AddressRepository,
	transactionRepo *repositories.TransactionRepository,
	walletRepo *repositories.WalletRepository,
	assetRepo *repositories.AssetRepository,
	withdrawalRepo *repositories.WithdrawalRepository,
	rateService *RateService) *WebhookService {
	return &WebhookService{
		addressRepo,
		transactionRepo,
		walletRepo,
		assetRepo,
		withdrawalRepo,
		rateService,
	}
}

func (w *WebhookService) BlockradarWebhook(payload common.BlockradarEvent) error {
	userData := make(map[string]string)
	m, err := json.Marshal(payload.Data.Address.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %v", err)
	}

	err = json.Unmarshal(m, &userData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal webhook metadata: %v", err)
	}

	userID := userData["user_id"]
	assetID := userData["asset_id"]
	if userID == "" || assetID == "" {
		return fmt.Errorf("missing user_id or asset_id in webhook metadata")
	}

	addressData, err := w.AddressRepo.GetAddressByUserAndAsset(userID, assetID, payload.Data.Address.Address)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("error fetching address data: %v", err)
	}

	if addressData == nil || addressData.Address != payload.Data.RecipientAddress {
		log.Error("address mismatch or data not found in webhook")
		return nil
	}

	existingTransactions, err := w.TransactionRepo.GetTransactionByColumn("hash", payload.Data.Hash)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("error checking existing transaction: %v", err)
	}

	var status string
	switch payload.Data.Status {
	case "SUCCESS":
		status = "completed"
	default:
		status = "failed"
	}

	coinAmount, err := strconv.ParseFloat(payload.Data.Amount, 64)
	if err != nil {
		return fmt.Errorf("invalid amount in webhook: %v", err)
	}

	rate, err := w.RateService.GetCurrentRate()
	if err != nil {
		log.Error("error fetching rate", zap.Error(err))
		return fmt.Errorf("error processing webhook: %v", err)
	}

	amount := coinAmount * rate.Rate

	hash := payload.Data.Hash

	// TODO: if the transaction fail, alert admin
	if len(existingTransactions) > 0 {
		if existingTransactions[0].Status != payload.Data.Status {
			err := w.TransactionRepo.UpdateTransactionStatus(existingTransactions[0].ID.String(), status)
			if err != nil {
				return fmt.Errorf("error updating transaction status: %v", err)
			}
		}
	} else {

		var amountUSD float64
		fmt.Println("payload.Data.Currency; ", payload.Data.Currency)
		if payload.Data.Currency == "USD" {
			amountUSD, err = strconv.ParseFloat(payload.Data.AmountPaid, 64)
			if err != nil {
				log.Error("error converting amount", zap.Error(err))
				return fmt.Errorf("error processing webhook: %v", err)
			}
		}
		err = w.WalletRepo.HandleTransactionAndUpdateBalance(userID, &database.Transaction{
			ID:              uuid.New(),
			UserID:          uuid.MustParse(userID),
			AssetID:         uuid.MustParse(assetID),
			Type:            strings.ToLower(payload.Data.Type),
			Amount:          decimal.NewFromFloat(coinAmount),
			Status:          status,
			Reference:       helpers.GenerateTransactionReference(),
			SourceReference: payload.Data.Reference,
			Hash:            hash,
			RateID:          rate.ID,
			Confirmations:   int64(payload.Data.Confirmations),
			AmountUSD:       amountUSD,
			Source:          "Blockradar",
		}, amount)

		if err != nil {
			return fmt.Errorf("error handling transaction and updating wallet balance: %v", err)
		}

		log.Info("transaction processed successfully", zap.String("transaction_id", payload.Data.Reference))
	}

	assetData, err := w.AssetRepo.FindAssetByID(assetID)
	if err != nil {
		return fmt.Errorf("error fetching asset: %v", err)
	}

	if status == "completed" {
		message := fmt.Sprintf(
			"🎉 Trade Successful! 🎉\n\n"+
				"Your trade of *%v %v* has been processed successfully. ✅\n\n"+
				"💰 Your balance has been updated.\n\n"+
				"🔍 Use /balance to check your updated balances. Happy trading! 🚀",
			coinAmount,
			assetData.Symbol,
		)

		if err = sendNotification(message, userID, hash, "telegram"); err != nil {
			log.Error("failed to publish notification: %v", zap.Error(err))
		}
	}

	return nil
}

func (w *WebhookService) MonnifyWebhook(payload common.MonnifyEvent) error {

	fmt.Println("payload; ", payload)
	var (
		message, userId, hash string
	)
	switch payload.EventType {
	case "SUCCESSFUL_DISBURSEMENT":
		transaction, err := w.TransactionRepo.GetTransactionBySourceReference(payload.EventData.Reference)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		err = w.TransactionRepo.UpdateTransactionStatus(transaction.ID.String(), "completed")
		if err != nil {
			return fmt.Errorf("error updating transaction status: %v", err)
		}
		message = fmt.Sprintf(
			"🎉 Success! Your withdrawal of *₦%v* has been processed. 🚀\n\n"+
				"💰 Expect to receive your funds shortly. If you have any concerns, feel free to reach out to our support team. 🛠️",
			transaction.Amount,
		)

		userId = transaction.UserID.String()
		hash = transaction.Hash
	case "FAILED_DISBURSEMENT", "REVERSED_DISBURSEMENT":
		// TODO: notify admin
		transaction, err := w.TransactionRepo.GetTransactionBySourceReference(payload.EventData.TransactionReference)
		if err != nil {
			return err
		}
		err = w.TransactionRepo.UpdateTransactionStatus(transaction.ID.String(), "failed")
		if err != nil {
			return fmt.Errorf("error updating transaction status: %v", err)
		}
		message = fmt.Sprintf(
			"⚠️ Oops! Your withdrawal of *₦%v* could not be processed. 😞\n\n"+
				"🔄 You can try again later or reach out to our support team for assistance. We're here to help! 🛠️",
			transaction.Amount,
		)

		userId = transaction.UserID.String()
		hash = transaction.Hash
		withdrawalData, err := w.WithdrawalRepo.GetWithdrawalByTransactionID(transaction.ID)
		if err != nil {
			return err
		}
		originalAmount := withdrawalData.Amount.Add(decimal.NewFromInt(int64(withdrawalData.Fee)))
		if err := w.WalletRepo.UpdateWalletBalance(transaction.UserID, originalAmount); err != nil {
			return err
		}

	default:
		return fmt.Errorf("unknown event type: %s", payload.EventType)
	}

	if err := sendNotification(message, userId, hash, "telegram"); err != nil {
		log.Error("failed to publish notification: %v", zap.Error(err))
	}

	return nil
}

func sendNotification(message, userID, hash, channel string) error {
	fmt.Println("sending notification:", channel)

	notificationJSON, err := database.HGet(common.RedisNotificationKey, hash)
	if err != nil && !errors.Is(err, redis.Nil) {
		log.Error("failed to fetch notification data from redis", zap.Error(err))
		return err
	}

	if notificationJSON != "" {
		var notification common.Notification
		if err := json.Unmarshal([]byte(notificationJSON), &notification); err != nil {
			log.Error("failed to unmarshal notification", zap.String("key", hash), zap.Error(err))
			return nil
		}
		if notification.Status == "delivered" {
			return nil
		}
	}

	data := common.Notification{
		Channel:   channel,
		Payload:   message,
		CreatedAt: time.Now(),
		Status:    "pending",
		To:        userID,
	}

	updatedNotification, err := json.Marshal(data)
	if err != nil {
		log.Error("failed to marshal notification", zap.Error(err))
		return err
	}

	if err := database.HSet(common.RedisNotificationKey, hash, updatedNotification); err != nil {
		log.Error("failed to store notification on redis", zap.Error(err))
		return err
	}

	if err := database.RedisPublish(common.RedisNotificationChannelKey, hash); err != nil {
		log.Error("failed to publish notification", zap.Error(err))
		return err
	}

	return nil
}
