# ADR 0004: Economic Data Source Scope

## Status

Accepted for the initial Phase 3 foundation; Indonesian sources remain open.

## Decision

- Enable BLS Public Data API for the CPI index (`CUSR0000SA0`) and the
  seasonally adjusted unemployment rate (`LNS14000000`). Cite BLS and the
  retrieval date; preserve unknown release timestamps as null.
- Use 31 days after the period end as the configured point-in-time fallback
  when a release timestamp is unknown. This is a conservative analysis lag,
  not a reported publication time.
- Do not persist FRED output unless the owner confirms that applicable terms
  and each underlying series license permit it, or written permission is
  obtained.
- Do not activate Indonesia or global series until their official mappings
  and persistent storage/display terms are verified.

## Rationale

BLS documents its API and states that published BLS materials are public
domain, subject to source citation. BLS also states it cannot vouch for
downstream analysis. FRED terms explicitly restrict storage and archiving of
FRED content, so API availability alone is insufficient for this application's
revision-preserving database.

## Open Decisions

- Confirm the official Indonesia series sources, API mappings, publication
  timestamps, and terms for storage and display.
- Confirm whether to request written permission or select licensed sources for
  FRED-mapped policy rates, CPI, Treasury yields, and global indicators.
- Decide whether the product needs an explicit BLS API terms notice in its UI
  or site terms before enabling automated retrieval.
