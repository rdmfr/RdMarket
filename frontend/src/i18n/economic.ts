const messages = {
  'economic.nav': 'Economy',
  'economic.phaseStatus': 'PHASE 3: IN PROGRESS',
  'economic.title': 'Economic Indicators',
  'economic.overview': 'Indicator overview',
  'economic.country': 'Country',
  'economic.category': 'Category',
  'economic.allCountries': 'All countries',
  'economic.allCategories': 'All categories',
  'economic.indicator': 'Indicator',
  'economic.latest': 'Latest',
  'economic.previous': 'Previous',
  'economic.change': 'Change',
  'economic.referencePeriod': 'Reference period',
  'economic.released': 'Released',
  'economic.trend': 'Trend',
  'economic.rising': 'Rising',
  'economic.falling': 'Falling',
  'economic.flat': 'Flat',
  'economic.status': 'Data status',
  'economic.source': 'Source',
  'economic.stale': 'Stale',
  'economic.current': 'Current',
  'economic.notEnoughData': 'Not enough data',
  'economic.noIndicators': 'No economic indicators are available.',
  'economic.loadError': 'Unable to retrieve economic indicators.',
  'economic.retry': 'Retry',
  'economic.refresh': 'Refresh',
  'economic.yes': 'Yes',
  'economic.no': 'No',
  'economic.disclaimer': 'Stored economic observations. Publication timing may use a configured lag when the source does not provide a release timestamp.',
} as const;

export type EconomicMessageKey = keyof typeof messages;

export function tEconomic(key: EconomicMessageKey): string {
  return messages[key];
}
