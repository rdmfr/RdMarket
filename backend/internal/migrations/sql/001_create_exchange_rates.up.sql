CREATE TABLE IF NOT EXISTS exchange_rates (
    id BIGSERIAL PRIMARY KEY,
    currency_pair VARCHAR(10) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    rate NUMERIC(16, 4) NOT NULL,
    source VARCHAR(64) NOT NULL,
    quality_status VARCHAR(16) NOT NULL DEFAULT 'ok',
    quality_reason VARCHAR(255),
    fetched_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_exchange_rates_quality_status CHECK (quality_status IN ('ok', 'suspect', 'rejected')),
    CONSTRAINT uq_exchange_rates_pair_time_source UNIQUE (currency_pair, timestamp, source)
);

CREATE INDEX IF NOT EXISTS idx_exchange_rates_currency_pair ON exchange_rates(currency_pair);
CREATE INDEX IF NOT EXISTS idx_exchange_rates_timestamp ON exchange_rates(timestamp);
CREATE INDEX IF NOT EXISTS idx_exchange_rates_pair_time ON exchange_rates(currency_pair, timestamp);
