<template>
  <Panel title="Market Condition">
    <div v-if="marketStore.loading.indicators && !marketStore.indicators" class="space-y-2.5 py-1">
      <SkeletonBlock width="100%" height="28px" />
      <SkeletonBlock width="100%" height="28px" />
      <SkeletonBlock width="100%" height="28px" />
    </div>

    <ErrorState
      v-else-if="marketStore.errors.indicators && !marketStore.indicators"
      :message="marketStore.errors.indicators"
      @retry="marketStore.fetchIndicators"
    />

    <div v-else-if="marketStore.indicators" class="space-y-2">
      <!-- Short Term Condition -->
      <div class="flex items-center justify-between py-1.5 px-2 bg-raised/40 border border-border/60 rounded-xs">
        <div class="flex flex-col">
          <span class="text-xs text-secondary font-sans">Short Term</span>
          <span class="text-[10px] text-muted font-mono">7-day observed delta</span>
        </div>
        <div class="text-right flex items-center gap-2">
          <span
            :class="[
              'text-xs font-mono font-medium',
              conditionColor(marketStore.indicators.market_condition.short_term_condition),
            ]"
          >
            {{ marketStore.indicators.market_condition.short_term_condition }}
          </span>
          <span class="text-[10px] font-mono text-muted tabular-nums">
            ({{ formatPercent(marketStore.indicators.market_condition.recent_return_7d * 100) }})
          </span>
        </div>
      </div>

      <!-- 30-Day Trend -->
      <div class="flex items-center justify-between py-1.5 px-2 bg-raised/40 border border-border/60 rounded-xs">
        <div class="flex flex-col">
          <span class="text-xs text-secondary font-sans">30 Day Trend</span>
          <span class="text-[10px] text-muted font-mono">Monthly trajectory</span>
        </div>
        <div class="text-right flex items-center gap-2">
          <span
            :class="[
              'text-xs font-mono font-medium',
              trendColor(marketStore.indicators.market_condition.trend_30_day),
            ]"
          >
            {{ marketStore.indicators.market_condition.trend_30_day }}
          </span>
          <span class="text-[10px] font-mono text-muted tabular-nums">
            ({{ formatPercent(marketStore.indicators.market_condition.return_30d * 100) }})
          </span>
        </div>
      </div>

      <!-- Volatility -->
      <div class="flex items-center justify-between py-1.5 px-2 bg-raised/40 border border-border/60 rounded-xs">
        <div class="flex flex-col">
          <span class="text-xs text-secondary font-sans">Observed Volatility</span>
          <span class="text-[10px] text-muted font-mono">30D rolling std dev</span>
        </div>
        <div class="text-right flex items-center gap-2">
          <span
            :class="[
              'text-xs font-mono font-medium',
              volColor(marketStore.indicators.market_condition.volatility_level),
            ]"
          >
            {{ marketStore.indicators.market_condition.volatility_level }}
          </span>
          <span class="text-[10px] font-mono text-muted tabular-nums">
            ({{ (marketStore.indicators.market_condition.observed_volatility * 100).toFixed(2) }}%)
          </span>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="text-[10px] text-muted font-sans text-center">
        Descriptive statistics only. Not financial advice.
      </div>
    </template>
  </Panel>
</template>

<script setup lang="ts">
import Panel from '@/components/common/Panel.vue';
import SkeletonBlock from '@/components/common/SkeletonBlock.vue';
import ErrorState from '@/components/common/ErrorState.vue';
import { useMarketStore } from '@/stores/market';
import { formatPercent } from '@/utils/formatters';

const marketStore = useMarketStore();

function conditionColor(cond: string) {
  if (cond === 'Positive') return 'text-up';
  if (cond === 'Negative') return 'text-down';
  return 'text-neutral';
}

function trendColor(trend: string) {
  if (trend === 'Upward') return 'text-up';
  if (trend === 'Downward') return 'text-down';
  return 'text-neutral';
}

function volColor(vol: string) {
  if (vol === 'High') return 'text-down';
  if (vol === 'Moderate') return 'text-accent';
  return 'text-up';
}
</script>
