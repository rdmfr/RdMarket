import { defineStore } from 'pinia';
import { ref } from 'vue';
import type { TimeRange } from '@/types/market';

export const useSettingsStore = defineStore('settings', () => {
  const theme = ref<'dark' | 'light'>((localStorage.getItem('rdmarket_theme') as 'dark' | 'light') || 'dark');
  const defaultRange = ref<TimeRange>((localStorage.getItem('rdmarket_range') as TimeRange) || '1M');
  const timezone = ref<string>('Asia/Jakarta');
  const autoRefreshInterval = ref<number>(
    parseInt(localStorage.getItem('rdmarket_refresh') || '60', 10)
  );

  function setTheme(newTheme: 'dark' | 'light') {
    theme.value = newTheme;
    localStorage.setItem('rdmarket_theme', newTheme);
    document.documentElement.setAttribute('data-theme', newTheme);
  }

  function toggleTheme() {
    setTheme(theme.value === 'dark' ? 'light' : 'dark');
  }

  function setDefaultRange(range: TimeRange) {
    defaultRange.value = range;
    localStorage.setItem('rdmarket_range', range);
  }

  function setAutoRefreshInterval(seconds: number) {
    autoRefreshInterval.value = seconds;
    localStorage.setItem('rdmarket_refresh', seconds.toString());
  }

  // Initialize root attribute
  if (typeof document !== 'undefined') {
    document.documentElement.setAttribute('data-theme', theme.value);
  }

  return {
    theme,
    defaultRange,
    timezone,
    autoRefreshInterval,
    setTheme,
    toggleTheme,
    setDefaultRange,
    setAutoRefreshInterval,
  };
});
