import { useState, useMemo } from 'react';
import { Button, Tag, Tooltip } from 'antd';
import { DownloadOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';

interface StationKPIRecord {
  id: string;
  stationName: string;
  stationSn: string;
  region: string;
  date: string;
  accessRate: number;
  hoSuccessRate: number;
  prbUtilization: number;
  dropRate: number;
  volteMos: number;
  onlineUsers: number;
  dataVolume: number;
}

const mockStationKPIs: StationKPIRecord[] = [
  { id: 'sk-001', stationName: '北京-eNB-0001', stationSn: 'ENB00001', region: '北京', date: '2024-06-01', accessRate: 99.85, hoSuccessRate: 99.92, prbUtilization: 45.2, dropRate: 0.02, volteMos: 4.21, onlineUsers: 234, dataVolume: 1024 * 150 },
  { id: 'sk-002', stationName: '北京-eNB-0002', stationSn: 'ENB00002', region: '北京', date: '2024-06-01', accessRate: 99.21, hoSuccessRate: 98.88, prbUtilization: 67.8, dropRate: 0.08, volteMos: 4.05, onlineUsers: 312, dataVolume: 1024 * 220 },
  { id: 'sk-003', stationName: '上海-eNB-0001', stationSn: 'ENB00010', region: '上海', date: '2024-06-01', accessRate: 99.95, hoSuccessRate: 99.98, prbUtilization: 38.5, dropRate: 0.01, volteMos: 4.35, onlineUsers: 189, dataVolume: 1024 * 120 },
  { id: 'sk-004', stationName: '广州-eNB-0001', stationSn: 'ENB00020', region: '广州', date: '2024-06-01', accessRate: 98.76, hoSuccessRate: 98.45, prbUtilization: 82.3, dropRate: 0.15, volteMos: 3.92, onlineUsers: 425, dataVolume: 1024 * 380 },
  { id: 'sk-005', stationName: '北京-gNB-0001', stationSn: 'GNB00001', region: '北京', date: '2024-06-01', accessRate: 99.98, hoSuccessRate: 99.99, prbUtilization: 28.9, dropRate: 0.005, volteMos: 4.48, onlineUsers: 98, dataVolume: 1024 * 580 },
  { id: 'sk-006', stationName: '深圳-eNB-0001', stationSn: 'ENB00030', region: '深圳', date: '2024-06-01', accessRate: 99.34, hoSuccessRate: 99.12, prbUtilization: 71.5, dropRate: 0.06, volteMos: 4.12, onlineUsers: 356, dataVolume: 1024 * 290 },
];

function getKPIColor(value: number, type: 'rate' | 'drop' | 'prb' | 'mos'): string {
  if (type === 'rate') return value >= 99.5 ? '#52c41a' : value >= 98 ? '#faad14' : '#ff4d4f';
  if (type === 'drop') return value <= 0.05 ? '#52c41a' : value <= 0.1 ? '#faad14' : '#ff4d4f';
  if (type === 'prb') return value <= 60 ? '#52c41a' : value <= 80 ? '#faad14' : '#ff4d4f';
  if (type === 'mos') return value >= 4.2 ? '#52c41a' : value >= 3.8 ? '#faad14' : '#ff4d4f';
  return '#595959';
}

export default function StationReport() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const filtered = mockStationKPIs.filter((r) => {
    if (filters.keyword) {
      const kw = String(filters.keyword).toLowerCase();
      if (!r.stationName.toLowerCase().includes(kw) && !r.stationSn.toLowerCase().includes(kw)) return false;
    }
    if (filters.region && r.region !== filters.region) return false;
    return true;
  });

  const startIndex = (page - 1) * pageSize;
  const paginated = filtered.slice(startIndex, startIndex + pageSize);

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('device.name'), type: 'input', placeholder: t('device.name') },
    { name: 'timeRange', label: t('perf.timeRange'), type: 'date-range' },
    {
      name: 'kpiType',
      label: t('perf.kpiName'),
      type: 'select',
      options: [
        { label: t('kpi.accessRate'), value: 'access_rate' },
        { label: t('kpi.handoverSuccessRate'), value: 'ho_success_rate' },
        { label: t('kpi.prbUtilization'), value: 'prb_utilization' },
        { label: t('kpi.dropRate'), value: 'drop_rate' },
        { label: t('kpi.volteMos'), value: 'volte_quality' },
      ],
    },
    {
      name: 'region',
      label: t('table.region'),
      type: 'select',
      options: [
        { label: '北京', value: '北京' },
        { label: '上海', value: '上海' },
        { label: '广州', value: '广州' },
        { label: '深圳', value: '深圳' },
      ],
    },
  ], [t]);

  const columns: DataTableColumn<StationKPIRecord & Record<string, unknown>>[] = useMemo(() => [
    { key: 'stationName', title: t('device.name'), dataIndex: 'stationName', ellipsis: true, width: 160 },
    { key: 'stationSn', title: t('device.sn'), dataIndex: 'stationSn', width: 120, render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span> },
    { key: 'region', title: t('table.region'), dataIndex: 'region', width: 80 },
    { key: 'date', title: t('table.time'), dataIndex: 'date', width: 110 },
    {
      key: 'accessRate', title: t('perf.kpiName') + '(%)', dataIndex: 'accessRate', width: 130,
      render: (val) => <span style={{ color: getKPIColor(Number(val), 'rate'), fontWeight: 500 }}>{Number(val).toFixed(2)}%</span>,
    },
    {
      key: 'hoSuccessRate', title: t('perf.kpiName') + '(%)', dataIndex: 'hoSuccessRate', width: 130,
      render: (val) => <span style={{ color: getKPIColor(Number(val), 'rate'), fontWeight: 500 }}>{Number(val).toFixed(2)}%</span>,
    },
    {
      key: 'prbUtilization', title: 'PRB(%)', dataIndex: 'prbUtilization', width: 120,
      render: (val) => <span style={{ color: getKPIColor(Number(val), 'prb'), fontWeight: 500 }}>{Number(val).toFixed(1)}%</span>,
    },
    {
      key: 'dropRate', title: t('perf.kpiName') + '(%)', dataIndex: 'dropRate', width: 100,
      render: (val) => <span style={{ color: getKPIColor(Number(val), 'drop'), fontWeight: 500 }}>{Number(val).toFixed(3)}%</span>,
    },
    {
      key: 'volteMos', title: 'VoLTE MOS', dataIndex: 'volteMos', width: 110,
      render: (val) => <span style={{ color: getKPIColor(Number(val), 'mos'), fontWeight: 500 }}>{Number(val).toFixed(2)}</span>,
    },
    { key: 'onlineUsers', title: t('table.total'), dataIndex: 'onlineUsers', width: 100 },
    {
      key: 'dataVolume', title: t('perf.value') + '(GB)', dataIndex: 'dataVolume', width: 120,
      render: (val) => `${(Number(val) / 1024).toFixed(2)} GB`,
    },
    {
      key: 'actions', title: t('table.operation'), dataIndex: 'id', width: 80, fixed: 'right',
      render: () => <Button type="link" size="small">{t('common.export')}</Button>,
    },
  ], [t]);

  return (
    <ListPageLayout
      title={t('nav.report.station')}
      subtitle={t('nav.report.station')}
      extra={<Button type="primary" icon={<DownloadOutlined />}>{t('common.batchExport')}</Button>}
    >
      <FilterBar
        filterId="station-report-filter"
        fields={filterFields}
        onSearch={(vals) => { setFilters(vals); setPage(1); }}
        onReset={() => { setFilters({}); setPage(1); }}
      />
      <DataTable
        tableId="station-report-list"
        columns={columns}
        dataSource={paginated as (StationKPIRecord & Record<string, unknown>)[]}
        loading={false}
        rowKey="id"
        total={filtered.length}
        pageSize={pageSize}
        currentPage={page}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onExport={(format) => void console.log('export', format)}
        scroll={{ x: 1300 }}
      />
    </ListPageLayout>
  );
}
