package jobs

import (
	log "kagewallet/logging"
	"kagewallet/repositories"
)

type Job struct {
	AddressRepo *repositories.AddressRepository
	UserRepo    *repositories.UserRepository
}

func NewJob(addressRepo *repositories.AddressRepository, userRepo *repositories.UserRepository) *Job {
	return &Job{
		addressRepo,
		userRepo,
	}
}

func (j *Job) Start() {
	log.Info("Starting job...")
	go ListenForNotifications()
	select {}
}
