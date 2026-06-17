import { useState, useMemo } from 'react';
import { formatSystemTime } from '@core/utils/systemTime';
import { Tag } from 'antd';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';

type HeartbeatType = 'request' | 'response' | 'timeout';

interface HeartbeatRecord {
  id: string;
  timestamp: string;
  messageType: HeartbeatType;
  deviceSn: string;
  deviceName: string;
  interval?: number;
  latency?: number;
}

const mockHeartbeats: HeartbeatRecord[] = [
  { id: 'hb-001', timestamp: '2024-06-01T08:00:00.000Z', messageType: 'request', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', interval: 30 },
  { id: 'hb-002', timestamp: '2024-06-01T08:00:00.150Z', messageType: 'response', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', latency: 150 },
  { id: 'hb-003', timestamp: '2024-06-01T08:00:30.000Z', messageType: 'request', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', interval: 30 },
  { id: 'hb-004', timestamp: '2024-06-01T08:00:30.120Z', messageType: 'response', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', latency: 120 },
  { id: 'hb-005', timestamp: '2024-06-01T08:01:00.000Z', messageType: 'request', deviceSn: 'GNB00001', deviceName: '北京-gNB-0001', interval: 30 },
  { id: 'hb-006', timestamp: '2024-06-01T08:01:30.000Z', messageType: 'timeout', deviceSn: 'ENB00002', deviceName: '北京-eNB-0002', interval: 30 },
  { id: 'hb-007', timestamp: '2024-06-01T08:02:00.000Z', messageType: 'request', deviceSn: 'GNB00001', deviceName: '北京-gNB-0001', interval: 30 },
  { id: 'hb-008', timestamp: '2024-06-01T08:02:00.200Z', messageType: 'response', deviceSn: 'GNB00001', deviceName: '北京-gNB-0001', latency: 200 },
  { id: 'hb-009', timestamp: '2024-06-01T08:02:30.000Z', messageType: 'timeout', deviceSn: 'ENB00002', deviceName: '北京-eNB-0002', interval: 30 },
  { id: 'hb-010', timestamp: '2024-06-01T08:03:00.000Z', messageType: 'request', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', interval: 30 },
];

const typeColorMap: Record<HeartbeatType, string> = {
  request: 'default',
  response: 'green',
  timeout: 'red',
};

const typeLabelMap: Record<HeartbeatType, string> = {
  request: '心跳请求',
  response: '心跳响应',
  timeout: '心跳超时',
};

export default function HeartbeatLog() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const filtered = mockHeartbeats.filter((r) => {
    if (filters.deviceSn && !r.deviceSn.includes(String(filters.deviceSn)) && !r.deviceName.includes(String(filters.deviceSn))) return false;
    if (filters.messageType && r.messageType !== filters.messageType) return false;
    return true;
  });

  const startIndex = (page - 1) * pageSize;
  const paginated = filtered.slice(startIndex, startIndex + pageSize);

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'deviceSn', label: t('device.sn'), type: 'input', placeholder: t('device.sn') },
    { name: 'timeRange', label: t('perf.timeRange'), type: 'date-range' },
    {
      name: 'messageType',
      label: t('table.type'),
      type: 'select',
      options: [
        { label: '心跳请求', value: 'request' },
        { label: '心跳响应', value: 'response' },
        { label: '心跳超时', value: 'timeout' },
      ],
    },
  ], [t]);

  const columns: DataTableColumn<HeartbeatRecord & Record<string, unknown>>[] = useMemo(() => [
    {
      key: 'timestamp',
      title: t('table.time'),
      dataIndex: 'timestamp',
      width: 190,
      render: (val) => (
        <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{formatSystemTime(String(val))}</span>
      ),
    },
    {
      key: 'messageType',
      title: t('table.type'),
      dataIndex: 'messageType',
      width: 120,
      render: (val) => {
        const tp = val as HeartbeatType;
        return <Tag color={typeColorMap[tp]}>{typeLabelMap[tp]}</Tag>;
      },
    },
    {
      key: 'deviceSn',
      title: t('device.sn'),
      dataIndex: 'deviceSn',
      width: 130,
      render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span>,
    },
    { key: 'deviceName', title: t('device.name'), dataIndex: 'deviceName', width: 180, ellipsis: true },
    {
      key: 'interval',
      title: t('perf.granularity'),
      dataIndex: 'interval',
      width: 120,
      render: (val) => val ? `${String(val)} 秒` : '—',
    },
    {
      key: 'latency',
      title: t('perf.value'),
      dataIndex: 'latency',
      width: 120,
      render: (val) => {
        if (!val) return '—';
        const ms = Number(val);
        const color = ms < 100 ? '#52c41a' : ms < 500 ? '#faad14' : '#ff4d4f';
        return <span style={{ color, fontWeight: ms > 300 ? 600 : 400 }}>{ms} ms</span>;
      },
    },
  ], [t]);

  return (
    <ListPageLayout title={t('nav.log.heartbeat')} subtitle={t('nav.log.heartbeat')}>
      <FilterBar
        filterId="heartbeat-log-filter"
        fields={filterFields}
        onSearch={(vals) => { setFilters(vals); setPage(1); }}
        onReset={() => { setFilters({}); setPage(1); }}
      />
      <DataTable
        tableId="heartbeat-log-list"
        columns={columns}
        dataSource={paginated as (HeartbeatRecord & Record<string, unknown>)[]}
        loading={false}
        rowKey="id"
        total={filtered.length}
        pageSize={pageSize}
        currentPage={page}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => setFilters({ ...filters })}
        scroll={{ x: 900 }}
      />
    </ListPageLayout>
  );
}
