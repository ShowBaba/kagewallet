package common

import (
	"time"

	tgApi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type SetPasswordInput struct {
	Password string
	UserID   string
}

type CreateAssetInput struct {
	Name         string `json:"name"`
	Symbol       string `json:"symbol"`
	Standard     string `json:"standard"`
	Instructions string `json:"instructions"`
	IsActive     bool   `json:"is_active"`
	LogoURL      string `json:"logo_url"`
}

type GenerateAddressResponse struct {
	Address     string `json:"address"`
	Instruction string `json:"instruction"`
}

type TelegramCallbackResponse struct {
	CallbackQueryID string
	Text            string
	ShowAlert       bool
	URL             string
	CacheTime       int
}

type TelegramMessageEdit struct {
	ChatID      int64
	MessageID   int
	NewText     string
	ReplyMarkup *tgApi.InlineKeyboardMarkup
	ParseMode   string
}

type BlockradarEvent struct {
	Event string `json:"event"`
	Data  struct {
		ID               string  `json:"id"`
		Reference        string  `json:"reference"`
		SenderAddress    string  `json:"senderAddress"`
		RecipientAddress string  `json:"recipientAddress"`
		Amount           string  `json:"amount"`
		AmountPaid       string  `json:"amountPaid"`
		Fee              *string `json:"fee"`
		Currency         string  `json:"currency"`
		BlockNumber      int64   `json:"blockNumber"`
		BlockHash        string  `json:"blockHash"`
		Hash             string  `json:"hash"`
		Confirmations    int     `json:"confirmations"`
		Confirmed        bool    `json:"confirmed"`
		GasPrice         string  `json:"gasPrice"`
		GasUsed          string  `json:"gasUsed"`
		GasFee           string  `json:"gasFee"`
		Status           string  `json:"status"`
		Type             string  `json:"type"`
		Note             *string `json:"note"`
		AmlScreening     struct {
			Provider string `json:"provider"`
			Status   string `json:"status"`
			Message  string `json:"message"`
		} `json:"amlScreening"`
		AssetSwept                 interface{} `json:"assetSwept"`
		AssetSweptAt               *time.Time  `json:"assetSweptAt"`
		AssetSweptGasFee           *string     `json:"assetSweptGasFee"`
		AssetSweptHash             *string     `json:"assetSweptHash"`
		AssetSweptSenderAddress    *string     `json:"assetSweptSenderAddress"`
		AssetSweptRecipientAddress *string     `json:"assetSweptRecipientAddress"`
		AssetSweptAmount           *string     `json:"assetSweptAmount"`
		Reason                     *string     `json:"reason"`
		Network                    string      `json:"network"`
		ChainID                    int64       `json:"chainId"`
		Metadata                   interface{} `json:"metadata"`
		CreatedAt                  time.Time   `json:"createdAt"`
		UpdatedAt                  time.Time   `json:"updatedAt"`
		Address                    struct {
			ID             string      `json:"id"`
			Address        string      `json:"address"`
			Name           string      `json:"name"`
			IsActive       bool        `json:"isActive"`
			Type           string      `json:"type"`
			DeriationPath  string      `json:"derivationPath"`
			Metadata       interface{} `json:"metadata"`
			Configurations struct {
				Aml struct {
					Status   string `json:"status"`
					Message  string `json:"message"`
					Provider string `json:"provider"`
				} ` json:"aml"`
				ShowPrivateKey        bool `json:"showPrivateKey"`
				DisableAutoSweep      bool `json:"disableAutoSweep"`
				EnableGaslessWithdraw bool `json:"enableGaslessWithdraw"`
			} `json:"configurations"`
			Network   string    `json:"network"`
			CreatedAt time.Time ` json:"createdAt"`
			UpdatedAt time.Time `json:"updatedAt"`
		} `json:"address"`
		Blockchain struct {
			ID              string `json:"id"`
			Name            string `json:"name"`
			Symbol          string `json:"symbol"`
			Slug            string `json:"slug"`
			DerivationPath  string `json:"derivationPath"`
			IsEvmCompatible bool   `json:"isEvmCompatible"`
			IsActive        bool   `json:"isActive"`
			TokenStandard   string `json:"tokenStandard"`
			CreatedAt       string `json:"createdAt"`
			UpdatedAt       string `json:"updatedAt"`
			LogoURL         string `json:"logoUrl"`
		} `json:"blockchain"`
	} `json:"data"`
}

type Notification struct {
	Payload   interface{} `json:"payload"`
	Subject   string      `json:"subject"`
	Channel   string      `json:"channel"`
	To        string      `json:"to"`
	Status    string      `json:"status"` // failed, pending or delivered
	CreatedAt time.Time   `json:"created_at"`
}

type TelegramChatMetadata struct {
	User      string    `json:"user"`
	ChatID    int64     `json:"chat_id"`
	UpdatedAt time.Time `json:"updated_at"`
}

type MonnifyEvent struct {
	EventType string `json:"eventType"`
	EventData struct {
		Amount                   int    `json:"amount"`
		TransactionReference     string `json:"transactionReference"`
		Fee                      int    `json:"fee"`
		TransactionDescription   string `json:"transactionDescription"`
		DestinationAccountNumber string `json:"destinationAccountNumber"`
		SessionId                string `json:"sessionId"`
		CreatedOn                string `json:"createdOn"`
		DestinationAccountName   string `json:"destinationAccountName"`
		Reference                string `json:"reference"`
		DestinationBankCode      string `json:"destinationBankCode"`
		CompletedOn              string `json:"completedOn"`
		Narration                string `json:"narration"`
		Currency                 string `json:"currency"`
		DestinationBankName      string `json:"destinationBankName"`
		Status                   string `json:"status"`
	} `json:"eventData"`
}
