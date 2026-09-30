import assert from 'node:assert/strict';
import test from 'node:test';
import { EconomicIndicatorSchema } from './economic';

const indicator = {
  code: 'US_CPI_INDEX_SA',
  name: 'Consumer Price Index, all items, seasonally adjusted',
  country: 'United States',
  category: 'inflation',
  unit: 'index, 1982-84=100',
  frequency: 'monthly',
  seasonal_adjustment: true,
  source_provider: 'bls_api',
  source_series_id: 'CUSR0000SA0',
  source_url: 'https://www.bls.gov/developers/',
  license_note: 'Public domain; cite BLS and retrieval date.',
  description: 'The index measures consumer prices.',
  change_mode: 'level',
  publication_lag_days: 31,
  latest_value: null,
  previous_value: null,
  change: null,
  reference_date: null,
  period_end: null,
  release_timestamp: null,
  retrieved_at: null,
  stale: null,
  staleness_days: null,
  trend: 'not_enough_data',
  sparkline: [],
};

test('economic indicator schema keeps absent observations nullable', () => {
  const parsed = EconomicIndicatorSchema.safeParse(indicator);
  assert.equal(parsed.success, true);
});

test('economic indicator schema rejects numeric coercion and unknown trend labels', () => {
  assert.equal(EconomicIndicatorSchema.safeParse({ ...indicator, latest_value: 319.8 }).success, false);
  assert.equal(EconomicIndicatorSchema.safeParse({ ...indicator, trend: 'upward_guess' }).success, false);
});