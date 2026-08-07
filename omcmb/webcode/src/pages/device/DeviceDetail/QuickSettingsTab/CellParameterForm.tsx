import { useCallback, useEffect, useMemo, useRef, useState, Fragment } from 'react';
import { Alert, AutoComplete, Button, Card, Checkbox, Col, Form, Input, Row, Select, Space, Spin, Switch, Table, Tag, Tooltip, Typography, message, notification } from 'antd';
import type { FormInstance } from 'antd';
import { CheckCircleOutlined, ClockCircleOutlined, CloseCircleOutlined, DeleteOutlined, SendOutlined, SyncOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useParameterSchema, useSearchParameters, useUpdateParameters } from '@core/hooks/api/useDeviceParameters';
import { deviceParameterApi } from '@core/services/api/deviceParameterApi';
import { useDeviceTaskStatus } from '@core/hooks/api/useDeviceTask';
import { notificationKeys } from '@core/hooks/api/useNotificationCenter';
import { getTimezoneAliasOptions, mapTimezoneAliasToDisplay } from '@core/utils/timezoneAliasConfig';
import {
  feedbackKey,
  useQuickSettingsFeedbackStore,
  type CellFeedback,
} from '@core/store/quickSettingsFeedbackStore';
import type { DeviceParameter, ParameterSchemaItem, ParameterUpdateRequest } from '@core/types/deviceParameter';
import type { DeviceTaskStatus } from '@core/types/deviceTask';
import type { QuickSettingsGroup, QuickSettingsParam } from '@core/types/quicksettings';
import {
  applyInstanceContext,
  getEffectiveEnumMeta,
  getFeedbackScopeContext,
  isBscCodecSupportParam,
  localizeEnumLabel,
  normalizeBscCodecSupportValue,
  parseQuickSettingsMultiCheckboxValue,
  resolveQuickSettingsParameterType,
  serializeBscCodecSupportValue,
  serializeQuickSettingsMultiCheckboxValue,
  validateMmeIp,
  validateMmeIpPlmnLimit,
  validateMmeIpPlmnRows,
  validatePlmn,
  validateValue,
  type QuickSettingsInstanceContext,
} from './validators';
import { inferDeviceTimeMode, isNrNetworkType, mapDeviceTimeModeLabel, shouldShowDeviceTimeNtpServerFields } from './deviceTimeMode';
import { formatDeviceFaultBrief } from './MultiInstanceTable';
import {
  isPlmnRowLimitReached,
  serializePlmnList,
  toPlmnRows,
  validatePlmnList,
  type PlmnListValidationError,
  type PlmnRow,
} from './plmnList';
import { AddRowButton } from './AddRowButton';
import {
  canApplySubmittedReadback,
  ParameterReadbackTimeoutError,
  waitForExpectedParameterValues,
} from './parameterReadback';
import {
  buildMlnMmePoolUpdates,
  isMlnIndexedMmePoolModel,
  MLN_MME_POOL_STANDARD_PREFIX,
  parseMlnMmePoolRows,
} from './mmeIpPlmnIndexed';
import {
  findMmePlmnOutsideServingList,
  getServingPlmnOptionsForModel,
  getServingPlmnSearchQuery,
  isServingPlmnRestrictedModel,
} from './mmePlmnSelection';
import {
  applyDeviceParameterSearchReadback,
  refreshDeviceParameterSearchQueries,
} from './parameterSearchRefresh';
import { useT } from '@/hooks/useT';

const { Text } = Typography;
const ERROR_FEEDBACK_DURATION_SECONDS = 2;

const BM_GSM_CELL_OP_STATE_PATTERN = /^Device\.Services\.GsmBTSCellDT\.\d+\.OpState$/;
const BITMASK_SELECT_PATHS = new Set([
  'Device.FAP.Synchronization.PpsTimeMode',
  'Device.FAP.GNSS.SyncSource',
]);
const BM_PPS_TIME_MODE_PATH = 'Device.FAP.Synchronization.PpsTimeMode';
const BM_GNSS_SYNC_SOURCE_PATH = 'Device.FAP.GNSS.SyncSource';
const BM_PTP_CONFIG_PREFIX = 'Device.FAP.PTP1588.';
const BM_GNSS_SYNC_SOURCE_BITS = {
  GPS: '1',
  GLONASS: '2',
  GALILEO: '4',
  BEIDOU: '8',
  QZSS: '16',
} as const;
const BM_GNSS_EXCLUSIVE_BITS = new Set<string>([
  BM_GNSS_SYNC_SOURCE_BITS.GLONASS,
  BM_GNSS_SYNC_SOURCE_BITS.BEIDOU,
  BM_GNSS_SYNC_SOURCE_BITS.GALILEO,
]);
const GNB_GPS_SYNC_SOURCE_PATH = 'Device.FAP.GPS.SyncSource';
const GNB_GNSS_SYNC_SOURCE_VALUES = {
  GPS: 'GPS',
  GLONASS: 'GLONASS',
  GALILEO: 'GALILEO',
  BEIDOU: 'BEIDOU',
  QZSS: 'QZSS',
} as const;
const GNB_GNSS_EXCLUSIVE_VALUES = new Set<string>([
  GNB_GNSS_SYNC_SOURCE_VALUES.GLONASS,
  GNB_GNSS_SYNC_SOURCE_VALUES.BEIDOU,
  GNB_GNSS_SYNC_SOURCE_VALUES.GALILEO,
]);
const GNB_FORCED_SYNC_PATH = 'Device.DeviceInfo.iForcedSyncControlSwitch';
const GNB_SYNC_MODE_GNSS_VALUES = new Set(['GPS_PPS', 'LOCAL_CLOCK_HOLDOVER_GPS_PPS', 'GPS_AND_PTP', '1']);
const GNB_SYNC_MODE_PTP_VALUES = new Set(['1588_PPS', 'GPS_AND_PTP', '2']);

type TFn = (id: string, values?: Record<string, string | number>) => string;

const NR_CARRIER_BANDWIDTH_OPTIONS_BY_SCS: Record<string, Array<{ value: string; label: string }>> = {
  '0': [
    { value: '25', label: '5MHz(25RB)' },
    { value: '52', label: '10MHz(52RB)' },
    { value: '79', label: '15MHz(79RB)' },
    { value: '106', label: '20MHz(106RB)' },
    { value: '133', label: '25MHz(133RB)' },
    { value: '160', label: '30MHz(160RB)' },
    { value: '216', label: '40MHz(216RB)' },
    { value: '270', label: '50MHz(270RB)' },
  ],
  '1': [
    { value: '11', label: '5MHz(11RB)' },
    { value: '24', label: '10MHz(24RB)' },
    { value: '38', label: '15MHz(38RB)' },
    { value: '51', label: '20MHz(51RB)' },
    { value: '65', label: '25MHz(65RB)' },
    { value: '78', label: '30MHz(78RB)' },
    { value: '106', label: '40MHz(106RB)' },
    { value: '133', label: '50MHz(133RB)' },
    { value: '162', label: '60MHz(162RB)' },
    { value: '189', label: '70MHz(189RB)' },
    { value: '217', label: '80MHz(217RB)' },
    { value: '245', label: '90MHz(245RB)' },
    { value: '273', label: '100MHz(273RB)' },
  ],
  '2': [
    { value: '11', label: '10MHz(11RB)' },
    { value: '18', label: '15MHz(18RB)' },
    { value: '24', label: '20MHz(24RB)' },
    { value: '31', label: '25MHz(31RB)' },
    { value: '38', label: '30MHz(38RB)' },
    { value: '51', label: '40MHz(51RB)' },
    { value: '65', label: '50MHz(65RB)' },
    { value: '79', label: '60MHz(79RB)' },
    { value: '93', label: '70MHz(93RB)' },
    { value: '107', label: '80MHz(107RB)' },
    { value: '121', label: '90MHz(121RB)' },
    { value: '135', label: '100MHz(135RB)' },
  ],
};

function getNrCarrierBandwidthOptions(paramName: string, dlScs: unknown, ulScs: unknown): Array<{ value: string; label: string }> {
  if (paramName === 'DLCarrierBandWidth') {
    return NR_CARRIER_BANDWIDTH_OPTIONS_BY_SCS[String(dlScs ?? '')] ?? [];
  }
  if (paramName === 'ULCarrierBandWidth') {
    return NR_CARRIER_BANDWIDTH_OPTIONS_BY_SCS[String(ulScs ?? '')] ?? [];
  }
  return [];
}

/**
 * GSM ARFCN -> 上下行频率(MHz) 派生计算。
 * 字典里没有 Frequency 叶子,这里按 3GPP TS 45.005 §2 的频段公式从 ARFCN 反推显示。
 * 参考: GSM-850/P-GSM/E-GSM/R-GSM/DCS-1800/PCS-1900。
 * 返回 null 表示落在保留段,UI 渲染为 "-"。
 */
function arfcnToFrequencyMHz(arfcnRaw: unknown): { ul: number; dl: number } | null {
  const n = Number(arfcnRaw);
  if (!Number.isFinite(n)) return null;
  // P-GSM 900: 1..124, UL = 890 + 0.2*n, DL = UL + 45
  if (n >= 1 && n <= 124) return { ul: 890 + 0.2 * n, dl: 890 + 0.2 * n + 45 };
  // E-GSM 900: 975..1023, UL = 890 + 0.2*(n-1024), DL = UL + 45
  if (n >= 975 && n <= 1023) return { ul: 890 + 0.2 * (n - 1024), dl: 890 + 0.2 * (n - 1024) + 45 };
  // GSM-850: 128..251, UL = 824.2 + 0.2*(n-128), DL = UL + 45
  if (n >= 128 && n <= 251) return { ul: 824.2 + 0.2 * (n - 128), dl: 824.2 + 0.2 * (n - 128) + 45 };
  // DCS-1800: 512..885, UL = 1710.2 + 0.2*(n-512), DL = UL + 95
  if (n >= 512 && n <= 885) return { ul: 1710.2 + 0.2 * (n - 512), dl: 1710.2 + 0.2 * (n - 512) + 95 };
  // PCS-1900: 512..810 (重叠 DCS,这里仅 PCS 命中区间留给设备侧约定)
  return null;
}

function formatFreqMHz(v: number): string {
  // 0.2 步长导致浮点误差,保留 1 位小数足够。
  return `${Math.round(v * 10) / 10}MHz`;
}

// FrequencyDisplay: 跟随表单内 CurrentArfcn 变化重算 UL/DL。独立组件避免整表单重渲染。
function FrequencyDisplay({ form, locale }: { form: FormInstance; locale: 'zh-CN' | 'en-US' }) {
  void locale;
  const t = useT();
  const arfcn = Form.useWatch('CurrentArfcn', form);
  const freq = arfcnToFrequencyMHz(arfcn);
  const labelText = t('device.cell.freqMHz');
  const ulText = t('device.cell.uplink');
  const dlText = t('device.cell.downlink');
  const display = freq
    ? `${ulText}: ${formatFreqMHz(freq.ul)}  ${dlText}: ${formatFreqMHz(freq.dl)}`
    : '-';
  return (
    <Col span={8}>
      <Form.Item
        label={
          <Space size={4}>
            <span>{labelText}</span>
            <Text type="secondary" style={{ fontSize: 12 }}>{t('device.cell.readonly')}</Text>
          </Space>
        }
      >
        <Input value={display} disabled />
      </Form.Item>
    </Col>
  );
}

function TransmissionPowerInput({
  countField,
  powerField,
  disabled,
  placeholder,
}: {
  countField: string;
  powerField: string;
  disabled: boolean;
  placeholder?: string;
}) {
  return (
    <Space.Compact style={{ width: '100%' }}>
      <Form.Item name={countField} noStyle>
        <Input disabled={disabled} suffix="*" style={{ width: 72 }} />
      </Form.Item>
      <Form.Item name={powerField} noStyle>
        <Input disabled={disabled} placeholder={placeholder} style={{ width: '100%' }} />
      </Form.Item>
    </Space.Compact>
  );
}

function GsmRuRelationInput({
  error,
  ruRouteItemByIdx,
}: {
  error?: string;
  ruRouteItemByIdx: Map<string, ParameterSchemaItem>;
}) {
  const t = useT();
  const routeOptions = useMemo(() => (
    Array.from(ruRouteItemByIdx.entries())
      .sort(([a], [b]) => Number(a) - Number(b))
      .map(([idx, item]) => {
        const reportedValue = item.currentValue == null ? '' : String(item.currentValue).trim();
        return {
          label: reportedValue ? `RU ${idx} -> ${reportedValue}` : `RU ${idx} -> ${t('device.cell.notReported')}`,
          value: reportedValue,
        };
      })
      .filter((o) => o.value)
  ), [ruRouteItemByIdx, t]);
  return (
    <Col span={8}>
      <Form.Item
        label="Route Index (绑定 RU)"
        name="GsmCellWithRuRelation"
        validateStatus={error ? 'error' : undefined}
        help={error}
      >
        <AutoComplete
          options={routeOptions}
          optionFilterProp="label"
          placeholder={t('device.cell.notReported')}
        />
      </Form.Item>
    </Col>
  );
}

// RouteIndexInput: BM LTE 小区专属。写回自身 LteCellWithRuList。
// 输入框支持从 RU RouteIndex 候选选择,也支持用户直接手动输入。
function RouteIndexInput({
  form,
  ruRouteItemByIdx,
  locale,
  fieldName,
  boundRuFieldName,
  currentRuIdx,
  disabled,
  error,
}: {
  form: FormInstance;
  ruRouteItemByIdx: Map<string, ParameterSchemaItem>;
  locale: 'zh-CN' | 'en-US';
  fieldName: string;
  boundRuFieldName?: string;
  currentRuIdx?: string;
  disabled: boolean;
  error?: string;
}) {
  void locale;
  const t = useT();
  const lastAutoRouteValueRef = useRef('');
  const watchedRuRel = Form.useWatch(boundRuFieldName ?? '__unusedRouteIndexBoundRu', form);
  const ruIdx = currentRuIdx ?? (watchedRuRel == null ? '' : String(watchedRuRel).trim());
  const routeItem = ruIdx ? ruRouteItemByIdx.get(ruIdx) : undefined;
  const routeValue = (() => {
    const reportedValue = routeItem?.currentValue == null ? '' : String(routeItem.currentValue).trim();
    return reportedValue;
  })();
  const routeOptions = useMemo(() => (
    Array.from(ruRouteItemByIdx.entries())
      .sort(([a], [b]) => Number(a) - Number(b))
      .map(([idx, item]) => {
        const reportedValue = item.currentValue == null ? '' : String(item.currentValue).trim();
        const value = reportedValue;
        return {
          label: value ? `RU ${idx} -> ${value}` : `RU ${idx} -> ${t('device.cell.notReported')}`,
          value,
        };
      })
      .filter((o) => o.value)
  ), [boundRuFieldName, ruRouteItemByIdx, t]);
  const labelText = t('device.cell.routeIndexBoundRu');
  useEffect(() => {
    if (!boundRuFieldName) return;
    const currentValue = form.getFieldValue(fieldName);
    const currentText = currentValue == null ? '' : String(currentValue);
    const lastAutoValue = lastAutoRouteValueRef.current;
    if (currentText === '' || currentText === lastAutoValue) {
      form.setFieldValue(fieldName, routeValue);
      lastAutoRouteValueRef.current = routeValue;
    }
  }, [boundRuFieldName, fieldName, form, routeValue]);

  const writable = !disabled && (boundRuFieldName ? Boolean(ruIdx) : true);
  return (
    <Col span={8}>
      <Form.Item
        label={
          <Space size={4}>
            <span>{labelText}</span>
            {ruIdx && <Text type="secondary" style={{ fontSize: 12 }}>{`RU ${ruIdx}`}</Text>}
            {!writable && <Text type="secondary" style={{ fontSize: 12 }}>{t('device.cell.readonly')}</Text>}
          </Space>
        }
        name={fieldName}
        validateStatus={error ? 'error' : undefined}
        help={error}
      >
        <AutoComplete
          disabled={!writable}
          options={routeOptions}
          optionFilterProp="label"
        />
      </Form.Item>
    </Col>
  );
}

// "上次提交"状态形状由 frontend-core/store/quickSettingsFeedbackStore (CellFeedback) 定义,
// 提升至 store 持久化,顶层 TabBar 切走再切回不丢反馈。

// LTE EARFCN -> DL 频率(MHz)映射表(3GPP TS 36.101 §5.7.3 子集,常用 BM 频段)。
const LTE_BAND_DL_BASE: Record<number, { earfcnLow: number; freqLow: number }> = {
  1: { earfcnLow: 0, freqLow: 2110 },
  3: { earfcnLow: 1200, freqLow: 1805 },
  5: { earfcnLow: 2400, freqLow: 869 },
  7: { earfcnLow: 2750, freqLow: 2620 },
  8: { earfcnLow: 3450, freqLow: 925 },
  38: { earfcnLow: 37750, freqLow: 2570 },
  39: { earfcnLow: 38250, freqLow: 1880 },
  40: { earfcnLow: 38650, freqLow: 2300 },
  41: { earfcnLow: 39650, freqLow: 2496 },
};
function lteEarfcnToMHz(band: unknown, earfcn: unknown): number | null {
  const b = Number(band);
  const e = Number(earfcn);
  if (!Number.isFinite(b) || !Number.isFinite(e)) return null;
  const base = LTE_BAND_DL_BASE[b];
  if (!base) return null;
  return base.freqLow + 0.1 * (e - base.earfcnLow);
}

// LteFrequencyDisplay: 跟随 BandIndicator + DLEarfcn 派生 DL 中心频率。
function LteFrequencyDisplay({ form, locale }: { form: FormInstance; locale: 'zh-CN' | 'en-US' }) {
  void locale;
  const t = useT();
  const band = Form.useWatch('BandIndicator', form);
  const earfcn = Form.useWatch('DLEarfcn', form);
  const f = lteEarfcnToMHz(band, earfcn);
  const labelText = t('device.cell.freqMHz');
  const display = f != null ? formatFreqMHz(f) : '-';
  return (
    <Col span={8}>
      <Form.Item
        label={
          <Space size={4}>
            <span>{labelText}</span>
            <Text type="secondary" style={{ fontSize: 12 }}>{t('device.cell.readonly')}</Text>
          </Space>
        }
      >
        <Input value={display} disabled />
      </Form.Item>
    </Col>
  );
}

// CellIdDerivedDisplay: 从 ECI(CellIdentity) 反推每小区 8 位 Cell ID(ECI mod 256)。
function CellIdDerivedDisplay({ form, locale }: { form: FormInstance; locale: 'zh-CN' | 'en-US' }) {
  void locale;
  const t = useT();
  const eci = Form.useWatch('ECI', form);
  const n = Number(eci);
  const cid = Number.isFinite(n) ? n % 256 : null;
  const labelText = 'Cell ID (ECI%256)';
  return (
    <Col span={8}>
      <Form.Item
        label={
          <Space size={4}>
            <span>{labelText}</span>
            <Text type="secondary" style={{ fontSize: 12 }}>{t('device.cell.readonly')}</Text>
          </Space>
        }
      >
        <Input value={cid != null ? String(cid) : '-'} disabled />
      </Form.Item>
    </Col>
  );
}

function formatTime(at: number): string {
  const d = new Date(at);
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

function isNtpServerPath(path: string): boolean {
  return /^Device\.Time\.NTPServer\d+$/.test(path);
}

/** T-0146:状态机 Tag 显示规则。 */
interface StatusTagSpec {
  color: string;
  icon: React.ReactNode;
  label: string;
}
function statusTagSpec(submit: CellFeedback, taskStatus: DeviceTaskStatus | undefined, t: TFn): StatusTagSpec {
  if (submit.submitStatus === 'failed_to_queue') {
    return { color: 'error', icon: <CloseCircleOutlined />, label: t('device.cell.tagQueueFailed') };
  }
  // After submitStatus = 'queued', branch on the real task status
  switch (taskStatus) {
    case 'completed':
      return { color: 'success', icon: <CheckCircleOutlined />, label: t('device.cell.tagAckSuccess') };
    case 'failed':
      return { color: 'error', icon: <CloseCircleOutlined />, label: t('device.cell.tagAckFailed') };
    case 'expired':
      return { color: 'warning', icon: <ClockCircleOutlined />, label: t('device.cell.tagTimeout') };
    case 'cancelled':
      return { color: 'default', icon: <CloseCircleOutlined />, label: t('device.cell.tagCancelled') };
    case 'sent':
      return { color: 'processing', icon: <SendOutlined />, label: t('device.cell.tagSent') };
    case 'pending':
    default:
      // pending, or task not yet fetched (just after mutateAsync)
      return { color: 'processing', icon: <SyncOutlined spin />, label: t('device.cell.tagPending') };
  }
}

function appendCurrentOptionWithLabel(
  options: Array<{ value: string; label: string }>,
  currentValue: string,
  currentLabel: string,
): Array<{ value: string; label: string }> {
  if (!currentValue || options.some((option) => option.value === currentValue)) {
    return options;
  }
  return [{ value: currentValue, label: currentLabel || currentValue }, ...options];
}

function appendCurrentEnumOption(
  options: Array<{ value: string; label: string }>,
  rawValue: unknown,
  xmlOptions?: Array<{ value: string; label: string }>,
): Array<{ value: string; label: string }> {
  const normalized = normalizeEnumValue(rawValue, xmlOptions);
  if (!normalized || options.some((option) => option.value === normalized)) {
    return options;
  }
  return [{ value: normalized, label: normalized }, ...options];
}

function isBitmaskSelectPath(path: string): boolean {
  return BITMASK_SELECT_PATHS.has(path);
}

function isBmPtpConfigPath(path: string): boolean {
  return path.startsWith(BM_PTP_CONFIG_PREFIX);
}

function isGnbPtpConfigPath(path: string): boolean {
  return path.startsWith(BM_PTP_CONFIG_PREFIX);
}

function isGnbCommonSyncPath(path: string): boolean {
  return path === GNB_FORCED_SYNC_PATH;
}

function isBmGnssSyncSourcePath(path: string): boolean {
  return path === BM_GNSS_SYNC_SOURCE_PATH;
}

function isStringMultiSelectPath(path: string): boolean {
  return path === GNB_GPS_SYNC_SOURCE_PATH;
}

function normalizeStringMultiSelectValue(raw: unknown): string[] {
  if (Array.isArray(raw)) {
    return raw.map((item) => String(item ?? '').trim()).filter(Boolean);
  }
  return String(raw ?? '')
    .split('_')
    .map((item) => item.trim())
    .filter(Boolean);
}

function serializeStringMultiSelectValue(raw: unknown): string {
  return normalizeStringMultiSelectValue(raw).join('_');
}

function isSwitchPath(path: string): boolean {
  return path === GNB_FORCED_SYNC_PATH || BM_GSM_CELL_OP_STATE_PATTERN.test(path);
}

function normalizeSwitchValue(raw: unknown): boolean {
  if (typeof raw === 'boolean') {
    return raw;
  }
  const value = String(raw ?? '').trim().toLowerCase();
  if (!value) return true;
  return value === '1' || value === 'true' || value === 'on';
}

function serializeSwitchValue(raw: unknown): string {
  return normalizeSwitchValue(raw) ? '1' : '0';
}

function normalizeSwitchComparableValue(raw: unknown): string {
  return serializeSwitchValue(raw);
}

function bitmaskHasBit(raw: unknown, bit: number): boolean {
  const numeric = Number(String(raw ?? '').trim());
  return Number.isInteger(numeric) && (numeric & bit) !== 0;
}

function normalizeBitmaskValue(raw: unknown): string {
  const numeric = Number(String(raw ?? '').trim());
  if (!Number.isFinite(numeric) || numeric < 0) return '';
  return String(numeric);
}

function normalizeBitmaskBits(raw: unknown, allowedValues: string[]): string[] {
  const allowed = new Set(allowedValues);
  if (Array.isArray(raw)) {
    return raw
      .map((item) => String(item ?? '').trim())
      .filter((item, idx, arr) => item && allowed.has(item) && arr.indexOf(item) === idx);
  }

  const text = String(raw ?? '').trim();
  if (!text) return [];
  if (text.includes(',')) {
    return text
      .split(',')
      .map((item) => item.trim())
      .filter((item, idx, arr) => item && allowed.has(item) && arr.indexOf(item) === idx);
  }

  const numeric = Number(text);
  if (!Number.isFinite(numeric) || numeric <= 0) return [];
  return allowedValues.filter((value) => {
    const bit = Number(value);
    return Number.isFinite(bit) && bit > 0 && (numeric & bit) !== 0;
  });
}

function isConstrainedGnssSyncSourcePath(path: string): boolean {
  return path === BM_GNSS_SYNC_SOURCE_PATH || path === GNB_GPS_SYNC_SOURCE_PATH;
}

function getGnssExclusiveValues(path: string): Set<string> {
  return path === BM_GNSS_SYNC_SOURCE_PATH ? BM_GNSS_EXCLUSIVE_BITS : GNB_GNSS_EXCLUSIVE_VALUES;
}

function getGnssQzssValue(path: string): string {
  return path === BM_GNSS_SYNC_SOURCE_PATH ? BM_GNSS_SYNC_SOURCE_BITS.QZSS : GNB_GNSS_SYNC_SOURCE_VALUES.QZSS;
}

function normalizeConstrainedGnssSyncSourceValue(raw: unknown, allowedValues: string[], path: string): string[] {
  return path === BM_GNSS_SYNC_SOURCE_PATH
    ? normalizeBitmaskBits(raw, allowedValues)
    : normalizeStringMultiSelectValue(raw).filter((item, idx, arr) => allowedValues.includes(item) && arr.indexOf(item) === idx);
}

function normalizeConstrainedGnssSyncSourceChange(
  raw: unknown,
  previous: string[],
  allowedValues: string[],
  path: string,
): { value: string[]; errorKey?: string } {
  const allowed = new Set(allowedValues);
  let value = normalizeConstrainedGnssSyncSourceValue(raw, allowedValues, path);
  const exclusiveValues = getGnssExclusiveValues(path);

  const exclusive = value.filter((item) => exclusiveValues.has(item));
  if (exclusive.length > 1) {
    const addedExclusive = exclusive.filter((item) => !previous.includes(item));
    const keep = addedExclusive.at(-1) ?? exclusive.at(-1);
    value = value.filter((item) => !exclusiveValues.has(item) || item === keep);
  }

  if (value.includes(getGnssQzssValue(path)) && value.length === 1) {
    return { value: [], errorKey: 'device.cell.syncSourceQzssAlone' };
  }

  return {
    value: value.filter((item, idx, arr) => allowed.has(item) && arr.indexOf(item) === idx),
  };
}

function buildBitmaskCombinationOptions(
  values: string[],
  labels: string[],
  locale: 'zh-CN' | 'en-US',
): Array<{ value: string; label: string }> {
  const baseOptions = values.map((value, idx) => ({
    value,
    label: localizeEnumLabel(labels[idx] || value, value, locale),
  })).filter((option) => {
    const bit = Number(option.value);
    return Number.isFinite(bit) && bit > 0;
  });
  const options: Array<{ value: string; label: string }> = [];
  const visit = (start: number, count: number, mask: number, parts: string[]) => {
    if (parts.length === count) {
      options.push({ value: String(mask), label: parts.join('+') });
      return;
    }
    for (let i = start; i < baseOptions.length; i += 1) {
      const option = baseOptions[i];
      visit(i + 1, count, mask | Number(option.value), [...parts, option.label]);
    }
  };
  for (let count = 1; count <= baseOptions.length; count += 1) {
    visit(0, count, 0, []);
  }
  return options;
}

function serializeBitmaskValue(raw: unknown): string {
  const values = Array.isArray(raw) ? raw : raw == null || raw === '' ? [] : [raw];
  const mask = values.reduce((acc, value) => {
    const bit = Number(String(value ?? '').trim());
    return Number.isFinite(bit) && bit > 0 ? acc | bit : acc;
  }, 0);
  return String(mask);
}

function serializeConstrainedGnssSyncSourceValue(raw: unknown, path: string): string {
  return path === BM_GNSS_SYNC_SOURCE_PATH
    ? serializeBitmaskValue(raw)
    : serializeStringMultiSelectValue(raw);
}

function validateBitmaskValue(value: string, allowedValues: string[], minValue?: number, maxValue?: number): string | null {
  if (!value) return '请输入值';
  const numeric = Number(value);
  if (!Number.isInteger(numeric) || numeric < 0) return '请输入非负整数';
  if (minValue !== undefined && numeric < minValue) return `最小值为 ${minValue}`;
  if (maxValue !== undefined && numeric > maxValue) return `最大值为 ${maxValue}`;

  const allowedMask = allowedValues.reduce((acc, raw) => {
    const bit = Number(String(raw ?? '').trim());
    return Number.isInteger(bit) && bit > 0 ? acc | bit : acc;
  }, 0);
  if (allowedMask <= 0) return null;
  if (numeric <= 0 || (numeric & ~allowedMask) !== 0) {
    return `允许的组合: ${allowedValues.join(', ')}`;
  }
  return null;
}

function validateConstrainedGnssSyncSourceValue(value: string, allowedValues: string[], path: string, t: TFn): string | null {
  const selected = normalizeConstrainedGnssSyncSourceValue(value, allowedValues, path);
  if (selected.length === 0) return t('device.cell.syncSourceRequired');
  if (selected.filter((item) => getGnssExclusiveValues(path).has(item)).length > 1) {
    return t('device.cell.syncSourceExclusive');
  }
  if (selected.includes(getGnssQzssValue(path)) && selected.length === 1) {
    return t('device.cell.syncSourceQzssAlone');
  }
  return null;
}

function formValueEquals(left: unknown, right: unknown): boolean {
  if (Array.isArray(left) || Array.isArray(right)) {
    const leftValues = Array.isArray(left) ? left : left == null ? [] : [left];
    const rightValues = Array.isArray(right) ? right : right == null ? [] : [right];
    return leftValues.map(String).join(',') === rightValues.map(String).join(',');
  }
  return left === right;
}

function formatTimeZoneDisplay(value: string): string {
  const normalized = String(value ?? '').trim();
  if (!normalized) return '';
  return mapTimezoneAliasToDisplay(normalized);
}

interface CellParameterFormProps {
  deviceId: string;
  paramModel?: string;
  active?: boolean;
  group: QuickSettingsGroup;
  instanceContext: QuickSettingsInstanceContext;
  locale: 'zh-CN' | 'en-US';
  onIpsecControlChange?: (value: string | undefined) => void;
  actionMode?: 'standalone' | 'staged';
}

interface BindSelectOption {
  value: string;
  label: string;
}

type BindSelectValueMode = 'path' | 'ip';

interface SpecialFieldConfig {
  kind: 'input' | 'mme-ip-plmn-table' | 'plmn-list-table' | 'bind-select';
  configPath: string;
  displayPath?: string;
  bindValueMode?: BindSelectValueMode;
  fallbackConfigPath?: string;
  fallbackDisplayPath?: string;
  fallbackBindValueMode?: BindSelectValueMode;
  placeholder?: string;
  forceWritable?: boolean;
}

type MmeIpPlmnRow = {
  key: string;
  mmeIp: string;
  plmn: string;
};

interface MmeIpPlmnTableProps {
  value?: MmeIpPlmnRow[];
  onChange?: (value: MmeIpPlmnRow[]) => void;
  disabled?: boolean;
  locale: 'zh-CN' | 'en-US';
  maxRows?: number;
  plmnOptions?: string[];
  restrictPlmnSelection?: boolean;
}

interface PlmnListTableProps {
  value?: PlmnRow[];
  onChange?: (value: PlmnRow[]) => void;
  disabled?: boolean;
  maxRows?: number;
}

function plmnValidationMessage(
  error: PlmnListValidationError | null,
  maxRows: number,
  t: TFn,
): string | null {
  if (error === 'format') return t('device.cell.plmnFormatInvalid');
  if (error === 'duplicate') return t('device.cell.plmnDuplicate');
  if (error === 'limit') return t('device.cell.plmnLimitReached', { max: maxRows });
  return null;
}

function PlmnListTable({
  value = [],
  onChange,
  disabled = false,
  maxRows = 6,
}: PlmnListTableProps) {
  const t = useT();
  const rows = toPlmnRows(value);
  const maxReached = isPlmnRowLimitReached(rows, maxRows);

  const updateRow = (key: string, plmn: string) => {
    onChange?.(rows.map((row) => (row.key === key ? { ...row, plmn } : row)));
  };

  const addRow = () => {
    if (maxReached) {
      message.warning(t('device.cell.plmnLimitReached', { max: maxRows }));
      return;
    }
    onChange?.([
      ...rows,
      { key: `plmn-${Date.now()}-${rows.length}`, plmn: '' },
    ]);
  };

  const columns = [
    {
      title: 'PLMN',
      dataIndex: 'plmn',
      key: 'plmn',
      render: (_: unknown, row: PlmnRow) => {
        const invalid = Boolean(row.plmn.trim()) && !/^\d{5,6}$/.test(row.plmn.trim());
        return (
          <Tooltip title={invalid ? t('device.cell.plmnFormatInvalid') : ''}>
            <Input
              value={row.plmn}
              disabled={disabled}
              status={invalid ? 'error' : undefined}
              placeholder="46000"
              onChange={(event) => updateRow(row.key, event.target.value)}
            />
          </Tooltip>
        );
      },
    },
    {
      title: t('table.operation'),
      key: 'actions',
      width: 80,
      render: (_: unknown, row: PlmnRow) => (
        <Button
          danger
          type="text"
          icon={<DeleteOutlined />}
          disabled={disabled}
          onClick={() => onChange?.(rows.filter((item) => item.key !== row.key))}
        />
      ),
    },
  ];

  return (
    <Space orientation="vertical" style={{ width: '100%' }} size={8}>
      <Table<PlmnRow>
        size="small"
        rowKey="key"
        pagination={false}
        dataSource={rows}
        columns={columns}
      />
      <AddRowButton
        onClick={addRow}
        disabled={disabled || maxReached}
      >
        {t('device.cell.addRow')}
      </AddRowButton>
      <Alert
        type={maxReached ? 'warning' : 'info'}
        showIcon
        message={t('device.cell.plmnLimitHint', { max: maxRows })}
      />
    </Space>
  );
}

function normalizeMmeIpPlmnRows(rows: MmeIpPlmnRow[]): MmeIpPlmnRow[] {
  return rows
    .map((row) => ({
      key: row.key,
      mmeIp: String(row.mmeIp ?? '').trim(),
      plmn: String(row.plmn ?? '').trim(),
    }))
    .filter((row) => row.mmeIp || row.plmn);
}

function parseMmeIpPlmnList(raw: unknown): MmeIpPlmnRow[] {
  const text = String(raw ?? '').trim();
  if (!text) return [];
  return text
    .split(/[;,\n]+/)
    .map((item, idx) => {
      const [mmeIp = '', plmn = ''] = item.split('+');
      return {
        key: `row-${idx}-${mmeIp.trim()}-${plmn.trim()}`,
        mmeIp: mmeIp.trim(),
        plmn: plmn.trim(),
      };
    });
}

function serializeMmeIpPlmnList(rows: MmeIpPlmnRow[]): string {
  return normalizeMmeIpPlmnRows(rows)
    .map((row) => `${row.mmeIp}+${row.plmn}`)
    .join(',');
}

function isMmeIpPlmnRows(value: unknown): value is MmeIpPlmnRow[] {
  return Array.isArray(value)
    && value.every((item) => (
      item && typeof item === 'object' && 'mmeIp' in item && 'plmn' in item
    ));
}

function toMmeIpPlmnRows(value: unknown): MmeIpPlmnRow[] {
  if (isMmeIpPlmnRows(value)) {
    return (value as MmeIpPlmnRow[]).map((row, idx) => ({
      ...row,
      key: row.key || `row-${idx}`,
      mmeIp: String(row.mmeIp ?? ''),
      plmn: String(row.plmn ?? ''),
    }));
  }
  return parseMmeIpPlmnList(value);
}

function MmeIpPlmnTable({
  value = [],
  onChange,
  disabled = false,
  locale,
  maxRows,
  plmnOptions = [],
  restrictPlmnSelection = false,
}: MmeIpPlmnTableProps) {
  void locale;
  const t = useT();
  const rows = isMmeIpPlmnRows(value) ? value : [];
  const maxReached = maxRows !== undefined && normalizeMmeIpPlmnRows(rows).length >= maxRows;

  const setRows = (nextRows: MmeIpPlmnRow[]) => {
    onChange?.(nextRows);
  };

  const updateCell = (key: string, field: 'mmeIp' | 'plmn', nextValue: string) => {
    setRows(rows.map((row) => (row.key === key ? { ...row, [field]: nextValue } : row)));
  };

  const addRow = () => {
    if (maxReached) {
      message.warning(t('device.cell.mmeIpPlmnLimitReached', { max: maxRows ?? 0 }));
      return;
    }
    setRows([
      ...rows,
      { key: `row-${Date.now()}-${rows.length}`, mmeIp: '', plmn: '' },
    ]);
  };

  const removeRow = (key: string) => {
    setRows(rows.filter((row) => row.key !== key));
  };

  const columns = [
    {
      title: locale === 'zh-CN' ? 'MME IP' : 'MME IP',
      dataIndex: 'mmeIp',
      key: 'mmeIp',
      render: (_: unknown, row: MmeIpPlmnRow) => (
        <Tooltip title={validateMmeIp(row.mmeIp) ?? ''}>
          <Input
            value={row.mmeIp}
            disabled={disabled}
            status={validateMmeIp(row.mmeIp) ? 'error' : undefined}
            placeholder="127.0.0.1"
            onChange={(e) => updateCell(row.key, 'mmeIp', e.target.value)}
          />
        </Tooltip>
      ),
    },
    {
      title: locale === 'zh-CN' ? 'PLMN' : 'PLMN',
      dataIndex: 'plmn',
      key: 'plmn',
      render: (_: unknown, row: MmeIpPlmnRow) => {
        const formatError = validatePlmn(row.plmn);
        const membershipError = restrictPlmnSelection
          && row.plmn
          && !plmnOptions.includes(row.plmn)
          ? t('device.cell.mmePlmnNotServing', {
            row: rows.findIndex((item) => item.key === row.key) + 1,
            plmn: row.plmn,
          })
          : null;
        const error = formatError ?? membershipError;
        return (
          <Tooltip title={error ?? ''}>
            {restrictPlmnSelection ? (
              <Select
                value={row.plmn || undefined}
                disabled={disabled}
                status={error ? 'error' : undefined}
                style={{ width: '100%' }}
                placeholder={t('device.cell.mmePlmnSelectPlaceholder')}
                notFoundContent={t('device.cell.mmePlmnNoOptions')}
                showSearch
                optionFilterProp="label"
                options={plmnOptions.map((plmn) => ({ value: plmn, label: plmn }))}
                onChange={(nextValue) => updateCell(row.key, 'plmn', nextValue)}
              />
            ) : (
              <Input
                value={row.plmn}
                disabled={disabled}
                status={error ? 'error' : undefined}
                placeholder="46000"
                onChange={(e) => updateCell(row.key, 'plmn', e.target.value)}
              />
            )}
          </Tooltip>
        );
      },
    },
    {
      title: t('table.operation'),
      key: 'actions',
      width: 80,
      render: (_: unknown, row: MmeIpPlmnRow) => (
        <Button
          danger
          type="text"
          icon={<DeleteOutlined />}
          disabled={disabled}
          onClick={() => removeRow(row.key)}
        />
      ),
    },
  ];

  return (
    <Space orientation="vertical" style={{ width: '100%' }} size={8}>
      <Table<MmeIpPlmnRow>
        size="small"
        rowKey="key"
        pagination={false}
        columns={columns}
        dataSource={rows}
      />
      <AddRowButton
        onClick={addRow}
        disabled={disabled || maxReached}
      >
        {t('device.cell.addRow')}
      </AddRowButton>
      {maxRows !== undefined && (
        <Alert
          type={maxReached ? 'warning' : 'info'}
          showIcon
          message={t('device.cell.mmeIpPlmnLimitHint', { max: maxRows })}
        />
      )}
    </Space>
  );
}

function buildSpecialFieldConfig(
  groupId: string,
  paramName: string,
  instanceContext: QuickSettingsInstanceContext,
): SpecialFieldConfig | null {
  if (groupId === 'gsm-abis' && paramName === 'GsmBtsBindMib') {
    return {
      kind: 'bind-select',
      configPath: applyInstanceContext('Device.Services.GsmBTSCellDT.{i}.GsmBtsBindMib', instanceContext),
      displayPath: applyInstanceContext('Device.Services.GsmBTSCellDT.{i}.GsmBtsIpAddr', instanceContext),
      forceWritable: true,
    };
  }
  if (groupId === 'enb-mme' && paramName === 'MmeIpPlmnList') {
    return {
      kind: 'mme-ip-plmn-table',
      configPath: applyInstanceContext('Device.Services.FAPService.{i}.FAPControl.LTE.Gateway.MmeIpPlmnList', instanceContext),
      forceWritable: true,
    };
  }
  if (groupId === 'enb-plmn' && paramName === 'ExistPlmnidList') {
    return {
      kind: 'plmn-list-table',
      configPath: applyInstanceContext('Device.Services.FAPService.{i}.FAPControl.LTE.Gateway.ExistPlmnidList', instanceContext),
      forceWritable: true,
    };
  }
  if (groupId === 'gnb-core' && paramName === 'gNBName') {
    return {
      kind: 'input',
      configPath: 'Device.Services.FAPService.1.FAPControl.NR.RAN.Common.gNBName',
      forceWritable: true,
    };
  }
  if (groupId === 'gnb-core' && paramName === 'NguBindInterface') {
    return {
      kind: 'bind-select',
      configPath: applyInstanceContext('Device.FAP.NguIpBind{i}.BindInterface', instanceContext),
      displayPath: applyInstanceContext('Device.FAP.NguIpBind{i}.NguLocalIp', instanceContext),
      bindValueMode: 'path',
      fallbackConfigPath: 'Device.LAN_HostConfigManagement.IPInterface.NgapMgmt.NguLocalIpAddrList',
      fallbackDisplayPath: 'Device.LAN_HostConfigManagement.IPInterface.NgapMgmt.NguLocalIpAddrList',
      fallbackBindValueMode: 'ip',
      forceWritable: true,
    };
  }
  return null;
}

function normalizeInterfaceType(value?: string | null): string {
  return String(value ?? '').trim().toLowerCase();
}

function buildBindSelectOptions(
  parameters: ParameterSchemaItem[],
  valueMode: BindSelectValueMode = 'path',
): BindSelectOption[] {
  const interfaceKinds = new Map<string, string>();
  for (const item of parameters) {
    const match = /^Device\.Ethernet\.Interface\.(\d+)\.interfaceType$/i.exec(item.path);
    if (!match) continue;
    interfaceKinds.set(match[1], normalizeInterfaceType(item.currentValue));
  }

  const options: BindSelectOption[] = [];
  for (const item of parameters) {
    const currentValue = String(item.currentValue ?? '').trim();
    if (!currentValue) continue;

    const directMatch = /^Device\.Ethernet\.Interface\.(\d+)\.(IPv[46]Address\.\d+\.IPAddress)$/.exec(item.path);
    if (directMatch) {
      if (interfaceKinds.get(directMatch[1]) === 'wan') {
        options.push({
          value: valueMode === 'ip' ? currentValue : item.path,
          label: `${currentValue}`,
        });
      }
      continue;
    }

    const vlanMatch = /^Device\.Ethernet\.Interface\.(\d+)\.VlanInterface\.\d+\.(IPv[46]Address\.\d+\.IPAddress)$/.exec(item.path);
    if (vlanMatch && interfaceKinds.get(vlanMatch[1]) === 'wan') {
      options.push({
        value: valueMode === 'ip' ? currentValue : item.path,
        label: `${currentValue}`,
      });
    }
  }

  return options.sort((left, right) => left.label.localeCompare(right.label));
}

function appendCurrentBindOption(
  options: BindSelectOption[],
  currentPath: string,
  displayValue: string,
): BindSelectOption[] {
  if (!currentPath || options.some((option) => option.value === currentPath)) {
    return options;
  }
  const label = displayValue || 'Unknown IP';
  return [{ value: currentPath, label }, ...options];
}

function getRawValueByPath(rawParameterByPath: Map<string, DeviceParameter>, path: string): DeviceParameter | undefined {
  return rawParameterByPath.get(path);
}

function findRawValueBySuffix(parameters: DeviceParameter[] | undefined, suffix: string): DeviceParameter | undefined {
  if (!parameters) return undefined;
  return parameters.find((item) => item.parameterPath.endsWith(suffix));
}

/**
 * 单实例分组表单（如「小区参数」）。
 *
 * 行为：
 *  1. 把 group.params 的 standardPath 中 {i} 替换为当前 fapInstance
 *  2. 通过 useParameterSchema 拉每条路径的 schema（类型/约束/当前值）
   *  3. 渲染为 3 列网格 Form
 *  4. 顶部"保存"按钮收集本表单全部脏字段，一次性 SetParameterValues
 *  5. Save 部分失败时按字段标红保留输入值（继承 Antd Form 校验/状态行为）
 */
export default function CellParameterForm({
  deviceId,
  paramModel,
  active = true,
  group,
  instanceContext,
  locale,
  onIpsecControlChange,
  actionMode = 'standalone',
}: CellParameterFormProps) {
  const t = useT();
  const [form] = Form.useForm();
  const gnssSyncSourceRef = useRef<string[]>([]);
  const watchedLocalTimeZoneName = Form.useWatch('LocalTimeZoneName', form);
  const watchedIpsecEnable = Form.useWatch('IPSEC_ENABLE', form);
  const watchedPpsTimeMode = Form.useWatch('PpsTimeMode', form);
  const watchedDeviceTimeEnable = Form.useWatch('Enable', form);
  const dlSubCarrierSpacing = Form.useWatch('DLSubCarrierSpacing', form);
  const ulSubCarrierSpacing = Form.useWatch('ULSubCarrierSpacing', form);
  const [deviceTimeShowNtpServerFields, setDeviceTimeShowNtpServerFields] = useState(true);
  const updateMutation = useUpdateParameters();
  const queryClient = useQueryClient();
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const feedbackScope = useMemo(() => getFeedbackScopeContext(group.id, instanceContext), [group.id, instanceContext]);

  // lastSubmit 由 zustand store 托管 —— DeviceDetail 整个被卸载(切顶层 tab)也保留反馈。
  const fbKey = feedbackKey(
    deviceId,
    group.id,
    feedbackScope.fapInstance,
    feedbackScope.networkType === 'nr' ? feedbackScope.cellInstance : undefined,
  );
  const lastSubmit = useQuickSettingsFeedbackStore((s) => {
    const f = s.entries[fbKey];
    return f && f.kind === 'cell' ? f : null;
  });
  const setFeedback = useQuickSettingsFeedbackStore((s) => s.setFeedback);
  const patchFeedback = useQuickSettingsFeedbackStore((s) => s.patchFeedback);
  const draft = useQuickSettingsFeedbackStore((s) => s.drafts[fbKey]);
  const draftRevision = useQuickSettingsFeedbackStore((s) => s.draftRevisions[fbKey] ?? 0);
  const setDraftField = useQuickSettingsFeedbackStore((s) => s.setDraftField);
  const clearDraft = useQuickSettingsFeedbackStore((s) => s.clearDraft);
  const isDeviceTimeGroup = group.id === 'device-time';
  const isBmSyncSourceGroup = group.id === 'bm-sync-source';
  const isGnbSyncSourceGroup = group.id === 'gnb-sync-source';
  const isMlnIndexedMmePool = group.id === 'enb-mme' && isMlnIndexedMmePoolModel(paramModel);
  const restrictMmePlmnSelection = group.id === 'enb-mme'
    && isServingPlmnRestrictedModel(paramModel);
  const preferSchemaCurrentValue = isDeviceTimeGroup
    || isGnbSyncSourceGroup
    || group.id === 'device-ipsec-control'
    || group.id === 'enb-plmn'
    || group.id === 'enb-mme'
    || group.id === 'gnb-core';
  const isEffectiveBitmaskPath = useCallback(
    (path: string) => !isGnbSyncSourceGroup && isBitmaskSelectPath(path),
    [isGnbSyncSourceGroup],
  );
  const isConstrainedGnssSyncSourceSelectPath = useCallback(
    (path: string) => (isBmSyncSourceGroup || isGnbSyncSourceGroup) && isConstrainedGnssSyncSourcePath(path),
    [isBmSyncSourceGroup, isGnbSyncSourceGroup],
  );
  const effectiveParams = useMemo<QuickSettingsParam[]>(() => {
    if (!isDeviceTimeGroup || group.params.some((param) => param.name === 'Enable')) {
      return group.params;
    }
    return [
      {
        name: 'Enable',
        titleZh: 'NTP 模式',
        titleEn: 'NTP Mode',
        standardPath: 'Device.Time.Enable',
        enumOptions: [
          { value: '1', label: 'NTP Server' },
          { value: '0', label: 'NTP Client' },
        ],
      },
      ...group.params,
    ];
  }, [group.params, isDeviceTimeGroup]);

  // 注: osmo-bsc 把 NumofTrxChannel / OmlConnectState 这两个 BSC 全局只读状态量
  // 暴露在 DeviceGSM.Bts.0 实例(固定索引 0)上,所有 BTS 实例共享同一份值。
  // 前端配置 (quicksettings/BSC.xml) 中 standardPath 固定写 Bts.0,
  // applyInstanceContext 对不含 {i} 的 path 原样返回,切换 BTS 实例时仍读到同一根 leaf。
  const resolveReadPath = useCallback(
    (standardPath: string) => applyInstanceContext(standardPath || '', instanceContext),
    [instanceContext],
  );

  // 单实例分组：每条 standardPath 单独查 schema（少量字段，不批量优化）
  // 注：useParameterSchema 接受 pathPrefix，前缀匹配即可；这里以分组共用前缀粗查再过滤
  // 为简化，取 group 中 standardPath 的公共前缀作 pathPrefix
  const commonPrefix = useMemo(
    () => (
      isDeviceTimeGroup || isGnbSyncSourceGroup
        ? ''
        : isMlnIndexedMmePool
        ? MLN_MME_POOL_STANDARD_PREFIX
        : commonPathPrefix(effectiveParams.map((p) => resolveReadPath(p.standardPath || '')))
    ),
    [isDeviceTimeGroup, isGnbSyncSourceGroup, isMlnIndexedMmePool, effectiveParams, resolveReadPath],
  );
  const { data: schemaResp, isLoading: isCommonSchemaLoading, refetch: refetchCommonSchema } = useParameterSchema(
    deviceId,
    commonPrefix,
    active && !isDeviceTimeGroup && !isGnbSyncSourceGroup,
  );
  const { data: deviceTimeSchemaResp, isLoading: isDeviceTimeSchemaLoading, refetch: refetchDeviceTimeSchema } = useParameterSchema(
    deviceId,
    'Device.Time.',
    active && isDeviceTimeGroup,
  );
  const {
    data: managementServerSchemaResp,
    isLoading: isManagementServerSchemaLoading,
    refetch: refetchManagementServerSchema,
  } = useParameterSchema(
    deviceId,
    'Device.ManagementServer.',
    active && isDeviceTimeGroup,
  );
  const { data: ethernetSchemaResp } = useParameterSchema(
    deviceId,
    'Device.Ethernet.Interface.',
    active && (group.id === 'gsm-abis' || group.id === 'gnb-core'),
  );
  const { data: mmeIpPlmnParams } = useSearchParameters(
    deviceId,
    group.id === 'enb-mme'
      ? (isMlnIndexedMmePool ? 'MmePoolConfigParam' : 'MmeIpPlmnList')
      : '',
    isMlnIndexedMmePool ? 100 : 50,
    active && group.id === 'enb-mme',
  );
  const { data: servingPlmnParams, isLoading: isServingPlmnLoading } = useSearchParameters(
    deviceId,
    restrictMmePlmnSelection ? getServingPlmnSearchQuery(paramModel) : '',
    50,
    active && restrictMmePlmnSelection,
  );
  const { data: nrCommonParams } = useSearchParameters(
    deviceId,
    group.id === 'gnb-core' ? 'Device.Services.FAPService.1.FAPControl.NR.RAN.Common.' : '',
    50,
    active && group.id === 'gnb-core',
  );
  const { data: nrNguParams } = useSearchParameters(
    deviceId,
    group.id === 'gnb-core' ? 'Device.FAP.NguIpBind' : '',
    200,
    active && group.id === 'gnb-core',
  );
  const { data: nrNguFallbackParams } = useSearchParameters(
    deviceId,
    group.id === 'gnb-core' ? 'Device.LAN_HostConfigManagement.IPInterface.NgapMgmt.' : '',
    50,
    active && group.id === 'gnb-core',
  );
  const { data: deviceTimeParams } = useSearchParameters(
    deviceId,
    group.id === 'device-time' ? 'Device.Time.' : '',
    200,
    active && group.id === 'device-time',
  );
  const { data: bmPpsTimeModeParams } = useSearchParameters(
    deviceId,
    isBmSyncSourceGroup ? BM_PPS_TIME_MODE_PATH : '',
    20,
    active && isBmSyncSourceGroup,
  );
  const { data: bmGnssSyncSourceParams } = useSearchParameters(
    deviceId,
    isBmSyncSourceGroup ? BM_GNSS_SYNC_SOURCE_PATH : '',
    20,
    active && isBmSyncSourceGroup,
  );
  const { data: bmPtpConfigParams } = useSearchParameters(
    deviceId,
    isBmSyncSourceGroup ? BM_PTP_CONFIG_PREFIX : '',
    50,
    active && isBmSyncSourceGroup,
  );
  const {
    data: gnbSyncFapSchemaResp,
    isLoading: isGnbSyncFapSchemaLoading,
    refetch: refetchGnbSyncFapSchema,
  } = useParameterSchema(
    deviceId,
    'Device.FAP.',
    active && isGnbSyncSourceGroup,
  );
  const {
    data: gnbSyncDeviceInfoSchemaResp,
    isLoading: isGnbSyncDeviceInfoSchemaLoading,
    refetch: refetchGnbSyncDeviceInfoSchema,
  } = useParameterSchema(
    deviceId,
    'Device.DeviceInfo.',
    active && isGnbSyncSourceGroup,
  );
  const { data: ipsecControlParams } = useSearchParameters(
    deviceId,
    group.id === 'device-ipsec-control' ? (effectiveParams[0]?.standardPath || 'IPSEC_ENABLE') : '',
    20,
    active && group.id === 'device-ipsec-control',
  );
  const visibleParams = useMemo(() => {
    if (isBmSyncSourceGroup) {
      const ppsValue = watchedPpsTimeMode
        ?? draft?.PpsTimeMode
        ?? bmPpsTimeModeParams?.find((item) => item.parameterPath === BM_PPS_TIME_MODE_PATH)?.parameterValue;
      const showGnssSyncSource = bitmaskHasBit(ppsValue, 1);
      const showPtpConfig = bitmaskHasBit(ppsValue, 2);
      return effectiveParams.filter((param) => {
        const path = param.standardPath || '';
        if (!showGnssSyncSource && isBmGnssSyncSourcePath(path)) {
          return false;
        }
        if (!showPtpConfig && isBmPtpConfigPath(path)) {
          return false;
        }
        return true;
      });
    }
    if (isGnbSyncSourceGroup) {
      const modeValue = watchedPpsTimeMode
        ?? draft?.PpsTimeMode
        ?? gnbSyncFapSchemaResp?.parameters.find((item) => item.path === BM_PPS_TIME_MODE_PATH)?.currentValue;
      const mode = String(modeValue ?? '');
      const showGnssFields = GNB_SYNC_MODE_GNSS_VALUES.has(mode);
      const showPtpFields = GNB_SYNC_MODE_PTP_VALUES.has(mode);
      return effectiveParams.filter((param) => {
        if (param.name === 'PpsTimeMode') {
          return true;
        }
        const path = resolveReadPath(param.standardPath || '');
        if (path === GNB_GPS_SYNC_SOURCE_PATH) {
          return showGnssFields;
        }
        if (isGnbPtpConfigPath(path)) {
          return showPtpFields;
        }
        if (isGnbCommonSyncPath(path)) {
          return showGnssFields || showPtpFields;
        }
        return false;
      });
    }
    if (!isDeviceTimeGroup) {
      return effectiveParams;
    }
    return effectiveParams.filter((param) => {
      const path = param.standardPath || '';
      if (!deviceTimeShowNtpServerFields && isNtpServerPath(path)) {
        return false;
      }
      return true;
    });
  }, [bmPpsTimeModeParams, draft, deviceTimeShowNtpServerFields, effectiveParams, gnbSyncFapSchemaResp, isBmSyncSourceGroup, isDeviceTimeGroup, isGnbSyncSourceGroup, watchedPpsTimeMode, resolveReadPath]);
  const visibleParamNameSet = useMemo(
    () => new Set(visibleParams.map((param) => param.name)),
    [visibleParams],
  );
  const isGsmCell = group.id === 'gsm-cell';
  const isLteCellGroup = group.id === 'enb-cell';
  const hasGsmRuRelation = visibleParamNameSet.has('GsmCellWithRuRelation');
  const hasGsmTxAntNum = visibleParamNameSet.has('GsmTxAntNum');
  const hasLteRuRouteIndex = visibleParamNameSet.has('LteCellWithRuList');
  const hasLteAntennaPortsCount = visibleParamNameSet.has('AntennaPortsCount');
  const isBmGsmCell = isGsmCell && hasGsmRuRelation;
  const isBmLteCell = isLteCellGroup && hasLteRuRouteIndex;
  const isHiddenTransmissionCountParam = useCallback((name: string) => (
    (isBmGsmCell && hasGsmTxAntNum && name === 'GsmTxAntNum') ||
    (isBmLteCell && hasLteAntennaPortsCount && name === 'AntennaPortsCount')
  ), [hasGsmTxAntNum, hasLteAntennaPortsCount, isBmGsmCell, isBmLteCell]);
  // XML 驱动的 extraInfoPath:在某个字段下方以小字展示另一个只读参数当前值(范围提示)。
  // 由 quicksettings XML 在 <param> 上声明 extraInfoPath="Device.X.Y",前端按该路径拉 schema,
  // 把 currentValue 按 [lo ~ hi] 格式渲染到对应 Form.Item 的 extra 槽位。
  const extraInfoPaths = useMemo(() => {
    const set = new Set<string>();
    for (const p of visibleParams) {
      if (p.extraInfoPath) set.add(p.extraInfoPath);
    }
    return Array.from(set);
  }, [visibleParams]);
  const extraInfoCommonPrefix = useMemo(() => {
    if (extraInfoPaths.length === 0) return '';
    if (extraInfoPaths.length === 1) return extraInfoPaths[0];
    let prefix = extraInfoPaths[0];
    for (const p of extraInfoPaths.slice(1)) {
      let i = 0;
      const max = Math.min(prefix.length, p.length);
      while (i < max && prefix[i] === p[i]) i++;
      prefix = prefix.slice(0, i);
    }
    return prefix;
  }, [extraInfoPaths]);
  const { data: extraInfoSchemaResp } = useParameterSchema(
    deviceId,
    extraInfoCommonPrefix,
    active && extraInfoPaths.length > 0,
  );
  const extraInfoValueByPath = useMemo(() => {
    const m = new Map<string, string>();
    for (const path of extraInfoPaths) {
      const item = extraInfoSchemaResp?.parameters?.find((q) => q.path === path);
      const v = item?.currentValue ?? '';
      m.set(path, typeof v === 'string' ? v : String(v ?? ''));
    }
    return m;
  }, [extraInfoPaths, extraInfoSchemaResp]);
  // 由 extraInfoPath 当前值反推数值范围 [min, max],用于该字段输入校验:
  // "24,30" / "24~30" / "24-30" 都拆成两端;非两段或非数字时返回 null,跳过校验。
  const extraInfoBoundsByName = useMemo(() => {
    const m = new Map<string, [number, number]>();
    for (const p of visibleParams) {
      if (!p.extraInfoPath) continue;
      const raw = extraInfoValueByPath.get(p.extraInfoPath);
      const bounds = parseExtraInfoBounds(raw ?? '');
      if (bounds) m.set(p.name, bounds);
    }
    return m;
  }, [visibleParams, extraInfoValueByPath]);

  // BM 小区专属:并行拉 RU 节点 schema,用于 GSM/LTE Route Index 候选。
  // 拉取与主 schema 解耦,避免污染 commonPrefix 退化成 Device. 触发全量拉取。
  const { data: ruSchemaResp } = useParameterSchema(
    deviceId,
    'Device.DeviceInfo.RU.',
    active && (isBmGsmCell || isBmLteCell),
  );
  const effectiveSchemaParameters = useMemo(
    () => (isDeviceTimeGroup
      ? [
        ...(deviceTimeSchemaResp?.parameters ?? []),
        ...(managementServerSchemaResp?.parameters ?? []),
      ]
      : isGnbSyncSourceGroup
      ? [
        ...(gnbSyncFapSchemaResp?.parameters ?? []),
        ...(gnbSyncDeviceInfoSchemaResp?.parameters ?? []),
      ]
      : (schemaResp?.parameters ?? [])),
    [isDeviceTimeGroup, isGnbSyncSourceGroup, deviceTimeSchemaResp, managementServerSchemaResp, gnbSyncFapSchemaResp, gnbSyncDeviceInfoSchemaResp, schemaResp],
  );
  const isSchemaLoading = isDeviceTimeGroup
    ? (isDeviceTimeSchemaLoading || isManagementServerSchemaLoading)
    : isGnbSyncSourceGroup
    ? (isGnbSyncFapSchemaLoading || isGnbSyncDeviceInfoSchemaLoading)
    : isCommonSchemaLoading;
  const hasSchemaData = isDeviceTimeGroup
    ? Boolean(deviceTimeSchemaResp && managementServerSchemaResp)
    : isGnbSyncSourceGroup
    ? Boolean(gnbSyncFapSchemaResp && gnbSyncDeviceInfoSchemaResp)
    : Boolean(schemaResp);
  const ruRouteItemByIdx = useMemo(() => {
    const map = new Map<string, ParameterSchemaItem>();
    ruSchemaResp?.parameters.forEach((p) => {
      // 形如 Device.DeviceInfo.RU.<n>.RouteIndex
      const m = /^Device\.DeviceInfo\.RU\.(\d+)\.RouteIndex$/.exec(p.path);
      if (m) {
        map.set(m[1], p);
      }
    });
    return map;
  }, [ruSchemaResp]);

  const schemaByPath = useMemo(() => {
    const map = new Map<string, ParameterSchemaItem>();
    effectiveSchemaParameters.forEach((p) => map.set(p.path, p));
    return map;
  }, [effectiveSchemaParameters]);
  const mlnMmePoolCurrentValues = useMemo(() => [
    ...(mmeIpPlmnParams ?? []).map((item) => ({
      parameterPath: item.parameterPath,
      parameterValue: item.parameterValue,
    })),
    ...effectiveSchemaParameters
      .filter((item) => item.currentValue !== undefined && item.currentValue !== null)
      .map((item) => ({
        parameterPath: item.path,
        parameterValue: String(item.currentValue),
      })),
  ], [effectiveSchemaParameters, mmeIpPlmnParams]);
  const servingPlmnOptions = useMemo(() => {
    return getServingPlmnOptionsForModel(
      paramModel,
      instanceContext.fapInstance,
      servingPlmnParams ?? [],
    );
  }, [instanceContext.fapInstance, paramModel, servingPlmnParams]);
  const rawParameterByPath = useMemo(() => {
    const map = new Map<string, DeviceParameter>();
    for (const item of [
      ...(mmeIpPlmnParams ?? []),
      ...(nrCommonParams ?? []),
      ...(nrNguParams ?? []),
      ...(nrNguFallbackParams ?? []),
      ...(deviceTimeParams ?? []),
      ...(bmPpsTimeModeParams ?? []),
      ...(bmGnssSyncSourceParams ?? []),
      ...(bmPtpConfigParams ?? []),
      ...(ipsecControlParams ?? []),
    ]) {
      map.set(item.parameterPath, item);
    }
    return map;
  }, [mmeIpPlmnParams, nrCommonParams, nrNguParams, nrNguFallbackParams, deviceTimeParams, bmPpsTimeModeParams, bmGnssSyncSourceParams, bmPtpConfigParams, ipsecControlParams]);
  useEffect(() => {
    if (!isDeviceTimeGroup) {
      setDeviceTimeShowNtpServerFields(true);
      return;
    }

    const currentMode = watchedDeviceTimeEnable
      ?? draft?.Enable
      ?? inferDeviceTimeMode(rawParameterByPath, schemaByPath, instanceContext.networkType);
    setDeviceTimeShowNtpServerFields(
      shouldShowDeviceTimeNtpServerFields(instanceContext.networkType, String(currentMode ?? '')),
    );
  }, [draft?.Enable, instanceContext.networkType, isDeviceTimeGroup, rawParameterByPath, schemaByPath, watchedDeviceTimeEnable]);
  const timeZoneParam = useMemo(
    () => visibleParams.find((param) => param.name === 'LocalTimeZoneName'),
    [visibleParams],
  );
  const timeZonePath = timeZoneParam ? resolveReadPath(timeZoneParam.standardPath || '') : '';
  const timeZoneSchemaItem = timeZonePath ? schemaByPath.get(timeZonePath) : undefined;
  const timezoneOptions = useMemo(() => {
    const enumValues = timeZoneSchemaItem?.constraints?.enumValues ?? [];
    if (enumValues.length > 0) {
      return enumValues.map((value) => ({
        value,
        label: formatTimeZoneDisplay(value),
      }));
    }
    return getTimezoneAliasOptions();
  }, [timeZoneSchemaItem]);
  const displayedTimezoneOptions = useMemo(
    () => timezoneOptions.map((option) => ({
      ...option,
      label: formatTimeZoneDisplay(option.label),
    })),
    [timezoneOptions],
  );
  const nrServerLabel = t('device.cell.ntpServer');
  const nrClientLabel = t('device.cell.ntpClient');
  const enableLabel = t('common.enable');
  const disableLabel = t('common.disable');
  const deviceTimeModeOptions = useMemo(() => {
    const meta = getEffectiveEnumMeta(schemaByPath.get('Device.Time.Enable')?.constraints, 'Device.Time.Enable');
    if (meta && meta.values.length > 0) {
      return meta.values.map((value, index) => ({
        value,
        label: mapDeviceTimeModeLabel(value, meta.labels[index] || value, instanceContext.networkType, {
          nrServer: nrServerLabel,
          nrClient: nrClientLabel,
          enable: enableLabel,
          disable: disableLabel,
        }),
      }));
    }
    if (isNrNetworkType(instanceContext.networkType)) {
      return [
        { value: '1', label: nrServerLabel },
        { value: '0', label: nrClientLabel },
      ];
    }
    return [
      { value: '1', label: enableLabel },
      { value: '0', label: disableLabel },
    ];
  }, [schemaByPath, instanceContext.networkType, nrServerLabel, nrClientLabel, enableLabel, disableLabel]);
  const deviceTimeModeOptionsKey = deviceTimeModeOptions.map((option) => option.value).join('\u0000');
  const bindSelectOptions = useMemo(
    () => buildBindSelectOptions(ethernetSchemaResp?.parameters ?? []),
    [ethernetSchemaResp],
  );
  const bindIpValueOptions = useMemo(
    () => buildBindSelectOptions(ethernetSchemaResp?.parameters ?? [], 'ip'),
    [ethernetSchemaResp],
  );
  const bindIpByPath = useMemo(() => {
    const map = new Map<string, string>();
    for (const option of bindSelectOptions) {
      map.set(option.value, option.label);
    }
    return map;
  }, [bindSelectOptions]);
  const specialConfigByName = useMemo(() => {
    const map = new Map<string, SpecialFieldConfig>();
    for (const param of visibleParams) {
      const config = buildSpecialFieldConfig(group.id, param.name, instanceContext);
      if (config) {
        map.set(param.name, config);
      }
    }
    return map;
  }, [visibleParams, group.id, instanceContext]);
  const resolveRuntimeSpecialConfig = useCallback((paramName: string): SpecialFieldConfig | undefined => {
    const special = specialConfigByName.get(paramName);
    if (!special?.fallbackConfigPath) {
      return special;
    }

    const primaryAvailable = Boolean(
      getRawValueByPath(rawParameterByPath, special.configPath)
      || (special.displayPath && getRawValueByPath(rawParameterByPath, special.displayPath))
      || (paramName === 'NguBindInterface' && (
        findRawValueBySuffix(nrNguParams, '.BindInterface')
        || findRawValueBySuffix(nrNguParams, '.NguLocalIp')
      )),
    );
    if (primaryAvailable) {
      return special;
    }
    return {
      ...special,
      configPath: special.fallbackConfigPath,
      displayPath: special.fallbackDisplayPath ?? special.fallbackConfigPath,
      bindValueMode: special.fallbackBindValueMode ?? special.bindValueMode,
    };
  }, [nrNguParams, rawParameterByPath, specialConfigByName]);

  // T-0159: 交叉镜像 — 反查表 resolved standardPath → form field name，
  // 让 onValuesChange 时能据 constraints.mirrorWith 找到对端 form field 并 setFieldValue 同步。
  const paramNameByPath = useMemo(() => {
    const map = new Map<string, string>();
    for (const p of visibleParams) {
      const path = resolveReadPath(p.standardPath || '');
      map.set(path, p.name);
    }
    return map;
  }, [visibleParams, instanceContext, resolveReadPath]);

  // 初始化字段值 —— 优先级：store draft > 当前会话已 touched > schema 原值。
  // 未保存草稿在跨顶层 TabBar 切换后恢复；任务终态回读后再 clearDraft，统一回到设备侧值。
  useEffect(() => {
    if (!hasSchemaData) return;
    visibleParams.forEach((p) => {
      const currentValue = form.getFieldValue(p.name);
      const draftPath = resolveReadPath(p.standardPath || '');
      const draftItem = schemaByPath.get(draftPath);
      const enumValuesForPath = p.enumOptions?.map((option) => option.value) ?? draftItem?.constraints?.enumValues ?? [];
      if (draft && draft[p.name] !== undefined) {
        const specialKind = resolveRuntimeSpecialConfig(p.name)?.kind;
        const nextValue = specialKind === 'mme-ip-plmn-table'
          ? toMmeIpPlmnRows(draft[p.name])
          : specialKind === 'plmn-list-table'
          ? toPlmnRows(draft[p.name])
          : isConstrainedGnssSyncSourceSelectPath(draftPath)
          ? normalizeConstrainedGnssSyncSourceValue(draft[p.name], enumValuesForPath, draftPath)
          : isEffectiveBitmaskPath(draftPath)
          ? normalizeBitmaskValue(draft[p.name])
          : p.type === 'multiCheckbox'
          ? isBscCodecSupportParam(p.standardPath ?? p.name)
            ? normalizeBscCodecSupportValue(draft[p.name])
            : parseQuickSettingsMultiCheckboxValue(draft[p.name])
          : isStringMultiSelectPath(draftPath)
          ? normalizeStringMultiSelectValue(draft[p.name])
          : isSwitchPath(draftPath)
          ? normalizeSwitchValue(draft[p.name])
          : String(draft[p.name] ?? '');
        if (!formValueEquals(currentValue, nextValue)) {
          form.setFieldValue(p.name, nextValue);
        }
        if (isConstrainedGnssSyncSourceSelectPath(draftPath)) {
          gnssSyncSourceRef.current = Array.isArray(nextValue) ? nextValue.map(String) : [];
        }
        return;
      }
      // 优先级 2: 用户当前存在未保存草稿时，保留本地编辑。
      // 仅 touched 但无 draft（例如刷新后 watcher 已清草稿）应允许被最新 schema 回填，
      // 否则会出现“刷新成功但需切页再回来才看到新值”。
      if (form.isFieldTouched(p.name) && draft?.[p.name] !== undefined) return;
      // 优先级 3: schema 原值
      const special = resolveRuntimeSpecialConfig(p.name);
      const path = special?.configPath ?? resolveReadPath(p.standardPath || '');
      const item = schemaByPath.get(path);
      const rawItem = getRawValueByPath(rawParameterByPath, path)
        ?? (special?.kind === 'mme-ip-plmn-table'
          ? findRawValueBySuffix(mmeIpPlmnParams, '.MmeIpPlmnList')
          : special?.kind === 'bind-select' && p.name === 'NguBindInterface'
            ? findRawValueBySuffix(nrNguParams, '.BindInterface') ?? findRawValueBySuffix(nrNguFallbackParams, '.NguLocalIpAddrList')
            : undefined);
      if (special?.kind === 'mme-ip-plmn-table') {
        if (isMlnIndexedMmePool) {
          form.setFieldValue(p.name, parseMlnMmePoolRows(mlnMmePoolCurrentValues));
        } else {
          const raw = preferSchemaCurrentValue
            ? (item?.currentValue ?? rawItem?.parameterValue ?? '')
            : (rawItem?.parameterValue ?? item?.currentValue ?? '');
          form.setFieldValue(p.name, toMmeIpPlmnRows(raw));
        }
      } else if (special?.kind === 'plmn-list-table') {
        const raw = preferSchemaCurrentValue
          ? (item?.currentValue ?? rawItem?.parameterValue ?? '')
          : (rawItem?.parameterValue ?? item?.currentValue ?? '');
        form.setFieldValue(p.name, toPlmnRows(raw));
      } else {
        // 这些分组会同时读 search + schema。优先采用 schema 当前值，避免 search 缓存
        // 在 refreshTick remount 后短暂覆盖刚回读的新值。
        let raw = preferSchemaCurrentValue
          ? (item?.currentValue ?? rawItem?.parameterValue ?? '')
          : (rawItem?.parameterValue ?? item?.currentValue ?? '');
        if (isDeviceTimeGroup && p.name === 'Enable' && (raw === '' || raw == null)) {
          raw = inferDeviceTimeMode(rawParameterByPath, schemaByPath, instanceContext.networkType);
        }
        // BSC osmo-bsc 不上报 DeviceGSM.Bts.{i}.ID,该字段语义即为 BTS 实例号本身,
        // 此处按实例号派生填充,避免显示"未上报"。
        if (
          (raw === '' || raw == null) &&
          (p.standardPath || '') === 'DeviceGSM.Bts.{i}.ID' &&
          instanceContext.fapInstance != null
        ) {
          raw = String(instanceContext.fapInstance);
        }
        const enumOptions = isDeviceTimeGroup && p.name === 'Enable' ? deviceTimeModeOptions : p.enumOptions;
        const allowedEnumValues = enumOptions?.map((option) => option.value) ?? item?.constraints?.enumValues ?? [];
        if (isConstrainedGnssSyncSourceSelectPath(path)) {
          const nextValue = normalizeConstrainedGnssSyncSourceValue(raw, allowedEnumValues, path);
          form.setFieldValue(p.name, nextValue);
          gnssSyncSourceRef.current = nextValue;
        } else if (isEffectiveBitmaskPath(path)) {
          form.setFieldValue(p.name, normalizeBitmaskValue(raw));
        } else if (p.type === 'multiCheckbox') {
          form.setFieldValue(
            p.name,
            isBscCodecSupportParam(path)
              ? normalizeBscCodecSupportValue(raw)
              : parseQuickSettingsMultiCheckboxValue(raw),
          );
        } else if (isStringMultiSelectPath(path)) {
          form.setFieldValue(p.name, normalizeStringMultiSelectValue(raw));
        } else if (isSwitchPath(path)) {
          form.setFieldValue(p.name, normalizeSwitchValue(raw));
        } else {
          form.setFieldValue(
            p.name,
            normalizeEnumValue(
              raw,
              enumOptions,
            ),
          );
        }
      }
    });
  }, [hasSchemaData, visibleParams, instanceContext, form, schemaByPath, rawParameterByPath, draft, resolveRuntimeSpecialConfig, mmeIpPlmnParams, nrNguParams, nrNguFallbackParams, preferSchemaCurrentValue, isDeviceTimeGroup, deviceTimeModeOptionsKey, isEffectiveBitmaskPath, isConstrainedGnssSyncSourceSelectPath, isMlnIndexedMmePool, mlnMmePoolCurrentValues]);

  const handleSave = async () => {
    const values = form.getFieldsValue() as Record<string, unknown>;
    const updates: ParameterUpdateRequest[] = [];
    const errors: Record<string, string> = {};

    for (const p of visibleParams) {
      // readonly leaf(如 BTS ID / 共享只读状态量)不参与下发:它们的 path 在 param-mappings
      // 里多为 not_found / access=READ_ONLY,带进 SetParameterValues 会被后端 MappingValidator
      // 整批拒成 400,导致用户改任何字段都"入队失败"。
      if (p.readonly) continue;

      const special = resolveRuntimeSpecialConfig(p.name);
      const path = special?.configPath ?? resolveReadPath(p.standardPath || '');
      const item = schemaByPath.get(path);
      const rawItem = getRawValueByPath(rawParameterByPath, path)
        ?? (special?.kind === 'mme-ip-plmn-table'
          ? findRawValueBySuffix(mmeIpPlmnParams, '.MmeIpPlmnList')
          : special?.kind === 'bind-select' && p.name === 'NguBindInterface'
            ? findRawValueBySuffix(nrNguParams, '.BindInterface') ?? findRawValueBySuffix(nrNguFallbackParams, '.NguLocalIpAddrList')
            : undefined);

      if (isDeviceTimeGroup && !deviceTimeShowNtpServerFields && isNtpServerPath(path)) {
        continue;
      }

      if (special?.kind === 'mme-ip-plmn-table') {
        const normalizedRows = normalizeMmeIpPlmnRows(toMmeIpPlmnRows(values[p.name]));
        values[p.name] = normalizedRows;
        const rowsErr = validateMmeIpPlmnRows(normalizedRows);
        const limitErr = validateMmeIpPlmnLimit(normalizedRows, p.maxValue);
        const membershipIssue = restrictMmePlmnSelection
          ? findMmePlmnOutsideServingList(normalizedRows, servingPlmnOptions)
          : null;
        const membershipErr = membershipIssue
          ? t('device.cell.mmePlmnNotServing', {
            row: membershipIssue.row,
            plmn: membershipIssue.plmn,
          })
          : null;
        if (rowsErr || limitErr || membershipErr) {
          errors[p.name] = rowsErr ?? limitErr ?? membershipErr ?? '';
          continue;
        }
        if (isMlnIndexedMmePool) {
          updates.push(...buildMlnMmePoolUpdates(normalizedRows, mlnMmePoolCurrentValues));
          continue;
        }
      }
      if (special?.kind === 'plmn-list-table') {
        const normalizedRows = toPlmnRows(values[p.name]);
        values[p.name] = normalizedRows;
        const maxRows = p.maxValue ?? 6;
        const validationError = plmnValidationMessage(
          validatePlmnList(normalizedRows, maxRows),
          maxRows,
          t,
        );
        if (validationError) {
          errors[p.name] = validationError;
          continue;
        }
      }

      const isBitmaskField = isEffectiveBitmaskPath(path);
      const isMultiCheckboxField = p.type === 'multiCheckbox';
      const isCodecSupportField = isMultiCheckboxField && isBscCodecSupportParam(path);
      const isStringMultiSelectField = isStringMultiSelectPath(path);
      const isSwitchField = isSwitchPath(path);
      const newVal = special?.kind === 'mme-ip-plmn-table'
        ? serializeMmeIpPlmnList(toMmeIpPlmnRows(values[p.name]))
        : special?.kind === 'plmn-list-table'
        ? serializePlmnList(toPlmnRows(values[p.name]))
        : isBitmaskField
        ? serializeBitmaskValue(values[p.name])
        : isMultiCheckboxField
        ? isCodecSupportField
          ? serializeBscCodecSupportValue(values[p.name])
          : serializeQuickSettingsMultiCheckboxValue(values[p.name])
        : isStringMultiSelectField
        ? serializeStringMultiSelectValue(values[p.name])
        : isSwitchField
        ? serializeSwitchValue(values[p.name])
        : String(values[p.name] ?? '');
      const rawOldVal = special?.kind === 'mme-ip-plmn-table'
        ? (preferSchemaCurrentValue
          ? (item?.currentValue ?? rawItem?.parameterValue ?? '')
          : (rawItem?.parameterValue ?? item?.currentValue ?? ''))
        : (preferSchemaCurrentValue
          ? (item?.currentValue ?? rawItem?.parameterValue ?? '')
          : (rawItem?.parameterValue ?? item?.currentValue ?? ''));
      const oldVal = isSwitchField
        ? normalizeSwitchComparableValue(rawOldVal)
        : isMultiCheckboxField
        ? isCodecSupportField
          ? serializeBscCodecSupportValue(rawOldVal)
          : serializeQuickSettingsMultiCheckboxValue(rawOldVal)
        : rawOldVal;
      if (newVal === oldVal) continue;

      const parameterType = resolveQuickSettingsParameterType(p.type, item?.type, rawItem?.parameterType);
      const allowedValues = p.enumOptions?.map((option) => option.value) ?? item?.constraints?.enumValues ?? [];
      const err = isHiddenTransmissionCountParam(p.name)
        ? null
        : isBitmaskField
        ? isConstrainedGnssSyncSourceSelectPath(path)
          ? validateConstrainedGnssSyncSourceValue(newVal, allowedValues, path, t)
          : validateBitmaskValue(
          newVal,
          allowedValues,
          item?.constraints?.minValue,
          item?.constraints?.maxValue,
        )
        : isConstrainedGnssSyncSourceSelectPath(path)
        ? validateConstrainedGnssSyncSourceValue(newVal, allowedValues, path, t)
        : isStringMultiSelectField
        ? null
        : isMultiCheckboxField
        ? validateValue(newVal, parameterType, item?.constraints)
        : isSwitchField
        ? null
        : validateValue(newVal, parameterType, item?.constraints);
      if (err) {
        errors[p.name] = err;
        continue;
      }
      // XML 驱动的 extraInfoPath 范围校验:超出 [min, max] 阻断保存。
      const extraBounds = extraInfoBoundsByName.get(p.name);
      if (!isHiddenTransmissionCountParam(p.name) && extraBounds) {
        const rangeErr = validateExtraInfoBounds(newVal, extraBounds);
        if (rangeErr) {
          errors[p.name] = rangeErr;
          continue;
        }
      }
      updates.push({
        parameterPath: rawItem?.parameterPath ?? path,
        parameterValue: newVal,
        parameterType,
      });
    }

    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors);
      message.error({ content: t('device.cell.validationFailed'), duration: ERROR_FEEDBACK_DURATION_SECONDS });
      return;
    }

    if (updates.length === 0) {
      message.info({ content: t('device.cell.noChange'), duration: 4 });
      return;
    }

    setFieldErrors({});
    const submittedDraftRevision = useQuickSettingsFeedbackStore.getState().draftRevisions[fbKey] ?? 0;
    try {
      // 快速设置始终修改设备/LMT 侧参数。HNBName、gNBName 与其他参数一样
      // 通过 SetParameterValues 下发，不得转成网管设备改名操作。
      const result = await updateMutation.mutateAsync({ deviceId, parameters: updates });
      message.success({
        content: t('device.cell.saveSuccessMsg', { count: updates.length }),
        duration: 6,
      });
      setFeedback(fbKey, {
        kind: 'cell',
        submitStatus: 'queued',
        taskId: result.taskId,
        count: updates.length,
        at: Date.now(),
        expectedReadback: Object.fromEntries(
          updates.map((update) => [update.parameterPath, update.parameterValue]),
        ),
        submittedDraftRevision,
      });
    } catch (err) {
      const errMsg = err instanceof Error ? err.message : String(err);
      notification.error({
        message: t('device.cell.dispatchFailed', { group: group.titleZh }),
        description: t('device.multi.queueFailedDesc', { count: updates.length, err: errMsg }),
        duration: ERROR_FEEDBACK_DURATION_SECONDS,
      });
      setFeedback(fbKey, {
        kind: 'cell',
        submitStatus: 'failed_to_queue',
        count: updates.length,
        at: Date.now(),
        errorMsg: errMsg,
      });
      console.error('CellParameterForm: update failed', err);
    } finally {
      // T-0157 C10: 无论成功失败都触发消息中心 invalidate ——
      // 成功路径：后端 task.created publish → 订阅器写"进行中"消息，~50ms 内拉回
      // 失败路径：CreateFailureNotifier 兜底写"入队失败"消息，~50ms 内拉回
      void queryClient.invalidateQueries({ queryKey: notificationKeys.all });
    }
  };

  // T-0146:Save 后用 task_id 轮询真实 CPE 应答状态;到终态后停轮询。
  const { data: lastTask } = useDeviceTaskStatus(active ? lastSubmit?.taskId : undefined);

  // 任务终态后只 refetch 当前实例的 schema(精确到 commonPrefix 该份查询),
  // 拿到设备侧最新值后回填表单。不做跨 device 的全量 invalidate。
  useEffect(() => {
    if (!active) return;
    if (!lastTask || !['completed', 'failed', 'expired', 'cancelled'].includes(lastTask.status)) return;
    // 非成功终态必须保留本地草稿，供用户修正后重试；失败提示由下方独立 effect 负责。
    if (lastTask.status !== 'completed') return;
    if (!canApplySubmittedReadback({
      taskId: lastTask.id,
      syncedForTaskId: lastSubmit?.syncedForTaskId,
      submittedDraftRevision: lastSubmit?.submittedDraftRevision,
      currentDraftRevision: draftRevision,
    })) return;
    if (group.id === 'enb-plmn') {
      const submittedPlmn = Object.entries(lastSubmit?.expectedReadback ?? {})
        .find(([path]) => path.endsWith('.ExistPlmnidList'));
      if (submittedPlmn) {
        applyDeviceParameterSearchReadback(queryClient, {
          deviceId,
          searchQuery: getServingPlmnSearchQuery(paramModel),
          replacePathPrefix: submittedPlmn[0],
          parameters: [{
            parameterPath: submittedPlmn[0],
            parameterValue: submittedPlmn[1],
          }],
        });
      }
    }
    let cancelled = false;
    const abortController = new AbortController();
    void (async () => {
      let refreshedSchemaByPath = new Map<string, ParameterSchemaItem>();
      const fetchRefreshedSchema = async (): Promise<Map<string, ParameterSchemaItem>> => {
        deviceParameterApi.invalidateParameterSchemaCache(deviceId);
        await Promise.all([
          queryClient.invalidateQueries({
            queryKey: ['devices', 'parameters', 'search', deviceId],
            refetchType: 'none',
          }),
          queryClient.invalidateQueries({
            queryKey: ['devices', 'parameter-schema', deviceId],
            refetchType: 'none',
          }),
        ]);
        const nextSchemaByPath = new Map<string, ParameterSchemaItem>();
        if (isDeviceTimeGroup) {
          const [refreshedDeviceTime, refreshedManagementServer] = await Promise.all([
            refetchDeviceTimeSchema(),
            refetchManagementServerSchema(),
          ]);
          for (const item of refreshedDeviceTime.data?.parameters ?? []) {
            nextSchemaByPath.set(item.path, item);
          }
          for (const item of refreshedManagementServer.data?.parameters ?? []) {
            nextSchemaByPath.set(item.path, item);
          }
        } else if (isGnbSyncSourceGroup) {
          const [refreshedGnbSyncFap, refreshedGnbSyncDeviceInfo] = await Promise.all([
            refetchGnbSyncFapSchema(),
            refetchGnbSyncDeviceInfoSchema(),
          ]);
          for (const item of refreshedGnbSyncFap.data?.parameters ?? []) {
            nextSchemaByPath.set(item.path, item);
          }
          for (const item of refreshedGnbSyncDeviceInfo.data?.parameters ?? []) {
            nextSchemaByPath.set(item.path, item);
          }
        } else {
          const refreshed = await refetchCommonSchema();
          for (const item of refreshed.data?.parameters ?? []) {
            nextSchemaByPath.set(item.path, item);
          }
        }
        return nextSchemaByPath;
      };
      try {
        const expectedReadback = new Map(Object.entries(lastSubmit?.expectedReadback ?? {}));
        if (group.id === 'enb-mme' && expectedReadback.size > 0) {
          // SPV completed 只代表设备接受设置。后端自动 GPV 落库前 schema 仍可能是旧值，
          // 因此持续读取目标 path，只有实际观察到本次提交值才允许覆盖表单和清理草稿。
          await waitForExpectedParameterValues({
            expected: expectedReadback,
            read: async () => {
              refreshedSchemaByPath = await fetchRefreshedSchema();
              return new Map(
                Array.from(refreshedSchemaByPath.entries()).map(([path, item]) => [
                  path,
                  String(item.currentValue ?? ''),
                ]),
              );
            },
            intervalMs: 500,
            timeoutMs: 30_000,
            signal: abortController.signal,
          });
        } else {
          // 其它既有快速设置分组保持原有刷新节奏；Issue 158 的 MME 路径不再依赖该固定延时。
          await new Promise((resolve) => window.setTimeout(resolve, 500));
          if (cancelled) return;
          refreshedSchemaByPath = await fetchRefreshedSchema();
        }
      } catch (err) {
        if (err instanceof Error && err.name === 'AbortError') return;
        if (!cancelled) {
          const errMsg = err instanceof ParameterReadbackTimeoutError
            ? t('device.cell.readbackPendingTimeout')
            : err instanceof Error
            ? err.message
            : String(err);
          notification.error({
            message: t('device.multi.readbackFailed', { group: group.titleZh }),
            description: errMsg,
            duration: ERROR_FEEDBACK_DURATION_SECONDS,
          });
        }
        return;
      }
      if (cancelled) return;
      const currentDraftRevision = useQuickSettingsFeedbackStore.getState().draftRevisions[fbKey] ?? 0;
      if (!canApplySubmittedReadback({
        taskId: lastTask.id,
        syncedForTaskId: lastSubmit?.syncedForTaskId,
        submittedDraftRevision: lastSubmit?.submittedDraftRevision,
        currentDraftRevision,
      })) return;
      const nextValues: Record<string, unknown> = {};
      const refreshedMlnMmePoolValues = Array.from(refreshedSchemaByPath.values())
        .filter((item) => item.currentValue !== undefined && item.currentValue !== null)
        .map((item) => ({
          parameterPath: item.path,
          parameterValue: String(item.currentValue),
        }));
      for (const p of effectiveParams) {
        const special = resolveRuntimeSpecialConfig(p.name);
        const path = special?.configPath ?? resolveReadPath(p.standardPath || '');
        const refreshedValue = refreshedSchemaByPath.get(path)?.currentValue ?? '';
        const allowedEnumValues = p.enumOptions?.map((option) => option.value) ?? refreshedSchemaByPath.get(path)?.constraints?.enumValues ?? [];
        nextValues[p.name] = special?.kind === 'mme-ip-plmn-table'
          ? (isMlnIndexedMmePool
            ? parseMlnMmePoolRows(refreshedMlnMmePoolValues)
            : toMmeIpPlmnRows(refreshedValue))
          : special?.kind === 'plmn-list-table'
          ? toPlmnRows(refreshedValue)
          : isConstrainedGnssSyncSourceSelectPath(path)
            ? normalizeConstrainedGnssSyncSourceValue(refreshedValue, allowedEnumValues, path)
          : isEffectiveBitmaskPath(path)
            ? normalizeBitmaskValue(refreshedValue)
          : p.type === 'multiCheckbox'
            ? isBscCodecSupportParam(path)
              ? normalizeBscCodecSupportValue(refreshedValue)
              : parseQuickSettingsMultiCheckboxValue(refreshedValue)
          : isStringMultiSelectPath(path)
            ? normalizeStringMultiSelectValue(refreshedValue)
          : isSwitchPath(path)
            ? normalizeSwitchValue(refreshedValue)
          : normalizeEnumValue(
            refreshedValue,
            isDeviceTimeGroup && p.name === 'Enable' ? deviceTimeModeOptions : p.enumOptions,
          );
      }
      const currentValues = form.getFieldsValue(true) as Record<string, unknown>;
      const changedValues: Record<string, unknown> = {};
      for (const [name, value] of Object.entries(nextValues)) {
        if (currentValues[name] !== value) {
          changedValues[name] = value;
        }
      }
      if (Object.keys(changedValues).length > 0) {
        form.setFieldsValue(changedValues);
      }
      if (group.id === 'enb-plmn') {
        const plmnParam = effectiveParams.find(
          (param) => resolveRuntimeSpecialConfig(param.name)?.kind === 'plmn-list-table',
        );
        const plmnPath = plmnParam
          ? resolveRuntimeSpecialConfig(plmnParam.name)?.configPath
            ?? resolveReadPath(plmnParam.standardPath || '')
          : '';
        const plmnValue = plmnPath
          ? refreshedSchemaByPath.get(plmnPath)?.currentValue
          : undefined;
        if (plmnPath && plmnValue !== undefined) {
          applyDeviceParameterSearchReadback(queryClient, {
            deviceId,
            searchQuery: getServingPlmnSearchQuery(paramModel),
            replacePathPrefix: plmnPath,
            parameters: [{
              parameterPath: plmnPath,
              parameterValue: String(plmnValue),
            }],
          });
        } else {
          await refreshDeviceParameterSearchQueries(queryClient, deviceId);
        }
      }
      const syncSourceValue = nextValues.SyncSource;
      if (Array.isArray(syncSourceValue)) {
        gnssSyncSourceRef.current = syncSourceValue.map(String);
      }
      patchFeedback(fbKey, { syncedForTaskId: lastTask.id });
      clearDraft(fbKey);
      setFieldErrors({});
    })();
    return () => {
      cancelled = true;
      abortController.abort();
    };
  }, [active, lastTask?.id, lastTask?.status, lastSubmit?.at, lastSubmit?.expectedReadback, lastSubmit?.submittedDraftRevision, lastSubmit?.syncedForTaskId, draftRevision, refetchCommonSchema, refetchDeviceTimeSchema, refetchManagementServerSchema, refetchGnbSyncFapSchema, refetchGnbSyncDeviceInfoSchema, effectiveParams, instanceContext, form, clearDraft, patchFeedback, fbKey, group.id, group.titleZh, resolveRuntimeSpecialConfig, t, isDeviceTimeGroup, isGnbSyncSourceGroup, deviceTimeModeOptionsKey, queryClient, deviceId, paramModel, isEffectiveBitmaskPath, isConstrainedGnssSyncSourceSelectPath, isMlnIndexedMmePool]);

  // T-0146:基站应答失败时弹一次 notification(只在 status 第一次变成 failed 时触发,避免重复弹)
  // notifiedFailedTaskId 同样存 store —— 切顶层 tab 再切回不会重复弹。
  useEffect(() => {
    if (
      lastTask &&
      lastTask.status === 'failed' &&
      lastSubmit &&
      lastSubmit.notifiedFailedTaskId !== lastTask.id
    ) {
      notification.error({
        message: t('device.multi.nackFailed', { group: group.titleZh }),
        description: lastTask.errorMessage || t('device.multi.unknownErrorHint'),
        duration: ERROR_FEEDBACK_DURATION_SECONDS,
      });
      patchFeedback(fbKey, { notifiedFailedTaskId: lastTask.id });
    }
  }, [lastTask, lastSubmit, group.titleZh, patchFeedback, fbKey, t]);

  const title = locale === 'zh-CN' ? group.titleZh : group.titleEn;
  const visibleLastSubmit = useMemo(() => {
    if (!lastSubmit) return null;
    if (lastSubmit.submitStatus === 'failed_to_queue' && Date.now() - lastSubmit.at > 60_000) {
      return null;
    }
    return lastSubmit;
  }, [lastSubmit]);

  useEffect(() => {
    if (group.id !== 'device-ipsec-control') return;
    onIpsecControlChange?.(watchedIpsecEnable === undefined ? undefined : String(watchedIpsecEnable));
  }, [group.id, onIpsecControlChange, watchedIpsecEnable]);

  return (
    <Card
      title={title}
      size="small"
      extra={actionMode === 'staged' ? null : (
        <Space>
          {visibleLastSubmit && (() => {
            const spec = statusTagSpec(visibleLastSubmit, lastTask?.status, t);
            // 任务终态 failed 且设备给了原因 → 在 Tag 上附加简要原因(护舉61~80字截取),
            // 同时 Tooltip 挂完整 errorMessage,避免用户只看到"保存应答失败"看不到为什么。
            const isFailed = lastTask?.status === 'failed' && Boolean(lastTask?.errorMessage);
            const briefFault = isFailed ? formatDeviceFaultBrief(lastTask?.errorMessage) : '';
            const tag = (
              <Tag icon={spec.icon} color={spec.color}>
                {spec.label} · {t('device.cell.itemsCount', { count: visibleLastSubmit.count })}
                {briefFault ? ` · ${briefFault}` : ''} · {formatTime(visibleLastSubmit.at)}
              </Tag>
            );
            return isFailed ? (
              <Tooltip title={lastTask?.errorMessage} placement="bottomRight">
                {tag}
              </Tooltip>
            ) : tag;
          })()}
          <Button
            type="primary"
            onClick={handleSave}
            loading={updateMutation.isPending || isServingPlmnLoading}
            disabled={isServingPlmnLoading}
          >
            {updateMutation.isPending ? t('device.cell.dispatching') : t('common.save')}
          </Button>
        </Space>
      )}
      style={{ marginBottom: 16 }}
    >
      <Spin spinning={isSchemaLoading}>
      <Form
        form={form}
        layout="vertical"
        onValuesChange={(changedValues) => {
          // 同步到 store draft，跨顶层 TabBar 切走切回可恢复
          for (const [name, value] of Object.entries(changedValues)) {
            const p = visibleParams.find((q) => q.name === name);
            const special = p ? resolveRuntimeSpecialConfig(p.name) : undefined;
            const path = p ? (special?.configPath ?? resolveReadPath(p.standardPath || '')) : '';
            const item = path ? schemaByPath.get(path) : undefined;
            const allowedEnumValues = p?.enumOptions?.map((option) => option.value) ?? item?.constraints?.enumValues ?? [];
            if (p && isConstrainedGnssSyncSourceSelectPath(path)) {
              const normalized = normalizeConstrainedGnssSyncSourceChange(value, gnssSyncSourceRef.current, allowedEnumValues, path);
              changedValues[name] = normalized.value;
              form.setFieldValue(name, normalized.value);
              gnssSyncSourceRef.current = normalized.value;
              setDraftField(fbKey, name, serializeConstrainedGnssSyncSourceValue(normalized.value, path));
              if (normalized.errorKey) {
                message.warning(t(normalized.errorKey));
              }
              continue;
            }
            if (special?.kind === 'mme-ip-plmn-table') {
              setDraftField(fbKey, name, toMmeIpPlmnRows(value));
            } else if (special?.kind === 'plmn-list-table') {
              setDraftField(fbKey, name, toPlmnRows(value));
            } else if (p?.type === 'multiCheckbox') {
              const normalized = isBscCodecSupportParam(path)
                ? normalizeBscCodecSupportValue(value)
                : parseQuickSettingsMultiCheckboxValue(value);
              changedValues[name] = normalized;
              if (!formValueEquals(form.getFieldValue(name), normalized)) {
                form.setFieldValue(name, normalized);
              }
              setDraftField(fbKey, name, normalized.join('-'));
            } else {
              setDraftField(fbKey, name, String(value ?? ''));
            }
          }
          const dependentBandwidthByScs: Array<[string, string]> = [
            ['DLSubCarrierSpacing', 'DLCarrierBandWidth'],
            ['ULSubCarrierSpacing', 'ULCarrierBandWidth'],
          ];
          for (const [scsName, bandwidthName] of dependentBandwidthByScs) {
            if (!(scsName in changedValues)) continue;
            const options = NR_CARRIER_BANDWIDTH_OPTIONS_BY_SCS[String(changedValues[scsName] ?? '')] ?? [];
            if (options.length === 0) continue;
            const currentBandwidth = String(form.getFieldValue(bandwidthName) ?? '');
            if (options.some((option) => option.value === currentBandwidth)) continue;
            const nextBandwidth = options[0].value;
            form.setFieldValue(bandwidthName, nextBandwidth);
            setDraftField(fbKey, bandwidthName, nextBandwidth);
          }
          // T-0159: 交叉镜像 — 改 A 字段时把 A 的新值同步写入镜像字段 B（如 TDD 上下行带宽必须相等）。
          // antd Form.setFieldValue 不会触发 onValuesChange，故不会无限递归。
          for (const [name, value] of Object.entries(changedValues)) {
            const p = visibleParams.find((q) => q.name === name);
            if (!p) continue;
            const special = resolveRuntimeSpecialConfig(name);
            const path = special?.configPath ?? resolveReadPath(p.standardPath || '');
            const sItem = schemaByPath.get(path);
            const mirrorPath = sItem?.constraints?.mirrorWith;
            if (!mirrorPath) continue;
            const resolvedMirror = applyInstanceContext(mirrorPath, instanceContext);
            const mirrorName = paramNameByPath.get(resolvedMirror);
            if (!mirrorName || mirrorName === name) continue;
            const current = form.getFieldValue(mirrorName);
            if (String(current ?? '') === String(value ?? '')) continue;
            form.setFieldValue(mirrorName, value);
            setDraftField(fbKey, mirrorName, String(value ?? ''));
          }
          // 实时按 schema 取值范围校验，更新 fieldErrors 让 Form.Item 即时标红
          setFieldErrors((prev) => {
            const next = { ...prev };
            for (const [name, value] of Object.entries(changedValues)) {
              const p = visibleParams.find((q) => q.name === name);
              if (!p) continue;
              if (isHiddenTransmissionCountParam(name)) {
                delete next[name];
                continue;
              }
              const special = resolveRuntimeSpecialConfig(name);
              const path = special?.configPath ?? resolveReadPath(p.standardPath || '');
              const sItem = schemaByPath.get(path);
              const normalizedValue = special?.kind === 'mme-ip-plmn-table'
                ? serializeMmeIpPlmnList(toMmeIpPlmnRows(value))
                : special?.kind === 'plmn-list-table'
                ? serializePlmnList(toPlmnRows(value))
                : isEffectiveBitmaskPath(path)
                ? serializeBitmaskValue(value)
                : p.type === 'multiCheckbox'
                ? isBscCodecSupportParam(path)
                  ? serializeBscCodecSupportValue(value)
                  : serializeQuickSettingsMultiCheckboxValue(value)
                : isStringMultiSelectPath(path)
                ? serializeStringMultiSelectValue(value)
                : isSwitchPath(path)
                ? serializeSwitchValue(value)
                : String(value ?? '');
              const allowedValues = p.enumOptions?.map((option) => option.value) ?? sItem?.constraints?.enumValues ?? [];
              const err = special?.kind === 'plmn-list-table'
                ? null
                : isEffectiveBitmaskPath(path)
                ? isConstrainedGnssSyncSourceSelectPath(path)
                  ? validateConstrainedGnssSyncSourceValue(normalizedValue, allowedValues, path, t)
                  : validateBitmaskValue(
                  normalizedValue,
                  allowedValues,
                  sItem?.constraints?.minValue,
                  sItem?.constraints?.maxValue,
                )
                : isConstrainedGnssSyncSourceSelectPath(path)
                ? validateConstrainedGnssSyncSourceValue(normalizedValue, allowedValues, path, t)
                : isStringMultiSelectPath(path)
                ? null
                : p.type === 'multiCheckbox'
                ? validateValue(normalizedValue, (sItem?.type as never) ?? 'string', sItem?.constraints)
                : isSwitchPath(path)
                ? null
                : validateValue(
                  normalizedValue,
                  (sItem?.type as never) ?? 'string',
                  sItem?.constraints,
                );
              const mmeLimitErr = special?.kind === 'mme-ip-plmn-table'
                ? validateMmeIpPlmnLimit(toMmeIpPlmnRows(value), p.maxValue)
                : null;
              const mmeRowsErr = special?.kind === 'mme-ip-plmn-table'
                ? validateMmeIpPlmnRows(toMmeIpPlmnRows(value))
                : null;
              const mmeMembershipIssue = special?.kind === 'mme-ip-plmn-table'
                && restrictMmePlmnSelection
                ? findMmePlmnOutsideServingList(toMmeIpPlmnRows(value), servingPlmnOptions)
                : null;
              const mmeMembershipErr = mmeMembershipIssue
                ? t('device.cell.mmePlmnNotServing', {
                  row: mmeMembershipIssue.row,
                  plmn: mmeMembershipIssue.plmn,
                })
                : null;
              const plmnRowsErr = special?.kind === 'plmn-list-table'
                ? plmnValidationMessage(
                  validatePlmnList(toPlmnRows(value), p.maxValue ?? 6),
                  p.maxValue ?? 6,
                  t,
                )
                : null;
              // XML 驱动的 extraInfoPath 范围校验:在 schema 校验之后追加;
              // schema 已报错时优先展示 schema 错误,避免双错信息互盖。
              const extraBounds = extraInfoBoundsByName.get(name);
              const rangeErr = !err && !mmeRowsErr && !mmeLimitErr && !plmnRowsErr && extraBounds
                ? validateExtraInfoBounds(normalizedValue, extraBounds)
                : null;
              const finalErr = err ?? mmeRowsErr ?? mmeLimitErr ?? mmeMembershipErr ?? plmnRowsErr ?? rangeErr;
              if (finalErr) next[name] = finalErr;
              else delete next[name];
              // 镜像字段同时清/重新校验（值刚被程序性写入，旧 error 应失效）
              const mirrorPath = sItem?.constraints?.mirrorWith;
              if (mirrorPath) {
                const resolvedMirror = applyInstanceContext(mirrorPath, instanceContext);
                const mirrorName = paramNameByPath.get(resolvedMirror);
                if (mirrorName && mirrorName !== name) {
                  const mItem = schemaByPath.get(resolvedMirror);
                  const mErr = validateValue(
                    normalizedValue,
                    (mItem?.type as never) ?? 'string',
                    mItem?.constraints,
                  );
                  if (mErr) next[mirrorName] = mErr;
                  else delete next[mirrorName];
                }
              }
            }
            return next;
          });
        }}
      >
        {(() => {
          if (!isDeviceTimeGroup) return null;
          const modeParam = visibleParams.find((param) => param.name === 'Enable');
          const modePath = modeParam ? resolveReadPath(modeParam.standardPath || '') : '';
          const modeItem = modePath ? schemaByPath.get(modePath) : undefined;
          const modeRawItem = modePath ? getRawValueByPath(rawParameterByPath, modePath) : undefined;
          const modeWritable = modeItem?.writable ?? modeRawItem?.writable ?? true;
          const modeOptions = deviceTimeModeOptions;
          return (
            <Row gutter={16}>
              <Col span={8}>
                <Form.Item label="NTP" name="Enable">
                  <Select
                    disabled={!modeWritable}
                    options={modeOptions}
                    onChange={(value) => {
                      setDeviceTimeShowNtpServerFields(
                        shouldShowDeviceTimeNtpServerFields(instanceContext.networkType, String(value ?? '')),
                      );
                    }}
                  />
                </Form.Item>
              </Col>
              <Col span={8}>
                <Form.Item
                  label="Time Zone"
                  name="LocalTimeZoneName"
                  validateStatus={fieldErrors.LocalTimeZoneName ? 'error' : undefined}
                  help={fieldErrors.LocalTimeZoneName}
                >
                  <Select
                    options={appendCurrentOptionWithLabel(
                      displayedTimezoneOptions,
                      String(watchedLocalTimeZoneName ?? ''),
                      formatTimeZoneDisplay(String(watchedLocalTimeZoneName ?? '')),
                    )}
                    showSearch
                    optionFilterProp="label"
                    optionLabelProp="label"
                    labelRender={({ value }) => formatTimeZoneDisplay(String(value ?? ''))}
                    placeholder="Select time zone"
                  />
                </Form.Item>
              </Col>
            </Row>
          );
        })()}
        <Row gutter={16}>
          {visibleParams.map((p) => {
            if (isBmGsmCell && p.name === 'GsmCellWithRuRelation') {
              return null;
            }
            if (
              (isBmGsmCell && hasGsmTxAntNum && p.name === 'GsmTxAntNum') ||
              (isBmLteCell && hasLteAntennaPortsCount && p.name === 'AntennaPortsCount') ||
              (isBmLteCell && p.name === 'LteCellWithRuList')
            ) {
              return null;
            }
            if (isDeviceTimeGroup && (p.name === 'LocalTimeZoneName' || p.name === 'Enable')) {
              return null;
            }
            const special = resolveRuntimeSpecialConfig(p.name);
            const path = special?.configPath ?? resolveReadPath(p.standardPath || '');
            const item = schemaByPath.get(path);
            const rawItem = getRawValueByPath(rawParameterByPath, path)
              ?? (special?.kind === 'mme-ip-plmn-table'
                ? findRawValueBySuffix(mmeIpPlmnParams, '.MmeIpPlmnList')
                : special?.kind === 'bind-select' && p.name === 'NguBindInterface'
                  ? findRawValueBySuffix(nrNguParams, '.BindInterface') ?? findRawValueBySuffix(nrNguFallbackParams, '.NguLocalIpAddrList')
                  : undefined);
            const displayPath = special?.displayPath;
            const displayValue = displayPath
              ? (preferSchemaCurrentValue
                ? (schemaByPath.get(displayPath)?.currentValue
                  ?? getRawValueByPath(rawParameterByPath, displayPath)?.parameterValue
                  ?? (special?.kind === 'bind-select' && p.name === 'NguBindInterface'
                    ? (findRawValueBySuffix(nrNguParams, '.NguLocalIp') ?? findRawValueBySuffix(nrNguFallbackParams, '.NguLocalIpAddrList'))?.parameterValue
                    : undefined)
                  ?? '')
                : (getRawValueByPath(rawParameterByPath, displayPath)?.parameterValue
                  ?? (special?.kind === 'bind-select' && p.name === 'NguBindInterface'
                    ? (findRawValueBySuffix(nrNguParams, '.NguLocalIp') ?? findRawValueBySuffix(nrNguFallbackParams, '.NguLocalIpAddrList'))?.parameterValue
                    : undefined)
                  ?? schemaByPath.get(displayPath)?.currentValue
                  ?? ''))
              : '';
            // 紧贴 ARFCN 之后插入派生的 Frequency(MHz) 显示行。
            // 命中分组:BSC 的 GSM 空口分组(gsm-cell) 与 BTS 的基站信息分组(bts-cell-info)。
            const renderFrequencyAfter =
              (group.id === 'gsm-cell' || group.id === 'bts-cell-info') &&
              p.name === 'CurrentArfcn';
            // BM 小区专属:在发射功率后插入"绑定 RU 的 Route Index"。
            // GSM 直接编辑 GsmCellWithRuRelation 原值;LTE 仍编辑 LteCellWithRuList。
            const renderGsmRuRelationAfter = isBmGsmCell && p.name === 'GsmBtsRFPower';
            const renderRuRouteAfter = isBmLteCell && p.name === 'PowerClass';
            // BM LTE 派生显示:Frequency / Cell ID / 2T4R 开关。
            const isLteCell = isLteCellGroup;
            const renderLteFreqAfter = isLteCell && p.name === 'DLEarfcn';
            const renderCellIdAfter = isLteCell && p.name === 'ECI';
            const isTransmissionPowerField =
              (isBmGsmCell && hasGsmTxAntNum && p.name === 'GsmBtsRFPower') ||
              (isBmLteCell && hasLteAntennaPortsCount && p.name === 'PowerClass');
            const transmissionCountField = isBmGsmCell ? 'GsmTxAntNum' : 'AntennaPortsCount';
            const isDeviceTimeParam = group.id === 'device-time';
            const isIpsecControlParam = group.id === 'device-ipsec-control';
            const writable = (isDeviceTimeParam || isIpsecControlParam)
              ? (special?.forceWritable
                ?? item?.writable
                ?? rawItem?.writable
                ?? true)
              : (special?.forceWritable
                ?? rawItem?.writable
                ?? item?.writable
                ?? false);
            const finalWritable = p.readonly ? false : writable;
            const error = fieldErrors[p.name];
            const isSwitchField = isSwitchPath(path);
            // XML hideRangeHint="true" 或 Switch 字段时不在 label 后展示 schema 推导的 [min ~ max]
            // (字典范围与业务允许值不一致的字段如 Band:字典 1..maxInt,业务允许集只有少数频段)。
            const constraintHint = p.hideRangeHint || isSwitchField ? '' : formatConstraintHint(item, t);
            const currentBindPath = String(form.getFieldValue(p.name) ?? rawItem?.parameterValue ?? item?.currentValue ?? '');
            const resolvedDisplayValue = displayValue || (special?.bindValueMode === 'ip' ? currentBindPath : bindIpByPath.get(currentBindPath)) || '';
            // XML 驱动:若 param 在 quicksettings XML 上声明了 extraInfoPath,
            // 把对应路径的当前值按 [lo ~ hi] 格式与 label 同一行显示(灰色小字)。
            const extraInfoRaw = p.extraInfoPath ? extraInfoValueByPath.get(p.extraInfoPath) ?? '' : '';
            const extraInfoFormatted = extraInfoRaw ? formatExtraInfoRange(extraInfoRaw) : '';
            const label = (
              <Space size={4}>
                <span style={special?.kind === 'mme-ip-plmn-table' || special?.kind === 'plmn-list-table' ? { whiteSpace: 'nowrap' } : undefined}>
                  {locale === 'zh-CN' ? p.titleZh : p.titleEn}
                </span>
                {(!writable || p.readonly) && <Text type="secondary" style={{ fontSize: 12 }}>{t('device.cell.readonly')}</Text>}
                {constraintHint && (
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {constraintHint}
                  </Text>
                )}
                {extraInfoFormatted && (
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {extraInfoFormatted}
                  </Text>
                )}
                {p.unit && (
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {`(${p.unit})`}
                  </Text>
                )}
                {special?.kind === 'bind-select' && resolvedDisplayValue && (
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {t('device.cell.currentIp', { ip: resolvedDisplayValue })}
                  </Text>
                )}
              </Space>
            );
            const enumMeta = getEffectiveEnumMeta(item?.constraints, path);
            // XML 驱动枚举:quicksettings <param> 上的 <option value=".." label=".."/> 优先于 schema
            // constraints。用于不宜修改 param-mapping 只想在 UI 层展示友好选项的场景
            // (如 RFEnable: 1→ON / 0→OFF)。
            const xmlEnumValues = p.enumOptions?.map((o) => o.value) ?? [];
            const xmlEnumLabels = p.enumOptions?.map((o) => o.label) ?? [];
            const nrCarrierBandwidthOptions = getNrCarrierBandwidthOptions(p.name, dlSubCarrierSpacing, ulSubCarrierSpacing);
            const effectiveEnumValues = nrCarrierBandwidthOptions.length > 0
              ? nrCarrierBandwidthOptions.map((o) => o.value)
              : xmlEnumValues.length > 0
              ? xmlEnumValues
              : (enumMeta?.values ?? []);
            const effectiveEnumLabels = nrCarrierBandwidthOptions.length > 0
              ? nrCarrierBandwidthOptions.map((o) => o.label)
              : xmlEnumValues.length > 0
              ? xmlEnumLabels
              : (enumMeta?.labels ?? []);
            const isEnum = !special && effectiveEnumValues.length > 0;
            const isBitmaskEnum = isEnum && isEffectiveBitmaskPath(path);
            const isConstrainedGnssSyncSourceEnum = isEnum && isConstrainedGnssSyncSourceSelectPath(path);
            const isStringMultiSelectEnum = isEnum && isStringMultiSelectPath(path);
            const isMultiCheckbox = !special && p.type === 'multiCheckbox';
            const isCodecSupport = isMultiCheckbox && isBscCodecSupportParam(path);
            const baseEnumOptions = isBitmaskEnum
              ? isConstrainedGnssSyncSourceEnum
                ? effectiveEnumValues.map((v, idx) => ({
                  value: v,
                  label: localizeEnumLabel(effectiveEnumLabels[idx] || v, v, locale),
                }))
                : buildBitmaskCombinationOptions(effectiveEnumValues, effectiveEnumLabels, locale)
              : effectiveEnumValues.map((v, idx) => ({
                value: v,
                label: localizeEnumLabel(effectiveEnumLabels[idx] || v, v, locale),
              }));
            const currentRawValue = preferSchemaCurrentValue
              ? (item?.currentValue ?? rawItem?.parameterValue ?? '')
              : (rawItem?.parameterValue ?? item?.currentValue ?? '');
            const enumOptions = isEnum && !isBitmaskEnum && !isStringMultiSelectEnum
              ? appendCurrentEnumOption(baseEnumOptions, currentRawValue, p.enumOptions)
              : baseEnumOptions;
            const extra = special?.kind === 'mme-ip-plmn-table'
              ? t(restrictMmePlmnSelection
                ? 'device.cell.mmeIpPlmnServingExtra'
                : 'device.cell.mmeIpPlmnExtra')
              : special?.kind === 'plmn-list-table'
              ? t('device.cell.plmnListExtra')
              : undefined;
            const effectiveBindOptions = special?.kind === 'bind-select'
              ? appendCurrentBindOption(
                  special.bindValueMode === 'ip' ? bindIpValueOptions : bindSelectOptions,
                  currentBindPath,
                  resolvedDisplayValue,
                )
              : [];
            const input = (
              <Col span={special?.kind === 'mme-ip-plmn-table' || special?.kind === 'plmn-list-table' ? 24 : 8} key={p.name}>
                <Form.Item
                  label={special?.kind === 'mme-ip-plmn-table' || special?.kind === 'plmn-list-table' ? undefined : label}
                  name={isTransmissionPowerField ? undefined : p.name}
                  valuePropName={!isTransmissionPowerField && isSwitchField ? 'checked' : undefined}
                  normalize={isCodecSupport ? normalizeBscCodecSupportValue : undefined}
                  validateStatus={error ? 'error' : undefined}
                  help={error}
                  extra={extra}
                >
                  {isTransmissionPowerField ? (
                    <TransmissionPowerInput
                      countField={transmissionCountField}
                      powerField={p.name}
                      disabled={!finalWritable}
                      placeholder={special?.placeholder || item?.defaultValue || (!finalWritable ? '未上报' : '')}
                    />
                  ) : special?.kind === 'bind-select' ? (
                    <Select
                      disabled={!finalWritable}
                      showSearch
                      optionFilterProp="label"
                      placeholder={t('device.cell.bindSelectPlaceholder')}
                      options={effectiveBindOptions}
                    />
                  ) : special?.kind === 'mme-ip-plmn-table' ? (
                    <MmeIpPlmnTable
                      disabled={!finalWritable}
                      locale={locale}
                      maxRows={p.maxValue}
                      plmnOptions={servingPlmnOptions}
                      restrictPlmnSelection={restrictMmePlmnSelection}
                    />
                  ) : special?.kind === 'plmn-list-table' ? (
                    <PlmnListTable
                      disabled={!finalWritable}
                      maxRows={p.maxValue ?? 6}
                    />
                  ) : isSwitchField ? (
                    <Switch
                      disabled={!finalWritable}
                      checkedChildren={t('common.on')}
                      unCheckedChildren={t('common.off')}
                    />
                  ) : isMultiCheckbox ? (
                    <Checkbox.Group
                      disabled={!finalWritable}
                      options={(p.checkboxOptions ?? []).map((v) => ({
                        value: v,
                        label: v,
                        disabled: isCodecSupport && v === 'fr',
                      }))}
                    />
                  ) : isEnum ? (
                    <Select
                      mode={isStringMultiSelectEnum || isConstrainedGnssSyncSourceEnum ? 'multiple' : undefined}
                      disabled={!finalWritable}
                      placeholder={item?.defaultValue || ''}
                      showSearch={isBitmaskEnum}
                      optionFilterProp="label"
                      options={enumOptions}
                    />
                  ) : (
                    <Input disabled={!finalWritable} placeholder={special?.placeholder || item?.defaultValue || (!finalWritable ? '未上报' : '')} />
                  )}
                </Form.Item>
              </Col>
            );
            return renderFrequencyAfter ? (
              <Fragment key={p.name}>
                {input}
                <FrequencyDisplay form={form} locale={locale} />
              </Fragment>
            ) : renderRuRouteAfter ? (
              <Fragment key={p.name}>
                {input}
                <RouteIndexInput
                  form={form}
                  ruRouteItemByIdx={ruRouteItemByIdx}
                  locale={locale}
                  fieldName="LteCellWithRuList"
                  disabled={false}
                  error={fieldErrors.LteCellWithRuList}
                />
              </Fragment>
            ) : renderGsmRuRelationAfter ? (
              <Fragment key={p.name}>
                {input}
                <GsmRuRelationInput
                  error={fieldErrors.GsmCellWithRuRelation}
                  ruRouteItemByIdx={ruRouteItemByIdx}
                />
              </Fragment>
            ) : renderLteFreqAfter ? (
              <Fragment key={p.name}>
                {input}
                <LteFrequencyDisplay form={form} locale={locale} />
              </Fragment>
            ) : renderCellIdAfter ? (
              <Fragment key={p.name}>
                {input}
                <CellIdDerivedDisplay form={form} locale={locale} />
              </Fragment>
            ) : input;
          })}
        </Row>
      </Form>
      </Spin>
    </Card>
  );
}

// formatExtraInfoRange 把 quicksettings XML extraInfoPath 指向的参数当前值
// 转成 [lo ~ hi] 形式 (与 OffsetToPointA 等数值范围提示视觉一致)。
// 设备上送的 SupportedPowerRange 实测形如 "24,30",也兼容 "24~30" / "24-30" 与空白分隔;
// 形如单值或非两段时,直接 [raw] 包裹回退,保持小字提示语义。
function formatExtraInfoRange(raw: string): string {
  const s = raw.trim();
  if (!s) return '';
  const parts = s.split(/[,~\-\s]+/).filter(Boolean);
  if (parts.length === 2) return `[${parts[0]} ~ ${parts[1]}]`;
  return `[${s}]`;
}

// parseExtraInfoBounds 由设备上送的 range 字符串解析数值上下界,用于输入校验。
// 仅在两端均为合法数字时返回 [min, max] (自动按数值排序),否则 null 跳过校验。
function parseExtraInfoBounds(raw: string): [number, number] | null {
  const s = raw.trim();
  if (!s) return null;
  const parts = s.split(/[,~\-\s]+/).filter(Boolean);
  if (parts.length !== 2) return null;
  const a = Number(parts[0]);
  const b = Number(parts[1]);
  if (!Number.isFinite(a) || !Number.isFinite(b)) return null;
  return a <= b ? [a, b] : [b, a];
}

// validateExtraInfoBounds 把字段输入值与 extraInfoPath 解析出的范围比对。
// 输入非数值时返回 null (跳过——上游 validateValue 已处理类型错误);
// 超出区间则返回中文错误信息 (与 validators.ts 风格保持一致)。
function validateExtraInfoBounds(value: string, bounds: [number, number]): string | null {
  const s = String(value ?? '').trim();
  if (!s) return null;
  const num = Number(s);
  if (!Number.isFinite(num)) return null;
  const [min, max] = bounds;
  if (num < min || num > max) return `取值范围 [${min} ~ ${max}]`;
  return null;
}

// normalizeEnumValue 把设备上送的字符串(可能是 "true"/"false"、"True"/"FALSE"、"1"/"0")
// 归一到 XML <option value="..."/> 的真实值,确保 Select 能选中正确项。
// 仅在 enumOptions 非空时生效;不存在等价匹配时原样返回(保留原始值,Select 留空)。
function normalizeEnumValue(raw: unknown, enumOptions?: { value: string; label: string }[]): string {
  const s = String(raw ?? '');
  if (!enumOptions || enumOptions.length === 0) return s;
  if (enumOptions.some((o) => o.value === s)) return s;
  const ci = s.toLowerCase();
  const direct = enumOptions.find((o) => o.value.toLowerCase() === ci);
  if (direct) return direct.value;
  const labelMatch = enumOptions.find((o) => o.label.toLowerCase() === ci);
  if (labelMatch) return labelMatch.value;
  // BOOLEAN 等价:true/1 与 false/0 互转,适配 TR-069 BOOLEAN 字段两种序列化。
  if ((ci === 'true' || ci === '1') && enumOptions.some((o) => o.value === '1')) return '1';
  if ((ci === 'false' || ci === '0') && enumOptions.some((o) => o.value === '0')) return '0';
  if ((ci === 'true' || ci === '1')) {
    const m = enumOptions.find((o) => o.value.toLowerCase() === 'true');
    if (m) return m.value;
  }
  if ((ci === 'false' || ci === '0')) {
    const m = enumOptions.find((o) => o.value.toLowerCase() === 'false');
    if (m) return m.value;
  }
  return s;
}

// formatConstraintHint 把 schema 取值范围渲染成 label 后的灰色提示。
// 后端 MinValue/MaxValue 是按类型复用的字段：string → 长度边界；int/unsignedInt → 值范围。
// 枚举字段不渲染提示（Select 组件已展示候选项，避免重复占位）。
function formatConstraintHint(schema: ParameterSchemaItem | undefined, t: TFn): string {
  if (!schema?.constraints) return '';
  const c = schema.constraints;
  if (c.enumValues && c.enumValues.length > 0) {
    return '';
  }
  const isString = schema.type === 'string';
  // Prefer explicit maxLength/minLength; fall back to minValue/maxValue (interpreted by type)
  const min = c.minLength ?? (isString ? c.minValue : c.minValue);
  const max = c.maxLength ?? (isString ? c.maxValue : c.maxValue);
  if (min !== undefined || max !== undefined) {
    // When the dictionary has no min, show 0 (avoid showing the user a meaningless -∞ lower bound).
    const lo = min ?? 0;
    const hi = max ?? '∞';
    return isString ? t('device.multi.hintLenRange', { lo, hi }) : `[${lo} ~ ${hi}]`;
  }
  return '';
}

function commonPathPrefix(paths: string[]): string {
  if (paths.length === 0) return '';
  let prefix = paths[0];
  for (const p of paths.slice(1)) {
    let i = 0;
    while (i < prefix.length && i < p.length && prefix[i] === p[i]) i++;
    prefix = prefix.slice(0, i);
  }
  // 截到最后一个 '.'，使前缀对齐对象级
  const idx = prefix.lastIndexOf('.');
  return idx > 0 ? prefix.slice(0, idx + 1) : prefix;
}
