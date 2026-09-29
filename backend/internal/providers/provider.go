package providers

import (
	"rdmarket-intelligence/backend/internal/models"
	"time"
)

// ExchangeRateProvider defines the pluggable contract for exchange rate feeds
type ExchangeRateProvider interface {
	GetProviderName() string
	FetchCurrentRate(base, target string) (*models.ExchangeRate, error)
	FetchHistoricalRates(base, target string, start, end time.Time) ([]models.ExchangeRate, error)
}
