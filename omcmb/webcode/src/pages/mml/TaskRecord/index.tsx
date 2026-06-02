import { useState, useMemo, useCallback } from 'react';
import { Alert, Button, Descriptions, Modal, Space, Table, Tag, Typography } from 'antd';
import { DownloadOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';

import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import XmlViewer from '@/components/XmlViewer';
import { useT } from '@/hooks/useT';

import type {
  MMLTask,
  MMLTaskStatus,
  MMLExecuteType,
  DeviceTaskResultItem,
  MMLTaskCommandDetail,
  MMLPathTranslationView,
  MMLTaskResultsStats,
  PathTranslationSource,
} from '@core/types/mml';
import {
  useMMLTasks,
  useMMLTaskResults,
} from '@core/hooks/api/useMML';
import {
  parseMmlDeviceTaskResult,
  type ParsedMmlResult,
  type ParsedParamValue,
} from '@core/utils/mmlResultParser';

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

  // ---- 查看 modal state ----------------------------------------------------
  // 任务记录为只读：记录由"执行 MML 命令 / 脚本任务执行"被动产生，不提供新建/编辑。
  const [viewing, setViewing] = useState<MMLTask | null>(null);
  const { data: resultsData, isLoading: resultsLoading } = useMMLTaskResults(
    viewing?.id ?? null,
    1,
    200,
  );

  const resultRows = useMemo<DeviceTaskResultItem[]>(
    () => (resultsData?.items ?? (viewing?.results as DeviceTaskResultItem[] | undefined) ?? []),
    [resultsData, viewing]
  );

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
      width: 90,
      fixed: 'right',
      render: (_, record) => (
        <Button
          type="link"
          size="small"
          onClick={() => setViewing(record)}
        >
          {t('common.view')}
        </Button>
      ),
    },
    { key: 'taskName', title: t('mml.taskName'), dataIndex: 'taskName', ellipsis: true },
    { key: 'creator',  title: t('mml.creator'),  dataIndex: 'creator', width: 100 },
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
        const done = (record.successCount ?? 0) + (record.failedCount ?? 0);
        return `${done}/${record.totalDevices ?? 0}`;
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
      render: (_: unknown, row: DeviceTaskResultItem) => {
        const raw = row.result?.rawOutput;
        // 失败优先：terminal 已经把 [Server] xxx 当 stderr 行打出来，modal 不能漏
        // 显示。failReason 由后端 device_task.error_message / fail_reason 映射，rawOutput
        // 可能为空或只含 SOAP fault 报文，单看 raw 用户读不懂。
        if (row.failReason) {
          const oneLine = row.failReason.replace(/\s+/g, ' ').trim();
          const preview = oneLine.length > 80 ? `${oneLine.slice(0, 80)}…` : oneLine;
          return (
            <Typography.Text type="danger" style={{ fontSize: 12 }}>
              {preview}
            </Typography.Text>
          );
        }
        if (!raw) return '-';
        // 单行简介：把 XML 折成一行，展示前 80 字符；完整内容在展开行里看
        const oneLine = raw.replace(/\s+/g, ' ').trim();
        const preview = oneLine.length > 80 ? `${oneLine.slice(0, 80)}…` : oneLine;
        return (
          <Typography.Text type="secondary" style={{ fontFamily: 'monospace', fontSize: 12 }}>
            {preview}
          </Typography.Text>
        );
      },
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
        expandable={{
          // T-0168: 列表行直接展开（替代旧 Modal 单一入口）。
          // 老 Modal "查看" 按钮保留作为单设备深入兜底，后期评估是否删。
          expandedRowRender: (task) => <TaskRowExpandedDetail task={task} t={t} />,
          rowExpandable: () => true,
        }}
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
        destroyOnHidden
      >
        {viewing && (
          <>
            <CommandSummary task={viewing} t={t} />
            {viewing.pathTranslationWarning?.anyMiss && (
              <Alert
                type="warning"
                showIcon
                style={{ marginBottom: 12 }}
                message={t('mml.pathTranslationWarning')}
                description={t('mml.pathTranslationWarningDetail', {
                  deviceCount: viewing.pathTranslationWarning.deviceCount,
                  pathCount: viewing.pathTranslationWarning.pathCount,
                })}
              />
            )}
            {resultRows.length > 0 ? (
              <Table
                size="small"
                rowKey={(row) => row.deviceSn || Math.random().toString(36).slice(2)}
                dataSource={resultRows}
                columns={resultColumns}
                loading={resultsLoading}
                pagination={false}
                scroll={{ y: 360 }}
                expandable={{
                  // 展开行：失败原因 Alert + 格式化 XML（XmlViewer，带颜色）+ 原始输出
                  // textarea（全文可滚动 / Ctrl+F 搜索 / 复制）。failReason 也允许触发展开
                  // —— Server fault 场景下后端可能不返回 raw，但报错信息已写入
                  // device_task.error_message，必须可见。
                  rowExpandable: (row) =>
                    Boolean(row.result?.rawOutput) || Boolean(row.failReason),
                  expandedRowRender: (row) => (
                    <DeviceResultExpanded row={row} t={t} />
                  ),
                }}
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

// ---------------------------------------------------------------------------
// CommandSummary — Modal 顶部"任务信息"面板：展示执行的命令 + 用户当时勾选的 path。
// ---------------------------------------------------------------------------

const OP_COLORS: Record<string, string> = {
  LST: 'blue',
  DSP: 'blue',
  MOD: 'orange',
  ADD: 'green',
  RMV: 'red',
};

interface CommandSummaryProps {
  task: MMLTask;
  t: (key: string, values?: Record<string, string | number>) => string;
}

function CommandSummary({ task, t }: CommandSummaryProps) {
  // 优先用 commandsDetail（含 op_type / param_paths）；回退到旧的 commands string[]。
  // 多命令场景（脚本任务）：每条命令一个 sub-block，避免挤在一行不可读。
  const items: MMLTaskCommandDetail[] = useMemo(() => {
    if (task.commandsDetail && task.commandsDetail.length > 0) return task.commandsDetail;
    return (task.commands ?? []).map((code) => ({ commandCode: code }));
  }, [task.commandsDetail, task.commands]);

  return (
    <div
      style={{
        marginBottom: 12,
        padding: 12,
        background: '#fafafa',
        border: '1px solid #f0f0f0',
        borderRadius: 6,
      }}
    >
      <Descriptions
        size="small"
        column={1}
        labelStyle={{ width: 90, color: '#595959' }}
        contentStyle={{ color: '#1f2937' }}
        items={[
          {
            key: 'taskName',
            label: t('mml.taskName'),
            children: <Typography.Text strong>{task.taskName}</Typography.Text>,
          },
          {
            key: 'devices',
            label: t('mml.deviceSn'),
            children: (
              <Typography.Text type="secondary">
                {t('mml.console.actionBar.devices')}: {task.deviceSns.length}
              </Typography.Text>
            ),
          },
          {
            key: 'commands',
            label: t('mml.executedCommand'),
            children: <CommandList items={items} t={t} />,
          },
        ]}
      />
    </div>
  );
}

interface CommandListProps {
  items: MMLTaskCommandDetail[];
  t: CommandSummaryProps['t'];
}

function CommandList({ items, t }: CommandListProps) {
  if (items.length === 0) {
    return <Typography.Text type="secondary">-</Typography.Text>;
  }
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      {items.map((it, idx) => (
        <CommandBlock key={`${it.commandCode}-${idx}`} item={it} t={t} />
      ))}
    </div>
  );
}

function CommandBlock({ item, t }: { item: MMLTaskCommandDetail; t: CommandSummaryProps['t'] }) {
  const op = (item.operationType ?? '').toUpperCase();
  const opColor = OP_COLORS[op] ?? 'default';
  const paths = item.paramPaths ?? [];
  return (
    <div>
      <Space size={6} wrap>
        {op && <Tag color={opColor} style={{ marginRight: 0 }}>{op}</Tag>}
        <Typography.Text code style={{ fontSize: 13 }}>
          {item.commandCode}
        </Typography.Text>
        {paths.length > 0 && (
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            · {t('mml.selectedPathsCount', { count: paths.length })}
          </Typography.Text>
        )}
      </Space>
      {paths.length > 0 && (
        <ul
          style={{
            margin: '6px 0 0 24px',
            padding: 0,
            color: '#4b5563',
            fontFamily: 'monospace',
            fontSize: 12,
            lineHeight: 1.7,
          }}
        >
          {paths.map((p, i) => (
            <li key={`${p}-${i}`} style={{ wordBreak: 'break-all' }}>
              {p}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// DeviceResultExpanded — 设备执行结果表"展开行"内容：失败原因 + 格式化 XML +
// 原始输出 textarea（用户可 Ctrl+F 搜索 / 整段复制）。
// ---------------------------------------------------------------------------

interface DeviceResultExpandedProps {
  row: DeviceTaskResultItem;
  t: CommandSummaryProps['t'];
}

function DeviceResultExpanded({ row, t }: DeviceResultExpandedProps) {
  const raw = row.result?.rawOutput;

  return (
    <Space direction="vertical" size={12} style={{ width: '100%' }}>
      {row.failReason && (
        <Alert
          type="error"
          showIcon
          message={t('mml.failReason')}
          description={
            <Typography.Paragraph
              copyable={{ text: row.failReason }}
              style={{
                margin: 0,
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-word',
                fontFamily: 'monospace',
                fontSize: 12,
              }}
            >
              {row.failReason}
            </Typography.Paragraph>
          }
        />
      )}
      <ParsedResultPanel row={row} t={t} />
      {raw && (
        <div>
          <Typography.Text strong style={{ fontSize: 12, color: '#595959' }}>
            {t('mml.formattedXml')}
          </Typography.Text>
          <div style={{ marginTop: 4 }}>
            {/* XmlViewer 已支持折叠/复制/搜索，原「原始输出」textarea 与此重复，已移除 */}
            <XmlViewer xml={raw} maxHeight={320} />
          </div>
        </div>
      )}
    </Space>
  );
}

// ---------------------------------------------------------------------------
// T-0168: 列表行展开 — 命令信息 + 路径转换详情 + 设备执行结果三段式
// ---------------------------------------------------------------------------

interface TaskRowExpandedDetailProps {
  task: MMLTask;
  t: (key: string, values?: Record<string, string | number>) => string;
}

/**
 * T-0168 列表行展开组件 — 替代 Modal 入口（老 Modal "查看" 按钮仍保留兜底）。
 *
 * 数据加载策略：useMMLTaskResults 仅在该行展开时调用（hook 内部 enabled = id 非空），
 * 列表渲染时不预拉，避免列表页性能受影响。
 *
 * 三段结构：
 *   1. product_resolved=false 时顶部橙色 Banner（PRD §GWT-3）
 *   2. CommandSummary 复用现有组件
 *   3. PathTranslationTable 新组件（T-0168 核心）
 *   4. DeviceResultsTable 简化版（仅必要列）
 */
function TaskRowExpandedDetail({ task, t }: TaskRowExpandedDetailProps) {
  const { data: resultsData, isLoading } = useMMLTaskResults(task.id, 1, 200);
  // T-0168: stats 字段包含 path_translations + 任务级翻译审计
  const stats = (resultsData as { stats?: MMLTaskResultsStats } | undefined)?.stats;
  const pathTranslations: MMLPathTranslationView[] = stats?.pathTranslations ?? [];
  const productResolved = stats?.productResolved ?? task.productResolved ?? true;
  const matchedProductClass = stats?.matchedProductClass ?? task.matchedProductClass;

  return (
    <Space direction="vertical" size={12} style={{ width: '100%' }}>
      {!productResolved && (
        <Alert
          type="warning"
          showIcon
          message={t('mml.productUnresolvedBanner')}
          description={t('mml.productUnresolvedBannerDetail', {
            productClass: matchedProductClass || '-',
          })}
        />
      )}

      <CommandSummary task={task} t={t} />

      <PathTranslationTable rows={pathTranslations} loading={isLoading} t={t} />

      {task.pathTranslationWarning?.anyMiss && (
        <Alert
          type="warning"
          showIcon
          message={t('mml.pathTranslationWarning')}
          description={t('mml.pathTranslationWarningDetail', {
            deviceCount: task.pathTranslationWarning.deviceCount,
            pathCount: task.pathTranslationWarning.pathCount,
          })}
        />
      )}
    </Space>
  );
}

// ---------------------------------------------------------------------------
// T-0168: PathTranslationTable — 路径转换详情表
// ---------------------------------------------------------------------------

const SOURCE_TAG_CONFIG: Record<PathTranslationSource, { color: string; key: string }> = {
  discovered:         { color: 'green',  key: 'mml.translation.sourceDiscovered' },
  default:            { color: 'green',  key: 'mml.translation.sourceDefault' },
  passthrough:        { color: 'gold',   key: 'mml.translation.sourcePassthrough' },
  orphan_passthrough: { color: 'orange', key: 'mml.translation.sourceOrphan' },
  mixed:              { color: 'blue',   key: 'mml.translation.sourceMixed' },
};

interface PathTranslationTableProps {
  rows: MMLPathTranslationView[];
  loading: boolean;
  t: TaskRowExpandedDetailProps['t'];
}

function PathTranslationTable({ rows, loading, t }: PathTranslationTableProps) {
  const columns: Array<{
    key: string;
    title: string;
    dataIndex?: keyof MMLPathTranslationView;
    width?: number;
    render?: (val: unknown, row: MMLPathTranslationView) => React.ReactNode;
  }> = [
    {
      key: 'standardPath',
      title: t('mml.translation.standardPath'),
      dataIndex: 'standardPath',
      render: (v) => <Typography.Text code style={{ fontSize: 12 }}>{String(v)}</Typography.Text>,
    },
    {
      key: 'privatePath',
      title: t('mml.translation.privatePath'),
      dataIndex: 'privatePath',
      render: (v, row) => (
        <Typography.Text
          code
          style={{
            fontSize: 12,
            color: row.translated ? '#1f2937' : '#9ca3af',
          }}
        >
          {String(v)}
        </Typography.Text>
      ),
    },
    {
      key: 'source',
      title: t('mml.translation.source'),
      dataIndex: 'translationSource',
      width: 220,
      render: (val) => {
        const v = val as PathTranslationSource;
        const cfg = SOURCE_TAG_CONFIG[v];
        return cfg ? <Tag color={cfg.color}>{t(cfg.key)}</Tag> : <Tag>{String(val ?? '-')}</Tag>;
      },
    },
  ];
  return (
    <div>
      <Typography.Text strong style={{ display: 'block', marginBottom: 6 }}>
        {t('mml.translation.tableTitle')}
      </Typography.Text>
      <Table
        size="small"
        rowKey="standardPath"
        dataSource={rows}
        columns={columns}
        loading={loading}
        pagination={false}
        locale={{ emptyText: t('mml.translation.noData') }}
      />
    </div>
  );
}

// ---------------------------------------------------------------------------
// ParsedResultPanel — 2026-05-28 任务记录"查看"展开行新增"执行结果解析"区。
//
// 根据 device_tasks.result JSONB (透传到 row.result.parsedData) 的 method 字段
// 分支渲染:
//   - GetParameterValuesResponse(LST) → 三列表格 path/值/类型,固定高度+滚动
//   - SetParameterValuesResponse (MOD) → status=0 立即生效 / =1 需重启
//   - AddObjectResponse → 实例号 + status
//   - DeleteObjectResponse → 成功
//   - RebootResponse / FactoryResetResponse → 指令已下发
//   - 其它 / 缺失 → 灰字提示"无法解析"
//
// 解析逻辑复用 frontend-core/src/utils/mmlResultParser.ts(已 unit-tested)。
// ---------------------------------------------------------------------------

interface ParsedResultPanelProps {
  row: DeviceTaskResultItem;
  t: CommandSummaryProps['t'];
}

function ParsedResultPanel({ row, t }: ParsedResultPanelProps) {
  // result.parsedData 由后端 deviceTaskRowToResultMap 透传整个 device_tasks.result
  // JSONB,内含 { method, raw_response, instance_number } —— 正是 parser 期望入参。
  const parsed = useMemo<ParsedMmlResult | null>(
    () => parseMmlDeviceTaskResult(row.result?.parsedData ?? null),
    [row.result?.parsedData],
  );

  return (
    <div>
      <Typography.Text strong style={{ fontSize: 12, color: '#595959' }}>
        {t('mml.taskResult.parsed.title')}
      </Typography.Text>
      <div style={{ marginTop: 4 }}>
        {parsed === null ? (
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {t('mml.taskResult.parsed.notParsable')}
          </Typography.Text>
        ) : (
          <ParsedResultBody parsed={parsed} t={t} />
        )}
      </div>
    </div>
  );
}

function ParsedResultBody({
  parsed,
  t,
}: {
  parsed: ParsedMmlResult;
  t: CommandSummaryProps['t'];
}) {
  switch (parsed.kind) {
    case 'gpv':
      return <ParsedGpvTable params={parsed.params ?? []} t={t} />;
    case 'spv':
      return <ParsedStatusLine status={parsed.status} t={t} />;
    case 'add':
      return (
        <Space size={8} wrap>
          <Tag color="success">
            {t('mml.taskResult.parsed.add.success', {
              n: parsed.instanceNumber ?? '-',
            })}
          </Tag>
          {parsed.status === 1 && (
            <Tag color="warning">{t('mml.taskResult.parsed.spv.reboot')}</Tag>
          )}
        </Space>
      );
    case 'delete':
      return <Tag color="success">{t('mml.taskResult.parsed.delete.success')}</Tag>;
    case 'reboot':
      return <Tag color="success">{t('mml.taskResult.parsed.reboot.success')}</Tag>;
    default:
      return (
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          {t('mml.taskResult.parsed.notParsable')}
        </Typography.Text>
      );
  }
}

function ParsedStatusLine({
  status,
  t,
}: {
  status?: number;
  t: CommandSummaryProps['t'];
}) {
  if (status === 0) {
    return <Tag color="success">{t('mml.taskResult.parsed.spv.immediate')}</Tag>;
  }
  if (status === 1) {
    return <Tag color="warning">{t('mml.taskResult.parsed.spv.reboot')}</Tag>;
  }
  return (
    <Typography.Text type="secondary" style={{ fontSize: 12 }}>
      {t('mml.taskResult.parsed.notParsable')}
    </Typography.Text>
  );
}

/** LST 解析表格:用户决策 2026-05-28 固定高度 280 + 滚动,
 *  避免 20+ path 把展开行撑得很长。 */
function ParsedGpvTable({
  params,
  t,
}: {
  params: ParsedParamValue[];
  t: CommandSummaryProps['t'];
}) {
  const columns = [
    {
      key: 'name',
      title: t('mml.taskResult.parsed.gpv.path'),
      dataIndex: 'name',
      render: (v: string) => (
        <Typography.Text
          code
          style={{
            fontSize: 12,
            wordBreak: 'break-all',
            whiteSpace: 'normal',
          }}
        >
          {v}
        </Typography.Text>
      ),
    },
    {
      key: 'value',
      title: t('mml.taskResult.parsed.gpv.value'),
      dataIndex: 'value',
      width: 220,
      render: (v: string) => (
        <Typography.Text
          style={{
            fontFamily: 'monospace',
            fontSize: 12,
            wordBreak: 'break-all',
            whiteSpace: 'normal',
          }}
        >
          {v === '' ? <span style={{ color: '#bfbfbf' }}>(empty)</span> : v}
        </Typography.Text>
      ),
    },
    {
      key: 'type',
      title: t('mml.taskResult.parsed.gpv.type'),
      dataIndex: 'type',
      width: 120,
      render: (v?: string) =>
        v ? (
          <Tag style={{ fontFamily: 'monospace', fontSize: 11 }}>{v}</Tag>
        ) : (
          <span style={{ color: '#bfbfbf' }}>-</span>
        ),
    },
  ];
  return (
    <Table
      size="small"
      rowKey={(_, idx) => String(idx)}
      dataSource={params}
      columns={columns}
      pagination={false}
      scroll={{ y: 280 }}
      locale={{ emptyText: t('mml.taskResult.parsed.gpv.empty') }}
    />
  );
}
