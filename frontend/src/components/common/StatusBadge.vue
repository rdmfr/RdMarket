<template>
  <div class="inline-flex items-center gap-1.5 px-1.5 py-0.5 text-[11px] font-mono tracking-wider">
    <span :class="['w-1.5 h-1.5 rounded-full shrink-0', dotClass]" />
    <span :class="['font-medium uppercase', textClass]">{{ status }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{
  status: 'LIVE' | 'DELAYED' | 'STALE' | 'OFFLINE' | string;
}>();

const dotClass = computed(() => {
  switch (props.status) {
    case 'LIVE':
      return 'bg-up';
    case 'DELAYED':
      return 'bg-accent';
    case 'STALE':
      return 'bg-neutral';
    case 'OFFLINE':
      return 'bg-down';
    default:
      return 'bg-neutral';
  }
});

const textClass = computed(() => {
  switch (props.status) {
    case 'LIVE':
      return 'text-up';
    case 'DELAYED':
      return 'text-accent';
    case 'STALE':
      return 'text-secondary';
    case 'OFFLINE':
      return 'text-down';
    default:
      return 'text-muted';
  }
});
</script>
