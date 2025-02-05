package helpers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/badoux/checkmail"
	"golang.org/x/crypto/bcrypt"
)

func DivideFromSymbol(word string, symbol string) string {
	formattedWord, _, _ := strings.Cut(word, symbol)
	return formattedWord
}

func TimeDiff(endDateStr string) int64 {
	endDate, err := time.Parse(time.RFC3339, endDateStr)
	if err != nil {
		fmt.Println("Error parsing end date:", err)
		return 0
	}

	currentDate := time.Now().UTC()

	timeDifference := endDate.Sub(currentDate).Seconds()

	if timeDifference <= 0 {
		return 0
	}
	return int64(timeDifference)
}

func StringInSlice(ss []string, s string) bool {
	for _, el := range ss {
		if el == s {
			return true
		}
	}

	return false
}

func HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedBytes), nil
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func IsValidEmail(email string) bool {
	err := checkmail.ValidateFormat(email)
	if err != nil {
		return false
	}
	return true
}

func StrPtr(value string) *string {
	return &value
}

func BoolPtr(b bool) *bool {
	return &b
}

func GenerateTransactionReference() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

func GenerateRandomHash(input string) (string, error) {
	hash := sha256.New()
	hash.Write([]byte(input))
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	hash.Write(randomBytes)
	finalHash := hash.Sum(nil)
	return "0x" + hex.EncodeToString(finalHash), nil
}
