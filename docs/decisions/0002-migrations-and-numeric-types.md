# ADR 0002: Versioned Migrations, Database Roles, and Numeric Types

## Status
Accepted

## Context
Financial market calculations and auditability require immutable schemas, zero floating-point rounding errors, and strict role separation. Automatic ORM schema mutation (e.g. GORM AutoMigrate) is prone to unreviewed table mutations, missing indexes, and un-rollbackable operations.

## Decisions

### 1. Versioned SQL Migrations
- Schema changes are managed via versioned `.up.sql` and `.down.sql` scripts located in `backend/internal/migrations/sql/`.
- Embedded into the Go binary with `embed.FS` for single-binary portability and container deployments.
- CLI subcommands: `server migrate-up` and `server migrate-down`.
- GORM `AutoMigrate` is strictly forbidden across all environments.

### 2. Money and Exchange Rates
- All currency rates are stored as `NUMERIC(16, 4)` in PostgreSQL.
- Floating-point types (`FLOAT`, `DOUBLE PRECISION`) are strictly forbidden in schema definitions to prevent IEEE-754 precision loss.
- Calculation routines at boundaries use controlled precision.

### 3. Timestamps & Timezones
- All timestamps use `TIMESTAMPTZ` and are strictly stored in UTC.
- Conversion to Asia/Jakarta (WIB) occurs exclusively at the presentation/formatter layer.

### 4. Database Roles & PgBouncer
- Migration 002 establishes two distinct roles:
  - `rdm_app`: Read/Write access on application tables for backend operations.
  - `rdm_readonly`: Read-only access for future forecasting, analytics, and reporting workers.
- Connection design is compatible with PgBouncer transaction pooling (no reliance on named prepared statements or session-level advisory locks).
