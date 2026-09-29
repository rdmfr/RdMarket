CREATE TABLE forecast_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_type VARCHAR(16) NOT NULL CHECK (job_type IN ('forecast', 'backtest')),
    status VARCHAR(16) NOT NULL DEFAULT 'queued'
        CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'cancelled')),
    currency_pair VARCHAR(10) NOT NULL,
    model_names JSONB NOT NULL,
    params JSONB NOT NULL DEFAULT '{}'::jsonb,
    progress_done INTEGER NOT NULL DEFAULT 0 CHECK (progress_done >= 0),
    progress_total INTEGER NOT NULL DEFAULT 0 CHECK (progress_total >= 0),
    error_code VARCHAR(64),
    error_message VARCHAR(512),
    requested_by VARCHAR(128),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ
);

CREATE TABLE forecast_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES forecast_jobs(id) ON DELETE CASCADE,
    currency_pair VARCHAR(10) NOT NULL,
    model_name VARCHAR(32) NOT NULL,
    model_version VARCHAR(32) NOT NULL,
    forecast_origin TIMESTAMPTZ NOT NULL,
    horizon INTEGER NOT NULL CHECK (horizon > 0),
    interval_levels JSONB NOT NULL,
    training_start TIMESTAMPTZ NOT NULL,
    training_end TIMESTAMPTZ NOT NULL,
    training_rows INTEGER NOT NULL CHECK (training_rows > 0),
    missing_observations INTEGER NOT NULL DEFAULT 0 CHECK (missing_observations >= 0),
    data_fingerprint VARCHAR(128) NOT NULL,
    hyperparameters JSONB NOT NULL DEFAULT '{}'::jsonb,
    log_transform BOOLEAN NOT NULL DEFAULT TRUE,
    code_version VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE forecasts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES forecast_runs(id) ON DELETE CASCADE,
    target_timestamp TIMESTAMPTZ NOT NULL,
    step INTEGER NOT NULL CHECK (step > 0),
    point NUMERIC(20, 8) NOT NULL,
    lower_80 NUMERIC(20, 8) NOT NULL,
    upper_80 NUMERIC(20, 8) NOT NULL,
    lower_95 NUMERIC(20, 8) NOT NULL,
    upper_95 NUMERIC(20, 8) NOT NULL,
    CONSTRAINT forecasts_run_step_unique UNIQUE (run_id, step),
    CONSTRAINT forecasts_interval_order CHECK (
        lower_95 <= lower_80 AND lower_80 <= point
        AND point <= upper_80 AND upper_80 <= upper_95
    )
);

CREATE TABLE backtest_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES forecast_jobs(id) ON DELETE CASCADE,
    currency_pair VARCHAR(10) NOT NULL,
    model_name VARCHAR(32) NOT NULL,
    model_version VARCHAR(32) NOT NULL,
    strategy VARCHAR(16) NOT NULL CHECK (strategy IN ('expanding', 'rolling')),
    initial_train_size INTEGER NOT NULL CHECK (initial_train_size > 0),
    step_size INTEGER NOT NULL CHECK (step_size > 0),
    horizon INTEGER NOT NULL CHECK (horizon > 0),
    window_size INTEGER CHECK (window_size IS NULL OR window_size > 0),
    n_folds INTEGER NOT NULL CHECK (n_folds > 0),
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ,
    data_fingerprint VARCHAR(128) NOT NULL,
    hyperparameters JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE backtest_predictions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    backtest_run_id UUID NOT NULL REFERENCES backtest_runs(id) ON DELETE CASCADE,
    fold INTEGER NOT NULL CHECK (fold >= 0),
    forecast_origin TIMESTAMPTZ NOT NULL,
    target_timestamp TIMESTAMPTZ NOT NULL,
    step INTEGER NOT NULL CHECK (step > 0),
    actual NUMERIC(20, 8) NOT NULL,
    predicted NUMERIC(20, 8) NOT NULL,
    lower_95 NUMERIC(20, 8) NOT NULL,
    upper_95 NUMERIC(20, 8) NOT NULL,
    CONSTRAINT backtest_predictions_fold_step_unique UNIQUE (backtest_run_id, fold, step),
    CONSTRAINT backtest_predictions_interval_order CHECK (lower_95 <= predicted AND predicted <= upper_95)
);

CREATE TABLE model_evaluations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    backtest_run_id UUID NOT NULL REFERENCES backtest_runs(id) ON DELETE CASCADE,
    model_name VARCHAR(32) NOT NULL,
    horizon INTEGER NOT NULL CHECK (horizon > 0),
    metric_name VARCHAR(64) NOT NULL,
    metric_value NUMERIC(24, 12),
    n_observations INTEGER NOT NULL CHECK (n_observations >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT model_evaluations_metric_unique UNIQUE (backtest_run_id, model_name, horizon, metric_name)
);

CREATE INDEX forecast_jobs_status_idx ON forecast_jobs (status);
CREATE INDEX forecast_jobs_created_at_idx ON forecast_jobs (created_at DESC);
CREATE INDEX forecast_runs_pair_model_created_idx ON forecast_runs (currency_pair, model_name, created_at DESC);
CREATE INDEX forecast_runs_job_idx ON forecast_runs (job_id);
CREATE INDEX forecasts_run_idx ON forecasts (run_id);
CREATE INDEX backtest_runs_pair_model_created_idx ON backtest_runs (currency_pair, model_name, created_at DESC);
CREATE INDEX backtest_runs_job_idx ON backtest_runs (job_id);
CREATE INDEX backtest_predictions_run_idx ON backtest_predictions (backtest_run_id);
CREATE INDEX model_evaluations_run_idx ON model_evaluations (backtest_run_id);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'rdm_forecast') THEN
        CREATE ROLE rdm_forecast NOLOGIN;
    END IF;
END
$$;

GRANT USAGE ON SCHEMA public TO rdm_forecast;
GRANT SELECT ON exchange_rates TO rdm_forecast;
GRANT SELECT, INSERT, UPDATE ON forecast_jobs, forecast_runs, forecasts,
    backtest_runs, backtest_predictions, model_evaluations TO rdm_forecast;
GRANT SELECT, INSERT, UPDATE, DELETE ON forecast_jobs, forecast_runs, forecasts,
    backtest_runs, backtest_predictions, model_evaluations TO rdm_app;
