import { z } from 'zod';

export const EconomicIndicatorSchema = z.object({
  code: z.string(),
  name: z.string(),
  country: z.string(),
  category: z.string(),
  unit: z.string(),
  frequency: z.enum(['daily', 'weekly', 'monthly', 'quarterly']),
  seasonal_adjustment: z.boolean(),
  source_provider: z.string(),
  source_series_id: z.string(),
  source_url: z.string(),
  license_note: z.string(),
  description: z.string(),
  change_mode: z.enum(['level', 'percent', 'bps']),
  publication_lag_days: z.number().int(),
  latest_value: z.string().nullable(),
  previous_value: z.string().nullable(),
  change: z.string().nullable(),
  reference_date: z.string().nullable(),
  period_end: z.string().nullable(),
  release_timestamp: z.string().nullable(),
  retrieved_at: z.string().nullable(),
  stale: z.boolean().nullable(),
  staleness_days: z.number().int().nullable(),
  trend: z.enum(['rising', 'falling', 'flat', 'not_enough_data']),
  sparkline: z.array(z.string()),
});

export const EconomicObservationSchema = z.object({
  reference_date: z.string(),
  period_end: z.string().nullable(),
  value: z.string(),
  release_timestamp: z.string().nullable(),
  retrieved_at: z.string(),
  revision: z.number().int().nonnegative(),
  is_latest: z.boolean(),
});

export type EconomicIndicator = z.infer<typeof EconomicIndicatorSchema>;