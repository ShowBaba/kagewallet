package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ShowBaba/kagewallet/helpers"
	log "github.com/ShowBaba/kagewallet/logging"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/ShowBaba/kagewallet/common"
	"github.com/ShowBaba/kagewallet/database"
)

type MonnifyService struct {
}

func NewMonnifyService() *MonnifyService {
	initialize()
	return &MonnifyService{}
}

var (
	baseURL             string
	apiKey              string
	secretKey           string
	sourceAccountNumber string
	banks               []Bank
	HTTPClient          *http.Client
)

func (m *MonnifyService) getAuthToken() (string, error) {
	token, err := database.GetRedisKey(common.RedisMonnifyToken)
	if err == nil && token != "" {
		return token, nil
	}
	authString := fmt.Sprintf("%s:%s", apiKey, secretKey)
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(authString))

	req, err := http.NewRequest("POST", fmt.Sprintf(`%s/api/v1/auth/login`, baseURL), bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+encodedAuth)
	resp, err := HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to authenticate: %s", resp.Status)
	}

	var authResponse AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResponse); err != nil {
		return "", err
	}
	err = database.SetRedisKey(common.RedisMonnifyToken, authResponse.ResponseBody.AccessToken, 50*time.Minute)
	return authResponse.ResponseBody.AccessToken, err
}

func (m *MonnifyService) GetBanks(page, limit int) (paginatedBanks []Bank, totalPages int, err error) {
	if len(banks) == 0 {
		if err := loadBanks(); err != nil {
			return nil, 0, fmt.Errorf("failed to load banks: %w", err)
		}
	}
	if page < 1 || limit < 1 {
		return nil, 0, fmt.Errorf("invalid page or limit")
	}
	totalBanks := len(banks)
	totalPages = int(math.Ceil(float64(totalBanks) / float64(limit)))

	if page > totalPages {
		return nil, totalPages, fmt.Errorf("page out of range")
	}
	start := (page - 1) * limit
	end := start + limit
	if start > totalBanks {
		return nil, totalPages, fmt.Errorf("no banks on this page")
	}
	if end > totalBanks {
		end = totalBanks
	}
	paginatedBanks = banks[start:end]
	return paginatedBanks, totalPages, nil
}

func (m *MonnifyService) GetBankByCode(code string) (Bank, error) {
	if len(banks) == 0 {
		if err := loadBanks(); err != nil {
			return Bank{}, fmt.Errorf("failed to load banks: %w", err)
		}
	}
	for _, bank := range banks {
		if bank.Code == code {
			return bank, nil
		}
	}
	return Bank{}, fmt.Errorf("bank with code %s not found", code)
}

func (m *MonnifyService) SearchBank(query string, page int, limit int) ([]Bank, int, error) {
	if len(banks) == 0 {
		if err := loadBanks(); err != nil {
			return nil, 0, fmt.Errorf("failed to load banks: %w", err)
		}
	}
	var filteredBanks []Bank
	query = strings.ToLower(query)

	for _, bank := range banks {
		if strings.Contains(strings.ToLower(bank.Name), query) {
			filteredBanks = append(filteredBanks, bank)
		}
	}
	totalBanks := len(filteredBanks)
	totalPages := int(math.Ceil(float64(totalBanks) / float64(limit)))

	if page > totalPages {
		return nil, totalPages, fmt.Errorf("page out of range")
	}
	start := (page - 1) * limit
	end := start + limit
	if start > totalBanks {
		return nil, totalPages, fmt.Errorf("no banks on this page")
	}
	if end > totalBanks {
		end = totalBanks
	}
	paginatedBanks := filteredBanks[start:end]
	return paginatedBanks, totalPages, nil
}

func (m *MonnifyService) ValidateBankAccount(accountNumber, bankCode string) (*AccountDetails, error) {
	url := fmt.Sprintf(`%s/api/v1/disbursements/account/validate?accountNumber=%v&bankCode=%v`, baseURL, accountNumber, bankCode)
	token, err := m.getAuthToken()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to validate bank account: %s", resp.Status)
	}

	var response ValidateBankResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return &AccountDetails{
		AccountNumber: response.ResponseBody.AccountNumber,
		AccountName:   response.ResponseBody.AccountName,
		BankCode:      response.ResponseBody.BankCode,
	}, nil
}

func (m *MonnifyService) InitiateTransfer(amount decimal.Decimal, bankCode, bankAccountNumber string) (string, interface{}, error) {
	url := fmt.Sprintf(`%s/api/v2/disbursements/single`, baseURL)
	token, err := m.getAuthToken()
	if err != nil {
		return "", nil, err
	}

	fmt.Println("token; ", token)

	reference := helpers.GenerateTransactionReference()
	var jsonStr = []byte(fmt.Sprintf(`
	{
		"amount": %v,
		"reference":"%s",
		"narration":"trf",
		"destinationBankCode": "%s",
		"destinationAccountNumber": "%s",
		"currency": "NGN",
		"sourceAccountNumber": "%s"
	})`, amount, reference, bankCode, bankAccountNumber, sourceAccountNumber))

	fmt.Println("jsonStr; ", string(jsonStr))
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		return "", nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}

	fmt.Println(string(body))

	fmt.Println("body; ", resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("failed to validate bank account: %s", resp.Status)
	}

	var response InitiateTransferResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", nil, fmt.Errorf("error unmarshalling JSON: %s", err)
	}
	return reference, response, nil
}

func (m *MonnifyService) ValidateTransferOTP(reference, otp string) error {
	url := fmt.Sprintf(`%s/api/v2/disbursements/single/validate-otp`, baseURL)
	token, err := m.getAuthToken()
	if err != nil {
		return err
	}

	var jsonStr = []byte(fmt.Sprintf(`
	{
		"reference":"%s",
		"authorizationCode": "%s"
	})`, reference, otp))

	fmt.Println("jsonStr; ", string(jsonStr))
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Println("body; ", string(body))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to validate bank account: %s", resp.Status)
	}

	var response InitiateTransferResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("error unmarshalling JSON: %s", err)
	}
	return nil
}

func loadBanks() error {
	// TODO: move bank codes data to redis for ease of update
	log.Info("bank codes loaded")
	basePath, _ := filepath.Abs("assets")
	data, err := os.ReadFile(fmt.Sprintf("%s/%s", basePath, "/monnify_banks.json"))
	if err != nil {
		return err
	}
	var banksData []Bank
	if err := json.Unmarshal(data, &banksData); err != nil {
		return err
	}
	banks = banksData
	return nil
}

type AccountDetails struct {
	AccountNumber string
	AccountName   string
	BankCode      string
}

type Bank struct {
	Name                 string `json:"name"`
	Code                 string `json:"code"`
	UssdTemplate         string `json:"ussdTemplate"`
	BaseUssdCode         string `json:"baseUssdCode"`
	TransferUssdTemplate string `json:"transferUssdTemplate"`
}

type AuthResponse struct {
	RequestSuccessful bool   `json:"requestSuccessful"`
	ResponseMessage   string `json:"responseMessage"`
	ResponseCode      string `json:"responseCode"`
	ResponseBody      struct {
		AccessToken string `json:"accessToken"`
		ExpiresIn   int    `json:"expiresIn"`
	} `json:"responseBody"`
}

type ValidateBankResponse struct {
	RequestSuccessful bool   `json:"requestSuccessful"`
	ResponseMessage   string `json:"responseMessage"`
	ResponseCode      string `json:"responseCode"`
	ResponseBody      struct {
		AccountNumber string `json:"accountNumber"`
		AccountName   string `json:"accountName"`
		BankCode      string `json:"bankCode"`
	} `json:"responseBody"`
}

type InitiateTransferResponse struct {
	RequestSuccessful bool   `json:"requestSuccessful"`
	ResponseMessage   string `json:"responseMessage"`
	ResponseCode      string `json:"responseCode"`
	ResponseBody      struct {
		Amount                   float64   `json:"amount"`
		Reference                string    `json:"reference"`
		Status                   string    `json:"status"`
		DateCreated              time.Time `json:"dateCreated"`
		TotalFee                 float64   `json:"totalFee"`
		DestinationAccountName   string    `json:"destinationAccountName"`
		DestinationBankName      string    `json:"destinationBankName"`
		DestinationAccountNumber string    `json:"destinationAccountNumber"`
		DestinationBankCode      string    `json:"destinationBankCode"`
	} `json:"responseBody"`
}

func initialize() {
	HTTPClient = &http.Client{}

	err := loadBanks()
	if err != nil {
		log.Error("error loading bank codes", zap.Error(err))
	}
	baseURL = "https://api.monnify.com"
	apiKey = os.Getenv("MONNIFY_API_KEY_LIVE")
	secretKey = os.Getenv("MONNIFY_SECRET_KEY_LIVE")
	sourceAccountNumber = os.Getenv("MONNIFY_SOURCE_ACCOUNT_NUMBER_LIVE")
	if os.Getenv("ENV") == "dev" {
		baseURL = "https://sandbox.monnify.com"
		apiKey = os.Getenv("MONNIFY_API_KEY_TEST")
		secretKey = os.Getenv("MONNIFY_SECRET_KEY_TEST")
		sourceAccountNumber = os.Getenv("MONNIFY_SOURCE_ACCOUNT_NUMBER_TEST")
	}
}
