package jobs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"

	log "github.com/ShowBaba/kagewallet/logging"
	"go.uber.org/zap"
)

func (j *Job) fetchBlockradarTransactions() error {
	var (
		totalWorkers    = 200
		wRequestCh      = make(chan workerRequest)
		errorCh         = make(chan error, totalWorkers)
		wg              = &sync.WaitGroup{}
		transactionChan = make(chan BlockradarTransaction)
		wallets         = []string{
			os.Getenv("BLOCKRADER_ETH_WALLET_ID"),
		}
	)

	for i := 0; i < totalWorkers; i++ {
		worker := &worker{
			id:        i,
			waitGroup: wg,
			RequestCh: wRequestCh,
			errorCh:   errorCh,
		}
		go worker.run()
		go func(workerID int, errorCh chan error) {
			for err := range errorCh {
				log.Error(fmt.Sprintf("Error from worker: %v", workerID), zap.Error(err))
			}
		}(i, errorCh)
	}

	go func() {
		defer wg.Done()
		wg.Add(1)
		for tnx := range transactionChan {
			addressData, err := j.AddressRepo.GetAddressByColumn("address", tnx.RecipientAddress)
			if err != nil {
				errorCh <- err
				continue
			}
			if addressData != nil {
				wRequestCh <- workerRequest{
					BlockradarTransaction: tnx,
				}
			}
		}
	}()

	for _, wallet := range wallets {
		wg.Add(1)
		go func(wallet string) {
			defer wg.Done()
			for {
				var (
					page  = 1
					count int
				)

				url := fmt.Sprintf("https://api.blockradar.co/v1/wallets/%s/transactions?page=%d", wallet, page)

				req, err := http.NewRequest("GET", url, nil)
				if err != nil {
					errorCh <- fmt.Errorf("failed to create HTTP request: %v", err)
					continue
				}

				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("x-api-key", os.Getenv("BLOCKRADAR_ETH_API_KEY"))

				client := &http.Client{}
				resp, err := client.Do(req)
				if err != nil {
					errorCh <- fmt.Errorf("failed to send HTTP request: %v", err)
					continue
				}
				// defer resp.Body.Close()

				respBody, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					errorCh <- fmt.Errorf("failed to read response body: %v", err)
					continue
				}
				var response FetchTransactionResponse
				if err := json.Unmarshal(respBody, &response); err != nil {
					errorCh <- fmt.Errorf("failed to unmarshal response: %v", err)
					continue
				}

				for _, transaction := range response.Data {
					transactionChan <- transaction
				}

				log.Info("processed ", zap.Int("", count))

				if len(response.Data) < 1 {
					break
				}
				page += 1
				count += len(response.Data)
			}
		}(wallet)
	}

	wg.Wait()
	return nil
}

type workerRequest struct {
	BlockradarTransaction BlockradarTransaction
}

type worker struct {
	id        int
	waitGroup *sync.WaitGroup
	RequestCh chan workerRequest
	errorCh   chan error
}

func (w *worker) run() {
	log.Info(fmt.Sprintf(`starting worker; %v`, w.id))
	w.waitGroup.Add(1)
	defer w.waitGroup.Done()
	for task := range w.RequestCh {
		address := task.BlockradarTransaction.RecipientAddress
		fmt.Println(address)
		// TODO: process transaction is it doesn't exist (or status is pending), avoid race conditions with the webhook
	}
}

type FetchTransactionResponse struct {
	Message    string                  `json:"message"`
	StatusCode int                     `json:"statusCode"`
	Data       []BlockradarTransaction `json:"data"`
	Analytics  struct {
		DepositsSumIn24Hours       int64
		WithdrawsSumIn24Hours      int64
		TransactionsSumIn24Hours   int64
		DepositsCountIn24Hours     int64
		WithdrawsCountIn24Hours    int64
		TransactionsCountIn24Hours int64
		TotalDepositsCount         int64
		TotalWithdrawsCount        int64
		TotalTransactionsCount     int64
	} `json:"analytics"`

	Meta struct {
		TotalItems   int64
		ItemCount    int64
		ItemsPerPage int64
		TotalPages   int64
		CurrentPage  int64
	} `json:"meta"`
}

type BlockradarTransaction struct {
	ID               string  `json:"id"`
	Reference        string  `json:"reference"`
	SenderAddress    string  `json:"senderAddress"`
	RecipientAddress string  `json:"recipientAddress"`
	Amount           string  `json:"amount"`
	AmountPaid       string  `json:"amountPaid"`
	Fee              *string `json:"fee"`
	Currency         string  `json:"currency"`
	BlockNumber      int     `json:"blockNumber"`
	BlockHash        string  `json:"blockHash"`
	Hash             string  `json:"hash"`
	Confirmations    int     `json:"confirmations"`
	Confirmed        bool    `json:"confirmed"`
	GasPrice         string  `json:"gasPrice"`
	GasUsed          string  `json:"gasUsed"`
	GasFee           string  `json:"gasFee"`
	Status           string  `json:"status"`
	Type             string  `json:"type"`
	Note             *string `json:"note"`
	AmlScreening     struct {
		Provider string `json:"provider"`
		Status   string `json:"status"`
		Message  string `json:"message"`
	} `json:"amlScreening"`
	AssetSwept                 bool   `json:"assetSwept"`
	AssetSweptAt               string `json:"assetSweptAt"`
	AssetSweptGasFee           string `json:"assetSweptGasFee"`
	AssetSweptHash             string `json:"assetSweptHash"`
	AssetSweptSenderAddress    string `json:"assetSweptSenderAddress"`
	AssetSweptRecipientAddress string `json:"assetSweptRecipientAddress"`
	AssetSweptAmount           string `json:"assetSweptAmount"`
	Reason                     string `json:"reason"`
	Network                    string `json:"network"`
	ChainID                    int    `json:"chainId"`
	Metadata                   interface{}
	CreatedAt                  string `json:"createdAt"`
	UpdatedAt                  string `json:"updatedAt"`
	Beneficiary                *string
	PaymentLink                *string
	Customer                   *string
}
