import { useEffect, useMemo, useState } from 'react';
import { Button, Card, Col, Row, Select, Space, Typography } from 'antd';
import { AppstoreOutlined, MenuOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import LineChart from '@/components/Charts/LineChart';
import { useAllKPIs, useMultipleKPISeries } from '@core/hooks/api/usePerformance';
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

// 默认从真实 KPI 目录里勾选的指标条数（目录就绪后取前 N 条）。
const DEFAULT_KPI_COUNT = 4;

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

export default function PerformanceCharts() {
  const t = useT();

  // 真实 KPI 目录（/pm/kpi/definitions）：kpiCode = 指标编号(K…)，kpiName = 中文显示名。
  const { data: kpiCatalog } = useAllKPIs();
  const catalog = useMemo(() => kpiCatalog ?? [], [kpiCatalog]);

  const KPI_OPTIONS = useMemo(
    () => catalog.map((k) => ({ label: k.kpiName || k.kpiCode, value: k.kpiCode })),
    [catalog],
  );
  // 快捷按钮区只展示前 12 条常用指标，全量目录走 FilterBar 多选下拉避免一墙按钮。
  const QUICK_KPI_OPTIONS = useMemo(() => KPI_OPTIONS.slice(0, 12), [KPI_OPTIONS]);
  // kpiCode → 显示名 / 单位 映射，供图表卡片标题与单位渲染。
  const kpiMeta = useMemo(() => {
    const m: Record<string, { name: string; unit: string }> = {};
    catalog.forEach((k) => {
      m[k.kpiCode] = { name: k.kpiName || k.kpiCode, unit: k.unit };
    });
    return m;
  }, [catalog]);

  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [selectedKPIs, setSelectedKPIs] = useState<string[]>([]);
  const [selectedDevice, setSelectedDevice] = useState<string>('ENB00001');
  const [displayMode, setDisplayMode] = useState<'overlay' | 'side-by-side'>('side-by-side');

  // 目录就绪后默认勾选前 N 条真实 KPI（默认即出真实图，不空等用户手选）。
  useEffect(() => {
    if (catalog.length === 0) return;
    setSelectedKPIs((prev) =>
      prev.length > 0 ? prev : catalog.slice(0, DEFAULT_KPI_COUNT).map((k) => k.kpiCode),
    );
  }, [catalog]);

  const activeKPIs = (filters.kpiCodes as string[] | undefined) ?? selectedKPIs;

  const { data: seriesData } = useMultipleKPISeries(
    activeKPIs,
    (filters.deviceSn as string) ?? selectedDevice,
  );

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
  ], [t, KPI_OPTIONS]);

  // 真实 series 以 kpiName=kpiCode 返回（pmApi.getKPISeries 用入参 code 作 kpiName）。
  const chartSeriesMap: Record<string, { name: string; unit: string; data: number[]; xData: string[] }> = {};
  for (const code of activeKPIs) {
    const meta = kpiMeta[code];
    const apiSeries = seriesData?.find((s) => s.kpiName === code);
    if (apiSeries && apiSeries.data.length > 0) {
      chartSeriesMap[code] = {
        name: meta?.name ?? code,
        unit: apiSeries.unit || meta?.unit || '',
        data: apiSeries.data.map((d) => d.value),
        xData: apiSeries.data.map((d) => formatTimestamp(d.timestamp)),
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
      const allSeries = activeKPIs.map((code) => ({
        name: chartSeriesMap[code]?.name ?? kpiMeta[code]?.name ?? code,
        data: chartSeriesMap[code]?.data ?? [],
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
            connectNulls
          />
        </Card>
      );
    }

    // Side-by-side mode
    const colSpan = activeKPIs.length === 1 ? 24 : activeKPIs.length === 2 ? 12 : 8;
    return (
      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        {activeKPIs.map((code) => {
          const s = chartSeriesMap[code];
          if (!s) return null;
          return (
            <Col key={code} span={colSpan}>
              <Card
                size="small"
                title={<span style={{ fontSize: 13 }}>{s.name} {s.unit ? <span style={{ color: '#8c8c8c', fontWeight: 400 }}>({s.unit})</span> : null}</span>}
              >
                <LineChart
                  series={[{ name: s.name, data: s.data }]}
                  xData={s.xData}
                  height={220}
                  smooth
                  areaFill
                  connectNulls
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
        onReset={() => {
          setFilters({});
          setSelectedKPIs(catalog.slice(0, DEFAULT_KPI_COUNT).map((k) => k.kpiCode));
        }}
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
        {QUICK_KPI_OPTIONS.map((kpi) => (
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
