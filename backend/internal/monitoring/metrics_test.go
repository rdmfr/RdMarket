package monitoring

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetricsEndpointRequiresInternalToken(t *testing.T) {
	app := fiber.New()
	cfg := &Config{Token: "internal-token"}
	RegisterRoutes(app, cfg)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("X-Metrics-Token", "internal-token")
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHTTPMetricsHaveBoundedLabels(t *testing.T) {
	reg := NewRegistry(RegistryConfig{Registerer: prometheus.NewRegistry()})
	metrics := reg.HTTP()
	if got := metrics.reqCounter.WithLabelValues("GET", "/api/v1/market/usdidr/current", "200"); got == nil {
		t.Fatal("expected labeled counter")
	}
	if got := testutil.CollectAndCount(metrics.reqCounter); got != 1 {
		t.Fatalf("expected one counter collector, got %d", got)
	}
}
