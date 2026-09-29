<template>
  <div class="inline-flex items-center bg-raised/70 border border-border rounded-xs p-0.5 select-none">
    <button
      v-for="opt in options"
      :key="opt.value"
      type="button"
      @click="$emit('update:modelValue', opt.value)"
      :class="[
        'px-2 py-0.5 text-[11px] font-mono font-medium rounded-xs transition-colors duration-100 whitespace-nowrap cursor-pointer relative',
        modelValue === opt.value
          ? 'text-accent font-semibold after:content-[\'\'] after:absolute after:bottom-0 after:left-1.5 after:right-1.5 after:h-[2px] after:bg-accent'
          : 'text-secondary hover:text-primary hover:bg-panel/50',
      ]"
    >
      {{ opt.label }}
    </button>
  </div>
</template>

<script setup lang="ts">
export interface SegmentOption {
  label: string;
  value: string;
}

defineProps<{
  modelValue: string;
  options: SegmentOption[];
}>();

defineEmits<{
  (e: 'update:modelValue', val: string): void;
}>();
</script>
