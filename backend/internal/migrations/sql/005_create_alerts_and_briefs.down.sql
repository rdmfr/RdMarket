REVOKE SELECT, INSERT, UPDATE, DELETE ON alert_rules, alert_rule_state, alert_events,
    notification_channels, alert_rule_channels, notification_deliveries, market_briefs FROM rdm_app;

DROP TABLE IF EXISTS notification_deliveries;
DROP TABLE IF EXISTS alert_rule_channels;
DROP TABLE IF EXISTS alert_events;
DROP TABLE IF EXISTS alert_rule_state;
DROP TABLE IF EXISTS alert_rules;
DROP TABLE IF EXISTS notification_channels;
DROP TABLE IF EXISTS market_briefs;
