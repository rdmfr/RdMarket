<template>
  <div class="p-6 flex flex-col items-center justify-center text-center">
    <p class="text-xs text-muted font-sans mb-3">{{ message || 'No data available' }}</p>
    <button
      v-if="actionLabel"
      type="button"
      class="px-2.5 py-1 text-xs font-sans font-medium bg-raised border border-border rounded-xs hover:border-text-secondary text-text cursor-pointer transition-colors"
      @click="$emit('action')"
    >
      {{ actionLabel }}
    </button>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  message?: string;
  actionLabel?: string;
}>();

defineEmits<{
  (e: 'action'): void;
}>();
</script>
