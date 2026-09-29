<template>
  <div class="flex items-center justify-between py-1.5 px-2 border-b border-border/60 hover:bg-raised/50 transition-colors duration-100 text-xs">
    <div class="flex items-center gap-1.5 text-secondary font-sans truncate mr-2">
      <span>{{ label }}</span>
      <span v-if="hint" :title="hint" class="cursor-help text-muted text-[10px]">ℹ</span>
    </div>

    <div class="font-mono tabular-nums text-right shrink-0">
      <slot>
        <span v-if="value !== undefined && value !== null && value !== ''" class="text-primary font-medium">
          {{ value }}
        </span>
        <span v-else class="text-muted cursor-help" title="Not enough data">
          —
        </span>
      </slot>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  label: string;
  value?: string | number | null;
  hint?: string;
}>();
</script>
