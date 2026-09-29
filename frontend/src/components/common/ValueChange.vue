<template>
  <span
    v-if="value !== null && value !== undefined && !isNaN(value)"
    :class="[
      'inline-flex items-center gap-1 font-mono tabular-nums text-xs font-medium',
      directionClass,
    ]"
  >
    <span aria-hidden="true" class="text-[9px] leading-none">{{ glyph }}</span>
    <span>{{ formattedText }}</span>
  </span>
  <span v-else class="text-muted font-mono text-xs cursor-help" title="Not enough data">
    —
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { formatChange, formatPercent } from '@/utils/formatters';

const props = withDefaults(
  defineProps<{
    value: number | null | undefined;
    isPercent?: boolean;
    whole?: boolean;
    showGlyph?: boolean;
  }>(),
  {
    isPercent: false,
    whole: false,
    showGlyph: true,
  }
);

const isPositive = computed(() => (props.value ?? 0) > 0);
const isNegative = computed(() => (props.value ?? 0) < 0);

const directionClass = computed(() => {
  if (isPositive.value) return 'text-up';
  if (isNegative.value) return 'text-down';
  return 'text-neutral';
});

const glyph = computed(() => {
  if (!props.showGlyph) return '';
  if (isPositive.value) return '▲';
  if (isNegative.value) return '▼';
  return '•';
});

const formattedText = computed(() => {
  if (props.isPercent) {
    return formatPercent(props.value);
  }
  return formatChange(props.value, props.whole);
});
</script>
