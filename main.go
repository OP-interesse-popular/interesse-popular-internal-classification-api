package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/OP-interesse-popular/interesse-popular-internal-classification/app"
	"github.com/OP-interesse-popular/interesse-popular-internal-classification/clients"
	"github.com/OP-interesse-popular/interesse-popular-internal-classification/config"
	"github.com/OP-interesse-popular/interesse-popular-internal-classification/database"
	"github.com/OP-interesse-popular/interesse-popular-internal-classification/middlewares"
	"github.com/OP-interesse-popular/interesse-popular-internal-classification/repositories"
	"github.com/OP-interesse-popular/interesse-popular-internal-classification/scheduler"
	"github.com/OP-interesse-popular/interesse-popular-internal-classification/services"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		panic(err)
	}
	cfg := config.Load()
	db, err := database.Connect(cfg.DBConnectionString)
	if err != nil {
		panic(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	defer sqlDB.Close()
	if err := sqlDB.Ping(); err != nil {
		panic(err)
	}
	if cfg.AutoMigrate {
		if err := database.Migrate(db); err != nil {
			panic(err)
		}
	}

	cache, err := repositories.NewParliamentarianCache(cfg.RedisURL, cfg.CacheTTL)
	if err != nil {
		panic(err)
	}
	if err := cache.Ping(context.Background()); err != nil {
		panic(err)
	}
	httpClient := &http.Client{Timeout: cfg.ExternalHTTPTimeout}
	camaraClient, err := clients.NewCamaraHTTPClient(cfg.CamaraBaseURL, httpClient)
	if err != nil {
		panic(err)
	}
	senadoClient, err := clients.NewSenadoHTTPClient(cfg.SenadoBaseURL, httpClient)
	if err != nil {
		panic(err)
	}
	partyRepository := repositories.NewPartyPoliticalDirection(db)
	synchronizer := services.NewParliamentarianSync(cache, camaraClient, senadoClient, partyRepository)
	scheduler.NewParliamentarianRefresh(synchronizer, cfg.CacheCheckInterval, cfg.RefreshInterval).Start(context.Background())

	middlewares.ConfigureLogger()
	router := gin.New()
	router.Use(middlewares.RequestID(), middlewares.SanitizedLogger(), middlewares.SanitizedRecovery())
	if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		panic(err)
	}
	application := app.New(router, db, cfg, synchronizer)
	application.RegisterRoutes()
	fmt.Printf("Starting internal classification API on port :%s\n", cfg.Port)
	if err := application.Router.Run(":" + cfg.Port); err != nil {
		panic(err)
	}
}
