# Database Conventions and Architecture

RdMarket stores exchange-rate observations and related operational records in PostgreSQL. Runtime schema changes use versioned SQL migrations under `backend/internal/migrations/sql/`; GORM `AutoMigrate` is never used.

## Core Schema Conventions
- **Timestamps**: All timestamps use `TIMESTAMPTZ` and are stored strictly in UTC.
- **Rates & Monetary Values**: All rates use `NUMERIC(16,4)` in PostgreSQL with explicit scale. Floating-point types (`FLOAT`, `DOUBLE`) are prohibited in table definitions.
- **Forecast values**: Forecast points, bounds, observed backtest values, and metric values use PostgreSQL `NUMERIC`; Python model calculations convert to numeric storage at the persistence boundary.
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
  - `rdm_readonly`: Provisioned for read-only analytical and reporting use.
- Migration `003_create_forecasting_tables` establishes `rdm_forecast` as a no-login role when it is absent, grants it `SELECT` on accepted exchange-rate storage and read/write access only to forecast result/job tables. Provision a separate login with a secret-managed password before connecting the Python service. Fresh development databases provision this login through `deploy/postgres/init-forecast-role.sh`.

## Phase 2 Forecasting Tables

Migration `003_create_forecasting_tables` adds `forecast_jobs`, `forecast_runs`,
`forecasts`, `backtest_runs`, `backtest_predictions`, and
`model_evaluations`. The job and run metadata use UTC `TIMESTAMPTZ`; all rates,
interval bounds, actual/predicted values, and evaluation metrics use
`NUMERIC`. Foreign keys cascade only when an explicit retention/cleanup
operation deletes a parent record. No results are removed automatically.

## Phase 3 Tables

Migration `004_create_economic_indicators` adds the economic series catalog,
revision-preserving observations, ingestion-run audit records, and derived
market events. Observation identity is `(series_id, reference_date, revision)`;
ingestion inserts a new revision and clears `is_latest` on the prior row rather
than overwriting its value. Release time is nullable, while `retrieved_at` is
always recorded. Economic values use `NUMERIC(24,10)`.

Migration `005_create_alerts_and_briefs` adds versioned alert rules and state,
deduplicated alert events, notification channels and delivery outbox rows, and
stored market briefs. Channel configuration uses `BYTEA`; application code
must encrypt it with AES-GCM before persistence and must never return plaintext
secrets.

## PgBouncer Readiness
- The application does not use server-side named prepared statements across transactions.
- No session-level temporary tables or advisory locks that break PgBouncer transaction pooling mode.
