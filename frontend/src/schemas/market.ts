import { z } from 'zod';

export const ApiErrorSchema = z.object({
  code: z.string(),
  message: z.string(),
});

export const CurrentRateSchema = z.object({
  currency_pair: z.string(),
  current_rate: z.number(),
  previous_close: z.number(),
  daily_change: z.number(),
  daily_change_percent: z.number(),
  timestamp: z.string(),
  source: z.string(),
  is_stale: z.boolean().default(false),
});

export const HistoricalPointSchema = z.object({
  timestamp: z.number(),
  rate: z.number().nullable(),
});

export const HistoricalRateSchema = z.object({
  currency_pair: z.string(),
  range: z.string(),
  points: z.array(HistoricalPointSchema),
  total_points: z.number(),
  source_point_count: z.number().optional(),
  missing_count: z.number().default(0),
  resolution: z.string().optional(),
  aggregation_method: z.string().optional(),
});

export const StatisticsSchema = z.object({
  currency_pair: z.string(),
  current_rate: z.number().nullable(),
  previous_close: z.number().nullable(),
  daily_change: z.number().nullable(),
  daily_change_percent: z.number().nullable(),
  weekly_change_percent: z.number().nullable(),
  monthly_change_percent: z.number().nullable(),
  high_52_week: z.number().nullable(),
  low_52_week: z.number().nullable(),
  average_rate: z.number().nullable(),
  minimum_rate: z.number().nullable(),
  maximum_rate: z.number().nullable(),
  observation_count: z.number(),
  has_sufficient_data: z.boolean(),
});

export const MarketConditionSchema = z.object({
  short_term_condition: z.string(),
  trend_30_day: z.string(),
  volatility_level: z.string(),
  recent_return_7d: z.number(),
  return_30d: z.number(),
  observed_volatility: z.number(),
  disclaimer: z.string(),
});

export const IndicatorsSchema = z.object({
  currency_pair: z.string(),
  range: z.string(),
  sma_7: z.array(HistoricalPointSchema),
  sma_30: z.array(HistoricalPointSchema),
  sma_90: z.array(HistoricalPointSchema),
  daily_return: z.number().nullable(),
  weekly_return: z.number().nullable(),
  rolling_volatility_30d: z.number().nullable(),
  market_condition: MarketConditionSchema,
});

export const DataSourceSchema = z.object({
  provider: z.string(),
  instrument: z.string(),
  status: z.string(),
  last_successful_update: z.string().nullable(),
  data_frequency: z.string(),
  number_of_observations: z.number(),
  base_currency: z.string(),
  target_currency: z.string(),
  capabilities: z.object({
    supports_intraday: z.boolean(),
    supports_history: z.boolean(),
    max_history_days: z.number(),
    granularity: z.string(),
    rate_limit: z.string(),
    requires_api_key: z.boolean(),
    license_note: z.string(),
  }),
});

export function createEnvelopeSchema<T extends z.ZodTypeAny>(dataSchema: T) {
  return z.object({
    data: dataSchema,
    meta: z.record(z.string(), z.unknown()).default({}),
    error: ApiErrorSchema.nullable(),
  });
}
