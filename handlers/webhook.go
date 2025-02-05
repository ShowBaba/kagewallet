package handlers

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"go.uber.org/zap"
	"kagewallet/common"
	log "kagewallet/logging"
	"kagewallet/services"
)

type WebhookHandler struct {
	WebhookService *services.WebhookService
}

func NewWebhookHandler(webhookService *services.WebhookService) *WebhookHandler {
	return &WebhookHandler{
		webhookService,
	}
}

func (wb *WebhookHandler) BlockradarWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		fmt.Println("body; ", string(body))

		var input common.BlockradarEvent
		if err := json.Unmarshal(body, &input); err != nil {
			log.Error("Failed to parse request body", zap.Error(err))
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			// w.WriteHeader(http.StatusOK)
			// fmt.Fprintln(w, "OK")
			return
		}

		signature := r.Header.Get("x-blockradar-signature")
		if signature == "" {
			http.Error(w, "Missing signature header", http.StatusUnauthorized)
			return
		}

		var apiKey string

		fmt.Println("signature; ", signature)

		switch input.Data.Blockchain.TokenStandard {
		case "ERC20":
			apiKey = os.Getenv("BLOCKRADAR_ETH_API_KEY")
		case "TRC20":
			apiKey = os.Getenv("BLOCKRADAR_TRON_API_KEY")
		case "BEP20":
			apiKey = os.Getenv("BLOCKRADAR_BNB_API_KEY")
		}

		hash := hmac.New(sha512.New, []byte(apiKey))
		hash.Write(body)
		computedSignature := hex.EncodeToString(hash.Sum(nil))

		if computedSignature != signature {
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}

		if err := wb.WebhookService.BlockradarWebhook(input); err != nil {
			log.Error("error processing blockradar webhook", zap.Error(err))
			http.Error(w, "Failed to process webhook", http.StatusInternalServerError)
			// w.WriteHeader(http.StatusOK)
			// fmt.Fprintln(w, "OK")
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	}
}

func (wb *WebhookHandler) MonnifyWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		fmt.Println("body; ", string(body))

		var input common.MonnifyEvent
		if err := json.Unmarshal(body, &input); err != nil {
			log.Error("Failed to parse request body", zap.Error(err))
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			// w.WriteHeader(http.StatusOK)
			// fmt.Fprintln(w, "OK")
			return
		}

		if err := wb.WebhookService.MonnifyWebhook(input); err != nil {
			log.Error("error processing monnify webhook", zap.Error(err))
			http.Error(w, "Failed to process webhook", http.StatusInternalServerError)
			// w.WriteHeader(http.StatusOK)
			// fmt.Fprintln(w, "OK")
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	}
}
