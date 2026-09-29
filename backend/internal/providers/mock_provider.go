package providers

import (
	"fmt"
	"math"
	"rdmarket-intelligence/backend/internal/models"
	"time"
)

// MockProvider provides deterministic data for testing and offline development
type MockProvider struct {
	BaseRate float64
	Source   string
}

func NewMockProvider(baseRate float64) *MockProvider {
	if baseRate <= 0 {
		baseRate = 16250.0
	}
	return &MockProvider{
		BaseRate: baseRate,
		Source:   "Mock Data Feed",
	}
}

func (p *MockProvider) GetProviderName() string {
	return p.Source
}

func (p *MockProvider) FetchCurrentRate(base, target string) (*models.ExchangeRate, error) {
	now := time.Now().UTC()
	return &models.ExchangeRate{
		CurrencyPair: fmt.Sprintf("%s/%s", base, target),
		Timestamp:    now,
		Rate:         p.BaseRate,
		Source:       p.Source,
	}, nil
}

func (p *MockProvider) FetchHistoricalRates(base, target string, start, end time.Time) ([]models.ExchangeRate, error) {
	var rates []models.ExchangeRate
	cur := start.UTC()
	idx := 0
	for !cur.After(end.UTC()) {
		// Only weekdays (financial markets convention: skip Sat, Sun)
		if cur.Weekday() != time.Saturday && cur.Weekday() != time.Sunday {
			// Deterministic pseudo-random variation based on sine wave
			variation := math.Sin(float64(idx)*0.1)*120.0 + math.Cos(float64(idx)*0.03)*80.0
			rate := p.BaseRate + variation
			rates = append(rates, models.ExchangeRate{
				CurrencyPair: fmt.Sprintf("%s/%s", base, target),
				Timestamp:    cur,
				Rate:         math.Round(rate*100) / 100,
				Source:       p.Source,
			})
			idx++
		}
		cur = cur.AddDate(0, 0, 1)
	}
	return rates, nil
}
