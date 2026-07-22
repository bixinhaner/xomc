/**
 * KpiExportTasksPanel — 任务管理 ▸「KPI 导出」Tab。
 *
 * 设计：~/Documents/notes/PM功能设计/kpi-export-design-20260604.md §3.3 / §6.2
 * 列：任务名 | 来源 | 状态 | 进度 | 行数 | 创建人 | 创建时间 | 操作(删除 / 失败重试)。不放下载。
 * 列表自动轮询（hook 内：有 pending/running 任务时定时重取），看状态由进行中刷新到成功/失败。
 */
import { useMemo, useState } from 'react';
import { Button, Card, Input, Modal, Progress, Select, Space, Tag, Tooltip, message } from 'antd';
import { DeleteOutlined, ReloadOutlined, RedoOutlined } from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';
import {
  useKpiExportTasks,
  useDeleteKpiExport,
  useRetryKpiExport,
} from '@core/hooks/api/useKpiExport';
import type { KpiExportSource, KpiExportStatus, KpiExportTask } from '@core/types/kpiExport';
import { SOURCE_LABEL_KEY, STATUS_TAG, statusProgress } from './shared';
import { formatSystemTime } from '@core/utils/systemTime';

const LIST_LIMIT = 500;

export default function KpiExportTasksPanel() {
  const t = useT();
  const [nameFilter, setNameFilter] = useState('');
  const [sourceFilter, setSourceFilter] = useState<KpiExportSource | ''>('');
  const [statusFilter, setStatusFilter] = useState<KpiExportStatus | ''>('');

  const { data, isLoading, refetch } = useKpiExportTasks({
    sourceType: sourceFilter || undefined,
    status: statusFilter || undefined,
    limit: LIST_LIMIT,
  });
  const deleteMutation = useDeleteKpiExport();
  const retryMutation = useRetryKpiExport();

  const rows = useMemo(() => {
    const list = data ?? [];
    const kw = nameFilter.trim().toLowerCase();
    return kw ? list.filter((x) => x.taskName.toLowerCase().includes(kw)) : list;
  }, [data, nameFilter]);

  function confirmDelete(id: string) {
    Modal.confirm({
      title: t('kpiExport.msg.deleteConfirmTitle'),
      content: t('kpiExport.msg.deleteConfirmContent'),
      okType: 'danger',
      onOk: async () => {
        try {
          await deleteMutation.mutateAsync(id);
          message.success(t('kpiExport.msg.deleteSuccess'));
        } catch {
          message.error(t('kpiExport.msg.deleteFailed'));
        }
      },
    });
  }

  async function handleRetry(record: KpiExportTask) {
    try {
      await retryMutation.mutateAsync(record);
      message.success(t('kpiExport.msg.retrySuccess'));
    } catch {
      message.error(t('kpiExport.msg.retryFailed'));
    }
  }

  const columns: DataTableColumn<KpiExportTask>[] = [
    { key: 'taskName', title: t('kpiExport.col.taskName'), dataIndex: 'taskName', width: 280, ellipsis: true },
    {
      key: 'source',
      title: t('kpiExport.col.source'),
      dataIndex: 'sourceType',
      width: 110,
      render: (v) => <Tag>{t(SOURCE_LABEL_KEY[v as KpiExportSource])}</Tag>,
    },
    {
      key: 'status',
      title: t('kpiExport.col.status'),
      dataIndex: 'status',
      width: 110,
      render: (v, r) => {
        const cfg = STATUS_TAG[v as KpiExportStatus];
        const tag = <Tag color={cfg.color}>{t(cfg.labelKey)}</Tag>;
        return r.status === 'failed' && r.error ? (
          <Tooltip title={r.error}>{tag}</Tooltip>
        ) : (
          tag
        );
      },
    },
    {
      key: 'progress',
      title: t('kpiExport.col.progress'),
      dataIndex: 'status',
      width: 160,
      render: (_, r) => {
        const pct = statusProgress(r.status);
        const pstatus =
          r.status === 'failed' ? 'exception' : r.status === 'succeeded' ? 'success' : 'active';
        return <Progress percent={pct} size="small" status={pstatus} />;
      },
    },
    {
      key: 'rowCount',
      title: t('kpiExport.col.rowCount'),
      dataIndex: 'rowCount',
      width: 110,
      render: (v) => (typeof v === 'number' ? v.toLocaleString() : '—'),
    },
    { key: 'createUser', title: t('kpiExport.col.createUser'), dataIndex: 'createUser', width: 140 },
    {
      key: 'createdAt',
      title: t('kpiExport.col.createdAt'),
      dataIndex: 'createdAt',
      width: 180,
      render: (v) => (v ? formatSystemTime(v as string, { format: 'YYYY-MM-DD HH:mm:ss', placeholder: '-' }) : '—'),
    },
    {
      key: 'actions',
      title: t('kpiExport.col.actions'),
      width: 180,
      fixed: 'right',
      render: (_, record) => (
        <Space>
          {record.status === 'failed' && (
            <Button
              type="link"
              size="small"
              icon={<RedoOutlined />}
              loading={retryMutation.isPending}
              onClick={() => {
                void handleRetry(record);
              }}
            >
              {t('kpiExport.action.retry')}
            </Button>
          )}
          <Button
            type="link"
            danger
            size="small"
            icon={<DeleteOutlined />}
            onClick={() => confirmDelete(record.id)}
          >
            {t('kpiExport.action.delete')}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16, flex: 1, minHeight: 0 }}>
      <Card size="small">
        <Space wrap>
          <Input
            placeholder={t('kpiExport.filter.taskName')}
            allowClear
            value={nameFilter}
            onChange={(e) => setNameFilter(e.target.value)}
            style={{ width: 220 }}
          />
          <Select<KpiExportSource | ''>
            placeholder={t('kpiExport.filter.source')}
            allowClear
            value={sourceFilter || undefined}
            onChange={(v) => setSourceFilter(v ?? '')}
            style={{ width: 150 }}
            options={[
              { value: 'dashboard', label: t('kpiExport.source.dashboard') },
              { value: 'device_view', label: t('kpiExport.source.deviceView') },
              { value: 'kpi_query', label: t('kpiExport.source.kpiQuery') },
              { value: 'adhoc', label: t('kpiExport.source.adhoc') },
            ]}
          />
          <Select<KpiExportStatus | ''>
            placeholder={t('kpiExport.filter.status')}
            allowClear
            value={statusFilter || undefined}
            onChange={(v) => setStatusFilter(v ?? '')}
            style={{ width: 150 }}
            options={[
              { value: 'pending', label: t('kpiExport.status.pending') },
              { value: 'running', label: t('kpiExport.status.running') },
              { value: 'succeeded', label: t('kpiExport.status.succeeded') },
              { value: 'failed', label: t('kpiExport.status.failed') },
            ]}
          />
          <Button icon={<ReloadOutlined />} onClick={() => refetch()}>
            {t('kpiExport.action.refresh')}
          </Button>
        </Space>
      </Card>
      <DataTable<KpiExportTask>
        tableId="kpi-export-tasks"
        columns={columns}
        dataSource={rows}
        loading={isLoading}
        rowKey={(r) => r.id}
      />
    </div>
  );
}
