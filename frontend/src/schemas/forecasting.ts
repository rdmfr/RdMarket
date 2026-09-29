import { z } from 'zod';
import { ApiErrorSchema, createEnvelopeSchema } from './market';

export const ForecastModelSchema = z.object({
  name: z.string(),
  version: z.string(),
  stability: z.enum(['stable', 'experimental']),
  label: z.string(),
  available: z.boolean(),
  unavailable_reason: z.string().nullable(),
  min_history: z.number(),
  minimum_observations: z.number(),
  description: z.string(),
});

export const ForecastJobSchema = z.object({
  id: z.string(),
  job_type: z.enum(['forecast', 'backtest']),
  status: z.string(),
  progress_done: z.number(),
  progress_total: z.number(),
  error_code: z.string().nullable(),
  error_message: z.string().nullable(),
  created_at: z.string(),
});

export const ForecastJobDetailSchema = ForecastJobSchema.extend({
  started_at: z.string().nullable(),
  finished_at: z.string().nullable(),
});

export const ForecastRunSchema = z.object({
  id: z.string(),
  model_name: z.string(),
  model_version: z.string(),
  forecast_origin: z.string(),
  horizon: z.number(),
  training_start: z.string(),
  training_end: z.string(),
  training_rows: z.number(),
  missing_observations: z.number(),
  data_fingerprint: z.string(),
  generated_at: z.string(),
  interval_levels: z.array(z.number()),
});

export const ForecastPointSchema = z.object({
  target_timestamp: z.string(),
  step: z.number(),
  point: z.number(),
  lower_80: z.number(),
  upper_80: z.number(),
  lower_95: z.number(),
  upper_95: z.number(),
});

export const ForecastHistoryPointSchema = z.object({
  timestamp: z.string(),
  rate: z.number(),
});

export const ForecastLatestSchema = z.object({
  run: ForecastRunSchema,
  forecasts: z.array(ForecastPointSchema),
  history: z.array(ForecastHistoryPointSchema),
  baseline_comparison: z.object({
    skill_score_vs_naive: z.number().nullable(),
    beats_naive: z.boolean(),
  }),
});

export const BacktestLeaderboardItemSchema = z.object({
  model_name: z.string(),
  horizon: z.number(),
  mae: z.number().nullable(),
  rmse: z.number().nullable(),
  mape: z.number().nullable(),
  mase: z.number().nullable(),
  directional_accuracy: z.number().nullable(),
  interval_coverage: z.number().nullable(),
  skill_score_vs_naive: z.number().nullable(),
  is_baseline: z.boolean(),
});

export const BacktestLeaderboardSchema = z.object({
  horizon: z.number(),
  items: z.array(BacktestLeaderboardItemSchema),
  no_model_beats_naive: z.boolean(),
  has_evaluations: z.boolean(),
});

export const ForecastModelsResponseSchema = createEnvelopeSchema(z.array(ForecastModelSchema));
export const ForecastJobResponseSchema = createEnvelopeSchema(ForecastJobSchema);
export const ForecastJobDetailResponseSchema = createEnvelopeSchema(ForecastJobDetailSchema);
export const ForecastJobsResponseSchema = createEnvelopeSchema(z.array(ForecastJobSchema));
export const ForecastLatestResponseSchema = createEnvelopeSchema(ForecastLatestSchema);
export const BacktestLeaderboardResponseSchema = createEnvelopeSchema(BacktestLeaderboardSchema);

export const CreateForecastJobRequestSchema = z.object({
  job_type: z.enum(['forecast', 'backtest']),
  models: z.array(z.string()),
  horizon: z.number().int().positive(),
  params: z.record(z.string(), z.unknown()),
  force: z.boolean(),
});

export const ForecastEnvelopeErrorSchema = ApiErrorSchema.nullable();

export type ForecastModel = z.infer<typeof ForecastModelSchema>;
export type ForecastJob = z.infer<typeof ForecastJobSchema>;
export type ForecastJobDetail = z.infer<typeof ForecastJobDetailSchema>;
export type ForecastLatest = z.infer<typeof ForecastLatestSchema>;
export type ForecastPoint = z.infer<typeof ForecastPointSchema>;
export type BacktestLeaderboard = z.infer<typeof BacktestLeaderboardSchema>;
export type BacktestLeaderboardItem = z.infer<typeof BacktestLeaderboardItemSchema>;
export type CreateForecastJobRequest = z.infer<typeof CreateForecastJobRequestSchema>;
