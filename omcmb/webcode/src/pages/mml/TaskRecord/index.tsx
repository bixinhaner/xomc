import { useState, useMemo, useCallback } from 'react';
import { Button, Empty, Modal, Pagination, Space, Table, Tag, Tooltip, Typography } from 'antd';
import { ProfileOutlined } from '@ant-design/icons';
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
} from '@core/hooks/api/useMML';
import { getMmlTaskProgress } from '@core/utils/mmlTaskProgress';

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

function resultDetailText(row: DeviceTaskResultItem | null): string {
  if (!row) return '';
  if (row.result?.rawOutput) return row.result.rawOutput;
  if (row.result?.parsedData) return JSON.stringify(row.result.parsedData, null, 2);
  return '';
}

function resultCommandText(row: DeviceTaskResultItem): string {
  return row.planRawLine || row.commandCode || row.mmlScript || '-';
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

  // ---- 查看 modal state ----------------------------------------------------
  // 任务记录为只读：记录由"执行 MML 命令 / 脚本任务执行"被动产生，不提供新建/编辑。
  const [viewing, setViewing] = useState<MMLTask | null>(null);
  const [detailRow, setDetailRow] = useState<DeviceTaskResultItem | null>(null);
  const [resultPage, setResultPage] = useState(1);
  const { data: resultsData, isLoading: resultsLoading } = useMMLTaskResults(viewing?.id ?? null, resultPage, TASK_RESULT_PAGE_SIZE);

  const resultRows = useMemo<DeviceTaskResultItem[]>(
    () => (resultsData?.items ?? []),
    [resultsData]
  );

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
      width: 260,
      ellipsis: true,
      render: (_: unknown, row) => {
        const command = resultCommandText(row);
        return (
          <Space direction="vertical" size={2} style={{ width: '100%' }}>
            {row.planLineNo ? (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {t('mml.scriptLineNo', { line: row.planLineNo })}
                {row.planOrder ? ` / ${row.planOrder}` : ''}
              </Typography.Text>
            ) : null}
            <Tooltip title={command}>
              <Typography.Text style={{ maxWidth: 240 }} ellipsis>
                {command}
              </Typography.Text>
            </Tooltip>
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
          disabled={!resultDetailText(row)}
          onClick={() => setDetailRow(row)}
        >
          {t('mml.viewRawResponse')}
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
  ], [t]);

  const columns: DataTableColumn<MMLTask>[] = useMemo(() => [
    {
      key: 'taskId',
      title: t('mml.taskId'),
      dataIndex: 'id',
      width: 200,
      render: (val: unknown) => (
        <Typography.Text
          copyable={{ text: String(val) }}
          style={{ fontSize: 12 }}
          ellipsis={{ tooltip: String(val) }}
        >
          {String(val)}
        </Typography.Text>
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
    {
      key: 'operation',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 70,
      fixed: 'right',
      render: (_, record) => (
        <Tooltip title={t('common.view')}>
          <Button
            type="text"
            size="small"
            icon={<ProfileOutlined />}
            onClick={() => {
              setResultPage(1);
              setViewing(record);
            }}
          />
        </Tooltip>
      ),
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
        hideToolbar
        scroll={{ x: 1400 }}
      />

      <Modal
        title={viewing ? t('mml.executionResult', { name: viewing.taskName || viewing.id }) : t('common.view')}
        open={Boolean(viewing)}
        onCancel={() => {
          setViewing(null);
          setDetailRow(null);
          setResultPage(1);
        }}
        footer={<Button onClick={() => {
          setViewing(null);
          setDetailRow(null);
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
                scroll={{ x: 1400, y: 360 }}
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
        title={detailRow ? t('mml.resultRawResponseTitle', { device: detailRow.deviceSn || '-' }) : t('mml.rawOutput')}
        open={Boolean(detailRow)}
        onCancel={() => setDetailRow(null)}
        footer={<Button onClick={() => setDetailRow(null)}>{t('common.close')}</Button>}
        width={860}
        destroyOnHidden
      >
        {detailRow && (
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            <Space wrap size={[8, 8]}>
              <Tag>{t('mml.resultDeviceCode')}: {detailRow.deviceSn || '-'}</Tag>
              {detailRow.deviceName ? <Tag>{t('mml.deviceName')}: {detailRow.deviceName}</Tag> : null}
              <Tag>{t('mml.resultCommand')}: {resultCommandText(detailRow)}</Tag>
              <Tag color={detailRow.result?.success ? 'success' : 'error'}>
                {detailRow.result?.success ? t('status.success') : t('status.failed')}
              </Tag>
            </Space>
            <Typography.Paragraph
              style={{
                maxHeight: 420,
                overflow: 'auto',
                whiteSpace: 'pre-wrap',
                fontFamily: 'monospace',
                fontSize: 12,
                background: 'rgba(0,0,0,0.04)',
                padding: 12,
                borderRadius: 4,
                marginBottom: 0,
              }}
            >
              {resultDetailText(detailRow) || '-'}
            </Typography.Paragraph>
          </Space>
        )}
      </Modal>
    </ListPageLayout>
  );
}
