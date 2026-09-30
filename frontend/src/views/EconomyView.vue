<template>
  <div class="max-w-[1920px] mx-auto space-y-3 pb-6">
    <div class="flex items-center justify-between gap-3">
      <h1 class="text-sm font-medium text-primary">{{ tEconomic('economic.title') }}</h1>
      <button
        type="button"
        class="h-8 px-2 border border-border rounded-xs text-secondary hover:text-primary hover:bg-raised text-xs"
        :disabled="loading"
        :aria-label="tEconomic('economic.refresh')"
        @click="load"
      >
        {{ tEconomic('economic.refresh') }}
      </button>
    </div>

    <Panel :title="tEconomic('economic.overview')" :no-padding="true">
      <template #controls>
        <div class="flex items-center gap-2">
          <label class="sr-only" for="economic-country">{{ tEconomic('economic.country') }}</label>
          <select id="economic-country" v-model="country" class="h-7 max-w-36 bg-raised border border-border rounded-xs px-2 text-[11px] text-secondary">
            <option value="all">{{ tEconomic('economic.allCountries') }}</option>
            <option v-for="item in countries" :key="item" :value="item">{{ item }}</option>
          </select>
          <label class="sr-only" for="economic-category">{{ tEconomic('economic.category') }}</label>
          <select id="economic-category" v-model="category" class="h-7 max-w-36 bg-raised border border-border rounded-xs px-2 text-[11px] text-secondary">
            <option value="all">{{ tEconomic('economic.allCategories') }}</option>
            <option v-for="item in categories" :key="item" :value="item">{{ item.replaceAll('_', ' ') }}</option>
          </select>
        </div>
      </template>

      <ErrorState
        v-if="error"
        :message="tEconomic('economic.loadError')"
        :retry-label="tEconomic('economic.retry')"
        @retry="load"
      />
      <div v-else-if="loading && indicators.length === 0" class="space-y-2 p-3">
        <SkeletonBlock v-for="row in 4" :key="row" height="24px" />
      </div>
      <EmptyState
        v-else-if="filteredIndicators.length === 0"
        :message="tEconomic('economic.noIndicators')"
        :action-label="tEconomic('economic.refresh')"
        @action="load"
      />
      <div v-else class="overflow-x-auto">
        <table class="w-full min-w-225 border-collapse text-left text-xs">
          <thead class="bg-raised text-[10px] uppercase text-muted">
            <tr>
              <th class="px-3 py-2 font-medium">{{ tEconomic('economic.indicator') }}</th>
              <th class="px-3 py-2 font-medium">{{ tEconomic('economic.latest') }}</th>
              <th class="px-3 py-2 font-medium">{{ tEconomic('economic.previous') }}</th>
              <th class="px-3 py-2 font-medium">{{ tEconomic('economic.change') }}</th>
              <th class="px-3 py-2 font-medium">{{ tEconomic('economic.referencePeriod') }}</th>
              <th class="px-3 py-2 font-medium">{{ tEconomic('economic.released') }}</th>
              <th class="px-3 py-2 font-medium">{{ tEconomic('economic.trend') }}</th>
              <th class="px-3 py-2 font-medium">{{ tEconomic('economic.status') }}</th>
              <th class="px-3 py-2 font-medium">{{ tEconomic('economic.source') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in filteredIndicators" :key="item.code" class="border-t border-border hover:bg-raised/40">
              <td class="px-3 py-2.5 align-top">
                <div class="text-primary">{{ item.name }}</div>
                <div class="mt-0.5 text-[10px] text-muted">{{ item.country }} · {{ item.unit }} · {{ item.frequency }}</div>
              </td>
              <td class="px-3 py-2.5 font-mono tabular-nums text-primary">{{ item.latest_value ?? missing }}</td>
              <td class="px-3 py-2.5 font-mono tabular-nums text-secondary">{{ item.previous_value ?? missing }}</td>
              <td class="px-3 py-2.5 font-mono tabular-nums" :class="changeClass(item.change)">{{ displayChange(item.change) }}</td>
              <td class="px-3 py-2.5 font-mono tabular-nums text-secondary">{{ formatDate(item.reference_date) }}</td>
              <td class="px-3 py-2.5 font-mono tabular-nums text-secondary">{{ item.release_timestamp ? formatTimestampWIB(item.release_timestamp) : missing }}</td>
              <td class="px-3 py-2.5 text-secondary">{{ trendText(item.trend) }}</td>
              <td class="px-3 py-2.5">
                <span v-if="item.stale === true" class="text-down">! {{ tEconomic('economic.stale') }}</span>
                <span v-else-if="item.stale === false" class="text-up">+ {{ tEconomic('economic.current') }}</span>
                <span v-else class="text-muted">{{ missing }}</span>
              </td>
              <td class="px-3 py-2.5 text-secondary">
                <a v-if="item.source_url" :href="item.source_url" target="_blank" rel="noreferrer" class="underline decoration-border-strong underline-offset-2">{{ item.source_provider }}</a>
                <span v-else>{{ item.source_provider }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <template #footer>
        <div class="text-[10px] text-muted">{{ tEconomic('economic.disclaimer') }}</div>
      </template>
    </Panel>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import EmptyState from '@/components/common/EmptyState.vue';
import ErrorState from '@/components/common/ErrorState.vue';
import Panel from '@/components/common/Panel.vue';
import SkeletonBlock from '@/components/common/SkeletonBlock.vue';
import { useEconomicIndicators } from '@/composables/useEconomicIndicators';
import { tEconomic } from '@/i18n/economic';
import { formatDateOnlyUTC, formatDisplayTime } from '@/utils/time';

const { indicators, loading, error, load } = useEconomicIndicators();
const country = ref('all');
const category = ref('all');
const missing = tEconomic('economic.notEnoughData');

const countries = computed(() => [...new Set(indicators.value.map((item) => item.country))].sort());
const categories = computed(() => [...new Set(indicators.value.map((item) => item.category))].sort());
const filteredIndicators = computed(() => indicators.value.filter((item) =>
  (country.value === 'all' || item.country === country.value) &&
  (category.value === 'all' || item.category === category.value)
));

function formatDate(value: string | null): string {
  return value ? formatDateOnlyUTC(value) : missing;
}

function formatTimestampWIB(value: string): string {
  return formatDisplayTime(value);
}

function displayChange(value: string | null): string {
  if (value === null) return missing;
  const sign = value.startsWith('-') ? '' : '+';
  const glyph = value.startsWith('-') ? '▼' : value === '0' ? '−' : '▲';
  return `${glyph} ${sign}${value}`;
}

function changeClass(value: string | null): string {
  if (value === null || value === '0') return 'text-neutral';
  return value.startsWith('-') ? 'text-down' : 'text-up';
}

function trendText(value: string): string {
  if (value === 'rising') return `▲ ${tEconomic('economic.rising')}`;
  if (value === 'falling') return `▼ ${tEconomic('economic.falling')}`;
  return value === 'flat' ? `− ${tEconomic('economic.flat')}` : missing;
}
</script>
