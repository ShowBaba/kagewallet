package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ShowBaba/blockradar-go"
	"github.com/ShowBaba/kagewallet/common"
	"github.com/ShowBaba/kagewallet/database"
	"github.com/ShowBaba/kagewallet/helpers"
	"github.com/ShowBaba/kagewallet/repositories"
	"github.com/google/uuid"
)

type AddressService struct {
	UserRepo    *repositories.UserRepository
	AddressRepo *repositories.AddressRepository
	AssetRepo   *repositories.AssetRepository
}

func NewAddressService(userRepo *repositories.UserRepository,
	addressRepo *repositories.AddressRepository, assetRepo *repositories.AssetRepository) *AddressService {
	return &AddressService{
		userRepo,
		addressRepo,
		assetRepo,
	}
}

func (a *AddressService) GetUserAddress(user *database.User, assetId string) (*common.GenerateAddressResponse, error) {
	asset, err := a.AssetRepo.FindAssetByID(assetId)
	if err != nil {
		return nil, err
	}
	existingAddress, err := a.AddressRepo.GetLastActiveAddressByUser(user.ID, &assetId)
	if err != nil {
		return nil, err
	}
	if existingAddress == nil {
		addressData, err := generateNewAddress(user.ID.String(), asset)
		if err != nil {
			return nil, err
		}
		if err := a.AddressRepo.CreateAddress(&database.Address{
			UserID:   user.ID,
			AssetID:  uuid.MustParse(assetId),
			Address:  addressData.Address,
			IsActive: helpers.BoolPtr(true),
		}); err != nil {
			return nil, err
		}
		return addressData, nil
	} else {
		return &common.GenerateAddressResponse{
			Address:     existingAddress.Address,
			Instruction: asset.Instructions,
		}, nil
	}
}

func generateNewAddress(userID string, asset *database.Asset) (*common.GenerateAddressResponse, error) {
	switch strings.ToUpper(asset.Symbol) {
	case "USDC":
		switch asset.Standard {
		case "ERC20":
			data, err := createBlockradarWalletAddress("ETH", userID, asset.ID.String())
			if err != nil {
				return nil, err
			}
			return &common.GenerateAddressResponse{
				Address:     data.Data.Address,
				Instruction: asset.Instructions,
			}, nil
		}
	case "USDT":
		switch asset.Standard {
		case "TRC20":
			data, err := createBlockradarWalletAddress("TRON", userID, asset.ID.String())
			if err != nil {
				return nil, err
			}
			return &common.GenerateAddressResponse{
				Address:     data.Data.Address,
				Instruction: asset.Instructions,
			}, nil
		case "BEP20":
			data, err := createBlockradarWalletAddress("BNB", userID, asset.ID.String())
			if err != nil {
				return nil, err
			}
			return &common.GenerateAddressResponse{
				Address:     data.Data.Address,
				Instruction: asset.Instructions,
			}, nil
		}
	}
	return nil, fmt.Errorf("%s not supported currently", asset.Symbol)
}

func createBlockradarWalletAddress(walletName, userId, assetId string) (*CreateBlockradarAddressResponse, error) {
	var (
		walletID string
		apiKey   string
	)

	switch walletName {
	case "ETH":
		walletID = os.Getenv("BLOCKRADER_ETH_WALLET_ID")
		apiKey = os.Getenv("BLOCKRADAR_ETH_API_KEY")
	case "TRON":
		walletID = os.Getenv("BLOCKRADER_TRON_WALLET_ID")
		apiKey = os.Getenv("BLOCKRADAR_TRON_API_KEY")
	case "BNB":
		walletID = os.Getenv("BLOCKRADER_BNB_WALLET_ID")
		apiKey = os.Getenv("BLOCKRADAR_BNB_API_KEY")
	}

	userData := map[string]string{
		"user_id":  userId,
		"asset_id": assetId,
	}

	metadata, err := json.Marshal(userData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user data: %v", err)
	}

	config := blockradar.Config{
		APIKey: apiKey,
	}
	client := blockradar.NewClient(config)

	generateAddressReq := &blockradar.GenerateAddressesRequest{
		WalletID:              walletID,
		Name:                  fmt.Sprintf(`Kage:%s wallet`, walletName),
		DisableAutoSweep:      false,
		EnableGaslessWithdraw: true,
		ShowPrivateKey:        false,
		Metadata:              metadata,
	}
	generateAddressRes, err := client.Address.GenerateAddress(generateAddressReq)
	if err != nil {
		log.Fatalf("Failed to generate address: %v", err)
	}

	response := CreateBlockradarAddressResponse{}

	dataBytes, err := json.Marshal(generateAddressRes.Data)
	if err != nil {
		log.Fatalf("Failed to marshal response data: %v", err)
	}

	if err := json.Unmarshal(dataBytes, &response.Data); err != nil {
		log.Fatalf("Failed to unmarshal response data: %v", err)
	}

	return &response, nil
}

type CreateBlockradarAddressResponse struct {
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode"`
	Data       struct {
		Address        string      `json:"address"`
		Name           string      `json:"name"`
		Type           string      `json:"type"`
		DerivationPath string      `json:"derivationPath"`
		Metadata       interface{} `json:"metadata"`
		Configurations struct {
			AML struct {
				Provider string `json:"provider"`
				Status   string `json:"status"`
				Message  string `json:"message"`
			} `json:"aml"`
			ShowPrivateKey        bool `json:"showPrivateKey"`
			DisableAutoSweep      bool `json:"disableAutoSweep"`
			EnableGaslessWithdraw bool `json:"enableGaslessWithdraw"`
		} `json:"configurations"`
		Network    string `json:"network"`
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
		ID        string `json:"id"`
		IsActive  bool   `json:"isActive"`
		CreatedAt string `json:"createdAt"`
		UpdatedAt string `json:"updatedAt"`
	} `json:"data"`
}
