import { apiClient } from './api';
import {
  CurrentRateSchema,
  HistoricalRateSchema,
  StatisticsSchema,
  IndicatorsSchema,
  DataSourceSchema,
  createEnvelopeSchema,
} from '@/schemas/market';
import type {
  CurrentRate,
  HistoricalRate,
  Statistics,
  Indicators,
  DataSource,
  TimeRange,
} from '@/types/market';
import { z } from 'zod';

export class MarketApiError extends Error {
  code: string;
  constructor(message: string, code = 'API_ERROR') {
    super(message);
    this.name = 'MarketApiError';
    this.code = code;
  }
}

export const marketApi = {
  async getCurrentRate(): Promise<CurrentRate> {
    try {
      const raw = await apiClient.get('market/usdidr/current').json();
      const schema = createEnvelopeSchema(CurrentRateSchema);
      const parsed = schema.safeParse(raw);

      if (!parsed.success) {
        console.error('CurrentRate validation error:', parsed.error);
        throw new MarketApiError('Invalid data received for current USD/IDR rate', 'SCHEMA_VALIDATION_ERROR');
      }

      if (parsed.data.error) {
        throw new MarketApiError(parsed.data.error.message, parsed.data.error.code);
      }

      return parsed.data.data;
    } catch (err: unknown) {
      if (err instanceof MarketApiError) throw err;
      throw new MarketApiError('Unable to retrieve current market data. Source responded with error.', 'NETWORK_ERROR');
    }
  },

  async getHistory(range: TimeRange = '1M', start?: string, end?: string): Promise<HistoricalRate> {
    try {
      const searchParams: Record<string, string> = { range };
      if (start) searchParams.start = start;
      if (end) searchParams.end = end;

      const raw = await apiClient.get('market/usdidr/history', { searchParams }).json();
      const schema = createEnvelopeSchema(HistoricalRateSchema);
      const parsed = schema.safeParse(raw);

      if (!parsed.success) {
        console.error('HistoricalRate validation error:', parsed.error);
        throw new MarketApiError('Invalid data received for historical rate series', 'SCHEMA_VALIDATION_ERROR');
      }

      if (parsed.data.error) {
        throw new MarketApiError(parsed.data.error.message, parsed.data.error.code);
      }

      return parsed.data.data;
    } catch (err: unknown) {
      if (err instanceof MarketApiError) throw err;
      throw new MarketApiError('Unable to retrieve historical exchange-rate data.', 'NETWORK_ERROR');
    }
  },

  async getStatistics(): Promise<Statistics> {
    try {
      const raw = await apiClient.get('market/usdidr/statistics').json();
      const schema = createEnvelopeSchema(StatisticsSchema);
      const parsed = schema.safeParse(raw);

      if (!parsed.success) {
        console.error('Statistics validation error:', parsed.error);
        throw new MarketApiError('Invalid data received for market statistics', 'SCHEMA_VALIDATION_ERROR');
      }

      if (parsed.data.error) {
        throw new MarketApiError(parsed.data.error.message, parsed.data.error.code);
      }

      return parsed.data.data;
    } catch (err: unknown) {
      if (err instanceof MarketApiError) throw err;
      throw new MarketApiError('Unable to calculate market statistics from stored data.', 'NETWORK_ERROR');
    }
  },

  async getIndicators(range: TimeRange = '1M'): Promise<Indicators> {
    try {
      const raw = await apiClient.get('market/usdidr/indicators', {
        searchParams: { range },
      }).json();
      const schema = createEnvelopeSchema(IndicatorsSchema);
      const parsed = schema.safeParse(raw);

      if (!parsed.success) {
        console.error('Indicators validation error:', parsed.error);
        throw new MarketApiError('Invalid data received for market indicators', 'SCHEMA_VALIDATION_ERROR');
      }

      if (parsed.data.error) {
        throw new MarketApiError(parsed.data.error.message, parsed.data.error.code);
      }

      return parsed.data.data;
    } catch (err: unknown) {
      if (err instanceof MarketApiError) throw err;
      throw new MarketApiError('Unable to calculate statistical indicators.', 'NETWORK_ERROR');
    }
  },

  async getDataSources(): Promise<DataSource[]> {
    try {
      const raw = await apiClient.get('data-sources').json();
      const schema = createEnvelopeSchema(z.array(DataSourceSchema));
      const parsed = schema.safeParse(raw);

      if (!parsed.success) {
        console.error('DataSources validation error:', parsed.error);
        throw new MarketApiError('Invalid data received for data sources', 'SCHEMA_VALIDATION_ERROR');
      }

      if (parsed.data.error) {
        throw new MarketApiError(parsed.data.error.message, parsed.data.error.code);
      }

      return parsed.data.data;
    } catch (err: unknown) {
      if (err instanceof MarketApiError) throw err;
      throw new MarketApiError('Unable to retrieve data source information.', 'NETWORK_ERROR');
    }
  },

  async checkHealth(): Promise<boolean> {
    try {
      const raw = await apiClient.get('health').json();
      return !!raw;
    } catch {
      return false;
    }
  },
};
