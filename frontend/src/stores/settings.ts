import { defineStore } from 'pinia';
import { ref } from 'vue';
import { z } from 'zod';
import type { TimeRange } from '@/types/market';

const StoredSettingsSchema = z.object({
  theme: z.enum(['dark', 'light']).default('dark'),
  defaultRange: z.enum(['1D', '7D', '1M', '3M', '6M', '1Y', '5Y']).default('1M'),
  timezone: z.string().default('Asia/Jakarta'),
  autoRefreshInterval: z.union([z.literal(0), z.literal(1), z.literal(5), z.literal(15), z.literal(30)]).default(5),
});

function readSettings() {
  try {
    const raw = localStorage.getItem('rdmarket_settings');
    return raw ? StoredSettingsSchema.parse(JSON.parse(raw)) : StoredSettingsSchema.parse({});
  } catch {
    return StoredSettingsSchema.parse({});
  }
}

export const useSettingsStore = defineStore('settings', () => {
  const stored = readSettings();
  const theme = ref<'dark' | 'light'>(stored.theme);
  const defaultRange = ref<TimeRange>(stored.defaultRange);
  const timezone = ref<string>(stored.timezone);
  const autoRefreshInterval = ref<number>(stored.autoRefreshInterval);

  function persist() {
    localStorage.setItem('rdmarket_settings', JSON.stringify({
      theme: theme.value,
      defaultRange: defaultRange.value,
      timezone: timezone.value,
      autoRefreshInterval: autoRefreshInterval.value,
    }));
  }

  function setTheme(newTheme: 'dark' | 'light') {
    theme.value = newTheme;
    persist();
    document.documentElement.setAttribute('data-theme', newTheme);
  }

  function toggleTheme() {
    setTheme(theme.value === 'dark' ? 'light' : 'dark');
  }

  function setDefaultRange(range: TimeRange) {
    defaultRange.value = range;
    persist();
  }

  function setAutoRefreshInterval(seconds: number) {
    autoRefreshInterval.value = seconds;
    persist();
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
