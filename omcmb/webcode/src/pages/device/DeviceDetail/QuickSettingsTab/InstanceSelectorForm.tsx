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
  EditOutlined,
  LoadingOutlined,
  PlusCircleFilled,
  PlusOutlined,
  SaveOutlined,
  SendOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useT } from '@/hooks/useT';
import {
  useAddObject,
  useDeleteObject,
  useParameterSchema,
  useUpdateParameters,
} from '@core/hooks/api/useDeviceParameters';
import { useDeviceTaskStatus } from '@core/hooks/api/useDeviceTask';
import { notificationKeys } from '@core/hooks/api/useNotificationCenter';
import { deviceTaskApi } from '@core/services/api/deviceTaskApi';
import { deviceParameterApi } from '@core/services/api/deviceParameterApi';
import {
  feedbackKey,
  useQuickSettingsFeedbackStore,
  type MultiFeedback,
} from '@core/store/quickSettingsFeedbackStore';
import type { ParameterSchemaItem, ParameterUpdateRequest } from '@core/types/deviceParameter';
import { quickSettingsRebootNotice } from './rebootNotice';
import { isDeviceTaskTerminal, type DeviceTaskStatus } from '@core/types/deviceTask';
import type { QuickSettingsGroup, QuickSettingsParam } from '@core/types/quicksettings';
import {
  applyInstanceContext,
  getEffectiveEnumMeta,
  localizeEnumLabel,
  parseQuickSettingsMultiCheckboxValue,
  serializeQuickSettingsMultiCheckboxValue,
  validateValue,
  type QuickSettingsInstanceContext,
} from './validators';
import MultiInstanceTable from './MultiInstanceTable';
import { rowsWithNestedInstances } from './subTableExpansion';
import {
  parameterReadbackValuesMatch,
  waitForExpectedParameterValues,
  waitForReadback,
} from './parameterReadback';

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
  columns: { key: string; title: string; options?: { value: string; label: string }[] }[];
}

function normalizeComparableFieldValue(value: unknown, param?: QuickSettingsParam): string {
  if (param?.type === 'multiCheckbox') {
    return serializeQuickSettingsMultiCheckboxValue(value);
  }
  return String(value ?? '');
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
  active?: boolean;
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

function statusTagSpec(action: MultiFeedback, taskStatus: DeviceTaskStatus | undefined, locale: 'zh-CN' | 'en-US'): StatusTagSpec {
  const actionLabel = action.action === 'save'
    ? (locale === 'zh-CN' ? '保存' : 'Save')
    : action.action === 'add'
      ? (locale === 'zh-CN' ? '新增' : 'Add')
      : (locale === 'zh-CN' ? '删除' : 'Delete');
  if (action.submitStatus === 'failed_to_queue') {
    return { color: 'error', icon: <CloseCircleOutlined />, label: locale === 'zh-CN' ? `${actionLabel}入队失败` : `${actionLabel} queue failed` };
  }
  if (!action.taskId) {
    return { color: 'processing', icon: <SyncOutlined spin />, label: locale === 'zh-CN' ? `${actionLabel}已入队` : `${actionLabel} queued` };
  }
  switch (taskStatus) {
    case 'completed':
      return { color: 'success', icon: <CheckCircleOutlined />, label: locale === 'zh-CN' ? `${actionLabel}成功` : `${actionLabel} succeeded` };
    case 'failed':
      return { color: 'error', icon: <CloseCircleOutlined />, label: locale === 'zh-CN' ? `${actionLabel}基站应答失败` : `${actionLabel} device response failed` };
    case 'expired':
      return { color: 'warning', icon: <ClockCircleOutlined />, label: locale === 'zh-CN' ? `${actionLabel}超时` : `${actionLabel} timed out` };
    case 'cancelled':
      return { color: 'default', icon: <CloseCircleOutlined />, label: locale === 'zh-CN' ? `${actionLabel}已取消` : `${actionLabel} cancelled` };
    case 'sent':
      return { color: 'processing', icon: <SendOutlined />, label: locale === 'zh-CN' ? `${actionLabel}已发送给基站` : `${actionLabel} sent to device` };
    case 'pending':
    default:
      return { color: 'processing', icon: <SyncOutlined spin />, label: locale === 'zh-CN' ? `${actionLabel}已入队,等待下发` : `${actionLabel} queued, waiting to send` };
  }
}

function formatConstraintHint(schema: ParameterSchemaItem | undefined, locale: 'zh-CN' | 'en-US'): string {
  if (!schema?.constraints) return '';
  const c = schema.constraints;
  if (c.enumValues && c.enumValues.length > 0) return '';
  const isString = schema.type === 'string';
  const min = c.minLength ?? c.minValue;
  const max = c.maxLength ?? c.maxValue;
  if (min !== undefined || max !== undefined) {
    const lo = min ?? '-∞';
    const hi = max ?? '∞';
    return isString ? (locale === 'zh-CN' ? `[长度 ${lo} ~ ${hi}]` : `[length ${lo} ~ ${hi}]`) : `[${lo} ~ ${hi}]`;
  }
  return '';
}

/** 按 BscFieldDef 做最小校验：required / int 范围。返回 '' 表示通过。 */
function validateFieldDef(def: BscFieldDef, value: string, locale: 'zh-CN' | 'en-US'): string {
  if (def.readonly) return '';
  if (def.required && value === '') return locale === 'zh-CN' ? '不能为空' : 'Required';
  if (value === '') return '';
  if (def.type === 'int') {
    const n = Number(value);
    if (!Number.isInteger(n)) return locale === 'zh-CN' ? '需要整数' : 'Integer required';
    if (def.minValue !== undefined && n < def.minValue) return locale === 'zh-CN' ? `不能小于 ${def.minValue}` : `Must be at least ${def.minValue}`;
    if (def.maxValue !== undefined && n > def.maxValue) return locale === 'zh-CN' ? `不能大于 ${def.maxValue}` : `Must be at most ${def.maxValue}`;
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
  locale: 'zh-CN' | 'en-US',
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
        const validator = (v: string) => validateFieldDef(def, v, locale);
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
                <span style={{ color: '#8c8c8c', fontSize: 12, marginLeft: 6 }}>{locale === 'zh-CN' ? '(只读)' : '(read-only)'}</span>
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
          const displayValue = rawValue !== '' && rawValue != null ? rawValue : (locale === 'zh-CN' ? '未上报' : 'Not reported');
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
                options={(def.enumOptions ?? []).map((option) => ({
                  ...option,
                  label: localizeEnumLabel(option.label, option.value, locale),
                }))}
                allowClear
              />
              {hintNode}
              {errNode}
            </div>
          );
        }

        if (def.type === 'multiCheckbox') {
          const selected = parseQuickSettingsMultiCheckboxValue(rawValue);
          return (
            <div key={def.leaf} style={{ minWidth: 0, gridColumn: '1 / -1' }}>
              {labelNode}
              <Checkbox.Group
                value={selected}
                options={(def.checkboxOptions ?? []).map((v) => ({ value: v, label: v }))}
                onChange={(vals) => setValue(path, serializeQuickSettingsMultiCheckboxValue(vals), validator)}
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
  locale: 'zh-CN' | 'en-US',
  listAddStyle = false,
  onSave?: () => void,
  savePending = false,
  expandedRowRender?: (row: { id: number }) => React.ReactNode,
  onEdit?: (rowIdx: number) => void,
  defaultExpandedRowKeys: number[] = [],
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
        if (col.options && col.options.length > 0) {
          return <Select value={getValue(path) || undefined} options={col.options} onChange={(value) => setValue(path, String(value))} size="small" style={{ width: '100%' }} disabled={!writable} />;
        }
        const nestedPrefix = col.key.includes('.') ? `${col.key.slice(0, col.key.lastIndexOf('.') + 1)}` : '';
        const addressingTypePath = `${objectPath}${selectedInstId}.${def.subObject}.${row.id}.${nestedPrefix}AddressingType`;
        const leafName = col.key.split('.').at(-1) || col.key;
        const isStaticOnlyField = ['IPAddress', 'SubnetMask', 'DefaultGateway'].includes(leafName);
        const disabledByDhcp = isStaticOnlyField && getValue(addressingTypePath) === 'DHCP';
        return <Input value={getValue(path)} onChange={(e) => setValue(path, e.target.value)} size="small" disabled={!writable || disabledByDhcp} />;
      },
    })),
    {
      title: locale === 'zh-CN' ? '操作' : 'Operate',
      key: '_op',
      width: onEdit ? 150 : 110,
      render: (_: unknown, row: { id: number }) => (
        <Space size={4}>
          {onEdit && (
            <Button
              type="link"
              size="small"
              icon={<EditOutlined />}
              title={locale === 'zh-CN' ? '修改' : 'Edit'}
              onClick={() => onEdit(row.id)}
              disabled={deletePending}
            >{locale === 'zh-CN' ? '修改' : 'Edit'}</Button>
          )}
          <Popconfirm
            title={locale === 'zh-CN' ? `确认删除 ${def.subObject}.${row.id} ？` : `Delete ${def.subObject}.${row.id}?`}
            onConfirm={() => onDelete(row.id)}
            disabled={deletePending}
          >
            <Button
              type="link"
              size="small"
              danger
              icon={<DeleteOutlined />}
              title={locale === 'zh-CN' ? '删除' : 'Delete'}
              loading={deletePending}
            >{locale === 'zh-CN' ? '删除' : 'Delete'}</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];
  return (
    <div
      style={{
        position: 'relative',
        marginBottom: listAddStyle ? 4 : 0,
        border: listAddStyle ? '1px solid var(--color-border, #d7dce5)' : undefined,
        borderRadius: 0,
        boxShadow: 'none',
        background: listAddStyle ? 'var(--color-bg-container, #fff)' : undefined,
        padding: listAddStyle ? '4px 4px 0' : undefined,
      }}
    >
      <Space size={8} style={{ position: 'absolute', top: -36, right: 8, zIndex: 1 }}>
        {onSave && (
          <Button
            type="primary"
            size="small"
            icon={<SaveOutlined />}
            onClick={onSave}
            loading={savePending}
          >{locale === 'zh-CN' ? '保存' : 'Save'}</Button>
        )}
        <Button
          type="primary"
          size="small"
          shape={listAddStyle ? undefined : 'circle'}
          icon={listAddStyle ? <PlusOutlined /> : <PlusCircleFilled />}
          onClick={onAdd}
          loading={addPending}
        >{listAddStyle ? (locale === 'zh-CN' ? '新增' : 'Add') : undefined}</Button>
      </Space>
      <Table
        size="small"
        pagination={false}
        rowKey="key"
        dataSource={dataSource}
        columns={columns}
        expandable={expandedRowRender ? { expandedRowRender, defaultExpandedRowKeys } : undefined}
        locale={{ emptyText: locale === 'zh-CN' ? '暂无数据' : 'No data' }}
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
  active = true,
  selectorGroup,
  childGroups,
  instanceContext,
  locale,
}: InstanceSelectorFormProps) {
  const t = useT();
  const objectPath = useMemo(() => {
    const resolved = applyInstanceContext(selectorGroup.objectPath || '', instanceContext, {
      preserveTrailingInstance: true,
    });
    return resolved.replace(/\{i\}\.$/, '');
  }, [selectorGroup, instanceContext]);

  const { data: schemaResp, isLoading, isFetching, refetch } = useParameterSchema(deviceId, objectPath, active);
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
  const wanInstanceId = useMemo(
    () => instanceIds.find((id) => schemaByPath.get(`${objectPath}${id}.interfaceType`)?.currentValue?.toLowerCase() === 'wan'),
    [instanceIds, objectPath, schemaByPath],
  );
  const lanInstanceId = useMemo(
    () => instanceIds.find((id) => schemaByPath.get(`${objectPath}${id}.interfaceType`)?.currentValue?.toLowerCase() === 'lan'),
    [instanceIds, objectPath, schemaByPath],
  );

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

  const paramByLeaf = useMemo(() => {
    const map = new Map<string, QuickSettingsParam>();
    for (const group of [selectorGroup, ...childGroups]) {
      for (const param of group.params) {
        const keys = [param.leaf, param.name].filter(Boolean) as string[];
        for (const key of keys) {
          if (!map.has(key)) {
            map.set(key, param);
          }
        }
      }
    }
    return map;
  }, [childGroups, selectorGroup]);

  const [selectedInstId, setSelectedInstId] = useState<string | null>(null);

  // schema 加载完成后优先选中 WAN；用户手动切换到其它物理口后保持其选择。
  useEffect(() => {
    if (instanceIds.length === 0) {
      if (selectedInstId !== null) setSelectedInstId(null);
      return;
    }
    if (!selectedInstId || !instanceIds.includes(selectedInstId)) {
      setSelectedInstId(wanInstanceId ?? instanceIds[0]);
    }
  }, [instanceIds, selectedInstId, wanInstanceId]);

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

  const { data: lastTask } = useDeviceTaskStatus(active ? lastAction?.taskId : undefined);

  // 任务进入终态 → 只刷新当前多实例的 schema(精确到 objectPath 该一份查询),
  // 并清掉乐观编辑值（保留 selectedInstId）。不作跨 device 的全量 invalidate，
  // 避免领居/顶层 selector 等无关查询被动重拉。
  useEffect(() => {
    if (!active) return;
    if (!lastTask || !isDeviceTaskTerminal(lastTask.status)) return;
    if (lastAction?.syncedForTaskId === lastTask.id) return;
    let cancelled = false;
    const abortController = new AbortController();
    void (async () => {
      try {
        const expected = new Map(Object.entries(lastAction?.expectedReadback ?? {}));
        if (lastTask.status === 'completed' && expected.size > 0) {
          await waitForExpectedParameterValues({
            expected,
            read: async () => {
              deviceParameterApi.invalidateParameterSchemaCache(deviceId, objectPath);
              const refreshed = await refetch();
              return new Map(
                (refreshed.data?.parameters ?? []).map((item) => [
                  item.path,
                  String(item.currentValue ?? ''),
                ]),
              );
            },
            intervalMs: 500,
            timeoutMs: 30_000,
            signal: abortController.signal,
          });
        } else if (
          lastTask.status === 'completed'
          && lastAction?.readbackObjectPath
          && (lastAction.action === 'add' || lastAction.action === 'delete')
        ) {
          const readbackObjectPath = lastAction.readbackObjectPath;
          const beforeInstances = new Set(lastAction.readbackInstancesBefore ?? []);
          const deletedInstances = new Set(lastAction.deletedInstIds ?? []);
          await waitForReadback({
            read: async () => {
              deviceParameterApi.invalidateParameterSchemaCache(deviceId, objectPath);
              return (await refetch()).data;
            },
            matches: (data) => {
              const actualInstances = data?.objects
                .find((item) => item.path === readbackObjectPath)
                ?.currentInstances.map(String) ?? [];
              if (lastAction.action === 'delete') {
                return [...deletedInstances].every((instanceId) => !actualInstances.includes(instanceId));
              }
              return actualInstances.some((instanceId) => !beforeInstances.has(instanceId));
            },
            intervalMs: 500,
            timeoutMs: 30_000,
            signal: abortController.signal,
          });
        } else {
          deviceParameterApi.invalidateParameterSchemaCache(deviceId, objectPath);
          await refetch();
        }
      } catch (err) {
        if (err instanceof Error && err.name === 'AbortError') return;
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
        if (lastTask.status === 'completed') {
          const expected = lastAction.expectedReadback ?? {};
          setFormEdits((previous) => {
            const next = { ...previous };
            for (const [path, expectedValue] of Object.entries(expected)) {
              const leaf = path.split('.').pop() ?? '';
              const param = paramByLeaf.get(leaf);
              const currentEdit = next[path];
              if (
                currentEdit !== undefined
                && parameterReadbackValuesMatch(
                  expectedValue,
                  normalizeComparableFieldValue(currentEdit, param),
                )
              ) {
                delete next[path];
              }
            }
            return next;
          });
        } else {
          setFormEdits({});
        }
        setFormErrors({});
      }
      patchFeedback(fbKey, { syncedForTaskId: lastTask.id });
    })();
    return () => {
      cancelled = true;
      abortController.abort();
    };
  }, [active, deviceId, fbKey, lastAction, lastTask, objectPath, paramByLeaf, patchFeedback, refetch, selectedInstId, selectorGroup.titleZh]);

  const awaitingSubmittedReadback = Boolean(
    lastAction?.taskId
    && lastAction.syncedForTaskId !== lastAction.taskId
    && lastAction.submitStatus === 'queued',
  );

  // 基站应答失败 → notification（只首次提示）
  useEffect(() => {
    if (
      lastTask &&
      lastTask.status === 'failed' &&
      lastAction &&
      lastAction.notifiedFailedTaskId !== lastTask.id
    ) {
      notification.error({
        message: locale === 'zh-CN'
          ? `基站应答失败(${selectorGroup.titleZh})`
          : `Device response failed(${selectorGroup.titleEn})`,
        description: lastTask.errorMessage || (locale === 'zh-CN' ? '未知错误,可在通知中心查看任务详情' : 'Unknown error. Check Notification Center for task details.'),
        duration: ERROR_FEEDBACK_DURATION_SECONDS,
      });
      patchFeedback(fbKey, { notifiedFailedTaskId: lastTask.id });
    }
  }, [lastTask, lastAction, selectorGroup.titleZh, selectorGroup.titleEn, patchFeedback, fbKey, locale]);

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

  const subInstanceIdsFor = useCallback(
    (interfaceId: string, subObject: string): number[] => {
      const prefix = `${objectPath}${interfaceId}.${subObject}.`;
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
    [objectPath, schemaByPath],
  );

  const waitForTaskTerminal = useCallback(async (taskId: string) => {
    const timeoutAt = Date.now() + 60000;
    while (Date.now() < timeoutAt) {
      const task = await deviceTaskApi.getTask(taskId);
      if (isDeviceTaskTerminal(task.status)) return task;
      await new Promise((resolve) => window.setTimeout(resolve, 1000));
    }
    throw new Error(locale === 'zh-CN' ? '等待任务完成超时' : 'Timed out waiting for task completion');
  }, [locale]);

  // 新增实例：用 Modal 收集新值 → AddObject → 等待 → 取新实例号 → SetParameterValues
  type AddModalState = { values: Record<string, string>; errors: Record<string, string> } | null;
  const [addModal, setAddModal] = useState<AddModalState>(null);
  type NrWanAddState = {
    enable: '0' | '1';
    vlanName: string;
    vlanId: string;
  };
  const [nrWanAdd, setNrWanAdd] = useState<NrWanAddState | null>(null);
  type VlanEditState = NrWanAddState & { instanceId: string };
  const [vlanEdit, setVlanEdit] = useState<VlanEditState | null>(null);

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
      message.error({ content: locale === 'zh-CN' ? '校验失败,请修正后再保存' : 'Validation failed. Please fix the fields and save again.', duration: ERROR_FEEDBACK_DURATION_SECONDS });
      return;
    }

    let newInstanceId: string | undefined;
    let addRebootTarget: number | undefined;
    try {
      const addResult = await addMutation.mutateAsync({ deviceId, objectPath });
      if (addResult.rebootRequired) addRebootTarget = addResult.rebootTarget;
      const addTask = await waitForTaskTerminal(addResult.taskId);
      if (addTask.status !== 'completed') {
        throw new Error(addTask.errorMessage || (locale === 'zh-CN' ? `新增实例失败(${addTask.status})` : `Add instance failed(${addTask.status})`));
      }
      const knownInstances = new Set(instanceIds);
      const refreshedData = await waitForReadback({
        read: async () => {
          deviceParameterApi.invalidateParameterSchemaCache(deviceId, objectPath);
          return (await refetch()).data;
        },
        matches: (data) => {
          const refreshedObject = data?.objects.find((item) => item.path === objectPath);
          return refreshedObject?.currentInstances
            .map(String)
            .some((id) => !knownInstances.has(id)) ?? false;
        },
        intervalMs: 500,
        timeoutMs: 30_000,
      });
      const nextObject = refreshedData?.objects.find((o) => o.path === objectPath);
      newInstanceId = nextObject?.currentInstances
        .map((n) => String(n))
        .find((id) => !knownInstances.has(id));
      if (!newInstanceId) throw new Error(locale === 'zh-CN' ? '新增实例成功,但未能识别新实例号' : 'Instance added, but the new instance ID could not be identified');
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : String(err);
      notification.error({
        message: locale === 'zh-CN' ? `新增失败(${selectorGroup.titleZh})` : `Add failed(${selectorGroup.titleEn})`,
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
      if (addRebootTarget !== undefined) {
        message.warning({ content: quickSettingsRebootNotice(t, addRebootTarget), duration: 8 });
      }
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
        expectedReadback: Object.fromEntries(
          updates.map((update) => [update.parameterPath, update.parameterValue]),
        ),
        detail: `新增实例 ${newInstanceId} ${updates.length} 项`,
        at: Date.now(),
      });
      message.success({
        content: `已新增实例 ${newInstanceId},并下发 ${updates.length} 项变更`,
        duration: 6,
      });
      if (result.rebootRequired && result.rebootTarget) {
        message.warning({ content: quickSettingsRebootNotice(t, result.rebootTarget), duration: 8 });
      }
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
    t,
    updateMutation,
    waitForTaskTerminal,
  ]);

  const handleDelete = useCallback(async (targetInstanceId?: string) => {
    const instanceId = targetInstanceId ?? selectedInstId;
    if (!instanceId) return;
    try {
      const result = await deleteMutation.mutateAsync({
        deviceId,
        objectPath: `${objectPath}${instanceId}.`,
      });
      message.success({
        content: `已下发 DeleteObject(${instanceId}),请在右上角铃铛查看任务结果`,
        duration: 6,
      });
      if (result.rebootRequired && result.rebootTarget) {
        message.warning({ content: quickSettingsRebootNotice(t, result.rebootTarget), duration: 8 });
      }
      setFeedback(fbKey, {
        kind: 'multi',
        action: 'delete',
        submitStatus: 'queued',
        taskId: result.taskId,
        deletedInstIds: [instanceId],
        readbackObjectPath: objectPath,
        detail: `实例 ${instanceId}`,
        at: Date.now(),
      });
      // 删后让 useEffect 重置 selectedInstId
      setSelectedInstId((prev) => (prev === instanceId ? null : prev));
      void refetch();
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : String(err);
      notification.error({
        message: `DeleteObject 入队失败(${selectorGroup.titleZh})`,
        description: `实例 ${instanceId} 删除失败:${errMsg}`,
        duration: ERROR_FEEDBACK_DURATION_SECONDS,
      });
      setFeedback(fbKey, {
        kind: 'multi',
        action: 'delete',
        submitStatus: 'failed_to_queue',
        detail: `实例 ${instanceId}:${errMsg}`,
        at: Date.now(),
      });
    } finally {
      void queryClient.invalidateQueries({ queryKey: notificationKeys.all });
    }
  }, [deleteMutation, deviceId, fbKey, selectorGroup.titleZh, objectPath, queryClient, refetch, selectedInstId, setFeedback, t]);

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
      const leaf = path.split('.').pop() ?? '';
      const param = paramByLeaf.get(leaf);
      const normalizedValue = normalizeComparableFieldValue(value, param);
      const oldVal = normalizeComparableFieldValue(item?.currentValue ?? '', param);
      if (normalizedValue === oldVal) continue;
      updates.push({
        parameterPath: path,
        parameterValue: normalizedValue,
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
        expectedReadback: Object.fromEntries(
          updates.map((update) => [update.parameterPath, update.parameterValue]),
        ),
        detail: `实例 ${selectedInstId} ${updates.length} 项`,
        at: Date.now(),
      });
      message.success({
        content: `已下发 ${updates.length} 项变更,正在等待基站应答(Tag 会自动刷新)`,
        duration: 6,
      });
      if (result.rebootRequired && result.rebootTarget) {
        message.warning({ content: quickSettingsRebootNotice(t, result.rebootTarget), duration: 8 });
      }
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
    paramByLeaf,
    selectedInstId,
    setFeedback,
    t,
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
        if (result.rebootRequired && result.rebootTarget) {
          message.warning({ content: quickSettingsRebootNotice(t, result.rebootTarget), duration: 8 });
        }
        setFeedback(fbKey, {
          kind: 'multi',
          action: 'add',
          submitStatus: 'queued',
          taskId: result.taskId,
          readbackObjectPath: subPath,
          readbackInstancesBefore: subInstanceIds(subObject).map(String),
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
    [addMutation, deviceId, fbKey, objectPath, queryClient, refetch, selectedInstId, setFeedback, subInstanceIds, t],
  );

  const handleNrWanAdd = useCallback(async () => {
    if (!selectedInstId || !nrWanAdd) return;
    const vlanId = Number(nrWanAdd.vlanId);
    if (!nrWanAdd.vlanName || nrWanAdd.vlanName.length > 13 || !Number.isInteger(vlanId) || vlanId < 2 || vlanId > 4094) {
      message.error(locale === 'zh-CN' ? 'VLAN Name 长度需为 1~13，VLAN ID 需为 2~4094 的整数' : 'VLAN Name must be 1-13 characters and VLAN ID an integer from 2 to 4094');
      return;
    }
    const vlanBase = `${objectPath}${selectedInstId}.VlanInterface.`;
    const knownVlanIds = new Set(subInstanceIds('VlanInterface').map(String));
    try {
      const vlanResult = await addMutation.mutateAsync({ deviceId, objectPath: vlanBase });
      const vlanTask = await waitForTaskTerminal(vlanResult.taskId);
      if (vlanTask.status !== 'completed') throw new Error(vlanTask.errorMessage || `AddObject VlanInterface ${vlanTask.status}`);

      const vlanSchema = await waitForReadback({
        read: async () => {
          deviceParameterApi.invalidateParameterSchemaCache(deviceId, objectPath);
          return (await refetch()).data;
        },
        matches: (data) => {
          const refreshedObject = data?.objects.find((item) => item.path === vlanBase);
          return refreshedObject?.currentInstances
            .map(String)
            .some((id) => !knownVlanIds.has(id)) ?? false;
        },
        intervalMs: 500,
        timeoutMs: 30_000,
      });
      const vlanObject = vlanSchema?.objects.find((item) => item.path === vlanBase);
      const vlanInstance = vlanObject?.currentInstances.map(String).find((id) => !knownVlanIds.has(id));
      if (!vlanInstance) throw new Error(locale === 'zh-CN' ? '未能识别新增 VLAN 对象编号' : 'Unable to identify the new VLAN object');

      const parameters: ParameterUpdateRequest[] = [
        { parameterPath: `${vlanBase}${vlanInstance}.Name`, parameterValue: nrWanAdd.vlanName, parameterType: 'string' },
        { parameterPath: `${vlanBase}${vlanInstance}.Id`, parameterValue: nrWanAdd.vlanId, parameterType: 'unsignedInt' },
        { parameterPath: `${vlanBase}${vlanInstance}.Enable`, parameterValue: nrWanAdd.enable, parameterType: 'boolean' },
      ];
      const updateResult = await updateMutation.mutateAsync({ deviceId, parameters });
      setFormEdits((prev) => ({
        ...prev,
        ...Object.fromEntries(parameters.map((parameter) => [parameter.parameterPath, parameter.parameterValue])),
      }));
      setFeedback(fbKey, {
        kind: 'multi', action: 'save', submitStatus: 'queued', taskId: updateResult.taskId,
        savedInstId: selectedInstId,
        expectedReadback: Object.fromEntries(
          parameters.map((parameter) => [parameter.parameterPath, parameter.parameterValue]),
        ),
        detail: `VLAN ${vlanInstance}`, at: Date.now(),
      });
      message.success({ content: locale === 'zh-CN' ? 'WAN/VLAN 对象已创建并下发' : 'WAN/VLAN object created and queued', duration: 6 });
            if (updateResult.rebootRequired && updateResult.rebootTarget) {
              message.warning({ content: quickSettingsRebootNotice(t, updateResult.rebootTarget), duration: 8 });
            }
      setNrWanAdd(null);
      void refetch();
    } catch (err) {
      notification.error({
        message: locale === 'zh-CN' ? '新增 WAN/VLAN 失败' : 'Failed to add WAN/VLAN',
        description: err instanceof Error ? err.message : String(err),
        duration: ERROR_FEEDBACK_DURATION_SECONDS,
      });
    }
  }, [addMutation, deviceId, fbKey, locale, nrWanAdd, objectPath, refetch, selectedInstId, setFeedback, subInstanceIds, t, updateMutation, waitForTaskTerminal]);

  const openVlanEdit = useCallback((rowId: number) => {
    if (!selectedInstId) return;
    const prefix = `${objectPath}${selectedInstId}.VlanInterface.${rowId}.`;
    const enable = fieldValueByPath(`${prefix}Enable`).toLowerCase();
    setVlanEdit({
      instanceId: String(rowId),
      enable: ['1', 'true', 'on'].includes(enable) ? '1' : '0',
      vlanName: fieldValueByPath(`${prefix}Name`),
      vlanId: fieldValueByPath(`${prefix}Id`),
    });
  }, [fieldValueByPath, objectPath, selectedInstId]);

  const handleVlanEdit = useCallback(async () => {
    if (!selectedInstId || !vlanEdit) return;
    const vlanId = Number(vlanEdit.vlanId);
    if (!vlanEdit.vlanName || vlanEdit.vlanName.length > 13 || !Number.isInteger(vlanId) || vlanId < 2 || vlanId > 4094) {
      message.error(locale === 'zh-CN' ? 'VLAN Name 长度需为 1~13，VLAN ID 需为 2~4094 的整数' : 'VLAN Name must be 1-13 characters and VLAN ID an integer from 2 to 4094');
      return;
    }
    const vlanBase = `${objectPath}${selectedInstId}.VlanInterface.${vlanEdit.instanceId}.`;
    const parameters: ParameterUpdateRequest[] = [
      { parameterPath: `${vlanBase}Name`, parameterValue: vlanEdit.vlanName, parameterType: 'string' },
      { parameterPath: `${vlanBase}Id`, parameterValue: vlanEdit.vlanId, parameterType: 'unsignedInt' },
      { parameterPath: `${vlanBase}Enable`, parameterValue: vlanEdit.enable, parameterType: 'boolean' },
    ];
    try {
      const result = await updateMutation.mutateAsync({ deviceId, parameters });
      setFormEdits((prev) => ({
        ...prev,
        ...Object.fromEntries(parameters.map((parameter) => [parameter.parameterPath, parameter.parameterValue])),
      }));
      setFeedback(fbKey, {
        kind: 'multi', action: 'save', submitStatus: 'queued', taskId: result.taskId,
        savedInstId: selectedInstId,
        expectedReadback: Object.fromEntries(
          parameters.map((parameter) => [parameter.parameterPath, parameter.parameterValue]),
        ),
        detail: `VLAN ${vlanEdit.instanceId}`, at: Date.now(),
      });
      setVlanEdit(null);
      message.success({ content: locale === 'zh-CN' ? 'VLAN 修改已下发' : 'VLAN update queued', duration: 5 });
      if (result.rebootRequired && result.rebootTarget) {
        message.warning({ content: quickSettingsRebootNotice(t, result.rebootTarget), duration: 8 });
      }
      void refetch();
    } catch (err) {
      notification.error({
        message: locale === 'zh-CN' ? '修改 VLAN 失败' : 'Failed to update VLAN',
        description: err instanceof Error ? err.message : String(err),
        duration: ERROR_FEEDBACK_DURATION_SECONDS,
      });
    }
  }, [deviceId, fbKey, locale, objectPath, refetch, selectedInstId, setFeedback, t, updateMutation, vlanEdit]);

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
        if (result.rebootRequired && result.rebootTarget) {
          message.warning({ content: quickSettingsRebootNotice(t, result.rebootTarget), duration: 8 });
        }
        setFeedback(fbKey, {
          kind: 'multi',
          action: 'delete',
          submitStatus: 'queued',
          taskId: result.taskId,
          deletedInstIds: [String(rowIdx)],
          readbackObjectPath: `${objectPath}${selectedInstId}.${subObject}.`,
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
    [deleteMutation, deviceId, fbKey, objectPath, queryClient, refetch, selectedInstId, setFeedback, t],
  );

  const title = locale === 'zh-CN' ? selectorGroup.titleZh : selectorGroup.titleEn;
  const maxInstances = selectorGroup.maxInstances && selectorGroup.maxInstances > 0 ? selectorGroup.maxInstances : undefined;
  const reachedMax = maxInstances !== undefined && instanceIds.length >= maxInstances;
  const isGnbNetworkInterface = selectorGroup.id === 'gnb-network-interface';
  const directChildGroups = childGroups.filter((group) => group.parentSelector === selectorGroup.id);
  const vlanAddressGroups = childGroups.filter((group) => group.parentSelector === 'gnb-interface-vlan');
  const visibleChildGroups = directChildGroups;
  const selectableNetworkInstanceIds = isGnbNetworkInterface
    ? instanceIds.filter((id) => schemaByPath.get(`${objectPath}${id}.Name`)?.currentValue !== 'ETH')
    : instanceIds;
  const selectedInterfaceName = selectedInstId
    ? schemaByPath.get(`${objectPath}${selectedInstId}.Name`)?.currentValue
    : undefined;
  const cardTitle = maxInstances !== undefined
    ? `${title}（${instanceIds.length}/${maxInstances}）`
    : title;

  const addBtn = (
    <Button
      icon={<PlusOutlined />}
      onClick={openAddModal}
      disabled={!canAdd || reachedMax || addMutation.isPending || updateMutation.isPending || awaitingSubmittedReadback}
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
              const visibleTaskStatus = lastTask?.status === 'completed'
                && lastAction.taskId
                && lastAction.syncedForTaskId !== lastTask.id
                ? 'sent'
                : lastTask?.status;
              const spec = statusTagSpec(lastAction, visibleTaskStatus, locale);
              return (
                <Tag icon={spec.icon} color={spec.color}>
                  {spec.label} · {lastAction.detail} · {formatTime(lastAction.at)}
                </Tag>
              );
            })()
          : null
      }
    >
      {!isGnbNetworkInterface && <Space style={{ marginBottom: 16 }} wrap>
        <Text strong>{locale === 'zh-CN' ? '选择实例:' : 'Select Instance:'}</Text>
        <Select
          value={selectedInstId ?? undefined}
          onChange={(v) => setSelectedInstId(String(v))}
          style={{ width: 240 }}
          loading={isLoading}
          disabled={isLoading || instanceIds.length === 0}
          placeholder={isLoading
            ? (locale === 'zh-CN' ? '加载中...' : 'Loading...')
            : instanceIds.length === 0
              ? (locale === 'zh-CN' ? '暂无实例,请新增' : 'No instances. Add one first.')
              : (locale === 'zh-CN' ? '请选择实例' : 'Select an instance')}
          options={instanceIds.map((id) => {
            const unitId = schemaByPath.get(`${objectPath}${id}.IpaUnitId`)?.currentValue;
            const label = unitId ? `Index:${id}  IpaUnitId:${unitId}` : `Index:${id}`;
            return { value: id, label };
          })}
          notFoundContent={locale === 'zh-CN' ? '暂无实例' : 'No instances'}
        />
        {reachedMax ? (
          <Tooltip title={locale === 'zh-CN' ? `已达上限 ${maxInstances}，如需新增请先删除其它实例` : `Limit ${maxInstances} reached. Delete another instance before adding.`}>
            <span style={{ display: 'inline-block', cursor: 'not-allowed' }}>{addBtn}</span>
          </Tooltip>
        ) : (
          addBtn
        )}
        <Popconfirm
          title={locale === 'zh-CN' ? `确认删除实例 ${selectedInstId ?? ''}？` : `Delete instance ${selectedInstId ?? ''}?`}
          onConfirm={() => void handleDelete()}
          disabled={!canDelete || !selectedInstId || deleteMutation.isPending || awaitingSubmittedReadback}
        >
          <Button
            danger
            icon={<DeleteOutlined />}
            disabled={!canDelete || !selectedInstId || deleteMutation.isPending || awaitingSubmittedReadback}
            loading={deleteMutation.isPending}
          >
            {locale === 'zh-CN' ? '删 除' : 'Delete'}
          </Button>
        </Popconfirm>
        <Button
          type="primary"
          icon={isFetching ? <LoadingOutlined spin /> : <SyncOutlined />}
          onClick={() => void refetch()}
          loading={isFetching}
        >
          {locale === 'zh-CN' ? '获取信息' : 'Refresh Info'}
        </Button>
        {isFetching ? (
          <Tag icon={<SyncOutlined spin />} color="processing">
            {locale === 'zh-CN' ? '正在更新…' : 'Updating...'}
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
            {locale === 'zh-CN' ? '已更新' : 'Updated'} {formatTime(lastSyncedAt.getTime())}
          </Tag>
        ) : null}
      </Space>}

      {isGnbNetworkInterface && (
        <>
          <Space style={{ marginBottom: 12 }} wrap>
            <Text strong>{locale === 'zh-CN' ? '接口列表：' : 'Interface List:'}</Text>
            <Select
              value={selectedInstId ?? undefined}
              onChange={(value) => setSelectedInstId(String(value))}
              style={{ width: 240 }}
              options={selectableNetworkInstanceIds.map((id) => ({
                value: id,
                label: schemaByPath.get(`${objectPath}${id}.Name`)?.currentValue || `Interface.${id}`,
              }))}
            />
          </Space>
          <div style={{ marginBottom: 8, fontWeight: 600 }}>
            {selectedInstId
              ? (locale === 'zh-CN'
                  ? `接口设置（${selectedInterfaceName || `Interface.${selectedInstId}`}）`
                  : `Interface Settings (${selectedInterfaceName || `Interface.${selectedInstId}`})`)
              : (locale === 'zh-CN' ? '未找到网络接口' : 'No network interface found')}
          </div>
        </>
      )}

      {isLoading ? (
        <div style={{ textAlign: 'center', padding: 24 }}>
          <Spin />
        </div>
      ) : !selectedInstId ? (
        <div style={{ color: '#8c8c8c', textAlign: 'center', padding: 24 }}>
          {instanceIds.length === 0
            ? (locale === 'zh-CN' ? '当前没有实例，点击"新增"创建' : 'No instances. Click Add to create one.')
            : (locale === 'zh-CN' ? '请在上方选择一个实例' : 'Select an instance above')}
        </div>
      ) : (
        <Spin spinning={isFetching} tip={locale === 'zh-CN' ? '正在更新…' : 'Updating...'} delay={150}>
          <Collapse
            defaultActiveKey={visibleChildGroups
              .filter((group) => {
                const subObject = extractSubObject(group.objectPath || '', selectorGroup.objectPath || '');
                return subInstanceIds(subObject).length > 0;
              })
              .map((group) => group.id)}
            items={visibleChildGroups.map((cg) => {
              const label = (
                <strong>{locale === 'zh-CN' ? cg.titleZh : cg.titleEn}</strong>
              );
              if (cg.style === 'subtable') {
                const subObject = extractSubObject(cg.objectPath || '', selectorGroup.objectPath || '');
                if (selectorGroup.id === 'gnb-network-interface' && selectedInstId && subObject !== 'VlanInterface') {
                  const listGroup: QuickSettingsGroup = {
                    ...cg,
                    id: `${cg.id}-interface-${selectedInstId}`,
                    parentSelector: undefined,
                    style: 'table',
                    objectPath: (cg.objectPath || '')
                      .replace('{i}', selectedInstId)
                      .replace('{j}', '{i}'),
                  };
                  return {
                    key: cg.id,
                    label,
                    children: (
                      <MultiInstanceTable
                        deviceId={deviceId}
                        active={active}
                        group={listGroup}
                        instanceContext={instanceContext}
                        locale={locale}
                      />
                    ),
                  };
                }
                const def: BscSubTableDef = {
                  subObject,
                  columns: cg.params.map((p) => ({
                    key: p.leaf || p.name,
                    title: locale === 'zh-CN' ? p.titleZh : p.titleEn,
                    options: p.enumOptions?.map((option) => ({ value: option.value, label: option.label })),
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
                    () => {
                      if (selectorGroup.id === 'gnb-network-interface' && subObject === 'VlanInterface') {
                        setNrWanAdd({ enable: '1', vlanName: '', vlanId: '' });
                        return;
                      }
                      void handleSubAdd(subObject);
                    },
                    (rowIdx) => void handleSubDelete(subObject, rowIdx),
                    addMutation.isPending,
                    deleteMutation.isPending,
                    getWritableByPath,
                    locale,
                    selectorGroup.id === 'gnb-network-interface',
                    cg.id === 'gnb-interface-vlan' ? () => void handleSave() : undefined,
                    cg.id === 'gnb-interface-vlan'
                      ? updateMutation.isPending || awaitingSubmittedReadback
                      : false,
                    cg.id === 'gnb-interface-vlan'
                      ? (row) => (
                          <Space direction="vertical" size={4} style={{ width: '100%' }}>
                            {vlanAddressGroups.map((addressGroup) => {
                              const listGroup: QuickSettingsGroup = {
                                ...addressGroup,
                                id: `${addressGroup.id}-interface-${selectedInstId}-vlan-${row.id}`,
                                parentSelector: undefined,
                                style: 'table',
                                objectPath: (addressGroup.objectPath || '')
                                  .replace('{i}', selectedInstId)
                                  .replace('{i}', String(row.id))
                                  .replace('{i}', '{i}'),
                              };
                              return (
                                <MultiInstanceTable
                                  key={listGroup.id}
                                  deviceId={deviceId}
                                  active={active}
                                  group={listGroup}
                                  instanceContext={instanceContext}
                                  locale={locale}
                                />
                              );
                            })}
                          </Space>
                        )
                      : undefined,
                    cg.id === 'gnb-interface-vlan' ? (rowIdx) => openVlanEdit(rowIdx) : undefined,
                    cg.id === 'gnb-interface-vlan'
                      ? rowsWithNestedInstances(
                          schemaByPath.keys(),
                          `${objectPath}${selectedInstId}.VlanInterface.`,
                          subInstanceIds(subObject),
                          vlanAddressGroups.map((group) => extractSubObject(group.objectPath || '', cg.objectPath || '')),
                        )
                      : [],
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
                  locale,
                ),
              };
            })}
          />
          {!isGnbNetworkInterface && <div style={{ marginTop: 16, textAlign: 'right' }}>
            <Button
              type="primary"
              icon={<SaveOutlined />}
              onClick={() => void handleSave()}
              loading={updateMutation.isPending || awaitingSubmittedReadback}
              disabled={Object.keys(formEdits).length === 0 || awaitingSubmittedReadback}
            >
              {locale === 'zh-CN' ? '保 存' : 'Save'}
            </Button>
          </div>}
        </Spin>
      )}

      {isGnbNetworkInterface && !isLoading && lanInstanceId && (
        <div style={{ marginTop: 8 }}>
          <div style={{ marginBottom: 8, fontWeight: 600 }}>
            {locale === 'zh-CN'
              ? `LAN 口设置（Interface.${lanInstanceId}）`
              : `LAN Port Settings (Interface.${lanInstanceId})`}
          </div>
          <Collapse
            defaultActiveKey={directChildGroups
              .filter((group) => {
                const subObject = extractSubObject(group.objectPath || '', selectorGroup.objectPath || '');
                return subObject !== 'VlanInterface' && subInstanceIdsFor(lanInstanceId, subObject).length > 0;
              })
              .map((group) => `lan-${group.id}`)}
            items={directChildGroups
              .filter((group) => {
                const subObject = extractSubObject(group.objectPath || '', selectorGroup.objectPath || '');
                return group.style === 'subtable' && subObject !== 'VlanInterface';
              })
              .map((group) => {
                const listGroup: QuickSettingsGroup = {
                  ...group,
                  id: `${group.id}-interface-${lanInstanceId}`,
                  parentSelector: undefined,
                  style: 'table',
                  objectPath: (group.objectPath || '')
                    .replace('{i}', lanInstanceId)
                    .replace('{j}', '{i}'),
                };
                return {
                  key: `lan-${group.id}`,
                  label: <strong>{locale === 'zh-CN' ? group.titleZh : group.titleEn}</strong>,
                  children: (
                    <MultiInstanceTable
                      deviceId={deviceId}
                      active={active}
                      group={listGroup}
                      instanceContext={instanceContext}
                      locale={locale}
                    />
                  ),
                };
              })}
          />
        </div>
      )}

      <Modal
        title={locale === 'zh-CN' ? '添加 WAN(VLAN)' : 'Add WAN(VLAN)'}
        open={Boolean(nrWanAdd)}
        onOk={() => void handleNrWanAdd()}
        onCancel={() => setNrWanAdd(null)}
        okText={locale === 'zh-CN' ? '确定' : 'OK'}
        cancelText={locale === 'zh-CN' ? '取消' : 'Cancel'}
        confirmLoading={addMutation.isPending || updateMutation.isPending}
        width={760}
        destroyOnHidden
      >
        {nrWanAdd && (
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: '18px 28px' }}>
            <div>
              <div style={{ marginBottom: 6 }}>Enable</div>
              <Select
                style={{ width: '100%' }}
                value={nrWanAdd.enable}
                options={[{ value: '1', label: 'ON' }, { value: '0', label: 'OFF' }]}
                onChange={(value) => setNrWanAdd((prev) => prev ? { ...prev, enable: value } : prev)}
              />
            </div>
            <div>
              <div style={{ marginBottom: 6 }}>VLAN Name <Text type="secondary">Length: 1~13 Characters</Text></div>
              <Input maxLength={13} value={nrWanAdd.vlanName} onChange={(event) => setNrWanAdd((prev) => prev ? { ...prev, vlanName: event.target.value } : prev)} />
            </div>
            <div>
              <div style={{ marginBottom: 6 }}>VLAN ID <Text type="secondary">2~4094, Integer</Text></div>
              <Input value={nrWanAdd.vlanId} onChange={(event) => setNrWanAdd((prev) => prev ? { ...prev, vlanId: event.target.value } : prev)} />
            </div>
          </div>
        )}
      </Modal>

      <Modal
        title={locale === 'zh-CN' ? `修改 WAN(VLAN) · 实例 ${vlanEdit?.instanceId ?? ''}` : `Edit WAN(VLAN) · Instance ${vlanEdit?.instanceId ?? ''}`}
        open={Boolean(vlanEdit)}
        onOk={() => void handleVlanEdit()}
        onCancel={() => setVlanEdit(null)}
        okText={locale === 'zh-CN' ? '保存' : 'Save'}
        cancelText={locale === 'zh-CN' ? '取消' : 'Cancel'}
        confirmLoading={updateMutation.isPending}
        width={760}
        destroyOnHidden
      >
        {vlanEdit && (
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: '18px 28px' }}>
            <div>
              <div style={{ marginBottom: 6 }}>Enable</div>
              <Select
                style={{ width: '100%' }}
                value={vlanEdit.enable}
                options={[{ value: '1', label: 'ON' }, { value: '0', label: 'OFF' }]}
                onChange={(value) => setVlanEdit((prev) => prev ? { ...prev, enable: value } : prev)}
              />
            </div>
            <div>
              <div style={{ marginBottom: 6 }}>VLAN Name <Text type="secondary">Length: 1~13 Characters</Text></div>
              <Input maxLength={13} value={vlanEdit.vlanName} onChange={(event) => setVlanEdit((prev) => prev ? { ...prev, vlanName: event.target.value } : prev)} />
            </div>
            <div>
              <div style={{ marginBottom: 6 }}>VLAN ID <Text type="secondary">2~4094, Integer</Text></div>
              <Input value={vlanEdit.vlanId} onChange={(event) => setVlanEdit((prev) => prev ? { ...prev, vlanId: event.target.value } : prev)} />
            </div>
          </div>
        )}
      </Modal>

      <Modal
        title={locale === 'zh-CN' ? `${title} · 新增实例` : `${title} · Add Instance`}
        open={Boolean(addModal)}
        onOk={() => void handleConfirmAdd()}
        onCancel={closeAddModal}
        okText={locale === 'zh-CN' ? '确认新增' : 'Add'}
        cancelText={locale === 'zh-CN' ? '取消' : 'Cancel'}
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
              const hint = formatConstraintHint(item, locale);

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
                        label: localizeEnumLabel(enumMeta.labels[idx] ?? v, v, locale),
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
