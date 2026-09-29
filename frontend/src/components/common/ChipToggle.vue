<template>
  <button
    type="button"
    @click="$emit('update:modelValue', !modelValue)"
    :class="[
      'inline-flex items-center gap-1.5 px-2 py-0.5 text-[11px] font-mono border rounded-xs select-none transition-colors duration-100 cursor-pointer',
      modelValue
        ? 'border-border-strong bg-raised text-primary'
        : 'border-border/60 bg-transparent text-muted hover:text-secondary hover:border-border',
    ]"
  >
    <!-- Color Swatch -->
    <span
      class="w-2 h-2 rounded-[1px] shrink-0"
      :style="{ backgroundColor: swatchColor }"
      :class="{ 'opacity-40': !modelValue }"
    />
    <span :class="{ 'line-through text-muted': !modelValue }">{{ label }}</span>
    <span class="text-[9px] text-muted ml-0.5 font-sans">{{ modelValue ? '✓' : '' }}</span>
  </button>
</template>

<script setup lang="ts">
defineProps<{
  modelValue: boolean;
  label: string;
  swatchColor: string;
}>();

defineEmits<{
  (e: 'update:modelValue', val: boolean): void;
}>();
</script>
