package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"rdmarket-intelligence/backend/internal/config"
	"rdmarket-intelligence/backend/internal/economic"
	"rdmarket-intelligence/backend/internal/events"
	"rdmarket-intelligence/backend/internal/forecasting"
	"rdmarket-intelligence/backend/internal/handlers"
	"rdmarket-intelligence/backend/internal/middleware"
	"rdmarket-intelligence/backend/internal/migrations"
	"rdmarket-intelligence/backend/internal/providers"
	"rdmarket-intelligence/backend/internal/repositories"
	"rdmarket-intelligence/backend/internal/services"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("[RdMarket] Configuration error (fail fast): %v", err)
	}

	// Subcommand handling
	if len(os.Args) > 1 {
		subcommand := os.Args[1]
		switch subcommand {
		case "migrate-up":
			runMigrateUp(cfg)
			return
		case "migrate-down":
			runMigrateDown(cfg)
			return
		case "seed-dev":
			runSeedDev(cfg)
			return
		case "help", "--help", "-h":
			fmt.Println("Usage: server [migrate-up | migrate-down | seed-dev]")
			return
		default:
			log.Fatalf("[RdMarket] Unknown command %q. Valid commands: migrate-up, migrate-down, seed-dev", subcommand)
		}
	}

	// Startup guard rail verification
	if cfg.AppEnv == "production" && cfg.Provider == "mock" {
		log.Fatalf("[RdMarket] Guard rail violation: MockProvider is strictly forbidden in production")
	}

	log.Printf("[RdMarket] Starting RdMarket Intelligence Backend (Env: %s)", cfg.AppEnv)
	log.Printf("[RdMarket] Loaded Config:\n%s", cfg.RedactedDump())

	// Connect to PostgreSQL
	db, sqlDB, err := initDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("[RdMarket] Failed to connect to database: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	// Apply migrations on startup
	if err := migrations.Up(sqlDB); err != nil {
		log.Fatalf("[RdMarket] Migration error on startup: %v", err)
	}
	log.Printf("[RdMarket] Database schema migrations up to date")

	// Initialize Repository
	repo := repositories.NewExchangeRateRepository(db)

	// Initialize Provider
	var provider providers.ExchangeRateProvider
	switch cfg.Provider {
	case "frankfurter":
		provider = providers.NewFrankfurterProvider(cfg.ExchangeRateAPIURL)
		log.Printf("[RdMarket] Active provider: %s (%s)", provider.GetProviderName(), cfg.ExchangeRateAPIURL)
	case "csv":
		provider = providers.NewCsvImportProvider()
		log.Printf("[RdMarket] Active provider: %s", provider.GetProviderName())
	default:
		provider = providers.NewMockProvider(16280.0)
		log.Printf("[RdMarket] Active provider: %s (Simulated data banner active)", provider.GetProviderName())
	}

	// Initialize Service
	service := services.NewMarketService(repo, provider, cfg)
	dispatcher := events.NewInProcessDispatcher()
	service.SetEventDispatcher(dispatcher)
	economicRepository := economic.NewRepository(db)
	catalog, catalogErr := economic.LoadCatalog()
	if catalogErr != nil {
		log.Printf("event=economic_catalog_load_failed error=%q", catalogErr.Error())
	} else {
		if err := economicRepository.SeedCatalog(context.Background(), catalog); err != nil {
			log.Printf("event=economic_catalog_seed_failed error=%q", err.Error())
		} else if cfg.EconomicRefreshEnabled {
			ingestionService := economic.NewIngestionService(
				economicRepository,
				map[string]economic.EconomicDataProvider{"bls_api": economic.NewBLSProvider(cfg.EconomicBLSAPIKey)},
				dispatcher,
			)
			ingestionService.Run(context.Background(), catalog, time.Duration(cfg.EconomicRefreshHours)*time.Hour)
		}
	}
	if err := service.SeedInitialDataIfEmpty("USD/IDR"); err != nil {
		log.Printf("[RdMarket] Initial market-data sync unavailable: %v", err)
	}
	go func() {
		ticker := time.NewTicker(time.Duration(cfg.IngestionIntervalMinutes) * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if err := service.SyncExternalData("USD/IDR"); err != nil {
				log.Printf("[RdMarket] Scheduled market-data sync failed: %v", err)
			}
		}
	}()

	// Setup Fiber App
	app := fiber.New(fiber.Config{
		AppName:      "RdMarket Intelligence API v1",
		ServerHeader: "RdMarket",
	})

	// Setup Middlewares
	middleware.SetupMiddleware(app, cfg)

	// Register Auth Routes
	authHandler := middleware.NewAuthHandler(cfg)
	apiV1 := app.Group("/api/v1")
	authHandler.RegisterRoutes(apiV1)
	economic.NewHTTPHandler(economic.NewAPIService(economicRepository, catalog)).RegisterRoutes(apiV1, authHandler)
	forecastClient := forecasting.NewHTTPForecastingClient(cfg.ForecastServiceURL, cfg.ForecastInternalToken)
	forecastService := forecasting.NewService(db, forecastClient, cfg)
	forecasting.NewHandler(forecastService, authHandler).RegisterRoutes(apiV1)

	// Register Market Routes
	marketHandler := handlers.NewMarketHandler(service)
	marketHandler.RegisterRoutes(app)

	addr := ":" + cfg.Port
	log.Printf("[RdMarket] Server listening on %s", addr)
	if err := app.Listen(addr); err != nil {
		log.Fatalf("[RdMarket] Server shutdown error: %v", err)
	}
}

func initDB(databaseURL string) (*gorm.DB, *sql.DB, error) {
	gormCfg := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	}
	db, err := gorm.Open(postgres.Open(databaseURL), gormCfg)
	if err != nil {
		return nil, nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}
	return db, sqlDB, nil
}

func runMigrateUp(cfg *config.Config) {
	log.Println("[RdMarket] Running migrations up...")
	_, sqlDB, err := initDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("[RdMarket] DB connection error: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	if err := migrations.Up(sqlDB); err != nil {
		log.Fatalf("[RdMarket] Migration up failed: %v", err)
	}
	log.Println("[RdMarket] Migrations up completed successfully.")
}

func runMigrateDown(cfg *config.Config) {
	log.Println("[RdMarket] Rolling back last migration...")
	_, sqlDB, err := initDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("[RdMarket] DB connection error: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	if err := migrations.Down(sqlDB); err != nil {
		log.Fatalf("[RdMarket] Migration down failed: %v", err)
	}
	log.Println("[RdMarket] Migration rollback completed successfully.")
}

func runSeedDev(cfg *config.Config) {
	if cfg.AppEnv == "production" {
		log.Fatalf("[RdMarket] Seeding dev data is strictly forbidden in production")
	}
	log.Println("[RdMarket] Seeding dev dataset for USD/IDR...")
	db, sqlDB, err := initDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("[RdMarket] DB connection error: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	repo := repositories.NewExchangeRateRepository(db)
	mockProv := providers.NewMockProvider(16280.0)
	service := services.NewMarketService(repo, mockProv, cfg)

	if err := service.SeedInitialDataIfEmpty("USD/IDR"); err != nil {
		log.Fatalf("[RdMarket] Seeding error: %v", err)
	}
	log.Println("[RdMarket] Dev seeding completed.")
}
