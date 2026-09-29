<template>
  <aside
    :class="[
      'bg-panel border-r border-border transition-all duration-150 ease-out z-20 flex flex-col justify-between select-none shrink-0',
      isExpanded ? 'w-48' : 'w-12',
    ]"
    @mouseenter="isExpanded = true"
    @mouseleave="isExpanded = false"
  >
    <!-- Top Nav Items -->
    <nav class="py-2 flex flex-col gap-0.5">
      <router-link
        v-for="item in navItems"
        :key="item.path"
        :to="item.path"
        :title="item.label"
        :class="[
          'relative flex items-center h-10 px-3 transition-colors duration-100 group cursor-pointer',
          $route.path === item.path
            ? 'text-primary bg-raised/60'
            : 'text-secondary hover:text-primary hover:bg-raised/30',
        ]"
      >
        <!-- 2px left accent bar on active item -->
        <span
          v-if="$route.path === item.path"
          class="absolute left-0 top-1 bottom-1 w-[2px] bg-accent"
        />

        <!-- Icon glyph (Financial Bloomberg style, 16px) -->
        <span class="w-6 flex items-center justify-center text-xs font-mono shrink-0">
          {{ item.icon }}
        </span>

        <!-- Nav Label -->
        <span
          :class="[
            'text-xs font-sans whitespace-nowrap overflow-hidden transition-opacity duration-100 ml-2.5',
            isExpanded ? 'opacity-100' : 'opacity-0 pointer-events-none',
            $route.path === item.path ? 'font-medium text-primary' : 'text-secondary',
          ]"
        >
          {{ item.label }}
        </span>
      </router-link>
    </nav>

    <!-- Bottom Rail Controls / Version info -->
    <div class="p-2 border-t border-border flex flex-col gap-2">
      <div
        :class="[
          'text-[10px] font-mono text-muted tracking-wider overflow-hidden transition-opacity duration-100 px-1',
          isExpanded ? 'opacity-100' : 'opacity-0',
        ]"
      >
        <div>PHASE 1: MONITORING</div>
        <div class="text-[9px] text-muted/80">v1.0.0-PROD</div>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { ref } from 'vue';

const isExpanded = ref(false);

const navItems = [
  { label: 'Dashboard', path: '/', icon: '⊞' },
  { label: 'Market Data', path: '/market-data', icon: '☰' },
  { label: 'Indicators', path: '/indicators', icon: '📈' },
  { label: 'Data Sources', path: '/data-sources', icon: '⚏' },
  { label: 'Settings', path: '/settings', icon: '⚙' },
];
</script>
