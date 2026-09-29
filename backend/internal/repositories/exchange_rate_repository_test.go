package repositories

import (
	"rdmarket-intelligence/backend/internal/models"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	repo := NewExchangeRateRepository(db)
	if err := repo.AutoMigrate(); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func TestExchangeRateRepository_SaveAndGetLatest(t *testing.T) {
	db := setupTestDB(t)
	repo := NewExchangeRateRepository(db)

	now := time.Now().UTC().Truncate(time.Second)
	rate1 := models.ExchangeRate{
		CurrencyPair: "USD/IDR",
		Timestamp:    now.Add(-2 * time.Hour),
		Rate:         16200.0,
		Source:       "TestProvider",
	}
	rate2 := models.ExchangeRate{
		CurrencyPair: "USD/IDR",
		Timestamp:    now,
		Rate:         16250.0,
		Source:       "TestProvider",
	}

	if err := repo.Save(&rate1); err != nil {
		t.Fatalf("failed to save rate1: %v", err)
	}
	if err := repo.Save(&rate2); err != nil {
		t.Fatalf("failed to save rate2: %v", err)
	}

	latest, err := repo.GetLatest("USD/IDR")
	if err != nil {
		t.Fatalf("failed to get latest: %v", err)
	}
	if latest == nil || latest.Rate != 16250.0 {
		t.Errorf("expected latest rate 16250.0, got %v", latest)
	}

	// Verify deduplication on save
	duplicate := models.ExchangeRate{
		CurrencyPair: "USD/IDR",
		Timestamp:    now,
		Rate:         16255.0,
		Source:       "TestProvider",
	}
	if err := repo.Save(&duplicate); err != nil {
		t.Fatalf("failed to update duplicate: %v", err)
	}

	latestUpdated, _ := repo.GetLatest("USD/IDR")
	if latestUpdated.Rate != 16255.0 {
		t.Errorf("expected updated rate 16255.0, got %v", latestUpdated.Rate)
	}
}

func TestExchangeRateRepository_GetStatistics(t *testing.T) {
	db := setupTestDB(t)
	repo := NewExchangeRateRepository(db)

	now := time.Now().UTC()
	rates := []models.ExchangeRate{
		{CurrencyPair: "USD/IDR", Timestamp: now.AddDate(0, 0, -30), Rate: 16000.0, Source: "Test"},
		{CurrencyPair: "USD/IDR", Timestamp: now.AddDate(0, 0, -7), Rate: 16100.0, Source: "Test"},
		{CurrencyPair: "USD/IDR", Timestamp: now.AddDate(0, 0, -1), Rate: 16200.0, Source: "Test"},
		{CurrencyPair: "USD/IDR", Timestamp: now, Rate: 16300.0, Source: "Test"},
	}

	if err := repo.SaveBatch(rates); err != nil {
		t.Fatalf("failed to save batch: %v", err)
	}

	stats, err := repo.GetStatistics("USD/IDR")
	if err != nil {
		t.Fatalf("failed to get statistics: %v", err)
	}

	if !stats.HasSufficientData {
		t.Errorf("expected sufficient data, got false")
	}
	if *stats.CurrentRate != 16300.0 {
		t.Errorf("expected current rate 16300, got %f", *stats.CurrentRate)
	}
	if *stats.PreviousClose != 16200.0 {
		t.Errorf("expected prev close 16200, got %f", *stats.PreviousClose)
	}
	if *stats.DailyChange != 100.0 {
		t.Errorf("expected daily change 100, got %f", *stats.DailyChange)
	}
}
