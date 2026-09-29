package providers

import (
	"rdmarket-intelligence/backend/internal/models"
	"time"
)

type Capabilities struct {
	SupportsIntraday bool   `json:"supports_intraday"`
	SupportsHistory  bool   `json:"supports_history"`
	MaxHistoryDays   int    `json:"max_history_days"`
	Granularity      string `json:"granularity"`
	RateLimit        string `json:"rate_limit"`
	RequiresAPIKey   bool   `json:"requires_api_key"`
	LicenseNote      string `json:"license_note"`
}

// ExchangeRateProvider defines the pluggable contract for exchange rate feeds
type ExchangeRateProvider interface {
	GetProviderName() string
	Capabilities() Capabilities
	FetchCurrentRate(base, target string) (*models.ExchangeRate, error)
	FetchHistoricalRates(base, target string, start, end time.Time) ([]models.ExchangeRate, error)
}
