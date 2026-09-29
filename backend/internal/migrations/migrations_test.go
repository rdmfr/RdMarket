package migrations

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestMigrationsUpAndDown(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping live migration test")
	}

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		t.Skipf("cannot open database: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.Ping(); err != nil {
		t.Skipf("cannot ping database: %v", err)
	}

	// 1. Fresh Up
	if err := Up(db); err != nil {
		t.Fatalf("migration fresh up failed: %v", err)
	}

	// 2. Down
	if err := Down(db); err != nil {
		t.Fatalf("migration down failed: %v", err)
	}

	// 3. Up Again
	if err := Up(db); err != nil {
		t.Fatalf("migration re-up failed: %v", err)
	}
}
