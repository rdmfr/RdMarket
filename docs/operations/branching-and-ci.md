# Branching and CI

## Branch model

- `main` is the protected production line. Changes enter through pull requests.
- `develop` is the protected development line. Changes enter through pull requests.
- `feature/*` branches are short-lived work branches such as `feature/dashboard` or `feature/market-api`.

Normal flow: create a feature branch from `develop`, commit and push, open a pull request into `develop`, wait for checks, resolve conversations, and merge. When stable, open a pull request from `develop` into `main`.

The owner creates `develop` and feature branches. This repository does not create branches automatically.

## CI/CD stages

| Stage | What | Where it is built |
| :-- | :-- | :-- |
| 1 CI | Tests and build on push and pull request for `main` and `develop` | `backend.yml`, `frontend.yml` |
| 2 Container | On push to `main`, build images and push to GHCR; no deployment | `docker.yml` |
| 3 Staging | Deploy `develop` to a staging server | Phase 5, owner decision |
| 4 Production | Deploy `main` to production with manual approval | Phase 5 |
| 5 Advanced | Integration tests, security scans, and health checks | Phase 5 |

Stages 3 through 5 do not exist yet. No deployment is configured.

Require status checks named `backend` and `frontend`; confirm their exact names after the first pull request run.

## GitHub branch protection checklist

Apply these settings to both `main` and `develop`:

- Require a pull request before merging.
- Require the `backend` and `frontend` status checks to pass.
- Require conversation resolution before merging.
- Block direct pushes.

`docker.yml` runs only on pushes to `main` because protected `main` should receive only changes whose CI checks have passed.

## GHCR

The first push creates a private package in GitHub Container Registry. The owner must set the desired package visibility and link the package to the repository. Image names must be lowercase.

Images use commit-specific `sha-` tags and the moving `edge` tag. `latest` is reserved for release tags created by the Phase 5 release workflow. Production compose files must not use `latest` or `edge`.

## Current limitations

The backend and frontend Dockerfiles are not present yet, so `docker.yml` cannot build images until Phase 0 adds them. The frontend has an existing peer conflict between Vite 8 and esbuild 0.25; the npm lockfile was generated with legacy peer resolution, but dependency cleanup is outside this layout task.
