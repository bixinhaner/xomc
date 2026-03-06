import { useState, useMemo } from 'react';
import { Button, message } from 'antd';
import { DownloadOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useT } from '@/hooks/useT';

interface StationKPIRow extends Record<string, unknown> {
  id: string;
  stationName: string;
  sn: string;
  rrc_sr: number;
  erab_sr: number;
  ho_sr: number;
  dl_throughput: number;
  ul_throughput: number;
  max_users: number;
  availability: number;
  timestamp: string;
}

const mockData: StationKPIRow[] = [
  { id: '1', stationName: '北京朝阳基站01', sn: 'ENB00001', rrc_sr: 99.2, erab_sr: 98.7, ho_sr: 97.5, dl_throughput: 145.6, ul_throughput: 45.2, max_users: 856, availability: 99.95, timestamp: '2026-03-02 10:00:00' },
  { id: '2', stationName: '北京海淀基站01', sn: 'ENB00002', rrc_sr: 98.9, erab_sr: 98.5, ho_sr: 96.8, dl_throughput: 132.1, ul_throughput: 42.8, max_users: 720, availability: 100, timestamp: '2026-03-02 10:00:00' },
  { id: '3', stationName: '上海浦东基站01', sn: 'ENB00003', rrc_sr: 97.5, erab_sr: 97.1, ho_sr: 94.2, dl_throughput: 118.3, ul_throughput: 38.6, max_users: 650, availability: 99.8, timestamp: '2026-03-02 10:00:00' },
  { id: '4', stationName: '北京5G基站01', sn: 'GNB00001', rrc_sr: 99.8, erab_sr: 99.6, ho_sr: 98.9, dl_throughput: 890.5, ul_throughput: 180.2, max_users: 256, availability: 100, timestamp: '2026-03-02 10:00:00' },
  { id: '5', stationName: '北京5G基站02', sn: 'GNB00002', rrc_sr: 99.5, erab_sr: 99.3, ho_sr: 98.2, dl_throughput: 720.8, ul_throughput: 156.4, max_users: 198, availability: 99.98, timestamp: '2026-03-02 10:00:00' },
];

const GRANULARITY_OPTIONS = [
  { label: '15min', value: '15min' },
  { label: '30min', value: '30min' },
  { label: '1h', value: '1h' },
  { label: '1d', value: '1d' },
];

export default function KPIStationReport() {
  const t = useT();
  const [, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'sn', label: t('device.sn'), type: 'input', placeholder: t('common.placeholder') },
    { name: 'stationName', label: t('device.name'), type: 'input', placeholder: t('common.placeholder') },
    { name: 'timeRange', label: t('perf.timeRange'), type: 'date-range', span: 2 },
    { name: 'granularity', label: t('perf.granularity'), type: 'select', options: GRANULARITY_OPTIONS },
  ], [t]);

  const columns: DataTableColumn<StationKPIRow>[] = useMemo(() => [
    { key: 'stationName', title: t('device.name'), dataIndex: 'stationName', width: 180, ellipsis: true, fixed: 'left' },
    { key: 'sn', title: t('device.sn'), dataIndex: 'sn', width: 120, mono: true, copyable: true },
    {
      key: 'rrc_sr',
      title: 'RRC(%)',
      dataIndex: 'rrc_sr',
      width: 150,
      render: (val) => {
        const v = val as number;
        return <span style={{ color: v < 95 ? '#ff4d4f' : v < 98 ? '#faad14' : '#52c41a' }}>{v.toFixed(2)}%</span>;
      },
    },
    {
      key: 'erab_sr',
      title: 'E-RAB(%)',
      dataIndex: 'erab_sr',
      width: 160,
      render: (val) => {
        const v = val as number;
        return <span style={{ color: v < 95 ? '#ff4d4f' : v < 98 ? '#faad14' : '#52c41a' }}>{v.toFixed(2)}%</span>;
      },
    },
    {
      key: 'ho_sr',
      title: 'HO(%)',
      dataIndex: 'ho_sr',
      width: 130,
      render: (val) => {
        const v = val as number;
        return <span style={{ color: v < 95 ? '#ff4d4f' : v < 97 ? '#faad14' : '#52c41a' }}>{v.toFixed(2)}%</span>;
      },
    },
    { key: 'dl_throughput', title: 'DL(Mbps)', dataIndex: 'dl_throughput', width: 150 },
    { key: 'ul_throughput', title: 'UL(Mbps)', dataIndex: 'ul_throughput', width: 150 },
    { key: 'max_users', title: t('table.total'), dataIndex: 'max_users', width: 110 },
    {
      key: 'availability',
      title: 'Avail(%)',
      dataIndex: 'availability',
      width: 130,
      render: (val) => {
        const v = val as number;
        return <span style={{ color: v < 99.9 ? '#ff4d4f' : '#52c41a' }}>{v.toFixed(2)}%</span>;
      },
    },
    { key: 'timestamp', title: t('perf.timestamp'), dataIndex: 'timestamp', width: 160 },
  ], [t]);

  return (
    <ListPageLayout
      title={t('nav.performance.kpiStation')}
      extra={
        <Button icon={<DownloadOutlined />} onClick={() => void message.info(t('common.exportInProgress'))}>
          {t('common.export')}
        </Button>
      }
    >
      <FilterBar
        filterId="kpi-station-report"
        fields={filterFields}
        onSearch={(vals) => setFilters(vals)}
        onReset={() => setFilters({})}
      />
      <DataTable<StationKPIRow>
        tableId="kpi-station-report"
        columns={columns}
        dataSource={mockData}
        loading={false}
        rowKey="id"
        total={mockData.length}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onExport={() => void message.info(t('common.exportInProgress'))}
        scroll={{ x: 1500 }}
      />
    </ListPageLayout>
  );
}
