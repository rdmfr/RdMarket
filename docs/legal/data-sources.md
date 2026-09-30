# Economic Data Sources

Only series whose source terms have been reviewed are enabled for automated
retrieval. Source terms can change; verify them before expanding this catalog or
using its values in a public or commercial deployment.

## Enabled

| Catalog code | Source | Series | Terms and handling |
|---|---|---|---|
| `US_CPI_INDEX_SA` | U.S. Bureau of Labor Statistics (BLS) Public Data API | `CUSR0000SA0` | BLS states its published material is public domain and requests citation. Cite BLS and retrieval date. BLS says it cannot vouch for downstream analysis. |
| `US_UNEMPLOYMENT_RATE_SA` | BLS Public Data API | `LNS14000000` | Same BLS public-domain and citation terms. |

The BLS API is documented at <https://www.bls.gov/developers/> and its API
terms are at <https://www.bls.gov/developers/termsOfService.htm>. The BLS
copyright notice is at <https://www.bls.gov/bls/linksite.htm>. Automated
requests use the official JSON API, not page scraping. The BLS adapter skips
annual-average `M13` rows when importing monthly series. BLS does not return
public release timestamps in the observations response, so `release_timestamp`
is stored as null. The catalog's 31-day publication lag is the configured
point-in-time fallback, not a claimed release time.

## Not Enabled

- FRED API series are not persisted. FRED's current terms prohibit storing,
  caching, archiving, or incorporating FRED content into another database, and
  the API requires an attribution notice and compliance with each source
  series' own restrictions. A suitable written permission or alternate
  license is required before adding a FRED-backed persistent feed.
- Bank Indonesia and Indonesian statistical series are not enabled because a
  stable official API and terms permitting this platform's persistent storage
  and display have not yet been verified. No values should be imported until
  the owner confirms the source and its rights.
- U.S. policy-rate and Treasury-yield series, global dollar indices, oil prices,
  and volatility indices are not enabled until an official series mapping and
  its applicable reuse terms are verified.
- CPI year-over-year inflation is not currently derived. The enabled CPI
  series is the underlying seasonally adjusted index; presenting a derived
  rate requires a tested derivation and explicit API metadata.

FRED terms: <https://fred.stlouisfed.org/legal/>. FRED API observations
documentation: <https://fred.stlouisfed.org/docs/api/fred/series_observations.html>.

## Manual CSV

The economic CSV parser accepts exactly these columns (the two final columns
are optional):

```csv
reference_date,value,release_timestamp,period_end
2026-09-01,4.25,2026-09-15T08:00:00Z,2026-09-30
```

`reference_date` and `period_end` use `YYYY-MM-DD`; a known release time uses
RFC3339 UTC (otherwise leave it empty). Values are plain decimal strings, not
scientific notation, and must fit `NUMERIC(24,10)`. The importer rejects
duplicate reference dates in one file and returns a reason for each invalid
row. The importing operator must ensure that their selected source permits
storage and display. Uploaded rows are not automatically treated as released
at the time of import.
