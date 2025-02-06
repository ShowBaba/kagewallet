package routes

import (
	"github.com/ShowBaba/kagewallet/handlers"
	"github.com/ShowBaba/kagewallet/helpers"
	"github.com/ShowBaba/kagewallet/repositories"
	"github.com/ShowBaba/kagewallet/services"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func RegisterAdminRoutes(router *mux.Router, db *gorm.DB) {
	var (
		rateRepo       = repositories.NewRateRepository(db)
		assetRepo      = repositories.NewAssetRepository(db)
		monnifyService = services.NewMonnifyService()
		adminService   = services.NewAdminService(rateRepo, assetRepo, monnifyService)
		adminHandler   = handlers.NewAdminHandler(adminService)
	)
	apiRouter := router.PathPrefix("/api/admin").Subrouter()
	apiRouter.HandleFunc("/create_asset", helpers.ValidateAdminToken(adminHandler.CreateAsset())).Methods("POST")
	apiRouter.HandleFunc("/update_asset/{id}", helpers.ValidateAdminToken(adminHandler.UpdateAssetHandler())).Methods("PATCH")
	apiRouter.HandleFunc("/create_rate", helpers.ValidateAdminToken(adminHandler.CreateRate())).Methods("POST")
	apiRouter.HandleFunc("/validate_monnify_otp", helpers.ValidateAdminToken(adminHandler.ValidateMonnifyTransferOTP())).Methods("POST")
	apiRouter.HandleFunc("/get_assets", helpers.ValidateAdminToken(adminHandler.GetAssets())).Methods("GET")
}
