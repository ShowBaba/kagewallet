package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"

	"github.com/ShowBaba/kagewallet/bot"
	"github.com/ShowBaba/kagewallet/database"
	"github.com/ShowBaba/kagewallet/jobs"
	log "github.com/ShowBaba/kagewallet/logging"
	"github.com/ShowBaba/kagewallet/repositories"
	"github.com/ShowBaba/kagewallet/routes"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm"
)

var (
	db   *gorm.DB
	env  = os.Getenv("ENV")
	tBot *bot.TelegramBot
)

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Error("error loading .env file: %s", zap.Error(err))
	}
	dbConfig := database.Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		Password: os.Getenv("DB_PASSWORD"),
		User:     os.Getenv("DB_USER"),
		DBName:   os.Getenv("DB_NAME"),
	}

	if env != "dev" {
		basePath, _ := filepath.Abs("assets")
		dbConfig.SSLRootCert = fmt.Sprintf("%s/%s", basePath, "/ca.pem")
		dbConfig.Host = os.Getenv("DB_HOST_LIVE")
		dbConfig.DBName = os.Getenv("DB_NAME_LIVE")
		dbConfig.Port = os.Getenv("DB_PORT_LIVE")
		dbConfig.Password = os.Getenv("DB_PASSWORD_LIVE")
		dbConfig.User = os.Getenv("DB_USER_LIVE")
	}

	log.Info("postgres DB", zap.Any("config", dbConfig))

	db, err = database.ConnectPg(&dbConfig, env)
	if err != nil {
		log.Fatal(fmt.Sprintf("error connecting to postgres %s", dbConfig.Host), zap.Error(err))
	}

	err = database.InitializeRedis(os.Getenv("REDIS_ADDRESS"), os.Getenv("REDIS_PASSWORD"), 0)
	if err != nil {
		log.Fatal("error connecting to redis", zap.Error(err))
	}

	tBot, err = bot.NewTelegramBot(os.Getenv("TELEGRAM_TOKEN"), db)
	if err != nil {
		log.Fatal("error initializing telegram bot ", zap.Error(err))
	}
}

func main() {
	if log.Logger == nil {
		log.InitializeLogger(zapcore.InfoLevel)
	}
	// f, err := os.Create("cpu.prof")
	// if err != nil {
	// 	log.Error("could not create CPU profile: ", zap.Error(err))
	// }
	// err = pprof.StartCPUProfile(f)
	// if err != nil {
	// 	log.Error("error starting cpu profile", zap.Error(err))
	// }
	// defer pprof.StopCPUProfile()
	// go func() {
	// 	time.Sleep(30 * time.Second)
	// 	memProfile, err := os.Create("mem.prof")
	// 	if err != nil {
	// 		log.Error("could not create memory profile: ", zap.Error(err))
	// 		return
	// 	}
	// 	defer memProfile.Close()
	//
	// 	if err := pprof.WriteHeapProfile(memProfile); err != nil {
	// 		log.Error("could not write memory profile: ", zap.Error(err))
	// 	}
	// }()

	router := mux.NewRouter()
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})
	router.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)

	routes.RegisterAdminRoutes(router, db)
	routes.RegisterWebhookRoutes(router, db)

	go func() {
		port := fmt.Sprintf(":%s", os.Getenv("PORT"))
		if env == "dev" {
			port = ":8000"
		}
		log.Info(fmt.Sprintf("Listening on %v", port))
		if err := http.ListenAndServe(port, router); err != nil {
			log.Fatal("error starting server", zap.Error(err))
		}
	}()

	go func() {
		var (
			addressRepo = repositories.NewAddressRepository(db)
			userRepo    = repositories.NewUserRepository(db)
			jobService  = jobs.NewJob(addressRepo, userRepo)
		)
		jobService.Start()
	}()

	if env == "dev" {
		tBot.ListenForUpdates()
	} else {
		router.HandleFunc("/webhook", tBot.Webhook)
	}

	select {}

}
