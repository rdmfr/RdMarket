<template>
  <div class="min-h-screen h-screen flex flex-col bg-base text-primary overflow-hidden">
    <!-- Top Navigation Bar -->
    <TopBar />

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
import TopBar from './TopBar.vue';
import SidebarRail from './SidebarRail.vue';
import { useMarketData } from '@/composables/useMarketData';

// Initialize market data ingestion & refresh cycle at layout level
useMarketData();
</script>
