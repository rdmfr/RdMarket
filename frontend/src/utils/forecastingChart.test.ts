import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import type Highcharts from 'highcharts';
import { toForecastChartSeries, forecastOriginTimestamp } from './forecastingChart';
import type { ForecastLatest } from '@/schemas/forecasting';

const sample: ForecastLatest = {
  run: {
    id: 'run-1',
    model_name: 'model',
    model_version: '1',
    forecast_origin: '2026-01-02T00:00:00Z',
    horizon: 1,
    training_start: '2025-01-01T00:00:00Z',
    training_end: '2026-01-02T00:00:00Z',
    training_rows: 2,
    missing_observations: 0,
    data_fingerprint: 'fingerprint',
    generated_at: '2026-01-02T00:01:00Z',
    interval_levels: [0.8, 0.95],
  },
  forecasts: [{
    target_timestamp: '2026-01-03T00:00:00Z',
    step: 1,
    point: 16000,
    lower_80: 15900,
    upper_80: 16100,
    lower_95: 15800,
    upper_95: 16200,
  }],
  history: [{ timestamp: '2026-01-02T00:00:00Z', rate: 15950 }],
  baseline_comparison: { skill_score_vs_naive: null, beats_naive: false },
};

describe('forecast chart transformations', () => {
  it('maps observed history, point forecasts and interval bounds without changing values', () => {
    const series = toForecastChartSeries(sample, {
      history: 'history',
      forecast: 'forecast',
      interval80: 'interval80',
      interval95: 'interval95',
    });
    assert.equal(series.length, 4);
    assert.deepEqual((series[0] as Highcharts.SeriesArearangeOptions).data, [[
      Date.parse('2026-01-03T00:00:00Z'), 15800, 16200,
    ]]);
    assert.deepEqual((series[1] as Highcharts.SeriesArearangeOptions).data, [[
      Date.parse('2026-01-03T00:00:00Z'), 15900, 16100,
    ]]);
    assert.deepEqual((series[2] as Highcharts.SeriesLineOptions).data, [[
      Date.parse('2026-01-02T00:00:00Z'), 15950,
    ]]);
    assert.deepEqual((series[3] as Highcharts.SeriesLineOptions).data, [[
      Date.parse('2026-01-03T00:00:00Z'), 16000,
    ]]);
  });

  it('returns the forecast origin for the chart marker', () => {
    assert.equal(forecastOriginTimestamp(sample), Date.parse(sample.run.forecast_origin));
    assert.equal(forecastOriginTimestamp({
      ...sample,
      run: { ...sample.run, forecast_origin: 'invalid' },
    }), null);
  });
});
