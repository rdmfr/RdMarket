CREATE TABLE economic_series (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(160) NOT NULL,
    country VARCHAR(80) NOT NULL,
    category VARCHAR(32) NOT NULL CHECK (category IN ('monetary_policy', 'inflation', 'growth', 'trade', 'rates', 'external', 'market')),
    unit VARCHAR(32) NOT NULL,
    frequency VARCHAR(16) NOT NULL CHECK (frequency IN ('daily', 'weekly', 'monthly', 'quarterly')),
    seasonal_adjustment BOOLEAN NOT NULL DEFAULT FALSE,
    source_provider VARCHAR(64) NOT NULL,
    source_series_id VARCHAR(128) NOT NULL,
    description TEXT NOT NULL,
    change_mode VARCHAR(16) NOT NULL CHECK (change_mode IN ('level', 'percent', 'bps')),
    publication_lag_days INTEGER NOT NULL CHECK (publication_lag_days >= 0),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE economic_observations (
    id BIGSERIAL PRIMARY KEY,
    series_id BIGINT NOT NULL REFERENCES economic_series(id) ON DELETE CASCADE,
    reference_date DATE NOT NULL,
    period_end DATE,
    value NUMERIC(24, 10) NOT NULL,
    release_timestamp TIMESTAMPTZ,
    retrieved_at TIMESTAMPTZ NOT NULL,
    revision INTEGER NOT NULL DEFAULT 0 CHECK (revision >= 0),
    is_latest BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_economic_observations_revision UNIQUE (series_id, reference_date, revision)
);

CREATE TABLE economic_ingestion_runs (
    id BIGSERIAL PRIMARY KEY,
    series_id BIGINT REFERENCES economic_series(id) ON DELETE SET NULL,
    provider VARCHAR(64) NOT NULL,
    status VARCHAR(16) NOT NULL CHECK (status IN ('running', 'succeeded', 'partial', 'failed')),
    rows_fetched INTEGER NOT NULL DEFAULT 0 CHECK (rows_fetched >= 0),
    rows_inserted INTEGER NOT NULL DEFAULT 0 CHECK (rows_inserted >= 0),
    rows_revised INTEGER NOT NULL DEFAULT 0 CHECK (rows_revised >= 0),
    error_code VARCHAR(64),
    error_message VARCHAR(512),
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ
);

CREATE TABLE market_events (
    id BIGSERIAL PRIMARY KEY,
    series_id BIGINT REFERENCES economic_series(id) ON DELETE SET NULL,
    event_type VARCHAR(32) NOT NULL CHECK (event_type IN ('policy_rate_change', 'release', 'other')),
    event_timestamp TIMESTAMPTZ NOT NULL,
    title VARCHAR(200) NOT NULL,
    detail TEXT NOT NULL,
    value_before NUMERIC(24, 10),
    value_after NUMERIC(24, 10),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_market_events_series_type_time UNIQUE (series_id, event_type, event_timestamp)
);

CREATE INDEX idx_economic_observations_series_reference
    ON economic_observations (series_id, reference_date DESC);
CREATE INDEX idx_economic_observations_series_latest
    ON economic_observations (series_id, is_latest);
CREATE INDEX idx_market_events_timestamp ON market_events (event_timestamp);

GRANT SELECT, INSERT, UPDATE, DELETE ON economic_series, economic_observations,
    economic_ingestion_runs, market_events TO rdm_app;
GRANT USAGE, SELECT ON SEQUENCE economic_series_id_seq, economic_observations_id_seq,
    economic_ingestion_runs_id_seq, market_events_id_seq TO rdm_app;