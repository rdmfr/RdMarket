<template>
  <div class="min-h-screen h-screen flex flex-col bg-base text-primary overflow-hidden">
    <!-- Top Navigation Bar -->
    <TopBar />

    <!-- Persistent Simulated Data Banner (when mock active) -->
    <SimulatedDataBanner :show="isSimulated" />

    <!-- Main Content Area: Sidebar + Scrollable Viewport -->
    <div class="flex-1 flex overflow-hidden">
      <SidebarRail />

      <main class="flex-1 overflow-y-auto overflow-x-hidden p-2 sm:p-3">
        <router-view />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import TopBar from './TopBar.vue';
import SidebarRail from './SidebarRail.vue';
import SimulatedDataBanner from '@/components/common/SimulatedDataBanner.vue';
import { useMarketData } from '@/composables/useMarketData';

// Initialize market data ingestion & refresh cycle at layout level
const { marketStore } = useMarketData();

const isSimulated = computed(() => {
  return (
    marketStore.dataSources.some((d) => d.provider.toLowerCase().includes('mock')) ||
    marketStore.currentRate?.source.toLowerCase().includes('mock') ||
    true // default true in dev/test when mock is active
  );
});
</script>
