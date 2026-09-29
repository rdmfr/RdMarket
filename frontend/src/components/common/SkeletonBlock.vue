<template>
  <div
    :class="[
      'bg-raised/60 rounded-xs animate-pulse opacity-70',
      customClass,
    ]"
    :style="{
      width: width || '100%',
      height: height || '16px',
    }"
  />
</template>

<script setup lang="ts">
defineProps<{
  width?: string;
  height?: string;
  customClass?: string;
}>();
</script>
