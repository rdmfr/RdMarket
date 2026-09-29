<template>
  <div ref="chartContainer" class="w-full h-full min-h-[200px]" />
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch } from 'vue';
import Highcharts from 'highcharts';

const props = defineProps<{
  options: Highcharts.Options;
}>();

const chartContainer = ref<HTMLElement | null>(null);
let chartInstance: Highcharts.Chart | null = null;

onMounted(() => {
  if (chartContainer.value) {
    chartInstance = Highcharts.chart(chartContainer.value, props.options);
  }
});

watch(
  () => props.options,
  (newOptions) => {
    if (chartInstance) {
      chartInstance.update(newOptions, true, true);
    }
  },
  { deep: true }
);

onBeforeUnmount(() => {
  if (chartInstance) {
    chartInstance.destroy();
    chartInstance = null;
  }
});
</script>
