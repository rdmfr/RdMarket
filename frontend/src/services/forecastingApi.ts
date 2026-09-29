import { apiClient, validateResponse } from './api';
import {
  BacktestLeaderboardResponseSchema,
  CreateForecastJobRequestSchema,
  ForecastJobDetailResponseSchema,
  ForecastJobResponseSchema,
  ForecastJobsResponseSchema,
  ForecastLatestResponseSchema,
  ForecastModelsResponseSchema,
  type CreateForecastJobRequest,
  type ForecastJob,
  type ForecastJobDetail,
  type ForecastLatest,
  type ForecastModel,
  type BacktestLeaderboard,
} from '@/schemas/forecasting';
import { t } from '@/i18n/forecasting';

export class ForecastingApiError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'ForecastingApiError';
  }
}

function unwrap<T>(envelope: { data: T; error: { code: string; message: string } | null }): T {
  if (envelope.error) throw new ForecastingApiError(envelope.error.message);
  return envelope.data;
}

export const forecastingApi = {
  async getModels(): Promise<ForecastModel[]> {
    try {
      const raw: unknown = await apiClient.get('forecast/usdidr/models').json();
      return unwrap(validateResponse(ForecastModelsResponseSchema, raw));
    } catch (error) {
      if (error instanceof ForecastingApiError) throw error;
      throw new ForecastingApiError(t('forecasting.modelsError'));
    }
  },

  async getLatest(model: string, horizon: number): Promise<ForecastLatest> {
    try {
      const raw: unknown = await apiClient.get('forecast/usdidr/latest', {
        searchParams: { model, horizon: String(horizon) },
      }).json();
      return unwrap(validateResponse(ForecastLatestResponseSchema, raw));
    } catch (error) {
      if (error instanceof ForecastingApiError) throw error;
      throw new ForecastingApiError(t('forecasting.forecastError'));
    }
  },

  async getLeaderboard(horizon: number): Promise<BacktestLeaderboard> {
    try {
      const raw: unknown = await apiClient.get('forecast/usdidr/leaderboard', {
        searchParams: { horizon: String(horizon) },
      }).json();
      return unwrap(validateResponse(BacktestLeaderboardResponseSchema, raw));
    } catch (error) {
      if (error instanceof ForecastingApiError) throw error;
      throw new ForecastingApiError(t('forecasting.leaderboardError'));
    }
  },

  async createJob(request: CreateForecastJobRequest): Promise<ForecastJob> {
    try {
      const body = validateResponse(CreateForecastJobRequestSchema, request);
      const raw: unknown = await apiClient.post('forecast/usdidr/jobs', { json: body }).json();
      return unwrap(validateResponse(ForecastJobResponseSchema, raw));
    } catch (error) {
      if (error instanceof ForecastingApiError) throw error;
      throw new ForecastingApiError(t('forecasting.launchError'));
    }
  },

  async getJobs(): Promise<ForecastJob[]> {
    try {
      const raw: unknown = await apiClient.get('forecast/usdidr/jobs').json();
      return unwrap(validateResponse(ForecastJobsResponseSchema, raw));
    } catch (error) {
      if (error instanceof ForecastingApiError) throw error;
      throw new ForecastingApiError(t('forecasting.jobsError'));
    }
  },

  async getJob(id: string): Promise<ForecastJobDetail> {
    try {
      const raw: unknown = await apiClient.get(`forecast/usdidr/jobs/${encodeURIComponent(id)}`).json();
      return unwrap(validateResponse(ForecastJobDetailResponseSchema, raw));
    } catch (error) {
      if (error instanceof ForecastingApiError) throw error;
      throw new ForecastingApiError(t('forecasting.jobError'));
    }
  },
};
