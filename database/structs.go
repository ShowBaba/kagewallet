package database

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type User struct {
	ID           uuid.UUID
	PasswordHash string
	Email        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	u.CreatedAt = time.Now().Local()
	u.UpdatedAt = time.Now().Local()
	u.ID = uuid.New()
	return
}

type Telegram struct {
	ID         uuid.UUID
	Username   string
	TelegramID int
	UserID     uuid.UUID
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (t *Telegram) BeforeCreate(tx *gorm.DB) (err error) {
	t.CreatedAt = time.Now().Local()
	t.UpdatedAt = time.Now().Local()
	t.ID = uuid.New()
	return
}

type TelegramCommandLog struct {
	ID          uuid.UUID
	CommandName string
	UserID      uuid.UUID
	UsageTime   time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (t *TelegramCommandLog) BeforeCreate(tx *gorm.DB) (err error) {
	t.CreatedAt = time.Now().Local()
	t.UpdatedAt = time.Now().Local()
	t.ID = uuid.New()
	return
}

type Rate struct {
	ID        uuid.UUID
	Rate      float64
	Source    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (e *Rate) BeforeCreate(tx *gorm.DB) (err error) {
	e.CreatedAt = time.Now().Local()
	e.UpdatedAt = time.Now().Local()
	e.ID = uuid.New()
	return
}

type Asset struct {
	ID           uuid.UUID
	Symbol       string
	Name         string
	LogoURL      string
	Standard     string
	Instructions string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (a *Asset) BeforeCreate(tx *gorm.DB) (err error) {
	a.CreatedAt = time.Now().Local()
	a.UpdatedAt = time.Now().Local()
	a.ID = uuid.New()
	return
}

type Address struct {
	ID        uuid.UUID
	Address   string
	AssetID   uuid.UUID
	UserID    uuid.UUID
	IsActive  *bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (a *Address) BeforeCreate(tx *gorm.DB) (err error) {
	a.CreatedAt = time.Now().Local()
	a.UpdatedAt = time.Now().Local()
	a.ID = uuid.New()
	return
}

type Transaction struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	AssetID         uuid.UUID
	Type            string
	Amount          decimal.Decimal
	AmountUSD       float64
	Status          string
	Reference       string
	Hash            string
	SourceReference string
	Source          string
	Confirmations   int64
	RateID          uuid.UUID
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (a *Transaction) BeforeCreate(tx *gorm.DB) (err error) {
	a.CreatedAt = time.Now().Local()
	a.UpdatedAt = time.Now().Local()
	a.ID = uuid.New()
	return
}

type Wallet struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Balance   decimal.Decimal
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (w *Wallet) BeforeCreate(tx *gorm.DB) (err error) {
	w.CreatedAt = time.Now().Local()
	w.UpdatedAt = time.Now().Local()
	w.ID = uuid.New()
	return
}

type Withdrawal struct {
	ID            uuid.UUID `json:"id"`
	TransactionID uuid.UUID
	AccountNumber string
	BankName      string
	BankCode      string
	UserID        uuid.UUID
	Status        string
	Amount        decimal.Decimal
	Fee           int
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (w *Wallet) Withdrawal(tx *gorm.DB) (err error) {
	w.CreatedAt = time.Now().Local()
	w.UpdatedAt = time.Now().Local()
	w.ID = uuid.New()
	return
}

type WalletWithDetails struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Balance   float64   `json:"balance"`
	Status    string    `json:"status"`
	UserName  string    `json:"user_name"`
	UserEmail string    `json:"user_email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TransactionWithAsset struct {
	TransactionID string    `json:"transaction_id"`
	Reference     string    `json:"reference"`
	UserID        string    `json:"user_id"`
	AssetID       string    `json:"asset_id"`
	Type          string    `json:"type"`
	Amount        float64   `json:"amount"`
	AmountUSD     float64   `json:"amount_usd"`
	Confirmations int64     `json:"confirmations"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	AssetName     string    `json:"asset_name"`
	AssetSymbol   string    `json:"asset_symbol"`
	AssetStandard string    `json:"asset_standard"`
	Rate          float64   `json:"rate"`
}
