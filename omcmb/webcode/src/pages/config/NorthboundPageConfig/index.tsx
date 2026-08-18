import { forwardRef, memo, type ReactNode, useCallback, useEffect, useImperativeHandle, useMemo, useRef, useState } from 'react';
import {
  AutoComplete,
  Button,
  Descriptions,
  Drawer,
  Dropdown,
  Form,
  Input,
  InputNumber,
  Modal,
  Pagination,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tabs,
  Tag,
  Tooltip,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  ApiOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  CopyOutlined,
  DatabaseOutlined,
  DeleteOutlined,
  DownloadOutlined,
  EditOutlined,
  EyeOutlined,
  FileSearchOutlined,
  FileTextOutlined,
  LoadingOutlined,
  MoreOutlined,
  PlayCircleOutlined,
  PlusOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  WifiOutlined,
} from '@ant-design/icons';

import ListPageLayout from '@/components/Layout/ListPageLayout';
import { northboundPageConfigApi } from '@core/services/api/northboundPageConfigApi';
import type {
  NorthboundFileGroup,
  NorthboundFileProfile,
  NorthboundFileRun,
  NorthboundAPIConfig,
  NorthboundAPIUser,
  NorthboundDeliveryTarget,
  NorthboundFieldDefinition,
  NorthboundInventoryField,
  NorthboundInventoryProfile,
  NorthboundPageConfigEvent,
  NorthboundPageConfigCompressionFormat,
  NorthboundPageConfigFormat,
  NorthboundPageConfigPeriod,
  NorthboundSNMPAlarmTarget,
  NorthboundSocketAlarmConfig,
  NorthboundUpdateFileProfileRequest,
  NorthboundUpdateInventoryProfileRequest,
} from '@core/services/api/northboundPageConfigApi';
import { loadPmMetrics, type PmMetric } from './pmMetricCatalog';
import { NorthboundI18nScope, useNorthboundI18n, useNorthboundLocale } from './i18n';
import styles from './index.module.css';

type Domain = 'CM' | 'PM' | 'MR' | 'LOG' | 'INVENTORY';
type Format = 'XML' | 'CSV' | 'TXT';
type CompressionFormat = 'zip' | 'gz';
type MetricType = 'counter' | 'kpi';
type Tech = 'LTE' | 'GNB' | 'GSM';
type FieldTechFilter = 'ALL' | Tech;
type InventoryType = 'ENB' | 'GNB' | 'GSM' | 'OMC';

function nowrapColumnTitle(title: string) {
  return <span className={styles.nowrapHeader}>{title}</span>;
}

interface ScenarioObject {
  code: string;
  tech?: Tech;
  profile?: string;
}

interface FileGroup {
  id: string;
  domain: Domain;
  format: Format;
  period: string;
  cron: string;
  path: string;
  name: string;
  csvSeparator?: string;
  compressionEnabled: boolean;
  compressionFormat: CompressionFormat;
  objects: ScenarioObject[];
  selectedFields?: string[];
}

interface LogConfig {
  mode: 'custom' | 'fix';
  cron: string;
  period: string;
  compression: string;
}

interface ScenarioRow {
  code: string;
  vendor: string;
  scenarioName: string;
  scenarioNameEn: string;
  description: string;
  flags: string[];
  name: string;
  enabled: boolean;
  groups: FileGroup[];
  logs?: LogConfig;
  createdAt?: string;
  updatedAt?: string;
}

interface ScenarioPeriodRow {
  key: string;
  domain: Domain;
  scope: string;
  format: Format;
  period: string;
  trigger: string;
  cron: string;
  objects: string;
  path: string;
  fileName: string;
  csvSeparator?: string;
  compressionEnabled: boolean;
  compressionFormat: CompressionFormat;
  selectedFields?: string[];
  objectProfiles?: Record<string, string>;
}

interface InventoryField {
  key: string;
  template: InventoryType;
  column: string;
  exportKey: string;
  source: string;
  dataType: string;
  renderer: string;
  enabled: boolean;
}

interface InventoryConfigRow {
  key: InventoryType;
  name: string;
  objectCode: string;
  tech: string;
  period: string;
  cron: string;
  format: Format;
  path: string;
  fileName: string;
  compressionEnabled: boolean;
  compressionFormat: CompressionFormat;
  createdAt?: string;
  updatedAt?: string;
}

interface FileProfileEditorValues {
  scenarioCode: string;
  vendor?: string;
  scenarioName: string;
  scenarioNameEn?: string;
  name: string;
  description?: string;
  domains?: Domain[];
  enabled?: boolean;
}

type DeliveryProtocol = 'FTP' | 'SFTP';
type DeliveryAuthMode = 'PASSWORD' | 'PRIVATE_KEY';
type DeliveryHostKeyPolicy = 'INSECURE' | 'FINGERPRINT';
type DeliveryTargetScope = 'file' | 'inventory' | 'socket';
type DeliveryTargetValidationField = 'host' | 'username' | 'credential';
type DeliveryTargetValidationErrors = Record<string, Partial<Record<DeliveryTargetValidationField, string>>>;
type ReportState = 'success' | 'failed' | 'running' | 'idle';
type ReportArtifactType = 'file' | 'message';
type ProfileHealthState = 'normal' | 'broken' | 'pending' | 'terminated';

const REPORT_TRANSFER_FAILED_TEXT = '生成成功，传输失败';
const REPORT_TRANSFER_FAILED_SHORT_TEXT = '传输失败';
const REPORT_TRANSFER_FAILED_DETAIL = '文件已生成，但 FTP/SFTP 传输目标上传失败，请查看投递结果。';

interface ProfileHealthInfo {
  state: ProfileHealthState;
  label: string;
  color: 'success' | 'error' | 'warning' | 'default';
  detail: string;
}

interface DeliveryTargetRow {
  key: string;
  name: string;
  enabled: boolean;
  protocol: DeliveryProtocol;
  host: string;
  port: number;
  username: string;
  credential: string;
  authMode: DeliveryAuthMode;
  remoteRoot: string;
  retryTimes: number;
  timeoutSeconds: number;
  passiveMode: boolean;
  hostKeyPolicy: DeliveryHostKeyPolicy;
  hostKeyFingerprint: string;
}

interface ReportStatusInfo {
  key: string;
  runId?: string;
  capabilityName: string;
  state: ReportState;
  statusText: string;
  lastTime: string;
  artifactType: ReportArtifactType;
  artifactName: string;
  artifactPath: string;
  size: string;
  targetSummary: string;
  detail: string;
  payload: string;
  deliveryNote?: string;
  previewTitle?: string;
  copyLabel?: string;
  resultTitle?: string;
}

interface DeliveryTestResultInfo {
  event: NorthboundPageConfigEvent;
  title: string;
}

interface DeliveryTestDisplayInfo {
  state: ReportState;
  statusText: string;
  target: string;
  tcpText: string;
  authText: string;
  message: string;
}

type DeliveryEventsByRunId = Record<string, NorthboundPageConfigEvent[]>;

interface MessageFieldRow {
  key: string;
  field: string;
  value: string;
}

interface DelimitedMessageRow {
  key: string;
  command: string;
  fieldCount: number;
  summary: string;
}

interface SnmpVarBindRow {
  key: string;
  index: number;
  oid: string;
  field: string;
  value: string;
}

type ApiMethod = 'GET' | 'POST' | 'PUT' | 'DELETE';
type ApiKind = '正式北向' | '业务复用' | '鉴权管理' | '日志管理';
type ApiFieldContract = '完整返回字段' | '当前返回字段' | '示例返回字段';

const csvSeparatorOptions = [
  { value: ',', label: '逗号 ,' },
  { value: '|', label: '竖线 |' },
  { value: ';', label: '分号 ;' },
  { value: '\\t', label: 'Tab' },
];

type SocketProfile = 'CTCC' | 'CUCC';
type SnmpVersion = 'v2' | 'v3';
type SnmpNotificationType = 'Trap' | 'Inform';
type SnmpV3SecurityLevel = 'noAuthNoPriv' | 'authNoPriv' | 'authPriv';

const snmpAuthProtocolOptions = [
  { label: 'SHA', value: 'SHA' },
  { label: 'SHA224', value: 'SHA224' },
  { label: 'SHA256', value: 'SHA256' },
  { label: 'SHA384', value: 'SHA384' },
  { label: 'SHA512', value: 'SHA512' },
  { label: 'MD5', value: 'MD5' },
];

const snmpPrivProtocolOptions = [
  { label: 'DES', value: 'DES' },
  { label: 'AES128', value: 'AES128' },
  { label: 'AES192', value: 'AES192' },
  { label: 'AES256', value: 'AES256' },
];

const snmpV3SecurityLevelOptions: Array<{ label: string; value: SnmpV3SecurityLevel }> = [
  { label: '认证并加密', value: 'authPriv' },
  { label: '仅认证', value: 'authNoPriv' },
  { label: '不认证不加密', value: 'noAuthNoPriv' },
];

const snmpDefaultCommunity = 'baicells';
const storedCredentialText = '已加密存储';
const credentialMaskText = '********';

interface SocketAlarmConfigRow {
  key: string;
  name: string;
  profile: SocketProfile;
  frame: string;
  listenIp: string;
  listenPort: number;
  maxClients: number;
  heartbeatPeriod: number;
  heartbeatTimes: number;
  encoding: string;
  realTimeEnabled: boolean;
  historyEnabled: boolean;
  syncMode: string;
  accountTypes: string;
  sequencePolicy: string;
}

interface SnmpAlarmTargetRow {
  key: string;
  name: string;
  version: SnmpVersion;
  notificationType: SnmpNotificationType;
  listenIp: string;
  listenPort: number;
  targetHost: string;
  targetPort: number;
  community?: string;
  securityName?: string;
  authProtocol?: string;
  authCredential?: string;
  privProtocol?: string;
  privCredential?: string;
  mibQueryEnabled: boolean;
  clearSeverityPolicy: string;
  timeoutSeconds: number;
  retries: number;
}

interface AlarmFieldMappingRow {
  key: string;
  order: number;
  field: string;
  cnName: string;
  source: string;
  dataType: string;
  enabled: boolean;
}

interface SocketAccountRow {
  key: string;
  channel: string;
  username: string;
  type: 'msg' | 'ftp';
  credential: string;
  enabled: boolean;
  purpose: string;
}

interface NorthboundApiRow {
  key: string;
  configKey?: string;
  apiKind?: ApiKind;
  dataType?: string;
  module: string;
  name: string;
  method: ApiMethod;
  url: string;
  auth: string;
  backendSource: string;
  fieldContract?: ApiFieldContract;
  responseFields?: string[];
  requestExample: string;
  responseExample: string;
}

type NorthboundApiDisplayAlias = Partial<NorthboundApiRow> & Pick<
  NorthboundApiRow,
  'key' | 'name' | 'method' | 'url' | 'requestExample' | 'responseExample'
>;

interface ApiUserRow {
  key: string;
  username: string;
  enabled: boolean;
  password: string;
  passwordSet: boolean;
  createdAt?: string;
}

interface ReportFieldRow {
  key: string;
  domain: Domain;
  objectCode: string;
  scope?: string;
  tech?: Tech;
  outputAlias: string;
  systemField: string;
  source: string;
  dataType: string;
  renderer: string;
  productClasses?: string[];
  metricType?: MetricType;
  statisType?: string;
  unit?: string;
  cnName?: string;
  enabled: boolean;
}

type FieldDefinition = Omit<ReportFieldRow, 'key' | 'domain' | 'objectCode' | 'enabled'>;

interface FieldTarget {
  key: string;
  domain: Domain;
  objectCode: string;
  scope: string;
  format: Format;
  profile: string;
  tech?: Tech;
}

const PATH_CM = '/#FTPRoot#/#Province#/#OMC-R#/CM/#DateTime#/';
const PATH_PM = '/#FTPRoot#/#Province#/#OMC-R#/PM/#DateTime#/';
const PATH_MR = '/#FTPRoot#/#Province#/#OMC-R#/MR/#DateTime#/';
const PATH_LOG = '/#FTPRoot#/LOGS/#Date#/';
const PATH_INVENTORY = '/#FTPRoot#/#Province#/#OMC-R#/Inventory/#Object#/#DateTime#/';
const CM_NAME = 'Baicells-#Object#-#LocalHost#-#DataVersion#-#DateTime#[-#Ri#][-#FileID#]';
const PM_NAME = 'Baicells-#Object#-#LocalHost#-#DataVersion#-#DateTime#[-#Ri#]-#DataPeriod#[-#FileID#]';
const MR_NAME = '#ModuleType#-Baicells-#Object#-#LocalHost#-#eNBID#-#DateTime#[-#Ri#].xml';
const LOG_CUSTOM_NAME = '#Object#_#Date#.txt';
const LOG_FIX_NAME = '#Object#_#PeriodStartTime#-24H.csv';
const INVENTORY_NAME = 'BaiOMC_#Object#_#DateTime#.csv';

const domainOptions = [
  { label: 'CM', value: 'CM' },
  { label: 'PM', value: 'PM' },
  { label: 'MR', value: 'MR' },
  { label: 'LOG', value: 'LOG' },
  { label: 'INVENTORY', value: 'INVENTORY' },
];

const fileDomainOptions = domainOptions.filter((option) => option.value !== 'INVENTORY');

const formatLabels: Record<Format, string> = {
  XML: 'XML',
  CSV: 'CSV',
  TXT: 'TXT',
};

const supportedFormatsByDomain: Record<Domain, Format[]> = {
  CM: ['XML', 'CSV'],
  PM: ['CSV'],
  MR: ['XML'],
  LOG: ['TXT', 'CSV'],
  INVENTORY: ['CSV'],
};

const defaultFormatByDomain: Record<Domain, Format> = {
  CM: 'XML',
  PM: 'CSV',
  MR: 'XML',
  LOG: 'TXT',
  INVENTORY: 'CSV',
};

const supportedFormatsByCMObject: Record<string, Format[]> = {
  CP: ['XML', 'CSV'],
  EP: ['XML', 'CSV'],
  CC: ['XML', 'CSV'],
  CE: ['XML', 'CSV'],
  COMS: ['CSV', 'XML'],
};

const formatOrder: Format[] = ['XML', 'CSV', 'TXT'];

const defaultRemotePathByDomain: Record<Domain, string> = {
  CM: PATH_CM,
  PM: PATH_PM,
  MR: PATH_MR,
  LOG: PATH_LOG,
  INVENTORY: PATH_INVENTORY,
};

function intersectFormats(left: Format[], right: Format[]): Format[] {
  return formatOrder.filter((format) => left.includes(format) && right.includes(format));
}

function supportedFormatsForObject(domain: Domain, objectCode: string): Format[] {
  if (domain === 'CM') {
    return supportedFormatsByCMObject[objectCode.trim().toUpperCase()] ?? supportedFormatsByDomain[domain];
  }
  return supportedFormatsByDomain[domain];
}

function selectedObjectCodes(objectsValue?: string): string[] {
  return splitObjects(objectsValue ?? '').map((token) => parseObjectToken(token).code).filter(Boolean);
}

function isCMCOMSOnlySelection(domain: Domain, objectCodes: string[]): boolean {
  return domain === 'CM' && objectCodes.length > 0 && objectCodes.every((code) => code.trim().toUpperCase() === 'COMS');
}

function supportedFormatsForSelection(domain: Domain, objectsValue?: string): Format[] {
  const baseFormats = supportedFormatsByDomain[domain];
  const objectCodes = selectedObjectCodes(objectsValue);
  if (objectCodes.length === 0) return baseFormats;
  if (isCMCOMSOnlySelection(domain, objectCodes)) return supportedFormatsByCMObject.COMS;
  const formats = objectCodes.reduce(
    (available, objectCode) => intersectFormats(available, supportedFormatsForObject(domain, objectCode)),
    baseFormats,
  );
  return formats.length > 0 ? formats : baseFormats;
}

function getFormatOptions(domain: Domain, objectsValue?: string) {
  return supportedFormatsForSelection(domain, objectsValue).map((value) => ({ label: formatLabels[value], value }));
}

function normalizeFormatForDomain(domain: Domain, format: Format, objectsValue?: string): Format {
  const formats = supportedFormatsForSelection(domain, objectsValue);
  const defaultFormat = defaultFormatForSelection(domain, objectsValue);
  if (formats.includes(format)) return format;
  return formats.includes(defaultFormat) ? defaultFormat : formats[0] ?? defaultFormatByDomain[domain];
}

function defaultFormatForSelection(domain: Domain, objectsValue?: string): Format {
  const objectCodes = selectedObjectCodes(objectsValue);
  if (isCMCOMSOnlySelection(domain, objectCodes)) return 'CSV';
  return defaultFormatByDomain[domain];
}

const periodOptions = [
  { label: '15 分钟', value: '15M' },
  { label: '60 分钟', value: '60M' },
  { label: '每日', value: '24H' },
];

const startMinuteOptions = Array.from({ length: 60 }, (_, minute) => String(minute).padStart(2, '0'))
  .map((minute) => ({
    label: `${minute} 分开始`,
    value: minute,
  }));

const defaultScopeOption = { label: '默认', value: '默认' };
const radioTechScopeOptions: Array<{ label: string; value: Tech }> = [
  { label: 'ENB', value: 'LTE' },
  { label: 'GNB', value: 'GNB' },
  { label: 'GSM', value: 'GSM' },
];
const logScopeOptions = [
  { label: 'custom', value: 'custom' },
  { label: 'fix', value: 'fix' },
];

const compressionFormatOptions = [
  { label: 'zip', value: 'zip' },
  { label: 'gz', value: 'gz' },
];

const allProductClassValue = 'ALL';
const fieldCandidateDefaultOptionLimit = 10;
const fieldCandidateSearchOptionLimit = 50;

const radioFieldTechFilterOptions: Array<{ label: string; value: FieldTechFilter }> = [
  { label: 'ENB', value: 'LTE' },
  { label: 'GNB', value: 'GNB' },
  { label: 'GSM', value: 'GSM' },
];
const nonRadioFieldTechFilterOptions: Array<{ label: string; value: FieldTechFilter }> = [
  { label: '不区分制式', value: 'ALL' },
];

const objectOptionsByDomain: Record<Domain, Array<{ label: string; value: string }>> = {
  CM: ['CP', 'EP', 'CC', 'CE', 'COMS'].map((value) => ({ label: value, value })),
  PM: ['PC', 'PE'].map((value) => ({ label: value, value })),
  MR: ['MRO', 'MRE', 'MRS'].map((value) => ({ label: value, value })),
  LOG: [
    { label: '登录/安全日志', value: 'login' },
    { label: '操作日志', value: 'operation' },
    { label: '登录/安全日志（固定格式）', value: 'login_fix' },
    { label: '操作日志（固定格式）', value: 'operation_fix' },
  ],
  INVENTORY: ['eNB', 'gNB', 'GSM', 'OMC'].map((value) => ({ label: value, value })),
};

const logObjectLabelByCode: Record<string, string> = {
  login: '登录/安全日志',
  operation: '操作日志',
  login_fix: '登录/安全日志（固定格式）',
  operation_fix: '操作日志（固定格式）',
};

const cmObjects: ScenarioObject[] = [{ code: 'CP' }, { code: 'EP' }, { code: 'CC' }, { code: 'CE' }];
const cmObjectsWithComs: ScenarioObject[] = [...cmObjects, { code: 'COMS' }];
const cmGnbObjects: ScenarioObject[] = [
  { code: 'CP', tech: 'GNB' },
  { code: 'EP', tech: 'GNB' },
  { code: 'CC', tech: 'GNB' },
  { code: 'CE', tech: 'GNB' },
];
const mrObjects: ScenarioObject[] = [{ code: 'MRO' }, { code: 'MRE' }, { code: 'MRS' }];

function group(
  id: string,
  domain: Domain,
  format: Format,
  period: string,
  cron: string,
  path: string,
  name: string,
  objects: ScenarioObject[],
  compressionEnabled = true,
  compressionFormat: CompressionFormat = 'zip',
): FileGroup {
  const objectsValue = joinObjects(objects.map((object) => object.code));
  return {
    id,
    domain,
    format: normalizeFormatForDomain(domain, format, objectsValue),
    period,
    cron,
    path,
    name,
    compressionEnabled,
    compressionFormat,
    objects,
  };
}

const rawScenarioRows: Array<Omit<ScenarioRow, 'vendor' | 'scenarioName' | 'scenarioNameEn' | 'description' | 'flags'>> = [
  {
    code: 'S0001',
    name: 'CM + PM PC + MR 标准 15M',
    enabled: false,
    groups: [
      group('cm-daily', 'CM', 'XML', '24H', '30 1 0 * * ?', PATH_CM, CM_NAME, cmObjects),
      group('pm-15m', 'PM', 'CSV', '15M', '0 5/15 * * * ?', PATH_PM, PM_NAME, [{ code: 'PC' }]),
      group('mr-15m', 'MR', 'XML', '15M', '0 0/15 * * * ?', PATH_MR, MR_NAME, mrObjects),
    ],
  },
  {
    code: 'S0002',
    name: 'CM + PM PE/PC + MR',
    enabled: false,
    groups: [
      group('cm-daily', 'CM', 'XML', '24H', '30 1 0 * * ?', PATH_CM, CM_NAME, cmObjects),
      group('pm-15m', 'PM', 'CSV', '15M', '0 5/15 * * * ?', PATH_PM, PM_NAME, [{ code: 'PE' }, { code: 'PC' }]),
      group('mr-15m', 'MR', 'XML', '15M', '0 0/15 * * * ?', PATH_MR, MR_NAME, mrObjects),
    ],
  },
  {
    code: 'S0003',
    name: 'CM + PM PE/PC + MR',
    enabled: false,
    groups: [
      group('cm-daily', 'CM', 'XML', '24H', '30 1 0 * * ?', PATH_CM, CM_NAME, cmObjects),
      group('pm-15m', 'PM', 'CSV', '15M', '0 5/15 * * * ?', PATH_PM, PM_NAME, [{ code: 'PE' }, { code: 'PC' }]),
      group('mr-15m', 'MR', 'XML', '15M', '0 0/15 * * * ?', PATH_MR, MR_NAME, mrObjects),
    ],
  },
  {
    code: 'S0004',
    name: 'CM + PM PE/PC + MR',
    enabled: false,
    groups: [
      group('cm-daily', 'CM', 'XML', '24H', '30 1 0 * * ?', PATH_CM, CM_NAME, cmObjects),
      group('pm-15m', 'PM', 'CSV', '15M', '0 5/15 * * * ?', PATH_PM, PM_NAME, [{ code: 'PE' }, { code: 'PC' }]),
      group('mr-15m', 'MR', 'XML', '15M', '0 0/15 * * * ?', PATH_MR, MR_NAME, mrObjects),
    ],
  },
  {
    code: 'S0005',
    name: 'PM 延迟生成 + 自定义日志',
    enabled: false,
    groups: [
      group('cm-daily', 'CM', 'XML', '24H', '30 1 0 * * ?', PATH_CM, CM_NAME, cmObjects),
      group('pm-15m-delayed', 'PM', 'CSV', '15M', '0 14/15 * * * ?', PATH_PM, PM_NAME, [{ code: 'PE' }, { code: 'PC' }]),
      group('mr-15m', 'MR', 'XML', '15M', '0 0/15 * * * ?', PATH_MR, MR_NAME, mrObjects),
    ],
    logs: { mode: 'custom', cron: '30 5 16 * * ?', period: '24H', compression: 'gz' },
  },
  {
    code: 'S0006',
    name: '无 MR 的 CM/PM 场景',
    enabled: false,
    groups: [
      group('cm-daily', 'CM', 'XML', '24H', '30 1 0 * * ?', PATH_CM, CM_NAME, cmObjects),
      group('pm-15m', 'PM', 'CSV', '15M', '0 5/15 * * * ?', PATH_PM, PM_NAME, [{ code: 'PC' }]),
    ],
  },
  {
    code: 'S0007',
    name: 'CM CSV + COMS + PM 60M',
    enabled: false,
    groups: [
      group('cm-daily-csv', 'CM', 'CSV', '24H', '0 0 17 * * ?', PATH_CM, CM_NAME, cmObjectsWithComs),
      group('pm-60m', 'PM', 'CSV', '60M', '0 20 * * * ?', PATH_PM, PM_NAME, [{ code: 'PC' }]),
    ],
  },
  {
    code: 'S0008',
    name: 'CM CSV + COMS + 固定日志',
    enabled: false,
    groups: [
      group('cm-daily-csv', 'CM', 'CSV', '24H', '30 1 0 * * ?', PATH_CM, CM_NAME, cmObjectsWithComs),
      group('pm-15m', 'PM', 'CSV', '15M', '0 5/15 * * * ?', PATH_PM, PM_NAME, [{ code: 'PC' }]),
      group('mr-15m', 'MR', 'XML', '15M', '0 0/15 * * * ?', PATH_MR, MR_NAME, mrObjects),
    ],
    logs: { mode: 'fix', cron: '30 5 16 * * ?', period: '24H', compression: 'gz' },
  },
  {
    code: 'S0009',
    name: 'PC 60M + MR',
    enabled: false,
    groups: [
      group('cm-daily', 'CM', 'XML', '24H', '30 1 0 * * ?', PATH_CM, CM_NAME, cmObjects),
      group('pm-pc-60m', 'PM', 'CSV', '60M', '0 0 1 * * ?', PATH_PM, PM_NAME, [{ code: 'PC' }]),
      group('mr-15m', 'MR', 'XML', '15M', '0 0/15 * * * ?', PATH_MR, MR_NAME, mrObjects),
    ],
  },
  {
    code: 'S0010',
    name: '按对象分目录输出',
    enabled: false,
    groups: [
      group('cm-daily-by-object', 'CM', 'XML', '24H', '30 1 0 * * ?', `${PATH_CM}#Object#/`, CM_NAME, cmObjects),
      group('pm-15m-by-object', 'PM', 'CSV', '15M', '0 5/15 * * * ?', `${PATH_PM}#Object#/`, PM_NAME, [{ code: 'PC' }]),
      group('mr-15m-by-object', 'MR', 'XML', '15M', '0 0/15 * * * ?', `${PATH_MR}#Object#/`, MR_NAME, mrObjects),
    ],
  },
  {
    code: 'S0011',
    name: 'CM + PM PC + MR',
    enabled: false,
    groups: [
      group('cm-daily', 'CM', 'XML', '24H', '30 1 0 * * ?', PATH_CM, CM_NAME, cmObjects),
      group('pm-15m', 'PM', 'CSV', '15M', '0 5/15 * * * ?', PATH_PM, PM_NAME, [{ code: 'PC' }]),
      group('mr-15m', 'MR', 'XML', '15M', '0 0/15 * * * ?', PATH_MR, MR_NAME, mrObjects),
    ],
  },
  {
    code: 'S0012',
    name: 'ENB + GNB 双制式',
    enabled: false,
    groups: [
      group('cm-daily-lte', 'CM', 'XML', '24H', '30 1 0 * * ?', PATH_CM, CM_NAME, cmObjects),
      group('cm-daily-gnb', 'CM', 'XML', '24H', '30 3 0 * * ?', `${PATH_CM}GNB/`, CM_NAME, cmGnbObjects),
      group('pm-15m-lte', 'PM', 'CSV', '15M', '0 5/15 * * * ?', PATH_PM, PM_NAME, [{ code: 'PC' }]),
      group('pm-pc-15m-gnb', 'PM', 'CSV', '15M', '0 8/15 * * * ?', `${PATH_PM}GNB/`, PM_NAME, [{ code: 'PC', tech: 'GNB', profile: 'pm.pc.gnb.csv.v1' }]),
      group('mr-15m', 'MR', 'XML', '15M', '0 0/15 * * * ?', PATH_MR, MR_NAME, mrObjects),
    ],
  },
  {
    code: 'S0013',
    name: 'pmresult 60M 多制式',
    enabled: false,
    groups: [
      group('cm-daily', 'CM', 'XML', '24H', '30 1 0 * * ?', PATH_CM, CM_NAME, cmObjects),
      group('pm-pc-60m-lte', 'PM', 'CSV', '60M', '0 25 * * * ?', PATH_PM, 'pmresult_152XXX_#DataPeriod#_#PeriodStartTime#_#PeriodEndTime#[-#FileID#]', [{ code: 'PC', profile: 'pm.pc.pmresult.csv.v1' }]),
      group('pm-pc-60m-gsm', 'PM', 'CSV', '60M', '0 20 * * * ?', PATH_PM, 'pmresult_#LocalHost#_#DataPeriod#_#PeriodStartTime#_#PeriodEndTime#[-#FileID#]', [{ code: 'PC', tech: 'GSM', profile: 'pm.pc.gsm.pmresult.csv.v1' }]),
      group('pm-pc-60m-gnb', 'PM', 'CSV', '60M', '0 30 * * * ?', `${PATH_PM}GNB/`, 'pmresult_XXXXXX_#DataPeriod#_#PeriodStartTime#_#PeriodEndTime#[-#FileID#]', [{ code: 'PC', tech: 'GNB', profile: 'pm.pc.gnb.csv.v1' }]),
      group('mr-15m', 'MR', 'XML', '15M', '0 0/15 * * * ?', PATH_MR, MR_NAME, mrObjects),
    ],
  },
  {
    code: 'S0014',
    name: 'CM + PM PC + MR',
    enabled: false,
    groups: [
      group('cm-daily', 'CM', 'XML', '24H', '30 1 0 * * ?', PATH_CM, CM_NAME, cmObjects),
      group('pm-15m', 'PM', 'CSV', '15M', '0 5/15 * * * ?', PATH_PM, PM_NAME, [{ code: 'PC' }]),
      group('mr-15m', 'MR', 'XML', '15M', '0 0/15 * * * ?', PATH_MR, MR_NAME, mrObjects),
    ],
  },
  {
    code: 'S0015',
    name: 'CM + PM PE/PC + MR',
    enabled: false,
    groups: [
      group('cm-daily', 'CM', 'XML', '24H', '30 1 0 * * ?', PATH_CM, CM_NAME, cmObjects),
      group('pm-15m', 'PM', 'CSV', '15M', '0 5/15 * * * ?', PATH_PM, PM_NAME, [{ code: 'PE' }, { code: 'PC' }]),
      group('mr-15m', 'MR', 'XML', '15M', '0 0/15 * * * ?', PATH_MR, MR_NAME, mrObjects),
    ],
  },
  {
    code: 'S0016',
    name: 'ENB + GSM PM',
    enabled: false,
    groups: [
      group('cm-daily', 'CM', 'XML', '24H', '30 1 0 * * ?', PATH_CM, CM_NAME, cmObjects),
      group('pm-15m-lte', 'PM', 'CSV', '15M', '0 5/15 * * * ?', PATH_PM, PM_NAME, [{ code: 'PC' }]),
      group('pm-pc-15m-gsm', 'PM', 'CSV', '15M', '0 5/15 * * * ?', `${PATH_PM}gsm/`, 'pmresult_#LocalHost#_#DataPeriod#_#PeriodStartTime#_#PeriodEndTime#[-#FileID#]', [{ code: 'PC', tech: 'GSM', profile: 'pm.pc.gsm.pmresult.csv.v1' }]),
      group('mr-15m', 'MR', 'XML', '15M', '0 0/15 * * * ?', PATH_MR, MR_NAME, mrObjects),
    ],
  },
  {
    code: 'S0017',
    name: 'CM + PM PC + MR',
    enabled: false,
    groups: [
      group('cm-daily', 'CM', 'XML', '24H', '30 1 0 * * ?', PATH_CM, CM_NAME, cmObjects),
      group('pm-15m', 'PM', 'CSV', '15M', '0 5/15 * * * ?', PATH_PM, PM_NAME, [{ code: 'PC' }]),
      group('mr-15m', 'MR', 'XML', '15M', '0 0/15 * * * ?', PATH_MR, MR_NAME, mrObjects),
    ],
  },
];

const scenarioMeta: Record<string, Pick<ScenarioRow, 'vendor' | 'scenarioName' | 'scenarioNameEn' | 'description' | 'flags'>> = {
  S0001: {
    vendor: 'Baicells',
    scenarioName: '标准场景',
    scenarioNameEn: 'Standard',
    description: '标准基线：CM 日文件、PM PC 15 分钟文件、MR 15 分钟文件。',
    flags: ['基线配置'],
  },
  S0002: {
    vendor: 'Baicells',
    scenarioName: '电信场景',
    scenarioNameEn: 'Telecom',
    description: 'Telecom 系列：在基线基础上启用 PE/PC 双性能对象，CSV 分隔符为竖线。',
    flags: ['PE/PC', 'csv |'],
  },
  S0003: {
    vendor: 'Baicells',
    scenarioName: '上海电信',
    scenarioNameEn: 'SH-Tele',
    description: 'SH-Tele：PE/PC 性能并行输出，CM 与 MR 保持标准周期。',
    flags: ['PE/PC', 'csv |'],
  },
  S0004: {
    vendor: 'Baicells',
    scenarioName: '陕西电信',
    scenarioNameEn: 'SN-Tele',
    description: 'SN-Tele：PE/PC 双性能对象，北向告警语言按中文策略处理。',
    flags: ['PE/PC', '告警中文', 'csv |'],
  },
  S0005: {
    vendor: 'Baicells',
    scenarioName: '江苏电信',
    scenarioNameEn: 'JS-Tele',
    description: 'JS-Tele：PE/PC 性能采用延迟触发，并启用 custom 日志外送。',
    flags: ['PE/PC', '日志 type1'],
  },
  S0006: {
    vendor: 'Baicells',
    scenarioName: '泰国 True',
    scenarioNameEn: 'Thai-True',
    description: 'Thai-True：精简 CM/PM 场景，不启用 MR；开启附加信息字段。',
    flags: ['无 MR', '附加信息', 'csv |'],
  },
  S0007: {
    vendor: 'Baicells',
    scenarioName: '泰国 AIS',
    scenarioNameEn: 'Thai-AIS',
    description: 'Thai-AIS：CM CSV + COMS，PM PC 按 60 分钟窗口输出，告警通道策略独立。',
    flags: ['CM CSV', 'COMS', 'PM 60M', 'SNMP 实时'],
  },
  S0008: {
    vendor: 'Baicells',
    scenarioName: 'V1 场景',
    scenarioNameEn: 'V1',
    description: 'V1：CM CSV + COMS，保留 PM/MR，并启用 fix 日志外送与 yyyyMMdd 目录。',
    flags: ['CM CSV', 'COMS', '日志 type2'],
  },
  S0009: {
    vendor: 'Baicells',
    scenarioName: '老挝电信',
    scenarioNameEn: 'Laos-Tele',
    description: 'Laos-Tele：标准 CM/MR，PM 增加 60 分钟 PC 输出；告警清除严重级别按 0 处理。',
    flags: ['PM 60M', '附加信息'],
  },
  S0010: {
    vendor: 'Baicells',
    scenarioName: '上海联通',
    scenarioNameEn: 'SHUcom',
    description: 'SHUcom：按对象分目录输出，开启 manifest 清单，PM 最后周期归档到当天目录。',
    flags: ['对象分目录', 'manifest', '联通 socket'],
  },
  S0011: {
    vendor: 'Baicells',
    scenarioName: 'ISAT MTN',
    scenarioNameEn: 'ISAT-NBI-MTN',
    description: 'ISAT-NBI-MTN：标准 CM 日文件、PM PC 15 分钟文件、MR 15 分钟文件。',
    flags: ['标准周期'],
  },
  S0012: {
    vendor: 'Baicells',
    scenarioName: '陕西移动',
    scenarioNameEn: 'Shaanxi Mobile',
    description: '陕西移动：ENB + GNB 双制式，CM/PM 按制式拆分输出。',
    flags: ['ENB/GNB', '双制式'],
  },
  S0013: {
    vendor: 'Baicells',
    scenarioName: 'ZED 场景',
    scenarioNameEn: 'ZED',
    description: 'ZED：pmresult 命名，ENB/GSM/GNB 性能文件按 60 分钟窗口输出。',
    flags: ['ENB/GSM/GNB', 'PM 60M', 'pmresult'],
  },
  S0014: {
    vendor: 'Baicells',
    scenarioName: '印尼 Telkomsel',
    scenarioNameEn: 'YinNi_Telkomsel_MNO',
    description: 'YinNi_Telkomsel_MNO：标准 CM 日文件、PM PC 15 分钟文件、MR 15 分钟文件。',
    flags: ['标准周期'],
  },
  S0015: {
    vendor: 'Baicells',
    scenarioName: '菲律宾 DITO',
    scenarioNameEn: 'FeiLvBin-DITO',
    description: 'FeiLvBin-DITO：PE/PC 性能并行输出，CSV 分隔符为竖线。',
    flags: ['PE/PC', 'csv |'],
  },
  S0016: {
    vendor: 'Baicells',
    scenarioName: 'MTN 场景',
    scenarioNameEn: 'MTN',
    description: 'MTN：ENB + GSM 性能混合输出，GSM 使用 pmresult 命名。',
    flags: ['ENB/GSM', 'show_site_id'],
  },
  S0017: {
    vendor: 'Baicells',
    scenarioName: '黑龙江场景',
    scenarioNameEn: 'HLongjianng',
    description: 'HLongjianng：标准 CM 日文件、PM PC 15 分钟文件、MR 15 分钟文件。',
    flags: ['标准周期'],
  },
};

const scenarioRows: ScenarioRow[] = rawScenarioRows.map((row) => ({
  ...row,
  ...scenarioMeta[row.code],
}));

const cmFieldsByObject: Record<string, FieldDefinition[]> = {
  CP: [
    ['dn', 'device.serial_number', 'devices.serial_number', 'string', 'quote', 'DN/设备序列号'],
    ['related_enb_dn', 'device.serial_number', 'devices.serial_number', 'string', 'quote', '关联 eNB DN'],
    ['related_enb_id', 'device_info.enb_id', 'device_info.enb_id', 'string', 'quote', '关联 eNB ID'],
    ['related_enb_userlabel', 'device.site_name', 'devices.site_name / device_info.device_name', 'string', 'quote', '关联 eNB 名称'],
    ['cel_id', 'device_info.cell_id', 'device_info.cell_id', 'string', 'quote', '小区 ID'],
    ['cel_userlabel', 'device_info.device_name', 'device_info.device_name / devices.site_name', 'string', 'quote', '小区名称'],
    ['referenceSignalPower', 'device_info.transmit_power', 'device_info.transmit_power', 'number', 'number', '参考信号功率'],
    ['Serial Number', 'device.serial_number', 'devices.serial_number', 'string', 'quote', '设备序列号'],
    ['Manufacturer', 'device.manufacturer', 'devices.manufacturer', 'string', 'quote', '厂家'],
    ['Model Name', 'device.model_name', 'devices.model_name', 'string', 'quote', '型号'],
    ['Product Type', 'device.product_class', 'devices.product_class', 'string', 'quote', '产品类型'],
    ['Product Name', 'product.name', 'products.name / devices.product_class', 'string', 'quote', '产品名称'],
    ['Hardware Version', 'device_info.hardware_version', 'device_info.hardware_version', 'string', 'quote', '硬件版本'],
    ['Software Version', 'device.firmware_version', 'devices.firmware_version', 'string', 'quote', '软件版本'],
    ['Device Group', 'device_groups.name', 'device_groups / device_group_members', 'string', 'quote', '设备组'],
    ['Site Name', 'device.site_name', 'devices.site_name / device_info.device_name', 'string', 'quote', '站点名称'],
    ['First Online Time', 'device_info.first_online_time', 'device_info.first_online_time', 'datetime', 'yyyy-MM-dd HH:mm:ss', '首次上线时间'],
    ['Last Period Time', 'device.last_inform_at', 'devices.last_inform_at', 'datetime', 'yyyy-MM-dd HH:mm:ss', '最近上报时间'],
  ].map(([outputAlias, systemField, source, dataType, renderer, cnName]) => ({ outputAlias, systemField, source, dataType, renderer, cnName })),
  EP: [
    ['dn', 'device.serial_number', 'devices.serial_number', 'string', 'quote', 'DN/设备序列号'],
    ['enb_id', 'device_info.enb_id', 'device_info.enb_id', 'string', 'quote', 'eNB ID'],
    ['enb_userlabel', 'device.site_name', 'devices.site_name / device_info.device_name', 'string', 'quote', 'eNB 名称'],
    ['IP Address', 'device.ip_address', 'devices.ip_address', 'string', 'quote', 'IP 地址'],
    ['MAC Address', 'device_info.mac', 'device_info.mac', 'string', 'quote', 'MAC 地址'],
    ['IPsec Address', 'device_info.ipsec_addr', 'device_info.ipsec_addr', 'string', 'quote', 'IPsec 地址'],
    ['PLMN', 'device_info.plmn', 'device_info.plmn', 'string', 'preserve text', 'PLMN'],
    ['TAC', 'device_info.tac', 'device_info.tac', 'string', 'quote', 'TAC'],
    ['MME Status', 'device_info.mme_status', 'device_info.mme_status', 'string', 'enum', 'MME 状态'],
    ['AMF Status', 'device_info.mme_status', 'device_info.mme_status as amf_status', 'string', 'enum', 'AMF 状态'],
    ['Sync Status', 'device_info.sync_status', 'device_info.sync_status', 'string', 'enum', '同步状态'],
    ['Longitude', 'device.longitude', 'devices.longitude', 'number', 'number', '经度'],
    ['Latitude', 'device.latitude', 'devices.latitude', 'number', 'number', '纬度'],
    ['Satellites', 'device_info.gps_satellites', 'device_info.gps_satellites', 'number', 'number', '卫星数'],
    ['Height(m)', 'device_info.gps_height', 'device_info.gps_height', 'number', 'number', '高度'],
  ].map(([outputAlias, systemField, source, dataType, renderer, cnName]) => ({ outputAlias, systemField, source, dataType, renderer, cnName })),
  CC: [
    ['dn', 'device_info.eci', 'device_info.eci', 'string', 'preserve text', '小区 DN/ECI'],
    ['related_enb_dn', 'device.serial_number', 'devices.serial_number', 'string', 'quote', '关联 eNB DN'],
    ['related_enb_id', 'device_info.enb_id', 'device_info.enb_id', 'string', 'quote', '关联 eNB ID'],
    ['related_enb_userlabel', 'device.site_name', 'devices.site_name / device_info.device_name', 'string', 'quote', '关联 eNB 名称'],
    ['cel_id', 'device_info.cell_id', 'device_info.cell_id', 'string', 'quote', '小区 ID'],
    ['userlabel', 'device_info.device_name', 'device_info.device_name / devices.site_name', 'string', 'quote', '小区名称'],
    ['pci', 'device_info.pci', 'device_info.pci', 'string', 'preserve text', 'PCI'],
    ['freq_mode', 'device_info.network_model', 'device_info.network_model -> FDD=2/TDD=4', 'string', 'enum', '频分/时分模式'],
    ['bandIndicator', 'device_info.band', 'device_info.band', 'string', 'quote', '频段指示'],
    ['tac', 'device_info.tac', 'device_info.tac', 'string', 'quote', 'TAC'],
    ['zc_idx', 'device_info.root_index', 'device_info.root_index', 'number', 'number', '根序列索引'],
    ['freq_pointno_ul', 'device_info.ul_earfcn', 'device_info.ul_earfcn', 'number', 'preserve text', '上行频点号'],
    ['freq_pointno_dl', 'device_info.freq_point', 'device_info.freq_point', 'number', 'preserve text', '下行频点号'],
    ['bandwidth_ul', 'device_info.bandwidth', 'device_info.bandwidth', 'number', 'number', '上行带宽'],
    ['bandwidth_dl', 'device_info.bandwidth', 'device_info.bandwidth', 'number', 'number', '下行带宽'],
    ['td_sfassignment', 'device_info.subframe_assignment', 'device_info.subframe_assignment', 'string', 'quote', 'TDD 子帧配比'],
    ['td_specialsfpatterns', 'device_info.special_subframe', 'device_info.special_subframe', 'string', 'quote', 'TDD 特殊子帧配比'],
    ['eNB ID', 'device_info.enb_id', 'device_info.enb_id', 'string', 'quote', 'eNB ID'],
    ['ECI', 'device_info.eci', 'device_info.eci', 'string', 'preserve text', 'ECI'],
    ['PCI', 'device_info.pci', 'device_info.pci', 'string', 'preserve text', 'PCI'],
    ['Cell ID', 'device_info.cell_id', 'device_info.cell_id', 'string', 'quote', '小区 ID'],
    ['Earfcn', 'device_info.freq_point', 'device_info.freq_point', 'number', 'preserve text', '频点'],
    ['UL Earfcn', 'device_info.ul_earfcn', 'device_info.ul_earfcn', 'number', 'preserve text', '上行频点'],
    ['Band', 'device_info.band', 'device_info.band', 'string', 'quote', '频段'],
    ['Bandwidth', 'device_info.bandwidth', 'device_info.bandwidth', 'number', 'number', '带宽'],
    ['Duplex Mode', 'device_info.network_model', 'device_info.network_model', 'string', 'quote', '双工模式'],
    ['Transmit Power', 'device_info.transmit_power', 'device_info.transmit_power', 'number', 'number', '发射功率'],
    ['Subframe Assignment', 'device_info.subframe_assignment', 'device_info.subframe_assignment', 'string', 'quote', '子帧配比'],
    ['Special Subframe Patterns', 'device_info.special_subframe', 'device_info.special_subframe', 'string', 'quote', '特殊子帧配比'],
  ].map(([outputAlias, systemField, source, dataType, renderer, cnName]) => ({ outputAlias, systemField, source, dataType, renderer, cnName })),
  CE: [
    ['dn', 'device.serial_number', 'devices.serial_number', 'string', 'quote', 'DN/设备序列号'],
    ['enb_id', 'device_info.enb_id', 'device_info.enb_id', 'string', 'quote', 'eNB ID'],
    ['userlabel', 'device.site_name', 'devices.site_name / device_info.device_name', 'string', 'quote', '设备名称'],
    ['enb_model', 'device.model_name', 'devices.model_name / devices.product_class', 'string', 'quote', 'eNB 型号'],
    ['ip_address', 'device.ip_address', 'devices.ip_address', 'string', 'quote', 'IP 地址'],
    ['software_version', 'device.firmware_version', 'devices.firmware_version', 'string', 'quote', '软件版本'],
    ['freq_mode', 'device_info.network_model', 'device_info.network_model -> FDD=2/TDD=4', 'string', 'enum', '频分/时分模式'],
    ['cel_num', 'device_info.num_of_cells', 'device_info.num_of_cells', 'number', 'number', '小区数量'],
    ['serialid', 'device.serial_number', 'devices.serial_number', 'string', 'quote', '设备序列号'],
    ['Cell Status', 'device_info.op_state', 'device_info.op_state', 'string', 'enum', '小区状态'],
    ['Online Status', 'device.is_online', 'devices.is_online', 'bool', 'enum', '在线状态'],
    ['Device Status', 'device.lifecycle_state', 'devices.lifecycle_state + devices.is_online', 'string', 'enum', '设备状态'],
    ['RF Status', 'device_info.rf_status', 'device_info.rf_status', 'string', 'enum', '射频状态'],
    ['KPI Report Status', 'device_info.kpi_status', 'device_info.kpi_status', 'string', 'enum', 'KPI 上报状态'],
    ['UE Count', 'device_info.ue_count', 'device_info.ue_count', 'number', 'number', 'UE 数'],
    ['Alarms', 'device_info.active_alarm_count', 'device_info.active_alarm_count', 'number', 'number', '活动告警数'],
    ['Highest Alarm Severity', 'device_info.highest_alarm_severity', 'device_info.highest_alarm_severity', 'string', 'enum', '最高告警级别'],
    ['System Uptime', 'device_info.run_time', 'device_info.run_time', 'number', 'number', '运行时长'],
    ['Accumulated Online Time(s)', 'device_info.cumulative_online_duration', 'device_info.cumulative_online_duration', 'number', 'number', '累计在线时长'],
  ].map(([outputAlias, systemField, source, dataType, renderer, cnName]) => ({ outputAlias, systemField, source, dataType, renderer, cnName })),
  COMS: [
    ['Serial Number', 'device.serial_number', 'devices.serial_number', 'string', 'quote', '设备序列号'],
    ['Vendor', 'device.manufacturer', 'devices.manufacturer', 'string', 'quote', '厂家'],
    ['Model', 'device.model_name', 'devices.model_name', 'string', 'quote', '型号'],
    ['ENODEB_ID', 'device_info.enb_id', 'device_info.enb_id', 'string', 'quote', 'eNB ID'],
    ['ENODEB_NAME', 'device.site_name', 'devices.site_name / device_info.device_name', 'string', 'quote', 'eNB 名称'],
    ['Cell ID', 'device_info.cell_id', 'device_info.cell_id', 'string', 'quote', '小区 ID'],
    ['CELL_NAME', 'device_info.device_name', 'device_info.device_name / devices.site_name', 'string', 'quote', '小区名称'],
    ['CELL_STATUS', 'device_info.op_state', 'device_info.op_state', 'string', 'enum', '小区状态'],
    ['ECI', 'device_info.eci', 'device_info.eci', 'string', 'preserve text', 'ECI'],
    ['TAC', 'device_info.tac', 'device_info.tac', 'string', 'quote', 'TAC'],
    ['PCI', 'device_info.pci', 'device_info.pci', 'string', 'preserve text', 'PCI'],
    ['UL_EARFCN', 'device_info.ul_earfcn', 'device_info.ul_earfcn', 'number', 'preserve text', '上行频点'],
    ['DL_EARFCN', 'device_info.freq_point', 'device_info.freq_point', 'number', 'preserve text', '下行频点'],
    ['BAND', 'device_info.band', 'device_info.band', 'string', 'quote', '频段'],
    ['BANDWIDTH', 'device_info.bandwidth', 'device_info.bandwidth', 'number', 'number', '带宽'],
    ['RS_POWER', 'device_info.transmit_power', 'device_info.transmit_power', 'number', 'number', '参考信号功率'],
    ['OMC-R', 'system.omc_r', 'northbound_profile.omc_r', 'string', 'quote', 'OMC-R'],
    ['Province', 'system.province', 'northbound_profile.province', 'string', 'quote', '省份'],
    ['LocalHost', 'runtime.local_host', 'runtime.local_host', 'string', 'quote', '本机标识'],
    ['DataVersion', 'runtime.data_version', 'northbound_profile.data_version', 'string', 'quote', '数据版本'],
    ['DateTime', 'task.window_end', 'northbound_export_runs.window_end', 'datetime', 'yyyyMMddHHmmss', '文件时间'],
  ].map(([outputAlias, systemField, source, dataType, renderer, cnName]) => ({ outputAlias, systemField, source, dataType, renderer, cnName })),
};

const mrFieldDefinitions: FieldDefinition[] = [
  ['file_type', 'mr.file_type', 'mr_files.file_type', 'string', 'quote', 'MR 类型'],
  ['device_sn', 'mr.device_sn', 'mr_files.device_sn', 'string', 'quote', '设备 SN'],
  ['object_key', 'mr.object_key', 'mr_files.object_key', 'string', 'quote', '对象标识'],
  ['collect_time', 'mr.collect_time', 'mr_files.collect_time', 'datetime', 'yyyy-MM-dd HH:mm:ss', '采集时间'],
  ['checksum', 'mr.checksum', 'mr_files.checksum', 'string', 'quote', '校验和'],
].map(([outputAlias, systemField, source, dataType, renderer, cnName]) => ({ outputAlias, systemField, source, dataType, renderer, cnName }));

const mrFieldsByObject: Record<string, FieldDefinition[]> = {
  MRO: mrFieldDefinitions,
  MRE: mrFieldDefinitions,
  MRS: mrFieldDefinitions,
};

const logFieldDefinitions: FieldDefinition[] = [
  ['log_time', 'log.log_time', 'sys_login_logs.login_at / sys_oper_logs.created_at', 'datetime', 'yyyy-MM-dd HH:mm:ss', '日志时间'],
  ['sys_source_name', 'log.sys_source_name', '系统固定值 baicells omc', 'string', 'quote', '系统来源'],
  ['account_name', 'log.account_name', 'sys_login_logs.username', 'string', 'quote', '账号名称'],
  ['terminal_name', 'log.terminal_name', '浏览器 / 客户端标识', 'string', 'quote', '终端名称'],
  ['terminal_ip', 'log.terminal_ip', 'sys_login_logs.ip_address / sys_oper_logs.ip_address', 'string', 'quote', '终端 IP'],
  ['log_result', 'log.result', '日志状态', 'string', 'enum', '日志结果'],
  ['log_start_time', 'log.log_start_time', 'sys_login_logs.login_at', 'datetime', 'yyyy-MM-dd HH:mm:ss', '日志开始时间'],
  ['log_end_time', 'log.log_end_time', 'sys_login_logs.login_at', 'datetime', 'yyyy-MM-dd HH:mm:ss', '日志结束时间'],
  ['main_name', 'log.main_name', 'sys_oper_logs.username', 'string', 'quote', '主账号'],
  ['sub_account', 'log.sub_account', '空值占位', 'string', 'quote', '子账号'],
  ['asset_name', 'log.asset_name', '空值占位', 'string', 'quote', '资产名称'],
  ['asset_ip', 'log.asset_ip', '空值占位', 'string', 'quote', '资产 IP'],
  ['asset_port', 'log.asset_port', '空值占位', 'string', 'quote', '资产端口'],
  ['asset_attribute', 'log.asset_attribute', '空值占位', 'string', 'quote', '资产属性'],
  ['log_data', 'log.detail', 'sys_oper_logs.action / detail / status', 'string', 'quote', '日志数据'],
  ['ID', 'log.id', 'sys_login_logs.id', 'string', 'quote', 'ID'],
  ['User Name', 'log.user_name', 'sys_login_logs.username / sys_oper_logs.username', 'string', 'quote', '用户名称'],
  ['IP Address', 'log.ip_address', 'sys_login_logs.ip_address / sys_oper_logs.ip_address', 'string', 'quote', 'IP 地址'],
  ['Log Name', 'log.log_name', '登录/操作名称', 'string', 'quote', '日志名称'],
  ['Record Detail', 'log.detail', '日志详情', 'string', 'quote', '记录详情'],
  ['Results', 'log.result_text', '日志状态', 'string', 'enum', '结果'],
  ['Failure Reason', 'log.failure_reason', '失败原因', 'string', 'quote', '失败原因'],
  ['Time', 'log.login_time', 'sys_login_logs.login_at', 'datetime', 'yyyy-MM-dd HH:mm:ss', '时间'],
  ['Op Start Time', 'log.op_start_time', 'sys_oper_logs.created_at', 'datetime', 'yyyy-MM-dd HH:mm:ss', '操作开始时间'],
  ['Op End Time', 'log.op_end_time', 'sys_oper_logs.created_at + cost_ms', 'datetime', 'yyyy-MM-dd HH:mm:ss', '操作结束时间'],
].map(([outputAlias, systemField, source, dataType, renderer, cnName]) => ({ outputAlias, systemField, source, dataType, renderer, cnName }));

const logFieldsByObject: Record<string, FieldDefinition[]> = {
  login: logFieldDefinitions.filter((field) => ['log_time', 'sys_source_name', 'account_name', 'terminal_name', 'terminal_ip', 'log_result', 'log_start_time', 'log_end_time'].includes(field.outputAlias)),
  operation: logFieldDefinitions.filter((field) => ['log_time', 'sys_source_name', 'terminal_name', 'terminal_ip', 'main_name', 'sub_account', 'asset_name', 'asset_ip', 'asset_port', 'asset_attribute', 'log_data'].includes(field.outputAlias)),
  login_fix: logFieldDefinitions.filter((field) => ['ID', 'User Name', 'IP Address', 'Log Name', 'Record Detail', 'Results', 'Failure Reason', 'Time'].includes(field.outputAlias)),
  operation_fix: logFieldDefinitions.filter((field) => ['User Name', 'IP Address', 'Log Name', 'Record Detail', 'Results', 'Failure Reason', 'Op Start Time', 'Op End Time'].includes(field.outputAlias)),
};

const extraFieldDefinitionsByDomainObject: Partial<Record<Domain, Record<string, FieldDefinition[]>>> = {
  CM: {
    CP: [
      ['Device Alias', 'device.site_name', 'devices.site_name / device_info.device_name', 'string', 'quote', '设备别名'],
      ['Management Status', 'device.lifecycle_state', 'devices.lifecycle_state', 'string', 'enum', '管理状态'],
      ['Create Time', 'device.created_at', 'devices.created_at', 'datetime', 'yyyy-MM-dd HH:mm:ss', '创建时间'],
    ].map(([outputAlias, systemField, source, dataType, renderer, cnName]) => ({ outputAlias, systemField, source, dataType, renderer, cnName })),
    EP: [],
    CC: [],
    CE: [],
    COMS: [
      ['Export Batch ID', 'runtime.export_batch_id', 'northbound_export_runs.id', 'string', 'quote', '导出批次'],
      ['OMC Name', 'system.omc_name', 'northbound_profile.omc_name', 'string', 'quote', 'OMC 名称'],
      ['REGION', 'system.province', 'northbound_profile.province', 'string', 'quote', '区域'],
      ['DATE_TIME', 'task.window_end', 'northbound_export_runs.window_end', 'datetime', 'yyyyMMddHHmmss', '文件时间'],
      ['DUPLEXING', 'device_info.network_model', 'device_info.network_model', 'string', 'quote', '双工模式'],
      ['MAXTXPOWER', 'device_info.transmit_power', 'device_info.transmit_power', 'number', 'number', '最大发射功率'],
    ].map(([outputAlias, systemField, source, dataType, renderer, cnName]) => ({ outputAlias, systemField, source, dataType, renderer, cnName })),
  },
  MR: {
    MRO: [
      ['file_size', 'mr.file_size', 'mr_files.file_size', 'number', 'number', '文件大小'],
      ['storage_path', 'mr.storage_path', 'mr_files.storage_path', 'string', 'quote', '存储路径'],
    ].map(([outputAlias, systemField, source, dataType, renderer, cnName]) => ({ outputAlias, systemField, source, dataType, renderer, cnName })),
    MRE: [
      ['file_size', 'mr.file_size', 'mr_files.file_size', 'number', 'number', '文件大小'],
      ['storage_path', 'mr.storage_path', 'mr_files.storage_path', 'string', 'quote', '存储路径'],
    ].map(([outputAlias, systemField, source, dataType, renderer, cnName]) => ({ outputAlias, systemField, source, dataType, renderer, cnName })),
    MRS: [
      ['file_size', 'mr.file_size', 'mr_files.file_size', 'number', 'number', '文件大小'],
      ['storage_path', 'mr.storage_path', 'mr_files.storage_path', 'string', 'quote', '存储路径'],
    ].map(([outputAlias, systemField, source, dataType, renderer, cnName]) => ({ outputAlias, systemField, source, dataType, renderer, cnName })),
  },
  LOG: {
    login: [
      ['browser', 'log.browser', 'sys_login_logs.browser', 'string', 'quote', '浏览器'],
      ['os', 'log.os', 'sys_login_logs.os', 'string', 'quote', '操作系统'],
    ].map(([outputAlias, systemField, source, dataType, renderer, cnName]) => ({ outputAlias, systemField, source, dataType, renderer, cnName })),
    operation: [
      ['action', 'log.action', 'sys_oper_logs.action', 'string', 'quote', '操作动作'],
      ['resource', 'log.resource', 'sys_oper_logs.target', 'string', 'quote', '资源对象'],
      ['user_agent', 'log.user_agent', 'sys_oper_logs.user_agent', 'string', 'quote', '客户端标识'],
    ].map(([outputAlias, systemField, source, dataType, renderer, cnName]) => ({ outputAlias, systemField, source, dataType, renderer, cnName })),
    login_fix: [
      ['browser', 'log.browser', 'sys_login_logs.browser', 'string', 'quote', '浏览器'],
      ['os', 'log.os', 'sys_login_logs.os', 'string', 'quote', '操作系统'],
    ].map(([outputAlias, systemField, source, dataType, renderer, cnName]) => ({ outputAlias, systemField, source, dataType, renderer, cnName })),
    operation_fix: [
      ['action', 'log.action', 'sys_oper_logs.action', 'string', 'quote', '操作动作'],
      ['resource', 'log.resource', 'sys_oper_logs.target', 'string', 'quote', '资源对象'],
      ['user_agent', 'log.user_agent', 'sys_oper_logs.user_agent', 'string', 'quote', '客户端标识'],
    ].map(([outputAlias, systemField, source, dataType, renderer, cnName]) => ({ outputAlias, systemField, source, dataType, renderer, cnName })),
  },
  INVENTORY: {
    eNB: [
      ['Northbound Label', 'inventory.enb.label', 'devices.name / devices.site_name', 'string', 'quote', '北向标签'],
      ['Snapshot Time', 'inventory.enb.snapshot_time', 'northbound_export_runs.window_end', 'datetime', 'yyyy-MM-dd HH:mm:ss', '快照时间'],
    ].map(([outputAlias, systemField, source, dataType, renderer, cnName]) => ({ outputAlias, systemField, source, dataType, renderer, cnName })),
    gNB: [
      ['Northbound Label', 'inventory.gnb.label', 'devices.name / devices.site_name', 'string', 'quote', '北向标签'],
      ['Snapshot Time', 'inventory.gnb.snapshot_time', 'northbound_export_runs.window_end', 'datetime', 'yyyy-MM-dd HH:mm:ss', '快照时间'],
    ].map(([outputAlias, systemField, source, dataType, renderer, cnName]) => ({ outputAlias, systemField, source, dataType, renderer, cnName })),
    GSM: [
      ['Northbound Label', 'inventory.gsm.label', 'devices.name / devices.site_name', 'string', 'quote', '北向标签'],
      ['Snapshot Time', 'inventory.gsm.snapshot_time', 'northbound_export_runs.window_end', 'datetime', 'yyyy-MM-dd HH:mm:ss', '快照时间'],
    ].map(([outputAlias, systemField, source, dataType, renderer, cnName]) => ({ outputAlias, systemField, source, dataType, renderer, cnName })),
    OMC: [
      ['OMC Name', 'inventory.omc.name', 'system_settings.omc_name', 'string', 'quote', 'OMC 名称'],
      ['Snapshot Time', 'inventory.omc.snapshot_time', 'northbound_export_runs.window_end', 'datetime', 'yyyy-MM-dd HH:mm:ss', '快照时间'],
    ].map(([outputAlias, systemField, source, dataType, renderer, cnName]) => ({ outputAlias, systemField, source, dataType, renderer, cnName })),
  },
};

const stationInventoryFieldDefinitions = [
  ['Serial Number', 'device.serial_number', 'devices.serial_number', 'string', 'quote'],
  ['Cell Status', 'device_info.op_state', 'device_info.op_state', 'string', 'quote'],
  ['Online Status', 'device.is_online', 'devices.is_online', 'bool', 'enum'],
  ['Alarms', 'device_info.active_alarm_count', 'device_info.active_alarm_count', 'number', 'number'],
  ['Cell Name', 'device_info.device_name', 'device_info.device_name / devices.site_name', 'string', 'quote'],
  ['Shop ID', 'device.site_id', 'devices.site_id', 'string', 'quote'],
  ['IP Address', 'device.ip_address', 'devices.ip_address', 'string', 'quote'],
  ['MAC Address', 'device_info.mac', 'device_info.mac', 'string', 'quote'],
  ['ECI', 'device_info.eci', 'device_info.eci', 'string', 'preserve text'],
  ['PCI', 'device_info.pci', 'device_info.pci', 'string', 'preserve text'],
  ['Earfcn', 'device_info.freq_point', 'device_info.freq_point', 'number', 'preserve text'],
  ['MME Status', 'device_info.mme_status', 'device_info.mme_status', 'string', 'quote'],
  ['KPI Report Status', 'device_info.kpi_status', 'device_info.kpi_status', 'string', 'quote'],
  ['Sync Status', 'device_info.sync_status', 'device_info.sync_status', 'string', 'quote'],
  ['UE Count', 'device_info.ue_count', 'device_info.ue_count', 'number', 'number'],
  ['Last Period Time', 'device.last_inform_at', 'devices.last_inform_at', 'datetime', 'preserve text'],
  ['Product Type', 'device.product_class', 'devices.product_class', 'string', 'quote'],
  ['Hardware Version', 'device_info.hardware_version', 'device_info.hardware_version', 'string', 'quote'],
  ['Software Version', 'device.firmware_version', 'devices.firmware_version', 'string', 'quote'],
  ['Device Group', 'device_groups.name', 'device_groups / device_group_members', 'string', 'quote'],
  ['RF Status', 'device_info.rf_status', 'device_info.rf_status', 'string', 'quote'],
  ['Satellites', 'device_info.gps_satellites', 'device_info.gps_satellites', 'number', 'number'],
  ['Longitude', 'device.longitude', 'devices.longitude', 'number', 'number'],
  ['Latitude', 'device.latitude', 'devices.latitude', 'number', 'number'],
  ['Height', 'device_info.gps_height', 'device_info.gps_height', 'number', 'number'],
  ['Duplex Mode', 'device_info.network_model', 'device_info.network_model', 'string', 'quote'],
  ['IPsec Address', 'device_info.ipsec_addr', 'device_info.ipsec_addr', 'string', 'quote'],
  ['PLMN', 'device_info.plmn', 'device_info.plmn', 'string', 'preserve text'],
  ['First Online Time', 'device_info.first_online_time', 'device_info.first_online_time', 'datetime', 'preserve text'],
  ['AMF Status', 'device_info.mme_status', 'device_info.mme_status as amf_status', 'string', 'quote'],
  ['TAC', 'device_info.tac', 'device_info.tac', 'string', 'quote'],
  ['Model Name', 'device.model_name', 'devices.model_name', 'string', 'quote'],
  ['Manufacturer', 'device.manufacturer', 'devices.manufacturer', 'string', 'quote'],
  ['Site Name', 'device.site_name', 'devices.site_name / device_info.device_name', 'string', 'quote'],
  ['Installation Detailed Address', 'device_info.address', 'device_info.address', 'string', 'quote'],
  ['Device Status', 'device.lifecycle_state', 'devices.lifecycle_state + devices.is_online', 'string', 'enum'],
  ['First Period Time', 'device_info.first_online_time', 'device_info.first_online_time', 'datetime', 'preserve text'],
  ['Product Name', 'product.name', 'products.name / devices.product_class', 'string', 'quote'],
  ['System Uptime', 'device_info.run_time', 'device_info.run_time', 'number', 'number'],
  ['Accumulated Online Time(s)', 'device_info.cumulative_online_duration', 'device_info.cumulative_online_duration', 'number', 'number'],
  ['Bandwidth', 'device_info.bandwidth', 'device_info.bandwidth', 'number', 'number'],
  ['Height(m)', 'device_info.gps_height', 'device_info.gps_height', 'number', 'number'],
  ['eNB ID', 'device_info.enb_id', 'device_info.enb_id', 'string', 'quote'],
  ['Cell ID', 'device_info.cell_id', 'device_info.cell_id', 'string', 'quote'],
  ['Subframe Assignment', 'device_info.subframe_assignment', 'device_info.subframe_assignment', 'string', 'quote'],
  ['Special Subframe Patterns', 'device_info.special_subframe', 'device_info.special_subframe', 'string', 'quote'],
] satisfies Array<[string, string, string, string, string]>;

const inventoryAliasByType: Record<InventoryType, Record<string, string>> = {
  ENB: {},
  GNB: {
    'Cell Name': 'gNB Name',
    ECI: 'NCI',
    Earfcn: 'NR-ARFCN',
    'MME Status': 'AMF Status',
    'eNB ID': 'gNB ID',
    'Subframe Assignment': 'Slot Assignment',
    'Special Subframe Patterns': 'Special Slot Patterns',
  },
  GSM: {
    'Cell Name': 'BTS Name',
    ECI: 'CGI',
    PCI: 'BSIC',
    Earfcn: 'BCCH ARFCN',
    'MME Status': 'BSC Status',
    'eNB ID': 'BTS ID',
    'Subframe Assignment': 'Channel Assignment',
    'Special Subframe Patterns': 'Channel Pattern',
  },
  OMC: {},
};

function buildRadioInventoryFields(type: Exclude<InventoryType, 'OMC'>): InventoryField[] {
  return stationInventoryFieldDefinitions.map(([column, exportKey, source, dataType, renderer], index) => ({
    key: `${type.toLowerCase()}-${index + 1}`,
    template: type,
    column: inventoryAliasByType[type][column] ?? column,
    exportKey,
    source,
    dataType,
    renderer,
    enabled: true,
  }));
}

const stationInventoryFields: InventoryField[] = [
  ...buildRadioInventoryFields('ENB'),
  ...buildRadioInventoryFields('GNB'),
  ...buildRadioInventoryFields('GSM'),
];

const omcInventoryFields: InventoryField[] = [
  ['OMC', 'eNB online', 'inventory.omc.enb_online', 'devices.is_online aggregate', 'number', 'number'],
  ['OMC', 'eNB active', 'inventory.omc.enb_active', 'device_info.op_state / lifecycle aggregate', 'number', 'number'],
  ['OMC', 'MME status', 'inventory.omc.mme_status', 'device_info.mme_status aggregate', 'string', 'quote'],
  ['OMC', 'UE Count', 'inventory.omc.ue_count', 'SUM(device_info.ue_count)', 'number', 'number'],
  ['OMC', 'Version', 'inventory.omc.version', 'buildinfo.ReleaseVersion', 'string', 'quote'],
].map(([template, column, exportKey, source, dataType, renderer], index) => ({
  key: `omc-${index + 1}`,
  template: template as InventoryType,
  column,
  exportKey,
  source,
  dataType,
  renderer,
  enabled: true,
}));

const inventoryFields = [...stationInventoryFields, ...omcInventoryFields];

const inventoryTypeOptions: Array<{ label: string; value: InventoryType }> = [
  { label: 'eNB', value: 'ENB' },
  { label: 'gNB', value: 'GNB' },
  { label: 'GSM', value: 'GSM' },
  { label: 'OMC', value: 'OMC' },
];

const deviceInfoFieldDefinitions = [
  ['device_id', 'Device Info Device ID', '设备信息设备 ID', 'string', 'quote'],
  ['device_name', 'Device Name', '设备名称', 'string', 'quote'],
  ['address', 'Address', '安装地址', 'string', 'quote'],
  ['remark', 'Remark', '备注', 'string', 'quote'],
  ['project_status', 'Project Status', '工程状态', 'string', 'quote'],
  ['height', 'Height', '海拔/高度', 'number', 'number'],
  ['eci', 'ECI', 'ECI', 'string', 'preserve text'],
  ['pci', 'PCI', 'PCI', 'string', 'preserve text'],
  ['cell_id', 'Cell ID', '小区 ID', 'string', 'quote'],
  ['freq_point', 'Frequency Point', '频点', 'number', 'preserve text'],
  ['bandwidth', 'Bandwidth', '带宽', 'number', 'number'],
  ['transmit_power', 'Transmit Power', '发射功率', 'number', 'number'],
  ['plmn', 'PLMN', 'PLMN', 'string', 'preserve text'],
  ['rf_status', 'RF Status', '射频状态', 'string', 'enum'],
  ['cell_status', 'Cell Status', '小区状态', 'string', 'enum'],
  ['mme_status', 'MME Status', 'MME/AMF 状态', 'string', 'enum'],
  ['sync_status', 'Sync Status', '同步状态', 'string', 'enum'],
  ['kpi_status', 'KPI Report Status', 'KPI 上报状态', 'string', 'enum'],
  ['num_of_cells', 'Number of Cells', '小区数量', 'number', 'number'],
  ['gps_status', 'GPS Status', 'GPS 状态', 'string', 'enum'],
  ['alarm_severity', 'Alarm Severity', '告警级别', 'string', 'enum'],
  ['license_status', 'License Status', 'License 状态', 'string', 'enum'],
  ['mac', 'MAC Address', 'MAC 地址', 'string', 'quote'],
  ['hardware_version', 'Hardware Version', '硬件版本', 'string', 'quote'],
  ['first_online_time', 'First Online Time', '首次上线时间', 'datetime', 'yyyy-MM-dd HH:mm:ss'],
  ['last_online_time', 'Last Online Time', '最近上线时间', 'datetime', 'yyyy-MM-dd HH:mm:ss'],
  ['last_offline_time', 'Last Offline Time', '最近离线时间', 'datetime', 'yyyy-MM-dd HH:mm:ss'],
  ['run_time', 'Run Time', '运行时长', 'number', 'number'],
  ['creator', 'Creator', '创建人', 'string', 'quote'],
  ['updater', 'Updater', '更新人', 'string', 'quote'],
  ['created_at', 'Created At', '创建时间', 'datetime', 'yyyy-MM-dd HH:mm:ss'],
  ['updated_at', 'Updated At', '更新时间', 'datetime', 'yyyy-MM-dd HH:mm:ss'],
  ['tac', 'TAC', 'TAC', 'string', 'quote'],
  ['band', 'Band', '频段', 'string', 'quote'],
  ['ul_earfcn', 'UL EARFCN', '上行频点', 'number', 'preserve text'],
  ['subframe_assignment', 'Subframe Assignment', '子帧配比', 'string', 'quote'],
  ['special_subframe', 'Special Subframe', '特殊子帧配比', 'string', 'quote'],
  ['root_index', 'Root Index', '根序列索引', 'number', 'preserve text'],
  ['gps_satellites', 'GPS Satellites', 'GPS 卫星数', 'number', 'number'],
  ['gps_height', 'GPS Height', 'GPS 高度', 'number', 'number'],
  ['lock_status', 'Lock Status', '锁定状态', 'string', 'enum'],
  ['enb_id', 'eNB ID', '基站 ID', 'string', 'quote'],
  ['network_model', 'Network Model', '网络制式/双工模式', 'string', 'enum'],
  ['lac', 'LAC', 'LAC', 'string', 'quote'],
  ['cumulative_online_duration', 'Cumulative Online Duration', '累计在线时长', 'number', 'number'],
  ['op_state', 'Operation State', '激活状态', 'string', 'enum'],
  ['admin_state', 'Admin State', '管理状态', 'string', 'enum'],
  ['ipsec_addr', 'IPsec Address', 'IPsec 地址', 'string', 'quote'],
  ['bsc_select', 'BSC Select', 'BSC 主备选择', 'string', 'enum'],
  ['oml_remote_ip', 'OML Remote IP', 'OML 主 BSC IP', 'string', 'quote'],
  ['oml_remote_ip_bak', 'OML Remote IP Backup', 'OML 备 BSC IP', 'string', 'quote'],
  ['ipa_unit_id', 'IPA Unit ID', 'IPA Unit ID', 'string', 'quote'],
  ['ue_count', 'UE Count', 'UE 数', 'number', 'number'],
  ['active_alarm_count', 'Active Alarm Count', '活动告警数', 'number', 'number'],
  ['name_sync_pending', 'Name Sync Pending', '名称同步待处理', 'bool', 'enum'],
  ['lmt_device_name', 'LMT Device Name', 'LMT 设备名称', 'string', 'quote'],
  ['highest_alarm_severity', 'Highest Alarm Severity', '最高告警级别', 'number', 'number'],
  ['highest_severity_alarm_count', 'Highest Severity Alarm Count', '最高级别告警数量', 'number', 'number'],
] satisfies Array<[string, string, string, string, string]>;

const deviceInfoInventoryFields: InventoryField[] = inventoryTypeOptions
  .filter((option) => option.value !== 'OMC')
  .flatMap((option) => deviceInfoFieldDefinitions.map(([column, outputAlias, , dataType, renderer]) => ({
    key: `${option.value.toLowerCase()}-device-info-${column}`,
    template: option.value,
    column: inventoryAliasByType[option.value][outputAlias] ?? outputAlias,
    exportKey: `device_info.${column}`,
    source: `device_info.${column}`,
    dataType,
    renderer,
    enabled: true,
  })));

const inventoryAvailableFields = [...inventoryFields, ...deviceInfoInventoryFields];

const deliveryProtocolOptions: Array<{ label: string; value: DeliveryProtocol }> = [
  { label: 'SFTP', value: 'SFTP' },
  { label: 'FTP', value: 'FTP' },
];

// No built-in delivery targets: operators add environment-specific FTP/SFTP
// destinations manually. Kept as an empty typed array so cloneDeliveryTargets
// returns a stable empty list before the backend load completes.
const deliveryTargetSeeds: Array<Omit<DeliveryTargetRow, 'key'>> = [];

function cloneDeliveryTargets(scopeKey: string): DeliveryTargetRow[] {
  return deliveryTargetSeeds.map((target, index) => ({
    ...target,
    key: `${scopeKey}-delivery-${index + 1}`,
  }));
}

function createDeliveryTargetKey(scopeKey: string): string {
  return `${scopeKey}-delivery-custom-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

function createDeliveryTarget(scopeKey: string, index: number): DeliveryTargetRow {
  return {
    key: createDeliveryTargetKey(scopeKey),
    name: `新增传输目标 ${index}`,
    enabled: false,
    protocol: 'SFTP',
    host: '',
    port: 22,
    username: '',
    credential: '',
    authMode: 'PASSWORD',
    remoteRoot: '/northupload',
    retryTimes: 3,
    timeoutSeconds: 30,
    passiveMode: false,
    hostKeyPolicy: 'INSECURE',
    hostKeyFingerprint: '',
  };
}

const initialInventoryConfigs: InventoryConfigRow[] = [
  {
    key: 'ENB',
    name: 'eNB Inventory',
    objectCode: 'eNB',
    tech: 'LTE',
    period: '24H',
    cron: '30 1 0 * * ?',
    format: 'CSV',
    path: '/#FTPRoot#/#Province#/#OMC-R#/Inventory/eNB/#DateTime#/',
    fileName: 'BaiOMC_eNB_#DateTime#.csv',
    compressionEnabled: false,
    compressionFormat: 'zip',
  },
  {
    key: 'GNB',
    name: 'gNB Inventory',
    objectCode: 'gNB',
    tech: 'GNB',
    period: '24H',
    cron: '30 2 0 * * ?',
    format: 'CSV',
    path: '/#FTPRoot#/#Province#/#OMC-R#/Inventory/gNB/#DateTime#/',
    fileName: 'BaiOMC_gNB_#DateTime#.csv',
    compressionEnabled: false,
    compressionFormat: 'zip',
  },
  {
    key: 'GSM',
    name: 'GSM Inventory',
    objectCode: 'GSM',
    tech: 'GSM',
    period: '24H',
    cron: '30 3 0 * * ?',
    format: 'CSV',
    path: '/#FTPRoot#/#Province#/#OMC-R#/Inventory/GSM/#DateTime#/',
    fileName: 'BaiOMC_GSM_#DateTime#.csv',
    compressionEnabled: false,
    compressionFormat: 'zip',
  },
  {
    key: 'OMC',
    name: 'OMC Inventory',
    objectCode: 'OMC',
    tech: '系统',
    period: '24H',
    cron: '30 4 0 * * ?',
    format: 'CSV',
    path: '/#FTPRoot#/#Province#/#OMC-R#/Inventory/OMC/#DateTime#/',
    fileName: 'BaiOMC_OMC_#DateTime#.csv',
    compressionEnabled: false,
    compressionFormat: 'zip',
  },
];

const defaultInventoryEnabled = Object.fromEntries(
  inventoryTypeOptions.map((option) => [option.value, false]),
) as Record<InventoryType, boolean>;

const defaultInventoryFieldRows = Object.fromEntries(
  inventoryTypeOptions.map((option) => [
    option.value,
    inventoryFields.filter((field) => field.template === option.value),
  ]),
) as Record<InventoryType, InventoryField[]>;

const socketAlarmConfigs: SocketAlarmConfigRow[] = [
  {
    key: 'socket-ctcc-server',
    name: '电信 Socket 告警',
    profile: 'CTCC',
    frame: '16 字节帧头 / 0x7EE7 / JSON',
    listenIp: '0.0.0.0',
    listenPort: 31232,
    maxClients: 20,
    heartbeatPeriod: 60,
    heartbeatTimes: 3,
    encoding: 'UTF-8',
    realTimeEnabled: true,
    historyEnabled: true,
    syncMode: '消息同步',
    accountTypes: 'msg',
    sequencePolicy: 'alarmSequenceId',
  },
  {
    key: 'socket-cucc-server',
    name: '联通 Socket 告警',
    profile: 'CUCC',
    frame: '9 字节帧头 / 0xFFFF / 命令串 + JSON',
    listenIp: '0.0.0.0',
    listenPort: 31233,
    maxClients: 20,
    heartbeatPeriod: 60,
    heartbeatTimes: 3,
    encoding: 'UTF-8',
    realTimeEnabled: true,
    historyEnabled: true,
    syncMode: '消息同步 + 文件同步',
    accountTypes: 'msg / ftp',
    sequencePolicy: '连续 alarmSeq',
  },
];

const snmpAlarmTargets: SnmpAlarmTargetRow[] = [
  {
    key: 'snmp-v2-primary',
    name: 'SNMP V2C',
    version: 'v2',
    notificationType: 'Trap',
    listenIp: '0.0.0.0',
    listenPort: 161,
    targetHost: '10.10.41.11',
    targetPort: 162,
    community: snmpDefaultCommunity,
    mibQueryEnabled: true,
    clearSeverityPolicy: '保留原级别',
    timeoutSeconds: 5,
    retries: 1,
  },
  {
    key: 'snmp-v3-inform',
    name: 'SNMP V3',
    version: 'v3',
    notificationType: 'Inform',
    listenIp: '0.0.0.0',
    listenPort: 161,
    targetHost: '10.10.41.12',
    targetPort: 163,
    securityName: 'notifyV3',
    authProtocol: 'SHA',
    privProtocol: 'DES',
    mibQueryEnabled: false,
    clearSeverityPolicy: '保留原级别',
    timeoutSeconds: 5,
    retries: 1,
  },
];

const defaultSocketEnabled = Object.fromEntries(
  socketAlarmConfigs.map((row) => [row.key, false]),
) as Record<string, boolean>;

const defaultSnmpEnabled = Object.fromEntries(
  snmpAlarmTargets.map((row) => [row.key, false]),
) as Record<string, boolean>;

const socketAccountTypeOptions = [
  { label: 'msg - 实时推送/消息同步', value: 'msg' },
  { label: 'ftp - 文件同步', value: 'ftp' },
];

const socketDefaultAccountsByProfile: Record<SocketProfile, SocketAccountRow[]> = {
  CTCC: [
    {
      key: 'msg-main',
      channel: '实时/消息同步账号',
      username: 'north-msg',
      type: 'msg',
      credential: '',
      enabled: true,
      purpose: '登录、实时告警、历史消息同步',
    },
  ],
  CUCC: [
    {
      key: 'msg-main',
      channel: '实时/消息同步账号',
      username: 'north-msg',
      type: 'msg',
      credential: '',
      enabled: true,
      purpose: '登录、实时告警、历史消息同步',
    },
    {
      key: 'ftp-main',
      channel: '文件同步账号',
      username: 'north-file',
      type: 'ftp',
      credential: '',
      enabled: true,
      purpose: '登录、告警文件同步请求',
    },
  ],
};

const socketCtccFields: AlarmFieldMappingRow[] = [
  { key: 'ctcc-alarm-seq', order: 1, field: 'alarmSequenceId', cnName: '告警序号', source: 'alarm.sequence_id', dataType: 'number', enabled: true },
  { key: 'ctcc-status', order: 2, field: 'alarmStatus', cnName: '告警状态', source: 'alarm.status', dataType: 'string', enabled: true },
  { key: 'ctcc-type', order: 3, field: 'alarmType', cnName: '告警类型', source: 'alarm.alarm_type', dataType: 'string', enabled: true },
  { key: 'ctcc-severity', order: 4, field: 'origSeverity', cnName: '告警级别', source: 'alarm.severity', dataType: 'string', enabled: true },
  { key: 'ctcc-event-time', order: 5, field: 'eventTime', cnName: '事件时间', source: 'alarm.raised_at / alarm.cleared_at', dataType: 'datetime', enabled: true },
  { key: 'ctcc-alarm-id', order: 6, field: 'alarmId', cnName: '告警 ID', source: 'alarm.alarm_identifier', dataType: 'string', enabled: true },
  { key: 'ctcc-problem-id', order: 7, field: 'specificProblemID', cnName: '厂家告警码', source: 'alarm.alarm_type / alarm_definition.vendor_code', dataType: 'string', enabled: true },
  { key: 'ctcc-problem', order: 8, field: 'specificProblem', cnName: '告警标题', source: 'alarm.description', dataType: 'string', enabled: true },
  { key: 'ctcc-ne', order: 9, field: 'neDn / neName / neType', cnName: '网元信息', source: 'alarm.device_sn / device.device_name / device.technology', dataType: 'string', enabled: true },
  { key: 'ctcc-object', order: 10, field: 'objectDn / objectName / objectType', cnName: '对象信息', source: 'alarm.alarm_source / alarm.network_location', dataType: 'string', enabled: true },
  { key: 'ctcc-add-info', order: 11, field: 'addInfo', cnName: '补充信息', source: 'alarm.additional_info', dataType: 'string', enabled: true },
  { key: 'ctcc-omc-time', order: 12, field: 'omcReceivedTime', cnName: 'OMC 接收时间', source: 'alarm.created_at', dataType: 'datetime', enabled: true },
  { key: 'ctcc-omc-id', order: 13, field: 'omcUID', cnName: 'OMC 标识', source: 'system.omc_id', dataType: 'string', enabled: true },
];

const socketCuccFields: AlarmFieldMappingRow[] = socketCtccFields.map((field) => {
  const remap: Record<string, string> = {
    alarmSequenceId: 'alarmSeq',
    'neDn / neName / neType': 'neUID / neName / neType',
    'objectDn / objectName / objectType': 'objectUID / objectName / objectType',
  };
  return {
    ...field,
    key: field.key.replace('ctcc', 'cucc'),
    field: remap[field.field] ?? field.field,
  };
});

const snmpAlarmFields: AlarmFieldMappingRow[] = [
  { key: 'snmp-notification-id', order: 1, field: 'notificationID', cnName: '通知序号', source: 'alarm.sequence_id', dataType: 'Integer32(1..2147483647)', enabled: true },
  { key: 'snmp-alarm-unique-id', order: 2, field: 'alarmUniqueId', cnName: '告警唯一标识', source: 'alarm.alarm_identifier', dataType: 'OCTET STRING(5)', enabled: true },
  { key: 'snmp-notification-type', order: 3, field: 'notificationType', cnName: '通知类型', source: 'alarm.status -> 0/1', dataType: 'OCTET STRING(1)', enabled: true },
  { key: 'snmp-event-time', order: 4, field: 'eventTime', cnName: '事件时间', source: 'alarm.raised_at / alarm.cleared_at -> epoch_ms', dataType: 'Counter64(13)', enabled: true },
  { key: 'snmp-equipment-sdn', order: 5, field: 'equipmentSDN', cnName: '网元标识', source: 'alarm.device_sn', dataType: 'OCTET STRING(1..45)', enabled: true },
  { key: 'snmp-equipment-name', order: 6, field: 'equipmentName', cnName: '网元名称', source: 'alarm.device_name / device.device_name', dataType: 'OCTET STRING(0..50)', enabled: true },
  { key: 'snmp-equipment-class', order: 7, field: 'equipmentClass', cnName: '网元类型', source: 'device.technology', dataType: 'OCTET STRING(0..45)', enabled: true },
  { key: 'snmp-object-sdn', order: 8, field: 'objectSDN', cnName: '对象标识', source: 'alarm.alarm_source', dataType: 'OCTET STRING(1..45)', enabled: true },
  { key: 'snmp-object-name', order: 9, field: 'objectInstanceName', cnName: '对象名称', source: 'alarm.network_location', dataType: 'OCTET STRING(0..50)', enabled: true },
  { key: 'snmp-object-class', order: 10, field: 'objectClass', cnName: '对象类型', source: 'alarm.event_type', dataType: 'OCTET STRING(0..45)', enabled: true },
  { key: 'snmp-additional-text', order: 11, field: 'additionalText', cnName: '定位信息', source: 'alarm.description', dataType: 'OCTET STRING(0..256)', enabled: true },
  { key: 'snmp-vendor-oui', order: 12, field: 'deviceVendorOUI', cnName: '厂商 OUI', source: 'device.oui / device.manufacturer', dataType: 'OCTET STRING(0..10)', enabled: true },
  { key: 'snmp-problem-id', order: 13, field: 'specificProblemID', cnName: '厂家告警码', source: 'alarm.alarm_type / alarm_definition.vendor_code', dataType: 'OCTET STRING(5)', enabled: true },
  { key: 'snmp-problem', order: 14, field: 'specificProblem', cnName: '告警标题', source: 'alarm.description', dataType: 'OCTET STRING(0..256)', enabled: true },
  { key: 'snmp-alarm-type', order: 15, field: 'alarmType', cnName: '告警类型', source: 'alarm.alarm_type', dataType: 'OCTET STRING(5)', enabled: true },
  { key: 'snmp-severity', order: 16, field: 'perceivedSeverity', cnName: '告警级别', source: 'alarm.severity', dataType: 'OCTET STRING(5..8)', enabled: true },
  { key: 'snmp-probable-cause', order: 17, field: 'probableCause', cnName: '可能原因', source: 'alarm.probable_cause', dataType: 'OCTET STRING(0..256)', enabled: true },
  { key: 'snmp-additional-info', order: 18, field: 'additionalInformation', cnName: '辅助信息', source: 'alarm.additional_info', dataType: 'OCTET STRING(0..256)', enabled: true },
];

const snmpAlarmOIDByField: Record<string, string> = {
  notificationID: '1.3.6.1.4.1.53058.1.1.1.1.1.1',
  alarmUniqueId: '1.3.6.1.4.1.53058.1.1.1.1.1.2',
  notificationType: '1.3.6.1.4.1.53058.1.1.1.1.1.3',
  eventTime: '1.3.6.1.4.1.53058.1.1.1.1.1.4',
  equipmentSDN: '1.3.6.1.4.1.53058.1.1.1.1.1.5',
  equipmentName: '1.3.6.1.4.1.53058.1.1.1.1.1.6',
  equipmentClass: '1.3.6.1.4.1.53058.1.1.1.1.1.7',
  objectSDN: '1.3.6.1.4.1.53058.1.1.1.1.1.8',
  objectInstanceName: '1.3.6.1.4.1.53058.1.1.1.1.1.9',
  objectClass: '1.3.6.1.4.1.53058.1.1.1.1.1.10',
  additionalText: '1.3.6.1.4.1.53058.1.1.1.1.1.11',
  deviceVendorOUI: '1.3.6.1.4.1.53058.1.1.1.1.1.12',
  specificProblemID: '1.3.6.1.4.1.53058.1.1.1.1.1.13',
  specificProblem: '1.3.6.1.4.1.53058.1.1.1.1.1.14',
  alarmType: '1.3.6.1.4.1.53058.1.1.1.1.1.15',
  perceivedSeverity: '1.3.6.1.4.1.53058.1.1.1.1.1.16',
  probableCause: '1.3.6.1.4.1.53058.1.1.1.1.1.17',
  additionalInformation: '1.3.6.1.4.1.53058.1.1.1.1.1.18',
};

const snmpAlarmFieldByOID = Object.fromEntries(
  snmpAlarmFields.map((field) => [snmpAlarmOIDByField[field.field], field]),
) as Record<string, AlarmFieldMappingRow | undefined>;

function snmpAlarmOIDForField(field: string): string {
  return snmpAlarmOIDByField[field] ?? '-';
}

function buildSampleSnmpPayload(notificationID = '920188', severity = 'major'): string {
  const values: Record<string, string> = {
    notificationID,
    alarmUniqueId: '40123',
    notificationType: '1',
    eventTime: '1785830400000',
    equipmentSDN: '867294050000001',
    equipmentName: 'Site-A',
    equipmentClass: 'LTE',
    objectSDN: 'SubNetwork=OMC,ManagedElement=867294050000001,Cell=1',
    objectInstanceName: 'Cell-1',
    objectClass: 'Cell',
    additionalText: 'Cell Unavailable',
    deviceVendorOUI: 'Baicells',
    specificProblemID: '40123',
    specificProblem: 'Cell Unavailable',
    alarmType: 'CELL_UNAVAILABLE',
    perceivedSeverity: severity,
    probableCause: 'Cell unavailable',
    additionalInformation: 'pci=123;earfcn=38400',
  };
  return snmpAlarmFields
    .map((field) => `${snmpAlarmOIDForField(field.field)} = ${values[field.field] ?? ''}`)
    .join('\n');
}

const reportStatusSamples: Record<string, Partial<ReportStatusInfo>> = {
  'file:S0001': {
    state: 'success',
    statusText: '上报正常',
    lastTime: '2026-08-03 15:15:04',
    artifactName: 'Baicells-PC-172.21.172.189-1.0-20260803151500-15.csv.zip',
    artifactPath: '/northupload/GD/BaiOMC/PM/20260803/',
    size: '2.4 MB',
    targetSummary: '主用 SFTP 成功，备用 FTP 成功',
    detail: '最近一次 PM 文件生成成功，两个启用传输目标均上传完成。',
    payload: 'startTime,endTime,deviceSn,metricPath,value\n20260803150000,20260803151500,867294050000001,pm.prb.ul.usage,37.42\n20260803150000,20260803151500,867294050000002,pm.prb.ul.usage,41.08\n',
  },
  'file:S0012': {
    state: 'failed',
    statusText: '上传失败',
    lastTime: '2026-08-03 14:00:12',
    artifactName: 'Baicells-CP-172.21.172.189-1.0-20260803140000.csv.zip',
    artifactPath: '/northupload/SN/BaiOMC/CM/20260803/',
    size: '804 KB',
    targetSummary: '主用 SFTP 超时，备用 FTP 未启用',
    detail: '文件已生成，传输目标连接超时。保留本地运行记录，可手动重试投递。',
  },
  'inventory:ENB': {
    state: 'success',
    statusText: '上报正常',
    lastTime: '2026-08-03 00:05:22',
    artifactName: 'BaiOMC_eNB_20260803000000.csv.zip',
    artifactPath: '/northupload/GD/BaiOMC/Inventory/eNB/20260803/',
    size: '618 KB',
    targetSummary: '主用 SFTP 成功',
    detail: 'eNB Inventory 文件已生成并上传，字段数与当前配置一致。',
    payload: 'SN,Name,IP,Cell Status,KPI Report Status\n867294050000001,Site-A,10.21.1.11,Enabled,Normal\n867294050000002,Site-B,10.21.1.12,Enabled,Normal\n',
  },
  'inventory:GNB': {
    state: 'running',
    statusText: '上报中',
    lastTime: '2026-08-03 16:45:02',
    artifactName: 'BaiOMC_gNB_20260803164500.csv',
    artifactPath: '/northupload/GD/BaiOMC/Inventory/gNB/20260803/',
    size: '生成中',
    targetSummary: '正在上传主用 SFTP',
    detail: '当前任务已完成取数，正在上传到启用传输目标。',
  },
  'socket:socket-ctcc': {
    state: 'success',
    statusText: '推送正常',
    lastTime: '2026-08-03 16:52:18',
    artifactType: 'message',
    artifactName: 'ALARM_OR_ACTIVE_ALARM #920188',
    artifactPath: 'socket://0.0.0.0:31232/session/nms-ctcc-01',
    size: '1.6 KB',
    targetSummary: '1 个客户端在线，最近 ACK 正常',
    detail: '实时告警已推送给已登录客户端，心跳正常。',
    payload: '{\n  "msgType": 10,\n  "alarmSequenceId": 920188,\n  "alarmStatus": "active",\n  "origSeverity": "major",\n  "eventTime": "2026-08-03 16:52:18",\n  "neDn": "SubNetwork=OMC,ManagedElement=867294050000001",\n  "specificProblem": "Cell Unavailable"\n}',
  },
  'socket:socket-cucc': {
    state: 'success',
    statusText: '同步正常',
    lastTime: '2026-08-03 16:50:31',
    artifactType: 'message',
    artifactName: 'ACK_SYNC_ALARM_MSG reqId=CUCC-20260803165031',
    artifactPath: 'socket://0.0.0.0:31233/session/nms-cucc-01',
    size: '2.1 KB',
    targetSummary: '消息同步成功，文件同步目标未启用',
    detail: '客户端按 alarmSeq 发起同步请求，OMC 已回放 12 条告警并恢复实时推送。',
    payload: 'ackSyncAlarmMsg;reqId=CUCC-20260803165031;result=0;alarmSeq=920176;count=12\nrealTimeAlarm;alarmSeq=920188;severity=major;objectUID=867294050000001;eventTime=20260803165031',
  },
  'snmp:snmp-v2-primary': {
    state: 'success',
    statusText: 'Trap 正常',
    lastTime: '2026-08-03 16:51:09',
    artifactType: 'message',
    artifactName: 'omcAlarmNotification notificationID=920188',
    artifactPath: 'snmp://10.10.41.11:162',
    size: '1.2 KB',
    targetSummary: 'Trap 发送成功',
    detail: 'SNMP V2C Trap 已发送到目标 NMS。',
    payload: buildSampleSnmpPayload('920188', 'major'),
  },
  'snmp:snmp-v3-inform': {
    state: 'failed',
    statusText: 'Inform 超时',
    lastTime: '2026-08-03 16:40:00',
    artifactType: 'message',
    artifactName: 'omcAlarmNotification notificationID=920171',
    artifactPath: 'snmp://10.10.41.12:163',
    size: '1.3 KB',
    targetSummary: 'Inform 等待响应超时，已重试 1 次',
    detail: '目标未返回 Inform ACK，建议检查 NMS 地址、安全用户和防火墙。',
    payload: 'version=v3; user=notifyV3; auth=SHA; priv=DES; notificationID=920171; perceivedSeverity=minor; result=timeout',
  },
};

const commonApiAuth = '北向 API 用户 token；Authorization: Bearer <access-token> 或 X-Northbound-Token';
const currentEnvelopeFields = ['ret', 'msg', 'data'];
const listResponseFields = ['ret', 'msg', 'data.items', 'data.total', 'data.page', 'data.page_size', 'data.total_pages', 'data.stats?'];
const apiUserPasswordMask = '********';

const northboundApiRows: NorthboundApiRow[] = [
  {
    key: 'auth-login',
    apiKind: '鉴权管理',
    module: '鉴权',
    name: '北向认证：获取访问 Token',
    method: 'POST',
    url: '/api/v1/northbound/v1/access/token',
    auth: '公开接口；只校验北向 API 专用用户',
    backendSource: 'omcgo/internal/northbound/pageconfig/service.go + cmd/app/provider/router.go',
    fieldContract: '完整返回字段',
    responseFields: [...currentEnvelopeFields, 'data.token', 'data.expires', 'data.access_token', 'data.expires_at', 'data.token_type'],
    requestExample: `POST /api/v1/northbound/v1/access/token
Content-Type: application/json

{
  "username": "northbound_api",
  "password": "******"
}`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "token": "nbt_AOa...",
    "access_token": "nbt_AOa...",
    "expires": 1800,
    "expires_at": "2026-07-30T02:00:00Z",
    "token_type": "Bearer"
  }
}`,
  },
  {
    key: 'nb-sync-full-device',
    apiKind: '正式北向',
    module: '同步',
    name: '设备全量同步：导出当前设备清单',
    method: 'GET',
    url: '/api/v1/northbound/v1/sync/full?data_type=device&format=json',
    auth: commonApiAuth,
    backendSource: 'omcgo/internal/northbound/router.go + internal/northbound/sync/service.go',
    fieldContract: '当前返回字段',
    responseFields: [...currentEnvelopeFields, 'data.data_type', 'data.items[]', 'data.total', 'data.synced_at', 'data.truncated?'],
    requestExample: `GET /api/v1/northbound/v1/sync/full?data_type=device&format=json
X-Northbound-Token: <access-token>`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "data_type": "device",
    "items": [
      {
        "id": "9a4d7b2f-2b2c-4f0a-9ec5-5e9b3a8c1001",
        "serial_number": "1202000240194",
        "product_class": "FAP-LTE-100",
        "technology": "LTE",
        "is_online": true
      }
    ],
    "total": 1,
    "synced_at": "2026-07-30T01:00:00Z",
    "truncated": false
  }
}`,
  },
  {
    key: 'nb-sync-incremental',
    apiKind: '正式北向',
    module: '同步',
    name: '增量同步',
    method: 'GET',
    url: '/api/v1/northbound/sync/incremental?data_type=alarm&since=2026-07-30T00:00:00Z&format=json',
    auth: commonApiAuth,
    backendSource: 'omcgo/internal/northbound/router.go + internal/northbound/sync/service.go',
    fieldContract: '当前返回字段',
    responseFields: [...currentEnvelopeFields, 'data.data_type', 'data.items[]', 'data.total', 'data.synced_at', 'data.truncated?'],
    requestExample: `GET /api/v1/northbound/sync/incremental?data_type=alarm&since=2026-07-30T00:00:00Z&format=json
X-Northbound-Token: <access-token>`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "data_type": "alarm",
    "items": [
      {
        "device_sn": "1202000240194",
        "severity": 2,
        "alarm_type": "mmeDisconnected",
        "status": "active",
        "raised_at": "2026-07-30T00:58:00Z"
      }
    ],
    "total": 1,
    "synced_at": "2026-07-30T01:00:00Z"
  }
}`,
  },
  {
    key: 'nb-export-pm',
    apiKind: '正式北向',
    module: 'PM',
    name: 'PM Counter 导出',
    method: 'POST',
    url: '/api/v1/northbound/export/pm',
    auth: commonApiAuth,
    backendSource: 'omcgo/internal/northbound/pm_handler.go',
    fieldContract: '完整返回字段',
    responseFields: [...listResponseFields, 'data.items[].time', 'data.items[].device_id', 'data.items[].oui', 'data.items[].device_sn', 'data.items[].cell_id', 'data.items[].counter_group', 'data.items[].counter_name', 'data.items[].counter_value', 'data.items[].granularity', 'data.items[].statis_type?', 'data.items[].unit?'],
    requestExample: `POST /api/v1/northbound/export/pm
X-Northbound-Token: <access-token>
Content-Type: application/json

{
  "device_id": "9a4d7b2f-2b2c-4f0a-9ec5-5e9b3a8c1001",
  "cell_id": "1",
  "counter_group": "RRC",
  "start_time": "2026-07-30T00:00:00Z",
  "end_time": "2026-07-30T01:00:00Z",
  "format": "json"
}`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "items": [
      {
        "time": "2026-07-30T00:45:00Z",
        "device_id": "9a4d7b2f-2b2c-4f0a-9ec5-5e9b3a8c1001",
        "device_sn": "1202000240194",
        "cell_id": "1",
        "counter_group": "RRC",
        "counter_name": "RRC.ConnEstabAtt",
        "counter_value": 100,
        "granularity": 15
      }
    ],
    "total": 1,
    "page": 1,
    "page_size": 100,
    "total_pages": 1
  }
}`,
  },
  {
    key: 'nb-export-alarms',
    apiKind: '正式北向',
    module: '告警',
    name: '告警导出',
    method: 'POST',
    url: '/api/v1/northbound/export/alarms',
    auth: commonApiAuth,
    backendSource: 'omcgo/internal/northbound/alarm_handler.go',
    fieldContract: '完整返回字段',
    responseFields: [...listResponseFields, 'data.items[].id', 'data.items[].device_id', 'data.items[].device_sn', 'data.items[].carrier', 'data.items[].severity', 'data.items[].alarm_type', 'data.items[].alarm_identifier', 'data.items[].description', 'data.items[].status', 'data.items[].raised_at', 'data.items[].acknowledged_at?', 'data.items[].cleared_at?', 'data.items[].device_name?', 'data.items[].technology?', 'data.items[].additional_info?'],
    requestExample: `POST /api/v1/northbound/export/alarms
X-Northbound-Token: <access-token>
Content-Type: application/json

{
  "device_sn": "1202000240194",
  "severity": "31002",
  "status": "active",
  "start_time": "2026-07-30T00:00:00Z",
  "end_time": "2026-07-30T01:00:00Z",
  "format": "json"
}`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "items": [
      {
        "device_sn": "1202000240194",
        "severity": 2,
        "alarm_type": "mmeDisconnected",
        "status": "active",
        "raised_at": "2026-07-30T00:58:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "page_size": 100,
    "total_pages": 1
  }
}`,
  },
  {
    key: 'nb-export-config',
    apiKind: '正式北向',
    module: '配置',
    name: '配置快照导出：按设备ID导出参数',
    method: 'GET',
    url: '/api/v1/northbound/v1/export/config/{deviceId}',
    auth: commonApiAuth,
    backendSource: 'omcgo/internal/northbound/config_handler.go',
    fieldContract: '完整返回字段',
    responseFields: [...currentEnvelopeFields, 'data.device_id', 'data.parameters[].device_id', 'data.parameters[].parameter_path', 'data.parameters[].parameter_value', 'data.parameters[].parameter_type', 'data.parameters[].writable', 'data.parameters[].last_updated_at', 'data.parameters[].fap_instance', 'data.parameters[].param_group', 'data.total'],
    requestExample: `GET /api/v1/northbound/v1/export/config/9a4d7b2f-2b2c-4f0a-9ec5-5e9b3a8c1001
X-Northbound-Token: <access-token>`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "device_id": "9a4d7b2f-2b2c-4f0a-9ec5-5e9b3a8c1001",
    "parameters": [
      {
        "parameter_path": "Device.DeviceInfo.SoftwareVersion",
        "parameter_value": "BaiBS_QRTB_2.9",
        "parameter_type": "string",
        "writable": false,
        "last_updated_at": "2026-07-30T00:59:00Z"
      }
    ],
    "total": 1
  }
}`,
  },
  {
    key: 'nb-push-target-list',
    apiKind: '正式北向',
    module: 'HTTP Push',
    name: 'Push 目标查询',
    method: 'GET',
    url: '/api/v1/northbound/push/targets',
    auth: commonApiAuth,
    backendSource: 'omcgo/internal/northbound/router.go + internal/northbound/push',
    fieldContract: '完整返回字段',
    responseFields: [...currentEnvelopeFields, 'data.items[].id', 'data.items[].url', 'data.items[].auth_type', 'data.items[].auth_token', 'data.items[].data_types', 'data.items[].format', 'data.items[].batch_size', 'data.items[].retry_count', 'data.items[].enabled', 'data.total'],
    requestExample: `GET /api/v1/northbound/push/targets
X-Northbound-Token: <access-token>`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "items": [
      {
        "id": "oss-primary",
        "url": "https://oss.example.com/nbi/events",
        "auth_type": "bearer",
        "data_types": ["alarm", "pm"],
        "format": "json",
        "batch_size": 100,
        "retry_count": 3,
        "enabled": false
      }
    ],
    "total": 1
  }
}`,
  },
  {
    key: 'nb-push-target-create',
    apiKind: '正式北向',
    module: 'HTTP Push',
    name: '新增 Push 目标',
    method: 'POST',
    url: '/api/v1/northbound/push/targets',
    auth: commonApiAuth,
    backendSource: 'omcgo/internal/northbound/router.go',
    fieldContract: '完整返回字段',
    responseFields: [...currentEnvelopeFields, 'data.message', 'data.id'],
    requestExample: `POST /api/v1/northbound/push/targets
X-Northbound-Token: <access-token>
Content-Type: application/json

{
  "id": "oss-primary",
  "url": "https://oss.example.com/nbi/events",
  "auth_type": "bearer",
  "auth_token": "<token>",
  "data_types": ["alarm", "pm"],
  "format": "json",
  "batch_size": 100,
  "retry_count": 3,
  "enabled": false
}`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "message": "push target added",
    "id": "oss-primary"
  }
}`,
  },
  {
    key: 'nb-push-target-delete',
    apiKind: '正式北向',
    module: 'HTTP Push',
    name: '删除 Push 目标',
    method: 'DELETE',
    url: '/api/v1/northbound/push/targets/{id}',
    auth: commonApiAuth,
    backendSource: 'omcgo/internal/northbound/router.go',
    fieldContract: '完整返回字段',
    responseFields: [...currentEnvelopeFields, 'data.id'],
    requestExample: `DELETE /api/v1/northbound/push/targets/oss-primary
X-Northbound-Token: <access-token>`,
    responseExample: `{
  "ret": 1,
  "msg": "push target removed",
  "data": {
    "id": "oss-primary"
  }
}`,
  },
  {
    key: 'nb-push-circuit',
    apiKind: '正式北向',
    module: 'HTTP Push',
    name: 'Push 熔断状态',
    method: 'GET',
    url: '/api/v1/northbound/push/targets/{id}/circuit',
    auth: commonApiAuth,
    backendSource: 'omcgo/internal/northbound/router.go',
    fieldContract: '完整返回字段',
    responseFields: [...currentEnvelopeFields, 'data.target_id', 'data.state', 'data.failure_count', 'data.threshold'],
    requestExample: `GET /api/v1/northbound/push/targets/oss-primary/circuit
X-Northbound-Token: <access-token>`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "target_id": "oss-primary",
    "state": "closed",
    "failure_count": 0,
    "threshold": 5
  }
}`,
  },
  {
    key: 'nb-push-circuit-reset',
    apiKind: '正式北向',
    module: 'HTTP Push',
    name: 'Push 熔断复位',
    method: 'POST',
    url: '/api/v1/northbound/push/targets/{id}/circuit/reset',
    auth: commonApiAuth,
    backendSource: 'omcgo/internal/northbound/router.go',
    fieldContract: '完整返回字段',
    responseFields: [...currentEnvelopeFields, 'data.target_id', 'data.state'],
    requestExample: `POST /api/v1/northbound/push/targets/oss-primary/circuit/reset
X-Northbound-Token: <access-token>`,
    responseExample: `{
  "ret": 1,
  "msg": "circuit breaker reset",
  "data": {
    "target_id": "oss-primary",
    "state": "closed"
  }
}`,
  },
  {
    key: 'nb-push-deadletter',
    apiKind: '正式北向',
    module: 'HTTP Push',
    name: '死信列表',
    method: 'GET',
    url: '/api/v1/northbound/push/deadletter?limit=20&offset=0',
    auth: commonApiAuth,
    backendSource: 'omcgo/internal/northbound/router.go + internal/northbound/push/outbox.go',
    fieldContract: '完整返回字段',
    responseFields: [...currentEnvelopeFields, 'data.items[].id', 'data.items[].event_id', 'data.items[].subject', 'data.items[].payload', 'data.items[].target_id', 'data.items[].status', 'data.items[].attempts', 'data.items[].max_attempts', 'data.items[].last_error?', 'data.items[].next_retry_at', 'data.items[].created_at', 'data.items[].updated_at', 'data.total'],
    requestExample: `GET /api/v1/northbound/push/deadletter?limit=20&offset=0
X-Northbound-Token: <access-token>`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "items": [
      {
        "id": "4a65c2aa-086b-4c1b-a3b7-000100010001",
        "event_id": "alarm-20260730005800-0001",
        "subject": "alarm.raised",
        "target_id": "oss-primary",
        "status": "dead",
        "attempts": 3,
        "max_attempts": 3,
        "last_error": "connect timeout"
      }
    ],
    "total": 1
  }
}`,
  },
  {
    key: 'nb-push-deadletter-replay',
    apiKind: '正式北向',
    module: 'HTTP Push',
    name: '死信重放',
    method: 'POST',
    url: '/api/v1/northbound/push/deadletter/{id}/replay',
    auth: commonApiAuth,
    backendSource: 'omcgo/internal/northbound/router.go + internal/northbound/push/outbox.go',
    fieldContract: '完整返回字段',
    responseFields: [...currentEnvelopeFields, 'data.id'],
    requestExample: `POST /api/v1/northbound/push/deadletter/4a65c2aa-086b-4c1b-a3b7-000100010001/replay
X-Northbound-Token: <access-token>`,
    responseExample: `{
  "ret": 1,
  "msg": "dead letter replayed",
  "data": {
    "id": "4a65c2aa-086b-4c1b-a3b7-000100010001"
  }
}`,
  },
  {
    key: 'nb-server-list',
    apiKind: '正式北向',
    module: '主备服务',
    name: '北向服务器查询',
    method: 'GET',
    url: '/api/v1/northbound/servers',
    auth: commonApiAuth,
    backendSource: 'omcgo/internal/northbound/server_handler.go',
    fieldContract: '完整返回字段',
    responseFields: [...currentEnvelopeFields, 'data.items[].id', 'data.items[].role', 'data.items[].host', 'data.items[].port', 'data.items[].description', 'data.items[].is_active', 'data.items[].created_at', 'data.items[].updated_at'],
    requestExample: `GET /api/v1/northbound/servers
X-Northbound-Token: <access-token>`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "items": [
      {
        "role": "primary",
        "host": "10.10.21.18",
        "port": 9443,
        "description": "主用 OSS",
        "is_active": true
      }
    ]
  }
}`,
  },
  {
    key: 'nb-server-active',
    apiKind: '正式北向',
    module: '主备服务',
    name: '切换主备服务器',
    method: 'PUT',
    url: '/api/v1/northbound/servers/active',
    auth: commonApiAuth,
    backendSource: 'omcgo/internal/northbound/server_handler.go',
    fieldContract: '完整返回字段',
    responseFields: [...currentEnvelopeFields, 'data.role'],
    requestExample: `PUT /api/v1/northbound/servers/active
X-Northbound-Token: <access-token>
Content-Type: application/json

{
  "role": "standby"
}`,
    responseExample: `{
  "ret": 1,
  "msg": "active northbound server switched",
  "data": {
    "role": "standby"
  }
}`,
  },
  {
    key: 'nb-server-update',
    apiKind: '正式北向',
    module: '主备服务',
    name: '编辑北向服务器',
    method: 'PUT',
    url: '/api/v1/northbound/servers/{role}',
    auth: commonApiAuth,
    backendSource: 'omcgo/internal/northbound/server_handler.go',
    fieldContract: '完整返回字段',
    responseFields: [...currentEnvelopeFields, 'data.role'],
    requestExample: `PUT /api/v1/northbound/servers/primary
X-Northbound-Token: <access-token>
Content-Type: application/json

{
  "host": "10.10.21.18",
  "port": 9443,
  "description": "主用 OSS"
}`,
    responseExample: `{
  "ret": 1,
  "msg": "northbound server updated",
  "data": {
    "role": "primary"
  }
}`,
  },
  {
    key: 'device-list',
    module: '设备',
    name: '设备查询：按SN/制式/分组分页查询',
    method: 'POST',
    url: '/api/v1/northbound/v1/device/query',
    auth: commonApiAuth,
    backendSource: 'omcgo/internal/northbound/legacy_facade.go',
    requestExample: `POST /api/v1/northbound/v1/device/query
X-Northbound-Token: <access-token>
Content-Type: application/json

{
  "page": 1,
  "rows": 20,
  "sn": "1202000240194",
  "technology": "LTE"
}`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "items": [
      {
        "id": "9a4d7b2f-2b2c-4f0a-9ec5-5e9b3a8c1001",
        "serial_number": "1202000240194",
        "product_class": "FAP-LTE-100",
        "technology": "LTE",
        "is_online": true,
        "lifecycle_state": "commissioned"
      }
    ],
    "total": 1,
    "page": 1,
    "page_size": 20
  }
}`,
  },
  {
    key: 'device-detail',
    module: '设备',
    name: '设备详情：基础信息与扩展信息',
    method: 'GET',
    url: '/api/v1/northbound/v1/device/infos/{sn}',
    auth: commonApiAuth,
    backendSource: 'omcgo/internal/northbound/legacy_facade.go',
    requestExample: `GET /api/v1/northbound/v1/device/infos/1202000240194
X-Northbound-Token: <access-token>`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "serial_number": "1202000240194",
    "device_name": "Site-A-1",
    "product_class": "FAP-LTE-100",
    "is_online": true,
    "technology": "LTE"
  }
}`,
  },
  {
    key: 'inventory-export',
    module: 'Inventory',
    name: '设备清单导出',
    method: 'GET',
    url: '/api/v1/devices/export?carrier=CMCC&status=online',
    auth: 'JWT 或 API Key，权限 scope=devices:export',
    backendSource: 'omcgo/internal/device/export_handler.go',
    requestExample: `GET /api/v1/devices/export?carrier=CMCC&status=online
Accept: text/csv
Authorization: Bearer <access-token>`,
    responseExample: `HTTP/1.1 200 OK
Content-Type: text/csv; charset=utf-8
Content-Disposition: attachment; filename=devices_20260730_010000.csv

serial_number,product_class,site_name,status,last_inform_at
1202000240194,FAP-LTE-100,SH-001,online,2026-07-30T01:00:00+08:00`,
  },
  {
    key: 'parameter-tree',
    module: '配置',
    name: '参数快照：读取系统已保存参数',
    method: 'GET',
    url: '/api/v1/northbound/v1/device/parameters/{sn}',
    auth: commonApiAuth,
    backendSource: 'omcgo/internal/northbound/legacy_facade.go',
    requestExample: `GET /api/v1/northbound/v1/device/parameters/1202000240194
X-Northbound-Token: <access-token>`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "sn": "1202000240194",
    "parameters": {
      "Device.DeviceInfo.SoftwareVersion": "BaiBS_QRTB_2.9"
    },
    "items": [
      {
        "parameter_path": "Device.DeviceInfo.SoftwareVersion",
        "parameter_value": "BaiBS_QRTB_2.9",
        "parameter_type": "string",
        "writable": false
      }
    ],
    "total": 1
  }
}`,
  },
  {
    key: 'parameter-set',
    module: '配置',
    name: '参数下发：异步设置设备参数',
    method: 'PUT',
    url: '/api/v1/northbound/v1/device/parameters/{sn}',
    auth: commonApiAuth,
    backendSource: 'omcgo/internal/northbound/legacy_facade.go',
    requestExample: `PUT /api/v1/northbound/v1/device/parameters/1202000240194
X-Northbound-Token: <access-token>
Content-Type: application/json

{
  "parameters": [
    {
      "path": "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.PhyCellID",
      "value": "100",
      "type": "unsignedInt"
    }
  ]
}`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "message": "set parameter values command queued",
    "parameters": 1,
    "task_id": "cmd-20260730010000-0001",
    "jobId": "cmd-20260730010000-0001",
    "sn": "1202000240194",
    "waitTime": 5
  }
}`,
  },
  {
    key: 'config-pull',
    module: '配置',
    name: '参数拉取：异步从设备读取指定路径',
    method: 'POST',
    url: '/api/v1/northbound/v1/device/parameters/query/{sn}',
    auth: commonApiAuth,
    backendSource: 'omcgo/internal/northbound/legacy_facade.go',
    requestExample: `POST /api/v1/northbound/v1/device/parameters/query/1202000240194
X-Northbound-Token: <access-token>
Content-Type: application/json

{
  "parameter_names": [
    "Device.DeviceInfo.SoftwareVersion",
    "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.PhyCellID"
  ]
}`,
    responseExample: `{
  "ret": 1,
  "msg": "configuration pull queued",
  "data": {
    "jobId": "northbound-manual:7b2e...",
    "sn": "1202000240194",
    "waitTime": 5,
    "task_count": 1,
    "status": "queued"
  }
}`,
  },
  {
    key: 'task-detail',
    module: '任务',
    name: '任务结果查询：按jobId查看异步结果',
    method: 'GET',
    url: '/api/v1/northbound/v1/job/result/{jobId}',
    auth: commonApiAuth,
    backendSource: 'omcgo/internal/northbound/legacy_facade.go + internal/task/service.go',
    requestExample: `GET /api/v1/northbound/v1/job/result/cmd-20260730010000-0001
X-Northbound-Token: <access-token>`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "id": "cmd-20260730010000-0001",
    "jobId": "cmd-20260730010000-0001",
    "device_sn": "1202000240194",
    "method": "SetParameterValues",
    "status": "completed",
    "legacy_status": "2",
    "result_code": "0",
    "completed_at": "2026-07-30T01:00:04+08:00"
  }
}`,
  },
  {
    key: 'device-reboot',
    module: '设备',
    name: '设备重启：下发Reboot任务',
    method: 'POST',
    url: '/api/v1/northbound/v1/device/reboot/{sn}',
    auth: commonApiAuth,
    backendSource: 'omcgo/internal/northbound/legacy_facade.go',
    requestExample: `POST /api/v1/northbound/v1/device/reboot/1202000240194
X-Northbound-Token: <access-token>`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "message": "reboot command queued",
    "task_id": "cmd-20260730010000-0002",
    "jobId": "cmd-20260730010000-0002",
    "sn": "1202000240194",
    "waitTime": 5
  }
}`,
  },
  {
    key: 'alarm-active',
    module: '告警',
    name: '活动告警查询',
    method: 'GET',
    url: '/api/v1/alarms/active?page=1&page_size=20&severity=major&device_sn=1202000240194',
    auth: 'JWT 或 API Key，权限 scope=alarms:read',
    backendSource: 'omcgo/internal/alarm/handler.go',
    requestExample: `GET /api/v1/alarms/active?page=1&page_size=20&severity=major&device_sn=1202000240194
Authorization: Bearer <access-token>`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "items": [
      {
        "id": "7adbb9f5-8e54-4882-a06f-c4d0f0c30001",
        "device_sn": "1202000240194",
        "severity": "major",
        "alarm_type": "mmeDisconnected",
        "occurred_at": "2026-07-30T00:58:00+08:00"
      }
    ],
    "total": 1
  }
}`,
  },
  {
    key: 'alarm-statistics',
    module: '告警',
    name: '告警统计查询',
    method: 'GET',
    url: '/api/v1/alarms/statistics?device_sn=1202000240194&start_time=2026-07-30T00:00:00%2B08:00&end_time=2026-07-30T01:00:00%2B08:00',
    auth: 'JWT 或 API Key，权限 scope=alarms:read',
    backendSource: 'omcgo/internal/alarm/handler.go',
    requestExample: `GET /api/v1/alarms/statistics?device_sn=1202000240194&start_time=2026-07-30T00:00:00%2B08:00&end_time=2026-07-30T01:00:00%2B08:00
Authorization: Bearer <access-token>`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "total": 12,
    "by_severity": {
      "critical": 1,
      "major": 3,
      "minor": 8
    }
  }
}`,
  },
  {
    key: 'pm-aggregated',
    module: 'PM',
    name: 'PM 聚合指标查询',
    method: 'GET',
    url: '/api/v1/pm/metrics/aggregated?granularity=15min&dimension=device&device_sn=1202000240194&metric_paths=KGNB0101,K001&technology=GNB&start_time=2026-07-30T00:00:00%2B08:00&end_time=2026-07-30T01:00:00%2B08:00',
    auth: 'JWT 或 API Key，权限 scope=pm:read',
    backendSource: 'omcgo/internal/pm/handler.go',
    requestExample: `GET /api/v1/pm/metrics/aggregated?granularity=15min&dimension=device&device_sn=1202000240194&metric_paths=KGNB0101,K001&technology=GNB
Authorization: Bearer <access-token>`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "items": [
      {
        "device_sn": "1202000240194",
        "metric_path": "KGNB0101",
        "value": 98.7,
        "bucket_start": "2026-07-30T00:45:00+08:00"
      }
    ],
    "total": 1
  }
}`,
  },
  {
    key: 'pm-export',
    module: 'PM',
    name: 'PM 导出任务创建',
    method: 'POST',
    url: '/api/v1/pm/exports',
    auth: 'JWT 或 API Key，权限 scope=pm:export',
    backendSource: 'omcgo/internal/pm/export/handler.go',
    requestExample: `POST /api/v1/pm/exports
Content-Type: application/json

{
  "source_type": "kpi_query",
  "task_name": "北向_PM_20260730010000",
  "params": {
    "granularity": "15min",
    "dimension": "device",
    "device_sns": ["1202000240194"],
    "metric_paths": ["KGNB0101", "K001"],
    "technologies": ["GNB"],
    "start_time": "2026-07-30T00:00:00+08:00",
    "end_time": "2026-07-30T01:00:00+08:00"
  }
}`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "id": "39a1f836-e8a7-4b5a-a8e4-01e2e0ad0001",
    "task_name": "北向_PM_20260730010000",
    "source_type": "kpi_query",
    "format": "csv",
    "status": "pending",
    "row_count": 0,
    "file_size": 0
  }
}`,
  },
  {
    key: 'mr-data',
    module: 'MR',
    name: 'MR 数据查询',
    method: 'GET',
    url: '/api/v1/mr/data?device_id={id}&mr_type=MRO&start_time=2026-07-30T00:00:00%2B08:00&end_time=2026-07-30T01:00:00%2B08:00',
    auth: 'JWT 或 API Key，权限 scope=pm:read',
    backendSource: 'omcgo/internal/mr/handler.go',
    requestExample: `GET /api/v1/mr/data?device_id=9a4d7b2f-2b2c-4f0a-9ec5-5e9b3a8c1001&mr_type=MRO&page=1&page_size=20
Authorization: Bearer <access-token>`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "items": [
      {
        "device_id": "9a4d7b2f-2b2c-4f0a-9ec5-5e9b3a8c1001",
        "mr_type": "MRO",
        "cell_id": "1",
        "collect_time": "2026-07-30T00:45:00+08:00"
      }
    ],
    "total": 1
  }
}`,
  },
  {
    key: 'mr-export',
    module: 'MR',
    name: 'MR 数据导出',
    method: 'POST',
    url: '/api/v1/mr/export',
    auth: 'JWT 或 API Key，权限 scope=pm:export',
    backendSource: 'omcgo/internal/mr/handler.go',
    requestExample: `POST /api/v1/mr/export
Content-Type: application/json

{
  "device_id": "9a4d7b2f-2b2c-4f0a-9ec5-5e9b3a8c1001",
  "mr_type": "MRO",
  "start_time": "2026-07-30T00:00:00+08:00",
  "end_time": "2026-07-30T01:00:00+08:00",
  "format": "csv"
}`,
    responseExample: `HTTP/1.1 200 OK
Content-Type: text/csv; charset=utf-8
Content-Disposition: attachment; filename=mr_export_20260730_010000.csv

device_id,mr_type,cell_id,collect_time
9a4d7b2f-2b2c-4f0a-9ec5-5e9b3a8c1001,MRO,1,2026-07-30T00:45:00+08:00`,
  },
];

const apiMetaByKey: Record<string, Partial<NorthboundApiRow>> = {
  'device-list': {
    apiKind: '业务复用',
    fieldContract: '当前返回字段',
    responseFields: [...listResponseFields, 'data.items[].id', 'data.items[].serial_number', 'data.items[].oui', 'data.items[].product_class', 'data.items[].manufacturer', 'data.items[].model_name', 'data.items[].carrier', 'data.items[].technology', 'data.items[].lifecycle_state', 'data.items[].is_online', 'data.items[].firmware_version', 'data.items[].ip_address', 'data.items[].device_name', 'data.items[].site_id', 'data.items[].last_inform_at?', 'data.items[].group_name?'],
  },
  'device-detail': {
    apiKind: '业务复用',
    fieldContract: '当前返回字段',
    responseFields: [...currentEnvelopeFields, 'data.device', 'data.device_info', 'data.parameters?', 'data.alarms?'],
  },
  'inventory-export': {
    apiKind: '业务复用',
    fieldContract: '当前返回字段',
    responseFields: ['HTTP 200 CSV stream', 'Content-Type', 'Content-Disposition', 'CSV columns depend on export service'],
  },
  'parameter-tree': {
    apiKind: '业务复用',
    fieldContract: '当前返回字段',
    responseFields: [...currentEnvelopeFields, 'data.name', 'data.full_path', 'data.is_leaf', 'data.children[]', 'data.children[].name', 'data.children[].full_path', 'data.children[].value?', 'data.children[].type?', 'data.children[].writable?'],
  },
  'parameter-set': {
    apiKind: '业务复用',
    fieldContract: '当前返回字段',
    responseFields: [...currentEnvelopeFields, 'data.message', 'data.parameters', 'data.reboot_required', 'data.task_id'],
  },
  'config-pull': {
    apiKind: '业务复用',
    fieldContract: '当前返回字段',
    responseFields: [...currentEnvelopeFields, 'data.device_id', 'data.command_id', 'data.request_id', 'data.batch_count', 'data.status'],
  },
  'task-detail': {
    apiKind: '业务复用',
    fieldContract: '当前返回字段',
    responseFields: [...currentEnvelopeFields, 'data.id', 'data.device_id?', 'data.device_sn?', 'data.method?', 'data.status', 'data.result_code?', 'data.error_message?', 'data.created_at?', 'data.completed_at?'],
  },
  'device-reboot': {
    apiKind: '业务复用',
    fieldContract: '当前返回字段',
    responseFields: [...currentEnvelopeFields, 'data.message'],
  },
  'alarm-active': {
    apiKind: '业务复用',
    fieldContract: '当前返回字段',
    responseFields: [...listResponseFields, 'data.items[].id', 'data.items[].device_sn', 'data.items[].severity', 'data.items[].alarm_type', 'data.items[].alarm_identifier', 'data.items[].description', 'data.items[].status', 'data.items[].raised_at'],
  },
  'alarm-statistics': {
    apiKind: '业务复用',
    fieldContract: '当前返回字段',
    responseFields: [...currentEnvelopeFields, 'data.total', 'data.by_severity'],
  },
  'pm-aggregated': {
    apiKind: '业务复用',
    fieldContract: '当前返回字段',
    responseFields: [...listResponseFields, 'data.items[].device_sn?', 'data.items[].metric_path', 'data.items[].value', 'data.items[].bucket_start', 'data.items[].technology?'],
  },
  'pm-export': {
    apiKind: '业务复用',
    fieldContract: '当前返回字段',
    responseFields: [...currentEnvelopeFields, 'data.id', 'data.task_name', 'data.source_type', 'data.format', 'data.status', 'data.row_count', 'data.file_size'],
  },
  'mr-data': {
    apiKind: '业务复用',
    fieldContract: '当前返回字段',
    responseFields: [...listResponseFields, 'data.items[].device_id', 'data.items[].mr_type', 'data.items[].cell_id?', 'data.items[].collect_time?'],
  },
  'mr-export': {
    apiKind: '业务复用',
    fieldContract: '当前返回字段',
    responseFields: ['HTTP 200 CSV stream', 'Content-Type', 'Content-Disposition', 'CSV columns depend on MR export service'],
  },
};

const apiDisplayAliases: Record<string, NorthboundApiDisplayAlias[]> = {};

const legacySupportedApiKeys = new Set([
  'auth-login',
  'nb-sync-full-device',
  'nb-export-config',
  'device-list',
  'device-status',
  'device-detail',
  'device-register-create',
  'device-register-list',
  'device-register-delete',
  'device-group-tree',
  'device-group-create',
  'device-group-update',
  'device-group-delete',
  'device-group-add-devices',
  'parameter-tree',
  'parameter-set',
  'parameter-cellname',
  'config-pull',
  'task-detail',
  'task-page',
  'device-task-create',
  'device-reboot',
  'device-reset',
  'device-log-collect',
]);

function apiConfigKey(row: NorthboundApiRow): string {
  return row.configKey ?? row.key;
}

function apiConfigKeys(rows: NorthboundApiRow[]): string[] {
  return Array.from(new Set(rows.map(apiConfigKey)));
}

function expandApiDisplayRows(rows: NorthboundApiRow[]): NorthboundApiRow[] {
  return rows.flatMap((row) => {
    const aliases = apiDisplayAliases[row.key];
    if (!aliases || aliases.length === 0) {
      return [{ ...row, configKey: row.configKey ?? row.key }];
    }
    return aliases.map((alias) => ({
      ...row,
      ...alias,
      configKey: row.key,
      apiKind: alias.apiKind ?? row.apiKind,
      module: alias.module ?? row.module,
      auth: alias.auth ?? row.auth,
      backendSource: alias.backendSource ?? row.backendSource,
      fieldContract: alias.fieldContract ?? row.fieldContract,
      responseFields: alias.responseFields ?? row.responseFields,
    }));
  });
}

const legacySupportedApiRows = expandApiDisplayRows(northboundApiRows.filter((row) => legacySupportedApiKeys.has(row.key)));

const defaultApiEnabled = Object.fromEntries(
  apiConfigKeys(legacySupportedApiRows).map((key) => [key, false]),
) as Record<string, boolean>;

const defaultEditorPeriodRows: ScenarioPeriodRow[] = [
  {
    key: 'cm-default',
    domain: 'CM',
    scope: 'LTE',
    format: 'XML',
    period: '24H',
    trigger: '每天 00:01:30',
    cron: '30 1 0 * * ?',
    objects: 'CP, EP, CC, CE',
    path: PATH_CM,
    fileName: 'Baicells-{CP|EP|CC|CE}-#LocalHost#-#DataVersion#-#DateTime#[-#Ri#][-#FileID#]',
    compressionEnabled: false,
    compressionFormat: 'zip',
  },
  {
    key: 'pm-default',
    domain: 'PM',
    scope: 'LTE',
    format: 'CSV',
    period: '15M',
    trigger: '每 15 分钟，05 分起',
    cron: '0 5/15 * * * ?',
    objects: 'PC',
    path: PATH_PM,
    fileName: 'Baicells-PC-#LocalHost#-#DataVersion#-#DateTime#[-#Ri#]-#DataPeriod#[-#FileID#]',
    compressionEnabled: false,
    compressionFormat: 'zip',
  },
  {
    key: 'mr-default',
    domain: 'MR',
    scope: '默认',
    format: 'XML',
    period: '15M',
    trigger: '每 15 分钟，00 分起',
    cron: '0 0/15 * * * ?',
    objects: 'MRO, MRE, MRS',
    path: PATH_MR,
    fileName: '#ModuleType#-Baicells-{MRO|MRE|MRS}-#LocalHost#-#eNBID#-#DateTime#[-#Ri#].xml',
    compressionEnabled: false,
    compressionFormat: 'zip',
  },
];

function clonePeriodRows(rows: ScenarioPeriodRow[]): ScenarioPeriodRow[] {
  return rows.map((row) => ({ ...row }));
}

function splitObjects(value: string): string[] {
  return value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);
}

function joinObjects(values: string[]): string {
  return values.join(', ');
}

function getObjectOptions(domain: Domain, currentValue: string) {
  const baseOptions = objectOptionsByDomain[domain];
  const existingValues = new Set(baseOptions.map((option) => option.value));
  const extraOptions = splitObjects(currentValue)
    .filter((value) => !existingValues.has(value))
    .map((value) => ({ label: formatObjectToken(value), value }));
  return [...baseOptions, ...extraOptions];
}

function isTech(value: string | undefined): value is Tech {
  return value === 'LTE' || value === 'GNB' || value === 'GSM';
}

function isRadioTechDomain(domain: Domain | undefined): boolean {
  return domain === 'CM' || domain === 'PM';
}

function normalizeTechValue(value: string | undefined): Tech | undefined {
  const normalized = value?.trim().toUpperCase();
  if (!normalized) return undefined;
  if (normalized === 'ENB' || normalized === 'LTE') return 'LTE';
  if (normalized === 'GNB' || normalized === 'NR') return 'GNB';
  if (normalized === 'GSM') return 'GSM';
  return undefined;
}

function formatTechLabel(tech: Tech | undefined): string {
  if (tech === 'LTE') return 'ENB';
  if (tech === 'GNB') return 'GNB';
  if (tech === 'GSM') return 'GSM';
  return '通用';
}

function formatScopeLabel(scope: string): string {
  const tech = normalizeTechValue(scope);
  if (tech) return formatTechLabel(tech);
  return scope || '默认';
}

function getScopeOptions(domain: Domain) {
  if (isRadioTechDomain(domain)) return radioTechScopeOptions;
  if (domain === 'LOG') return logScopeOptions;
  return [defaultScopeOption];
}

function parseObjectToken(token: string): { code: string; tech?: Tech } {
  const [code, tech] = token.split('/').map((item) => item.trim());
  return {
    code,
    tech: normalizeTechValue(tech),
  };
}

function formatObjectToken(token: string): string {
  const parsed = parseObjectToken(token);
  if (!parsed.tech && logObjectLabelByCode[parsed.code]) return logObjectLabelByCode[parsed.code];
  return parsed.tech ? `${parsed.code} / ${formatTechLabel(parsed.tech)}` : parsed.code;
}

function objectProfileKey(objectCode: string, tech?: Tech): string {
  return `${objectCode.trim().toUpperCase()}:${tech ?? ''}`;
}

function getStoredObjectProfile(row: ScenarioPeriodRow, objectCode: string, tech?: Tech): string | undefined {
  if (!row.objectProfiles) return undefined;
  const exact = row.objectProfiles[objectProfileKey(objectCode, tech)];
  if (exact !== undefined) return exact;
  return row.objectProfiles[objectProfileKey(objectCode)];
}

function profileMatchesPeriodFormat(profile: string, format: Format): boolean {
  if (!profile) return true;
  const normalized = profile.toLowerCase();
  if (normalized.includes('.xml.')) return format === 'XML';
  if (normalized.includes('.csv.')) return format === 'CSV';
  return true;
}

function formatFieldTargetLabel(target: FieldTarget): string {
  const tech = getTargetTech(target);
  return tech ? `${target.objectCode} / ${formatTechLabel(tech)}` : target.objectCode;
}

function formatReportFieldObject(row: ReportFieldRow): string {
  if (row.domain === 'LOG' && logObjectLabelByCode[row.objectCode]) return logObjectLabelByCode[row.objectCode];
  return row.objectCode;
}

function resolveTechLabel(value?: string): Tech | undefined {
  return normalizeTechValue(value);
}

function getTargetTech(target: FieldTarget | undefined): Tech | undefined {
  return target?.tech ?? resolveTechLabel(target?.scope);
}

function getFieldTechFilterOptions(domain: Domain, targets: FieldTarget[]): Array<{ label: string; value: FieldTechFilter }> {
  if (!isRadioTechDomain(domain)) return nonRadioFieldTechFilterOptions;
  const availableTechs = new Set(
    targets
      .filter((target) => target.domain === domain)
      .map(getTargetTech)
      .filter(isTech),
  );
  if (availableTechs.size === 0) return radioFieldTechFilterOptions;
  return radioFieldTechFilterOptions.filter((option) => option.value !== 'ALL' && availableTechs.has(option.value));
}

function getFieldTechFilterForTarget(target: FieldTarget | undefined): FieldTechFilter {
  if (!target || !isRadioTechDomain(target.domain)) return 'ALL';
  return getTargetTech(target) ?? 'LTE';
}

function getFieldTechFilterForDomain(
  domain: Domain,
  targets: FieldTarget[],
  currentFilter?: FieldTechFilter,
): FieldTechFilter {
  if (!isRadioTechDomain(domain)) return 'ALL';
  if (currentFilter && currentFilter !== 'ALL' && targets.some((target) => (
    target.domain === domain && targetMatchesTechFilter(target, currentFilter)
  ))) {
    return currentFilter;
  }
  return getFieldTechFilterForTarget(targets.find((target) => target.domain === domain));
}

function getRowTech(row: ReportFieldRow): Tech | undefined {
  return row.tech ?? resolveTechLabel(row.scope);
}

function formatReportFieldTech(row: ReportFieldRow): string {
  return formatTechLabel(getRowTech(row));
}

function targetMatchesTechFilter(target: FieldTarget, filter: FieldTechFilter): boolean {
  if (filter === 'ALL') return true;
  return getTargetTech(target) === filter;
}

function rowMatchesTechFilter(row: ReportFieldRow, filter: FieldTechFilter): boolean {
  if (filter === 'ALL') return true;
  return getRowTech(row) === filter;
}

function getPmTechsFromScope(scope: string): Tech[] {
  const normalizedScope = normalizeTechValue(scope);
  if (normalizedScope) return [normalizedScope];
  const scopeText = scope.toUpperCase();
  const techs: Tech[] = [];
  if (scopeText.includes('LTE') || scopeText.includes('ENB')) techs.push('LTE');
  if (scopeText.includes('GNB')) techs.push('GNB');
  if (scopeText.includes('GSM')) techs.push('GSM');
  return techs.length > 0 ? [...new Set(techs)] : ['LTE'];
}

function resolveEditorProfile(row: ScenarioPeriodRow, objectCode: string, tech?: Tech): string {
  const storedProfile = getStoredObjectProfile(row, objectCode, tech);
  if (storedProfile !== undefined && profileMatchesPeriodFormat(storedProfile, row.format)) {
    return storedProfile;
  }
  const objectName = objectCode.toLowerCase();
  const format = row.format.toLowerCase();
  if (row.domain === 'CM' && row.scope === 'GNB') return `cm.${objectName}.gnb.${format}.v1`;
  if (row.domain === 'CM') return `cm.${objectName}.${format}.v1`;
  if (row.domain === 'PM' && objectCode === 'PC' && tech === 'GNB') return 'pm.pc.gnb.csv.v1';
  if (row.domain === 'PM' && objectCode === 'PC' && tech === 'GSM') return 'pm.pc.gsm.pmresult.csv.v1';
  if (row.domain === 'PM' && objectCode === 'PC' && row.fileName.toLowerCase().includes('pmresult')) return 'pm.pc.pmresult.csv.v1';
  if (row.domain === 'PM') return `pm.${objectName}.csv.v1`;
  if (row.domain === 'MR') return `mr.${objectName}.xml.v1`;
  if (row.domain === 'INVENTORY') return `inventory.${objectName}.csv.v1`;
  return `${row.domain.toLowerCase()}.${objectName}.${format}.v1`;
}

function getFieldTargets(rows: ScenarioPeriodRow[]): FieldTarget[] {
  const uniqueTargets = new Map<string, FieldTarget>();
  rows.forEach((row) => {
    splitObjects(row.objects).forEach((objectToken) => {
      const parsedObject = parseObjectToken(objectToken);
      const rowScopeTech = normalizeTechValue(row.scope);
      const techs = parsedObject.tech
        ? [parsedObject.tech]
        : row.domain === 'PM'
          ? getPmTechsFromScope(row.scope)
          : row.domain === 'CM' && rowScopeTech
            ? [rowScopeTech]
            : [undefined];
      techs.forEach((tech) => {
        const scope = tech ?? row.scope;
        const profile = resolveEditorProfile(row, parsedObject.code, tech);
        const key = `${row.domain}:${parsedObject.code}:${scope}:${profile}`;
        if (!uniqueTargets.has(key)) {
          uniqueTargets.set(key, {
            key,
            domain: row.domain,
            objectCode: parsedObject.code,
            scope,
            format: row.format,
            profile,
            tech,
          });
        }
      });
    });
  });
  return [...uniqueTargets.values()];
}

function getFirstFieldTarget(rows: ScenarioPeriodRow[]): FieldTarget | undefined {
  return getFieldTargets(rows)[0];
}

const productClassesByTech: Record<Tech, string[]> = {
  LTE: ['FAP-LTE-100', 'FAP/pBS41010/SC', 'FAP/MLN/SC'],
  GNB: ['gNB-100'],
  GSM: ['FAP/PGSM', 'FAP/BTS'],
};

function getTargetProductClasses(target: FieldTarget): string[] {
  const tech = getTargetTech(target);
  if (tech) return productClassesByTech[tech];
  return [allProductClassValue];
}

function buildFieldRows(
  domain: Domain,
  objectCode: string,
  definitions: FieldDefinition[],
  productClasses = [allProductClassValue],
  target?: FieldTarget,
): ReportFieldRow[] {
  return definitions.map((field, index) => ({
    ...field,
    key: `${domain}-${objectCode}-${field.systemField}-${index}`,
    domain,
    objectCode,
    scope: target?.scope,
    tech: target?.tech,
    productClasses: field.productClasses ?? productClasses,
    enabled: true,
  }));
}

function mapApiFieldToReportRow(
  field: NorthboundFieldDefinition,
  target: FieldTarget,
  index: number,
): ReportFieldRow {
  return {
    key: field.key || `${target.domain}-${target.objectCode}-${field.system_field}-${index}`,
    domain: target.domain,
    objectCode: target.objectCode,
    scope: target.scope,
    tech: isTech(field.tech) ? field.tech : target.tech,
    outputAlias: field.output_alias,
    systemField: field.system_field,
    source: field.source,
    dataType: field.data_type,
    renderer: field.renderer,
    productClasses: field.product_class
      ? field.product_class.split(',').map((item) => item.trim()).filter(Boolean)
      : getTargetProductClasses(target),
    metricType: field.metric_type === 'counter' || field.metric_type === 'kpi' ? field.metric_type : undefined,
    statisType: field.statis_type,
    unit: field.unit,
    cnName: field.cn_name,
    enabled: true,
  };
}

function getInventoryFieldRows(target: FieldTarget): ReportFieldRow[] {
  const { objectCode } = target;
  const normalizedType = objectCode.toUpperCase();
  const template: InventoryType = normalizedType === 'OMC'
    ? 'OMC'
    : normalizedType === 'GNB'
      ? 'GNB'
      : normalizedType === 'GSM'
        ? 'GSM'
        : 'ENB';
  return inventoryFields
    .filter((field) => field.template === template)
    .map((field) => ({
      key: `INVENTORY-${objectCode}-${field.key}`,
      domain: 'INVENTORY',
      objectCode,
      scope: target.scope,
      tech: target.tech,
      outputAlias: field.column,
      systemField: field.exportKey,
      source: field.source,
      dataType: field.dataType,
      renderer: field.renderer,
      productClasses: [allProductClassValue],
      enabled: true,
    }));
}

const pmFieldRowsCache = new WeakMap<PmMetric[], Map<string, ReportFieldRow[]>>();

function getPmFieldRows(target: FieldTarget, metrics: PmMetric[], limit?: number): ReportFieldRow[] {
  const tech = getTargetTech(target);
  if (!tech) return [];
  let metricCache = pmFieldRowsCache.get(metrics);
  if (!metricCache) {
    metricCache = new Map<string, ReportFieldRow[]>();
    pmFieldRowsCache.set(metrics, metricCache);
  }
  const cacheKey = `${target.objectCode}:${target.scope}:${target.profile}:${tech}:${limit ?? 'all'}`;
  const cachedRows = metricCache.get(cacheKey);
  if (cachedRows) return cachedRows;
  const exactProfileMetrics = metrics.filter((metric) => metric.tech === tech && metric.profile === target.profile);
  const profileMetrics = exactProfileMetrics.length > 0
    ? exactProfileMetrics
    : metrics.filter((metric) => metric.tech === tech && target.objectCode === 'PC');
  if (profileMetrics.length === 0) return [];
  const rows: ReportFieldRow[] = profileMetrics.map((metric) => ({
    key: `PM-${target.objectCode}-${metric.key}`,
    domain: 'PM' as const,
    objectCode: target.objectCode,
    scope: target.scope,
    tech: target.tech,
    outputAlias: metric.reportKey,
    systemField: metric.metricPath,
    source: `pm_metrics.${metric.statisType}`,
    dataType: 'number',
    renderer: metric.statisType,
    productClasses: productClassesByTech[tech],
    metricType: metric.metricType,
    statisType: metric.statisType,
    unit: metric.unit,
    cnName: metric.cnName,
    enabled: true,
  }));
  const resultRows = typeof limit === 'number' ? rows.slice(0, limit) : rows;
  metricCache.set(cacheKey, resultRows);
  return resultRows;
}

function getReportFieldRows(target: FieldTarget | undefined, metrics: PmMetric[]): ReportFieldRow[] {
  if (!target) return [];
  if (target.domain === 'CM') {
    return buildFieldRows(
      'CM',
      target.objectCode,
      [
        ...(cmFieldsByObject[target.objectCode] ?? []),
        ...(extraFieldDefinitionsByDomainObject.CM?.[target.objectCode] ?? []),
      ],
      getTargetProductClasses(target),
      target,
    );
  }
  if (target.domain === 'PM') return getPmFieldRows(target, metrics);
  if (target.domain === 'MR') return buildFieldRows('MR', target.objectCode, mrFieldsByObject[target.objectCode] ?? [], getTargetProductClasses(target), target);
  if (target.domain === 'LOG') return buildFieldRows('LOG', target.objectCode, logFieldsByObject[target.objectCode] ?? [], getTargetProductClasses(target), target);
  return getInventoryFieldRows(target);
}

function getExtraFieldRows(target: FieldTarget | undefined): ReportFieldRow[] {
  if (!target) return [];
  const definitions = extraFieldDefinitionsByDomainObject[target.domain]?.[target.objectCode] ?? [];
  return buildFieldRows(target.domain, target.objectCode, definitions, getTargetProductClasses(target), target);
}

function getDeviceInfoReportFieldRows(target: FieldTarget | undefined): ReportFieldRow[] {
  if (!target) return [];
  const normalizedObject = target.objectCode.toUpperCase();
  const supportsDeviceInfo = target.domain === 'CM'
    || (target.domain === 'INVENTORY' && ['ENB', 'GNB', 'GSM'].includes(normalizedObject));
  if (!supportsDeviceInfo) return [];
  const definitions = deviceInfoFieldDefinitions.map(([column, outputAlias, cnName, dataType, renderer]) => ({
    outputAlias,
    systemField: `device_info.${column}`,
    source: `device_info.${column}`,
    dataType,
    renderer,
    cnName,
  }));
  return buildFieldRows(target.domain, target.objectCode, definitions, getTargetProductClasses(target), target);
}

function dedupeFieldCandidateRows(rows: ReportFieldRow[]): ReportFieldRow[] {
  const seen = new Set<string>();
  return rows.filter((row) => {
    const id = fieldCandidateIdentity(row);
    if (seen.has(id)) return false;
    seen.add(id);
    return true;
  });
}

function getAvailableReportFieldRows(target: FieldTarget | undefined, metrics: PmMetric[]): ReportFieldRow[] {
  if (!target) return [];
  if (target.domain === 'PM') return getPmFieldRows(target, metrics);
  return dedupeFieldCandidateRows([
    ...getReportFieldRows(target, metrics),
    ...getExtraFieldRows(target),
    ...getDeviceInfoReportFieldRows(target),
  ]);
}

function fieldCandidateMatchesSearch(row: ReportFieldRow, searchText: string): boolean {
  const keyword = searchText.trim().toLowerCase();
  if (!keyword) return true;
  return [
    row.outputAlias,
    row.cnName,
    row.systemField,
    row.source,
    row.metricType,
    row.statisType,
    row.unit,
    row.dataType,
    row.renderer,
    getRowTech(row),
  ]
    .filter(Boolean)
    .some((value) => String(value).toLowerCase().includes(keyword));
}

function getVisibleFieldCandidateRows(rows: ReportFieldRow[], searchText: string): ReportFieldRow[] {
  const keyword = searchText.trim();
  const matchedRows = keyword ? rows.filter((row) => fieldCandidateMatchesSearch(row, keyword)) : rows;
  return matchedRows.slice(0, keyword ? fieldCandidateSearchOptionLimit : fieldCandidateDefaultOptionLimit);
}

function formatProductClasses(productClasses?: string[]): string {
  if (!productClasses || productClasses.length === 0 || productClasses.includes(allProductClassValue)) return '通用';
  if (productClasses.length <= 2) return productClasses.join(', ');
  return `${productClasses.slice(0, 2).join(', ')} +${productClasses.length - 2}`;
}

function fieldIdentity(row: ReportFieldRow): string {
  return `${row.domain}:${row.objectCode}:${row.outputAlias}:${row.systemField}`;
}

function fieldCandidateIdentity(row: ReportFieldRow): string {
  return `${row.domain}:${row.objectCode}:${row.systemField}`.toLowerCase();
}

// Backend FieldDefinition.Key format (catalog.go): `${domain}.${objectCode}.${systemField}`, lowercased.
// This is the persistent identifier stored in group.selected_fields and matched by the generator.
function fieldKey(row: ReportFieldRow): string {
  return `${row.domain}.${row.objectCode}.${row.systemField}`.toLowerCase();
}

// The identifier persisted in group.selected_fields. For CM/MR/LOG/Inventory this is the
// backend field Key; for PM the generator treats selected_fields as metric_path list, so PM
// rows serialize their metric_path (systemField) directly.
function selectionKey(row: ReportFieldRow): string {
  return row.domain === 'PM' ? row.systemField : fieldKey(row);
}

// Builds the editor's fieldRowsByTarget from a profile's persisted per-group selection.
// For each field target: if its group has an explicit selected_fields list, narrow the
// target's base rows to those keys; otherwise leave the entry unset so the editor falls
// back to fieldBaseRows (all built-in fields shown as ON).
function hydrateFieldRowsByTarget(
  periodRows: ScenarioPeriodRow[],
  metrics: PmMetric[],
): Record<string, ReportFieldRow[]> {
  const result: Record<string, ReportFieldRow[]> = {};
  getFieldTargets(periodRows).forEach((target) => {
    const ownerRow = periodRows.find((row) => periodRowTargetKeys(row).includes(target.key));
    const selection = ownerRow?.selectedFields;
    if (!selection || selection.length === 0) return;
    const wanted = new Set(selection);
    const baseRows = getReportFieldRows(target, metrics);
    const filtered = baseRows.filter((r) => wanted.has(selectionKey(r)));
    result[target.key] = filtered.length > 0 ? filtered : baseRows;
  });
  return result;
}

function defaultPeriodRow(domain: Domain = 'CM', includeObjects = true): ScenarioPeriodRow {
  const firstObjects = includeObjects
    ? objectOptionsByDomain[domain].slice(0, domain === 'CM' ? 4 : 1).map((option) => option.value)
    : [];
  const objectsValue = joinObjects(firstObjects);
  const periodByDomain: Record<Domain, string> = {
    CM: '24H',
    PM: '15M',
    MR: '15M',
    LOG: '24H',
    INVENTORY: '24H',
  };
  const scopeByDomain: Record<Domain, string> = {
    CM: 'LTE',
    PM: 'LTE',
    MR: '默认',
    LOG: 'custom',
    INVENTORY: '默认',
  };
  const period = periodByDomain[domain];
  const cron = defaultCronForPeriod(period);
  return {
    key: `custom-${Date.now()}`,
    domain,
    scope: scopeByDomain[domain],
    format: defaultFormatForSelection(domain, objectsValue),
    period,
    trigger: formatScheduleLabel(period, cron),
    cron,
    objects: objectsValue,
    path: defaultRemotePathByDomain[domain],
    fileName: domain === 'CM'
      ? 'Baicells-{CP|EP|CC|CE}-#LocalHost#-#DataVersion#-#DateTime#[-#Ri#][-#FileID#]'
      : domain === 'PM'
        ? 'Baicells-PC-#LocalHost#-#DataVersion#-#DateTime#[-#Ri#]-#DataPeriod#[-#FileID#]'
        : domain === 'MR'
          ? '#ModuleType#-Baicells-{MRO|MRE|MRS}-#LocalHost#-#eNBID#-#DateTime#[-#Ri#].xml'
        : domain === 'LOG'
            ? LOG_CUSTOM_NAME
            : INVENTORY_NAME,
    compressionEnabled: domain === 'LOG',
    compressionFormat: domain === 'LOG' ? 'gz' : 'zip',
  };
}

function getScenarioObjectTech(groupItem: FileGroup, object: ScenarioObject): Tech | undefined {
  const tech = normalizeTechValue(object.tech);
  if (tech) return tech;
  if (groupItem.domain === 'CM' || groupItem.domain === 'PM') return 'LTE';
  return undefined;
}

function splitFileGroupByTech(groupItem: FileGroup): FileGroup[] {
  if (groupItem.domain !== 'CM' && groupItem.domain !== 'PM') return [groupItem];
  const objectTechPairs = groupItem.objects.map((object) => ({
    object,
    tech: getScenarioObjectTech(groupItem, object),
  }));
  const techs = [...new Set(objectTechPairs.map((pair) => pair.tech).filter(isTech))];
  if (techs.length <= 1) return [groupItem];
  return techs.map((tech) => ({
    ...groupItem,
    id: `${groupItem.id}-${tech.toLowerCase()}`,
    objects: objectTechPairs
      .filter((pair) => pair.tech === tech)
      .map((pair) => ({ ...pair.object, tech })),
  }));
}

function getGroupTechLabel(groupItem: FileGroup): string {
  const techs = [...new Set(groupItem.objects.map((object) => getScenarioObjectTech(groupItem, object)).filter(isTech))];
  if (techs.length > 0) return techs[0];
  return '默认';
}

function getGroupObjectsLabel(groupItem: FileGroup): string {
  return [...new Set(groupItem.objects.map((object) => object.code))].join(', ');
}

function getGroupObjectProfiles(groupItem: FileGroup): Record<string, string> | undefined {
  const profiles: Record<string, string> = {};
  groupItem.objects.forEach((object) => {
    const tech = getScenarioObjectTech(groupItem, object);
    const profile = object.profile ?? '';
    profiles[objectProfileKey(object.code, tech)] = profile;
    if (profiles[objectProfileKey(object.code)] === undefined) {
      profiles[objectProfileKey(object.code)] = profile;
    }
  });
  return Object.keys(profiles).length > 0 ? profiles : undefined;
}

function getGroupFileNameTemplate(groupItem: FileGroup): string {
  const objectNames = groupItem.objects.map((object) => object.code);
  if (groupItem.name.includes('#Object#')) {
    const objectToken = objectNames.length > 1 ? `{${objectNames.join('|')}}` : objectNames[0];
    return groupItem.name.replace('#Object#', objectToken);
  }
  return groupItem.name;
}

const outputExtensionByFormat: Record<Format, string> = {
  XML: 'xml',
  CSV: 'csv',
  TXT: 'txt',
};

function stripOptionalTemplateSegments(template: string): string {
  return template.replace(/\[-#[^\]]+#\]/g, '');
}

function pickFirstTemplateChoice(template: string): string {
  return template.replace(/\{([^{}|]+)(?:\|[^{}]*)*\}/g, '$1');
}

function getFirstObjectCode(row: ScenarioPeriodRow): string {
  const [firstObject] = splitObjects(row.objects);
  if (!firstObject) return row.domain;
  return parseObjectToken(firstObject).code;
}

function getPreviewDataPeriod(period: string): string {
  if (period === '15M') return '15';
  if (period === '60M') return '60';
  if (period === '24H') return '1440';
  if (period === '7D') return '10080';
  if (period === '1MO') return '43200';
  return period.replace(/\D/g, '') || period;
}

function formatPeriodLabel(period: string): string {
  if (period === '15M') return '15 分钟';
  if (period === '60M') return '60 分钟';
  if (period === '24H') return '每日';
  if (period === '7D') return '7 天';
  if (period === '1MO') return '每月';
  return period;
}

function getPeriodMinutes(period: string): number {
  if (period === '1MO') return 30 * 24 * 60;
  if (period === '7D') return 7 * 24 * 60;
  if (period === '24H') return 24 * 60;
  if (period === '60M') return 60;
  if (period === '15M') return 15;
  return Number(period.replace(/\D/g, '')) || 15;
}

function formatRecurringPeriodLabel(period: string): string {
  if (period === '24H') return '每日';
  if (period === '7D') return '每 7 天';
  if (period === '1MO') return '每月';
  return `每 ${getPeriodMinutes(period)} 分钟`;
}

function getHealthGraceMinutes(periodMinutes: number): number {
  return Math.min(Math.max(Math.ceil(periodMinutes * 0.25), 30), 24 * 60);
}

function getHealthWindowMs(periodMinutes: number): number {
  return (periodMinutes + getHealthGraceMinutes(periodMinutes)) * 60 * 1000;
}

function formatPeriodMinutesForHealth(minutes: number): string {
  if (minutes >= 30 * 24 * 60) return '每月';
  if (minutes >= 7 * 24 * 60) return '每 7 天';
  if (minutes >= 24 * 60) return '每日';
  if (minutes >= 60 && minutes % 60 === 0) return `每 ${minutes / 60} 小时`;
  return `每 ${minutes} 分钟`;
}

function profileTimestampMs(...values: Array<string | undefined>): number | undefined {
  const parsed = values
    .map((value) => {
      if (!value) return NaN;
      const ms = Date.parse(value);
      const date = new Date(ms);
      if (!Number.isFinite(ms) || date.getUTCFullYear() < 2000) return NaN;
      return ms;
    })
    .filter(Number.isFinite);
  return parsed.length > 0 ? Math.max(...parsed) : undefined;
}

function latestSuccessRunMap(items: NorthboundFileRun[]): Record<string, NorthboundFileRun> {
  return items.reduce<Record<string, NorthboundFileRun>>((acc, run) => {
    const key = run.profile_code;
    if (!key) return acc;
    const currentMs = profileTimestampMs(run.created_at) ?? 0;
    const previousMs = profileTimestampMs(acc[key]?.created_at) ?? 0;
    if (!acc[key] || currentMs >= previousMs) {
      acc[key] = run;
    }
    return acc;
  }, {});
}

function scenarioMaxPeriodMinutes(row: ScenarioRow): number {
  const periods = getScenarioPeriodRows(row).map((periodRow) => getPeriodMinutes(periodRow.period));
  return Math.max(...periods, 15);
}

function buildProfileHealth(
  enabled: boolean,
  periodMinutes: number,
  latestSuccessRun: NorthboundFileRun | undefined,
  createdAt: string | undefined,
  updatedAt: string | undefined,
  nowMs = Date.now(),
): ProfileHealthInfo {
  if (!enabled) {
    return {
      state: 'terminated',
      label: '终止',
      color: 'default',
      detail: '配置开关关闭，当前不会触发文件生成或上报。',
    };
  }

  const windowMs = getHealthWindowMs(periodMinutes);
  const graceMinutes = getHealthGraceMinutes(periodMinutes);
  const maxWaitText = `${formatPeriodMinutesForHealth(periodMinutes)} + ${graceMinutes} 分钟容忍`;
  const lastSuccessMs = profileTimestampMs(latestSuccessRun?.created_at, latestSuccessRun?.window_end);
  if (lastSuccessMs) {
    const stale = nowMs - lastSuccessMs > windowMs;
    return {
      state: stale ? 'broken' : 'normal',
      label: stale ? '异常' : '正常',
      color: stale ? 'error' : 'success',
      detail: stale
        ? `最近成功上报时间已超过最大周期（${maxWaitText}），请检查调度、数据源和传输目标。`
        : `最近成功上报在最大周期内（${maxWaitText}），调度状态正常。`,
    };
  }

  const baselineMs = profileTimestampMs(updatedAt, createdAt);
  if (baselineMs && nowMs - baselineMs > windowMs) {
    return {
      state: 'broken',
      label: '异常',
      color: 'error',
      detail: `启用后超过最大周期（${maxWaitText}）仍未产生成功上报记录，请检查调度、数据源和传输目标。`,
    };
  }

  return {
    state: 'pending',
    label: '待上报',
    color: 'warning',
    detail: `配置已启用，正在等待首次成功上报；超过最大周期（${maxWaitText}）后仍无成功记录会显示异常。`,
  };
}

function defaultCronForPeriod(period: string): string {
  if (period === '24H') return '0 1 0 * * ?';
  if (period === '7D') return '0 0 0 ? * MON';
  if (period === '1MO') return '0 0 0 1 * ?';
  return `0 0/${getPeriodMinutes(period)} * * * ?`;
}

function getDailyTimeValue(cron: string): string {
  const [, minute, hour] = cron.split(/\s+/);
  if (/^\d+$/.test(hour ?? '') && /^\d+$/.test(minute ?? '')) {
    return `${hour.padStart(2, '0')}:${minute.padStart(2, '0')}`;
  }
  return '00:01';
}

function getIntervalStartMinuteValue(cron: string): string {
  const [, minute] = cron.split(/\s+/);
  if (minute?.includes('/')) {
    return minute.split('/')[0].padStart(2, '0');
  }
  if (/^\d+$/.test(minute ?? '')) return minute.padStart(2, '0');
  return '00';
}

function cronFromDailyTime(time: string): string {
  const [hour = '0', minute = '1'] = time.split(':');
  return `0 ${Number(minute)} ${Number(hour)} * * ?`;
}

function cronFromIntervalStart(period: string, startMinute: string): string {
  if (period === '7D') return `0 ${Number(startMinute)} 0 ? * MON`;
  if (period === '1MO') return `0 ${Number(startMinute)} 0 1 * ?`;
  return `0 ${Number(startMinute)}/${getPeriodMinutes(period)} * * * ?`;
}

function formatScheduleLabel(period: string, cron: string): string {
  if (period === '24H') return `每日 ${getDailyTimeValue(cron)} 生成`;
  return `${formatRecurringPeriodLabel(period)}，${getIntervalStartMinuteValue(cron)} 分开始生成`;
}

function normalizeApiPeriod(period: string | undefined): NorthboundPageConfigPeriod {
  const value = period ?? '24H';
  return ['15M', '60M', '24H', '7D', '1MO'].includes(value)
    ? value as NorthboundPageConfigPeriod
    : '24H';
}

function normalizeApiFormat(domain: Domain, format: string | undefined, objectsValue?: string): NorthboundPageConfigFormat {
  return normalizeFormatForDomain(domain, (format ?? defaultFormatByDomain[domain]) as Format, objectsValue) as NorthboundPageConfigFormat;
}

function normalizeApiCompressionFormat(format: string | undefined): NorthboundPageConfigCompressionFormat {
  return format === 'gz' ? 'gz' : 'zip';
}

function normalizeCSVSeparatorInput(value: string | undefined): string | undefined {
  if (value === '\t') return '\\t';
  const trimmed = value?.trim() ?? '';
  if (!trimmed) return undefined;
  if (trimmed === '\\t' || trimmed.toLowerCase() === 'tab') return '\\t';
  return [...trimmed][0];
}

function csvSeparatorDisplay(value: string | undefined): string {
  const normalized = normalizeCSVSeparatorInput(value);
  if (!normalized || normalized === ',') return '默认逗号';
  if (normalized === '|') return '竖线 |';
  if (normalized === ';') return '分号 ;';
  if (normalized === '\\t') return 'Tab';
  return normalized;
}

function cronFromPeriodStartMinute(period: string, startMinute: number | undefined): string {
  const minute = Number.isInteger(startMinute) && startMinute !== undefined
    ? Math.max(0, Math.min(59, startMinute))
    : 0;
  if (period === '24H') return `0 ${minute} 0 * * ?`;
  return cronFromIntervalStart(period, String(minute).padStart(2, '0'));
}

function startMinuteFromCron(period: string, cron: string): number {
  const value = period === '24H'
    ? getDailyTimeValue(cron).split(':')[1]
    : getIntervalStartMinuteValue(cron);
  const minute = Number(value);
  return Number.isFinite(minute) ? Math.max(0, Math.min(59, minute)) : 0;
}

function normalizeInventoryType(value: string): InventoryType {
  const upperValue = value.toUpperCase();
  if (upperValue === 'GNB') return 'GNB';
  if (upperValue === 'GSM') return 'GSM';
  if (upperValue === 'OMC') return 'OMC';
  return 'ENB';
}

function hasChineseText(value?: string) {
  return /[\u4e00-\u9fff]/.test(value ?? '');
}

function normalizeScenarioNames(
  profile: NorthboundFileProfile,
  fallback?: Pick<ScenarioRow, 'scenarioName' | 'scenarioNameEn'>,
) {
  const apiName = profile.scenario_name?.trim() ?? '';
  const apiNameEn = profile.scenario_name_en?.trim() ?? '';
  const fallbackName = fallback?.scenarioName?.trim() ?? '';
  const fallbackNameEn = fallback?.scenarioNameEn?.trim() ?? '';
  const swapped = apiName !== '' && apiNameEn !== '' && !hasChineseText(apiName) && hasChineseText(apiNameEn);
  if (swapped) {
    return {
      scenarioName: apiNameEn,
      scenarioNameEn: apiName,
    };
  }
  const duplicatedDefaultEnglish = fallbackName !== ''
    && fallbackNameEn !== ''
    && apiName === apiNameEn
    && apiNameEn === fallbackNameEn;
  return {
    scenarioName: duplicatedDefaultEnglish
      ? fallbackName
      : (apiName || fallbackName || profile.name || profile.code),
    scenarioNameEn: apiNameEn || fallbackNameEn || apiName || profile.code,
  };
}

function mapApiScenarioObject(object: { code: string; tech?: string; profile?: string }): ScenarioObject {
  return {
    code: object.code,
    tech: isTech(object.tech) ? object.tech : undefined,
    profile: object.profile,
  };
}

function mapApiFileProfile(profile: NorthboundFileProfile): ScenarioRow {
  const fallback = scenarioRows.find((row) => row.code === profile.code);
  const groups: FileGroup[] = profile.groups.length > 0
    ? profile.groups.map((item) => {
        const domain = item.domain as Domain;
        const period = normalizeApiPeriod(item.period);
        const objects = item.objects.map(mapApiScenarioObject);
        const objectsValue = joinObjects(objects.map((object) => object.code));
        return {
          id: item.id,
          domain,
          format: normalizeApiFormat(domain, item.format, objectsValue),
          period,
          cron: cronFromPeriodStartMinute(period, item.start_minute),
          path: item.path_template,
          name: item.file_name_template,
          csvSeparator: item.csv_separator,
          compressionEnabled: item.compression_enabled,
          compressionFormat: normalizeApiCompressionFormat(item.compression_format),
          objects,
          selectedFields: item.selected_fields,
        };
      })
    : fallback?.groups ?? [];

  const scenarioNames = normalizeScenarioNames(profile, fallback);
  return {
    code: profile.code,
    vendor: profile.vendor || fallback?.vendor || 'Baicells',
    scenarioName: scenarioNames.scenarioName,
    scenarioNameEn: scenarioNames.scenarioNameEn,
    description: profile.description || fallback?.description || profile.name,
    flags: profile.flags?.length ? profile.flags : fallback?.flags ?? [],
    name: profile.name || fallback?.name || profile.code,
    enabled: profile.enabled,
    groups,
    logs: fallback?.logs,
    createdAt: profile.created_at,
    updatedAt: profile.updated_at,
  };
}

function mapApiInventoryProfile(profile: NorthboundInventoryProfile): InventoryConfigRow {
  const key = normalizeInventoryType(profile.code || profile.object_code);
  const fallback = initialInventoryConfigs.find((row) => row.key === key);
  const period = normalizeApiPeriod(profile.period);
  return {
    key,
    name: profile.name || fallback?.name || `${profile.object_code} Inventory`,
    objectCode: profile.object_code || fallback?.objectCode || key,
    tech: profile.tech || fallback?.tech || key,
    period,
    cron: cronFromPeriodStartMinute(period, profile.start_minute),
    format: 'CSV',
    path: profile.path_template || fallback?.path || PATH_INVENTORY,
    fileName: profile.file_name_template || fallback?.fileName || INVENTORY_NAME,
    compressionEnabled: profile.compression_enabled,
    compressionFormat: normalizeApiCompressionFormat(profile.compression_format),
    createdAt: profile.created_at,
    updatedAt: profile.updated_at,
  };
}

function mapApiInventoryField(field: NorthboundInventoryField, template: InventoryType): InventoryField {
  return {
    key: field.key,
    template,
    column: field.output_alias,
    exportKey: field.system_field,
    source: field.source,
    dataType: field.data_type,
    renderer: field.renderer,
    enabled: field.enabled,
  };
}

function mapApiInventoryCandidateField(field: NorthboundFieldDefinition, template: InventoryType): InventoryField {
  return {
    key: field.key,
    template,
    column: field.output_alias,
    exportKey: field.system_field,
    source: field.source,
    dataType: field.data_type,
    renderer: field.renderer,
    enabled: true,
  };
}

function dedupeInventoryFields(fields: InventoryField[]): InventoryField[] {
  const seen = new Set<string>();
  return fields.filter((field) => {
    const id = `${field.template}:${field.exportKey}`.toLowerCase();
    if (seen.has(id)) return false;
    seen.add(id);
    return true;
  });
}

function scenarioEnabledMap(rows: ScenarioRow[]): Record<string, boolean> {
  return Object.fromEntries(rows.map((row) => [row.code, row.enabled]));
}

function getNextCustomScenarioCode(rows: ScenarioRow[]): string {
  const usedCodes = new Set(rows.map((row) => row.code.toUpperCase()));
  let maxCodeNumber = 0;
  rows.forEach((row) => {
    const match = row.code.toUpperCase().match(/^S(\d{4})$/);
    if (!match) return;
    maxCodeNumber = Math.max(maxCodeNumber, Number(match[1]));
  });
  for (let index = Math.min(maxCodeNumber + 1, 9999); index <= 9999; index += 1) {
    const code = `S${String(index).padStart(4, '0')}`;
    if (!usedCodes.has(code)) {
      return code;
    }
  }
  for (let index = 1; index <= maxCodeNumber; index += 1) {
    const code = `S${String(index).padStart(4, '0')}`;
    if (!usedCodes.has(code)) {
      return code;
    }
  }
  return 'S9999';
}

function inventoryEnabledMap(rows: InventoryConfigRow[], sourceRows?: NorthboundInventoryProfile[]): Record<InventoryType, boolean> {
  const sourceByKey = new Map((sourceRows ?? []).map((row) => [normalizeInventoryType(row.code || row.object_code), row]));
  return rows.reduce<Record<InventoryType, boolean>>((acc, row) => {
    acc[row.key] = sourceByKey.get(row.key)?.enabled ?? false;
    return acc;
  }, { ...defaultInventoryEnabled });
}

function mapApiDeliveryTarget(target: NorthboundDeliveryTarget): DeliveryTargetRow {
  return {
    key: target.key,
    name: target.name,
    enabled: target.enabled,
    protocol: target.protocol,
    host: target.host,
    port: target.port,
    username: target.username,
    credential: target.credential || (target.credential_set ? storedCredentialText : ''),
    authMode: target.auth_mode,
    remoteRoot: target.remote_root,
    retryTimes: target.retry_times,
    timeoutSeconds: target.timeout_seconds,
    passiveMode: target.passive_mode,
    hostKeyPolicy: target.host_key_policy ?? 'INSECURE',
    hostKeyFingerprint: target.host_key_fingerprint ?? '',
  };
}

function serializeDeliveryTargets(
  scope: 'file' | 'inventory' | 'socket',
  ownerCode: string,
  rows: DeliveryTargetRow[],
) {
  return {
    scope,
    owner_code: ownerCode,
    items: rows.map((row) => ({
      scope,
      owner_code: ownerCode,
      key: row.key,
      name: row.name,
      enabled: row.enabled,
      protocol: row.protocol,
      host: row.host,
      port: row.port,
      username: row.username,
      credential: row.credential === storedCredentialText ? '' : row.credential,
      credential_set: row.credential === storedCredentialText,
      auth_mode: row.authMode,
      remote_root: row.remoteRoot,
      retry_times: row.retryTimes,
      timeout_seconds: row.timeoutSeconds,
      passive_mode: row.passiveMode,
      host_key_policy: row.protocol === 'SFTP' ? row.hostKeyPolicy : 'INSECURE',
      host_key_fingerprint: row.protocol === 'SFTP' ? row.hostKeyFingerprint : '',
    })),
  };
}

function serializeDeliveryTarget(
  scope: 'file' | 'inventory' | 'socket',
  ownerCode: string,
  row: DeliveryTargetRow,
) {
  return serializeDeliveryTargets(scope, ownerCode, [row]).items[0];
}

function deliveryTargetHasCredential(row: DeliveryTargetRow): boolean {
  const credential = row.credential ?? '';
  return credential === storedCredentialText || credential.trim() !== '';
}

function normalizeFileDeliveryOwnerCode(code?: string): string {
  return (code ?? '').trim().toUpperCase();
}

function fileDeliveryScopeKey(ownerCode?: string): string {
  const normalized = normalizeFileDeliveryOwnerCode(ownerCode);
  return normalized ? `file-${normalized.toLowerCase()}` : 'file';
}

function mapApiSnmpTarget(target: NorthboundSNMPAlarmTarget): SnmpAlarmTargetRow {
  const fallback = snmpAlarmTargets.find((row) => row.key === target.key);
  const version = snmpVersionForKey(target.key, target.version);
  const community = version === 'v2'
    ? (target.community?.trim() || (target.community_set ? storedCredentialText : snmpDefaultCommunity))
    : undefined;
  return {
    key: target.key,
    name: target.name || fallback?.name || target.key,
    version,
    notificationType: target.notification_type,
    listenIp: target.listen_ip,
    listenPort: target.listen_port,
    targetHost: target.target_host,
    targetPort: target.target_port,
    community,
    securityName: target.security_name,
    authProtocol: target.auth_protocol,
    authCredential: target.auth_credential || (target.auth_credential_set ? storedCredentialText : ''),
    privProtocol: target.priv_protocol,
    privCredential: target.priv_credential || (target.priv_credential_set ? storedCredentialText : ''),
    mibQueryEnabled: target.mib_query_enabled,
    clearSeverityPolicy: target.clear_severity_policy,
    timeoutSeconds: target.timeout_seconds,
    retries: target.retries,
  };
}

function serializeSnmpTarget(row: SnmpAlarmTargetRow, enabled: boolean): NorthboundSNMPAlarmTarget {
  const isV2 = row.version === 'v2';
  const isV3 = row.version === 'v3';
  const communityIsStored = row.community === storedCredentialText;
  const community = communityIsStored
    ? ''
    : (row.community?.trim() || (isV2 ? snmpDefaultCommunity : undefined));
  const authCredentialIsStored = row.authCredential === storedCredentialText;
  const privCredentialIsStored = row.privCredential === storedCredentialText;
  return {
    key: row.key,
    name: row.name,
    enabled,
    version: row.version,
    notification_type: row.notificationType,
    listen_ip: row.listenIp,
    listen_port: row.listenPort,
    target_host: row.targetHost,
    target_port: row.targetPort,
    community: isV2 ? community : undefined,
    community_set: isV2 ? communityIsStored : false,
    security_name: isV3 ? row.securityName : undefined,
    auth_protocol: isV3 ? row.authProtocol : undefined,
    auth_credential: isV3 ? (authCredentialIsStored ? '' : row.authCredential) : undefined,
    auth_credential_set: isV3 ? authCredentialIsStored : false,
    priv_protocol: isV3 ? row.privProtocol : undefined,
    priv_credential: isV3 ? (privCredentialIsStored ? '' : row.privCredential) : undefined,
    priv_credential_set: isV3 ? privCredentialIsStored : false,
    clear_severity_policy: row.clearSeverityPolicy,
    mib_query_enabled: row.mibQueryEnabled,
    timeout_seconds: row.timeoutSeconds,
    retries: row.retries,
  };
}

function getSnmpConfigBlocker(row: SnmpAlarmTargetRow, reportEnabled: boolean): string | null {
  if (reportEnabled && !row.targetHost.trim()) {
    return '请先编辑通知目标 IP/域名，再启用真实上报';
  }
  if (row.version === 'v3' && (reportEnabled || row.mibQueryEnabled)) {
    if (!row.securityName?.trim()) {
      return '请先填写 SNMP v3 安全名';
    }
    const securityLevel = snmpV3SecurityLevel(row);
    if (securityLevel !== 'noAuthNoPriv' && !row.authCredential?.trim()) {
      return '请先填写 SNMP v3 认证密码';
    }
    if (securityLevel !== 'noAuthNoPriv' && snmpCredentialTooShort(row.authCredential)) {
      return 'SNMP v3 认证密码至少 8 位';
    }
    if (securityLevel === 'authPriv' && !row.privCredential?.trim()) {
      return '请先填写 SNMP v3 加密密码';
    }
    if (securityLevel === 'authPriv' && snmpCredentialTooShort(row.privCredential)) {
      return 'SNMP v3 加密密码至少 8 位';
    }
  }
  return null;
}

function snmpCredentialTooShort(value?: string) {
  const credential = value?.trim() ?? '';
  return credential !== '' && credential !== storedCredentialText && credential.length < 8;
}

function mapApiSocketConfig(config: NorthboundSocketAlarmConfig): SocketAlarmConfigRow {
  const fallback = socketAlarmConfigs.find((row) => row.key === config.key);
  const syncMode = config.client_sync_enabled
    ? config.profile === 'CUCC'
      ? '消息同步 + 文件补录'
      : '消息同步'
    : '仅实时推送';
  return {
    key: config.key,
    name: config.name || fallback?.name || config.key,
    profile: config.profile,
    frame: fallback?.frame ?? 'Socket 告警帧',
    listenIp: config.listen_ip,
    listenPort: config.listen_port,
    maxClients: config.max_clients ?? fallback?.maxClients ?? 20,
    heartbeatPeriod: config.heartbeat_seconds,
    heartbeatTimes: config.heartbeat_times ?? fallback?.heartbeatTimes ?? 3,
    encoding: fallback?.encoding ?? 'UTF-8',
    realTimeEnabled: config.realtime_push_enabled,
    historyEnabled: config.client_sync_enabled,
    syncMode,
    accountTypes: [...new Set(config.accounts.map((account) => account.type))].join(' / ') || fallback?.accountTypes || 'msg',
    sequencePolicy: fallback?.sequencePolicy ?? '连续 alarmSeq',
  };
}

function mapApiSocketAccounts(config: NorthboundSocketAlarmConfig): SocketAccountRow[] {
  return config.accounts.map((account) => ({
    key: account.key,
    channel: account.channel,
    username: account.username,
    type: account.type,
    credential: account.credential || (account.credential_set ? storedCredentialText : ''),
    enabled: account.enabled,
    purpose: account.purpose,
  }));
}

function serializeSocketConfig(row: SocketAlarmConfigRow, accounts: SocketAccountRow[], enabled: boolean): NorthboundSocketAlarmConfig {
  return {
    key: row.key,
    name: row.name,
    enabled,
    profile: row.profile,
    mode: 'server',
    listen_ip: row.listenIp,
    listen_port: row.listenPort,
    max_clients: row.maxClients,
    realtime_push_enabled: row.realTimeEnabled,
    client_sync_enabled: row.historyEnabled,
    heartbeat_seconds: row.heartbeatPeriod,
    heartbeat_times: row.heartbeatTimes,
    idle_timeout_seconds: row.heartbeatPeriod * Math.max(row.heartbeatTimes, 1),
    accounts: accounts.map((account) => ({
      key: account.key,
      enabled: account.enabled,
      channel: account.channel,
      username: account.username,
      type: account.type,
      credential: account.credential === storedCredentialText ? '' : account.credential,
      purpose: account.purpose,
    })),
  };
}

function mapApiConfig(row: NorthboundAPIConfig): NorthboundApiRow {
  const fallback = northboundApiRows.find((item) => item.key === row.key);
  return {
    key: row.key,
    configKey: row.key,
    apiKind: (row.kind as ApiKind) || fallback?.apiKind,
    dataType: row.data_type,
    module: fallback?.module ?? apiModuleLabel(row.data_type),
    name: row.name,
    method: row.method,
    url: row.path,
    auth: fallback?.auth ?? commonApiAuth,
    backendSource: row.source || fallback?.backendSource || '',
    fieldContract: fallback?.fieldContract ?? '当前返回字段',
    responseFields: (row.response_contract?.fields as string[] | undefined) ?? fallback?.responseFields,
    requestExample: fallback?.requestExample ?? `${row.method} ${row.path}`,
    responseExample: fallback?.responseExample ?? '{ "ret": 1, "msg": "ok", "data": {} }',
  };
}

function apiModuleLabel(dataType: string): string {
  const labels: Record<string, string> = {
    auth: '认证鉴权',
    鉴权: '认证鉴权',
    api_user: '北向用户管理',
    api_log: '北向接口日志',
    device: '设备管理',
    设备: '设备管理',
    group: '设备组管理',
    设备组: '设备组管理',
    config: '参数配置',
    参数: '参数配置',
    配置: '参数配置',
    task: '异步任务',
    任务: '异步任务',
    advanced_task: '高级任务',
    log_collect: '设备日志收集',
    日志: '设备日志收集',
    alarm: '告警查询',
    告警: '告警查询',
    pm: '性能管理',
    PM: '性能管理',
    mr: 'MR 数据',
    MR: 'MR 数据',
    同步: '数据同步',
    Inventory: 'Inventory 导出',
    '主备服务': '主备服务器',
  };
  return labels[dataType] ?? dataType;
}

function mapApiUser(row: NorthboundAPIUser): ApiUserRow {
  return {
    key: row.id || `api-user:${row.username}`,
    username: row.username,
    enabled: row.enabled,
    password: row.password || (row.password_set ? apiUserPasswordMask : ''),
    passwordSet: Boolean(row.password_set),
    createdAt: row.created_at,
  };
}

function newApiUserKey(): string {
  return `api-user-draft:${Date.now()}:${Math.random().toString(36).slice(2)}`;
}

function serializeApiUsers(rows: ApiUserRow[]) {
  return {
    items: rows.map((row) => ({
      username: row.username.trim(),
      enabled: row.enabled,
      password: row.password === apiUserPasswordMask ? '' : row.password.trim(),
    })),
  };
}

// Derives the field-target keys (domain:objectCode:scope:profile) that belong to a
// single period row / group. Mirrors getFieldTargets so lookups into fieldRowsByTarget
// match exactly.
function periodRowTargetKeys(row: ScenarioPeriodRow): string[] {
  const keys: string[] = [];
  splitObjects(row.objects).forEach((objectToken) => {
    const parsedObject = parseObjectToken(objectToken);
    const rowScopeTech = normalizeTechValue(row.scope);
    const techs = parsedObject.tech
      ? [parsedObject.tech]
      : row.domain === 'PM'
        ? getPmTechsFromScope(row.scope)
        : row.domain === 'CM' && rowScopeTech
          ? [rowScopeTech]
          : [undefined];
    techs.forEach((tech) => {
      const scope = tech ?? row.scope;
      const profile = resolveEditorProfile(row, parsedObject.code, tech);
      keys.push(`${row.domain}:${parsedObject.code}:${scope}:${profile}`);
    });
  });
  return keys;
}

function serializeEditorPeriodRows(
  rows: ScenarioPeriodRow[],
  fieldRowsByTarget: Record<string, ReportFieldRow[]> = {},
): NorthboundFileGroup[] {
  return rows.map((row) => {
    const format = normalizeApiFormat(row.domain, row.format, row.objects);
    const period = normalizeApiPeriod(row.period);
    // Collect this group's selected field keys (backend Key format) from any field
    // target that belongs to it and has been edited. Omitted when empty = export all.
    const selectedSet = new Set<string>();
    periodRowTargetKeys(row).forEach((targetKey) => {
      const targetRows = fieldRowsByTarget[targetKey];
      if (!targetRows) return;
      targetRows.forEach((r) => {
        if (r.enabled) selectedSet.add(selectionKey(r));
      });
    });
    return {
      id: row.key,
      domain: row.domain,
      format,
      period,
      start_minute: startMinuteFromCron(period, row.cron),
      path_template: row.path || defaultRemotePathByDomain[row.domain],
      file_name_template: row.fileName || defaultPeriodRow(row.domain).fileName,
      csv_separator: row.csvSeparator,
      compression_enabled: row.compressionEnabled,
      compression_format: row.compressionFormat,
      selected_fields: selectedSet.size > 0 ? [...selectedSet] : undefined,
      objects: splitObjects(row.objects).map((objectToken) => {
        const parsed = parseObjectToken(objectToken);
        const tech = parsed.tech ?? normalizeTechValue(row.scope);
        const profile = resolveEditorProfile(row, parsed.code, tech);
        return {
          code: parsed.code,
          tech,
          profile: profile || undefined,
        };
      }),
    };
  });
}

function validateEditorPeriodRows(rows: ScenarioPeriodRow[]): string | undefined {
  if (rows.length === 0) {
    return '请至少新增一个文件对象';
  }
  for (const row of rows) {
    if ((row.domain === 'CM' || row.domain === 'PM') && !normalizeTechValue(row.scope)) {
      return `${row.domain} 制式只能选择 ENB、GNB 或 GSM`;
    }
    if (splitObjects(row.objects).length === 0) {
      return `${row.domain} 请至少选择一个对象`;
    }
    if (!supportedFormatsForSelection(row.domain, row.objects).includes(row.format)) {
      return `${row.domain} 当前对象不支持 ${row.format} 格式`;
    }
    if (!row.path.trim()) {
      return `${row.domain} 请填写上传目录模板`;
    }
    if (!row.fileName.trim()) {
      return `${row.domain} 请填写文件名模板`;
    }
  }
  return undefined;
}

function serializeInventoryConfig(
  row: InventoryConfigRow,
  enabled: boolean,
  fields: InventoryField[],
): NorthboundUpdateInventoryProfileRequest {
  return {
    name: row.name,
    object_code: row.objectCode,
    tech: row.tech,
    period: normalizeApiPeriod(row.period),
    start_minute: startMinuteFromCron(row.period, row.cron),
    path_template: row.path,
    file_name_template: row.fileName,
    compression_enabled: row.compressionEnabled,
    compression_format: row.compressionFormat,
    enabled,
    status: enabled ? 'normal' : 'terminated',
    fields: fields.map((field) => ({
      key: field.key,
      output_alias: field.column,
      system_field: field.exportKey,
      source: field.source,
      data_type: field.dataType,
      renderer: field.renderer,
      enabled: field.enabled,
    })),
  };
}

function replaceTemplateTokens(template: string, row: ScenarioPeriodRow, dateOnly: boolean): string {
  const objectCode = getFirstObjectCode(row);
  const tokens: Record<string, string> = {
    '#FTPRoot#': 'northupload',
    '#Province#': 'GD',
    '#OMC-R#': 'BaiOMC',
    '#DateTime#': dateOnly ? '20260730' : '20260730010000',
    '#Date#': '20260730',
    '#PeriodStartTime#': '20260730000000',
    '#PeriodEndTime#': '20260730010000',
    '#LocalHost#': '172.21.172.189',
    '#DataVersion#': '1.0',
    '#DataPeriod#': getPreviewDataPeriod(row.period),
    '#ModuleType#': 'OMC',
    '#eNBID#': '100001',
    '#Object#': objectCode,
  };
  return Object.entries(tokens).reduce(
    (result, [token, value]) => result.split(token).join(value),
    pickFirstTemplateChoice(stripOptionalTemplateSegments(template)),
  );
}

function ensureOutputExtension(fileName: string, format: Format): string {
  const ext = outputExtensionByFormat[format];
  const normalized = fileName.replace(/\.(csv|xml|txt)$/i, '');
  return `${normalized}.${ext}`;
}

function getOutputTemplatePreview(row: ScenarioPeriodRow): { path: string; fileName: string } {
  const normalizedFormat = normalizeFormatForDomain(row.domain, row.format, row.objects);
  const pathTemplate = row.path || defaultRemotePathByDomain[row.domain];
  const fileNameTemplate = row.fileName || defaultPeriodRow(row.domain).fileName;
  const path = replaceTemplateTokens(pathTemplate, row, true);
  const fileName = ensureOutputExtension(replaceTemplateTokens(fileNameTemplate, row, false), normalizedFormat);
  return {
    path,
    fileName: row.compressionEnabled ? `${fileName}.${row.compressionFormat}` : fileName,
  };
}

function getInventoryTemplatePreview(row: InventoryConfigRow): { path: string; fileName: string } {
  return getOutputTemplatePreview({
    key: row.key,
    domain: 'INVENTORY',
    scope: row.tech,
    format: row.format,
    period: row.period,
    trigger: formatScheduleLabel(row.period, row.cron),
    cron: row.cron,
    objects: row.objectCode,
    path: row.path,
    fileName: row.fileName,
    compressionEnabled: row.compressionEnabled,
    compressionFormat: row.compressionFormat,
  });
}

function mergeReportStatus(defaultInfo: ReportStatusInfo): ReportStatusInfo {
  return {
    ...defaultInfo,
    ...(reportStatusSamples[defaultInfo.key] ?? {}),
  };
}

function buildFileReportStatus(row: ScenarioRow): ReportStatusInfo {
  const firstPeriodRow = getScenarioPeriodRows(row)[0] ?? defaultPeriodRow();
  const preview = getOutputTemplatePreview(firstPeriodRow);
  return mergeReportStatus({
    key: `file:${row.code}`,
    capabilityName: `${row.code} 北向文件`,
    state: 'success',
    statusText: '上报正常',
    lastTime: '2026-08-03 15:15:00',
    artifactType: 'file',
    artifactName: preview.fileName,
    artifactPath: preview.path,
    size: '1.2 MB',
    targetSummary: '等待启用传输目标',
    detail: '最近一次生成记录可查看。任务关闭时不会继续调度。',
    payload: 'object,period,status\nCP,20260803151500,success\nPC,20260803151500,success\n',
  });
}

function buildInventoryReportStatus(row: InventoryConfigRow): ReportStatusInfo {
  const preview = getInventoryTemplatePreview(row);
  return mergeReportStatus({
    key: `inventory:${row.key}`,
    capabilityName: `${row.objectCode} Inventory`,
    state: 'success',
    statusText: '上报正常',
    lastTime: '2026-08-03 00:05:00',
    artifactType: 'file',
    artifactName: preview.fileName,
    artifactPath: preview.path,
    size: '512 KB',
    targetSummary: '等待启用传输目标',
    detail: '最近一次 Inventory 文件可查看。任务关闭时不会继续调度。',
    payload: 'SN,Name,IP,Status\n867294050000001,Site-A,10.21.1.11,Normal\n',
  });
}

function buildSocketReportStatus(row: SocketAlarmConfigRow): ReportStatusInfo {
  return mergeReportStatus({
    key: `socket:${row.key}`,
    capabilityName: row.name,
    state: 'success',
    statusText: '推送正常',
    lastTime: '2026-08-03 16:50:00',
    artifactType: 'message',
    artifactName: `${row.profile} alarm message`,
    artifactPath: `socket://${endpointText(row.listenIp, row.listenPort)}`,
    size: '1.4 KB',
    targetSummary: '暂无在线客户端',
    detail: '最近一次 Socket 告警报文可查看。配置关闭时不会消费告警队列。',
    payload: row.profile === 'CUCC'
      ? 'realTimeAlarm;alarmSeq=920188;severity=major;objectUID=867294050000001'
      : '{ "msgType": 10, "alarmSequenceId": 920188, "origSeverity": "major" }',
  });
}

function buildSnmpReportStatus(row: SnmpAlarmTargetRow): ReportStatusInfo {
  return mergeReportStatus({
    key: `snmp:${row.key}`,
    capabilityName: snmpVersionLabel(row.version),
    state: 'success',
    statusText: `${row.notificationType} 正常`,
    lastTime: '2026-08-03 16:50:00',
    artifactType: 'message',
    artifactName: `omcAlarmNotification ${row.notificationType}`,
    artifactPath: `snmp://${endpointText(row.targetHost, row.targetPort)}`,
    size: '1.2 KB',
    targetSummary: '等待发送',
    detail: '最近一次 SNMP 告警报文可查看。目标关闭时不会发送 Trap/Inform。',
    payload: buildSampleSnmpPayload('920188', 'major'),
  });
}

function buildApiReportStatus(row: NorthboundApiRow): ReportStatusInfo {
  const meta = getApiMeta(row);
  const responseFields = row.responseFields ?? meta.responseFields ?? currentEnvelopeFields;
  const displayResponseFields = apiFieldDisplayList(responseFields);
  const module = apiModuleDisplay(row);
  const kind = apiKindLabel(meta.apiKind);
  return {
    key: `api:${row.key}`,
    capabilityName: row.name,
    state: 'success',
    statusText: '接口可用',
    lastTime: '-',
    artifactType: 'message',
    artifactName: `${row.method} ${row.name}`,
    artifactPath: row.url,
    size: `${displayResponseFields.length} 字段`,
    targetSummary: `${module} / ${kind} / 返回字段 ${displayResponseFields.length} 项`,
    detail: '已按当前系统配置生成接口调用说明和返回字段清单。',
    payload: JSON.stringify({
      接口名称: row.name,
      业务模块: module,
      接口类型: kind,
      请求方式: row.method,
      接口地址: row.url,
      认证方式: normalizeLegacyApiText(row.auth),
      返回字段范围: meta.fieldContract,
      请求示例: normalizeLegacyApiText(row.requestExample),
      返回字段: displayResponseFields,
    }, null, 2),
    previewTitle: '接口检查结果',
    copyLabel: '复制结果',
    resultTitle: `${row.name} 接口检查结果`,
  };
}

function effectiveReportStatus(info: ReportStatusInfo, enabled: boolean): ReportStatusInfo {
  if (enabled) return info;
  return {
    ...info,
    state: 'idle',
    statusText: '未启用',
    detail: `配置开关关闭，当前不会触发上报。${info.detail}`,
  };
}

function runTriggerDescription(run: NorthboundFileRun): string {
  const triggerReason = typeof run.summary?.trigger_reason === 'string'
    ? run.summary.trigger_reason.toLowerCase()
    : '';
  const triggerText = triggerReason === 'auto' || triggerReason === 'automatic' || triggerReason === 'scheduled'
    ? '自动生成记录'
    : '手动生成记录';
  const subject = run.profile_code || run.object_code || '当前任务';
  return `${subject} ${triggerText}。`;
}

function buildRunReportStatus(run: NorthboundFileRun, fallbackCapabilityName: string): ReportStatusInfo {
  const state: ReportState = run.status === 'success'
    ? 'success'
    : run.status === 'running'
      ? 'running'
      : 'failed';
  const payload = run.artifact_content
    || run.error_message
    || '（未获取到文件内容：该记录窗口内无数据，或内容已被保留期清理）';
  return {
    key: `run:${run.id}`,
    runId: run.id,
    capabilityName: fallbackCapabilityName,
    state,
    statusText: state === 'success' ? '生成成功' : state === 'running' ? '生成中' : '生成失败',
    lastTime: formatRunTime(run.created_at),
    artifactType: 'file',
    artifactName: run.artifact_name || '-',
    artifactPath: run.artifact_path || '-',
    size: formatBytes(run.artifact_size),
    targetSummary: `生成 ${run.row_count} 行`,
    detail: run.error_message || runTriggerDescription(run),
    payload,
  };
}

function parseCsvRows(content: string): string[][] {
  const rows: string[][] = [];
  let row: string[] = [];
  let cell = '';
  let quoted = false;
  const normalized = content.replace(/^\uFEFF/, '');

  for (let index = 0; index < normalized.length; index += 1) {
    const character = normalized[index];
    if (character === '"') {
      if (quoted && normalized[index + 1] === '"') {
        cell += '"';
        index += 1;
      } else {
        quoted = !quoted;
      }
      continue;
    }
    if (character === ',' && !quoted) {
      row.push(cell);
      cell = '';
      continue;
    }
    if (character === '\n' && !quoted) {
      row.push(cell.replace(/\r$/, ''));
      if (row.some((value) => value.length > 0)) rows.push(row);
      row = [];
      cell = '';
      continue;
    }
    cell += character;
  }

  if (cell.length > 0 || row.length > 0) {
    row.push(cell.replace(/\r$/, ''));
    if (row.some((value) => value.length > 0)) rows.push(row);
  }
  return rows;
}

function CsvReportPreview({ content }: { content: string }) {
  const rows = useMemo(() => parseCsvRows(content), [content]);
  const headers = rows[0] ?? [];
  const dataRows = rows.slice(1);
  const scrollRef = useRef<HTMLDivElement>(null);
  const [scrollMetrics, setScrollMetrics] = useState({ scrollLeft: 0, clientWidth: 0, scrollWidth: 0 });
  const [requestedPage, setRequestedPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const totalPages = Math.max(1, Math.ceil(dataRows.length / pageSize));
  const currentPage = Math.min(requestedPage, totalPages);
  const visibleRows = dataRows.slice((currentPage - 1) * pageSize, currentPage * pageSize);
  const showPagination = dataRows.length > 0;
  useEffect(() => {
    setRequestedPage(1);
  }, [content]);
  useEffect(() => {
    const scrollElement = scrollRef.current;
    if (!scrollElement) return undefined;

    const syncScrollMetrics = () => {
      setScrollMetrics({
        scrollLeft: scrollElement.scrollLeft,
        clientWidth: scrollElement.clientWidth,
        scrollWidth: scrollElement.scrollWidth,
      });
    };
    const resizeObserver = new ResizeObserver(syncScrollMetrics);
    resizeObserver.observe(scrollElement);
    const table = scrollElement.querySelector('table');
    if (table) resizeObserver.observe(table);
    scrollElement.addEventListener('scroll', syncScrollMetrics, { passive: true });
    syncScrollMetrics();
    return () => {
      resizeObserver.disconnect();
      scrollElement.removeEventListener('scroll', syncScrollMetrics);
    };
  }, [content, headers.length, currentPage]);

  const horizontalScrollMax = Math.max(0, scrollMetrics.scrollWidth - scrollMetrics.clientWidth);
  const hasHorizontalScroll = horizontalScrollMax > 0;
  if (headers.length < 2) return <pre className={styles.codeBlock}>{content}</pre>;

  return (
    <div className={styles.csvPreview}>
      <div className={styles.csvPreviewMeta}>
        <Tag color="blue">CSV</Tag>
        <Typography.Text type="secondary">
          {headers.length} 列 · {dataRows.length} 行
        </Typography.Text>
      </div>
      <div ref={scrollRef} className={styles.csvPreviewScroll}>
        <table className={styles.csvPreviewTable}>
          <thead>
            <tr>
              <th className={styles.csvPreviewIndex}>#</th>
              {headers.map((header, index) => (
                <th key={`header-${index}`} scope="col" title={header || `第 ${index + 1} 列`}>
                  {header || `第 ${index + 1} 列`}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {dataRows.length > 0 ? visibleRows.map((dataRow, rowIndex) => (
              <tr key={`row-${(currentPage - 1) * pageSize + rowIndex}`}>
                <td className={styles.csvPreviewIndex}>{(currentPage - 1) * pageSize + rowIndex + 1}</td>
                {headers.map((_, columnIndex) => {
                  const value = dataRow[columnIndex] ?? '';
                  return (
                    <td key={`cell-${(currentPage - 1) * pageSize + rowIndex}-${columnIndex}`} title={value}>
                      {value || '—'}
                    </td>
                  );
                })}
              </tr>
            )) : (
              <tr>
                <td className={styles.csvPreviewEmpty} colSpan={headers.length + 1}>暂无数据行</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
      {hasHorizontalScroll && (
        <div className={styles.csvHorizontalScroll}>
          <input
            aria-label="表格横向滚动"
            type="range"
            min={0}
            max={horizontalScrollMax}
            step={1}
            value={Math.min(scrollMetrics.scrollLeft, horizontalScrollMax)}
            onChange={(event) => {
              if (scrollRef.current) scrollRef.current.scrollLeft = Number(event.currentTarget.value);
            }}
          />
        </div>
      )}
      {showPagination && (
        <div className={styles.csvPreviewFooter}>
          <Pagination
            current={currentPage}
            pageSize={pageSize}
            total={dataRows.length}
            size="small"
            hideOnSinglePage={false}
            showSizeChanger
            pageSizeOptions={[20, 50, 100]}
            showTotal={(total, range) => `${range[0]}-${range[1]} / ${total} 行`}
            onChange={(nextPage, nextPageSize) => {
              setPageSize(nextPageSize);
              setRequestedPage(nextPageSize === pageSize ? nextPage : 1);
            }}
          />
        </div>
      )}
    </div>
  );
}

function formatMessagePayload(content: string): { kind: string; text: string; parsed?: unknown } {
  const trimmed = content.trim();
  if (!trimmed) return { kind: '空报文', text: '（无报文内容）' };
  try {
    const parsed = JSON.parse(trimmed) as unknown;
    return { kind: 'JSON', text: JSON.stringify(parsed, null, 2), parsed };
  } catch {
    // Fall through to protocol text handling.
  }
  if (parseDelimitedMessages(content).length > 0) {
    return { kind: '键值报文', text: content };
  }
  return { kind: '文本报文', text: content };
}

function valuePreview(value: unknown): string {
  if (value === null || value === undefined) return '-';
  if (typeof value === 'string') return value;
  if (typeof value === 'number' || typeof value === 'boolean') return String(value);
  if (Array.isArray(value)) return `${value.length} 项`;
  try {
    const text = JSON.stringify(value);
    return text.length > 500 ? `${text.slice(0, 500)}...` : text;
  } catch {
    return String(value);
  }
}

function parseJSONPayload(payload: string): Record<string, unknown> | null {
  try {
    const parsed = JSON.parse(payload);
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>;
    }
  } catch {
    return null;
  }
  return null;
}

function friendlyBool(value: unknown): string {
  if (value === true) return '支持';
  if (value === false) return '不支持';
  return '-';
}

function friendlyApiArtifactName(value?: string): string {
  const raw = value || '接口检查';
  return raw.replace(/\s+contract$/i, ' 接口检查');
}

function friendlyApiEventPayload(event: NorthboundPageConfigEvent): string {
  const parsed = parseJSONPayload(event.payload || '') ?? {};
  const summary = event.summary ?? {};
  const responseContract = parsed.response_contract;
  const responseFields = responseContract
    && typeof responseContract === 'object'
    && !Array.isArray(responseContract)
    && Array.isArray((responseContract as Record<string, unknown>).fields)
    ? (responseContract as Record<string, unknown>).fields
    : responseContract;

  return JSON.stringify({
    接口名称: friendlyApiArtifactName(event.artifact_name).replace(/\s+接口检查$/, ''),
    请求方式: parsed.method ?? summary.method ?? '-',
    接口地址: parsed.path ?? event.artifact_path ?? '-',
    接口类型: summary.kind ?? '-',
    数据类型: summary.data_type ?? '-',
    旧系统支持: friendlyBool(parsed.old_system ?? summary.old_system_supported),
    当前系统支持: friendlyBool(parsed.current_supported ?? summary.current_supported),
    返回字段: responseFields ?? '-',
  }, null, 2);
}

function getJsonFieldRows(value: unknown): MessageFieldRow[] {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return [];
  return Object.entries(value as Record<string, unknown>).map(([field, fieldValue]) => ({
    key: field,
    field,
    value: valuePreview(fieldValue),
  }));
}

function parseDelimitedMessages(content: string): DelimitedMessageRow[] {
  return content
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line, index) => {
      const parts = line.split(';').map((part) => part.trim()).filter(Boolean);
      const command = parts[0] ?? '-';
      const fields = parts.slice(1).filter((part) => part.includes('='));
      return {
        key: `${index}-${command}`,
        command,
        fieldCount: fields.length,
        summary: fields.slice(0, 8).join('; ') || line,
      };
    })
    .filter((row) => row.fieldCount > 0);
}

function parseSnmpVarBinds(content: string): SnmpVarBindRow[] {
  return content
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line, index) => {
      const match = line.match(/^(\.?\d+(?:\.\d+)+)\s*=\s*(.*)$/);
      if (!match) return null;
      const oid = match[1].replace(/^\./, '');
      const field = Object.entries(snmpAlarmFieldByOID)
        .find(([fieldOID]) => oid === fieldOID || oid.startsWith(`${fieldOID}.`))?.[1];
      return {
        key: `${index}-${oid}`,
        index: index + 1,
        oid,
        field: field ? `${field.field} / ${field.cnName}` : '-',
        value: match[2] ?? '',
      };
    })
    .filter((row): row is SnmpVarBindRow => Boolean(row));
}

function MessageReportPreview({ content }: { content: string }) {
  const formatted = useMemo(() => formatMessagePayload(content), [content]);
  const jsonRows = useMemo(() => getJsonFieldRows(formatted.parsed), [formatted.parsed]);
  const delimitedRows = useMemo(() => parseDelimitedMessages(content), [content]);
  const snmpRows = useMemo(() => parseSnmpVarBinds(formatted.text), [formatted.text]);
  const lineCount = content ? content.split(/\r?\n/).length : 0;
  const kind = snmpRows.length > 0 ? 'SNMP VarBind' : formatted.kind;
  const fieldColumns: ColumnsType<MessageFieldRow> = [
    { title: '字段', dataIndex: 'field', width: 220, render: (value: string) => <span className={styles.monoText}>{value}</span> },
    { title: '值', dataIndex: 'value', render: (value: string) => <Typography.Text ellipsis={{ tooltip: value }}>{value}</Typography.Text> },
  ];
  const snmpColumns: ColumnsType<SnmpVarBindRow> = [
    { title: '序号', dataIndex: 'index', width: 70, fixed: 'left' },
    { title: 'OID', dataIndex: 'oid', width: 320, render: (value: string) => <Typography.Text className={styles.monoText} ellipsis={{ tooltip: value }}>{value}</Typography.Text> },
    { title: '字段', dataIndex: 'field', width: 240, render: (value: string) => <Typography.Text ellipsis={{ tooltip: value }}>{value}</Typography.Text> },
    { title: '值', dataIndex: 'value', render: (value: string) => <Typography.Text ellipsis={{ tooltip: value }}>{value || '-'}</Typography.Text> },
  ];
  const delimitedColumns: ColumnsType<DelimitedMessageRow> = [
    { title: '序号', width: 70, render: (_, __, index) => index + 1 },
    { title: '命令', dataIndex: 'command', width: 180, render: (value: string) => <span className={styles.monoText}>{value}</span> },
    { title: '字段数', dataIndex: 'fieldCount', width: 90, render: (value: number) => <Tag>{value}</Tag> },
    { title: '摘要', dataIndex: 'summary', render: (value: string) => <Typography.Text ellipsis={{ tooltip: value }}>{value}</Typography.Text> },
  ];

  return (
    <div className={styles.messagePreview}>
      <div className={styles.csvPreviewMeta}>
        <Tag color={kind === 'JSON' ? 'blue' : kind === '键值报文' ? 'purple' : kind === 'SNMP VarBind' ? 'cyan' : 'default'}>{kind}</Tag>
        <Typography.Text type="secondary">
          {formatBytes(formatted.text.length)} · {lineCount} 行
        </Typography.Text>
      </div>
      {snmpRows.length > 0 && (
        <Table<SnmpVarBindRow>
          className={styles.compactScenarioTable}
          columns={snmpColumns}
          dataSource={snmpRows}
          rowKey="key"
          size="small"
          pagination={{ pageSize: 20, showSizeChanger: false, hideOnSinglePage: snmpRows.length <= 20 }}
          scroll={{ x: 960, y: 300 }}
        />
      )}
      {jsonRows.length > 0 && (
        <Table<MessageFieldRow>
          className={styles.compactScenarioTable}
          columns={fieldColumns}
          dataSource={jsonRows}
          rowKey="key"
          size="small"
          pagination={{ pageSize: 20, showSizeChanger: false, hideOnSinglePage: jsonRows.length <= 20 }}
          scroll={{ x: 760, y: 260 }}
        />
      )}
      {snmpRows.length === 0 && delimitedRows.length > 0 && (
        <Table<DelimitedMessageRow>
          className={styles.compactScenarioTable}
          columns={delimitedColumns}
          dataSource={delimitedRows}
          rowKey="key"
          size="small"
          pagination={{ pageSize: 20, showSizeChanger: false, hideOnSinglePage: delimitedRows.length <= 20 }}
          scroll={{ x: 820, y: 260 }}
        />
      )}
      <pre className={styles.codeBlock}>{formatted.text}</pre>
    </div>
  );
}

function buildEventReportStatus(event: NorthboundPageConfigEvent, fallbackCapabilityName: string): ReportStatusInfo {
  const state = reportStateFromEvent(event);
  const isApiEvent = event.capability === 'api';
  const payload = isApiEvent
    ? friendlyApiEventPayload(event)
    : event.payload || JSON.stringify(event.summary ?? {}, null, 2);
  return {
    key: `event:${event.id}`,
    capabilityName: fallbackCapabilityName,
    state,
    statusText: eventStatusText(event),
    lastTime: formatRunTime(event.created_at),
    artifactType: event.artifact_type === 'file' ? 'file' : 'message',
    artifactName: isApiEvent ? friendlyApiArtifactName(event.artifact_name) : event.artifact_name || '-',
    artifactPath: event.artifact_path || '-',
    size: formatBytes(payload.length),
    targetSummary: eventTargetSummary(event),
    detail: event.error_message || eventDetailText(event),
    payload,
    previewTitle: isApiEvent ? '接口检查结果' : undefined,
    copyLabel: isApiEvent ? '复制结果' : undefined,
    resultTitle: isApiEvent ? `${fallbackCapabilityName || '北向 API'} 接口检查结果` : undefined,
  };
}

function stringValue(value: unknown): string {
  return typeof value === 'string' ? value : '';
}

function boolValue(value: unknown): boolean | undefined {
  return typeof value === 'boolean' ? value : undefined;
}

function deliveryTargetScopePrefix(scope: DeliveryTargetScope, ownerCode: string) {
  return `${scope}:${ownerCode}:`;
}

function deliveryTargetTestKey(scope: DeliveryTargetScope, ownerCode: string, key: string) {
  return `${deliveryTargetScopePrefix(scope, ownerCode)}${key}`;
}

function deliveryTargetTestTitle(row: DeliveryTargetRow): string {
  const name = row.name.trim() || '传输目标';
  const protocolPattern = new RegExp(`(^|[^A-Z0-9])${row.protocol}($|[^A-Z0-9])`, 'i');
  return protocolPattern.test(name) ? name : `${name} ${row.protocol}`;
}

function validateDeliveryTargetRows(
  scope: DeliveryTargetScope,
  ownerCode: string,
  rows: DeliveryTargetRow[],
) {
  const errors: DeliveryTargetValidationErrors = {};
  const rowMessages: string[] = [];
  rows.forEach((row, index) => {
    const rowErrors: Partial<Record<DeliveryTargetValidationField, string>> = {};
    const missingFields: string[] = [];
    if (!row.host.trim()) {
      rowErrors.host = '请输入主机';
      missingFields.push('主机');
    }
    if (!row.username.trim()) {
      rowErrors.username = '请输入用户名';
      missingFields.push('用户名');
    }
    if (!deliveryTargetHasCredential(row)) {
      rowErrors.credential = '请输入密码';
      missingFields.push('密码');
    }
    if (missingFields.length > 0) {
      errors[deliveryTargetTestKey(scope, ownerCode, row.key)] = rowErrors;
      rowMessages.push(`第 ${index + 1} 行缺少${missingFields.join('、')}`);
    }
  });
  return {
    errors,
    message: rowMessages.join('；'),
    valid: rowMessages.length === 0,
  };
}

function replaceDeliveryTargetValidationErrors(
  current: DeliveryTargetValidationErrors,
  scope: DeliveryTargetScope,
  ownerCode: string,
  errors: DeliveryTargetValidationErrors,
) {
  const prefix = deliveryTargetScopePrefix(scope, ownerCode);
  const next = Object.fromEntries(
    Object.entries(current).filter(([key]) => !key.startsWith(prefix)),
  ) as DeliveryTargetValidationErrors;
  return { ...next, ...errors };
}

function deliveryTargetValidationFieldsForPatch(patch: Partial<DeliveryTargetRow>): DeliveryTargetValidationField[] {
  const fields: DeliveryTargetValidationField[] = [];
  if ('host' in patch) fields.push('host');
  if ('username' in patch) fields.push('username');
  if ('credential' in patch) fields.push('credential');
  return fields;
}

function deliveryTestEventValue(event: NorthboundPageConfigEvent, key: string): unknown {
  const summary = event.summary ?? {};
  const payload = parseJSONPayload(event.payload || '') ?? {};
  return summary[key] ?? payload[key];
}

function deliveryTestTargetText(event: NorthboundPageConfigEvent): string {
  const protocol = stringValue(deliveryTestEventValue(event, 'protocol'));
  const host = stringValue(deliveryTestEventValue(event, 'host'));
  const port = deliveryTestEventValue(event, 'port');
  if (protocol && host) {
    return `${protocol.toLowerCase()}://${host}${port ? `:${port}` : ''}`;
  }
  return event.artifact_path || '-';
}

function deliveryTestAuthText(event: NorthboundPageConfigEvent): string {
  const supported = boolValue(deliveryTestEventValue(event, 'auth_probe_supported'));
  const passed = boolValue(deliveryTestEventValue(event, 'auth_probe_passed'));
  if (supported === false) return '未执行认证探测';
  if (passed === true) return '认证通过';
  if (passed === false) return '认证失败';
  return '-';
}

function buildDeliveryTestDisplay(event: NorthboundPageConfigEvent): DeliveryTestDisplayInfo {
  const state = reportStateFromEvent(event);
  const message = event.error_message
    || stringValue(deliveryTestEventValue(event, 'message'))
    || (state === 'success' ? '连接测试通过' : '连接测试失败');
  const tcpReachable = boolValue(deliveryTestEventValue(event, 'tcp_reachable'));
  return {
    state,
    statusText: state === 'success' ? '测试通过' : '测试失败',
    target: deliveryTestTargetText(event),
    tcpText: tcpReachable === true ? 'TCP 可达' : tcpReachable === false ? 'TCP 不可达' : '-',
    authText: deliveryTestAuthText(event),
    message: state === 'success' ? '连接测试通过' : message,
  };
}

function reportStateFromEvent(event: NorthboundPageConfigEvent): ReportState {
  if (event.status === 'success') return 'success';
  if (event.status === 'running') return 'running';
  if (event.status === 'terminated') return 'idle';
  return 'failed';
}

function eventTypeLabel(event: NorthboundPageConfigEvent): string {
  const labels: Record<string, string> = {
    alarm_push: '实时告警',
    connection_rejected: '连接拒绝',
    connection_test: '连接测试',
    contract_check: '接口检查',
    delivery: '文件投递',
    file_sync: '文件补录',
    heartbeat: '心跳',
    login: '登录',
    message_test: '样例报文',
    run: '任务运行',
    server_sessions_reloaded: '会话重载',
    server_start_failed: '端口失败',
    server_started: '端口启动',
    server_stopped: '端口停止',
    sync_request: '同步查询',
  };
  return labels[event.event_type] ?? event.event_type;
}

function eventListSummary(event: NorthboundPageConfigEvent): string {
  const summary = event.summary ?? {};
  const parts: string[] = [];
  const error = event.error_message || valuePreview(summary.error);
  if (event.error_message) parts.push(event.error_message);
  const message = summary.message;
  if (typeof message === 'string' && message) parts.push(message);
  const remoteAddr = summary.remote_addr;
  if (typeof remoteAddr === 'string' && remoteAddr) parts.push(`客户端 ${remoteAddr}`);
  const requestID = summary.request_id;
  if (typeof requestID === 'string' && requestID) parts.push(`请求 ${requestID}`);
  const alarmID = summary.alarm_id;
  if (typeof alarmID === 'string' && alarmID) parts.push(`告警 ${alarmID}`);
  const replayCount = summary.replay_count;
  if (typeof replayCount === 'number') parts.push(`回放 ${replayCount} 条`);
  const sentSessions = summary.sent_sessions;
  if (typeof sentSessions === 'number') parts.push(`发送 ${sentSessions} 个会话`);
  const failedSessions = summary.failed_sessions;
  if (typeof failedSessions === 'number' && failedSessions > 0) parts.push(`失败 ${failedSessions} 个会话`);
  const remotePaths = summary.remote_paths;
  if (Array.isArray(remotePaths) && remotePaths.length > 0) parts.push(`文件 ${remotePaths.length} 个路径`);
  if (parts.length > 0) return [...new Set(parts)].join(' · ');
  if (error && error !== '-') return error;
  return eventTargetSummary(event);
}

// summarizeDeliveryNote aggregates one run's delivery events (all share the same
// summary.run_id) into a single supplementary line, e.g.
// "投递 2 个目标（成功 1 / 失败 1）：ftp://10.0.0.1:21 成功；sftp://... 失败（auth rejected）".
function summarizeDeliveryNote(events: NorthboundPageConfigEvent[]): string {
  if (events.length === 0) return '';
  const success = events.filter((e) => e.status === 'success').length;
  const failed = events.length - success;
  const lines = events.map((e) => {
    const s = e.summary ?? {};
    const protocol = (s.protocol as string | undefined) || '';
    const host = (s.host as string | undefined) || '-';
    const port = s.port ? `:${s.port}` : '';
    const target = `${protocol.toLowerCase()}://${host}${port}`;
    if (e.status === 'success') return `${target} 成功`;
    const reason = e.error_message || (s.message as string | undefined) || '失败';
    return `${target} 失败（${reason}）`;
  });
  return `投递 ${events.length} 个目标（成功 ${success} / 失败 ${failed}）：${lines.join('；')}`;
}

function mergeRunDeliveryStatus(info: ReportStatusInfo, events: NorthboundPageConfigEvent[]): ReportStatusInfo {
  const deliveryNote = summarizeDeliveryNote(events);
  if (!deliveryNote) return info;
  const failed = events.some((event) => event.status !== 'success');
  if (!failed || info.state !== 'success') return { ...info, deliveryNote };
  return {
    ...info,
    state: 'failed',
    statusText: REPORT_TRANSFER_FAILED_TEXT,
    targetSummary: deliveryNote,
    detail: [info.detail, REPORT_TRANSFER_FAILED_DETAIL].filter(Boolean).join(' '),
    deliveryNote,
  };
}

function eventRunId(event: NorthboundPageConfigEvent): string {
  const value = event.summary?.run_id;
  return typeof value === 'string' ? value : '';
}

function groupDeliveryEventsByRunId(events: NorthboundPageConfigEvent[]): DeliveryEventsByRunId {
  return events.reduce<DeliveryEventsByRunId>((acc, event) => {
    if (event.event_type !== 'delivery') return acc;
    const runId = eventRunId(event);
    if (!runId) return acc;
    acc[runId] = [...(acc[runId] ?? []), event];
    return acc;
  }, {});
}

function runHasFailedDelivery(run: NorthboundFileRun, eventsByRunId: DeliveryEventsByRunId): boolean {
  if (run.status !== 'success') return false;
  return (eventsByRunId[run.id] ?? []).some((event) => event.status !== 'success');
}

function selectInitialReportRun(
  items: NorthboundFileRun[],
  eventsByRunId: DeliveryEventsByRunId,
): NorthboundFileRun | undefined {
  return items.find((item) => runHasFailedDelivery(item, eventsByRunId))
    ?? items.find((item) => item.status === 'success')
    ?? items[0];
}

function reportRunDisplayState(
  run: NorthboundFileRun,
  eventsByRunId: DeliveryEventsByRunId,
): { state: ReportState; label: string; tooltip?: string } {
  const deliveryEvents = eventsByRunId[run.id] ?? [];
  if (run.status === 'success' && deliveryEvents.some((event) => event.status !== 'success')) {
    const note = summarizeDeliveryNote(deliveryEvents);
    return {
      state: 'failed',
      label: REPORT_TRANSFER_FAILED_SHORT_TEXT,
      tooltip: note ? `${REPORT_TRANSFER_FAILED_TEXT}：${note}` : REPORT_TRANSFER_FAILED_TEXT,
    };
  }
  if (run.status === 'success') return { state: 'success', label: '成功' };
  if (run.status === 'running') return { state: 'running', label: '生成中' };
  return { state: 'failed', label: '失败' };
}

function eventStatusText(event: NorthboundPageConfigEvent): string {
  if (event.status === 'success') return '正常';
  if (event.status === 'running') return '处理中';
  if (event.status === 'terminated') return '未启用';
  return '失败';
}

function eventTargetSummary(event: NorthboundPageConfigEvent): string {
  const summary = event.summary ?? {};
  const message = summary.message;
  if (typeof message === 'string' && message) return message;
  if (event.error_message) return event.error_message;
  if (event.event_type === 'connection_test') return '连接测试已记录';
  if (event.event_type === 'delivery') return '文件投递结果已记录';
  if (event.event_type === 'message_test') return '测试报文已记录';
  if (event.event_type === 'contract_check') return '接口检查已记录';
  if (event.event_type === 'run') return '运行记录已生成';
  return '状态已记录';
}

function eventDetailText(event: NorthboundPageConfigEvent): string {
  if (event.capability === 'delivery' && event.event_type === 'delivery') return 'FTP/SFTP 文件投递结果来自后端，包含远端路径、尝试次数和耗时。';
  if (event.capability === 'delivery') return 'FTP/SFTP 目标连接探测结果来自后端。';
  if (event.capability === 'snmp') return 'SNMP 告警字段和 OID 顺序按 omcAlarmMIB.mib 生成。';
  if (event.capability === 'socket') return 'Socket 服务端登录、心跳、同步和实时告警推送结果可查看。';
  if (event.capability === 'api') return '北向 API 接口检查结果可查看，包含请求方式、接口地址、支持状态和返回字段清单。';
  return '北向页面化配置事件。';
}

function formatRunTime(value?: string): string {
  if (!value) return '-';
  return value.replace('T', ' ').slice(0, 19);
}

function formatBytes(value?: number): string {
  const size = Number(value ?? 0);
  if (size >= 1024 * 1024) return `${(size / 1024 / 1024).toFixed(2)} MB`;
  if (size >= 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${size} B`;
}

function getScenarioPeriodRows(scenario: ScenarioRow): ScenarioPeriodRow[] {
  const rows: ScenarioPeriodRow[] = scenario.groups.flatMap(splitFileGroupByTech).map((groupItem) => ({
    key: groupItem.id,
    domain: groupItem.domain,
    scope: getGroupTechLabel(groupItem),
    format: groupItem.format,
    period: groupItem.period,
    trigger: formatScheduleLabel(groupItem.period, groupItem.cron),
    cron: groupItem.cron,
    objects: getGroupObjectsLabel(groupItem),
    path: groupItem.path,
    fileName: getGroupFileNameTemplate(groupItem),
    csvSeparator: groupItem.csvSeparator,
    compressionEnabled: groupItem.compressionEnabled,
    compressionFormat: groupItem.compressionFormat,
    selectedFields: groupItem.selectedFields,
    objectProfiles: getGroupObjectProfiles(groupItem),
  }));

  if (scenario.logs) {
    rows.push({
      key: 'logs',
      domain: 'LOG',
      scope: scenario.logs.mode,
      format: scenario.logs.mode === 'custom' ? 'TXT' : 'CSV',
      period: scenario.logs.period,
      trigger: formatScheduleLabel(scenario.logs.period, scenario.logs.cron),
      cron: scenario.logs.cron,
      objects: scenario.logs.mode === 'custom' ? 'login, operation' : 'login_fix, operation_fix',
      path: PATH_LOG,
      fileName: scenario.logs.mode === 'custom'
        ? LOG_CUSTOM_NAME
        : LOG_FIX_NAME,
      compressionEnabled: true,
      compressionFormat: 'gz',
    });
  }

  return rows;
}

interface DomainSummary {
  domain: Domain;
  objects: string[];
  scopes: string[];
  periods: string[];
  formats: Format[];
}

function summarizeValues(values: string[], max = 3): string {
  const uniqueValues = [...new Set(values.filter(Boolean))];
  if (uniqueValues.length <= max) return uniqueValues.join('/');
  return `${uniqueValues.slice(0, max).join('/')} +${uniqueValues.length - max}`;
}

function summarizeScenarioDomains(scenario: ScenarioRow): DomainSummary[] {
  const summaries = new Map<Domain, DomainSummary>();
  getScenarioPeriodRows(scenario).forEach((periodRow) => {
    const existing = summaries.get(periodRow.domain) ?? {
      domain: periodRow.domain,
      objects: [],
      scopes: [],
      periods: [],
      formats: [],
    };
    existing.objects.push(...splitObjects(periodRow.objects).map((object) => parseObjectToken(object).code));
    existing.scopes.push(formatScopeLabel(periodRow.scope));
    existing.periods.push(formatPeriodLabel(periodRow.period));
    existing.formats.push(periodRow.format);
    summaries.set(periodRow.domain, existing);
  });
  return domainOptions
    .map((option) => summaries.get(option.value as Domain))
    .filter((summary): summary is DomainSummary => Boolean(summary));
}

function getScenarioOutputTooltip(scenario: ScenarioRow): string {
  return summarizeScenarioDomains(scenario)
    .map((summary) => `${summary.domain}: ${summarizeValues(summary.objects, 8)} (${summarizeValues(summary.scopes, 4) || '默认'})`)
    .join('\n');
}

function getScenarioScheduleTooltip(scenario: ScenarioRow): string {
  return getScenarioPeriodRows(scenario)
    .map((periodRow) => `${periodRow.domain} ${formatScopeLabel(periodRow.scope)}: ${formatPeriodLabel(periodRow.period)} / ${periodRow.trigger}`)
    .join('\n');
}

function getScenarioFileTooltip(scenario: ScenarioRow): string {
  return getScenarioPeriodRows(scenario)
    .map((periodRow) => `${periodRow.domain}: ${periodRow.path}${periodRow.fileName}`)
    .join('\n');
}

function getScenarioCompressionTags(scenario: ScenarioRow): string[] {
  return [...new Set(getScenarioPeriodRows(scenario).map((periodRow) => (
    periodRow.compressionEnabled ? periodRow.compressionFormat.toUpperCase() : '不压缩'
  )))];
}

function getScenarioDomains(scenario: ScenarioRow): Domain[] {
  const domains = scenario.groups.map((groupItem) => groupItem.domain);
  if (scenario.logs) domains.push('LOG');
  return [...new Set(domains)];
}

function domainTag(domain: Domain) {
  const colors: Record<Domain, string> = {
    CM: 'blue',
    PM: 'cyan',
    MR: 'purple',
    LOG: 'gold',
    INVENTORY: 'green',
  };
  return <Tag color={colors[domain]}>{domain}</Tag>;
}

function typeTag(type: MetricType) {
  return <Tag color={type === 'kpi' ? 'purple' : 'blue'}>{type.toUpperCase()}</Tag>;
}

function apiMethodTag(method: ApiMethod) {
  const colors: Record<ApiMethod, string> = {
    GET: 'green',
    POST: 'blue',
    PUT: 'orange',
    DELETE: 'red',
  };
  return <Tag color={colors[method]}>{method}</Tag>;
}

function getApiMeta(row: NorthboundApiRow) {
  const meta = apiMetaByKey[row.key] ?? {};
  return {
    apiKind: row.apiKind ?? meta.apiKind ?? '业务复用',
    fieldContract: row.fieldContract ?? meta.fieldContract ?? '当前返回字段',
    responseFields: row.responseFields ?? meta.responseFields ?? currentEnvelopeFields,
  };
}

const hiddenApiCatalogFields = new Set(['resparams2']);

function apiFieldDisplayList(fields: string[]): string[] {
  const normalizedFields = fields
    .map((field) => normalizeApiFieldDisplay(field))
    .filter((field) => field.length > 0 && !hiddenApiCatalogFields.has(field.toLowerCase()));
  return [...new Set(normalizedFields)];
}

function normalizeApiFieldDisplay(field: string): string {
  const normalized = field.trim();
  if (!normalized) return '';
  if (normalized.startsWith('CSV:')) return normalized.slice(4).trim();
  if (normalized.includes(' ') || normalized.includes('/')) return normalized;
  return normalized.replace(/\?/g, '').replace(/\[\]/g, '');
}

function apiModuleDisplay(row: NorthboundApiRow): string {
  const normalizedModule = apiModuleLabel(row.module);
  if (normalizedModule !== row.module) return normalizedModule;
  if (row.dataType) return apiModuleLabel(row.dataType);
  const fallbackKey = row.key.split('-')[0] ?? '';
  const fallbackModule = apiModuleLabel(fallbackKey);
  if (fallbackModule && fallbackModule !== fallbackKey) return fallbackModule;
  return row.module || '北向接口';
}

function apiModuleColor(module: string): string {
  const colors: Record<string, string> = {
    认证鉴权: 'geekblue',
    北向用户管理: 'cyan',
    北向接口日志: 'gold',
    数据同步: 'blue',
    设备管理: 'green',
    设备组管理: 'lime',
    参数配置: 'purple',
    异步任务: 'orange',
    高级任务: 'volcano',
    设备日志收集: 'magenta',
    告警查询: 'red',
    性能管理: 'blue',
    'MR 数据': 'purple',
    'Inventory 导出': 'cyan',
    'HTTP Push': 'cyan',
    主备服务器: 'geekblue',
  };
  return colors[module] ?? 'default';
}

function apiModuleTag(row: NorthboundApiRow) {
  const module = apiModuleDisplay(row);
  return <Tag color={apiModuleColor(module)}>{module}</Tag>;
}

function apiKindLabel(kind?: ApiKind): string {
  const labels: Record<ApiKind, string> = {
    正式北向: '标准北向',
    业务复用: '旧系统兼容',
    鉴权管理: '认证管理',
    日志管理: '日志管理',
  };
  return labels[(kind ?? '业务复用') as ApiKind] ?? kind ?? '旧系统兼容';
}

function apiKindTag(kind?: ApiKind) {
  const label = apiKindLabel(kind);
  const colors: Record<string, string> = {
    标准北向: 'processing',
    旧系统兼容: 'default',
    认证管理: 'geekblue',
    日志管理: 'gold',
  };
  return <Tag color={colors[label] ?? 'default'}>{label}</Tag>;
}

function normalizeLegacyApiText(value: string) {
  return value
    .replaceAll('JWT 或 API Key，', 'JWT，');
}

function apiUsageSummary(row: NorthboundApiRow): string {
  const module = apiModuleDisplay(row);
  if (module === '认证鉴权') return '外部系统先调用该接口获取访问 Token，再调用其它北向 API。';
  if (module === '北向用户管理') return '用于外部系统维护北向 API 调用账号，包括查询、新增、修改和删除。';
  if (module === '北向接口日志') return '用于外部系统查询或导出北向 API 调用日志。';
  if (module === '数据同步') return '用于外部系统拉取设备、告警等北向同步数据。';
  if (module === '设备管理') return '用于外部系统查询设备清单、设备详情、设备状态、注册设备或触发设备操作。';
  if (module === '设备组管理') return '用于外部系统同步和维护设备分组，以及维护分组内设备关系。';
  if (module === '参数配置') return '用于外部系统查询或设置设备参数，返回结果以当前系统设备参数能力为准。';
  if (module === '告警查询') return '用于外部系统查询当前告警、历史告警或告警统计信息。';
  if (module === '性能管理') return '用于外部系统导出性能数据或查询聚合指标。';
  if (module === 'MR 数据') return '用于外部系统查询或导出 MR 数据。';
  if (module === '设备日志收集') return '用于外部系统触发设备运行/故障日志收集并查询处理结果。';
  if (module === '异步任务' || module === '高级任务') return '用于外部系统查询或创建异步任务，并通过任务 ID 查看执行结果。';
  if (module === 'HTTP Push') return '用于外部系统维护 HTTP Push 目标、熔断和死信重放。';
  if (module === '主备服务器') return '用于外部系统查询或维护北向主备服务器配置。';
  return '用于外部系统调用当前系统已开放的北向业务能力。';
}

function statusTag(enabled: boolean) {
  return enabled ? <Tag color="success">启用</Tag> : <Tag color="default">关闭</Tag>;
}

function reportStateTag(state: ReportState, label: string, tooltip?: string) {
  const colors: Record<ReportState, string> = {
    success: 'success',
    failed: 'error',
    running: 'processing',
    idle: 'default',
  };
  const tag = <Tag color={colors[state]}>{label}</Tag>;
  if (tooltip && tooltip !== label) return <Tooltip title={tooltip}>{tag}</Tooltip>;
  return tag;
}

function deliveryProtocolTag(protocol: DeliveryProtocol) {
  return <Tag color={protocol === 'SFTP' ? 'blue' : 'cyan'}>{protocol}</Tag>;
}

function scenarioDisplayName(scenario: ScenarioRow, locale: string) {
  if (locale === 'en-US') {
    return scenario.scenarioNameEn || scenario.scenarioName || scenario.code;
  }
  return scenario.scenarioName || scenario.scenarioNameEn || scenario.code;
}

function socketProfileTag(profile: SocketProfile, translate: (value: string) => string = (value) => value) {
  return <Tag color={profile === 'CTCC' ? 'blue' : 'purple'}>{translate(profile === 'CTCC' ? '电信' : '联通')}</Tag>;
}

function snmpVersionLabel(version: SnmpVersion) {
  return version === 'v2' ? 'SNMP V2C' : 'SNMP V3';
}

function snmpVersionForKey(key: string, fallback: SnmpVersion): SnmpVersion {
  if (key === 'snmp-v2-primary') return 'v2';
  if (key === 'snmp-v3-inform') return 'v3';
  return fallback;
}

function snmpVersionTag(version: SnmpVersion) {
  const colors: Record<SnmpVersion, string> = {
    v2: 'blue',
    v3: 'purple',
  };
  return <Tag color={colors[version]}>{version === 'v2' ? 'V2C' : 'V3'}</Tag>;
}

function snmpV3SecurityLevel(row: SnmpAlarmTargetRow): SnmpV3SecurityLevel {
  if (row.privProtocol?.trim()) {
    return 'authPriv';
  }
  if (row.authProtocol?.trim()) {
    return 'authNoPriv';
  }
  return 'noAuthNoPriv';
}

function snmpV3SecurityLevelLabel(level: SnmpV3SecurityLevel) {
  switch (level) {
    case 'authPriv':
      return '认证并加密';
    case 'authNoPriv':
      return '仅认证';
    case 'noAuthNoPriv':
    default:
      return '不认证不加密';
  }
}

function applySnmpV3SecurityLevel(row: SnmpAlarmTargetRow, level: SnmpV3SecurityLevel): SnmpAlarmTargetRow {
  if (level === 'noAuthNoPriv') {
    return { ...row, authProtocol: '', authCredential: '', privProtocol: '', privCredential: '' };
  }
  if (level === 'authNoPriv') {
    return {
      ...row,
      authProtocol: row.authProtocol?.trim() || 'SHA',
      privProtocol: '',
      privCredential: '',
    };
  }
  return {
    ...row,
    authProtocol: row.authProtocol?.trim() || 'SHA',
    privProtocol: row.privProtocol?.trim() || 'DES',
  };
}

function snmpNotificationTag(type: SnmpNotificationType) {
  return <Tag color={type === 'Trap' ? 'blue' : 'green'}>{type}</Tag>;
}

function getSocketFields(profile: SocketProfile) {
  return profile === 'CUCC' ? socketCuccFields : socketCtccFields;
}

function getSocketAccountPurpose(type: SocketAccountRow['type']) {
  return type === 'ftp' ? '登录、告警文件同步请求' : '登录、实时告警、历史消息同步';
}

function socketCredentialPlaceholder(value?: string) {
  return value === storedCredentialText ? '未修改保持原密码' : '请输入密码';
}

interface MaskedCredentialInputProps {
  value?: string;
  placeholder?: string;
  readOnly?: boolean;
  status?: 'error' | 'warning';
  width?: number | string;
  minLength?: number;
  storedValues?: string[];
  onChange?: (value: string) => void;
}

function normalizeMaskedCredentialInput(
  nextValue: string,
  currentValue: string,
  visible: boolean,
  storedValues: string[],
): string {
  const isStored = storedValues.includes(currentValue);
  if (!nextValue) return '';
  if (isStored && nextValue.startsWith(credentialMaskText)) {
    return nextValue.slice(credentialMaskText.length);
  }
  if (isStored && !visible && credentialMaskText.startsWith(nextValue)) return '';
  return nextValue;
}

function MaskedCredentialInput({
  value,
  placeholder = '请输入密码',
  readOnly = false,
  status,
  width = '100%',
  minLength,
  storedValues = [storedCredentialText],
  onChange,
}: MaskedCredentialInputProps) {
  const [visible, setVisible] = useState(false);
  const currentValue = value ?? '';
  const hasValue = currentValue !== '';
  const isStored = storedValues.includes(currentValue);
  const displayValue = hasValue && isStored ? credentialMaskText : currentValue;

  return (
    <Input.Password
      value={displayValue}
      readOnly={readOnly}
      size="small"
      minLength={minLength}
      placeholder={placeholder}
      status={status}
      className={styles.monoText}
      style={{ width }}
      visibilityToggle={{
        visible,
        onVisibleChange: setVisible,
      }}
      onChange={(event) => {
        if (!onChange || readOnly) return;
        onChange(normalizeMaskedCredentialInput(event.target.value, currentValue, visible, storedValues));
      }}
    />
  );
}

function MaskedCredentialPreview({ value, width = 150 }: { value?: string; width?: number | string }) {
  if (!value) return <span />;
  return <MaskedCredentialInput value={value} readOnly width={width} />;
}

function cloneDefaultSocketAccounts(profile: SocketProfile, configKey: string) {
  return socketDefaultAccountsByProfile[profile].map((account) => ({
    ...account,
    key: `${configKey}-${account.key}`,
  }));
}

function endpointText(host: string, port: number) {
  return `${host}:${port}`;
}

const defaultFieldTarget = getFirstFieldTarget(defaultEditorPeriodRows);

// PeriodTextInput isolates the heavy "上传目录模板"/"文件名模板" text inputs from the
// giant parent component. Typing only re-renders this tiny cell (local state); the
// upstream editorPeriodRows is updated once on blur, so a keystroke no longer triggers
// a full re-render of the 7683-line page (the previous cause of editor typing lag).
// The effect keeps local in sync when the row value changes externally (e.g. a domain
// switch resets the template), so the field stays fully controlled at the row level.
interface PeriodTextInputProps {
  value: string;
  rowKey: string;
  field: 'path' | 'fileName';
  onCommit: (rowKey: string, field: 'path' | 'fileName', value: string) => void;
  className?: string;
  ariaLabel?: string;
}

const PeriodTextInput = memo(function PeriodTextInput({
  value,
  rowKey,
  field,
  onCommit,
  className,
  ariaLabel,
}: PeriodTextInputProps) {
  const [local, setLocal] = useState(value);
  useEffect(() => {
    setLocal(value);
  }, [value]);
  return (
    <Input
      className={className}
      aria-label={ariaLabel}
      value={local}
      onChange={(event) => setLocal(event.target.value)}
      onBlur={() => {
        if (local !== value) {
          onCommit(rowKey, field, local);
        }
      }}
    />
  );
});

interface CSVSeparatorInputProps {
  value?: string;
  rowKey: string;
  onCommit: (rowKey: string, value: string | undefined) => void;
}

const CSVSeparatorInput = memo(function CSVSeparatorInput({
  value,
  rowKey,
  onCommit,
}: CSVSeparatorInputProps) {
  const [local, setLocal] = useState(value ?? '');
  useEffect(() => {
    setLocal(value ?? '');
  }, [value]);
  return (
    <AutoComplete
      value={local}
      options={csvSeparatorOptions}
      placeholder="默认 ,"
      style={{ width: 112 }}
      onChange={setLocal}
      onBlur={() => {
        const normalized = normalizeCSVSeparatorInput(local);
        setLocal(normalized ?? '');
        if ((normalized ?? '') !== (value ?? '')) {
          onCommit(rowKey, normalized);
        }
      }}
      onSelect={(selectedValue) => {
        const normalized = normalizeCSVSeparatorInput(selectedValue);
        setLocal(normalized ?? '');
        if ((normalized ?? '') !== (value ?? '')) {
          onCommit(rowKey, normalized);
        }
      }}
    />
  );
});

interface FieldConfigSectionProps {
  periodRows: ScenarioPeriodRow[];
  pmMetricRows: PmMetric[];
  pmLoading: boolean;
  initialFieldRowsByTarget: Record<string, ReportFieldRow[]>;
}
export interface FieldConfigSectionHandle {
  getFieldRowsByTarget: () => Record<string, ReportFieldRow[]>;
}

// Field/metric config isolated in its own memoized component so toggling/adding/removing
// fields re-renders only this section, not the whole page. The parent reads the current
// selection back via the imperative ref at save time.
const FieldConfigSection = memo(forwardRef<FieldConfigSectionHandle, FieldConfigSectionProps>(
  function FieldConfigSection({ periodRows, pmMetricRows, pmLoading, initialFieldRowsByTarget }, ref) {
    const nt = useNorthboundI18n();
    const firstTarget = getFirstFieldTarget(periodRows);
    const [fieldConfigDomain, setFieldConfigDomain] = useState<Domain>(firstTarget?.domain ?? 'CM');
    const [fieldConfigTargetKey, setFieldConfigTargetKey] = useState(firstTarget?.key ?? '');
    const [fieldTechFilter, setFieldTechFilter] = useState<FieldTechFilter>(() => getFieldTechFilterForTarget(firstTarget));
    const [fieldCandidateKey, setFieldCandidateKey] = useState<string>();
    const [fieldCandidateSearch, setFieldCandidateSearch] = useState('');
    const [fieldCandidateOpen, setFieldCandidateOpen] = useState(false);
    const [fieldRowsByTarget, setFieldRowsByTarget] = useState<Record<string, ReportFieldRow[]>>(initialFieldRowsByTarget);
    const [backendFieldCatalog, setBackendFieldCatalog] = useState<Record<string, ReportFieldRow[]>>({});

    useImperativeHandle(ref, () => ({ getFieldRowsByTarget: () => fieldRowsByTarget }), [fieldRowsByTarget]);

    const fieldTargets = useMemo(() => getFieldTargets(periodRows), [periodRows]);
    const hasFieldTargets = fieldTargets.length > 0;
    const fieldDomainOptions = useMemo(
      () => [...new Set(fieldTargets.map((t) => t.domain))].map((d) => ({ label: d, value: d })),
      [fieldTargets],
    );
    const activeFieldTechFilter = useMemo(
      () => getFieldTechFilterForDomain(fieldConfigDomain, fieldTargets, fieldTechFilter),
      [fieldConfigDomain, fieldTargets, fieldTechFilter],
    );
    const fieldTechFilterOptions = useMemo(
      () => getFieldTechFilterOptions(fieldConfigDomain, fieldTargets),
      [fieldConfigDomain, fieldTargets],
    );
    const visibleFieldTargets = useMemo(
      () => fieldTargets.filter((t) => t.domain === fieldConfigDomain).filter((t) => targetMatchesTechFilter(t, activeFieldTechFilter)),
      [fieldTargets, fieldConfigDomain, activeFieldTechFilter],
    );
    const fieldTargetOptions = useMemo(
      () => visibleFieldTargets.map((t) => ({ label: formatFieldTargetLabel(t), value: t.key })),
      [visibleFieldTargets],
    );
    const selectedFieldTarget = useMemo(
      () => visibleFieldTargets.find((t) => t.key === fieldConfigTargetKey) ?? visibleFieldTargets[0],
      [visibleFieldTargets, fieldConfigTargetKey],
    );
    useEffect(() => {
      if (fieldTargets.length === 0) {
        setFieldConfigTargetKey('');
        setFieldCandidateKey(undefined);
        setFieldCandidateSearch('');
        setFieldTechFilter('ALL');
        return;
      }
      const domainExists = fieldTargets.some((t) => t.domain === fieldConfigDomain);
      const nextDomain = domainExists ? fieldConfigDomain : fieldTargets[0].domain;
      if (!domainExists) {
        setFieldConfigDomain(nextDomain);
      }
      const nextFilter = getFieldTechFilterForDomain(nextDomain, fieldTargets, fieldTechFilter);
      if (nextFilter !== fieldTechFilter) {
        setFieldTechFilter(nextFilter);
      }
      const targetExists = fieldTargets.some((t) => (
        t.key === fieldConfigTargetKey
        && t.domain === nextDomain
        && targetMatchesTechFilter(t, nextFilter)
      ));
      if (!targetExists) {
        const nextTarget = fieldTargets.find((t) => t.domain === nextDomain && targetMatchesTechFilter(t, nextFilter))
          ?? fieldTargets.find((t) => t.domain === nextDomain)
          ?? fieldTargets[0];
        setFieldConfigTargetKey(nextTarget.key);
      }
    }, [fieldConfigDomain, fieldConfigTargetKey, fieldTargets, fieldTechFilter]);
    useEffect(() => {
      setFieldCandidateKey(undefined);
      setFieldCandidateSearch('');
      setFieldCandidateOpen(false);
    }, [selectedFieldTarget?.key]);
    const backendCatalogKey = selectedFieldTarget
      ? `${selectedFieldTarget.domain}:${selectedFieldTarget.objectCode}:${getTargetTech(selectedFieldTarget) ?? 'ALL'}:${selectedFieldTarget.profile}`
      : '';
    const selectedFieldKeysForTarget = useMemo(() => {
      if (!selectedFieldTarget) return [];
      const ownerRow = periodRows.find((row) => periodRowTargetKeys(row).includes(selectedFieldTarget.key));
      return ownerRow?.selectedFields ?? [];
    }, [periodRows, selectedFieldTarget?.key]);
    const shouldBuildFieldCandidates = fieldCandidateOpen
      || Boolean(fieldCandidateKey)
      || Boolean(fieldCandidateSearch.trim())
      || selectedFieldKeysForTarget.length > 0;
    const shouldLoadBackendFieldCatalog = shouldBuildFieldCandidates || Boolean(selectedFieldTarget?.profile);
    useEffect(() => {
      const target = selectedFieldTarget;
      if (!target) return;
      const cacheKey = backendCatalogKey;
      if (!shouldLoadBackendFieldCatalog || !cacheKey || backendFieldCatalog[cacheKey]) return;
      const tech = getTargetTech(target);
      let cancelled = false;
      void northboundPageConfigApi.getFields({
        domain: target.domain,
        object: target.objectCode,
        ...(tech ? { tech } : {}),
        ...(target.profile ? { profile: target.profile } : {}),
      })
        .then((resp) => {
          if (cancelled) return;
          const rows: ReportFieldRow[] = (resp.items ?? []).map((f, index) => mapApiFieldToReportRow(f, target, index));
          setBackendFieldCatalog((prev) => (prev[cacheKey] ? prev : { ...prev, [cacheKey]: rows }));
        })
        .catch(() => { /* fall back to static catalog */ });
      return () => { cancelled = true; };
    }, [backendCatalogKey, selectedFieldTarget, backendFieldCatalog, shouldLoadBackendFieldCatalog]);
    const backendRowsForSelectedTarget = backendCatalogKey ? backendFieldCatalog[backendCatalogKey] : undefined;
    const fieldBaseRows = useMemo(
      () => {
        if (selectedFieldTarget?.profile && backendRowsForSelectedTarget && backendRowsForSelectedTarget.length > 0) {
          return backendRowsForSelectedTarget;
        }
        return getReportFieldRows(selectedFieldTarget, pmMetricRows);
      },
      [backendRowsForSelectedTarget, selectedFieldTarget, pmMetricRows],
    );
    const fieldCandidatePoolRows = useMemo(
      () => {
        if (!shouldBuildFieldCandidates) return [];
        return dedupeFieldCandidateRows([
          ...(backendCatalogKey && backendFieldCatalog[backendCatalogKey] ? backendFieldCatalog[backendCatalogKey] : []),
          ...getAvailableReportFieldRows(selectedFieldTarget, pmMetricRows),
        ]);
      },
      [backendCatalogKey, backendFieldCatalog, selectedFieldTarget, pmMetricRows, shouldBuildFieldCandidates],
    );
    const targetFieldRows = useMemo(() => {
      if (!selectedFieldTarget) return [];
      return fieldRowsByTarget[selectedFieldTarget.key] ?? fieldBaseRows;
    }, [fieldBaseRows, fieldRowsByTarget, selectedFieldTarget]);
    useEffect(() => {
      if (!selectedFieldTarget) return;
      const selectedFields = selectedFieldKeysForTarget;
      if (!selectedFields || selectedFields.length === 0) return;
      const wanted = new Set(selectedFields.map((key) => key.toLowerCase()));
      const selectedRows = dedupeFieldCandidateRows([...fieldBaseRows, ...fieldCandidatePoolRows])
        .filter((row) => wanted.has(selectionKey(row).toLowerCase()));
      if (selectedRows.length === 0) return;
      setFieldRowsByTarget((prev) => {
        const currentRows = prev[selectedFieldTarget.key] ?? [];
        const existing = new Set(currentRows.map((row) => selectionKey(row).toLowerCase()));
        const missingRows = selectedRows.filter((row) => !existing.has(selectionKey(row).toLowerCase()));
        if (missingRows.length === 0 && currentRows.length > 0) return prev;
        return {
          ...prev,
          [selectedFieldTarget.key]: currentRows.length > 0
            ? [...currentRows, ...missingRows]
            : selectedRows,
        };
      });
    }, [fieldBaseRows, fieldCandidatePoolRows, selectedFieldKeysForTarget, selectedFieldTarget]);
    const fieldConfigRows = useMemo(
      () => targetFieldRows.filter((row) => rowMatchesTechFilter(row, activeFieldTechFilter)),
      [activeFieldTechFilter, targetFieldRows],
    );
    const fieldCandidateRows = useMemo(() => {
      if (!shouldBuildFieldCandidates) return [];
      const currentIds = new Set(targetFieldRows.map(fieldCandidateIdentity));
      return fieldCandidatePoolRows
        .filter((row) => !currentIds.has(fieldCandidateIdentity(row)))
        .filter((row) => rowMatchesTechFilter(row, activeFieldTechFilter));
    }, [fieldCandidatePoolRows, activeFieldTechFilter, shouldBuildFieldCandidates, targetFieldRows]);
    const displayedFieldCandidateRows = useMemo(
      () => {
        const rows = getVisibleFieldCandidateRows(fieldCandidateRows, fieldCandidateSearch);
        const selectedRow = fieldCandidateRows.find((row) => row.key === fieldCandidateKey);
        if (!selectedRow || rows.some((row) => row.key === selectedRow.key)) return rows;
        return [selectedRow, ...rows];
      },
      [fieldCandidateRows, fieldCandidateKey, fieldCandidateSearch],
    );
    const fieldCandidateOptions = useMemo(
      () => displayedFieldCandidateRows.map((row) => ({
        label: `${formatReportFieldTech(row)} ${row.cnName ?? row.outputAlias} / ${row.outputAlias} / ${row.systemField}`,
        value: row.key,
      })),
      [displayedFieldCandidateRows],
    );

    const updateCurrentFieldRows = useCallback((rows: ReportFieldRow[]) => {
      if (!selectedFieldTarget) return;
      setFieldRowsByTarget((prev) => ({ ...prev, [selectedFieldTarget.key]: rows }));
    }, [selectedFieldTarget]);
    const addFieldConfigRow = useCallback(() => {
      const row = fieldCandidateRows.find((candidate) => candidate.key === fieldCandidateKey);
      if (!row) return;
      const enabledRow = { ...row, enabled: true };
      const baseOrder = new Map(fieldCandidatePoolRows.map((item, index) => [fieldCandidateIdentity(item), index]));
      const nextRows = [...targetFieldRows, enabledRow].sort((a, b) => (
        (baseOrder.get(fieldCandidateIdentity(a)) ?? Number.MAX_SAFE_INTEGER)
        - (baseOrder.get(fieldCandidateIdentity(b)) ?? Number.MAX_SAFE_INTEGER)
      ));
      updateCurrentFieldRows(nextRows);
      setFieldCandidateKey(undefined);
      setFieldCandidateSearch('');
    }, [fieldCandidateRows, fieldCandidateKey, fieldCandidatePoolRows, targetFieldRows, updateCurrentFieldRows]);
    const removeFieldConfigRow = useCallback((row: ReportFieldRow) => {
      updateCurrentFieldRows(targetFieldRows.filter((item) => fieldIdentity(item) !== fieldIdentity(row)));
    }, [targetFieldRows, updateCurrentFieldRows]);
    const toggleFieldEnabled = useCallback((row: ReportFieldRow, enabled: boolean) => {
      updateCurrentFieldRows(
        targetFieldRows.map((item) => (fieldIdentity(item) === fieldIdentity(row) ? { ...item, enabled } : item)),
      );
    }, [targetFieldRows, updateCurrentFieldRows]);
    const changeFieldConfigDomain = useCallback((domain: Domain) => {
      const nextFilter = getFieldTechFilterForDomain(domain, fieldTargets);
      setFieldConfigDomain(domain);
      setFieldTechFilter(nextFilter);
      const nextTarget = fieldTargets.find((t) => t.domain === domain && targetMatchesTechFilter(t, nextFilter))
        ?? fieldTargets.find((t) => t.domain === domain);
      if (nextTarget) setFieldConfigTargetKey(nextTarget.key);
    }, [fieldTargets]);
    const changeFieldTechFilter = useCallback((techFilter: FieldTechFilter) => {
      const nextFilter = getFieldTechFilterForDomain(fieldConfigDomain, fieldTargets, techFilter);
      setFieldTechFilter(nextFilter);
      const nextTarget = fieldTargets.find((t) => t.domain === fieldConfigDomain && targetMatchesTechFilter(t, nextFilter));
      setFieldConfigTargetKey(nextTarget?.key ?? '');
    }, [fieldTargets, fieldConfigDomain]);

    const editableFieldConfigColumns: ColumnsType<ReportFieldRow> = [
      {
        title: '操作',
        width: 74,
        fixed: 'left',
        render: (_, row) => (
          <Popconfirm
            title={nt('确认删除该字段/指标？')}
            description={nt('删除后需保存草稿才会生效。')}
            okText={nt('删除')}
            cancelText={nt('取消')}
            okButtonProps={{ danger: true }}
            onConfirm={() => removeFieldConfigRow(row)}
          >
            <Tooltip title="从当前模板删除">
              <Button
                aria-label={`删除字段 ${row.outputAlias}`}
                type="text"
                danger
                size="small"
                icon={<DeleteOutlined />}
              />
            </Tooltip>
          </Popconfirm>
        ),
      },
      {
        title: '上报',
        width: 76,
        render: (_, row) => <Switch size="small" checked={row.enabled} onChange={(checked) => toggleFieldEnabled(row, checked)} checkedChildren="开" unCheckedChildren="关" />,
      },
      { title: '对象', width: 90, render: (_, row) => <Tag>{formatReportFieldObject(row)}</Tag> },
      { title: '制式', width: 80, render: (_, row) => <Tag>{formatReportFieldTech(row)}</Tag> },
      {
        title: '输出别名',
        dataIndex: 'outputAlias',
        width: 260,
        render: (value: string) => <Input defaultValue={value} className={styles.monoText} />,
      },
      {
        title: '系统字段/指标',
        dataIndex: 'systemField',
        width: 260,
        render: (value: string) => <span className={styles.monoText}>{value}</span>,
      },
      {
        title: '取数字段',
        dataIndex: 'source',
        width: 300,
        render: (value: string) => <span className={styles.monoText}>{value}</span>,
      },
      {
        title: '适用范围',
        dataIndex: 'productClasses',
        width: 180,
        render: (value?: string[]) => <span>{formatProductClasses(value)}</span>,
      },
      {
        title: '类型/口径',
        width: 150,
        render: (_, row) => row.metricType
          ? <Space size={4}>{typeTag(row.metricType)}<Tag>{row.statisType}</Tag></Space>
          : <Tag>{row.dataType}</Tag>,
      },
      {
        title: '单位/渲染',
        width: 130,
        render: (_, row) => row.unit ? <span>{row.unit}</span> : <Tag>{row.renderer}</Tag>,
      },
      { title: '中文名', dataIndex: 'cnName', width: 220, render: (value?: string) => value || '-' },
    ];

    return (
      <div className={styles.editorSection}>
        <div className={styles.editorSectionHeader}>
          <Typography.Text strong>字段/指标配置</Typography.Text>
          <Space size={8} wrap className={styles.fieldConfigTools}>
            <Select
              value={hasFieldTargets ? fieldConfigDomain : undefined}
              placeholder="业务域"
              style={{ width: 120 }}
              options={fieldDomainOptions}
              disabled={!hasFieldTargets}
              onChange={(value) => changeFieldConfigDomain(value as Domain)}
            />
            <Select
              value={activeFieldTechFilter}
              style={{ width: 132 }}
              options={fieldTechFilterOptions}
              disabled={!hasFieldTargets || !isRadioTechDomain(fieldConfigDomain)}
              onChange={(value) => changeFieldTechFilter(value as FieldTechFilter)}
            />
            <Select
              value={selectedFieldTarget?.key}
              placeholder="对象"
              style={{ width: 132 }}
              options={fieldTargetOptions}
              disabled={!hasFieldTargets}
              onChange={setFieldConfigTargetKey}
              notFoundContent="无对象"
            />
            <Select
              allowClear
              showSearch
              value={fieldCandidateKey}
              searchValue={fieldCandidateSearch}
              placeholder="搜索可新增字段/指标"
              filterOption={false}
              style={{ width: 340 }}
              options={fieldCandidateOptions}
              disabled={!selectedFieldTarget}
              onOpenChange={setFieldCandidateOpen}
              onSearch={setFieldCandidateSearch}
              onChange={(value) => {
                setFieldCandidateKey(value);
                setFieldCandidateSearch('');
              }}
              onClear={() => setFieldCandidateSearch('')}
              notFoundContent="无可新增项"
            />
            <Button
              icon={<PlusOutlined />}
              onClick={addFieldConfigRow}
              disabled={!fieldCandidateKey}
            >
              新增
            </Button>
          </Space>
        </div>
        <Table<ReportFieldRow>
          columns={editableFieldConfigColumns}
          dataSource={fieldConfigRows}
          rowKey="key"
          size="small"
          virtual
          loading={pmLoading && selectedFieldTarget?.domain === 'PM'}
          pagination={fieldConfigRows.length > 100 ? {
            pageSize: 50,
            showSizeChanger: true,
            pageSizeOptions: [20, 50, 100],
            showTotal: (total) => `共 ${total} 项`,
          } : false}
          locale={{ emptyText: hasFieldTargets ? '暂无数据' : '暂无字段目标，请先新增对象' }}
          scroll={{ x: 1820, y: 320 }}
        />
      </div>
    );
  },
));

export default function NorthboundPageConfig() {
  const nt = useNorthboundI18n();
  const locale = useNorthboundLocale();
  const [configForm] = Form.useForm();
  const [fileProfiles, setFileProfiles] = useState<ScenarioRow[]>(scenarioRows);
  const [selectedScenario, setSelectedScenario] = useState<ScenarioRow | null>(null);
  const [viewFieldConfigDomain, setViewFieldConfigDomain] = useState<Domain>(defaultFieldTarget?.domain ?? 'CM');
  const [viewFieldConfigTargetKey, setViewFieldConfigTargetKey] = useState(defaultFieldTarget?.key ?? '');
  const [viewFieldTechFilter, setViewFieldTechFilter] = useState<FieldTechFilter>(() => getFieldTechFilterForTarget(defaultFieldTarget));
  const [viewBackendFieldCatalog, setViewBackendFieldCatalog] = useState<Record<string, ReportFieldRow[]>>({});
  const [scenarioEnabled, setScenarioEnabled] = useState<Record<string, boolean>>(
    () => Object.fromEntries(scenarioRows.map((s) => [s.code, s.enabled])),
  );
  const [fileProfileSaving, setFileProfileSaving] = useState<Record<string, boolean>>({});
  const [editorPeriodRows, setEditorPeriodRows] = useState<ScenarioPeriodRow[]>(() => clonePeriodRows(defaultEditorPeriodRows));
  const [editorOpen, setEditorOpen] = useState(false);
  const fieldConfigRef = useRef<FieldConfigSectionHandle>(null);
  const [editorSession, setEditorSession] = useState(0);
  const [initialFieldRows, setInitialFieldRows] = useState<Record<string, ReportFieldRow[]>>({});
  const [editorMode, setEditorMode] = useState<'create' | 'edit'>('create');
  const editorScenarioCodeValue = Form.useWatch('scenarioCode', configForm);
  const [fileDeliveryTargetsByProfile, setFileDeliveryTargetsByProfile] = useState<Record<string, DeliveryTargetRow[]>>({});
  const [selectedInventoryType, setSelectedInventoryType] = useState<InventoryType>('ENB');
  const [viewInventoryType, setViewInventoryType] = useState<InventoryType | null>(null);
  const [inventoryEditorOpen, setInventoryEditorOpen] = useState(false);
  const [inventoryConfigs, setInventoryConfigs] = useState<InventoryConfigRow[]>(initialInventoryConfigs);
  const [inventoryEnabled, setInventoryEnabled] = useState<Record<InventoryType, boolean>>(defaultInventoryEnabled);
  const [inventoryProfileSaving, setInventoryProfileSaving] = useState<Record<string, boolean>>({});
  const [inventoryFieldRowsByType, setInventoryFieldRowsByType] = useState<Record<InventoryType, InventoryField[]>>(defaultInventoryFieldRows);
  const [inventoryFieldCatalogByType, setInventoryFieldCatalogByType] = useState<Partial<Record<InventoryType, InventoryField[]>>>({});
  const [inventoryDeliveryTargetsByType, setInventoryDeliveryTargetsByType] = useState<Record<InventoryType, DeliveryTargetRow[]>>(
    () => Object.fromEntries(inventoryTypeOptions.map((option) => [
      option.value,
      cloneDeliveryTargets(`inventory-${option.value.toLowerCase()}`),
    ])) as Record<InventoryType, DeliveryTargetRow[]>,
  );
  const [inventoryCandidateKey, setInventoryCandidateKey] = useState<string>();
  const [apiEnabled, setApiEnabled] = useState<Record<string, boolean>>(defaultApiEnabled);
  const [apiRows, setApiRows] = useState<NorthboundApiRow[]>(legacySupportedApiRows);
  const [apiUsers, setApiUsers] = useState<ApiUserRow[]>([]);
  const [apiUserSaving, setApiUserSaving] = useState(false);
  const [apiSwitchSaving, setApiSwitchSaving] = useState(false);
  const [apiCatalogOpen, setApiCatalogOpen] = useState(false);
  const [selectedApi, setSelectedApi] = useState<NorthboundApiRow | null>(null);
  const pageConfigLoadingRef = useRef(false);
  const apiUserDirtyRef = useRef(false);
  const apiUserSavingRef = useRef(false);
  const [selectedReportStatus, setSelectedReportStatus] = useState<ReportStatusInfo | null>(null);
  const [reportRunList, setReportRunList] = useState<NorthboundFileRun[]>([]);
  const [reportEventList, setReportEventList] = useState<NorthboundPageConfigEvent[]>([]);
  const [reportEventTotal, setReportEventTotal] = useState(0);
  const [reportEventLoading, setReportEventLoading] = useState(false);
  const [reportCapabilityName, setReportCapabilityName] = useState<string>('');
  const [reportDeliveryEventsByRunId, setReportDeliveryEventsByRunId] = useState<DeliveryEventsByRunId>({});
  const [deliveryTargetValidationErrors, setDeliveryTargetValidationErrors] = useState<DeliveryTargetValidationErrors>({});
  const [deliveryTestResult, setDeliveryTestResult] = useState<DeliveryTestResultInfo | null>(null);
  const [testingDeliveryTargetKey, setTestingDeliveryTargetKey] = useState('');
  const [fileLatestSuccessRuns, setFileLatestSuccessRuns] = useState<Record<string, NorthboundFileRun>>({});
  const [inventoryLatestSuccessRuns, setInventoryLatestSuccessRuns] = useState<Record<string, NorthboundFileRun>>({});
  const [fileProfileRunning, setFileProfileRunning] = useState<Record<string, boolean>>({});
  const [inventoryProfileRunning, setInventoryProfileRunning] = useState<Record<string, boolean>>({});
  const [socketConfigs, setSocketConfigs] = useState<SocketAlarmConfigRow[]>(socketAlarmConfigs);
  const [socketEnabled, setSocketEnabled] = useState<Record<string, boolean>>(defaultSocketEnabled);
  const [selectedSocket, setSelectedSocket] = useState<SocketAlarmConfigRow | null>(null);
  const [socketEditor, setSocketEditor] = useState<SocketAlarmConfigRow | null>(null);
  const [socketEditorEnabled, setSocketEditorEnabled] = useState(false);
  const [socketEditorAccounts, setSocketEditorAccounts] = useState<SocketAccountRow[]>([]);
  const [socketEditorDeliveryTargets, setSocketEditorDeliveryTargets] = useState<DeliveryTargetRow[]>([]);
  const [socketAccountRowsByConfig, setSocketAccountRowsByConfig] = useState<Record<string, SocketAccountRow[]>>(
    () => Object.fromEntries(socketAlarmConfigs.map((config) => [
      config.key,
      cloneDefaultSocketAccounts(config.profile, config.key),
    ])),
  );
  const [socketDeliveryTargetsByConfig, setSocketDeliveryTargetsByConfig] = useState<Record<string, DeliveryTargetRow[]>>(
    () => Object.fromEntries(socketAlarmConfigs.map((config) => [
      config.key,
      cloneDeliveryTargets(`${config.key}-file-sync`),
    ])),
  );
  const [snmpTargets, setSnmpTargets] = useState<SnmpAlarmTargetRow[]>(snmpAlarmTargets);
  const [snmpEnabled, setSnmpEnabled] = useState<Record<string, boolean>>(defaultSnmpEnabled);
  const [snmpSaving, setSnmpSaving] = useState<Record<string, boolean>>({});
  const [selectedSnmp, setSelectedSnmp] = useState<SnmpAlarmTargetRow | null>(null);
  const [snmpEditor, setSnmpEditor] = useState<SnmpAlarmTargetRow | null>(null);
  const [pmMetricRows, setPmMetricRows] = useState<PmMetric[]>([]);
  const [pmLoading, setPmLoading] = useState(true);
  const [pageConfigLoading, setPageConfigLoading] = useState(false);
  const apiManagedKeys = useMemo(() => apiConfigKeys(apiRows), [apiRows]);
  const apiEnabledCount = useMemo(
    () => apiManagedKeys.filter((key) => apiEnabled[key]).length,
    [apiEnabled, apiManagedKeys],
  );
  const apiGloballyEnabled = apiManagedKeys.length > 0 && apiEnabledCount === apiManagedKeys.length;
  const apiPartiallyEnabled = apiEnabledCount > 0 && apiEnabledCount < apiManagedKeys.length;

  const loadPageConfig = useCallback(async (silent = false) => {
    pageConfigLoadingRef.current = true;
    if (!silent) setPageConfigLoading(true);
    try {
      const [
        fileProfileResp,
        inventoryProfileResp,
        fileDeliveryResp,
        inventoryDeliveryResp,
        socketDeliveryResp,
        snmpResp,
        socketResp,
        apiResp,
        apiUserResp,
        fileLatestRunResp,
        inventoryLatestRunResp,
      ] = await Promise.all([
        northboundPageConfigApi.getFileProfiles(),
        northboundPageConfigApi.getInventoryProfiles(),
        northboundPageConfigApi.getDeliveryTargets({ scope: 'file' }),
        northboundPageConfigApi.getDeliveryTargets({ scope: 'inventory' }),
        northboundPageConfigApi.getDeliveryTargets({ scope: 'socket' }),
        northboundPageConfigApi.getSNMPAlarmTargets(),
        northboundPageConfigApi.getSocketAlarmConfigs(),
        northboundPageConfigApi.getAPIConfigs(),
        northboundPageConfigApi.getAPIUsers(),
        northboundPageConfigApi.listRuns({
          profile_kind: 'file',
          status: 'success',
          latest_per_profile: true,
          limit: 1000,
        }),
        northboundPageConfigApi.listRuns({
          profile_kind: 'inventory',
          status: 'success',
          latest_per_profile: true,
          limit: 1000,
        }),
      ]);
      const nextFileProfiles = fileProfileResp.items.map(mapApiFileProfile);
      const nextInventoryProfiles = inventoryProfileResp.items
        .map(mapApiInventoryProfile)
        .sort((a, b) => (
          inventoryTypeOptions.findIndex((option) => option.value === a.key)
          - inventoryTypeOptions.findIndex((option) => option.value === b.key)
        ));

      setFileProfiles(nextFileProfiles);
      setScenarioEnabled(scenarioEnabledMap(nextFileProfiles));
      setFileLatestSuccessRuns(latestSuccessRunMap(fileLatestRunResp.items));
      setSelectedScenario((current) => (
        current ? nextFileProfiles.find((row) => row.code === current.code) ?? current : current
      ));

      setInventoryConfigs(nextInventoryProfiles);
      setInventoryEnabled(inventoryEnabledMap(nextInventoryProfiles, inventoryProfileResp.items));
      setInventoryLatestSuccessRuns(latestSuccessRunMap(inventoryLatestRunResp.items));
      const nextInventoryFields = { ...defaultInventoryFieldRows };
      inventoryProfileResp.items.forEach((profile) => {
        const key = normalizeInventoryType(profile.code || profile.object_code);
        if (profile.fields) {
          nextInventoryFields[key] = profile.fields.map((field) => mapApiInventoryField(field, key));
        }
      });
      setInventoryFieldRowsByType(nextInventoryFields);
      setSelectedInventoryType((current) => (
        nextInventoryProfiles.some((row) => row.key === current)
          ? current
          : nextInventoryProfiles[0]?.key ?? current
      ));
      setViewInventoryType((current) => (
        current && nextInventoryProfiles.some((row) => row.key === current) ? current : null
      ));

      const nextFileTargets = fileDeliveryResp.items.reduce<Record<string, DeliveryTargetRow[]>>((acc, item) => {
        const ownerCode = normalizeFileDeliveryOwnerCode(item.owner_code);
        acc[ownerCode] = [...(acc[ownerCode] ?? []), mapApiDeliveryTarget(item)];
        return acc;
      }, {});
      setFileDeliveryTargetsByProfile(nextFileTargets);
      const nextInventoryTargets = inventoryDeliveryResp.items.reduce<Record<InventoryType, DeliveryTargetRow[]>>((acc, item) => {
        const key = normalizeInventoryType(item.owner_code);
        acc[key] = [...(acc[key] ?? []), mapApiDeliveryTarget(item)];
        return acc;
      }, {} as Record<InventoryType, DeliveryTargetRow[]>);
      if (Object.keys(nextInventoryTargets).length > 0) {
        setInventoryDeliveryTargetsByType((prev) => ({ ...prev, ...nextInventoryTargets }));
      }
      const nextSocketTargets = socketDeliveryResp.items.reduce<Record<string, DeliveryTargetRow[]>>((acc, item) => {
        acc[item.owner_code] = [...(acc[item.owner_code] ?? []), mapApiDeliveryTarget(item)];
        return acc;
      }, {});
      if (Object.keys(nextSocketTargets).length > 0) {
        setSocketDeliveryTargetsByConfig((prev) => ({ ...prev, ...nextSocketTargets }));
      }

      const nextSnmpTargets = snmpResp.items.map(mapApiSnmpTarget);
      if (nextSnmpTargets.length > 0) {
        setSnmpTargets(nextSnmpTargets);
        setSnmpEnabled(Object.fromEntries(snmpResp.items.map((row) => [row.key, row.enabled])));
      }

      const nextSocketConfigs = socketResp.items.map(mapApiSocketConfig);
      if (nextSocketConfigs.length > 0) {
        setSocketConfigs(nextSocketConfigs);
        setSocketEnabled(Object.fromEntries(socketResp.items.map((row) => [row.key, row.enabled])));
        setSocketAccountRowsByConfig((prev) => ({
          ...prev,
          ...Object.fromEntries(socketResp.items.map((row) => [row.key, mapApiSocketAccounts(row)])),
        }));
      }

      const nextApiRows = expandApiDisplayRows(apiResp.items.map(mapApiConfig));
      if (nextApiRows.length > 0) {
        setApiRows(nextApiRows);
        setApiEnabled(Object.fromEntries(apiResp.items.map((row) => [row.key, row.enabled])));
      }
      if (!apiUserDirtyRef.current && !apiUserSavingRef.current) {
        setApiUsers(apiUserResp.items.map(mapApiUser));
      }

      if (!silent) void message.success(nt('北向页面配置已刷新'));
    } catch {
      if (!silent) {
        void message.error(nt('北向页面配置加载失败，已保留当前页面数据'));
      }
    } finally {
      pageConfigLoadingRef.current = false;
      if (!silent) setPageConfigLoading(false);
    }
  }, []);

  useEffect(() => {
    let cancelled = false;
    setPmLoading(true);
    loadPmMetrics()
      .then((metrics) => {
        if (!cancelled) setPmMetricRows(metrics);
      })
      .catch(() => {
        if (!cancelled) void message.error(nt('指标目录加载失败'));
      })
      .finally(() => {
        if (!cancelled) setPmLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    void loadPageConfig(true);
  }, [loadPageConfig]);

  useEffect(() => {
    const timer = window.setInterval(() => {
      if (document.visibilityState === 'hidden') return;
      if (pageConfigLoadingRef.current || apiUserDirtyRef.current || apiUserSavingRef.current) return;
      if (editorOpen || inventoryEditorOpen || socketEditor) return;
      void loadPageConfig(true);
    }, 30000);
    return () => window.clearInterval(timer);
  }, [editorOpen, inventoryEditorOpen, loadPageConfig, socketEditor]);

  const selectedInventoryConfig = useMemo(
    () => inventoryConfigs.find((row) => row.key === selectedInventoryType) ?? inventoryConfigs[0],
    [inventoryConfigs, selectedInventoryType],
  );
  const viewInventoryConfig = useMemo(
    () => viewInventoryType ? inventoryConfigs.find((row) => row.key === viewInventoryType) ?? null : null,
    [inventoryConfigs, viewInventoryType],
  );
  const currentInventoryFields = inventoryFieldRowsByType[selectedInventoryType] ?? [];
  const viewInventoryFields = viewInventoryType ? inventoryFieldRowsByType[viewInventoryType] ?? [] : [];
  useEffect(() => {
    if (!inventoryEditorOpen || !selectedInventoryConfig || selectedInventoryConfig.key === 'OMC') return;
    if (inventoryFieldCatalogByType[selectedInventoryConfig.key]) return;
    let cancelled = false;
    void northboundPageConfigApi.getFields({
      domain: 'INVENTORY',
      object: selectedInventoryConfig.objectCode,
    })
      .then((resp) => {
        if (cancelled) return;
        const rows = (resp.items ?? []).map((field) => mapApiInventoryCandidateField(field, selectedInventoryConfig.key));
        setInventoryFieldCatalogByType((prev) => (
          prev[selectedInventoryConfig.key] ? prev : { ...prev, [selectedInventoryConfig.key]: rows }
        ));
      })
      .catch(() => { /* local fallback stays available */ });
    return () => { cancelled = true; };
  }, [inventoryEditorOpen, inventoryFieldCatalogByType, selectedInventoryConfig]);
  const inventoryCandidateRows = useMemo(() => {
    const selectedIds = new Set(currentInventoryFields.map((field) => field.exportKey.toLowerCase()));
    const candidatePool = dedupeInventoryFields([
      ...(inventoryFieldCatalogByType[selectedInventoryType] ?? []),
      ...inventoryAvailableFields.filter((field) => field.template === selectedInventoryType),
    ]);
    return candidatePool.filter((field) => !selectedIds.has(field.exportKey.toLowerCase()));
  }, [currentInventoryFields, inventoryFieldCatalogByType, selectedInventoryType]);
  const inventoryCandidateOptions = useMemo(
    () => inventoryCandidateRows.map((field) => ({
      label: `${field.column} / ${field.exportKey}`,
      value: field.key,
    })),
    [inventoryCandidateRows],
  );

  const getInventoryDeliveryTargets = (key: InventoryType) => (
    inventoryDeliveryTargetsByType[key] ?? cloneDeliveryTargets(`inventory-${key.toLowerCase()}`)
  );

  const getFileDeliveryTargets = (ownerCode?: string) => {
    const key = normalizeFileDeliveryOwnerCode(ownerCode);
    if (!key) return cloneDeliveryTargets(fileDeliveryScopeKey(key));
    return fileDeliveryTargetsByProfile[key] ?? cloneDeliveryTargets(fileDeliveryScopeKey(key));
  };

  const setFileDeliveryTargetsForOwner = (
    ownerCode: string,
    updater: (rows: DeliveryTargetRow[]) => DeliveryTargetRow[],
  ) => {
    const key = normalizeFileDeliveryOwnerCode(ownerCode);
    setFileDeliveryTargetsByProfile((prev) => {
      const currentRows = prev[key] ?? cloneDeliveryTargets(fileDeliveryScopeKey(key));
      return { ...prev, [key]: updater(currentRows) };
    });
  };

  const getSocketDeliveryTargets = (config: SocketAlarmConfigRow) => (
    socketDeliveryTargetsByConfig[config.key] ?? cloneDeliveryTargets(`${config.key}-file-sync`)
  );

  const enabledDeliverySummary = (rows: DeliveryTargetRow[]) => {
    const enabledRows = rows.filter((row) => row.enabled);
    if (enabledRows.length === 0) return '未启用传输目标';
    return enabledRows.map((row) => `${row.name}(${row.protocol})`).join('、');
  };

  const setDeliveryTargetValidationResult = (
    scope: DeliveryTargetScope,
    ownerCode: string,
    errors: DeliveryTargetValidationErrors,
  ) => {
    setDeliveryTargetValidationErrors((prev) => replaceDeliveryTargetValidationErrors(prev, scope, ownerCode, errors));
  };

  const validateDeliveryTargetsBeforeSave = (
    scope: DeliveryTargetScope,
    ownerCode: string,
    rows: DeliveryTargetRow[],
  ) => {
    const result = validateDeliveryTargetRows(scope, ownerCode, rows);
    setDeliveryTargetValidationResult(scope, ownerCode, result.errors);
    if (!result.valid) {
      void message.warning(nt(`传输目标填写不完整，请补充或删除空行：${result.message}`));
    }
    return result.valid;
  };

  const clearDeliveryTargetValidationFields = (
    scope: DeliveryTargetScope,
    ownerCode: string,
    key: string,
    fields?: DeliveryTargetValidationField[],
  ) => {
    const validationKey = deliveryTargetTestKey(scope, ownerCode, key);
    setDeliveryTargetValidationErrors((prev) => {
      const current = prev[validationKey];
      if (!current) return prev;
      const next = { ...prev };
      if (!fields || fields.length === 0) {
        delete next[validationKey];
        return next;
      }
      const nextRowErrors = { ...current };
      fields.forEach((field) => {
        delete nextRowErrors[field];
      });
      if (Object.keys(nextRowErrors).length === 0) delete next[validationKey];
      else next[validationKey] = nextRowErrors;
      return next;
    });
  };

  const deliveryTargetValidationSummary = (scope: DeliveryTargetScope, ownerCode: string) => {
    if (!ownerCode) return '';
    const prefix = deliveryTargetScopePrefix(scope, ownerCode);
    return Object.keys(deliveryTargetValidationErrors).some((key) => key.startsWith(prefix))
      ? '传输目标填写不完整，请补充必填项或删除空行'
      : '';
  };

  const renderDeliveryTargetValidationSummary = (scope: DeliveryTargetScope, ownerCode: string) => {
    const summary = deliveryTargetValidationSummary(scope, ownerCode);
    return summary ? (
      <Typography.Text type="danger" className={styles.deliveryTargetValidationSummary}>
        {nt(summary)}
      </Typography.Text>
    ) : null;
  };

  const renderDeliveryTargetEditorField = (node: ReactNode, error?: string) => (
    <div className={styles.deliveryTargetEditorField}>
      {node}
      {error ? (
        <Typography.Text type="danger" className={styles.deliveryTargetFieldError}>
          {nt(error)}
        </Typography.Text>
      ) : null}
    </div>
  );

  const updateFileDeliveryTarget = (ownerCode: string, key: string, patch: Partial<DeliveryTargetRow>) => {
    clearDeliveryTargetValidationFields('file', ownerCode, key, deliveryTargetValidationFieldsForPatch(patch));
    setFileDeliveryTargetsForOwner(ownerCode, (rows) => rows.map((row) => (row.key === key ? { ...row, ...patch } : row)));
  };

  const addFileDeliveryTarget = (ownerCode: string) => {
    setFileDeliveryTargetsForOwner(ownerCode, (rows) => [
      ...rows,
      createDeliveryTarget(fileDeliveryScopeKey(ownerCode), rows.length + 1),
    ]);
  };

  const removeFileDeliveryTarget = (ownerCode: string, key: string) => {
    clearDeliveryTargetValidationFields('file', ownerCode, key);
    setFileDeliveryTargetsForOwner(ownerCode, (rows) => rows.filter((row) => row.key !== key));
  };

  const updateInventoryDeliveryTarget = (inventoryType: InventoryType, key: string, patch: Partial<DeliveryTargetRow>) => {
    clearDeliveryTargetValidationFields('inventory', inventoryType, key, deliveryTargetValidationFieldsForPatch(patch));
    setInventoryDeliveryTargetsByType((prev) => ({
      ...prev,
      [inventoryType]: (prev[inventoryType] ?? cloneDeliveryTargets(`inventory-${inventoryType.toLowerCase()}`)).map((row) => (
        row.key === key ? { ...row, ...patch } : row
      )),
    }));
  };

  const addInventoryDeliveryTarget = (inventoryType: InventoryType) => {
    setInventoryDeliveryTargetsByType((prev) => {
      const rows = prev[inventoryType] ?? cloneDeliveryTargets(`inventory-${inventoryType.toLowerCase()}`);
      return {
        ...prev,
        [inventoryType]: [...rows, createDeliveryTarget(`inventory-${inventoryType.toLowerCase()}`, rows.length + 1)],
      };
    });
  };

  const removeInventoryDeliveryTarget = (inventoryType: InventoryType, key: string) => {
    clearDeliveryTargetValidationFields('inventory', inventoryType, key);
    setInventoryDeliveryTargetsByType((prev) => ({
      ...prev,
      [inventoryType]: (prev[inventoryType] ?? cloneDeliveryTargets(`inventory-${inventoryType.toLowerCase()}`)).filter((row) => row.key !== key),
    }));
  };

  const updateSocketEditorDeliveryTarget = (key: string, patch: Partial<DeliveryTargetRow>) => {
    if (socketEditor) {
      clearDeliveryTargetValidationFields('socket', socketEditor.key, key, deliveryTargetValidationFieldsForPatch(patch));
    }
    setSocketEditorDeliveryTargets((rows) => rows.map((row) => (
      row.key === key ? { ...row, ...patch } : row
    )));
  };

  const addSocketEditorDeliveryTarget = (configKey: string) => {
    setSocketEditorDeliveryTargets((rows) => [
      ...rows,
      createDeliveryTarget(`${configKey}-file-sync`, rows.length + 1),
    ]);
  };

  const removeSocketEditorDeliveryTarget = (key: string) => {
    if (socketEditor) clearDeliveryTargetValidationFields('socket', socketEditor.key, key);
    setSocketEditorDeliveryTargets((rows) => rows.filter((row) => row.key !== key));
  };

  const testDeliveryTarget = (
    row: DeliveryTargetRow,
    scope: DeliveryTargetScope,
    ownerCode: string,
  ) => {
    const missingFields: string[] = [];
    if (!row.host.trim()) missingFields.push('主机地址');
    if (!row.username.trim()) missingFields.push('用户名');
    if (!deliveryTargetHasCredential(row)) missingFields.push('密码');
    if (missingFields.length > 0) {
      void message.warning(nt(`请先填写${missingFields.join('、')}`));
      return;
    }
    const testKey = deliveryTargetTestKey(scope, ownerCode, row.key);
    const testTitle = deliveryTargetTestTitle(row);
    setTestingDeliveryTargetKey(testKey);
    void northboundPageConfigApi.testDeliveryTarget(serializeDeliveryTarget(scope, ownerCode, row))
      .then((event) => {
        setDeliveryTestResult({ event, title: testTitle });
        if (event.status === 'success') {
          void message.success(nt(`${testTitle} 连接测试通过`));
        } else {
          void message.warning(nt(`${testTitle} 连接测试未通过`));
        }
      })
      .catch(() => {
        void message.error(nt(`${testTitle} 连接测试失败`));
      })
      .finally(() => {
        setTestingDeliveryTargetKey((current) => (current === testKey ? '' : current));
      });
  };

  const getSocketAccounts = (config: SocketAlarmConfigRow) => (
    socketAccountRowsByConfig[config.key] ?? cloneDefaultSocketAccounts(config.profile, config.key)
  );

  const openSocketEditor = (config: SocketAlarmConfigRow) => {
    setSocketEditor(config);
    setSocketEditorEnabled(Boolean(socketEnabled[config.key]));
    setSocketEditorAccounts(getSocketAccounts(config).map((account) => ({ ...account })));
    setSocketEditorDeliveryTargets(getSocketDeliveryTargets(config).map((target) => ({ ...target })));
  };

  const closeSocketEditor = () => {
    setSocketEditor(null);
    setSocketEditorEnabled(false);
    setSocketEditorAccounts([]);
    setSocketEditorDeliveryTargets([]);
  };

  const addSocketEditorAccount = (config: SocketAlarmConfigRow) => {
    setSocketEditorAccounts((currentRows) => {
      const hasFtpAccount = currentRows.some((account) => account.type === 'ftp');
      const nextType: SocketAccountRow['type'] = config.profile === 'CUCC' && !hasFtpAccount ? 'ftp' : 'msg';
      const nextIndex = currentRows.length + 1;
      return [
        ...currentRows,
        {
          key: `${config.key}-account-${Date.now()}`,
          channel: nextType === 'ftp' ? '文件同步账号' : `实时/消息同步账号 ${nextIndex}`,
          username: '',
          type: nextType,
          credential: '',
          enabled: true,
          purpose: getSocketAccountPurpose(nextType),
        },
      ];
    });
  };

  const updateSocketEditorAccount = (accountKey: string, patch: Partial<SocketAccountRow>) => {
    setSocketEditorAccounts((rows) => rows.map((account) => (
      account.key === accountKey ? { ...account, ...patch } : account
    )));
  };

  const removeSocketEditorAccount = (accountKey: string) => {
    setSocketEditorAccounts((rows) => rows.filter((account) => account.key !== accountKey));
  };

  const updateInventoryConfig = (key: InventoryType, patch: Partial<InventoryConfigRow>) => {
    setInventoryConfigs((rows) => rows.map((row) => (row.key === key ? { ...row, ...patch } : row)));
  };

  const updateInventoryFieldAlias = (fieldKey: string, column: string) => {
    setInventoryFieldRowsByType((prev) => ({
      ...prev,
      [selectedInventoryType]: (prev[selectedInventoryType] ?? []).map((field) => (
        field.key === fieldKey ? { ...field, column } : field
      )),
    }));
  };

  const updateInventoryFieldEnabled = (fieldKey: string, enabled: boolean) => {
    setInventoryFieldRowsByType((prev) => ({
      ...prev,
      [selectedInventoryType]: (prev[selectedInventoryType] ?? []).map((field) => (
        field.key === fieldKey ? { ...field, enabled } : field
      )),
    }));
  };

  const removeInventoryField = (fieldKey: string) => {
    setInventoryFieldRowsByType((prev) => ({
      ...prev,
      [selectedInventoryType]: (prev[selectedInventoryType] ?? []).filter((field) => field.key !== fieldKey),
    }));
  };

  const addInventoryField = () => {
    const field = inventoryCandidateRows.find((candidate) => candidate.key === inventoryCandidateKey);
    if (!field) return;
    setInventoryFieldRowsByType((prev) => ({
      ...prev,
      [selectedInventoryType]: [...(prev[selectedInventoryType] ?? []), { ...field, enabled: true }],
    }));
    setInventoryCandidateKey(undefined);
  };

  const openInventoryView = (row: InventoryConfigRow) => {
    setSelectedInventoryType(row.key);
    setViewInventoryType(row.key);
  };

  const openInventoryEditor = (row: InventoryConfigRow) => {
    setSelectedInventoryType(row.key);
    setInventoryCandidateKey(undefined);
    setInventoryEditorOpen(true);
  };

  const applyFileProfile = (profile: NorthboundFileProfile) => {
    const nextProfile = mapApiFileProfile(profile);
    setFileProfiles((rows) => (
      rows.some((row) => row.code === nextProfile.code)
        ? rows.map((row) => (row.code === nextProfile.code ? nextProfile : row))
        : [nextProfile, ...rows]
    ));
    setScenarioEnabled((prev) => ({ ...prev, [nextProfile.code]: nextProfile.enabled }));
    setSelectedScenario((current) => (current?.code === nextProfile.code ? nextProfile : current));
  };

  const applyInventoryProfile = (profile: NorthboundInventoryProfile) => {
    const nextProfile = mapApiInventoryProfile(profile);
    setInventoryConfigs((rows) => (
      rows.map((row) => (row.key === nextProfile.key ? nextProfile : row))
    ));
    setInventoryEnabled((prev) => ({ ...prev, [nextProfile.key]: profile.enabled }));
    if (profile.fields) {
      setInventoryFieldRowsByType((prev) => ({
        ...prev,
        [nextProfile.key]: profile.fields!.map((field) => mapApiInventoryField(field, nextProfile.key)),
      }));
    }
  };

  const setFileSaving = (code: string, saving: boolean) => {
    setFileProfileSaving((prev) => {
      const next = { ...prev };
      if (saving) next[code] = true;
      else delete next[code];
      return next;
    });
  };

  const setInventorySaving = (code: string, saving: boolean) => {
    setInventoryProfileSaving((prev) => {
      const next = { ...prev };
      if (saving) next[code] = true;
      else delete next[code];
      return next;
    });
  };

  const setSnmpTargetSaving = (key: string, saving: boolean) => {
    setSnmpSaving((prev) => {
      const next = { ...prev };
      if (saving) next[key] = true;
      else delete next[key];
      return next;
    });
  };

  const saveInventoryDraft = () => {
    if (!selectedInventoryConfig) return;
    const enabled = Boolean(inventoryEnabled[selectedInventoryConfig.key]);
    const fields = inventoryFieldRowsByType[selectedInventoryConfig.key] ?? [];
    const deliveryRows = getInventoryDeliveryTargets(selectedInventoryConfig.key);
    if (!validateDeliveryTargetsBeforeSave('inventory', selectedInventoryConfig.key, deliveryRows)) return;
    setInventorySaving(selectedInventoryConfig.key, true);
    void Promise.all([
      northboundPageConfigApi.updateInventoryProfile(
        selectedInventoryConfig.key,
        serializeInventoryConfig(selectedInventoryConfig, enabled, fields),
      ),
      northboundPageConfigApi.replaceDeliveryTargets(
        serializeDeliveryTargets(
          'inventory',
          selectedInventoryConfig.key,
          deliveryRows,
        ),
      ),
    ])
      .then(([profile]) => {
        applyInventoryProfile(profile);
        void message.success(nt(`${profile.object_code} Inventory 配置已保存`));
        setInventoryEditorOpen(false);
      })
      .catch(() => {
        void message.error(nt(`${selectedInventoryConfig.objectCode} Inventory 配置保存失败`));
      })
      .finally(() => setInventorySaving(selectedInventoryConfig.key, false));
  };

  const getInventoryFieldTooltip = (key: InventoryType) => {
    const fields = inventoryFieldRowsByType[key] ?? [];
    return fields.map((field) => `${field.column} / ${field.exportKey}`).join('\n') || '未配置字段';
  };

  const getInventoryScheduleTooltip = (row: InventoryConfigRow) => [
    `统计周期：${formatPeriodLabel(row.period)}`,
    `生成计划：${formatScheduleLabel(row.period, row.cron)}`,
  ].join('\n');

  const getInventoryFileTooltip = (row: InventoryConfigRow) => {
    const preview = getInventoryTemplatePreview(row);
    return [
      `上传目录模板：${row.path}`,
      `文件名模板：${row.fileName}`,
      `预览目录：${preview.path}`,
      `预览文件：${preview.fileName}`,
    ].join('\n');
  };

  const viewPeriodRows = useMemo(
    () => selectedScenario ? getScenarioPeriodRows(selectedScenario) : [],
    [selectedScenario],
  );
  const viewFieldTargets = useMemo(() => getFieldTargets(viewPeriodRows), [viewPeriodRows]);
  const viewFieldDomainOptions = useMemo(
    () => [...new Set(viewFieldTargets.map((target) => target.domain))]
      .map((domain) => ({ label: domain, value: domain })),
    [viewFieldTargets],
  );
  const activeViewFieldTechFilter = useMemo(
    () => getFieldTechFilterForDomain(viewFieldConfigDomain, viewFieldTargets, viewFieldTechFilter),
    [viewFieldConfigDomain, viewFieldTargets, viewFieldTechFilter],
  );
  const viewFieldTechFilterOptions = useMemo(
    () => getFieldTechFilterOptions(viewFieldConfigDomain, viewFieldTargets),
    [viewFieldConfigDomain, viewFieldTargets],
  );
  const visibleViewFieldTargets = useMemo(
    () => viewFieldTargets
      .filter((target) => target.domain === viewFieldConfigDomain)
      .filter((target) => targetMatchesTechFilter(target, activeViewFieldTechFilter)),
    [viewFieldConfigDomain, viewFieldTargets, activeViewFieldTechFilter],
  );
  const viewFieldTargetOptions = useMemo(
    () => visibleViewFieldTargets.map((target) => ({
      label: formatFieldTargetLabel(target),
      value: target.key,
    })),
    [visibleViewFieldTargets],
  );
  const selectedViewFieldTarget = useMemo(
    () => visibleViewFieldTargets.find((target) => target.key === viewFieldConfigTargetKey) ?? visibleViewFieldTargets[0],
    [viewFieldConfigTargetKey, visibleViewFieldTargets],
  );
  const viewBackendCatalogKey = selectedViewFieldTarget
    ? `${selectedViewFieldTarget.domain}:${selectedViewFieldTarget.objectCode}:${getTargetTech(selectedViewFieldTarget) ?? 'ALL'}:${selectedViewFieldTarget.profile}`
    : '';
  useEffect(() => {
    const target = selectedViewFieldTarget;
    if (!target?.profile || !viewBackendCatalogKey || viewBackendFieldCatalog[viewBackendCatalogKey]) return;
    const tech = getTargetTech(target);
    let cancelled = false;
    void northboundPageConfigApi.getFields({
      domain: target.domain,
      object: target.objectCode,
      ...(tech ? { tech } : {}),
      profile: target.profile,
    })
      .then((resp) => {
        if (cancelled) return;
        const rows = (resp.items ?? []).map((field, index) => mapApiFieldToReportRow(field, target, index));
        setViewBackendFieldCatalog((prev) => (prev[viewBackendCatalogKey] ? prev : { ...prev, [viewBackendCatalogKey]: rows }));
      })
      .catch(() => { /* fall back to static catalog */ });
    return () => { cancelled = true; };
  }, [selectedViewFieldTarget, viewBackendCatalogKey, viewBackendFieldCatalog]);
  const viewBackendRowsForSelectedTarget = viewBackendCatalogKey ? viewBackendFieldCatalog[viewBackendCatalogKey] : undefined;
  const viewFieldRows = useMemo(
    () => {
      const rows = selectedViewFieldTarget?.profile && viewBackendRowsForSelectedTarget && viewBackendRowsForSelectedTarget.length > 0
        ? viewBackendRowsForSelectedTarget
        : getReportFieldRows(selectedViewFieldTarget, pmMetricRows);
      return rows.filter((row) => rowMatchesTechFilter(row, activeViewFieldTechFilter));
    },
    [activeViewFieldTechFilter, pmMetricRows, selectedViewFieldTarget, viewBackendRowsForSelectedTarget],
  );
  const viewFileDeliveryTargets = selectedScenario ? getFileDeliveryTargets(selectedScenario.code) : [];
  const editorFileOwnerCode = normalizeFileDeliveryOwnerCode(
    typeof editorScenarioCodeValue === 'string'
      ? editorScenarioCodeValue
      : editorMode === 'edit'
        ? selectedScenario?.code
        : undefined,
  );
  const editorFileDeliveryTargets = getFileDeliveryTargets(editorFileOwnerCode);

  useEffect(() => {
    setInventoryCandidateKey(undefined);
  }, [selectedInventoryType]);

  useEffect(() => {
    if (!selectedScenario || viewFieldTargets.length === 0) return;
    const domainHasTargets = viewFieldTargets.some((target) => target.domain === viewFieldConfigDomain);
    if (!domainHasTargets) {
      const [nextTarget] = viewFieldTargets;
      setViewFieldConfigDomain(nextTarget.domain);
      setViewFieldTechFilter(getFieldTechFilterForTarget(nextTarget));
      setViewFieldConfigTargetKey(nextTarget.key);
      return;
    }
    const nextFilter = getFieldTechFilterForDomain(viewFieldConfigDomain, viewFieldTargets, viewFieldTechFilter);
    if (nextFilter !== viewFieldTechFilter) {
      setViewFieldTechFilter(nextFilter);
    }
    const nextVisibleTargets = viewFieldTargets
      .filter((target) => target.domain === viewFieldConfigDomain)
      .filter((target) => targetMatchesTechFilter(target, nextFilter));
    if (nextVisibleTargets.length === 0) {
      if (viewFieldConfigTargetKey) setViewFieldConfigTargetKey('');
      return;
    }
    const currentTarget = nextVisibleTargets.find((target) => target.key === viewFieldConfigTargetKey);
    if (currentTarget) return;
    setViewFieldConfigTargetKey(nextVisibleTargets[0].key);
  }, [selectedScenario, viewFieldConfigDomain, viewFieldConfigTargetKey, viewFieldTargets, viewFieldTechFilter]);

  const openViewDrawer = (row: ScenarioRow) => {
    const periodRows = getScenarioPeriodRows(row);
    const firstTarget = getFirstFieldTarget(periodRows);
    setSelectedScenario(row);
    setViewFieldTechFilter(getFieldTechFilterForTarget(firstTarget));
    if (firstTarget) {
      setViewFieldConfigDomain(firstTarget.domain);
      setViewFieldConfigTargetKey(firstTarget.key);
    }
  };

  const changeViewFieldConfigDomain = (domain: Domain) => {
    const nextFilter = getFieldTechFilterForDomain(domain, viewFieldTargets);
    setViewFieldConfigDomain(domain);
    setViewFieldTechFilter(nextFilter);
    const nextTarget = viewFieldTargets.find((target) => target.domain === domain && targetMatchesTechFilter(target, nextFilter))
      ?? viewFieldTargets.find((target) => target.domain === domain);
    setViewFieldConfigTargetKey(nextTarget?.key ?? '');
  };

  const changeViewFieldTechFilter = (techFilter: FieldTechFilter) => {
    const nextFilter = getFieldTechFilterForDomain(viewFieldConfigDomain, viewFieldTargets, techFilter);
    setViewFieldTechFilter(nextFilter);
    const nextTarget = viewFieldTargets.find((target) => target.domain === viewFieldConfigDomain && targetMatchesTechFilter(target, nextFilter));
    setViewFieldConfigTargetKey(nextTarget?.key ?? '');
  };

  const openCreateEditor = () => {
    setEditorMode('create');
    setEditorPeriodRows([]);
    setInitialFieldRows({});
    setEditorSession((n) => n + 1);
    const nextCode = getNextCustomScenarioCode(fileProfiles);
    configForm.setFieldsValue({
      scenarioCode: nextCode,
      vendor: '',
      scenarioName: '',
      scenarioNameEn: '',
      name: '',
      description: '',
      domains: [],
      periods: [],
      enabled: false,
    });
    setEditorOpen(true);
  };

  const openEditEditor = (row: ScenarioRow) => {
    const periodRows = clonePeriodRows(getScenarioPeriodRows(row));
    setEditorMode('edit');
    setEditorPeriodRows(periodRows);
    setInitialFieldRows(hydrateFieldRowsByTarget(periodRows, pmMetricRows));
    setEditorSession((n) => n + 1);
    configForm.setFieldsValue({
      scenarioCode: row.code,
      vendor: row.vendor,
      scenarioName: row.scenarioName,
      scenarioNameEn: row.scenarioNameEn,
      name: row.name,
      description: row.description,
      domains: [...new Set(row.groups.map((g) => g.domain))],
      periods: [...new Set(row.groups.map((g) => g.period))],
      enabled: scenarioEnabled[row.code],
    });
    setEditorOpen(true);
  };

  const saveEditorDraft = () => {
    void configForm.validateFields().then((values: FileProfileEditorValues) => {
      const code = values.scenarioCode.trim().toUpperCase();
      if (!code) {
        void message.warning(nt('请输入配置编号'));
        return;
      }
      if (!/^S\d{4}$/.test(code)) {
        void message.warning(nt('配置编号必须使用 S0000 格式'));
        return;
      }
      const enabled = Boolean(values.enabled);
      const vendor = values.vendor?.trim() || selectedScenario?.vendor || 'Baicells';
      const fieldRowsByTarget = fieldConfigRef.current?.getFieldRowsByTarget() ?? {};
      const periodValidationError = validateEditorPeriodRows(editorPeriodRows);
      if (periodValidationError) {
        void message.warning(nt(periodValidationError));
        return;
      }
      const deliveryRows = getFileDeliveryTargets(code);
      if (!validateDeliveryTargetsBeforeSave('file', code, deliveryRows)) return;
      const groups = serializeEditorPeriodRows(editorPeriodRows, fieldRowsByTarget);
      const request: NorthboundUpdateFileProfileRequest = {
        name: values.name.trim(),
        vendor,
        scenario_name: values.scenarioName.trim(),
        scenario_name_en: values.scenarioNameEn?.trim() || values.scenarioName.trim(),
        description: values.description?.trim() ?? '',
        flags: editorMode === 'create' ? ['custom'] : selectedScenario?.flags ?? [],
        enabled,
        status: enabled ? 'normal' : 'terminated',
        groups,
      };

      setFileSaving(code, true);
      if (editorMode === 'create') {
        const createRequest: NorthboundFileProfile = {
          id: code,
          code,
          name: request.name || code,
          vendor,
          scenario_name: request.scenario_name || request.name || code,
          scenario_name_en: request.scenario_name_en || request.scenario_name || request.name || code,
          description: request.description ?? '',
          flags: request.flags ?? ['custom'],
          enabled,
          status: enabled ? 'normal' : 'terminated',
          groups,
        };
        void Promise.all([
          northboundPageConfigApi.createFileProfile(createRequest),
          northboundPageConfigApi.replaceDeliveryTargets(
            serializeDeliveryTargets('file', code, deliveryRows),
          ),
        ])
          .then(([profile, deliveryResp]) => {
            applyFileProfile(profile);
            setSelectedScenario(mapApiFileProfile(profile));
            setFileDeliveryTargetsByProfile((prev) => ({
              ...prev,
              [normalizeFileDeliveryOwnerCode(profile.code || code)]: deliveryResp.items.map(mapApiDeliveryTarget),
            }));
            void message.success(nt(`新增配置已保存：${profile.name}`));
            setEditorOpen(false);
          })
          .catch(() => {
            void message.error(nt(`配置保存失败：${code}`));
          })
          .finally(() => setFileSaving(code, false));
        return;
      }

      void Promise.all([
        northboundPageConfigApi.updateFileProfile(code, request),
        northboundPageConfigApi.replaceDeliveryTargets(
          serializeDeliveryTargets('file', code, deliveryRows),
        ),
      ])
        .then(([profile, deliveryResp]) => {
          applyFileProfile(profile);
          setFileDeliveryTargetsByProfile((prev) => ({
            ...prev,
            [normalizeFileDeliveryOwnerCode(profile.code || code)]: deliveryResp.items.map(mapApiDeliveryTarget),
          }));
          void message.success(nt(`编辑配置已保存：${profile.name}`));
          setEditorOpen(false);
        })
        .catch(() => {
          void message.error(nt(`配置保存失败：${code}`));
        })
        .finally(() => setFileSaving(code, false));
    });
  };

  const updateEditorPeriodRow = useCallback((key: string, patch: Partial<ScenarioPeriodRow>) => {
    setEditorPeriodRows((rows) => rows.map((row) => {
      if (row.key !== key) return row;
      const nextRow = { ...row, ...patch };
      const hasFormatPatch = Object.prototype.hasOwnProperty.call(patch, 'format');
      const hasSelectionPatch = Object.prototype.hasOwnProperty.call(patch, 'domain') || Object.prototype.hasOwnProperty.call(patch, 'objects');
      const oldDefaultFormat = defaultFormatForSelection(row.domain, row.objects);
      const nextDefaultFormat = defaultFormatForSelection(nextRow.domain, nextRow.objects);
      const preferredFormat = !hasFormatPatch && hasSelectionPatch && row.format === oldDefaultFormat
        ? nextDefaultFormat
        : nextRow.format;
      return {
        ...nextRow,
        format: normalizeFormatForDomain(nextRow.domain, preferredFormat, nextRow.objects),
      };
    }));
  }, []);

  // Stable commit handler for PeriodTextInput (identity-stable so the memoized cell does
  // not re-render on every parent render).
  const commitPeriodTextField = useCallback((rowKey: string, field: 'path' | 'fileName', value: string) => {
    setEditorPeriodRows((rows) => rows.map((row) => (row.key === rowKey ? { ...row, [field]: value } : row)));
  }, []);

  const updateEditorPeriod = (key: string, period: string) => {
    updateEditorPeriodRow(key, { period, cron: defaultCronForPeriod(period) });
  };

  const updateInventoryPeriod = (key: InventoryType, period: string) => {
    updateInventoryConfig(key, { period, cron: defaultCronForPeriod(period) });
  };

  const updateEditorPeriodDomain = (key: string, domain: Domain) => {
    const nextDefault = defaultPeriodRow(domain, false);
    setEditorPeriodRows((rows) =>
      rows.map((row) => (row.key === key ? {
        ...row,
        domain,
        scope: nextDefault.scope,
        format: nextDefault.format,
        period: nextDefault.period,
        cron: nextDefault.cron,
        objects: '',
        path: nextDefault.path,
        fileName: nextDefault.fileName,
        csvSeparator: nextDefault.csvSeparator,
        compressionEnabled: nextDefault.compressionEnabled,
        compressionFormat: nextDefault.compressionFormat,
      } : row)),
    );
  };

  const addEditorPeriodRow = () => {
    const selectedDomains = (configForm.getFieldValue('domains') ?? []) as Domain[];
    const [domain] = selectedDomains;
    if (!domain) {
      void message.warning(nt('请先选择业务域'));
      return;
    }
    setEditorPeriodRows((rows) => [...rows, defaultPeriodRow(domain, false)]);
  };

  const removeEditorPeriodRow = (key: string) => {
    setEditorPeriodRows((rows) => rows.filter((row) => row.key !== key));
  };

  const refreshScenarioList = () => {
    void loadPageConfig();
  };

  const renderTablePagination = (ariaLabel: string, pageSize = 10) => ({
    pageSize,
    showSizeChanger: false,
    showTotal: () => (
      <Tooltip title="刷新">
        <Button
          aria-label={ariaLabel}
          type="text"
          size="small"
          icon={<ReloadOutlined />}
          onClick={refreshScenarioList}
        />
      </Tooltip>
    ),
  });

  const renderScheduleEditor = (
    period: string,
    cron: string,
    onChange: (nextCron: string) => void,
  ) => {
    if (period === '24H') {
      return (
        <Input
          aria-label="每日生成时间"
          type="time"
          value={getDailyTimeValue(cron)}
          onChange={(event) => onChange(cronFromDailyTime(event.target.value))}
        />
      );
    }
    return (
      <Space size={8} wrap>
        <Tag>{formatRecurringPeriodLabel(period)}</Tag>
        <Select
          aria-label="起始分钟"
          value={getIntervalStartMinuteValue(cron)}
          style={{ width: 116 }}
          options={startMinuteOptions}
          onChange={(startMinute) => onChange(cronFromIntervalStart(period, startMinute))}
        />
      </Space>
    );
  };

  const downloadReportArtifact = async (info: ReportStatusInfo) => {
    if (info.artifactType !== 'file') {
      setSelectedReportStatus(info);
      return;
    }
    const blob = info.runId
      ? await northboundPageConfigApi.downloadRun(info.runId)
      : new Blob([info.payload], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = info.artifactName;
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
  };

  const copyReportPayload = (info: ReportStatusInfo) => {
    const finish = (success: boolean) => {
      if (success) {
        void message.success(nt('报文已复制'));
      } else {
        void message.error(nt('报文复制失败'));
      }
    };
    if (navigator.clipboard?.writeText) {
      void navigator.clipboard.writeText(info.payload)
        .then(() => finish(true))
        .catch(() => finish(false));
      return;
    }
    const textarea = document.createElement('textarea');
    textarea.value = info.payload;
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    document.body.appendChild(textarea);
    textarea.select();
    const success = document.execCommand('copy');
    textarea.remove();
    finish(success);
  };

  const openSingleEventReport = (event: NorthboundPageConfigEvent, capabilityName: string) => {
    setReportRunList([]);
    setReportEventList([event]);
    setReportEventTotal(1);
    setReportEventLoading(false);
    setReportCapabilityName(capabilityName);
    setReportDeliveryEventsByRunId({});
    setSelectedReportStatus(buildEventReportStatus(event, capabilityName));
  };

  const renderStatusCell = (health: ProfileHealthInfo) => {
    return (
      <Tooltip title={nt(health.detail)}>
        <Tag data-northbound-i18n-skip="true" color={health.color}>
          {nt(health.label)}
        </Tag>
      </Tooltip>
    );
  };

  const renderRunningStatusCell = (detail: string) => (
    <Tooltip title={nt(detail)}>
      <Tag color="processing" icon={<LoadingOutlined spin />}>
        {nt('执行中')}
      </Tag>
    </Tooltip>
  );

  const showLatestRunReport = (
    profileKind: 'file' | 'inventory',
    profileCode: string,
    fallback: ReportStatusInfo,
  ) => {
    const isFile = profileKind === 'file';
    setReportEventList([]);
    setReportEventTotal(0);
    setReportEventLoading(false);
    setReportDeliveryEventsByRunId({});
    setReportCapabilityName(fallback.capabilityName);
    void northboundPageConfigApi
      .listRuns({ profile_kind: profileKind, profile_code: profileCode, limit: isFile ? 200 : 1 })
      .then(async (result) => {
        if (isFile) {
          // listRuns 默认 created_at DESC，按 object_code 去重保留每个对象最新一条
          const seen = new Set<string>();
          const deduped = result.items.filter((item) => {
            const key = item.object_code || item.id;
            if (seen.has(key)) return false;
            seen.add(key);
            return true;
          });
          if (deduped.length === 0) {
            setReportRunList([]);
            setSelectedReportStatus(fallback);
            return;
          }
          openReportDrawer(deduped, fallback.capabilityName, fallback);
          return;
        }
        const latest = result.items[0];
        if (!latest) {
          setReportRunList([]);
          setSelectedReportStatus(fallback);
          return;
        }
        setReportRunList([]);
        const run = await northboundPageConfigApi.getRun(latest.id);
        setSelectedReportStatus(await resolveRunReportStatus(run, fallback.capabilityName));
      })
      .catch(() => {
        setReportRunList([]);
        setSelectedReportStatus(fallback);
        void message.warning(nt('未读取到最近上报记录，已显示配置预览'));
      });
  };

  const resolveRunReportStatus = async (
    run: NorthboundFileRun,
    fallbackCapabilityName: string,
    deliveryEvents?: NorthboundPageConfigEvent[],
  ): Promise<ReportStatusInfo> => {
    // listRuns / dedupe items omit artifact_content; fetch the full run so the
    // preview reflects the selected object's actual file content.
    let fullRun = run;
    if (!run.artifact_content) {
      try {
        fullRun = await northboundPageConfigApi.getRun(run.id);
      } catch {
        fullRun = run;
      }
    }
    const runStatus = buildRunReportStatus(fullRun, fallbackCapabilityName);
    if (deliveryEvents) {
      return mergeRunDeliveryStatus(runStatus, deliveryEvents);
    }
    try {
      const events = await northboundPageConfigApi.listEvents({
        capability: 'delivery',
        owner_code: run.profile_code,
        limit: 50,
        include_payload: false,
      });
      // Precisely match this run's delivery events by summary.run_id. Never fall
      // back to an unrelated event — the old `?? events.items[0]` crossed objects.
      const deliveries = events.items.filter((event) => eventRunId(event) === run.id);
      return mergeRunDeliveryStatus(runStatus, deliveries);
    } catch {
      return runStatus;
    }
  };

  const openReportDrawer = (
    items: NorthboundFileRun[],
    capabilityName: string,
    fallback: ReportStatusInfo | null,
  ) => {
    setReportCapabilityName(capabilityName);
    setReportRunList(items);
    setReportEventList([]);
    setReportEventTotal(0);
    setReportDeliveryEventsByRunId({});
    const firstAvailableRun = items.find((item) => item.status === 'success') ?? items[0];
    if (!firstAvailableRun) {
      setSelectedReportStatus(fallback);
      return;
    }
    void northboundPageConfigApi.listEvents({
      capability: 'delivery',
      owner_code: firstAvailableRun.profile_code,
      limit: 200,
      include_payload: false,
    })
      .then((events) => {
        const eventsByRunId = groupDeliveryEventsByRunId(events.items);
        setReportDeliveryEventsByRunId(eventsByRunId);
        const firstRun = selectInitialReportRun(items, eventsByRunId) ?? firstAvailableRun;
        void resolveRunReportStatus(firstRun, capabilityName, eventsByRunId[firstRun.id] ?? [])
          .then((info) => {
            setSelectedReportStatus(info ?? fallback);
          });
      })
      .catch(() => {
        void resolveRunReportStatus(firstAvailableRun, capabilityName).then((info) => {
          setSelectedReportStatus(info ?? fallback);
        });
      });
  };

  const selectReportEvent = (event: NorthboundPageConfigEvent, capabilityName: string) => {
    setSelectedReportStatus(buildEventReportStatus(event, capabilityName));
    void northboundPageConfigApi.getEvent(event.id)
      .then((fullEvent) => {
        setSelectedReportStatus(buildEventReportStatus(fullEvent, capabilityName));
      })
      .catch(() => {
        void message.warning(nt('未读取到完整事件详情，已显示列表摘要'));
      });
  };

  const showLatestEventReport = (
    capability: NorthboundPageConfigEvent['capability'],
    targetKey: string,
    fallback: ReportStatusInfo,
  ) => {
    setReportRunList([]);
    setReportEventList([]);
    setReportEventTotal(0);
    setReportDeliveryEventsByRunId({});
    setReportCapabilityName(fallback.capabilityName);
    setSelectedReportStatus(fallback);
    setReportEventLoading(true);
    void northboundPageConfigApi.listEvents({
      capability,
      target_key: targetKey,
      limit: 50,
      include_payload: false,
    })
      .then((result) => {
        const latest = result.items[0];
        setReportEventList(result.items);
        setReportEventTotal(result.total);
        if (latest) {
          selectReportEvent(latest, fallback.capabilityName);
        } else {
          setSelectedReportStatus(fallback);
        }
      })
      .catch(() => {
        setSelectedReportStatus(fallback);
        void message.warning(nt('未读取到最近事件，已显示配置预览'));
      })
      .finally(() => {
        setReportEventLoading(false);
      });
  };

  const persistFileProfileEnabled = (row: ScenarioRow, checked: boolean) => {
    const previous = Boolean(scenarioEnabled[row.code]);
    setScenarioEnabled((prev) => ({ ...prev, [row.code]: checked }));
    setFileSaving(row.code, true);
    void northboundPageConfigApi.updateFileProfile(row.code, {
      enabled: checked,
      status: checked ? 'normal' : 'terminated',
    })
      .then(applyFileProfile)
      .catch(() => {
        setScenarioEnabled((prev) => ({ ...prev, [row.code]: previous }));
        void message.error(nt(`${row.code} 启停状态保存失败`));
      })
      .finally(() => setFileSaving(row.code, false));
  };

  const persistInventoryEnabled = (row: InventoryConfigRow, checked: boolean) => {
    const previous = Boolean(inventoryEnabled[row.key]);
    setInventoryEnabled((prev) => ({ ...prev, [row.key]: checked }));
    setInventorySaving(row.key, true);
    void northboundPageConfigApi.updateInventoryProfile(row.key, {
      enabled: checked,
      status: checked ? 'normal' : 'terminated',
    })
      .then(applyInventoryProfile)
      .catch(() => {
        setInventoryEnabled((prev) => ({ ...prev, [row.key]: previous }));
        void message.error(nt(`${row.objectCode} Inventory 启停状态保存失败`));
      })
      .finally(() => setInventorySaving(row.key, false));
  };

  const hasFailedDeliveryForRuns = async (profileCode: string, runs: NorthboundFileRun[]) => {
    const runIds = new Set(runs.filter((run) => run.status === 'success').map((run) => run.id));
    if (runIds.size === 0) return false;
    try {
      const events = await northboundPageConfigApi.listEvents({
        capability: 'delivery',
        owner_code: profileCode,
        limit: 200,
        include_payload: false,
      });
      const eventsByRunId = groupDeliveryEventsByRunId(events.items);
      return runs.some((run) => runIds.has(run.id) && runHasFailedDelivery(run, eventsByRunId));
    } catch {
      return false;
    }
  };

  const runFileProfile = (row: ScenarioRow) => {
    if (fileProfileRunning[row.code]) {
      void message.open({
        key: `northbound-file-run-${row.code}`,
        type: 'info',
        content: nt(`${row.code} 正在生成并上传，请稍候`),
        duration: 2,
      });
      return;
    }
    const messageKey = `northbound-file-run-${row.code}`;
    setFileProfileRunning((prev) => ({ ...prev, [row.code]: true }));
    void message.open({
      key: messageKey,
      type: 'loading',
      content: nt(`${row.code} 正在生成文件并上传到传输目标...`),
      duration: 0,
    });
    void northboundPageConfigApi.runFileProfile(row.code, { limit: 200 })
      .then(async (result) => {
        const latestSuccess = latestSuccessRunMap(result.items)[row.code];
        if (latestSuccess) {
          setFileLatestSuccessRuns((prev) => ({ ...prev, [row.code]: latestSuccess }));
        }
        openReportDrawer(result.items, `${row.code} 北向文件`, null);
        const deliveryFailed = await hasFailedDeliveryForRuns(row.code, result.items);
        void message.open({
          key: messageKey,
          type: deliveryFailed ? 'warning' : 'success',
          content: nt(deliveryFailed
            ? `${row.code} 文件生成完成，传输目标上传失败，请查看上报结果`
            : `${row.code} 已生成 ${result.total} 条上报记录`),
          duration: deliveryFailed ? 5 : 3,
        });
      })
      .catch(() => {
        void message.open({
          key: messageKey,
          type: 'error',
          content: nt(`${row.code} 手动执行失败`),
          duration: 5,
        });
      })
      .finally(() => {
        setFileProfileRunning((prev) => ({ ...prev, [row.code]: false }));
      });
  };

  const runInventoryProfile = (row: InventoryConfigRow) => {
    if (inventoryProfileRunning[row.key]) {
      void message.open({
        key: `northbound-inventory-run-${row.key}`,
        type: 'info',
        content: nt(`${row.objectCode} Inventory 正在生成并上传，请稍候`),
        duration: 2,
      });
      return;
    }
    const messageKey = `northbound-inventory-run-${row.key}`;
    setInventoryProfileRunning((prev) => ({ ...prev, [row.key]: true }));
    void message.open({
      key: messageKey,
      type: 'loading',
      content: nt(`${row.objectCode} Inventory 正在生成文件并上传到传输目标...`),
      duration: 0,
    });
    void northboundPageConfigApi.runInventoryProfile(row.key, { limit: 200 })
      .then(async (run) => {
        if (run.status === 'success') {
          setInventoryLatestSuccessRuns((prev) => ({ ...prev, [row.key]: run }));
        }
        void resolveRunReportStatus(run, `${row.objectCode} Inventory`)
          .then(setSelectedReportStatus);
        const deliveryFailed = await hasFailedDeliveryForRuns(row.key, [run]);
        void message.open({
          key: messageKey,
          type: deliveryFailed ? 'warning' : 'success',
          content: nt(deliveryFailed
            ? `${row.objectCode} Inventory 生成完成，传输目标上传失败，请查看上报结果`
            : `${row.objectCode} Inventory 已生成上报记录`),
          duration: deliveryFailed ? 5 : 3,
        });
      })
      .catch(() => {
        void message.open({
          key: messageKey,
          type: 'error',
          content: nt(`${row.objectCode} Inventory 手动执行失败`),
          duration: 5,
        });
      })
      .finally(() => {
        setInventoryProfileRunning((prev) => ({ ...prev, [row.key]: false }));
      });
  };

  const persistSnmpEnabled = (row: SnmpAlarmTargetRow, checked: boolean) => {
    const displayName = snmpVersionLabel(row.version);
    const blocker = getSnmpConfigBlocker(row, checked);
    if (blocker) {
      void message.warning(nt(`${displayName} ${blocker}`));
      return;
    }
    const previous = Boolean(snmpEnabled[row.key]);
    setSnmpTargetSaving(row.key, true);
    setSnmpEnabled((prev) => ({ ...prev, [row.key]: checked }));
    void northboundPageConfigApi.updateSNMPAlarmTarget(row.key, serializeSnmpTarget(row, checked))
      .then((target) => {
        const next = mapApiSnmpTarget(target);
        setSnmpTargets((rows) => rows.map((item) => (item.key === next.key ? next : item)));
        setSnmpEnabled((prev) => ({ ...prev, [next.key]: target.enabled }));
      })
      .catch(() => {
        setSnmpEnabled((prev) => ({ ...prev, [row.key]: previous }));
        void message.error(nt(`${displayName} SNMP 启停状态保存失败`));
      })
      .finally(() => setSnmpTargetSaving(row.key, false));
  };

  const persistSocketEnabled = (row: SocketAlarmConfigRow, checked: boolean) => {
    const previous = Boolean(socketEnabled[row.key]);
    const accounts = getSocketAccounts(row);
    setSocketEnabled((prev) => ({ ...prev, [row.key]: checked }));
    void northboundPageConfigApi.updateSocketAlarmConfig(row.key, serializeSocketConfig(row, accounts, checked))
      .then((config) => {
        const next = mapApiSocketConfig(config);
        setSocketConfigs((rows) => rows.map((item) => (item.key === next.key ? next : item)));
        setSocketEnabled((prev) => ({ ...prev, [next.key]: config.enabled }));
        setSocketAccountRowsByConfig((prev) => ({ ...prev, [config.key]: mapApiSocketAccounts(config) }));
      })
      .catch(() => {
        setSocketEnabled((prev) => ({ ...prev, [row.key]: previous }));
        void message.error(nt(`${row.name} Socket 启停状态保存失败`));
      });
  };

  const persistAllApiEnabled = (checked: boolean) => {
    const previous = { ...apiEnabled };
    setApiSwitchSaving(true);
    setApiEnabled((prev) => ({
      ...prev,
      ...Object.fromEntries(apiManagedKeys.map((key) => [key, checked])),
    }));
    void northboundPageConfigApi.updateAllAPIConfigs(checked)
      .then((resp) => {
        const configRows = resp.items.map(mapApiConfig);
        setApiRows(expandApiDisplayRows(configRows));
        setApiEnabled(Object.fromEntries(resp.items.map((row) => [row.key, row.enabled])));
        void message.success(nt(`北向 API 总开关已${checked ? '启用' : '停用'}`));
      })
      .catch(() => {
        setApiEnabled(previous);
        void message.error(nt('北向 API 总开关保存失败'));
      })
      .finally(() => setApiSwitchSaving(false));
  };

  const patchApiUser = (key: string, patch: Partial<ApiUserRow>) => {
    apiUserDirtyRef.current = true;
    setApiUsers((rows) => rows.map((row) => (row.key === key ? { ...row, ...patch } : row)));
  };

  const addApiUser = () => {
    const index = apiUsers.length + 1;
    apiUserDirtyRef.current = true;
    setApiUsers((rows) => [
      ...rows,
      {
        key: newApiUserKey(),
        username: `north_api_${index}`,
        enabled: false,
        password: '',
        passwordSet: false,
      },
    ]);
  };

  const removeApiUser = (key: string) => {
    apiUserDirtyRef.current = true;
    setApiUsers((rows) => rows.filter((row) => row.key !== key));
  };

  const saveApiUsers = () => {
    apiUserSavingRef.current = true;
    setApiUserSaving(true);
    void northboundPageConfigApi.replaceAPIUsers(serializeApiUsers(apiUsers))
      .then((resp) => {
        apiUserDirtyRef.current = false;
        setApiUsers(resp.items.map(mapApiUser));
        void message.success(nt('北向 API 用户已保存'));
      })
      .catch(() => {
        void message.error(nt('北向 API 用户保存失败'));
      })
      .finally(() => {
        apiUserSavingRef.current = false;
        setApiUserSaving(false);
      });
  };

  const testSocketAlarm = (row: SocketAlarmConfigRow) => {
    void northboundPageConfigApi.testSocketAlarmConfig(row.key)
      .then((event) => {
        openSingleEventReport(event, row.name);
        void message.success(nt(`${row.name} 样例报文已记录`));
      })
      .catch(() => {
        void message.error(nt(`${row.name} 样例报文生成失败`));
      });
  };

  const testSnmpAlarm = (row: SnmpAlarmTargetRow) => {
    const displayName = snmpVersionLabel(row.version);
    void northboundPageConfigApi.testSNMPAlarmTarget(row.key)
      .then((event) => {
        openSingleEventReport(event, displayName);
        if (event.status === 'success') {
          void message.success(nt(`${displayName} 测试报文已生成`));
        } else {
          void message.warning(nt(`${displayName} 测试报文已生成，但目标配置不完整`));
        }
      })
      .catch(() => {
        void message.error(nt(`${displayName} 测试报文生成失败`));
      });
  };

  const testApiContract = (row: NorthboundApiRow) => {
    void northboundPageConfigApi.testAPIConfig(apiConfigKey(row))
      .then((event) => {
        openSingleEventReport(event, row.name);
        void message.success(nt(`${row.name} 接口检查已记录`));
      })
      .catch(() => {
        void message.error(nt(`${row.name} 接口检查失败`));
      });
  };

  const saveSocketEditor = (row: SocketAlarmConfigRow) => {
    const enabled = socketEditorEnabled;
    const accounts = socketEditorAccounts;
    const deliveryRows = socketEditorDeliveryTargets;
    if (!validateDeliveryTargetsBeforeSave('socket', row.key, deliveryRows)) return;
    void Promise.all([
      northboundPageConfigApi.updateSocketAlarmConfig(row.key, serializeSocketConfig(row, accounts, enabled)),
      northboundPageConfigApi.replaceDeliveryTargets(
        serializeDeliveryTargets('socket', row.key, deliveryRows),
      ),
    ])
      .then(([config, deliveryResp]) => {
        const next = mapApiSocketConfig(config);
        setSocketConfigs((rows) => rows.map((item) => (item.key === next.key ? next : item)));
        setSocketEnabled((prev) => ({ ...prev, [next.key]: config.enabled }));
        setSocketAccountRowsByConfig((prev) => ({ ...prev, [config.key]: mapApiSocketAccounts(config) }));
        setSocketDeliveryTargetsByConfig((prev) => ({
          ...prev,
          [config.key]: deliveryResp.items.map(mapApiDeliveryTarget),
        }));
        void message.success(nt(`${next.name} 已保存`));
        closeSocketEditor();
      })
      .catch(() => {
        void message.error(nt(`${row.name} Socket 配置保存失败`));
      });
  };

  const saveSnmpEditor = (row: SnmpAlarmTargetRow) => {
    const enabled = Boolean(snmpEnabled[row.key]);
    const displayName = snmpVersionLabel(row.version);
    const blocker = getSnmpConfigBlocker(row, enabled);
    if (blocker) {
      void message.warning(nt(`${displayName} ${blocker}`));
      return;
    }
    setSnmpTargetSaving(row.key, true);
    void northboundPageConfigApi.updateSNMPAlarmTarget(row.key, serializeSnmpTarget(row, enabled))
      .then((target) => {
        const next = mapApiSnmpTarget(target);
        setSnmpTargets((rows) => rows.map((item) => (item.key === next.key ? next : item)));
        setSnmpEnabled((prev) => ({ ...prev, [next.key]: target.enabled }));
        void message.success(nt(`${snmpVersionLabel(next.version)} 已保存`));
        setSnmpEditor(null);
      })
      .catch(() => {
        void message.error(nt(`${displayName} SNMP 配置保存失败`));
      })
      .finally(() => setSnmpTargetSaving(row.key, false));
  };

  const scenarioColumns: ColumnsType<ScenarioRow> = [
    {
      title: '操作',
      width: 104,
      fixed: 'left',
      render: (_, row) => {
        const running = Boolean(fileProfileRunning[row.code]);
        return (
          <div className={styles.rowControl}>
            <Switch
              size="small"
              checked={scenarioEnabled[row.code]}
              checkedChildren="开"
              unCheckedChildren="关"
              loading={Boolean(fileProfileSaving[row.code])}
              onClick={(_, event) => event.stopPropagation()}
              onChange={(checked) => persistFileProfileEnabled(row, checked)}
            />
            <Dropdown
              trigger={['click']}
              menu={{
                items: [
                  { key: 'view', icon: <EyeOutlined />, label: '查看' },
                  { key: 'edit', icon: <EditOutlined />, label: '编辑' },
                  { key: 'report', icon: <FileSearchOutlined />, label: '上报结果' },
                  { key: 'copy', icon: <CopyOutlined />, label: '复制模板' },
                  {
                    key: 'run',
                    icon: running ? <LoadingOutlined spin /> : <PlayCircleOutlined />,
                    label: running ? '执行中' : '手动执行',
                    disabled: running,
                  },
                ],
                onClick: ({ key, domEvent }) => {
                  domEvent.stopPropagation();
                  if (key === 'view') {
                    openViewDrawer(row);
                    return;
                  }
                  if (key === 'edit') {
                    openEditEditor(row);
                    return;
                  }
                  if (key === 'report') {
                    showLatestRunReport(
                      'file',
                      row.code,
                      effectiveReportStatus(buildFileReportStatus(row), Boolean(scenarioEnabled[row.code])),
                    );
                    return;
                  }
                  if (key === 'copy') {
                    void message.info(nt(`${row.code} 已复制为草稿`));
                    return;
                  }
                  runFileProfile(row);
                },
              }}
            >
              <Button
                aria-label={`更多操作 ${row.code}`}
                type="text"
                size="small"
                icon={running ? <LoadingOutlined spin /> : <MoreOutlined />}
                onClick={(event) => event.stopPropagation()}
              />
            </Dropdown>
          </div>
        );
      },
    },
    {
      title: '场景号',
      dataIndex: 'code',
      width: 96,
      render: (value: string) => <Typography.Text strong className={styles.scenarioCode}>{value}</Typography.Text>,
    },
    {
      title: '场景名称',
      width: 140,
      render: (_, row) => <Typography.Text ellipsis>{scenarioDisplayName(row, locale)}</Typography.Text>,
    },
    {
      title: '状态',
      width: 104,
      render: (_, row) => (
        fileProfileRunning[row.code]
          ? renderRunningStatusCell(`${row.code} 正在生成文件并上传传输目标`)
          : renderStatusCell(buildProfileHealth(
            Boolean(scenarioEnabled[row.code]),
            scenarioMaxPeriodMinutes(row),
            fileLatestSuccessRuns[row.code],
            row.createdAt,
            row.updatedAt,
          ))
      ),
    },
    {
      title: '输出内容',
      width: 360,
      render: (_, row) => (
        <Tooltip title={<pre className={styles.tooltipPre}>{getScenarioOutputTooltip(row)}</pre>}>
          <div className={styles.summaryCell}>
            <div className={styles.summaryChips}>
              {summarizeScenarioDomains(row).slice(0, 4).map((summary) => (
                <span key={summary.domain} className={styles.summaryChip}>
                  <span className={styles.summaryDomain}>{summary.domain}</span>
                  <span className={styles.summaryValue}>{summarizeValues(summary.objects)}</span>
                </span>
              ))}
            </div>
          </div>
        </Tooltip>
      ),
    },
    {
      title: '调度',
      width: 240,
      render: (_, row) => (
        <Tooltip title={<pre className={styles.tooltipPre}>{getScenarioScheduleTooltip(row)}</pre>}>
          <div className={styles.summaryCell}>
            <div className={styles.summaryChips}>
              {summarizeScenarioDomains(row).slice(0, 4).map((summary) => (
                <span key={summary.domain} className={styles.summaryChipMuted}>
                  {summary.domain} {summarizeValues(summary.periods, 2)}
                </span>
              ))}
            </div>
          </div>
        </Tooltip>
      ),
    },
    {
      title: '文件规则',
      width: 250,
      render: (_, row) => (
        <Tooltip title={<pre className={styles.tooltipPre}>{getScenarioFileTooltip(row)}</pre>}>
          <div className={styles.summaryCell}>
            <div className={styles.summaryChips}>
              {[...new Set(getScenarioPeriodRows(row).map((periodRow) => periodRow.format))].map((format) => (
                <span key={format} className={styles.formatChip}>{format}</span>
              ))}
            </div>
          </div>
        </Tooltip>
      ),
    },
    {
      title: nowrapColumnTitle('压缩'),
      width: 124,
      render: (_, row) => (
        <div className={styles.compressionList}>
          {getScenarioCompressionTags(row).map((compression) => (
            <Tag key={compression} color={compression === '不压缩' ? 'default' : 'green'}>{compression}</Tag>
          ))}
        </div>
      ),
    },
  ];

  const inventoryConfigColumns: ColumnsType<InventoryConfigRow> = [
    {
      title: '操作',
      width: 104,
      fixed: 'left',
      render: (_, row) => {
        const running = Boolean(inventoryProfileRunning[row.key]);
        return (
          <div className={styles.rowControl}>
            <Switch
              size="small"
              checked={inventoryEnabled[row.key]}
              checkedChildren="开"
              unCheckedChildren="关"
              loading={Boolean(inventoryProfileSaving[row.key])}
              onClick={(_, event) => event.stopPropagation()}
              onChange={(checked) => persistInventoryEnabled(row, checked)}
            />
            <Dropdown
              trigger={['click']}
              menu={{
                items: [
                  { key: 'view', icon: <EyeOutlined />, label: '查看' },
                  { key: 'edit', icon: <EditOutlined />, label: '编辑' },
                  { key: 'report', icon: <FileSearchOutlined />, label: '上报结果' },
                  {
                    key: 'run',
                    icon: running ? <LoadingOutlined spin /> : <PlayCircleOutlined />,
                    label: running ? '执行中' : '手动执行',
                    disabled: running,
                  },
                ],
                onClick: ({ key, domEvent }) => {
                  domEvent.stopPropagation();
                  if (key === 'view') {
                    openInventoryView(row);
                    return;
                  }
                  if (key === 'edit') {
                    openInventoryEditor(row);
                    return;
                  }
                  if (key === 'report') {
                    showLatestRunReport(
                      'inventory',
                      row.key,
                      effectiveReportStatus(buildInventoryReportStatus(row), Boolean(inventoryEnabled[row.key])),
                    );
                    return;
                  }
                  runInventoryProfile(row);
                },
              }}
            >
              <Button
                aria-label={`更多操作 ${row.objectCode} Inventory`}
                type="text"
                size="small"
                icon={running ? <LoadingOutlined spin /> : <MoreOutlined />}
                onClick={(event) => event.stopPropagation()}
              />
            </Dropdown>
          </div>
        );
      },
    },
    {
      title: '类型',
      dataIndex: 'objectCode',
      width: 96,
      fixed: 'left',
      render: (value: string) => <Typography.Text strong>{value}</Typography.Text>,
    },
    {
      title: '名称',
      dataIndex: 'name',
      width: 170,
      render: (value: string) => <Typography.Text ellipsis>{value}</Typography.Text>,
    },
    {
      title: '状态',
      width: 104,
      render: (_, row) => (
        inventoryProfileRunning[row.key]
          ? renderRunningStatusCell(`${row.objectCode} Inventory 正在生成文件并上传传输目标`)
          : renderStatusCell(buildProfileHealth(
            Boolean(inventoryEnabled[row.key]),
            getPeriodMinutes(row.period),
            inventoryLatestSuccessRuns[row.key],
            row.createdAt,
            row.updatedAt,
          ))
      ),
    },
    {
      title: '输出内容',
      width: 280,
      render: (_, row) => (
        <Tooltip title={<pre className={styles.tooltipPre}>{getInventoryFieldTooltip(row.key)}</pre>}>
          <div className={styles.summaryCell}>
            <div className={styles.summaryChips}>
              <span className={styles.summaryChip}>
                <span className={styles.summaryDomain}>{row.objectCode}</span>
                <span className={styles.summaryValue}>{row.tech}</span>
              </span>
              <span className={styles.summaryChipMuted}>字段 {inventoryFieldRowsByType[row.key]?.length ?? 0}</span>
            </div>
          </div>
        </Tooltip>
      ),
    },
    {
      title: '调度',
      width: 160,
      render: (_, row) => (
        <Tooltip title={<pre className={styles.tooltipPre}>{getInventoryScheduleTooltip(row)}</pre>}>
          <div className={styles.summaryCell}>
            <div className={styles.summaryChips}>
              <span className={styles.summaryChipMuted}>{formatPeriodLabel(row.period)}</span>
            </div>
          </div>
        </Tooltip>
      ),
    },
    {
      title: '文件规则',
      width: 220,
      render: (_, row) => (
        <Tooltip title={<pre className={styles.tooltipPre}>{getInventoryFileTooltip(row)}</pre>}>
          <div className={styles.summaryCell}>
            <div className={styles.summaryChips}>
              <span className={styles.formatChip}>{row.format}</span>
              <span className={styles.summaryChipMuted}>{row.objectCode}</span>
            </div>
          </div>
        </Tooltip>
      ),
    },
    {
      title: nowrapColumnTitle('压缩'),
      width: 124,
      render: (_, row) => row.compressionEnabled
        ? <Tag color="green">{row.compressionFormat}</Tag>
        : <Tag>不压缩</Tag>,
    },
  ];

  const readonlyInventoryFieldColumns: ColumnsType<InventoryField> = [
    {
      title: '上报',
      width: 76,
      fixed: 'left',
      render: (_, record) => <Switch size="small" checked={record.enabled} disabled checkedChildren="开" unCheckedChildren="关" />,
    },
    {
      title: '输出别名',
      dataIndex: 'column',
      width: 260,
      fixed: 'left',
      render: (value: string) => <span className={styles.monoText}>{value}</span>,
    },
    { title: '系统字段 key', dataIndex: 'exportKey', width: 260, render: (value: string) => <span className={styles.monoText}>{value}</span> },
    { title: '系统取数字段', dataIndex: 'source', width: 340 },
    { title: '类型', dataIndex: 'dataType', width: 100 },
    { title: '渲染规则', dataIndex: 'renderer', width: 140, render: (value: string) => <Tag>{value}</Tag> },
  ];

  const inventoryFieldColumns: ColumnsType<InventoryField> = [
    {
      title: '上报',
      width: 76,
      fixed: 'left',
      render: (_, record) => (
        <Switch
          size="small"
          checked={record.enabled}
          checkedChildren="开"
          unCheckedChildren="关"
          onChange={(enabled) => updateInventoryFieldEnabled(record.key, enabled)}
        />
      ),
    },
    {
      title: '输出别名',
      dataIndex: 'column',
      width: 260,
      fixed: 'left',
      render: (value: string, record) => (
        <Input
          value={value}
          className={styles.monoText}
          onChange={(event) => updateInventoryFieldAlias(record.key, event.target.value)}
        />
      ),
    },
    { title: '系统字段 key', dataIndex: 'exportKey', width: 260, render: (value: string) => <span className={styles.monoText}>{value}</span> },
    { title: '系统取数字段', dataIndex: 'source', width: 340 },
    { title: '类型', dataIndex: 'dataType', width: 100 },
    { title: '渲染规则', dataIndex: 'renderer', width: 140, render: (value: string) => <Tag>{value}</Tag> },
    {
      title: '操作',
      width: 74,
      fixed: 'right',
      render: (_, record) => (
        <Tooltip title="从当前 Inventory 移除">
          <Button
            aria-label={`删除 Inventory 字段 ${record.column}`}
            type="text"
            danger
            size="small"
            icon={<DeleteOutlined />}
            onClick={() => removeInventoryField(record.key)}
          />
        </Tooltip>
      ),
    },
  ];

  const apiColumns: ColumnsType<NorthboundApiRow> = [
    {
      title: '操作',
      width: 64,
      fixed: 'left',
      render: (_, row) => (
        <Dropdown
          trigger={['click']}
          menu={{
            items: [
              { key: 'view', icon: <EyeOutlined />, label: '查看' },
              { key: 'report', icon: <FileSearchOutlined />, label: '最近检查结果' },
              { key: 'test', icon: <PlayCircleOutlined />, label: '检查接口' },
            ],
            onClick: ({ key, domEvent }) => {
              domEvent.stopPropagation();
              if (key === 'view') setSelectedApi(row);
              if (key === 'report') showLatestEventReport('api', apiConfigKey(row), effectiveReportStatus(buildApiReportStatus(row), Boolean(apiEnabled[apiConfigKey(row)])));
              if (key === 'test') testApiContract(row);
            },
          }}
        >
          <Button
            aria-label={`更多操作 ${row.name}`}
            type="text"
            size="small"
            icon={<MoreOutlined />}
            onClick={(event) => event.stopPropagation()}
          />
        </Dropdown>
      ),
    },
    {
      title: '模块',
      width: 150,
      render: (_, row) => apiModuleTag(row),
    },
    {
      title: '接口名称',
      dataIndex: 'name',
      width: 330,
      render: (value: string) => (
        <Tooltip title={value}>
          <Typography.Text strong ellipsis>{value}</Typography.Text>
        </Tooltip>
      ),
    },
    {
      title: '方法',
      dataIndex: 'method',
      width: 86,
      render: (value: ApiMethod) => apiMethodTag(value),
    },
    {
      title: '接口 URL',
      dataIndex: 'url',
      width: 440,
      render: (value: string) => <Typography.Text className={styles.monoText} ellipsis>{value}</Typography.Text>,
    },
  ];

  const apiUserColumns: ColumnsType<ApiUserRow> = [
    {
      title: '启用',
      dataIndex: 'enabled',
      width: 76,
      fixed: 'left',
      render: (value: boolean, row) => (
        <Switch
          size="small"
          checked={value}
          checkedChildren="开"
          unCheckedChildren="关"
          onChange={(enabled) => patchApiUser(row.key, { enabled })}
        />
      ),
    },
    {
      title: '用户名',
      dataIndex: 'username',
      width: 210,
      fixed: 'left',
      render: (value: string, row) => (
        <Input
          value={value}
          className={styles.monoText}
          onChange={(event) => patchApiUser(row.key, { username: event.target.value })}
        />
      ),
    },
    {
      title: '密码',
      dataIndex: 'password',
      width: 220,
      render: (value: string, row) => (
        <MaskedCredentialInput
          value={value}
          placeholder={row.passwordSet ? '未修改保持原密码' : '请输入密码'}
          storedValues={[apiUserPasswordMask]}
          onChange={(password) => patchApiUser(row.key, { password })}
        />
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      width: 180,
      render: (value: string | undefined) => (
        <span className={styles.monoText}>{formatRunTime(value)}</span>
      ),
    },
    {
      title: '操作',
      width: 74,
      fixed: 'right',
      render: (_, row) => (
        <Tooltip title="删除用户">
          <Button
            danger
            size="small"
            type="text"
            icon={<DeleteOutlined />}
            onClick={() => removeApiUser(row.key)}
          />
        </Tooltip>
      ),
    },
  ];

  const readonlyFieldConfigColumns: ColumnsType<ReportFieldRow> = [
    {
      title: '上报',
      width: 76,
      fixed: 'left',
      render: (_, row) => <Switch size="small" checked={row.enabled} disabled checkedChildren="开" unCheckedChildren="关" />,
    },
    { title: '对象', width: 90, fixed: 'left', render: (_, row) => <Tag>{formatReportFieldObject(row)}</Tag> },
    { title: '制式', width: 80, render: (_, row) => <Tag>{formatReportFieldTech(row)}</Tag> },
    {
      title: '输出别名',
      dataIndex: 'outputAlias',
      width: 260,
      render: (value: string) => <span className={styles.monoText}>{value}</span>,
    },
    {
      title: '系统字段/指标',
      dataIndex: 'systemField',
      width: 260,
      render: (value: string) => <span className={styles.monoText}>{value}</span>,
    },
    {
      title: '取数字段',
      dataIndex: 'source',
      width: 300,
      render: (value: string) => <span className={styles.monoText}>{value}</span>,
    },
    {
      title: '适用范围',
      dataIndex: 'productClasses',
      width: 180,
      render: (value?: string[]) => <span>{formatProductClasses(value)}</span>,
    },
    {
      title: '类型/口径',
      width: 150,
      render: (_, row) => row.metricType
        ? <Space size={4}>{typeTag(row.metricType)}<Tag>{row.statisType}</Tag></Space>
        : <Tag>{row.dataType}</Tag>,
    },
    {
      title: '单位/渲染',
      width: 130,
      render: (_, row) => row.unit ? <span>{row.unit}</span> : <Tag>{row.renderer}</Tag>,
    },
    { title: '中文名', dataIndex: 'cnName', width: 220, render: (value?: string) => value || '-' },
  ];


  const alarmFieldColumns: ColumnsType<AlarmFieldMappingRow> = [
    { title: '上报', width: 76, fixed: 'left', render: (_, row) => <Switch size="small" checked={row.enabled} disabled checkedChildren="开" unCheckedChildren="关" /> },
    { title: '序号', dataIndex: 'order', width: 70, fixed: 'left' },
    { title: '输出字段', dataIndex: 'field', width: 220, fixed: 'left', render: (value: string) => <span className={styles.monoText}>{value}</span> },
    { title: '字段名称', dataIndex: 'cnName', width: 150 },
    { title: 'xomc 数据源', dataIndex: 'source', width: 320, render: (value: string) => <span className={styles.monoText}>{value}</span> },
    { title: '类型', dataIndex: 'dataType', width: 120, render: (value: string) => <Tag>{value}</Tag> },
  ];

  const readonlyAlarmFieldColumns: ColumnsType<AlarmFieldMappingRow> = [
    { title: '上报', width: 76, fixed: 'left', render: (_, row) => statusTag(row.enabled) },
    { title: '序号', dataIndex: 'order', width: 70, fixed: 'left' },
    { title: '输出字段', dataIndex: 'field', width: 220, fixed: 'left', render: (value: string) => <span className={styles.monoText}>{value}</span> },
    { title: '字段名称', dataIndex: 'cnName', width: 150 },
    { title: 'xomc 数据源', dataIndex: 'source', width: 320, render: (value: string) => <span className={styles.monoText}>{value}</span> },
    { title: '类型', dataIndex: 'dataType', width: 120, render: (value: string) => <Tag>{value}</Tag> },
  ];

  const snmpAlarmFieldColumns: ColumnsType<AlarmFieldMappingRow> = [
    { title: '上报', width: 76, fixed: 'left', render: (_, row) => <Switch size="small" checked={row.enabled} disabled checkedChildren="开" unCheckedChildren="关" /> },
    { title: '序号', dataIndex: 'order', width: 70, fixed: 'left' },
    { title: '输出字段', dataIndex: 'field', width: 220, fixed: 'left', render: (value: string) => <span className={styles.monoText}>{value}</span> },
    { title: 'OID', width: 300, render: (_, row) => <Typography.Text className={styles.monoText} ellipsis={{ tooltip: snmpAlarmOIDForField(row.field) }}>{snmpAlarmOIDForField(row.field)}</Typography.Text> },
    { title: '字段名称', dataIndex: 'cnName', width: 150 },
    { title: 'xomc 数据源', dataIndex: 'source', width: 320, render: (value: string) => <span className={styles.monoText}>{value}</span> },
    { title: '类型', dataIndex: 'dataType', width: 150, render: (value: string) => <Tag>{value}</Tag> },
  ];

  const readonlySnmpAlarmFieldColumns: ColumnsType<AlarmFieldMappingRow> = [
    { title: '上报', width: 76, fixed: 'left', render: (_, row) => statusTag(row.enabled) },
    { title: '序号', dataIndex: 'order', width: 70, fixed: 'left' },
    { title: '输出字段', dataIndex: 'field', width: 220, fixed: 'left', render: (value: string) => <span className={styles.monoText}>{value}</span> },
    { title: 'OID', width: 300, render: (_, row) => <Typography.Text className={styles.monoText} ellipsis={{ tooltip: snmpAlarmOIDForField(row.field) }}>{snmpAlarmOIDForField(row.field)}</Typography.Text> },
    { title: '字段名称', dataIndex: 'cnName', width: 150 },
    { title: 'xomc 数据源', dataIndex: 'source', width: 320, render: (value: string) => <span className={styles.monoText}>{value}</span> },
    { title: '类型', dataIndex: 'dataType', width: 150, render: (value: string) => <Tag>{value}</Tag> },
  ];

  const deliveryTargetTableScroll = { x: 1140 };
  const deliveryTargetEditorTableScroll = { x: 1460 };
  const fileListTableScroll = { x: 1378, y: 'calc(100vh - 260px)' };

  const deliveryTargetColumns: ColumnsType<DeliveryTargetRow> = [
    { title: '启用', dataIndex: 'enabled', width: 76, render: (value: boolean) => <Switch size="small" checked={value} disabled checkedChildren="开" unCheckedChildren="关" /> },
    { title: '目标名称', dataIndex: 'name', width: 150, render: (value: string) => <Typography.Text strong ellipsis>{value}</Typography.Text> },
    { title: '协议', dataIndex: 'protocol', width: 86, render: (value: DeliveryProtocol) => deliveryProtocolTag(value) },
    { title: '地址', width: 180, render: (_, row) => <span className={styles.monoText}>{endpointText(row.host || '-', row.port)}</span> },
    { title: '账号', dataIndex: 'username', width: 140, render: (value: string) => <span className={styles.monoText}>{value || '-'}</span> },
    { title: '密码', width: 150, render: (_, row) => <MaskedCredentialPreview value={row.credential} /> },
    { title: '#FTPRoot#', dataIndex: 'remoteRoot', width: 220, render: (value: string) => <span className={styles.monoText}>{value}</span> },
    { title: '重试/超时', width: 130, render: (_, row) => <Tag>{row.retryTimes} 次 / {row.timeoutSeconds}s</Tag> },
  ];

  const createDeliveryTargetEditorColumns = (
    onPatch: (key: string, patch: Partial<DeliveryTargetRow>) => void,
    onRemove: (key: string) => void,
    scope: DeliveryTargetScope,
    ownerCode: string,
  ): ColumnsType<DeliveryTargetRow> => [
    {
      title: '操作',
      width: 104,
      fixed: 'left',
      render: (_, row) => (
        <Space size={4}>
          <Tooltip title="测试连接">
            <Button
              size="small"
              type="text"
              icon={<PlayCircleOutlined />}
              loading={testingDeliveryTargetKey === deliveryTargetTestKey(scope, ownerCode, row.key)}
              onClick={() => testDeliveryTarget(row, scope, ownerCode)}
            />
          </Tooltip>
          <Popconfirm
            title={nt('确认删除该传输目标？')}
            description={nt('删除后需保存草稿才会生效。')}
            okText={nt('删除')}
            cancelText={nt('取消')}
            okButtonProps={{ danger: true }}
            onConfirm={() => onRemove(row.key)}
          >
            <Tooltip title="删除目标">
              <Button
                aria-label={`删除传输目标 ${row.name || row.key}`}
                danger
                size="small"
                type="text"
                icon={<DeleteOutlined />}
              />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
    {
      title: '启用',
      dataIndex: 'enabled',
      width: 76,
      render: (value: boolean, row) => (
        <Switch
          size="small"
          checked={value}
          checkedChildren="开"
          unCheckedChildren="关"
          onChange={(checked) => onPatch(row.key, { enabled: checked })}
        />
      ),
    },
    {
      title: '目标名称',
      dataIndex: 'name',
      width: 160,
      render: (value: string, row) => (
        <Input value={value} onChange={(event) => onPatch(row.key, { name: event.target.value })} />
      ),
    },
    {
      title: '协议',
      dataIndex: 'protocol',
      width: 104,
      render: (value: DeliveryProtocol, row) => (
        <Select
          value={value}
          style={{ width: 88 }}
          options={deliveryProtocolOptions}
          onChange={(protocol) => onPatch(row.key, {
            protocol,
            port: protocol === 'SFTP' ? 22 : 21,
          })}
        />
      ),
    },
    {
      title: '主机',
      dataIndex: 'host',
      width: 170,
      render: (value: string, row) => {
        const error = deliveryTargetValidationErrors[deliveryTargetTestKey(scope, ownerCode, row.key)]?.host;
        return renderDeliveryTargetEditorField(
          <Input
            value={value}
            status={error ? 'error' : undefined}
            className={styles.monoText}
            placeholder="IP/域名"
            onChange={(event) => onPatch(row.key, { host: event.target.value })}
          />,
          error,
        );
      },
    },
    {
      title: '端口',
      dataIndex: 'port',
      width: 96,
      render: (value: number, row) => (
        <InputNumber value={value} min={1} max={65535} style={{ width: 80 }} onChange={(port) => onPatch(row.key, { port: Number(port ?? 0) })} />
      ),
    },
    {
      title: '用户名',
      dataIndex: 'username',
      width: 150,
      render: (value: string, row) => {
        const error = deliveryTargetValidationErrors[deliveryTargetTestKey(scope, ownerCode, row.key)]?.username;
        return renderDeliveryTargetEditorField(
          <Input
            value={value}
            status={error ? 'error' : undefined}
            className={styles.monoText}
            placeholder="用户名"
            onChange={(event) => onPatch(row.key, { username: event.target.value })}
          />,
          error,
        );
      },
    },
    {
      title: '密码',
      dataIndex: 'credential',
      width: 190,
      render: (value: string, row) => {
        const error = deliveryTargetValidationErrors[deliveryTargetTestKey(scope, ownerCode, row.key)]?.credential;
        return renderDeliveryTargetEditorField(
          <MaskedCredentialInput
            value={value}
            status={error ? 'error' : undefined}
            placeholder={value === storedCredentialText ? '未修改保持原凭据' : '请输入凭据'}
            onChange={(credential) => onPatch(row.key, {
              credential: credential || (value === storedCredentialText ? storedCredentialText : ''),
            })}
          />,
          error,
        );
      },
    },
    {
      title: '#FTPRoot#',
      dataIndex: 'remoteRoot',
      width: 220,
      render: (value: string, row) => (
        <Input value={value} className={styles.monoText} onChange={(event) => onPatch(row.key, { remoteRoot: event.target.value })} />
      ),
    },
    {
      title: '重试',
      dataIndex: 'retryTimes',
      width: 82,
      render: (value: number, row) => (
        <InputNumber value={value} min={0} max={20} style={{ width: 66 }} onChange={(retryTimes) => onPatch(row.key, { retryTimes: Number(retryTimes ?? 0) })} />
      ),
    },
    {
      title: '超时秒',
      dataIndex: 'timeoutSeconds',
      width: 96,
      render: (value: number, row) => (
        <InputNumber value={value} min={1} max={300} style={{ width: 78 }} onChange={(timeoutSeconds) => onPatch(row.key, { timeoutSeconds: Number(timeoutSeconds ?? 1) })} />
      ),
    },
  ];

  const socketAccountColumns: ColumnsType<SocketAccountRow> = [
    { title: '启用', dataIndex: 'enabled', width: 76, render: (value: boolean) => <Switch size="small" checked={value} disabled checkedChildren="开" unCheckedChildren="关" /> },
    { title: '账号用途', dataIndex: 'channel', width: 150 },
    { title: '用户名', dataIndex: 'username', width: 160, render: (value: string) => <span className={styles.monoText}>{value || '-'}</span> },
    { title: '类型', dataIndex: 'type', width: 90, render: (value: string) => <Tag>{value}</Tag> },
    { title: '密码', dataIndex: 'credential', width: 180, render: (value: string) => <MaskedCredentialPreview value={value} width={160} /> },
    { title: '能力范围', dataIndex: 'purpose', width: 220 },
  ];

  const socketAccountEditorColumns: ColumnsType<SocketAccountRow> = [
    {
      title: '启用',
      dataIndex: 'enabled',
      width: 76,
      fixed: 'left',
      render: (value: boolean, row) => (
        <Switch
          size="small"
          checked={value}
          checkedChildren="开"
          unCheckedChildren="关"
          onChange={(checked) => {
            if (!socketEditor) return;
            updateSocketEditorAccount(row.key, { enabled: checked });
          }}
        />
      ),
    },
    {
      title: '类型',
      dataIndex: 'type',
      width: 190,
      render: (value: SocketAccountRow['type'], row) => (
        <Select
          value={value}
          style={{ width: '100%' }}
          options={socketAccountTypeOptions.filter((option) => socketEditor?.profile === 'CUCC' || option.value === 'msg')}
          onChange={(nextType: SocketAccountRow['type']) => {
            if (!socketEditor) return;
            updateSocketEditorAccount(row.key, {
              type: nextType,
              channel: nextType === 'ftp' ? '文件同步账号' : '实时/消息同步账号',
              purpose: getSocketAccountPurpose(nextType),
            });
          }}
        />
      ),
    },
    {
      title: '用户名',
      dataIndex: 'username',
      width: 180,
      render: (value: string, row) => (
        <Input
          value={value}
          placeholder="请输入用户名"
          onChange={(event) => {
            if (!socketEditor) return;
            updateSocketEditorAccount(row.key, { username: event.target.value });
          }}
        />
      ),
    },
    {
      title: '密码',
      dataIndex: 'credential',
      width: 220,
      render: (value: string, row) => (
        <MaskedCredentialInput
          value={value}
          placeholder={socketCredentialPlaceholder(value)}
          onChange={(credential) => {
            if (!socketEditor) return;
            updateSocketEditorAccount(row.key, {
              credential: credential || (value === storedCredentialText ? storedCredentialText : ''),
            });
          }}
        />
      ),
    },
    { title: '能力范围', dataIndex: 'purpose', width: 220 },
    {
      title: '操作',
      width: 72,
      fixed: 'right',
      render: (_, row) => (
        <Tooltip title="删除账号">
          <Button
            danger
            size="small"
            type="text"
            icon={<DeleteOutlined />}
            disabled={!socketEditor || socketEditorAccounts.length <= 1}
            onClick={() => {
              if (!socketEditor) return;
              removeSocketEditorAccount(row.key);
            }}
          />
        </Tooltip>
      ),
    },
  ];

  const socketColumns: ColumnsType<SocketAlarmConfigRow> = [
    {
      title: '操作',
      width: 104,
      fixed: 'left',
      render: (_, row) => (
        <div className={styles.rowControl}>
          <Switch
            size="small"
            checked={socketEnabled[row.key]}
            checkedChildren="开"
            unCheckedChildren="关"
            onClick={(_, event) => event.stopPropagation()}
            onChange={(checked) => persistSocketEnabled(row, checked)}
          />
          <Dropdown
            trigger={['click']}
            menu={{
              items: [
                { key: 'view', icon: <EyeOutlined />, label: '查看' },
                { key: 'edit', icon: <EditOutlined />, label: '编辑' },
                { key: 'report', icon: <FileSearchOutlined />, label: '上报结果' },
                { key: 'test', icon: <PlayCircleOutlined />, label: '生成样例' },
              ],
              onClick: ({ key, domEvent }) => {
                domEvent.stopPropagation();
                if (key === 'view') setSelectedSocket(row);
                if (key === 'edit') openSocketEditor(row);
                if (key === 'report') showLatestEventReport('socket', row.key, effectiveReportStatus(buildSocketReportStatus(row), Boolean(socketEnabled[row.key])));
                if (key === 'test') testSocketAlarm(row);
              },
            }}
          >
            <Button
              aria-label={`更多操作 ${row.name}`}
              type="text"
              size="small"
              icon={<MoreOutlined />}
              onClick={(event) => event.stopPropagation()}
            />
          </Dropdown>
        </div>
      ),
    },
    { title: '配置名称', dataIndex: 'name', width: 176, fixed: 'left', render: (value: string) => <Typography.Text strong ellipsis>{value}</Typography.Text> },
    { title: '协议', width: 120, render: (_, row) => <Space size={4}>{socketProfileTag(row.profile, nt)}<Tag>{row.encoding}</Tag></Space> },
    { title: '监听地址', width: 170, render: (_, row) => <span className={styles.monoText}>{endpointText(row.listenIp, row.listenPort)}</span> },
    {
      title: '服务能力',
      width: 280,
      render: (_, row) => (
        <Space size={4} wrap>
          {row.realTimeEnabled ? <Tag color="green">实时推送</Tag> : <Tag>实时关闭</Tag>}
          {row.historyEnabled ? <Tag color="blue">同步查询</Tag> : <Tag>无同步</Tag>}
          {row.profile === 'CUCC' && row.historyEnabled ? <Tag color="purple">文件补录</Tag> : null}
        </Space>
      ),
    },
    {
      title: '会话与心跳',
      width: 190,
      render: (_, row) => (
        <Space size={4} wrap>
          <Tag>{row.maxClients} 连接</Tag>
          <Tag>{row.heartbeatPeriod}s</Tag>
          <Tag>{row.heartbeatTimes} 次超时</Tag>
        </Space>
      ),
    },
    {
      title: '账号',
      width: 150,
      render: (_, row) => {
        const accounts = getSocketAccounts(row);
        const typeCounts = accounts.reduce<Record<string, number>>((acc, account) => {
          if (!account.enabled) return acc;
          acc[account.type] = (acc[account.type] ?? 0) + 1;
          return acc;
        }, {});
        return (
          <Space size={4}>
            {Object.entries(typeCounts).map(([type, count]) => (
              <Tag key={type}>{type}{count > 1 ? ` x${count}` : ''}</Tag>
            ))}
          </Space>
        );
      },
    },
    { title: '字段模板', width: 110, render: (_, row) => <Tag>{getSocketFields(row.profile).length} 字段</Tag> },
  ];

  const snmpColumns: ColumnsType<SnmpAlarmTargetRow> = [
    {
      title: '操作',
      width: 104,
      fixed: 'left',
      render: (_, row) => (
        <div className={styles.rowControl}>
          <Switch
            size="small"
            checked={snmpEnabled[row.key]}
            checkedChildren="开"
            unCheckedChildren="关"
            loading={Boolean(snmpSaving[row.key])}
            onClick={(_, event) => event.stopPropagation()}
            onChange={(checked) => persistSnmpEnabled(row, checked)}
          />
          <Dropdown
            trigger={['click']}
            menu={{
              items: [
                { key: 'view', icon: <EyeOutlined />, label: '查看' },
                { key: 'edit', icon: <EditOutlined />, label: '编辑' },
                { key: 'report', icon: <FileSearchOutlined />, label: '上报结果' },
                { key: 'test', icon: <PlayCircleOutlined />, label: '测试发送' },
              ],
              onClick: ({ key, domEvent }) => {
                domEvent.stopPropagation();
                if (key === 'view') setSelectedSnmp(row);
                if (key === 'edit') setSnmpEditor(row);
                if (key === 'report') showLatestEventReport('snmp', row.key, effectiveReportStatus(buildSnmpReportStatus(row), Boolean(snmpEnabled[row.key])));
                if (key === 'test') testSnmpAlarm(row);
              },
            }}
          >
            <Button
              aria-label={`更多操作 ${snmpVersionLabel(row.version)}`}
              type="text"
              size="small"
              icon={<MoreOutlined />}
              onClick={(event) => event.stopPropagation()}
            />
          </Dropdown>
        </div>
      ),
    },
    {
      title: '版本',
      width: 108,
      fixed: 'left',
      render: (_, row) => snmpVersionTag(row.version),
    },
    { title: '通知', width: 100, render: (_, row) => snmpNotificationTag(row.notificationType) },
    {
      title: 'Agent 监听',
      width: 170,
      render: (_, row) => row.mibQueryEnabled
        ? <span className={styles.monoText}>{endpointText(row.listenIp, row.listenPort)}</span>
        : <Tag>未开放查询</Tag>,
    },
    {
      title: '通知目标',
      width: 170,
      render: (_, row) => snmpEnabled[row.key]
        ? <span className={styles.monoText}>{endpointText(row.targetHost, row.targetPort)}</span>
        : <Tag>未启用上报</Tag>,
    },
    {
      title: '查询/清除策略',
      width: 220,
      render: (_, row) => (
        <Space size={4} wrap>
          {row.mibQueryEnabled ? <Tag color="blue">允许 MIB 查询</Tag> : <Tag>未开放查询</Tag>}
          <Tag>{row.clearSeverityPolicy}</Tag>
        </Space>
      ),
    },
    {
      title: '重试',
      width: 120,
      render: (_, row) => row.notificationType === 'Inform' ? <Tag>{row.timeoutSeconds}s / {row.retries} 次</Tag> : <span>-</span>,
    },
  ];

  const fileTab = (
    <div className={styles.tabContent}>
      <div className={styles.fileToolbar}>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreateEditor}>
          新增配置
        </Button>
      </div>
      <Table<ScenarioRow>
        className={`${styles.compactScenarioTable} ${styles.mainListTable}`}
        columns={scenarioColumns}
        dataSource={fileProfiles}
        rowKey="code"
        size="small"
        loading={pageConfigLoading}
        pagination={renderTablePagination('刷新北向文件配置', 20)}
        scroll={fileListTableScroll}
        rowClassName={(row) => (selectedScenario?.code === row.code ? styles.selectedRow : '')}
        onRow={(row) => ({ onClick: () => openViewDrawer(row) })}
      />
    </div>
  );

  const inventoryTab = (
    <div className={styles.tabContent}>
      <Table<InventoryConfigRow>
        className={styles.compactScenarioTable}
        columns={inventoryConfigColumns}
        dataSource={inventoryConfigs}
        rowKey="key"
        size="small"
        loading={pageConfigLoading}
        pagination={renderTablePagination('刷新 Inventory 文件')}
        scroll={{ x: 1122, y: 560 }}
        rowClassName={(row) => (row.key === selectedInventoryType ? styles.selectedRow : '')}
        onRow={(row) => ({ onClick: () => openInventoryView(row) })}
      />
    </div>
  );

  const apiTab = (
    <div className={styles.tabContent}>
      <div className={styles.apiToolbar}>
        <Space>
          <Switch
            checked={apiGloballyEnabled}
            loading={apiSwitchSaving}
            checkedChildren="开"
            unCheckedChildren="关"
            onChange={persistAllApiEnabled}
          />
          <Typography.Text strong>北向 API 总开关</Typography.Text>
          <Tag color={apiGloballyEnabled ? 'success' : apiPartiallyEnabled ? 'warning' : 'default'}>
            {apiGloballyEnabled ? '已启用' : apiPartiallyEnabled ? '部分开启' : '已停用'}
          </Tag>
        </Space>
        <Space>
          <Button
            size="small"
            icon={<FileSearchOutlined />}
            onClick={() => setApiCatalogOpen(true)}
          >
            查看接口清单
          </Button>
          <Button
            size="small"
            icon={<ReloadOutlined />}
            loading={pageConfigLoading}
            onClick={() => loadPageConfig(false)}
          >
            刷新
          </Button>
        </Space>
      </div>
      <div className={styles.editorSection}>
        <div className={styles.editorSectionHeader}>
          <Space>
            <SafetyCertificateOutlined />
            <Typography.Text strong>北向 API 用户</Typography.Text>
          </Space>
          <Space>
            <Button size="small" icon={<PlusOutlined />} onClick={addApiUser}>
              新增
            </Button>
            <Button
              size="small"
              type="primary"
              loading={apiUserSaving}
              onClick={saveApiUsers}
            >
              保存
            </Button>
          </Space>
        </div>
        <Table<ApiUserRow>
          columns={apiUserColumns}
          dataSource={apiUsers}
          rowKey="key"
          size="small"
          pagination={false}
          scroll={{ x: 720, y: 260 }}
          locale={{ emptyText: '未配置北向 API 用户，外部系统不能调用北向 API' }}
        />
      </div>
    </div>
  );

  const socketTab = (
    <div className={styles.tabContent}>
      <Table<SocketAlarmConfigRow>
        className={styles.compactScenarioTable}
        columns={socketColumns}
        dataSource={socketConfigs}
        rowKey="key"
        size="small"
        pagination={renderTablePagination('刷新 Socket 告警')}
        scroll={{ x: 1388, y: 560 }}
        rowClassName={(row) => (selectedSocket?.key === row.key || socketEditor?.key === row.key ? styles.selectedRow : '')}
        onRow={(row) => ({ onClick: () => setSelectedSocket(row) })}
      />
    </div>
  );

  const snmpTab = (
    <div className={styles.tabContent}>
      <Table<SnmpAlarmTargetRow>
        className={styles.compactScenarioTable}
        columns={snmpColumns}
        dataSource={snmpTargets}
        rowKey="key"
        size="small"
        pagination={renderTablePagination('刷新 SNMP 告警')}
        scroll={{ x: 1180, y: 560 }}
        rowClassName={(row) => (selectedSnmp?.key === row.key || snmpEditor?.key === row.key ? styles.selectedRow : '')}
        onRow={(row) => ({ onClick: () => setSelectedSnmp(row) })}
      />
    </div>
  );

  const viewInventoryPreview = viewInventoryConfig ? getInventoryTemplatePreview(viewInventoryConfig) : null;
  const selectedInventoryPreview = selectedInventoryConfig ? getInventoryTemplatePreview(selectedInventoryConfig) : null;
  const viewInventoryDeliveryTargets = viewInventoryConfig ? getInventoryDeliveryTargets(viewInventoryConfig.key) : [];
  const selectedInventoryDeliveryTargets = selectedInventoryConfig ? getInventoryDeliveryTargets(selectedInventoryConfig.key) : [];

  const reportRunColumns: ColumnsType<NorthboundFileRun> = [
    { title: '业务域', dataIndex: 'domain', width: 72, render: (v: string) => v || '-' },
    { title: '对象', dataIndex: 'object_code', width: 96, render: (v: string) => v || '-' },
    {
      title: '状态',
      dataIndex: 'status',
      width: 112,
      render: (_, run) => {
        const { state, label, tooltip } = reportRunDisplayState(run, reportDeliveryEventsByRunId);
        return reportStateTag(state, label, tooltip);
      },
    },
    { title: '最近时间', dataIndex: 'created_at', width: 160, render: (v: string) => formatRunTime(v) },
    { title: '文件名', dataIndex: 'artifact_name', ellipsis: true, render: (v: string) => v || '-' },
    { title: '大小', dataIndex: 'artifact_size', width: 88, render: (v: number) => formatBytes(v) },
  ];
  const reportEventColumns: ColumnsType<NorthboundPageConfigEvent> = [
    { title: '时间', dataIndex: 'created_at', width: 160, render: (value: string) => formatRunTime(value) },
    { title: '事件', dataIndex: 'event_type', width: 110, render: (_, event) => <Tag>{eventTypeLabel(event)}</Tag> },
    { title: '状态', dataIndex: 'status', width: 88, render: (_, event) => reportStateTag(reportStateFromEvent(event), eventStatusText(event)) },
    {
      title: '目标/内容',
      width: 220,
      render: (_, event) => (
        <div className={styles.summaryCell}>
          <Typography.Text ellipsis={{ tooltip: event.artifact_name || '-' }}>{event.artifact_name || '-'}</Typography.Text>
          <Typography.Text type="secondary" ellipsis={{ tooltip: event.artifact_path || '-' }}>
            {event.artifact_path || '-'}
          </Typography.Text>
        </div>
      ),
    },
    {
      title: '摘要',
      render: (_, event) => {
        const summary = eventListSummary(event);
        return <Typography.Text ellipsis={{ tooltip: summary }}>{summary}</Typography.Text>;
      },
    },
  ];
  const reportRunDisplayStates = reportRunList.map((item) => reportRunDisplayState(item, reportDeliveryEventsByRunId).state);
  const reportSuccessCount = reportRunDisplayStates.filter((state) => state === 'success').length;
  const reportFailedCount = reportRunDisplayStates.filter((state) => state === 'failed').length;
  const reportRunningCount = reportRunDisplayStates.filter((state) => state === 'running').length;
  const reportSummary = reportRunList.length > 1
    ? `共 ${reportRunList.length} 个对象：成功 ${reportSuccessCount} · 失败 ${reportFailedCount}${reportRunningCount > 0 ? ` · 生成中 ${reportRunningCount}` : ''}`
    : '';
  const selectedApiMeta = selectedApi ? getApiMeta(selectedApi) : null;
  const selectedApiResponseFields = selectedApiMeta?.responseFields ?? [];
  const selectedApiDisplayResponseFields = useMemo(
    () => apiFieldDisplayList(selectedApiResponseFields),
    [selectedApiResponseFields],
  );
  const deliveryTestDisplay = deliveryTestResult
    ? buildDeliveryTestDisplay(deliveryTestResult.event)
    : null;

  return (
    <NorthboundI18nScope>
    <div className={styles.page}>
      <ListPageLayout>
        <Tabs
          className={styles.tabs}
          defaultActiveKey="file"
          items={[
            { key: 'file', label: <span><FileTextOutlined />北向文件配置</span>, children: fileTab },
            { key: 'inventory', label: <span><DatabaseOutlined />Inventory 文件</span>, children: inventoryTab },
            { key: 'socket', label: <span><WifiOutlined />Socket 告警</span>, children: socketTab },
            { key: 'snmp', label: <span><SafetyCertificateOutlined />SNMP 告警</span>, children: snmpTab },
            { key: 'api', label: <span><ApiOutlined />北向 API</span>, children: apiTab },
          ]}
        />
      </ListPageLayout>

      <Modal
        title={deliveryTestResult ? `${deliveryTestResult.title} 测试结果` : '传输目标测试结果'}
        open={Boolean(deliveryTestResult)}
        onCancel={() => setDeliveryTestResult(null)}
        width={480}
        className={styles.deliveryTestModal}
        destroyOnClose
        footer={(
          <Button type="primary" onClick={() => setDeliveryTestResult(null)}>
            知道了
          </Button>
        )}
      >
        {deliveryTestDisplay && (
          <div className={styles.deliveryTestPanel}>
            <div
              className={[
                styles.deliveryTestStatus,
                deliveryTestDisplay.state === 'success'
                  ? styles.deliveryTestStatusSuccess
                  : styles.deliveryTestStatusFailed,
              ].join(' ')}
            >
              <span className={styles.deliveryTestStatusIcon}>
                {deliveryTestDisplay.state === 'success'
                  ? <CheckCircleOutlined />
                  : <CloseCircleOutlined />}
              </span>
              <div className={styles.deliveryTestStatusText}>
                <Typography.Text strong className={styles.deliveryTestStatusTitle}>
                  {deliveryTestDisplay.statusText}
                </Typography.Text>
                <Typography.Text type="secondary" className={styles.deliveryTestStatusSummary}>
                  {deliveryTestDisplay.state === 'success' ? '传输目标连接正常' : '传输目标连接未通过'}
                </Typography.Text>
              </div>
            </div>
            <div className={styles.deliveryTestInfoGrid}>
              <div className={`${styles.deliveryTestInfoItem} ${styles.deliveryTestInfoWide}`}>
                <span className={styles.deliveryTestLabel}>目标地址</span>
                <Typography.Text
                  className={`${styles.deliveryTestValue} ${styles.deliveryTestMono}`}
                  ellipsis={{ tooltip: deliveryTestDisplay.target }}
                >
                  {deliveryTestDisplay.target}
                </Typography.Text>
              </div>
              <div className={styles.deliveryTestInfoItem}>
                <span className={styles.deliveryTestLabel}>网络连接</span>
                <span className={styles.deliveryTestValue}>{deliveryTestDisplay.tcpText}</span>
              </div>
              <div className={styles.deliveryTestInfoItem}>
                <span className={styles.deliveryTestLabel}>账号认证</span>
                <span className={styles.deliveryTestValue}>{deliveryTestDisplay.authText}</span>
              </div>
            </div>
            <div className={styles.deliveryTestMessage}>
              <span className={styles.deliveryTestLabel}>
                {deliveryTestDisplay.state === 'success' ? '结果说明' : '失败信息'}
              </span>
              <Typography.Paragraph
                className={[
                  styles.deliveryTestMessageText,
                  deliveryTestDisplay.state === 'failed' ? styles.deliveryTestFailureText : '',
                ].join(' ')}
                type={deliveryTestDisplay.state === 'failed' ? 'danger' : undefined}
              >
                {deliveryTestDisplay.message}
              </Typography.Paragraph>
            </div>
          </div>
        )}
      </Modal>

      <Drawer
        title={
          selectedReportStatus?.resultTitle
            || (reportCapabilityName || selectedReportStatus?.capabilityName
              ? `${reportCapabilityName || selectedReportStatus?.capabilityName} 上报结果`
              : '上报结果')
        }
        open={Boolean(selectedReportStatus)}
        onClose={() => {
          setSelectedReportStatus(null);
          setReportRunList([]);
          setReportEventList([]);
          setReportEventTotal(0);
          setReportEventLoading(false);
          setReportCapabilityName('');
          setReportDeliveryEventsByRunId({});
        }}
        size="large"
        rootClassName={styles.inventoryDrawer}
        destroyOnClose
        extra={selectedReportStatus ? (
          <Space>
            {selectedReportStatus.artifactType === 'message' && (
              <Button
                icon={<CopyOutlined />}
                onClick={() => copyReportPayload(selectedReportStatus)}
              >
                {selectedReportStatus.copyLabel ?? '复制报文'}
              </Button>
            )}
            {selectedReportStatus.artifactType === 'file' && (
              <Button
                type="primary"
                icon={<DownloadOutlined />}
                onClick={() => {
                  void downloadReportArtifact(selectedReportStatus).catch(() => {
                    void message.error(nt('上报文件下载失败'));
                  });
                }}
              >
                下载最新文件
              </Button>
            )}
          </Space>
        ) : undefined}
      >
        {selectedReportStatus && (
          <Space orientation="vertical" size={16} className={styles.drawerBody}>
            <div className={styles.editorSection}>
              <Descriptions bordered size="small" column={2}>
                <Descriptions.Item label="状态">{reportStateTag(selectedReportStatus.state, selectedReportStatus.statusText)}</Descriptions.Item>
                <Descriptions.Item label="最近时间">{selectedReportStatus.lastTime || '-'}</Descriptions.Item>
                <Descriptions.Item label="目标/地址" span={2}>
                  <Typography.Text ellipsis={{ tooltip: selectedReportStatus.artifactPath }}>
                    {selectedReportStatus.artifactPath}
                  </Typography.Text>
                </Descriptions.Item>
                <Descriptions.Item label="说明" span={2}>{selectedReportStatus.detail}</Descriptions.Item>
              </Descriptions>
            </div>
            {(reportEventLoading || reportEventList.length > 0) && (
              <div className={styles.editorSection}>
                <div className={styles.editorSectionHeader}>
                  <Typography.Text strong>最近事件</Typography.Text>
                  <Typography.Text type="secondary">
                    共 {reportEventTotal} 条，显示最近 {reportEventList.length} 条
                  </Typography.Text>
                </div>
                <Table<NorthboundPageConfigEvent>
                  className={styles.compactScenarioTable}
                  columns={reportEventColumns}
                  dataSource={reportEventList}
                  rowKey="id"
                  size="small"
                  loading={reportEventLoading}
                  pagination={{ pageSize: 10, showSizeChanger: false, hideOnSinglePage: reportEventList.length <= 10 }}
                  scroll={{ x: 980, y: 280 }}
                  rowClassName={(row) => (selectedReportStatus?.key === `event:${row.id}` ? styles.selectedRow : '')}
                  onRow={(row) => ({
                    onClick: () => selectReportEvent(row, reportCapabilityName || selectedReportStatus?.capabilityName || ''),
                  })}
                />
              </div>
            )}
            {reportRunList.length > 1 && (
              <div className={styles.editorSection}>
                <div className={styles.editorSectionHeader}>
                  <Typography.Text strong>对象上报结果</Typography.Text>
                </div>
                <Table<NorthboundFileRun>
                  className={styles.compactScenarioTable}
                  columns={reportRunColumns}
                  dataSource={reportRunList}
                  rowKey="id"
                  size="small"
                  pagination={false}
                  scroll={{ y: 260 }}
                  title={() => <Typography.Text type="secondary">{reportSummary}</Typography.Text>}
                  rowClassName={(row) => (selectedReportStatus?.runId === row.id ? styles.selectedRow : '')}
                  onRow={(row) => ({
                    onClick: () => {
                      void resolveRunReportStatus(
                        row,
                        reportCapabilityName || selectedReportStatus?.capabilityName || '',
                        reportDeliveryEventsByRunId[row.id] ?? [],
                      )
                        .then(setSelectedReportStatus);
                    },
                  })}
                />
              </div>
            )}
            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>
                  {selectedReportStatus.previewTitle
                    ?? (selectedReportStatus.artifactType === 'file' ? '文件内容预览' : '上报报文')}
                </Typography.Text>
                {reportStateTag(selectedReportStatus.state, selectedReportStatus.statusText)}
              </div>
              {selectedReportStatus.deliveryNote && (
                <Typography.Text type="secondary" style={{ display: 'block', marginBottom: 8 }}>
                  {selectedReportStatus.deliveryNote}
                </Typography.Text>
              )}
	              {selectedReportStatus.artifactType === 'file' && /\.csv(?:\.(?:zip|gz))?$/i.test(selectedReportStatus.artifactName) ? (
	                <CsvReportPreview content={selectedReportStatus.payload} />
	              ) : selectedReportStatus.artifactType === 'message' ? (
	                <MessageReportPreview content={selectedReportStatus.payload} />
	              ) : (
	                <pre className={styles.codeBlock}>{selectedReportStatus.payload}</pre>
	              )}
            </div>
          </Space>
        )}
      </Drawer>

      <Drawer
        title="北向 API 接口清单"
        open={apiCatalogOpen}
        onClose={() => setApiCatalogOpen(false)}
        size="large"
        rootClassName={styles.inventoryDrawer}
        destroyOnClose
        extra={(
          <Button
            size="small"
            icon={<ReloadOutlined />}
            loading={pageConfigLoading}
            onClick={() => loadPageConfig(false)}
          >
            刷新
          </Button>
        )}
      >
        <Table<NorthboundApiRow>
          className={styles.compactScenarioTable}
          columns={apiColumns}
          dataSource={apiRows}
          rowKey="key"
          size="small"
          pagination={renderTablePagination('刷新北向 API')}
          scroll={{ x: 1100, y: 560 }}
          rowClassName={(row) => (selectedApi?.key === row.key ? styles.selectedRow : '')}
          onRow={(row) => ({ onClick: () => setSelectedApi(row) })}
        />
      </Drawer>

      <Drawer
        title={selectedApi ? `${selectedApi.name} API` : '北向 API'}
        open={Boolean(selectedApi)}
        onClose={() => setSelectedApi(null)}
        size="large"
        rootClassName={styles.inventoryDrawer}
        destroyOnClose
      >
        {selectedApi && (
          <Space orientation="vertical" size={16} className={styles.drawerBody}>
            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>基础信息</Typography.Text>
              </div>
              <Descriptions bordered size="small" column={2}>
                <Descriptions.Item label="接口名称">{selectedApi.name}</Descriptions.Item>
                <Descriptions.Item label="业务模块">{apiModuleTag(selectedApi)}</Descriptions.Item>
                <Descriptions.Item label="方法">{apiMethodTag(selectedApi.method)}</Descriptions.Item>
                <Descriptions.Item label="接口类型">{apiKindTag(selectedApiMeta?.apiKind)}</Descriptions.Item>
                <Descriptions.Item label="总开关状态">{statusTag(Boolean(apiEnabled[apiConfigKey(selectedApi)]))}</Descriptions.Item>
                <Descriptions.Item label="认证方式">{normalizeLegacyApiText(selectedApi.auth)}</Descriptions.Item>
                <Descriptions.Item label="返回字段">{selectedApiMeta?.fieldContract ?? '-'}</Descriptions.Item>
                <Descriptions.Item label="字段数量">{selectedApiDisplayResponseFields.length} 项</Descriptions.Item>
                <Descriptions.Item label="接口 URL" span={2}>
                  <span className={styles.monoText}>{selectedApi.url}</span>
                </Descriptions.Item>
                <Descriptions.Item label="接口说明" span={2}>
                  {apiUsageSummary(selectedApi)}
                </Descriptions.Item>
              </Descriptions>
            </div>

            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>返回字段清单</Typography.Text>
                <Typography.Text type="secondary">共 {selectedApiDisplayResponseFields.length} 项</Typography.Text>
              </div>
              <div className={styles.apiFieldList}>
                {selectedApiDisplayResponseFields.map((field) => (
                  <Tag key={field} className={styles.apiFieldTag}>
                    {field}
                  </Tag>
                ))}
              </div>
            </div>

            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>请求参数示例</Typography.Text>
              </div>
              <pre className={styles.codeBlock}>{normalizeLegacyApiText(selectedApi.requestExample)}</pre>
            </div>

            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>返回结果示例</Typography.Text>
              </div>
              <pre className={styles.codeBlock}>{selectedApi.responseExample}</pre>
            </div>
          </Space>
        )}
      </Drawer>

      <Drawer
        title={selectedSocket ? selectedSocket.name : 'Socket 告警'}
        open={Boolean(selectedSocket)}
        onClose={() => setSelectedSocket(null)}
        size="large"
        rootClassName={styles.inventoryDrawer}
        destroyOnClose
        extra={selectedSocket ? (
          <Space>
            <Switch
              checked={socketEnabled[selectedSocket.key]}
              checkedChildren="开"
              unCheckedChildren="关"
              onChange={(checked) => persistSocketEnabled(selectedSocket, checked)}
            />
            <Button
              icon={<EditOutlined />}
	              onClick={() => {
	                const socketConfig = selectedSocket;
	                setSelectedSocket(null);
	                openSocketEditor(socketConfig);
	              }}
            >
              编辑
            </Button>
          </Space>
        ) : undefined}
      >
        {selectedSocket && (
          <Space orientation="vertical" size={16} className={styles.drawerBody}>
            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>基础信息</Typography.Text>
              </div>
              <Descriptions bordered size="small" column={2}>
                <Descriptions.Item label="协议场景">{socketProfileTag(selectedSocket.profile, nt)}</Descriptions.Item>
                <Descriptions.Item label="启用配置">{statusTag(Boolean(socketEnabled[selectedSocket.key]))}</Descriptions.Item>
                <Descriptions.Item label="监听地址">
                  <span className={styles.monoText}>{endpointText(selectedSocket.listenIp, selectedSocket.listenPort)}</span>
                </Descriptions.Item>
                <Descriptions.Item label="最大连接">{selectedSocket.maxClients}</Descriptions.Item>
                <Descriptions.Item label="心跳周期">{selectedSocket.heartbeatPeriod} 秒</Descriptions.Item>
                <Descriptions.Item label="超时阈值">{selectedSocket.heartbeatTimes} 次</Descriptions.Item>
                <Descriptions.Item label="协议帧" span={2}>{selectedSocket.frame}</Descriptions.Item>
                <Descriptions.Item label="实时推送">{selectedSocket.realTimeEnabled ? '开启' : '关闭'}</Descriptions.Item>
                <Descriptions.Item label="同步查询">{selectedSocket.historyEnabled ? selectedSocket.syncMode : '关闭'}</Descriptions.Item>
                <Descriptions.Item label="序号策略">{selectedSocket.sequencePolicy}</Descriptions.Item>
                <Descriptions.Item label="字符编码">{selectedSocket.encoding}</Descriptions.Item>
              </Descriptions>
            </div>

            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>账号管理</Typography.Text>
              </div>
              <Table<SocketAccountRow>
                columns={socketAccountColumns}
                dataSource={getSocketAccounts(selectedSocket)}
                rowKey="key"
                size="small"
                pagination={false}
                scroll={{ x: 900 }}
              />
            </div>

            {selectedSocket.profile === 'CUCC' && (
              <div className={styles.editorSection}>
                <div className={styles.editorSectionHeader}>
                  <Typography.Text strong>文件同步传输目标</Typography.Text>
                  <Typography.Text type="secondary">{enabledDeliverySummary(getSocketDeliveryTargets(selectedSocket))}</Typography.Text>
                </div>
                <Table<DeliveryTargetRow>
                  columns={deliveryTargetColumns}
                  dataSource={getSocketDeliveryTargets(selectedSocket)}
                  rowKey="key"
                  size="small"
                  pagination={false}
                  scroll={deliveryTargetTableScroll}
                />
              </div>
            )}

            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>告警字段映射</Typography.Text>
              </div>
              <Table<AlarmFieldMappingRow>
                columns={readonlyAlarmFieldColumns}
                dataSource={getSocketFields(selectedSocket.profile)}
                rowKey="key"
                size="small"
                pagination={false}
                scroll={{ x: 960, y: 360 }}
              />
            </div>
          </Space>
        )}
      </Drawer>

      <Drawer
        title={socketEditor ? `编辑 ${socketEditor.name}` : '编辑 Socket 告警'}
        open={Boolean(socketEditor)}
        onClose={closeSocketEditor}
        size="large"
        rootClassName={styles.inventoryDrawer}
        destroyOnClose
        extra={socketEditor ? (
          <Button
            type="primary"
            onClick={() => saveSocketEditor(socketEditor)}
          >
            保存
          </Button>
        ) : undefined}
      >
        {socketEditor && (
          <Space orientation="vertical" size={16} className={styles.drawerBody}>
            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>服务参数</Typography.Text>
              </div>
              <Form layout="vertical" className={styles.compactForm}>
	                <div className={styles.inventoryFormGrid}>
	                  <Form.Item label="协议场景">
	                    <Space size={4} wrap>
	                      {socketProfileTag(socketEditor.profile, nt)}
	                      <Tag>{socketEditor.encoding}</Tag>
	                    </Space>
	                  </Form.Item>
	                  <Form.Item label="启用配置">
	                    <Switch
	                      checked={socketEditorEnabled}
	                      checkedChildren="开"
	                      unCheckedChildren="关"
	                      onChange={setSocketEditorEnabled}
	                    />
                  </Form.Item>
                  <Form.Item label="监听地址">
                    <Input value={socketEditor.listenIp} onChange={(event) => setSocketEditor((current) => (current ? { ...current, listenIp: event.target.value } : current))} />
                  </Form.Item>
                  <Form.Item label="监听端口">
                    <InputNumber value={socketEditor.listenPort} min={1} max={65535} style={{ width: '100%' }} onChange={(listenPort) => setSocketEditor((current) => (current ? { ...current, listenPort: Number(listenPort ?? 1) } : current))} />
                  </Form.Item>
                  <Form.Item label="最大连接数">
                    <InputNumber value={socketEditor.maxClients} min={1} max={10000} style={{ width: '100%' }} onChange={(maxClients) => setSocketEditor((current) => (current ? { ...current, maxClients: Number(maxClients ?? 1) } : current))} />
                  </Form.Item>
                  <Form.Item label="心跳周期（秒）">
                    <InputNumber value={socketEditor.heartbeatPeriod} min={5} max={3600} style={{ width: '100%' }} onChange={(heartbeatPeriod) => setSocketEditor((current) => (current ? { ...current, heartbeatPeriod: Number(heartbeatPeriod ?? 5) } : current))} />
                  </Form.Item>
                  <Form.Item label="超时阈值（次）">
                    <InputNumber value={socketEditor.heartbeatTimes} min={1} max={100} style={{ width: '100%' }} onChange={(heartbeatTimes) => setSocketEditor((current) => (current ? { ...current, heartbeatTimes: Number(heartbeatTimes ?? 1) } : current))} />
                  </Form.Item>
                </div>
              </Form>
            </div>

            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>同步策略</Typography.Text>
              </div>
              <Form layout="vertical" className={styles.compactForm}>
                <div className={styles.inventoryFormGrid}>
                  <Form.Item label="实时推送">
                    <Switch checked={socketEditor.realTimeEnabled} checkedChildren="开" unCheckedChildren="关" onChange={(realTimeEnabled) => setSocketEditor((current) => (current ? { ...current, realTimeEnabled } : current))} />
                  </Form.Item>
                  <Form.Item label="同步查询">
                    <Switch checked={socketEditor.historyEnabled} checkedChildren="开" unCheckedChildren="关" onChange={(historyEnabled) => setSocketEditor((current) => (current ? { ...current, historyEnabled } : current))} />
                  </Form.Item>
                  <Form.Item label="同步能力">
                    <Space size={4} wrap>
                      <Tag color={socketEditor.historyEnabled ? 'blue' : undefined}>消息同步</Tag>
                      {socketEditor.profile === 'CUCC' ? (
                        <Tag color={socketEditor.historyEnabled ? 'purple' : undefined}>文件补录</Tag>
                      ) : (
                        <Tag>不支持文件补录</Tag>
                      )}
                    </Space>
                  </Form.Item>
                  <Form.Item label="序号字段">
                    <span className={styles.monoText}>{socketEditor.sequencePolicy}</span>
                  </Form.Item>
                </div>
              </Form>
            </div>

            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>账号管理</Typography.Text>
                <Button size="small" icon={<PlusOutlined />} onClick={() => addSocketEditorAccount(socketEditor)}>
                  新增账号
                </Button>
              </div>
              <Table<SocketAccountRow>
                columns={socketAccountEditorColumns}
                dataSource={socketEditorAccounts}
                rowKey="key"
                size="small"
                pagination={false}
                scroll={{ x: 1000 }}
              />
            </div>

            {socketEditor.profile === 'CUCC' && (
              <div className={styles.editorSection}>
                <div className={styles.editorSectionHeader}>
                  <Space size={8} wrap>
                    <Typography.Text strong>文件同步传输目标</Typography.Text>
                    {renderDeliveryTargetValidationSummary('socket', socketEditor.key)}
                  </Space>
                  <Button size="small" icon={<PlusOutlined />} onClick={() => addSocketEditorDeliveryTarget(socketEditor.key)}>
                    新增目标
                  </Button>
                </div>
                <Table<DeliveryTargetRow>
                  columns={createDeliveryTargetEditorColumns(
                    updateSocketEditorDeliveryTarget,
                    removeSocketEditorDeliveryTarget,
                    'socket',
                    socketEditor.key,
                  )}
                  dataSource={socketEditorDeliveryTargets}
                  rowKey="key"
                  size="small"
                  pagination={false}
                  scroll={deliveryTargetEditorTableScroll}
                />
              </div>
            )}

            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>字段映射</Typography.Text>
              </div>
              <Table<AlarmFieldMappingRow>
                columns={alarmFieldColumns}
                dataSource={getSocketFields(socketEditor.profile)}
                rowKey="key"
                size="small"
                pagination={false}
                scroll={{ x: 960, y: 320 }}
              />
            </div>
          </Space>
        )}
      </Drawer>

      <Drawer
        title={selectedSnmp ? `${snmpVersionLabel(selectedSnmp.version)} 告警` : 'SNMP 告警'}
        open={Boolean(selectedSnmp)}
        onClose={() => setSelectedSnmp(null)}
        size="large"
        rootClassName={styles.inventoryDrawer}
        destroyOnClose
      >
        {selectedSnmp && (
          <Space orientation="vertical" size={16} className={styles.drawerBody}>
            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>基础信息</Typography.Text>
              </div>
              <Descriptions bordered size="small" column={2}>
                <Descriptions.Item label="版本">{snmpVersionTag(selectedSnmp.version)}</Descriptions.Item>
                <Descriptions.Item label="通知类型">{snmpNotificationTag(selectedSnmp.notificationType)}</Descriptions.Item>
                <Descriptions.Item label="告警上报">{statusTag(Boolean(snmpEnabled[selectedSnmp.key]))}</Descriptions.Item>
                <Descriptions.Item label="允许 MIB 查询">{selectedSnmp.mibQueryEnabled ? '开启' : '关闭'}</Descriptions.Item>
                {selectedSnmp.mibQueryEnabled && (
                  <Descriptions.Item label="Agent 监听">
                    <span className={styles.monoText}>{endpointText(selectedSnmp.listenIp, selectedSnmp.listenPort)}</span>
                  </Descriptions.Item>
                )}
                {Boolean(snmpEnabled[selectedSnmp.key]) && (
                  <Descriptions.Item label="通知目标">
                    <span className={styles.monoText}>{endpointText(selectedSnmp.targetHost, selectedSnmp.targetPort)}</span>
                  </Descriptions.Item>
                )}
                {selectedSnmp.version === 'v2' && (Boolean(snmpEnabled[selectedSnmp.key]) || selectedSnmp.mibQueryEnabled) && (
                  <Descriptions.Item label="Community" span={2}>
                    <MaskedCredentialPreview
                      value={selectedSnmp.community === storedCredentialText
                        ? storedCredentialText
                        : (selectedSnmp.community || snmpDefaultCommunity)}
                      width={180}
                    />
                  </Descriptions.Item>
                )}
                {selectedSnmp.version === 'v3' && (Boolean(snmpEnabled[selectedSnmp.key]) || selectedSnmp.mibQueryEnabled) && (
                  <>
                    <Descriptions.Item label="安全级别">{snmpV3SecurityLevelLabel(snmpV3SecurityLevel(selectedSnmp))}</Descriptions.Item>
                    <Descriptions.Item label="安全名">{selectedSnmp.securityName || '-'}</Descriptions.Item>
                    {snmpV3SecurityLevel(selectedSnmp) !== 'noAuthNoPriv' && (
                      <Descriptions.Item label="认证算法">{selectedSnmp.authProtocol || '-'}</Descriptions.Item>
                    )}
                    {snmpV3SecurityLevel(selectedSnmp) === 'authPriv' && (
                      <Descriptions.Item label="加密算法">{selectedSnmp.privProtocol || '-'}</Descriptions.Item>
                    )}
                  </>
                )}
                {Boolean(snmpEnabled[selectedSnmp.key]) && (
                  <Descriptions.Item label="清除告警级别">{selectedSnmp.clearSeverityPolicy}</Descriptions.Item>
                )}
                {Boolean(snmpEnabled[selectedSnmp.key]) && selectedSnmp.notificationType === 'Inform' && (
                  <Descriptions.Item label="超时/重试">{selectedSnmp.timeoutSeconds}s / {selectedSnmp.retries} 次</Descriptions.Item>
                )}
                <Descriptions.Item label="MIB 根" span={2}>
                  <span className={styles.monoText}>1.3.6.1.4.1.53058.1.1</span>
                </Descriptions.Item>
                <Descriptions.Item label="告警表" span={2}>
                  <span className={styles.monoText}>1.3.6.1.4.1.53058.1.1.1.1.1.1</span>
                </Descriptions.Item>
              </Descriptions>
            </div>

            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>omcAlarmEntry 字段映射</Typography.Text>
              </div>
              <Table<AlarmFieldMappingRow>
                columns={readonlySnmpAlarmFieldColumns}
                dataSource={snmpAlarmFields}
                rowKey="key"
                size="small"
                pagination={false}
                scroll={{ x: 1280, y: 380 }}
              />
            </div>
          </Space>
        )}
      </Drawer>

      <Drawer
        title={snmpEditor ? `编辑 ${snmpVersionLabel(snmpEditor.version)}` : '编辑 SNMP 告警'}
        open={Boolean(snmpEditor)}
        onClose={() => setSnmpEditor(null)}
        size="large"
        rootClassName={styles.inventoryDrawer}
        destroyOnClose
        extra={snmpEditor ? (
          <Button
            type="primary"
            loading={Boolean(snmpSaving[snmpEditor.key])}
            onClick={() => saveSnmpEditor(snmpEditor)}
          >
            保存
          </Button>
        ) : undefined}
      >
        {snmpEditor && (
          <Space orientation="vertical" size={16} className={styles.drawerBody}>
            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>目标参数</Typography.Text>
              </div>
              <Form layout="vertical" className={styles.compactForm}>
                <div className={styles.inventoryFormGrid}>
                  {Boolean(snmpEnabled[snmpEditor.key]) && (
                    <Form.Item label="通知类型">
                      <Select value={snmpEditor.notificationType} options={[{ label: 'Trap', value: 'Trap' }, { label: 'Inform', value: 'Inform' }]} onChange={(notificationType) => setSnmpEditor((current) => (current ? { ...current, notificationType } : current))} />
                    </Form.Item>
                  )}
                  <Form.Item label="启用告警上报">
                    <Switch
                      checked={snmpEnabled[snmpEditor.key]}
                      checkedChildren="开"
                      unCheckedChildren="关"
                      loading={Boolean(snmpSaving[snmpEditor.key])}
                      onChange={(checked) => setSnmpEnabled((prev) => ({ ...prev, [snmpEditor.key]: checked }))}
                    />
                  </Form.Item>
                  <Form.Item label="允许 MIB 查询">
                    <Tooltip title={snmpEditor.version === 'v3' ? 'v3 MIB walk/get 使用安全名，并按安全级别校验认证/加密参数' : undefined}>
                      <Switch
                        checked={snmpEditor.mibQueryEnabled}
                        checkedChildren="开"
                        unCheckedChildren="关"
                        onChange={(mibQueryEnabled) => setSnmpEditor((current) => (current ? { ...current, mibQueryEnabled } : current))}
                      />
                    </Tooltip>
                  </Form.Item>
                  {snmpEditor.mibQueryEnabled && (
                    <>
                      <Form.Item label="Agent IP">
                        <Input value={snmpEditor.listenIp} onChange={(event) => setSnmpEditor((current) => (current ? { ...current, listenIp: event.target.value } : current))} />
                      </Form.Item>
                      <Form.Item label="Agent 端口">
                        <InputNumber value={snmpEditor.listenPort} min={1} max={65535} style={{ width: '100%' }} onChange={(listenPort) => setSnmpEditor((current) => (current ? { ...current, listenPort: Number(listenPort ?? 1) } : current))} />
                      </Form.Item>
                    </>
                  )}
                  {Boolean(snmpEnabled[snmpEditor.key]) && (
                    <>
                      <Form.Item label="目标 IP">
                        <Input value={snmpEditor.targetHost} onChange={(event) => setSnmpEditor((current) => (current ? { ...current, targetHost: event.target.value } : current))} />
                      </Form.Item>
                      <Form.Item label="目标端口">
                        <InputNumber value={snmpEditor.targetPort} min={1} max={65535} style={{ width: '100%' }} onChange={(targetPort) => setSnmpEditor((current) => (current ? { ...current, targetPort: Number(targetPort ?? 1) } : current))} />
                      </Form.Item>
                    </>
                  )}
                </div>
              </Form>
            </div>

            {(Boolean(snmpEnabled[snmpEditor.key]) || snmpEditor.mibQueryEnabled) && (
              <div className={styles.editorSection}>
                <div className={styles.editorSectionHeader}>
                  <Typography.Text strong>安全与运行</Typography.Text>
                </div>
                <Form layout="vertical" className={styles.compactForm}>
                  <div className={styles.inventoryFormGrid}>
                    {snmpEditor.version === 'v2' && (Boolean(snmpEnabled[snmpEditor.key]) || snmpEditor.mibQueryEnabled) && (
                      <Form.Item label="Community">
                        <MaskedCredentialInput
                          value={snmpEditor.community === storedCredentialText ? storedCredentialText : (snmpEditor.community || snmpDefaultCommunity)}
                          placeholder={snmpEditor.community === storedCredentialText ? '未修改保持原 community' : `默认 ${snmpDefaultCommunity}`}
                          onChange={(community) => setSnmpEditor((current) => (current ? {
                            ...current,
                            community: community || (current.community === storedCredentialText ? storedCredentialText : ''),
                          } : current))}
                        />
                      </Form.Item>
                    )}
                    {snmpEditor.version === 'v3' && (Boolean(snmpEnabled[snmpEditor.key]) || snmpEditor.mibQueryEnabled) && (
                      <>
                        <Form.Item label="安全级别">
                          <Select
                            value={snmpV3SecurityLevel(snmpEditor)}
                            options={snmpV3SecurityLevelOptions}
                            onChange={(securityLevel: SnmpV3SecurityLevel) => setSnmpEditor((current) => (current ? applySnmpV3SecurityLevel(current, securityLevel) : current))}
                          />
                        </Form.Item>
                        <Form.Item label="安全名">
                          <Input value={snmpEditor.securityName} onChange={(event) => setSnmpEditor((current) => (current ? { ...current, securityName: event.target.value } : current))} />
                        </Form.Item>
                        {snmpV3SecurityLevel(snmpEditor) !== 'noAuthNoPriv' && (
                          <>
                            <Form.Item label="认证算法">
                              <Select value={snmpEditor.authProtocol || 'SHA'} options={snmpAuthProtocolOptions} onChange={(authProtocol) => setSnmpEditor((current) => (current ? { ...current, authProtocol } : current))} />
                            </Form.Item>
                            <Form.Item label="认证密码">
                              <MaskedCredentialInput
                                value={snmpEditor.authCredential}
                                minLength={8}
                                placeholder={snmpEditor.authCredential === storedCredentialText ? '未修改保持原密码' : '至少 8 位认证密码'}
                                onChange={(authCredential) => setSnmpEditor((current) => (current ? {
                                  ...current,
                                  authCredential: authCredential || (current.authCredential === storedCredentialText ? storedCredentialText : ''),
                                } : current))}
                              />
                            </Form.Item>
                          </>
                        )}
                        {snmpV3SecurityLevel(snmpEditor) === 'authPriv' && (
                          <>
                            <Form.Item label="加密算法">
                              <Select value={snmpEditor.privProtocol || 'DES'} options={snmpPrivProtocolOptions} onChange={(privProtocol) => setSnmpEditor((current) => (current ? { ...current, privProtocol } : current))} />
                            </Form.Item>
                            <Form.Item label="加密密码">
                              <MaskedCredentialInput
                                value={snmpEditor.privCredential}
                                minLength={8}
                                placeholder={snmpEditor.privCredential === storedCredentialText ? '未修改保持原密码' : '至少 8 位加密密码'}
                                onChange={(privCredential) => setSnmpEditor((current) => (current ? {
                                  ...current,
                                  privCredential: privCredential || (current.privCredential === storedCredentialText ? storedCredentialText : ''),
                                } : current))}
                              />
                            </Form.Item>
                          </>
                        )}
                      </>
                    )}
                    {Boolean(snmpEnabled[snmpEditor.key]) && (
                      <Form.Item label="清除告警级别">
                        <Select value={snmpEditor.clearSeverityPolicy} options={[{ label: '保留原级别', value: '保留原级别' }, { label: '清除置 0', value: '清除置 0' }]} onChange={(clearSeverityPolicy) => setSnmpEditor((current) => (current ? { ...current, clearSeverityPolicy } : current))} />
                      </Form.Item>
                    )}
                    {Boolean(snmpEnabled[snmpEditor.key]) && snmpEditor.notificationType === 'Inform' && (
                      <>
                        <Form.Item label="超时（秒）">
                          <InputNumber value={snmpEditor.timeoutSeconds} min={1} style={{ width: '100%' }} onChange={(timeoutSeconds) => setSnmpEditor((current) => (current ? { ...current, timeoutSeconds: Number(timeoutSeconds ?? 1) } : current))} />
                        </Form.Item>
                        <Form.Item label="重试次数">
                          <InputNumber value={snmpEditor.retries} min={0} style={{ width: '100%' }} onChange={(retries) => setSnmpEditor((current) => (current ? { ...current, retries: Number(retries ?? 0) } : current))} />
                        </Form.Item>
                      </>
                    )}
                  </div>
                </Form>
              </div>
            )}

            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>字段映射</Typography.Text>
              </div>
              <Table<AlarmFieldMappingRow>
                columns={snmpAlarmFieldColumns}
                dataSource={snmpAlarmFields}
                rowKey="key"
                size="small"
                pagination={false}
                scroll={{ x: 1280, y: 320 }}
              />
            </div>
          </Space>
        )}
      </Drawer>

      <Drawer
        title={viewInventoryConfig ? `${viewInventoryConfig.objectCode} Inventory 文件` : 'Inventory 文件'}
        open={Boolean(viewInventoryConfig)}
        onClose={() => setViewInventoryType(null)}
        size="large"
        rootClassName={styles.inventoryDrawer}
        destroyOnClose
        extra={viewInventoryConfig ? (
          <Button
            aria-label={`编辑 ${viewInventoryConfig.objectCode} Inventory`}
            icon={<EditOutlined />}
            onClick={() => {
              const inventoryConfig = viewInventoryConfig;
              setViewInventoryType(null);
              openInventoryEditor(inventoryConfig);
            }}
          >
            编辑
          </Button>
        ) : undefined}
      >
        {viewInventoryConfig && (
          <Space orientation="vertical" size={16} className={styles.drawerBody}>
            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>基础信息</Typography.Text>
              </div>
              <Descriptions bordered size="small" column={2}>
                <Descriptions.Item label="Inventory 类型">{viewInventoryConfig.objectCode}</Descriptions.Item>
                <Descriptions.Item label="名称">{viewInventoryConfig.name}</Descriptions.Item>
                <Descriptions.Item label="制式">{viewInventoryConfig.tech}</Descriptions.Item>
                <Descriptions.Item label="启用配置">{statusTag(Boolean(inventoryEnabled[viewInventoryConfig.key]))}</Descriptions.Item>
                <Descriptions.Item label="统计周期">{formatPeriodLabel(viewInventoryConfig.period)}</Descriptions.Item>
                <Descriptions.Item label="格式"><Tag>{viewInventoryConfig.format}</Tag></Descriptions.Item>
                <Descriptions.Item label="压缩">
                  {viewInventoryConfig.compressionEnabled ? <Tag color="green">{viewInventoryConfig.compressionFormat}</Tag> : <Tag>不压缩</Tag>}
                </Descriptions.Item>
                <Descriptions.Item label="字段数">{viewInventoryFields.length}</Descriptions.Item>
              </Descriptions>
            </div>

            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>输出规则</Typography.Text>
                <Typography.Text type="secondary">{enabledDeliverySummary(viewInventoryDeliveryTargets)}</Typography.Text>
              </div>
              <Descriptions bordered size="small" column={1}>
                <Descriptions.Item label="生成计划">
                  {formatScheduleLabel(viewInventoryConfig.period, viewInventoryConfig.cron)}
                </Descriptions.Item>
                <Descriptions.Item label="上传目录模板">
                  <span className={styles.monoText}>{viewInventoryConfig.path}</span>
                </Descriptions.Item>
                <Descriptions.Item label="文件名模板">
                  <span className={styles.monoText}>{viewInventoryConfig.fileName}</span>
                </Descriptions.Item>
                <Descriptions.Item label="预览">
                  {viewInventoryPreview && (
                    <span className={styles.previewCell}>
                      <span className={styles.previewPath}>{viewInventoryPreview.path}</span>
                      <span className={styles.previewFile}>{viewInventoryPreview.fileName}</span>
                    </span>
                  )}
                </Descriptions.Item>
              </Descriptions>
            </div>

            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>传输目标</Typography.Text>
              </div>
              <Table<DeliveryTargetRow>
                columns={deliveryTargetColumns}
                dataSource={viewInventoryDeliveryTargets}
                rowKey="key"
                size="small"
                pagination={false}
                scroll={deliveryTargetTableScroll}
              />
            </div>

            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>字段配置</Typography.Text>
              </div>
              <Table<InventoryField>
                columns={readonlyInventoryFieldColumns}
                dataSource={viewInventoryFields}
                rowKey="key"
                size="small"
                pagination={false}
                scroll={{ x: 1250, y: 360 }}
              />
            </div>
          </Space>
        )}
      </Drawer>

      <Drawer
        title={selectedInventoryConfig ? `编辑 ${selectedInventoryConfig.objectCode} Inventory 文件` : '编辑 Inventory 文件'}
        open={inventoryEditorOpen}
        onClose={() => setInventoryEditorOpen(false)}
        size="large"
        rootClassName={styles.inventoryDrawer}
        destroyOnClose
        extra={
          <Space>
            <Button aria-label="取消编辑 Inventory" onClick={() => setInventoryEditorOpen(false)}>取消</Button>
            <Button
              aria-label="保存 Inventory 配置草稿"
              type="primary"
              loading={Boolean(selectedInventoryConfig && inventoryProfileSaving[selectedInventoryConfig.key])}
              onClick={saveInventoryDraft}
            >
              保存草稿
            </Button>
          </Space>
        }
      >
        {selectedInventoryConfig && (
          <Space orientation="vertical" size={16} className={styles.drawerBody}>
            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>基础信息</Typography.Text>
              </div>
              <Form layout="vertical" className={styles.compactForm}>
                <div className={styles.inventoryFormGrid}>
                  <Form.Item label="Inventory 类型">
                    <Input value={selectedInventoryConfig.objectCode} disabled />
                  </Form.Item>
                  <Form.Item label="名称">
                    <Input
                      value={selectedInventoryConfig.name}
                      onChange={(event) => updateInventoryConfig(selectedInventoryConfig.key, { name: event.target.value })}
                    />
                  </Form.Item>
                  <Form.Item label="制式">
                    <Input value={selectedInventoryConfig.tech} disabled />
                  </Form.Item>
                  <Form.Item label="统计周期">
                    <Select
                      value={selectedInventoryConfig.period}
                      options={periodOptions}
                      onChange={(period) => updateInventoryPeriod(selectedInventoryConfig.key, period)}
                    />
                  </Form.Item>
                  <Form.Item label="格式">
                    <Select value={selectedInventoryConfig.format} disabled options={getFormatOptions('INVENTORY')} />
                  </Form.Item>
                  <Form.Item label="启用配置">
                    <Switch
                      checked={inventoryEnabled[selectedInventoryConfig.key]}
                      checkedChildren="开"
                      unCheckedChildren="关"
                      loading={Boolean(inventoryProfileSaving[selectedInventoryConfig.key])}
                      onChange={(checked) => setInventoryEnabled((prev) => ({ ...prev, [selectedInventoryConfig.key]: checked }))}
                    />
                  </Form.Item>
                  <Form.Item label="压缩">
                    <Switch
                      checked={selectedInventoryConfig.compressionEnabled}
                      checkedChildren="开"
                      unCheckedChildren="关"
                      onChange={(compressionEnabled) => updateInventoryConfig(selectedInventoryConfig.key, { compressionEnabled })}
                    />
                  </Form.Item>
                  <Form.Item label="压缩格式">
                    <Select
                      value={selectedInventoryConfig.compressionFormat}
                      disabled={!selectedInventoryConfig.compressionEnabled}
                      options={compressionFormatOptions}
                      onChange={(compressionFormat) => updateInventoryConfig(selectedInventoryConfig.key, { compressionFormat: compressionFormat as CompressionFormat })}
                    />
                  </Form.Item>
                </div>
              </Form>
            </div>

            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>输出规则</Typography.Text>
              </div>
              <Form layout="vertical" className={styles.compactForm}>
                <Form.Item label="生成计划">
                  {renderScheduleEditor(
                    selectedInventoryConfig.period,
                    selectedInventoryConfig.cron,
                    (cron) => updateInventoryConfig(selectedInventoryConfig.key, { cron }),
                  )}
                </Form.Item>
                <Form.Item label="上传目录模板">
                  <Input
                    value={selectedInventoryConfig.path}
                    className={styles.monoText}
                    onChange={(event) => updateInventoryConfig(selectedInventoryConfig.key, { path: event.target.value })}
                  />
                </Form.Item>
                <Form.Item label="文件名模板">
                  <Input
                    value={selectedInventoryConfig.fileName}
                    className={styles.monoText}
                    onChange={(event) => updateInventoryConfig(selectedInventoryConfig.key, { fileName: event.target.value })}
                  />
                </Form.Item>
                <Form.Item label="预览">
                  {selectedInventoryPreview && (
                    <span className={styles.previewCell}>
                      <span className={styles.previewPath}>{selectedInventoryPreview.path}</span>
                      <span className={styles.previewFile}>{selectedInventoryPreview.fileName}</span>
                    </span>
                  )}
                </Form.Item>
              </Form>
            </div>

            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Space size={8} wrap>
                  <Typography.Text strong>传输目标</Typography.Text>
                  {renderDeliveryTargetValidationSummary('inventory', selectedInventoryConfig.key)}
                </Space>
                <Button size="small" icon={<PlusOutlined />} onClick={() => addInventoryDeliveryTarget(selectedInventoryConfig.key)}>
                  新增目标
                </Button>
              </div>
              <Table<DeliveryTargetRow>
                columns={createDeliveryTargetEditorColumns(
                  (targetKey, patch) => updateInventoryDeliveryTarget(selectedInventoryConfig.key, targetKey, patch),
                  (targetKey) => removeInventoryDeliveryTarget(selectedInventoryConfig.key, targetKey),
                  'inventory',
                  selectedInventoryConfig.key,
                )}
                dataSource={selectedInventoryDeliveryTargets}
                rowKey="key"
                size="small"
                pagination={false}
                scroll={deliveryTargetEditorTableScroll}
              />
            </div>

            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>字段配置</Typography.Text>
                <Space size={8} wrap className={styles.fieldConfigTools}>
                  <Select
                    allowClear
                    showSearch
                    value={inventoryCandidateKey}
                    placeholder="搜索可新增字段"
                    optionFilterProp="label"
                    style={{ width: 340 }}
                    options={inventoryCandidateOptions}
                    onChange={setInventoryCandidateKey}
                    notFoundContent="无可新增字段"
                  />
                  <Button
                    icon={<PlusOutlined />}
                    disabled={!inventoryCandidateKey}
                    onClick={addInventoryField}
                  >
                    新增字段
                  </Button>
                </Space>
              </div>
              <Table<InventoryField>
                columns={inventoryFieldColumns}
                dataSource={currentInventoryFields}
                rowKey="key"
                size="small"
                pagination={false}
                scroll={{ x: 1250, y: 360 }}
              />
            </div>
          </Space>
        )}
      </Drawer>

      <Drawer
        title={selectedScenario ? `${selectedScenario.code} 北向文件配置` : '北向文件配置'}
        open={Boolean(selectedScenario)}
        onClose={() => setSelectedScenario(null)}
        size="large"
        destroyOnClose
        extra={selectedScenario ? (
          <Button
            aria-label={`编辑 ${selectedScenario.code}`}
            icon={<EditOutlined />}
            onClick={() => {
              const scenario = selectedScenario;
              setSelectedScenario(null);
              openEditEditor(scenario);
            }}
          >
            编辑
          </Button>
        ) : undefined}
      >
        {selectedScenario && (
          <Space orientation="vertical" size={16} className={styles.drawerBody}>
            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>基础信息</Typography.Text>
              </div>
              <Descriptions bordered size="small" column={2}>
                <Descriptions.Item label="场景号/配置编号">{selectedScenario.code}</Descriptions.Item>
                <Descriptions.Item label="场景中文名称">{selectedScenario.scenarioName}</Descriptions.Item>
                <Descriptions.Item label="场景英文名称">{selectedScenario.scenarioNameEn}</Descriptions.Item>
                <Descriptions.Item label="配置名称">{selectedScenario.name}</Descriptions.Item>
                <Descriptions.Item label="场景说明" span={2}>{selectedScenario.description}</Descriptions.Item>
                <Descriptions.Item label="业务域" span={2}>
                  <Space size={4} wrap>{getScenarioDomains(selectedScenario).map((domain) => domainTag(domain))}</Space>
                </Descriptions.Item>
                <Descriptions.Item label="启用配置">{statusTag(Boolean(scenarioEnabled[selectedScenario.code]))}</Descriptions.Item>
              </Descriptions>
            </div>

            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>对象管理</Typography.Text>
              </div>
              <Table<ScenarioPeriodRow>
                columns={[
                  { title: '业务域', dataIndex: 'domain', width: 112, render: (value: Domain) => domainTag(value) },
                  {
                    title: '对象',
                    dataIndex: 'objects',
                    width: 320,
                    render: (value: string) => (
                      <Space size={4} wrap>
                        {splitObjects(value).map((object) => <Tag key={object}>{formatObjectToken(object)}</Tag>)}
                      </Space>
                    ),
                  },
                  { title: '制式/模式', dataIndex: 'scope', width: 128, render: (value: string) => <Tag>{formatScopeLabel(value)}</Tag> },
                  { title: '格式', dataIndex: 'format', width: 104, render: (value: Format) => <Tag>{value}</Tag> },
                  {
                    title: 'CSV 分隔符',
                    dataIndex: 'csvSeparator',
                    width: 132,
                    render: (value: string | undefined, record) => (
                      normalizeFormatForDomain(record.domain, record.format, record.objects) === 'CSV'
                        ? <Tag>{csvSeparatorDisplay(value)}</Tag>
                        : <Tag>不适用</Tag>
                    ),
                  },
                  { title: '统计周期', dataIndex: 'period', width: 128, render: (value: string) => <Typography.Text strong>{formatPeriodLabel(value)}</Typography.Text> },
                  { title: '生成计划', width: 210, render: (_, record) => formatScheduleLabel(record.period, record.cron) },
                  {
                    title: nowrapColumnTitle('压缩'),
                    width: 116,
                    render: (_, record) => record.compressionEnabled ? <Tag color="green">开启</Tag> : <Tag>关闭</Tag>,
                  },
                  { title: nowrapColumnTitle('压缩格式'), dataIndex: 'compressionFormat', width: 152, render: (value: CompressionFormat) => <Tag>{value}</Tag> },
                  { title: '上传目录模板', dataIndex: 'path', width: 360, render: (value: string) => <span className={styles.monoText}>{value}</span> },
                  { title: '文件名模板', dataIndex: 'fileName', width: 460, render: (value: string) => <span className={styles.fileNameTemplate}>{value}</span> },
                  {
                    title: '预览',
                    width: 420,
                    render: (_, record) => {
                      const preview = getOutputTemplatePreview(record);
                      return (
                        <span className={styles.previewCell}>
                          <span className={styles.previewPath}>{preview.path}</span>
                          <span className={styles.previewFile}>{preview.fileName}</span>
                        </span>
                      );
                    },
                  },
                ]}
                dataSource={viewPeriodRows}
                rowKey="key"
                size="small"
                pagination={false}
                scroll={{ x: 2580 }}
              />
            </div>

            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>传输目标</Typography.Text>
                <Typography.Text type="secondary">{enabledDeliverySummary(viewFileDeliveryTargets)}</Typography.Text>
              </div>
              <Table<DeliveryTargetRow>
                columns={deliveryTargetColumns}
                dataSource={viewFileDeliveryTargets}
                rowKey="key"
                size="small"
                pagination={false}
              scroll={deliveryTargetTableScroll}
              />
            </div>

            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>字段/指标配置</Typography.Text>
                <Space size={8} wrap className={styles.fieldConfigTools}>
                  <Select
                    value={viewFieldConfigDomain}
                    style={{ width: 120 }}
                    options={viewFieldDomainOptions}
                    onChange={(value) => changeViewFieldConfigDomain(value as Domain)}
                  />
                  <Select
                    value={activeViewFieldTechFilter}
                    style={{ width: 132 }}
                    options={viewFieldTechFilterOptions}
                    disabled={!isRadioTechDomain(viewFieldConfigDomain)}
                    onChange={(value) => changeViewFieldTechFilter(value as FieldTechFilter)}
                  />
                  <Select
                    value={selectedViewFieldTarget?.key}
                    placeholder="对象"
                    style={{ width: 132 }}
                    options={viewFieldTargetOptions}
                    onChange={setViewFieldConfigTargetKey}
                    notFoundContent="无对象"
                  />
                </Space>
              </div>
              <Table<ReportFieldRow>
                columns={readonlyFieldConfigColumns}
                dataSource={viewFieldRows}
                rowKey="key"
                size="small"
                virtual
                loading={pmLoading && selectedViewFieldTarget?.domain === 'PM'}
                pagination={viewFieldRows.length > 100 ? {
                  pageSize: 50,
                  showSizeChanger: true,
                  pageSizeOptions: [20, 50, 100],
                  showTotal: (total) => `共 ${total} 项`,
                } : false}
                scroll={{ x: 1726, y: 320 }}
              />
            </div>
          </Space>
        )}
      </Drawer>

      <Drawer
        title={editorMode === 'create' ? '新增北向文件配置' : '编辑北向文件配置'}
        open={editorOpen}
        onClose={() => setEditorOpen(false)}
        size="large"
        destroyOnClose
        extra={
          <Space>
            <Button aria-label="取消编辑" onClick={() => setEditorOpen(false)}>取消</Button>
            <Button
              aria-label="保存配置草稿"
              type="primary"
              loading={Boolean(editorFileOwnerCode && fileProfileSaving[editorFileOwnerCode])}
              onClick={saveEditorDraft}
            >
              保存草稿
            </Button>
          </Space>
        }
      >
        <Form form={configForm} layout="vertical">
          <div className={styles.editorSection}>
            <div className={styles.editorSectionHeader}>
              <Typography.Text strong>基础信息</Typography.Text>
            </div>
            <Form.Item label="场景号/配置编号" name="scenarioCode" rules={[{ required: true, message: '请输入配置编号' }]}>
              <Input />
            </Form.Item>
            <Form.Item label="厂商" name="vendor">
              <Input />
            </Form.Item>
            <Form.Item label="场景中文名称" name="scenarioName" rules={[{ required: true, message: '请输入场景中文名称' }]}>
              <Input />
            </Form.Item>
            <Form.Item label="场景英文名称" name="scenarioNameEn">
              <Input />
            </Form.Item>
            <Form.Item label="配置名称" name="name" rules={[{ required: true, message: '请输入配置名称' }]}>
              <Input />
            </Form.Item>
            <Form.Item label="场景说明" name="description">
              <Input.TextArea rows={4} />
            </Form.Item>
            <Space size={12} align="start" wrap>
              <Form.Item label="业务域" name="domains" rules={[{ required: true, message: '请选择业务域' }]}>
                <Select
                  mode="multiple"
                  style={{ width: 260 }}
                  options={fileDomainOptions}
                />
              </Form.Item>
            </Space>
          </div>
          <div className={styles.editorSection}>
            <div className={styles.editorSectionHeader}>
              <Typography.Text strong>对象管理</Typography.Text>
              <Button size="small" icon={<PlusOutlined />} onClick={addEditorPeriodRow}>
                新增对象
              </Button>
            </div>
            <Table<ScenarioPeriodRow>
              columns={[
                {
                  title: '操作',
                  width: 80,
                  fixed: 'left',
                  render: (_, record) => (
                    <Popconfirm
                      title={nt('确认删除该对象？')}
                      description={nt('删除后需保存草稿才会生效。')}
                      okText={nt('删除')}
                      cancelText={nt('取消')}
                      okButtonProps={{ danger: true }}
                      disabled={editorMode === 'edit' && editorPeriodRows.length <= 1}
                      onConfirm={() => removeEditorPeriodRow(record.key)}
                    >
                      <Tooltip title="删除对象">
                        <Button
                          aria-label={`删除 ${record.domain} 对象`}
                          type="text"
                          danger
                          size="small"
                          icon={<DeleteOutlined />}
                          disabled={editorMode === 'edit' && editorPeriodRows.length <= 1}
                        />
                      </Tooltip>
                    </Popconfirm>
                  ),
                },
                {
                  title: '业务域',
                  dataIndex: 'domain',
                  width: 112,
                  render: (value: Domain, record) => (
                    <Select
                      value={value}
                      style={{ width: 96 }}
                      options={fileDomainOptions}
                      onChange={(nextDomain) => updateEditorPeriodDomain(record.key, nextDomain as Domain)}
                    />
                  ),
                },
                {
                  title: '对象',
                  dataIndex: 'objects',
                  width: 320,
                  render: (value: string, record) => (
                    <Select
                      mode="multiple"
                      value={splitObjects(value)}
                      maxTagCount={8}
                      style={{ width: 300 }}
                      options={getObjectOptions(record.domain, value)}
                      onChange={(nextObjects) => updateEditorPeriodRow(record.key, { objects: joinObjects(nextObjects) })}
                    />
                  ),
                },
                {
                  title: '制式/模式',
                  dataIndex: 'scope',
                  width: 128,
                  render: (value: string, record) => (
                    <Select
                      value={value}
                      style={{ width: 110 }}
                      options={getScopeOptions(record.domain)}
                      onChange={(nextScope) => updateEditorPeriodRow(record.key, { scope: nextScope })}
                    />
                  ),
                },
                {
                  title: '格式',
                  dataIndex: 'format',
                  width: 104,
                  render: (value: Format, record) => (
                    <Select
                      value={normalizeFormatForDomain(record.domain, value, record.objects)}
                      style={{ width: 86 }}
                      options={getFormatOptions(record.domain, record.objects)}
                      onChange={(nextFormat) => updateEditorPeriodRow(record.key, { format: normalizeFormatForDomain(record.domain, nextFormat as Format, record.objects) })}
                    />
                  ),
                },
                {
                  title: 'CSV 分隔符',
                  dataIndex: 'csvSeparator',
                  width: 142,
                  render: (value: string | undefined, record) => (
                    normalizeFormatForDomain(record.domain, record.format, record.objects) === 'CSV'
                      ? (
                          <CSVSeparatorInput
                            value={value}
                            rowKey={record.key}
                            onCommit={(rowKey, csvSeparator) => updateEditorPeriodRow(rowKey, { csvSeparator })}
                          />
                        )
                      : <Tag>不适用</Tag>
                  ),
                },
                {
                  title: '统计周期',
                  dataIndex: 'period',
                  width: 128,
                  render: (value: string, record) => (
                    <Select
                      value={value}
                      style={{ width: 104 }}
                      options={periodOptions}
                      onChange={(nextPeriod) => updateEditorPeriod(record.key, nextPeriod)}
                    />
                  ),
                },
                {
                  title: '生成计划',
                  width: 230,
                  render: (_, record) => renderScheduleEditor(
                    record.period,
                    record.cron,
                    (cron) => updateEditorPeriodRow(record.key, { cron }),
                  ),
                },
                {
                  title: nowrapColumnTitle('压缩'),
                  width: 116,
                  render: (_, record) => (
                    <Switch
                      size="small"
                      checked={record.compressionEnabled}
                      checkedChildren="开"
                      unCheckedChildren="关"
                      onChange={(checked) => updateEditorPeriodRow(record.key, { compressionEnabled: checked })}
                    />
                  ),
                },
                {
                  title: nowrapColumnTitle('压缩格式'),
                  dataIndex: 'compressionFormat',
                  width: 152,
                  render: (value: CompressionFormat, record) => (
                    <Select
                      value={value}
                      style={{ width: 96 }}
                      options={compressionFormatOptions}
                      onChange={(nextFormat) => updateEditorPeriodRow(record.key, { compressionFormat: nextFormat as CompressionFormat })}
                    />
                  ),
                },
	                {
	                  title: '上传目录模板',
	                  dataIndex: 'path',
	                  width: 360,
	                  render: (value: string, record) => (
	                    <PeriodTextInput
                      value={value}
                      rowKey={record.key}
                      field='path'
                      onCommit={commitPeriodTextField}
                      className={styles.monoText}
                    />
	                  ),
	                },
	                {
	                  title: '文件名模板',
	                  dataIndex: 'fileName',
	                  width: 460,
	                  render: (value: string, record) => (
	                    <PeriodTextInput
                      value={value}
                      rowKey={record.key}
                      field='fileName'
                      onCommit={commitPeriodTextField}
                      className={styles.monoText}
                    />
	                  ),
	                },
	                {
	                  title: '预览',
	                  width: 420,
	                  render: (_, record) => {
	                    const preview = getOutputTemplatePreview(record);
	                    return (
	                      <span className={styles.previewCell}>
	                        <span className={styles.previewPath}>{preview.path}</span>
	                        <span className={styles.previewFile}>{preview.fileName}</span>
	                      </span>
	                    );
	                  },
	                },
              ]}
              dataSource={editorPeriodRows}
              rowKey="key"
              locale={{ emptyText: '暂无对象，请新增对象后选择业务域和对象' }}
	              size="small"
	              pagination={false}
	              scroll={{ x: 2670 }}
	            />
          </div>
          <div className={styles.editorSection}>
            <div className={styles.editorSectionHeader}>
              <Space size={8} wrap>
                <Typography.Text strong>传输目标</Typography.Text>
                {renderDeliveryTargetValidationSummary('file', editorFileOwnerCode)}
              </Space>
              <Button
                size="small"
                icon={<PlusOutlined />}
                disabled={!editorFileOwnerCode}
                onClick={() => addFileDeliveryTarget(editorFileOwnerCode)}
              >
                新增目标
              </Button>
            </div>
            <Table<DeliveryTargetRow>
              columns={createDeliveryTargetEditorColumns(
                (targetKey, patch) => updateFileDeliveryTarget(editorFileOwnerCode, targetKey, patch),
                (targetKey) => removeFileDeliveryTarget(editorFileOwnerCode, targetKey),
                'file',
                editorFileOwnerCode,
              )}
              dataSource={editorFileDeliveryTargets}
              rowKey="key"
              size="small"
              pagination={false}
              scroll={deliveryTargetEditorTableScroll}
            />
          </div>
          <FieldConfigSection
            ref={fieldConfigRef}
            key={editorSession}
            periodRows={editorPeriodRows}
            pmMetricRows={pmMetricRows}
            pmLoading={pmLoading}
            initialFieldRowsByTarget={initialFieldRows}
          />

          <Form.Item label="启用配置" name="enabled" valuePropName="checked">
            <Switch checkedChildren="开" unCheckedChildren="关" />
          </Form.Item>
        </Form>
      </Drawer>
    </div>
    </NorthboundI18nScope>
  );
}
