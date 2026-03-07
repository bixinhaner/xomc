import { useState, useMemo } from 'react';
import { Button, Card, Col, Row, Select, Space, Typography } from 'antd';
import { AppstoreOutlined, MenuOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import LineChart from '@/components/Charts/LineChart';
import { useMultipleKPISeries } from '@/hooks/api/usePerformance';
import { useMock } from '@/services/apiSwitch';
import { useT } from '@/hooks/useT';

function formatTimestamp(ts: string): string {
  if (!ts.includes('T')) return ts;
  try {
    const d = new Date(ts);
    return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
  } catch {
    return ts;
  }
}

const KPI_OPTIONS = [
  { label: 'RRC建立成功率', value: 'RRC_SR' },
  { label: 'E-RAB建立成功率', value: 'ERAB_SR' },
  { label: '切换成功率', value: 'HO_SR' },
  { label: '下行吞吐量', value: 'DL_THROUGHPUT' },
  { label: '上行吞吐量', value: 'UL_THROUGHPUT' },
  { label: '最大用户数', value: 'MAX_USERS' },
  { label: '无线可用率', value: 'AVAILABILITY' },
  { label: 'PDCP丢包率', value: 'PDCP_LOSS' },
];

const DEVICE_OPTIONS = [
  { label: 'ENB00001 - 北京朝阳基站01', value: 'ENB00001' },
  { label: 'ENB00002 - 北京海淀基站01', value: 'ENB00002' },
  { label: 'ENB00003 - 上海浦东基站01', value: 'ENB00003' },
  { label: 'GNB00001 - 北京5G基站01', value: 'GNB00001' },
  { label: 'GNB00002 - 北京5G基站02', value: 'GNB00002' },
];

const TIME_RANGE_OPTIONS = [
  { label: '1h', value: '1h' },
  { label: '6h', value: '6h' },
  { label: '24h', value: '24h' },
  { label: '7d', value: '7d' },
  { label: '30d', value: '30d' },
];

// Mock time series data generation
function generateMockTimeSeries(kpiCode: string, points = 24) {
  const baseValues: Record<string, { base: number; range: number; unit: string }> = {
    RRC_SR: { base: 99, range: 1.5, unit: '%' },
    ERAB_SR: { base: 98.5, range: 1.5, unit: '%' },
    HO_SR: { base: 97, range: 3, unit: '%' },
    DL_THROUGHPUT: { base: 140, range: 40, unit: 'Mbps' },
    UL_THROUGHPUT: { base: 45, range: 15, unit: 'Mbps' },
    MAX_USERS: { base: 800, range: 200, unit: '个' },
    AVAILABILITY: { base: 99.9, range: 0.15, unit: '%' },
    PDCP_LOSS: { base: 0.05, range: 0.08, unit: '%' },
  };
  const cfg = baseValues[kpiCode] ?? { base: 100, range: 10, unit: '' };
  return {
    kpiName: KPI_OPTIONS.find((k) => k.value === kpiCode)?.label ?? kpiCode,
    unit: cfg.unit,
    data: Array.from({ length: points }, (_, i) => ({
      timestamp: `${String(i).padStart(2, '0')}:00`,
      value: Math.max(0, cfg.base - cfg.range / 2 + Math.random() * cfg.range),
    })),
  };
}

export default function PerformanceCharts() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [selectedKPIs, setSelectedKPIs] = useState<string[]>(['RRC_SR', 'DL_THROUGHPUT']);
  const [selectedDevice, setSelectedDevice] = useState<string>('ENB00001');
  const [displayMode, setDisplayMode] = useState<'overlay' | 'side-by-side'>('side-by-side');

  const { data: seriesData } = useMultipleKPISeries(
    (filters.kpiCodes as string[]) ?? selectedKPIs,
    (filters.deviceSn as string) ?? selectedDevice,
  );

  const activeKPIs = (filters.kpiCodes as string[] | undefined) ?? selectedKPIs;

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'deviceSn', label: t('device.name'), type: 'select', options: DEVICE_OPTIONS, span: 2 },
    { name: 'timeRange', label: t('perf.timeRange'), type: 'select', options: TIME_RANGE_OPTIONS },
    {
      name: 'kpiCodes',
      label: t('perf.kpiName'),
      type: 'multi-select',
      options: KPI_OPTIONS,
      span: 2,
    },
  ], [t]);

  // Use either API data or mock data
  const chartSeriesMap: Record<string, { name: string; unit: string; data: number[]; xData: string[] }> = {};
  for (const kpi of activeKPIs) {
    // Match by kpiName (mock returns label) or by kpiCode (real API returns code)
    const kpiLabel = KPI_OPTIONS.find((k) => k.value === kpi)?.label;
    const apiSeries = seriesData?.find(
      (s) => s.kpiName === kpiLabel || s.kpiName === kpi
    );
    if (apiSeries && apiSeries.data.length > 0) {
      chartSeriesMap[kpi] = {
        name: kpiLabel || apiSeries.kpiName,
        unit: apiSeries.unit,
        data: apiSeries.data.map((d) => d.value),
        xData: apiSeries.data.map((d) => formatTimestamp(d.timestamp)),
      };
    } else if (useMock) {
      const mock = generateMockTimeSeries(kpi);
      chartSeriesMap[kpi] = {
        name: mock.kpiName,
        unit: mock.unit,
        data: mock.data.map((d) => d.value),
        xData: mock.data.map((d) => d.timestamp),
      };
    }
  }

  const handleSearch = (vals: Record<string, unknown>) => {
    setFilters(vals);
    if (vals.kpiCodes) setSelectedKPIs(vals.kpiCodes as string[]);
    if (vals.deviceSn) setSelectedDevice(vals.deviceSn as string);
  };

  const renderCharts = () => {
    if (activeKPIs.length === 0) {
      return (
        <div style={{ padding: 40, textAlign: 'center', color: '#8c8c8c' }}>
          {t('common.pleaseSelect')}
        </div>
      );
    }

    if (displayMode === 'overlay') {
      // All KPIs in one chart (only makes sense if they share the same unit)
      const allSeries = activeKPIs.map((kpi) => ({
        name: chartSeriesMap[kpi]?.name ?? kpi,
        data: chartSeriesMap[kpi]?.data ?? [],
      }));
      const xData = chartSeriesMap[activeKPIs[0]]?.xData ?? [];

      return (
        <Card title={t('nav.performance.charts')} style={{ marginTop: 16 }}>
          <LineChart
            series={allSeries}
            xData={xData}
            height={400}
            smooth
            areaFill={false}
          />
        </Card>
      );
    }

    // Side-by-side mode
    const colSpan = activeKPIs.length === 1 ? 24 : activeKPIs.length === 2 ? 12 : 8;
    return (
      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        {activeKPIs.map((kpi) => {
          const s = chartSeriesMap[kpi];
          if (!s) return null;
          return (
            <Col key={kpi} span={colSpan}>
              <Card
                size="small"
                title={<span style={{ fontSize: 13 }}>{s.name} <span style={{ color: '#8c8c8c', fontWeight: 400 }}>({s.unit})</span></span>}
              >
                <LineChart
                  series={[{ name: s.name, data: s.data }]}
                  xData={s.xData}
                  height={220}
                  smooth
                  areaFill
                />
              </Card>
            </Col>
          );
        })}
      </Row>
    );
  };

  return (
    <ListPageLayout title={t('nav.performance.charts')}>
      <FilterBar
        filterId="performance-charts"
        fields={filterFields}
        onSearch={handleSearch}
        onReset={() => { setFilters({}); setSelectedKPIs(['RRC_SR', 'DL_THROUGHPUT']); }}
        extra={
          <Space>
            <Typography.Text style={{ fontSize: 12 }}>{t('common.view')}:</Typography.Text>
            <Select
              size="small"
              value={displayMode}
              onChange={(val) => setDisplayMode(val as 'overlay' | 'side-by-side')}
              options={[
                { label: 'overlay', value: 'overlay', icon: <AppstoreOutlined /> },
                { label: 'side-by-side', value: 'side-by-side', icon: <MenuOutlined /> },
              ]}
              style={{ width: 120 }}
            />
          </Space>
        }
      />

      {/* Quick KPI selector */}
      <div style={{ padding: '8px 0', display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
        <Typography.Text style={{ fontSize: 12, flexShrink: 0 }}>{t('common.pleaseSelect')}:</Typography.Text>
        {KPI_OPTIONS.map((kpi) => (
          <Button
            key={kpi.value}
            size="small"
            type={selectedKPIs.includes(kpi.value) ? 'primary' : 'default'}
            onClick={() => {
              setSelectedKPIs((prev) =>
                prev.includes(kpi.value)
                  ? prev.filter((k) => k !== kpi.value)
                  : [...prev, kpi.value],
              );
            }}
          >
            {kpi.label}
          </Button>
        ))}
      </div>

      {renderCharts()}
    </ListPageLayout>
  );
}
