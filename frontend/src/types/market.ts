export type TimeRange = '1D' | '7D' | '1M' | '3M' | '6M' | '1Y' | '5Y';

export interface CurrentRate {
  currency_pair: string;
  current_rate: number;
  previous_close: number;
  daily_change: number;
  daily_change_percent: number;
  timestamp: string; // ISO string
  source: string;
  is_stale: boolean;
}

export interface HistoricalPoint {
  timestamp: number; // Unix timestamp in ms
  rate: number | null; // Nullable for explicit missing data
}

export interface HistoricalRate {
  currency_pair: string;
  range: string;
  points: HistoricalPoint[];
  total_points: number;
  source_point_count?: number;
  missing_count: number;
  resolution?: string;
  aggregation_method?: string;
}

export interface Statistics {
  currency_pair: string;
  current_rate: number | null;
  previous_close: number | null;
  daily_change: number | null;
  daily_change_percent: number | null;
  weekly_change_percent: number | null;
  monthly_change_percent: number | null;
  high_52_week: number | null;
  low_52_week: number | null;
  average_rate: number | null;
  minimum_rate: number | null;
  maximum_rate: number | null;
  observation_count: number;
  has_sufficient_data: boolean;
}

export interface MarketCondition {
  short_term_condition: 'Positive' | 'Negative' | 'Neutral' | string;
  trend_30_day: 'Upward' | 'Downward' | 'Sideways' | string;
  volatility_level: 'Low' | 'Moderate' | 'High' | string;
  recent_return_7d: number;
  return_30d: number;
  observed_volatility: number;
  disclaimer: string;
}

export interface Indicators {
  currency_pair: string;
  range: string;
  sma_7: HistoricalPoint[];
  sma_30: HistoricalPoint[];
  sma_90: HistoricalPoint[];
  daily_return: number | null;
  weekly_return: number | null;
  rolling_volatility_30d: number | null;
  market_condition: MarketCondition;
}

export interface DataSource {
  provider: string;
  instrument: string;
  status: 'Connected' | 'Degraded' | 'Offline' | string;
  last_successful_update: string | null;
  data_frequency: string;
  number_of_observations: number;
  base_currency: string;
  target_currency: string;
}

export interface ApiError {
  code: string;
  message: string;
}

export interface ApiResponse<T> {
  data: T;
  meta: Record<string, unknown>;
  error: ApiError | null;
}
