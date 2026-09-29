<template>
  <Panel title="Data Ingestion & Provider Telemetry">
    <div v-if="marketStore.loading.dataSources && marketStore.dataSources.length === 0" class="py-2">
      <SkeletonBlock width="100%" height="28px" />
    </div>

    <ErrorState
      v-else-if="marketStore.errors.dataSources && marketStore.dataSources.length === 0"
      :message="marketStore.errors.dataSources"
      @retry="marketStore.fetchDataSources"
    />

    <div v-else class="overflow-x-auto">
      <div
        v-for="source in marketStore.dataSources"
        :key="source.provider"
        class="flex flex-wrap items-center justify-between gap-x-6 gap-y-2 py-1 px-2 bg-raised/30 border border-border/60 rounded-xs text-xs font-mono"
      >
        <div class="flex items-center gap-3">
          <div class="flex flex-col">
            <span class="text-[10px] text-muted uppercase font-sans">Primary Feed</span>
            <span class="text-primary font-medium">{{ source.provider }}</span>
          </div>
          <span class="text-border-strong">|</span>
          <div class="flex flex-col">
            <span class="text-[10px] text-muted uppercase font-sans">Pair</span>
            <span class="text-primary font-medium">{{ source.instrument }}</span>
          </div>
        </div>

        <div class="flex flex-wrap items-center gap-4 text-xs">
          <div class="flex items-center gap-1.5">
            <span class="text-muted font-sans text-[11px]">Status:</span>
            <span
              :class="[
                'font-medium',
                source.status === 'Connected' ? 'text-up' : 'text-down',
              ]"
            >
              {{ source.status }}
            </span>
          </div>

          <div class="flex items-center gap-1.5">
            <span class="text-muted font-sans text-[11px]">Frequency:</span>
            <span class="text-secondary">{{ source.data_frequency }}</span>
          </div>

          <div class="flex items-center gap-1.5">
            <span class="text-muted font-sans text-[11px]">Observations:</span>
            <span class="text-primary tabular-nums font-semibold">
              {{ formatCount(source.number_of_observations) }}
            </span>
          </div>

          <div class="flex items-center gap-1.5">
            <span class="text-muted font-sans text-[11px]">Last Sync:</span>
            <span class="text-secondary tabular-nums">
              {{ formatTimestampWIB(source.last_successful_update) }}
            </span>
          </div>
        </div>
      </div>
    </div>
  </Panel>
</template>

<script setup lang="ts">
import Panel from '@/components/common/Panel.vue';
import SkeletonBlock from '@/components/common/SkeletonBlock.vue';
import ErrorState from '@/components/common/ErrorState.vue';
import { useMarketStore } from '@/stores/market';
import { formatCount, formatTimestampWIB } from '@/utils/formatters';

const marketStore = useMarketStore();
</script>
