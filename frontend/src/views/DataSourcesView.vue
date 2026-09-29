<template>
  <div class="max-w-[1920px] mx-auto space-y-3 pb-6">
    <Panel title="Market Data Ingestion Pipeline & Provider Feeds">
      <template #controls>
        <button
          type="button"
          @click="marketStore.fetchDataSources"
          class="px-2.5 py-1 bg-raised hover:bg-border text-primary border border-border rounded-xs text-[11px] font-mono cursor-pointer transition-colors"
        >
          Check Connectivity
        </button>
      </template>

      <div v-if="marketStore.loading.dataSources && marketStore.dataSources.length === 0" class="space-y-3 py-2">
        <SkeletonBlock width="100%" height="80px" />
      </div>

      <ErrorState
        v-else-if="marketStore.errors.dataSources && marketStore.dataSources.length === 0"
        :message="marketStore.errors.dataSources"
        @retry="marketStore.fetchDataSources"
      />

      <div v-else class="space-y-4">
        <!-- Feed Cards -->
        <div
          v-for="feed in marketStore.dataSources"
          :key="feed.provider"
          class="border border-border rounded-xs p-4 bg-raised/30 space-y-3 font-mono"
        >
          <div class="flex flex-wrap items-center justify-between gap-2 border-b border-border/60 pb-3">
            <div class="flex items-center gap-3">
              <span class="text-sm font-semibold text-primary">{{ feed.provider }}</span>
              <span class="px-2 py-0.5 text-[10px] bg-panel border border-border rounded-xs text-secondary">
                {{ feed.instrument }}
              </span>
            </div>

            <div class="flex items-center gap-2">
              <span class="text-xs text-muted font-sans">Status:</span>
              <span
                :class="[
                  'text-xs font-semibold px-2 py-0.5 border rounded-xs',
                  feed.status === 'Connected'
                    ? 'border-up/40 bg-up/10 text-up'
                    : 'border-down/40 bg-down/10 text-down',
                ]"
              >
                {{ feed.status }}
              </span>
            </div>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 text-xs">
            <div class="space-y-0.5">
              <div class="text-[11px] text-muted font-sans uppercase">Base Currency</div>
              <div class="text-primary font-medium">{{ feed.base_currency }} (United States Dollar)</div>
            </div>

            <div class="space-y-0.5">
              <div class="text-[11px] text-muted font-sans uppercase">Target Currency</div>
              <div class="text-primary font-medium">{{ feed.target_currency }} (Indonesian Rupiah)</div>
            </div>

            <div class="space-y-0.5">
              <div class="text-[11px] text-muted font-sans uppercase">Update Cadence</div>
              <div class="text-secondary">{{ feed.data_frequency }}</div>
            </div>

            <div class="space-y-0.5">
              <div class="text-[11px] text-muted font-sans uppercase">Total Stored Observations</div>
              <div class="text-primary font-medium tabular-nums">{{ formatCount(feed.number_of_observations) }} records</div>
            </div>
          </div>

          <div class="pt-2 border-t border-border/40 flex items-center justify-between text-[11px] text-muted">
            <div class="flex items-center gap-1.5">
              <span class="font-sans">Last Successful Transmission:</span>
              <span class="text-secondary tabular-nums">{{ formatTimestampWIB(feed.last_successful_update) }}</span>
            </div>
            <div class="text-[10px]">
              PostgreSQL UTC Storage · Deduplicated on (pair, time, source)
            </div>
          </div>
        </div>

        <!-- Ingestion Pipeline Specs -->
        <div class="p-3 bg-panel border border-border rounded-xs space-y-2 text-xs font-mono">
          <div class="text-xs font-semibold text-primary uppercase tracking-wider">
            Architecture & Ingestion Guarantees
          </div>
          <ul class="text-[11px] text-secondary space-y-1 list-disc list-inside">
            <li>Provider Abstraction: <span class="text-primary">ExchangeRateProvider</span> interface isolates HTTP transport from business logic.</li>
            <li>Database Deduplication: Unique composite constraint prevents duplicate records for identical timestamps.</li>
            <li>Timezone Handling: Storage is strictly UTC; display values are converted to Asia/Jakarta (WIB).</li>
            <li>Missing Data Policy: Market holidays and closures are preserved as genuine gaps without fabricated interpolation.</li>
          </ul>
        </div>
      </div>
    </Panel>
  </div>
</template>

<script setup lang="ts">
import Panel from '@/components/common/Panel.vue';
import SkeletonBlock from '@/components/common/SkeletonBlock.vue';
import ErrorState from '@/components/common/ErrorState.vue';
import { useMarketStore } from '@/stores/market';
import { formatCount, formatTimestampWIB } from '@/utils/formatters';

const marketStore = useMarketStore();
</script>
