<template>
  <Panel title="Historical Exchange Rate" :noPadding="true" class="h-full">
    <template #controls>
      <!-- Time Range Selector -->
      <SegmentedControl
        :modelValue="marketStore.selectedRange"
        :options="rangeOptions"
        @update:modelValue="onRangeChange"
      />
    </template>

    <div class="relative w-full h-[380px] sm:h-[420px] p-2 flex flex-col justify-between">
      <!-- Loading Overlay -->
      <div
        v-if="marketStore.loading.history"
        class="absolute inset-0 bg-base/60 backdrop-none flex items-center justify-center z-10 select-none"
      >
        <div class="flex items-center gap-2 text-xs font-mono text-secondary bg-panel px-3 py-1.5 border border-border rounded-xs">
          <span class="w-2 h-2 rounded-full bg-accent animate-pulse" />
          <span>Loading market data...</span>
        </div>
      </div>

      <!-- Error State -->
      <div
        v-if="marketStore.errors.history && (!marketStore.historicalData || marketStore.historicalData.points.length === 0)"
        class="absolute inset-0 p-4 flex items-center justify-center z-10"
      >
        <ErrorState
          :message="marketStore.errors.history"
          @retry="marketStore.fetchHistory"
        />
      </div>

      <!-- Highcharts Container -->
      <div ref="chartContainer" class="w-full h-full min-h-[340px]" />
    </div>

    <!-- Chart Legend & Indicator Chips Footer -->
    <template #footer>
      <div class="flex flex-wrap items-center justify-between gap-2 py-0.5">
        <div class="flex flex-wrap items-center gap-2">
          <span class="text-[10px] text-muted font-mono uppercase tracking-wider">Series:</span>
          <!-- Actual Rate Indicator -->
          <div class="inline-flex items-center gap-1.5 px-2 py-0.5 text-[11px] font-mono border border-border rounded-xs bg-raised/30">
            <span class="w-2 h-0.5 bg-series-1" />
            <span class="text-primary font-medium">USD/IDR Rate</span>
          </div>

          <!-- SMA Toggles -->
          <ChipToggle
            :modelValue="marketStore.activeIndicators.sma7"
            label="SMA 7"
            swatchColor="var(--series-2)"
            @update:modelValue="toggleIndicator('sma7')"
          />
          <ChipToggle
            :modelValue="marketStore.activeIndicators.sma30"
            label="SMA 30"
            swatchColor="var(--series-3)"
            @update:modelValue="toggleIndicator('sma30')"
          />
          <ChipToggle
            :modelValue="marketStore.activeIndicators.sma90"
            label="SMA 90"
            swatchColor="var(--series-4)"
            @update:modelValue="toggleIndicator('sma90')"
          />
        </div>

        <div class="text-[11px] font-mono text-muted tabular-nums">
          <span v-if="marketStore.historicalData">
            {{ marketStore.historicalData.total_points }} points
          </span>
          <span v-if="marketStore.historicalData && marketStore.historicalData.missing_count > 0" class="ml-2 text-neutral">
            ({{ marketStore.historicalData.missing_count }} missing)
          </span>
        </div>
      </div>
    </template>
  </Panel>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, nextTick } from 'vue';
import Highcharts from 'highcharts';
import Panel from '@/components/common/Panel.vue';
import SegmentedControl, { type SegmentOption } from '@/components/common/SegmentedControl.vue';
import ChipToggle from '@/components/common/ChipToggle.vue';
import ErrorState from '@/components/common/ErrorState.vue';
import { useMarketStore } from '@/stores/market';
import { useSettingsStore } from '@/stores/settings';
import { applyHighchartsTheme } from '@/components/charts/theme';
import type { TimeRange } from '@/types/market';

const marketStore = useMarketStore();
const settingsStore = useSettingsStore();

const chartContainer = ref<HTMLDivElement | null>(null);
let chartInstance: Highcharts.Chart | null = null;

const rangeOptions: SegmentOption[] = [
  { label: '1D', value: '1D' },
  { label: '7D', value: '7D' },
  { label: '1M', value: '1M' },
  { label: '3M', value: '3M' },
  { label: '6M', value: '6M' },
  { label: '1Y', value: '1Y' },
  { label: '5Y', value: '5Y' },
];

function onRangeChange(val: string) {
  marketStore.setRange(val as TimeRange);
}

function toggleIndicator(key: 'sma7' | 'sma30' | 'sma90') {
  marketStore.toggleIndicator(key);
}

function initChart() {
  if (!chartContainer.value) return;

  applyHighchartsTheme(settingsStore.theme === 'dark');

  chartInstance = Highcharts.chart(chartContainer.value, {
    chart: {
      type: 'line',
      zooming: {
        type: 'x',
      },
    },
    series: [
      {
        type: 'line',
        name: 'USD/IDR',
        data: [],
        color: 'var(--series-1)',
        lineWidth: 1.5,
        zIndex: 5,
        connectNulls: false,
      },
      {
        type: 'line',
        name: 'SMA 7',
        data: [],
        color: 'var(--series-2)',
        lineWidth: 1,
        dashStyle: 'ShortDash',
        visible: marketStore.activeIndicators.sma7,
        zIndex: 4,
        connectNulls: false,
      },
      {
        type: 'line',
        name: 'SMA 30',
        data: [],
        color: 'var(--series-3)',
        lineWidth: 1,
        dashStyle: 'ShortDash',
        visible: marketStore.activeIndicators.sma30,
        zIndex: 3,
        connectNulls: false,
      },
      {
        type: 'line',
        name: 'SMA 90',
        data: [],
        color: 'var(--series-4)',
        lineWidth: 1,
        dashStyle: 'ShortDash',
        visible: marketStore.activeIndicators.sma90,
        zIndex: 2,
        connectNulls: false,
      },
    ],
  });

  updateChartData();
}

function updateChartData() {
  if (!chartInstance) return;

  const points = marketStore.historicalData?.points || [];
  const chartData = points.map((p) => [p.timestamp, p.rate]);

  // Actual rate series
  if (chartInstance.series[0]) {
    chartInstance.series[0].setData(chartData, false);
  }

  // Indicators series
  const indicators = marketStore.indicators;
  if (chartInstance.series[1]) {
    const sma7Data = (indicators?.sma_7 || []).map((p) => [p.timestamp, p.rate]);
    chartInstance.series[1].setData(sma7Data, false);
    chartInstance.series[1].setVisible(marketStore.activeIndicators.sma7, false);
  }

  if (chartInstance.series[2]) {
    const sma30Data = (indicators?.sma_30 || []).map((p) => [p.timestamp, p.rate]);
    chartInstance.series[2].setData(sma30Data, false);
    chartInstance.series[2].setVisible(marketStore.activeIndicators.sma30, false);
  }

  if (chartInstance.series[3]) {
    const sma90Data = (indicators?.sma_90 || []).map((p) => [p.timestamp, p.rate]);
    chartInstance.series[3].setData(sma90Data, false);
    chartInstance.series[3].setVisible(marketStore.activeIndicators.sma90, false);
  }

  chartInstance.redraw(false);
}

// Watchers
watch(
  () => [marketStore.historicalData, marketStore.indicators],
  () => {
    updateChartData();
  },
  { deep: true }
);

watch(
  () => marketStore.activeIndicators,
  () => {
    if (!chartInstance) return;
    if (chartInstance.series[1]) chartInstance.series[1].setVisible(marketStore.activeIndicators.sma7, false);
    if (chartInstance.series[2]) chartInstance.series[2].setVisible(marketStore.activeIndicators.sma30, false);
    if (chartInstance.series[3]) chartInstance.series[3].setVisible(marketStore.activeIndicators.sma90, false);
    chartInstance.redraw(false);
  },
  { deep: true }
);

watch(
  () => settingsStore.theme,
  (newTheme) => {
    if (chartInstance) {
      applyHighchartsTheme(newTheme === 'dark');
      chartInstance.destroy();
      initChart();
    }
  }
);

onMounted(() => {
  nextTick(() => {
    initChart();
  });
});

onUnmounted(() => {
  if (chartInstance) {
    chartInstance.destroy();
    chartInstance = null;
  }
});
</script>
