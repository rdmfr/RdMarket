<template>
  <Panel title="Market Statistics">
    <template #controls>
      <span class="text-[10px] font-mono text-muted uppercase">
        {{ marketStore.statistics?.has_sufficient_data ? 'Calculated from stored data' : 'Insufficient data' }}
      </span>
    </template>

    <div v-if="marketStore.loading.stats && !marketStore.statistics" class="space-y-2 py-1">
      <SkeletonBlock v-for="i in 6" :key="i" width="100%" height="22px" />
    </div>

    <ErrorState
      v-else-if="marketStore.errors.stats && !marketStore.statistics"
      :message="marketStore.errors.stats"
      @retry="marketStore.fetchStatistics"
    />

    <div v-else-if="marketStore.statistics" class="grid grid-cols-1 sm:grid-cols-2 gap-x-4">
      <!-- Column 1: Short Term & Changes -->
      <div class="space-y-0.5">
        <StatRow label="Current Rate" hint="Latest recorded USD/IDR exchange rate">
          <span class="text-primary font-medium">
            Rp {{ formatIDR(marketStore.statistics.current_rate) }}
          </span>
        </StatRow>

        <StatRow label="Previous Close" hint="Rate from preceding trading session">
          <span class="text-secondary">
            Rp {{ formatIDR(marketStore.statistics.previous_close) }}
          </span>
        </StatRow>

        <StatRow label="Daily Change" hint="Difference between current and previous close">
          <ValueChange :value="marketStore.statistics.daily_change" />
        </StatRow>

        <StatRow label="Daily Change %" hint="Percentage return vs previous close">
          <ValueChange :value="marketStore.statistics.daily_change_percent" :isPercent="true" />
        </StatRow>

        <StatRow label="Weekly Change %" hint="7-calendar-day return">
          <ValueChange :value="marketStore.statistics.weekly_change_percent" :isPercent="true" />
        </StatRow>

        <StatRow label="Monthly Change %" hint="30-calendar-day return">
          <ValueChange :value="marketStore.statistics.monthly_change_percent" :isPercent="true" />
        </StatRow>
      </div>

      <!-- Column 2: 52-Week & Aggregates -->
      <div class="space-y-0.5">
        <StatRow label="52 Week High" hint="Highest price recorded in the past 52 weeks">
          <span class="text-primary">
            Rp {{ formatIDR(marketStore.statistics.high_52_week) }}
          </span>
        </StatRow>

        <StatRow label="52 Week Low" hint="Lowest price recorded in the past 52 weeks">
          <span class="text-primary">
            Rp {{ formatIDR(marketStore.statistics.low_52_week) }}
          </span>
        </StatRow>

        <StatRow label="Average Rate" hint="Arithmetic mean of stored observations over past year">
          <span class="text-secondary">
            Rp {{ formatIDR(marketStore.statistics.average_rate) }}
          </span>
        </StatRow>

        <StatRow label="Minimum Rate" hint="Observed minimum in active dataset">
          <span class="text-secondary">
            Rp {{ formatIDR(marketStore.statistics.minimum_rate) }}
          </span>
        </StatRow>

        <StatRow label="Maximum Rate" hint="Observed maximum in active dataset">
          <span class="text-secondary">
            Rp {{ formatIDR(marketStore.statistics.maximum_rate) }}
          </span>
        </StatRow>

        <StatRow label="Observations Count" hint="Total verified records in database">
          <span class="text-muted font-mono">
            {{ formatCount(marketStore.statistics.observation_count) }}
          </span>
        </StatRow>
      </div>
    </div>
  </Panel>
</template>

<script setup lang="ts">
import Panel from '@/components/common/Panel.vue';
import StatRow from '@/components/common/StatRow.vue';
import ValueChange from '@/components/common/ValueChange.vue';
import SkeletonBlock from '@/components/common/SkeletonBlock.vue';
import ErrorState from '@/components/common/ErrorState.vue';
import { useMarketStore } from '@/stores/market';
import { formatIDR, formatCount } from '@/utils/formatters';

const marketStore = useMarketStore();
</script>
