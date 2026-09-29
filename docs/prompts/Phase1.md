# Phase 1: Market Monitoring

Read `CLAUDE.md`, `README.md`, `docs/`, and the Phase 0 foundation first. Follow `CLAUDE.md` for the whole
phase (architecture, data integrity, wording, design, security).

Reuse what Phase 0 already provides. Do not recreate or restyle it: migrations tool, `exchange_rates`
table, provider interface and capability model, data-validation layer, ingestion-run records, auth,
envelope and error-code registry, OpenAPI + contract check, time module, formatter, i18n, tokens,
primitives, chart wrapper and theme, Playwright setup.

Build ONLY Phase 1. Do NOT implement machine learning, forecasting, prediction models, LSTM, ARIMA, SARIMA,
Prophet, economic indicators, or alerts. Only design so Phase 2 can be added cleanly (section 14).

---

## 1. Objective

A modern USD/IDR market monitoring dashboard that lets the user:

- See the latest USD/IDR rate, daily movement, and the source and timestamp of the data
- View historical data and interactive charts, and change the time range
- See basic market statistics and basic statistical indicators (SMA, returns, volatility)
- Understand current market conditions through transparent, rule-based labels
- See data-source status

It must feel like a real financial analytics product, not a generic admin dashboard.

## 2. Layout

Sidebar: Dashboard, Market Data, Indicators, Data Sources, Settings.

Top bar: application name "RdMarket Intelligence", current instrument USD/IDR, data status
(LIVE | DELAYED | STALE | OFFLINE, derived from real data age versus the expected update frequency),
last updated timestamp, theme toggle, user/settings menu.

Dashboard sections (desktop 1440px+ layout: chart about 70% left, summary and condition about 30% right):

1. Market Summary
2. Historical Chart
3. Market Statistics
4. Basic Indicators
5. Market Condition
6. Data Source Information

Tablet: single column, summary pinned above the chart. Mobile: reasonable but secondary; desktop is primary.
The first screen (summary + chart) must be visible without scrolling on a 1080p display.

## 3. Market Summary

A primary USD/IDR panel displaying: current rate, daily change, daily change percentage, previous close,
data timestamp (with timezone label, for example "29 Sep 2026 14:30 WIB") and the data source.
All values come from real API data. Nothing is hardcoded. If there is no data, show an error/empty state.

## 4. Historical chart

Use Highcharts through the Phase 0 base wrapper and centralized theme. Create `UsdIdrChart.vue`:
props in, no fetching.

- Line chart of the actual rate, date/time on X, rate on Y (right side), crosshair, shared tooltip, zoom,
  responsive behavior, and a range selector
- Range options: 1D, 7D, 1M, 3M, 6M, 1Y, 5Y
- Each range change triggers an API request. Never filter a large dataset on the frontend
- `GET /api/v1/market/usdidr/history?range=1M`, plus optional `start` and `end`
- Ranges that the active provider cannot serve (for example 1D without intraday data) are disabled with a
  short explanation taken from the provider capabilities, never rendered as an empty or fake chart
- Server-side downsampling for long ranges: cap the number of returned points (configurable, default about
  1,500) with a documented method (for example bucketed last value or LTTB). `meta` states the resolution,
  aggregation method, and point count
- Distinguish actual data from unavailable data: missing periods are real gaps (null points, no line
  connection) with a "No data" legend entry. Never interpolate or fabricate. Suspect points are visibly
  marked and excluded from calculations (configurable)
- Toggleable indicators on the chart: SMA 7, SMA 30, SMA 90 (compact chips with a swatch matching the series
  color)
- "Loading market data..." state, and a clear error state with Retry

## 5. Market statistics

Calculate from stored data (rejected data excluded; suspect data excluded by default via config):

- Current rate, previous close, daily change, daily change %
- Weekly change % (latest versus the last observation on or before 7 days earlier)
- Monthly change % (latest versus the last observation on or before one calendar month earlier)
- 52-week high and low (rolling 365 days)
- Average, minimum, and maximum rate over the selected range

Definitions (document them in README and config): "previous close" is the last accepted observation of the
previous market day, with the day boundary defined in Asia/Jakarta. If the data does not reach far enough
back for a value, or coverage is below a configurable threshold, show "Not enough data" instead of a number.
Present the statistics as a dense key/value table, not as a grid of big cards.

## 6. Basic indicators (backend calculations)

Implement only: SMA 7, SMA 30, SMA 90, daily return, weekly return, rolling volatility.

- Compute in the backend as pure, unit-tested domain functions. To make SMA values valid at the start of a
  requested range, read extra warm-up history before the range start. Points without enough history are
  null, never approximated
- Rolling volatility: standard deviation of daily log returns over a configurable window (default 30),
  optionally annualized; document the formula and annualization factor
- Do NOT implement RSI, MACD, Bollinger Bands, or other advanced technical analysis in this phase
- `GET /api/v1/market/usdidr/indicators?range=...` returns aligned series plus latest values. The chart
  toggles SMA series. Returns and volatility are shown in the Indicators view and in the panel

## 7. Market condition (transparent rules)

A descriptive panel computed from actual data with documented thresholds in backend configuration
(with comments explaining each default):

- Short term: recent return (default 7-day) above +threshold = Positive, below -threshold = Negative,
  otherwise Neutral
- 30 day trend: based on the 30-day return and the position of price relative to SMA 30 = Upward, Downward,
  or Sideways
- Volatility: rolling volatility versus configured thresholds = Low, Moderate, or High

Each label exposes the inputs and thresholds used (a small "why this label" popover). No AI-generated text.
Never present the labels as financial advice. Include the muted note "Descriptive statistics only. Not
financial advice."

## 8. Data collection (ingestion service)

Use the Phase 0 provider abstraction and validation layer.

- In-process scheduler with a configurable interval that respects the provider's rate limit and quota
  (never poll aggressively). The frontend never triggers external provider calls, only the Go backend does
- On startup, if the database is empty, run backfill according to the provider capability
  (history endpoint or CSV import); otherwise store what is available and say so in the Data Sources page
- Idempotent upserts using the unique constraint (currency_pair + timestamp + source). UTC storage
- Every run recorded (provider, status, rows fetched, inserted, rejected, suspect, errors, timing)
- Timeouts, retries with backoff, and structured logs. On provider failure keep serving stored data with a
  stale flag. Never invent data
- `MockProvider` only outside production, always with the SIMULATED DATA banner

## 9. Go API

Version `/api/v1`, Phase 0 envelope, proper HTTP status codes, and updates to `docs/openapi.yaml` for every
endpoint.

```
GET /api/v1/market/usdidr/current
GET /api/v1/market/usdidr/history        (range, start, end)
GET /api/v1/market/usdidr/statistics
GET /api/v1/market/usdidr/indicators     (range)
GET /api/v1/market/usdidr/condition
GET /api/v1/data-sources
GET /api/v1/health                       (already in Phase 0)
```

Example error:

```json
{ "data": null, "meta": {}, "error": { "code": "DATA_SOURCE_UNAVAILABLE", "message": "Market data source is temporarily unavailable" } }
```

Handle: external source unavailable, database unavailable, invalid range or date parameters (422), empty
dataset, insufficient data, and timeouts. `meta` carries provider, data timestamp, staleness, resolution,
and `simulated` where relevant.

`GET /api/v1/data-sources` returns per provider: name, instrument, status (Connected | Degraded |
Unavailable, derived from recent ingestion runs), last successful update, data frequency, number of
observations, capabilities (intraday, history depth), and counts of suspect/rejected observations.
All values are real.

## 10. Frontend

Follow the Phase 0 structure and flow: Component -> composable -> Pinia store -> API service -> Ky -> Go API.

- Views: `DashboardView`, `MarketDataView` (table of observations with range, quality flag, source,
  paginated), `IndicatorsView`, `DataSourcesView`, `SettingsView`
- Components under `components/market`, `components/charts`, `components/indicators`, `components/common`
- Store `market.ts`: currentRate, historicalData, statistics, indicators, condition, selectedRange, loading,
  error, lastUpdated; actions `fetchCurrentRate`, `fetchHistory`, `fetchStatistics`, `fetchIndicators`.
  Chart-local UI state (visible SMA toggles, zoom) stays out of the global store
- Store `settings.ts`: default chart range, timezone, theme, auto-refresh interval, locale. Persisted in
  localStorage with Zod-validated parsing (fall back to defaults on invalid data)
- Composables: `useMarketData`, `useMarketChart`, `useAutoRefresh`
- Zod schemas for every response (CurrentRate, HistoricalRate, Statistics, Indicators, Condition,
  DataSource, ApiError). On validation failure: do not continue silently, log the error, and show a proper
  error state
- Loading: skeletons matching the final layout. Chart: "Loading market data...". Failure: plain sentence
  ("Unable to retrieve current market data.") + Retry button. Empty and insufficient data handled
  explicitly
- Data Sources page: provider, instrument, status, last successful update, data frequency, observations,
  capabilities, quality counts

## 11. Settings and auto refresh

Settings: default chart range, timezone, theme, auto-refresh interval (Off, 1, 5, 15, 30 minutes).

Auto refresh: a single timer, never overlapping, cleaned up on unmount, paused while the tab is hidden, and
it only requests the latest data from the Go backend. Do not create a new timer per component.

## 12. Testing

Backend: provider parsing, ingestion idempotency, validation layer (bounds, jumps, staleness), statistics
calculation (including "Not enough data" paths), indicator calculations (known-value tests, warm-up
behavior), market-condition rules, market service, API handlers, repository logic (integration tests on
PostgreSQL), and contract tests.

Frontend: Zod schemas (valid and invalid payloads), data transformation for chart series and gaps, settings
parsing, auto-refresh timer behavior, and formatter behavior.

E2E (Playwright): dashboard loads, change range triggers a new request and the chart updates, indicator
toggles, error state and Retry, Data Sources page, settings persistence. Use synthetic data in tests only.
Capture screenshots and review them against the anti-slop checklist in `CLAUDE.md`.

## 13. Documentation

Update `README.md` (Phase 1 overview, architecture, API documentation, external data provider,
calculation definitions, market-condition thresholds, how to run, roadmap) and `docs/openapi.yaml`.

## 14. Forecasting readiness (do NOT implement forecasting)

Keep the architecture ready for Phase 2 (Python forecasting service, backtesting, model evaluation,
prediction intervals) without changing the core market-data architecture:

- Repository and service methods take `currency_pair` as a parameter, even though only USD/IDR is exposed
- A clean accessor returns the "modeling series" (excludes rejected and suspect data) so later services
  reuse the same cleaning rules
- Statistics and indicator calculators are pure domain functions, reusable later
- The schema stays stable, and the read-only database role documented in Phase 0 is ready
- No coupling between market endpoints and any future service

## 15. Final deliverable and verification

The user must be able to: start with Docker Compose, connect PostgreSQL, configure the exchange-rate
provider, retrieve and store USD/IDR data, view the current rate and historical charts, change the
timeframe, view statistics and SMA/volatility indicators, view data-source status, and see proper loading
and error handling.

Verify and report: frontend and backend build, migrations apply (fresh and existing), all endpoints work,
frontend communicates with backend, Zod validates every response, contract check passes, no secrets
committed, no fake data presented as real, forbidden wording absent, Docker Compose starts, tests and E2E
pass, list of changes made to Phase 0 files, open decisions.

Build incrementally and confirm each step:

1. ingestion service, scheduler, backfill, provider wiring, data-sources status
2. domain calculators: statistics, indicators, condition (pure functions, tests first)
3. services, repositories, handlers, OpenAPI updates
4. frontend API service, schemas, store, composables
5. Market Summary, statistics table, condition panel
6. `UsdIdrChart` with range selector, gaps, indicator toggles
7. Market Data, Indicators, Data Sources, Settings views, auto refresh
8. E2E tests, screenshot review, docs, final verification

Prioritize a working, maintainable implementation over extra features.