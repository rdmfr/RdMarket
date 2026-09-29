<template>
  <div class="bg-panel border border-border rounded-xs overflow-hidden flex flex-col">
    <!-- Compact Panel Header Strip -->
    <div
      v-if="title || $slots.header || $slots.controls"
      class="px-3 py-2 border-b border-border flex items-center justify-between min-h-[30px] bg-panel select-none"
    >
      <div class="flex items-center gap-2">
        <span
          v-if="title"
          class="text-[11px] font-medium uppercase tracking-[0.04em] text-muted font-sans"
        >
          {{ title }}
        </span>
        <slot name="header" />
      </div>

      <div class="flex items-center gap-2">
        <slot name="controls" />
      </div>
    </div>

    <!-- Panel Body -->
    <div :class="['flex-1', noPadding ? '' : 'p-3']">
      <slot />
    </div>

    <!-- Optional Footer Strip -->
    <div
      v-if="$slots.footer"
      class="px-3 py-1.5 border-t border-border bg-raised/40 text-[11px] text-muted font-sans"
    >
      <slot name="footer" />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  title?: string;
  noPadding?: boolean;
}>();
</script>
