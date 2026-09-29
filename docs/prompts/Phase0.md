Build Phase 0 (Foundation) of "RdMarket Intelligence", a USD/IDR financial
market monitoring platform. This phase contains NO product features. It
creates the foundation that Phases 1-5 will build on, and makes the hard
decisions early so they are not painful to change later.

Stack (fixed, do not add others without need):
Frontend: Vue 3, TypeScript (strict), Vite, Pinia, Vue Router, Tailwind CSS,
Ky, Zod, Highcharts, Iconify, vue-i18n (allowed addition).
Backend: Go, GoFiber, GORM, PostgreSQL.
Infra: Docker, Docker Compose.
Later (not now): Python forecasting service.
Do not use React, Next.js, Nuxt, Laravel, a Node.js backend, or Firebase.
Do not add Redis, message queues, or extra Docker services.

==================================================
DATA SOURCE DECISION (FILLED IN BY THE OWNER, DO NOT OVERRIDE)
==================================================

Primary USD/IDR provider: <FILL IN, e.g. name + docs URL>
Access type: <free tier | paid | official reference rate>
Historical depth available: <e.g. 5 years daily, or "current only">
Intraday data available: <yes/no, granularity>
Rate limit / quota: <e.g. 1,000 requests/month>
Backfill method: <API history endpoint | CSV import | none>
License / display restrictions: <notes>
Secondary provider for cross-check:<optional>
UI language(s): <id | en | both, default: ...>

If a field is left blank, do NOT invent a value. Implement the capability
model below, mark the field as "unknown" in docs/decisions/0001-data-source.md,
and list it as an open decision in the final report.

==================================================

1. # REPOSITORY AND TOOLING

Monorepo:
/frontend /backend /docs /deploy /e2e
CLAUDE.md at the repository root (already provided by the owner; do not rewrite it, only append
learned repo-specific commands to a clearly marked section if needed). Phase prompts live in docs/prompts/.
The repository may already contain `.github/workflows/` files and a root `docker-compose.yml`: extend and
reuse them, never recreate or rename them.
Makefile (or Taskfile) with: setup, dev, build, test, lint, migrate-up,
migrate-down, seed-dev, e2e, compose-up, compose-down, contract-check.

Quality gates:

- Go: gofmt, go vet, golangci-lint (sensible minimal config), race-enabled
  tests
- Frontend: npm (package-lock.json committed), ESLint, Prettier, vue-tsc type-check, Vitest
- Pre-commit hooks (lightweight; secrets scan included)
- CI with GitHub Actions in `.github/workflows/`. The repository already contains `backend.yml`,
  `frontend.yml`, and `docker.yml`: extend them, do not recreate or rename them, and keep their job ids
  (`backend`, `frontend`, `docker`) because the owner requires them as status checks. Triggers: push and
  pull_request on `main` and `develop`, no path filters. Minimal top-level `permissions: contents: read` and
  a `concurrency` group (not for `docker.yml`).
  - `backend.yml`: gofmt check, go vet, golangci-lint, race-enabled tests, integration tests against a
    Postgres service container, migration test (fresh up, last migration down then up), build
  - `frontend.yml`: `npm ci`, ESLint, Prettier check, vue-tsc, Vitest, build
  - `repo-checks.yml` (job id `repo-checks`): secret scan, contract check, docker compose config and smoke
    test, and the hard-coded UI string and forbidden style checks that are not part of the frontend lint
  - `e2e.yml` (job id `e2e`): Playwright smoke tests against the compose test profile
  - `docker.yml`: image publishing (defined further in Phase 5). In this phase only make sure both
    Dockerfiles build
- .gitignore, .editorconfig, .dockerignore, .env.example (no real secrets)

================================================== 2. CONFIGURATION
==================================================

- Central typed config in Go loaded from environment variables.
- Fail fast at startup with a clear message if required config is missing or
  invalid. Never start half-configured.
- Never log secrets. Provide a redacted config dump for debugging.
- APP_ENV (development | test | production) drives guard rails (see section 6).

================================================== 3. DATABASE FOUNDATION
==================================================

- Use versioned SQL migrations with golang-migrate (or goose; pick one and
  document it). DO NOT use GORM AutoMigrate anywhere.
- Every migration has an up and a down. CI proves: fresh database migrates up,
  and the last migration rolls back and up again.
- Conventions (document in docs/database.md):
  - Timestamps: TIMESTAMPTZ, always stored in UTC
  - Money/rates: NUMERIC with explicit precision, never float
  - Naming: snake_case, plural tables, explicit constraint and index names
  - Every table has created_at; mutable tables have updated_at
- Establish the schema for Phase 1's exchange_rates with the required
  uniqueness (currency_pair + timestamp + source) and indexes, and ADD:
  quality_status (ok | suspect | rejected), quality_reason (nullable),
  fetched_at. Phase 1 will use this table; do not build product logic yet.
- Create separate database roles now: app (read/write on app tables), and a
  documented placeholder for a read-only role for future services.
- Design so PgBouncer can be added later without code changes (no
  session-level features that break transaction pooling).

================================================== 4. PROVIDER AND DATA-QUALITY FOUNDATION
==================================================

Provider abstraction (interfaces and capability model only, plus the mock):

ExchangeRateProvider
Capabilities() -> {
supportsIntraday, supportsHistory, maxHistoryDays, granularity,
rateLimit, requiresApiKey, licenseNote
}
FetchLatest(pair), FetchHistory(pair, from, to)
Implementations: one real provider per the DATA SOURCE DECISION (if enough
information was given), CsvImportProvider (always), MockProvider.

- The frontend must be able to learn from the API what the active provider
  supports (e.g. no intraday) so the UI can disable or explain unavailable
  ranges instead of showing empty or fake charts.
- Data validation layer (pure, unit-tested, in domain code):
  - Sanity bounds (configurable min/max plausible USD/IDR)
  - Jump detection versus the previous observation (configurable % threshold)
  - Staleness detection
  - Duplicate and out-of-order handling
    Suspicious data is stored with quality_status = suspect and a reason. It is
    never silently dropped and never silently trusted. Rejected data is logged
    with the reason.
- Backfill job/command with rate limiting, resumability, and idempotency.
- Record ingestion runs (provider, status, counts, errors, timing).

================================================== 5. API FOUNDATION AND CONTRACT
==================================================

- /api/v1 with the response envelope { data, meta, error } from the first
  endpoint on.
- Central error code registry (Go) with matching TypeScript union; one place
  to add codes.
- Middleware: request ID, structured JSON logging (with request ID), panic
  recovery, timeouts, CORS (explicit origins), rate limiting, security
  headers, request size limits.
- Health endpoints: /api/v1/health (liveness) and /api/v1/ready (checks DB
  and required dependencies). Docker healthchecks use them.
- OpenAPI 3.1 spec in /docs/openapi.yaml is the single source of truth.
  - Backend: a contract test validates real responses against the spec.
  - Frontend: Zod schemas are checked against the spec (generate them or add
    a test that fails when they diverge).
  - `make contract-check` runs both and is part of CI.
- Shared time module (Go and TS): parse/format UTC, display in a chosen
  timezone (default Asia/Jakarta), one place only.

================================================== 6. AUTH AND ENVIRONMENT GUARD RAILS
==================================================

Minimal single-tenant admin authentication now (later phases reuse it):

- Admin credentials from env (username + password hash, never plaintext)
- httpOnly + Secure + SameSite session cookie, CSRF protection for
  state-changing requests, login rate limiting, generic error messages
- Middleware to protect state-changing endpoints; read-only market endpoints
  stay public by default (AUTH_REQUIRE_READ=false)
- Nullable user_id convention on future user-owned tables

Guard rails:

- MockProvider and dev seed data are forbidden when APP_ENV=production: the
  app refuses to start.
- When MockProvider is active in any other environment, every API response
  includes meta.simulated=true and the UI shows a persistent, non-dismissible
  "SIMULATED DATA" banner.
- Secrets only via environment; secret scanning in pre-commit and CI.

================================================== 7. FRONTEND FOUNDATION
==================================================

The frontend UI already exists in `frontend/` (created before Phase 0) and its visual design is preserved (see
"Existing UI (preserve)" in `CLAUDE.md`). Start with an audit, not a scaffold. Inventory the existing views,
components, tokens, fonts, charts, stores, services, and schemas, and write the findings to
`docs/design-audit.md`: what exists, which Phase 0 items it already covers, deviations from the anti-slop
rules, and any mock or hard-coded data, LLM SDK usage or API keys, external scripts or fonts (including
import maps and CDN links), and forbidden wording. For every item in this section: reuse the existing
implementation if one exists and works and extend it; create it only if it is missing. Do not restyle,
re-layout, rename, or replace existing components or screens. Existing files may be changed only to fix bugs,
remove hard-coded or fabricated data, replace forbidden wording, self-host external assets with identical
appearance, remove LLM calls and frontend secrets, and wire to the API.

- Folder structure: components/{market,charts,indicators,common}, views,
  stores, services, schemas, types, composables, router, layouts, i18n.
- Centralized Ky client (base URL, timeout, retry policy, error mapping to
  the shared error code union, credentials/CSRF handling, Zod validation
  helper that logs failures and surfaces a typed error).
- i18n with vue-i18n from day one. All user-visible strings via message keys.
  Supported locales per the DATA SOURCE DECISION. Number/date/currency
  formatting through one shared formatter module using Intl with the active
  locale and timezone. Lint rule or test that fails on hard-coded UI strings
  in templates where practical, applied to new or changed files only. Existing hard-coded strings are listed in
  `docs/design-audit.md` and migrated only when their file is otherwise touched or the owner approves a
  dedicated task.
- Design tokens:
  - tokens.css with dark (default) and light themes, mapped into the Tailwind
    theme
  - self-hosted fonts (UI: IBM Plex Sans; numbers: JetBrains Mono or IBM Plex
    Mono, tabular-nums)
  - Enforcement: lint rules (stylelint/ESLint) for raw hex/rgb colors outside tokens.css, box-shadow,
    gradients, and radius greater than 4px. They are errors for files created after Phase 0 and warnings for
    files that already existed (one baseline list, for example ESLint/stylelint overrides). CI must not fail
    because of the existing UI. The existing tokens.css is extended, not replaced, and new token values that
    represent existing colors keep the exact same value.
- Shared primitives: Panel (with header strip), StatRow, ValueChange (signed, glyph, not color alone),
  StatusBadge, SegmentedControl, ChipToggle, SkeletonBlock, ErrorState, EmptyState, SimulatedDataBanner. For
  each one, reuse an existing equivalent component if there is one (document the mapping in
  `docs/design-system.md`, do not rename or restyle it). Build a primitive only if it is missing, in the
  existing visual style, and add tests for the ones you build or touch.
- Layout shell: sidebar (icon rail), top bar (title, instrument, data status,
  timestamp, theme toggle, user menu with login state), router with auth
  guard.
- Accessibility baseline: WCAG AA contrast for both themes (verify tokens),
  full keyboard navigation for primitives, visible focus states, aria labels,
  prefers-reduced-motion respected, chart text alternatives planned.
- Highcharts: one centralized theme module and a base chart wrapper
  (props in, no data fetching). Not a full chart yet.

================================================== 8. TESTING FOUNDATION
==================================================

- Backend: unit tests for validation layer, config, error mapping, time
  handling; integration tests with a real PostgreSQL (containerized) for
  repositories and migrations; contract tests against openapi.yaml.
- Frontend: Vitest for formatter, time module, Ky client error mapping, Zod
  helper, primitives (render + a11y checks).
- E2E: Playwright in /e2e with a docker-compose test profile. Implement smoke
  tests now: app loads, health/ready pass, login works, simulated banner
  appears when mock is active, theme toggle persists.
- Visual review loop: Playwright captures screenshots of the existing shell and primitives (dark and light,
  desktop and tablet widths) into /e2e/screenshots as the visual baseline. Review NEW or CHANGED components
  against the anti-slop checklist in CLAUDE.md and fix violations. Violations in existing screens are listed in
  `docs/design-audit.md`, not fixed.

================================================== 9. DOCKER
==================================================

Services: frontend, backend, postgres only. Multi-stage builds, non-root
users, pinned base images, healthchecks, restart policies, named volumes for
Postgres. Development compose is the existing root `docker-compose.yml` (hot reload). The production compose skeleton
is `deploy/compose.prod.yml` (hardened in Phase 5).
Do not publish the database port in production compose.

================================================== 10. DOCUMENTATION
==================================================

- README.md: overview, architecture diagram (Markdown/Mermaid), quick start,
  Make targets, env vars, database conventions, provider abstraction,
  contract workflow, testing, CI, roadmap Phases 0-5.
- docs/decisions/0001-data-source.md: the owner's decisions, capability
  summary, open questions.
- docs/decisions/0002-migrations-and-numeric-types.md
- docs/decisions/0003-openapi-as-contract.md
- docs/database.md, docs/design-system.md (tokens, type scale, primitives,
  lint rules; if the file already exists from the moved DESIGN.md, extend it and document the existing look
  as it is).

================================================== 11. FINAL DELIVERABLE AND VERIFICATION
==================================================

The owner must be able to: clone, copy .env.example, run `make compose-up`,
open the empty shell, log in, and see health/ready pass; run `make test`,
`make lint`, `make e2e`, `make contract-check` successfully; and start Phase 1
without changing any foundation decision.

Verify and report:

- Backend and frontend build; all tests pass; lint clean
- Migrations: fresh up, down, up again
- Contract check passes
- CI workflow runs cleanly (or is proven by running its steps locally)
- Production start is refused with MockProvider enabled (test)
- Simulated banner appears with mock (E2E)
- No secrets committed; secret scan clean
- Screenshots reviewed against the anti-slop checklist
- List of open decisions from the DATA SOURCE DECISION that were left blank

Build incrementally and confirm each step before continuing:

1. repo, tooling, Make targets, CI skeleton
2. config, logging, middleware, error registry
3. migrations tool, first migration, DB integration tests
4. OpenAPI skeleton, envelope, health/ready, contract tests
5. provider interfaces, capability model, mock, validation layer, CSV
   provider, real provider if specified
6. auth and guard rails
7. frontend scaffold, Ky client, i18n, formatter, tokens and lint rules
8. primitives, layout shell, chart wrapper and theme
9. E2E smoke tests and screenshot review
10. Docker, docs, final verification

Prioritize correctness of foundations over speed. Do not build product
features.
