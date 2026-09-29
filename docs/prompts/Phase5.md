# Phase 5: Deployment, Security Hardening & Release

Read `CLAUDE.md`, `README.md`, `docs/`, and the existing code first. Follow `CLAUDE.md` for the whole phase,
including the "Git, GitHub, and attribution" rules.

Phases 0 to 4 already exist and work. Extend them; do not rebuild or refactor them. Existing endpoints,
tables, and components must keep working. List every change made to existing files in the final report.

Do NOT add product features. This phase makes the platform safe to deploy, recover, and publish.

The repository is public: https://github.com/rdmfr/RdMarket. Everything committed is visible to everyone.

---

## DEPLOYMENT DECISIONS (FILLED IN BY THE OWNER, DO NOT OVERRIDE)

```
Hosting target:                    <VPS provider | home server | managed platform>
Operating system / arch:           <e.g. Ubuntu LTS, amd64>
Domain name:                       <e.g. example.com, or "none yet">
TLS method:                        <Let's Encrypt via certbot | Caddy | provider-managed>
Public access to read endpoints:   <yes/no; AUTH_REQUIRE_READ value>
Registry for images:               GitHub Container Registry (already used by docker.yml)
Staging environment:               <same host, separate compose project | separate host | none>
Staging auto-deploy from develop:  <yes | no>
Backup storage location (off-host):<S3-compatible bucket | other provider | local disk only>
Backup retention (days):           <e.g. 14 daily, 8 weekly>
Recovery targets:                  <RPO / RTO the owner accepts>
Monitoring stack in production:    <yes/no>
Source license:                    <MIT | Apache-2.0 | GPL-3.0 | proprietary | undecided>
Notification email / sender:       <address>
```

If a field is blank, do NOT invent a value. Implement the documented default, mark the field "undecided" in
`docs/decisions/0010-deployment.md`, and list it as an open decision in the final report. In particular,
never choose a license on the owner's behalf: create the file only when the owner has decided.

---

## 1. Objective

Take the platform from "works on a developer machine" to "safe to run on the internet and safe to publish":

- A repeatable production deployment
- Security hardening at application, container, and host-documentation level
- Verified backups and a tested restore procedure
- A CI/CD pipeline that builds, scans, and releases
- A clean, publishable GitHub repository without secrets, leftovers, or AI attribution

## 2. Repository hygiene and first public push

The repository `rdmfr/RdMarket` is public and currently empty. The first push creates `main` and sets the
public history, so audit before it happens. Branch protection cannot exist before the first push: after it,
the owner creates `develop` and applies the protection settings below. From then on all changes go through
pull requests (`feature/*` -> `develop` -> `main`) and version tags are created on `main` only.

- Never run `git push` yourself. Prepare everything, produce a checklist, and let the owner commit and push.
- Pre-push audit (`make audit`, also run in CI):
  - secret scan of the working tree AND the full git history (gitleaks or equivalent)
  - forbidden files: `.env`, `*.pem`, `*.key`, database dumps, local IDE and OS files, build output,
    `node_modules`, large binaries, sample data that is not synthetic
  - no personal data, local absolute paths, or private URLs in code, docs, or configs
  - **no AI attribution** (see section 3)
  - license file present only if decided by the owner
- Required repository files: `README.md` (accurate for the current state), `.gitignore`, `.gitattributes`
  (normalize line endings), `.editorconfig`, `SECURITY.md` (how to report vulnerabilities, without promising
  response times the owner cannot keep), `docs/` (architecture, operations, decisions). `CONTRIBUTING.md` and
  issue/PR templates are optional and must be minimal.
- Dependency updates: configure Dependabot or Renovate for Go modules, npm, Python, GitHub Actions, and Docker
  base images, with grouped and low-noise settings.
- Repository settings the owner should apply (documented in `docs/operations/github-setup.md`, not automated):
  branch protection on `main` and `develop` (pull request required, required status checks, conversation
  resolution, no direct pushes; required checks are the job ids `backend`, `frontend`, `repo-checks`, `e2e`
  and, once they exist, `attribution-check` and `security`), secret scanning and push protection, Dependabot
  alerts (version updates target `develop`), private vulnerability reporting, 2FA, signed commits
  (optional), default branch `main`, and GitHub Environments `staging` and `production` with their secrets and
  required reviewers on `production`.
- Commit history: the owner writes and pushes the commits. Recommend small, coherent commits using Conventional
  Commits. If the history must be cleaned before the first push, document the safe procedure for the owner to
  run themselves. Never rewrite published history.

## 3. No AI attribution (hard requirement)

Nothing pushed to GitHub, or produced for it, may contain watermarks, credits, or contribution markers that
attribute work to an AI tool or assistant. The owner is the sole author.

Forbidden anywhere in commits, PRs, issues, releases, code, comments, docs, and metadata:

- `Co-authored-by:` or `Signed-off-by:` trailers naming an AI tool, assistant, bot, or vendor
- Lines such as "Generated with ...", "Created by ...", "Written by ...", "Made with AI", "AI-assisted", robot
  emoji, or tool signatures and links
- Author or committer identities other than the owner's (never change git config, never use a bot identity)
- File headers, footers, banners, or `authors` and `maintainers` fields crediting a tool
- Docker/OCI labels, package metadata, changelog entries, release notes, or README badges crediting a tool

Enforcement (implement all):

1. `commit-msg` git hook (installed by `make setup`) that rejects messages matching a configurable pattern list
   (`.github/forbidden-attribution.txt`, case-insensitive, regular expressions). Include patterns for
   co-author and "generated with" lines, common AI assistant and vendor names in trailers, and robot emoji
2. CI job `attribution-check` that runs on every push and PR: scans all commit messages in the range, the PR
   title and description, and tracked files against the same pattern list. Files that legitimately instruct
  tooling (for example `CLAUDE.md`, `docs/prompts/`) are handled by an explicit allowlist that the owner
   controls, and the allowlist is documented
3. `make audit` runs the same scan locally before pushing
4. Document in `docs/operations/git-workflow.md` how to disable default attribution in coding assistants (for
   Claude Code, the `attribution` setting with empty `commit` and `pr` values in the user-level settings file;
   verify the setting name against the tool's current documentation) and that the owner reviews every commit
   message before pushing
5. The assistant itself: never runs `git push`, never commits unless asked, never edits git identity, and
   writes commit message suggestions in plain descriptive language with no attribution lines

Whether agent-instruction files (`CLAUDE.md`, `docs/prompts/`, `.claude/`) are committed or git-ignored is
the owner's decision. Implement whichever the owner chooses and document it. If the owner or an organization
requires AI-use disclosure for a specific purpose (course, employer, license, platform rules), that
requirement is the owner's to follow.

## 4. Production deployment

Structure under `deploy/`:

```
deploy/
  compose.prod.yml
  nginx/            (or the chosen reverse proxy configuration)
  env/              (templates only, never real values)
  scripts/          (deploy, rollback, backup, restore, healthcheck)
  monitoring/       (from Phase 4, optional)
```

- Production compose: pinned image tags (by version and digest where practical), `restart: unless-stopped`,
  healthchecks, resource limits, log rotation, named volumes, an internal network for backend, database, and
  forecasting, and only the reverse proxy published to the host. The database and forecasting ports are never
  published.
- TLS according to the DEPLOYMENT DECISIONS (automatic certificate issuance and renewal, HTTP to HTTPS
  redirect, HSTS). If the decision is blank, document the options and implement the reverse-proxy
  configuration without inventing a domain.
- Reverse proxy: request size limits, timeouts, gzip/brotli, security headers, static asset caching with hashed
  filenames, `Cache-Control: no-store` for authenticated responses, rate limiting for auth and job endpoints.
  `/metrics` and the forecasting internal API are never routed through the proxy (Phase 4 metrics stay on the
  internal network), and the `Cache-Control` and `ETag` headers that Phase 4 sets on public read endpoints are
  preserved, not overridden.
- Environment: production refuses to start with `MockProvider`, missing secrets, default credentials, weak
  session secrets, or `APP_ENV` other than `production`. Provide a preflight command (`make preflight`) that
  validates a production environment file and prints a redacted report.
- Admin bootstrap: documented, one-time, secure procedure to create the admin credentials (password hash
  generation command, no plaintext in files or shell history guidance).
- Database in production: dedicated roles (app, migration, read-only forecasting, backup), least privilege,
  no superuser use by the application, `pg_stat_statements` if Phase 4 monitoring is enabled.
- Deployment procedure: pull images, run backup, run migrations (separate step, direct DB connection), start new
  containers, run post-deploy smoke tests, keep the previous version available for rollback.
- Rollback procedure: previous image tag plus a decision guide for down-migration versus restore from backup.
  Test both in a staging-like environment.
- Staging: a compose-based staging configuration that mirrors production, used by CI and by the owner for
  rehearsals, with synthetic data and the simulated-data banner where the mock provider is used. Where it
  runs (same host with a separate compose project, a separate host, or none) follows the DEPLOYMENT DECISIONS.

## 5. Security hardening

**Application**
- Strict `Content-Security-Policy` (no inline scripts, hashed or nonce-based exceptions only if unavoidable; if
  the existing UI relies on inline scripts, inline styles, or external assets, fix that with no visual change),
  `Strict-Transport-Security`, `X-Content-Type-Options`, `Referrer-Policy`, `Permissions-Policy`,
  `frame-ancestors` restrictions. Verify Highcharts and fonts work under the policy.
- Cookies: `HttpOnly`, `Secure`, `SameSite`, short lifetime with rotation on login, invalidation on logout,
  CSRF protection verified.
- Auth: login rate limiting and lockout backoff, password hashing parameters documented and tested, generic
  errors, audit log for auth events and administrative actions (no secrets in logs).
- CORS with explicit origins, request size limits, input validation on every endpoint, safe error messages
  (no stack traces or internal details to clients).
- Rate limits per IP and per route class (public read, auth, job creation, notification test, CSV import).
- Webhook SSRF protections re-verified with tests. CSV import hardened against oversized files, formula
  injection when exported, and malformed encodings.
- Secrets: never in images, logs, or the repository; environment variables or Docker secrets; documented
  rotation procedure for the session secret, encryption key (with re-encryption of stored channel secrets),
  provider keys, SMTP credentials, and database passwords.

**Containers and supply chain**
- Non-root users, read-only root filesystem where possible, `no-new-privileges`, dropped capabilities, minimal
  base images, no build tools in runtime images, pinned versions.
- Scan images (Trivy or equivalent) and dependencies (`govulncheck`, `npm audit`, `pip-audit`) in CI. Failing
  thresholds are documented and configurable, with an explicit process for accepted risks.
- Generate an SBOM for release images (optional but preferred).
- Pin GitHub Actions by commit SHA and use minimal `permissions` for workflows.

**Documentation**
- `docs/security/threat-model.md`: assets, entry points, trust boundaries, main threats and mitigations (keep it
  practical), including the provider, webhook, CSV import, admin login, and internal service boundaries.
- `docs/security/host-hardening.md`: firewall, SSH key-only login, automatic security updates, non-root deploy
  user, fail2ban or equivalent, disk encryption notes, time synchronization. Documentation only. Do not
  change any host.

## 6. Backup and disaster recovery

- Scheduled logical backups (`pg_dump`, custom format) with encryption, integrity checksum, and off-host
  upload according to the DEPLOYMENT DECISIONS. Backups of secrets and environment files are the owner's
  responsibility and documented separately, never stored unencrypted next to database backups.
- Retention policy from the decisions, with pruning that can never delete the most recent verified backup.
- Restore script and a **restore drill**: `make restore-drill` restores the latest backup into a temporary
  database, runs integrity checks (row counts, latest timestamps, migration version, application smoke test)
  and reports success or failure. Run it in CI against a synthetic dataset and document how the owner runs
  it against a real backup on a schedule.
- Backup monitoring: freshness metric and operational alert when the latest successful backup is too old
  (if Phase 4 monitoring is enabled).
- Document RPO and RTO from the decisions, and what is not covered (for example point-in-time recovery, which
  may be documented as a future option such as WAL archiving).
- Runbooks: restore from backup, failed migration, corrupted or missing data, provider outage, lost secrets,
  key rotation.

## 7. CI/CD and releases

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
- `CHANGELOG.md` in Keep a Changelog format, written in plain language by change type (Added, Changed, Fixed,
  Security). Versioning follows SemVer. Release notes contain no attribution lines.
- Database migration policy: forward-only in production by default, every migration reviewed for lock impact
  and reversibility, the migration step always preceded by a verified backup.

## 8. Legal, data licensing, and disclaimers

- Review and document the terms of use and attribution requirements of every data provider in use (currently
  Frankfurter / ECB reference rates and any indicator source) in `docs/legal/data-sources.md`, including
  redistribution limits for a public site. Do not assume permission. Anything unclear is listed as an open
  question for the owner.
- Add a persistent, terse footer note and an "About & Disclaimer" page through i18n: analytical tool, data
  from named sources with timestamps, reference rates are not transaction rates, not financial advice. Follow
  the wording rules in `CLAUDE.md`.
- Privacy: document what personal data exists (admin account, IP addresses in logs, cookies), retention of
  logs, and that no third-party trackers or analytics are used. Do not add analytics or third-party scripts.

## 9. Operational documentation

Create or update: `docs/operations/deployment.md`, `rollback.md`, `backup-restore.md`, `secrets-rotation.md`,
`github-setup.md`, `git-workflow.md`, `incident-response.md` (severity levels, first steps, communication),
and `docs/decisions/0010-deployment.md`. Update `README.md` (production section, security summary, release
process, roadmap marks Phase 5 complete). Every runbook can be followed by someone new using only the
document.

## 10. Testing and verification

- Automated: security-header tests against the running proxy, cookie flag tests, rate limit tests, preflight
  refusal tests (mock provider, weak secrets, default credentials), attribution-check tests with sample
  passing and failing messages and files, backup and restore drill test, migration upgrade test from the
  previous release schema, container hardening assertions (non-root, no published DB port).
- E2E on the production-like staging compose: login, dashboard, indicators, alert creation and delivery to a
  test channel, brief page, forecast job (if the service is enabled), disclaimer page, degraded states.
- Post-deploy smoke test script used by both CI and the deployment procedure.
- Manual checklist for the owner in `docs/operations/launch-checklist.md`: DNS, TLS validity, headers scan,
  admin login, backups configured and restored once, monitoring and operational alerts firing on a test,
  secrets rotated from any development values, repository settings applied, first-push audit passed.

## 11. Final deliverable

The owner must be able to: deploy a tagged release with a documented procedure; roll back; restore from a
backup and prove it with the drill; see security headers and TLS working; run `make audit` and get a clean
report before pushing; push a clean repository to GitHub whose history and files contain no secrets and no AI
attribution; publish a release with a changelog; follow runbooks for common incidents.

Verify and report: builds, lint, type-check, all tests, contract check; migrations (fresh, upgrade);
`make audit`, `make preflight`, `make restore-drill` all pass; attribution scan clean on files, and on commit
message samples; no secrets in the tree or history; production refuses unsafe configuration; database and
forecasting ports not published; image scans and dependency audits within thresholds; Phases 1 to 4 tests
still pass; list of changes to existing files; list of open decisions left blank in the DEPLOYMENT DECISIONS.

Build incrementally and confirm each step:

1. repository hygiene: ignore rules, audit tooling, secret scan, `attribution-check`, commit-msg hook
2. production compose, reverse proxy, TLS configuration, preflight, admin bootstrap
3. application security hardening (headers, cookies, rate limits, audit log) with tests
4. container and supply-chain hardening, CI scans, pinned actions
5. backup, restore drill, retention, backup monitoring
6. CI/CD: release workflow, deployment workflow, changelog, migration policy
7. legal and disclaimer pages, data-source and privacy documentation
8. staging configuration, E2E and smoke tests, upgrade-migration test
9. runbooks, threat model, host hardening notes, launch checklist
10. final verification and report

Prioritize safety, recoverability, and a clean public repository over speed.