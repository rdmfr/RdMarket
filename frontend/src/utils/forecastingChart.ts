import type Highcharts from 'highcharts';
import type { ForecastLatest } from '@/schemas/forecasting';

export type ForecastChartSeries = Highcharts.SeriesOptionsType;

export interface ForecastChartLabels {
  history: string;
  forecast: string;
  interval80: string;
  interval95: string;
}

export function toForecastChartSeries(
  data: ForecastLatest,
  labels: ForecastChartLabels
): ForecastChartSeries[] {
  const history: Highcharts.PointOptionsType[] = data.history.map((point) => [
    Date.parse(point.timestamp),
    point.rate,
  ]);
  const forecast: Highcharts.PointOptionsType[] = data.forecasts.map((point) => [
    Date.parse(point.target_timestamp),
    point.point,
  ]);
  const range80 = data.forecasts.map((point) => [
    Date.parse(point.target_timestamp),
    point.lower_80,
    point.upper_80,
  ] as [number, number, number]);
  const range95 = data.forecasts.map((point) => [
    Date.parse(point.target_timestamp),
    point.lower_95,
    point.upper_95,
  ] as [number, number, number]);

  return [
    {
      type: 'arearange',
      name: labels.interval95,
      data: range95,
      color: 'var(--series-4)',
      fillOpacity: 0.12,
      lineWidth: 0,
      zIndex: 0,
      marker: { enabled: false },
      enableMouseTracking: true,
    },
    {
      type: 'arearange',
      name: labels.interval80,
      data: range80,
      color: 'var(--series-3)',
      fillOpacity: 0.2,
      lineWidth: 0,
      zIndex: 1,
      marker: { enabled: false },
    },
    {
      type: 'line',
      name: labels.history,
      data: history,
      color: 'var(--series-1)',
      lineWidth: 1.5,
      zIndex: 2,
      marker: { enabled: false },
    },
    {
      type: 'line',
      name: labels.forecast,
      data: forecast,
      color: 'var(--series-2)',
      lineWidth: 1.5,
      dashStyle: 'ShortDash',
      zIndex: 3,
      marker: { enabled: false },
    },
  ] as ForecastChartSeries[];
}

export function forecastOriginTimestamp(data: ForecastLatest): number | null {
  const timestamp = Date.parse(data.run.forecast_origin);
  return Number.isNaN(timestamp) ? null : timestamp;
}
