export interface PmMetric {
  key: string;
  tech: 'LTE' | 'GNB' | 'GSM';
  metricPath: string;
  reportKey: string;
  metricType: 'counter' | 'kpi';
  statisType: string;
  unit: string;
  cnName: string;
  profile: string;
  sourceFile: string;
}

interface PmMetricCatalog {
  stats: Record<PmMetric['tech'], { total: number; counter: number; kpi: number }>;
  metrics: PmMetric[];
}

const pmMetricCatalogUrl = new URL('./pmMetricCatalog.json', import.meta.url).href;

export const pmMetricStats: PmMetricCatalog['stats'] = {
  LTE: { total: 1408, counter: 1333, kpi: 75 },
  GNB: { total: 282, counter: 217, kpi: 65 },
  GSM: { total: 73, counter: 45, kpi: 28 },
};

export async function loadPmMetrics(): Promise<PmMetric[]> {
  const response = await fetch(pmMetricCatalogUrl);
  if (!response.ok) {
    throw new Error(`Failed to load PM metric catalog: ${response.status}`);
  }
  const catalog = (await response.json()) as PmMetricCatalog;
  return catalog.metrics;
}
