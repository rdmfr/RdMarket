<template>
  <div class="p-4 border border-down/30 bg-down/5 rounded-xs flex flex-col items-start gap-2 text-xs">
    <div class="flex items-center gap-2">
      <span class="text-down font-mono font-bold">!</span>
      <span class="text-primary font-medium">{{ message || 'Unable to retrieve current market data.' }}</span>
    </div>

    <p v-if="cause" class="text-secondary text-[11px] leading-relaxed">
      {{ cause }}
    </p>

    <button
      v-if="showRetry"
      type="button"
      @click="$emit('retry')"
      class="mt-1 px-3 py-1 bg-raised hover:bg-border text-primary border border-border rounded-xs text-[11px] font-mono cursor-pointer transition-colors"
    >
      Retry
    </button>
  </div>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    message?: string;
    cause?: string;
    showRetry?: boolean;
  }>(),
  {
    message: 'Unable to retrieve current market data.',
    showRetry: true,
  }
);

defineEmits<{
  (e: 'retry'): void;
}>();
</script>
