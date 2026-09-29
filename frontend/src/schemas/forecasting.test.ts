import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  BacktestLeaderboardSchema,
  ForecastModelSchema,
} from './forecasting';

describe('forecasting response schemas', () => {
  it('retains experimental model availability and its reason', () => {
    const model = ForecastModelSchema.parse({
      name: 'lstm',
      version: 'keras-lstm-v1',
      stability: 'experimental',
      label: 'experimental',
      available: false,
      unavailable_reason: 'Not installed: tensorflow',
      min_history: 750,
      minimum_observations: 750,
      description: 'LSTM on log-returns',
    });
    assert.equal(model.available, false);
    assert.equal(model.unavailable_reason, 'Not installed: tensorflow');
  });

  it('distinguishes no evaluations from a measured baseline comparison', () => {
    const result = BacktestLeaderboardSchema.parse({
      horizon: 5,
      items: [],
      no_model_beats_naive: false,
      has_evaluations: false,
    });
    assert.equal(result.has_evaluations, false);
    assert.equal(result.no_model_beats_naive, false);
  });
});
