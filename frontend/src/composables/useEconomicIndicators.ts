import { onMounted, ref } from 'vue';
import { economicApi } from '@/services/economicApi';
import type { EconomicIndicator } from '@/schemas/economic';

export function useEconomicIndicators() {
  const indicators = ref<EconomicIndicator[]>([]);
  const loading = ref(false);
  const error = ref(false);

  async function load() {
    loading.value = true;
    error.value = false;
    try {
      indicators.value = await economicApi.getIndicators();
    } catch {
      error.value = true;
    } finally {
      loading.value = false;
    }
  }

  onMounted(load);

  return { indicators, loading, error, load };
}