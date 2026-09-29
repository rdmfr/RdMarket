<template>
  <div class="max-w-[1920px] mx-auto space-y-3 pb-6">
    <Panel title="Historical USD/IDR Ledger & Market Observations">
      <template #controls>
        <div class="flex items-center gap-2">
          <!-- Time Range Selector -->
          <SegmentedControl
            :modelValue="marketStore.selectedRange"
            :options="rangeOptions"
            @update:modelValue="onRangeChange"
          />

          <!-- Export CSV Button -->
          <button
            type="button"
            @click="exportCSV"
            class="px-2.5 py-1 bg-raised hover:bg-border text-primary border border-border rounded-xs text-[11px] font-mono cursor-pointer transition-colors"
          >
            Export CSV
          </button>
        </div>
      </template>

      <!-- Search & Meta Strip -->
      <div class="flex flex-wrap items-center justify-between gap-3 mb-3 pb-2 border-b border-border/60 text-xs">
        <div class="flex items-center gap-2">
          <span class="text-muted font-mono text-[11px]">Filter:</span>
          <input
            v-model="searchQuery"
            type="text"
            placeholder="Search date or rate..."
            class="bg-raised border border-border px-2 py-1 rounded-xs text-xs font-mono text-primary placeholder:text-muted focus:outline-none focus:border-accent w-48"
          />
        </div>

        <div class="text-muted font-mono text-[11px] tabular-nums">
          Showing {{ filteredPoints.length }} of {{ marketStore.historicalData?.total_points || 0 }} records
        </div>
      </div>

      <!-- Loading / Error / Table Content -->
      <div v-if="marketStore.loading.history" class="space-y-1.5 py-2">
        <SkeletonBlock v-for="i in 10" :key="i" width="100%" height="24px" />
      </div>

      <ErrorState
        v-else-if="marketStore.errors.history"
        :message="marketStore.errors.history"
        @retry="marketStore.fetchHistory"
      />

      <div v-else class="overflow-x-auto">
        <table class="w-full text-left border-collapse text-xs font-mono">
          <thead>
            <tr class="bg-raised/70 border-b border-border text-muted text-[11px] uppercase tracking-wider">
              <th class="py-2 px-3">Date (WIB)</th>
              <th class="py-2 px-3">Pair</th>
              <th class="py-2 px-3 text-right">Exchange Rate</th>
              <th class="py-2 px-3 text-right">Change</th>
              <th class="py-2 px-3 text-right">Change %</th>
              <th class="py-2 px-3 text-right">Source</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border/40">
            <tr
              v-for="(row, idx) in paginatedPoints"
              :key="row.timestamp"
              class="hover:bg-raised/40 transition-colors"
            >
              <td class="py-2 px-3 text-primary tabular-nums">
                {{ formatTimestampWIB(row.timestamp) }}
              </td>
              <td class="py-2 px-3 text-secondary">
                USD/IDR
              </td>
              <td class="py-2 px-3 text-right tabular-nums text-primary font-medium">
                <span v-if="row.rate !== null">Rp {{ formatIDR(row.rate) }}</span>
                <span v-else class="text-muted cursor-help" title="Not enough data">—</span>
              </td>
              <td class="py-2 px-3 text-right tabular-nums">
                <ValueChange :value="calculateRowChange(idx)" />
              </td>
              <td class="py-2 px-3 text-right tabular-nums">
                <ValueChange :value="calculateRowChangePercent(idx)" :isPercent="true" />
              </td>
              <td class="py-2 px-3 text-right text-muted text-[11px]">
                {{ marketStore.currentRate?.source || 'ECB / Verified Feed' }}
              </td>
            </tr>

            <tr v-if="filteredPoints.length === 0">
              <td colspan="6" class="py-8 text-center text-muted font-sans text-xs">
                No exchange rate records match the search filter.
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Pagination Strip -->
      <template #footer>
        <div class="flex items-center justify-between text-xs font-mono py-1">
          <span class="text-muted text-[11px]">
            Page {{ currentPage }} of {{ totalPages || 1 }}
          </span>

          <div class="flex items-center gap-1">
            <button
              type="button"
              :disabled="currentPage <= 1"
              @click="currentPage--"
              class="px-2 py-0.5 bg-panel border border-border rounded-xs text-[11px] disabled:opacity-40 disabled:cursor-not-allowed hover:bg-raised text-secondary hover:text-primary cursor-pointer"
            >
              Previous
            </button>
            <button
              type="button"
              :disabled="currentPage >= totalPages"
              @click="currentPage++"
              class="px-2 py-0.5 bg-panel border border-border rounded-xs text-[11px] disabled:opacity-40 disabled:cursor-not-allowed hover:bg-raised text-secondary hover:text-primary cursor-pointer"
            >
              Next
            </button>
          </div>
        </div>
      </template>
    </Panel>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import Panel from '@/components/common/Panel.vue';
import SegmentedControl, { type SegmentOption } from '@/components/common/SegmentedControl.vue';
import ValueChange from '@/components/common/ValueChange.vue';
import SkeletonBlock from '@/components/common/SkeletonBlock.vue';
import ErrorState from '@/components/common/ErrorState.vue';
import { useMarketStore } from '@/stores/market';
import { formatIDR, formatTimestampWIB } from '@/utils/formatters';
import type { TimeRange } from '@/types/market';

const marketStore = useMarketStore();
const searchQuery = ref('');
const currentPage = ref(1);
const pageSize = 20;

const rangeOptions: SegmentOption[] = [
  { label: '7D', value: '7D' },
  { label: '1M', value: '1M' },
  { label: '3M', value: '3M' },
  { label: '6M', value: '6M' },
  { label: '1Y', value: '1Y' },
  { label: '5Y', value: '5Y' },
];

function onRangeChange(val: string) {
  currentPage.value = 1;
  marketStore.setRange(val as TimeRange);
}

// Sorted newest first
const sortedPoints = computed(() => {
  const pts = marketStore.historicalData?.points || [];
  return [...pts].reverse();
});

const filteredPoints = computed(() => {
  const query = searchQuery.value.trim().toLowerCase();
  if (!query) return sortedPoints.value;

  return sortedPoints.value.filter((p) => {
    const dateStr = formatTimestampWIB(p.timestamp).toLowerCase();
    const rateStr = p.rate ? p.rate.toString() : '';
    return dateStr.includes(query) || rateStr.includes(query);
  });
});

const totalPages = computed(() => Math.ceil(filteredPoints.value.length / pageSize));

const paginatedPoints = computed(() => {
  const start = (currentPage.value - 1) * pageSize;
  return filteredPoints.value.slice(start, start + pageSize);
});

function calculateRowChange(paginatedIdx: number): number | null {
  const current = paginatedPoints.value[paginatedIdx];
  const next = paginatedPoints.value[paginatedIdx + 1]; // next is older in reverse list
  if (!current || !next || current.rate === null || next.rate === null) return null;
  return current.rate - next.rate;
}

function calculateRowChangePercent(paginatedIdx: number): number | null {
  const current = paginatedPoints.value[paginatedIdx];
  const next = paginatedPoints.value[paginatedIdx + 1];
  if (!current || !next || current.rate === null || next.rate === null || next.rate === 0) return null;
  return ((current.rate - next.rate) / next.rate) * 100;
}

function exportCSV() {
  const rows = [
    ['Timestamp', 'Date WIB', 'Currency Pair', 'Rate', 'Source'],
    ...sortedPoints.value.map((p) => [
      p.timestamp,
      `"${formatTimestampWIB(p.timestamp)}"`,
      'USD/IDR',
      p.rate ?? '',
      marketStore.currentRate?.source || 'Exchange Feed',
    ]),
  ];

  const csvContent = 'data:text/csv;charset=utf-8,' + rows.map((e) => e.join(',')).join('\n');
  const encodedUri = encodeURI(csvContent);
  const link = document.createElement('a');
  link.setAttribute('href', encodedUri);
  link.setAttribute('download', `USD_IDR_history_${marketStore.selectedRange}.csv`);
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
}
</script>
