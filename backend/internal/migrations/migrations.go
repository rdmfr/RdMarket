package migrations

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

//go:embed sql/*.sql
var migrationFiles embed.FS

type migration struct {
	version  int
	name     string
	upFile   string
	downFile string
}

func ensureSchemaMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version BIGINT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	return nil
}

func getAvailableMigrations() ([]migration, error) {
	entries, err := migrationFiles.ReadDir("sql")
	if err != nil {
		return nil, fmt.Errorf("read migration dir: %w", err)
	}

	migMap := make(map[int]*migration)

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".up.sql") {
			parts := strings.SplitN(name, "_", 2)
			v, err := strconv.Atoi(parts[0])
			if err != nil {
				continue
			}
			baseName := strings.TrimSuffix(parts[1], ".up.sql")
			if migMap[v] == nil {
				migMap[v] = &migration{version: v, name: baseName}
			}
			migMap[v].upFile = "sql/" + name
		} else if strings.HasSuffix(name, ".down.sql") {
			parts := strings.SplitN(name, "_", 2)
			v, err := strconv.Atoi(parts[0])
			if err != nil {
				continue
			}
			baseName := strings.TrimSuffix(parts[1], ".down.sql")
			if migMap[v] == nil {
				migMap[v] = &migration{version: v, name: baseName}
			}
			migMap[v].downFile = "sql/" + name
		}
	}

	var list []migration
	for _, m := range migMap {
		list = append(list, *m)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].version < list[j].version
	})
	return list, nil
}

// Apply runs all pending migrations in ascending order
func Apply(db *sql.DB) error {
	return Up(db)
}

// Up runs all pending up migrations
func Up(db *sql.DB) error {
	if err := ensureSchemaMigrationsTable(db); err != nil {
		return err
	}

	all, err := getAvailableMigrations()
	if err != nil {
		return err
	}

	for _, m := range all {
		var applied int
		row := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = $1", m.version)
		if err := row.Scan(&applied); err != nil {
			// Fallback for sqlite / generic placeholders if needed
			row = db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", m.version)
			_ = row.Scan(&applied)
		}

		if applied > 0 {
			continue
		}

		content, err := migrationFiles.ReadFile(m.upFile)
		if err != nil {
			return fmt.Errorf("read up migration %s: %w", m.upFile, err)
		}

		if err := execStatements(db, string(content)); err != nil {
			return fmt.Errorf("execute migration %d (%s): %w", m.version, m.upFile, err)
		}

		_, err = db.Exec("INSERT INTO schema_migrations (version, applied_at) VALUES ($1, CURRENT_TIMESTAMP)", m.version)
		if err != nil {
			_, err = db.Exec("INSERT INTO schema_migrations (version, applied_at) VALUES (?, CURRENT_TIMESTAMP)", m.version)
			if err != nil {
				return fmt.Errorf("record migration %d: %w", m.version, err)
			}
		}
	}
	return nil
}

// Down rolls back the single most recently applied migration
func Down(db *sql.DB) error {
	if err := ensureSchemaMigrationsTable(db); err != nil {
		return err
	}

	all, err := getAvailableMigrations()
	if err != nil {
		return err
	}

	// Find highest applied version
	var latestVersion int
	err = db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&latestVersion)
	if err != nil {
		return fmt.Errorf("find latest migration: %w", err)
	}

	if latestVersion == 0 {
		return nil // nothing to roll back
	}

	var target *migration
	for i := range all {
		if all[i].version == latestVersion {
			target = &all[i]
			break
		}
	}

	if target == nil || target.downFile == "" {
		return fmt.Errorf("no down migration found for version %d", latestVersion)
	}

	content, err := migrationFiles.ReadFile(target.downFile)
	if err != nil {
		return fmt.Errorf("read down migration %s: %w", target.downFile, err)
	}

	if err := execStatements(db, string(content)); err != nil {
		return fmt.Errorf("execute down migration %d: %w", target.version, err)
	}

	_, err = db.Exec("DELETE FROM schema_migrations WHERE version = $1", target.version)
	if err != nil {
		_, err = db.Exec("DELETE FROM schema_migrations WHERE version = ?", target.version)
		if err != nil {
			return fmt.Errorf("remove record for migration %d: %w", target.version, err)
		}
	}

	return nil
}

func execStatements(db *sql.DB, sqlScript string) error {
	// Execute full script block
	_, err := db.Exec(sqlScript)
	if err != nil {
		// If driver requires individual statement execution, split by ';'
		statements := strings.Split(sqlScript, ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, execErr := db.Exec(stmt); execErr != nil {
				return execErr
			}
		}
	}
	return nil
}
