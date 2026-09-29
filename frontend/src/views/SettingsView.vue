<template>
  <div class="max-w-[1920px] mx-auto space-y-3 pb-6">
    <Panel title="Platform Configuration & Preferences">
      <div class="max-w-2xl space-y-5 py-2 font-mono text-xs">
        <!-- Theme Selection -->
        <div class="flex items-center justify-between py-2 border-b border-border/60">
          <div>
            <div class="text-primary font-medium">Interface Color Theme</div>
            <div class="text-[11px] text-muted font-sans">
              Financial Bloomberg dark slate or warm-neutral paper light
            </div>
          </div>
          <div class="flex items-center gap-1 bg-raised border border-border p-0.5 rounded-xs">
            <button
              type="button"
              @click="settingsStore.setTheme('dark')"
              :class="[
                'px-2.5 py-1 text-[11px] rounded-xs cursor-pointer transition-colors',
                settingsStore.theme === 'dark' ? 'bg-panel text-accent font-semibold border border-border' : 'text-secondary hover:text-primary',
              ]"
            >
              Dark (Terminal)
            </button>
            <button
              type="button"
              @click="settingsStore.setTheme('light')"
              :class="[
                'px-2.5 py-1 text-[11px] rounded-xs cursor-pointer transition-colors',
                settingsStore.theme === 'light' ? 'bg-panel text-accent font-semibold border border-border' : 'text-secondary hover:text-primary',
              ]"
            >
              Light (Paper)
            </button>
          </div>
        </div>

        <!-- Default Chart Timeframe -->
        <div class="flex items-center justify-between py-2 border-b border-border/60">
          <div>
            <div class="text-primary font-medium">Default Chart Timeframe</div>
            <div class="text-[11px] text-muted font-sans">
              Initial observation window when launching the dashboard
            </div>
          </div>
          <select
            v-model="selectedDefaultRange"
            @change="onDefaultRangeChange"
            class="bg-raised border border-border text-primary px-2.5 py-1 rounded-xs text-xs font-mono cursor-pointer focus:outline-none focus:border-accent"
          >
            <option value="1D">1 Day (1D)</option>
            <option value="7D">7 Days (7D)</option>
            <option value="1M">1 Month (1M)</option>
            <option value="3M">3 Months (3M)</option>
            <option value="6M">6 Months (6M)</option>
            <option value="1Y">1 Year (1Y)</option>
            <option value="5Y">5 Years (5Y)</option>
          </select>
        </div>

        <!-- Auto Refresh Cadence -->
        <div class="flex items-center justify-between py-2 border-b border-border/60">
          <div>
            <div class="text-primary font-medium">Auto-Refresh Cadence</div>
            <div class="text-[11px] text-muted font-sans">
              Background polling interval for latest exchange-rate observation
            </div>
          </div>
          <select
            v-model="selectedRefresh"
            @change="onRefreshChange"
            class="bg-raised border border-border text-primary px-2.5 py-1 rounded-xs text-xs font-mono cursor-pointer focus:outline-none focus:border-accent"
          >
            <option :value="0">Off (Manual refresh only)</option>
            <option :value="60">Every 1 minute</option>
            <option :value="300">Every 5 minutes</option>
            <option :value="900">Every 15 minutes</option>
            <option :value="1800">Every 30 minutes</option>
          </select>
        </div>

        <!-- Timezone (Fixed) -->
        <div class="flex items-center justify-between py-2 border-b border-border/60">
          <div>
            <div class="text-primary font-medium">Display Timezone</div>
            <div class="text-[11px] text-muted font-sans">
              Canonical Indonesian market operational hours
            </div>
          </div>
          <div class="px-2.5 py-1 bg-raised/50 border border-border/60 rounded-xs text-secondary font-mono text-[11px]">
            Asia/Jakarta (WIB, UTC+7)
          </div>
        </div>

        <!-- Phase 2 Roadmap Status -->
        <div class="p-3 bg-panel border border-border rounded-xs space-y-1.5 text-xs">
          <div class="text-[11px] font-semibold text-accent uppercase tracking-wider">
            Architecture Ready: Phase 2 Forecast Compatibility
          </div>
          <p class="text-[11px] text-muted leading-relaxed font-sans">
            PostgreSQL schema, repository abstractions, and API response envelope models are strictly prepared to connect an upcoming Python ARIMA/Prophet/ML forecasting microservice in Phase 2 without rewriting existing data collectors or monitoring pipelines.
          </p>
        </div>
      </div>
    </Panel>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import Panel from '@/components/common/Panel.vue';
import { useSettingsStore } from '@/stores/settings';
import type { TimeRange } from '@/types/market';

const settingsStore = useSettingsStore();

const selectedDefaultRange = ref<TimeRange>(settingsStore.defaultRange);
const selectedRefresh = ref<number>(settingsStore.autoRefreshInterval);

function onDefaultRangeChange() {
  settingsStore.setDefaultRange(selectedDefaultRange.value);
}

function onRefreshChange() {
  settingsStore.setAutoRefreshInterval(Number(selectedRefresh.value));
}
</script>
