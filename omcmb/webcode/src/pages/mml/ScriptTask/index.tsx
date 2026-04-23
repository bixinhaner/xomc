import { useState, useMemo, useCallback } from 'react';
import { Button, Dropdown, Modal, Space, Table, Tag, message } from 'antd';
import type { MenuProps } from 'antd';
import {
  PlusOutlined,
  MoreOutlined,
  InfoCircleOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  StopOutlined,
  DeleteOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';

import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ScriptTaskDrawer from '@/pages/mml/components/ScriptTaskDrawer';
import { useT } from '@/hooks/useT';

import type {
  MMLScript,
  MMLScriptStatus,
  MMLScriptType,
  MMLScriptResult,
} from '@core/types/mml';
import {
  useMMLScripts,
  useDeleteMMLScripts,
  useStartMMLScript,
  usePauseMMLScript,
  useCancelMMLScript,
} from '@core/hooks/api/useMML';
import { useDictionary } from '@core/hooks/api/useSystem';

// -------------------------------------------------------------------------
// Display mappings — mml_scripts columns
// -------------------------------------------------------------------------

const SCRIPT_TYPE_TAGS: Record<MMLScriptType, { color: string; key: string }> = {
  manual: { color: 'geekblue', key: 'mml.immediateExecute' },
  batch:  { color: 'purple',   key: 'mml.periodicTask' },
};

// mml_scripts.status 分两族：定义态（active/archived）与执行态（pending..cancelled）。
// 两族都用同一张表达映射，未知值 fallback 到 plain Tag。
const SCRIPT_STATUS_TAGS: Record<MMLScriptStatus, { color: string; key: string }> = {
  active:    { color: 'blue',       key: 'status.success' },
  archived:  { color: 'default',    key: 'mml.completedStatus' },
  pending:   { color: 'default',    key: 'mml.pendingStatus' },
  running:   { color: 'processing', key: 'mml.runningStatus' },
  paused:    { color: 'warning',    key: 'mml.pausedStatus' },
  completed: { color: 'success',    key: 'mml.completedStatus' },
  failed:    { color: 'error',      key: 'mml.failedStatus' },
  cancelled: { color: 'error',      key: 'mml.cancelledStatus' },
};

const SCRIPT_RESULT_TAGS: Record<MMLScriptResult, { color: string; key: string }> = {
  success: { color: 'success', key: 'status.success' },
  partial: { color: 'warning', key: 'mml.partialSuccess' },
  failed:  { color: 'error',   key: 'status.failed' },
};

// Shape of a single device execution row displayed in the 查看 modal.
// Parsed tolerantly from the loosely-typed MMLScript.result JSONB blob.
interface ScriptExecutionRow {
  deviceSn: string;
  status: string;
  executionTimeMs?: number;
  output?: string;
}

// ---- 小工具 ---------------------------------------------------------------

function formatTime(iso?: string | null): string {
  if (!iso) return '-';
  const d = dayjs(iso);
  return d.isValid() ? d.format('YYYY-MM-DD HH:mm:ss') : '-';
}

function parseExecutionRows(raw: Record<string, unknown> | undefined): ScriptExecutionRow[] {
  if (!raw) return [];
  const candidate =
    Array.isArray(raw) ? raw :
    Array.isArray((raw as Record<string, unknown>).items)
      ? ((raw as Record<string, unknown>).items as unknown[])
      : [];
  return candidate
    .filter((v): v is Record<string, unknown> => v !== null && typeof v === 'object')
    .map((entry) => {
      const deviceSn = (entry.device_sn as string) || (entry.deviceSn as string) || '';
      const status =
        (entry.status as string) ||
        (entry.success === true ? 'success' : entry.success === false ? 'failed' : '');
      const execMs =
        (entry.execution_time as number) ??
        (entry.executionTime as number) ??
        (entry.duration_ms as number);
      const output =
        (entry.output as string) ||
        (entry.raw_output as string) ||
        (entry.rawOutput as string) ||
        '';
      return {
        deviceSn,
        status,
        executionTimeMs: typeof execMs === 'number' ? execMs : undefined,
        output,
      };
    });
}

// 从 mml_scripts.result JSONB 中猜一个 overall 结果标识（success/partial/failed）。
function deriveResult(raw: Record<string, unknown> | undefined): MMLScriptResult | undefined {
  if (!raw) return undefined;
  const overall = (raw as Record<string, unknown>).overall as string | undefined;
  if (overall === 'success' || overall === 'partial' || overall === 'failed') return overall;
  const rows = parseExecutionRows(raw);
  if (rows.length === 0) return undefined;
  const succ = rows.filter((r) => r.status === 'success').length;
  if (succ === rows.length) return 'success';
  if (succ === 0) return 'failed';
  return 'partial';
}

// 根据 status 决定三点菜单各操作是否可用。原则：
//   pending    → 可"开始"
//   running    → 可"暂停"、"终止"
//   paused     → 可"开始"、"终止"
//   completed / failed / cancelled / archived → 三项生命周期操作均不可用
//   active（仅定义态）→ 三项均不可用（未开跑）
// 删除：running 下禁用，其余可用。
function lifecycleEnable(status: MMLScriptStatus | string | undefined): {
  canStart: boolean; canPause: boolean; canCancel: boolean; canDelete: boolean;
} {
  switch (status) {
    case 'pending':
      return { canStart: true,  canPause: false, canCancel: false, canDelete: true };
    case 'running':
      return { canStart: false, canPause: true,  canCancel: true,  canDelete: false };
    case 'paused':
      return { canStart: true,  canPause: false, canCancel: true,  canDelete: true };
    default:
      // completed / failed / cancelled / archived / active / unknown
      return { canStart: false, canPause: false, canCancel: false, canDelete: true };
  }
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export default function ScriptTask() {
  const t = useT();

  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [filters, setFilters] = useState<{
    taskName?: string;
    startDate?: string;
    endDate?: string;
    type?: string;
    status?: string;
    result?: string;
  }>({});

  const { data: productTypeDict } = useDictionary('product_type');
  const productTypeOptions = useMemo(() => {
    const details = productTypeDict?.sysDictionaryDetails;
    return details?.length ? details.map((d) => ({ label: d.label, value: d.value })) : [];
  }, [productTypeDict]);

  const { data, isLoading, refetch } = useMMLScripts({
    page,
    pageSize,
    search: filters.taskName,
  });
  const deleteScriptsMutation = useDeleteMMLScripts();
  const startScriptMutation = useStartMMLScript();
  const pauseScriptMutation = usePauseMMLScript();
  const cancelScriptMutation = useCancelMMLScript();

  // Scripts list filtered client-side by date-range / type / status / result（后端
  // /mml/scripts 暂不支持这几个 query，沿用前端过滤）。
  const scripts = useMemo(() => {
    const items = data?.items ?? [];
    return items.filter((s) => {
      if (filters.type && filters.type !== 'all' && s.type !== filters.type) return false;
      if (filters.status && filters.status !== 'all' && s.status !== filters.status) return false;
      if (filters.result && filters.result !== 'all') {
        if (deriveResult(s.result) !== filters.result) return false;
      }
      if (filters.startDate) {
        const ts = s.startTime ? dayjs(s.startTime) : null;
        if (!ts || !ts.isValid() || ts.isBefore(dayjs(filters.startDate))) return false;
      }
      if (filters.endDate) {
        const ts = s.endTime ? dayjs(s.endTime) : null;
        if (!ts || !ts.isValid() || ts.isAfter(dayjs(filters.endDate))) return false;
      }
      return true;
    });
  }, [data, filters]);

  // ---- filter 字段（对齐 image-7） ---------------------------------------
  const filterFields: FilterField[] = useMemo(() => [
    {
      name: 'taskName',
      label: t('mml.taskName'),
      type: 'input',
      placeholder: t('mml.inputTaskNameRequired'),
    },
    {
      name: 'timeRange',
      label: t('mml.startTime'),
      type: 'date-range',
    },
    {
      name: 'type',
      label: t('mml.type'),
      type: 'select',
      placeholder: t('mml.type'),
      options: [
        { label: t('common.all'),          value: 'all' },
        { label: t('mml.immediateExecute'), value: 'manual' },
        { label: t('mml.periodicTask'),    value: 'batch' },
      ],
    },
    {
      name: 'status',
      label: t('mml.status'),
      type: 'select',
      placeholder: t('mml.status'),
      options: [
        { label: t('common.all'),           value: 'all' },
        { label: t('mml.pendingStatus'),    value: 'pending' },
        { label: t('mml.runningStatus'),    value: 'running' },
        { label: t('mml.pausedStatus'),     value: 'paused' },
        { label: t('mml.completedStatus'),  value: 'completed' },
        { label: t('mml.failedStatus'),     value: 'failed' },
        { label: t('mml.cancelledStatus'),  value: 'cancelled' },
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
    const range = values.timeRange as [Dayjs, Dayjs] | undefined;
    setFilters({
      taskName: typeof values.taskName === 'string' ? values.taskName.trim() : undefined,
      startDate: range?.[0]?.isValid() ? range[0].startOf('day').toISOString() : undefined,
      endDate: range?.[1]?.isValid() ? range[1].endOf('day').toISOString() : undefined,
      type: typeof values.type === 'string' ? values.type : undefined,
      status: typeof values.status === 'string' ? values.status : undefined,
      result: typeof values.result === 'string' ? values.result : undefined,
    });
  }, []);

  const handleReset = useCallback(() => {
    setFilters({});
    setPage(1);
  }, []);

  // ---- 模态状态 ----------------------------------------------------------
  const [viewing, setViewing] = useState<MMLScript | null>(null);
  const [info, setInfo] = useState<MMLScript | null>(null);
  const executionRows = useMemo(() => parseExecutionRows(viewing?.result), [viewing]);

  // ---- 新增：共享 Drawer（布局见 ScriptTaskDrawer / docs/design/image-8.png）
  const [createOpen, setCreateOpen] = useState(false);

  // ---- 行操作 -----------------------------------------------------------
  const runMutation = useCallback(
    (
      op: 'start' | 'pause' | 'cancel',
      record: MMLScript,
    ) => {
      const mutation =
        op === 'start' ? startScriptMutation :
        op === 'pause' ? pauseScriptMutation :
        cancelScriptMutation;
      const successKey =
        op === 'start' ? 'mml.taskStarted' :
        op === 'pause' ? 'mml.taskPaused' :
        'mml.taskCancelled';
      const failKey =
        op === 'start' ? 'mml.startFailed' :
        op === 'pause' ? 'mml.pauseFailed' :
        'mml.cancelFailed';
      mutation.mutate(record.id, {
        onSuccess: () => void message.success(t(successKey, { name: record.scriptName })),
        onError: (err) => void message.error(
          t(failKey, { error: err instanceof Error ? err.message : 'Unknown' })
        ),
      });
    },
    [startScriptMutation, pauseScriptMutation, cancelScriptMutation, t],
  );

  // 敏感操作：删除 / 终止 使用 Modal.confirm 做二次确认。
  const confirmDelete = useCallback((record: MMLScript) => {
    Modal.confirm({
      title: t('common.confirmDelete'),
      content: t('mml.confirmDeleteScript', { name: record.scriptName }),
      okText: t('common.delete'),
      okButtonProps: { danger: true },
      cancelText: t('common.cancel'),
      onOk: () =>
        new Promise<void>((resolve, reject) => {
          deleteScriptsMutation.mutate([record.id], {
            onSuccess: () => {
              void message.success(t('common.deleteSuccess'));
              resolve();
            },
            onError: (err) => {
              void message.error(err instanceof Error ? err.message : 'Unknown');
              reject(err);
            },
          });
        }),
    });
  }, [deleteScriptsMutation, t]);

  const confirmCancel = useCallback((record: MMLScript) => {
    Modal.confirm({
      title: t('mml.confirmCancelTitle'),
      content: t('mml.confirmCancelScript', { name: record.scriptName }),
      okText: t('mml.terminateTask'),
      okButtonProps: { danger: true },
      cancelText: t('common.cancel'),
      onOk: () => runMutation('cancel', record),
    });
  }, [runMutation, t]);

  // 三点菜单 —— enable/disable 由 status 决定；开始/暂停可逆无需二次确认，
  // 终止 / 删除 为敏感操作统一走确认。
  const getActionMenu = useCallback((record: MMLScript): MenuProps['items'] => {
    const { canStart, canPause, canCancel, canDelete } = lifecycleEnable(record.status);
    return [
      { key: 'info',   icon: <InfoCircleOutlined />, label: t('mml.info'),          onClick: () => setInfo(record) },
      { key: 'start',  icon: <PlayCircleOutlined />, label: t('common.start'),
        disabled: !canStart, onClick: () => runMutation('start', record) },
      { key: 'pause',  icon: <PauseCircleOutlined />,label: t('common.pause'),
        disabled: !canPause, onClick: () => runMutation('pause', record) },
      { key: 'cancel', icon: <StopOutlined />,       label: t('mml.terminateTask'),
        disabled: !canCancel, onClick: () => confirmCancel(record) },
      { key: 'delete', icon: <DeleteOutlined />,     label: t('common.delete'),
        danger: true, disabled: !canDelete, onClick: () => confirmDelete(record) },
    ];
  }, [t, runMutation, confirmCancel, confirmDelete]);

  const columns: DataTableColumn<MMLScript>[] = useMemo(() => [
    {
      key: 'operation',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 120,
      fixed: 'right',
      render: (_, record) => (
        <Space size={4}>
          <Button type="link" size="small" onClick={() => setViewing(record)}>
            {t('common.view')}
          </Button>
          <Dropdown menu={{ items: getActionMenu(record) }} trigger={['click']}>
            <Button type="link" size="small" icon={<MoreOutlined />} />
          </Dropdown>
        </Space>
      ),
    },
    // 列标题使用"任务名称"（对齐 image-7 & 页面语义："脚本任务"），数据仍来自
    // mml_scripts.script_name。
    { key: 'taskName', title: t('mml.taskName'), dataIndex: 'scriptName', ellipsis: true },
    { key: 'creator',  title: t('mml.creator'),  dataIndex: 'creator', width: 120 },
    {
      key: 'createTime',
      title: t('mml.createTime'),
      dataIndex: 'createTime',
      width: 170,
      render: (v: string) => formatTime(v),
    },
    {
      key: 'type',
      title: t('mml.type'),
      dataIndex: 'type',
      width: 100,
      render: (v: MMLScriptType) => {
        const tag = SCRIPT_TYPE_TAGS[v];
        return tag ? <Tag color={tag.color}>{t(tag.key)}</Tag> : <Tag>{v || '-'}</Tag>;
      },
    },
    {
      key: 'status',
      title: t('mml.status'),
      dataIndex: 'status',
      width: 100,
      render: (v: MMLScriptStatus) => {
        const tag = SCRIPT_STATUS_TAGS[v];
        return tag ? <Tag color={tag.color}>{t(tag.key)}</Tag> : <Tag>{v || '-'}</Tag>;
      },
    },
    {
      key: 'progress',
      title: t('mml.progress'),
      dataIndex: 'progress',
      width: 90,
      render: (v?: number) => `${Math.max(0, Math.min(100, Math.round(v ?? 0)))}%`,
    },
    {
      key: 'result',
      title: t('mml.result'),
      width: 100,
      render: (_: unknown, record: MMLScript) => {
        const r = deriveResult(record.result);
        if (!r) return '-';
        const tag = SCRIPT_RESULT_TAGS[r];
        return tag ? <Tag color={tag.color}>{t(tag.key)}</Tag> : <Tag>{r}</Tag>;
      },
    },
    {
      key: 'startTime',
      title: t('mml.startTime'),
      dataIndex: 'startTime',
      width: 170,
      render: (v?: string) => formatTime(v),
    },
    {
      key: 'endTime',
      title: t('mml.endTime'),
      dataIndex: 'endTime',
      width: 170,
      render: (v?: string) => formatTime(v),
    },
  ], [t, getActionMenu]);

  // 查看 modal 列 —— 对应 image-5.png：设备SN / 状态 / 执行时间 / 输出
  const resultColumns = useMemo(() => [
    { key: 'deviceSn', title: t('mml.deviceSn'), dataIndex: 'deviceSn', width: 160 },
    {
      key: 'status',
      title: t('mml.status'),
      dataIndex: 'status',
      width: 100,
      render: (v: string) => {
        if (v === 'success') return <Tag color="success">{t('status.success')}</Tag>;
        if (v === 'failed') return <Tag color="error">{t('status.failed')}</Tag>;
        return <Tag>{v || '-'}</Tag>;
      },
    },
    {
      key: 'executionTimeMs',
      title: t('mml.executionTime'),
      dataIndex: 'executionTimeMs',
      width: 120,
      align: 'right' as const,
      render: (v?: number) => (typeof v === 'number' ? v : '-'),
    },
    {
      key: 'output',
      title: t('mml.output'),
      dataIndex: 'output',
      ellipsis: true,
      render: (v: string) => v || '-',
    },
  ], [t]);

  return (
    <ListPageLayout
      title={t('nav.mml.script')}
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
          {t('common.add')}
        </Button>
      }
    >
      <FilterBar
        filterId="mml-script-task"
        fields={filterFields}
        onSearch={handleSearch}
        onReset={handleReset}
      />
      <DataTable<MMLScript>
        tableId="mml-script-task"
        columns={columns}
        dataSource={scripts}
        loading={isLoading}
        rowKey="id"
        total={data?.total ?? 0}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => void refetch()}
        scroll={{ x: 1500 }}
      />

      {/* 查看：执行结果（对应 image-5.png） */}
      <Modal
        title={viewing ? t('mml.executionResult', { name: viewing.scriptName }) : t('common.view')}
        open={Boolean(viewing)}
        onCancel={() => setViewing(null)}
        footer={null}
        width={820}
        destroyOnClose
      >
        {viewing && (
          <>
            {executionRows.length > 0 ? (
              <Table
                size="small"
                rowKey={(row) => row.deviceSn || Math.random().toString(36).slice(2)}
                dataSource={executionRows}
                columns={resultColumns}
                pagination={false}
                scroll={{ y: 360 }}
              />
            ) : (
              <div style={{ padding: 24, textAlign: 'center', color: '#999' }}>
                {t('mml.noExecutionResult')}
              </div>
            )}
          </>
        )}
      </Modal>

      {/* 信息：脚本基本信息（对应 image-6.png 三点菜单的首项"信息"） */}
      <Modal
        title={t('mml.scriptDetail')}
        open={Boolean(info)}
        onCancel={() => setInfo(null)}
        footer={null}
        width={600}
        destroyOnClose
      >
        {info && (
          <div style={{ padding: '8px 0' }}>
            <p><strong>{t('mml.scriptNameLabel')}</strong>{info.scriptName}</p>
            <p><strong>{t('mml.description')}: </strong>{info.description || '-'}</p>
            <p><strong>{t('mml.deviceType')}: </strong>{info.deviceType || '-'}</p>
            <p>
              <strong>{t('mml.type')}: </strong>
              {SCRIPT_TYPE_TAGS[info.type] ? t(SCRIPT_TYPE_TAGS[info.type].key) : info.type ?? '-'}
            </p>
            <p>
              <strong>{t('mml.status')}: </strong>
              {SCRIPT_STATUS_TAGS[info.status] ? t(SCRIPT_STATUS_TAGS[info.status].key) : info.status ?? '-'}
            </p>
            <p><strong>{t('mml.creatorLabel')}</strong>{info.creator || '-'}</p>
            <p><strong>{t('mml.startTime')}: </strong>{formatTime(info.startTime)}</p>
            <p><strong>{t('mml.endTime')}: </strong>{formatTime(info.endTime)}</p>
            <p><strong>{t('mml.updateTime')}: </strong>{formatTime(info.updateTime)}</p>
            {info.content && (
              <div style={{ marginTop: 12 }}>
                <strong>{t('mml.scriptContent')}</strong>
                <pre
                  style={{
                    background: '#f5f5f5',
                    padding: 12,
                    borderRadius: 4,
                    maxHeight: 300,
                    overflow: 'auto',
                    fontSize: 13,
                    fontFamily: 'monospace',
                    marginTop: 4,
                  }}
                >
                  {info.content}
                </pre>
              </div>
            )}
          </div>
        )}
      </Modal>

      {/* 新增 MML 脚本任务（布局对齐 docs/design/image-8.png） */}
      <ScriptTaskDrawer
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        onSuccess={() => void refetch()}
      />
    </ListPageLayout>
  );
}
