package services

import (
	"github.com/ShowBaba/kagewallet/database"
	"github.com/ShowBaba/kagewallet/repositories"
)

type RateService struct {
	RateRepo *repositories.RateRepository
}

func NewRateService(rateRepo *repositories.RateRepository) *RateService {
	return &RateService{RateRepo: rateRepo}
}

func (r *RateService) GetCurrentRate() (*database.Rate, error) {
	return r.RateRepo.GetLatestRate()
}
