import { useState, useMemo } from 'react';
import { Card, Button, Select, Space, Tag } from 'antd';
import { DownloadOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import LineChart from '@/components/Charts/LineChart';
import type { LineSeries } from '@/components/Charts/LineChart';
import { useT } from '@/hooks/useT';

const kpiOptions = [
  { label: '无线接通率', value: 'accessRate' },
  { label: '切换成功率', value: 'hoSuccessRate' },
  { label: 'PRB利用率', value: 'prbUtil' },
  { label: '掉话率', value: 'dropRate' },
  { label: '在线用户数', value: 'onlineUsers' },
  { label: '下行吞吐量(Mbps)', value: 'dlThroughput' },
  { label: '上行吞吐量(Mbps)', value: 'ulThroughput' },
];

const generateTimePoints = (count: number, granularity: string): string[] => {
  const points: string[] = [];
  const now = new Date('2024-06-01T00:00:00.000Z');
  const intervalMs = granularity === '15min' ? 15 * 60 * 1000 : granularity === '1h' ? 60 * 60 * 1000 : 24 * 60 * 60 * 1000;
  for (let i = 0; i < count; i++) {
    const d = new Date(now.getTime() + i * intervalMs);
    if (granularity === '1d') {
      points.push(d.toISOString().slice(0, 10));
    } else {
      points.push(d.toISOString().slice(0, 16).replace('T', ' '));
    }
  }
  return points;
};

const generateKPIData = (kpi: string, count: number): number[] => {
  const configs: Record<string, { base: number; range: number; min: number; max: number }> = {
    accessRate: { base: 99.5, range: 1, min: 97, max: 100 },
    hoSuccessRate: { base: 99.7, range: 0.8, min: 97, max: 100 },
    prbUtil: { base: 55, range: 30, min: 10, max: 95 },
    dropRate: { base: 0.05, range: 0.08, min: 0, max: 0.3 },
    onlineUsers: { base: 250, range: 150, min: 50, max: 500 },
    dlThroughput: { base: 80, range: 60, min: 10, max: 200 },
    ulThroughput: { base: 30, range: 20, min: 5, max: 80 },
  };
  const c = configs[kpi] ?? { base: 50, range: 30, min: 0, max: 100 };
  return Array.from({ length: count }, () => {
    const val = c.base + (Math.random() - 0.5) * c.range;
    return Math.max(c.min, Math.min(c.max, Number(val.toFixed(3))));
  });
};

interface KPIDataRow {
  id: string;
  timestamp: string;
  deviceSn: string;
  [key: string]: string | number;
}

export default function HistoricalKPI() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [selectedKPIs, setSelectedKPIs] = useState<string[]>(['accessRate', 'hoSuccessRate']);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const granularity = (filters.granularity as string) ?? '1h';
  const dataCount = granularity === '15min' ? 96 : granularity === '1h' ? 24 : 30;
  const xData = generateTimePoints(dataCount, granularity);

  const series: LineSeries[] = selectedKPIs.map((kpi) => ({
    name: kpiOptions.find((o) => o.value === kpi)?.label ?? kpi,
    data: generateKPIData(kpi, dataCount),
  }));

  const tableRows: KPIDataRow[] = xData.slice(0, 50).map((ts, i) => {
    const row: KPIDataRow = { id: `kpi-${i}`, timestamp: ts, deviceSn: 'ENB00001' };
    selectedKPIs.forEach((kpi) => {
      row[kpi] = series.find((s) => s.name === kpiOptions.find((o) => o.value === kpi)?.label)?.data[i] ?? 0;
    });
    return row;
  });

  const startIndex = (page - 1) * pageSize;
  const paginated = tableRows.slice(startIndex, startIndex + pageSize);

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'timeRange', label: t('perf.timeRange'), type: 'date-range' },
    {
      name: 'granularity',
      label: t('perf.granularity'),
      type: 'select',
      options: [
        { label: '15min', value: '15min' },
        { label: '1h', value: '1h' },
        { label: '1d', value: '1d' },
      ],
    },
    {
      name: 'deviceSn',
      label: t('device.sn'),
      type: 'input',
      placeholder: t('device.sn'),
    },
    {
      name: 'region',
      label: t('table.region'),
      type: 'select',
      options: [
        { label: '北京', value: '北京' },
        { label: '上海', value: '上海' },
        { label: '广州', value: '广州' },
      ],
    },
  ], [t]);

  const columns: DataTableColumn<KPIDataRow & Record<string, unknown>>[] = useMemo(() => [
    {
      key: 'timestamp', title: t('table.time'), dataIndex: 'timestamp', width: 160,
      render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span>,
    },
    { key: 'deviceSn', title: t('device.sn'), dataIndex: 'deviceSn', width: 120, mono: true },
    ...selectedKPIs.map((kpi) => ({
      key: kpi,
      title: kpiOptions.find((o) => o.value === kpi)?.label ?? kpi,
      dataIndex: kpi,
      width: 140,
      render: (val: unknown) => {
        const n = Number(val);
        if (kpi === 'accessRate' || kpi === 'hoSuccessRate') {
          const color = n >= 99.5 ? '#52c41a' : n >= 98 ? '#faad14' : '#ff4d4f';
          return <span style={{ color, fontWeight: 500 }}>{n.toFixed(2)}%</span>;
        }
        if (kpi === 'dropRate') {
          const color = n <= 0.05 ? '#52c41a' : n <= 0.1 ? '#faad14' : '#ff4d4f';
          return <span style={{ color, fontWeight: 500 }}>{n.toFixed(3)}%</span>;
        }
        if (kpi === 'prbUtil') {
          const color = n <= 60 ? '#52c41a' : n <= 80 ? '#faad14' : '#ff4d4f';
          return <span style={{ color, fontWeight: 500 }}>{n.toFixed(1)}%</span>;
        }
        return <span>{typeof val === 'number' ? val.toFixed(2) : String(val)}</span>;
      },
    })),
  ], [t, selectedKPIs]);

  return (
    <ListPageLayout
      title={t('nav.report.historicalKpi')}
      subtitle={t('nav.report.historicalKpi')}
      extra={<Button icon={<DownloadOutlined />}>{t('common.export')}</Button>}
    >
      <FilterBar
        filterId="historical-kpi-filter"
        fields={filterFields}
        onSearch={(vals) => { setFilters(vals); setPage(1); }}
        onReset={() => { setFilters({}); setPage(1); }}
      />

      <Card
        title={t('perf.kpiName')}
        extra={
          <Space>
            <span style={{ fontSize: 13, color: '#666' }}>{t('common.pleaseSelect')}:</span>
            <Select
              mode="multiple"
              value={selectedKPIs}
              onChange={setSelectedKPIs}
              options={kpiOptions}
              style={{ width: 350 }}
              maxTagCount={3}
              placeholder={t('common.pleaseSelect')}
            />
          </Space>
        }
        style={{ marginBottom: 16 }}
      >
        {selectedKPIs.length > 0 ? (
          <LineChart
            xData={xData}
            series={series}
            height={320}
            smooth
          />
        ) : (
          <div style={{ textAlign: 'center', padding: 40, color: '#999' }}>{t('common.pleaseSelect')}</div>
        )}
      </Card>

      <Card title={t('common.detail')}>
        <DataTable
          tableId="historical-kpi-data"
          columns={columns}
          dataSource={paginated as (KPIDataRow & Record<string, unknown>)[]}
          loading={false}
          rowKey="id"
          total={tableRows.length}
          pageSize={pageSize}
          currentPage={page}
          onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
          onExport={(format) => void console.log('export', format)}
          scroll={{ x: 800 }}
        />
      </Card>
    </ListPageLayout>
  );
}
