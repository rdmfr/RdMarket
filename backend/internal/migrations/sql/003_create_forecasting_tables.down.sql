REVOKE SELECT, INSERT, UPDATE ON forecast_jobs, forecast_runs, forecasts,
    backtest_runs, backtest_predictions, model_evaluations FROM rdm_forecast;
REVOKE SELECT ON exchange_rates FROM rdm_forecast;
REVOKE SELECT, INSERT, UPDATE, DELETE ON forecast_jobs, forecast_runs, forecasts,
    backtest_runs, backtest_predictions, model_evaluations FROM rdm_app;

DROP TABLE IF EXISTS model_evaluations;
DROP TABLE IF EXISTS backtest_predictions;
DROP TABLE IF EXISTS backtest_runs;
DROP TABLE IF EXISTS forecasts;
DROP TABLE IF EXISTS forecast_runs;
DROP TABLE IF EXISTS forecast_jobs;
