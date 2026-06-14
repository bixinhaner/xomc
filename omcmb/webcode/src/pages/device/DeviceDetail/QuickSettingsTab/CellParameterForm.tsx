import { useEffect, useMemo, useState, Fragment } from 'react';
import { Button, Card, Col, Form, Input, Row, Select, Space, Spin, Table, Tag, Typography, message, notification } from 'antd';
import type { FormInstance } from 'antd';
import { CheckCircleOutlined, ClockCircleOutlined, CloseCircleOutlined, DeleteOutlined, PlusOutlined, SendOutlined, SyncOutlined } from '@ant-design/icons';
import { useQueryClient } from '@tanstack/react-query';
import { useParameterSchema, useSearchParameters, useUpdateParameters } from '@core/hooks/api/useDeviceParameters';
import { useDeviceTaskStatus } from '@core/hooks/api/useDeviceTask';
import { notificationKeys } from '@core/hooks/api/useNotificationCenter';
import {
  feedbackKey,
  useQuickSettingsFeedbackStore,
  type CellFeedback,
} from '@core/store/quickSettingsFeedbackStore';
import type { DeviceParameter, ParameterSchemaItem, ParameterUpdateRequest } from '@core/types/deviceParameter';
import type { DeviceTaskStatus } from '@core/types/deviceTask';
import type { QuickSettingsGroup } from '@core/types/quicksettings';
import { applyInstanceContext, getEffectiveEnumMeta, validateValue, type QuickSettingsInstanceContext } from './validators';
import { useT } from '@/hooks/useT';

const { Text } = Typography;
const ERROR_FEEDBACK_DURATION_SECONDS = 2;

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

interface CellParameterFormProps {
  deviceId: string;
  group: QuickSettingsGroup;
  instanceContext: QuickSettingsInstanceContext;
  locale: 'zh-CN' | 'en-US';
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
export default function CellParameterForm({ deviceId, group, instanceContext, locale }: CellParameterFormProps) {
  const t = useT();
  const [form] = Form.useForm();
  const updateMutation = useUpdateParameters();
  const queryClient = useQueryClient();
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  // lastSubmit 由 zustand store 托管 —— DeviceDetail 整个被卸载(切顶层 tab)也保留反馈。
  const fbKey = feedbackKey(
    deviceId,
    group.id,
    instanceContext.fapInstance,
    instanceContext.networkType === 'nr' ? instanceContext.cellInstance : undefined,
  );
  const lastSubmit = useQuickSettingsFeedbackStore((s) => {
    const f = s.entries[fbKey];
    return f && f.kind === 'cell' ? f : null;
  });
  const setFeedback = useQuickSettingsFeedbackStore((s) => s.setFeedback);
  const patchFeedback = useQuickSettingsFeedbackStore((s) => s.patchFeedback);
  const draft = useQuickSettingsFeedbackStore((s) => s.drafts[fbKey]);
  const setDraftField = useQuickSettingsFeedbackStore((s) => s.setDraftField);
  const clearDraft = useQuickSettingsFeedbackStore((s) => s.clearDraft);

  // 单实例分组：每条 standardPath 单独查 schema（少量字段，不批量优化）
  // 注：useParameterSchema 接受 pathPrefix，前缀匹配即可；这里以分组共用前缀粗查再过滤
  // 为简化，取 group 中 standardPath 的公共前缀作 pathPrefix
  const commonPrefix = useMemo(
    () => commonPathPrefix(group.params.map((p) => applyInstanceContext(p.standardPath || '', instanceContext))),
    [group, instanceContext],
  );
  const { data: schemaResp, isLoading, refetch } = useParameterSchema(deviceId, commonPrefix);
  const { data: ethernetSchemaResp } = useParameterSchema(
    deviceId,
    'Device.Ethernet.Interface.',
    group.id === 'gsm-abis' || group.id === 'gnb-core',
  );
  const { data: mmeIpPlmnParams } = useSearchParameters(
    deviceId,
    group.id === 'enb-mme' ? 'MmeIpPlmnList' : '',
    50,
    group.id === 'enb-mme',
  );
  const { data: nrCommonParams } = useSearchParameters(
    deviceId,
    group.id === 'gnb-core' ? 'Device.Services.FAPService.1.FAPControl.NR.RAN.Common.' : '',
    50,
    group.id === 'gnb-core',
  );
  const { data: nrNguParams } = useSearchParameters(
    deviceId,
    group.id === 'gnb-core' ? 'Device.FAP.NguIpBind' : '',
    200,
    group.id === 'gnb-core',
  );
  const { data: deviceTimeParams } = useSearchParameters(
    deviceId,
    group.id === 'device-time' ? 'Device.Time.' : '',
    200,
    group.id === 'device-time',
  );

  // BM GSM 专属:并行拉 RU 节点 schema,用于在 gsm-cell 表单中展示"绑定 RU 的 Route Index"。
  // 拉取与主 schema 解耦,避免污染 commonPrefix 退化成 Device. 触发全量拉取。
  const isGsmCell = group.id === 'gsm-cell';
  const { data: ruSchemaResp } = useParameterSchema(
    deviceId,
    'Device.DeviceInfo.RU.',
    isGsmCell,
  );
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
    schemaResp?.parameters.forEach((p) => map.set(p.path, p));
    return map;
  }, [schemaResp]);
  const rawParameterByPath = useMemo(() => {
    const map = new Map<string, DeviceParameter>();
    for (const item of [...(mmeIpPlmnParams ?? []), ...(nrCommonParams ?? []), ...(nrNguParams ?? []), ...(deviceTimeParams ?? [])]) {
      map.set(item.parameterPath, item);
    }
    return map;
  }, [mmeIpPlmnParams, nrCommonParams, nrNguParams, deviceTimeParams]);
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
    for (const param of group.params) {
      const config = buildSpecialFieldConfig(group.id, param.name, instanceContext);
      if (config) {
        map.set(param.name, config);
      }
    }
    return map;
  }, [group.id, group.params, instanceContext]);

  // T-0159: 交叉镜像 — 反查表 resolved standardPath → form field name，
  // 让 onValuesChange 时能据 constraints.mirrorWith 找到对端 form field 并 setFieldValue 同步。
  const paramNameByPath = useMemo(() => {
    const map = new Map<string, string>();
    for (const p of group.params) {
      const path = applyInstanceContext(p.standardPath || '', instanceContext);
      map.set(path, p.name);
    }
    return map;
  }, [group, instanceContext]);

  // 初始化字段值 —— 优先级：store draft > 当前会话已 touched > schema 原值。
  // 未保存草稿在跨顶层 TabBar 切换后恢复；任务终态回读后再 clearDraft，统一回到设备侧值。
  useEffect(() => {
    if (!schemaResp) return;
    group.params.forEach((p) => {
      if (draft && draft[p.name] !== undefined) {
        if (specialConfigByName.get(p.name)?.kind === 'mme-ip-plmn-table') {
          form.setFieldValue(p.name, toMmeIpPlmnRows(draft[p.name]));
        } else {
          form.setFieldValue(p.name, String(draft[p.name] ?? ''));
        }
        return;
      }
      // 优先级 2: 用户在当前会话已 touched
      if (form.isFieldTouched(p.name)) return;
      // 优先级 3: schema 原值
      const special = specialConfigByName.get(p.name);
      const path = special?.configPath ?? applyInstanceContext(p.standardPath || '', instanceContext);
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
        form.setFieldValue(p.name, rawItem?.parameterValue ?? item?.currentValue ?? '');
      }
    });
  }, [schemaResp, group, instanceContext, form, schemaByPath, rawParameterByPath, draft, specialConfigByName]);

  const handleSave = async () => {
    const values = form.getFieldsValue() as Record<string, unknown>;
    const updates: ParameterUpdateRequest[] = [];
    const errors: Record<string, string> = {};

    for (const p of group.params) {
      const special = specialConfigByName.get(p.name);
      const path = special?.configPath ?? applyInstanceContext(p.standardPath || '', instanceContext);
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
    try {
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
  const { data: lastTask } = useDeviceTaskStatus(lastSubmit?.taskId);

  useEffect(() => {
    if (!lastTask || !['completed', 'failed', 'expired', 'cancelled'].includes(lastTask.status)) return;
    let cancelled = false;
    void (async () => {
      let refreshed;
      try {
        refreshed = await refetch();
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
      const refreshedSchemaByPath = new Map((refreshed.data?.parameters ?? []).map((item) => [item.path, item]));
      for (const p of group.params) {
        const special = specialConfigByName.get(p.name);
        const path = special?.configPath ?? applyInstanceContext(p.standardPath || '', instanceContext);
        const refreshedValue = refreshedSchemaByPath.get(path)?.currentValue ?? '';
        nextValues[p.name] = special?.kind === 'mme-ip-plmn-table'
          ? toMmeIpPlmnRows(refreshedValue)
          : refreshedValue;
      }
      form.setFieldsValue(nextValues);
      clearDraft(fbKey);
      setFieldErrors({});
    })();
    return () => {
      cancelled = true;
    };
  }, [lastTask?.id, lastTask?.status, refetch, group.params, instanceContext, form, clearDraft, fbKey, group.titleZh, specialConfigByName, t]);

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

  return (
    <Card
      title={title}
      size="small"
      extra={
        <Space>
          {visibleLastSubmit && (() => {
            const spec = statusTagSpec(visibleLastSubmit, lastTask?.status, t);
            return (
              <Tag icon={spec.icon} color={spec.color}>
                {spec.label} · {t('device.cell.itemsCount', { count: visibleLastSubmit.count })} · {formatTime(visibleLastSubmit.at)}
              </Tag>
            );
          })()}
          <Button type="primary" onClick={handleSave} loading={updateMutation.isPending}>
            {updateMutation.isPending ? t('device.cell.dispatching') : t('common.save')}
          </Button>
        </Space>
      }
      style={{ marginBottom: 16 }}
    >
      <Spin spinning={isLoading}>
      <Form
        form={form}
        layout="vertical"
        onValuesChange={(changedValues) => {
          // 同步到 store draft，跨顶层 TabBar 切走切回可恢复
          for (const [name, value] of Object.entries(changedValues)) {
            const p = group.params.find((q) => q.name === name);
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
            const p = group.params.find((q) => q.name === name);
            if (!p) continue;
            const special = specialConfigByName.get(name);
            const path = special?.configPath ?? applyInstanceContext(p.standardPath || '', instanceContext);
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
              const p = group.params.find((q) => q.name === name);
              if (!p) continue;
              const special = specialConfigByName.get(name);
              const path = special?.configPath ?? applyInstanceContext(p.standardPath || '', instanceContext);
              const sItem = schemaByPath.get(path);
              const normalizedValue = special?.kind === 'mme-ip-plmn-table'
                ? serializeMmeIpPlmnList(toMmeIpPlmnRows(value))
                : String(value ?? '');
              const err = validateValue(
                normalizedValue,
                (sItem?.type as never) ?? 'string',
                sItem?.constraints,
              );
              if (err) next[name] = err;
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
        <Row gutter={16}>
          {group.params.map((p) => {
            const special = specialConfigByName.get(p.name);
            const path = special?.configPath ?? applyInstanceContext(p.standardPath || '', instanceContext);
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
            // 紧贴 ARFCN 之后插入派生的 Frequency(MHz) 显示行(GSM 空口分组专属)。
            const renderFrequencyAfter = group.id === 'gsm-cell' && p.name === 'CurrentArfcn';
            // BM GSM 专属:在 BscSelect 后插入"绑定 RU 的 Route Index"派生行。
            const renderRuRouteAfter = isGsmCell && p.name === 'BscSelect';
            // BM LTE 派生显示:Frequency / Cell ID / 2T4R 开关。
            const isLteCell = group.id === 'enb-cell';
            const renderLteFreqAfter = isLteCell && p.name === 'DLEarfcn';
            const renderCellIdAfter = isLteCell && p.name === 'ECI';
            const render2T4RAfter = isLteCell && p.name === 'AntennaPortsCount';
            const isDeviceTimeParam = group.id === 'device-time';
            const writable = special?.forceWritable
              ?? rawItem?.writable
              ?? item?.writable
              ?? (isDeviceTimeParam ? true : false);
            const error = fieldErrors[p.name];
            const constraintHint = formatConstraintHint(item, t);
            const currentBindPath = String(form.getFieldValue(p.name) ?? rawItem?.parameterValue ?? item?.currentValue ?? '');
            const resolvedDisplayValue = displayValue || bindIpByPath.get(currentBindPath) || '';
            const label = (
              <Space size={4}>
                <span style={special?.kind === 'mme-ip-plmn-table' ? { whiteSpace: 'nowrap' } : undefined}>
                  {locale === 'zh-CN' ? p.titleZh : p.titleEn}
                </span>
                {!writable && <Text type="secondary" style={{ fontSize: 12 }}>{t('device.cell.readonly')}</Text>}
                {constraintHint && (
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {constraintHint}
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
            const isEnum = !special && Boolean(enumMeta && enumMeta.values.length > 0);
            const extra = special?.kind === 'mme-ip-plmn-table'
              ? t('device.cell.mmeIpPlmnExtra')
              : special?.kind === 'bind-select'
                ? undefined
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
                      disabled={!writable}
                      showSearch
                      optionFilterProp="label"
                      placeholder={t('device.cell.bindSelectPlaceholder')}
                      options={effectiveBindOptions}
                    />
                  ) : special?.kind === 'mme-ip-plmn-table' ? (
                    <MmeIpPlmnTable disabled={!writable} locale={locale} />
                  ) : isEnum ? (
                    <Select
                      disabled={!writable}
                      placeholder={item?.defaultValue || ''}
                      options={enumMeta!.values.map((v, idx) => ({
                        value: v,
                        label: enumMeta!.labels[idx] || v,
                      }))}
                    />
                  ) : (
                    <Input disabled={!writable} placeholder={special?.placeholder || item?.defaultValue || ''} />
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
