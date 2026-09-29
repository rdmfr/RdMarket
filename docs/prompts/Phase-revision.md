# Phase prompt revisions: align Phase 0 and Phase 5 with the repository layout, branch model, and GitHub Actions

Read `CLAUDE.md` first and follow it (including "Git, GitHub, and attribution"). Never run `git commit` or
`git push`.

Task: apply the edits below to `docs/prompts/Phase0.md` to `Phase5.md` (after
`Restructure.md` has been executed). Change nothing else. For each edit, search for the exact FIND text. If it is not found
exactly, do not guess: report which edit was skipped and why. Phases 1 and 3 need no workflow edit (only the UI edits in section F). Phase 2 edits are in
section C, Phase 4 edits are in section D, and the edits that preserve the existing UI (Phases 0, 1, 2, 3, 5)
are in section F.

Report at the end: every edit applied, every edit skipped, and any open decision.

---

## A. Phase0.md

### A1. Monorepo and CLAUDE.md location

FIND:
```
Monorepo:
/frontend /backend /docs /deploy /e2e
CLAUDE.md (already provided by the owner; do not rewrite it, only append
learned repo-specific commands to a clearly marked section if needed)
```

REPLACE WITH:
```
Monorepo:
/frontend /backend /docs /deploy /e2e
CLAUDE.md at the repository root (already provided by the owner; do not rewrite it, only append
learned repo-specific commands to a clearly marked section if needed). Phase prompts live in docs/prompts/.
The repository may already contain `.github/workflows/` files and a root `docker-compose.yml`: extend and
reuse them, never recreate or rename them.
```

### A2. Package manager

FIND:
```
- Frontend: ESLint, Prettier, vue-tsc type-check, Vitest
```

REPLACE WITH:
```
- Frontend: npm (package-lock.json committed), ESLint, Prettier, vue-tsc type-check, Vitest
```

### A3. CI workflow

FIND:
```
- CI workflow (GitHub Actions or equivalent) that runs on every push/PR:
  lint, type-check, unit tests, migration test on a fresh Postgres, build
  frontend and backend, contract check, docker compose smoke test
```

REPLACE WITH:
```
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
```

### A4. Compose files

FIND:
```
Separate dev and production compose files (dev with hot reload).
```

REPLACE WITH:
```
Development compose is the existing root `docker-compose.yml` (hot reload). The production compose skeleton
is `deploy/compose.prod.yml` (hardened in Phase 5).
```

---

## B. Phase5.md

### B1. Path of the prompt folder (global)

Replace every occurrence of `src/docs/prompts/` with `docs/prompts/`.

### B2. Registry and staging decisions

FIND:
```
Registry for images:               <GitHub Container Registry | other | none>
```

REPLACE WITH:
```
Registry for images:               GitHub Container Registry (already used by docker.yml)
Staging environment:               <same host, separate compose project | separate host | none>
Staging auto-deploy from develop:  <yes | no>
```

### B3. First push and branch model

FIND:
```
The repository `rdmfr/RdMarket` is public and currently empty. The first push sets the public history, so
audit before it happens.
```

REPLACE WITH:
```
The repository `rdmfr/RdMarket` is public and currently empty. The first push creates `main` and sets the
public history, so audit before it happens. Branch protection cannot exist before the first push: after it,
the owner creates `develop` and applies the protection settings below. From then on all changes go through
pull requests (`feature/*` -> `develop` -> `main`) and version tags are created on `main` only.
```

### B4. Repository settings

FIND:
```
- Repository settings the owner should apply (documented in `docs/operations/github-setup.md`, not automated):
  branch protection on `main`, required status checks, secret scanning and push protection, Dependabot alerts,
  private vulnerability reporting, 2FA, signed commits (optional), default branch name.
```

REPLACE WITH:
```
- Repository settings the owner should apply (documented in `docs/operations/github-setup.md`, not automated):
  branch protection on `main` and `develop` (pull request required, required status checks, conversation
  resolution, no direct pushes; required checks are the job ids `backend`, `frontend`, `repo-checks`, `e2e`
  and, once they exist, `attribution-check` and `security`), secret scanning and push protection, Dependabot
  alerts (version updates target `develop`), private vulnerability reporting, 2FA, signed commits
  (optional), default branch `main`, and GitHub Environments `staging` and `production` with their secrets and
  required reviewers on `production`.
```

### B5. Staging

FIND:
```
- Staging: a compose-based staging configuration that mirrors production, used by CI and by the owner for
  rehearsals, with synthetic data and the simulated-data banner where the mock provider is used.
```

REPLACE WITH:
```
- Staging: a compose-based staging configuration that mirrors production, used by CI and by the owner for
  rehearsals, with synthetic data and the simulated-data banner where the mock provider is used. Where it
  runs (same host with a separate compose project, a separate host, or none) follows the DEPLOYMENT DECISIONS.
```

### B6. CI, release, and deployment workflows

FIND:
```
- CI (extend the earlier pipeline): lint, type-check, unit tests, migration test (fresh, and upgrade from the
  previous release schema), contract check, build, E2E, security scans, `attribution-check`, secret scan,
  image build, compose smoke test, restore drill.
- Release workflow triggered by a version tag: build multi-stage images, tag with semantic version and short
  commit SHA, push to the registry chosen in the decisions, attach the SBOM if generated, create a GitHub
  Release with notes taken from the changelog.
- Deployment workflow: manual approval, deploys a specific tag to the target through a documented script. No
  deployment credentials in the repository. Use repository secrets and environment protection rules.
```

REPLACE WITH:
```
- CI: extend the existing files in `.github/workflows/` (`backend.yml`, `frontend.yml`, `repo-checks.yml`,
  `e2e.yml`, `docker.yml`, and `forecasting.yml` if Phase 2 created it) and keep their job ids stable. Add
  `security.yml` (job `security`: govulncheck, npm audit, pip-audit, Trivy image scan, optional SBOM; also on
  a weekly schedule), `attribution-check.yml` (job `attribution-check`), and `restore-drill.yml` (job
  `restore-drill`). The migration upgrade test (from the previous release schema) goes into `backend.yml`.
  All run on push and pull_request for `main` and `develop`, with no path filters on required checks.
- Image tags: `docker.yml` (push to `main`) publishes `sha-<7 chars>` and `edge`. `release.yml` (tag `v*`,
  created on `main`) builds multi-stage images, tags them `vX.Y.Z`, `sha-<7 chars>`, and `latest`, pushes to
  GHCR, attaches the SBOM if generated, and creates a GitHub Release with notes from the changelog.
  Production and staging compose files pin version tags (digest where practical) and never use `latest` or
  `edge`.
- `deploy-staging.yml`: only if the owner decided staging auto-deploy. Triggered by push to `develop` and by
  manual dispatch, environment `staging`, deploys the `sha-` image of that commit, runs the post-deploy smoke
  test.
- `deploy-production.yml`: manual dispatch with a version tag input, environment `production` with required
  reviewers (manual approval), deploys that specific tag through the documented script, runs the smoke test,
  and keeps the previous version available for rollback. No deployment credentials in the repository; use
  environment secrets.
```

### B7. Reverse proxy and Phase 4 endpoints

FIND:
```
- Reverse proxy: request size limits, timeouts, gzip/brotli, security headers, static asset caching with hashed
  filenames, `Cache-Control: no-store` for authenticated responses, rate limiting for auth and job endpoints.
```

REPLACE WITH:
```
- Reverse proxy: request size limits, timeouts, gzip/brotli, security headers, static asset caching with hashed
  filenames, `Cache-Control: no-store` for authenticated responses, rate limiting for auth and job endpoints.
  `/metrics` and the forecasting internal API are never routed through the proxy (Phase 4 metrics stay on the
  internal network), and the `Cache-Control` and `ETag` headers that Phase 4 sets on public read endpoints are
  preserved, not overridden.
```

---

## C. Phase2.md

### C1. Python service folder

FIND:
```
- Python service structure:
```

REPLACE WITH:
```
- Python service structure (the `forecasting/` folder is a new top-level folder at the repository root, next to
  `frontend/` and `backend/`):
```

### C2. Compose files

FIND:
```
Add one service: `forecasting`. Multi-stage slim Dockerfile, pinned dependencies, non-root user,
```

REPLACE WITH:
```
Add one service: `forecasting`, defined in the root `docker-compose.yml` and in the `deploy/compose.prod.yml`
skeleton (no published port in production). Multi-stage slim Dockerfile at `forecasting/Dockerfile`, pinned
dependencies, non-root user,
```

### C3. CI workflow and image

FIND:
```
E2E: launch a backtest from the UI, watch it complete, view the leaderboard, generate a forecast, view it on
the chart, degraded state when the service is down. Synthetic data in tests only.
```

REPLACE WITH:
```
E2E: launch a backtest from the UI, watch it complete, view the leaderboard, generate a forecast, view it on
the chart, degraded state when the service is down. Synthetic data in tests only.

CI: add `.github/workflows/forecasting.yml` (job id `forecasting`) that installs the pinned dependencies, runs
pytest, and runs `pip-audit`. Triggers: push and pull_request on `main` and `develop`, no path filters, minimal
`permissions: contents: read`. Add the `rdmarket-forecasting` image (`forecasting/Dockerfile`) to `docker.yml`
with the same tags as the other images (`sha-<7 chars>` and `edge`). Extend `backend.yml`, `frontend.yml`, and
`e2e.yml` only where the new Go, Vue, and E2E tests need it, keep their job ids stable, and list every
workflow change in the final report.
```

---

## D. Phase4.md

### D1. Compose profiles

FIND:
```
Allowed additions (all optional in the base compose file, enabled through Compose profiles so
`docker compose up` still runs the product without them):
```

REPLACE WITH:
```
Allowed additions (all optional, defined in the development compose file (root `docker-compose.yml`) and, when
the owner enables monitoring in production, in `deploy/compose.prod.yml`; enabled through Compose profiles so
`docker compose up` still runs the product without them):
```

### D2. CI

FIND:
```
- Load-test scripts run in CI in a small smoke configuration (fast), with the full run available through
  `make loadtest`.
```

REPLACE WITH:
```
- CI: add `.github/workflows/perf.yml` (job id `perf`) that runs the k6 smoke configuration and the Lighthouse
  budgets against the compose stack; the full run stays available through `make loadtest`. Do not make `perf`
  a required check until its results are stable. The bundle size budget check is a step in `frontend.yml`. The
  backend suite against PgBouncer is a separate job `backend-pgbouncer` in `backend.yml`, with the `pooling`
  profile started for that job. Triggers: push and pull_request on `main` and `develop`, minimal
  `permissions`. Keep the existing job ids stable and list every workflow change in the final report.
```

---

## E. Phases that need no workflow edit

Phase 1 and Phase 3 do not define workflows. The rule in `CLAUDE.md` ("when a phase adds tests, migrations, a
service, or an image, extend the matching workflow and list the change") covers them. The Phase 3 file was
provided twice with identical content.

---

## F. Preserve the existing UI (created with Google AI Studio)

The UI already exists. These edits change the phases from "build the UI" to "audit, fix, wire, and extend it".
They rely on the "Existing UI (preserve)" section of `CLAUDE.md`.

### F1. Phase0.md, section 7 (frontend foundation)

FIND:
```
Scaffold only; no product screens beyond a placeholder dashboard shell.
```

REPLACE WITH:
```
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
```

### F2. Phase0.md, i18n lint

FIND:
```
  locale and timezone. Lint rule or test that fails on hard-coded UI strings
  in templates where practical.
```

REPLACE WITH:
```
  locale and timezone. Lint rule or test that fails on hard-coded UI strings
  in templates where practical, applied to new or changed files only. Existing hard-coded strings are listed in
  `docs/design-audit.md` and migrated only when their file is otherwise touched or the owner approves a
  dedicated task.
```

### F3. Phase0.md, lint enforcement

FIND:
```
  - Enforcement: lint rule (stylelint/ESLint) that fails on raw hex/rgb
    colors outside tokens.css, on box-shadow, on gradients, and on radius
    greater than 4px
```

REPLACE WITH:
```
  - Enforcement: lint rules (stylelint/ESLint) for raw hex/rgb colors outside tokens.css, box-shadow,
    gradients, and radius greater than 4px. They are errors for files created after Phase 0 and warnings for
    files that already existed (one baseline list, for example ESLint/stylelint overrides). CI must not fail
    because of the existing UI. The existing tokens.css is extended, not replaced, and new token values that
    represent existing colors keep the exact same value.
```

### F4. Phase0.md, primitives

FIND:
```
- Shared primitives, built and tested now: Panel (with header strip),
  StatRow, ValueChange (signed, glyph, not color alone), StatusBadge,
  SegmentedControl, ChipToggle, SkeletonBlock, ErrorState, EmptyState,
  SimulatedDataBanner.
```

REPLACE WITH:
```
- Shared primitives: Panel (with header strip), StatRow, ValueChange (signed, glyph, not color alone),
  StatusBadge, SegmentedControl, ChipToggle, SkeletonBlock, ErrorState, EmptyState, SimulatedDataBanner. For
  each one, reuse an existing equivalent component if there is one (document the mapping in
  `docs/design-system.md`, do not rename or restyle it). Build a primitive only if it is missing, in the
  existing visual style, and add tests for the ones you build or touch.
```

### F5. Phase0.md, screenshot review loop

FIND:
```
- Visual review loop: Playwright captures screenshots of the shell and every
  primitive (dark and light, desktop and tablet widths) into
  /e2e/screenshots. Review them against the anti-slop checklist in CLAUDE.md
  and fix violations before finishing.
```

REPLACE WITH:
```
- Visual review loop: Playwright captures screenshots of the existing shell and primitives (dark and light,
  desktop and tablet widths) into /e2e/screenshots as the visual baseline. Review NEW or CHANGED components
  against the anti-slop checklist in CLAUDE.md and fix violations. Violations in existing screens are listed in
  `docs/design-audit.md`, not fixed.
```

### F6. Phase0.md, design system doc

FIND:
```
docs/design-system.md (tokens, type scale, primitives,
  lint rules).
```

REPLACE WITH:
```
docs/design-system.md (tokens, type scale, primitives,
  lint rules; if the file already exists from the moved DESIGN.md, extend it and document the existing look
  as it is).
```

### F7. Phase1.md, intro

FIND:
```
Reuse what Phase 0 already provides. Do not recreate or restyle it:
```

REPLACE WITH:
```
The frontend UI already exists and its visual design is preserved (see "Existing UI (preserve)" in
`CLAUDE.md`). Reuse, wire to the real API, and extend the existing views and components; add only what is
missing. Where the layout details in sections 2, 4, and 10 of this prompt differ from the existing UI, keep
the existing look and list the differences in the final report. Reuse what Phase 0 already provides. Do not
recreate or restyle it:
```

### F8. Phase1.md, screenshots

FIND:
```
Capture screenshots and review them against the anti-slop checklist in `CLAUDE.md`.
```

REPLACE WITH:
```
Capture screenshots; review new or changed screens against the anti-slop checklist in `CLAUDE.md`, and only
report violations in existing screens (`docs/design-audit.md`).
```

### F9. Phase2.md, frontend

FIND:
```
Follow `CLAUDE.md` design and wording rules. New route and sidebar item
```

REPLACE WITH:
```
Follow `CLAUDE.md` design and wording rules, and reuse the existing UI's components, tokens, and visual
language ("Existing UI (preserve)" in `CLAUDE.md`). New route and sidebar item
```

### F10. Phase3.md, frontend

FIND:
```
Follow `CLAUDE.md` design and wording rules. New colors
```

REPLACE WITH:
```
Follow `CLAUDE.md` design and wording rules, and reuse the existing UI's components, tokens, and visual
language ("Existing UI (preserve)" in `CLAUDE.md`). New colors
```

### F11. Phase5.md, CSP

FIND:
```
- Strict `Content-Security-Policy` (no inline scripts, hashed or nonce-based exceptions only if unavoidable),
```

REPLACE WITH:
```
- Strict `Content-Security-Policy` (no inline scripts, hashed or nonce-based exceptions only if unavoidable; if
  the existing UI relies on inline scripts, inline styles, or external assets, fix that with no visual change),
```

Phase 4 needs no edit for this: its Lighthouse accessibility and layout budgets report on the existing UI and
do not require redesign. Failing budgets caused by the existing design are recorded in the performance log and
left to the owner.