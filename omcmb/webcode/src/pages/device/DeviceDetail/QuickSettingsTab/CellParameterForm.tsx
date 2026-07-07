import { useCallback, useEffect, useMemo, useRef, useState, Fragment } from 'react';
import { Button, Card, Col, Form, Input, Row, Select, Space, Spin, Table, Tag, Tooltip, Typography, message, notification } from 'antd';
import type { FormInstance } from 'antd';
import { CheckCircleOutlined, ClockCircleOutlined, CloseCircleOutlined, DeleteOutlined, PlusOutlined, SendOutlined, SyncOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useParameterSchema, useSearchParameters, useUpdateParameters } from '@core/hooks/api/useDeviceParameters';
import { useDeviceTaskStatus } from '@core/hooks/api/useDeviceTask';
import { notificationKeys } from '@core/hooks/api/useNotificationCenter';
import { useRenameDevice } from '@core/hooks/api/useDevices';
import { useDeviceNameSyncMode } from '@core/hooks/api/useDeviceNameSyncMode';
import { getTimezoneAliasOptions, mapTimezoneAliasToDisplay } from '@core/utils/timezoneAliasConfig';
import {
  feedbackKey,
  useQuickSettingsFeedbackStore,
  type CellFeedback,
} from '@core/store/quickSettingsFeedbackStore';
import type { DeviceParameter, ParameterSchemaItem, ParameterUpdateRequest } from '@core/types/deviceParameter';
import type { DeviceTaskStatus } from '@core/types/deviceTask';
import type { QuickSettingsGroup, QuickSettingsParam } from '@core/types/quicksettings';
import { applyInstanceContext, getEffectiveEnumMeta, getFeedbackScopeContext, validateValue, type QuickSettingsInstanceContext } from './validators';
import { formatDeviceFaultBrief } from './MultiInstanceTable';
import { useT } from '@/hooks/useT';

const { Text } = Typography;
const ERROR_FEEDBACK_DURATION_SECONDS = 2;

// FAPService.1 HNBName 标准路径：命中此 path 的字段走 rename 接口（不走普通 SPV 下发）
const HNB_NAME_PATH = 'Device.Services.FAPService.1.AccessMgmt.LTE.HNBName';

type TFn = (id: string, values?: Record<string, string | number>) => string;

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
    <Col span={12}>
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

// BoundRuRouteIndexDisplay: BM GSM 专属。监听表单 GsmCellWithRuRelation,
// 用其值作为 RU 实例 idx,从外部传入的 ruRouteByIdx 中取出 RouteIndex 显示。
function BoundRuRouteIndexDisplay({
  form,
  ruRouteByIdx,
  locale,
}: {
  form: FormInstance;
  ruRouteByIdx: Map<string, string>;
  locale: 'zh-CN' | 'en-US';
}) {
  void locale;
  const t = useT();
  const ruRel = Form.useWatch('GsmCellWithRuRelation', form);
  const ruIdx = ruRel == null ? '' : String(ruRel).trim();
  const routeIndex = ruIdx && ruRouteByIdx.has(ruIdx) ? ruRouteByIdx.get(ruIdx)! : '-';
  const labelText = t('device.cell.routeIndexBoundRu');
  const display = ruIdx ? `RU ${ruIdx} → ${routeIndex}` : '-';
  return (
    <Col span={12}>
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
    <Col span={12}>
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
    <Col span={12}>
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

// AntennaPortsAs2T4RDisplay: 由 AntennaPortsCount 派生 2T4R 开关(2 -> OFF, 4 -> ON)。
function AntennaPortsAs2T4RDisplay({ form, locale }: { form: FormInstance; locale: 'zh-CN' | 'en-US' }) {
  void locale;
  const t = useT();
  const ports = Form.useWatch('AntennaPortsCount', form);
  const n = Number(ports);
  const labelText = t('device.cell.switch2T4R');
  const v = n === 4 ? 'ON' : n === 2 ? 'OFF' : '-';
  return (
    <Col span={12}>
      <Form.Item
        label={
          <Space size={4}>
            <span>{labelText}</span>
            <Text type="secondary" style={{ fontSize: 12 }}>{t('device.cell.readonly')}</Text>
          </Space>
        }
      >
        <Input value={v} disabled />
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

function formatTimeZoneDisplay(value: string): string {
  const normalized = String(value ?? '').trim();
  if (!normalized) return '';
  return mapTimezoneAliasToDisplay(normalized);
}

function inferDeviceTimeMode(
  rawParameterByPath: Map<string, DeviceParameter>,
  schemaByPath: Map<string, ParameterSchemaItem>,
): string {
  const current = rawParameterByPath.get('Device.Time.Enable')?.parameterValue
    ?? schemaByPath.get('Device.Time.Enable')?.currentValue;
  const normalized = String(current ?? '').trim().toLowerCase();
  if (normalized === '1' || normalized === 'true') return '1';
  if (normalized === '0' || normalized === 'false') return '0';

  const hasNtpServers = ['Device.Time.NTPServer1', 'Device.Time.NTPServer2', 'Device.Time.NTPServer3', 'Device.Time.NTPServer4', 'Device.Time.NTPServer5']
    .some((path) => String(
      rawParameterByPath.get(path)?.parameterValue
        ?? schemaByPath.get(path)?.currentValue
        ?? '',
    ).trim() !== '');
  return hasNtpServers ? '0' : '1';
}

interface CellParameterFormProps {
  deviceId: string;
  active?: boolean;
  group: QuickSettingsGroup;
  instanceContext: QuickSettingsInstanceContext;
  locale: 'zh-CN' | 'en-US';
  onIpsecControlChange?: (value: string | undefined) => void;
}

interface BindSelectOption {
  value: string;
  label: string;
}

interface SpecialFieldConfig {
  kind: 'input' | 'mme-ip-plmn-table' | 'bind-select';
  configPath: string;
  displayPath?: string;
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

function MmeIpPlmnTable({ value = [], onChange, disabled = false, locale }: MmeIpPlmnTableProps) {
  void locale;
  const t = useT();
  const rows = isMmeIpPlmnRows(value) ? value : [];

  const setRows = (nextRows: MmeIpPlmnRow[]) => {
    onChange?.(nextRows);
  };

  const updateCell = (key: string, field: 'mmeIp' | 'plmn', nextValue: string) => {
    setRows(rows.map((row) => (row.key === key ? { ...row, [field]: nextValue } : row)));
  };

  const addRow = () => {
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
        <Input
          value={row.mmeIp}
          disabled={disabled}
          placeholder="127.0.0.1"
          onChange={(e) => updateCell(row.key, 'mmeIp', e.target.value)}
        />
      ),
    },
    {
      title: locale === 'zh-CN' ? 'PLMN' : 'PLMN',
      dataIndex: 'plmn',
      key: 'plmn',
      render: (_: unknown, row: MmeIpPlmnRow) => (
        <Input
          value={row.plmn}
          disabled={disabled}
          placeholder="46000"
          onChange={(e) => updateCell(row.key, 'plmn', e.target.value)}
        />
      ),
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
      <Button
        icon={<PlusOutlined />}
        onClick={addRow}
        disabled={disabled}
      >
        {t('device.cell.addRow')}
      </Button>
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
      forceWritable: true,
    };
  }
  return null;
}

function normalizeInterfaceType(value?: string | null): string {
  return String(value ?? '').trim().toLowerCase();
}

function buildBindSelectOptions(parameters: ParameterSchemaItem[]): BindSelectOption[] {
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
          value: item.path,
          label: `${currentValue}`,
        });
      }
      continue;
    }

    const vlanMatch = /^Device\.Ethernet\.Interface\.(\d+)\.VlanInterface\.\d+\.(IPv[46]Address\.\d+\.IPAddress)$/.exec(item.path);
    if (vlanMatch && interfaceKinds.get(vlanMatch[1]) === 'wan') {
      options.push({
        value: item.path,
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
 *  3. 渲染为 2 列网格 Form
 *  4. 顶部"保存"按钮收集本表单全部脏字段，一次性 SetParameterValues
 *  5. Save 部分失败时按字段标红保留输入值（继承 Antd Form 校验/状态行为）
 */
export default function CellParameterForm({ deviceId, active = true, group, instanceContext, locale, onIpsecControlChange }: CellParameterFormProps) {
  const t = useT();
  const [form] = Form.useForm();
  const latestLocalEditAtRef = useRef(0);
  const watchedLocalTimeZoneName = Form.useWatch('LocalTimeZoneName', form);
  const watchedIpsecEnable = Form.useWatch('IPSEC_ENABLE', form);
  const updateMutation = useUpdateParameters();
  const renameMutation = useRenameDevice(deviceId);
  const nameSyncMode = useDeviceNameSyncMode();
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
  const clearFeedback = useQuickSettingsFeedbackStore((s) => s.clearFeedback);
  const draft = useQuickSettingsFeedbackStore((s) => s.drafts[fbKey]);
  const setDraftField = useQuickSettingsFeedbackStore((s) => s.setDraftField);
  const clearDraft = useQuickSettingsFeedbackStore((s) => s.clearDraft);
  const isDeviceTimeGroup = group.id === 'device-time';
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
    () => (isDeviceTimeGroup ? '' : commonPathPrefix(effectiveParams.map((p) => resolveReadPath(p.standardPath || '')))),
    [isDeviceTimeGroup, effectiveParams, resolveReadPath],
  );
  const { data: schemaResp, isLoading: isCommonSchemaLoading, refetch: refetchCommonSchema } = useParameterSchema(
    deviceId,
    commonPrefix,
    active && !isDeviceTimeGroup,
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
    group.id === 'enb-mme' ? 'MmeIpPlmnList' : '',
    50,
    active && group.id === 'enb-mme',
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
  const { data: deviceTimeParams } = useSearchParameters(
    deviceId,
    group.id === 'device-time' ? 'Device.Time.' : '',
    200,
    active && group.id === 'device-time',
  );
  const { data: ipsecControlParams } = useSearchParameters(
    deviceId,
    group.id === 'device-ipsec-control' ? (effectiveParams[0]?.standardPath || 'IPSEC_ENABLE') : '',
    20,
    active && group.id === 'device-ipsec-control',
  );
  const visibleParams = useMemo(() => {
    if (!isDeviceTimeGroup || deviceTimeParams === undefined) {
      return effectiveParams;
    }
    const availableTimePaths = new Set(deviceTimeParams.map((item) => item.parameterPath));
    return effectiveParams.filter((param) => {
      const path = param.standardPath || '';
      if (!isNtpServerPath(path)) {
        return true;
      }
      return availableTimePaths.has(path);
    });
  }, [deviceTimeParams, effectiveParams, isDeviceTimeGroup]);
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

  // BM GSM 专属:并行拉 RU 节点 schema,用于在 gsm-cell 表单中展示"绑定 RU 的 Route Index"。
  // 拉取与主 schema 解耦,避免污染 commonPrefix 退化成 Device. 触发全量拉取。
  const isGsmCell = group.id === 'gsm-cell';
  const { data: ruSchemaResp } = useParameterSchema(
    deviceId,
    'Device.DeviceInfo.RU.',
    active && isGsmCell,
  );
  const effectiveSchemaParameters = useMemo(
    () => (isDeviceTimeGroup
      ? [
        ...(deviceTimeSchemaResp?.parameters ?? []),
        ...(managementServerSchemaResp?.parameters ?? []),
      ]
      : (schemaResp?.parameters ?? [])),
    [isDeviceTimeGroup, deviceTimeSchemaResp, managementServerSchemaResp, schemaResp],
  );
  const isSchemaLoading = isDeviceTimeGroup
    ? (isDeviceTimeSchemaLoading || isManagementServerSchemaLoading)
    : isCommonSchemaLoading;
  const hasSchemaData = isDeviceTimeGroup
    ? Boolean(deviceTimeSchemaResp && managementServerSchemaResp)
    : Boolean(schemaResp);
  const ruRouteByIdx = useMemo(() => {
    const map = new Map<string, string>();
    ruSchemaResp?.parameters.forEach((p) => {
      // 形如 Device.DeviceInfo.RU.<n>.RouteIndex
      const m = /^Device\.DeviceInfo\.RU\.(\d+)\.RouteIndex$/.exec(p.path);
      if (m) {
        const v = p.currentValue ?? '';
        map.set(m[1], typeof v === 'string' ? v : String(v));
      }
    });
    return map;
  }, [ruSchemaResp]);

  const schemaByPath = useMemo(() => {
    const map = new Map<string, ParameterSchemaItem>();
    effectiveSchemaParameters.forEach((p) => map.set(p.path, p));
    return map;
  }, [effectiveSchemaParameters]);
  const rawParameterByPath = useMemo(() => {
    const map = new Map<string, DeviceParameter>();
    for (const item of [...(mmeIpPlmnParams ?? []), ...(nrCommonParams ?? []), ...(nrNguParams ?? []), ...(deviceTimeParams ?? []), ...(ipsecControlParams ?? [])]) {
      map.set(item.parameterPath, item);
    }
    return map;
  }, [mmeIpPlmnParams, nrCommonParams, nrNguParams, deviceTimeParams, ipsecControlParams]);
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
  const deviceTimeModeOptions = useMemo(() => {
    const meta = getEffectiveEnumMeta(schemaByPath.get('Device.Time.Enable')?.constraints, 'Device.Time.Enable');
    if (meta && meta.values.length > 0) {
      return meta.values.map((value, index) => ({
        value,
        label: meta.labels[index] || value,
      }));
    }
    return [
      { value: '1', label: 'NTP Server' },
      { value: '0', label: 'NTP Client' },
    ];
  }, [schemaByPath]);
  const deviceTimeModeOptionsKey = deviceTimeModeOptions.map((option) => option.value).join('\u0000');
  const bindSelectOptions = useMemo(
    () => buildBindSelectOptions(ethernetSchemaResp?.parameters ?? []),
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
      if (draft && draft[p.name] !== undefined) {
        const nextValue = specialConfigByName.get(p.name)?.kind === 'mme-ip-plmn-table'
          ? toMmeIpPlmnRows(draft[p.name])
          : String(draft[p.name] ?? '');
        if (currentValue !== nextValue) {
          form.setFieldValue(p.name, nextValue);
        }
        return;
      }
      // 优先级 2: 用户在当前会话已 touched
      if (form.isFieldTouched(p.name)) return;
      // 优先级 3: schema 原值
      const special = specialConfigByName.get(p.name);
      const path = special?.configPath ?? resolveReadPath(p.standardPath || '');
      const item = schemaByPath.get(path);
      const rawItem = getRawValueByPath(rawParameterByPath, path)
        ?? (special?.kind === 'mme-ip-plmn-table'
          ? findRawValueBySuffix(mmeIpPlmnParams, '.MmeIpPlmnList')
          : special?.kind === 'bind-select' && p.name === 'NguBindInterface'
            ? findRawValueBySuffix(nrNguParams, '.BindInterface')
            : undefined);
      if (special?.kind === 'mme-ip-plmn-table') {
        form.setFieldValue(p.name, toMmeIpPlmnRows(rawItem?.parameterValue ?? item?.currentValue ?? ''));
      } else {
        let raw = rawItem?.parameterValue ?? item?.currentValue ?? '';
        if (isDeviceTimeGroup && p.name === 'Enable' && (raw === '' || raw == null)) {
          raw = inferDeviceTimeMode(rawParameterByPath, schemaByPath);
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
        form.setFieldValue(
          p.name,
          normalizeEnumValue(
            raw,
            isDeviceTimeGroup && p.name === 'Enable' ? deviceTimeModeOptions : p.enumOptions,
          ),
        );
      }
    });
  }, [hasSchemaData, visibleParams, instanceContext, form, schemaByPath, rawParameterByPath, draft, specialConfigByName, mmeIpPlmnParams, nrNguParams, isDeviceTimeGroup, deviceTimeModeOptionsKey]);

  const handleSave = async () => {
    const values = form.getFieldsValue() as Record<string, unknown>;
    const updates: ParameterUpdateRequest[] = [];
    const errors: Record<string, string> = {};

    for (const p of visibleParams) {
      // readonly leaf(如 BTS ID / 共享只读状态量)不参与下发:它们的 path 在 param-mappings
      // 里多为 not_found / access=READ_ONLY,带进 SetParameterValues 会被后端 MappingValidator
      // 整批拒成 400,导致用户改任何字段都"入队失败"。
      if (p.readonly) continue;

      const special = specialConfigByName.get(p.name);
      const path = special?.configPath ?? resolveReadPath(p.standardPath || '');
      const item = schemaByPath.get(path);
      const rawItem = getRawValueByPath(rawParameterByPath, path)
        ?? (special?.kind === 'mme-ip-plmn-table'
          ? findRawValueBySuffix(mmeIpPlmnParams, '.MmeIpPlmnList')
          : special?.kind === 'bind-select' && p.name === 'NguBindInterface'
            ? findRawValueBySuffix(nrNguParams, '.BindInterface')
            : undefined);

      if (special?.kind === 'mme-ip-plmn-table') {
        values[p.name] = normalizeMmeIpPlmnRows(toMmeIpPlmnRows(values[p.name]));
      }

      const newVal = special?.kind === 'mme-ip-plmn-table'
        ? serializeMmeIpPlmnList(toMmeIpPlmnRows(values[p.name]))
        : String(values[p.name] ?? '');
      const oldVal = rawItem?.parameterValue ?? item?.currentValue ?? '';
      if (newVal === oldVal) continue;

      const parameterType = rawItem?.parameterType ?? (item?.type as never) ?? 'string';
      const err = validateValue(newVal, parameterType, item?.constraints);
      if (err) {
        errors[p.name] = err;
        continue;
      }
      // XML 驱动的 extraInfoPath 范围校验:超出 [min, max] 阻断保存。
      const extraBounds = extraInfoBoundsByName.get(p.name);
      if (extraBounds) {
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
    // 识别 FAPService.1 HNBName → 改走 rename 接口（不走普通 SPV 下发）
    const hnbUpdate = updates.find((u) => u.parameterPath === HNB_NAME_PATH);
    const regularUpdates = hnbUpdate
      ? updates.filter((u) => u.parameterPath !== HNB_NAME_PATH)
      : updates;
    try {
      let renameTaskId: string | undefined;
      if (hnbUpdate) {
        const renameResult = await renameMutation.mutateAsync(hnbUpdate.parameterValue);
        renameTaskId = renameResult.taskId;
      }
      if (regularUpdates.length > 0) {
        const result = await updateMutation.mutateAsync({ deviceId, parameters: regularUpdates });
        latestLocalEditAtRef.current = 0;
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
        });
      } else if (hnbUpdate) {
        // 只有 rename：auto_omc_to_lmt 可能返回设备侧 taskId；prompt / 仅 OMC 侧成功则无任务进度。
        latestLocalEditAtRef.current = 0;
        message.success({
          content: t('device.cell.saveSuccessMsg', { count: 1 }),
          duration: 6,
        });
        if (renameTaskId) {
          setFeedback(fbKey, {
            kind: 'cell',
            submitStatus: 'queued',
            taskId: renameTaskId,
            count: 1,
            at: Date.now(),
          });
        } else {
          clearFeedback(fbKey);
          clearDraft(fbKey);
        }
      }
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
    if ((lastSubmit?.at ?? 0) < latestLocalEditAtRef.current) return;
    let cancelled = false;
    void (async () => {
      const refreshedSchemaByPath = new Map<string, ParameterSchemaItem>();
      try {
        if (isDeviceTimeGroup) {
          const [refreshedDeviceTime, refreshedManagementServer] = await Promise.all([
            refetchDeviceTimeSchema(),
            refetchManagementServerSchema(),
          ]);
          for (const item of refreshedDeviceTime.data?.parameters ?? []) {
            refreshedSchemaByPath.set(item.path, item);
          }
          for (const item of refreshedManagementServer.data?.parameters ?? []) {
            refreshedSchemaByPath.set(item.path, item);
          }
        } else {
          const refreshed = await refetchCommonSchema();
          for (const item of refreshed.data?.parameters ?? []) {
            refreshedSchemaByPath.set(item.path, item);
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
        return;
      }
      if (cancelled) return;
      const nextValues: Record<string, unknown> = {};
      for (const p of effectiveParams) {
        const special = specialConfigByName.get(p.name);
        const path = special?.configPath ?? resolveReadPath(p.standardPath || '');
        const refreshedValue = refreshedSchemaByPath.get(path)?.currentValue ?? '';
        nextValues[p.name] = special?.kind === 'mme-ip-plmn-table'
          ? toMmeIpPlmnRows(refreshedValue)
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
      clearDraft(fbKey);
      setFieldErrors({});
    })();
    return () => {
      cancelled = true;
    };
  }, [active, lastTask?.id, lastTask?.status, lastSubmit?.at, refetchCommonSchema, refetchDeviceTimeSchema, refetchManagementServerSchema, effectiveParams, instanceContext, form, clearDraft, fbKey, group.titleZh, specialConfigByName, t, isDeviceTimeGroup, deviceTimeModeOptionsKey, queryClient, deviceId]);

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
      extra={
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
          <Button type="primary" onClick={handleSave} loading={updateMutation.isPending}>
            {updateMutation.isPending ? t('device.cell.dispatching') : t('common.save')}
          </Button>
        </Space>
      }
      style={{ marginBottom: 16 }}
    >
      <Spin spinning={isSchemaLoading}>
      <Form
        form={form}
        layout="vertical"
        onValuesChange={(changedValues) => {
          latestLocalEditAtRef.current = Date.now();
          // 同步到 store draft，跨顶层 TabBar 切走切回可恢复
          for (const [name, value] of Object.entries(changedValues)) {
            const p = visibleParams.find((q) => q.name === name);
            const special = p ? specialConfigByName.get(p.name) : undefined;
            if (special?.kind === 'mme-ip-plmn-table') {
              setDraftField(fbKey, name, toMmeIpPlmnRows(value));
            } else {
              setDraftField(fbKey, name, String(value ?? ''));
            }
          }
          // T-0159: 交叉镜像 — 改 A 字段时把 A 的新值同步写入镜像字段 B（如 TDD 上下行带宽必须相等）。
          // antd Form.setFieldValue 不会触发 onValuesChange，故不会无限递归。
          for (const [name, value] of Object.entries(changedValues)) {
            const p = visibleParams.find((q) => q.name === name);
            if (!p) continue;
            const special = specialConfigByName.get(name);
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
              const special = specialConfigByName.get(name);
              const path = special?.configPath ?? resolveReadPath(p.standardPath || '');
              const sItem = schemaByPath.get(path);
              const normalizedValue = special?.kind === 'mme-ip-plmn-table'
                ? serializeMmeIpPlmnList(toMmeIpPlmnRows(value))
                : String(value ?? '');
              const err = validateValue(
                normalizedValue,
                (sItem?.type as never) ?? 'string',
                sItem?.constraints,
              );
              // XML 驱动的 extraInfoPath 范围校验:在 schema 校验之后追加;
              // schema 已报错时优先展示 schema 错误,避免双错信息互盖。
              const extraBounds = extraInfoBoundsByName.get(name);
              const rangeErr = !err && extraBounds
                ? validateExtraInfoBounds(normalizedValue, extraBounds)
                : null;
              const finalErr = err ?? rangeErr;
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
              <Col span={12}>
                <Form.Item label="NTP" name="Enable">
                  <Select
                    disabled={!modeWritable}
                    options={modeOptions}
                  />
                </Form.Item>
              </Col>
              <Col span={12}>
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
            if (isDeviceTimeGroup && (p.name === 'LocalTimeZoneName' || p.name === 'Enable')) {
              return null;
            }
            const special = specialConfigByName.get(p.name);
            const path = special?.configPath ?? resolveReadPath(p.standardPath || '');
            const item = schemaByPath.get(path);
            const rawItem = getRawValueByPath(rawParameterByPath, path)
              ?? (special?.kind === 'mme-ip-plmn-table'
                ? findRawValueBySuffix(mmeIpPlmnParams, '.MmeIpPlmnList')
                : special?.kind === 'bind-select' && p.name === 'NguBindInterface'
                  ? findRawValueBySuffix(nrNguParams, '.BindInterface')
                  : undefined);
            const displayPath = special?.displayPath;
            const displayValue = displayPath
              ? getRawValueByPath(rawParameterByPath, displayPath)?.parameterValue
                ?? (special?.kind === 'bind-select' && p.name === 'NguBindInterface'
                  ? findRawValueBySuffix(nrNguParams, '.NguLocalIp')?.parameterValue
                  : undefined)
                ?? schemaByPath.get(displayPath)?.currentValue
                ?? ''
              : '';
            // 紧贴 ARFCN 之后插入派生的 Frequency(MHz) 显示行。
            // 命中分组:BSC 的 GSM 空口分组(gsm-cell) 与 BTS 的基站信息分组(bts-cell-info)。
            const renderFrequencyAfter =
              (group.id === 'gsm-cell' || group.id === 'bts-cell-info') &&
              p.name === 'CurrentArfcn';
            // BM GSM 专属:在 BscSelect 后插入"绑定 RU 的 Route Index"派生行。
            const renderRuRouteAfter = isGsmCell && p.name === 'BscSelect';
            // BM LTE 派生显示:Frequency / Cell ID / 2T4R 开关。
            const isLteCell = group.id === 'enb-cell';
            const renderLteFreqAfter = isLteCell && p.name === 'DLEarfcn';
            const renderCellIdAfter = isLteCell && p.name === 'ECI';
            const render2T4RAfter = isLteCell && p.name === 'AntennaPortsCount';
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
            // 策略联动：auto_lmt_to_omc 下 FAPService.1 HNBName 禁用（引导在 LMT 侧改名）
            const resolvedStdPath = special?.configPath ?? resolveReadPath(p.standardPath || '');
            const lmtLocked = resolvedStdPath === HNB_NAME_PATH && nameSyncMode === 'auto_lmt_to_omc';
            const finalWritable = lmtLocked ? false : writable;
            const error = fieldErrors[p.name];
            // XML hideRangeHint="true" 时不在 label 后展示 schema 推导的 [min ~ max]
            // (字典范围与业务允许值不一致的字段如 Band:字典 1..maxInt,业务允许集只有少数频段)。
            const constraintHint = p.hideRangeHint ? '' : formatConstraintHint(item, t);
            const currentBindPath = String(form.getFieldValue(p.name) ?? rawItem?.parameterValue ?? item?.currentValue ?? '');
            const resolvedDisplayValue = displayValue || bindIpByPath.get(currentBindPath) || '';
            // XML 驱动:若 param 在 quicksettings XML 上声明了 extraInfoPath,
            // 把对应路径的当前值按 [lo ~ hi] 格式与 label 同一行显示(灰色小字)。
            const extraInfoRaw = p.extraInfoPath ? extraInfoValueByPath.get(p.extraInfoPath) ?? '' : '';
            const extraInfoFormatted = extraInfoRaw ? formatExtraInfoRange(extraInfoRaw) : '';
            const label = (
              <Space size={4}>
                <span style={special?.kind === 'mme-ip-plmn-table' ? { whiteSpace: 'nowrap' } : undefined}>
                  {locale === 'zh-CN' ? p.titleZh : p.titleEn}
                </span>
                {lmtLocked && <Text type="secondary" style={{ fontSize: 12 }}>{t('device.cell.lmtLockedHint')}</Text>}
                {!lmtLocked && !writable && <Text type="secondary" style={{ fontSize: 12 }}>{t('device.cell.readonly')}</Text>}
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
            const effectiveEnumValues = xmlEnumValues.length > 0
              ? xmlEnumValues
              : (enumMeta?.values ?? []);
            const effectiveEnumLabels = xmlEnumValues.length > 0
              ? xmlEnumLabels
              : (enumMeta?.labels ?? []);
            const isEnum = !special && effectiveEnumValues.length > 0;
            const extra = special?.kind === 'mme-ip-plmn-table'
              ? t('device.cell.mmeIpPlmnExtra')
              : undefined;
            const effectiveBindOptions = special?.kind === 'bind-select'
              ? appendCurrentBindOption(
                  bindSelectOptions,
                  currentBindPath,
                  resolvedDisplayValue,
                )
              : [];
            const input = (
              <Col span={special?.kind === 'mme-ip-plmn-table' ? 24 : 12} key={p.name}>
                <Form.Item
                  label={special?.kind === 'mme-ip-plmn-table' ? undefined : label}
                  name={p.name}
                  validateStatus={error ? 'error' : undefined}
                  help={error}
                  extra={extra}
                >
                  {special?.kind === 'bind-select' ? (
                    <Select
                      disabled={!finalWritable}
                      showSearch
                      optionFilterProp="label"
                      placeholder={t('device.cell.bindSelectPlaceholder')}
                      options={effectiveBindOptions}
                    />
                  ) : special?.kind === 'mme-ip-plmn-table' ? (
                    <MmeIpPlmnTable disabled={!finalWritable} locale={locale} />
                  ) : isEnum ? (
                    <Select
                      disabled={!finalWritable}
                      placeholder={item?.defaultValue || ''}
                      options={effectiveEnumValues.map((v, idx) => ({
                        value: v,
                        label: effectiveEnumLabels[idx] || v,
                      }))}
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
                <BoundRuRouteIndexDisplay form={form} ruRouteByIdx={ruRouteByIdx} locale={locale} />
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
            ) : render2T4RAfter ? (
              <Fragment key={p.name}>
                {input}
                <AntennaPortsAs2T4RDisplay form={form} locale={locale} />
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
