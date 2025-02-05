package repositories

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"kagewallet/database"
)

type RateRepository struct {
	DB *gorm.DB
}

func NewRateRepository(db *gorm.DB) *RateRepository {
	return &RateRepository{
		DB: db,
	}
}

func (r *RateRepository) AddNewRate(rate float64, source string) error {
	newRate := database.Rate{
		ID:        uuid.New(),
		Rate:      rate,
		Source:    source,
		CreatedAt: time.Now(),
	}

	return r.DB.Create(&newRate).Error
}

func (r *RateRepository) GetLatestRate() (*database.Rate, error) {
	var rate database.Rate
	err := r.DB.Order("created_at DESC").First(&rate).Error
	if err != nil {
		return nil, err
	}
	return &rate, nil
}

// func (r *RateRepository) GetLatestRatesForActiveAssets() ([]struct {
// 	Symbol   string
// 	Name     string
// 	Rate     float64
// 	Standard string
// }, error) {
// 	var results []struct {
// 		Symbol   string
// 		Name     string
// 		Rate     float64
// 		Standard string
// 	}
//
// 	err := r.DB.Raw(`
// 		SELECT a.symbol, a.name, a.standard, r.rate
// 		FROM asset a
// 		JOIN (
// 			SELECT asset_id, MAX(created_at) AS max_date
// 			FROM rate
// 			GROUP BY asset_id
// 		) latest ON a.id = latest.asset_id
// 		JOIN rate r ON latest.asset_id = r.asset_id AND latest.max_date = r.created_at
// 		WHERE a.is_active = true
// 	`).Scan(&results).Error
//
// 	return results, err
// }
