# ADR 0001: Data Source Decision & Provider Capability Model

## Status
Proposed / Open Decisions Pending Owner Specification

## Context
Phase 0 establishes the provider abstraction for the USD/IDR financial monitoring platform. The owner prompt provides a template for specifying the primary provider, access tier, quotas, and licensing. Where fields were left unspecified in `docs/prompts/Phase0.md`, this record marks them as "unknown" without inventing assumptions.

## Decision Summary

| Decision Attribute | Configured / Recorded Value | Status |
|-------------------|-----------------------------|--------|
| Primary USD/IDR provider | Unknown (Frankfurter / ECB fallback configured for dev) | Open Decision |
| Access type | Unknown (Free tier assumption in dev) | Open Decision |
| Historical depth available | Unknown (Mock provider provides 10-year synthetic) | Open Decision |
| Intraday data available | Unknown (Capability flag set to false by default) | Open Decision |
| Rate limit / quota | Unknown (Configurable per-provider in Go) | Open Decision |
| Backfill method | Unknown (CSV import provider + Mock backfill supported) | Open Decision |
| License / display restrictions | Unknown (To be confirmed before Phase 1 external feed) | Open Decision |
| Secondary provider for cross-check | None configured | Optional / Open |
| UI Language(s) | id, en (default: en) | Open Decision |

## Provider Capability Model
The `ExchangeRateProvider` Go interface exposes:
```go
type Capabilities struct {
    SupportsIntraday bool   `json:"supports_intraday"`
    SupportsHistory  bool   `json:"supports_history"`
    MaxHistoryDays   int    `json:"max_history_days"`
    Granularity      string `json:"granularity"`
    RateLimit        string `json:"rate_limit"`
    RequiresAPIKey   bool   `json:"requires_api_key"`
    LicenseNote      string `json:"license_note"`
}
```

The frontend accesses this via `/api/v1/data-sources` so UI range pickers dynamically adjust rather than showing blank screens or fabricating points.

## Open Questions for Owner
1. Which official reference rate or commercial provider will be contracted for Phase 1 (e.g. Bank Indonesia JISDOR, central bank feed, or commercial aggregator)?
2. What is the contractual rate limit and quota for production?
3. What are the display and redistribution terms of the licensed feed?
