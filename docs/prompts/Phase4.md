# Phase 4: Production Monitoring & Optimization

Read `CLAUDE.md`, `README.md`, `docs/`, and the existing code first. Follow `CLAUDE.md` for the whole phase,
including the "Git, GitHub, and attribution" rules (no AI attribution anywhere, never push).

Phases 0 to 3 already exist and work. Extend them; do not rebuild or refactor them. Existing endpoints,
tables, and components must keep working. List every change made to existing files in the final report.

Do NOT add product features. Do NOT implement deployment, TLS, backups, or release tooling (that is Phase 5).
Do NOT add news, sentiment, or AI-generated text.

Principle: measure first, optimize second. Every optimization needs a before and after measurement recorded in
`docs/operations/performance-log.md`. Do not add complexity (caches, partitions, extra services) without
measured evidence that it is needed.

---

## 1. Objective

Make the platform observable, measurable, and efficient under realistic load, so that problems in data
freshness, ingestion, jobs, alert delivery, database, and the API are visible before users notice them.

## 2. Technology and service additions

Allowed additions (all optional, defined in the development compose file (root `docker-compose.yml`) and, when
the owner enables monitoring in production, in `deploy/compose.prod.yml`; enabled through Compose profiles so
`docker compose up` still runs the product without them):

- Prometheus client libraries: Go (`prometheus/client_golang`) and Python (`prometheus_client`)
- Services under a `monitoring` profile: Prometheus, Grafana, postgres_exporter
- Service under a `pooling` profile: PgBouncer
- k6 for load testing (run as a container or local binary, not a permanent service)
- `pg_stat_statements` extension for query analysis

Not allowed unless a measured need is documented in an ADR (`docs/decisions/`): Redis, Kafka, message queues,
distributed tracing stacks, log aggregation stacks, service meshes. OpenTelemetry tracing is optional and only
acceptable if it stays small and can be disabled by configuration.

## 3. Metrics

Expose Prometheus metrics on `/metrics` for the Go backend and the Python forecasting service.

- `/metrics` is never public: bind it to the internal network or protect it (internal token or a separate
  internal port). It must not appear in the public OpenAPI spec.
- Control label cardinality: use route patterns (`/api/v1/market/usdidr/history`), never raw paths, IDs,
  query strings, or user-supplied values as labels.

Required metrics (names are suggestions, keep them consistent and documented):

HTTP: `http_requests_total{method,route,status}`, `http_request_duration_seconds{method,route}` (histogram),
`http_requests_in_flight`, `http_response_size_bytes`.

Data pipeline: `ingestion_runs_total{provider,status}`, `ingestion_last_success_timestamp_seconds{provider}`,
`ingestion_duration_seconds{provider}`, `data_freshness_seconds{pair}` (age of the latest accepted
observation), `observations_total{pair}`, `observations_flagged_total{status}` (suspect, rejected),
`economic_ingestion_runs_total{status}`, `economic_series_stale{code}`.

Database: connection pool stats (open, in use, idle, wait count, wait duration), query duration histogram by
repository method name, migration version gauge.

Forecasting: `forecast_jobs_total{status}`, `forecast_job_duration_seconds{type,model}`,
`forecast_jobs_running`, `forecast_service_up` (as seen from Go).

Alerts and briefs: `alert_evaluations_total{rule_type,result}`, `alert_triggers_total{rule_type,severity}`,
`notification_deliveries_total{channel,status}`, `notification_outbox_backlog`,
`notification_delivery_duration_seconds{channel}`, `brief_generation_total{status}`,
`brief_last_success_timestamp_seconds`.

Runtime: standard Go and Python process and runtime collectors.

## 4. Logging

- Keep structured JSON logs with request ID. Add log level configuration and make sure no secrets, tokens, or
  full request bodies are logged. Add a test that redaction works.
- Docker log rotation (`max-size`, `max-file`) for every service in the compose files.
- Log ingestion, job, and delivery outcomes at consistent levels with stable field names, so they can be
  filtered later.

## 5. Health, readiness, and dashboards

- Extend `GET /api/v1/ready` (or a separate internal endpoint) to report dependency state: database, provider
  freshness, forecasting service (informational, must not fail readiness), outbox backlog (informational).
- Provision Grafana dashboards as code (JSON in `deploy/monitoring/grafana/`), with datasources provisioned
  through files. Dashboards, each with a short description panel:
  1. Service overview: request rate, error rate, latency percentiles, in-flight requests
  2. Data pipeline: freshness, ingestion success and duration, flagged observations, economic series
     staleness
  3. Database: connections, pool wait, slow queries, table and index size
  4. Jobs and alerts: forecast job outcomes and duration, alert triggers, delivery success, outbox backlog
- Dashboards follow the terminal-style principle: dense, dark, no decoration.
- Grafana is protected by authentication (no anonymous admin, no default credentials in the repository).
  Credentials come from environment variables.

## 6. Operational alerting (separate from product alerts)

Operational alert rules watch the platform itself. Do not mix them into the Phase 3 product alert engine.

- Rules as code (`deploy/monitoring/alerts/`), evaluated by Prometheus. Route notifications using the smallest
  possible setup (Grafana-provisioned contact points or Alertmanager, choose one and document why).
- Starting rules (thresholds are proposals for the owner to confirm and tune after baseline measurement):
  data freshness beyond expected frequency plus tolerance; ingestion failing N consecutive runs; API 5xx rate;
  p95 latency above budget; database unreachable; pool wait time rising; outbox backlog growing; forecast
  jobs failing repeatedly; disk usage high; container restarting repeatedly; brief not generated on schedule.
- Every rule has a severity, a runbook link, and a "for" duration to avoid flapping.

## 7. Service level objectives (proposed, owner confirms)

Document in `docs/operations/slo.md`, with the measurement query for each:

- API availability (proposed 99.5% monthly for read endpoints)
- p95 latency for `current` and `history` endpoints under the target load (proposed budgets: current < 150 ms,
  history < 400 ms)
- Data freshness: latest observation age within the expected provider frequency plus a documented tolerance
- Ingestion success rate over a rolling window
- Notification delivery success and time-to-delivery for in-app and email

State clearly that the numbers are proposals until baseline measurements exist.

## 8. Performance work

Order of work: baseline, find bottlenecks, fix, re-measure.

**Load testing (k6, scripts in `deploy/loadtest/`)**: scenarios for dashboard load (current + history +
statistics + indicators), range switching (1M, 1Y, 5Y), economy and alerts pages, mixed traffic, and a soak
test. Run against a seeded database with realistic data volume (generate synthetic volume for testing only,
never in a real environment). Record results in the performance log.

**Backend**
- Enable and use `pg_stat_statements`. Review the slowest and most frequent queries with `EXPLAIN (ANALYZE,
  BUFFERS)`. Add or adjust indexes only with evidence, through a new versioned migration (up and down).
- Tune the connection pool (max open, max idle, connection lifetime) through configuration, not constants.
- Confirm downsampling caps response size for long ranges. Add response compression (gzip or brotli) where it
  measurably helps.
- HTTP caching for public read endpoints: `Cache-Control` and `ETag` with short TTLs tied to the data update
  cadence, so unchanged data returns `304`. Never cache authenticated or state-changing responses.
- Consider an in-process TTL cache for hot read paths only if measurements justify it. Redis is not allowed
  without an ADR that shows in-process caching is insufficient.
- Partitioning of large tables only if a table's measured size and query plans justify it. Daily data for one
  pair is small, so expect this to be unnecessary. Record the conclusion.
- Review autovacuum and statistics settings for tables with frequent inserts and updates.

**Frontend**
- Route-level code splitting, lazy loading of heavy modules (Highcharts modules, rarely used views).
- Bundle size budget checked in CI, plus a Lighthouse (or equivalent) run with budgets for performance,
  accessibility, and layout shift on the dashboard.
- Avoid redundant requests: deduplicate in-flight requests, respect `304`, share timers (already required),
  pause polling in hidden tabs.
- Check chart rendering cost with the largest allowed point counts.

## 9. PgBouncer readiness and optional activation

PgBouncer is enabled through the `pooling` profile and controlled by configuration. The base setup connects
directly to PostgreSQL.

- Use transaction pooling. Ensure GORM and the driver work with it: disable or configure prepared-statement
  caching as required (for example the driver's simple-protocol option, or PgBouncer's prepared-statement
  support if the version allows it). Document the choice.
- No session-level features in application code (session advisory locks, `SET` without `LOCAL`, temporary
  tables, `LISTEN/NOTIFY`).
- Migrations always connect directly to PostgreSQL, never through PgBouncer.
- Add an integration test that runs the backend test suite against PgBouncer in transaction mode.
- Document when to enable it, pool sizes, and the measured effect in the performance log.

## 10. Data and job pipeline hardening

- Ingestion and job workers expose queue and backlog metrics, enforce timeouts, and shut down gracefully
  (finish or release in-flight work on SIGTERM).
- Verify scheduler behavior on restart: no duplicate runs, no lost runs, no overlapping runs.
- Add a data-integrity checker command (`make data-check`) that reports gaps, duplicates, flagged rows,
  stale series, and orphaned records, and is safe to run in production (read-only).

## 11. Runbooks and documentation

Create `docs/operations/` with: `monitoring.md` (what each metric and dashboard means), `slo.md`,
`performance-log.md`, `load-testing.md`, and runbooks (one page each, symptom, checks, fix, escalation):
data source outage, stale data, database down or slow, connection pool exhaustion, outbox backlog, forecasting
service failing, disk full, high error rate. Update `README.md` (monitoring section, compose profiles, new env
vars, roadmap) and `docs/design-system.md` if dashboards introduce anything new. Record ADRs for PgBouncer,
alert routing choice, and caching decisions.

## 12. Configuration

Extend `.env.example` (no real secrets), for example: `METRICS_ENABLED`, `METRICS_BIND_ADDR`,
`METRICS_TOKEN`, `LOG_LEVEL`, `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, `DB_CONN_MAX_LIFETIME`,
`GRAFANA_ADMIN_USER`, `GRAFANA_ADMIN_PASSWORD`, `PGBOUNCER_ENABLED`, `HTTP_CACHE_TTL_*`.
Provide sensible defaults and document each.

## 13. Testing

- Unit tests: metric registration and labels (no high-cardinality labels), log redaction, cache and `ETag`
  behavior, freshness calculations, integrity checker.
- Integration tests: `/metrics` is not reachable from the public path, readiness reflects dependency state,
  backend suite passes through PgBouncer.
- CI: add `.github/workflows/perf.yml` (job id `perf`) that runs the k6 smoke configuration and the Lighthouse
  budgets against the compose stack; the full run stays available through `make loadtest`. Do not make `perf`
  a required check until its results are stable. The bundle size budget check is a step in `frontend.yml`. The
  backend suite against PgBouncer is a separate job `backend-pgbouncer` in `backend.yml`, with the `pooling`
  profile started for that job. Triggers: push and pull_request on `main` and `develop`, minimal
  `permissions`. Keep the existing job ids stable and list every workflow change in the final report.
- E2E: existing suites still pass, and the dashboard still works when Prometheus and Grafana are absent.

## 14. Quality rules

No product behavior changes. No new user-facing wording that violates `CLAUDE.md`. No secrets in dashboards,
rules, or logs. Everything (dashboards, rules, provisioning) is versioned as code. Existing Phase 1 to 3
endpoints remain independent of optional components.

## 15. Final deliverable and verification

The owner must be able to: start the product with `docker compose up` (no monitoring required); start the
monitoring stack with `docker compose --profile monitoring up`; see the four dashboards populated from real
metrics; trigger a controlled failure (stop the provider, stop forecasting) and see it reflected in
dashboards and operational alerts; run `make loadtest` and read results; optionally enable PgBouncer through
the `pooling` profile; read a performance log with before and after numbers for every optimization.

Verify and report: builds, lint, type-check, tests, contract check; migrations (fresh and existing database);
`/metrics` not public; no high-cardinality labels; Grafana credentials not in the repository; no secrets
committed; Phase 1 to 3 tests still pass; baseline and final load-test numbers; list of changes to existing
files; open decisions; no AI attribution text anywhere in files, commit messages, or generated docs.

Build incrementally and confirm each step:

1. metrics library, HTTP and runtime metrics, `/metrics` protection
2. pipeline, database, job, alert, and brief metrics
3. logging hardening and log rotation
4. Prometheus and Grafana under the `monitoring` profile, provisioned dashboards
5. operational alert rules and routing, runbooks
6. baseline load tests and `pg_stat_statements` analysis
7. backend optimizations driven by evidence, with re-measurement
8. frontend bundle, lazy loading, and Lighthouse budgets
9. PgBouncer profile, compatibility tests
10. data-integrity checker, graceful shutdown checks
11. documentation, ADRs, final verification

Prioritize evidence, reliability, and simplicity over the number of tools.