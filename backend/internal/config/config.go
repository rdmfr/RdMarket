package config

import (
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL        string
	ExchangeRateAPIURL string
	ExchangeRateAPIKey string
	AppEnv             string
	Port               string

	// Market condition thresholds (transparent rule-based parameters)
	ShortTermThreshold float64 // e.g. 0.002 (0.20%)
	Trend30DThreshold  float64 // e.g. 0.005 (0.50%)
	VolatilityLowLimit float64 // e.g. 0.004 (0.40% daily std dev)
	VolatilityHighLimit float64 // e.g. 0.010 (1.00% daily std dev)
}

func LoadConfig() *Config {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/rdmarket?sslmode=disable"
	}

	apiURL := os.Getenv("EXCHANGE_RATE_API_URL")
	if apiURL == "" {
		apiURL = "https://api.frankfurter.dev/v1"
	}

	apiKey := os.Getenv("EXCHANGE_RATE_API_KEY")

	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	shortTermThreshold := getEnvFloat("SHORT_TERM_THRESHOLD", 0.002)
	trend30DThreshold := getEnvFloat("TREND_30D_THRESHOLD", 0.005)
	volLowLimit := getEnvFloat("VOLATILITY_LOW_LIMIT", 0.004)
	volHighLimit := getEnvFloat("VOLATILITY_HIGH_LIMIT", 0.010)

	return &Config{
		DatabaseURL:        dbURL,
		ExchangeRateAPIURL: apiURL,
		ExchangeRateAPIKey: apiKey,
		AppEnv:             appEnv,
		Port:               port,
		ShortTermThreshold: shortTermThreshold,
		Trend30DThreshold:  trend30DThreshold,
		VolatilityLowLimit: volLowLimit,
		VolatilityHighLimit: volHighLimit,
	}
}

func getEnvFloat(key string, fallback float64) float64 {
	if val := os.Getenv(key); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return fallback
}
