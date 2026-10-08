package config

import (
	"os"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestLoadConfig_Defaults(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv("APP_ENV", "development")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected valid default config, got: %v", err)
	}
	if cfg.AppEnv != "development" {
		t.Errorf("expected AppEnv development, got: %s", cfg.AppEnv)
	}
	if cfg.Port != "8080" {
		t.Errorf("expected port 8080, got: %s", cfg.Port)
	}
	if cfg.Provider != "mock" {
		t.Errorf("expected provider mock in dev, got: %s", cfg.Provider)
	}
	if !cfg.EconomicRefreshEnabled || cfg.EconomicRefreshHours != 24 {
		t.Errorf("unexpected economic refresh defaults: enabled=%v hours=%d", cfg.EconomicRefreshEnabled, cfg.EconomicRefreshHours)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(cfg.AdminPasswordHash), []byte("admin123")); err != nil {
		t.Errorf("expected the default development password hash to match the documented E2E credential: %v", err)
	}
}

func TestLoadConfig_ProductionRefusesMock(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv("APP_ENV", "production")
	_ = os.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/rdmarket")
	_ = os.Setenv("EXCHANGE_RATE_PROVIDER", "mock")
	_ = os.Setenv("ADMIN_PASSWORD_HASH", "$2a$10$abcdefghijklmnopqrstuvwxyz1234567890")
	_ = os.Setenv("SESSION_SECRET", "this-is-a-long-enough-32-byte-secret-key!")

	_, err := LoadConfig()
	if err == nil {
		t.Fatalf("expected error when APP_ENV=production and provider=mock, got nil")
	}
	if !strings.Contains(err.Error(), "guard rail violation") && !strings.Contains(err.Error(), "forbidden") {
		t.Errorf("expected guard rail violation error, got: %v", err)
	}
}

func TestLoadConfig_ProductionRequiresDatabaseURL(t *testing.T) {
	os.Clearenv()
	_ = os.Setenv("APP_ENV", "production")
	_ = os.Setenv("EXCHANGE_RATE_PROVIDER", "frankfurter")

	_, err := LoadConfig()
	if err == nil {
		t.Fatalf("expected error when DATABASE_URL is missing in production, got nil")
	}
}

func TestConfig_RedactedDump(t *testing.T) {
	cfg := &Config{
		DatabaseURL:        "postgres://rduser:SuperSecretPassword123@db.example.com:5432/rdmarket",
		ExchangeRateAPIKey: "SecretApiKey123",
		EconomicBLSAPIKey:  "SecretBLSKey123",
		AdminPasswordHash:  "HashedPasswordValue",
		SessionSecret:      "SuperSecretSessionKeyVal32Characters!",
		AppEnv:             "production",
		Provider:           "frankfurter",
	}

	dump := cfg.RedactedDump()
	if strings.Contains(dump, "SuperSecretPassword123") {
		t.Errorf("RedactedDump leaked DB password!")
	}
	if strings.Contains(dump, "SecretApiKey123") {
		t.Errorf("RedactedDump leaked API key!")
	}
	if strings.Contains(dump, "SecretBLSKey123") {
		t.Errorf("RedactedDump leaked BLS API key!")
	}
	if strings.Contains(dump, "HashedPasswordValue") {
		t.Errorf("RedactedDump leaked admin password hash!")
	}
	if strings.Contains(dump, "SuperSecretSessionKeyVal32Characters!") {
		t.Errorf("RedactedDump leaked session secret!")
	}
	if !strings.Contains(dump, "REDACTED") {
		t.Errorf("RedactedDump should contain 'REDACTED'")
	}
}

func TestLoadConfig_ValidatesEconomicRefreshSettings(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("ECONOMIC_REFRESH_ENABLED", "sometimes")
	if _, err := LoadConfig(); err == nil || !strings.Contains(err.Error(), "ECONOMIC_REFRESH_ENABLED") {
		t.Fatalf("expected invalid economic refresh flag to be rejected, got %v", err)
	}
	t.Setenv("ECONOMIC_REFRESH_ENABLED", "true")
	t.Setenv("ECONOMIC_REFRESH_INTERVAL_HOURS", "0")
	if _, err := LoadConfig(); err == nil || !strings.Contains(err.Error(), "ECONOMIC_REFRESH_INTERVAL_HOURS") {
		t.Fatalf("expected invalid economic refresh interval to be rejected, got %v", err)
	}
}

func TestLoadConfig_ValidatesForecastSettings(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("FORECAST_MAX_CONCURRENT_JOBS", "0")
	if _, err := LoadConfig(); err == nil || !strings.Contains(err.Error(), "FORECAST_MAX_CONCURRENT_JOBS") {
		t.Fatalf("expected invalid job limit to be rejected, got %v", err)
	}

	t.Setenv("FORECAST_MAX_CONCURRENT_JOBS", "2")
	t.Setenv("FORECAST_SERVICE_URL", "file:///etc/passwd")
	if _, err := LoadConfig(); err == nil || !strings.Contains(err.Error(), "FORECAST_SERVICE_URL") {
		t.Fatalf("expected invalid forecast service URL to be rejected, got %v", err)
	}
}

func TestLoadConfig_ValidatesLogLevel(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("LOG_LEVEL", "verbose")
	if _, err := LoadConfig(); err == nil || !strings.Contains(err.Error(), "LOG_LEVEL") {
		t.Fatalf("expected invalid log level to be rejected, got %v", err)
	}
}

func TestLoadConfig_ProductionRequiresStrongForecastToken(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://localhost:5432/rdmarket")
	t.Setenv("EXCHANGE_RATE_PROVIDER", "frankfurter")
	t.Setenv("ADMIN_PASSWORD_HASH", "$2a$10$abcdefghijklmnopqrstuvwxyz1234567890")
	t.Setenv("SESSION_SECRET", "this-is-a-long-enough-32-byte-secret-key!")
	t.Setenv("FORECAST_INTERNAL_TOKEN", "short-token")

	if _, err := LoadConfig(); err == nil || !strings.Contains(err.Error(), "FORECAST_INTERNAL_TOKEN") {
		t.Fatalf("expected weak forecast token to be rejected in production, got %v", err)
	}
}
