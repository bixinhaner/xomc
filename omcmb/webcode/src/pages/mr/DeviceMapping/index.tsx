import { useState } from 'react';
import { Button, Tag, Space, Switch, message } from 'antd';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useMRMappings, useToggleMRMapping } from '@core/hooks/api/useMR';
import type { MRDeviceMapping } from '@core/mock/data/mr';

const filterFields: FilterField[] = [
  { name: 'deviceSn', label: '设备SN', type: 'input', placeholder: '请输入设备SN' },
  {
    name: 'enabled',
    label: '映射状态',
    type: 'select',
    options: [
      { label: '已启用', value: 'true' },
      { label: '已禁用', value: 'false' },
    ],
  },
];

const mockMappings: MRDeviceMapping[] = [
  { id: 'map-001', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', cellId: 'cell-001', cellName: '北京-小区-001', samplingInterval: 15, enabled: true, totalRecords: 12500, lastCollectTime: '2024-06-01T08:00:00.000Z' },
  { id: 'map-002', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', cellId: 'cell-002', cellName: '北京-小区-002', samplingInterval: 15, enabled: true, totalRecords: 11200, lastCollectTime: '2024-06-01T08:00:00.000Z' },
  { id: 'map-003', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', cellId: 'cell-003', cellName: '北京-小区-003', samplingInterval: 15, enabled: false, totalRecords: 8900 },
  { id: 'map-004', deviceSn: 'ENB00002', deviceName: '北京-eNB-0002', cellId: 'cell-004', cellName: '北京-小区-004', samplingInterval: 30, enabled: true, totalRecords: 15600, lastCollectTime: '2024-06-01T08:00:00.000Z' },
  { id: 'map-005', deviceSn: 'GNB00001', deviceName: '北京-gNB-0001', cellId: 'nr-cell-001', cellName: '北京-NR小区-001', samplingInterval: 15, enabled: true, totalRecords: 5400, lastCollectTime: '2024-06-01T08:00:00.000Z' },
  { id: 'map-006', deviceSn: 'ENB00010', deviceName: '上海-eNB-0001', cellId: 'cell-010', cellName: '上海-小区-010', samplingInterval: 15, enabled: true, totalRecords: 9800, lastCollectTime: '2024-06-01T08:00:00.000Z' },
];

export default function DeviceMapping() {
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [localMappings, setLocalMappings] = useState<MRDeviceMapping[]>(mockMappings);

  const { data, isLoading, refetch } = useMRMappings({
    deviceSn: filters.deviceSn as string | undefined,
    enabled: filters.enabled !== undefined ? filters.enabled === 'true' : undefined,
    page,
    pageSize,
  });

  const toggleMapping = useToggleMRMapping();

  const allMappings = (data?.items?.length ?? 0) > 0
    ? (data?.items ?? []) as MRDeviceMapping[]
    : localMappings.filter((m) => {
        if (filters.deviceSn && !m.deviceSn.includes(String(filters.deviceSn))) return false;
        if (filters.enabled !== undefined) {
          const enabledFilter = filters.enabled === 'true';
          if (m.enabled !== enabledFilter) return false;
        }
        return true;
      });

  const startIndex = (page - 1) * pageSize;
  const paginated = data?.items ? allMappings : allMappings.slice(startIndex, startIndex + pageSize);

  const columns: DataTableColumn<MRDeviceMapping & Record<string, unknown>>[] = [
    { key: 'deviceSn', title: '设备SN', dataIndex: 'deviceSn', width: 130, render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span> },
    { key: 'deviceName', title: '设备名称', dataIndex: 'deviceName', ellipsis: true, width: 160 },
    { key: 'cellId', title: '小区ID', dataIndex: 'cellId', width: 130, render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span> },
    { key: 'cellName', title: '小区名称', dataIndex: 'cellName', ellipsis: true, width: 150 },
    { key: 'samplingInterval', title: '采样间隔(分钟)', dataIndex: 'samplingInterval', width: 130 },
    { key: 'totalRecords', title: '采集记录数', dataIndex: 'totalRecords', width: 110 },
    {
      key: 'enabled', title: '映射状态', dataIndex: 'enabled', width: 100,
      render: (val, record) => {
        const m = record as MRDeviceMapping;
        return (
          <Switch
            checked={Boolean(val)}
            size="small"
            onChange={(checked) => {
              if (data?.items) {
                toggleMapping.mutate({ id: m.id, enabled: checked }, {
                  onSuccess: () => void message.success(`已${checked ? '启用' : '禁用'} ${m.deviceSn} ${m.cellId} 映射`),
                });
              } else {
                setLocalMappings((prev) => prev.map((mp) => mp.id === m.id ? { ...mp, enabled: checked } : mp));
                void message.success(`已${checked ? '启用' : '禁用'} ${m.deviceSn} ${m.cellId} 映射`);
              }
            }}
          />
        );
      },
    },
    {
      key: 'lastCollectTime', title: '最后采集时间', dataIndex: 'lastCollectTime', width: 160,
      render: (val) => val ? new Date(String(val)).toLocaleString('zh-CN') : '—',
    },
    {
      key: 'actions', title: '操作', dataIndex: 'id', width: 80,
      render: () => <Button type="link" size="small">编辑</Button>,
    },
  ];

  return (
    <ListPageLayout title="设备小区映射" subtitle="管理设备与小区的MR数据采集映射关系">
      <FilterBar
        filterId="mr-device-mapping-filter"
        fields={filterFields}
        onSearch={(vals) => { setFilters(vals); setPage(1); }}
        onReset={() => { setFilters({}); setPage(1); }}
      />
      <DataTable
        tableId="mr-device-mapping-list"
        columns={columns}
        dataSource={paginated as (MRDeviceMapping & Record<string, unknown>)[]}
        loading={isLoading}
        rowKey="id"
        total={data?.total ?? allMappings.length}
        pageSize={pageSize}
        currentPage={page}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => void refetch()}
        scroll={{ x: 1000 }}
      />
    </ListPageLayout>
  );
}
