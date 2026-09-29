<template>
  <header class="h-10 px-4 bg-panel border-b border-border flex items-center justify-between select-none z-20">
    <!-- Zone 1: Brand & Current Instrument -->
    <div class="flex items-center gap-3">
      <router-link to="/" class="flex items-center gap-2 text-primary hover:text-accent transition-colors">
        <span class="w-2.5 h-2.5 bg-accent rounded-[1px] shrink-0" />
        <span class="font-sans font-semibold tracking-tight text-xs sm:text-sm whitespace-nowrap">
          RdMarket Intelligence
        </span>
      </router-link>

      <div class="h-3 w-[1px] bg-border-strong hidden sm:block" />

      <!-- Instrument Indicator -->
      <div class="hidden sm:flex items-center gap-1.5 px-2 py-0.5 bg-raised border border-border rounded-xs">
        <span class="text-[11px] font-mono font-semibold text-primary">USD/IDR</span>
        <span class="text-[10px] text-muted font-sans">Spot</span>
      </div>
    </div>

    <!-- Zone 2: Monospace Timestamp & Live Status -->
    <div class="hidden md:flex items-center gap-4 text-xs font-mono">
      <div class="flex items-center gap-1.5 text-secondary">
        <span class="text-muted font-sans text-[11px]">System Time:</span>
        <span class="tabular-nums">{{ currentClock }}</span>
      </div>

      <div class="h-3 w-[1px] bg-border-strong" />

      <div class="flex items-center gap-1">
        <span class="text-muted font-sans text-[11px]">Feed:</span>
        <StatusBadge :status="marketStore.marketStatus" />
      </div>
    </div>

    <!-- Zone 3: Actions & Controls -->
    <div class="flex items-center gap-2">
      <!-- Manual Refresh Button -->
      <button
        type="button"
        @click="onRefresh"
        title="Refresh market data"
        class="h-7 px-2.5 bg-raised hover:bg-border text-secondary hover:text-primary border border-border rounded-xs text-xs font-mono flex items-center gap-1.5 cursor-pointer transition-colors"
      >
        <span :class="['text-[11px] font-mono leading-none', isRefreshing ? 'animate-spin' : '']">⟳</span>
        <span class="hidden sm:inline text-[11px]">Refresh</span>
      </button>

      <!-- Theme Toggle -->
      <button
        type="button"
        @click="settingsStore.toggleTheme"
        :title="settingsStore.theme === 'dark' ? 'Switch to Light Theme' : 'Switch to Dark Theme'"
        class="h-7 px-2 bg-raised hover:bg-border text-secondary hover:text-primary border border-border rounded-xs text-xs font-mono cursor-pointer transition-colors"
      >
        <span class="text-[11px] font-sans">{{ settingsStore.theme === 'dark' ? '☀ Light' : '☾ Dark' }}</span>
      </button>

      <!-- Settings Link -->
      <router-link
        to="/settings"
        title="Settings"
        class="h-7 px-2 bg-raised hover:bg-border text-secondary hover:text-primary border border-border rounded-xs text-xs flex items-center justify-center cursor-pointer transition-colors"
      >
        <span class="text-[11px] font-mono">⚙</span>
      </router-link>
    </div>
  </header>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue';
import StatusBadge from '@/components/common/StatusBadge.vue';
import { useMarketStore } from '@/stores/market';
import { useSettingsStore } from '@/stores/settings';

const marketStore = useMarketStore();
const settingsStore = useSettingsStore();

const isRefreshing = ref(false);
const currentClock = ref('');

let clockTimer: number | null = null;

function updateClock() {
  const d = new Date();
  const options: Intl.DateTimeFormatOptions = {
    timeZone: 'Asia/Jakarta',
    day: '2-digit',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  };
  currentClock.value = `${new Intl.DateTimeFormat('id-ID', options).format(d)} WIB`;
}

async function onRefresh() {
  isRefreshing.value = true;
  await marketStore.fetchAll();
  setTimeout(() => {
    isRefreshing.value = false;
  }, 300);
}

onMounted(() => {
  updateClock();
  clockTimer = window.setInterval(updateClock, 1000);
});

onUnmounted(() => {
  if (clockTimer) clearInterval(clockTimer);
});
</script>
