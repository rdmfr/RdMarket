# Phase 3: Economic Indicators + Alerts & Market Intelligence

Read `CLAUDE.md`, `README.md`, `docs/`, and the existing code first. Follow `CLAUDE.md` for the whole phase
(architecture, data integrity, wording, design, security).

Phases 0, 1, and 2 already exist and work. Extend them; do not rebuild or refactor them. Existing endpoints,
tables, and components must keep working. List every change made to existing files in the final report.

This file delivers two roadmap phases together:

- Part A: Economic Indicators
- Part B: Alerts & Market Intelligence

Do NOT implement production monitoring/optimization, news scraping, sentiment analysis, NLP, or any
LLM/AI-generated text. All generated text comes from deterministic templates over real data.

---

## 1. Objective

Part A: add macroeconomic context so the user can monitor key Indonesian, US, and global indicators, view
their history, compare them with USD/IDR, and explore transparent statistical co-movement.

Part B: let the user define alert rules on market, indicator, and data-health conditions, receive
notifications through configurable channels, and read a deterministic, fully traceable daily market brief
and regime summary.

Never claim that an indicator or event "causes" USD/IDR movement, never predict with certainty, and never
give advice. Alerts describe observed conditions only.

## 2. Technology

No new frameworks. Same stack as in `CLAUDE.md`. Allowed small additions: an SMTP client from the Go
ecosystem, a small in-process cron/scheduler. Reuse the Phase 0 authentication, migrations tool, provider
pattern, error registry, OpenAPI contract, i18n, tokens, and primitives. The Python forecasting service is
touched only for the optional experiment in section 10. No Redis, queues, or new Docker services:
notification delivery uses a database-backed outbox.

---

# PART A: ECONOMIC INDICATORS

## 3. Indicator catalog

Define the catalog as data, not code (for example `config/indicators.yaml`). Adding an indicator must need
only configuration plus a provider mapping, with no change to services, handlers, or frontend components.

Candidates (include those with a reliable, legitimately usable source; skip the rest and document why in the
README):

- Indonesia: Bank Indonesia policy rate, CPI inflation (YoY), FX reserves, trade balance, 10Y government
  bond yield, GDP growth (quarterly)
- United States: federal funds rate, CPI inflation (YoY), 10Y Treasury yield, unemployment rate
- Global: broad/trade-weighted dollar index, crude oil price, optional volatility index

Each entry defines: code, display name, country, category (monetary policy | inflation | growth | trade |
rates | external | market), unit, frequency (daily | weekly | monthly | quarterly), seasonal adjustment flag,
source provider, source series id, one factual description sentence, change mode (level | percent | bps),
and publication lag in days.

## 4. Data sources and honesty

Reuse the provider abstraction pattern:

```
EconomicDataProvider
├── FredProvider        (if used; key from env)
├── CsvImportProvider   (manual import, always available)
├── ...other providers as needed
└── MockProvider        (tests and non-production only, SIMULATED DATA banner)
```

- Do not assume an API exists. For each indicator research and use an official or clearly licensed source.
  Document source, terms of use, and update frequency in the README.
- Do NOT scrape sites whose terms forbid it. Do not depend on fragile HTML scrapers.
- If no good API exists for an indicator, support it through `CsvImportProvider` (documented column format,
  strict validation) and mark it "manual".
- Keys only via environment variables. Never fabricate, estimate, or interpolate indicator values.
- Every displayed indicator shows its source and last successful update.

## 5. Data model (point-in-time aware)

Model three time concepts: reference period (the period the value describes), release timestamp (when the
public could first know it), and `retrieved_at` (when we fetched it). New migrations (UTC, do not modify
existing tables):

`economic_series`: id, code (unique), name, country, category, unit, frequency, seasonal_adjustment,
source_provider, source_series_id, description, change_mode, publication_lag_days, is_active, created_at,
updated_at.

`economic_observations`: id, series_id, reference_date, period_end (nullable), value (NUMERIC),
release_timestamp (nullable), retrieved_at, revision (int default 0), is_latest (bool), created_at.
UNIQUE (series_id, reference_date, revision). Store revisions instead of overwriting: a revised value inserts
revision+1 and flips `is_latest`. If `release_timestamp` is unknown, leave NULL and apply the configurable
per-series publication lag for any point-in-time use. Never silently assume zero lag.

`economic_ingestion_runs`: id, series_id (nullable), provider, status, rows_fetched, rows_inserted,
rows_revised, error_code, error_message, started_at, finished_at.

`market_events`: id, series_id (nullable), event_type (policy_rate_change | release | other),
event_timestamp, title, detail, value_before, value_after, created_at.
UNIQUE (series_id, event_type, event_timestamp). Derived only from real data, never hand-invented.

Indexes: (series_id, reference_date desc), (series_id, is_latest), (event_timestamp).

## 6. Ingestion

- Go ingestion service through the provider interface, scheduled refresh per series by frequency, no
  overlapping runs of the same series
- Idempotent and revision-aware, with timeouts, retries with backoff, structured logs, and failure isolation
  (one failing series never stops the others)
- Backfill command with rate limiting. Every run recorded in `economic_ingestion_runs`
- Automatically and idempotently create `market_events` for detected policy-rate changes
- Emit internal domain events through a small in-process dispatcher interface:
  `ExchangeRateUpdated` (after each new USD/IDR observation) and `EconomicObservationsUpdated` (after new or
  revised economic observations). The alert engine subscribes to these without the ingestion code knowing
  about alerts. Retrofit the Phase 1 ingestion to emit `ExchangeRateUpdated` with a minimal change

## 7. Analysis (transparent rules, unit tested)

Per indicator: latest value, previous value, change (per change mode), reference period, release date,
source, staleness (data age versus expected frequency, documented tolerance), short sparkline (real data
only), trend descriptor (rising / falling / flat) with documented configurable thresholds.

Relationship versus USD/IDR:

- Resample USD/IDR to the indicator's frequency with a documented rule (period-end or period-average). Never
  forward-fill an indicator beyond its next release. Use release-aware alignment when the release timestamp or
  lag is known
- Use changes/returns, not raw levels, to reduce spurious correlation from trending series. Say so in the UI
- Rolling correlation (configurable window) and lagged cross-correlation over a bounded lag range, with sample
  size next to every coefficient
- Minimum sample size threshold: below it return "Not enough data"
- Include a significance indicator only if implemented correctly, otherwise omit and document
- Label results "Historical co-movement", never "drives" or "predicts"

All thresholds, windows, and lags live in backend config with documented defaults.

---

# PART B: ALERTS & MARKET INTELLIGENCE

## 8. Alert system

Authentication comes from Phase 0. All alert, channel, and import endpoints require auth. Keep the nullable
`user_id` on alert tables so multi-user can be added later.

### 8.1 Rule types (typed, validated, versioned schema)

Model conditions as a typed discriminated union in Go (never free-form JSON executed blindly), stored as
JSONB with `schema_version`:

1. `price_threshold`: USD/IDR above or below a level
2. `percent_change`: change over N days/hours beyond X%
3. `volatility_regime`: rolling volatility enters a chosen regime
4. `sma_cross`: SMA 7/30 or 30/90 crossover (up or down)
5. `indicator_release`: new release for a series, optionally with |change| >= threshold
6. `indicator_stale`: series overdue beyond tolerance
7. `policy_rate_change`: detected policy-rate change (BI or US)
8. `data_source_health`: USD/IDR data older than X, or provider ingestion failing N times in a row
9. `forecast_interval_breach`: latest actual outside the stored 95% prediction interval (only if Phase 2 data
   exists; degrade gracefully and never block other rules)

A new rule type must need only one evaluator (implementing a common interface) plus one registration.

### 8.2 Evaluation engine

- Event-driven: rules subscribe to `ExchangeRateUpdated` and `EconomicObservationsUpdated`. A scheduled
  evaluator handles time-based rules (stale, data health). No tight polling loops
- Pure evaluator functions `(rule, data snapshot) -> result`, unit tested without a database
- Deduplication and idempotency: UNIQUE (rule_id, dedupe_key), for example rule + reference date + crossing
  direction. Re-processing the same data never creates duplicate alerts
- Rule state: last_evaluated_at, last_triggered_at, current_state (for edge-triggered rules)
- Noise control: cooldown, hysteresis for thresholds (re-arm only after moving back past a margin), max
  triggers per rule per day, optional quiet hours (per timezone) for external channels only. Suppressed
  triggers are still recorded with a suppression reason, never silently dropped
- Alerts evaluate only real stored data. If data is missing or insufficient the result is "not evaluable"
  with a reason, never a guess
- Every triggered alert stores a snapshot of the facts used (values, timestamps, thresholds, source)

### 8.3 Notification delivery (outbox pattern)

- Triggering an alert writes `notification_deliveries` rows (status pending) in the same transaction as the
  alert event
- A delivery worker sends them with bounded concurrency, exponential backoff, max attempts, and per-channel
  rate limits. Status: pending | sending | sent | failed | suppressed. One failing channel never blocks others
- Channels: `in_app` (always), `email` (SMTP from env), `webhook` (HTTPS POST, JSON body, HMAC-SHA256
  signature header, timestamp header, timeout), `telegram` (optional, only if simple and clean)
- Webhook safety (SSRF): resolve and block private, loopback, link-local, and metadata-service ranges;
  require HTTPS in production; block redirects to blocked ranges; cap response size and time
- Channel secrets are encrypted at rest with a key from env (AES-GCM), returned masked by the API, never
  logged. Provide a "send test notification" action per channel
- Notification text is a deterministic template, for example: "USD/IDR crossed above 16,500 (latest 16,512,
  29 Sep 2026 14:30 WIB, source: <provider>)." No advice or prediction language

### 8.4 Data model (new migrations, UTC)

`alert_rules`: id, user_id (nullable), name, rule_type, params (jsonb), schema_version, severity (info |
notice | warning), is_enabled, cooldown_seconds, hysteresis (jsonb), max_per_day, quiet_hours (jsonb),
created_at, updated_at (channel selection through a join table `alert_rule_channels`).

`alert_rule_state`: rule_id (pk), last_evaluated_at, last_triggered_at, current_state (jsonb), last_error.

`alert_events`: id, rule_id, triggered_at, severity, title, message, facts (jsonb), dedupe_key, status (open |
acknowledged | resolved | snoozed), acknowledged_at, snoozed_until, suppression_reason.
UNIQUE (rule_id, dedupe_key).

`notification_channels`: id, user_id (nullable), type, name, config_encrypted, is_enabled, last_success_at,
last_error, created_at, updated_at.

`notification_deliveries`: id, alert_event_id, channel_id, status, attempts, next_attempt_at, last_error,
sent_at, created_at.

Indexes: (rule_id, triggered_at desc), (status, next_attempt_at), (alert_event_id), (is_enabled).
Retention: configurable cleanup of old alert events and deliveries, only when configured.

## 9. Market intelligence (deterministic, no AI text)

### 9.1 Regime summary

Computed from stored data with documented thresholds in backend config:

- Trend regime: price relative to SMA 30/90 and slope (up / down / sideways)
- Volatility regime: current rolling volatility versus its own historical percentile (low / moderate /
  elevated / high), percentile window documented
- Move significance: today's move as a percentile of historical daily moves
- Each label returns the exact inputs and thresholds used so the UI can show "why this label"

### 9.2 Daily market brief

- Generated by a scheduled job (config flag, default ON, time configurable in Asia/Jakarta, computed from
  UTC data) and on demand
- Structure: a structured facts object (JSON) plus rendered text from deterministic templates over that
  object. Every sentence maps to a fact in the object. Store both
- Sections: USD/IDR snapshot, regime summary, notable moves, indicator releases since the previous brief
  (real releases only), policy-rate changes, alerts triggered in the period, data-quality notes (staleness,
  gaps, provider failures), and the latest forecast summary WITH interval and baseline caveat (only if
  Phase 2 data exists and is fresh)
- Never include a section with fabricated or placeholder content. If a section has no data, omit it or state
  "No new releases" factually
- Table `market_briefs`: id, brief_date, generated_at, timezone, facts (jsonb), rendered_text,
  template_version, data_fingerprint. UNIQUE (brief_date, template_version)

### 9.3 Upcoming releases

Show an "expected next release" only when it can be derived from real information (a source-provided
calendar, or a documented frequency-based estimate clearly labeled "estimated"). Never present an estimate as
a confirmed schedule.

## 10. Optional: exogenous forecast experiment

Implement ONLY if everything else is stable. Keep it small.

- Add one or two exogenous-feature models to the Phase 2 Python service (for example SARIMAX with lagged
  indicator changes) behind the existing `ForecastModel` interface, labeled "experimental"
- Point-in-time safe: at each backtest origin use only indicator values whose release timestamp (or
  configured lag) is on or before the origin. Add a test proving no leakage
- Same folds as baselines, same leaderboard, compared against naive. No Phase 2 schema change except feature
  metadata in existing JSON columns
- If it cannot be done cleanly and leak-free, skip it and document it as future work

## 11. Go API (`/api/v1`, Phase 0 envelope, update `docs/openapi.yaml` for every endpoint)

Proper status codes (401, 403, 404, 409, 413, 422, 429, 503), validated input, thin handlers.

Economic:

```
GET  /api/v1/economic/indicators
GET  /api/v1/economic/indicators/{code}
GET  /api/v1/economic/indicators/{code}/history?range=&start=&end=&revisions=
GET  /api/v1/economic/indicators/{code}/relationship?window=&maxLag=
GET  /api/v1/economic/events?from=&to=&type=
GET  /api/v1/economic/sources
POST /api/v1/economic/import/{code}      (auth; multipart CSV, dry-run option, size limit,
                                          per-row accept/reject reasons)
```

Alerts (auth required):

```
GET/POST        /api/v1/alerts/rules
GET/PUT/DELETE  /api/v1/alerts/rules/{id}
POST            /api/v1/alerts/rules/{id}/enable | /disable
POST            /api/v1/alerts/rules/{id}/dry-run      (evaluate now, send nothing)
GET             /api/v1/alerts/rule-types              (types with param schemas and descriptions)
GET             /api/v1/alerts/events?status=&severity=&from=&to=
POST            /api/v1/alerts/events/{id}/acknowledge | /snooze
GET             /api/v1/alerts/unread-count
GET/POST        /api/v1/alerts/channels
PUT/DELETE      /api/v1/alerts/channels/{id}
POST            /api/v1/alerts/channels/{id}/test
GET             /api/v1/alerts/deliveries?eventId=
```

Intelligence (public read unless `AUTH_REQUIRE_READ=true`):

```
GET  /api/v1/intelligence/regime
GET  /api/v1/intelligence/briefs?from=&to=
GET  /api/v1/intelligence/briefs/latest
GET  /api/v1/intelligence/briefs/{date}
POST /api/v1/intelligence/briefs/generate     (auth)
GET  /api/v1/intelligence/releases/upcoming
```

Extend `GET /api/v1/data-sources` (or link to the new endpoints) so the Data Sources page also shows
economic series status and notification channel health, without breaking the existing contract.

Independence: Phase 1 endpoints never depend on economic, alert, or forecasting components. If the Python
service, an economic provider, or SMTP is down, everything else keeps working and the affected UI shows a
degraded state.

## 12. Frontend

Follow `CLAUDE.md` design and wording rules, and reuse the existing UI's components, tokens, and visual
language ("Existing UI (preserve)" in `CLAUDE.md`). New colors (alert severities, event markers, series) go into
`tokens.css` and `docs/design-system.md`, never hard-coded. Severity never relies on color alone (text label

- glyph). Sidebar additions: Economy, Alerts, Intelligence. Top bar: alert bell with the real unread count.

**Economy**

- Overview: dense indicator table grouped by country/category (name, latest, previous, change signed with
  glyph, reference period, released, trend, sparkline, staleness badge, source). No grid of big stat cards.
  Compact filters
- Indicator detail: history chart (shared theme, API-driven ranges), metadata (source, frequency, unit,
  seasonal adjustment, description, publication lag rule), visible revision history
- Compare: USD/IDR versus one indicator in dual-axis and indexed-to-100 modes, policy-rate change markers and
  events as plot lines, relationship panel (rolling correlation chart, lagged correlation table with sample
  sizes, alignment method in one line). Muted notice: "Historical co-movement only. Correlation does not
  imply causation. Not financial advice."
- Events: chronological list with real before/after values
- Phase 1 chart: optional event-marker toggle (default off), reusing the existing chart config

**Alerts**

- Inbox: table of alert events (severity, time in WIB, rule, message, status), filters, acknowledge and snooze,
  expandable "facts used" panel with exact values and thresholds
- Rules: table plus a rule editor generated from the rule-types endpoint, with inline validation, "Dry run",
  cooldown/hysteresis/quiet-hours fields, channel selection, and a plain-language summary of each rule
  ("Notify when USD/IDR is above 16,500. Re-arms below 16,450.")
- Channels: list with status, last success, last error, test button. Secrets are write-only fields shown
  masked. Delivery log per alert event
- Optional: alert markers on the USD/IDR chart (toggle, default off)

**Intelligence**

- Daily Brief rendered from stored facts, date navigation, collapsible sections, "why this label" popover
  exposing inputs and thresholds, template version and data fingerprint in small muted text
- Regime panel: trend, volatility, move significance with inputs visible
- Upcoming releases: confirmed versus estimated clearly distinguished
- Persistent muted notice: "Descriptive statistics from stored data. Not financial advice."

Architecture: Component -> composable -> Pinia store -> API service -> Ky -> Go API. Stores only for shared
state (economic, alerts including unread count, intelligence). Zod schemas for every response (log and show
validation errors). Unread-count and inbox refresh use one shared timer, no overlapping timers, cleanup on
unmount, paused when the tab is hidden. Skeletons matching the final layout, plain-sentence errors with
Retry, no-illustration empty states, 401 handling that redirects to login without losing form state.

Wording: allowed "Historical co-movement", "Latest release", "Observed change", "Trend", "Regime",
"Threshold crossed", "Condition met", "Reference period". Forbidden: anything listed in `CLAUDE.md`, in the UI,
notifications, and briefs.

## 13. Configuration and security

Extend `.env.example` (no real secrets):

```
ECONOMIC_FRED_API_KEY=            (only if used)
ECONOMIC_REFRESH_ENABLED=true
ECONOMIC_REFRESH_INTERVAL=
ECONOMIC_BACKFILL_RATE_LIMIT=
ECONOMIC_IMPORT_MAX_BYTES=
APP_ENCRYPTION_KEY=               (32-byte base64, channel secret encryption)
ALERTS_ENGINE_ENABLED=true
ALERTS_MAX_DELIVERY_ATTEMPTS=
ALERTS_WORKER_CONCURRENCY=
ALERTS_RETENTION_DAYS=
SMTP_HOST=
SMTP_PORT=
SMTP_USERNAME=
SMTP_PASSWORD=
SMTP_FROM=
BRIEF_ENABLED=true
BRIEF_SCHEDULE_TIME=              (local time, Asia/Jakarta)
```

Keep the indicator catalog, regime thresholds, correlation windows, lags, publication lags, cooldown
defaults, and brief template settings in versioned config files with comments explaining each default.
Security: no secrets in the frontend or repository, secrets encrypted at rest and masked in responses, CSRF on
state-changing requests, request size limits, input validation on every endpoint, structured logs without
secrets, webhook SSRF protection.

## 14. Docker

No new services. Everything runs inside the existing backend (and optionally the forecasting service for
section 10). `docker compose up` from scratch applies all migrations and seeds the indicator catalog and rule
type registry idempotently. The backend starts and serves Phase 1 even if SMTP, economic providers, or
forecasting are unavailable.

## 15. Testing

Go: provider parsing with recorded sample payloads; ingestion idempotency (run twice, no duplicates),
revision handling, failure isolation, event emission; alignment and correlation edge cases (gaps, mixed
frequencies, insufficient samples); policy-rate event detection; every alert evaluator as a pure function
(edge triggering, hysteresis, cooldown, dedupe, "not evaluable" paths, replaying the same data twice); outbox
worker (retries, backoff, max attempts, channel isolation, suppression recording); webhook signing and SSRF
blocking (private/loopback/link-local/metadata ranges, redirects); secret encryption round trip and masking;
brief generator (deterministic output for a fixed fact set, every sentence maps to a fact, empty-section
behavior); a test that scans templates for forbidden wording; handlers, services, repositories; contract tests.
Frontend: schemas, formatting, indexing-to-100 transform, marker mapping, rule-form-to-payload mapping,
polling composable, "Not enough data" paths.
E2E: import CSV, compare an indicator, create and dry-run a rule, trigger an alert with synthetic data and see
it in the inbox with facts, test a channel, read the daily brief, degraded states. Synthetic data in tests
only. Review screenshots against the anti-slop checklist.

## 16. Quality and honesty

No fake, estimated, or interpolated indicator values. Every number is traceable to a source, reference period,
and retrieval time. Revisions are preserved. Point-in-time correctness is documented and tested. Every alert
and brief sentence is reproducible from stored facts. No AI/LLM-generated text. Update `README.md` (Phase 3
overview, updated architecture diagram including the event dispatcher, alert engine, outbox worker, and brief
job, indicator catalog with sources and licensing notes, publication-lag and revision handling, alignment and
correlation methodology, alert rule types and evaluation semantics, notification channels and security model,
regime and brief methodology, limitations, endpoints, env vars, CSV import format, roadmap), and
`docs/design-system.md`.

Limitations to document: correlations are unstable over time, low-frequency samples are small, alerts depend
on provider data quality and latency, briefs are descriptive only.

## 17. Final deliverable and verification

The user must be able to: run `docker compose up` and have Phases 0 to 3 working; see a grouped indicator table
with real values, release dates, sources, and staleness; import a CSV for an indicator without an API; open an
indicator with history, metadata, and revisions; compare an indicator with USD/IDR with correlation analysis,
sample sizes, and warnings; see policy-rate changes as chart markers; create alert rules of every type,
dry-run and enable them; configure in-app, email, and webhook channels and send a test; see triggered alerts
with the exact facts used and acknowledge or snooze them; confirm no duplicate alerts on data re-processing;
read the latest daily brief and regime summary with "why this label"; see economic source status and channel
health on the Data Sources page; keep using Phases 1 and 2 even if providers, SMTP, or forecasting are down;
add a new indicator by configuration only and a new rule type by adding one evaluator and one registration.

Verify and report: builds, lint, type-check, and tests pass; migrations apply on a fresh database and on an
existing Phase 2 database; ingestion idempotent and revisions handled; alert engine deterministic on replay;
outbox retries work and failed channels do not block others; SSRF protections verified by tests; secrets never
appear in responses, logs, or the repository; Zod validates every response; contract check passes; no
fabricated values; forbidden wording absent; anti-slop checklist passes; existing Phase 1 and 2 tests still
pass; list of changes to existing files and open decisions.

Build incrementally and confirm each step:

1. migrations, catalog seed, models, repositories (economic tables first, then alert and brief tables)
2. domain event dispatcher, and the minimal `ExchangeRateUpdated` emission in Phase 1 ingestion
3. economic provider interface, CSV provider, first real provider, ingestion with revisions
4. indicator snapshot service and endpoints, event detection
5. alignment and correlation services and endpoint
6. frontend: Economy overview and indicator detail
7. frontend: Compare, relationship panel, event markers, Data Sources update
8. alert engine: evaluator interface, first four rule types, state, dedupe, cooldown, hysteresis
9. outbox worker, in-app + email + webhook channels, encryption, SSRF protection, test notifications
10. remaining rule types, alert API, dry-run
11. frontend: Alerts (inbox, rules, channels) and the bell
12. regime service, brief generator, endpoints, scheduled job
13. frontend: Intelligence (brief, regime, upcoming releases)
14. optional exogenous experiment (only if leak-free and stable)
15. documentation and final verification

Prioritize data integrity, point-in-time correctness, reliable and non-duplicated alerting, security of
secrets, and honest, traceable presentation over the number of indicators or rule types.
