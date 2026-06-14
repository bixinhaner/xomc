import { useState, useMemo, useCallback } from 'react';
import { Tag, App } from 'antd';
import { DeleteOutlined } from '@ant-design/icons';

import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useT } from '@/hooks/useT';
import { usePMFileDevices, useBatchDeletePMFiles } from '@core/hooks/api/usePerformance';
import type { PMFileDeviceItem } from '@core/services/api/pmApi';

const PAGE_SIZE = 20;

export default function PerformanceFiles() {
  const t = useT();
  const { modal, message } = App.useApp();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(PAGE_SIZE);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);

  const keyword = typeof filters.keyword === 'string' ? filters.keyword.trim() : '';

  const params = useMemo(
    () => ({ page, pageSize, ...(keyword ? { keyword } : {}) }),
    [page, pageSize, keyword]
  );

  const { data, isLoading } = usePMFileDevices(params);
  const batchDelete = useBatchDeletePMFiles();

  const rows = data?.items ?? [];
  const total = data?.total ?? 0;

  const filterFields: FilterField[] = useMemo(
    () => [
      { name: 'keyword', label: t('device.sn'), type: 'input', placeholder: `${t('device.sn')} / ${t('perf.siteName')}` },
    ],
    [t]
  );

  const handleBatchDelete = useCallback(
    (keys: React.Key[]) => {
      const sns = keys.map((k) => String(k));
      if (sns.length === 0) return;
      modal.confirm({
        title: t('perf.batchDeleteFiles'),
        content: t('backup.deviceCountUnit', { count: sns.length }),
        okText: t('common.confirm'),
        cancelText: t('common.cancel'),
        okButtonProps: { danger: true },
        onOk: () =>
          batchDelete.mutateAsync(sns).then((res) => {
            if (res.failed.length > 0) {
              void message.warning(
                t('perf.batchDeletePartial', { succeeded: res.succeeded.length, failed: res.failed.length })
              );
            } else {
              void message.success(t('perf.batchDeleteSuccess', { count: res.succeeded.length }));
            }
            setSelectedKeys([]);
          }),
      });
    },
    [batchDelete, modal, message, t]
  );

  const batchActions: BatchAction[] = useMemo(
    () => [
      {
        key: 'delete',
        label: t('perf.batchDeleteFiles'),
        icon: <DeleteOutlined />,
        danger: true,
        onClick: handleBatchDelete,
      },
    ],
    [handleBatchDelete, t]
  );

  const columns: DataTableColumn<PMFileDeviceItem & Record<string, unknown>>[] = useMemo(
    () => [
      { key: 'deviceSn', title: t('device.sn'), dataIndex: 'deviceSn', width: 200, mono: true, copyable: true },
      {
        key: 'siteName',
        title: t('perf.siteName'),
        dataIndex: 'siteName',
        width: 200,
        ellipsis: true,
        render: (val) => (val as string) || '-',
      },
      {
        key: 'productClass',
        title: t('device.productClass'),
        dataIndex: 'productClass',
        width: 140,
        render: (val) => (val as string) || '-',
      },
      {
        key: 'fileCount',
        title: t('perf.fileCount'),
        dataIndex: 'fileCount',
        width: 100,
        render: (val) => ((val as number) ?? 0).toLocaleString(),
      },
      {
        key: 'lastCollectTime',
        title: t('perf.lastCollect'),
        dataIndex: 'lastCollectTime',
        width: 180,
        render: (val) => (val as string) || '-',
      },
      {
        key: 'reporting',
        title: t('table.status'),
        dataIndex: 'reporting',
        width: 100,
        render: (val) =>
          (val as boolean) ? (
            <Tag color="success">{t('perf.reporting')}</Tag>
          ) : (
            <Tag color="default">{t('perf.silent')}</Tag>
          ),
      },
    ],
    [t]
  );

  return (
    <ListPageLayout title={t('nav.performance.files')}>
      <FilterBar
        filterId="performance-files"
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
      <DataTable<PMFileDeviceItem & Record<string, unknown>>
        tableId="performance-files"
        columns={columns}
        dataSource={rows as (PMFileDeviceItem & Record<string, unknown>)[]}
        loading={isLoading}
        rowKey="deviceSn"
        selectable
        selectedRowKeys={selectedKeys}
        onSelectionChange={(keys) => setSelectedKeys(keys)}
        batchActions={batchActions}
        total={total}
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
