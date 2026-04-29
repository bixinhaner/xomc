import { useState, useMemo } from 'react';
import { Card, Button, Select, Space, message } from 'antd';
import { DownloadOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import LineChart from '@/components/Charts/LineChart';
import type { LineSeries } from '@/components/Charts/LineChart';
// T-0022: KPI option list + mock series helpers centralised in frontend-core.
// Backend `/pm/kpi/timeseries` endpoint is the future replacement for the
// generators; the page consumes a stable shape so the swap is local to this file.
import {
  HISTORICAL_KPI_OPTIONS,
  generateHistoricalKPIData,
  generateHistoricalKPITimePoints,
} from '@core/mock/data/reports';
import { useReportRecords, useDownloadReport } from '@core/hooks/api/useReports';
import { useT } from '@/hooks/useT';

type Granularity = '15min' | '1h' | '1d';

interface KPIDataRow {
  id: string;
  timestamp: string;
  deviceSn: string;
  [key: string]: string | number;
}

export default function HistoricalKPI() {
  const t = useT();
  const kpiOptions = useMemo(() =>
    HISTORICAL_KPI_OPTIONS.map((k) => ({ label: t(k.labelKey), value: k.value })),
  [t]);
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [selectedKPIs, setSelectedKPIs] = useState<string[]>(['accessRate', 'hoSuccessRate']);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  // Probe the records endpoint so the page is wired to the real API surface.
  // The chart series itself still renders from local generators until the
  // backend exposes a per-KPI time-series endpoint.
  const { isError: recordsError } = useReportRecords({ page: 1, pageSize: 50 });
  const downloadReport = useDownloadReport();

  const granularity = ((filters.granularity as Granularity | undefined) ?? '1h') as Granularity;
  const dataCount = granularity === '15min' ? 96 : granularity === '1h' ? 24 : 30;
  const xData = generateHistoricalKPITimePoints(dataCount, granularity);

  const series: LineSeries[] = selectedKPIs.map((kpi) => ({
    name: kpiOptions.find((o) => o.value === kpi)?.label ?? kpi,
    data: generateHistoricalKPIData(kpi, dataCount),
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
      extra={
        <Button
          icon={<DownloadOutlined />}
          loading={downloadReport.isPending}
          disabled={recordsError}
          onClick={async () => {
            try {
              // The page does not yet have a chosen record id; once the table
              // surfaces a per-row download action we pass the recordId here.
              await downloadReport.mutateAsync('latest');
              void message.success(t('common.exportInProgress'));
            } catch {
              void message.warning(t('common.exportInProgress'));
            }
          }}
        >
          {t('common.export')}
        </Button>
      }
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
