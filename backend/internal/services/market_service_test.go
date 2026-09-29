package services

import (
	"rdmarket-intelligence/backend/internal/config"
	"rdmarket-intelligence/backend/internal/models"
	"rdmarket-intelligence/backend/internal/providers"
	"rdmarket-intelligence/backend/internal/repositories"
	"testing"
	"time"
)

func TestMarketService_Calculations(t *testing.T) {
	repo := repositories.NewMockExchangeRateRepository()

	cfg := &config.Config{
		ShortTermThreshold:  0.002,
		Trend30DThreshold:   0.005,
		VolatilityLowLimit:  0.004,
		VolatilityHighLimit: 0.010,
	}

	mockProvider := providers.NewMockProvider(16200.0)
	service := NewMarketService(repo, mockProvider, cfg)

	// Seed 40 days of sequential rates
	now := time.Now().UTC()
	var rates []models.ExchangeRate
	baseRate := 16000.0
	for i := 40; i >= 0; i-- {
		// Upward trending
		rate := baseRate + float64(40-i)*10.0
		rates = append(rates, models.ExchangeRate{
			CurrencyPair: "USD/IDR",
			Timestamp:    now.AddDate(0, 0, -i),
			Rate:         rate,
			Source:       "Test",
		})
	}
	_ = repo.SaveBatch(rates)

	// Test Current Rate
	current, err := service.GetCurrentRate("USD/IDR")
	if err != nil {
		t.Fatalf("failed to get current rate: %v", err)
	}
	if current.CurrentRate != 16400.0 {
		t.Errorf("expected current rate 16400.0, got %f", current.CurrentRate)
	}
	if current.DailyChange <= 0 {
		t.Errorf("expected positive daily change, got %f", current.DailyChange)
	}

	// Test Indicators
	indicators, err := service.GetIndicators("USD/IDR", "1M")
	if err != nil {
		t.Fatalf("failed to get indicators: %v", err)
	}
	if len(indicators.SMA7) == 0 {
		t.Errorf("expected SMA7 points, got empty")
	}
	if indicators.MarketCondition.Trend30Day != "Upward" {
		t.Errorf("expected Trend30Day Upward, got %s", indicators.MarketCondition.Trend30Day)
	}
	if indicators.MarketCondition.Disclaimer == "" {
		t.Errorf("expected non-empty disclaimer")
	}
}
