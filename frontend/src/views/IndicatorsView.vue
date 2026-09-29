<template>
  <div class="max-w-[1920px] mx-auto space-y-3 pb-6">
    <Panel title="Statistical Indicator Methodology & Metrics">
      <template #controls>
        <span class="text-[10px] font-mono text-muted uppercase">Rule-Based Analytics Engine</span>
      </template>

      <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <!-- Left: Current Computed Indicators -->
        <div class="space-y-3">
          <div class="text-xs font-semibold text-primary uppercase font-mono tracking-wider pb-1 border-b border-border">
            Active Indicators (USD/IDR)
          </div>

          <div class="space-y-1">
            <StatRow label="7-Day Simple Moving Average (SMA 7)" hint="Short-term trend smoothed over 7 observation intervals">
              <span class="text-primary font-medium">
                <span v-if="latestSMA7">Rp {{ formatIDR(latestSMA7) }}</span>
                <span v-else class="text-muted cursor-help" title="Not enough data">—</span>
              </span>
            </StatRow>

            <StatRow label="30-Day Simple Moving Average (SMA 30)" hint="Intermediate baseline smoothed over 30 days">
              <span class="text-primary font-medium">
                <span v-if="latestSMA30">Rp {{ formatIDR(latestSMA30) }}</span>
                <span v-else class="text-muted cursor-help" title="Not enough data">—</span>
              </span>
            </StatRow>

            <StatRow label="90-Day Simple Moving Average (SMA 90)" hint="Quarterly structural baseline over 90 days">
              <span class="text-primary font-medium">
                <span v-if="latestSMA90">Rp {{ formatIDR(latestSMA90) }}</span>
                <span v-else class="text-muted cursor-help" title="Not enough data">—</span>
              </span>
            </StatRow>

            <StatRow label="Daily Return" hint="(Latest Rate - Previous Rate) / Previous Rate">
              <ValueChange
                v-if="marketStore.indicators?.daily_return !== null && marketStore.indicators?.daily_return !== undefined"
                :value="marketStore.indicators.daily_return * 100"
                :isPercent="true"
              />
              <span v-else class="text-muted cursor-help" title="Not enough data">—</span>
            </StatRow>

            <StatRow label="Weekly Return" hint="Return over past 7 calendar days">
              <ValueChange
                v-if="marketStore.indicators?.weekly_return !== null && marketStore.indicators?.weekly_return !== undefined"
                :value="marketStore.indicators.weekly_return * 100"
                :isPercent="true"
              />
              <span v-else class="text-muted cursor-help" title="Not enough data">—</span>
            </StatRow>

            <StatRow label="30-Day Rolling Volatility" hint="Sample standard deviation of daily percentage returns over 30 days">
              <span class="text-primary font-medium">
                <span v-if="marketStore.indicators?.rolling_volatility_30d !== null && marketStore.indicators?.rolling_volatility_30d !== undefined">
                  {{ (marketStore.indicators.rolling_volatility_30d * 100).toFixed(2) }}%
                </span>
                <span v-else class="text-muted cursor-help" title="Not enough data">—</span>
              </span>
            </StatRow>
          </div>
        </div>

        <!-- Right: Transparent Classification Rules -->
        <div class="space-y-3">
          <div class="text-xs font-semibold text-primary uppercase font-mono tracking-wider pb-1 border-b border-border">
            Condition Classification Rules
          </div>

          <div class="space-y-2 text-xs font-mono">
            <div class="p-2.5 bg-raised/40 border border-border/60 rounded-xs space-y-1">
              <div class="text-secondary font-medium">Short Term Condition</div>
              <p class="text-muted text-[11px] leading-relaxed">
                Evaluated against the 7-day observed return (threshold = ±0.20%):
              </p>
              <div class="text-[11px] text-primary space-y-0.5">
                <div>• Return &gt; +0.20% → <span class="text-up font-semibold">Positive</span></div>
                <div>• Return &lt; -0.20% → <span class="text-down font-semibold">Negative</span></div>
                <div>• Otherwise → <span class="text-neutral font-semibold">Neutral</span></div>
              </div>
            </div>

            <div class="p-2.5 bg-raised/40 border border-border/60 rounded-xs space-y-1">
              <div class="text-secondary font-medium">30-Day Trend</div>
              <p class="text-muted text-[11px] leading-relaxed">
                Evaluated against the 30-day cumulative rate return (threshold = ±0.50%):
              </p>
              <div class="text-[11px] text-primary space-y-0.5">
                <div>• Return &gt; +0.50% → <span class="text-up font-semibold">Upward</span></div>
                <div>• Return &lt; -0.50% → <span class="text-down font-semibold">Downward</span></div>
                <div>• Otherwise → <span class="text-neutral font-semibold">Sideways</span></div>
              </div>
            </div>

            <div class="p-2.5 bg-raised/40 border border-border/60 rounded-xs space-y-1">
              <div class="text-secondary font-medium">Observed Volatility Level</div>
              <p class="text-muted text-[11px] leading-relaxed">
                Evaluated against daily return standard deviation over 30 sessions:
              </p>
              <div class="text-[11px] text-primary space-y-0.5">
                <div>• Volatility &lt; 0.40% → <span class="text-up font-semibold">Low</span></div>
                <div>• Volatility &gt; 1.00% → <span class="text-down font-semibold">High</span></div>
                <div>• 0.40% – 1.00% → <span class="text-accent font-semibold">Moderate</span></div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="text-[11px] text-muted text-center font-sans">
          All indicators are purely backward-looking descriptive statistical calculations. Not financial advice.
        </div>
      </template>
    </Panel>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import Panel from '@/components/common/Panel.vue';
import StatRow from '@/components/common/StatRow.vue';
import ValueChange from '@/components/common/ValueChange.vue';
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
