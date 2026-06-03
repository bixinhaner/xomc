/**
 * T-0164-P7 / G7 自定义聚合任务管理页面。
 *
 * T-0186：
 *   - 列表分两区：内置区（is_builtin=true）+ 自建区（is_builtin=false），各一张 Table。
 *   - 任务详情 Drawer 加 Tabs：运行历史（pm_adhoc_task_runs）+ 结果（AdhocResultPanel）。
 *   - 详情顶部制式渲染为只读 Tag（建后不可改、无切换控件）。
 *
 * T-0164 收尾：
 *   G7-Gap-2  结果查看用 AdhocResultPanel（G6 panel 风格，多指标多 series ECharts）
 *   G6-Gap-12 联动：PanelConfigDrawer 跳转携带 ?preset=panel&device_sns=...&metric_paths=...&granularities=...
 */

import { useEffect, useMemo, useState } from 'react';
import { useIntl, type IntlShape } from 'react-intl';
import { useNavigate, useSearchParams } from 'react-router-dom';
import {
  Button,
  Card,
  Drawer,
  Modal,
  Space,
  Table,
  Tabs,
  Tag,
  Progress,
  Descriptions,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { PlusOutlined, DeleteOutlined, EditOutlined } from '@ant-design/icons';
import {
  usePmAdhocList,
  usePmAdhocRuns,
  useCancelPmAdhoc,
} from '@core/hooks/api/usePmAdhoc';
import type {
  AdhocMode,
  AdhocStatus,
  AdhocTask,
  AdhocTaskRun,
} from '@core/types/pmAdhoc';
import { CreateAdhocTaskDrawer, type CreateAdhocPreset } from './CreateAdhocTaskDrawer';
import { AdhocResultPanel } from './AdhocResultPanel';
import BuiltinMetricEditModal from './BuiltinMetricEditModal';
import SelectedMetricsTags from './SelectedMetricsTags';

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

const TECHNOLOGY_LABEL_KEY: Record<string, string> = {
  lte: 'perf.adhoc.techLte',
  nr: 'perf.adhoc.techNr',
  gsm: 'perf.adhoc.techGsm',
};

function statusLabel(intl: IntlShape, s?: string): string {
  const id = s ? STATUS_LABEL_KEY[s] : undefined;
  return id ? intl.formatMessage({ id }) : (s ?? '');
}

function dimensionLabel(intl: IntlShape, d?: string): string {
  const id = d ? DIMENSION_LABEL_KEY[d] : undefined;
  return id ? intl.formatMessage({ id }) : (d ?? '');
}

function technologyLabel(intl: IntlShape, t?: string): string {
  const id = t ? TECHNOLOGY_LABEL_KEY[t] : undefined;
  return id ? intl.formatMessage({ id }) : (t ?? '');
}

function fmtTime(v?: string): string {
  if (!v) return '—';
  const d = new Date(v);
  return Number.isNaN(d.getTime()) ? v : d.toLocaleString();
}

/** 运行历史表（任务详情 Tab）。 */
function RunHistoryTab({ taskId }: { taskId: string }) {
  const intl = useIntl();
  // 运行中任务会持续产生 run，轮询刷新
  const { data: runs = [], isLoading } = usePmAdhocRuns(taskId, { refetchInterval: 10000 });

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
      { title: intl.formatMessage({ id: 'perf.adhoc.colRowsTotal' }), dataIndex: 'rowsTotal', width: 90 },
      {
        title: intl.formatMessage({ id: 'perf.adhoc.colError' }),
        dataIndex: 'error',
        ellipsis: true,
        render: (e: string) => (e ? <Typography.Text type="danger">{e}</Typography.Text> : '—'),
      },
    ],
    [intl],
  );

  return (
    <Table<AdhocTaskRun>
      rowKey="id"
      size="small"
      loading={isLoading}
      dataSource={runs}
      columns={columns}
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
  onView,
  onCancel,
  onEdit,
}: {
  tasks: AdhocTask[];
  loading: boolean;
  // 内置区 = true：操作列给「编辑指标」（开轻量弹窗）；自建区 = false：给「编辑」（进向导编辑页）。
  isBuiltinArea: boolean;
  onView: (t: AdhocTask) => void;
  onCancel: (id: string) => void;
  onEdit: (t: AdhocTask) => void;
}) {
  const intl = useIntl();
  const columns: ColumnsType<AdhocTask> = useMemo(
    () => [
      { title: intl.formatMessage({ id: 'perf.adhoc.colName' }), dataIndex: 'name' },
      {
        title: intl.formatMessage({ id: 'perf.adhoc.colMode' }),
        dataIndex: 'mode',
        width: 90,
        render: (m: AdhocMode) =>
          m === 'continuous' ? (
            <Tag color="purple">{intl.formatMessage({ id: 'perf.adhoc.modeContinuous' })}</Tag>
          ) : (
            <Tag>{intl.formatMessage({ id: 'perf.adhoc.modeOneshot' })}</Tag>
          ),
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
        render: (p: number, r) =>
          r.status === 'running' || r.status === 'succeeded' ? (
            <Progress percent={p} size="small" />
          ) : (
            '—'
          ),
      },
      // 创建时间列仅自建区保留；内置区去掉（内置任务创建时间无意义）。设备数列两区都去。
      ...(isBuiltinArea
        ? []
        : [
            {
              title: intl.formatMessage({ id: 'perf.adhoc.colCreatedAt' }),
              dataIndex: 'createdAt',
              width: 180,
              render: (v: string) => fmtTime(v),
            } as ColumnsType<AdhocTask>[number],
          ]),
      {
        title: intl.formatMessage({ id: 'perf.adhoc.colOperation' }),
        width: 230,
        render: (_, r) => (
          <Space>
            <Button size="small" onClick={() => onView(r)}>
              {intl.formatMessage({ id: 'perf.adhoc.btnViewDetail' })}
            </Button>
            {/* T-0194：内置区给「编辑指标」（只改指标集），自建区给「编辑」（进向导编辑页） */}
            <Button size="small" icon={<EditOutlined />} onClick={() => onEdit(r)}>
              {isBuiltinArea
                ? intl.formatMessage({ id: 'perf.adhoc.btnEditMetrics' })
                : intl.formatMessage({ id: 'perf.adhoc.btnEdit' })}
            </Button>
            {(r.status === 'pending' || r.status === 'running' || r.status === 'scheduled') && (
              <Button size="small" danger icon={<DeleteOutlined />} onClick={() => onCancel(r.id)}>
                {intl.formatMessage({ id: 'perf.adhoc.btnCancel' })}
              </Button>
            )}
          </Space>
        ),
      },
    ],
    [intl, isBuiltinArea, onView, onCancel, onEdit],
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
  // T-0186：分两区，各发一次 list（内置 / 自建）。
  const { data: builtinTasks = [], isLoading: builtinLoading } = usePmAdhocList({
    refetchInterval: 5000,
    isBuiltin: true,
  });
  const { data: customTasks = [], isLoading: customLoading } = usePmAdhocList({
    refetchInterval: 5000,
    isBuiltin: false,
  });
  const cancelMut = useCancelPmAdhoc();
  const navigate = useNavigate();

  const [searchParams, setSearchParams] = useSearchParams();
  const [createOpen, setCreateOpen] = useState(false);
  const [createPreset, setCreatePreset] = useState<CreateAdhocPreset | undefined>(undefined);
  const [selectedTask, setSelectedTask] = useState<AdhocTask | null>(null);
  // T-0194：内置任务「编辑指标」弹窗状态。
  const [builtinEditTask, setBuiltinEditTask] = useState<AdhocTask | null>(null);

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

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      <Card
        title={intl.formatMessage({ id: 'perf.adhoc.cardBuiltin' })}
        size="small"
        extra={<Tag color="blue">{intl.formatMessage({ id: 'perf.adhoc.systemPreset' })}</Tag>}
      >
        <TaskTable
          tasks={builtinTasks}
          loading={builtinLoading}
          isBuiltinArea
          onView={setSelectedTask}
          onCancel={handleCancel}
          onEdit={handleEditBuiltin}
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
          onView={setSelectedTask}
          onCancel={handleCancel}
          onEdit={handleEditCustom}
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
            ? intl.formatMessage({ id: 'perf.adhoc.detailTitle' }, { name: selectedTask.name })
            : intl.formatMessage({ id: 'perf.adhoc.detailTitleDefault' })
        }
        width={920}
        open={Boolean(selectedTask)}
        onClose={() => setSelectedTask(null)}
        destroyOnClose
      >
        {selectedTask && (
          <>
            <Descriptions size="small" column={2} bordered style={{ marginBottom: 16 }}>
              <Descriptions.Item label={intl.formatMessage({ id: 'perf.adhoc.descMode' })}>
                {selectedTask.mode === 'continuous'
                  ? intl.formatMessage({ id: 'perf.adhoc.modeContinuous' })
                  : intl.formatMessage({ id: 'perf.adhoc.modeOneshot' })}
              </Descriptions.Item>
              <Descriptions.Item label={intl.formatMessage({ id: 'perf.adhoc.descDimension' })}>
                {dimensionLabel(intl, selectedTask.dimension)}
              </Descriptions.Item>
              <Descriptions.Item label={intl.formatMessage({ id: 'perf.adhoc.descTech' })}>
                {/* T-0186：制式只读，建后不可改，无切换控件 */}
                {selectedTask.technology ? (
                  <Tag color="geekblue">{technologyLabel(intl, selectedTask.technology)}</Tag>
                ) : (
                  <Tag>{intl.formatMessage({ id: 'perf.adhoc.anyTech' })}</Tag>
                )}
              </Descriptions.Item>
              <Descriptions.Item label={intl.formatMessage({ id: 'perf.adhoc.descStatus' })}>
                <Tag color={statusColor[selectedTask.status]}>
                  {statusLabel(intl, selectedTask.status)}
                </Tag>
              </Descriptions.Item>
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
            </Descriptions>

            <Tabs
              defaultActiveKey="runs"
              items={[
                {
                  key: 'runs',
                  label: intl.formatMessage({ id: 'perf.adhoc.tabRuns' }),
                  children: <RunHistoryTab taskId={selectedTask.id} />,
                },
                {
                  key: 'results',
                  label: intl.formatMessage({ id: 'perf.adhoc.tabResults' }),
                  children: <AdhocResultPanel taskId={selectedTask.id} />,
                },
              ]}
            />
          </>
        )}
      </Drawer>

      {/* T-0194：内置任务「编辑指标」弹窗 */}
      <BuiltinMetricEditModal
        task={builtinEditTask}
        open={Boolean(builtinEditTask)}
        onClose={() => setBuiltinEditTask(null)}
      />
    </Space>
  );
}
