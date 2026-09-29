# Existing UI Inventory & Design Audit

## Scope
The Vue UI in `frontend/` was created prior to Phase 0. As established in `CLAUDE.md`, the visual language and existing layout are strictly preserved. This audit inventories the existing components, records additions created in Phase 0, and notes integrity fixes applied.

## Inventory of Existing Assets

### 1. Views
- `frontend/src/views/DashboardView.vue`: Primary USD/IDR dashboard (summary stats, chart, market condition indicators).
- `frontend/src/views/MarketDataView.vue`: Detailed market quotes and statistics table.
- `frontend/src/views/IndicatorsView.vue`: Technical indicators (SMA 7/30/90, return, rolling volatility).
- `frontend/src/views/DataSourcesView.vue`: Feed connection status and provider capabilities.
- `frontend/src/views/SettingsView.vue`: User preferences, auto-refresh interval, theme selection.

### 2. Layout Shell
- `frontend/src/layouts/AppLayout.vue`: Master container with top navigation, sidebar rail, and main scroll area.
- `frontend/src/layouts/TopBar.vue`: Instrument ticker, connection status, timestamp, theme toggle.
- `frontend/src/layouts/SidebarRail.vue`: Compact icon rail navigation.

### 3. Shared Primitives (Evaluated & Mapped)
| Required Phase 0 Primitive | Existing / Status | File Location | Notes |
|---|---|---|---|
| `Panel` | Existing | `components/common/Panel.vue` | Flat border, compact header strip, footer slot |
| `StatRow` | Existing | `components/common/StatRow.vue` | Tabular key/value metric row |
| `ValueChange` | Existing | `components/common/ValueChange.vue` | Signed (+/-), glyph arrow, color |
| `StatusBadge` | Existing | `components/common/StatusBadge.vue` | State indicator with dot marker |
| `SegmentedControl` | Existing | `components/common/SegmentedControl.vue` | Dense pill tab toggle (1D, 7D, 1M, etc.) |
| `ChipToggle` | Existing | `components/common/ChipToggle.vue` | Indicator active/inactive toggle chip |
| `SkeletonBlock` | Existing | `components/common/SkeletonBlock.vue` | Layout-matched placeholder shimmer |
| `ErrorState` | Existing | `components/common/ErrorState.vue` | Terse error message with Retry button |
| `EmptyState` | Added in Phase 0 | `components/common/EmptyState.vue` | Flat border, text and one action |
| `SimulatedDataBanner` | Added in Phase 0 | `components/common/SimulatedDataBanner.vue` | Persistent, non-dismissible mock indicator |

### 4. Charts & Highcharts Integration
- `components/charts/theme.ts`: Centralized dark/light Highcharts theme module using design tokens.
- `components/charts/BaseChart.vue`: Added in Phase 0 as a pure props-in chart wrapper (no data fetching).
- `components/charts/UsdIdrChart.vue`: Existing chart implementation.

### 5. Integrity & Anti-Slop Fixes Applied in Phase 0
- **Dependency Cleanup**: Removed unused React packages (`react`, `react-dom`, `lucide-react`, `@vitejs/plugin-react`), Express server leftovers, and `@google/genai` from `package.json`.
- **Secrets & Attribution**: Confirmed no hard-coded secrets, LLM API calls, or external AI tooling references exist in the frontend code bundle.
- **Contract Enforcement**: Added Ky v2 centralized client with CSRF token injection and Zod schema validation in `services/api.ts`.
- **Time Module**: Implemented pure Intl formatters in `utils/formatters.ts` and `utils/time.ts` with explicit Asia/Jakarta (WIB) timezone display.
