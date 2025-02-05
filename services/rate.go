package services

import (
	"kagewallet/database"
	"kagewallet/repositories"
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
