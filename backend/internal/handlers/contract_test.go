package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"rdmarket-intelligence/backend/internal/config"
	"rdmarket-intelligence/backend/internal/middleware"
	"rdmarket-intelligence/backend/internal/models"
	"rdmarket-intelligence/backend/internal/providers"
	"rdmarket-intelligence/backend/internal/repositories"
	"rdmarket-intelligence/backend/internal/services"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"gopkg.in/yaml.v3"
)

type OpenAPISpec struct {
	OpenAPI string                            `yaml:"openapi"`
	Paths   map[string]map[string]interface{} `yaml:"paths"`
}

func TestOpenAPIContract(t *testing.T) {
	// 1. Read and parse OpenAPI spec
	specPath := "../../../docs/openapi.yaml"
	specBytes, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("failed to read OpenAPI spec at %s: %v", specPath, err)
	}

	var spec OpenAPISpec
	if err := yaml.Unmarshal(specBytes, &spec); err != nil {
		t.Fatalf("failed to parse OpenAPI YAML: %v", err)
	}

	if spec.OpenAPI != "3.1.0" {
		t.Errorf("expected OpenAPI 3.1.0, got %s", spec.OpenAPI)
	}

	// 2. Setup application
	cfg := &config.Config{
		AppEnv:              "test",
		Port:                "8080",
		Provider:            "mock",
		CORSAllowedOrigins:  "http://localhost:3000",
		AdminUsername:       "admin",
		AdminPasswordHash:   "$2a$10$a1gL5wE.qV3jPZ8bE6n9..w/c11aY3XgE6K2rV2Z3F7M1Yk5qUq55",
		SessionSecret:       "test-secret-at-least-32-chars-long!",
		AuthRequireRead:     false,
		ShortTermThreshold:  0.002,
		Trend30DThreshold:   0.005,
		VolatilityLowLimit:  0.004,
		VolatilityHighLimit: 0.010,
	}

	repo := repositories.NewMockExchangeRateRepository()
	now := time.Now().UTC()
	_ = repo.SaveBatch([]models.ExchangeRate{
		{CurrencyPair: "USD/IDR", Timestamp: now.AddDate(0, 0, -2), Rate: 16100.0, Source: "ContractTest"},
		{CurrencyPair: "USD/IDR", Timestamp: now.AddDate(0, 0, -1), Rate: 16150.0, Source: "ContractTest"},
		{CurrencyPair: "USD/IDR", Timestamp: now, Rate: 16200.0, Source: "ContractTest"},
	})

	mockProv := providers.NewMockProvider(16200.0)
	svc := services.NewMarketService(repo, mockProv, cfg)
	handler := NewMarketHandler(svc)
	authHandler := middleware.NewAuthHandler(cfg)

	app := fiber.New()
	middleware.SetupMiddleware(app, cfg)
	authHandler.RegisterRoutes(app.Group("/api/v1"))
	handler.RegisterRoutes(app)

	// 3. Test required endpoints in OpenAPI contract
	endpointsToTest := []struct {
		method string
		path   string
		expect int
	}{
		{"GET", "/api/v1/health", http.StatusOK},
		{"GET", "/api/v1/ready", http.StatusOK},
		{"GET", "/api/v1/data-sources", http.StatusOK},
		{"GET", "/api/v1/market/usdidr/current", http.StatusOK},
		{"GET", "/api/v1/market/usdidr/history?range=7D", http.StatusOK},
		{"GET", "/api/v1/market/usdidr/statistics", http.StatusOK},
		{"GET", "/api/v1/market/usdidr/indicators?range=1M", http.StatusOK},
		{"GET", "/api/v1/market/usdidr/condition?range=1M", http.StatusOK},
		{"GET", "/api/v1/auth/me", http.StatusOK},
	}

	for _, ep := range endpointsToTest {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			req := httptest.NewRequest(ep.method, ep.path, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			if resp.StatusCode != ep.expect {
				t.Errorf("expected status %d, got %d", ep.expect, resp.StatusCode)
			}

			body, _ := io.ReadAll(resp.Body)
			var raw map[string]interface{}
			if err := json.Unmarshal(body, &raw); err != nil {
				t.Fatalf("response is not valid JSON: %v", err)
			}

			// Validate ResponseEnvelope structure: must have data, meta, error
			if _, hasData := raw["data"]; !hasData {
				t.Errorf("contract violation on %s: missing 'data' field", ep.path)
			}
			if _, hasMeta := raw["meta"]; !hasMeta {
				t.Errorf("contract violation on %s: missing 'meta' field", ep.path)
			}
			if _, hasError := raw["error"]; !hasError {
				t.Errorf("contract violation on %s: missing 'error' field", ep.path)
			}

			// In test environment with Mock provider, meta.simulated should be true
			if metaMap, ok := raw["meta"].(map[string]interface{}); ok {
				if sim, ok := metaMap["simulated"].(bool); ok && !sim {
					t.Errorf("contract violation on %s: simulated should be true with MockProvider", ep.path)
				}
			}
		})
	}
}
