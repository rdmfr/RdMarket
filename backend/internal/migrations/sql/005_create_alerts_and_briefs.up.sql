CREATE TABLE alert_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(128),
    name VARCHAR(160) NOT NULL,
    rule_type VARCHAR(40) NOT NULL,
    params JSONB NOT NULL,
    schema_version INTEGER NOT NULL CHECK (schema_version > 0),
    severity VARCHAR(16) NOT NULL CHECK (severity IN ('info', 'notice', 'warning')),
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    cooldown_seconds INTEGER NOT NULL DEFAULT 0 CHECK (cooldown_seconds >= 0),
    hysteresis JSONB NOT NULL DEFAULT '{}'::jsonb,
    max_per_day INTEGER NOT NULL DEFAULT 0 CHECK (max_per_day >= 0),
    quiet_hours JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE alert_rule_state (
    rule_id UUID PRIMARY KEY REFERENCES alert_rules(id) ON DELETE CASCADE,
    last_evaluated_at TIMESTAMPTZ,
    last_triggered_at TIMESTAMPTZ,
    current_state JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_error VARCHAR(512)
);

CREATE TABLE alert_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID NOT NULL REFERENCES alert_rules(id) ON DELETE CASCADE,
    triggered_at TIMESTAMPTZ NOT NULL,
    severity VARCHAR(16) NOT NULL CHECK (severity IN ('info', 'notice', 'warning')),
    title VARCHAR(200) NOT NULL,
    message TEXT NOT NULL,
    facts JSONB NOT NULL,
    dedupe_key VARCHAR(256) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'acknowledged', 'resolved', 'snoozed')),
    acknowledged_at TIMESTAMPTZ,
    snoozed_until TIMESTAMPTZ,
    suppression_reason VARCHAR(256),
    CONSTRAINT uq_alert_events_rule_dedupe UNIQUE (rule_id, dedupe_key)
);

CREATE TABLE notification_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(128),
    type VARCHAR(16) NOT NULL CHECK (type IN ('in_app', 'email', 'webhook', 'telegram')),
    name VARCHAR(120) NOT NULL,
    config_encrypted BYTEA NOT NULL,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    last_success_at TIMESTAMPTZ,
    last_error VARCHAR(512),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE alert_rule_channels (
    rule_id UUID NOT NULL REFERENCES alert_rules(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES notification_channels(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (rule_id, channel_id)
);

CREATE TABLE notification_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_event_id UUID NOT NULL REFERENCES alert_events(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES notification_channels(id) ON DELETE CASCADE,
    status VARCHAR(16) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sending', 'sent', 'failed', 'suppressed')),
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    next_attempt_at TIMESTAMPTZ,
    last_error VARCHAR(512),
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_notification_deliveries_event_channel UNIQUE (alert_event_id, channel_id)
);

CREATE TABLE market_briefs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    brief_date DATE NOT NULL,
    generated_at TIMESTAMPTZ NOT NULL,
    timezone VARCHAR(64) NOT NULL,
    facts JSONB NOT NULL,
    rendered_text TEXT NOT NULL,
    template_version VARCHAR(32) NOT NULL,
    data_fingerprint VARCHAR(128) NOT NULL,
    CONSTRAINT uq_market_briefs_date_template UNIQUE (brief_date, template_version)
);

CREATE INDEX idx_alert_events_rule_triggered ON alert_events (rule_id, triggered_at DESC);
CREATE INDEX idx_notification_deliveries_status_next ON notification_deliveries (status, next_attempt_at);
CREATE INDEX idx_notification_deliveries_event ON notification_deliveries (alert_event_id);
CREATE INDEX idx_notification_channels_enabled ON notification_channels (is_enabled);
CREATE INDEX idx_market_briefs_date ON market_briefs (brief_date DESC);

GRANT SELECT, INSERT, UPDATE, DELETE ON alert_rules, alert_rule_state, alert_events,
    notification_channels, alert_rule_channels, notification_deliveries, market_briefs TO rdm_app;
