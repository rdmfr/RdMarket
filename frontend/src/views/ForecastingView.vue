<template>
  <div class="max-w-[1920px] mx-auto space-y-3 pb-6">
    <Panel :title="t('forecasting.title')">
      <div class="flex flex-wrap gap-1 border-b border-border mb-3" role="tablist">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          type="button"
          role="tab"
          :aria-selected="activeTab === tab.id"
          :class="[
            'px-3 py-2 text-xs border-b-2 transition-colors cursor-pointer',
            activeTab === tab.id
              ? 'text-primary border-accent'
              : 'text-secondary border-transparent hover:text-primary',
          ]"
          @click="activeTab = tab.id"
        >
          {{ t(tab.label) }}
        </button>
      </div>

      <section v-if="activeTab === 'forecast'" role="tabpanel" class="space-y-3">
        <Panel :title="t('forecasting.forecastTitle')">
          <div class="flex flex-wrap items-end gap-3 mb-3">
            <label class="flex flex-col gap-1 text-[11px] text-muted">
              {{ t('forecasting.model') }}
              <select
                v-model="selectedModel"
                :disabled="modelsLoading || availableModels.length === 0"
                class="h-8 min-w-52 px-2 bg-raised text-primary border border-border rounded-xs text-xs focus-visible:outline-2 focus-visible:outline-accent"
              >
                <option value="">{{ t('forecasting.selectModel') }}</option>
                <option v-for="model in availableModels" :key="model.name" :value="model.name">
                  {{ model.name }} · {{ model.version }}
                </option>
              </select>
            </label>
            <label class="flex flex-col gap-1 text-[11px] text-muted">
              {{ t('forecasting.horizon') }}
              <select
                v-model="forecastHorizonText"
                class="h-8 w-36 px-2 bg-raised text-primary border border-border rounded-xs text-xs tabular-nums focus-visible:outline-2 focus-visible:outline-accent"
              >
                <option v-for="horizon in horizons" :key="horizon" :value="String(horizon)">{{ horizon }}</option>
              </select>
            </label>
            <button
              type="button"
              :disabled="!canRunForecast || launching"
              class="h-8 px-3 bg-raised hover:bg-border text-primary border border-border rounded-xs text-xs disabled:opacity-50 disabled:cursor-not-allowed"
              @click="launchForecast"
            >
              {{ launching ? t('forecasting.loading') : t('forecasting.launchForecast') }}
            </button>
            <button
              v-if="latest"
              type="button"
              class="h-8 px-3 bg-raised hover:bg-border text-secondary border border-border rounded-xs text-xs"
              @click="loadLatest"
            >
              {{ t('forecasting.retry') }}
            </button>
          </div>

          <div v-if="modelsLoading" class="space-y-2 py-2">
            <SkeletonBlock height="24px" />
            <SkeletonBlock height="280px" />
          </div>
          <ErrorState
            v-else-if="modelsError"
            :message="modelsError"
            :retry-label="t('forecasting.retry')"
            @retry="loadModels"
          />
          <ErrorState
            v-else-if="forecastError"
            :message="forecastError"
            :retry-label="t('forecasting.retry')"
            @retry="loadLatest"
          />
          <div v-else-if="!models.length" class="py-4 text-xs text-muted">
            {{ t('forecasting.noModels') }}
          </div>
          <EmptyState
            v-else-if="!availableModels.length"
            :message="t('forecasting.notInstalled')"
          />
          <div v-else-if="latestLoading" class="space-y-2 py-2">
            <SkeletonBlock height="280px" />
            <SkeletonBlock height="36px" />
          </div>
          <EmptyState
            v-else-if="!selectedModel || !forecastHorizon"
            :message="t('forecasting.selectModel')"
          />
          <EmptyState
            v-else-if="!latest || latest.forecasts.length === 0"
            :message="t('forecasting.noForecast')"
          />
          <div v-else class="space-y-3">
            <div class="border border-border rounded-xs bg-panel p-2">
              <div class="h-[340px] min-h-[260px]" :aria-label="t('forecasting.forecastChart')" role="img">
                <BaseChart :options="chartOptions" />
              </div>
            </div>

            <div class="overflow-x-auto border border-border rounded-xs">
              <table class="w-full min-w-[640px] text-xs">
                <caption class="px-3 py-2 text-left text-[11px] uppercase tracking-wide text-muted border-b border-border">
                  {{ t('forecasting.values') }}
                </caption>
                <thead class="bg-raised/60 text-[10px] uppercase text-muted">
                  <tr>
                    <th v-for="header in forecastHeaders" :key="header.key" class="px-2 py-2 text-right first:text-left font-medium">
                      {{ t(header.label) }}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="point in latest.forecasts" :key="`${point.step}-${point.target_timestamp}`" class="border-t border-border/70">
                    <td class="px-2 py-2 text-secondary tabular-nums">{{ formatTimestampWIB(point.target_timestamp) }}</td>
                    <td class="px-2 py-2 text-right text-secondary tabular-nums">{{ formatCount(point.step) }}</td>
                    <td class="px-2 py-2 text-right text-primary tabular-nums">{{ formatIDR(point.point) }}</td>
                    <td class="px-2 py-2 text-right text-secondary tabular-nums">{{ formatIDR(point.lower_80) }}</td>
                    <td class="px-2 py-2 text-right text-secondary tabular-nums">{{ formatIDR(point.upper_80) }}</td>
                    <td class="px-2 py-2 text-right text-secondary tabular-nums">{{ formatIDR(point.lower_95) }}</td>
                    <td class="px-2 py-2 text-right text-secondary tabular-nums">{{ formatIDR(point.upper_95) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>

            <Panel :title="t('forecasting.metadata')">
              <dl class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-x-4 gap-y-2 text-xs">
                <div v-for="item in metadataRows" :key="item.label" class="flex justify-between gap-3 border-b border-border/60 py-1">
                  <dt class="text-muted">{{ t(item.label) }}</dt>
                  <dd class="text-secondary text-right tabular-nums break-all">{{ item.value }}</dd>
                </div>
              </dl>
              <div class="mt-3 pt-2 border-t border-border flex flex-wrap gap-x-5 gap-y-1 text-xs">
                <span class="text-muted">{{ t('forecasting.skillScore') }}:
                  <span class="text-secondary tabular-nums">{{ formatForecastMetric(latest.baseline_comparison.skill_score_vs_naive) }}</span>
                </span>
                <span class="text-muted">{{ t('forecasting.beatsNaive') }}:
                  <span class="text-secondary">{{ latest.baseline_comparison.beats_naive ? t('forecasting.yes') : t('forecasting.no') }}</span>
                </span>
              </div>
            </Panel>
          </div>

          <p class="mt-3 text-[11px] text-muted text-center">{{ t('forecasting.disclaimer') }}</p>
        </Panel>
      </section>

      <section v-else-if="activeTab === 'backtests'" role="tabpanel" class="space-y-3">
        <Panel :title="t('forecasting.backtestTitle')">
          <div class="flex flex-wrap items-end gap-3 mb-3">
            <label class="flex flex-col gap-1 text-[11px] text-muted">
              {{ t('forecasting.horizon') }}
              <select
                v-model="backtestHorizonText"
                class="h-8 w-36 px-2 bg-raised text-primary border border-border rounded-xs text-xs tabular-nums focus-visible:outline-2 focus-visible:outline-accent"
              >
                <option v-for="horizon in horizons" :key="horizon" :value="String(horizon)">{{ horizon }}</option>
              </select>
            </label>
            <button
              type="button"
              :disabled="!backtestHorizon || selectedBacktestModels.length === 0 || launching"
              class="h-8 px-3 bg-raised hover:bg-border text-primary border border-border rounded-xs text-xs disabled:opacity-50 disabled:cursor-not-allowed"
              @click="launchBacktest"
            >
              {{ launching ? t('forecasting.loading') : t('forecasting.launch') }}
            </button>
            <button
              type="button"
              :disabled="!backtestHorizon || leaderboardLoading"
              class="h-8 px-3 bg-raised hover:bg-border text-secondary border border-border rounded-xs text-xs disabled:opacity-50"
              @click="loadLeaderboard"
            >
              {{ t('forecasting.retry') }}
            </button>
          </div>
          <div v-if="modelsError" class="mb-3 text-xs text-down">{{ modelsError }}</div>
          <div v-else-if="!modelsLoading && models.length" class="mb-3">
            <div class="text-[11px] text-muted mb-2">{{ t('forecasting.modelsToRun') }}</div>
            <div class="flex flex-wrap gap-x-4 gap-y-2">
              <label v-for="model in models" :key="model.name" class="inline-flex items-center gap-2 text-xs text-secondary">
                <input v-model="selectedBacktestModels" type="checkbox" :value="model.name" :disabled="!model.available" class="accent-[var(--accent)] disabled:opacity-50">
                {{ model.name }}<span v-if="model.stability === 'experimental'"> · {{ t('forecasting.experimental') }}</span>
                <span v-if="!model.available" class="text-muted"> · {{ model.unavailable_reason || t('forecasting.notInstalled') }}</span>
              </label>
            </div>
          </div>
          <p v-else-if="!modelsLoading" class="mb-3 text-xs text-muted">{{ t('forecasting.selectModels') }}</p>

          <ErrorState
            v-if="leaderboardError"
            :message="leaderboardError"
            :retry-label="t('forecasting.retry')"
            @retry="loadLeaderboard"
          />
          <div v-else-if="leaderboardLoading" class="space-y-2 py-2">
            <SkeletonBlock height="28px" />
            <SkeletonBlock height="120px" />
          </div>
          <template v-else-if="leaderboard">
            <div
              :class="[
                'mb-3 border rounded-xs px-3 py-2 text-xs',
                !leaderboard.has_evaluations || leaderboard.no_model_beats_naive
                  ? 'border-neutral/40 bg-raised/40 text-secondary'
                  : 'border-accent/40 bg-raised/40 text-primary',
              ]"
            >
              {{ !leaderboard.has_evaluations
                ? t('forecasting.noLeaderboard')
                : leaderboard.no_model_beats_naive
                  ? t('forecasting.noModelBeatsNaive')
                  : t('forecasting.modelBeatsNaive') }}
            </div>
            <div class="overflow-x-auto border border-border rounded-xs">
              <table class="w-full min-w-[1050px] text-xs">
                <thead class="bg-raised/60 text-[10px] uppercase text-muted">
                  <tr>
                    <th v-for="header in leaderboardHeaders" :key="header.key" class="px-2 py-2 text-right first:text-left font-medium">
                      {{ t(header.label) }}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="item in leaderboard.items" :key="`${item.model_name}-${item.horizon}`" class="border-t border-border/70">
                    <td class="px-2 py-2 text-primary">
                      {{ item.model_name }}
                      <span v-if="item.is_baseline" class="ml-1 px-1.5 py-0.5 border border-border rounded-xs text-[9px] text-muted">
                        {{ t('forecasting.isBaseline') }}
                      </span>
                      <span v-else-if="modelStability(item.model_name) === 'experimental'" class="ml-1 px-1.5 py-0.5 border border-border rounded-xs text-[9px] text-muted">
                        {{ t('forecasting.experimental') }}
                      </span>
                    </td>
                    <td class="px-2 py-2 text-right text-secondary tabular-nums">{{ formatForecastMetric(item.mae) }}</td>
                    <td class="px-2 py-2 text-right text-secondary tabular-nums">{{ formatForecastMetric(item.rmse) }}</td>
                    <td class="px-2 py-2 text-right text-secondary tabular-nums">{{ formatForecastMetric(item.mape) }}</td>
                    <td class="px-2 py-2 text-right text-secondary tabular-nums">{{ formatForecastMetric(item.mase) }}</td>
                    <td class="px-2 py-2 text-right text-secondary tabular-nums">{{ formatForecastMetric(item.directional_accuracy) }}</td>
                    <td class="px-2 py-2 text-right text-secondary tabular-nums">{{ formatForecastMetric(item.interval_coverage) }}</td>
                    <td class="px-2 py-2 text-right text-secondary tabular-nums">{{ formatForecastMetric(item.skill_score_vs_naive) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <EmptyState v-if="leaderboard.items.length === 0" :message="t('forecasting.noLeaderboard')" />
          </template>
          <EmptyState v-else-if="!backtestHorizon" :message="t('forecasting.noLeaderboard')" />
          <p class="mt-3 text-[11px] text-muted">{{ t('forecasting.naiveExplanation') }}</p>
        </Panel>

        <Panel :title="t('forecasting.jobs')">
          <ErrorState
            v-if="jobsError"
            :message="jobsError"
            :retry-label="t('forecasting.retry')"
            @retry="loadJobs"
          />
          <div v-else-if="jobsLoading" class="space-y-2">
            <SkeletonBlock height="32px" />
            <SkeletonBlock height="32px" />
          </div>
          <template v-else>
            <div v-if="pollingError" class="mb-2 flex items-center justify-between text-xs text-secondary">
              <span>{{ t('forecasting.jobError') }}</span>
              <button type="button" class="underline underline-offset-2" @click="retryPolling">
                {{ t('forecasting.retry') }}
              </button>
            </div>
            <div v-if="backtestJobs.length" class="overflow-x-auto">
              <table class="w-full min-w-[560px] text-xs">
                <thead class="text-[10px] uppercase text-muted">
                  <tr>
                    <th class="px-2 py-2 text-left font-medium">{{ t('forecasting.status') }}</th>
                    <th class="px-2 py-2 text-right font-medium">{{ t('forecasting.progress') }}</th>
                    <th class="px-2 py-2 text-right font-medium">{{ t('forecasting.created') }}</th>
                    <th class="px-2 py-2 text-left font-medium">{{ t('forecasting.error') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="job in backtestJobs" :key="job.id" class="border-t border-border/70">
                    <td class="px-2 py-2 text-primary">{{ job.status }}</td>
                    <td class="px-2 py-2 text-right text-secondary tabular-nums">
                      {{ formatCount(job.progress_done) }} / {{ formatCount(job.progress_total) }}
                    </td>
                    <td class="px-2 py-2 text-right text-secondary tabular-nums">{{ formatTimestampWIB(job.created_at) }}</td>
                    <td class="px-2 py-2 text-down">{{ job.error_message || t('forecasting.notAvailable') }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <EmptyState v-else :message="t('forecasting.noJobs')" />
          </template>
        </Panel>
      </section>

      <section v-else role="tabpanel">
        <Panel :title="t('forecasting.tabs.models')">
          <ErrorState
            v-if="modelsError"
            :message="modelsError"
            :retry-label="t('forecasting.retry')"
            @retry="loadModels"
          />
          <div v-else-if="modelsLoading" class="space-y-2">
            <SkeletonBlock height="52px" />
            <SkeletonBlock height="52px" />
          </div>
          <div v-else-if="models.length" class="overflow-x-auto">
            <table class="w-full min-w-[900px] text-xs">
              <thead class="bg-raised/60 text-[10px] uppercase text-muted">
                <tr>
                  <th v-for="header in modelHeaders" :key="header.key" class="px-2 py-2 text-left font-medium">
                    {{ t(header.label) }}
                  </th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="model in models" :key="model.name" class="border-t border-border/70">
                  <td class="px-2 py-2 text-primary">{{ model.label }}</td>
                  <td class="px-2 py-2 text-secondary">{{ model.name }}</td>
                  <td class="px-2 py-2 text-secondary tabular-nums">{{ model.version }}</td>
                  <td class="px-2 py-2 text-secondary">{{ model.stability === 'experimental' ? t('forecasting.experimental') : t('forecasting.stable') }}</td>
                  <td class="px-2 py-2 text-secondary">{{ model.available ? t('forecasting.available') : t('forecasting.notInstalled') }}</td>
                  <td class="px-2 py-2 text-right text-secondary tabular-nums">{{ formatCount(model.minimum_observations) }}</td>
                  <td class="px-2 py-2 text-secondary">{{ model.description }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <EmptyState v-else :message="t('forecasting.noModels')" />
        </Panel>
      </section>
    </Panel>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import Highcharts from 'highcharts';
import 'highcharts/highcharts-more';
import Panel from '@/components/common/Panel.vue';
import EmptyState from '@/components/common/EmptyState.vue';
import ErrorState from '@/components/common/ErrorState.vue';
import SkeletonBlock from '@/components/common/SkeletonBlock.vue';
import BaseChart from '@/components/charts/BaseChart.vue';
import { applyHighchartsTheme } from '@/components/charts/theme';
import { useSettingsStore } from '@/stores/settings';
import { useJobPolling } from '@/composables/useJobPolling';
import { forecastingApi, ForecastingApiError } from '@/services/forecastingApi';
import type { BacktestLeaderboard, ForecastJob, ForecastLatest, ForecastModel } from '@/schemas/forecasting';
import { t, type ForecastingMessageKey } from '@/i18n/forecasting';
import { formatCount, formatIDR, formatTimestampWIB } from '@/utils/formatters';
import { forecastOriginTimestamp, toForecastChartSeries } from '@/utils/forecastingChart';

type TabId = 'forecast' | 'backtests' | 'models';

const tabs: { id: TabId; label: ForecastingMessageKey }[] = [
  { id: 'forecast', label: 'forecasting.tabs.forecast' },
  { id: 'backtests', label: 'forecasting.tabs.backtests' },
  { id: 'models', label: 'forecasting.tabs.models' },
];
const activeTab = ref<TabId>('forecast');
const models = ref<ForecastModel[]>([]);
const horizons = [1, 5, 10, 21, 63];
const availableModels = computed(() => models.value.filter((model) => model.available));
const modelsLoading = ref(true);
const modelsError = ref('');
const selectedModel = ref('');
const forecastHorizonText = ref('1');
const backtestHorizonText = ref('1');
const selectedBacktestModels = ref<string[]>([]);
const latest = ref<ForecastLatest | null>(null);
const latestLoading = ref(false);
const forecastError = ref('');
const leaderboard = ref<BacktestLeaderboard | null>(null);
const leaderboardLoading = ref(false);
const leaderboardError = ref('');
const jobs = ref<ForecastJob[]>([]);
const jobsLoading = ref(true);
const jobsError = ref('');
const launching = ref(false);
const settingsStore = useSettingsStore();

const forecastHorizon = computed(() => positiveInteger(forecastHorizonText.value));
const backtestHorizon = computed(() => positiveInteger(backtestHorizonText.value));
const canRunForecast = computed(() => Boolean(selectedModel.value && forecastHorizon.value));
const backtestJobs = computed(() => jobs.value.filter((job) => job.job_type === 'backtest'));

function positiveInteger(value: string): number | null {
  if (!value.trim()) return null;
  const number = Number(value);
  return Number.isSafeInteger(number) && number > 0 ? number : null;
}

function formatForecastMetric(value: number | null): string {
  if (value === null || !Number.isFinite(value)) return t('forecasting.notAvailable');
  return new Intl.NumberFormat('id-ID', { maximumFractionDigits: 4 }).format(value);
}

async function loadModels() {
  modelsLoading.value = true;
  modelsError.value = '';
  try {
    models.value = await forecastingApi.getModels();
    if (!availableModels.value.some((model) => model.name === selectedModel.value)) {
      selectedModel.value = availableModels.value[0]?.name ?? '';
    }
    selectedBacktestModels.value = availableModels.value.map((model) => model.name);
  } catch (error) {
    modelsError.value = error instanceof ForecastingApiError ? error.message : t('forecasting.modelsError');
  } finally {
    modelsLoading.value = false;
  }
}

let latestRequest = 0;
async function loadLatest() {
  const model = selectedModel.value;
  const horizon = forecastHorizon.value;
  if (!model || !horizon) return;
  const request = ++latestRequest;
  latestLoading.value = true;
  latest.value = null;
  forecastError.value = '';
  try {
    const response = await forecastingApi.getLatest(model, horizon);
    if (request === latestRequest) latest.value = response;
  } catch (error) {
    if (request === latestRequest) {
      forecastError.value = error instanceof ForecastingApiError ? error.message : t('forecasting.forecastError');
    }
  } finally {
    if (request === latestRequest) latestLoading.value = false;
  }
}

let leaderboardRequest = 0;
async function loadLeaderboard() {
  const horizon = backtestHorizon.value;
  if (!horizon) {
    leaderboard.value = null;
    return;
  }
  const request = ++leaderboardRequest;
  leaderboardLoading.value = true;
  leaderboard.value = null;
  leaderboardError.value = '';
  try {
    const response = await forecastingApi.getLeaderboard(horizon);
    if (request === leaderboardRequest) leaderboard.value = response;
  } catch (error) {
    if (request === leaderboardRequest) {
      leaderboardError.value = error instanceof ForecastingApiError ? error.message : t('forecasting.leaderboardError');
    }
  } finally {
    if (request === leaderboardRequest) leaderboardLoading.value = false;
  }
}

async function loadJobs() {
  jobsLoading.value = true;
  jobsError.value = '';
  try {
    jobs.value = await forecastingApi.getJobs();
  } catch (error) {
    jobsError.value = error instanceof ForecastingApiError ? error.message : t('forecasting.jobsError');
  } finally {
    jobsLoading.value = false;
  }
}

async function refreshJob(id: string): Promise<ForecastJob> {
  const job = await forecastingApi.getJob(id);
  const terminal = ['completed', 'succeeded', 'failed', 'cancelled'].includes(job.status.toLowerCase());
  if (terminal && job.job_type === 'forecast') await loadLatest();
  if (terminal && job.job_type === 'backtest' && activeTab.value === 'backtests') await loadLeaderboard();
  return job;
}

const { pollingError, retryPolling } = useJobPolling(jobs, refreshJob);

async function launchJob(jobType: 'forecast' | 'backtest', modelNames: string[], horizon: number) {
  if (launching.value) return;
  launching.value = true;
  jobsError.value = '';
  try {
    const created = await forecastingApi.createJob({
      job_type: jobType,
      models: modelNames,
      horizon,
      params: {},
      force: false,
    });
    jobs.value.unshift(created);
  } catch (error) {
    jobsError.value = error instanceof ForecastingApiError ? error.message : t('forecasting.launchError');
  } finally {
    launching.value = false;
  }
}

function launchForecast() {
  const horizon = forecastHorizon.value;
  if (selectedModel.value && horizon) void launchJob('forecast', [selectedModel.value], horizon);
}

function launchBacktest() {
  const horizon = backtestHorizon.value;
  if (horizon && selectedBacktestModels.value.length) {
    void launchJob('backtest', [...selectedBacktestModels.value], horizon);
  }
}

watch([selectedModel, forecastHorizon], () => {
  if (selectedModel.value && forecastHorizon.value) void loadLatest();
  else {
    latestRequest += 1;
    latest.value = null;
    forecastError.value = '';
  }
});
watch([activeTab, backtestHorizon], ([tab, horizon]) => {
  if (tab === 'backtests' && horizon) void loadLeaderboard();
  else if (!horizon) {
    leaderboardRequest += 1;
    leaderboard.value = null;
    leaderboardError.value = '';
  }
});

const metadataRows = computed(() => {
  if (!latest.value) return [];
  const run = latest.value.run;
  return [
    { label: 'forecasting.model', value: run.model_name },
    { label: 'forecasting.version', value: run.model_version },
    { label: 'forecasting.origin', value: formatTimestampWIB(run.forecast_origin) },
    { label: 'forecasting.trainingStart', value: formatTimestampWIB(run.training_start) },
    { label: 'forecasting.trainingEnd', value: formatTimestampWIB(run.training_end) },
    { label: 'forecasting.trainingRows', value: formatCount(run.training_rows) },
    { label: 'forecasting.missing', value: formatCount(run.missing_observations) },
    { label: 'forecasting.fingerprint', value: run.data_fingerprint },
    { label: 'forecasting.generated', value: formatTimestampWIB(run.generated_at) },
    {
      label: 'forecasting.interval80',
      value: run.interval_levels.includes(0.8) ? '80%' : t('forecasting.notAvailable'),
    },
    {
      label: 'forecasting.interval95',
      value: run.interval_levels.includes(0.95) ? '95%' : t('forecasting.notAvailable'),
    },
  ];
});

const forecastHeaders: { key: string; label: ForecastingMessageKey }[] = [
  { key: 'timestamp', label: 'forecasting.timestamp' },
  { key: 'step', label: 'forecasting.step' },
  { key: 'point', label: 'forecasting.point' },
  { key: 'lower80', label: 'forecasting.lower80' },
  { key: 'upper80', label: 'forecasting.upper80' },
  { key: 'lower95', label: 'forecasting.lower95' },
  { key: 'upper95', label: 'forecasting.upper95' },
];
const leaderboardHeaders: { key: string; label: ForecastingMessageKey }[] = [
  { key: 'model', label: 'forecasting.modelName' },
  { key: 'mae', label: 'forecasting.mae' },
  { key: 'rmse', label: 'forecasting.rmse' },
  { key: 'mape', label: 'forecasting.mape' },
  { key: 'mase', label: 'forecasting.mase' },
  { key: 'directional', label: 'forecasting.directionalAccuracy' },
  { key: 'coverage', label: 'forecasting.intervalCoverage' },
  { key: 'skill', label: 'forecasting.skillScore' },
];
const modelHeaders: { key: string; label: ForecastingMessageKey }[] = [
  { key: 'label', label: 'forecasting.label' },
  { key: 'name', label: 'forecasting.modelName' },
  { key: 'version', label: 'forecasting.version' },
  { key: 'stability', label: 'forecasting.stability' },
  { key: 'available', label: 'forecasting.availability' },
  { key: 'minHistory', label: 'forecasting.minHistory' },
  { key: 'description', label: 'forecasting.description' },
];

function modelStability(name: string): string | undefined {
  return models.value.find((model) => model.name === name)?.stability;
}

const chartLabels = {
  history: t('forecasting.history'),
  forecast: t('forecasting.pointForecast'),
  interval80: t('forecasting.interval80'),
  interval95: t('forecasting.interval95'),
};
const chartOptions = computed<Highcharts.Options>(() => {
  applyHighchartsTheme(settingsStore.theme === 'dark');
  const origin = latest.value ? forecastOriginTimestamp(latest.value) : null;
  return {
    chart: { type: 'line', height: 340, spacing: [8, 8, 8, 8] },
    title: { text: undefined },
    legend: {
      enabled: true,
      align: 'left',
      verticalAlign: 'bottom',
      itemStyle: { color: 'var(--text-secondary)', fontSize: '10px', fontWeight: '400' },
      itemHoverStyle: { color: 'var(--text-primary)' },
    },
    xAxis: {
      type: 'datetime',
      plotLines: origin === null ? [] : [{
        value: origin,
        color: 'var(--accent)',
        width: 1,
        dashStyle: 'ShortDash',
        zIndex: 4,
        label: {
          text: t('forecasting.origin'),
          rotation: 0,
          y: 12,
          style: { color: 'var(--accent)', fontSize: '9px' },
        },
      }],
    },
    tooltip: { shared: true, valueDecimals: 2 },
    plotOptions: {
      arearange: { enableMouseTracking: true, lineWidth: 0 },
      series: { animation: false },
    },
    series: latest.value ? toForecastChartSeries(latest.value, chartLabels) : [],
  };
});

watch(() => settingsStore.theme, () => {
  if (latest.value) applyHighchartsTheme(settingsStore.theme === 'dark');
});

onMounted(() => {
  void loadModels();
  void loadJobs();
});
</script>
