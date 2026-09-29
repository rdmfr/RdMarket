# Branching and CI

## Branch model

- `master` is the protected production line. Changes enter through pull requests.
- `develop` is the protected integration line for completed feature work.
- `feature/*` branches are short-lived work branches such as `feature/dashboard` or `feature/market-api`.

Normal flow: branch `feature/*` from `develop`, commit and push, then open a pull request into `develop`. After integration checks pass, promote `develop` to `master` with a separate pull request. Do not push feature work directly to either protected branch.

The owner creates branches and applies GitHub protection settings. The assistant only creates or pushes branches when the owner explicitly authorizes it.

## CI/CD stages

| Stage | What | Where it is built |
| :-- | :-- | :-- |
| 1 CI | Tests and build on pushes to `master`, `develop`, and `feature/*`, and pull requests into `develop` or `master` | `backend.yml`, `frontend.yml`, `forecasting.yml` |
| 2 Container | On push to `master`, build and publish images to GHCR; pull requests build without publishing | `docker.yml` (frontend, backend, forecasting) |
| 3 Staging | Staging deployment source and trigger | Phase 5, owner decision |
| 4 Production | Deploy a specific release tag from `master` with manual approval | Phase 5 |
| 5 Advanced | Integration tests, security scans, and health checks | Phase 5 |

Stages 3 through 5 do not exist yet. No deployment is configured.

Require status checks named `backend`, `frontend`, and `forecasting`; confirm their exact names after the first pull request run.

## GitHub branch protection checklist

Apply these settings to both `develop` and `master`:

- Require a pull request before merging.
- Require the `backend`, `frontend`, and `forecasting` status checks to pass.
- Require conversation resolution before merging.
- Block direct pushes.

`docker.yml` publishes only on pushes to `master`. Pull requests into either protected branch build images without publishing them.

## GHCR

The first push creates a private package in GitHub Container Registry. The owner must set the desired package visibility and link the package to the repository. Image names must be lowercase.

Images use commit-specific `sha-` tags and the moving `edge` tag on `master`. `latest` is reserved for release tags created by the Phase 5 release workflow. Production compose files must not use `latest` or `edge`.

## Current limitations

The E2E workflow starts the development Compose stack and runs the Playwright suite. The frontend has an existing peer conflict between Vite 8 and esbuild 0.25; the npm lockfile was created using legacy peer resolution, but dependency cleanup is outside this layout task.
