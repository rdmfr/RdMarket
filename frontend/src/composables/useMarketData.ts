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
    const intervalMinutes = settingsStore.autoRefreshInterval;
    if (intervalMinutes > 0) {
      refreshTimer = window.setInterval(() => {
        if (!document.hidden) marketStore.fetchCurrentRate();
      }, intervalMinutes * 60 * 1000);
    }
  }

  onMounted(() => {
    // If no data loaded yet, load initial state
    if (!marketStore.currentRate) {
      marketStore.fetchAll();
    }
    startTimer();
    document.addEventListener('visibilitychange', handleVisibilityChange);
  });

  onUnmounted(() => {
    clearTimer();
    document.removeEventListener('visibilitychange', handleVisibilityChange);
  });

  function handleVisibilityChange() {
    if (document.hidden) clearTimer();
    else startTimer();
  }

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
