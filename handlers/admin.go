package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ShowBaba/kagewallet/common"
	"github.com/ShowBaba/kagewallet/services"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type AdminHandler struct {
	AdminService *services.AdminService
}

func NewAdminHandler(adminService *services.AdminService) *AdminHandler {
	return &AdminHandler{
		adminService,
	}
}

func (a *AdminHandler) CreateAsset() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		var input common.CreateAssetInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		if exists, err := a.AdminService.AssetExists(input.Name, input.Symbol, input.Standard); err != nil {
			http.Error(w, fmt.Sprintf("Error checking asset existence: %v", err), http.StatusInternalServerError)
			return
		} else if exists {
			http.Error(w, "Asset with the same name, symbol, and standard already exists", http.StatusConflict)
			return
		}

		if err := a.AdminService.CreateAsset(input); err != nil {
			http.Error(w, fmt.Sprintf("Failed to create asset: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "Asset created successfully"})
	}
}

func (a *AdminHandler) UpdateAssetHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		vars := mux.Vars(r)
		assetID, err := uuid.Parse(vars["id"])
		if err != nil {
			http.Error(w, "Invalid asset ID", http.StatusBadRequest)
			return
		}

		var updates map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		if err := a.AdminService.UpdateAsset(assetID, updates); err != nil {
			http.Error(w, fmt.Sprintf("Failed to update asset: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Asset updated successfully"})
	}
}

func (a *AdminHandler) CreateRate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		var input struct {
			Rate float64 `json:"rate"`
		}

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		if err := a.AdminService.CreateRate(input.Rate, "Admin"); err != nil {
			http.Error(w, fmt.Sprintf("Failed to create exchange rate: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "Exchange rate created successfully"})
	}
}

func (a *AdminHandler) ValidateMonnifyTransferOTP() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		var input struct {
			OTP       string `json:"otp"`
			Reference string `json:"reference"`
		}

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		if err := a.AdminService.ValidateMonnifyTransferOTP(input.Reference, input.OTP); err != nil {
			http.Error(w, fmt.Sprintf("Failed to validate otp: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "OTP validated successfully"})
	}
}

func (a *AdminHandler) GetAssets() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		activeParam := r.URL.Query().Get("active")
		fmt.Println("activeParam:", activeParam)
		var active bool
		var err error
		if activeParam != "" {
			active, err = strconv.ParseBool(activeParam)
			if err != nil {
				http.Error(w, "Invalid value for 'active' parameter", http.StatusBadRequest)
				return
			}
		}

		assets, err := a.AdminService.GetAssets(active)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to fetch assets: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(assets); err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode assets: %v", err), http.StatusInternalServerError)
			return
		}
	}
}
