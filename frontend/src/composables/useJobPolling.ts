import { onBeforeUnmount, onMounted, ref, watch, type Ref } from 'vue';
import type { ForecastJob } from '@/schemas/forecasting';

const ACTIVE_STATUSES = new Set(['queued', 'pending', 'running', 'in_progress']);

export function isForecastJobActive(job: Pick<ForecastJob, 'status'>): boolean {
  return ACTIVE_STATUSES.has(job.status.toLowerCase());
}

export function useJobPolling(
  jobs: Ref<ForecastJob[]>,
  refreshJob: (id: string) => Promise<ForecastJob>,
  intervalMs = 2000
) {
  const pollingError = ref(false);
  let timer: ReturnType<typeof setTimeout> | null = null;
  let inFlight = false;
  let disposed = false;

  function clearTimer() {
    if (timer !== null) {
      clearTimeout(timer);
      timer = null;
    }
  }

  function schedule(delay = intervalMs) {
    clearTimer();
    if (
      disposed ||
      document.visibilityState === 'hidden' ||
      inFlight ||
      !jobs.value.some(isForecastJobActive)
    ) return;

    timer = setTimeout(() => {
      timer = null;
      void poll();
    }, delay);
  }

  async function poll() {
    if (disposed || inFlight || document.visibilityState === 'hidden') return;
    inFlight = true;
    const activeJobs = jobs.value.filter(isForecastJobActive);
    try {
      for (const job of activeJobs) {
        const updated = await refreshJob(job.id);
        const index = jobs.value.findIndex((candidate) => candidate.id === updated.id);
        if (index >= 0) jobs.value.splice(index, 1, updated);
      }
      pollingError.value = false;
    } catch {
      pollingError.value = true;
    } finally {
      inFlight = false;
      schedule();
    }
  }

  function onVisibilityChange() {
    if (document.visibilityState === 'hidden') clearTimer();
    else schedule(0);
  }

  watch(jobs, () => schedule(), { deep: true });
  onMounted(() => {
    document.addEventListener('visibilitychange', onVisibilityChange);
    schedule();
  });
  onBeforeUnmount(() => {
    disposed = true;
    clearTimer();
    document.removeEventListener('visibilitychange', onVisibilityChange);
  });

  return { pollingError, retryPolling: () => schedule(0) };
}
