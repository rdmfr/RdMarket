# permanent.md: RdMarket Intelligence

Repository: https://github.com/rdmfr/RdMarket (public). Owner: rdmfr.
Everything committed is visible to everyone. Treat every file as public.

Note on file name: the phase prompts call this file `CLAUDE.md`. In the
repository keep it as `CLAUDE.md` (Claude Code loads that name automatically).
`permanent.md` is the same content. Do not maintain two divergent copies.

## What this is

USD/IDR market monitoring and analysis platform. An ANALYTICAL tool, not a
trading advisor. It never claims certainty about future rates and never gives
advice.

## Phases

0 Foundation
1 Market Monitoring
2 Forecasting & Backtesting
3 Economic Indicators + Alerts & Market Intelligence (one prompt file
  delivers both parts; Part A Economic Indicators, Part B Alerts & Market
  Intelligence)
4 Production Monitoring & Optimization
5 Deployment, Security Hardening & Release

- Only implement the phase the current prompt asks for. Never pull in features
  from later phases. A prompt may deliver several parts together; follow the
  prompt's own scope list.
- Never refactor earlier phases unless the prompt says so; extend them and list
  every change made to existing files in the final report.
- Existing endpoints, tables, components, and tests must keep working.
- Phase 1 endpoints never depend on economic, alert, forecasting, or
  monitoring components. If an optional service (forecasting, SMTP, economic
  provider, monitoring) is down, everything else keeps working and the
  affected UI shows a degraded state.
- Phase 5 adds no product features.

## Repository layout

`frontend/`, `backend/`, `docs/` (including `docs/prompts/`), `deploy/`, `e2e/`, and from Phase 2 `forecasting/`
(Python service), plus `CLAUDE.md`, `Makefile`, root `docker-compose.yml` (development), and `.github/`.

## Stack (fixed)

Frontend: Vue 3, TypeScript strict, Vite, Pinia, Vue Router, Tailwind, Ky,
Zod, Highcharts, Iconify, vue-i18n.
Backend: Go, GoFiber, GORM, PostgreSQL.
Forecasting (Phase 2+): Python service (statsmodels).
Infra: Docker, Docker Compose, GitHub Actions.
Allowed small additions when a phase needs them: an SMTP client, a small
in-process scheduler/cron, security and audit tooling (gitleaks, Trivy,
govulncheck, npm audit, pip-audit).
No Redis, queues, or extra services unless a phase prompt explicitly adds
them. Notification delivery uses a database-backed outbox.
Forbidden: React, Next.js, Nuxt, Laravel, Node backend, Firebase, GORM
AutoMigrate, float types for money/rates.

## Architecture rules

- Backend layering: Handler -> Service -> Repository -> Database.
  External data: Service -> Provider -> External API.
  Handlers are thin. No business logic in handlers.
- Frontend flow: Component -> Composable -> Pinia store -> API service ->
  Ky -> Go API. No HTTP calls in components. Chart components receive props
  and never fetch.
- Pinia only for genuinely shared state.
- Provider abstractions for every external data source (exchange rates,
  economic data).
- Domain logic (statistics, indicators, alert evaluators, regime, brief
  templates) is pure functions, unit tested without a database.
- Internal domain events go through the in-process dispatcher
  (`ExchangeRateUpdated`, `EconomicObservationsUpdated`). Producers never know
  about consumers.
- Catalogs (indicators, alert rule types) are data/config plus one
  registration, not scattered code.
- One Highcharts theme, one formatter module, one time module, one error-code
  registry. No duplicated config or ad-hoc formatting.
- API contract: docs/openapi.yaml is the source of truth. Zod schemas and Go
  responses must match it (`make contract-check`).
- Response envelope always: { data, meta, error }.
- All DB changes via versioned SQL migrations with up and down. Do not modify
  existing tables from earlier phases unless the prompt says so. In production
  migrations are forward-only, run as a separate step, and always preceded by
  a verified backup.
- Timestamps: UTC in storage (TIMESTAMPTZ); convert to Asia/Jakarta only for
  display. Rates and economic values: NUMERIC.

## Data integrity (non-negotiable)

- NEVER fabricate, estimate, interpolate, or randomize market or economic
  data. Missing data is shown as "Not enough data" or an em dash.
- Estimates (e.g. expected release dates) are always labeled "estimated".
- Suspect data is flagged, never silently dropped or trusted.
- Mock/dev data is only allowed outside production and always shows the
  persistent "SIMULATED DATA" banner. Production refuses to start with it.
- Every number is traceable to a source, timestamp, and retrieval time
  (economic data also: reference period and release timestamp).
- Preserve revisions; never overwrite historical economic data.
- Never assume zero publication lag; unknown release time uses the configured
  per-series lag.
- Point-in-time correctness: no look-ahead leakage in any analysis, alert
  evaluation, brief, or backtest.
- Alerts evaluate only real stored data. Missing or insufficient data means
  "not evaluable" with a reason, never a guess. Suppressed alerts are recorded
  with a reason, never dropped.
- No AI/LLM-generated text in product output. Text (notifications, briefs,
  labels) comes from deterministic templates over real data; every sentence
  maps to a stored fact.
- Only use data sources whose terms allow the use. Do not scrape sites that
  forbid it. Document terms in docs/legal/data-sources.md.

## Wording rules

Allowed: "Current trend", "Historical movement", "Observed volatility",
"Statistical indicator", "Forecast", "Prediction interval", "Historical
co-movement", "Condition met", "Threshold crossed", "Latest release",
"Observed change", "Trend", "Regime", "Reference period".
Forbidden (UI, notifications, briefs, docs): "will rise/fall", "guaranteed",
"buy", "sell", "bullish/bearish signal", "drives", "causes", "predicts"
(as a claim of certainty), "recommended", "AI predicts", and any advice or
certainty language.
Forecasts are always shown with intervals and a baseline comparison nearby.
Market-condition, forecast, correlation, and brief areas carry a muted note:
not financial advice. A test scans templates and i18n files for forbidden
wording.

## Existing UI (preserve)

The UI already exists in `frontend/` (created before the phases, with Google AI Studio). Its visual design is
preserved. The job is to fix, wire, and extend it, not to redesign it.

- Do not restyle, re-layout, rename, or replace existing screens and components unless the owner asks.
  Allowed changes to existing files: fix bugs, wire to the real API, add features, extract hard-coded colors
  into tokens with identical values, and the integrity fixes listed below.
- New screens and components reuse the existing components, tokens, fonts, spacing, and layout so the product
  stays consistent. Where the anti-slop rules below conflict with the existing look, the existing look wins
  for consistency; record the deviation in `docs/design-audit.md` and let the owner decide later.
- Integrity fixes apply to existing UI too, because they are not style: no fabricated, mock, or hard-coded
  market or economic data (replace with API data or "Not enough data"/empty states, with the SIMULATED DATA
  banner for mock); forbidden or advice wording replaced with allowed wording (text change only); no
  LLM/AI-generated text, no LLM API calls, no API keys or `process.env` secrets in the frontend bundle; no
  external scripts, CDN imports, import maps, or trackers (self-host the same fonts and assets so the look does
  not change); the fixed stack (charts through Highcharts; if an existing chart uses another library, replace
  it internally and match the existing look, owner confirms).
- Style lint rules (raw colors, shadows, gradients, radius) are errors for new files and warnings for files
  that existed before Phase 0 (one baseline list). They never fail CI on the existing UI.
- i18n: new or changed strings go through keys. Existing hard-coded strings are listed in the audit and
  migrated only when their file is otherwise touched, or through a dedicated owner-approved task.
- Every change to an existing component keeps its diff limited to what the task needs.

## Design rules (anti-slop)

These rules apply to NEW screens and components and to code a phase changes. Existing UI: see "Existing UI".
Terminal-style, dense, dark by default, data first.

- Colors ONLY from tokens.css (including alert severities, event markers,
  series colors). No raw hex in components.
- Fonts: IBM Plex Sans (UI), mono with tabular-nums for ALL numbers. No
  Inter/Poppins.
- Radius <= 4px. No box-shadow. No gradients. No glassmorphism. No emojis.
  No decorative illustrations. Motion <= 150ms, no entrance animations.
- Panels are flat with 1px borders and a compact header strip. Tables and
  key/value rows, not grids of big stat cards.
- One icon set, 16px, stroke 1.5, no icons in colored circles.
- Up/down and severity never rely on color alone: sign/text + glyph + color.
- Skeletons match final layout. Errors: one plain sentence + Retry. Empty
  states: text and one action, no artwork.
- Copy is terse and factual. All UI strings via i18n keys.
- Accessibility: AA contrast, keyboard navigable, visible focus,
  reduced-motion respected.
- Timers: one shared timer per concern, never overlapping, cleaned up on
  unmount, paused while the tab is hidden.
  Reject and redo any NEW or CHANGED screen that violates this list. Review Playwright
  screenshots of new or changed screens against it before finishing. Violations found in existing screens are
  listed in `docs/design-audit.md` and not fixed unless the owner asks.

## Security

- No secrets in code, frontend, images, logs, or the repository; env vars or
  Docker secrets only; keep .env.example current.
- Secrets in the database are encrypted at rest (AES-GCM, key from env) and
  masked in API responses. Never logged.
- Validate all input server-side. CSRF on state-changing requests. Rate limit
  auth, job, notification-test, and CSV-import endpoints. Request size limits.
  Safe error messages (no stack traces to clients).
- Block SSRF for any user-supplied URL (webhooks): resolve and block private,
  loopback, link-local, and metadata ranges; block redirects into them;
  HTTPS in production; cap time and response size.
- Mutating endpoints require auth. Cookies: HttpOnly, Secure, SameSite.
- Production: strict CSP and security headers, non-root containers, read-only
  root filesystem where possible, no-new-privileges, dropped capabilities.
  Database and forecasting ports are never published.
- Production refuses to start with MockProvider, missing or weak secrets,
  default credentials, or APP_ENV other than production.
- Audit log for auth events and admin actions (no secrets).
- No third-party trackers, analytics, or external scripts.

## Git, GitHub, and attribution

The repository is public. The owner is the sole author.

- Never run `git push`. Never commit unless explicitly asked. Never change git
  config, author, or committer identity. Never use a bot identity.
- Prepare changes, suggest commit messages in plain descriptive language
  (Conventional Commits), and let the owner review, commit, and push.
- NO AI attribution anywhere in commits, PRs, issues, releases, code,
  comments, docs, package/OCI metadata, changelog, or badges:
  no `Co-authored-by:` or `Signed-off-by:` trailers naming a tool or vendor,
  no "Generated with ...", "Created by ...", "AI-assisted", robot emoji, or
  tool signatures or links, no `authors`/`maintainers` fields crediting a tool.
- Enforcement: `commit-msg` hook (installed by `make setup`), CI job
  `attribution-check`, and `make audit`, all using
  `.github/forbidden-attribution.txt` (case-insensitive regex list). Files that
  legitimately instruct tooling (`CLAUDE.md`, `permanent.md`, `.claude/`,
  `docs/prompts/`) are on an owner-controlled, documented allowlist.
- Whether agent-instruction files are committed or git-ignored is the owner's
  decision; implement the owner's choice and document it.
- If a course, employer, license, or platform requires AI-use disclosure, that
  requirement is the owner's to follow.
- Never choose a license on the owner's behalf; create LICENSE only when the
  owner has decided.
- Never commit: `.env`, keys, `*.pem`, dumps, IDE/OS files, build output,
  `node_modules`, non-synthetic sample data, local absolute paths, personal
  data, private URLs.
- Never rewrite published history.

## CI/CD (GitHub Actions)

Branch model (owner decision):

- `main` is the production line, `develop` is the development line, `feature/*` are short-lived branches.
- Flow: `feature/*` -> pull request into `develop` -> checks pass -> merge; later `develop` -> pull request
  into `main`. Version tags `v*` are created on `main` only.
- `main` and `develop` are protected: pull request required, required status checks, conversation
  resolution, no direct pushes. The owner applies these settings in GitHub.
- The assistant never creates branches, tags, or pushes. The owner does.

CI/CD stages (called "Stage" so they are not confused with product Phases 0 to 5):
1 CI, 2 Container (GHCR), 3 Staging, 4 Production, 5 Advanced.
Stages 1 and 2 exist from Phase 0. Stages 3 to 5 are built in Phase 5. No deployment before Phase 5.

Workflow files live in `.github/workflows/`, one file per component or concern. Extend the existing files, never
recreate or rename them. Job ids are the required status check names and must stay stable.

| File | Job id | Built in | Purpose |
|---|---|---|---|
| backend.yml | backend | Phase 0 | gofmt, vet, golangci-lint, race tests, Postgres integration and migration tests, build; job `backend-pgbouncer` from Phase 4 |
| frontend.yml | frontend | Phase 0 | npm ci, ESLint, Prettier, vue-tsc, Vitest, build; bundle size budget from Phase 4 |
| repo-checks.yml | repo-checks | Phase 0 | secret scan, contract check, forbidden wording scan, compose smoke test |
| e2e.yml | e2e | Phase 0 | Playwright against the compose test profile |
| docker.yml | docker | Phase 0 | push to main: build and push images to GHCR |
| forecasting.yml | forecasting | Phase 2 | Python tests and audit; image added to docker.yml |
| perf.yml | perf | Phase 4 | k6 smoke and Lighthouse budgets; advisory (not required) until stable |
| security.yml | security | Phase 5 | govulncheck, npm audit, pip-audit, Trivy, optional SBOM |
| attribution-check.yml | attribution-check | Phase 5 | forbidden attribution scan |
| restore-drill.yml | restore-drill | Phase 5 | backup restore drill |
| release.yml | release | Phase 5 | tag v*: versioned images and GitHub Release |
| deploy-staging.yml | deploy-staging | Phase 5 | only if the owner chose staging auto-deploy from develop |
| deploy-production.yml | deploy-production | Phase 5 | manual dispatch, environment approval, specific tag |

Rules:

- CI workflows run on push and pull_request for `main` and `develop`. No path filters on workflows whose jobs
  are required checks (a skipped required check blocks merging).
- Package manager is npm: `npm ci`, `package-lock.json` committed. Go version follows `backend/go.mod`.
- When a phase adds tests, migrations, a service, or an image, extend the matching workflow and list the
  change in the final report. A new service gets its own workflow and image.
- Third-party actions are pinned by full commit SHA with a version comment (owner decision on when: now or
  Phase 5; until then tags are allowed and listed as an open decision). Minimal top-level
  `permissions` (default `contents: read`), raised per job only when needed. `concurrency` groups cancel
  superseded runs (never for image publishing or deployment). No `pull_request_target` with untrusted checkout.
- Secrets only from repository or environment secrets. Never echo them. Deployment credentials never live in
  the repository.
- Images live in GHCR, lowercase: `ghcr.io/rdmfr/rdmarket-<component>`. Push to `main` publishes `sha-<7 chars>`
  and `edge`. A `v*` tag publishes `vX.Y.Z`, `sha-<7 chars>`, and `latest`. Production and staging compose files
  pin version tags (digest where practical) and never use `latest` or `edge`.
- Deployment: staging auto-deploy from `develop` only if the owner decided so; production is always manual
  dispatch of a specific tag with required reviewers on the `production` environment.
- Dependabot (or Renovate) covers Go modules, npm, pip, GitHub Actions, and Docker base images, targets
  `develop`, with grouped low-noise updates.
- Release notes come from CHANGELOG.md. Optional SBOM.
- Repository settings (applied by the owner, documented in docs/operations/github-setup.md): branch
  protection on `main` and `develop` with the job ids above as required checks, secret scanning and push
  protection, Dependabot alerts, private vulnerability reporting, 2FA, environments `staging` and `production`.

## Workflow

- Read this file, README.md, docs/, and existing code before changing
  anything. Follow existing conventions.
- Build incrementally in the order the prompt gives; confirm each step
  (build + tests) before moving on.
- Prefer small components and functions; no huge Vue components or Go
  handlers; no duplicated logic.
- Write tests with the code: pure logic gets unit tests, DB code gets
  integration tests against real PostgreSQL, critical flows get E2E.
- Use synthetic data in tests only.
- Owner-filled decision blocks in prompts (data source, deployment) are never
  overridden. A blank field is never invented: implement the documented
  default, mark it "undecided" in the matching docs/decisions file, and list
  it as an open decision in the final report.
- Before finishing a phase: run build, lint, type-check, tests, contract
  check, migrations (fresh + existing DB), E2E; verify no secrets committed;
  verify forbidden wording and forbidden attribution are absent; list all
  changes to files from earlier phases; list open decisions and known
  limitations.
- If a requirement is ambiguous or a needed decision is missing, state the
  assumption in the final report instead of inventing data or behavior.

## Commands

(Filled in during Phase 0 and extended per phase. Append repo-specific
commands only inside this section.)

Phase 0: make setup / dev / build / test / lint / seed-dev / e2e /
migrate-up / migrate-down / contract-check / compose-up / compose-down
Phase 5 additions: make audit / preflight / restore-drill
Each Make target runs the same steps as the matching CI job. Dev compose file: root docker-compose.yml.
Production compose: deploy/compose.prod.yml.
