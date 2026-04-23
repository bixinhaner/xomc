import { useState, useMemo, useCallback } from 'react';
import { Button, Modal, Space, Table, Tag, message } from 'antd';
import { EyeOutlined, DownloadOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';

import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useT } from '@/hooks/useT';

import type {
  MMLTask,
  MMLTaskStatus,
  MMLExecuteType,
  MMLTaskResult,
  DeviceTaskResultItem,
} from '@core/types/mml';
import {
  useMMLTasks,
  useMMLTaskResults,
  useDeleteMMLTask,
} from '@core/hooks/api/useMML';

// -------------------------------------------------------------------------
// Display mappings — mml_tasks columns
// -------------------------------------------------------------------------

const EXECUTE_TYPE_TAGS: Record<MMLExecuteType, { color: string; key: string }> = {
  immediate: { color: 'green',  key: 'mml.immediateExecute' },
  suspended: { color: 'orange', key: 'mml.suspended' },
  scheduled: { color: 'blue',   key: 'mml.scheduledExecute' },
  periodic:  { color: 'purple', key: 'mml.periodicTask' },
};

const TASK_STATUS_TAGS: Record<MMLTaskStatus, { color: string; key: string }> = {
  pending:   { color: 'default',    key: 'mml.pendingStatus' },
  running:   { color: 'processing', key: 'mml.runningStatus' },
  paused:    { color: 'warning',    key: 'mml.pausedStatus' },
  completed: { color: 'success',    key: 'mml.completedStatus' },
  cancelled: { color: 'error',      key: 'mml.cancelledStatus' },
  failed:    { color: 'error',      key: 'mml.failedStatus' },
};

const TASK_RESULT_TAGS: Record<MMLTaskResult, { color: string; key: string }> = {
  success: { color: 'success', key: 'status.success' },
  partial: { color: 'warning', key: 'mml.partialSuccess' },
  failed:  { color: 'error',   key: 'status.failed' },
};

function formatTime(iso?: string | null): string {
  if (!iso) return '-';
  const d = dayjs(iso);
  return d.isValid() ? d.format('YYYY-MM-DD HH:mm:ss') : '-';
}

// Derive per-device result rows displayed in the 查看 modal.
// Each row mirrors image-5.png: 设备SN / 状态 / 执行时间 / 输出.
function deriveExecutionMs(r: DeviceTaskResultItem): number | undefined {
  if (typeof r.result?.executionTime === 'number' && r.result.executionTime > 0) {
    return r.result.executionTime;
  }
  if (r.startedAt && r.finishedAt) {
    const s = dayjs(r.startedAt);
    const e = dayjs(r.finishedAt);
    if (s.isValid() && e.isValid()) return e.diff(s, 'millisecond');
  }
  return undefined;
}

export default function TaskRecord() {
  const t = useT();

  // ---- paging / filter state -----------------------------------------------
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [filters, setFilters] = useState<{
    taskName?: string;
    status?: string;
    executeType?: string;
    result?: string;
  }>({});

  const { data, isLoading, refetch } = useMMLTasks({
    page,
    pageSize,
    taskName: filters.taskName,
    status: filters.status && filters.status !== 'all' ? filters.status : undefined,
    executeType: filters.executeType && filters.executeType !== 'all' ? filters.executeType : undefined,
    result: filters.result && filters.result !== 'all' ? filters.result : undefined,
  });
  const deleteTaskMutation = useDeleteMMLTask();

  const tasks = useMemo(() => data?.items ?? [], [data]);

  const filterFields: FilterField[] = useMemo(() => [
    {
      name: 'taskName',
      label: t('mml.taskName'),
      type: 'input',
      placeholder: t('mml.inputTaskNameRequired'),
    },
    {
      name: 'executeType',
      label: t('mml.type'),
      type: 'select',
      placeholder: t('mml.type'),
      options: [
        { label: t('common.all'), value: 'all' },
        { label: t('mml.immediateExecute'), value: 'immediate' },
        { label: t('mml.suspended'),       value: 'suspended' },
        { label: t('mml.scheduledExecute'), value: 'scheduled' },
        { label: t('mml.periodicTask'),    value: 'periodic' },
      ],
    },
    {
      name: 'status',
      label: t('mml.status'),
      type: 'select',
      placeholder: t('mml.status'),
      options: [
        { label: t('common.all'),                value: 'all' },
        { label: t('mml.pendingStatus'),         value: 'pending' },
        { label: t('mml.runningStatus'),         value: 'running' },
        { label: t('mml.pausedStatus'),          value: 'paused' },
        { label: t('mml.completedStatus'),       value: 'completed' },
        { label: t('mml.cancelledStatus'),       value: 'cancelled' },
        { label: t('mml.failedStatus'),          value: 'failed' },
      ],
    },
    {
      name: 'result',
      label: t('mml.result'),
      type: 'select',
      placeholder: t('mml.result'),
      options: [
        { label: t('common.all'),         value: 'all' },
        { label: t('status.success'),     value: 'success' },
        { label: t('mml.partialSuccess'), value: 'partial' },
        { label: t('status.failed'),      value: 'failed' },
      ],
    },
  ], [t]);

  const handleSearch = useCallback((values: Record<string, unknown>) => {
    setPage(1);
    setFilters({
      taskName: typeof values.taskName === 'string' ? values.taskName.trim() : undefined,
      status: typeof values.status === 'string' ? values.status : undefined,
      executeType: typeof values.executeType === 'string' ? values.executeType : undefined,
      result: typeof values.result === 'string' ? values.result : undefined,
    });
  }, []);

  const handleReset = useCallback(() => {
    setFilters({});
    setPage(1);
  }, []);

  const handleDelete = useCallback((task: MMLTask) => {
    deleteTaskMutation.mutate(task.id, {
      onSuccess: () => void message.success(t('mml.taskDeleted', { name: task.taskName })),
      onError: (err) => void message.error(
        t('mml.deleteFailed', { error: err instanceof Error ? err.message : 'Unknown' })
      ),
    });
  }, [deleteTaskMutation, t]);

  // ---- 查看 modal state ----------------------------------------------------
  const [viewing, setViewing] = useState<MMLTask | null>(null);
  const { data: resultsData, isLoading: resultsLoading } = useMMLTaskResults(
    viewing?.id ?? null,
    1,
    200,
  );

  const resultRows = useMemo(() => resultsData?.items ?? viewing?.results ?? [], [resultsData, viewing]);

  const handleExportResults = useCallback(() => {
    if (!viewing || resultRows.length === 0) return;
    const header = `${t('mml.deviceSn')},${t('mml.status')},${t('mml.executionTime')},${t('mml.output')}\n`;
    const rows = resultRows
      .map((item) => {
        const ms = deriveExecutionMs(item);
        const status = item.status ?? (item.result?.success ? 'success' : 'failed');
        const output = item.result?.rawOutput ?? '';
        return [
          item.deviceSn,
          status,
          ms ?? '',
          `"${String(output).replace(/"/g, '""')}"`,
        ].join(',');
      })
      .join('\n');
    const blob = new Blob(['﻿' + header + rows], { type: 'text/csv;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${viewing.taskName}_${t('mml.exportResult')}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  }, [viewing, resultRows, t]);

  const columns: DataTableColumn<MMLTask>[] = useMemo(() => [
    {
      key: 'operation',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 140,
      fixed: 'right',
      render: (_, record) => (
        <Space size={4}>
          <Button
            type="link"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => setViewing(record)}
          >
            {t('common.view')}
          </Button>
          <Button
            type="link"
            size="small"
            danger
            disabled={record.status === 'running'}
            onClick={() => handleDelete(record)}
          >
            {t('common.delete')}
          </Button>
        </Space>
      ),
    },
    { key: 'taskName', title: t('mml.taskName'), dataIndex: 'taskName', ellipsis: true },
    { key: 'creator',  title: t('mml.creator'),  dataIndex: 'creator', width: 100 },
    {
      key: 'executeType',
      title: t('mml.type'),
      dataIndex: 'executeType',
      width: 100,
      render: (v: MMLExecuteType) => {
        const tag = EXECUTE_TYPE_TAGS[v];
        return tag ? <Tag color={tag.color}>{t(tag.key)}</Tag> : <Tag>{v || '-'}</Tag>;
      },
    },
    {
      key: 'status',
      title: t('mml.status'),
      dataIndex: 'status',
      width: 100,
      render: (v: MMLTaskStatus) => {
        const tag = TASK_STATUS_TAGS[v];
        return tag ? <Tag color={tag.color}>{t(tag.key)}</Tag> : <Tag>{v || '-'}</Tag>;
      },
    },
    {
      key: 'progress',
      title: t('mml.progress'),
      width: 100,
      render: (_: unknown, record: MMLTask) => {
        const done = (record.successCount ?? 0) + (record.failedCount ?? 0);
        return `${done}/${record.totalDevices ?? 0}`;
      },
    },
    {
      key: 'result',
      title: t('mml.result'),
      dataIndex: 'result',
      width: 100,
      render: (v?: MMLTaskResult) => {
        if (!v) return '-';
        const tag = TASK_RESULT_TAGS[v];
        return tag ? <Tag color={tag.color}>{t(tag.key)}</Tag> : <Tag>{v}</Tag>;
      },
    },
    {
      key: 'startedAt',
      title: t('mml.startTime'),
      dataIndex: 'startedAt',
      width: 160,
      render: (v?: string) => formatTime(v),
    },
    {
      key: 'finishedAt',
      title: t('mml.endTime'),
      dataIndex: 'finishedAt',
      width: 160,
      render: (v?: string) => formatTime(v),
    },
    {
      key: 'createdAt',
      title: t('mml.createTime'),
      dataIndex: 'createdAt',
      width: 160,
      render: (v: string) => formatTime(v),
    },
  ], [t, handleDelete]);

  const resultColumns = useMemo(() => [
    { key: 'deviceSn', title: t('mml.deviceSn'), dataIndex: 'deviceSn', width: 160 },
    {
      key: 'status',
      title: t('mml.status'),
      dataIndex: 'status',
      width: 100,
      render: (_: unknown, row: DeviceTaskResultItem) => {
        const explicit = row.status;
        const implicit = row.result?.success ? 'success' : 'failed';
        const s = explicit ?? implicit;
        if (s === 'success' || s === 'completed') return <Tag color="success">{t('status.success')}</Tag>;
        if (s === 'failed') return <Tag color="error">{t('status.failed')}</Tag>;
        return <Tag>{s || '-'}</Tag>;
      },
    },
    {
      key: 'executionTime',
      title: t('mml.executionTime'),
      width: 120,
      align: 'right' as const,
      render: (_: unknown, row: DeviceTaskResultItem) => deriveExecutionMs(row) ?? '-',
    },
    {
      key: 'output',
      title: t('mml.output'),
      ellipsis: true,
      render: (_: unknown, row: DeviceTaskResultItem) => row.result?.rawOutput || '-',
    },
  ], [t]);

  return (
    <ListPageLayout title={t('nav.mml.taskRecord')}>
      <FilterBar
        filterId="mml-task-record"
        fields={filterFields}
        onSearch={handleSearch}
        onReset={handleReset}
      />
      <DataTable<MMLTask>
        tableId="mml-task-record"
        columns={columns}
        dataSource={tasks}
        loading={isLoading}
        rowKey="id"
        total={data?.total ?? 0}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => void refetch()}
        scroll={{ x: 1400 }}
      />

      <Modal
        title={viewing ? t('mml.executionResult', { name: viewing.taskName }) : t('common.view')}
        open={Boolean(viewing)}
        onCancel={() => setViewing(null)}
        footer={
          <Space>
            <Button icon={<DownloadOutlined />} onClick={handleExportResults} disabled={!resultRows.length}>
              {t('mml.exportResult')}
            </Button>
            <Button onClick={() => setViewing(null)}>{t('common.close')}</Button>
          </Space>
        }
        width={820}
        destroyOnClose
      >
        {viewing && (
          <>
            {resultRows.length > 0 ? (
              <Table
                size="small"
                rowKey={(row) => row.deviceSn || Math.random().toString(36).slice(2)}
                dataSource={resultRows}
                columns={resultColumns}
                loading={resultsLoading}
                pagination={false}
                scroll={{ y: 360 }}
              />
            ) : (
              <div style={{ padding: 24, textAlign: 'center', color: '#999' }}>
                {resultsLoading ? t('common.loading') : t('mml.noExecutionResult')}
              </div>
            )}
          </>
        )}
      </Modal>
    </ListPageLayout>
  );
}
