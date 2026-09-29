# Phase 2: Forecasting & Backtesting (Revised: statistical + ML + LSTM)

Read `CLAUDE.md`, `README.md`, `docs/`, and the existing code first. Follow `CLAUDE.md` for the whole phase.
Phases 0 and 1 already exist and work. Extend them; do not rebuild or refactor them. Existing endpoints,
tables, and components must keep working. List every change made to existing files in the final report.

Do NOT implement economic indicators, alerts, briefs, or anything from later phases.

---

## 1. Objective

Add statistical, machine learning, and deep learning forecasting plus rigorous backtesting for USD/IDR so the
user can:

- Generate forecasts for multiple horizons
- See prediction intervals (uncertainty), not just point estimates
- Compare statistical models, ML models, and an LSTM against simple baselines
- See honest, reproducible backtest results
- See which model performed best historically, and by how much
- Overlay forecasts on the existing historical chart

Core principle: forecasting exchange rates is inherently uncertain. Never present a forecast as certain,
and always show how a model performed against naive baselines. A more complex model (ML, LSTM) is NOT assumed
to be better. If no model beats the naive baseline, the UI says so plainly.

## 2. Technology additions

New service: Python Forecasting Service (internal only).

- Python 3.12, FastAPI, pandas, numpy
- statsmodels (ETS, ARIMA, SARIMA)
- scikit-learn (Ridge, Gradient Boosting, scalers, pipelines)
- Keras 3 with the TensorFlow backend, CPU only (`tensorflow-cpu`) for the LSTM
- SQLAlchemy 2.x + psycopg, Pydantic v2, pytest, pinned dependencies (uv or pip-tools)
- `pmdarima` only if it installs cleanly, otherwise use statsmodels
- Do NOT add PyTorch, Transformers, or any second deep learning framework. Keras/TensorFlow is the only one
- Prophet only as an optional, isolated model that does not slow down the Docker build
- Do NOT add Celery, Redis, Kafka, or RabbitMQ. Use a database-backed job table (section 8)

**Dependency groups** (in `pyproject.toml`):

- `core`: FastAPI, pandas, numpy, statsmodels, SQLAlchemy, psycopg, Pydantic (statistics and baselines)
- `ml`: scikit-learn, keras, tensorflow-cpu

The Dockerfile uses a build arg `INSTALL_ML=true|false` (default `true` in compose). When the `ml` group is not
installed, ML and LSTM models are absent from the registry and `/models` reports them with `available: false`
and a reason. The service must start and serve the statistical models either way.

Go and Vue stacks stay the same. No new frontend libraries unless strictly necessary.

## 3. Architecture

````
PostgreSQL
    ↓
Python Forecasting Service (reads exchange_rates, writes results)
    ↓
Forecast tables in PostgreSQL
    ↓
Go API (reads results, validates requests, triggers jobs)
    ↓
Vue Dashboard
````

- The Go API remains the only public backend. The browser never talks to Python.
- Go triggers jobs by calling the Python service over the internal Docker network through a
  `ForecastingClient` interface in Go (real implementation and mock implementation for tests).
- Python writes results to PostgreSQL. Go only reads results.
- Python uses a dedicated database role: SELECT on `exchange_rates`, INSERT/UPDATE on forecasting tables only
  (provide the SQL in a migration or init script; build on the Phase 0 read-only role placeholder).
- Python service structure (the `forecasting/` folder is a new top-level folder at the repository root, next to
  `frontend/` and `backend/`):

````
forecasting/
  app/
    main.py, config.py
    api/            FastAPI routes: /health, /jobs, /models
    domain/         pure logic, no I/O: metrics, splits, intervals
    models/         one file per model, common interface
    repositories/   database access
    services/       forecast_service, backtest_service
    schemas/        Pydantic
  tests/
  Dockerfile
  pyproject.toml
````

Every model implements one interface:

````python
class ForecastModel(Protocol):
    name: str
    version: str
    def fit(self, series: pd.Series) -> None: ...
    def predict(self, horizon: int, alpha: float) -> ForecastResult: ...
````

`ForecastResult` holds the point forecast, lower and upper bounds, and interval levels. Adding a model must
need only one new file plus one registry entry, with no change to services, API, or database schema.

## 4. Data handling rules

- Use daily frequency for modeling: one observation per calendar day, from the Phase 1 "modeling series"
  accessor (excludes rejected and suspect data). Document the daily-close rule in the README.
- Do not forward-fill or interpolate missing days silently. Either model on the observed business-day index
  or reindex and record the gap count. Store the number of missing observations in run metadata.
- Minimum history per model (configurable, for example ARIMA >= 250 observations, SARIMA >= 500, LSTM >= 750).
  If insufficient, the job fails with error code `INSUFFICIENT_DATA`, never a fake result.
- Strictly prevent look-ahead leakage: every fit uses only data with timestamp <= forecast origin. This
  includes scalers, feature statistics, validation splits, and early stopping for ML and LSTM models.
- Work on log rates by default (config option per model). Document why.
- Record a data fingerprint per run (first timestamp, last timestamp, row count, hash of the series).

## 5. Models (implement in this order)

**Baselines (mandatory, the yardstick):**

1. **Naive** (last value carried forward, random-walk prediction interval)
2. **Drift** (naive plus average historical change)
3. **Seasonal naive** (weekly, optional in config)

**Statistical models:**

4. **ETS / exponential smoothing** (statsmodels)
5. **ARIMA** (order chosen by a documented, bounded search on AIC using training data only)
6. **SARIMA** (bounded search, capped runtime)

**Machine learning (labeled "experimental" in the API and UI):**

7. **Ridge** and **Gradient Boosting** (scikit-learn) on lagged log-returns and rolling statistics.
   Use a scikit-learn `Pipeline`; the scaler is fit on training data only.
8. **LSTM** (Keras) on log-returns with a sliding window.

**Optional (only if it stays simple and clean):** Prophet, isolated.

Each model: returns prediction intervals (default 80% and 95%), has parameters loaded from config, is
deterministic given the same data and seed, and fails gracefully (convergence errors, singular matrices,
NaN loss) with a structured error stored in the job record, never a crash of the whole service.

### LSTM rules

- Small architecture, fully configurable: 1 to 2 LSTM layers, 32 to 64 units, dropout, window length,
  learning rate, batch size, max epochs (`FORECAST_LSTM_MAX_EPOCHS`).
- Early stopping using a time-based validation split taken from the END of the training data. No shuffling
  across time.
- Scaler is fit on the training portion of each fold only.
- Multi-step strategy: `recursive` or `direct`, chosen in config and documented in the README.
- Determinism: seed Python, numpy, and Keras (`keras.utils.set_random_seed`) and enable TensorFlow op
  determinism (`tf.config.experimental.enable_op_determinism()`). Record seed and library versions in
  `hyperparameters`.
- Prediction intervals: an LSTM does not give intervals natively. Use empirical quantiles of validation
  residuals (or simple split-conformal prediction), scaled across the horizon. Document the method plainly.
  Never claim Bayesian or exact intervals. 80% and 95% are still required.
- Backtest cost control: `refit_every_n_folds` (default larger for LSTM than for statistical models) and a
  model-specific `max_folds`. Both are recorded in run metadata.
- Failures (NaN loss, divergence, out of memory) are stored as structured errors on the job.

The `ForecastModel` interface stays the same for all models, including LSTM.

Horizons: 1, 5, 10, 21, 63 days. Define in config whether horizons are calendar or business days and keep
that consistent everywhere.

## 6. Database (new migrations, UTC, use the Phase 0 migrations tool)

`forecast_jobs`: id, job_type (forecast | backtest), status (queued | running | succeeded | failed |
cancelled), currency_pair, model_names, params (jsonb), progress_done, progress_total, error_code,
error_message, requested_by (nullable), created_at, started_at, finished_at.

`forecast_runs`: id, job_id, currency_pair, model_name, model_version, forecast_origin, horizon,
interval_levels (jsonb), training_start, training_end, training_rows, missing_observations,
data_fingerprint, hyperparameters (jsonb), log_transform, code_version, created_at.

`forecasts`: id, run_id, target_timestamp, step, point, lower_80, upper_80, lower_95, upper_95.
UNIQUE (run_id, step).

`backtest_runs`: id, job_id, currency_pair, model_name, model_version, strategy (expanding | rolling),
initial_train_size, step_size, horizon, window_size (nullable), n_folds, started_at, finished_at,
data_fingerprint, hyperparameters (jsonb), created_at.

`backtest_predictions`: id, backtest_run_id, fold, forecast_origin, target_timestamp, step, actual,
predicted, lower_95, upper_95.

`model_evaluations`: id, backtest_run_id, model_name, horizon, metric_name, metric_value, n_observations,
created_at (long format so new metrics need no schema change).

Indexes: (currency_pair, model_name, created_at desc), (run_id), (backtest_run_id), (job_id), and status on
`forecast_jobs`.

Duplicate work: a job with identical (model, params, data_fingerprint, horizon) reuses the existing result
unless `force=true`. Retention: a config value for how many runs per model to keep with an opt-in cleanup
routine. Nothing is deleted by default. All new numeric columns use NUMERIC, never float.

## 7. Backtesting (the most important part)

Walk-forward (rolling-origin) evaluation:

- Strategies: expanding window (default) and rolling window
- Configurable initial train size, step size, horizon, and max folds
- At each origin: fit on data up to the origin only, predict the horizon, store predictions and actuals
- Configurable refit frequency (refit every fold by default; per-model override, for example LSTM)
- Time-based splits only. Never shuffle. Never use a random train/test split
- Run baselines on exactly the same folds as every other model, including ML and LSTM
- No tuning on test folds. Any hyperparameter search or early stopping uses training data only
- Results for ML and LSTM must always be read against Naive, which the UI shows nearby

Metrics (per horizon and overall, on price level and on returns where meaningful): MAE, RMSE, MAPE (guard
against division by zero), sMAPE, MASE, directional accuracy, prediction interval coverage (share of actuals
inside the 95% interval), average interval width, skill score versus naive (1 - model error / naive error).

Add a Diebold-Mariano style comparison versus the naive baseline (or a clearly documented alternative) so the
UI can say whether a difference is statistically meaningful. If you cannot implement it correctly, omit it
and document the omission.

Backtests run as background jobs with progress (folds completed / total). Unit test the split logic
thoroughly, including a test that proves no future data leaks into any fold.

## 8. Job execution

- Go creates a `forecast_jobs` row (queued) and calls `POST /internal/jobs` on the Python service
- The Python service runs jobs in a bounded worker pool (configurable, default 1, because TensorFlow uses
  substantial CPU and memory), updates status and progress, writes results, marks succeeded or failed
- On startup, any job left in "running" is marked failed with `SERVICE_RESTARTED`
- Per-job timeout (configurable, with a higher default or override for deep learning models). Timed-out jobs
  fail cleanly
- Optional scheduled daily forecast inside the Python service (APScheduler acceptable), config flag default OFF
- The internal API requires a shared token (`FORECAST_INTERNAL_TOKEN`)

## 9. Go API (`/api/v1`, Phase 0 envelope, update `docs/openapi.yaml`)

````
GET  /api/v1/forecast/usdidr/models
GET  /api/v1/forecast/usdidr/latest?model=&horizon=
GET  /api/v1/forecast/usdidr/runs/{id}
GET  /api/v1/forecast/usdidr/jobs                       (paginated)
GET  /api/v1/forecast/usdidr/jobs/{id}
POST /api/v1/forecast/usdidr/jobs                       (auth; 202 Accepted)
GET  /api/v1/forecast/usdidr/backtests
GET  /api/v1/forecast/usdidr/backtests/{id}
GET  /api/v1/forecast/usdidr/backtests/{id}/predictions (paginated)
GET  /api/v1/forecast/usdidr/leaderboard?horizon=
````

Rules: validate input in the Go service layer (allowed models, horizon bounds, concurrent-job limits, rate
limit on job creation); status codes 202, 404, 409 (duplicate running job), 422, 503
(`FORECAST_SERVICE_UNAVAILABLE`). The `/models` response includes `stability` (stable | experimental),
`available` (bool), `unavailable_reason`, and minimum data requirement. The leaderboard always includes
baselines, shows skill score versus naive, and flags in plain language when a model does not beat naive.
Phase 1 endpoints must not depend on the forecasting service. Thin handlers, tests for handlers, service,
repositories, and the mock client.

## 10. Frontend

Follow `CLAUDE.md` design and wording rules, and reuse the existing UI's components, tokens, and visual
language ("Existing UI (preserve)" in `CLAUDE.md`). New route and sidebar item "Forecast" with sections
Forecast, Backtests, Models. New series tokens (forecast line, interval bands) go into `tokens.css` and
`docs/design-system.md`.

**Forecast view:** compact model and horizon selectors; chart reusing the existing chart pattern (props in,
no fetching) with the recent actual history, the forecast as a dashed line, 80% and 95% interval bands as
low-opacity fills, and a vertical "forecast origin" plot line; forecast table (step, date, point, lower,
upper); metadata panel (model, version, origin, training window, short data fingerprint, generated at,
interval levels); persistent muted notice: "Statistical forecast with uncertainty. Exchange rates are
influenced by factors no model captures. Not financial advice."

**Backtests view:** leaderboard table (model, MAE, RMSE, MAPE, MASE, directional accuracy, interval coverage,
skill vs naive) with baselines always shown and marked; experimental models carry an "Experimental" label;
if no model beats naive show "No model outperformed the naive baseline on this horizon."; predicted-versus-
actual chart; error by horizon; job launcher; job list with status, progress, and error details.

**Models view:** list with version, stable/experimental label, availability status (ML and LSTM show
"Not installed in this build" when unavailable), minimum data requirement, short method description.

Architecture: Component -> composable -> Pinia store -> API service -> Ky -> Go API. Forecast store for
shared state only. Zod schemas for every response. Job polling: one timer, stops on a terminal state and on
unmount, never overlapping. Skeletons, plain-sentence errors with Retry, no-illustration empty states.
A forecast is never shown without its interval and disclaimer, and model results are never shown without the
baseline comparison nearby. If the forecasting service is down, the forecast UI shows a degraded state and
Phase 1 keeps working.

## 11. Configuration

Extend `.env.example` (no real secrets):

````
FORECAST_SERVICE_URL=
FORECAST_DB_USER=
FORECAST_DB_PASSWORD=
FORECAST_MAX_CONCURRENT_JOBS=1
FORECAST_JOB_TIMEOUT_SECONDS=
FORECAST_SCHEDULER_ENABLED=false
FORECAST_DEFAULT_INTERVALS=0.8,0.95
FORECAST_INTERNAL_TOKEN=
FORECAST_ENABLE_ML=true
FORECAST_RANDOM_SEED=42
FORECAST_LSTM_MAX_EPOCHS=
````

The Python service rejects requests without the token and is never reachable from outside the Docker network
(no published port in production; a dev override may publish it).

## 12. Docker

Add one service: `forecasting`, defined in the root `docker-compose.yml` and in the `deploy/compose.prod.yml`
skeleton (no published port in production). Multi-stage slim Dockerfile at `forecasting/Dockerfile` with the
`INSTALL_ML` build arg, pinned dependencies, non-root user, healthcheck on `/health`, depends on PostgreSQL
being healthy. The backend depends on forecasting only softly and must start and serve Phase 1 even if
forecasting is down. Document resource limits in the README (with ML/LSTM enabled, plan for roughly 2 to 3 GB
of memory and note the larger image size).

## 13. Testing

**Python:** metrics (known-value tests), walk-forward splitter (no leakage, fold counts, expanding versus
rolling), each model on synthetic series (correct horizon length, lower <= point <= upper, determinism with
a seed), insufficient-data and failure paths, duplicate-run detection.
ML and LSTM: small synthetic series with few epochs to keep CI fast; determinism with a seed; a test proving
scalers and validation splits never see data after the forecast origin; a test that ML/LSTM models are
skipped and reported `available: false` when the `ml` group is not installed. Mark heavy tests with a pytest
marker (for example `slow`) so they can be split out of the fast CI path.

**Go:** handlers, forecast service, repositories, `ForecastingClient` mock, 503 degraded behavior, validation.

**Frontend:** Zod schemas, chart series and interval-band transformation, job polling composable (single
timer, cleanup), formatter behavior for forecast numbers.

**E2E:** launch a backtest from the UI, watch it complete, view the leaderboard, generate a forecast, view it
on the chart, degraded state when the service is down. Synthetic data in tests only.

**CI:** add `.github/workflows/forecasting.yml` (job id `forecasting`) that installs the pinned dependencies
(`core` and `ml`), runs pytest, and runs `pip-audit`. Triggers: push and pull_request on `main` and
`develop`, no path filters, minimal `permissions: contents: read`. Add the `rdmarket-forecasting` image
(`forecasting/Dockerfile`) to `docker.yml` with the same tags as the other images (`sha-<7 chars>` and
`edge`). Extend `backend.yml`, `frontend.yml`, and `e2e.yml` only where the new Go, Vue, and E2E tests need
it, keep their job ids stable, and list every workflow change in the final report.

## 14. Quality and honesty

Every stored result is reproducible (same data fingerprint, model version, hyperparameters, and seed give the
same output). No fake forecasts or placeholder numbers. Do not tune models on the test folds. Hyperparameter
search uses training data only.

Document known limitations in the README:

- Exchange rates approximate a random walk at short horizons, and models often fail to beat naive.
- LSTM and ML models are not automatically better than statistical models or baselines. They frequently fail
  to beat Naive on exchange rates, and the UI must state that as it is.
- LSTM intervals are empirical approximations, not exact probabilistic intervals.
- Structural breaks and policy events are not modeled.

Update `README.md` (Phase 2 overview, architecture diagram, model list, backtesting methodology, metric
definitions, endpoints, env vars, Docker service, dependency groups, limitations, roadmap) and
`docs/design-system.md`.

## 15. Final deliverable and verification

The user must be able to: run `docker compose up` and get Phase 1 plus the forecasting service; trigger a
backtest for baselines, ETS, ARIMA, SARIMA, and (when available) Ridge, Gradient Boosting, and LSTM from the
UI; see job progress; see a leaderboard with baselines and skill score versus naive; generate a forecast for a
chosen model and horizon; see 80% and 95% intervals on the chart; see honest statements when no model beats
naive; keep using Phase 1 if the forecasting service is down; add a new model by adding one file and one
registry entry.

Verify and report: Go, Python, and frontend build and test; migrations on a fresh database and on an existing
Phase 1 database; Docker Compose starts (with and without `INSTALL_ML`); end-to-end flow works; Zod validates
every response; contract check passes; no secrets committed; no forecast without interval and disclaimer;
Phase 1 unchanged and working; anti-slop checklist passes; list of changes to existing files.

## 16. Build order (incremental, confirm each step)

1. Migrations and Go read/write plumbing with a mock forecasting client
2. Python service skeleton, config, DB access, health, internal auth, dependency groups
3. Baselines (Naive, Drift, Seasonal naive), metrics, walk-forward backtesting
4. ETS, ARIMA, SARIMA
5. Go endpoints and job flow
6. Vue forecast, backtest, and models views
7. Ridge and Gradient Boosting, then LSTM (Keras), then the optional scheduler (only if everything above is
   stable)
8. Documentation and final verification

Prioritize correctness, reproducibility, and honesty about uncertainty over model quantity or visual flash.
````
````
