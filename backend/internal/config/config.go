package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DatabaseURL        string `json:"database_url"`
	ExchangeRateAPIURL string `json:"exchange_rate_api_url"`
	ExchangeRateAPIKey string `json:"exchange_rate_api_key"`
	Provider           string `json:"provider"`
	AppEnv             string `json:"app_env"`
	Port               string `json:"port"`
	CORSAllowedOrigins string `json:"cors_allowed_origins"`
	AdminUsername      string `json:"admin_username"`
	AdminPasswordHash  string `json:"admin_password_hash"`
	SessionSecret      string `json:"session_secret"`
	AuthRequireRead    bool   `json:"auth_require_read"`

	// Market condition thresholds (transparent rule-based parameters)
	ShortTermThreshold       float64 `json:"short_term_threshold"`
	Trend30DThreshold        float64 `json:"trend_30d_threshold"`
	VolatilityLowLimit       float64 `json:"volatility_low_limit"`
	VolatilityHighLimit      float64 `json:"volatility_high_limit"`
	IngestionIntervalMinutes int     `json:"ingestion_interval_minutes"`
	HistoryPointLimit        int     `json:"history_point_limit"`
	ForecastServiceURL       string  `json:"forecast_service_url"`
	ForecastInternalToken    string  `json:"forecast_internal_token"`
	ForecastMaxConcurrent    int     `json:"forecast_max_concurrent_jobs"`
}

func LoadConfig() (*Config, error) {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}
	if appEnv != "development" && appEnv != "test" && appEnv != "production" {
		return nil, fmt.Errorf("invalid APP_ENV %q: must be development, test, or production", appEnv)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		if appEnv == "production" {
			return nil, fmt.Errorf("DATABASE_URL is required in production environment")
		}
		dbURL = "postgres://postgres:postgres@localhost:5432/rdmarket?sslmode=disable"
	}

	provider := os.Getenv("EXCHANGE_RATE_PROVIDER")
	if provider == "" {
		provider = "mock"
	}
	provider = strings.ToLower(provider)
	if provider != "frankfurter" && provider != "mock" && provider != "csv" {
		return nil, fmt.Errorf("invalid EXCHANGE_RATE_PROVIDER %q: must be frankfurter, mock, or csv", provider)
	}

	// Guard rail: Mock provider is strictly forbidden in production
	if appEnv == "production" && provider == "mock" {
		return nil, fmt.Errorf("guard rail violation: MockProvider is strictly forbidden when APP_ENV=production")
	}

	apiURL := os.Getenv("EXCHANGE_RATE_API_URL")
	if apiURL == "" {
		apiURL = "https://api.frankfurter.dev/v1"
	}
	apiKey := os.Getenv("EXCHANGE_RATE_API_KEY")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	corsOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = "http://localhost:3000"
	}

	adminUser := os.Getenv("ADMIN_USERNAME")
	if adminUser == "" {
		adminUser = "admin"
	}

	adminHash := os.Getenv("ADMIN_PASSWORD_HASH")
	sessionSecret := os.Getenv("SESSION_SECRET")
	if appEnv == "production" {
		if adminHash == "" {
			return nil, fmt.Errorf("ADMIN_PASSWORD_HASH is required in production")
		}
		if len(sessionSecret) < 32 {
			return nil, fmt.Errorf("SESSION_SECRET must be at least 32 characters in production")
		}
	} else {
		if adminHash == "" {
			// default dev bcrypt hash for "admin123"
			adminHash = "$2a$10$l7ofOGwF/s0KSIAgKdRz1.1FDfS8/h.LLopGGuL10YugLiZl69qPa"
		}
		if sessionSecret == "" {
			sessionSecret = "dev-insecure-session-secret-key-32-chars-ok"
		}
	}

	authReqRead := os.Getenv("AUTH_REQUIRE_READ") == "true"

	shortTermThreshold := getEnvFloat("SHORT_TERM_THRESHOLD", 0.002)
	trend30DThreshold := getEnvFloat("TREND_30D_THRESHOLD", 0.005)
	volLowLimit := getEnvFloat("VOLATILITY_LOW_LIMIT", 0.004)
	volHighLimit := getEnvFloat("VOLATILITY_HIGH_LIMIT", 0.010)
	ingestionIntervalMinutes := getEnvInt("INGESTION_INTERVAL_MINUTES", 15)
	if ingestionIntervalMinutes < 1 {
		return nil, fmt.Errorf("INGESTION_INTERVAL_MINUTES must be at least 1")
	}
	historyPointLimit := getEnvInt("HISTORY_POINT_LIMIT", 1500)
	if historyPointLimit < 100 {
		return nil, fmt.Errorf("HISTORY_POINT_LIMIT must be at least 100")
	}
	forecastServiceURL := strings.TrimRight(os.Getenv("FORECAST_SERVICE_URL"), "/")
	if forecastServiceURL == "" {
		forecastServiceURL = "http://forecasting:8000"
	}
	parsedForecastURL, err := url.ParseRequestURI(forecastServiceURL)
	if err != nil || (parsedForecastURL.Scheme != "http" && parsedForecastURL.Scheme != "https") || parsedForecastURL.Host == "" {
		return nil, fmt.Errorf("FORECAST_SERVICE_URL must be a valid HTTP or HTTPS URL")
	}
	forecastToken := os.Getenv("FORECAST_INTERNAL_TOKEN")
	if appEnv == "production" && len(forecastToken) < 32 {
		return nil, fmt.Errorf("FORECAST_INTERNAL_TOKEN must be at least 32 characters in production")
	}
	forecastMaxConcurrent := 2
	if value := os.Getenv("FORECAST_MAX_CONCURRENT_JOBS"); value != "" {
		forecastMaxConcurrent, err = strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("FORECAST_MAX_CONCURRENT_JOBS must be an integer")
		}
	}
	if forecastMaxConcurrent < 1 || forecastMaxConcurrent > 20 {
		return nil, fmt.Errorf("FORECAST_MAX_CONCURRENT_JOBS must be between 1 and 20")
	}

	return &Config{
		DatabaseURL:              dbURL,
		ExchangeRateAPIURL:       apiURL,
		ExchangeRateAPIKey:       apiKey,
		Provider:                 provider,
		AppEnv:                   appEnv,
		Port:                     port,
		CORSAllowedOrigins:       corsOrigins,
		AdminUsername:            adminUser,
		AdminPasswordHash:        adminHash,
		SessionSecret:            sessionSecret,
		AuthRequireRead:          authReqRead,
		ShortTermThreshold:       shortTermThreshold,
		Trend30DThreshold:        trend30DThreshold,
		VolatilityLowLimit:       volLowLimit,
		VolatilityHighLimit:      volHighLimit,
		IngestionIntervalMinutes: ingestionIntervalMinutes,
		HistoryPointLimit:        historyPointLimit,
		ForecastServiceURL:       forecastServiceURL,
		ForecastInternalToken:    forecastToken,
		ForecastMaxConcurrent:    forecastMaxConcurrent,
	}, nil
}

// RedactedDump returns a JSON representation of config with sensitive fields masked
func (c *Config) RedactedDump() string {
	type SafeConfig struct {
		DatabaseURL        string  `json:"database_url"`
		ExchangeRateAPIURL string  `json:"exchange_rate_api_url"`
		ExchangeRateAPIKey string  `json:"exchange_rate_api_key"`
		Provider           string  `json:"provider"`
		AppEnv             string  `json:"app_env"`
		Port               string  `json:"port"`
		CORSAllowedOrigins string  `json:"cors_allowed_origins"`
		AdminUsername      string  `json:"admin_username"`
		AdminPasswordHash  string  `json:"admin_password_hash"`
		SessionSecret      string  `json:"session_secret"`
		AuthRequireRead    bool    `json:"auth_require_read"`
		ShortTermThreshold float64 `json:"short_term_threshold"`
		Trend30DThreshold  float64 `json:"trend_30d_threshold"`
	}

	redactedDB := c.DatabaseURL
	if u, err := url.Parse(c.DatabaseURL); err == nil && u.User != nil {
		u.User = url.UserPassword(u.User.Username(), "REDACTED")
		redactedDB = u.String()
	}

	maskedKey := ""
	if c.ExchangeRateAPIKey != "" {
		maskedKey = "REDACTED"
	}

	safe := SafeConfig{
		DatabaseURL:        redactedDB,
		ExchangeRateAPIURL: c.ExchangeRateAPIURL,
		ExchangeRateAPIKey: maskedKey,
		Provider:           c.Provider,
		AppEnv:             c.AppEnv,
		Port:               c.Port,
		CORSAllowedOrigins: c.CORSAllowedOrigins,
		AdminUsername:      c.AdminUsername,
		AdminPasswordHash:  "REDACTED",
		SessionSecret:      "REDACTED",
		AuthRequireRead:    c.AuthRequireRead,
		ShortTermThreshold: c.ShortTermThreshold,
		Trend30DThreshold:  c.Trend30DThreshold,
	}

	data, _ := json.MarshalIndent(safe, "", "  ")
	return string(data)
}

func getEnvFloat(key string, fallback float64) float64 {
	if val := os.Getenv(key); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}
