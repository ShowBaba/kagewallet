package routes

import (
	"github.com/ShowBaba/kagewallet/handlers"
	"github.com/ShowBaba/kagewallet/repositories"
	"github.com/ShowBaba/kagewallet/services"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func RegisterWebhookRoutes(router *mux.Router, db *gorm.DB) {
	var (
		addressRepo     = repositories.NewAddressRepository(db)
		transactionRepo = repositories.NewTransactionRepository(db)
		walletRepo      = repositories.NewWalletRepository(db)
		rateRepo        = repositories.NewRateRepository(db)
		assetRepo       = repositories.NewAssetRepository(db)
		rateService     = services.NewRateService(rateRepo)
		withdrawalRepo  = repositories.NewWithdrawalRepository(db)
		webhookService  = services.NewWebhookService(addressRepo, transactionRepo, walletRepo, assetRepo, withdrawalRepo, rateService)
		webhookHandler  = handlers.NewWebhookHandler(webhookService)
		apiRouter       = router.PathPrefix("/api/webhook").Subrouter()
	)
	apiRouter.HandleFunc("/blockradar", webhookHandler.BlockradarWebhook()).Methods("POST")
	apiRouter.HandleFunc("/monnify", webhookHandler.MonnifyWebhook()).Methods("POST")
}
