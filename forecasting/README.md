# RdMarket Forecasting Service

Internal Python 3.12 service. The Go API remains the only public API; this
service exposes `/health`, `/models`, and bearer-token protected
`/internal/jobs` routes on the private service network.

## Runtime configuration

Set `FORECAST_INTERNAL_TOKEN` to a random secret of at least 32 characters and
provide either `FORECAST_DATABASE_URL` or the dedicated credentials
`FORECAST_DB_USER`, `FORECAST_DB_PASSWORD`, and optionally
`FORECAST_DB_HOST`, `FORECAST_DB_PORT`, `FORECAST_DB_NAME`. The database role
must have SELECT on `exchange_rates` and write privileges only on the
forecasting tables created by the backend migration. This service never
creates or migrates schema.

`FORECAST_MAX_CONCURRENT_JOBS` defaults to 1; `FORECAST_QUEUE_SIZE` defaults
to 4; and `FORECAST_JOB_TIMEOUT_SECONDS` defaults to 1800. The bounded ARIMA
search evaluates 18 candidate orders. SARIMA evaluates eight candidates with
season length five. `FORECAST_CALENDAR_DAYS=false` means forecast horizons are
observed weekday sessions; `true` means calendar days.

The modeling series uses the last accepted (`quality_status='ok'`) exchange
rate per UTC day and excludes weekends. These are observed daily closes only:
missing weekdays are not filled or interpolated, and their count is retained
in forecast metadata. Models use log rates by default so forecasts and
interval endpoints remain positive after exponentiation.

## Local checks

```powershell
python -m pip install -e ".[test]"
python -m pytest
```

Tests use generated in-memory series only. The service does not call external
data providers. Forecast intervals use deterministic normal approximations;
they do not account for structural breaks, policy events, or all uncertainty.
Exchange rates often approximate a random walk at short horizons, and the
statistical models may not outperform the naive baseline.
