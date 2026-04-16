import { useState, useMemo } from 'react';
import { Button, Space, Tag, message } from 'antd';

import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useT } from '@/hooks/useT';

interface PerfFileRow extends Record<string, unknown> {
  id: string;
  fileName: string;
  fileSize: number;
  granularity: '15min' | '1h' | '24h';
  collectTime: string;
  deviceSn: string;
  deviceName: string;
  status: 'ready' | 'processing' | 'error';
}

const GRANULARITY_COLORS: Record<string, string> = {
  '15min': 'blue',
  '1h': 'purple',
  '24h': 'green',
};

const mockData: PerfFileRow[] = [
  { id: '1', fileName: 'A20260302.1000-1015_ENB00001.xml.gz', fileSize: 1024 * 45, granularity: '15min', collectTime: '2026-03-02 10:15:00', deviceSn: 'ENB00001', deviceName: '北京朝阳基站01', status: 'ready' },
  { id: '2', fileName: 'A20260302.1000-1015_ENB00002.xml.gz', fileSize: 1024 * 38, granularity: '15min', collectTime: '2026-03-02 10:15:00', deviceSn: 'ENB00002', deviceName: '北京海淀基站01', status: 'ready' },
  { id: '3', fileName: 'A20260302.0900-1000_ENB00001.xml.gz', fileSize: 1024 * 162, granularity: '1h', collectTime: '2026-03-02 10:00:00', deviceSn: 'ENB00001', deviceName: '北京朝阳基站01', status: 'ready' },
  { id: '4', fileName: 'A20260301.0000-2400_ENB00001.xml.gz', fileSize: 1024 * 1850, granularity: '24h', collectTime: '2026-03-02 00:05:00', deviceSn: 'ENB00001', deviceName: '北京朝阳基站01', status: 'ready' },
  { id: '5', fileName: 'A20260302.1000-1015_GNB00001.xml.gz', fileSize: 1024 * 52, granularity: '15min', collectTime: '2026-03-02 10:15:00', deviceSn: 'GNB00001', deviceName: '北京5G基站01', status: 'processing' },
  { id: '6', fileName: 'A20260302.1000-1015_ENB00003.xml.gz', fileSize: 0, granularity: '15min', collectTime: '2026-03-02 10:15:00', deviceSn: 'ENB00003', deviceName: '上海浦东基站01', status: 'error' },
];

function formatBytes(bytes: number): string {
  if (bytes === 0) return '-';
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export default function PerformanceFiles() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const STATUS_MAP: Record<string, { color: string; text: string }> = useMemo(() => ({
    ready: { color: 'success', text: t('status.success') },
    processing: { color: 'processing', text: t('status.running') },
    error: { color: 'error', text: t('status.failed') },
  }), [t]);

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'deviceSn', label: t('device.sn'), type: 'input' },
    { name: 'fileName', label: t('table.name'), type: 'input' },
    {
      name: 'granularity',
      label: t('perf.granularity'),
      type: 'select',
      options: [
        { label: '15min', value: '15min' },
        { label: '1h', value: '1h' },
        { label: '1d', value: '24h' },
      ],
    },
    {
      name: 'status',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: t('status.success'), value: 'ready' },
        { label: t('status.running'), value: 'processing' },
        { label: t('status.failed'), value: 'error' },
      ],
    },
    { name: 'collectTime', label: t('perf.timestamp'), type: 'date-range', span: 2 },
  ], [t]);

  const filteredData = mockData.filter((row) => {
    if (filters.deviceSn && !row.deviceSn.includes(filters.deviceSn as string)) return false;
    if (filters.fileName && !row.fileName.includes(filters.fileName as string)) return false;
    if (filters.granularity && row.granularity !== filters.granularity) return false;
    if (filters.status && row.status !== filters.status) return false;
    return true;
  });

  const columns: DataTableColumn<PerfFileRow>[] = useMemo(() => [
    { key: 'fileName', title: t('table.name'), dataIndex: 'fileName', width: 340, ellipsis: true, mono: true },
    {
      key: 'fileSize',
      title: t('table.total'),
      dataIndex: 'fileSize',
      width: 100,
      render: (val) => formatBytes(val as number),
    },
    {
      key: 'granularity',
      title: t('perf.granularity'),
      dataIndex: 'granularity',
      width: 90,
      render: (val) => <Tag color={GRANULARITY_COLORS[val as string]}>{val as string}</Tag>,
    },
    { key: 'collectTime', title: t('perf.timestamp'), dataIndex: 'collectTime', width: 160 },
    { key: 'deviceSn', title: t('device.sn'), dataIndex: 'deviceSn', width: 120, mono: true },
    { key: 'deviceName', title: t('device.name'), dataIndex: 'deviceName', width: 180, ellipsis: true },
    {
      key: 'status',
      title: t('table.status'),
      dataIndex: 'status',
      width: 90,
      render: (val) => {
        const cfg = STATUS_MAP[val as string] ?? STATUS_MAP.ready;
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    {
      key: 'action',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 120,
      fixed: 'right',
      render: (_, record) => (
        <Space size={4}>
          <Button
            type="link"
            size="small"
            disabled={record.status !== 'ready'}
            onClick={() => void message.success(`${t('common.download')}: ${record.fileName as string}`)}
          >
            {t('common.download')}
          </Button>
          <Button
            type="link"
            size="small"
            disabled={record.status !== 'ready'}
            onClick={() => void message.info(`${t('common.view')}: ${record.fileName as string}`)}
          >
            {t('common.view')}
          </Button>
        </Space>
      ),
    },
  ], [t, STATUS_MAP]);

  return (
    <ListPageLayout title={t('nav.performance.files')}>
      <FilterBar
        filterId="performance-files"
        fields={filterFields}
        onSearch={(vals) => setFilters(vals)}
        onReset={() => setFilters({})}
      />
      <DataTable<PerfFileRow>
        tableId="performance-files"
        columns={columns}
        dataSource={filteredData}
        loading={false}
        rowKey="id"
        total={filteredData.length}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onExport={() => void message.info(t('common.exportInProgress'))}
        scroll={{ x: 1400 }}
      />
    </ListPageLayout>
  );
}
