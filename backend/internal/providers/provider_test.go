package providers

import (
	"testing"
	"time"
)

func TestMockProvider_FetchCurrentRate(t *testing.T) {
	provider := NewMockProvider(16300.0)
	rate, err := provider.FetchCurrentRate("USD", "IDR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rate.CurrencyPair != "USD/IDR" {
		t.Errorf("expected USD/IDR, got %s", rate.CurrencyPair)
	}
	if rate.Rate != 16300.0 {
		t.Errorf("expected 16300.0, got %f", rate.Rate)
	}
}

func TestMockProvider_FetchHistoricalRates(t *testing.T) {
	provider := NewMockProvider(16300.0)
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)

	rates, err := provider.FetchHistoricalRates("USD", "IDR", start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rates) == 0 {
		t.Errorf("expected historical rates, got empty slice")
	}
	for _, r := range rates {
		if r.Timestamp.Weekday() == time.Saturday || r.Timestamp.Weekday() == time.Sunday {
			t.Errorf("unexpected weekend rate in market data: %v", r.Timestamp)
		}
	}
}
