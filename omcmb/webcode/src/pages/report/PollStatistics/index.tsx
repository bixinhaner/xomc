import { useState, useMemo } from 'react';
import { Button, Tag, Progress, Tooltip } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';

type PollStatus = 'running' | 'paused' | 'stopped' | 'error';

interface PollStatRecord {
  id: string;
  taskName: string;
  deviceRange: number;
  collectionPeriod: string;
  successRate: number;
  successCount: number;
  failCount: number;
  lastExecution: string;
  status: PollStatus;
  collectionType: string;
  avgDuration: number;
}

const mockPollStats: PollStatRecord[] = [
  { id: 'ps-001', taskName: 'eNB性能指标采集_15min', deviceRange: 150, collectionPeriod: '15分钟', successRate: 98.7, successCount: 148, failCount: 2, lastExecution: '2024-06-01T08:00:00.000Z', status: 'running', collectionType: '性能指标', avgDuration: 45 },
  { id: 'ps-002', taskName: 'gNB性能指标采集_15min', deviceRange: 45, collectionPeriod: '15分钟', successRate: 100, successCount: 45, failCount: 0, lastExecution: '2024-06-01T08:00:00.000Z', status: 'running', collectionType: '性能指标', avgDuration: 32 },
  { id: 'ps-003', taskName: '设备状态轮询_1min', deviceRange: 215, collectionPeriod: '1分钟', successRate: 99.5, successCount: 214, failCount: 1, lastExecution: '2024-06-01T08:01:00.000Z', status: 'running', collectionType: '设备状态', avgDuration: 8 },
  { id: 'ps-004', taskName: '告警数据同步_5min', deviceRange: 215, collectionPeriod: '5分钟', successRate: 97.2, successCount: 209, failCount: 6, lastExecution: '2024-06-01T08:00:00.000Z', status: 'running', collectionType: '告警数据', avgDuration: 18 },
  { id: 'ps-005', taskName: 'eNB MR数据采集_1h', deviceRange: 150, collectionPeriod: '1小时', successRate: 94.7, successCount: 142, failCount: 8, lastExecution: '2024-06-01T08:00:00.000Z', status: 'error', collectionType: 'MR数据', avgDuration: 120 },
  { id: 'ps-006', taskName: '全网性能日采集', deviceRange: 215, collectionPeriod: '24小时', successRate: 96.3, successCount: 207, failCount: 8, lastExecution: '2024-06-01T01:00:00.000Z', status: 'running', collectionType: '性能指标', avgDuration: 380 },
  { id: 'ps-007', taskName: '北京区eNB专项采集', deviceRange: 35, collectionPeriod: '30分钟', successRate: 88.6, successCount: 31, failCount: 4, lastExecution: '2024-06-01T07:30:00.000Z', status: 'paused', collectionType: '性能指标', avgDuration: 65 },
];

const statusColorMap: Record<PollStatus, string> = {
  running: 'processing',
  paused: 'warning',
  stopped: 'default',
  error: 'error',
};

export default function PollStatistics() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const filtered = mockPollStats.filter((r) => {
    if (filters.keyword && !r.taskName.includes(String(filters.keyword))) return false;
    if (filters.status && r.status !== filters.status) return false;
    if (filters.collectionType && r.collectionType !== filters.collectionType) return false;
    return true;
  });

  const startIndex = (page - 1) * pageSize;
  const paginated = filtered.slice(startIndex, startIndex + pageSize);

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('table.name'), type: 'input', placeholder: t('table.name') },
    {
      name: 'status',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: t('status.running'), value: 'running' },
        { label: t('status.pending'), value: 'paused' },
        { label: t('status.cancelled'), value: 'stopped' },
        { label: t('status.failed'), value: 'error' },
      ],
    },
    {
      name: 'collectionType',
      label: t('table.type'),
      type: 'select',
      options: [
        { label: '性能指标', value: '性能指标' },
        { label: '设备状态', value: '设备状态' },
        { label: '告警数据', value: '告警数据' },
        { label: 'MR数据', value: 'MR数据' },
      ],
    },
  ], [t]);

  const columns: DataTableColumn<PollStatRecord & Record<string, unknown>>[] = useMemo(() => [
    { key: 'taskName', title: t('table.name'), dataIndex: 'taskName', ellipsis: true, width: 220 },
    { key: 'collectionType', title: t('table.type'), dataIndex: 'collectionType', width: 100 },
    {
      key: 'deviceRange', title: t('table.total'), dataIndex: 'deviceRange', width: 90,
      render: (val) => String(val),
    },
    { key: 'collectionPeriod', title: t('perf.granularity'), dataIndex: 'collectionPeriod', width: 100 },
    {
      key: 'successRate', title: t('table.success') + '(%)', dataIndex: 'successRate', width: 150,
      render: (val, record) => {
        const r = record as PollStatRecord;
        const rate = Number(val);
        const status = rate >= 99 ? 'success' : rate >= 95 ? 'normal' : 'exception';
        return (
          <div>
            <Progress percent={rate} size="small" status={status} format={(p) => `${p?.toFixed(1)}%`} />
            <div style={{ fontSize: 11, color: '#999', marginTop: 2 }}>
              {t('table.success')} {r.successCount}/{r.deviceRange}, {t('table.failed')} {r.failCount}
            </div>
          </div>
        );
      },
    },
    {
      key: 'avgDuration', title: t('perf.value'), dataIndex: 'avgDuration', width: 100,
      render: (val) => {
        const secs = Number(val);
        return secs >= 60 ? `${Math.floor(secs / 60)}m${secs % 60}s` : `${secs}s`;
      },
    },
    {
      key: 'lastExecution', title: t('table.time'), dataIndex: 'lastExecution', width: 160,
      render: (val) => new Date(String(val)).toLocaleString('zh-CN'),
    },
    {
      key: 'status', title: t('table.status'), dataIndex: 'status', width: 100,
      render: (val) => {
        const s = val as PollStatus;
        return <Tag color={statusColorMap[s]}>{s === 'running' ? t('status.running') : s === 'paused' ? t('status.pending') : s === 'stopped' ? t('status.cancelled') : t('status.failed')}</Tag>;
      },
    },
    {
      key: 'actions', title: t('table.operation'), dataIndex: 'id', width: 80, fixed: 'right',
      render: () => (
        <Button type="link" size="small">查看</Button>
      ),
    },
  ], [t]);

  return (
    <ListPageLayout
      title={t('nav.report.pollStats')}
      subtitle={t('nav.report.pollStats')}
      extra={
        <Button icon={<ReloadOutlined />} onClick={() => setFilters({ ...filters })}>{t('common.refresh')}</Button>
      }
    >
      <FilterBar
        filterId="poll-statistics-filter"
        fields={filterFields}
        onSearch={(vals) => { setFilters(vals); setPage(1); }}
        onReset={() => { setFilters({}); setPage(1); }}
      />
      <DataTable
        tableId="poll-statistics-list"
        columns={columns}
        dataSource={paginated as (PollStatRecord & Record<string, unknown>)[]}
        loading={false}
        rowKey="id"
        total={filtered.length}
        pageSize={pageSize}
        currentPage={page}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => setFilters({ ...filters })}
        scroll={{ x: 1100 }}
      />
    </ListPageLayout>
  );
}
