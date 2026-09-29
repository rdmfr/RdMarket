# Database Conventions and Architecture

RdMarket stores exchange-rate observations and related operational records in PostgreSQL. Runtime schema changes use versioned SQL migrations under `backend/internal/migrations/sql/`; GORM `AutoMigrate` is never used.

## Core Schema Conventions
- **Timestamps**: All timestamps use `TIMESTAMPTZ` and are stored strictly in UTC.
- **Rates & Monetary Values**: All rates use `NUMERIC(16,4)` in PostgreSQL with explicit scale. Floating-point types (`FLOAT`, `DOUBLE`) are prohibited in table definitions.
- **Naming**: snake_case plural table names (`exchange_rates`, `schema_migrations`), explicit constraint and index names (`uq_exchange_rates_pair_time_source`, `chk_exchange_rates_quality_status`).
- **Audit Columns**: Every table includes `created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP`. Mutable tables include `updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP`.

## Table: `exchange_rates`
Stores observation points retrieved from feeds, CSV backfills, or mocks.
- `id`: BIGSERIAL PRIMARY KEY
- `currency_pair`: VARCHAR(10) NOT NULL (e.g. `USD/IDR`)
- `timestamp`: TIMESTAMPTZ NOT NULL (observation timestamp in UTC)
- `rate`: NUMERIC(16,4) NOT NULL
- `source`: VARCHAR(64) NOT NULL (provider identifier)
- `quality_status`: VARCHAR(16) NOT NULL (`ok`, `suspect`, `rejected`)
- `quality_reason`: VARCHAR(255) (explanatory reason for suspect/rejected records)
- `fetched_at`: TIMESTAMPTZ (retrieval timestamp)
- `created_at`, `updated_at`: TIMESTAMPTZ

Unique constraint:
- `uq_exchange_rates_pair_time_source (currency_pair, timestamp, source)`

Indexes:
- `idx_exchange_rates_currency_pair` on `(currency_pair)`
- `idx_exchange_rates_timestamp` on `(timestamp)`
- `idx_exchange_rates_pair_time` on `(currency_pair, timestamp)`

## Database Role Separation
- Migration `002_create_roles` establishes:
  - `rdm_app`: Used by the backend application container for normal read/write operations.
  - `rdm_readonly`: Provisioned for read-only analytical services, reporting, and forecasting pipelines.

## PgBouncer Readiness
- The application does not use server-side named prepared statements across transactions.
- No session-level temporary tables or advisory locks that break PgBouncer transaction pooling mode.
