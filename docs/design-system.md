# RdMarket Intelligence — Design Constitution & Token System

Reference aesthetic: Bloomberg Terminal / TradingView / Koyfin / Grafana.
Designed specifically as an ultra-dense, low-fatigue financial analytics workspace that a professional currency trader can keep open on a secondary monitor all day.

---

## 1. Design Tokens (CSS Variables)

Defined centrally in `src/tokens.css` and mapped into Tailwind v4 `@theme` configuration. No hardcoded ad-hoc hex codes are permitted in Vue components.

### Dark Theme (Default)
| Token | Hex | Semantic Role |
| :--- | :--- | :--- |
| `--bg-base` | `#0B0E11` | Application background |
| `--bg-panel` | `#12161B` | Module panel surfaces |
| `--bg-raised` | `#181D23` | Table headers, row hover states, inputs |
| `--border` | `#232A32` | 1px module borders & grid dividers |
| `--border-strong` | `#2F3842` | Focused borders, active separator lines |
| `--text-primary` | `#E6EAED` | Primary figures, rates, headers |
| `--text-secondary`| `#9AA5B1` | Metric labels, secondary descriptions |
| `--text-muted` | `#667180` | Category labels, footnotes, timestamps |
| `--up` | `#2EBD85` | Price increase / positive delta |
| `--down` | `#F6465D` | Price decrease / negative delta |
| `--neutral` | `#9AA5B1` | Flat delta / neutral conditions |
| `--accent` | `#D9A441` | Single amber accent (selected tab, active indicator) |
| `--series-1` | `#E6EAED` | Actual USD/IDR exchange rate line |
| `--series-2` | `#D9A441` | SMA 7 series line |
| `--series-3` | `#4FA3D9` | SMA 30 series line |
| `--series-4` | `#B48EE0` | SMA 90 series line |

### Light Theme (Warm Paper)
| Token | Hex | Semantic Role |
| :--- | :--- | :--- |
| `--bg-base` | `#F6F5F2` | Warm off-white canvas |
| `--bg-panel` | `#FFFFFF` | Solid white cards |
| `--bg-raised` | `#EDECE8` | Table headers, inputs |
| `--border` | `#DDDAD3` | Hairline dividers |
| `--border-strong` | `#CBC7BD` | Active structural lines |
| `--text-primary` | `#1A1D20` | Crisp dark slate type |
| `--text-secondary`| `#5A6472` | Secondary labels |
| `--text-muted` | `#8B95A1` | Metadata & units |
| `--up` | `#1F9D6D` | Controlled emerald green |
| `--down` | `#D9384E` | Controlled crimson red |
| `--neutral` | `#5A6472` | Neutral state |
| `--accent` | `#B88228` | Deep warm amber |

---

## 2. Typography & Numerals

1. **Interface Font Stack**: `"IBM Plex Sans"`, `-apple-system`, `BlinkMacSystemFont`, `"Segoe UI"`, `sans-serif`. (Zero reliance on Inter/Poppins).
2. **Numeric Font Stack**: `"JetBrains Mono"`, `"IBM Plex Mono"`, `monospace`.
3. **Tabular Numerals**: Every rate, percentage, delta, timestamp, axis label, and table cell enforces `font-variant-numeric: tabular-nums` (`font-mono`) to prevent layout shift and align decimal columns vertically.
4. **Scale Discipline**:
   - Hero rate: `32px–40px` (only the primary spot quote is permitted this size).
   - Standard labels: `13px`.
   - Table cells: `12px`.
   - Uppercase metadata / kickers: `10px–11px` with `tracking-[0.04em]` in `--text-muted`.
5. **Weights**: Strictly `400` and `500`, maximum `600` for bold figures. No `700/800` display weights.

---

## 3. Spatial Math, Depth & Surface Rules

- **Zero Card-Lift & Zero Drop-Shadows**: Flat edge-to-edge modular grid. Elevation is created strictly by background step (`--bg-base` → `--bg-panel` → `--bg-raised`) and 1px border lines.
- **Border Radius**: Maximum `2px` (`rounded-xs`) on panels, inputs, and segmented controls; `4px` maximum anywhere.
- **Panel Header Strip**: Every panel features a compact `30px` header strip with `11px` uppercase label on the left, optional controls/meta on the right, and a hairline bottom divider.
- **Padding**: Compact `8px–12px`. No airy 24px+ SaaS landing page padding.
- **Missing Data**: Displayed exclusively as an em dash `—` with tooltip `"Not enough data"`. Never blank, never 0, never arbitrary interpolation.

---

## 4. Charting Discipline (Highcharts)

- Centered in `frontend/src/components/charts/theme.ts`.
- Transparent background; right-side Y-axis (financial standard).
- Actual spot rate rendered as a 1.5px solid line; SMA lines rendered as 1px dashed lines.
- No gradient area fills; no default markers (hover-only).
- Missing data rendered as real gaps (`connectNulls: false`).
- `animation: false` by default for instant rendering and reduced-motion compliance.

---

## 5. Anti-Slop Compliance Matrix

- [x] Zero gradient backgrounds, text, buttons, or chart fills
- [x] Zero box-shadows or card elevation lift on hover
- [x] Dense key-value table layout instead of repeated big-number stat boxes
- [x] No emojis, illustrations, or marketing copy
- [x] All numbers paired with explicit sign (`+` / `-`) and glyph (`▲` / `▼`) alongside semantic color
- [x] IBM Plex Sans + JetBrains Mono tabular-nums throughout
- [x] Amber (`#D9A441`) as the sole restrained accent color
- [x] Fully prepared for Phase 2 forecast series (`--series-forecast`, confidence interval bands)
