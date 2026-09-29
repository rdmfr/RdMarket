# ADR 0003: OpenAPI 3.1 as the Single Source of Truth

## Status
Accepted

## Context
In a decoupled monorepo, API drift between the Go backend and Vue/TypeScript frontend is a primary source of runtime bugs. RdMarket requires contract-first development.

## Decisions

### 1. Contract Specification
- `docs/openapi.yaml` (OpenAPI 3.1.0) is the canonical source of truth for all HTTP API endpoints.
- New endpoints, parameters, or data fields must first be declared in `docs/openapi.yaml` before implementation.

### 2. Standard Response Envelope
All API endpoints return a uniform JSON envelope:
```json
{
  "data": {},
  "meta": {
    "simulated": false
  },
  "error": null
}
```
In error conditions:
```json
{
  "data": null,
  "meta": {},
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Authentication required"
  }
}
```

### 3. Error Code Registry Synchronization
- Go: `backend/internal/errors/codes.go` defines error constants.
- TypeScript: `frontend/src/types/errors.ts` mirrors the error constants.
- Error codes must match exactly.

### 4. Automated Contract Verification
- Backend: `backend/internal/handlers/contract_test.go` automatically reads `docs/openapi.yaml` and verifies that live endpoints return status codes and response bodies conforming to the spec.
- Frontend: `validateResponse()` in `frontend/src/services/api.ts` parses incoming payloads with Zod schemas.
- `make contract-check` runs the contract suite locally and in CI.
