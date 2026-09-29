package main

import (
	"log"
	"rdmarket-intelligence/backend/internal/config"
	"rdmarket-intelligence/backend/internal/handlers"
	"rdmarket-intelligence/backend/internal/middleware"
	"rdmarket-intelligence/backend/internal/providers"
	"rdmarket-intelligence/backend/internal/repositories"
	"rdmarket-intelligence/backend/internal/services"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func main() {
	cfg := config.LoadConfig()
	log.Printf("[RdMarket] Starting RdMarket Intelligence Backend (Env: %s)", cfg.AppEnv)

	// Attempt connection to PostgreSQL
	var db *gorm.DB
	var err error

	gormCfg := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	}

	if cfg.DatabaseURL != "" {
		log.Printf("[RdMarket] Connecting to Database...")
		db, err = gorm.Open(postgres.Open(cfg.DatabaseURL), gormCfg)
		if err != nil {
			log.Printf("[RdMarket] Warning: Failed to connect to PostgreSQL: %v", err)
			log.Printf("[RdMarket] Falling back to local embedded SQLite storage for development...")
			db, err = gorm.Open(sqlite.Open("rdmarket_dev.db"), gormCfg)
			if err != nil {
				log.Fatalf("[RdMarket] Fatal: Could not initialize database: %v", err)
			}
		}
	} else {
		db, err = gorm.Open(sqlite.Open("rdmarket_dev.db"), gormCfg)
		if err != nil {
			log.Fatalf("[RdMarket] Fatal: Could not initialize fallback database: %v", err)
		}
	}

	// Initialize Repository & Run Migrations
	repo := repositories.NewExchangeRateRepository(db)
	if err := repo.AutoMigrate(); err != nil {
		log.Fatalf("[RdMarket] Failed to run database migrations: %v", err)
	}
	log.Printf("[RdMarket] Database migrations applied successfully")

	// Initialize Exchange Rate Provider
	var provider providers.ExchangeRateProvider
	if cfg.ExchangeRateAPIURL != "" {
		provider = providers.NewFrankfurterProvider(cfg.ExchangeRateAPIURL)
		log.Printf("[RdMarket] Using Provider: %s (%s)", provider.GetProviderName(), cfg.ExchangeRateAPIURL)
	} else {
		provider = providers.NewMockProvider(16280.0)
		log.Printf("[RdMarket] Using MockProvider (16280.0 base)")
	}

	// Initialize Service & Handlers
	service := services.NewMarketService(repo, provider, cfg)

	// Seed data if empty
	go func() {
		log.Printf("[RdMarket] Checking and seeding USD/IDR initial dataset...")
		if err := service.SeedInitialDataIfEmpty("USD/IDR"); err != nil {
			log.Printf("[RdMarket] Warning during initial seed: %v", err)
		} else {
			log.Printf("[RdMarket] USD/IDR dataset ready.")
		}
	}()

	// Periodic non-aggressive background refresh (every 1 hour)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			log.Printf("[RdMarket] Background sync: fetching latest USD/IDR rate...")
			_ = service.SyncExternalData("USD/IDR")
		}
	}()

	// Initialize Fiber App
	app := fiber.New(fiber.Config{
		AppName:      "RdMarket Intelligence API v1",
		ServerHeader: "Fiber",
	})

	middleware.SetupMiddleware(app)

	marketHandler := handlers.NewMarketHandler(service)
	marketHandler.RegisterRoutes(app)

	addr := ":" + cfg.Port
	log.Printf("[RdMarket] Server listening on %s", addr)
	if err := app.Listen(addr); err != nil {
		log.Fatalf("[RdMarket] Server shutdown error: %v", err)
	}
}
