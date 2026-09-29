<template>
  <Panel title="Market Summary">
    <template #controls>
      <StatusBadge :status="marketStore.marketStatus" />
    </template>

    <div v-if="marketStore.loading.current && !marketStore.currentRate" class="space-y-3 py-1">
      <SkeletonBlock width="80px" height="14px" />
      <SkeletonBlock width="220px" height="36px" />
      <div class="flex gap-3">
        <SkeletonBlock width="80px" height="18px" />
        <SkeletonBlock width="70px" height="18px" />
      </div>
      <div class="pt-3 border-t border-border/60 flex justify-between">
        <SkeletonBlock width="100px" height="14px" />
        <SkeletonBlock width="120px" height="14px" />
      </div>
    </div>

    <ErrorState
      v-else-if="marketStore.errors.current && !marketStore.currentRate"
      :message="marketStore.errors.current"
      @retry="marketStore.fetchCurrentRate"
    />

    <div v-else-if="marketStore.currentRate" class="flex flex-col justify-between h-full">
      <!-- Instrument Header -->
      <div class="flex items-center justify-between">
        <div class="flex items-baseline gap-2">
          <span class="text-xs font-semibold tracking-wider text-muted font-mono uppercase">USD / IDR</span>
          <span class="text-[11px] text-muted">US Dollar to Indonesian Rupiah</span>
        </div>
        <span class="text-[10px] text-muted font-mono uppercase">{{ marketStore.currentRate.source }}</span>
      </div>

      <!-- Main Rate Display (Hero: 36px) -->
      <div class="my-2.5">
        <div class="flex items-baseline gap-2">
          <span class="text-xs font-mono text-muted">Rp</span>
          <span class="text-3xl sm:text-4xl font-mono tabular-nums font-semibold tracking-tight text-primary">
            {{ formatIDR(marketStore.currentRate.current_rate, true) }}
          </span>
          <span class="text-xs font-mono text-muted tabular-nums">
            ,{{ ((marketStore.currentRate.current_rate % 1) * 100).toFixed(0).padStart(2, '0') }}
          </span>
        </div>

        <!-- Price Change Row -->
        <div class="flex items-center gap-3 mt-1.5">
          <ValueChange
            :value="marketStore.currentRate.daily_change"
            :whole="false"
          />
          <ValueChange
            :value="marketStore.currentRate.daily_change_percent"
            :isPercent="true"
          />
          <span class="text-[11px] text-muted font-sans">vs previous close</span>
        </div>
      </div>

      <!-- Footer Details -->
      <div class="pt-2 border-t border-border/60 flex flex-wrap items-center justify-between gap-y-1 text-xs font-mono">
        <div class="flex items-center gap-1.5 text-secondary">
          <span class="text-muted font-sans">Previous Close:</span>
          <span class="text-primary tabular-nums">
            Rp {{ formatIDR(marketStore.currentRate.previous_close) }}
          </span>
        </div>

        <div class="flex items-center gap-1.5 text-muted text-[11px]">
          <span class="font-sans">Updated:</span>
          <span class="text-secondary tabular-nums">
            {{ formatTimestampWIB(marketStore.currentRate.timestamp) }}
          </span>
          <span v-if="relativeAge" class="text-muted text-[10px]">
            ({{ relativeAge }})
          </span>
        </div>
      </div>
    </div>
  </Panel>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import Panel from '@/components/common/Panel.vue';
import StatusBadge from '@/components/common/StatusBadge.vue';
import ValueChange from '@/components/common/ValueChange.vue';
import SkeletonBlock from '@/components/common/SkeletonBlock.vue';
import ErrorState from '@/components/common/ErrorState.vue';
import { useMarketStore } from '@/stores/market';
import { formatIDR, formatTimestampWIB, formatRelativeTime } from '@/utils/formatters';

const marketStore = useMarketStore();

const relativeAge = computed(() => {
  if (!marketStore.currentRate) return '';
  return formatRelativeTime(marketStore.currentRate.timestamp);
});
</script>
