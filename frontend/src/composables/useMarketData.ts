import { onMounted, onUnmounted, watch } from 'vue';
import { useMarketStore } from '@/stores/market';
import { useSettingsStore } from '@/stores/settings';

export function useMarketData() {
  const marketStore = useMarketStore();
  const settingsStore = useSettingsStore();

  let refreshTimer: number | null = null;

  function clearTimer() {
    if (refreshTimer !== null) {
      window.clearInterval(refreshTimer);
      refreshTimer = null;
    }
  }

  function startTimer() {
    clearTimer();
    const intervalSec = settingsStore.autoRefreshInterval;
    if (intervalSec > 0) {
      refreshTimer = window.setInterval(() => {
        // Only refresh current rate periodically to avoid overloading
        marketStore.fetchCurrentRate();
      }, intervalSec * 1000);
    }
  }

  onMounted(() => {
    // If no data loaded yet, load initial state
    if (!marketStore.currentRate) {
      marketStore.fetchAll();
    }
    startTimer();
  });

  onUnmounted(() => {
    clearTimer();
  });

  watch(
    () => settingsStore.autoRefreshInterval,
    () => {
      startTimer();
    }
  );

  return {
    marketStore,
    refreshAll: marketStore.fetchAll,
    refreshCurrentRate: marketStore.fetchCurrentRate,
  };
}
