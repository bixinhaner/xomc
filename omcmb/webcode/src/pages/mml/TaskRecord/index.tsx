import { useState, useMemo, useCallback } from 'react';
import type { Key } from 'react';
import { Button, Empty, Modal, Pagination, Popover, Space, Table, Tag, Tooltip, Typography, message } from 'antd';
import { DeleteOutlined, ProfileOutlined, StopOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
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
  MMLTaskOrigin,
  DeviceTaskResultItem,
} from '@core/types/mml';
import {
  useMMLTasks,
  useMMLTaskResults,
  useCancelMMLTasks,
  useDeleteMMLTasks,
} from '@core/hooks/api/useMML';
import { getMmlTaskProgress } from '@core/utils/mmlTaskProgress';
import {
  parseMmlDeviceTaskResult,
  type ParsedMmlResult,
  type ParsedParamValue,
} from '@core/utils/mmlResultParser';
import { parseMmlCommandDisplay } from '@core/utils/mmlCommandDisplay';

// -------------------------------------------------------------------------
// Display mappings — mml_tasks columns
// -------------------------------------------------------------------------

const EXECUTE_TYPE_TAGS: Record<MMLExecuteType, { color: string; key: string }> = {
  immediate: { color: 'green',  key: 'mml.immediateExecute' },
  suspended: { color: 'orange', key: 'mml.suspended' },
  scheduled: { color: 'blue',   key: 'mml.scheduledExecute' },
  periodic:  { color: 'purple', key: 'mml.periodicTask' },
};

const TASK_ORIGIN_TAGS: Record<MMLTaskOrigin, { color: string; key: string }> = {
  console: { color: 'cyan', key: 'mml.taskOrigin.console' },
  script: { color: 'purple', key: 'mml.taskOrigin.script' },
};

const TASK_STATUS_TAGS: Record<MMLTaskStatus, { color: string; key: string }> = {
  pending:   { color: 'default',    key: 'mml.pendingStatus' },
  running:   { color: 'processing', key: 'mml.runningStatus' },
  paused:    { color: 'warning',    key: 'mml.pausedStatus' },
  completed: { color: 'success',    key: 'mml.completedStatus' },
  cancelled: { color: 'error',      key: 'mml.cancelledStatus' },
  failed:    { color: 'error',      key: 'mml.failedStatus' },
};

const TASK_RESULT_PAGE_SIZE = 20;
const CANCELLABLE_TASK_STATUSES = new Set<MMLTaskStatus>(['pending', 'running', 'paused']);

const DEVICE_RESULT_STATUS_TAGS: Record<string, { color: string; key: string }> = {
  pending: { color: 'default', key: 'mml.pendingStatus' },
  running: { color: 'processing', key: 'mml.runningStatus' },
  completed: { color: 'success', key: 'mml.completedStatus' },
  failed: { color: 'error', key: 'mml.failedStatus' },
};

function formatTime(iso?: string | null): string {
  if (!iso) return '-';
  const d = dayjs(iso);
  return d.isValid() ? d.format('YYYY-MM-DD HH:mm:ss') : '-';
}

function stringifyMessagePayload(payload: unknown): string {
  if (payload === null || payload === undefined || payload === '') return '';
  if (typeof payload === 'string') return payload;
  try {
    return JSON.stringify(payload, null, 2);
  } catch {
    return String(payload);
  }
}

function requestMessageText(row: DeviceTaskResultItem | null): string {
  if (!row?.request) return '';
  if (row.request.rawRequest) return row.request.rawRequest;
  return stringifyMessagePayload({
    method: row.request.method,
    ...(row.request.cwmpId ? { cwmp_id: row.request.cwmpId } : {}),
    ...(row.request.commandKey ? { command_key: row.request.commandKey } : {}),
    ...(row.request.payload !== undefined ? { params: row.request.payload } : {}),
  });
}

function responseMessageText(row: DeviceTaskResultItem | null): string {
  if (!row) return '';
  if (row.result?.rawOutput) return row.result.rawOutput;
  if (row.result?.parsedData) return JSON.stringify(row.result.parsedData, null, 2);
  return '';
}

function getErrorMessage(error: unknown): string {
  if (!error || typeof error !== 'object') return '';
  const candidate = error as { userMessage?: unknown; message?: unknown };
  if (typeof candidate.userMessage === 'string' && candidate.userMessage.trim()) {
    return candidate.userMessage;
  }
  if (typeof candidate.message === 'string' && candidate.message.trim()) {
    return candidate.message;
  }
  return '';
}

function hasMessageText(row: DeviceTaskResultItem | null): boolean {
  return Boolean(requestMessageText(row) || responseMessageText(row));
}

function resultCommandText(row: DeviceTaskResultItem): string {
  return row.mmlScript || row.planRawLine || row.commandCode || '-';
}

function operationColor(operation: string): string {
  switch (operation.toUpperCase()) {
    case 'LST':
      return 'blue';
    case 'MOD':
      return 'green';
    case 'ADD':
      return 'purple';
    case 'DEL':
    case 'RMV':
      return 'orange';
    default:
      return 'default';
  }
}

function parsedResult(row: DeviceTaskResultItem | null): ParsedMmlResult | null {
  if (!row?.result?.parsedData) return null;
  return parseMmlDeviceTaskResult(row.result.parsedData);
}

function hasParsedDetail(row: DeviceTaskResultItem): boolean {
  const parsed = parsedResult(row);
  if (!parsed) return false;
  if (parsed.kind === 'gpv') return (parsed.params?.length ?? 0) > 0;
  return parsed.kind !== 'unknown';
}

function parsedResultSummary(parsed: ParsedMmlResult, t: (key: string, values?: Record<string, string | number>) => string): string {
  switch (parsed.kind) {
    case 'spv':
      return parsed.status === 1
        ? t('mml.taskResult.parsed.spv.reboot')
        : t('mml.taskResult.parsed.spv.immediate');
    case 'add':
      return t('mml.taskResult.parsed.add.success', { n: parsed.instanceNumber ?? '-' });
    case 'delete':
      return t('mml.taskResult.parsed.delete.success');
    case 'reboot':
      return t('mml.taskResult.parsed.reboot.success');
    default:
      return t('mml.taskResult.parsed.notParsable');
  }
}

export default function TaskRecord() {
  const t = useT();

  // ---- paging / filter state -----------------------------------------------
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [filters, setFilters] = useState<{
    taskName?: string;
    taskOrigin?: string;
    status?: string;
    executeType?: string;
    result?: string;
  }>({});

  const { data, isLoading, refetch } = useMMLTasks({
    page,
    pageSize,
    taskName: filters.taskName,
    taskOrigin: filters.taskOrigin && filters.taskOrigin !== 'all' ? filters.taskOrigin : undefined,
    status: filters.status && filters.status !== 'all' ? filters.status : undefined,
    executeType: filters.executeType && filters.executeType !== 'all' ? filters.executeType : undefined,
    result: filters.result && filters.result !== 'all' ? filters.result : undefined,
  });
  const tasks = useMemo(() => data?.items ?? [], [data]);
  const cancelMutation = useCancelMMLTasks();
  const deleteMutation = useDeleteMMLTasks();
  const [selectedTaskIds, setSelectedTaskIds] = useState<Key[]>([]);
  const [selectedTaskStatusById, setSelectedTaskStatusById] = useState<Record<string, MMLTaskStatus>>({});

  const filterFields: FilterField[] = useMemo(() => [
    {
      name: 'taskName',
      label: t('mml.taskName'),
      type: 'input',
      placeholder: t('mml.inputTaskNameRequired'),
    },
    {
      name: 'taskOrigin',
      label: t('mml.taskOrigin'),
      type: 'select',
      placeholder: t('mml.taskOrigin'),
      options: [
        { label: t('common.all'), value: 'all' },
        { label: t('mml.taskOrigin.console'), value: 'console' },
        { label: t('mml.taskOrigin.script'), value: 'script' },
      ],
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
      taskOrigin: typeof values.taskOrigin === 'string' ? values.taskOrigin : undefined,
      status: typeof values.status === 'string' ? values.status : undefined,
      executeType: typeof values.executeType === 'string' ? values.executeType : undefined,
      result: typeof values.result === 'string' ? values.result : undefined,
    });
  }, []);

  const handleReset = useCallback(() => {
    setFilters({});
    setPage(1);
  }, []);

  const confirmBatchCancel = () => {
    const ids = selectedTaskIds.map(String);
    if (ids.length === 0) return;
    const currentTaskById = new Map(tasks.map((task) => [task.id, task]));
    const nonCancellableCount = ids.filter((id) => {
      const status = currentTaskById.get(id)?.status ?? selectedTaskStatusById[id];
      return !status || !CANCELLABLE_TASK_STATUSES.has(status);
    }).length;
    if (nonCancellableCount > 0) {
      void message.warning(t('mml.batchCancelInvalidTasksBlocked', { count: nonCancellableCount }));
      return;
    }
    Modal.confirm({
      title: t('mml.confirmCancelTitle'),
      content: t('mml.confirmBatchCancelTasks', { count: ids.length }),
      okText: t('mml.terminateTask'),
      cancelText: t('common.cancel'),
      okButtonProps: { danger: true },
      onOk: () => new Promise<void>((resolve, reject) => {
        cancelMutation.mutate(ids, {
          onSuccess: () => {
            setSelectedTaskIds([]);
            setSelectedTaskStatusById({});
            void refetch();
            void message.success(t('mml.batchCancelSuccess', { count: ids.length }));
            resolve();
          },
          onError: (error) => {
            const detail = getErrorMessage(error) || t('common.unknown');
            setSelectedTaskIds([]);
            setSelectedTaskStatusById({});
            void refetch();
            void message.error(t('mml.batchCancelFailed', { error: detail }));
            reject(error);
          },
        });
      }),
    });
  };

  const confirmBatchDelete = () => {
    const ids = selectedTaskIds.map(String);
    if (ids.length === 0) return;
    const currentTaskById = new Map(tasks.map((task) => [task.id, task]));
    const runningCount = ids.filter((id) => (
      currentTaskById.get(id)?.status ?? selectedTaskStatusById[id]
    ) === 'running').length;
    if (runningCount > 0) {
      void message.warning(t('mml.batchDeleteRunningTasksBlocked', { count: runningCount }));
      return;
    }
    Modal.confirm({
      title: t('common.confirmDelete'),
      content: t('mml.confirmBatchDeleteTasks', { count: ids.length }),
      okText: t('common.delete'),
      cancelText: t('common.cancel'),
      okButtonProps: { danger: true },
      onOk: () => new Promise<void>((resolve, reject) => {
        deleteMutation.mutate(ids, {
          onSuccess: () => {
            setSelectedTaskIds([]);
            setSelectedTaskStatusById({});
            void refetch();
            void message.success(t('common.deleteSuccess'));
            resolve();
          },
          onError: (error) => {
            const detail = getErrorMessage(error);
            setSelectedTaskIds([]);
            setSelectedTaskStatusById({});
            void refetch();
            void message.error(detail ? t('mml.deleteFailed', { error: detail }) : t('common.deleteFailed'));
            reject(error);
          },
        });
      }),
    });
  };

  // ---- 查看 modal state ----------------------------------------------------
  // 任务记录为只读：记录由"执行 MML 命令 / 脚本任务执行"被动产生，不提供新建/编辑。
  const [viewing, setViewing] = useState<MMLTask | null>(null);
  const [detailRow, setDetailRow] = useState<DeviceTaskResultItem | null>(null);
  const [rawRow, setRawRow] = useState<DeviceTaskResultItem | null>(null);
  const [resultPage, setResultPage] = useState(1);
  const { data: resultsData, isLoading: resultsLoading } = useMMLTaskResults(viewing?.id ?? null, resultPage, TASK_RESULT_PAGE_SIZE);

  const resultRows = useMemo<DeviceTaskResultItem[]>(
    () => (resultsData?.items ?? []),
    [resultsData]
  );

  const renderCommandCompact = useCallback((command: string, maxTargetWidth = 220) => {
    const parsed = parseMmlCommandDisplay(command);
    const paramsContent = parsed.parameterCount > 0 ? (
      <div style={{ width: 520, maxWidth: '70vw' }}>
        <Space size={6} style={{ marginBottom: 8 }}>
          {parsed.operation ? <Tag color={operationColor(parsed.operation)}>{parsed.operation}</Tag> : null}
          <Typography.Text strong>{parsed.target || parsed.commandHead}</Typography.Text>
        </Space>
        <div
          style={{
            maxHeight: 280,
            overflow: 'auto',
            border: '1px solid var(--color-border-secondary, rgba(128,128,128,0.24))',
            borderRadius: 6,
          }}
        >
          {parsed.params.map((param, index) => (
            <div
              key={`${param.key}-${index}`}
              style={{
                display: 'grid',
                gridTemplateColumns: '190px minmax(0, 1fr)',
                gap: 12,
                padding: '7px 10px',
                borderBottom: '1px solid var(--color-border-secondary, rgba(128,128,128,0.24))',
              }}
            >
              <Typography.Text code style={{ fontSize: 12, wordBreak: 'break-all' }}>
                {param.key}
              </Typography.Text>
              <Typography.Text style={{ fontSize: 12, wordBreak: 'break-all' }}>
                {param.value || '-'}
              </Typography.Text>
            </div>
          ))}
        </div>
      </div>
    ) : null;

    return (
      <Space size={6} wrap={false} style={{ maxWidth: '100%' }}>
        {parsed.operation ? (
          <Tag color={operationColor(parsed.operation)} style={{ marginInlineEnd: 0, flex: '0 0 auto' }}>
            {parsed.operation}
          </Tag>
        ) : null}
        <Tooltip title={parsed.parameterCount > 0 ? undefined : command}>
          <Typography.Text style={{ minWidth: 0, maxWidth: maxTargetWidth }} ellipsis>
            {parsed.target || parsed.commandHead || command}
          </Typography.Text>
        </Tooltip>
        {paramsContent ? (
          <Popover
            title={t('mml.scriptParamsTitle')}
            content={paramsContent}
            trigger="click"
            placement="bottomLeft"
          >
            <Button type="link" size="small" style={{ padding: 0, flex: '0 0 auto' }}>
              {t('mml.scriptParamsCount', { count: parsed.parameterCount })}
            </Button>
          </Popover>
        ) : null}
      </Space>
    );
  }, [t]);

  const parsedParamColumns: ColumnsType<ParsedParamValue> = useMemo(() => [
    {
      key: 'name',
      title: t('mml.taskResult.parsed.gpv.path'),
      dataIndex: 'name',
      render: (value: string) => (
        <Typography.Text code style={{ fontSize: 12, wordBreak: 'break-all' }}>
          {value}
        </Typography.Text>
      ),
    },
    {
      key: 'value',
      title: t('mml.taskResult.parsed.gpv.value'),
      dataIndex: 'value',
      width: 220,
      render: (value: string) => (
        <Typography.Text style={{ fontFamily: 'monospace', fontSize: 12, wordBreak: 'break-all' }}>
          {value || '-'}
        </Typography.Text>
      ),
    },
  ], [t]);

  const resultColumns: ColumnsType<DeviceTaskResultItem> = useMemo(() => [
    {
      key: 'deviceSn',
      title: t('mml.resultDeviceCode'),
      dataIndex: 'deviceSn',
      width: 190,
      fixed: 'left',
      render: (value: unknown) => (
        <Typography.Text copyable={{ text: String(value || '') }} style={{ maxWidth: 170 }} ellipsis>
          {String(value || '-')}
        </Typography.Text>
      ),
    },
    {
      key: 'deviceName',
      title: t('mml.deviceName'),
      dataIndex: 'deviceName',
      width: 140,
      ellipsis: true,
      render: (value: unknown) => String(value || '-'),
    },
    {
      key: 'command',
      title: t('mml.resultCommand'),
      width: 360,
      render: (_: unknown, row) => {
        const command = resultCommandText(row);
        return (
          <Space orientation="vertical" size={2} style={{ width: '100%' }}>
            {row.planLineNo ? (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {t('mml.scriptLineNo', { line: row.planLineNo })}
                {row.planOrder ? ` / ${row.planOrder}` : ''}
              </Typography.Text>
            ) : null}
            {renderCommandCompact(command, 180)}
          </Space>
        );
      },
    },
    {
      key: 'status',
      title: t('mml.status'),
      dataIndex: 'status',
      width: 110,
      render: (value: unknown) => {
        const tag = DEVICE_RESULT_STATUS_TAGS[String(value || '')];
        return tag ? <Tag color={tag.color}>{t(tag.key)}</Tag> : <Tag>{String(value || '-')}</Tag>;
      },
    },
    {
      key: 'result',
      title: t('mml.result'),
      width: 100,
      render: (_: unknown, row) => {
        if (row.status && row.status !== 'completed') return <Tag>{t('mml.pendingStatus')}</Tag>;
        const ok = Boolean(row.result?.success);
        return <Tag color={ok ? 'success' : 'error'}>{ok ? t('status.success') : t('status.failed')}</Tag>;
      },
    },
    {
      key: 'failReason',
      title: t('mml.failReason'),
      dataIndex: 'failReason',
      width: 180,
      ellipsis: true,
      render: (value: unknown) => {
        const text = String(value || '-');
        return text === '-' ? text : (
          <Tooltip title={text}>
            <Typography.Text type="danger" style={{ maxWidth: 160 }} ellipsis>
              {text}
            </Typography.Text>
          </Tooltip>
        );
      },
    },
    {
      key: 'detail',
      title: t('mml.detail'),
      width: 90,
      render: (_: unknown, row) => (
        <Button
          type="link"
          size="small"
          disabled={!hasParsedDetail(row)}
          onClick={() => setDetailRow(row)}
        >
          {t('common.view')}
        </Button>
      ),
    },
    {
      key: 'rawMessage',
      title: t('mml.messageDisplay'),
      width: 110,
      render: (_: unknown, row) => (
        <Button
          type="link"
          size="small"
          disabled={!hasMessageText(row)}
          onClick={() => setRawRow(row)}
        >
          {t('common.view')}
        </Button>
      ),
    },
    {
      key: 'startedAt',
      title: t('mml.startTime'),
      dataIndex: 'startedAt',
      width: 160,
      render: (value: unknown) => formatTime(value as string | undefined),
    },
    {
      key: 'finishedAt',
      title: t('mml.endTime'),
      dataIndex: 'finishedAt',
      width: 160,
      render: (value: unknown) => formatTime(value as string | undefined),
    },
  ], [renderCommandCompact, t]);

  const columns: DataTableColumn<MMLTask>[] = useMemo(() => [
    {
      key: 'operation',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 90,
      render: (_: unknown, record: MMLTask) => (
        <Button
          aria-label={t('common.view')}
          type="link"
          size="small"
          icon={<ProfileOutlined />}
          onClick={() => {
            setResultPage(1);
            setViewing(record);
          }}
        >
          {t('common.view')}
        </Button>
      ),
    },
    { key: 'taskName', title: t('mml.taskName'), dataIndex: 'taskName', ellipsis: true },
    { key: 'creator',  title: t('mml.creator'),  dataIndex: 'creator', width: 100 },
    {
      key: 'taskOrigin',
      title: t('mml.taskOrigin'),
      dataIndex: 'taskOrigin',
      width: 120,
      render: (val: unknown) => {
        const v = val as MMLTaskOrigin;
        const tag = TASK_ORIGIN_TAGS[v];
        return tag ? <Tag color={tag.color}>{t(tag.key)}</Tag> : <Tag>{v || '-'}</Tag>;
      },
    },
    {
      key: 'executeType',
      title: t('mml.type'),
      dataIndex: 'executeType',
      width: 100,
      render: (val: unknown) => {
        const v = val as MMLExecuteType;
        const tag = EXECUTE_TYPE_TAGS[v];
        return tag ? <Tag color={tag.color}>{t(tag.key)}</Tag> : <Tag>{v || '-'}</Tag>;
      },
    },
    {
      key: 'status',
      title: t('mml.status'),
      dataIndex: 'status',
      width: 100,
      render: (val: unknown) => {
        const v = val as MMLTaskStatus;
        const tag = TASK_STATUS_TAGS[v];
        return tag ? <Tag color={tag.color}>{t(tag.key)}</Tag> : <Tag>{v || '-'}</Tag>;
      },
    },
    {
      key: 'progress',
      title: t('mml.progress'),
      width: 100,
      render: (_: unknown, record: MMLTask) => {
        const { done, total } = getMmlTaskProgress(record);
        return `${done}/${total}`;
      },
    },
    {
      key: 'startedAt',
      title: t('mml.startTime'),
      dataIndex: 'startedAt',
      width: 160,
      render: (val: unknown) => formatTime(val as string | undefined),
    },
    {
      key: 'finishedAt',
      title: t('mml.endTime'),
      dataIndex: 'finishedAt',
      width: 160,
      render: (val: unknown) => formatTime(val as string | undefined),
    },
    {
      key: 'createdAt',
      title: t('mml.createTime'),
      dataIndex: 'createdAt',
      width: 160,
      render: (val: unknown) => formatTime(val as string),
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
      <Space size={8} wrap style={{ marginBottom: 12 }}>
        <Button
          disabled={selectedTaskIds.length === 0}
          icon={<StopOutlined />}
          loading={cancelMutation.isPending}
          onClick={confirmBatchCancel}
        >
          {t('mml.batchCancelTasks')}
        </Button>
        <Button
          danger
          disabled={selectedTaskIds.length === 0}
          icon={<DeleteOutlined />}
          loading={deleteMutation.isPending}
          onClick={confirmBatchDelete}
        >
          {t('common.batchDelete')}
        </Button>
        {selectedTaskIds.length > 0 ? (
          <Typography.Text type="secondary">
            {t('table.selected', { count: selectedTaskIds.length })}
          </Typography.Text>
        ) : null}
      </Space>
      <DataTable<MMLTask>
        tableId="mml-task-record"
        columns={columns}
        dataSource={tasks}
        loading={isLoading}
        rowKey="id"
        selectable
        selectedRowKeys={selectedTaskIds}
        onSelectionChange={(keys, rows) => {
          const nextIds = new Set(keys.map(String));
          setSelectedTaskIds(keys);
          setSelectedTaskStatusById((prev) => {
            const next: Record<string, MMLTaskStatus> = {};
            nextIds.forEach((id) => {
              const previousStatus = prev[id];
              if (previousStatus) next[id] = previousStatus;
            });
            rows.forEach((task) => {
              if (nextIds.has(task.id)) next[task.id] = task.status;
            });
            tasks.forEach((task) => {
              if (nextIds.has(task.id)) next[task.id] = task.status;
            });
            return next;
          });
        }}
        preserveSelectedRowKeys
        total={data?.total ?? 0}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => void refetch()}
        hideToolbar
        scroll={{ x: 1200 }}
      />

      <Modal
        title={viewing ? t('mml.executionResult', { name: viewing.taskName || viewing.id }) : t('common.view')}
        open={Boolean(viewing)}
        onCancel={() => {
          setViewing(null);
          setDetailRow(null);
          setRawRow(null);
          setResultPage(1);
        }}
        footer={<Button onClick={() => {
          setViewing(null);
          setDetailRow(null);
          setRawRow(null);
          setResultPage(1);
        }}>{t('common.close')}</Button>}
        width={1180}
        destroyOnHidden
      >
        {viewing && (
          <div style={{ minHeight: 360 }}>
            <Space wrap size={[8, 8]} style={{ marginBottom: 12 }}>
              <Tag>{t('mml.status')}: {t(TASK_STATUS_TAGS[viewing.status]?.key ?? 'mml.status')}</Tag>
              <Tag>{t('mml.taskOrigin')}: {t(TASK_ORIGIN_TAGS[viewing.taskOrigin]?.key ?? 'mml.taskOrigin')}</Tag>
              <Tag>{t('mml.deviceCountLabel')}{viewing.totalDevices ?? 0}</Tag>
              <Tag>{t('mml.successCountLabel')}{viewing.successCount ?? 0}</Tag>
              <Tag>{t('mml.failedCountLabel')}{viewing.failedCount ?? 0}</Tag>
            </Space>

            {resultRows.length === 0 && !resultsLoading ? (
              <Empty description={t('mml.noExecutionResult')} />
            ) : (
              <Table<DeviceTaskResultItem>
                size="small"
                columns={resultColumns}
                dataSource={resultRows}
                loading={resultsLoading}
                pagination={false}
                rowKey={(row, index) => row.deviceTaskId || `${row.deviceSn}-${row.commandIndex ?? index}`}
                scroll={{ x: 1520, y: 360 }}
              />
            )}
            {(resultsData?.total ?? 0) > TASK_RESULT_PAGE_SIZE && (
              <div style={{ marginTop: 12, textAlign: 'right' }}>
                <Pagination
                  size="small"
                  current={resultPage}
                  pageSize={TASK_RESULT_PAGE_SIZE}
                  total={resultsData?.total ?? 0}
                  showSizeChanger={false}
                  onChange={setResultPage}
                />
              </div>
            )}
          </div>
        )}
      </Modal>

      <Modal
        title={detailRow ? t('mml.resultDetailTitle', { device: detailRow.deviceSn || '-' }) : t('mml.detail')}
        open={Boolean(detailRow)}
        onCancel={() => setDetailRow(null)}
        footer={<Button onClick={() => setDetailRow(null)}>{t('common.close')}</Button>}
        width={860}
        destroyOnHidden
      >
        {detailRow && (
          <Space orientation="vertical" size={12} style={{ width: '100%' }}>
            <Space wrap size={[8, 8]}>
              <Tag>{t('mml.resultDeviceCode')}: {detailRow.deviceSn || '-'}</Tag>
              {detailRow.deviceName ? <Tag>{t('mml.deviceName')}: {detailRow.deviceName}</Tag> : null}
              <Tag>{t('mml.resultCommand')}: {resultCommandText(detailRow)}</Tag>
              <Tag color={detailRow.result?.success ? 'success' : 'error'}>
                {detailRow.result?.success ? t('status.success') : t('status.failed')}
              </Tag>
            </Space>
            {(() => {
              const parsed = parsedResult(detailRow);
              if (!parsed) {
                return <Empty description={t('mml.taskResult.parsed.notParsable')} />;
              }
              if (parsed.kind === 'gpv') {
                return (
                  <Table<ParsedParamValue>
                    size="small"
                    rowKey={(row, index) => `${row.name}-${index}`}
                    columns={parsedParamColumns}
                    dataSource={parsed.params ?? []}
                    pagination={false}
                    scroll={{ y: 360 }}
                    locale={{ emptyText: t('mml.taskResult.parsed.gpv.empty') }}
                  />
                );
              }
              return (
                <Typography.Paragraph
                  style={{
                    background: 'rgba(0,0,0,0.04)',
                    padding: 12,
                    borderRadius: 4,
                    marginBottom: 0,
                  }}
                >
                  {parsedResultSummary(parsed, t)}
                </Typography.Paragraph>
              );
            })()}
          </Space>
        )}
      </Modal>

      <Modal
        title={rawRow ? t('mml.rawMessageTitle', { device: rawRow.deviceSn || '-' }) : t('mml.messageDisplay')}
        open={Boolean(rawRow)}
        onCancel={() => setRawRow(null)}
        footer={<Button onClick={() => setRawRow(null)}>{t('common.close')}</Button>}
        width={860}
        destroyOnHidden
      >
        {rawRow && (
          <Space orientation="vertical" size={12} style={{ width: '100%' }}>
            <Space wrap size={[8, 8]}>
              <Tag>{t('mml.resultDeviceCode')}: {rawRow.deviceSn || '-'}</Tag>
              {rawRow.deviceName ? <Tag>{t('mml.deviceName')}: {rawRow.deviceName}</Tag> : null}
              <Tag color={rawRow.result?.success ? 'success' : 'error'}>
                {rawRow.result?.success ? t('status.success') : t('status.failed')}
              </Tag>
            </Space>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, maxWidth: '100%' }}>
              <Typography.Text type="secondary">{t('mml.resultCommand')}:</Typography.Text>
              <div style={{ minWidth: 0, flex: 1 }}>
                {renderCommandCompact(resultCommandText(rawRow), 360)}
              </div>
            </div>
            {[
              { key: 'request', title: t('mml.requestMessage'), text: requestMessageText(rawRow) },
              { key: 'response', title: t('mml.responseMessage'), text: responseMessageText(rawRow) },
            ].map((section) => (
              <div key={section.key}>
                <Typography.Text strong>{section.title}</Typography.Text>
                <pre
                  style={{
                    maxHeight: 260,
                    overflow: 'auto',
                    whiteSpace: 'pre-wrap',
                    wordBreak: 'break-all',
                    fontFamily: 'monospace',
                    fontSize: 12,
                    lineHeight: 1.55,
                    background: 'rgba(0,0,0,0.04)',
                    padding: 12,
                    borderRadius: 4,
                    margin: '8px 0 0',
                    maxWidth: '100%',
                  }}
                >
                  {section.text || '-'}
                </pre>
              </div>
            ))}
          </Space>
        )}
      </Modal>
    </ListPageLayout>
  );
}
