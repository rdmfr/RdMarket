REVOKE SELECT, INSERT, UPDATE, DELETE ON economic_series, economic_observations,
    economic_ingestion_runs, market_events FROM rdm_app;
REVOKE USAGE, SELECT ON SEQUENCE economic_series_id_seq, economic_observations_id_seq,
    economic_ingestion_runs_id_seq, market_events_id_seq FROM rdm_app;

DROP TABLE IF EXISTS market_events;
DROP TABLE IF EXISTS economic_ingestion_runs;
DROP TABLE IF EXISTS economic_observations;
DROP TABLE IF EXISTS economic_series;
