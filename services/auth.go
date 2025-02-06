package services

import (
	"fmt"
	"strings"

	"github.com/ShowBaba/kagewallet/common"
	"github.com/ShowBaba/kagewallet/helpers"
	"github.com/ShowBaba/kagewallet/repositories"
)

type AuthService struct {
	UserRepo *repositories.UserRepository
}

func NewAuthService(userRepo *repositories.UserRepository) *AuthService {
	return &AuthService{UserRepo: userRepo}
}

func (a *AuthService) SetPassword(input common.SetPasswordInput) error {
	hashedPassword, err := helpers.HashPassword(strings.TrimSpace(input.Password))
	if err != nil {
		return err
	}
	return a.UserRepo.UpdatePassword(input.UserID, hashedPassword)
}

func (a *AuthService) ConfirmPassword(userID string, inputPassword string) (bool, error) {
	user, err := a.UserRepo.FindOneByID(userID)
	if err != nil {
		return false, fmt.Errorf("failed to retrieve user password: %w", err)
	}
	isValid := helpers.CheckPasswordHash(strings.TrimSpace(inputPassword), user.PasswordHash)
	return isValid, nil
}
