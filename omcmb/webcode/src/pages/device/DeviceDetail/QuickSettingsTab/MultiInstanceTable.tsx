import { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Card, Input, Modal, Popconfirm, Select, Space, Table, Tag, Tooltip, Typography, message, notification } from 'antd';
import { CheckCircleOutlined, ClockCircleOutlined, CloseCircleOutlined, DeleteOutlined, EditOutlined, PlusOutlined, SendOutlined, SyncOutlined } from '@ant-design/icons';
import type { ColumnType } from 'antd/es/table';
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

const { Text } = Typography;
const ERROR_FEEDBACK_DURATION_SECONDS = 2;

// "上次操作"状态形状由 frontend-core/store/quickSettingsFeedbackStore (MultiFeedback) 定义,
// 提升至 store 持久化,顶层 TabBar 切走再切回不丢反馈。

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
function statusTagSpec(action: MultiFeedback, taskStatus: DeviceTaskStatus | undefined): StatusTagSpec {
  const actionLabel = action.action === 'save' ? '保存' : action.action === 'add' ? '新增' : '删除';
  if (action.submitStatus === 'failed_to_queue') {
    return { color: 'error', icon: <CloseCircleOutlined />, label: `${actionLabel}入队失败` };
  }
  // AddObject / DeleteObject 当前不返 task_id;仅 Save 走完整状态机
  if (!action.taskId) {
    return { color: 'processing', icon: <SyncOutlined spin />, label: `${actionLabel}已入队` };
  }
  switch (taskStatus) {
    case 'completed':
      return { color: 'success', icon: <CheckCircleOutlined />, label: `${actionLabel}成功` };
    case 'failed':
      return { color: 'error', icon: <CloseCircleOutlined />, label: `${actionLabel}基站应答失败` };
    case 'expired':
      return { color: 'warning', icon: <ClockCircleOutlined />, label: `${actionLabel}超时` };
    case 'cancelled':
      return { color: 'default', icon: <CloseCircleOutlined />, label: `${actionLabel}已取消` };
    case 'sent':
      return { color: 'processing', icon: <SendOutlined />, label: `${actionLabel}已发送给基站` };
    case 'pending':
    default:
      return { color: 'processing', icon: <SyncOutlined spin />, label: `${actionLabel}已入队,等待下发` };
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
  titleZh: string;
  titleEn: string;
  width?: number;
  readOnly?: boolean;
  getValue?: (row: TableRow, ctx: QuickSettingsInstanceContext) => string;
  formatValue?: (value: string) => string;
}

const BM_SPECIAL_COLUMNS: Record<string, SpecialColumnSpec[]> = {
  'enb-neighbor-freq': [
    { key: 'EUTRACarrierARFCN', leaf: 'EUTRACarrierARFCN', titleZh: '频点', titleEn: 'Frequency', width: 180, formatValue: formatEarfcnDisplay },
    { key: 'QOffsetFreq', leaf: 'QOffsetFreq', titleZh: 'Q-OffsetRange', titleEn: 'Q-OffsetRange', width: 140 },
    { key: 'QRxLevMinSIB5', leaf: 'QRxLevMinSIB5', titleZh: 'Q-RxLevMin', titleEn: 'Q-RxLevMin', width: 130 },
    { key: 'CellReselectionPriority', leaf: 'CellReselectionPriority', titleZh: '重选优先级', titleEn: 'Reselection Priority', width: 130 },
    { key: 'ThreshXHigh', leaf: 'ThreshXHigh', titleZh: '高重选门限', titleEn: 'Reselection Thresh High', width: 130 },
    { key: 'ThreshXLow', leaf: 'ThreshXLow', titleZh: '低重选门限', titleEn: 'Reselection Thresh Low', width: 130 },
    { key: 'PMax', leaf: 'PMax', titleZh: 'UE最大发送功率', titleEn: 'UE Max Tx Power', width: 150 },
    { key: 'TReselectionEUTRA', leaf: 'TReselectionEUTRA', titleZh: '重选定时器', titleEn: 'TReselectionEUTRA', width: 130 },
  ],
  'enb-neighbor-cell': [
    { key: 'cellIndex', titleZh: 'cellIndex', titleEn: 'cellIndex', width: 110, readOnly: true, getValue: (_row, ctx) => `Cell ${ctx.fapInstance}` },
    { key: 'EUTRACarrierARFCN', leaf: 'EUTRACarrierARFCN', titleZh: '频点', titleEn: 'Frequency', width: 180, formatValue: formatEarfcnDisplay },
    { key: 'PhyCellID', leaf: 'PhyCellID', titleZh: 'PCI', titleEn: 'PCI', width: 100 },
    { key: 'QOffset', leaf: 'QOffset', titleZh: 'QOffset', titleEn: 'QOffset', width: 110 },
    { key: 'CIO', leaf: 'CIO', titleZh: 'CIO', titleEn: 'CIO', width: 100 },
    { key: 'TAC', leaf: 'TAC', titleZh: 'TAC', titleEn: 'TAC', width: 110, readOnly: true },
    { key: 'PLMNID', leaf: 'PLMNID', titleZh: 'PLMN', titleEn: 'PLMN', width: 140 },
    { key: 'CID', leaf: 'CID', titleZh: 'ECI', titleEn: 'ECI', width: 130, readOnly: true },
    { key: 'EnbType', leaf: 'EnbType', titleZh: 'eNodeB Type', titleEn: 'eNodeB Type', width: 140, readOnly: true, formatValue: formatEnbTypeDisplay },
  ],
};

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
    throw new Error('等待任务完成超时');
  }, []);

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
            message: `设备侧数据回读失败(${group.titleZh})`,
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
        message: `基站应答失败(${group.titleZh})`,
        description: lastTask.errorMessage || '未知错误,可在通知中心查看任务详情',
        duration: ERROR_FEEDBACK_DURATION_SECONDS,
      });
      patchFeedback(fbKey, { notifiedFailedTaskId: lastTask.id });
    }
  }, [lastTask, lastAction, group.titleZh, patchFeedback, fbKey]);

  // 单层确认：外层 Popconfirm 已二次确认，这里直接执行删除逻辑（原 Modal.confirm 套层移除）。
  const handleDelete = async (instId: string) => {
    try {
      // T-0157 C7: 后端现返回 { taskId } → 消费 taskId 让 Tag 走完整状态机
      const result = await deleteMutation.mutateAsync({ deviceId, objectPath: `${objectPath}${instId}.` });
      message.success({
        content: `已下发 DeleteObject(${instId}),请在右上角铃铛查看任务结果`,
        duration: 6,
      });
      setFeedback(fbKey, {
        kind: 'multi',
        action: 'delete',
        submitStatus: 'queued',
        taskId: result.taskId,
        detail: `实例 ${instId}`,
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
        message: `DeleteObject 入队失败(${group.titleZh})`,
        description: `实例 ${instId} 删除失败:${errMsg}`,
        duration: ERROR_FEEDBACK_DURATION_SECONDS,
      });
      setFeedback(fbKey, {
        kind: 'multi',
        action: 'delete',
        submitStatus: 'failed_to_queue',
        detail: `实例 ${instId}:${errMsg}`,
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
    if (updateMutation.isPending) return;
    setEditModal(null);
  }, [updateMutation.isPending]);

  const handleSaveEditModal = useCallback(async () => {
    if (!editModal) return;

    const errors: Record<string, string> = {};
    const updates: ParameterUpdateRequest[] = [];
    const pendingEdits: Record<string, string> = {};
    let targetInstanceId = editModal.instanceId;

    if (editModal.mode === 'add') {
      try {
        const addResult = await addMutation.mutateAsync({ deviceId, objectPath });
        const addTask = await waitForTaskTerminal(addResult.taskId);
        if (addTask.status !== 'completed') {
          throw new Error(addTask.errorMessage || `新增实例失败(${addTask.status})`);
        }

        const refreshed = await refetch();
        const nextObject = refreshed.data?.objects.find((o) => o.path === objectPath);
        const knownInstances = new Set(instanceIds);
        targetInstanceId = nextObject?.currentInstances.map((n) => String(n)).find((instId) => !knownInstances.has(instId));
        if (!targetInstanceId) {
          throw new Error('新增实例成功，但未能识别新实例号');
        }
      } catch (err) {
        const errMsg = err instanceof Error ? err.message : String(err);
        notification.error({
          message: `新增失败(${group.titleZh})`,
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
      message.error({ content: '编辑页校验失败,请修正后再保存', duration: ERROR_FEEDBACK_DURATION_SECONDS });
      return;
    }

    if (updates.length === 0) {
      message.info({ content: editModal.mode === 'add' ? '新增实例成功' : '该行无变更', duration: 4 });
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
        detail: editModal.mode === 'add' ? `新增实例 ${targetInstanceId} ${updates.length} 项` : `第 ${targetInstanceId} 行 ${updates.length} 项`,
        at: Date.now(),
      });
      message.success({
        content: editModal.mode === 'add'
          ? `已新增实例 ${targetInstanceId}，并下发 ${updates.length} 项变更`
          : `第 ${targetInstanceId} 行已下发 ${updates.length} 项变更,正在等待基站应答(Tag 会自动刷新)`,
        duration: 6,
      });
      setEditModal(null);
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : String(err);
      notification.error({
        message: `${editModal.mode === 'add' ? '新增实例' : `第 ${targetInstanceId} 行`}入队失败(${group.titleZh})`,
        description: `${updates.length} 项变更入队失败:${errMsg}。输入值已保留,可修正后重试。`,
        duration: ERROR_FEEDBACK_DURATION_SECONDS,
      });
      setFeedback(fbKey, {
        kind: 'multi',
        action: editModal.mode === 'add' ? 'add' : 'save',
        submitStatus: 'failed_to_queue',
        detail: `${targetInstanceId ?? '新增实例'}:${errMsg}`,
        at: Date.now(),
      });
    } finally {
      void queryClient.invalidateQueries({ queryKey: notificationKeys.all });
    }
  }, [addMutation, deviceId, displayColumns, editModal, fbKey, group.titleZh, groupParamLeafSet, instanceIds, leafSchemaByLeaf, objectPath, queryClient, refetch, schemaByPath, setDraftField, setFeedback, updateMutation, waitForTaskTerminal]);

  const columns: ColumnType<TableRow>[] = [
    {
      title: '实例',
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
          const hint = formatConstraintHint(tplItem);
          if (hint) return hint;
        }
        return '';
      })();
      const baseTitle = locale === 'zh-CN' ? column.titleZh : column.titleEn;
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
      title: '操作',
      key: 'actions',
      width: 148,
      fixed: 'right',
      render: (_v: unknown, row: TableRow) => (
        <Space size={4}>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openEditModal(row)}>
            修改
          </Button>
          <Popconfirm title="确认删除？" onConfirm={() => row.instanceId && void handleDelete(row.instanceId)} disabled={!canDelete}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />} disabled={!canDelete}>
              删除
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
      新 增
    </Button>
  );

  return (
    <Card
      title={cardTitle}
      size="small"
      extra={
        <Space>
          {lastAction && (() => {
            const spec = statusTagSpec(lastAction, lastTask?.status);
            return (
              <Tag icon={spec.icon} color={spec.color}>
                {spec.label} · {lastAction.detail} · {formatTime(lastAction.at)}
              </Tag>
            );
          })()}
          {reachedMax ? (
            <Tooltip title={`已达上限 ${maxInstances}，如需新增请先删除其它实例`}>
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
        title={editModal?.mode === 'add' ? `${title} · 新增实例` : `${title} · 修改实例 ${editModal?.instanceId ?? ''}`}
        open={Boolean(editModal)}
        onOk={() => void handleSaveEditModal()}
        onCancel={closeEditModal}
        okText={editModal?.mode === 'add' ? '确认新增' : '确认下发'}
        cancelText="取消"
        confirmLoading={updateMutation.isPending || addMutation.isPending}
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
            const label = locale === 'zh-CN' ? column.titleZh : column.titleEn;
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
function formatConstraintHint(schema?: ParameterSchemaItem): string {
  if (!schema?.constraints) return '';
  const c = schema.constraints;
  if (c.enumValues && c.enumValues.length > 0) return '';
  const isString = schema.type === 'string';
  const min = c.minLength ?? c.minValue;
  const max = c.maxLength ?? c.maxValue;
  if (min !== undefined || max !== undefined) {
    const lo = min ?? '-∞';
    const hi = max ?? '∞';
    return isString ? `[长度 ${lo} ~ ${hi}]` : `[${lo} ~ ${hi}]`;
  }
  return '';
}
