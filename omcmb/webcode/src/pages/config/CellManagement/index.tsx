import { useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button, Tag } from 'antd';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useT } from '@/hooks/useT';
import { useDeviceList } from '@core/hooks/api/useDevices';
import type { Device } from '@core/types/device';

// 设备记录中确实承载小区信息的字段（来自 Device 监控扩展字段组）。
// 小区标识统一使用后端 cellIdentity 字段。
function cellIdentity(d: Device): string {
  return d.cellId || d.eci || d.nrCellId || d.enbId || d.gnbId || '';
}

const PAGE_SIZE = 20;

export default function CellManagement() {
  const t = useT();
  const navigate = useNavigate();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(PAGE_SIZE);

  // v3 一次性拉 300 台设备在客户端派生小区视图，这里保持同一数据源与派生口径。
  const { data, isLoading } = useDeviceList({ page: 1, pageSize: 300 });

  // 仅保留承载小区标识的设备（与 v3 一致）
  const cells = useMemo(() => (data?.items ?? []).filter((d) => cellIdentity(d) !== ''), [data]);

  const keyword = typeof filters.keyword === 'string' ? filters.keyword.trim().toLowerCase() : '';
  const tech = typeof filters.tech === 'string' ? filters.tech : 'all';

  const filtered = useMemo(
    () =>
      cells.filter((d) => {
        if (tech === 'lte' && d.networkType?.toLowerCase() !== 'lte') return false;
        if (tech === 'nr' && !(d.networkType?.toLowerCase() === 'nr' || d.nrCellId)) return false;
        if (!keyword) return true;
        return (
          cellIdentity(d).toLowerCase().includes(keyword) ||
          d.sn?.toLowerCase().includes(keyword) ||
          d.name?.toLowerCase().includes(keyword)
        );
      }),
    [cells, keyword, tech]
  );

  // 客户端分页：DataTable 在传入 onPageChange 时按服务端分页处理（不再自动切片），
  // 这里数据是前端派生的全量集合，需自行切片。
  const pagedData = useMemo(
    () => filtered.slice((page - 1) * pageSize, page * pageSize),
    [filtered, page, pageSize]
  );

  const filterFields: FilterField[] = useMemo(
    () => [
      { name: 'keyword', label: t('nav.config.cell'), type: 'input', placeholder: `Cell ID / ${t('device.sn')} / ${t('table.name')}` },
      {
        name: 'tech',
        label: t('table.type'),
        type: 'select',
        options: [
          { label: 'LTE', value: 'lte' },
          { label: 'NR', value: 'nr' },
        ],
      },
    ],
    [t]
  );

  const columns: DataTableColumn<Device & Record<string, unknown>>[] = useMemo(
    () => [
      {
        key: 'cellIdentity',
        title: t('nav.config.cell'),
        dataIndex: 'cellId',
        width: 160,
        mono: true,
        copyable: true,
        render: (_, record) => cellIdentity(record) || '-',
      },
      {
        key: 'name',
        title: t('device.name'),
        dataIndex: 'name',
        width: 200,
        ellipsis: true,
        render: (val, record) => (val as string) || record.sn || '-',
      },
      { key: 'sn', title: t('device.sn'), dataIndex: 'sn', width: 160, mono: true },
      {
        key: 'networkType',
        title: t('table.type'),
        dataIndex: 'networkType',
        width: 90,
        render: (val) => ((val as string) || '-').toUpperCase(),
      },
      { key: 'pci', title: 'PCI', dataIndex: 'pci', width: 80, render: (val) => (val as string) || '-' },
      { key: 'tac', title: 'TAC', dataIndex: 'tac', width: 90, render: (val) => (val as string) || '-' },
      { key: 'band', title: 'Band', dataIndex: 'band', width: 90, render: (val) => (val as string) || '-' },
      { key: 'bandwidth', title: t('perf.unit'), dataIndex: 'bandwidth', width: 100, render: (val) => (val as string) || '-' },
      {
        key: 'cellStatus',
        title: t('table.status'),
        dataIndex: 'cellStatus',
        width: 110,
        render: (val, record) => (val as string) || record.opState || '-',
      },
      {
        key: 'isOnline',
        title: t('device.connStatus'),
        dataIndex: 'isOnline',
        width: 100,
        render: (val) =>
          (val as boolean) ? (
            <Tag color="green">{t('status.online')}</Tag>
          ) : (
            <Tag color="red">{t('status.offline')}</Tag>
          ),
      },
      {
        key: 'action',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 120,
        fixed: 'right',
        render: (_, record) => (
          <Button type="link" size="small" onClick={() => navigate(`/device/list?sn=${encodeURIComponent(record.sn)}`)}>
            {t('common.view')}
          </Button>
        ),
      },
    ],
    [t, navigate]
  );

  return (
    <ListPageLayout title={t('nav.config.cell')}>
      <FilterBar
        filterId="cell-management"
        fields={filterFields}
        onSearch={(vals) => {
          setFilters(vals);
          setPage(1);
        }}
        onReset={() => {
          setFilters({});
          setPage(1);
        }}
      />
      <DataTable<Device & Record<string, unknown>>
        tableId="cell-management"
        columns={columns}
        dataSource={pagedData as (Device & Record<string, unknown>)[]}
        loading={isLoading}
        rowKey="id"
        total={filtered.length}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, s) => {
          setPage(p);
          setPageSize(s);
        }}
        scroll={{ x: 'max-content', y: 'calc(100vh - 320px)' }}
      />
    </ListPageLayout>
  );
}
