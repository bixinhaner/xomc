import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Button,
  Card,
  Checkbox,
  Collapse,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Spin,
  Table,
  Tag,
  Tooltip,
  Typography,
  message,
  notification,
} from 'antd';
import {
  CheckCircleOutlined,
  ClockCircleOutlined,
  CloseCircleOutlined,
  DeleteOutlined,
  LoadingOutlined,
  PlusCircleFilled,
  PlusOutlined,
  SaveOutlined,
  SendOutlined,
  SyncOutlined,
} from '@ant-design/icons';
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
import type { QuickSettingsGroup, QuickSettingsParam } from '@core/types/quicksettings';
import {
  applyInstanceContext,
  getEffectiveEnumMeta,
  validateValue,
  type QuickSettingsInstanceContext,
} from './validators';

/** 字段控件类型(原 bscPanelDefs.ts，现内联，由 XML 驱动)。 */
type BscFieldType = 'string' | 'int' | 'enum' | 'multiCheckbox';

interface BscFieldDef {
  leaf: string;
  label: string;
  type: BscFieldType;
  readonly?: boolean;
  required?: boolean;
  hint?: string;
  enumOptions?: { value: string; label: string }[];
  checkboxOptions?: string[];
  minValue?: number;
  maxValue?: number;
}

interface BscSubTableDef {
  /** 子表对象名（拼为 DeviceGSM.Bts.{i}.<subObject>.{j}.<col>） */
  subObject: string;
  /** 列定义 */
  columns: { key: string; title: string }[];
}

/** 把 XML 来的 QuickSettingsParam 转成本地 BscFieldDef。 */
function paramToFieldDef(param: QuickSettingsParam, locale: 'zh-CN' | 'en-US'): BscFieldDef {
  const label = locale === 'zh-CN' ? param.titleZh : param.titleEn;
  const t = (param.type as BscFieldType) || 'string';
  return {
    leaf: param.leaf || param.name,
    label: label || param.name,
    type: t === 'string' || t === 'int' || t === 'enum' || t === 'multiCheckbox' ? t : 'string',
    readonly: param.readonly,
    required: param.required,
    hint: param.hint,
    enumOptions: param.enumOptions?.map((o) => ({ value: o.value, label: o.label })),
    checkboxOptions: param.checkboxOptions,
    minValue: param.minValue,
    maxValue: param.maxValue,
  };
}

/** 从子表 group 的 objectPath 推断 subObject 名。e.g. DeviceGSM.Bts.{i}.Trx.{j}. → Trx。 */
function extractSubObject(childObjectPath: string, parentObjectPath: string): string {
  if (!childObjectPath || !parentObjectPath) return '';
  if (!childObjectPath.startsWith(parentObjectPath)) return '';
  const suffix = childObjectPath.slice(parentObjectPath.length);
  // suffix like "Trx.{j}." → 取首段
  return suffix.split('.')[0] || '';
}

const { Text } = Typography;
const ERROR_FEEDBACK_DURATION_SECONDS = 2;

interface InstanceSelectorFormProps {
  deviceId: string;
  /** 顶层多实例选择器 group（style="table"），如 bsc-bts-access。 */
  selectorGroup: QuickSettingsGroup;
  /** 嵌入到该选择器下的子 group 列表（style="form" 或 style="subtable"，parentSelector === selectorGroup.id）。 */
  childGroups: QuickSettingsGroup[];
  instanceContext: QuickSettingsInstanceContext;
  locale: 'zh-CN' | 'en-US';
}

interface StatusTagSpec {
  color: string;
  icon: React.ReactNode;
  label: string;
}

function formatTime(at: number): string {
  const d = new Date(at);
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

function statusTagSpec(action: MultiFeedback, taskStatus: DeviceTaskStatus | undefined): StatusTagSpec {
  const actionLabel = action.action === 'save' ? '保存' : action.action === 'add' ? '新增' : '删除';
  if (action.submitStatus === 'failed_to_queue') {
    return { color: 'error', icon: <CloseCircleOutlined />, label: `${actionLabel}入队失败` };
  }
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

/** 按 BscFieldDef 做最小校验：required / int 范围。返回 '' 表示通过。 */
function validateFieldDef(def: BscFieldDef, value: string): string {
  if (def.readonly) return '';
  if (def.required && value === '') return '不能为空';
  if (value === '') return '';
  if (def.type === 'int') {
    const n = Number(value);
    if (!Number.isInteger(n)) return '需要整数';
    if (def.minValue !== undefined && n < def.minValue) return `不能小于 ${def.minValue}`;
    if (def.maxValue !== undefined && n > def.maxValue) return `不能大于 ${def.maxValue}`;
  }
  return '';
}

/** BTS Config / Handover 折叠面板内通用 3 列字段网格。 */
function renderFieldGrid(
  fields: BscFieldDef[],
  selectedInstId: string,
  objectPath: string,
  getValue: (path: string) => string,
  setValue: (path: string, value: string, validator?: (v: string) => string) => void,
  errors: Record<string, string>,
  getWritable: (path: string) => boolean,
): React.ReactNode {
  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(3, minmax(0, 1fr))',
        columnGap: 24,
        rowGap: 16,
        width: '100%',
      }}
    >
      {fields.map((def) => {
        const path = `${objectPath}${selectedInstId}.${def.leaf}`;
        const rawValue = getValue(path);
        const err = errors[path] ?? '';
        const validator = (v: string) => validateFieldDef(def, v);
        // 以 schema 为准：后端 writable=false 严格只读；def.readonly 是本地补充只读（如 ID 虚拟字段）。
        const isReadOnly = def.readonly === true || getWritable(path) === false;
        const labelNode = (
          <div style={{ marginBottom: 6, fontWeight: 500 }}>
            <span>
              {def.required && !isReadOnly && (
                <span style={{ color: '#ff4d4f', marginRight: 2 }}>*</span>
              )}
              {def.label}
              {isReadOnly && (
                <span style={{ color: '#8c8c8c', fontSize: 12, marginLeft: 6 }}>(只读)</span>
              )}
            </span>
          </div>
        );
        const hintNode = def.hint ? (
          <div style={{ color: '#8c8c8c', fontSize: 12, marginTop: 4 }}>{def.hint}</div>
        ) : null;
        const errNode = err ? (
          <div style={{ color: '#ff4d4f', fontSize: 12, marginTop: 4 }}>{err}</div>
        ) : null;

        if (isReadOnly) {
          const displayValue = rawValue !== '' && rawValue != null ? rawValue : '未上报';
          return (
            <div key={def.leaf} style={{ minWidth: 0 }}>
              {labelNode}
              <Input value={displayValue} disabled />
              {hintNode}
            </div>
          );
        }

        if (def.type === 'enum') {
          return (
            <div key={def.leaf} style={{ minWidth: 0 }}>
              {labelNode}
              <Select
                value={rawValue || undefined}
                onChange={(next) => setValue(path, String(next), validator)}
                style={{ width: '100%' }}
                status={err ? 'error' : undefined}
                options={def.enumOptions ?? []}
                allowClear
              />
              {hintNode}
              {errNode}
            </div>
          );
        }

        if (def.type === 'multiCheckbox') {
          const selected = rawValue ? rawValue.split(',').map((s) => s.trim()).filter(Boolean) : [];
          return (
            <div key={def.leaf} style={{ minWidth: 0, gridColumn: '1 / -1' }}>
              {labelNode}
              <Checkbox.Group
                value={selected}
                options={(def.checkboxOptions ?? []).map((v) => ({ value: v, label: v }))}
                onChange={(vals) => setValue(path, (vals as string[]).join(','), validator)}
              />
              {hintNode}
              {errNode}
            </div>
          );
        }

        return (
          <div key={def.leaf} style={{ minWidth: 0 }}>
            {labelNode}
            <Input
              value={rawValue}
              onChange={(e) => setValue(path, e.target.value, validator)}
              status={err ? 'error' : undefined}
            />
            {hintNode}
            {errNode}
          </div>
        );
      })}
    </div>
  );
}

/** 4G / 2G Neighbor / Trx 折叠面板内的可编辑子表。 */
function renderSubTable(
  def: BscSubTableDef,
  selectedInstId: string,
  objectPath: string,
  rowIds: number[],
  getValue: (path: string) => string,
  setValue: (path: string, value: string) => void,
  onAdd: () => void,
  onDelete: (rowIdx: number) => void,
  addPending: boolean,
  deletePending: boolean,
  getWritable: (path: string) => boolean,
): React.ReactNode {
  const dataSource = rowIds.map((id) => ({ key: id, id }));
  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 80,
    },
    ...def.columns.map((col) => ({
      title: col.title,
      key: col.key,
      render: (_: unknown, row: { id: number }) => {
        const path = `${objectPath}${selectedInstId}.${def.subObject}.${row.id}.${col.key}`;
        const writable = getWritable(path);
        return (
          <Input
            value={getValue(path)}
            onChange={(e) => setValue(path, e.target.value)}
            size="small"
            disabled={!writable}
          />
        );
      },
    })),
    {
      title: 'Operate',
      key: '_op',
      width: 90,
      render: (_: unknown, row: { id: number }) => (
        <Popconfirm
          title={`确认删除 ${def.subObject}.${row.id} ？`}
          onConfirm={() => onDelete(row.id)}
          disabled={deletePending}
        >
          <Button
            type="text"
            danger
            icon={<DeleteOutlined />}
            loading={deletePending}
          />
        </Popconfirm>
      ),
    },
  ];
  return (
    <div style={{ position: 'relative' }}>
      <Button
        type="primary"
        shape="circle"
        size="small"
        icon={<PlusCircleFilled />}
        onClick={onAdd}
        loading={addPending}
        style={{ position: 'absolute', top: -36, right: 8, zIndex: 1 }}
      />
      <Table
        size="small"
        pagination={false}
        rowKey="key"
        dataSource={dataSource}
        columns={columns}
        locale={{ emptyText: '暂无数据' }}
      />
    </div>
  );
}

/**
 * BSC 风格的多实例分组渲染：
 *  - 顶部下拉选择 BTS 实例 + 新增 / 删除 / 获取信息 按钮
 *  - 下方内联展示当前选中实例的全部参数表单（无需点编辑）
 *  - 与 MultiInstanceTable 共用 schema / feedback / 草稿 store，行为语义保持一致
 */
export default function InstanceSelectorForm({
  deviceId,
  selectorGroup,
  childGroups,
  instanceContext,
  locale,
}: InstanceSelectorFormProps) {
  const objectPath = useMemo(() => {
    const resolved = applyInstanceContext(selectorGroup.objectPath || '', instanceContext, {
      preserveTrailingInstance: true,
    });
    return resolved.replace(/\{i\}\.$/, '');
  }, [selectorGroup, instanceContext]);

  const { data: schemaResp, isLoading, isFetching, refetch } = useParameterSchema(deviceId, objectPath);
  const updateMutation = useUpdateParameters();
  const addMutation = useAddObject();
  const deleteMutation = useDeleteObject();
  const queryClient = useQueryClient();

  const objectEntry = useMemo(
    () => schemaResp?.objects.find((o) => o.path === objectPath),
    [schemaResp, objectPath],
  );
  const canAdd = objectEntry?.canAdd ?? Boolean(selectorGroup.objectPath && objectPath);
  const canDelete = objectEntry?.canDeleteAny ?? false;

  const schemaByPath = useMemo(() => {
    const map = new Map<string, ParameterSchemaItem>();
    schemaResp?.parameters.forEach((p) => map.set(p.path, p));
    return map;
  }, [schemaResp]);

  const instanceIds = useMemo(() => {
    if (!objectEntry) return [] as string[];
    return objectEntry.currentInstances
      .map((n) => String(n))
      .sort((a, b) => Number(a) - Number(b));
  }, [objectEntry]);

  /** 字段 leaf → schema（任取第一个有 schema 的实例做模板）。新增/默认值/校验沿用。 */
  const leafSchemaByLeaf = useMemo(() => {
    const map = new Map<string, ParameterSchemaItem>();
    for (const param of selectorGroup.params) {
      const leaf = param.leaf || '';
      if (!leaf || map.has(leaf)) continue;
      const existing = instanceIds
        .map((id) => schemaByPath.get(`${objectPath}${id}.${leaf}`))
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
  }, [selectorGroup.params, instanceIds, objectPath, schemaByPath, schemaResp?.parameters]);

  const groupParamLeafSet = useMemo(
    () => new Set(selectorGroup.params.map((param) => param.leaf).filter(Boolean)),
    [selectorGroup.params],
  );

  const [selectedInstId, setSelectedInstId] = useState<string | null>(null);

  // schema 加载完成后自动选中第一个实例（若当前没选 / 选中已不存在）
  useEffect(() => {
    if (instanceIds.length === 0) {
      if (selectedInstId !== null) setSelectedInstId(null);
      return;
    }
    if (!selectedInstId || !instanceIds.includes(selectedInstId)) {
      setSelectedInstId(instanceIds[0]);
    }
  }, [instanceIds, selectedInstId]);

  // 表单本地编辑值（仅记录用户改过的字段；其余从 schema 取）
  const [formEdits, setFormEdits] = useState<Record<string, string>>({});
  const [formErrors, setFormErrors] = useState<Record<string, string>>({});

  // 获取信息状态感知：记录最近一次刷新时间，并在刷新完成瞬间触发一次高亮闪烁。
  const [lastSyncedAt, setLastSyncedAt] = useState<Date | null>(null);
  const [justRefreshed, setJustRefreshed] = useState(false);
  const prevFetchingRef = useRef(false);
  useEffect(() => {
    if (prevFetchingRef.current && !isFetching) {
      setLastSyncedAt(new Date());
      setJustRefreshed(true);
      const t = setTimeout(() => setJustRefreshed(false), 1500);
      prevFetchingRef.current = isFetching;
      return () => clearTimeout(t);
    }
    prevFetchingRef.current = isFetching;
  }, [isFetching]);

  // 切换实例时清掉上一实例的编辑值
  useEffect(() => {
    setFormEdits({});
    setFormErrors({});
  }, [selectedInstId]);

  const fbKey = feedbackKey(
    deviceId,
    selectorGroup.id,
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

  // 任务进入终态 → 只刷新当前多实例的 schema(精确到 objectPath 该一份查询),
  // 并清掉乐观编辑值（保留 selectedInstId）。不作跨 device 的全量 invalidate，
  // 避免领居/顶层 selector 等无关查询被动重拉。
  useEffect(() => {
    if (!lastTask || !isDeviceTaskTerminal(lastTask.status)) return;
    let cancelled = false;
    void (async () => {
      try {
        await refetch();
      } catch (err) {
        if (!cancelled) {
          notification.error({
            message: `设备侧数据回读失败(${selectorGroup.titleZh})`,
            description: err instanceof Error ? err.message : String(err),
            duration: ERROR_FEEDBACK_DURATION_SECONDS,
          });
        }
        return;
      }
      if (cancelled) return;
      // 仅 save 操作影响当前实例字段；add/delete 已通过 setSelectedInstId 副作用处理
      if (lastAction?.action === 'save' && lastAction.savedInstId === selectedInstId) {
        setFormEdits({});
        setFormErrors({});
      }
    })();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [lastTask?.id, lastTask?.status]);

  // 基站应答失败 → notification（只首次提示）
  useEffect(() => {
    if (
      lastTask &&
      lastTask.status === 'failed' &&
      lastAction &&
      lastAction.notifiedFailedTaskId !== lastTask.id
    ) {
      notification.error({
        message: `基站应答失败(${selectorGroup.titleZh})`,
        description: lastTask.errorMessage || '未知错误,可在通知中心查看任务详情',
        duration: ERROR_FEEDBACK_DURATION_SECONDS,
      });
      patchFeedback(fbKey, { notifiedFailedTaskId: lastTask.id });
    }
  }, [lastTask, lastAction, selectorGroup.titleZh, patchFeedback, fbKey]);

  const fieldValueByPath = useCallback(
    (path: string): string => {
      if (path in formEdits) return formEdits[path];
      return schemaByPath.get(path)?.currentValue ?? '';
    },
    [formEdits, schemaByPath],
  );

  const setFieldValueByPath = useCallback(
    (path: string, value: string, validator?: (v: string) => string) => {
      setFormEdits((prev) => ({ ...prev, [path]: value }));
      if (validator) {
        const err = validator(value);
        setFormErrors((prev) => ({ ...prev, [path]: err }));
      } else if (formErrors[path]) {
        setFormErrors((prev) => ({ ...prev, [path]: '' }));
      }
    },
    [formErrors],
  );

  /**
   * TODO: 后端 BSC.xml 中大量字段错误标记为 READ_ONLY，待后端参数属性修正后再启用此逻辑。
   * 当前阶段一律按可编辑处理，避免阻塞下发功能联调。
   */
  const getWritableByPath = useCallback((_path: string): boolean => true, []);

  /** 把 BSC 子表对象（Neighbor4G / Neighbor2G / Trx）当前已存在的实例号枚举出来。 */
  const subInstanceIds = useCallback(
    (subObject: string): number[] => {
      if (!selectedInstId) return [];
      const prefix = `${objectPath}${selectedInstId}.${subObject}.`;
      const ids = new Set<number>();
      for (const path of schemaByPath.keys()) {
        if (!path.startsWith(prefix)) continue;
        const rest = path.slice(prefix.length);
        const dot = rest.indexOf('.');
        if (dot < 0) continue;
        const n = Number(rest.slice(0, dot));
        if (Number.isInteger(n)) ids.add(n);
      }
      return Array.from(ids).sort((a, b) => a - b);
    },
    [objectPath, schemaByPath, selectedInstId],
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

  // 新增实例：用 Modal 收集新值 → AddObject → 等待 → 取新实例号 → SetParameterValues
  type AddModalState = { values: Record<string, string>; errors: Record<string, string> } | null;
  const [addModal, setAddModal] = useState<AddModalState>(null);

  const buildInitialAddValues = useCallback((): Record<string, string> => {
    return Object.fromEntries(
      selectorGroup.params.map((param) => {
        const leaf = param.leaf || '';
        return [leaf, leafSchemaByLeaf.get(leaf)?.defaultValue ?? ''];
      }),
    );
  }, [selectorGroup.params, leafSchemaByLeaf]);

  const openAddModal = useCallback(() => {
    setAddModal({ values: buildInitialAddValues(), errors: {} });
  }, [buildInitialAddValues]);

  const closeAddModal = useCallback(() => {
    if (addMutation.isPending || updateMutation.isPending) return;
    setAddModal(null);
  }, [addMutation.isPending, updateMutation.isPending]);

  const setAddModalValue = useCallback(
    (leaf: string, value: string) => {
      setAddModal((prev) => {
        if (!prev) return prev;
        const item = leafSchemaByLeaf.get(leaf);
        const err = validateValue(value, (item?.type as never) ?? 'string', item?.constraints) ?? '';
        return {
          values: { ...prev.values, [leaf]: value },
          errors: { ...prev.errors, [leaf]: err },
        };
      });
    },
    [leafSchemaByLeaf],
  );

  const handleConfirmAdd = useCallback(async () => {
    if (!addModal) return;

    // 校验
    const errors: Record<string, string> = {};
    for (const param of selectorGroup.params) {
      const leaf = param.leaf || '';
      if (!leaf) continue;
      const item = leafSchemaByLeaf.get(leaf);
      const value = addModal.values[leaf] ?? '';
      const err = validateValue(value, (item?.type as never) ?? 'string', item?.constraints);
      if (err) errors[leaf] = err;
    }
    if (Object.keys(errors).length > 0) {
      setAddModal((prev) => (prev ? { ...prev, errors } : prev));
      message.error({ content: '校验失败,请修正后再保存', duration: ERROR_FEEDBACK_DURATION_SECONDS });
      return;
    }

    let newInstanceId: string | undefined;
    try {
      const addResult = await addMutation.mutateAsync({ deviceId, objectPath });
      const addTask = await waitForTaskTerminal(addResult.taskId);
      if (addTask.status !== 'completed') {
        throw new Error(addTask.errorMessage || `新增实例失败(${addTask.status})`);
      }
      const refreshed = await refetch();
      const nextObject = refreshed.data?.objects.find((o) => o.path === objectPath);
      const knownInstances = new Set(instanceIds);
      newInstanceId = nextObject?.currentInstances
        .map((n) => String(n))
        .find((id) => !knownInstances.has(id));
      if (!newInstanceId) throw new Error('新增实例成功,但未能识别新实例号');
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : String(err);
      notification.error({
        message: `新增失败(${selectorGroup.titleZh})`,
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

    // 下发字段值
    const updates: ParameterUpdateRequest[] = [];
    for (const param of selectorGroup.params) {
      const leaf = param.leaf || '';
      if (!leaf) continue;
      const item = leafSchemaByLeaf.get(leaf);
      const value = addModal.values[leaf] ?? '';
      updates.push({
        parameterPath: `${objectPath}${newInstanceId}.${leaf}`,
        parameterValue: value,
        parameterType: (item?.type as never) ?? 'string',
      });
    }

    if (updates.length === 0) {
      message.success({ content: `已新增实例 ${newInstanceId}`, duration: 4 });
      setAddModal(null);
      setSelectedInstId(newInstanceId);
      return;
    }

    try {
      const result = await updateMutation.mutateAsync({ deviceId, parameters: updates });
      setFeedback(fbKey, {
        kind: 'multi',
        action: 'save',
        submitStatus: 'queued',
        taskId: result.taskId,
        savedInstId: newInstanceId,
        detail: `新增实例 ${newInstanceId} ${updates.length} 项`,
        at: Date.now(),
      });
      message.success({
        content: `已新增实例 ${newInstanceId},并下发 ${updates.length} 项变更`,
        duration: 6,
      });
      setAddModal(null);
      setSelectedInstId(newInstanceId);
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : String(err);
      notification.error({
        message: `新增实例字段下发入队失败(${selectorGroup.titleZh})`,
        description: `${updates.length} 项变更入队失败:${errMsg}`,
        duration: ERROR_FEEDBACK_DURATION_SECONDS,
      });
      setFeedback(fbKey, {
        kind: 'multi',
        action: 'save',
        submitStatus: 'failed_to_queue',
        detail: `${newInstanceId}:${errMsg}`,
        at: Date.now(),
      });
    } finally {
      void queryClient.invalidateQueries({ queryKey: notificationKeys.all });
    }
  }, [
    addModal,
    addMutation,
    deviceId,
    fbKey,
    selectorGroup.params,
    selectorGroup.titleZh,
    instanceIds,
    leafSchemaByLeaf,
    objectPath,
    queryClient,
    refetch,
    setFeedback,
    updateMutation,
    waitForTaskTerminal,
  ]);

  const handleDelete = useCallback(async () => {
    if (!selectedInstId) return;
    try {
      const result = await deleteMutation.mutateAsync({
        deviceId,
        objectPath: `${objectPath}${selectedInstId}.`,
      });
      message.success({
        content: `已下发 DeleteObject(${selectedInstId}),请在右上角铃铛查看任务结果`,
        duration: 6,
      });
      setFeedback(fbKey, {
        kind: 'multi',
        action: 'delete',
        submitStatus: 'queued',
        taskId: result.taskId,
        detail: `实例 ${selectedInstId}`,
        at: Date.now(),
      });
      // 删后让 useEffect 重置 selectedInstId
      setSelectedInstId((prev) => (prev === selectedInstId ? null : prev));
      void refetch();
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : String(err);
      notification.error({
        message: `DeleteObject 入队失败(${selectorGroup.titleZh})`,
        description: `实例 ${selectedInstId} 删除失败:${errMsg}`,
        duration: ERROR_FEEDBACK_DURATION_SECONDS,
      });
      setFeedback(fbKey, {
        kind: 'multi',
        action: 'delete',
        submitStatus: 'failed_to_queue',
        detail: `实例 ${selectedInstId}:${errMsg}`,
        at: Date.now(),
      });
    } finally {
      void queryClient.invalidateQueries({ queryKey: notificationKeys.all });
    }
  }, [deleteMutation, deviceId, fbKey, selectorGroup.titleZh, objectPath, queryClient, refetch, selectedInstId, setFeedback]);

  const handleSave = useCallback(async () => {
    if (!selectedInstId) return;

    const blockingErrors: Record<string, string> = {};
    const updates: ParameterUpdateRequest[] = [];

    for (const [path, value] of Object.entries(formEdits)) {
      if (formErrors[path]) {
        blockingErrors[path] = formErrors[path];
        continue;
      }
      const item = schemaByPath.get(path);
      if (item && item.writable === false) continue;
      const oldVal = item?.currentValue ?? '';
      if (value === oldVal) continue;
      updates.push({
        parameterPath: path,
        parameterValue: value,
        parameterType: (item?.type as never) ?? 'string',
      });
    }

    if (Object.keys(blockingErrors).length > 0) {
      message.error({ content: '校验失败,请修正后再保存', duration: ERROR_FEEDBACK_DURATION_SECONDS });
      return;
    }

    if (updates.length === 0) {
      message.info({ content: '无变更', duration: 4 });
      return;
    }

    try {
      const result = await updateMutation.mutateAsync({ deviceId, parameters: updates });
      setFeedback(fbKey, {
        kind: 'multi',
        action: 'save',
        submitStatus: 'queued',
        taskId: result.taskId,
        savedInstId: selectedInstId,
        detail: `实例 ${selectedInstId} ${updates.length} 项`,
        at: Date.now(),
      });
      message.success({
        content: `已下发 ${updates.length} 项变更,正在等待基站应答(Tag 会自动刷新)`,
        duration: 6,
      });
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : String(err);
      notification.error({
        message: `入队失败(${selectorGroup.titleZh})`,
        description: `${updates.length} 项变更入队失败:${errMsg}`,
        duration: ERROR_FEEDBACK_DURATION_SECONDS,
      });
      setFeedback(fbKey, {
        kind: 'multi',
        action: 'save',
        submitStatus: 'failed_to_queue',
        detail: `${selectedInstId}:${errMsg}`,
        at: Date.now(),
      });
    } finally {
      void queryClient.invalidateQueries({ queryKey: notificationKeys.all });
    }
  }, [
    deviceId,
    fbKey,
    formEdits,
    formErrors,
    selectorGroup.titleZh,
    queryClient,
    schemaByPath,
    selectedInstId,
    setFeedback,
    updateMutation,
  ]);

  /** 子表（Neighbor4G / Neighbor2G / Trx）行级 AddObject。 */
  const handleSubAdd = useCallback(
    async (subObject: string) => {
      if (!selectedInstId) return;
      const subPath = `${objectPath}${selectedInstId}.${subObject}.`;
      try {
        const result = await addMutation.mutateAsync({ deviceId, objectPath: subPath });
        message.success({ content: `已下发 AddObject(${subObject})`, duration: 4 });
        setFeedback(fbKey, {
          kind: 'multi',
          action: 'add',
          submitStatus: 'queued',
          taskId: result.taskId,
          detail: subObject,
          at: Date.now(),
        });
        void refetch();
      } catch (err) {
        const errMsg = err instanceof Error ? err.message : String(err);
        notification.error({
          message: `${subObject} 新增失败`,
          description: errMsg,
          duration: ERROR_FEEDBACK_DURATION_SECONDS,
        });
        setFeedback(fbKey, {
          kind: 'multi',
          action: 'add',
          submitStatus: 'failed_to_queue',
          detail: `${subObject}:${errMsg}`,
          at: Date.now(),
        });
      } finally {
        void queryClient.invalidateQueries({ queryKey: notificationKeys.all });
      }
    },
    [addMutation, deviceId, fbKey, objectPath, queryClient, refetch, selectedInstId, setFeedback],
  );

  /** 子表行级 DeleteObject。 */
  const handleSubDelete = useCallback(
    async (subObject: string, rowIdx: number) => {
      if (!selectedInstId) return;
      const subRowPath = `${objectPath}${selectedInstId}.${subObject}.${rowIdx}.`;
      try {
        const result = await deleteMutation.mutateAsync({ deviceId, objectPath: subRowPath });
        message.success({
          content: `已下发 DeleteObject(${subObject}.${rowIdx})`,
          duration: 4,
        });
        setFeedback(fbKey, {
          kind: 'multi',
          action: 'delete',
          submitStatus: 'queued',
          taskId: result.taskId,
          detail: `${subObject}.${rowIdx}`,
          at: Date.now(),
        });
        void refetch();
      } catch (err) {
        const errMsg = err instanceof Error ? err.message : String(err);
        notification.error({
          message: `${subObject} 删除失败`,
          description: errMsg,
          duration: ERROR_FEEDBACK_DURATION_SECONDS,
        });
        setFeedback(fbKey, {
          kind: 'multi',
          action: 'delete',
          submitStatus: 'failed_to_queue',
          detail: `${subObject}.${rowIdx}:${errMsg}`,
          at: Date.now(),
        });
      } finally {
        void queryClient.invalidateQueries({ queryKey: notificationKeys.all });
      }
    },
    [deleteMutation, deviceId, fbKey, objectPath, queryClient, refetch, selectedInstId, setFeedback],
  );

  const title = locale === 'zh-CN' ? selectorGroup.titleZh : selectorGroup.titleEn;
  const maxInstances = selectorGroup.maxInstances && selectorGroup.maxInstances > 0 ? selectorGroup.maxInstances : undefined;
  const reachedMax = maxInstances !== undefined && instanceIds.length >= maxInstances;
  const cardTitle = maxInstances !== undefined
    ? `${title}（${instanceIds.length}/${maxInstances}）`
    : title;

  const addBtn = (
    <Button
      icon={<PlusOutlined />}
      onClick={openAddModal}
      disabled={!canAdd || reachedMax || addMutation.isPending || updateMutation.isPending}
    >
      新 增
    </Button>
  );

  return (
    <Card
      title={cardTitle}
      size="small"
      style={{ marginBottom: 16 }}
      extra={
        lastAction
          ? (() => {
              const spec = statusTagSpec(lastAction, lastTask?.status);
              return (
                <Tag icon={spec.icon} color={spec.color}>
                  {spec.label} · {lastAction.detail} · {formatTime(lastAction.at)}
                </Tag>
              );
            })()
          : null
      }
    >
      <Space style={{ marginBottom: 16 }} wrap>
        <Text strong>{locale === 'zh-CN' ? '选择实例:' : 'Select Instance:'}</Text>
        <Select
          value={selectedInstId ?? undefined}
          onChange={(v) => setSelectedInstId(String(v))}
          style={{ width: 240 }}
          loading={isLoading}
          disabled={isLoading || instanceIds.length === 0}
          placeholder={isLoading ? '加载中...' : instanceIds.length === 0 ? '暂无实例,请新增' : '请选择实例'}
          options={instanceIds.map((id) => {
            const unitId = schemaByPath.get(`${objectPath}${id}.IpaUnitId`)?.currentValue;
            const label = unitId ? `Index:${id}  IpaUnitId:${unitId}` : `Index:${id}`;
            return { value: id, label };
          })}
          notFoundContent="暂无实例"
        />
        {reachedMax ? (
          <Tooltip title={`已达上限 ${maxInstances}，如需新增请先删除其它实例`}>
            <span style={{ display: 'inline-block', cursor: 'not-allowed' }}>{addBtn}</span>
          </Tooltip>
        ) : (
          addBtn
        )}
        <Popconfirm
          title={`确认删除实例 ${selectedInstId ?? ''}？`}
          onConfirm={() => void handleDelete()}
          disabled={!canDelete || !selectedInstId || deleteMutation.isPending}
        >
          <Button
            danger
            icon={<DeleteOutlined />}
            disabled={!canDelete || !selectedInstId || deleteMutation.isPending}
            loading={deleteMutation.isPending}
          >
            删 除
          </Button>
        </Popconfirm>
        <Button
          type="primary"
          icon={isFetching ? <LoadingOutlined spin /> : <SyncOutlined />}
          onClick={() => void refetch()}
          loading={isFetching}
        >
          获取信息
        </Button>
        {isFetching ? (
          <Tag icon={<SyncOutlined spin />} color="processing">
            正在更新…
          </Tag>
        ) : lastSyncedAt ? (
          <Tag
            icon={<CheckCircleOutlined />}
            color={justRefreshed ? 'success' : 'default'}
            style={{
              transition: 'all 0.4s ease',
              boxShadow: justRefreshed ? '0 0 0 3px rgba(82,196,26,0.25)' : 'none',
            }}
          >
            已更新 {formatTime(lastSyncedAt.getTime())}
          </Tag>
        ) : null}
      </Space>

      {isLoading ? (
        <div style={{ textAlign: 'center', padding: 24 }}>
          <Spin />
        </div>
      ) : !selectedInstId ? (
        <div style={{ color: '#8c8c8c', textAlign: 'center', padding: 24 }}>
          {instanceIds.length === 0 ? '当前没有实例，点击"新增"创建' : '请在上方选择一个实例'}
        </div>
      ) : (
        <Spin spinning={isFetching} tip="正在更新…" delay={150}>
          <Collapse
            defaultActiveKey={childGroups.map((g) => g.id)}
            items={childGroups.map((cg) => {
              const label = (
                <strong>{locale === 'zh-CN' ? cg.titleZh : cg.titleEn}</strong>
              );
              if (cg.style === 'subtable') {
                const subObject = extractSubObject(cg.objectPath || '', selectorGroup.objectPath || '');
                const def: BscSubTableDef = {
                  subObject,
                  columns: cg.params.map((p) => ({
                    key: p.leaf || p.name,
                    title: locale === 'zh-CN' ? p.titleZh : p.titleEn,
                  })),
                };
                return {
                  key: cg.id,
                  label,
                  children: renderSubTable(
                    def,
                    selectedInstId,
                    objectPath,
                    subInstanceIds(subObject),
                    fieldValueByPath,
                    setFieldValueByPath,
                    () => void handleSubAdd(subObject),
                    (rowIdx) => void handleSubDelete(subObject, rowIdx),
                    addMutation.isPending,
                    deleteMutation.isPending,
                    getWritableByPath,
                  ),
                };
              }
              // 默认按 form 渲染（form 或未指定 style）
              const fields: BscFieldDef[] = cg.params.map((p) => paramToFieldDef(p, locale));
              return {
                key: cg.id,
                label,
                children: renderFieldGrid(
                  fields,
                  selectedInstId,
                  objectPath,
                  fieldValueByPath,
                  setFieldValueByPath,
                  formErrors,
                  getWritableByPath,
                ),
              };
            })}
          />
          <div style={{ marginTop: 16, textAlign: 'right' }}>
            <Button
              type="primary"
              icon={<SaveOutlined />}
              onClick={() => void handleSave()}
              loading={updateMutation.isPending}
              disabled={Object.keys(formEdits).length === 0}
            >
              保 存
            </Button>
          </div>
        </Spin>
      )}

      <Modal
        title={`${title} · 新增实例`}
        open={Boolean(addModal)}
        onOk={() => void handleConfirmAdd()}
        onCancel={closeAddModal}
        okText="确认新增"
        cancelText="取消"
        confirmLoading={addMutation.isPending || updateMutation.isPending}
        width={760}
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
          {addModal &&
            selectorGroup.params.map((param) => {
              const leaf = param.leaf || '';
              if (!leaf || !groupParamLeafSet.has(leaf)) return null;
              const item = leafSchemaByLeaf.get(leaf);
              const enumMeta = getEffectiveEnumMeta(item?.constraints, item?.path);
              const value = addModal.values[leaf] ?? '';
              const err = addModal.errors[leaf] || '';
              const label = locale === 'zh-CN' ? param.titleZh : param.titleEn;
              const hint = formatConstraintHint(item);

              return (
                <div key={leaf} style={{ minWidth: 0 }}>
                  <div style={{ marginBottom: 6, fontWeight: 500 }}>
                    <Space size={4}>
                      <span>{label}</span>
                      {hint && (
                        <Text type="secondary" style={{ fontSize: 12 }}>
                          {hint}
                        </Text>
                      )}
                    </Space>
                  </div>
                  {enumMeta && enumMeta.values.length > 0 ? (
                    <Select
                      value={value || undefined}
                      onChange={(next) => setAddModalValue(leaf, String(next))}
                      style={{ width: '100%' }}
                      status={err ? 'error' : undefined}
                      options={enumMeta.values.map((v, idx) => ({
                        value: v,
                        label: enumMeta.labels[idx] ?? v,
                      }))}
                    />
                  ) : (
                    <Input
                      value={value}
                      onChange={(e) => setAddModalValue(leaf, e.target.value)}
                      status={err ? 'error' : undefined}
                    />
                  )}
                  {err && (
                    <div style={{ color: '#ff4d4f', fontSize: 12, marginTop: 4 }}>{err}</div>
                  )}
                </div>
              );
            })}
        </div>
      </Modal>
    </Card>
  );
}
