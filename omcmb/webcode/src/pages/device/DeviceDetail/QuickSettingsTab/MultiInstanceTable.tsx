import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Button, Card, Input, Modal, Popconfirm, Select, Space, Table, Tag, Tooltip, Typography, message, notification } from 'antd';
import { CheckCircleOutlined, ClockCircleOutlined, CloseCircleOutlined, DeleteOutlined, EditOutlined, PlusOutlined, SendOutlined, SyncOutlined } from '@ant-design/icons';
import type { ColumnType } from 'antd/es/table';
import { useParams } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import {
  useAddObject,
  useDeleteObject,
  useParameterSchema,
  useUpdateParameters,
} from '@core/hooks/api/useDeviceParameters';
import { useDeviceTaskStatus } from '@core/hooks/api/useDeviceTask';
import { notificationKeys } from '@core/hooks/api/useNotificationCenter';
import { deviceTaskApi } from '@core/services/api/deviceTaskApi';
import { configSyncApi } from '@core/services/api/configSyncApi';
import {
  feedbackKey,
  useQuickSettingsFeedbackStore,
  type MultiFeedback,
} from '@core/store/quickSettingsFeedbackStore';
import type { ParameterSchemaItem, ParameterUpdateRequest } from '@core/types/deviceParameter';
import { isDeviceTaskTerminal, type DeviceTaskStatus } from '@core/types/deviceTask';
import type { QuickSettingsGroup } from '@core/types/quicksettings';
import {
  applyInstanceContext,
  formatEnumDisplayValue,
  getEffectiveEnumMeta,
  validateValue,
  type QuickSettingsInstanceContext,
} from './validators';

import { useT } from '@/hooks/useT';

const { Text } = Typography;
const ERROR_FEEDBACK_DURATION_SECONDS = 2;

type TFn = (id: string, values?: Record<string, string | number>) => string;

// "上次操作"状态形状由 frontend-core/store/quickSettingsFeedbackStore (MultiFeedback) 定义,
// 提升至 store 持久化,顶层 TabBar 切走再切回不丢反馈。

// 把后端任务 ErrorMessage(`[Client] Invalid arguments — path faults: [<path>: <code> <path>:  <reason>] [...]`)
// 提取为简短设备原因列表(如 `Value must be even; Invalid arfcn value`),用于 Tag 内联展示。
// 解析失败时退化为整段截断。完整原文仍通过 Tooltip 提供。
function formatDeviceFaultBrief(msg: string | undefined | null): string {
  if (!msg) return '';
  const reasons: string[] = [];
  const re = /9\d{3}\s+[^:]+:\s+([^\]]+?)\]/g;
  let match: RegExpExecArray | null;
  while ((match = re.exec(msg)) !== null) {
    reasons.push(match[1].trim());
  }
  const joined = reasons.length > 0 ? reasons.join('; ') : msg;
  return joined.length > 80 ? `${joined.slice(0, 77)}...` : joined;
}

function formatTime(at: number): string {
  const d = new Date(at);
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

/** T-0146:状态机 Tag 显示规则(与 CellParameterForm 一致)。 */
interface StatusTagSpec {
  color: string;
  icon: React.ReactNode;
  label: string;
}
function statusTagSpec(action: MultiFeedback, taskStatus: DeviceTaskStatus | undefined, t: TFn): StatusTagSpec {
  const actionLabel = action.action === 'save'
    ? t('device.multi.actionSave')
    : action.action === 'add'
      ? t('device.multi.actionAdd')
      : t('device.multi.actionDelete');
  if (action.submitStatus === 'failed_to_queue') {
    return { color: 'error', icon: <CloseCircleOutlined />, label: t('device.multi.tagQueueFailed', { action: actionLabel }) };
  }
  // AddObject / DeleteObject currently return no task_id; only Save goes through the full state machine
  if (!action.taskId) {
    return { color: 'processing', icon: <SyncOutlined spin />, label: t('device.multi.tagQueued', { action: actionLabel }) };
  }
  switch (taskStatus) {
    case 'completed':
      return { color: 'success', icon: <CheckCircleOutlined />, label: t('device.multi.tagSuccess', { action: actionLabel }) };
    case 'failed':
      return { color: 'error', icon: <CloseCircleOutlined />, label: t('device.multi.tagNackFailed', { action: actionLabel }) };
    case 'expired':
      return { color: 'warning', icon: <ClockCircleOutlined />, label: t('device.multi.tagTimeout', { action: actionLabel }) };
    case 'cancelled':
      return { color: 'default', icon: <CloseCircleOutlined />, label: t('device.multi.tagCancelled', { action: actionLabel }) };
    case 'sent':
      return { color: 'processing', icon: <SendOutlined />, label: t('device.multi.tagSent', { action: actionLabel }) };
    case 'pending':
    default:
      return { color: 'processing', icon: <SyncOutlined spin />, label: t('device.multi.tagPending', { action: actionLabel }) };
  }
}

interface MultiInstanceTableProps {
  deviceId: string;
  group: QuickSettingsGroup;
  instanceContext: QuickSettingsInstanceContext;
  locale: 'zh-CN' | 'en-US';
}

interface RowEditState {
  /** 行内字段当前编辑值（leaf → value）。空 = 未编辑（取 schema 当前值）。 */
  edits: Record<string, string>;
  /** 行内字段错误（leaf → message）。 */
  errors: Record<string, string>;
}

interface TableRow {
  key: string;
  instanceId?: string;
}

interface EditModalState {
  mode: 'add' | 'edit';
  instanceId?: string;
  values: Record<string, string>;
  errors: Record<string, string>;
}

interface SpecialColumnSpec {
  key: string;
  leaf?: string;
  /** i18n message id；优先于 titleZh/titleEn（用于消除硬编码中文标题）。 */
  titleKey?: string;
  titleZh?: string;
  titleEn: string;
  width?: number;
  readOnly?: boolean;
  getValue?: (row: TableRow, ctx: QuickSettingsInstanceContext) => string;
  formatValue?: (value: string) => string;
}

const BM_SPECIAL_COLUMNS: Record<string, SpecialColumnSpec[]> = {
  'enb-neighbor-freq': [
    { key: 'EUTRACarrierARFCN', leaf: 'EUTRACarrierARFCN', titleKey: 'device.multi.col.frequency', titleEn: 'Frequency', width: 180, formatValue: formatEarfcnDisplay },
    { key: 'QOffsetFreq', leaf: 'QOffsetFreq', titleEn: 'Q-OffsetRange', width: 140 },
    { key: 'QRxLevMinSIB5', leaf: 'QRxLevMinSIB5', titleEn: 'Q-RxLevMin', width: 130 },
    { key: 'CellReselectionPriority', leaf: 'CellReselectionPriority', titleKey: 'device.multi.col.reselPriority', titleEn: 'Reselection Priority', width: 130 },
    { key: 'ThreshXHigh', leaf: 'ThreshXHigh', titleKey: 'device.multi.col.reselThreshHigh', titleEn: 'Reselection Thresh High', width: 130 },
    { key: 'ThreshXLow', leaf: 'ThreshXLow', titleKey: 'device.multi.col.reselThreshLow', titleEn: 'Reselection Thresh Low', width: 130 },
    { key: 'PMax', leaf: 'PMax', titleKey: 'device.multi.col.ueMaxTxPower', titleEn: 'UE Max Tx Power', width: 150 },
    { key: 'TReselectionEUTRA', leaf: 'TReselectionEUTRA', titleKey: 'device.multi.col.reselTimer', titleEn: 'TReselectionEUTRA', width: 130 },
  ],
  'enb-neighbor-cell': [
    { key: 'cellIndex', titleEn: 'cellIndex', width: 110, readOnly: true, getValue: (_row, ctx) => `Cell ${ctx.fapInstance}` },
    { key: 'EUTRACarrierARFCN', leaf: 'EUTRACarrierARFCN', titleKey: 'device.multi.col.frequency', titleEn: 'Frequency', width: 180, formatValue: formatEarfcnDisplay },
    { key: 'PhyCellID', leaf: 'PhyCellID', titleEn: 'PCI', width: 100 },
    { key: 'QOffset', leaf: 'QOffset', titleEn: 'QOffset', width: 110 },
    { key: 'CIO', leaf: 'CIO', titleEn: 'CIO', width: 100 },
    { key: 'TAC', leaf: 'TAC', titleEn: 'TAC', width: 110, readOnly: true },
    { key: 'PLMNID', leaf: 'PLMNID', titleEn: 'PLMN', width: 140 },
    { key: 'CID', leaf: 'CID', titleEn: 'ECI', width: 130, readOnly: true },
    { key: 'EnbType', leaf: 'EnbType', titleEn: 'eNodeB Type', width: 140, readOnly: true, formatValue: formatEnbTypeDisplay },
  ],
};

/**
 * BSC 邻区打包标量映射：把 quicksettings XML 中的“多实例邻区表”映射到 BTS 父对象上的两个单标量字符串。
 * 设备实际不上报 `DeviceGSM.Bts.{i}.Neighbor{2G,4G}.{j}.<leaf>` 子对象，而是：
 *
 *   GET <listLeaf>            → 设备返回当前整张邻区表（多条空白分隔，单条字段用 `-` 分隔）
 *   SET <addLeaf> = "<one>"  → 设备追加一条邻区（整条 EARFCN-thr_hi-thr_lo-prio-qrxlv-meas）
 *   SET <delLeaf> = "<key>"  → 设备从表中删除一条匹配项（注意：Del 仅接受单字段作为 key，不是整条！）
 *
 *   2G  list/add/del : NeighborCgiAdd  / NeighborCgiAdd  / NeighborCgiDel
 *                       （读取与追加使用同一个 Add 字段）
 *                       Add 格式: MCC-MNC-LAC-CI-ARFCN-BSIC
 *                       Del key : CI（cells[3]）
 *   4G  list/add/del : Si2quaterNeighborListAdd / Si2quaterNeighborListAdd / Si2quaterNeighborListDel
 *                       Add 格式: EARFCN-thresh_hi-thresh_lo-prio-qrxlv-meas
 *                       Del key : EARFCN（cells[0]）
 *
 * 实测：Del 发整条字符串会被设备解释为 "EARFCN=整条" 而返回 9007 Invalid EARFCN value；
 * 因此 spec 提供 delKey(cells) 把行内单元抽出作为 Del 字段的 key。
 *
 * 本组件不拼接整张表完整覆盖（设备不接受多条拼接的 SET），不走 AddObject/DeleteObject。
 */
const PACKED_NEIGHBOR_TABLE_BY_GROUP_ID: Record<
  string,
  {
    parentObjectPath: string;
    listLeaf: string;
    addLeaf: string;
    delLeaf: string;
    /** 从已解析的行 cells 中提取“删除”SPV 所需的单字段 key（设备 Del 字段只接受单 key，不是整条）。 */
    delKey: (cells: string[]) => string;
  }
> = {
  'bsc-bts-neighbor2g': {
    parentObjectPath: 'DeviceGSM.Bts.{i}.',
    listLeaf: 'NeighborCgiAdd',
    addLeaf: 'NeighborCgiAdd',
    delLeaf: 'NeighborCgiDel',
    // leaf 名 NeighborCgiDel → osmo-bsc `neighbor del cgi <mcc> <mnc> <lac> <ci>`
    // 完整 entry 是 MCC-MNC-LAC-CI-ARFCN-BSIC，del key 取前 4 段。
    delKey: (cells) =>
      `${(cells[0] ?? '').trim()}-${(cells[1] ?? '').trim()}-${(cells[2] ?? '').trim()}-${(cells[3] ?? '').trim()}`,
  },
  'bsc-bts-neighbor4g': {
    parentObjectPath: 'DeviceGSM.Bts.{i}.',
    listLeaf: 'Si2quaterNeighborListAdd',
    addLeaf: 'Si2quaterNeighborListAdd',
    delLeaf: 'Si2quaterNeighborListDel',
    // EARFCN-thr_hi-thr_lo-prio-qrxlv-meas → EARFCN
    delKey: (cells) => (cells[0] ?? '').trim(),
  },
};

function parsePackedNeighborList(packed: string): string[][] {
  if (!packed) return [];
  return packed
    .trim()
    .split(/\s+/)
    .filter((entry) => entry.length > 0)
    .map((entry) => entry.split('-'));
}

/** 单条邻区转字符串：字段用 `-` 拼接，与设备格式一致；cells 中任何字段都不能含空白/`-`。 */
function serializeNeighborEntry(cells: string[]): string {
  return cells.map((c) => c.trim()).join('-');
}

interface PackedScalarNeighborTableProps {
  deviceId: string;
  group: QuickSettingsGroup;
  instanceContext: QuickSettingsInstanceContext;
  locale: 'zh-CN' | 'en-US';
  spec: {
    parentObjectPath: string;
    listLeaf: string;
    addLeaf: string;
    delLeaf: string;
    delKey: (cells: string[]) => string;
  };
}

/**
 * 打包标量邻区表：从 BTS 父对象单标量解析出邻区列表展示，并支持新增/删除。
 *
 * 写入路径（不走 AddObject / DeleteObject —— 设备未实现）：
 *   1. 用户在 Modal 中填字段 / 点击行删除
 *   2. 前端在内存里维护 cellsList，按 `serializePackedNeighborList` 重拼成完整字符串
 *   3. 调用 `useUpdateParameters` 对 `scalarPath` 整体 SetParameterValues
 *   4. 成功后 refetch 父路径 schema，重新解析展示
 *
 * 字段约束：必填、不含 `-` 与空白（避免破坏分隔符），其他校验依赖设备侧。
 */
function PackedScalarNeighborTable({
  deviceId,
  group,
  instanceContext,
  locale,
  spec,
}: PackedScalarNeighborTableProps) {
  const t = useT();
  // 后端 /config/sync/pull/:deviceId 用设备 SN 查找（ensureDeviceExists by SN），不接受 UUID。
  // URL 形如 /device/detail/{sn}?tab=... ，从路由参数拿。
  const { sn: deviceSn = '' } = useParams<{ sn: string }>();
  // 解析 BTS 实例号（占位符为 {i}），得到父对象路径，e.g. "DeviceGSM.Bts.1."
  const parentPath = useMemo(() => {
    const resolved = applyInstanceContext(spec.parentObjectPath, instanceContext, {
      preserveTrailingInstance: true,
    });
    // 末尾仍可能残留 {i}.；把它替换成 fapInstance(BTS 实例号)。
    return resolved.replace(/\{i\}\.$/, `${instanceContext.fapInstance}.`);
  }, [spec.parentObjectPath, instanceContext]);

  const scalarPath = `${parentPath}${spec.listLeaf}`;
  const addPath = `${parentPath}${spec.addLeaf}`;
  const delPath = `${parentPath}${spec.delLeaf}`;

  // 拉父路径下的所有参数，从中找出打包标量。父路径粒度命中只读 schema 已足够。
  const { data: schemaResp, isLoading, refetch, isFetching } = useParameterSchema(deviceId, parentPath);
  const updateMutation = useUpdateParameters();

  const packedValue = useMemo(() => {
    const item = schemaResp?.parameters.find((p) => p.path === scalarPath);
    return item?.currentValue ?? '';
  }, [schemaResp, scalarPath]);

  const lastSyncedAt = useMemo(() => {
    const item = schemaResp?.parameters.find((p) => p.path === scalarPath);
    return item?.lastSyncedAt ?? null;
  }, [schemaResp, scalarPath]);

  const cellsList = useMemo(() => parsePackedNeighborList(packedValue), [packedValue]);
  const rows = useMemo(
    () => cellsList.map((cells, idx) => ({ key: idx + 1, id: idx + 1, cells })),
    [cellsList],
  );

  // 列定义中字段顺序严格跟随 group.params（来自 quicksettings XML），对应打包条目内 `-` 分隔字段的下标。
  const fieldLeaves = useMemo(
    () => group.params.map((param) => param.leaf || param.name),
    [group.params],
  );

  // 新增弹窗状态：null = 关闭；values 以 leaf 为 key。
  const [addModal, setAddModal] = useState<{
    values: Record<string, string>;
    errors: Record<string, string>;
  } | null>(null);

  const maxInstances = group.maxInstances && group.maxInstances > 0 ? group.maxInstances : undefined;
  const reachedMax = maxInstances !== undefined && rows.length >= maxInstances;
  const canMutate = fieldLeaves.length > 0; // 没有字段定义就退回纯只读视图（兜底）

  // 与多实例表一致：ref 同步防重点，state 驱动按钮 loading。
  // 覆盖整个 writeSingleEntry 生命周期（mutation + task 轮询 + pullConfig + refetch）。
  const submittingRef = useRef(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const queryClient = useQueryClient();
  const fbKey = feedbackKey(
    deviceId,
    group.id,
    instanceContext.fapInstance,
    instanceContext.networkType === 'nr' ? instanceContext.cellInstance : undefined,
  );
  const lastAction = useQuickSettingsFeedbackStore((s) => {
    const f = s.entries[fbKey];
    return f && f.kind === 'multi' ? f : null;
  });
  const setFeedback = useQuickSettingsFeedbackStore((s) => s.setFeedback);
  const patchFeedback = useQuickSettingsFeedbackStore((s) => s.patchFeedback);
  const { data: lastTask } = useDeviceTaskStatus(lastAction?.taskId);

  const waitForTaskTerminal = useCallback(async (taskId: string) => {
    const timeoutAt = Date.now() + 60000;
    while (Date.now() < timeoutAt) {
      const task = await deviceTaskApi.getTask(taskId);
      if (isDeviceTaskTerminal(task.status)) return task;
      await new Promise((resolve) => window.setTimeout(resolve, 1000));
    }
    throw new Error(t('device.multi.waitTaskTimeout'));
  }, [t]);

  // 任务终态为 failed 时弹一次通知（与外部 MultiInstanceTable 一致的田崯避免重复玄象）。
  useEffect(() => {
    if (
      lastTask &&
      lastTask.status === 'failed' &&
      lastAction &&
      lastAction.notifiedFailedTaskId !== lastTask.id
    ) {
      const actionLabel = lastAction.action === 'add'
        ? t('device.multi.actionAdd')
        : t('device.multi.actionDelete');
      notification.error({
        message: t('device.multi.tagNackFailed', { action: actionLabel }),
        description: lastTask.errorMessage || t('device.multi.unknownErrorHint'),
        duration: ERROR_FEEDBACK_DURATION_SECONDS,
      });
      patchFeedback(fbKey, { notifiedFailedTaskId: lastTask.id });
    }
  }, [lastTask, lastAction, patchFeedback, fbKey, t]);

  /**
   * 调用设备侧“单条 Add”或“单条 Del”。设备不接受多条拼接覆盖；Add 在父 list 上 append，
   * Del 从父 list 中移除匹配项。注意 Add 发送整条字段串，Del 仅发送 spec.delKey 抽出的单字段 key
   * （EARFCN 或 CI），实测发整条 Del 会被设备误解析为 "EARFCN=整条" 返回 9007。
   * 写完后 refetch 拉回设备侧最新完整表。
   *
   * 后端会在 SPV 完成后自动 GPV 同步请求 path 的 leaf，但 UI 读的是 listLeaf=Add。
   * Del 操作时要额外 pullConfig(listLeaf) 让 device_parameters 中 Add 行刷新。
   */
  const writeSingleEntry = useCallback(
    async (opType: 'add' | 'del', cells: string[], opSuccessMsg: string) => {
      if (submittingRef.current) return; // 同 tick 双击 / 任务轮询期重点击 都被驳回
      submittingRef.current = true;
      setIsSubmitting(true);
      const value = opType === 'add' ? serializeNeighborEntry(cells) : spec.delKey(cells);
      const parameterPath = opType === 'add' ? addPath : delPath;
      const actionLabel = opType === 'add' ? t('device.multi.actionAdd') : t('device.multi.actionDelete');
      try {
        const result = await updateMutation.mutateAsync({
          deviceId,
          parameters: [{ parameterPath, parameterValue: value, parameterType: 'string' }],
        });
        setFeedback(fbKey, {
          kind: 'multi',
          action: opType === 'add' ? 'add' : 'delete',
          submitStatus: 'queued',
          taskId: result.taskId,
          detail: opSuccessMsg,
          at: Date.now(),
        });
        message.success(opSuccessMsg);
        // 等设备侧任务终态，防止按钮提前释放后用户双击造成重复 Add/Del。
        if (result.taskId) {
          try {
            await waitForTaskTerminal(result.taskId);
          } catch {
            // 超时不阻断：Tag 后续仍会随 useDeviceTaskStatus 轮询更新。
          }
        }
        // Del 后额外拉一次 listLeaf，同步最新设备状态到 DB。Add 不需要（listLeaf===addLeaf）。
        // 注意：configSyncApi.pullConfig 后端按 SN 预检，不接受 UUID。
        if (opType === 'del' && spec.listLeaf !== spec.delLeaf && deviceSn) {
          try {
            await configSyncApi.pullConfig(deviceSn, [scalarPath]);
            // GPV 是异步任务，给 ACS+设备 一点时间完成后再 refetch，避免 device_parameters 滞后导致 UI 仍显示已删行。
            await new Promise((resolve) => setTimeout(resolve, 1500));
          } catch (pullErr) {
            // 同步失败不阻断主流程；UI 表头计数可能滞后，下次手动刷新会拼正。
            // eslint-disable-next-line no-console
            console.warn('[PackedScalarNeighborTable] pullConfig listLeaf failed', pullErr);
          }
        }
        await refetch();
      } catch (err) {
        const detail = err instanceof Error ? err.message : String(err);
        setFeedback(fbKey, {
          kind: 'multi',
          action: opType === 'add' ? 'add' : 'delete',
          submitStatus: 'failed_to_queue',
          detail: t('device.multi.detailFailed', { target: actionLabel, err: detail }),
          at: Date.now(),
        });
        notification.error({
          message: t('device.multi.tagQueueFailed', { action: actionLabel }),
          description: detail,
          duration: ERROR_FEEDBACK_DURATION_SECONDS,
        });
      } finally {
        void queryClient.invalidateQueries({ queryKey: notificationKeys.all });
        submittingRef.current = false;
        setIsSubmitting(false);
      }
    },
    [addPath, delPath, deviceId, deviceSn, fbKey, queryClient, refetch, scalarPath, setFeedback, spec, t, updateMutation, waitForTaskTerminal],
  );

  const openAddModal = useCallback(() => {
    if (!canMutate || reachedMax) return;
    setAddModal({
      values: Object.fromEntries(fieldLeaves.map((leaf) => [leaf, ''])),
      errors: {},
    });
  }, [canMutate, fieldLeaves, reachedMax]);

  const closeAddModal = useCallback(() => setAddModal(null), []);

  const setAddModalValue = useCallback((leaf: string, value: string) => {
    setAddModal((prev) => {
      if (!prev) return prev;
      const nextErrors = { ...prev.errors };
      delete nextErrors[leaf];
      return { values: { ...prev.values, [leaf]: value }, errors: nextErrors };
    });
  }, []);

  const handleAddSubmit = useCallback(async () => {
    if (!addModal) return;
    const errors: Record<string, string> = {};
    const cells: string[] = [];
    for (const leaf of fieldLeaves) {
      const raw = (addModal.values[leaf] ?? '').trim();
      if (!raw) {
        errors[leaf] = t('device.multi.packed.fieldRequired');
        cells.push('');
        continue;
      }
      if (/[\s-]/.test(raw)) {
        // `-` 是字段分隔符，空白是条目分隔符，两者都不能出现在字段值里。
        errors[leaf] = t('device.multi.packed.fieldInvalidChar');
        cells.push(raw);
        continue;
      }
      cells.push(raw);
    }
    if (Object.keys(errors).length > 0) {
      setAddModal((prev) => (prev ? { ...prev, errors } : prev));
      return;
    }
    const nextEntryNumber = cellsList.length + 1;
    setAddModal(null);
    await writeSingleEntry(
      'add',
      cells,
      t('device.multi.detailAdd', { instId: String(nextEntryNumber), count: cells.length }),
    );
  }, [addModal, cellsList.length, fieldLeaves, t, writeSingleEntry]);

  const handleDeleteRow = useCallback(
    async (rowIdx: number) => {
      if (!canMutate) return;
      const target = cellsList[rowIdx];
      if (!target) return;
      await writeSingleEntry(
        'del',
        target,
        t('device.multi.deleteDispatched', { instId: String(rowIdx + 1) }),
      );
    },
    [canMutate, cellsList, t, writeSingleEntry],
  );

  const columns: ColumnType<{ key: number; id: number; cells: string[] }>[] = [
    {
      title: t('device.multi.instance'),
      dataIndex: 'id',
      key: 'id',
      width: 80,
      fixed: 'left',
      render: (_v: unknown, row) => <Text strong>{row.id}</Text>,
    },
    ...group.params.map<ColumnType<{ key: number; id: number; cells: string[] }>>((param, colIdx) => ({
      title: locale === 'zh-CN' ? param.titleZh : param.titleEn,
      key: param.leaf || param.name,
      width: 140,
      render: (_v: unknown, row) => <Text>{row.cells[colIdx] ?? '-'}</Text>,
    })),
  ];
  if (canMutate) {
    columns.push({
      title: t('device.multi.packed.colActions'),
      key: '__op',
      width: 90,
      fixed: 'right',
      render: (_v: unknown, row) => (
        <Popconfirm
          title={t('device.multi.deleteConfirm')}
          description={t('device.multi.detailInstance', { instId: String(row.id) })}
          okButtonProps={{ danger: true, loading: updateMutation.isPending || isSubmitting }}
          onConfirm={() => void handleDeleteRow(row.id - 1)}
        >
          <Button size="small" type="link" danger icon={<DeleteOutlined />} disabled={isSubmitting}>
            {t('device.multi.actionDelete')}
          </Button>
        </Popconfirm>
      ),
    });
  }

  const title = locale === 'zh-CN' ? group.titleZh : group.titleEn;
  const cardTitle = maxInstances !== undefined
    ? `${title}（${rows.length}/${maxInstances}）`
    : `${title}（${rows.length}）`;

  return (
    <Card
      title={cardTitle}
      size="small"
      extra={
        <Space>
          {lastAction && (() => {
            const tagSpec = statusTagSpec(lastAction, lastTask?.status, t);
            const isFailed = lastTask?.status === 'failed' && Boolean(lastTask?.errorMessage);
            const briefFault = isFailed ? formatDeviceFaultBrief(lastTask?.errorMessage) : '';
            const tag = (
              <Tag icon={tagSpec.icon} color={tagSpec.color}>
                {tagSpec.label} · {lastAction.detail}
                {briefFault ? ` · ${briefFault}` : ''} · {formatTime(lastAction.at)}
              </Tag>
            );
            return isFailed ? (
              <Tooltip title={lastTask?.errorMessage} placement="bottomRight">
                {tag}
              </Tooltip>
            ) : tag;
          })()}
          {canMutate ? (
            <Tooltip title={reachedMax ? t('device.multi.reachedMaxTooltip', { max: String(maxInstances ?? '') }) : ''}>
              <Button
                size="small"
                type="primary"
                icon={<PlusOutlined />}
                disabled={reachedMax || updateMutation.isPending || isSubmitting}
                onClick={openAddModal}
              >
                {t('device.multi.actionAdd')}
              </Button>
            </Tooltip>
          ) : (
            <Tag color="default">{t('device.multi.packed.readonlyTag')}</Tag>
          )}
          <Button
            size="small"
            icon={<SyncOutlined spin={isFetching} />}
            onClick={() => void refetch()}
            loading={isFetching}
          >
            {t('common.refresh')}
          </Button>
        </Space>
      }
      style={{ marginBottom: 16 }}
    >
      <Table
        rowKey="key"
        dataSource={rows}
        columns={columns}
        loading={isLoading}
        size="small"
        pagination={false}
        scroll={{ x: 'max-content', y: 240 }}
        sticky
        locale={{ emptyText: t('device.multi.packed.emptyText') }}
      />
      <div style={{ marginTop: 8, color: '#8c8c8c', fontSize: 12 }}>
        {t('device.multi.packed.sourceLabel')}{scalarPath}
        {lastSyncedAt ? ` · ${t('device.multi.packed.lastSynced', { time: formatTime(new Date(lastSyncedAt).getTime()) })}` : ''}
      </div>

      <Modal
        title={t('device.multi.modalAddTitle', { title })}
        open={!!addModal}
        onCancel={closeAddModal}
        onOk={() => void handleAddSubmit()}
        okText={t('device.multi.confirmAdd')}
        confirmLoading={updateMutation.isPending || isSubmitting}
        destroyOnClose
        maskClosable={false}
      >
        {addModal && (
          <Space direction="vertical" size="small" style={{ width: '100%' }}>
            {group.params.map((param) => {
              const leaf = param.leaf || param.name;
              const label = locale === 'zh-CN' ? param.titleZh : param.titleEn;
              const err = addModal.errors[leaf];
              return (
                <div key={leaf}>
                  <div style={{ marginBottom: 4, fontSize: 12 }}>
                    <Text strong>{label}</Text>
                    <Text type="secondary" style={{ marginLeft: 8 }}>{leaf}</Text>
                  </div>
                  <Input
                    size="small"
                    value={addModal.values[leaf] ?? ''}
                    status={err ? 'error' : undefined}
                    onChange={(e) => setAddModalValue(leaf, e.target.value)}
                    placeholder={label}
                  />
                  {err && <Text type="danger" style={{ fontSize: 12 }}>{err}</Text>}
                </div>
              );
            })}
          </Space>
        )}
      </Modal>
    </Card>
  );
}

/**
 * 多实例分组表格（异频邻区频点列表 / 邻区列表）。
 *
 * 行为：
 *  1. objectPath 中外层 FAPService.{i} 替换后作为 pathPrefix 查 schema
 *  2. schema.objects 中 path 等于 objectPath 的项给出 currentInstances（实例号数组）
 *  3. 每个实例渲染为一行；行内字段值从 schema.parameters[path=objectPath+instance+leaf] 取
 *  4. 行级 Save：收集脏字段 → 一次 SetParameterValues
 *  5. 新增：先 AddObject 拿新实例号 → 用户填值 → 行 Save 触发 SetParameterValues（两步）
 *  6. 删除：DeleteObject
 *  7. 失败标红保留输入值，"重试"按钮原值重发
 */
export default function MultiInstanceTable({ deviceId, group, instanceContext, locale }: MultiInstanceTableProps) {
  const t = useT();
  // BSC 邻区兼容：部分 GSM 设备不按 TR-181 子对象上报，而是把整张邻区列表打包到 BTS 父对象单标量。
  // 这种 group 不存在 currentInstances，常规多实例渲染会出现「暂无数据」。改走打包标量解析路径。
  const packedSpec = PACKED_NEIGHBOR_TABLE_BY_GROUP_ID[group.id];
  if (packedSpec) {
    return (
      <PackedScalarNeighborTable
        deviceId={deviceId}
        group={group}
        instanceContext={instanceContext}
        locale={locale}
        spec={packedSpec}
      />
    );
  }
  // group.objectPath 形如 "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}."
  // - 外层 FAPService.{i} → 用 fapInstance 替换
  // - 内层 Carrier.{i}. 末段是实例号占位符 — 剥离后得到父对象路径,用于查 schema.objects / AddObject / 拼接行 path 前缀
  const objectPath = useMemo(() => {
    const resolved = applyInstanceContext(group.objectPath || '', instanceContext, {
      preserveTrailingInstance: true,
    });
    return resolved.replace(/\{i\}\.$/, '');
  }, [group, instanceContext]);
  const { data: schemaResp, isLoading, refetch } = useParameterSchema(deviceId, objectPath);
  const updateMutation = useUpdateParameters();
  const addMutation = useAddObject();
  const deleteMutation = useDeleteObject();
  const queryClient = useQueryClient();
  const specialColumns = BM_SPECIAL_COLUMNS[group.id] ?? null;
  const [editModal, setEditModal] = useState<EditModalState | null>(null);
  // 状态包住整段 handleSaveEditModal(含 AddObject mutation 后的 waitForTaskTerminal 轮询),
  // 避免用户在任务未终止时以为“没反应”重复点击导致重复 AddObject。
  // ref 同步起效(防同 tick 双击); state 为 Modal confirmLoading 提供视觉反馈。
  const submittingRef = useRef(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // 行编辑状态：以 instanceId 为 key，仅保留用户编辑过的字段（避免 effect 同步 schema 触发级联 render）
  const [rowEdits, setRowEdits] = useState<Map<string, RowEditState>>(new Map());

  // lastAction 由 zustand store 托管 —— DeviceDetail 卸载(切顶层 tab)也保留反馈。
  const fbKey = feedbackKey(
    deviceId,
    group.id,
    instanceContext.fapInstance,
    instanceContext.networkType === 'nr' ? instanceContext.cellInstance : undefined,
  );
  const lastAction = useQuickSettingsFeedbackStore((s) => {
    const f = s.entries[fbKey];
    return f && f.kind === 'multi' ? f : null;
  });
  const setFeedback = useQuickSettingsFeedbackStore((s) => s.setFeedback);
  const patchFeedback = useQuickSettingsFeedbackStore((s) => s.patchFeedback);
  const draft = useQuickSettingsFeedbackStore((s) => s.drafts[fbKey]);
  const setDraftField = useQuickSettingsFeedbackStore((s) => s.setDraftField);
  const clearDraftPrefix = useQuickSettingsFeedbackStore((s) => s.clearDraftPrefix);

  // schema.objects 给出 currentInstances；schema.parameters 给出值
  const objectEntry = useMemo(
    () => schemaResp?.objects.find((o) => o.path === objectPath),
    [schemaResp, objectPath],
  );
  // 后端 schema 目前只会为"已有实例"的对象返回 ObjectSchemaItem，空列表时 objectEntry 可能缺失。
  // 这类场景下仍允许直接走 AddObject，由后端最终校验 objectPath 是否可新增。
  const canAdd = objectEntry?.canAdd ?? Boolean(group.objectPath && objectPath);
  const canDelete = objectEntry?.canDeleteAny ?? false;

  const schemaByPath = useMemo(() => {
    const map = new Map<string, ParameterSchemaItem>();
    schemaResp?.parameters.forEach((p) => map.set(p.path, p));
    return map;
  }, [schemaResp]);

  // 实例号列表直接从 schema 派生（不再走 setState in effect）
  const instanceIds = useMemo(() => {
    if (!objectEntry) return [] as string[];
    return objectEntry.currentInstances.map((n) => String(n)).sort((a, b) => Number(a) - Number(b));
  }, [objectEntry]);

  const paramSchemaByLeaf = useMemo(() => {
    const map = new Map<string, ParameterSchemaItem>();
    for (const param of group.params) {
      const leaf = param.leaf || '';
      if (!leaf || map.has(leaf)) continue;
      const existing = instanceIds
        .map((instId) => schemaByPath.get(`${objectPath}${instId}.${leaf}`))
        .find(Boolean);
      if (existing) {
        map.set(leaf, existing);
        continue;
      }
      const fallback = schemaResp?.parameters.find(
        (item) => item.path.startsWith(objectPath) && item.path.endsWith(`.${leaf}`),
      );
      if (fallback) map.set(leaf, fallback);
    }
    return map;
  }, [group.params, instanceIds, objectPath, schemaByPath, schemaResp?.parameters]);

  const groupParamLeafSet = useMemo(() => new Set(group.params.map((param) => param.leaf).filter(Boolean)), [group.params]);

  const leafSchemaByLeaf = useMemo(() => {
    const map = new Map<string, ParameterSchemaItem>();
    const leaves = new Set<string>();
    group.params.forEach((param) => {
      if (param.leaf) leaves.add(param.leaf);
    });
    specialColumns?.forEach((column) => {
      if (column.leaf) leaves.add(column.leaf);
    });

    leaves.forEach((leaf) => {
      const existing = instanceIds
        .map((instId) => schemaByPath.get(`${objectPath}${instId}.${leaf}`))
        .find(Boolean);
      if (existing) {
        map.set(leaf, existing);
        return;
      }
      const fallback = schemaResp?.parameters.find(
        (item) => item.path.startsWith(objectPath) && item.path.endsWith(`.${leaf}`),
      );
      if (fallback) map.set(leaf, fallback);
    });

    return map;
  }, [group.params, instanceIds, objectPath, schemaByPath, schemaResp?.parameters, specialColumns]);

  const buildInitialEditValues = useCallback((): Record<string, string> => {
    const values = Object.fromEntries(
      group.params.map((param) => {
        const leaf = param.leaf || '';
        return [leaf, paramSchemaByLeaf.get(leaf)?.defaultValue ?? ''];
      }),
    );
    return values;
  }, [group.params, paramSchemaByLeaf]);

  const tableRows = useMemo<TableRow[]>(() => {
    return instanceIds.map((instId) => ({ key: instId, instanceId: instId }));
  }, [instanceIds]);

  useEffect(() => {
    if (!draft) return;
    setRowEdits((prev) => {
      if (prev.size > 0) return prev;

      const next = new Map<string, RowEditState>();
      for (const [name, value] of Object.entries(draft)) {
        const splitIndex = name.indexOf('.');
        if (splitIndex <= 0) continue;

        const instId = name.slice(0, splitIndex);
        const leaf = name.slice(splitIndex + 1);
        const item = schemaByPath.get(`${objectPath}${instId}.${leaf}`) ?? leafSchemaByLeaf.get(leaf);
        const err = validateValue(String(value ?? ''), (item?.type as never) ?? 'string', item?.constraints);
        const current = next.get(instId) ?? { edits: {}, errors: {} };

        current.edits[leaf] = String(value ?? '');
        if (err) {
          current.errors[leaf] = err;
        }
        next.set(instId, current);
      }
      return next;
    });
  }, [draft, objectPath, schemaByPath, leafSchemaByLeaf]);

  const cellValue = useCallback(
    (instId: string, leaf: string): string => {
      const edit = rowEdits.get(instId);
      if (edit && leaf in edit.edits) return edit.edits[leaf];
      const path = `${objectPath}${instId}.${leaf}`;
      return schemaByPath.get(path)?.currentValue ?? '';
    },
    [rowEdits, objectPath, schemaByPath],
  );

  const waitForTaskTerminal = useCallback(async (taskId: string) => {
    const timeoutAt = Date.now() + 60000;
    while (Date.now() < timeoutAt) {
      const task = await deviceTaskApi.getTask(taskId);
      if (isDeviceTaskTerminal(task.status)) return task;
      await new Promise((resolve) => window.setTimeout(resolve, 1000));
    }
    throw new Error(t('device.multi.waitTaskTimeout'));
  }, [t]);

  // T-0146:Save 后用 task_id 轮询真实 CPE 应答状态;到终态后停轮询。
  // AddObject / DeleteObject 暂不走 taskId(后端 useAddObject/useDeleteObject 未返 task),
  // Tag 只显示"入队成功/失败"语义。
  const { data: lastTask } = useDeviceTaskStatus(lastAction?.taskId);

  // 任务进入任一终态后再次刷新 schema:
  //  - DeleteObject:摘掉已删实例(原始用途)
  //  - SPV save:completed 时拿到新值；failed/expired/cancelled 时回到设备侧真实值，
  //    同步清掉该行 rowEdits + draft，避免页面继续显示乐观输入。
  useEffect(() => {
    if (!lastTask || !isDeviceTaskTerminal(lastTask.status)) return;
    let cancelled = false;
    void (async () => {
      try {
        await refetch();
      } catch (err) {
        if (!cancelled) {
          const errMsg = err instanceof Error ? err.message : String(err);
          notification.error({
            message: t('device.multi.readbackFailed', { group: group.titleZh }),
            description: errMsg,
            duration: ERROR_FEEDBACK_DURATION_SECONDS,
          });
        }
        return;
      }
      if (cancelled) return;
      const ids = lastAction?.action === 'save'
        ? lastAction.savedInstIds ?? (lastAction.savedInstId ? [lastAction.savedInstId] : [])
        : [];
      if (ids.length > 0) {
        setRowEdits((prev) => {
          const next = new Map(prev);
          for (const id of ids) next.delete(id);
          return next;
        });
        for (const id of ids) clearDraftPrefix(fbKey, `${id}.`);
      }
    })();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [lastTask?.id, lastTask?.status]);

  // T-0146:基站应答失败时弹一次 notification(仅在 status 第一次变成 failed 时触发)
  // notifiedFailedTaskId 同样存 store —— 切顶层 tab 再切回不会重复弹。
  useEffect(() => {
    if (
      lastTask &&
      lastTask.status === 'failed' &&
      lastAction &&
      lastAction.notifiedFailedTaskId !== lastTask.id
    ) {
      notification.error({
        message: t('device.multi.nackFailed', { group: group.titleZh }),
        description: lastTask.errorMessage || t('device.multi.unknownErrorHint'),
        duration: ERROR_FEEDBACK_DURATION_SECONDS,
      });
      patchFeedback(fbKey, { notifiedFailedTaskId: lastTask.id });
    }
  }, [lastTask, lastAction, group.titleZh, patchFeedback, fbKey, t]);

  // 单层确认：外层 Popconfirm 已二次确认，这里直接执行删除逻辑（原 Modal.confirm 套层移除）。
  const handleDelete = async (instId: string) => {
    try {
      // T-0157 C7: 后端现返回 { taskId } → 消费 taskId 让 Tag 走完整状态机
      const result = await deleteMutation.mutateAsync({ deviceId, objectPath: `${objectPath}${instId}.` });
      message.success({
        content: t('device.multi.deleteDispatched', { instId }),
        duration: 6,
      });
      setFeedback(fbKey, {
        kind: 'multi',
        action: 'delete',
        submitStatus: 'queued',
        taskId: result.taskId,
        detail: t('device.multi.detailInstance', { instId }),
        at: Date.now(),
      });
      // 实例已删 → 清该行可能残留的 draft + rowEdits（避免下次重挂载尝试恢复已不存在的实例）
      clearDraftPrefix(fbKey, `${instId}.`);
      setRowEdits((prev) => {
        const next = new Map(prev);
        next.delete(instId);
        return next;
      });
      void refetch();
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : String(err);
      notification.error({
        message: t('device.multi.deleteQueueFailed', { group: group.titleZh }),
        description: t('device.multi.deleteFailedDesc', { instId, err: errMsg }),
        duration: ERROR_FEEDBACK_DURATION_SECONDS,
      });
      setFeedback(fbKey, {
        kind: 'multi',
        action: 'delete',
        submitStatus: 'failed_to_queue',
        detail: t('device.multi.detailInstanceErr', { instId, err: errMsg }),
        at: Date.now(),
      });
      console.error('MultiInstanceTable: DeleteObject failed', err);
    } finally {
      void queryClient.invalidateQueries({ queryKey: notificationKeys.all });
    }
  };

  const displayColumns = useMemo<SpecialColumnSpec[]>(() => {
    if (!specialColumns) {
      return group.params.map((param) => ({
        key: param.leaf || param.name,
        leaf: param.leaf || '',
        titleZh: param.titleZh,
        titleEn: param.titleEn,
      }));
    }
    return specialColumns;
  }, [group.params, specialColumns]);

  const openEditModal = useCallback((row: TableRow) => {
    if (!row.instanceId) return;
    const values: Record<string, string> = {};
    displayColumns.forEach((column) => {
      if (!column.leaf) return;
      values[column.leaf] = cellValue(row.instanceId!, column.leaf);
    });
    setEditModal({ mode: 'edit', instanceId: row.instanceId, values, errors: {} });
  }, [cellValue, displayColumns]);

  const openAddModal = useCallback(() => {
    setEditModal({ mode: 'add', values: buildInitialEditValues(), errors: {} });
  }, [buildInitialEditValues]);

  const setEditModalValue = useCallback((leaf: string, value: string) => {
    setEditModal((prev) => {
      if (!prev) return prev;
      const item = prev.instanceId
        ? (schemaByPath.get(`${objectPath}${prev.instanceId}.${leaf}`) ?? leafSchemaByLeaf.get(leaf))
        : leafSchemaByLeaf.get(leaf);
      const err = validateValue(value, (item?.type as never) ?? 'string', item?.constraints) ?? '';
      return {
        ...prev,
        values: { ...prev.values, [leaf]: value },
        errors: { ...prev.errors, [leaf]: err },
      };
    });
  }, [leafSchemaByLeaf, objectPath, schemaByPath]);

  const closeEditModal = useCallback(() => {
    if (updateMutation.isPending || isSubmitting) return;
    setEditModal(null);
  }, [updateMutation.isPending, isSubmitting]);

  const handleSaveEditModal = useCallback(async () => {
    if (!editModal) return;
    if (submittingRef.current) return; // 同步幂等守卫: 同 tick 双击 以及 task 轮询窗口内重点
    submittingRef.current = true;
    setIsSubmitting(true);
    try {

    const errors: Record<string, string> = {};
    const updates: ParameterUpdateRequest[] = [];
    const pendingEdits: Record<string, string> = {};
    let targetInstanceId = editModal.instanceId;

    if (editModal.mode === 'add') {
      try {
        const addResult = await addMutation.mutateAsync({ deviceId, objectPath });
        const addTask = await waitForTaskTerminal(addResult.taskId);
        if (addTask.status !== 'completed') {
          throw new Error(addTask.errorMessage || t('device.multi.addInstanceFailed', { status: addTask.status }));
        }

        // 优先从 AddObject task.result.instance_number 取新实例号（后端 acs/handler.go::handleAddObjectResponse
        // 解析 SOAP AddObjectResponse 写入)。该字段最权威,不受 schema endpoint 同步延迟影响。
        const instNumberRaw = addTask.result?.instance_number;
        if (typeof instNumberRaw === 'number' && instNumberRaw > 0) {
          targetInstanceId = String(instNumberRaw);
        }

        // 后备：schema diff 兜底（旧路径,与 BS 后端 GPV 写库存在 race;仅在 result 缺失时使用）。
        if (!targetInstanceId) {
          const refreshed = await refetch();
          const nextObject = refreshed.data?.objects.find((o) => o.path === objectPath);
          const knownInstances = new Set(instanceIds);
          targetInstanceId = nextObject?.currentInstances.map((n) => String(n)).find((instId) => !knownInstances.has(instId));
        } else {
          // 拿到 instance_number 后仍主动 refetch 一次,让 schemaByPath 有该实例的默认值供后续 SPV 对比；
          // 但不阻塞:即使 refetch 还没看到新实例,SPV 也能按用户填值直接发。
          await refetch();
        }

        if (!targetInstanceId) {
          throw new Error(t('device.multi.addInstanceNoId'));
        }
      } catch (err) {
        const errMsg = err instanceof Error ? err.message : String(err);
        notification.error({
          message: t('device.multi.addFailed', { group: group.titleZh }),
          description: errMsg,
          duration: ERROR_FEEDBACK_DURATION_SECONDS,
        });
        setFeedback(fbKey, {
          kind: 'multi',
          action: 'add',
          submitStatus: 'failed_to_queue',
          detail: errMsg,
          at: Date.now(),
        });
        return;
      }
    }

    if (!targetInstanceId) return;

    for (const column of displayColumns) {
      const leaf = column.leaf || '';
      if (!leaf || !groupParamLeafSet.has(leaf) || column.readOnly) continue;

      const value = editModal.values[leaf] ?? '';
      const path = `${objectPath}${targetInstanceId}.${leaf}`;
      const item = schemaByPath.get(path) ?? leafSchemaByLeaf.get(leaf);
      const err = validateValue(value, (item?.type as never) ?? 'string', item?.constraints);
      if (err) {
        errors[leaf] = err;
        continue;
      }

      const oldVal = item?.currentValue ?? '';
      if (value === oldVal) continue;

      pendingEdits[leaf] = value;
      updates.push({
        parameterPath: path,
        parameterValue: value,
        parameterType: (item?.type as never) ?? 'string',
      });
    }

    if (Object.keys(errors).length > 0) {
      setEditModal((prev) => prev ? { ...prev, errors } : prev);
      message.error({ content: t('device.multi.editValidationFailed'), duration: ERROR_FEEDBACK_DURATION_SECONDS });
      return;
    }

    if (updates.length === 0) {
      message.info({ content: editModal.mode === 'add' ? t('device.multi.addInstanceSuccess') : t('device.multi.noRowChange'), duration: 4 });
      setEditModal(null);
      return;
    }

    try {
      const result = await updateMutation.mutateAsync({ deviceId, parameters: updates });
      setRowEdits((prev) => {
        const next = new Map(prev);
        next.set(targetInstanceId, { edits: pendingEdits, errors: {} });
        return next;
      });
      Object.entries(pendingEdits).forEach(([leaf, value]) => {
        setDraftField(fbKey, `${targetInstanceId}.${leaf}`, value);
      });
      setFeedback(fbKey, {
        kind: 'multi',
        action: 'save',
        submitStatus: 'queued',
        taskId: result.taskId,
        savedInstId: targetInstanceId,
        detail: editModal.mode === 'add' ? t('device.multi.detailAdd', { instId: targetInstanceId, count: updates.length }) : t('device.multi.detailRow', { instId: targetInstanceId, count: updates.length }),
        at: Date.now(),
      });
      message.success({
        content: editModal.mode === 'add'
          ? t('device.multi.addSuccessMsg', { instId: targetInstanceId, count: updates.length })
          : t('device.multi.saveSuccessMsg', { instId: targetInstanceId, count: updates.length }),
        duration: 6,
      });
      setEditModal(null);
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : String(err);
      notification.error({
        message: editModal.mode === 'add'
          ? t('device.multi.addQueueFailed', { group: group.titleZh })
          : t('device.multi.rowQueueFailed', { instId: targetInstanceId ?? '', group: group.titleZh }),
        description: t('device.multi.queueFailedDesc', { count: updates.length, err: errMsg }),
        duration: ERROR_FEEDBACK_DURATION_SECONDS,
      });
      setFeedback(fbKey, {
        kind: 'multi',
        action: editModal.mode === 'add' ? 'add' : 'save',
        submitStatus: 'failed_to_queue',
        detail: t('device.multi.detailFailed', { target: targetInstanceId ?? t('device.multi.actionAdd'), err: errMsg }),
        at: Date.now(),
      });
    } finally {
      void queryClient.invalidateQueries({ queryKey: notificationKeys.all });
    }
    } finally {
      submittingRef.current = false;
      setIsSubmitting(false);
    }
  }, [addMutation, deviceId, displayColumns, editModal, fbKey, group.titleZh, groupParamLeafSet, instanceIds, leafSchemaByLeaf, objectPath, queryClient, refetch, schemaByPath, setDraftField, setFeedback, updateMutation, waitForTaskTerminal, t]);

  const columns: ColumnType<TableRow>[] = [
    {
      title: t('device.multi.instance'),
      dataIndex: 'instanceId',
      key: 'instanceId',
      width: 80,
      fixed: 'left',
      render: (_v: unknown, row: TableRow) => (
        <Text strong>{row.instanceId}</Text>
      ),
    },
    ...displayColumns.map<ColumnType<TableRow>>((column) => {
      const leaf = column.leaf || '';
      // 列内字段约束在同一组所有实例下一致（schema 走 {i} 模板）；优先取首个有 schema 的实例作为模板。
      const titleHint = (() => {
        if (!leaf) return '';
        for (const inst of instanceIds) {
          const tplItem = schemaByPath.get(`${objectPath}${inst}.${leaf}`);
          const hint = formatConstraintHint(tplItem, t);
          if (hint) return hint;
        }
        return '';
      })();
      const baseTitle = column.titleKey ? t(column.titleKey) : (locale === 'zh-CN' ? (column.titleZh ?? column.titleEn) : column.titleEn);
      return {
        title: titleHint ? (
          <Space size={4} wrap>
            <span>{baseTitle}</span>
            <Text type="secondary" style={{ fontSize: 12 }}>{titleHint}</Text>
          </Space>
        ) : baseTitle,
        key: column.key,
        dataIndex: column.key,
        width: column.width ?? 150,
        render: (_v: unknown, row: TableRow) => {
          const value = leaf
            ? (row.instanceId ? cellValue(row.instanceId, leaf) : '')
            : (column.getValue?.(row, instanceContext) ?? '');
          const item = leaf && row.instanceId
            ? (schemaByPath.get(`${objectPath}${row.instanceId}.${leaf}`) ?? leafSchemaByLeaf.get(leaf))
            : leafSchemaByLeaf.get(leaf);
          // 列自定义 formatValue 接收原始值；若未定义，再退到 enum label 兜底。
          // 这两者互斥：列已经声明 formatValue 表示有自定义显示，不应再被 enum 兜底改写。
          const formattedValue = column.formatValue
            ? column.formatValue(value)
            : (leaf ? formatEnumDisplayValue(value, item?.constraints, item?.path) : value);
          return <Text>{formattedValue || '-'}</Text>;
        },
      };
    }),
    {
      title: t('table.operation'),
      key: 'actions',
      width: 148,
      fixed: 'right',
      render: (_v: unknown, row: TableRow) => (
        <Space size={4}>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openEditModal(row)}>
            {t('common.edit')}
          </Button>
          <Popconfirm title={t('device.multi.deleteConfirm')} onConfirm={() => row.instanceId && void handleDelete(row.instanceId)} disabled={!canDelete}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />} disabled={!canDelete}>
              {t('common.delete')}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const title = locale === 'zh-CN' ? group.titleZh : group.titleEn;
  const maxInstances = group.maxInstances && group.maxInstances > 0 ? group.maxInstances : undefined;
  const reachedMax = maxInstances !== undefined && instanceIds.length >= maxInstances;
  const cardTitle = maxInstances !== undefined
    ? `${title}（${instanceIds.length}/${maxInstances}）`
    : title;
  const addDisabled = !canAdd || reachedMax || updateMutation.isPending || addMutation.isPending;
  const addBtn = (
    <Button type="default" icon={<PlusOutlined />} onClick={openAddModal} disabled={addDisabled}>
      {t('common.add')}
    </Button>
  );

  return (
    <Card
      title={cardTitle}
      size="small"
      extra={
        <Space>
          {lastAction && (() => {
            const spec = statusTagSpec(lastAction, lastTask?.status, t);
            const isFailed = lastTask?.status === 'failed' && Boolean(lastTask?.errorMessage);
            const briefFault = isFailed ? formatDeviceFaultBrief(lastTask?.errorMessage) : '';
            const tag = (
              <Tag icon={spec.icon} color={spec.color}>
                {spec.label} · {lastAction.detail}
                {briefFault ? ` · ${briefFault}` : ''} · {formatTime(lastAction.at)}
              </Tag>
            );
            return isFailed ? (
              <Tooltip title={lastTask?.errorMessage} placement="bottomRight">
                {tag}
              </Tooltip>
            ) : tag;
          })()}
          {reachedMax ? (
            <Tooltip title={t('device.multi.reachedMaxTooltip', { max: maxInstances ?? 0 })}>
              <span style={{ display: 'inline-block', cursor: 'not-allowed' }}>{addBtn}</span>
            </Tooltip>
          ) : (
            addBtn
          )}
        </Space>
      }
      style={{ marginBottom: 16 }}
    >
      <Table<TableRow>
        rowKey={(row) => row.key}
        dataSource={tableRows}
        columns={columns}
        loading={isLoading}
        size="small"
        pagination={false}
        scroll={{ x: 'max-content', y: 240 }}
        sticky
      />
      <Modal
        title={editModal?.mode === 'add' ? t('device.multi.modalAddTitle', { title }) : t('device.multi.modalEditTitle', { title, instId: editModal?.instanceId ?? '' })}
        open={Boolean(editModal)}
        onOk={() => void handleSaveEditModal()}
        onCancel={closeEditModal}
        okText={editModal?.mode === 'add' ? t('device.multi.confirmAdd') : t('device.paramEdit.confirmDispatch')}
        cancelText={t('common.cancel')}
        confirmLoading={updateMutation.isPending || addMutation.isPending || isSubmitting}
        width={960}
        destroyOnHidden
      >
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(2, minmax(0, 1fr))',
            columnGap: 16,
            rowGap: 16,
            width: '100%',
          }}
        >
          {editModal && displayColumns.map((column) => {
            const leaf = column.leaf || '';
            if (editModal.mode === 'add' && (!leaf || column.readOnly || !groupParamLeafSet.has(leaf))) {
              return null;
            }
            const value = leaf
              ? (editModal.values[leaf] ?? '')
              : (editModal.instanceId ? (column.getValue?.({ key: editModal.instanceId, instanceId: editModal.instanceId }, instanceContext) ?? '') : '');
            const item = leaf && editModal.instanceId ? (schemaByPath.get(`${objectPath}${editModal.instanceId}.${leaf}`) ?? leafSchemaByLeaf.get(leaf)) : leafSchemaByLeaf.get(leaf);
            const isEditable = Boolean(leaf) && groupParamLeafSet.has(leaf) && !column.readOnly && (editModal.mode === 'add' ? true : (item?.writable ?? true));
            const enumMeta = getEffectiveEnumMeta(item?.constraints, item?.path);
            const error = leaf ? editModal.errors[leaf] : '';
            const label = column.titleKey ? t(column.titleKey) : (locale === 'zh-CN' ? (column.titleZh ?? column.titleEn) : column.titleEn);
            // 同列渲染：column.formatValue 收原始值；未提供则退到 enum 兜底。
            const displayValue = column.formatValue
              ? column.formatValue(value)
              : (leaf ? formatEnumDisplayValue(value, item?.constraints, item?.path) : value);

            return (
              <div key={column.key} style={{ minWidth: 0 }}>
                <div style={{ marginBottom: 6, fontWeight: 500 }}>{label}</div>
                {isEditable && enumMeta && enumMeta.values.length > 0 ? (
                  <Select
                    value={value || undefined}
                    onChange={(next) => leaf && setEditModalValue(leaf, String(next))}
                    style={{ width: '100%' }}
                    status={error ? 'error' : undefined}
                    options={enumMeta.values.map((enumValue, idx) => ({ value: enumValue, label: enumMeta.labels[idx] ?? enumValue }))}
                  />
                ) : isEditable ? (
                  <Input
                    value={value}
                    onChange={(e) => leaf && setEditModalValue(leaf, e.target.value)}
                    status={error ? 'error' : undefined}
                  />
                ) : (
                  <Input value={displayValue} disabled />
                )}
                {column.formatValue && displayValue && displayValue !== value && isEditable && (
                  <div style={{ color: '#8c8c8c', fontSize: 12, marginTop: 4 }}>{displayValue}</div>
                )}
                {error && (
                  <div style={{ color: '#ff4d4f', fontSize: 12, marginTop: 4 }}>{error}</div>
                )}
              </div>
            );
          })}
        </div>
      </Modal>
    </Card>
  );
}

function formatEnbTypeDisplay(value: string): string {
  if (value === '0') return 'Macro';
  if (value === '1') return 'Home';
  return value;
}

function formatEarfcnDisplay(value: string): string {
  if (!value) return '';
  const earfcn = Number(value);
  if (!Number.isFinite(earfcn)) return value;

  const frequency = resolveEarfcnFrequency(earfcn);
  if (frequency === null) return value;
  return `${earfcn}(${frequency.toFixed(1)}MHz)`;
}

function resolveEarfcnFrequency(earfcn: number): number | null {
  if (earfcn >= 36000 && earfcn <= 36199) return 1900 + 0.1 * (earfcn - 36000);
  if (earfcn >= 36200 && earfcn <= 36349) return 2010 + 0.1 * (earfcn - 36200);
  if (earfcn >= 36350 && earfcn <= 36949) return 1850 + 0.1 * (earfcn - 36350);
  if (earfcn >= 36950 && earfcn <= 37549) return 1930 + 0.1 * (earfcn - 36950);
  if (earfcn >= 37550 && earfcn <= 37749) return 1910 + 0.1 * (earfcn - 37550);
  if (earfcn >= 37750 && earfcn <= 38249) return 2570 + 0.1 * (earfcn - 37750);
  if (earfcn >= 38250 && earfcn <= 38649) return 1880 + 0.1 * (earfcn - 38250);
  if (earfcn >= 38650 && earfcn <= 39649) return 2300 + 0.1 * (earfcn - 38650);
  if (earfcn >= 39650 && earfcn <= 41589) return 2496 + 0.1 * (earfcn - 39650);
  if (earfcn >= 41590 && earfcn <= 43589) return 3400 + 0.1 * (earfcn - 41590);
  if (earfcn >= 43590 && earfcn <= 45589) return 3600 + 0.1 * (earfcn - 43590);
  if (earfcn >= 18000 && earfcn <= 18599) return 1920 + 0.1 * (earfcn - 18000);
  if (earfcn >= 0 && earfcn <= 599) return 2110 + 0.1 * earfcn;
  if (earfcn >= 18600 && earfcn <= 19199) return 1850 + 0.1 * (earfcn - 18600);
  if (earfcn >= 600 && earfcn <= 1199) return 1930 + 0.1 * (earfcn - 600);
  if (earfcn >= 19200 && earfcn <= 19949) return 1710 + 0.1 * (earfcn - 19200);
  if (earfcn >= 1200 && earfcn <= 1949) return 1805 + 0.1 * (earfcn - 1200);
  if (earfcn >= 19950 && earfcn <= 20399) return 1710 + 0.1 * (earfcn - 19950);
  if (earfcn >= 1950 && earfcn <= 2399) return 2110 + 0.1 * (earfcn - 1950);
  if (earfcn >= 20400 && earfcn <= 20649) return 824 + 0.1 * (earfcn - 20400);
  if (earfcn >= 2400 && earfcn <= 2649) return 869 + 0.1 * (earfcn - 2400);
  if (earfcn >= 20650 && earfcn <= 20749) return 830 + 0.1 * (earfcn - 20650);
  if (earfcn >= 2650 && earfcn <= 2749) return 875 + 0.1 * (earfcn - 2650);
  if (earfcn >= 20750 && earfcn <= 21449) return 2500 + 0.1 * (earfcn - 20750);
  if (earfcn >= 2750 && earfcn <= 3449) return 2620 + 0.1 * (earfcn - 2750);
  if (earfcn >= 3450 && earfcn <= 3799) return 925 + 0.1 * (earfcn - 3450);
  if (earfcn >= 5010 && earfcn <= 5179) return 729 + 0.1 * (earfcn - 5010);
  if (earfcn >= 5180 && earfcn <= 5279) return 746 + 0.1 * (earfcn - 5180);
  if (earfcn >= 5730 && earfcn <= 5849) return 734 + 0.1 * (earfcn - 5730);
  if (earfcn >= 6150 && earfcn <= 6449) return 791 + 0.1 * (earfcn - 6150);
  if (earfcn >= 9210 && earfcn <= 9659) return 758 + 0.1 * (earfcn - 9210);
  if (earfcn >= 55240 && earfcn <= 56740) return 3550 + 0.1 * (earfcn - 55240);
  if (earfcn >= 46790 && earfcn <= 54539) return 5150 + 0.1 * (earfcn - 46790);
  if (earfcn >= 63000 && earfcn <= 63999) return 5150 + 0.1 * (earfcn - 63000);
  if (earfcn >= 64000 && earfcn <= 64999) return 5725 + 0.1 * (earfcn - 64000);
  return null;
}

// 与 CellParameterForm 同语义：枚举不输出（Select 候选项已自解释），数值/长度输出 [min ~ max]。
function formatConstraintHint(schema: ParameterSchemaItem | undefined, t: TFn): string {
  if (!schema?.constraints) return '';
  const c = schema.constraints;
  if (c.enumValues && c.enumValues.length > 0) return '';
  const isString = schema.type === 'string';
  const min = c.minLength ?? c.minValue;
  const max = c.maxLength ?? c.maxValue;
  if (min !== undefined || max !== undefined) {
    const lo = min ?? '-∞';
    const hi = max ?? '∞';
    return isString ? t('device.multi.hintLenRange', { lo, hi }) : `[${lo} ~ ${hi}]`;
  }
  return '';
}
