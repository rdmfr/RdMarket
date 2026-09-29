package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"rdmarket-intelligence/backend/internal/config"
	"rdmarket-intelligence/backend/internal/models"
	"rdmarket-intelligence/backend/internal/providers"
	"rdmarket-intelligence/backend/internal/repositories"
	"rdmarket-intelligence/backend/internal/services"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func setupTestApp(t *testing.T) *fiber.App {
	repo := repositories.NewMockExchangeRateRepository()

	now := time.Now().UTC()
	_ = repo.SaveBatch([]models.ExchangeRate{
		{CurrencyPair: "USD/IDR", Timestamp: now.AddDate(0, 0, -2), Rate: 16100.0, Source: "Test"},
		{CurrencyPair: "USD/IDR", Timestamp: now.AddDate(0, 0, -1), Rate: 16150.0, Source: "Test"},
		{CurrencyPair: "USD/IDR", Timestamp: now, Rate: 16200.0, Source: "Test"},
	})

	cfg := &config.Config{
		ShortTermThreshold:  0.002,
		Trend30DThreshold:   0.005,
		VolatilityLowLimit:  0.004,
		VolatilityHighLimit: 0.010,
	}

	mockProvider := providers.NewMockProvider(16200.0)
	svc := services.NewMarketService(repo, mockProvider, cfg)
	handler := NewMarketHandler(svc)

	app := fiber.New()
	handler.RegisterRoutes(app)
	return app
}

func TestHealthEndpoint(t *testing.T) {
	app := setupTestApp(t)

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var env models.ApiResponse[map[string]interface{}]
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if env.Data["status"] != "healthy" {
		t.Errorf("expected healthy, got %v", env.Data["status"])
	}
}

func TestCurrentRateEndpoint(t *testing.T) {
	app := setupTestApp(t)

	req := httptest.NewRequest("GET", "/api/v1/market/usdidr/current", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var env models.ApiResponse[models.CurrentRateData]
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("failed to unmarshal current rate: %v", err)
	}

	if env.Data.CurrencyPair != "USD/IDR" {
		t.Errorf("expected USD/IDR, got %s", env.Data.CurrencyPair)
	}
	if env.Data.CurrentRate != 16200.0 {
		t.Errorf("expected 16200.0, got %f", env.Data.CurrentRate)
	}
}

func TestHistoryEndpoint(t *testing.T) {
	app := setupTestApp(t)

	req := httptest.NewRequest("GET", "/api/v1/market/usdidr/history?range=7D", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var env models.ApiResponse[models.HistoricalRateData]
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("failed to unmarshal history: %v", err)
	}

	if len(env.Data.Points) == 0 {
		t.Errorf("expected history points, got empty")
	}
}
