import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import { marketApi, MarketApiError } from '@/services/marketApi';
import type {
  CurrentRate,
  HistoricalRate,
  Statistics,
  Indicators,
  DataSource,
  TimeRange,
} from '@/types/market';

export const useMarketStore = defineStore('market', () => {
  // State
  const currentRate = ref<CurrentRate | null>(null);
  const historicalData = ref<HistoricalRate | null>(null);
  const statistics = ref<Statistics | null>(null);
  const indicators = ref<Indicators | null>(null);
  const dataSources = ref<DataSource[]>([]);
  const selectedRange = ref<TimeRange>('1M');

  // Indicators toggle state
  const activeIndicators = ref({
    sma7: true,
    sma30: true,
    sma90: false,
  });

  // Loading states
  const loading = ref({
    current: false,
    history: false,
    stats: false,
    indicators: false,
    dataSources: false,
  });

  // Error states
  const errors = ref<{
    current: string | null;
    history: string | null;
    stats: string | null;
    indicators: string | null;
    dataSources: string | null;
  }>({
    current: null,
    history: null,
    stats: null,
    indicators: null,
    dataSources: null,
  });

  const lastUpdated = ref<Date | null>(null);

  // Computed status: LIVE, DELAYED, STALE, OFFLINE derived from data age
  const marketStatus = computed<'LIVE' | 'DELAYED' | 'STALE' | 'OFFLINE'>(() => {
    if (errors.value.current && !currentRate.value) {
      return 'OFFLINE';
    }
    if (!currentRate.value) {
      return 'OFFLINE';
    }
    const updateTime = new Date(currentRate.value.timestamp).getTime();
    const ageMinutes = (Date.now() - updateTime) / (1000 * 60);

    // Foreign exchange closes on weekends; daily quotes might be up to 72h old on Mon morning
    if (currentRate.value.is_stale || ageMinutes > 60 * 48) {
      return 'STALE';
    }
    if (ageMinutes > 60 * 4) {
      return 'DELAYED';
    }
    return 'LIVE';
  });

  // Actions
  async function fetchCurrentRate() {
    loading.value.current = true;
    errors.value.current = null;
    try {
      const data = await marketApi.getCurrentRate();
      currentRate.value = data;
      lastUpdated.value = new Date();
    } catch (err: unknown) {
      const msg = err instanceof MarketApiError ? err.message : 'Unable to retrieve current market data.';
      errors.value.current = msg;
    } finally {
      loading.value.current = false;
    }
  }

  async function fetchHistory(range: TimeRange = selectedRange.value) {
    loading.value.history = true;
    errors.value.history = null;
    try {
      const data = await marketApi.getHistory(range);
      historicalData.value = data;
    } catch (err: unknown) {
      const msg = err instanceof MarketApiError ? err.message : 'Unable to retrieve historical data.';
      errors.value.history = msg;
    } finally {
      loading.value.history = false;
    }
  }

  async function fetchStatistics() {
    loading.value.stats = true;
    errors.value.stats = null;
    try {
      const data = await marketApi.getStatistics();
      statistics.value = data;
    } catch (err: unknown) {
      const msg = err instanceof MarketApiError ? err.message : 'Unable to calculate market statistics.';
      errors.value.stats = msg;
    } finally {
      loading.value.stats = false;
    }
  }

  async function fetchIndicators(range: TimeRange = selectedRange.value) {
    loading.value.indicators = true;
    errors.value.indicators = null;
    try {
      const data = await marketApi.getIndicators(range);
      indicators.value = data;
    } catch (err: unknown) {
      const msg = err instanceof MarketApiError ? err.message : 'Unable to calculate indicators.';
      errors.value.indicators = msg;
    } finally {
      loading.value.indicators = false;
    }
  }

  async function fetchDataSources() {
    loading.value.dataSources = true;
    errors.value.dataSources = null;
    try {
      const data = await marketApi.getDataSources();
      dataSources.value = data;
    } catch (err: unknown) {
      const msg = err instanceof MarketApiError ? err.message : 'Unable to retrieve data sources.';
      errors.value.dataSources = msg;
    } finally {
      loading.value.dataSources = false;
    }
  }

  async function setRange(range: TimeRange) {
    if (selectedRange.value === range) return;
    selectedRange.value = range;
    await Promise.all([fetchHistory(range), fetchIndicators(range)]);
  }

  function toggleIndicator(key: 'sma7' | 'sma30' | 'sma90') {
    activeIndicators.value[key] = !activeIndicators.value[key];
  }

  async function fetchAll() {
    await Promise.all([
      fetchCurrentRate(),
      fetchHistory(selectedRange.value),
      fetchStatistics(),
      fetchIndicators(selectedRange.value),
      fetchDataSources(),
    ]);
  }

  return {
    currentRate,
    historicalData,
    statistics,
    indicators,
    dataSources,
    selectedRange,
    activeIndicators,
    loading,
    errors,
    lastUpdated,
    marketStatus,
    fetchCurrentRate,
    fetchHistory,
    fetchStatistics,
    fetchIndicators,
    fetchDataSources,
    setRange,
    toggleIndicator,
    fetchAll,
  };
});
