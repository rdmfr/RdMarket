<template>
  <Panel title="Statistical Indicators">
    <template #controls>
      <span class="text-[10px] font-mono text-muted uppercase">Phase 1 Analytics</span>
    </template>

    <div v-if="marketStore.loading.indicators && !marketStore.indicators" class="space-y-2 py-1">
      <SkeletonBlock v-for="i in 5" :key="i" width="100%" height="22px" />
    </div>

    <ErrorState
      v-else-if="marketStore.errors.indicators && !marketStore.indicators"
      :message="marketStore.errors.indicators"
      @retry="marketStore.fetchIndicators"
    />

    <div v-else-if="marketStore.indicators" class="space-y-3">
      <!-- Indicators Toggle List -->
      <div class="space-y-1">
        <!-- SMA 7 -->
        <div class="flex items-center justify-between py-1.5 px-2 border-b border-border/60 hover:bg-raised/40 transition-colors">
          <div class="flex items-center gap-2">
            <span class="w-2.5 h-1 bg-series-2 rounded-[1px]" />
            <div class="flex flex-col">
              <span class="text-xs text-primary font-medium">SMA 7</span>
              <span class="text-[10px] text-muted font-mono">7-day moving average</span>
            </div>
          </div>
          <div class="flex items-center gap-3">
            <span class="font-mono text-xs tabular-nums text-primary font-medium">
              <span v-if="latestSMA7">Rp {{ formatIDR(latestSMA7) }}</span>
              <span v-else class="text-muted cursor-help" title="Not enough data">—</span>
            </span>
            <button
              type="button"
              @click="marketStore.toggleIndicator('sma7')"
              :class="[
                'text-[10px] font-mono px-2 py-0.5 border rounded-xs cursor-pointer transition-colors',
                marketStore.activeIndicators.sma7
                  ? 'border-series-2 text-series-2 bg-series-2/10'
                  : 'border-border text-muted hover:text-secondary',
              ]"
            >
              {{ marketStore.activeIndicators.sma7 ? 'Active' : 'Off' }}
            </button>
          </div>
        </div>

        <!-- SMA 30 -->
        <div class="flex items-center justify-between py-1.5 px-2 border-b border-border/60 hover:bg-raised/40 transition-colors">
          <div class="flex items-center gap-2">
            <span class="w-2.5 h-1 bg-series-3 rounded-[1px]" />
            <div class="flex flex-col">
              <span class="text-xs text-primary font-medium">SMA 30</span>
              <span class="text-[10px] text-muted font-mono">30-day moving average</span>
            </div>
          </div>
          <div class="flex items-center gap-3">
            <span class="font-mono text-xs tabular-nums text-primary font-medium">
              <span v-if="latestSMA30">Rp {{ formatIDR(latestSMA30) }}</span>
              <span v-else class="text-muted cursor-help" title="Not enough data">—</span>
            </span>
            <button
              type="button"
              @click="marketStore.toggleIndicator('sma30')"
              :class="[
                'text-[10px] font-mono px-2 py-0.5 border rounded-xs cursor-pointer transition-colors',
                marketStore.activeIndicators.sma30
                  ? 'border-series-3 text-series-3 bg-series-3/10'
                  : 'border-border text-muted hover:text-secondary',
              ]"
            >
              {{ marketStore.activeIndicators.sma30 ? 'Active' : 'Off' }}
            </button>
          </div>
        </div>

        <!-- SMA 90 -->
        <div class="flex items-center justify-between py-1.5 px-2 border-b border-border/60 hover:bg-raised/40 transition-colors">
          <div class="flex items-center gap-2">
            <span class="w-2.5 h-1 bg-series-4 rounded-[1px]" />
            <div class="flex flex-col">
              <span class="text-xs text-primary font-medium">SMA 90</span>
              <span class="text-[10px] text-muted font-mono">Quarterly baseline</span>
            </div>
          </div>
          <div class="flex items-center gap-3">
            <span class="font-mono text-xs tabular-nums text-primary font-medium">
              <span v-if="latestSMA90">Rp {{ formatIDR(latestSMA90) }}</span>
              <span v-else class="text-muted cursor-help" title="Not enough data">—</span>
            </span>
            <button
              type="button"
              @click="marketStore.toggleIndicator('sma90')"
              :class="[
                'text-[10px] font-mono px-2 py-0.5 border rounded-xs cursor-pointer transition-colors',
                marketStore.activeIndicators.sma90
                  ? 'border-series-4 text-series-4 bg-series-4/10'
                  : 'border-border text-muted hover:text-secondary',
              ]"
            >
              {{ marketStore.activeIndicators.sma90 ? 'Active' : 'Off' }}
            </button>
          </div>
        </div>
      </div>

      <!-- Returns and Volatility -->
      <div class="grid grid-cols-3 gap-2 pt-1 border-t border-border/60 text-center font-mono">
        <div class="bg-raised/40 p-2 border border-border/50 rounded-xs">
          <div class="text-[10px] text-muted uppercase font-sans">Daily Return</div>
          <div class="mt-1 text-xs">
            <ValueChange
              v-if="marketStore.indicators.daily_return !== null"
              :value="marketStore.indicators.daily_return * 100"
              :isPercent="true"
            />
            <span v-else class="text-muted cursor-help" title="Not enough data">—</span>
          </div>
        </div>

        <div class="bg-raised/40 p-2 border border-border/50 rounded-xs">
          <div class="text-[10px] text-muted uppercase font-sans">Weekly Return</div>
          <div class="mt-1 text-xs">
            <ValueChange
              v-if="marketStore.indicators.weekly_return !== null"
              :value="marketStore.indicators.weekly_return * 100"
              :isPercent="true"
            />
            <span v-else class="text-muted cursor-help" title="Not enough data">—</span>
          </div>
        </div>

        <div class="bg-raised/40 p-2 border border-border/50 rounded-xs">
          <div class="text-[10px] text-muted uppercase font-sans">30D Volatility</div>
          <div class="mt-1 text-xs font-semibold text-primary tabular-nums">
            <span v-if="marketStore.indicators.rolling_volatility_30d !== null">
              {{ (marketStore.indicators.rolling_volatility_30d * 100).toFixed(2) }}%
            </span>
            <span v-else class="text-muted cursor-help" title="Not enough data">—</span>
          </div>
        </div>
      </div>
    </div>
  </Panel>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import Panel from '@/components/common/Panel.vue';
import ValueChange from '@/components/common/ValueChange.vue';
import SkeletonBlock from '@/components/common/SkeletonBlock.vue';
import ErrorState from '@/components/common/ErrorState.vue';
import { useMarketStore } from '@/stores/market';
import { formatIDR } from '@/utils/formatters';

const marketStore = useMarketStore();

const latestSMA7 = computed(() => {
  const pts = marketStore.indicators?.sma_7;
  if (!pts || pts.length === 0) return null;
  return pts[pts.length - 1].rate;
});

const latestSMA30 = computed(() => {
  const pts = marketStore.indicators?.sma_30;
  if (!pts || pts.length === 0) return null;
  return pts[pts.length - 1].rate;
});

const latestSMA90 = computed(() => {
  const pts = marketStore.indicators?.sma_90;
  if (!pts || pts.length === 0) return null;
  return pts[pts.length - 1].rate;
});
</script>
