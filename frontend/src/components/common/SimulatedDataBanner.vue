<template>
  <div
    v-if="show"
    role="status"
    aria-live="polite"
    class="bg-amber-950/80 border-b border-amber-600/50 text-amber-300 px-3 py-1 text-[11px] font-mono tracking-wider flex items-center justify-between select-none"
  >
    <div class="flex items-center gap-2">
      <span class="inline-block w-2 h-2 rounded-xs bg-amber-400"></span>
      <span class="font-bold">SIMULATED DATA</span>
      <span class="text-amber-300/80 text-[10px] hidden sm:inline">— Active provider is MockProvider. Rates and indicators are synthetic for development only.</span>
    </div>
    <span class="text-[10px] uppercase text-amber-400/90 font-sans">NON-PRODUCTION</span>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  show?: boolean;
}>();
</script>
