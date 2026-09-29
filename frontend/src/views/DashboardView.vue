<template>
  <div class="space-y-2 sm:space-y-3 max-w-[1920px] mx-auto pb-4">
    <!-- Top Row: Chart (~70%) + Summary & Condition (~30%) -->
    <div class="grid grid-cols-1 xl:grid-cols-12 gap-2 sm:gap-3 items-stretch">
      <!-- Section 2: Historical Chart (Tall, min 420px, left ~70%) -->
      <div class="xl:col-span-8 flex flex-col min-h-[460px]">
        <UsdIdrChart />
      </div>

      <!-- Right Column: Summary Card & Condition Panel (~30%) -->
      <div class="xl:col-span-4 flex flex-col gap-2 sm:gap-3">
        <!-- Section 1: Market Summary -->
        <MarketSummaryCard />

        <!-- Section 4: Market Condition -->
        <MarketConditionPanel />
      </div>
    </div>

    <!-- Middle Row: Statistics Table & Basic Indicators Side by Side -->
    <div class="grid grid-cols-1 lg:grid-cols-12 gap-2 sm:gap-3">
      <!-- Section 3: Market Statistics (Dense Key/Value table) -->
      <div class="lg:col-span-7 flex flex-col">
        <MarketStatisticsTable />
      </div>

      <!-- Section 4: Basic Indicators -->
      <div class="lg:col-span-5 flex flex-col">
        <IndicatorsPanel />
      </div>
    </div>

    <!-- Bottom Row: Section 5: Data Source Information -->
    <div>
      <DataSourceStrip />
    </div>
  </div>
</template>

<script setup lang="ts">
import UsdIdrChart from '@/components/charts/UsdIdrChart.vue';
import MarketSummaryCard from '@/components/market/MarketSummaryCard.vue';
import MarketConditionPanel from '@/components/market/MarketConditionPanel.vue';
import MarketStatisticsTable from '@/components/market/MarketStatisticsTable.vue';
import IndicatorsPanel from '@/components/market/IndicatorsPanel.vue';
import DataSourceStrip from '@/components/market/DataSourceStrip.vue';
</script>
