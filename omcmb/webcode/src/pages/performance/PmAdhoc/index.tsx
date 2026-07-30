/**
 * T-0164-P7 / G7 自定义聚合任务管理页面。
 *
 * T-0186：
 *   - 列表分两区：内置区（is_builtin=true）+ 自建区（is_builtin=false），各一张 Table。
 *   - 任务详情直接展示在线聚合结果；新执行模型不提供补跑进度。
 *   - 详情顶部制式渲染为只读 Tag（建后不可改、无切换控件）。
 *
 * T-0164 收尾：
 *   G7-Gap-2  结果查看用 AdhocResultPanel（G6 panel 风格，多指标多 series ECharts）
 *   G6-Gap-12 联动：PanelConfigDrawer 跳转携带 ?preset=panel&device_sns=...&metric_paths=...&granularities=...
 */

import { useEffect, useMemo, useState } from 'react';
import { formatSystemTime } from '@core/utils/systemTime';
import { useIntl, type IntlShape } from 'react-intl';
import { useNavigate, useSearchParams } from 'react-router-dom';
import {
  Button,
  Card,
  Drawer,
  Modal,
  Space,
  Table,
  Tag,
  Progress,
  Descriptions,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { PlusOutlined, DeleteOutlined, EditOutlined, StopOutlined, PlayCircleOutlined } from '@ant-design/icons';
import {
  usePmAdhocList,
  usePmAdhocRuns,
  useCancelPmAdhoc,
  useDeletePmAdhoc,
  useResumePmAdhoc,
} from '@core/hooks/api/usePmAdhoc';
import { useUserStore } from '@core/store/userStore';
import {
  useAdhocProgressStream,
  type AdhocLiveProgress,
} from '@core/hooks/api/useAdhocProgress';
import type {
  AdhocStatus,
  AdhocTask,
  AdhocTaskRun,
} from '@core/types/pmAdhoc';
import { useTechnologyDictionary } from '@core/hooks/api/useTechnologyDictionary';
import { displayAdhocTaskName } from '../adhocTaskDisplay';

// issue #399：SSE 接通后进度由事件实时驱动，轮询降为低频兜底（状态翻转 + 断连兜底）。
const ADHOC_POLL_FALLBACK_MS = 30000;
// 运行中（含 pending）才订阅 SSE；scheduled/终态不建连。
function isRunningStatus(s: AdhocStatus): boolean {
  return s === 'running' || s === 'pending';
}
import { CreateAdhocTaskDrawer, type CreateAdhocPreset } from './CreateAdhocTaskDrawer';
import { AdhocResultPanel } from './AdhocResultPanel';
import BuiltinMetricEditModal from './BuiltinMetricEditModal';
import SelectedMetricsTags from './SelectedMetricsTags';
import {
  clearBuiltinMetricDraft,
  clearCustomWizardDraft,
  getActiveBuiltinMetricTaskId,
  getActiveCustomWizard,
} from './pmAdhocDraftState';

const statusColor: Record<AdhocStatus, string> = {
  pending: 'default',
  running: 'processing',
  scheduled: 'cyan',
  succeeded: 'success',
  failed: 'error',
  canceled: 'warning',
};

// 状态 / 维度 / 制式 标签：保留"无键回退原值"语义（未命中映射时显原始枚举）。
const STATUS_LABEL_KEY: Record<string, string> = {
  pending: 'perf.adhoc.statusPending',
  running: 'perf.adhoc.statusRunning',
  scheduled: 'perf.adhoc.statusScheduled',
  succeeded: 'perf.adhoc.statusSucceeded',
  failed: 'perf.adhoc.statusFailed',
  canceled: 'perf.adhoc.statusCanceled',
};

const DIMENSION_LABEL_KEY: Record<string, string> = {
  device: 'perf.adhoc.dimDevice',
  aggregate_group: 'perf.adhoc.dimAggregateGroup',
  product: 'perf.adhoc.dimProduct',
  band: 'perf.adhoc.dimBand',
  network: 'perf.adhoc.dimNetwork',
  device_group: 'perf.adhoc.dimDeviceGroup',
};

const GRANULARITY_LABEL_KEY: Record<string, string> = {
  hourly: 'perf.adhoc.gran.hourly',
  daily: 'perf.adhoc.gran.daily',
  weekly: 'perf.adhoc.gran.weekly',
  monthly: 'perf.adhoc.gran.monthly',
};

const VISIBILITY_LABEL_KEY: Record<string, string> = {
  private: 'perf.adhoc.visibilityPrivate',
  public: 'perf.adhoc.visibilityPublic',
};

export const RUN_HISTORY_SCROLL_X = 1250;

// 「聚合范围」说明句语料键：仅产品/设备组/频段三维度是「按制式全量、按维度分组」（无子集可选），
// 各给一句范围声明（带制式 / 不限制式两版，避免空制式时多余空格）。其余维度不渲染该行
// （设备/自选设备维度已由「选定设备」行表达，全网无分组）。
const SCOPE_KEY: Record<string, { withTech: string; anyTech: string }> = {
  product: { withTech: 'perf.adhoc.scopeAllProduct', anyTech: 'perf.adhoc.scopeAllProductAny' },
  device_group: {
    withTech: 'perf.adhoc.scopeAllDeviceGroup',
    anyTech: 'perf.adhoc.scopeAllDeviceGroupAny',
  },
  band: { withTech: 'perf.adhoc.scopeAllBand', anyTech: 'perf.adhoc.scopeAllBandAny' },
};

function statusLabel(intl: IntlShape, s?: string): string {
  const id = s ? STATUS_LABEL_KEY[s] : undefined;
  return id ? intl.formatMessage({ id }) : (s ?? '');
}

function dimensionLabel(intl: IntlShape, d?: string): string {
  const id = d ? DIMENSION_LABEL_KEY[d] : undefined;
  return id ? intl.formatMessage({ id }) : (d ?? '');
}

export function granularityLabel(
  intl: Pick<IntlShape, 'formatMessage'>,
  granularities?: string[],
): string {
  if (!granularities || granularities.length === 0) {
    return '—';
  }
  return granularities
    .map((g) => {
      const id = GRANULARITY_LABEL_KEY[g];
      return id ? intl.formatMessage({ id }) : g;
    })
    .join(' / ');
}

function fmtTime(v?: string | null): string {
  if (!v) return '—';
  return formatSystemTime(v, { placeholder: '—' });
}

/** 运行历史表（任务详情 Tab）。 */
export function RunHistoryTab({ taskId }: { taskId: string }) {
  const intl = useIntl();
  // 运行中任务会持续产生 run；issue #399 SSE 接通后轮询降为低频兜底。
  const { data: runs = [], isLoading } = usePmAdhocRuns(taskId, {
    refetchInterval: ADHOC_POLL_FALLBACK_MS,
  });

  // issue #399：当本任务有运行中的 run 时，订阅该任务进度 SSE 实时驱动；
  // 终态/无运行中 run 时不传 id → 不建连。
  const hasRunningRun = runs.some((r) => isRunningStatus(r.status));
  const live = useAdhocProgressStream(hasRunningRun ? [taskId] : []);
  const liveRows = live.get(taskId)?.rows;

  const columns: ColumnsType<AdhocTaskRun> = useMemo(
    () => [
      { title: intl.formatMessage({ id: 'perf.adhoc.colSeq' }), dataIndex: 'runSeq', width: 70 },
      {
        title: intl.formatMessage({ id: 'perf.adhoc.colGranularity' }),
        dataIndex: 'granularity',
        width: 100,
        render: (g: string) => g || '—',
      },
      {
        title: intl.formatMessage({ id: 'perf.adhoc.colDimension' }),
        dataIndex: 'dimension',
        width: 100,
        render: (d: string) => (d ? dimensionLabel(intl, d) : '—'),
      },
      {
        title: intl.formatMessage({ id: 'perf.adhoc.colWindow' }),
        width: 240,
        render: (_, r) =>
          r.windowStart || r.windowEnd
            ? `${fmtTime(r.windowStart)} ~ ${fmtTime(r.windowEnd)}`
            : intl.formatMessage({ id: 'perf.adhoc.rollingWindow' }),
      },
      {
        title: intl.formatMessage({ id: 'perf.adhoc.colStatus' }),
        dataIndex: 'status',
        width: 90,
        render: (s: AdhocStatus) => <Tag color={statusColor[s]}>{statusLabel(intl, s)}</Tag>,
      },
      { title: intl.formatMessage({ id: 'perf.adhoc.colQueuedAt' }), dataIndex: 'queuedAt', width: 180, render: (v: string) => fmtTime(v) },
      { title: intl.formatMessage({ id: 'perf.adhoc.colFinishedAt' }), dataIndex: 'finishedAt', width: 180, render: (v: string) => fmtTime(v) },
      {
        title: intl.formatMessage({ id: 'perf.adhoc.colRowsTotal' }),
        dataIndex: 'rowsTotal',
        width: 90,
        // issue #399：运行中的 run 行数由 SSE progress 事件实时驱动（live 优先）。
        render: (v: number, r) =>
          isRunningStatus(r.status) && liveRows != null ? liveRows : v,
      },
      {
        title: intl.formatMessage({ id: 'perf.adhoc.colError' }),
        dataIndex: 'error',
        ellipsis: true,
        render: (e: string) => (e ? <Typography.Text type="danger">{e}</Typography.Text> : '—'),
      },
    ],
    [intl, liveRows],
  );

  return (
    <Table<AdhocTaskRun>
      rowKey="id"
      size="small"
      loading={isLoading}
      dataSource={runs}
      columns={columns}
      scroll={{ x: RUN_HISTORY_SCROLL_X }}
      pagination={{ pageSize: 10, size: 'small' }}
      expandable={{
        // failed 行 error 全文可展开
        rowExpandable: (r) => Boolean(r.error),
        expandedRowRender: (r) => (
          <Typography.Paragraph
            type="danger"
            style={{ whiteSpace: 'pre-wrap', marginBottom: 0 }}
          >
            {r.error}
          </Typography.Paragraph>
        ),
      }}
    />
  );
}

/** 单张任务表（内置区 / 自建区共用）。 */
function TaskTable({
  tasks,
  loading,
  isBuiltinArea,
  liveProgress,
  onView,
  onCancel,
  onEdit,
  onDelete,
  onResume,
  currentUsername,
  isSuperAdmin,
}: {
  tasks: AdhocTask[];
  loading: boolean;
  // 内置区 = true：操作列给「编辑指标」（开轻量弹窗）；自建区 = false：给「编辑」（进向导编辑页）。
  isBuiltinArea: boolean;
  // issue #399：运行中任务的实时进度（live 优先于轮询拿到的 task.progress）。
  liveProgress: ReadonlyMap<string, AdhocLiveProgress>;
  onView: (t: AdhocTask) => void;
  onCancel: (id: string) => void;
  onEdit: (t: AdhocTask) => void;
  // issue #392：删除终态自建任务（仅自建区传入；内置区不传，按钮恒不渲染）。
  onDelete?: (t: AdhocTask) => void;
  // #674：恢复已取消任务。
  onResume?: (id: string) => void;
  currentUsername: string;
  isSuperAdmin: boolean;
}) {
  const intl = useIntl();
  const { labelForTechnology } = useTechnologyDictionary();
  const columns: ColumnsType<AdhocTask> = useMemo(
    () => [
      {
        title: intl.formatMessage({ id: 'perf.adhoc.colName' }),
        dataIndex: 'name',
        render: (_: string, r) => displayAdhocTaskName(r, labelForTechnology),
      },
      {
        title: intl.formatMessage({ id: 'perf.adhoc.colStatus' }),
        dataIndex: 'status',
        width: 100,
        render: (s: AdhocStatus) => <Tag color={statusColor[s]}>{statusLabel(intl, s)}</Tag>,
      },
      {
        title: intl.formatMessage({ id: 'perf.adhoc.colProgress' }),
        dataIndex: 'progress',
        width: 140,
        // issue #399：运行中进度 live 优先（SSE 事件驱动），缺 live 时回退轮询拿到的 task.progress。
        render: (p: number, r) =>
          r.status === 'running' || r.status === 'succeeded' ? (
            <Progress percent={liveProgress.get(r.id)?.progress ?? p} size="small" />
          ) : (
            '—'
          ),
      },
      // 创建时间列仅自建区保留；内置区去掉（内置任务创建时间无意义）。设备数列两区都去。
      ...(isBuiltinArea
        ? []
        : [
            {
              title: intl.formatMessage({ id: 'perf.adhoc.colVisibility' }),
              dataIndex: 'visibility',
              width: 90,
              render: (v: string) => (
                <Tag color={v === 'public' ? 'green' : undefined}>
                  {intl.formatMessage({ id: VISIBILITY_LABEL_KEY[v] ?? 'perf.adhoc.visibilityPrivate' })}
                </Tag>
              ),
            } as ColumnsType<AdhocTask>[number],
            {
              title: intl.formatMessage({ id: 'perf.adhoc.colCreator' }),
              dataIndex: 'creator',
              width: 120,
              render: (v: string) => v || '—',
            } as ColumnsType<AdhocTask>[number],
            {
              title: intl.formatMessage({ id: 'perf.adhoc.colCreatedAt' }),
              dataIndex: 'createdAt',
              width: 180,
              render: (v: string) => fmtTime(v),
            } as ColumnsType<AdhocTask>[number],
            {
              title: intl.formatMessage({ id: 'perf.adhoc.colPlannedEndAt' }),
              dataIndex: 'plannedEndAt',
              width: 180,
              render: (_: string | undefined, r: AdhocTask) =>
                r.mode === 'continuous' ? fmtTime(r.plannedEndAt) : '—',
            } as ColumnsType<AdhocTask>[number],
          ]),
      {
        title: intl.formatMessage({ id: 'perf.adhoc.colOperation' }),
        width: 230,
        render: (_, r) => {
          const isOwner = r.creator === currentUsername;
          const publicTask = r.visibility === 'public';
          const canManage = isBuiltinArea || isSuperAdmin || isOwner || publicTask;
          const canCancel = isBuiltinArea || isSuperAdmin || isOwner;
          return (
            <Space>
              <Button size="small" onClick={() => onView(r)}>
                {intl.formatMessage({ id: 'perf.adhoc.btnViewDetail' })}
              </Button>
              {/* T-0194：内置区给「编辑指标」（只改指标集），自建区给「编辑」（进向导编辑页） */}
              {canManage && (
                <Button size="small" icon={<EditOutlined />} onClick={() => onEdit(r)}>
                  {isBuiltinArea
                    ? intl.formatMessage({ id: 'perf.adhoc.btnEditMetrics' })
                    : intl.formatMessage({ id: 'perf.adhoc.btnEdit' })}
                </Button>
              )}
              {canCancel && (r.status === 'pending' || r.status === 'running' || r.status === 'scheduled') && (
                <Button size="small" danger icon={<StopOutlined />} onClick={() => onCancel(r.id)}>
                  {intl.formatMessage({ id: 'perf.adhoc.btnCancel' })}
                </Button>
              )}
              {/* #674：已取消任务给「启用」恢复执行。 */}
              {canManage && onResume && r.status === 'canceled' && (
                <Button size="small" type="primary" ghost icon={<PlayCircleOutlined />} onClick={() => onResume(r.id)}>
                  {intl.formatMessage({ id: 'perf.adhoc.btnResume' })}
                </Button>
              )}
              {/* issue #392：终态(成功/失败/已取消)自建任务给「删除」（onDelete 仅自建区传入）。 */}
              {canManage && onDelete &&
                (r.status === 'succeeded' || r.status === 'failed' || r.status === 'canceled') && (
                  <Button
                    size="small"
                    danger
                    icon={<DeleteOutlined />}
                    onClick={() => onDelete(r)}
                  >
                    {intl.formatMessage({ id: 'perf.adhoc.btnDelete' })}
                  </Button>
                )}
            </Space>
          );
        },
      },
    ],
    [
      intl,
      isBuiltinArea,
      liveProgress,
      onView,
      onCancel,
      onEdit,
      onDelete,
      onResume,
      currentUsername,
      isSuperAdmin,
      labelForTechnology,
    ],
  );

  return (
    <Table<AdhocTask>
      rowKey="id"
      size="small"
      loading={loading}
      dataSource={tasks}
      columns={columns}
      pagination={{ pageSize: 10, size: 'small' }}
    />
  );
}

export default function PmAdhocPage() {
  const intl = useIntl();
  const { labelForTechnology } = useTechnologyDictionary();
  const currentUsername = useUserStore((s) => s.currentUser?.username ?? '');
  const isSuperAdmin = useUserStore((s) => Boolean(s.currentUser?.isSuperAdmin));
  // T-0186：分两区，各发一次 list（内置 / 自建）。
  const { data: builtinTasks = [], isLoading: builtinLoading } = usePmAdhocList({
    refetchInterval: ADHOC_POLL_FALLBACK_MS,
    isBuiltin: true,
  });
  const { data: customTasks = [], isLoading: customLoading } = usePmAdhocList({
    refetchInterval: ADHOC_POLL_FALLBACK_MS,
    isBuiltin: false,
  });

  // issue #399：收集两区运行中（含 pending）任务 id，订阅进度 SSE；终态/scheduled 不订阅。
  const runningIds = useMemo(
    () =>
      [...builtinTasks, ...customTasks]
        .filter((t) => isRunningStatus(t.status))
        .map((t) => t.id),
    [builtinTasks, customTasks],
  );
  const liveProgress = useAdhocProgressStream(runningIds);

  const cancelMut = useCancelPmAdhoc();
  const deleteMut = useDeletePmAdhoc();
  const resumeMut = useResumePmAdhoc();
  const navigate = useNavigate();

  const [searchParams, setSearchParams] = useSearchParams();
  const [createOpen, setCreateOpen] = useState(false);
  const [createPreset, setCreatePreset] = useState<CreateAdhocPreset | undefined>(undefined);
  const [selectedTask, setSelectedTask] = useState<AdhocTask | null>(null);
  // T-0194：内置任务「编辑指标」弹窗状态。
  const [builtinEditTask, setBuiltinEditTask] = useState<AdhocTask | null>(null);

  useEffect(() => {
    const activeCustomWizard = getActiveCustomWizard();
    if (!activeCustomWizard) return;
    if (activeCustomWizard.mode === 'new') {
      navigate('/performance/pm-adhoc/new');
      return;
    }
    if (customLoading) return;
    if (!customTasks.some((task) => task.id === activeCustomWizard.taskId)) {
      clearCustomWizardDraft(activeCustomWizard);
      return;
    }
    navigate(`/performance/pm-adhoc/${activeCustomWizard.taskId}/edit`);
  }, [customLoading, customTasks, navigate]);

  useEffect(() => {
    if (builtinEditTask || builtinLoading) return;
    const activeTaskId = getActiveBuiltinMetricTaskId();
    if (!activeTaskId) return;
    const activeTask = builtinTasks.find((task) => task.id === activeTaskId);
    if (activeTask) {
      setBuiltinEditTask(activeTask);
      return;
    }
    clearBuiltinMetricDraft(activeTaskId);
  }, [builtinEditTask, builtinLoading, builtinTasks]);

  // G6-Gap-12 联动：URL preset 触发自动打开 Drawer
  useEffect(() => {
    if (searchParams.get('preset') === 'panel') {
      const deviceSns = (searchParams.get('device_sns') ?? '')
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean);
      const metricPaths = (searchParams.get('metric_paths') ?? '')
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean);
      const granularities = (searchParams.get('granularities') ?? '')
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean);
      setCreatePreset({
        name: intl.formatMessage({ id: 'perf.adhoc.derivedName' }, { count: deviceSns.length }),
        deviceSns,
        metricPaths,
        granularities,
      });
      setCreateOpen(true);
      const next = new URLSearchParams(searchParams);
      next.delete('preset');
      next.delete('device_sns');
      next.delete('metric_paths');
      next.delete('granularities');
      setSearchParams(next, { replace: true });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // T-0194：自建任务编辑 → 进向导编辑页（/:id/edit）；内置任务编辑 → 开「编辑指标」弹窗。
  const handleEditCustom = (t: AdhocTask) => {
    navigate(`/performance/pm-adhoc/${t.id}/edit`);
  };
  const handleEditBuiltin = (t: AdhocTask) => {
    setBuiltinEditTask(t);
  };

  const handleCancel = (id: string) => {
    Modal.confirm({
      title: intl.formatMessage({ id: 'perf.adhoc.cancelTaskTitle' }),
      content: intl.formatMessage({ id: 'perf.adhoc.cancelTaskContent' }),
      onOk: async () => {
        await cancelMut.mutateAsync(id);
        message.success(intl.formatMessage({ id: 'perf.adhoc.canceled' }));
      },
    });
  };

  // #674：恢复已取消任务（二次确认 → resume → 列表自动刷新，状态变更）。
  const handleResume = (id: string) => {
    Modal.confirm({
      title: intl.formatMessage({ id: 'perf.adhoc.resumeTaskTitle' }),
      content: intl.formatMessage({ id: 'perf.adhoc.resumeTaskContent' }),
      onOk: async () => {
        try {
          await resumeMut.mutateAsync(id);
          message.success(intl.formatMessage({ id: 'perf.adhoc.resumed' }));
        } catch (e) {
          message.error(
            intl.formatMessage(
              { id: 'perf.adhoc.resumeFailed' },
              { msg: (e as Error).message },
            ),
          );
        }
      },
    });
  };

  // issue #392：删除终态自建任务（二次确认 → 删除 → 列表自动刷新，任务消失）。
  const handleDelete = (t: AdhocTask) => {
    Modal.confirm({
      title: intl.formatMessage({ id: 'perf.adhoc.deleteTaskTitle' }),
      content: intl.formatMessage({ id: 'perf.adhoc.deleteTaskContent' }),
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          await deleteMut.mutateAsync(t.id);
          message.success(intl.formatMessage({ id: 'perf.adhoc.deleted' }));
        } catch (e) {
          message.error(
            intl.formatMessage(
              { id: 'perf.adhoc.deleteFailed' },
              { msg: (e as Error).message },
            ),
          );
        }
      },
    });
  };

  if (builtinEditTask) {
    return (
      <BuiltinMetricEditModal
        task={builtinEditTask}
        open
        onClose={() => {
          clearBuiltinMetricDraft(builtinEditTask.id);
          setBuiltinEditTask(null);
        }}
      />
    );
  }

  return (
    <Space orientation="vertical" size="large" style={{ width: '100%' }}>
      <Card
        title={intl.formatMessage({ id: 'perf.adhoc.cardBuiltin' })}
        size="small"
        extra={<Tag color="blue">{intl.formatMessage({ id: 'perf.adhoc.systemPreset' })}</Tag>}
      >
        <TaskTable
          tasks={builtinTasks}
          loading={builtinLoading}
          isBuiltinArea
          liveProgress={liveProgress}
          onView={setSelectedTask}
          onCancel={handleCancel}
          onEdit={handleEditBuiltin}
          onResume={handleResume}
          currentUsername={currentUsername}
          isSuperAdmin={isSuperAdmin}
        />
      </Card>

      <Card
        title={intl.formatMessage({ id: 'perf.adhoc.cardCustom' })}
        size="small"
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => navigate('/performance/pm-adhoc/new')}
          >
            {intl.formatMessage({ id: 'perf.adhoc.btnNewTask' })}
          </Button>
        }
      >
        <TaskTable
          tasks={customTasks}
          loading={customLoading}
          isBuiltinArea={false}
          liveProgress={liveProgress}
          onView={setSelectedTask}
          onCancel={handleCancel}
          onEdit={handleEditCustom}
          onDelete={handleDelete}
          onResume={handleResume}
          currentUsername={currentUsername}
          isSuperAdmin={isSuperAdmin}
        />
      </Card>

      <CreateAdhocTaskDrawer
        open={createOpen}
        preset={createPreset}
        onClose={() => {
          setCreateOpen(false);
          setCreatePreset(undefined);
        }}
      />

      <Drawer
        title={
          selectedTask
            ? intl.formatMessage(
                { id: 'perf.adhoc.detailTitle' },
                { name: displayAdhocTaskName(selectedTask, labelForTechnology) },
              )
            : intl.formatMessage({ id: 'perf.adhoc.detailTitleDefault' })
        }
        size={920}
        open={Boolean(selectedTask)}
        onClose={() => setSelectedTask(null)}
        destroyOnClose
      >
        {selectedTask && (
          <>
            <Descriptions size="small" column={2} bordered style={{ marginBottom: 16 }}>
              <Descriptions.Item label={intl.formatMessage({ id: 'perf.adhoc.descDimension' })}>
                {dimensionLabel(intl, selectedTask.dimension)}
              </Descriptions.Item>
              <Descriptions.Item label={intl.formatMessage({ id: 'perf.adhoc.descGranularity' })}>
                {granularityLabel(intl, selectedTask.granularities)}
              </Descriptions.Item>
              <Descriptions.Item label={intl.formatMessage({ id: 'perf.adhoc.descTech' })}>
                {/* T-0186：制式只读，建后不可改，无切换控件 */}
                {selectedTask.technology ? (
                  <Tag color="geekblue">{labelForTechnology(selectedTask.technology)}</Tag>
                ) : (
                  <Tag>{intl.formatMessage({ id: 'perf.adhoc.anyTech' })}</Tag>
                )}
              </Descriptions.Item>
              <Descriptions.Item label={intl.formatMessage({ id: 'perf.adhoc.descStatus' })}>
                <Tag color={statusColor[selectedTask.status]}>
                  {statusLabel(intl, selectedTask.status)}
                </Tag>
              </Descriptions.Item>
              {!selectedTask.isBuiltin && (
                <Descriptions.Item label={intl.formatMessage({ id: 'perf.adhoc.descVisibility' })}>
                  <Tag color={selectedTask.visibility === 'public' ? 'green' : undefined}>
                    {intl.formatMessage({
                      id: selectedTask.visibility === 'public'
                        ? 'perf.adhoc.visibilityPublic'
                        : 'perf.adhoc.visibilityPrivate',
                    })}
                  </Tag>
                </Descriptions.Item>
              )}
              {!selectedTask.isBuiltin && selectedTask.mode === 'continuous' && (
                <Descriptions.Item label={intl.formatMessage({ id: 'perf.adhoc.descPlannedEndAt' })}>
                  {fmtTime(selectedTask.plannedEndAt)}
                </Descriptions.Item>
              )}
              {/* T-0194：已选指标 — 按制式解析为可读指标名（编号+名），查不到回退显编号 */}
              <Descriptions.Item
                label={intl.formatMessage({ id: 'perf.adhoc.descSelectedMetrics' })}
                span={2}
              >
                <SelectedMetricsTags
                  metricPaths={selectedTask.metricPaths}
                  technology={selectedTask.technology}
                />
              </Descriptions.Item>
              {/* 选定设备 — 按设备/自选设备维度圈定的设备清单（数据已落库回传，仅详情未渲染）。
                  全量聚合维度（产品/频段/全网/设备组）不挑具体设备 → deviceSns 空则不显示此行。 */}
              {selectedTask.deviceSns.length > 0 && (
                <Descriptions.Item
                  label={intl.formatMessage({ id: 'perf.adhoc.descDevices' })}
                  span={2}
                >
                  <Typography.Text type="secondary" style={{ marginRight: 8 }}>
                    {intl.formatMessage(
                      { id: 'perf.adhoc.descDevicesCount' },
                      { count: selectedTask.deviceSns.length },
                    )}
                  </Typography.Text>
                  {selectedTask.deviceSns.map((sn) => (
                    <Tag key={sn} style={{ marginBottom: 4 }}>
                      {sn}
                    </Tag>
                  ))}
                </Descriptions.Item>
              )}
              {/* 聚合范围 — 产品/设备组/频段维度是「按制式全量聚合、按维度分组」，无子集可选，
                  故给一句范围声明（如「包含全部 LTE 产品（按产品分组）」）而非清单。制式见上一行，
                  此处只在句中点出制式短名；不限制式时用 anyTech 版避免多余空格。 */}
              {SCOPE_KEY[selectedTask.dimension] && (
                <Descriptions.Item
                  label={intl.formatMessage({ id: 'perf.adhoc.descScope' })}
                  span={2}
                >
                  {intl.formatMessage(
                    {
                      id: selectedTask.technology
                        ? SCOPE_KEY[selectedTask.dimension].withTech
                        : SCOPE_KEY[selectedTask.dimension].anyTech,
                    },
                    {
                      tech: selectedTask.technology ? labelForTechnology(selectedTask.technology) : '',
                    },
                  )}
                </Descriptions.Item>
              )}
            </Descriptions>

            <AdhocResultPanel taskId={selectedTask.id} />
          </>
        )}
      </Drawer>
    </Space>
  );
}
