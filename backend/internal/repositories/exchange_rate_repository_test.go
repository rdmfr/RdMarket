package repositories

import (
	"os"
	"rdmarket-intelligence/backend/internal/migrations"
	"rdmarket-intelligence/backend/internal/models"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func setupPostgresDB(t *testing.T) *gorm.DB {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping PostgreSQL integration test")
	}

	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Skipf("cannot connect to PostgreSQL (%v); skipping integration test", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Skipf("cannot get sql.DB handle (%v); skipping integration test", err)
	}

	if err := sqlDB.Ping(); err != nil {
		t.Skipf("cannot ping PostgreSQL (%v); skipping integration test", err)
	}

	if err := migrations.Up(sqlDB); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return db
}

func TestExchangeRateRepository_SaveAndGetLatest(t *testing.T) {
	db := setupPostgresDB(t)
	repo := NewExchangeRateRepository(db)

	now := time.Now().UTC().Truncate(time.Second)
	testRate := &models.ExchangeRate{
		CurrencyPair:  "USD/IDR",
		Timestamp:     now,
		Rate:          16250.50,
		Source:        "Test_Postgres",
		QualityStatus: "ok",
		FetchedAt:     &now,
	}

	if err := repo.Save(testRate); err != nil {
		t.Fatalf("failed to save rate: %v", err)
	}

	latest, err := repo.GetLatest("USD/IDR")
	if err != nil {
		t.Fatalf("failed to get latest rate: %v", err)
	}
	if latest == nil {
		t.Fatalf("expected non-nil latest rate")
	}
	if latest.Rate != 16250.50 {
		t.Errorf("expected 16250.50, got %f", latest.Rate)
	}
}
