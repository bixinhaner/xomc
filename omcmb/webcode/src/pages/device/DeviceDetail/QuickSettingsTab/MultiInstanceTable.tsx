import { useCallback, useEffect, useMemo, useRef, useState, type CSSProperties } from 'react';
import { Button, Card, Input, Modal, Popconfirm, Select, Space, Table, Tag, Tooltip, Typography, message, notification } from 'antd';
import { CheckCircleOutlined, ClockCircleOutlined, CloseCircleOutlined, DeleteOutlined, EditOutlined, PlusOutlined, SendOutlined, SyncOutlined } from '@ant-design/icons';
import type { ColumnType } from 'antd/es/table';
import { useParams } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import {
  useAddObject,
  useDeleteObject,
  useParameterSchema,
  useSearchParameters,
  useUpdateParameters,
} from '@core/hooks/api/useDeviceParameters';
import { useSyncDeviceParams } from '@core/hooks/api/useDevices';
import { useDeviceTaskStatus } from '@core/hooks/api/useDeviceTask';
import { notificationKeys } from '@core/hooks/api/useNotificationCenter';
import { deviceTaskApi } from '@core/services/api/deviceTaskApi';
import { configSyncApi } from '@core/services/api/configSyncApi';
import { deviceParameterApi } from '@core/services/api/deviceParameterApi';
import {
  feedbackKey,
  useQuickSettingsFeedbackStore,
  type MultiFeedback,
} from '@core/store/quickSettingsFeedbackStore';
import type { ParameterConstraints, ParameterSchemaItem, ParameterSchemaResponse, ParameterType, ParameterUpdateRequest } from '@core/types/deviceParameter';
import { isDeviceTaskTerminal, type DeviceTaskStatus } from '@core/types/deviceTask';
import type { QuickSettingsGroup, QuickSettingsParam } from '@core/types/quicksettings';
import {
  applyInstanceContext,
  formatEnumDisplayValue,
  getEffectiveEnumMeta,
  getFeedbackScopeContext,
  localizeEnumLabel,
  resolveQuickSettingsParameterType,
  serializeQuickSettingsMultiCheckboxValue,
  validateLteQOffsetValue,
  validateValue,
  type QuickSettingsInstanceContext,
} from './validators';
import { buildEffectivePlmnRows, validatePlmnList } from './plmnList';
import {
  buildIpsecSubmissionPlan,
  executeIpsecSubmissionPlan,
  type IpsecSubmissionPhase,
} from './ipsecSubmission';
import {
  applyDeviceParameterSearchReadback,
  refreshDeviceParameterSearchQueries,
} from './parameterSearchRefresh';

import { useT } from '@/hooks/useT';

const { Text } = Typography;
const ERROR_FEEDBACK_DURATION_SECONDS = 2;
const IPSEC_GROUP_IDS = new Set(['device-ipsec', 'gnb-ipsec']);
const IPSEC_ENABLE_LEAF = 'TUNNEL_ENABLE';
const IPSEC_GLOBAL_ENABLE_PATH_BY_GROUP: Record<string, string> = {
  'device-ipsec': 'Device.Services.FAPService.Ipsec.IPSEC_ENABLE',
  'gnb-ipsec': 'Device.IPsec.Enable',
};
const IPSEC_GLOBAL_ENABLE_QUERY_BY_GROUP: Record<string, string> = {
  'device-ipsec': 'IPSEC_ENABLE',
  'gnb-ipsec': 'Device.IPsec.Enable',
};

type TFn = (id: string, values?: Record<string, string | number>) => string;

function isEnabledValue(value: unknown): boolean {
  const normalized = String(value ?? '').trim().toLowerCase();
  return normalized === '1' || normalized === 'true' || normalized === 'on' || normalized === 'enable' || normalized === 'enabled';
}

function normalizeIpsecEnableValue(value: unknown): string {
  return isEnabledValue(value) ? 'true' : 'false';
}

function toDeviceIpsecEnableValue(value: unknown): string {
  return isEnabledValue(value) ? '1' : '0';
}

function normalizeComparableQuickSettingsValue(value: unknown, param?: QuickSettingsParam): string {
  if (param?.type === 'multiCheckbox') {
    return serializeQuickSettingsMultiCheckboxValue(value);
  }
  return String(value ?? '');
}

function isObjectInstanceNotFoundError(err: unknown): boolean {
  const msg = err instanceof Error ? err.message : String(err ?? '');
  const normalized = msg.toLowerCase();
  return normalized.includes('object instance not found') || normalized.includes('instance not found');
}

// "上次操作"状态形状由 frontend-core/store/quickSettingsFeedbackStore (MultiFeedback) 定义,
// 提升至 store 持久化,顶层 TabBar 切走再切回不丢反馈。

// 把后端任务 ErrorMessage(`[Client] Invalid arguments — path faults: [<path>: <code> <path>:  <reason>] [...]`)
// 提取为简短设备原因列表(如 `Value must be even; Invalid arfcn value`),用于 Tag 内联展示。
// 解析失败时退化为整段截断。完整原文仍通过 Tooltip 提供。
export function formatDeviceFaultBrief(msg: string | undefined | null): string {
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

export function formatTime(at: number): string {
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
export function statusTagSpec(action: MultiFeedback, taskStatus: DeviceTaskStatus | undefined, t: TFn): StatusTagSpec {
  const actionLabel = action.action === 'save'
    ? t('device.multi.actionSave')
    : action.action === 'add'
      ? t('device.multi.actionAdd')
      : action.action === 'add_rollback'
        ? t('device.multi.actionAddRollback')
        : t('device.multi.actionDelete');
  if (action.submitStatus === 'failed_to_queue') {
    return { color: 'error', icon: <CloseCircleOutlined />, label: t('device.multi.tagQueueFailed', { action: actionLabel }) };
  }
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
  active?: boolean;
  group: QuickSettingsGroup;
  instanceContext: QuickSettingsInstanceContext;
  locale: 'zh-CN' | 'en-US';
  ipsecControlValue?: string;
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
  pending?: 'add' | 'edit' | 'delete';
}

interface EditModalState {
  mode: 'add' | 'edit';
  instanceId?: string;
  values: Record<string, string>;
  errors: Record<string, string>;
}

interface PendingAddRow {
  tempId: string;
  values: Record<string, string>;
}

interface PackedScalarRow {
  key: string;
  id: number;
  sourceIndex?: number;
  cells: string[];
  pending?: 'add';
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
  virtual?: {
    type: ParameterType;
    constraints: ParameterConstraints;
    required?: boolean;
  };
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
    { key: 'TAC', leaf: 'TAC', titleEn: 'TAC', width: 110 },
    { key: 'PLMNID', leaf: 'PLMNID', titleEn: 'PLMN', width: 140 },
    { key: 'CID', leaf: 'CID', titleEn: 'ECI', width: 130, readOnly: true },
    { key: 'NeighborCellEnbType', leaf: 'NeighCellEnbType', titleEn: 'eNodeB Type', width: 140, formatValue: formatEnbTypeDisplay },
    { key: 'X2Flag', leaf: 'X2Flag', titleEn: 'X2 Flag', width: 120 },
  ],
};

const INTER_FREQ_GROUP_ID = 'enb-neighbor-freq';
const INTER_FREQ_EARFCN_LEAF = 'EUTRACarrierARFCN';
const NEIGHBOR_CELL_GROUP_ID = 'enb-neighbor-cell';
const NEIGHBOR_CELL_ECI_LEAF = 'CID';
const NEIGHBOR_ENB_ID_FIELD = '__neighborEnbId';
const NEIGHBOR_LOCAL_CELL_ID_FIELD = '__neighborCellId';
const NEIGHBOR_CELL_DUPLICATE_LEAVES = ['EUTRACarrierARFCN', 'PhyCellID', 'PLMNID'] as const;
const GNB_NR_NEIGHBOR_CELL_GROUP_ID = 'gnb-nr-neighbor-cell';
const GNB_NR_NEIGHBOR_SSB_LEAF = 'ssbFrequency';
const GNB_NR_INTER_FREQ_SSB_LEAF = 'SSBFrequency';
const GNB_NR_INTER_FREQ_ENABLE_LEAF = 'Enable';
const GNB_NETWORK_IPV4_GROUP_IDS = new Set([
  'gnb-interface-ipv4',
  'gnb-interface-vlan-ipv4',
]);
const GNB_NETWORK_IPV6_GROUP_IDS = new Set([
  'gnb-interface-ipv6',
  'gnb-interface-vlan-ipv6',
]);
const GNB_NETWORK_IPV4_STATIC_LEAVES = new Set([
  'IPAddress',
  'SubnetMask',
  'DefaultGateway',
]);
const GNB_NETWORK_IPV6_STATIC_LEAVES = new Set([
  'IPAddress',
  'PrefixLength',
  'DefaultGateway',
]);

function isGnbNetworkIpv4Group(groupId: string): boolean {
  return Array.from(GNB_NETWORK_IPV4_GROUP_IDS).some(
    (group) => groupId === group || groupId.startsWith(`${group}-interface-`),
  );
}

function isGnbNetworkAddressGroup(groupId: string): boolean {
  return isGnbNetworkIpv4Group(groupId) || Array.from(GNB_NETWORK_IPV6_GROUP_IDS).some(
    (group) => groupId === group || groupId.startsWith(`${group}-interface-`),
  );
}

function isGnbNetworkStaticField(groupId: string, leaf: string): boolean {
  const isIpv4 = isGnbNetworkIpv4Group(groupId);
  const isIpv6 = Array.from(GNB_NETWORK_IPV6_GROUP_IDS).some(
    (group) => groupId === group || groupId.startsWith(`${group}-interface-`),
  );
  return (isIpv4 && GNB_NETWORK_IPV4_STATIC_LEAVES.has(leaf))
    || (isIpv6 && GNB_NETWORK_IPV6_STATIC_LEAVES.has(leaf));
}

function isGnbNetworkIpv4FieldVisible(
  groupId: string,
  leaf: string,
  values: Record<string, string>,
  addressingTypeLeaf: string,
): boolean {
  if (!isGnbNetworkStaticField(groupId, leaf)) return true;
  const addressingType = String(values[addressingTypeLeaf] ?? '').trim().toLowerCase();
  return addressingType === 'static';
}

function composeLteEci(enbId: string, cellId: string): string {
  return String(Number(enbId) * 256 + Number(cellId));
}

function multiTableScroll(hasRows: boolean): { x?: 'max-content'; y: number } {
  return hasRows ? { x: 'max-content', y: 240 } : { y: 240 };
}

function multiColumnWidth(hasRows: boolean, width: number | undefined): number | undefined {
  return hasRows ? width : undefined;
}

function deriveGnbNrInterFreqObjectPath(nrNeighborObjectPath: string): string {
  return nrNeighborObjectPath.replace(
    '.NR.RAN.NeighborList.NRCell.',
    '.NR.RAN.Mobility.ConnMode.NR.InterFreq.Carrier.',
  );
}

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

function quickParamType(param: QuickSettingsParam | undefined): ParameterType {
  switch (param?.type) {
    case 'int':
    case 'unsignedInt':
    case 'boolean':
    case 'dateTime':
    case 'base64':
    case 'hexBinary':
    case 'object':
      return param.type;
    default:
      return 'string';
  }
}

function quickParamConstraints(param: QuickSettingsParam | undefined): ParameterConstraints | undefined {
  if (!param) return undefined;
  const constraints: ParameterConstraints = {};
  if (param.minValue !== undefined) constraints.minValue = param.minValue;
  if (param.maxValue !== undefined) constraints.maxValue = param.maxValue;
  if (param.enumOptions && param.enumOptions.length > 0) {
    constraints.enumValues = param.enumOptions.map((option) => option.value);
    constraints.enumLabels = param.enumOptions.map((option) => option.label);
  }
  return Object.keys(constraints).length > 0 ? constraints : undefined;
}

function effectiveParamType(item: ParameterSchemaItem | undefined, param: QuickSettingsParam | undefined): ParameterType {
  return resolveQuickSettingsParameterType(
    param?.type,
    item?.type,
  );
}

function effectiveParamConstraints(
  item: ParameterSchemaItem | undefined,
  param: QuickSettingsParam | undefined,
): ParameterConstraints | undefined {
  const quick = quickParamConstraints(param);
  const schema = item?.constraints;
  if (!quick && !schema) return undefined;
  return {
    ...(quick ?? {}),
    ...(schema ?? {}),
  };
}

function validateQuickSettingsCellValue(
  leaf: string,
  value: string,
  parameterType: ParameterType,
  constraints?: ParameterConstraints,
): string | null {
  const normalizedLeaf = leaf.trim().toLowerCase();
  if (normalizedLeaf === 'qoffset' || normalizedLeaf === 'qoffsetfreq') {
    return validateLteQOffsetValue(value);
  }
  return validateValue(value, parameterType, constraints);
}

function feedbackTagStyle(): CSSProperties {
  return {
    maxWidth: 560,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
    verticalAlign: 'middle',
  };
}

function formatEffectiveConstraintHint(
  item: ParameterSchemaItem | undefined,
  param: QuickSettingsParam | undefined,
  t: TFn,
): string {
  const constraints = effectiveParamConstraints(item, param);
  if (!constraints) return '';
  if (constraints.enumValues && constraints.enumValues.length > 0) return '';
  const isString = effectiveParamType(item, param) === 'string';
  const min = constraints.minLength ?? constraints.minValue;
  const max = constraints.maxLength ?? constraints.maxValue;
  if (min === undefined && max === undefined) return '';
  const lo = min ?? '-∞';
  const hi = max ?? '∞';
  return isString ? t('device.multi.hintLenRange', { lo, hi }) : `[${lo} ~ ${hi}]`;
}

function formatQuickParamConstraintHint(param: QuickSettingsParam | undefined, t: TFn): string {
  if (!param || (param.enumOptions && param.enumOptions.length > 0)) return '';
  if (param.minValue === undefined && param.maxValue === undefined) return '';
  const lo = param.minValue ?? '-∞';
  const hi = param.maxValue ?? '∞';
  return quickParamType(param) === 'string' ? t('device.multi.hintLenRange', { lo, hi }) : `[${lo} ~ ${hi}]`;
}

interface PackedScalarNeighborTableProps {
  deviceId: string;
  active?: boolean;
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
  active = true,
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
  const { data: schemaResp, isLoading, refetch, isFetching } = useParameterSchema(deviceId, parentPath, active);
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
  const [pendingPackedAdds, setPendingPackedAdds] = useState<string[][]>([]);
  const [pendingPackedDeletes, setPendingPackedDeletes] = useState<Set<number>>(() => new Set());
  const rows = useMemo<PackedScalarRow[]>(() => {
    const existingRows = cellsList
      .map((cells, idx) => ({ key: `existing:${idx}`, id: idx + 1, sourceIndex: idx, cells }))
      .filter((row) => !pendingPackedDeletes.has(row.sourceIndex));
    const addRows = pendingPackedAdds.map((cells, idx) => ({
      key: `new:${idx}`,
      id: existingRows.length + idx + 1,
      cells,
      pending: 'add' as const,
    }));
    return [...existingRows, ...addRows];
  }, [cellsList, pendingPackedAdds, pendingPackedDeletes]);

  // 列定义中字段顺序严格跟随 group.params（来自 quicksettings XML），对应打包条目内 `-` 分隔字段的下标。
  const fieldLeaves = useMemo(
    () => group.params.map((param) => param.leaf || param.name),
    [group.params],
  );
  const paramByLeaf = useMemo(() => {
    const map = new Map<string, QuickSettingsParam>();
    group.params.forEach((param) => {
      map.set(param.leaf || param.name, param);
    });
    return map;
  }, [group.params]);

  // 新增弹窗状态：null = 关闭；values 以 leaf 为 key。
  const [addModal, setAddModal] = useState<{
    values: Record<string, string>;
    errors: Record<string, string>;
  } | null>(null);

  const maxInstances = group.maxInstances && group.maxInstances > 0 ? group.maxInstances : undefined;
  const reachedMax = maxInstances !== undefined && rows.length >= maxInstances;
  const canMutate = fieldLeaves.length > 0; // 没有字段定义就退回纯只读视图（兜底）
  const pendingPackedChangeCount = pendingPackedAdds.length + pendingPackedDeletes.size;

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
  const { data: lastTask } = useDeviceTaskStatus(active ? lastAction?.taskId : undefined, { intervalMs: 800 });

  const waitForTaskTerminal = useCallback(async (taskId: string) => {
    const timeoutAt = Date.now() + 60000;
    while (Date.now() < timeoutAt) {
      const task = await deviceTaskApi.getTask(taskId);
      if (isDeviceTaskTerminal(task.status)) return task;
      await new Promise((resolve) => window.setTimeout(resolve, 400));
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
      values: Object.fromEntries(
        fieldLeaves.map((leaf) => {
          const param = paramByLeaf.get(leaf);
          return [leaf, param?.defaultValue ?? ''];
        }),
      ),
      errors: {},
    });
  }, [canMutate, fieldLeaves, paramByLeaf, reachedMax]);

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
      const param = paramByLeaf.get(leaf);
      const raw = (addModal.values[leaf] ?? '').trim();
      if (!raw) {
        if (param?.required) {
          errors[leaf] = t('device.multi.packed.fieldRequired');
        }
        cells.push('');
        continue;
      }
      if (/[\s-]/.test(raw)) {
        // `-` 是字段分隔符，空白是条目分隔符，两者都不能出现在字段值里。
        errors[leaf] = t('device.multi.packed.fieldInvalidChar');
        cells.push(raw);
        continue;
      }
      const err = validateValue(raw, quickParamType(param), quickParamConstraints(param));
      if (err) {
        errors[leaf] = err;
        cells.push(raw);
        continue;
      }
      cells.push(raw);
    }
    if (Object.keys(errors).length > 0) {
      setAddModal((prev) => (prev ? { ...prev, errors } : prev));
      return;
    }
    setAddModal(null);
    setPendingPackedAdds((prev) => [...prev, cells]);
    message.success({ content: t('device.multi.batchStaged'), duration: 3 });
  }, [addModal, fieldLeaves, paramByLeaf, t]);

  const handleDeleteRow = useCallback(
    async (row: PackedScalarRow) => {
      if (!canMutate) return;
      if (row.pending === 'add') {
        const addIndex = pendingPackedAdds.findIndex((cells) => cells === row.cells);
        if (addIndex >= 0) {
          setPendingPackedAdds((prev) => prev.filter((_cells, idx) => idx !== addIndex));
        }
        return;
      }
      if (row.sourceIndex === undefined) return;
      setPendingPackedDeletes((prev) => {
        const next = new Set(prev);
        next.add(row.sourceIndex!);
        return next;
      });
    },
    [canMutate, pendingPackedAdds],
  );

  const hasRows = rows.length > 0;
  const handleSubmitPackedBatch = useCallback(async () => {
    if (pendingPackedChangeCount === 0 || isSubmitting) return;
    setIsSubmitting(true);
    try {
      for (const rowIdx of Array.from(pendingPackedDeletes).sort((a, b) => a - b)) {
        const target = cellsList[rowIdx];
        if (!target) continue;
        await writeSingleEntry(
          'del',
          target,
          t('device.multi.deleteDispatched', { instId: String(rowIdx + 1) }),
        );
      }
      for (const cells of pendingPackedAdds) {
        await writeSingleEntry(
          'add',
          cells,
          t('device.multi.detailAdd', { instId: String(cellsList.length + 1), count: cells.length }),
        );
      }
      setPendingPackedAdds([]);
      setPendingPackedDeletes(new Set());
    } finally {
      setIsSubmitting(false);
    }
  }, [cellsList, isSubmitting, pendingPackedAdds, pendingPackedChangeCount, pendingPackedDeletes, t, writeSingleEntry]);

  const clearPackedBatch = useCallback(() => {
    setPendingPackedAdds([]);
    setPendingPackedDeletes(new Set());
  }, []);

  const columns: ColumnType<PackedScalarRow>[] = [
    {
      title: t('device.multi.instance'),
      dataIndex: 'id',
      key: 'id',
      width: multiColumnWidth(hasRows, 80),
      fixed: hasRows ? 'left' : undefined,
      render: (_v: unknown, row) => (
        <Space size={4}>
          <Text strong>{row.id}</Text>
          {row.pending === 'add' && <Tag color="blue">{t('device.multi.pendingAdd')}</Tag>}
        </Space>
      ),
    },
    ...group.params.map<ColumnType<PackedScalarRow>>((param, colIdx) => ({
      title: locale === 'zh-CN' ? param.titleZh : param.titleEn,
      key: param.leaf || param.name,
      width: multiColumnWidth(hasRows, 140),
      render: (_v: unknown, row) => <Text>{row.cells[colIdx] ?? '-'}</Text>,
    })),
  ];
  if (canMutate && hasRows) {
    columns.push({
      title: t('device.multi.packed.colActions'),
      key: '__op',
      width: multiColumnWidth(hasRows, 90),
      fixed: hasRows ? 'right' : undefined,
      render: (_v: unknown, row) => (
        <Popconfirm
          title={t('device.multi.deleteConfirm')}
          description={t('device.multi.detailInstance', { instId: String(row.id) })}
          okButtonProps={{ danger: true, loading: updateMutation.isPending || isSubmitting }}
          onConfirm={() => void handleDeleteRow(row)}
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
      styles={{ body: hasRows ? undefined : { padding: 0 } }}
      extra={
        <Space>
          {lastAction && lastAction.action !== 'add_rollback' && (() => {
            const tagSpec = statusTagSpec(lastAction, lastTask?.status, t);
            const isFailed = lastTask?.status === 'failed' && Boolean(lastTask?.errorMessage);
            const briefFault = isFailed ? formatDeviceFaultBrief(lastTask?.errorMessage) : '';
            const tag = (
              <Tag icon={tagSpec.icon} color={tagSpec.color} style={feedbackTagStyle()}>
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
          {pendingPackedChangeCount > 0 && (
            <>
              <Button size="small" onClick={clearPackedBatch} disabled={isSubmitting}>
                {t('device.multi.batchClear')}
              </Button>
              <Button
                size="small"
                type="primary"
                icon={<SendOutlined />}
                loading={isSubmitting}
                onClick={() => void handleSubmitPackedBatch()}
              >
                {t('device.multi.batchSubmitWithCount', { count: pendingPackedChangeCount })}
              </Button>
            </>
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
      {hasRows && (
        <>
          <Table
            rowKey="key"
            dataSource={rows}
            columns={columns}
            loading={isLoading}
            size="small"
            pagination={false}
            scroll={multiTableScroll(true)}
            sticky
          />
          <div style={{ marginTop: 8, color: '#8c8c8c', fontSize: 12 }}>
            {t('device.multi.packed.sourceLabel')}{scalarPath}
            {lastSyncedAt ? ` · ${t('device.multi.packed.lastSynced', { time: formatTime(new Date(lastSyncedAt).getTime()) })}` : ''}
          </div>
        </>
      )}

      <Modal
        title={t('device.multi.modalAddTitle', { title })}
        open={!!addModal}
        onCancel={closeAddModal}
        onOk={() => void handleAddSubmit()}
        okText={t('device.multi.confirmAdd')}
        confirmLoading={updateMutation.isPending || isSubmitting}
        destroyOnClose
        mask={{ closable: false }}
      >
        {addModal && (
          <Space direction="vertical" size="small" style={{ width: '100%' }}>
            {group.params.map((param) => {
              const leaf = param.leaf || param.name;
              const label = locale === 'zh-CN' ? param.titleZh : param.titleEn;
              const err = addModal.errors[leaf];
              const hint = formatQuickParamConstraintHint(param, t);
              return (
                <div key={leaf}>
                  <div style={{ marginBottom: 4, fontSize: 12 }}>
                    {param.required && <Text type="danger" style={{ marginRight: 4 }}>*</Text>}
                    <Text strong>{label}</Text>
                    <Text type="secondary" style={{ marginLeft: 8 }}>{leaf}</Text>
                    {hint && <Text type="secondary" style={{ marginLeft: 8 }}>{hint}</Text>}
                  </div>
                  {param.enumOptions && param.enumOptions.length > 0 ? (
                    <Select
                      size="small"
                      value={addModal.values[leaf] || undefined}
                      status={err ? 'error' : undefined}
                      onChange={(value) => setAddModalValue(leaf, value ?? '')}
                      placeholder="Select"
                      allowClear
                      style={{ width: '100%' }}
                      optionFilterProp="label"
                      options={param.enumOptions.map((option) => ({
                        value: option.value,
                        label: option.label || option.value,
                      }))}
                    />
                  ) : (
                    <Input
                      size="small"
                      value={addModal.values[leaf] ?? ''}
                      status={err ? 'error' : undefined}
                      onChange={(e) => setAddModalValue(leaf, e.target.value)}
                      placeholder={label}
                    />
                  )}
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
export default function MultiInstanceTable({ deviceId, active = true, group, instanceContext, locale, ipsecControlValue }: MultiInstanceTableProps) {
  const t = useT();
  const feedbackScope = useMemo(() => getFeedbackScopeContext(group.id, instanceContext), [group.id, instanceContext]);
  const isIpsecGroup = IPSEC_GROUP_IDS.has(group.id);
  const ipsecGlobalEnablePath = IPSEC_GLOBAL_ENABLE_PATH_BY_GROUP[group.id] ?? '';
  const ipsecGlobalEnableQuery = IPSEC_GLOBAL_ENABLE_QUERY_BY_GROUP[group.id] ?? 'IPSEC_ENABLE';
  const { data: ipsecGlobalParams } = useSearchParameters(deviceId, ipsecGlobalEnableQuery, 20, active && isIpsecGroup);
  const ipsecControlDraftKey = useMemo(
    () => feedbackKey(
      deviceId,
      'device-ipsec-control',
      1,
      undefined,
    ),
    [deviceId],
  );
  const ipsecControlDraft = useQuickSettingsFeedbackStore((s) => s.drafts[ipsecControlDraftKey]?.IPSEC_ENABLE);
  const currentIpsecGlobalValue = useMemo(
    () => ipsecGlobalParams?.find((item) => item.parameterPath === ipsecGlobalEnablePath)?.parameterValue,
    [ipsecGlobalEnablePath, ipsecGlobalParams],
  );
  const isIpsecGlobalStateReady = !isIpsecGroup || currentIpsecGlobalValue !== undefined;
  const currentIpsecEnabled = isEnabledValue(currentIpsecGlobalValue);
  const targetIpsecEnabled = isEnabledValue(
    ipsecControlDraft ?? ipsecControlValue ?? currentIpsecGlobalValue,
  );
  const hasIpsecGlobalChange = Boolean(
    isIpsecGroup
    && currentIpsecGlobalValue !== undefined
    && (ipsecControlDraft !== undefined || ipsecControlValue !== undefined)
    && targetIpsecEnabled !== currentIpsecEnabled,
  );
  const ipsecGlobalEnabled = useMemo(() => {
    if (!isIpsecGroup) return true;
    return targetIpsecEnabled;
  }, [isIpsecGroup, targetIpsecEnabled]);
  // BSC 邻区兼容：部分 GSM 设备不按 TR-181 子对象上报，而是把整张邻区列表打包到 BTS 父对象单标量。
  // 这种 group 不存在 currentInstances，常规多实例渲染会出现「暂无数据」。改走打包标量解析路径。
  const packedSpec = PACKED_NEIGHBOR_TABLE_BY_GROUP_ID[group.id];
  // group.objectPath 形如 "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}."
  // - 外层 FAPService.{i} → 用 fapInstance 替换
  // - 内层 Carrier.{i}. 末段是实例号占位符 — 剥离后得到父对象路径,用于查 schema.objects / AddObject / 拼接行 path 前缀
  const objectPath = useMemo(() => {
    const resolved = applyInstanceContext(group.objectPath || '', instanceContext, {
      preserveTrailingInstance: true,
    });
    return resolved.replace(/\{i\}\.$/, '');
  }, [group, instanceContext]);
  const { data: schemaResp, isLoading, refetch } = useParameterSchema(deviceId, objectPath, active && !packedSpec);
  const nrInterFreqObjectPath = useMemo(
    () => group.id === GNB_NR_NEIGHBOR_CELL_GROUP_ID ? deriveGnbNrInterFreqObjectPath(objectPath) : '',
    [group.id, objectPath],
  );
  const { data: nrInterFreqSchemaResp, isLoading: nrInterFreqLoading } = useParameterSchema(
    deviceId,
    nrInterFreqObjectPath,
    active && Boolean(nrInterFreqObjectPath),
  );
  const updateMutation = useUpdateParameters();
  const addMutation = useAddObject();
  const deleteMutation = useDeleteObject();
  const scopedSyncMutation = useSyncDeviceParams();
  const queryClient = useQueryClient();
  // 乐观删除集合:Add 流程 SPV 失败时立刻把对应实例号加进来,渲染层过滤掉。
  // 与 schema cache 解耦 — 不受 deleteMutation onSuccess invalidate 触发的 refetch 干扰,
  // 等设备真删 ACK 且 refetch 拉到不含该实例的 schema 后再从集合移除。
  const [optimisticallyRemoved, setOptimisticallyRemoved] = useState<Set<number>>(() => new Set());
  const specialColumns = BM_SPECIAL_COLUMNS[group.id] ?? null;
  const [editModal, setEditModal] = useState<EditModalState | null>(null);
  // 状态包住整段 handleSaveEditModal(含 AddObject mutation 后的 waitForTaskTerminal 轮询),
  // 避免用户在任务未终止时以为“没反应”重复点击导致重复 AddObject。
  // ref 同步起效(防同 tick 双击); state 为 Modal confirmLoading 提供视觉反馈。
  const submittingRef = useRef(false);
  const syncedSuccessNotifiedTaskIdsRef = useRef<Set<string>>(new Set());
  const pendingAddCounterRef = useRef(0);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // 行编辑状态：以 instanceId 为 key，仅保留用户编辑过的字段（避免 effect 同步 schema 触发级联 render）
  const [rowEdits, setRowEdits] = useState<Map<string, RowEditState>>(new Map());
  const [pendingAdds, setPendingAdds] = useState<PendingAddRow[]>([]);
  const [pendingDeletes, setPendingDeletes] = useState<Set<string>>(() => new Set());

  // lastAction 由 zustand store 托管 —— DeviceDetail 卸载(切顶层 tab)也保留反馈。
  const fbKey = feedbackKey(
    deviceId,
    group.id,
    feedbackScope.fapInstance,
    feedbackScope.networkType === 'nr' ? feedbackScope.cellInstance : undefined,
  );
  const lastAction = useQuickSettingsFeedbackStore((s) => {
    const f = s.entries[fbKey];
    return f && f.kind === 'multi' ? f : null;
  });
  const setFeedback = useQuickSettingsFeedbackStore((s) => s.setFeedback);
  const patchFeedback = useQuickSettingsFeedbackStore((s) => s.patchFeedback);
  const draft = useQuickSettingsFeedbackStore((s) => s.drafts[fbKey]);
  const tunnelDraftRevision = useQuickSettingsFeedbackStore((s) => s.draftRevisions[fbKey] ?? 0);
  const ipsecControlDraftRevision = useQuickSettingsFeedbackStore(
    (s) => s.draftRevisions[ipsecControlDraftKey] ?? 0,
  );
  const setDraftField = useQuickSettingsFeedbackStore((s) => s.setDraftField);
  const clearDraft = useQuickSettingsFeedbackStore((s) => s.clearDraft);
  const clearDraftPrefix = useQuickSettingsFeedbackStore((s) => s.clearDraftPrefix);
  const { data: lastTask } = useDeviceTaskStatus(active ? lastAction?.taskId : undefined, { intervalMs: 800 });

  // schema.objects 给出 currentInstances；schema.parameters 给出值
  const objectEntry = useMemo(
    () => schemaResp?.objects.find((o) => o.path === objectPath),
    [schemaResp, objectPath],
  );
  // 后端 schema 目前只会为"已有实例"的对象返回 ObjectSchemaItem，空列表时 objectEntry 可能缺失。
  // 这类场景下仍允许直接走 AddObject，由后端最终校验 objectPath 是否可新增。
  // NR 网络子对象的部分旧模型错误上报 canAdd/canDeleteAny=false；实际对象支持标准 AddObject/DeleteObject。
  const isNrNetworkChildGroup = /^gnb-interface-(?:ipv4|ipv6|vlan-ipv4|vlan-ipv6)-interface-\d+(?:-vlan-\d+)?$/.test(group.id);
  const canAdd = isNrNetworkChildGroup || (objectEntry?.canAdd ?? Boolean(group.objectPath && objectPath));
  const canDelete = isNrNetworkChildGroup || (objectEntry?.canDeleteAny ?? false);

  const schemaByPath = useMemo(() => {
    const map = new Map<string, ParameterSchemaItem>();
    schemaResp?.parameters.forEach((p) => map.set(p.path, p));
    return map;
  }, [schemaResp]);

  const nrInterFreqSsbState = useMemo(() => {
    const known = new Set<string>();
    const enabled = new Set<string>();
    const instances = nrInterFreqSchemaResp?.objects.find((o) => o.path === nrInterFreqObjectPath)?.currentInstances ?? [];
    const params = new Map<string, ParameterSchemaItem>();
    nrInterFreqSchemaResp?.parameters.forEach((item) => params.set(item.path, item));

    instances.forEach((instId) => {
      const prefix = `${nrInterFreqObjectPath}${instId}.`;
      const ssb = String(params.get(`${prefix}${GNB_NR_INTER_FREQ_SSB_LEAF}`)?.currentValue ?? '').trim();
      if (!ssb) return;
      known.add(ssb);
      const enableItem = params.get(`${prefix}${GNB_NR_INTER_FREQ_ENABLE_LEAF}`);
      if (!enableItem || isEnabledValue(enableItem.currentValue)) {
        enabled.add(ssb);
      }
    });

    return { known, enabled };
  }, [nrInterFreqObjectPath, nrInterFreqSchemaResp]);

  const hiddenInstanceNumbers = useMemo(() => {
    const hidden = new Set(optimisticallyRemoved);
    if (
      lastAction?.action === 'save'
      && lastAction.saveMode === 'add'
      && lastTask
      && isDeviceTaskTerminal(lastTask.status)
      && lastTask.status !== 'completed'
      && typeof lastAction.instanceNumber === 'number'
      && lastAction.instanceNumber > 0
    ) {
      hidden.add(lastAction.instanceNumber);
    }
    return hidden;
  }, [lastAction, lastTask, optimisticallyRemoved]);

  // 实例号列表直接从 schema 派生（不再走 setState in effect）
  const instanceIds = useMemo(() => {
    if (!objectEntry) return [] as string[];
    return objectEntry.currentInstances
      .filter((n) => !hiddenInstanceNumbers.has(n))
      .map((n) => String(n))
      .sort((a, b) => Number(a) - Number(b));
  }, [objectEntry, hiddenInstanceNumbers]);

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

  const groupParamLeafSet = useMemo(() => {
    const leaves = new Set<string>();
    for (const param of group.params) {
      if (param.leaf) leaves.add(param.leaf);
    }
    return leaves;
  }, [group.params]);

  const groupParamByLeaf = useMemo(() => {
    const map = new Map<string, QuickSettingsParam>();
    for (const param of group.params) {
      if (param.leaf) map.set(param.leaf, param);
    }
    return map;
  }, [group.params]);

  const gnbNetworkAddressingTypeLeaf = useMemo(
    () => group.params.find((param) => param.name === 'AddressingType' || param.name === 'Origin')?.leaf
      ?? 'AddressingType',
    [group.params],
  );

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

  const submittedPendingAdds = useMemo<PendingAddRow[]>(() => {
    if (!lastAction?.pendingAddRows || lastAction.pendingAddRows.length === 0) return [];
    const preserveFailedIpsecAdds = isIpsecGroup && lastAction.ipsecOperationStatus === 'failed';
    if (lastAction.submitStatus !== 'queued' && !preserveFailedIpsecAdds) return [];
    if (lastAction.taskId && lastAction.syncedForTaskId === lastAction.taskId) return [];
    if (
      !preserveFailedIpsecAdds
      && lastTask
      && isDeviceTaskTerminal(lastTask.status)
      && lastTask.status !== 'completed'
    ) return [];
    const localTempIds = new Set(pendingAdds.map((row) => row.tempId));
    return lastAction.pendingAddRows.filter((row) => !localTempIds.has(row.tempId));
  }, [isIpsecGroup, lastAction, lastTask, pendingAdds]);

  const buildInitialEditValues = useCallback((): Record<string, string> => {
    const values = Object.fromEntries(
      group.params.map((param) => {
        const leaf = param.leaf || '';
        return [leaf, paramSchemaByLeaf.get(leaf)?.defaultValue ?? param.defaultValue ?? ''];
      }),
    );
    return values;
  }, [group.params, paramSchemaByLeaf]);

  const tableRows = useMemo<TableRow[]>(() => {
    const deleted = pendingDeletes;
    const existingRows = instanceIds
      .filter((instId) => !deleted.has(instId))
      .map((instId) => ({
        key: instId,
        instanceId: instId,
        pending: rowEdits.has(instId) ? 'edit' as const : undefined,
      }));
    const visiblePendingAdds = [...pendingAdds, ...submittedPendingAdds];
    const addRows = visiblePendingAdds.map((row, idx) => ({
      key: row.tempId,
      instanceId: row.tempId,
      pending: 'add' as const,
      sortIndex: instanceIds.length + idx + 1,
    }));
    return [...existingRows, ...addRows];
  }, [instanceIds, pendingAdds, pendingDeletes, rowEdits, submittedPendingAdds]);

  const scopedSyncPaths = useMemo(() => {
    const paths = new Set<string>();
    if (group.objectPath) {
      paths.add(applyInstanceContext(group.objectPath, instanceContext, { preserveTrailingInstance: true }));
    }
    for (const param of group.params) {
      if (param.standardPath) {
        paths.add(applyInstanceContext(param.standardPath, instanceContext));
      }
      if (param.extraInfoPath) {
        paths.add(applyInstanceContext(param.extraInfoPath, instanceContext));
      }
    }
    if (isIpsecGroup && ipsecGlobalEnablePath) {
      paths.add(ipsecGlobalEnablePath);
    }
    return Array.from(paths).filter(Boolean).sort();
  }, [group.objectPath, group.params, instanceContext, ipsecGlobalEnablePath, isIpsecGroup]);

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
        const param = groupParamByLeaf.get(leaf);
        const err = validateQuickSettingsCellValue(leaf, String(value ?? ''), effectiveParamType(item, param), effectiveParamConstraints(item, param));
        const current = next.get(instId) ?? { edits: {}, errors: {} };

        current.edits[leaf] = String(value ?? '');
        if (err) {
          current.errors[leaf] = err;
        }
        next.set(instId, current);
      }
      return next;
    });
  }, [draft, groupParamByLeaf, objectPath, schemaByPath, leafSchemaByLeaf]);

  useEffect(() => {
    if (!isIpsecGroup) return;
    const shouldRestoreDeletes = lastAction?.ipsecOperationStatus === 'running'
      || lastAction?.ipsecOperationStatus === 'awaiting-readback'
      || lastAction?.ipsecOperationStatus === 'failed';
    if (!shouldRestoreDeletes || !lastAction.pendingDeleteInstIds?.length) return;
    setPendingDeletes(new Set(lastAction.pendingDeleteInstIds));
  }, [
    isIpsecGroup,
    lastAction?.at,
    lastAction?.ipsecOperationStatus,
    lastAction?.pendingDeleteInstIds,
  ]);

  useEffect(() => {
    if (
      !isIpsecGroup
      || lastAction?.ipsecOperationStatus !== 'completed'
      || draft
    ) return;
    setRowEdits(new Map());
    setPendingAdds([]);
    setPendingDeletes(new Set());
  }, [draft, isIpsecGroup, lastAction?.at, lastAction?.ipsecOperationStatus]);

  const cellValue = useCallback(
    (instId: string, leaf: string): string => {
      const pendingAdd = pendingAdds.find((row) => row.tempId === instId)
        ?? submittedPendingAdds.find((row) => row.tempId === instId);
      if (pendingAdd) return pendingAdd.values[leaf] ?? '';
      const edit = rowEdits.get(instId);
      if (edit && leaf in edit.edits) return edit.edits[leaf];
      const path = `${objectPath}${instId}.${leaf}`;
      return schemaByPath.get(path)?.currentValue ?? '';
    },
    [pendingAdds, submittedPendingAdds, rowEdits, objectPath, schemaByPath],
  );

  const hideStaleInstance = useCallback((instId: string) => {
    if (!/^\d+$/.test(instId)) return;
    const inst = Number(instId);
    setOptimisticallyRemoved((prev) => {
      if (prev.has(inst)) return prev;
      const next = new Set(prev);
      next.add(inst);
      return next;
    });
    clearDraftPrefix(fbKey, `${instId}.`);
    setRowEdits((prev) => {
      const next = new Map(prev);
      next.delete(instId);
      return next;
    });
  }, [clearDraftPrefix, fbKey]);

  const clearTunnelLocalBatchChanges = useCallback(() => {
    setPendingAdds([]);
    setPendingDeletes(new Set());
    setRowEdits(new Map());
    clearDraftPrefix(fbKey, '');
  }, [clearDraftPrefix, fbKey]);

  const clearLocalBatchChanges = useCallback(() => {
    clearTunnelLocalBatchChanges();
    if (isIpsecGroup) {
      clearDraft(ipsecControlDraftKey);
      patchFeedback(fbKey, {
        pendingAddRows: [],
        pendingDeleteInstIds: [],
      });
    }
  }, [
    clearDraft,
    clearTunnelLocalBatchChanges,
    fbKey,
    ipsecControlDraftKey,
    isIpsecGroup,
    patchFeedback,
  ]);

  const restoreHiddenInstance = useCallback((instId: string) => {
    if (!/^\d+$/.test(instId)) return;
    const inst = Number(instId);
    setOptimisticallyRemoved((prev) => {
      if (!prev.has(inst)) return prev;
      const next = new Set(prev);
      next.delete(inst);
      return next;
    });
  }, []);

  const waitForTaskTerminal = useCallback(async (taskId: string) => {
    const timeoutAt = Date.now() + 60000;
    while (Date.now() < timeoutAt) {
      const task = await deviceTaskApi.getTask(taskId);
      if (isDeviceTaskTerminal(task.status)) return task;
      await new Promise((resolve) => window.setTimeout(resolve, 400));
    }
    throw new Error(t('device.multi.waitTaskTimeout'));
  }, [t]);

  const syncRelatedParameters = useCallback(async (
    options: { preserveLocalEdits?: boolean } = {},
  ): Promise<{ synced: boolean; schema?: ParameterSchemaResponse }> => {
    const preserveLocalEdits = options.preserveLocalEdits ?? false;
    if (scopedSyncPaths.length === 0) {
      const refreshed = await refetch();
      return { synced: false, schema: refreshed.data };
    }
    const startedAt = Date.now();
    let synced = false;
    try {
      const syncResult = await scopedSyncMutation.mutateAsync({ deviceId, parameterPaths: scopedSyncPaths });
      if (syncResult.gpvTaskCount === 0) {
        deviceParameterApi.invalidateParameterSchemaCache(deviceId, objectPath);
        const refreshed = await refetch();
        if (!preserveLocalEdits) {
          setRowEdits(new Map());
        }
        return { synced: false, schema: refreshed.data };
      }
      const timeoutAt = Date.now() + 60000;
      while (Date.now() < timeoutAt) {
        const status = await deviceParameterApi.getSyncStatus(deviceId);
        if (status.status !== 'syncing') {
          const successAt = status.lastParamSyncAt;
          const failedAt = status.lastParamSyncFailedAt;
          if (successAt && Date.parse(successAt) >= startedAt - 1000) {
            synced = true;
            break;
          }
          if (failedAt && Date.parse(failedAt) >= startedAt - 1000) {
            break;
          }
        }
        await new Promise((resolve) => window.setTimeout(resolve, 1000));
      }
    } catch (err) {
      // 同步失败不阻断当前操作反馈；至少再读一次本地 schema。
      console.warn('[MultiInstanceTable] sync related parameters failed', { objectPath, err });
    }
    deviceParameterApi.invalidateParameterSchemaCache(deviceId, objectPath);
    const refreshed = await refetch();
    if (!preserveLocalEdits) {
      setRowEdits(new Map());
    }
    if (synced) {
      setOptimisticallyRemoved(new Set());
    }
    return { synced, schema: refreshed.data };
  }, [deviceId, objectPath, refetch, scopedSyncMutation, scopedSyncPaths]);

  // 任务进入任一终态后再刷新当前多实例 schema:
  //  - DeleteObject:摘掉已删实例(原始用途)。handleDelete 里 API ACK 时已 refetch 一次，
  //    但设备尚未真删 → schema 仍含该实例。在这里 task 终态后再 refetch 一次当前表 schema
  //    即可与设备侧一致（只 GET 当前多实例，不做跨 device 全量 invalidate）。
  //  - SPV save:completed 时拿到新值；failed/expired/cancelled 时回到设备侧真实值，
  //    同步清掉该行 rowEdits + draft，避免页面继续显示乐观输入。
  //  - Add 流程的 SPV 失败 → 自动 DeleteObject 回滚刚创建的空实例，避免设备侧残留"半成品":
  //    判断条件 = action==='save' 且 saveMode==='add' 且 lastAction.instanceNumber>0 且 SPV 未成功
  //    且 还未为该 taskId 做过回滚(invalidatedForTaskId 去重)。与 index.tsx BSC 须知一致。
  useEffect(() => {
    if (!active) return;
    // IPSec 统一提交在 handler 内逐阶段等待并完成统一回读，旧的单任务 effect 不得抢先清草稿。
    if (isIpsecGroup && lastAction?.ipsecOperationStatus) return;
    if (!lastTask || !isDeviceTaskTerminal(lastTask.status)) return;
    // 旧的 action=add 只表示 AddObject 阶段，不需要 SPV 终态回读；
    // 批量提交里的新增行会把最后一次 SPV taskId 也记为 add，并带 addedInstIds/savedInstIds，
    // 必须继续走下面的 syncRelatedParameters，否则 Tag 会停在“已发送给基站”。
    if (
      lastAction?.action === 'add'
      && !(lastAction.addedInstIds && lastAction.addedInstIds.length > 0)
      && !(lastAction.savedInstIds && lastAction.savedInstIds.length > 0)
    ) return;
    let cancelled = false;
    void (async () => {
      const ids = lastAction?.savedInstIds ?? (
        lastAction?.action === 'save' && lastAction.savedInstId ? [lastAction.savedInstId] : []
      );
      if (ids.length > 0) {
        setRowEdits((prev) => {
          const next = new Map(prev);
          for (const id of ids) next.delete(id);
          return next;
        });
        for (const id of ids) clearDraftPrefix(fbKey, `${id}.`);
      }

      const deleteInstIds = (() => {
        if (!lastAction || lastAction.action !== 'delete') return undefined;
        if (lastAction.deletedInstIds && lastAction.deletedInstIds.length > 0) return lastAction.deletedInstIds;
        if (lastAction.savedInstIds && lastAction.savedInstIds.length > 0) return lastAction.savedInstIds;
        if (lastAction.savedInstId && /^\d+$/.test(lastAction.savedInstId)) return [lastAction.savedInstId];
        const match = lastAction.detail.match(/\d+/);
        return match?.[0] ? [match[0]] : undefined;
      })();
      if (deleteInstIds) {
        if (lastTask.status === 'completed') {
          deleteInstIds.forEach((instId) => hideStaleInstance(instId));
        } else {
          deleteInstIds.forEach((instId) => restoreHiddenInstance(instId));
        }
      }
      if (
        lastAction?.pendingAddRows
        && lastAction.pendingAddRows.length > 0
        && lastTask.status !== 'completed'
      ) {
        const failedTempIds = new Set(lastAction.pendingAddRows.map((row) => row.tempId));
        setPendingAdds((prev) => prev.filter((row) => !failedTempIds.has(row.tempId)));
      }

      let didRollbackAddedInstance = false;
      const addRollbackInst = (() => {
        if (!lastAction || lastAction.action !== 'save') return undefined;
        if (lastAction.saveMode !== 'add') return undefined;
        if (typeof lastAction.instanceNumber === 'number' && lastAction.instanceNumber > 0) return lastAction.instanceNumber;
        return undefined;
      })();

      // Add 流程的 SPV 未成功 → 静默自动 DeleteObject 回滚刚创建的空实例。
      // 用户感知:只看到 T-0146 弹的"基站应答失败"通知 + Tag(由原 save+failed 渲染);
      //          表格里既不会出现"自动删除"Tag,也不会出现残留的空实例行。
      // 实现要点:
      //   1) 先把 inst 加入 optimisticallyRemoved → instanceIds useMemo 过滤掉该行,
      //      表格立刻去掉它。该集合是 UI 层过滤,不受 deleteMutation onSuccess invalidate
      //      触发的 schema refetch 干扰(否则乐观删除会被立即覆盖回来)。
      //   2) 后台 deleteMutation 真删 + 等任务终态后再 refetch,继续保留过滤状态。
      //      原因:后端 schema 缓存可能滞后,过早撤回过滤会把已删行从旧缓存里带回来。
      //   3) 不调 setFeedback(避免 Tag 变成 add_rollback)。
      if (
        lastAction
        && lastAction.action === 'save'
        && lastAction.saveMode === 'add'
        && lastTask.status !== 'completed'
        && typeof addRollbackInst === 'number'
        && addRollbackInst > 0
        && lastAction.invalidatedForTaskId !== lastTask.id
      ) {
        didRollbackAddedInstance = true;
        const inst = addRollbackInst;
        // 去重标记必须在 await 之前写 — 避免 React StrictMode 等重复调度中双发。
        patchFeedback(fbKey, { invalidatedForTaskId: lastTask.id });
        // 1) UI 层立刻乐观删除该行。
        setOptimisticallyRemoved((prev) => {
          if (prev.has(inst)) return prev;
          const next = new Set(prev);
          next.add(inst);
          return next;
        });
        // 清掉被回滚实例可能残留的 rowEdits + draft（这些都是用户感知不到的内部状态）。
        clearDraftPrefix(fbKey, `${inst}.`);
        setRowEdits((prev) => {
          const next = new Map(prev);
          next.delete(String(inst));
          return next;
        });
        try {
          const { taskId: rollbackTaskId } = await deleteMutation.mutateAsync({
            deviceId,
            objectPath: `${objectPath}${inst}.`,
          });
          // 等设备真删完再 refetch。过滤状态继续保留,避免 schema 缓存滞后导致已删行回显。
          // 失败/超时仅 warn,且保留过滤状态(用户仍看不到该行,直到下次主动刷新)。
          try {
            await waitForTaskTerminal(rollbackTaskId);
            if (!cancelled) {
              await syncRelatedParameters();
            }
          } catch (waitErr) {
            console.warn('[MultiInstanceTable] silent add-rollback wait failed', { inst, waitErr });
          }
        } catch (err) {
          // 静默策略:回滚入队失败也不打扰用户,只在 console 里留痕便于排查。
          console.warn('[MultiInstanceTable] silent add-rollback enqueue failed', { inst, err });
        }
      }
      if (!cancelled && !didRollbackAddedInstance) {
        try {
          const syncResult = await syncRelatedParameters();
          if (lastTask.status === 'completed' && group.id === 'enb-plmn') {
            const plmnParameters = (syncResult.schema?.parameters ?? [])
              .filter(
                (item) => item.path.startsWith(objectPath) && item.path.endsWith('.PLMNID'),
              )
              .map((item) => ({
                parameterPath: item.path,
                parameterValue: String(item.currentValue ?? ''),
              }));
            if (syncResult.schema && plmnParameters.length > 0) {
              applyDeviceParameterSearchReadback(queryClient, {
                deviceId,
                searchQuery: 'PLMNList',
                replacePathPrefix: objectPath,
                parameters: plmnParameters,
              });
            } else {
              await refreshDeviceParameterSearchQueries(queryClient, deviceId);
            }
          }
          if (!cancelled && lastAction?.taskId === lastTask.id) {
            if (isIpsecGroup && lastAction.ipsecTargetEnabled !== undefined) {
              const globalParams = await deviceParameterApi.searchParameters(
                deviceId,
                ipsecGlobalEnableQuery,
                20,
              );
              const globalValue = globalParams.find(
                (item) => item.parameterPath === ipsecGlobalEnablePath,
              )?.parameterValue;
              if (
                globalValue === undefined
                || isEnabledValue(globalValue) !== lastAction.ipsecTargetEnabled
              ) {
                throw new Error(t('device.ipsec.globalReadbackMismatch'));
              }
            }
            const refreshedObject = syncResult.schema?.objects.find((o) => o.path === objectPath);
            const refreshedInstances = refreshedObject?.currentInstances ?? [];
            const listMatchesAction = (() => {
              if (lastTask.status !== 'completed') return false;
              if (lastAction.syncedForTaskId === lastTask.id) return false;
              if (syncedSuccessNotifiedTaskIdsRef.current.has(lastTask.id)) return false;
              if (lastAction.action === 'save' && lastAction.saveMode === 'add') {
                const inst = lastAction.instanceNumber;
                return typeof inst === 'number' && refreshedInstances.includes(inst);
              }
              const addedIds = lastAction.addedInstIds ?? (
                lastAction.action === 'add' ? (lastAction.savedInstIds ?? []) : []
              );
              const deletedIds = lastAction.deletedInstIds ?? (
                lastAction.action === 'delete' ? (deleteInstIds ?? []) : []
              );
              const editedIds = lastAction.editedInstIds ?? (
                lastAction.action === 'save' ? (lastAction.savedInstIds ?? []) : []
              );
              if (addedIds.length > 0 || deletedIds.length > 0 || editedIds.length > 0) {
                const addedOk = addedIds.every((instId) => /^\d+$/.test(instId) && refreshedInstances.includes(Number(instId)));
                const deletedOk = deletedIds.every((instId) => /^\d+$/.test(instId) && !refreshedInstances.includes(Number(instId)));
                return addedOk && deletedOk;
              }
              return lastAction.action === 'save';
            })();
            if (listMatchesAction) {
              syncedSuccessNotifiedTaskIdsRef.current.add(lastTask.id);
              patchFeedback(fbKey, { syncedForTaskId: lastTask.id });
              clearLocalBatchChanges();
              message.success({
                content: lastAction.detail,
                duration: 6,
              });
            }
          }
        } catch (err) {
          if (!cancelled) {
            const errMsg = err instanceof Error ? err.message : String(err);
            notification.error({
              message: t('device.multi.readbackFailed', { group: group.titleZh }),
              description: errMsg,
              duration: ERROR_FEEDBACK_DURATION_SECONDS,
            });
          }
        }
      }
    })();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [active, lastTask?.id, lastTask?.status]);

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

  // 单层确认：外层 Popconfirm 已二次确认，这里只标记为待删除，统一由批量提交下发。
  const handleDelete = async (instId: string) => {
    if (isIpsecGroup && !ipsecGlobalEnabled) {
      return;
    }
    if (pendingAdds.some((row) => row.tempId === instId)) {
      setPendingAdds((prev) => prev.filter((row) => row.tempId !== instId));
      setRowEdits((prev) => {
        const next = new Map(prev);
        next.delete(instId);
        return next;
      });
      return;
    }
    setPendingDeletes((prev) => {
      const next = new Set(prev);
      next.add(instId);
      return next;
    });
    clearDraftPrefix(fbKey, `${instId}.`);
    setRowEdits((prev) => {
      const next = new Map(prev);
      next.delete(instId);
      return next;
    });
  };

  const buildUpdatesForRow = useCallback((instId: string, edits: Record<string, string>): ParameterUpdateRequest[] => {
    const updates: ParameterUpdateRequest[] = [];
    const valuesForVisibility = isGnbNetworkAddressGroup(group.id)
      ? {
          ...edits,
          [gnbNetworkAddressingTypeLeaf]: edits[gnbNetworkAddressingTypeLeaf]
            ?? cellValue(instId, gnbNetworkAddressingTypeLeaf),
        }
      : edits;
    for (const [leaf, value] of Object.entries(edits)) {
      if (!isGnbNetworkIpv4FieldVisible(group.id, leaf, valuesForVisibility, gnbNetworkAddressingTypeLeaf)) continue;
      const item = schemaByPath.get(`${objectPath}${instId}.${leaf}`) ?? leafSchemaByLeaf.get(leaf);
      const param = groupParamByLeaf.get(leaf);
      const nextValue = normalizeComparableQuickSettingsValue(value, param);
      const oldVal = normalizeComparableQuickSettingsValue(item?.currentValue ?? '', param);
      if (nextValue === oldVal) continue;
      updates.push({
        parameterPath: `${objectPath}${instId}.${leaf}`,
        parameterValue: nextValue,
        parameterType: effectiveParamType(item, param),
      });
    }
    return updates;
  }, [cellValue, gnbNetworkAddressingTypeLeaf, group.id, groupParamByLeaf, leafSchemaByLeaf, objectPath, schemaByPath]);

  const displayColumns = useMemo<SpecialColumnSpec[]>(() => {
    if (!specialColumns) {
      return group.params.map((param) => ({
        key: param.leaf || param.name,
        leaf: param.leaf || '',
        titleZh: param.titleZh,
        titleEn: param.titleEn,
      }));
    }
    return specialColumns.map((column) => {
      const directParam = group.params.find((param) => (param.leaf || param.name) === column.leaf);
      const semanticParam = group.params.find((param) => param.name === column.key);
      const param = directParam ?? semanticParam;
      if (!param?.leaf || param.leaf === column.leaf) return column;
      return { ...column, leaf: param.leaf };
    });
  }, [group.params, specialColumns]);

  const neighborEnbTypeLeaf = group.params.find(
    (param) => param.name === 'NeighborCellEnbType',
  )?.leaf;
  const usesSplitNeighborCellIdentity = group.id === NEIGHBOR_CELL_GROUP_ID
    && groupParamLeafSet.has(NEIGHBOR_CELL_ECI_LEAF);

  const addModalColumns = useMemo<SpecialColumnSpec[]>(() => {
    if (!usesSplitNeighborCellIdentity) return displayColumns;
    const byLeaf = new Map(displayColumns.map((column) => [column.leaf, column]));
    const requiredUnsigned = (maxValue: number): SpecialColumnSpec['virtual'] => ({
      type: 'unsignedInt',
      constraints: { minValue: 0, maxValue },
      required: true,
    });
    return [
      {
        key: NEIGHBOR_ENB_ID_FIELD,
        leaf: NEIGHBOR_ENB_ID_FIELD,
        titleKey: 'device.multi.neighborEnbId',
        titleEn: 'eNB ID',
        virtual: requiredUnsigned(1048575),
      },
      {
        key: NEIGHBOR_LOCAL_CELL_ID_FIELD,
        leaf: NEIGHBOR_LOCAL_CELL_ID_FIELD,
        titleKey: 'device.multi.neighborCellId',
        titleEn: 'Cell ID',
        virtual: requiredUnsigned(255),
      },
      byLeaf.get('EUTRACarrierARFCN'),
      byLeaf.get('PhyCellID'),
      byLeaf.get('QOffset'),
      byLeaf.get('CIO'),
      byLeaf.get(group.params.find((param) => param.name === 'TAC')?.leaf),
      neighborEnbTypeLeaf ? byLeaf.get(neighborEnbTypeLeaf) : undefined,
      byLeaf.get('X2Flag'),
    ].filter((column): column is SpecialColumnSpec => Boolean(column));
  }, [displayColumns, group.params, neighborEnbTypeLeaf, usesSplitNeighborCellIdentity]);

  const openEditModal = useCallback((row: TableRow) => {
    if (!row.instanceId) return;
    const values: Record<string, string> = {};
    displayColumns.forEach((column) => {
      if (!column.leaf) return;
      values[column.leaf] = isIpsecGroup && column.leaf === IPSEC_ENABLE_LEAF
        ? normalizeIpsecEnableValue(cellValue(row.instanceId!, column.leaf))
        : cellValue(row.instanceId!, column.leaf);
    });
    setEditModal({ mode: 'edit', instanceId: row.instanceId, values, errors: {} });
  }, [cellValue, displayColumns, isIpsecGroup]);

  const openAddModal = useCallback(() => {
    const initialValues = buildInitialEditValues();
    if (usesSplitNeighborCellIdentity) {
      initialValues[NEIGHBOR_ENB_ID_FIELD] = '';
      initialValues[NEIGHBOR_LOCAL_CELL_ID_FIELD] = '';
    }
    if (isIpsecGroup) {
      initialValues[IPSEC_ENABLE_LEAF] = normalizeIpsecEnableValue(initialValues[IPSEC_ENABLE_LEAF]);
    }
    setEditModal({ mode: 'add', values: initialValues, errors: {} });
  }, [buildInitialEditValues, isIpsecGroup, usesSplitNeighborCellIdentity]);

  const setEditModalValue = useCallback((leaf: string, value: string) => {
    setEditModal((prev) => {
      if (!prev) return prev;
      const virtual = prev.mode === 'add'
        ? addModalColumns.find((column) => column.leaf === leaf)?.virtual
        : undefined;
      const item = prev.instanceId
        ? (schemaByPath.get(`${objectPath}${prev.instanceId}.${leaf}`) ?? leafSchemaByLeaf.get(leaf))
        : leafSchemaByLeaf.get(leaf);
      const param = groupParamByLeaf.get(leaf);
      const normalizedValue = isIpsecGroup && leaf === IPSEC_ENABLE_LEAF
        ? toDeviceIpsecEnableValue(value)
        : value;
      const err = virtual
        ? validateQuickSettingsCellValue(leaf, normalizedValue, virtual.type, virtual.constraints) ?? ''
        : validateQuickSettingsCellValue(leaf, normalizedValue, effectiveParamType(item, param), effectiveParamConstraints(item, param)) ?? '';
      return {
        ...prev,
        values: { ...prev.values, [leaf]: value },
        errors: { ...prev.errors, [leaf]: err },
      };
    });
  }, [addModalColumns, groupParamByLeaf, isIpsecGroup, leafSchemaByLeaf, objectPath, schemaByPath]);

  const closeEditModal = useCallback(() => {
    if (updateMutation.isPending || isSubmitting) return;
    setEditModal(null);
  }, [updateMutation.isPending, isSubmitting]);

  const validateEditModalValues = useCallback((
    modal: EditModalState,
    targetInstanceId?: string,
  ): { errors: Record<string, string>; updates: ParameterUpdateRequest[]; pendingEdits: Record<string, string> } => {
    const errors: Record<string, string> = {};
    const updates: ParameterUpdateRequest[] = [];
    const pendingEdits: Record<string, string> = {};
    const modalColumns = modal.mode === 'add' ? addModalColumns : displayColumns;
    let composedNeighborEci = '';
    const comparableInstIds = tableRows
      .map((row) => row.instanceId)
      .filter((instId): instId is string => Boolean(instId) && instId !== modal.instanceId);

    if (modal.mode === 'add' && group.id === INTER_FREQ_GROUP_ID) {
      const newEarfcn = String(modal.values[INTER_FREQ_EARFCN_LEAF] ?? '').trim();
      if (newEarfcn && comparableInstIds.some((instId) => cellValue(instId, INTER_FREQ_EARFCN_LEAF).trim() === newEarfcn)) {
        errors[INTER_FREQ_EARFCN_LEAF] = t('device.multi.interFreqDuplicate', { value: newEarfcn });
      }
    }

    if (modal.mode === 'add' && group.id === NEIGHBOR_CELL_GROUP_ID) {
      let duplicateLeaves: readonly string[] = NEIGHBOR_CELL_DUPLICATE_LEAVES;
      let duplicateValues = duplicateLeaves.map((leaf) => String(modal.values[leaf] ?? '').trim());
      if (usesSplitNeighborCellIdentity) {
        const enbId = String(modal.values[NEIGHBOR_ENB_ID_FIELD] ?? '').trim();
        const cellId = String(modal.values[NEIGHBOR_LOCAL_CELL_ID_FIELD] ?? '').trim();
        if (enbId && cellId) composedNeighborEci = composeLteEci(enbId, cellId);
        duplicateLeaves = [NEIGHBOR_CELL_ECI_LEAF, 'EUTRACarrierARFCN', 'PhyCellID'];
        duplicateValues = [
          composedNeighborEci,
          String(modal.values.EUTRACarrierARFCN ?? '').trim(),
          String(modal.values.PhyCellID ?? '').trim(),
        ];
      }
      const hasIdentity = duplicateValues.every(Boolean);
      const duplicated = hasIdentity && comparableInstIds.some((instId) =>
        duplicateLeaves.every((leaf, idx) => cellValue(instId, leaf).trim() === duplicateValues[idx]),
      );
      if (duplicated) {
        const msg = t('device.multi.neighborCellDuplicate');
        for (const leaf of usesSplitNeighborCellIdentity
          ? [NEIGHBOR_ENB_ID_FIELD, NEIGHBOR_LOCAL_CELL_ID_FIELD, 'EUTRACarrierARFCN', 'PhyCellID']
          : NEIGHBOR_CELL_DUPLICATE_LEAVES) {
          errors[leaf] = errors[leaf] ?? msg;
        }
      }
    }

    if (group.id === GNB_NR_NEIGHBOR_CELL_GROUP_ID) {
      const ssbFrequency = String(modal.values[GNB_NR_NEIGHBOR_SSB_LEAF] ?? '').trim();
      if (ssbFrequency) {
        if (nrInterFreqLoading || !nrInterFreqSchemaResp) {
          errors[GNB_NR_NEIGHBOR_SSB_LEAF] = errors[GNB_NR_NEIGHBOR_SSB_LEAF] ?? t('device.multi.nrInterFreqLoading');
        } else if (!nrInterFreqSsbState.known.has(ssbFrequency)) {
          errors[GNB_NR_NEIGHBOR_SSB_LEAF] = errors[GNB_NR_NEIGHBOR_SSB_LEAF] ?? t('device.multi.nrInterFreqMissing', { value: ssbFrequency });
        } else if (!nrInterFreqSsbState.enabled.has(ssbFrequency)) {
          errors[GNB_NR_NEIGHBOR_SSB_LEAF] = errors[GNB_NR_NEIGHBOR_SSB_LEAF] ?? t('device.multi.nrInterFreqDisabled', { value: ssbFrequency });
        }
      }
    }

    for (const column of modalColumns) {
      const leaf = column.leaf || '';
      if (!leaf || column.readOnly) continue;
      if (!isGnbNetworkIpv4FieldVisible(group.id, leaf, modal.values, gnbNetworkAddressingTypeLeaf)) continue;

      const rawValue = modal.values[leaf] ?? '';
      if (column.virtual) {
        if (String(rawValue).trim() === '') {
          if (column.virtual.required) {
            errors[leaf] = errors[leaf] ?? t('device.multi.packed.fieldRequired');
          }
          continue;
        }
        const err = validateQuickSettingsCellValue(
          leaf,
          rawValue,
          column.virtual.type,
          column.virtual.constraints,
        );
        if (err) errors[leaf] = errors[leaf] ?? err;
        continue;
      }
      if (!groupParamLeafSet.has(leaf)) continue;
      const param = groupParamByLeaf.get(leaf);
      if (modal.mode === 'add' && String(rawValue).trim() === '') {
        if (param?.required) {
          errors[leaf] = errors[leaf] ?? t('device.multi.packed.fieldRequired');
        }
        continue;
      }
      const item = targetInstanceId
        ? (schemaByPath.get(`${objectPath}${targetInstanceId}.${leaf}`) ?? leafSchemaByLeaf.get(leaf))
        : leafSchemaByLeaf.get(leaf);
      const value = isIpsecGroup && leaf === IPSEC_ENABLE_LEAF
        ? toDeviceIpsecEnableValue(rawValue)
        : rawValue;
      const parameterType = effectiveParamType(item, param);
      const constraints = effectiveParamConstraints(item, param);
      const err = validateQuickSettingsCellValue(leaf, value, parameterType, constraints);
      if (err) {
        errors[leaf] = errors[leaf] ?? err;
        continue;
      }

      if (!targetInstanceId) {
        pendingEdits[leaf] = normalizeComparableQuickSettingsValue(value, param);
        continue;
      }

      const nextValue = normalizeComparableQuickSettingsValue(value, param);
      const oldVal = normalizeComparableQuickSettingsValue(item?.currentValue ?? '', param);
      if (nextValue === oldVal) continue;

      pendingEdits[leaf] = nextValue;
      updates.push({
        parameterPath: `${objectPath}${targetInstanceId}.${leaf}`,
        parameterValue: nextValue,
        parameterType,
      });
    }

    if (
      modal.mode === 'add'
      && usesSplitNeighborCellIdentity
      && composedNeighborEci
      && !errors[NEIGHBOR_ENB_ID_FIELD]
      && !errors[NEIGHBOR_LOCAL_CELL_ID_FIELD]
    ) {
      pendingEdits[NEIGHBOR_CELL_ECI_LEAF] = composedNeighborEci;
    }

    return { errors, updates, pendingEdits };
  }, [addModalColumns, cellValue, displayColumns, gnbNetworkAddressingTypeLeaf, group.id, groupParamByLeaf, groupParamLeafSet, isIpsecGroup, leafSchemaByLeaf, nrInterFreqLoading, nrInterFreqSchemaResp, nrInterFreqSsbState.enabled, nrInterFreqSsbState.known, objectPath, schemaByPath, tableRows, t, usesSplitNeighborCellIdentity]);

  const rollbackAddedInstance = useCallback(async (instId: string | undefined, reason: string) => {
    if (!instId || !/^\d+$/.test(instId)) return;
    const inst = Number(instId);
    setOptimisticallyRemoved((prev) => {
      if (prev.has(inst)) return prev;
      const next = new Set(prev);
      next.add(inst);
      return next;
    });
    clearDraftPrefix(fbKey, `${inst}.`);
    setRowEdits((prev) => {
      const next = new Map(prev);
      next.delete(String(inst));
      return next;
    });
    try {
      const { taskId: rollbackTaskId } = await deleteMutation.mutateAsync({
        deviceId,
        objectPath: `${objectPath}${inst}.`,
      });
      await waitForTaskTerminal(rollbackTaskId);
      await syncRelatedParameters({ preserveLocalEdits: isIpsecGroup });
    } catch (err) {
      if (isObjectInstanceNotFoundError(err)) {
        await syncRelatedParameters({ preserveLocalEdits: isIpsecGroup });
        return;
      }
      console.warn('[MultiInstanceTable] add rollback failed', { inst, reason, err });
      notification.warning({
        message: t('device.multi.addRollbackFailed', { instId: inst, group: group.titleZh }),
        description: t('device.multi.addRollbackQueueFailedDesc', { instId: inst, err: err instanceof Error ? err.message : String(err) }),
        duration: 6,
      });
    }
  }, [clearDraftPrefix, deleteMutation, deviceId, fbKey, group.titleZh, isIpsecGroup, objectPath, syncRelatedParameters, t, waitForTaskTerminal]);

  const handleSaveEditModal = useCallback(async () => {
    if (!editModal) return;
    const targetInstanceId = editModal.instanceId;
    const isPendingAdd = Boolean(targetInstanceId && pendingAdds.some((row) => row.tempId === targetInstanceId));
    const { errors, updates, pendingEdits } = validateEditModalValues(
      editModal,
      editModal.mode === 'add' || isPendingAdd ? undefined : targetInstanceId,
    );

    if (Object.keys(errors).length > 0) {
      setEditModal((prev) => prev ? { ...prev, errors } : prev);
      message.error({ content: t('device.multi.editValidationFailed'), duration: ERROR_FEEDBACK_DURATION_SECONDS });
      return;
    }

    if (editModal.mode === 'add') {
      const tempId = `new:${Date.now()}:${pendingAddCounterRef.current++}`;
      setPendingAdds((prev) => [...prev, { tempId, values: pendingEdits }]);
      setEditModal(null);
      message.success({ content: t('device.multi.batchStaged'), duration: 3 });
      return;
    }

    if (!targetInstanceId) return;

    if (isPendingAdd) {
      setPendingAdds((prev) => prev.map((row) => row.tempId === targetInstanceId ? { ...row, values: pendingEdits } : row));
      setEditModal(null);
      message.success({ content: t('device.multi.batchStaged'), duration: 3 });
      return;
    }

    if (updates.length === 0) {
      setRowEdits((prev) => {
        const next = new Map(prev);
        next.delete(targetInstanceId);
        return next;
      });
      clearDraftPrefix(fbKey, `${targetInstanceId}.`);
      setEditModal(null);
      message.info({ content: t('device.multi.noRowChange'), duration: 4 });
      return;
    }

    setRowEdits((prev) => {
      const next = new Map(prev);
      next.set(targetInstanceId, { edits: pendingEdits, errors: {} });
      return next;
    });
    Object.entries(pendingEdits).forEach(([leaf, value]) => {
      setDraftField(fbKey, `${targetInstanceId}.${leaf}`, value);
    });
    setEditModal(null);
    message.success({ content: t('device.multi.batchStaged'), duration: 3 });
  }, [clearDraftPrefix, editModal, fbKey, pendingAdds, setDraftField, t, validateEditModalValues]);

  const tunnelPendingChangeCount = pendingAdds.length + pendingDeletes.size + rowEdits.size;
  const pendingChangeCount = tunnelPendingChangeCount + (hasIpsecGlobalChange ? 1 : 0);
  const batchAwaitingDevice = Boolean(
    (
      isIpsecGroup
      && (
        lastAction?.ipsecOperationStatus === 'running'
        || lastAction?.ipsecOperationStatus === 'awaiting-readback'
      )
    )
    || (
      lastAction?.submitStatus === 'queued'
      && lastAction.taskId
      && lastAction.syncedForTaskId !== lastAction.taskId
      && (!lastTask || !isDeviceTaskTerminal(lastTask.status) || lastTask.status === 'completed')
    ),
  );

  const handleSubmitBatch = useCallback(async () => {
    if (pendingChangeCount === 0 || submittingRef.current || batchAwaitingDevice) return;
    if (!isIpsecGlobalStateReady) {
      notification.error({
        message: t('device.ipsec.unifiedSubmitFailed'),
        description: t('device.ipsec.globalStateUnavailable'),
        duration: ERROR_FEEDBACK_DURATION_SECONDS,
      });
      return;
    }
    const deletedInstIds = Array.from(pendingDeletes);
    const editedRows = Array.from(rowEdits.entries());
    const addRows = [...pendingAdds];
    const validationErrors: string[] = [];
    const validateDraftCell = (label: string, leaf: string, value: string, targetInstId?: string): string | null => {
      const item = targetInstId
        ? (schemaByPath.get(`${objectPath}${targetInstId}.${leaf}`) ?? leafSchemaByLeaf.get(leaf))
        : leafSchemaByLeaf.get(leaf);
      const param = groupParamByLeaf.get(leaf);
      if (String(value ?? '').trim() === '') {
        if (param?.required) return `${label}.${leaf}: ${t('device.multi.packed.fieldRequired')}`;
        return null;
      }
      const err = validateQuickSettingsCellValue(
        leaf,
        value,
        effectiveParamType(item, param),
        effectiveParamConstraints(item, param),
      );
      return err ? `${label}.${leaf}: ${err}` : null;
    };

    addRows.forEach((row, idx) => {
      Object.entries(row.values).forEach(([leaf, value]) => {
        if (!isGnbNetworkIpv4FieldVisible(group.id, leaf, row.values, gnbNetworkAddressingTypeLeaf)) return;
        const err = validateDraftCell(`+${idx + 1}`, leaf, value);
        if (err) validationErrors.push(err);
      });
    });
    editedRows.forEach(([instId, state]) => {
      const valuesForVisibility = isGnbNetworkAddressGroup(group.id)
        ? {
            ...state.edits,
            [gnbNetworkAddressingTypeLeaf]: state.edits[gnbNetworkAddressingTypeLeaf]
              ?? cellValue(instId, gnbNetworkAddressingTypeLeaf),
          }
        : state.edits;
      Object.entries(state.edits).forEach(([leaf, value]) => {
        if (!isGnbNetworkIpv4FieldVisible(group.id, leaf, valuesForVisibility, gnbNetworkAddressingTypeLeaf)) return;
        const err = validateDraftCell(instId, leaf, value, instId);
        if (err) validationErrors.push(err);
      });
    });
    if (group.id === 'enb-plmn') {
      const finalPlmnRows = buildEffectivePlmnRows({
        existingRows: instanceIds.map((instId) => ({
          key: instId,
          plmn: schemaByPath.get(`${objectPath}${instId}.PLMNID`)?.currentValue ?? '',
        })),
        deletedKeys: pendingDeletes,
        editedValues: new Map(editedRows.map(([instId, state]) => [
          instId,
          state.edits.PLMNID
            ?? schemaByPath.get(`${objectPath}${instId}.PLMNID`)?.currentValue
            ?? '',
        ])),
        addedRows: addRows.map((row) => ({
          key: row.tempId,
          plmn: row.values.PLMNID ?? '',
        })),
      });
      const plmnError = validatePlmnList(finalPlmnRows, group.maxInstances ?? 6);
      if (plmnError === 'format') {
        validationErrors.push(t('device.cell.plmnFormatInvalid'));
      } else if (plmnError === 'duplicate') {
        validationErrors.push(t('device.cell.plmnDuplicate'));
      } else if (plmnError === 'limit') {
        validationErrors.push(t('device.cell.plmnLimitReached', { max: group.maxInstances ?? 6 }));
      }
    }

    if (validationErrors.length > 0) {
      const detail = validationErrors.slice(0, 4).join('; ');
      notification.error({
        message: t('device.multi.batchSubmitFailed', { group: group.titleZh }),
        description: validationErrors.length > 4 ? `${detail}; ...` : detail,
        duration: ERROR_FEEDBACK_DURATION_SECONDS,
      });
      return;
    }

    const ipsecPlan = isIpsecGroup
      ? buildIpsecSubmissionPlan({
          currentEnabled: currentIpsecEnabled,
          targetEnabled: targetIpsecEnabled,
          hasGlobalChange: hasIpsecGlobalChange,
          hasTunnelChanges: tunnelPendingChangeCount > 0,
        })
      : null;
    if (ipsecPlan && !ipsecPlan.ok) {
      notification.error({
        message: t('device.ipsec.unifiedSubmitFailed'),
        description: t('device.ipsec.tunnelRequiresEnabled'),
        duration: ERROR_FEEDBACK_DURATION_SECONDS,
      });
      return;
    }

    submittingRef.current = true;
    setIsSubmitting(true);
    const savedInstIds: string[] = [];
    const submittedAddedInstIds: string[] = [];
    const submittedDeletedInstIds: string[] = [];
    const submittedEditedInstIds: string[] = [];
    const submittedTaskIds: string[] = [];
    const completedIpsecPhases: IpsecSubmissionPhase[] = [];
    let activeIpsecPhase: IpsecSubmissionPhase | undefined;
    let lastTaskId: string | undefined;
    let addCreatedInst: string | undefined;
    let currentAddTempId: string | undefined;
    const submittedTunnelDraftRevision = tunnelDraftRevision;
    const submittedControlDraftRevision = ipsecControlDraftRevision;
    const batchAction = addRows.length > 0 ? 'add' : deletedInstIds.length > 0 ? 'delete' : 'save';

    if (isIpsecGroup) {
      setFeedback(fbKey, {
        kind: 'multi',
        action: batchAction,
        submitStatus: 'queued',
        pendingAddRows: addRows,
        pendingDeleteInstIds: deletedInstIds,
        ipsecTaskIds: [],
        ipsecTargetEnabled: targetIpsecEnabled,
        ipsecCompletedPhases: [],
        ipsecOperationStatus: 'running',
        detail: t('device.ipsec.preparingSubmit'),
        at: Date.now(),
      });
    }

    const recordTask = (taskId: string | undefined) => {
      lastTaskId = taskId;
      if (!taskId) return;
      submittedTaskIds.push(taskId);
      if (isIpsecGroup) {
        patchFeedback(fbKey, {
          taskId,
          ipsecTaskIds: [...submittedTaskIds],
          ipsecActivePhase: activeIpsecPhase,
        });
      }
    };

    const requireCompletedTask = async (taskId: string, phase: IpsecSubmissionPhase) => {
      const task = await waitForTaskTerminal(taskId);
      if (task.status !== 'completed') {
        const fallbackKey = phase === 'enable-global'
          ? 'device.ipsec.enableFailed'
          : phase === 'disable-global'
            ? 'device.ipsec.disableFailed'
            : 'device.ipsec.tunnelFailed';
        throw new Error(task.errorMessage || t(fallbackKey));
      }
    };

    const submitGlobalPhase = async (phase: 'enable-global' | 'disable-global') => {
      const result = await updateMutation.mutateAsync({
        deviceId,
        parameters: [{
          parameterPath: ipsecGlobalEnablePath,
          parameterValue: targetIpsecEnabled ? '1' : '0',
          parameterType: 'boolean',
        }],
      });
      recordTask(result.taskId);
      if (result.taskId) {
        await requireCompletedTask(result.taskId, phase);
      }
    };

    const submitTunnelChanges = async () => {
      for (const instId of deletedInstIds) {
        try {
          const result = await deleteMutation.mutateAsync({ deviceId, objectPath: `${objectPath}${instId}.` });
          recordTask(result.taskId);
          if (isIpsecGroup && result.taskId) {
            await requireCompletedTask(result.taskId, 'apply-tunnels');
          }
          savedInstIds.push(instId);
          submittedDeletedInstIds.push(instId);
          if (isIpsecGroup) {
            patchFeedback(fbKey, {
              pendingDeleteInstIds: deletedInstIds.filter(
                (pendingId) => !submittedDeletedInstIds.includes(pendingId),
              ),
            });
          }
          hideStaleInstance(instId);
        } catch (err) {
          if (isObjectInstanceNotFoundError(err)) {
            hideStaleInstance(instId);
            savedInstIds.push(instId);
            submittedDeletedInstIds.push(instId);
            if (isIpsecGroup) {
              patchFeedback(fbKey, {
                pendingDeleteInstIds: deletedInstIds.filter(
                  (pendingId) => !submittedDeletedInstIds.includes(pendingId),
                ),
              });
            }
            continue;
          }
          throw err;
        }
      }

      const editUpdates = editedRows.flatMap(([instId, state]) => buildUpdatesForRow(instId, state.edits));
      if (editUpdates.length > 0) {
        const result = await updateMutation.mutateAsync({ deviceId, parameters: editUpdates });
        recordTask(result.taskId);
        if (isIpsecGroup && result.taskId) {
          await requireCompletedTask(result.taskId, 'apply-tunnels');
        }
        editedRows.forEach(([instId]) => {
          savedInstIds.push(instId);
          submittedEditedInstIds.push(instId);
        });
      }

      for (const row of addRows) {
        currentAddTempId = row.tempId;
        const addResult = await addMutation.mutateAsync({ deviceId, objectPath });
        recordTask(addResult.taskId);
        const addTask = await waitForTaskTerminal(addResult.taskId);
        if (addTask.status !== 'completed') {
          throw new Error(addTask.errorMessage || t('device.multi.addInstanceFailed', { status: addTask.status }));
        }
        const instNumberRaw = addTask.result?.instance_number;
        if (typeof instNumberRaw !== 'number' || instNumberRaw <= 0) {
          throw new Error(t('device.multi.addInstanceNoId'));
        }
        addCreatedInst = String(instNumberRaw);
        const updates = Object.entries(row.values)
          .filter(([leaf]) => isGnbNetworkIpv4FieldVisible(group.id, leaf, row.values, gnbNetworkAddressingTypeLeaf))
          .map(([leaf, value]) => {
            const item = leafSchemaByLeaf.get(leaf);
            const param = groupParamByLeaf.get(leaf);
            return {
              parameterPath: `${objectPath}${addCreatedInst}.${leaf}`,
              parameterValue: value,
              parameterType: effectiveParamType(item, param),
            };
          });
        if (updates.length > 0) {
          const result = await updateMutation.mutateAsync({ deviceId, parameters: updates });
          recordTask(result.taskId);
          if (result.taskId) {
            const spvTask = await waitForTaskTerminal(result.taskId);
            if (spvTask.status !== 'completed') {
              throw new Error(spvTask.errorMessage || t('device.multi.addInstanceFailed', { status: spvTask.status }));
            }
          }
          savedInstIds.push(addCreatedInst);
          submittedAddedInstIds.push(addCreatedInst);
        }
        addCreatedInst = undefined;
        currentAddTempId = undefined;
      }
    };

    try {
      if (ipsecPlan?.ok) {
        await executeIpsecSubmissionPlan(ipsecPlan.phases, async (phase) => {
          activeIpsecPhase = phase;
          const phaseDetailKey = phase === 'enable-global'
            ? 'device.ipsec.enabling'
            : phase === 'disable-global'
              ? 'device.ipsec.disabling'
              : 'device.ipsec.applyingTunnels';
          patchFeedback(fbKey, {
            ipsecActivePhase: phase,
            detail: t(phaseDetailKey),
          });
          if (phase === 'apply-tunnels') {
            await submitTunnelChanges();
          } else {
            await submitGlobalPhase(phase);
          }
          completedIpsecPhases.push(phase);
          activeIpsecPhase = undefined;
          patchFeedback(fbKey, {
            ipsecActivePhase: undefined,
            ipsecCompletedPhases: [...completedIpsecPhases],
          });
        });
      } else {
        await submitTunnelChanges();
      }
      const submittedDetail = isIpsecGroup
        ? t('device.ipsec.unifiedSubmitted', { count: pendingChangeCount })
        : t('device.multi.batchSubmitted', { count: pendingChangeCount });

      if (isIpsecGroup) {
        patchFeedback(fbKey, {
          taskId: lastTaskId,
          ipsecTaskIds: [...submittedTaskIds],
          ipsecCompletedPhases: [...completedIpsecPhases],
          ipsecActivePhase: undefined,
          ipsecOperationStatus: 'awaiting-readback',
          detail: t('device.ipsec.confirmingReadback'),
        });
        const syncResult = await syncRelatedParameters({ preserveLocalEdits: true });
        const globalParams = await deviceParameterApi.searchParameters(
          deviceId,
          ipsecGlobalEnableQuery,
          20,
        );
        const globalValue = globalParams.find(
          (item) => item.parameterPath === ipsecGlobalEnablePath,
        )?.parameterValue;
        if (
          globalValue === undefined
          || isEnabledValue(globalValue) !== targetIpsecEnabled
        ) {
          throw new Error(t('device.ipsec.globalReadbackMismatch'));
        }
        queryClient.setQueryData(
          ['devices', 'parameters', 'search', deviceId, ipsecGlobalEnableQuery, 20],
          globalParams,
        );
        await queryClient.invalidateQueries({
          queryKey: ['devices', 'parameter-schema', deviceId],
        });
        const refreshedInstances = syncResult.schema?.objects.find(
          (item) => item.path === objectPath,
        )?.currentInstances ?? [];
        const addedOk = submittedAddedInstIds.every(
          (instId) => /^\d+$/.test(instId) && refreshedInstances.includes(Number(instId)),
        );
        const deletedOk = submittedDeletedInstIds.every(
          (instId) => /^\d+$/.test(instId) && !refreshedInstances.includes(Number(instId)),
        );
        if (!addedOk || !deletedOk) {
          throw new Error(t('device.multi.readbackFailed', { group: group.titleZh }));
        }

        const latestStore = useQuickSettingsFeedbackStore.getState();
        const tunnelDraftUnchanged = (latestStore.draftRevisions[fbKey] ?? 0)
          === submittedTunnelDraftRevision;
        const controlDraftUnchanged = (latestStore.draftRevisions[ipsecControlDraftKey] ?? 0)
          === submittedControlDraftRevision;
        if (tunnelDraftUnchanged) {
          clearTunnelLocalBatchChanges();
        }
        if (controlDraftUnchanged) {
          clearDraft(ipsecControlDraftKey);
        }
        setFeedback(fbKey, {
          kind: 'multi',
          action: batchAction,
          submitStatus: 'queued',
          taskId: lastTaskId,
          savedInstIds,
          addedInstIds: submittedAddedInstIds,
          deletedInstIds: submittedDeletedInstIds,
          editedInstIds: submittedEditedInstIds,
          pendingAddRows: tunnelDraftUnchanged ? undefined : addRows,
          pendingDeleteInstIds: tunnelDraftUnchanged ? undefined : deletedInstIds.filter(
            (instId) => !submittedDeletedInstIds.includes(instId),
          ),
          syncedForTaskId: lastTaskId,
          ipsecTaskIds: [...submittedTaskIds],
          ipsecTargetEnabled: targetIpsecEnabled,
          ipsecCompletedPhases: [...completedIpsecPhases],
          ipsecOperationStatus: 'completed',
          detail: submittedDetail,
          at: Date.now(),
        });
        message.success({ content: submittedDetail, duration: 4 });
      } else {
        setFeedback(fbKey, {
          kind: 'multi',
          action: batchAction,
          submitStatus: 'queued',
          taskId: lastTaskId,
          savedInstIds,
          addedInstIds: submittedAddedInstIds,
          deletedInstIds: submittedDeletedInstIds,
          editedInstIds: submittedEditedInstIds,
          pendingAddRows: addRows,
          detail: submittedDetail,
          at: Date.now(),
        });
        message.success({ content: submittedDetail, duration: 4 });
      }
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : String(err);
      if (addCreatedInst) {
        await rollbackAddedInstance(addCreatedInst, errMsg);
      }
      if (currentAddTempId && !isIpsecGroup) {
        setPendingAdds((prev) => prev.filter((row) => row.tempId !== currentAddTempId));
      }
      if (isIpsecGroup) {
        try {
          await syncRelatedParameters({ preserveLocalEdits: true });
        } catch (syncErr) {
          console.warn('[MultiInstanceTable] IPSec failure refresh failed', { objectPath, syncErr });
        }
      }
      setFeedback(fbKey, {
        kind: 'multi',
        action: batchAction,
        submitStatus: 'failed_to_queue',
        taskId: lastTaskId,
        savedInstIds,
        addedInstIds: submittedAddedInstIds,
        deletedInstIds: submittedDeletedInstIds,
        editedInstIds: submittedEditedInstIds,
        pendingAddRows: isIpsecGroup ? addRows : undefined,
        pendingDeleteInstIds: isIpsecGroup
          ? deletedInstIds.filter((instId) => !submittedDeletedInstIds.includes(instId))
          : undefined,
        ipsecTaskIds: isIpsecGroup ? submittedTaskIds : undefined,
        ipsecTargetEnabled: isIpsecGroup ? targetIpsecEnabled : undefined,
        ipsecCompletedPhases: isIpsecGroup ? completedIpsecPhases : undefined,
        ipsecFailedPhase: isIpsecGroup ? activeIpsecPhase : undefined,
        ipsecOperationStatus: isIpsecGroup ? 'failed' : undefined,
        ipsecActivePhase: isIpsecGroup ? activeIpsecPhase : undefined,
        detail: t('device.multi.detailFailed', { target: t('device.multi.batchSubmit'), err: errMsg }),
        at: Date.now(),
      });
      notification.error({
        message: isIpsecGroup
          ? t('device.ipsec.unifiedSubmitFailed')
          : t('device.multi.batchSubmitFailed', { group: group.titleZh }),
        description: errMsg,
        duration: ERROR_FEEDBACK_DURATION_SECONDS,
      });
    } finally {
      void queryClient.invalidateQueries({ queryKey: notificationKeys.all });
      submittingRef.current = false;
      setIsSubmitting(false);
    }
  }, [
    addMutation,
    batchAwaitingDevice,
    buildUpdatesForRow,
    cellValue,
    clearDraft,
    clearTunnelLocalBatchChanges,
    currentIpsecEnabled,
    deleteMutation,
    deviceId,
    fbKey,
    group.titleZh,
    group.id,
    gnbNetworkAddressingTypeLeaf,
    group.maxInstances,
    groupParamByLeaf,
    hasIpsecGlobalChange,
    hideStaleInstance,
    instanceIds,
    leafSchemaByLeaf,
    objectPath,
    pendingAdds,
    pendingChangeCount,
    pendingDeletes,
    ipsecControlDraftKey,
    ipsecControlDraftRevision,
    ipsecGlobalEnablePath,
    ipsecGlobalEnableQuery,
    isIpsecGlobalStateReady,
    isIpsecGroup,
    patchFeedback,
    queryClient,
    rollbackAddedInstance,
    rowEdits,
    schemaByPath,
    setFeedback,
    syncRelatedParameters,
    t,
    targetIpsecEnabled,
    tunnelDraftRevision,
    tunnelPendingChangeCount,
    updateMutation,
    waitForTaskTerminal,
  ]);

  const hasRows = tableRows.length > 0;
  const columns: ColumnType<TableRow>[] = [
    {
      title: t('device.multi.instance'),
      dataIndex: 'instanceId',
      key: 'instanceId',
      width: multiColumnWidth(hasRows, 80),
      fixed: hasRows ? 'left' : undefined,
      render: (_v: unknown, row: TableRow, idx: number) => {
        const displayId = row.pending === 'add' ? `+${idx + 1}` : row.instanceId;
        return (
          <Space size={4}>
            <Text strong>{displayId}</Text>
            {row.pending === 'add' && <Tag color="blue">{t('device.multi.pendingAdd')}</Tag>}
            {row.pending === 'edit' && <Tag color="gold">{t('device.multi.pendingEdit')}</Tag>}
          </Space>
        );
      },
    },
    ...displayColumns.map<ColumnType<TableRow>>((column) => {
      const leaf = column.leaf || '';
      // 列内字段约束在同一组所有实例下一致（schema 走 {i} 模板）；优先取首个有 schema 的实例作为模板。
      const titleHint = (() => {
        if (!leaf) return '';
        const param = groupParamByLeaf.get(leaf);
        for (const inst of instanceIds) {
          const tplItem = schemaByPath.get(`${objectPath}${inst}.${leaf}`);
          const hint = formatEffectiveConstraintHint(tplItem, param, t);
          if (hint) return hint;
        }
        return formatEffectiveConstraintHint(leafSchemaByLeaf.get(leaf), param, t);
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
        width: multiColumnWidth(hasRows, column.width ?? 150),
        render: (_v: unknown, row: TableRow) => {
          const value = leaf
            ? (row.instanceId ? cellValue(row.instanceId, leaf) : '')
            : (column.getValue?.(row, instanceContext) ?? '');
          const item = leaf && row.instanceId
            ? (schemaByPath.get(`${objectPath}${row.instanceId}.${leaf}`) ?? leafSchemaByLeaf.get(leaf))
            : leafSchemaByLeaf.get(leaf);
          const param = groupParamByLeaf.get(leaf);
          const constraints = effectiveParamConstraints(item, param);
          // 列自定义 formatValue 接收原始值；若未定义，再退到 enum label 兜底。
          // 这两者互斥：列已经声明 formatValue 表示有自定义显示，不应再被 enum 兜底改写。
          const formattedValue = column.formatValue
            ? column.formatValue(value)
            : (leaf ? formatEnumDisplayValue(value, constraints, item?.path, locale) : value);
          return <Text>{formattedValue || '-'}</Text>;
        },
      };
    }),
  ];
  if (hasRows) {
    columns.push({
      title: t('table.operation'),
      key: 'actions',
      width: multiColumnWidth(hasRows, 148),
      fixed: hasRows ? 'right' : undefined,
      render: (_v: unknown, row: TableRow) => (
        <Space size={4}>
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => openEditModal(row)}
            // 统一提交后，直到设备返回结果前禁止继续编辑，避免覆盖在途下发。
            disabled={isSubmitting || batchAwaitingDevice || (isIpsecGroup && !ipsecGlobalEnabled)}
          >
            {t('common.edit')}
          </Button>
          <Popconfirm
            title={t('device.multi.deleteConfirm')}
            onConfirm={() => row.instanceId && void handleDelete(row.instanceId)}
            disabled={
              (!canDelete && row.pending !== 'add')
              || isSubmitting
              || batchAwaitingDevice
              || (isIpsecGroup && !ipsecGlobalEnabled)
            }
          >
            <Button
              type="link"
              size="small"
              danger
              icon={<DeleteOutlined />}
              disabled={
                (!canDelete && row.pending !== 'add')
                || isSubmitting
                || batchAwaitingDevice
                || (isIpsecGroup && !ipsecGlobalEnabled)
              }
            >
              {t('common.delete')}
            </Button>
          </Popconfirm>
        </Space>
      ),
    });
  }

  const title = locale === 'zh-CN' ? group.titleZh : group.titleEn;
  const maxInstances = group.maxInstances && group.maxInstances > 0 ? group.maxInstances : undefined;
  const effectiveInstanceCount = instanceIds.length - pendingDeletes.size + pendingAdds.length + submittedPendingAdds.length;
  const reachedMax = maxInstances !== undefined && effectiveInstanceCount >= maxInstances;
  const cardTitle = maxInstances !== undefined
    ? `${title}（${effectiveInstanceCount}/${maxInstances}）`
    : title;
  const addDisabled = !canAdd || reachedMax || updateMutation.isPending || addMutation.isPending || isSubmitting || batchAwaitingDevice || (isIpsecGroup && !ipsecGlobalEnabled);
  const editModalIpsecEnabled = !isIpsecGroup || ipsecGlobalEnabled;
  const addBtn = (
    <Button type="default" icon={<PlusOutlined />} onClick={openAddModal} disabled={addDisabled}>
      {t('common.add')}
    </Button>
  );

  if (packedSpec) {
    return (
      <PackedScalarNeighborTable
        deviceId={deviceId}
        active={active}
        group={group}
        instanceContext={instanceContext}
        locale={locale}
        spec={packedSpec}
      />
    );
  }

  return (
    <Card
      title={cardTitle}
      size="small"
      styles={{ body: hasRows ? undefined : { padding: 0 } }}
      extra={
        <Space>
          {lastAction && lastAction.action !== 'add_rollback' && (() => {
            const visibleTaskStatus = lastTask?.status === 'completed'
              && lastAction.taskId
              && lastAction.syncedForTaskId !== lastTask.id
              ? 'sent'
              : lastTask?.status;
            const spec = statusTagSpec(lastAction, visibleTaskStatus, t);
            const isFailed = visibleTaskStatus === 'failed' && Boolean(lastTask?.errorMessage);
            const briefFault = isFailed ? formatDeviceFaultBrief(lastTask?.errorMessage) : '';
            const tag = (
              <Tag icon={spec.icon} color={spec.color} style={feedbackTagStyle()}>
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
          {pendingChangeCount > 0 && (
            <>
              <Button size="small" onClick={clearLocalBatchChanges} disabled={isSubmitting || batchAwaitingDevice}>
                {t('device.multi.batchClear')}
              </Button>
              <Button
                size="small"
                type="primary"
                icon={<SendOutlined />}
                loading={isSubmitting || batchAwaitingDevice}
                disabled={batchAwaitingDevice || !isIpsecGlobalStateReady}
                onClick={() => void handleSubmitBatch()}
              >
                {t(
                  isIpsecGroup
                    ? 'device.ipsec.unifiedSubmitWithCount'
                    : 'device.multi.batchSubmitWithCount',
                  { count: pendingChangeCount },
                )}
              </Button>
            </>
          )}
        </Space>
      }
      style={{ marginBottom: isGnbNetworkAddressGroup(group.id) ? 8 : 16 }}
    >
      {hasRows && (
        <Table<TableRow>
          rowKey={(row) => row.key}
          dataSource={tableRows}
          columns={columns}
          loading={isLoading}
          size="small"
          pagination={false}
          scroll={multiTableScroll(true)}
          sticky
        />
      )}
      <Modal
        title={editModal?.mode === 'add' ? t('device.multi.modalAddTitle', { title }) : t('device.multi.modalEditTitle', { title, instId: editModal?.instanceId ?? '' })}
        open={Boolean(editModal)}
        onOk={() => void handleSaveEditModal()}
        onCancel={closeEditModal}
        okText={editModal?.mode === 'add' ? t('device.multi.confirmAdd') : t('device.paramEdit.confirmDispatch')}
        cancelText={t('common.cancel')}
        confirmLoading={updateMutation.isPending || addMutation.isPending || isSubmitting}
        okButtonProps={{ disabled: Boolean(editModal) && isIpsecGroup && !ipsecGlobalEnabled }}
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
          {editModal && (editModal.mode === 'add' ? addModalColumns : displayColumns).map((column) => {
            const leaf = column.leaf || '';
            if (editModal.mode === 'add' && (!leaf || column.readOnly || (!column.virtual && !groupParamLeafSet.has(leaf)))) {
              return null;
            }
            if (!isGnbNetworkIpv4FieldVisible(group.id, leaf, editModal.values, gnbNetworkAddressingTypeLeaf)) {
              return null;
            }
            const value = leaf
              ? (editModal.values[leaf] ?? '')
              : (editModal.instanceId ? (column.getValue?.({ key: editModal.instanceId, instanceId: editModal.instanceId }, instanceContext) ?? '') : '');
            const item = leaf && editModal.instanceId ? (schemaByPath.get(`${objectPath}${editModal.instanceId}.${leaf}`) ?? leafSchemaByLeaf.get(leaf)) : leafSchemaByLeaf.get(leaf);
            const param = groupParamByLeaf.get(leaf);
            const constraints = effectiveParamConstraints(item, param);
            const isIpsecToggleField = isIpsecGroup && leaf === IPSEC_ENABLE_LEAF;
            const canEditByToggle = !isIpsecGroup || isIpsecToggleField || editModalIpsecEnabled;
            const isEditable = Boolean(leaf)
              && (Boolean(column.virtual) || groupParamLeafSet.has(leaf))
              && !column.readOnly
              && canEditByToggle
              && (editModal.mode === 'add' ? true : ((item?.writable ?? true)));
            const enumMeta = getEffectiveEnumMeta(constraints, item?.path);
            const configuredEnumOptions = param?.enumOptions && param.enumOptions.length > 0
              ? {
                  values: param.enumOptions.map((option) => option.value),
                  labels: param.enumOptions.map((option) => localizeEnumLabel(option.label, option.value, locale)),
                }
              : undefined;
            const effectiveEnumOptions = isIpsecToggleField
              ? {
                  values: ['true', 'false'],
                  labels: locale === 'zh-CN' ? ['开启', '关闭'] : ['Enabled', 'Disabled'],
                }
              : (configuredEnumOptions ?? enumMeta);
            const error = leaf ? editModal.errors[leaf] : '';
            const label = column.titleKey ? t(column.titleKey) : (locale === 'zh-CN' ? (column.titleZh ?? column.titleEn) : column.titleEn);
            const rangeHint = effectiveEnumOptions
              ? ''
              : column.virtual
              ? `[${column.virtual.constraints.minValue ?? '-∞'} ~ ${column.virtual.constraints.maxValue ?? '∞'}]`
              : (leaf ? formatEffectiveConstraintHint(item, param, t) : '');
            // 同列渲染：column.formatValue 收原始值；未提供则退到 enum 兜底。
            const displayValue = column.formatValue
              ? column.formatValue(value)
              : (leaf ? formatEnumDisplayValue(value, constraints, item?.path, locale) : value);

            return (
              <div key={column.key} style={{ minWidth: 0 }}>
                <div style={{ marginBottom: 6, fontWeight: 500 }}>
                  <Space size={4} wrap>
                    <span>
                      {(param?.required || column.virtual?.required) && isEditable && (
                        <span style={{ color: '#ff4d4f', marginRight: 2 }}>*</span>
                      )}
                      {label}
                    </span>
                    {rangeHint && <Text type="secondary" style={{ fontSize: 12 }}>{rangeHint}</Text>}
                  </Space>
                </div>
                {isEditable && effectiveEnumOptions && effectiveEnumOptions.values.length > 0 ? (
                  <Select
                    aria-label={label}
                    value={(isIpsecToggleField ? normalizeIpsecEnableValue(value) : value) || undefined}
                    onChange={(next) => leaf && setEditModalValue(leaf, String(next))}
                    style={{ width: '100%' }}
                    status={error ? 'error' : undefined}
                    options={effectiveEnumOptions.values.map((enumValue, idx) => ({ value: enumValue, label: effectiveEnumOptions.labels[idx] ?? enumValue }))}
                  />
                ) : isEditable ? (
                  <Input
                    aria-label={label}
                    value={value}
                    onChange={(e) => leaf && setEditModalValue(leaf, e.target.value)}
                    status={error ? 'error' : undefined}
                  />
                ) : (
                  <Input aria-label={label} value={displayValue} disabled />
                )}
                {column.formatValue && !effectiveEnumOptions && displayValue && displayValue !== value && isEditable && (
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
