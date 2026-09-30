import { createRouter, createWebHistory } from 'vue-router';
import DashboardView from '@/views/DashboardView.vue';
import MarketDataView from '@/views/MarketDataView.vue';
import IndicatorsView from '@/views/IndicatorsView.vue';
import DataSourcesView from '@/views/DataSourcesView.vue';
import SettingsView from '@/views/SettingsView.vue';
import ForecastingView from '@/views/ForecastingView.vue';
import EconomyView from '@/views/EconomyView.vue';

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'dashboard',
      component: DashboardView,
    },
    {
      path: '/market-data',
      name: 'market-data',
      component: MarketDataView,
    },
    {
      path: '/indicators',
      name: 'indicators',
      component: IndicatorsView,
    },
    {
      path: '/forecast',
      name: 'forecast',
      component: ForecastingView,
    },
    {
      path: '/economy',
      name: 'economy',
      component: EconomyView,
    },
    {
      path: '/data-sources',
      name: 'data-sources',
      component: DataSourcesView,
    },
    {
      path: '/settings',
      name: 'settings',
      component: SettingsView,
    },
  ],
});

export default router;

