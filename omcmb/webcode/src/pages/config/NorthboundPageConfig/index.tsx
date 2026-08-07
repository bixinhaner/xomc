import { forwardRef, memo, useCallback, useEffect, useImperativeHandle, useMemo, useRef, useState } from 'react';
import {
  Button,
  Descriptions,
  Drawer,
  Dropdown,
  Form,
  Input,
  InputNumber,
  Pagination,
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
  CopyOutlined,
  DatabaseOutlined,
  DeleteOutlined,
  DownloadOutlined,
  EditOutlined,
  EyeOutlined,
  FileSearchOutlined,
  FileTextOutlined,
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
  NorthboundAPIClient,
  NorthboundDeliveryTarget,
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
import styles from './index.module.css';

type Domain = 'CM' | 'PM' | 'MR' | 'LOG' | 'INVENTORY';
type Format = 'XML' | 'CSV' | 'TXT';
type CompressionFormat = 'zip' | 'gz';
type MetricType = 'counter' | 'kpi';
type Tech = 'LTE' | 'GNB' | 'GSM';
type FieldTechFilter = 'ALL' | Tech;
type InventoryType = 'ENB' | 'GNB' | 'GSM' | 'OMC';

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
}

interface FileProfileEditorValues {
  scenarioCode: string;
  vendor?: string;
  scenarioName: string;
  scenarioNameEn?: string;
  name: string;
  description?: string;
  enabled?: boolean;
}

type DeliveryProtocol = 'FTP' | 'SFTP';
type DeliveryAuthMode = 'PASSWORD' | 'PRIVATE_KEY';
type DeliveryHostKeyPolicy = 'INSECURE' | 'FINGERPRINT';
type ReportState = 'success' | 'failed' | 'running' | 'idle';
type ReportArtifactType = 'file' | 'message';

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
}

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
type ApiKind = '正式北向' | '业务复用' | '鉴权管理';
type ApiCoverage = '旧能力对齐' | '部分覆盖' | '当前新增';
type ApiFieldContract = '完整契约' | '当前契约' | '示例契约';

type SocketProfile = 'CTCC' | 'CUCC';
type SnmpVersion = 'v2' | 'v3';
type SnmpNotificationType = 'Trap' | 'Inform';

const snmpAuthProtocolOptions = [
  { label: 'SHA', value: 'SHA' },
  { label: 'SHA224', value: 'SHA224' },
  { label: 'SHA256', value: 'SHA256' },
  { label: 'SHA384', value: 'SHA384' },
  { label: 'SHA512', value: 'SHA512' },
  { label: 'MD5', value: 'MD5' },
];

const snmpPrivProtocolOptions = [
  { label: 'AES128', value: 'AES128' },
  { label: 'AES192', value: 'AES192' },
  { label: 'AES256', value: 'AES256' },
  { label: 'DES', value: 'DES' },
];

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
  apiKind?: ApiKind;
  module: string;
  name: string;
  method: ApiMethod;
  url: string;
  auth: string;
  backendSource: string;
  oldMapping?: string;
  coverage?: ApiCoverage;
  fieldContract?: ApiFieldContract;
  responseFields?: string[];
  compatibilityNote?: string;
  requestExample: string;
  responseExample: string;
}

interface ApiClientRow {
  clientKey: string;
  name: string;
  enabled: boolean;
  tokenSecret: string;
  tokenSet: boolean;
  allowedApiKeys: string;
  ipWhitelist: string;
  expiresAt?: string;
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
const PATH_LOG = '/#FTPRoot#/#Province#/#OMC-R#/LOGS/#DateTime#/';
const PATH_INVENTORY = '/#FTPRoot#/#Province#/#OMC-R#/Inventory/#Object#/#DateTime#/';
const CM_NAME = 'Baicells-#Object#-#LocalHost#-#DataVersion#-#DateTime#[-#Ri#][-#FileID#]';
const PM_NAME = 'Baicells-#Object#-#LocalHost#-#DataVersion#-#DateTime#[-#Ri#]-#DataPeriod#[-#FileID#]';
const MR_NAME = '#ModuleType#-Baicells-#Object#-#LocalHost#-#eNBID#-#DateTime#[-#Ri#].xml';
const LOG_CUSTOM_NAME = 'Northbound-log-{login|operation}-#DateTime#.txt';
const LOG_FIX_NAME = 'Northbound-log-fix-{login|operation}-#DateTime#.csv';
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
  CM: ['CSV'],
  PM: ['CSV'],
  MR: ['XML'],
  LOG: ['TXT', 'CSV'],
  INVENTORY: ['CSV'],
};

const defaultFormatByDomain: Record<Domain, Format> = {
  CM: 'CSV',
  PM: 'CSV',
  MR: 'XML',
  LOG: 'TXT',
  INVENTORY: 'CSV',
};

const defaultRemotePathByDomain: Record<Domain, string> = {
  CM: PATH_CM,
  PM: PATH_PM,
  MR: PATH_MR,
  LOG: PATH_LOG,
  INVENTORY: PATH_INVENTORY,
};

function getFormatOptions(domain: Domain) {
  return supportedFormatsByDomain[domain].map((value) => ({ label: formatLabels[value], value }));
}

function normalizeFormatForDomain(domain: Domain, format: Format): Format {
  return supportedFormatsByDomain[domain].includes(format) ? format : defaultFormatByDomain[domain];
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

const scopeOptions = [
  { label: '默认', value: '默认' },
  { label: 'LTE', value: 'LTE' },
  { label: 'GNB', value: 'GNB' },
  { label: 'GSM', value: 'GSM' },
  { label: 'LTE/GNB', value: 'LTE/GNB' },
  { label: 'LTE/GSM', value: 'LTE/GSM' },
  { label: 'custom', value: 'custom' },
  { label: 'fix', value: 'fix' },
];

const compressionFormatOptions = [
  { label: 'zip', value: 'zip' },
  { label: 'gz', value: 'gz' },
];

const allProductClassValue = 'ALL';

const fieldTechFilterOptions: Array<{ label: string; value: FieldTechFilter }> = [
  { label: '全部制式', value: 'ALL' },
  { label: 'LTE', value: 'LTE' },
  { label: 'GNB', value: 'GNB' },
  { label: 'GSM', value: 'GSM' },
];

const objectOptionsByDomain: Record<Domain, Array<{ label: string; value: string }>> = {
  CM: ['CP', 'EP', 'CC', 'CE', 'COMS'].map((value) => ({ label: value, value })),
  PM: ['PC', 'PE'].map((value) => ({ label: value, value })),
  MR: ['MRO', 'MRE', 'MRS'].map((value) => ({ label: value, value })),
  LOG: ['登录日志', '操作日志', '登录固定格式', '操作固定格式'].map((value) => ({ label: value, value })),
  INVENTORY: ['eNB', 'gNB', 'GSM', 'OMC'].map((value) => ({ label: value, value })),
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
  return {
    id,
    domain,
    format: normalizeFormatForDomain(domain, format),
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
    name: 'LTE + GNB 双制式',
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
    name: 'LTE + GSM PM',
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
    description: '陕西移动：LTE + GNB 双制式，CM/PM 按制式拆分输出。',
    flags: ['LTE/GNB', '双制式'],
  },
  S0013: {
    vendor: 'Baicells',
    scenarioName: 'ZED 场景',
    scenarioNameEn: 'ZED',
    description: 'ZED：pmresult 命名，LTE/GSM/GNB 性能文件按 60 分钟窗口输出。',
    flags: ['LTE/GSM/GNB', 'PM 60M', 'pmresult'],
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
    description: 'MTN：LTE + GSM 性能混合输出，GSM 使用 pmresult 命名。',
    flags: ['LTE/GSM', 'show_site_id'],
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
  ['login_time', 'log.login_time', 'audit_logs.created_at', 'datetime', 'yyyy-MM-dd HH:mm:ss', '登录时间'],
  ['username', 'log.username', 'audit_logs.username', 'string', 'quote', '用户名'],
  ['client_ip', 'log.client_ip', 'audit_logs.client_ip', 'string', 'quote', '客户端 IP'],
  ['operation_time', 'log.operation_time', 'operation_logs.created_at', 'datetime', 'yyyy-MM-dd HH:mm:ss', '操作时间'],
  ['operator', 'log.operator', 'operation_logs.operator', 'string', 'quote', '操作人'],
  ['module', 'log.module', 'operation_logs.module', 'string', 'quote', '模块'],
  ['action', 'log.action', 'operation_logs.action', 'string', 'quote', '动作'],
  ['result', 'log.result', 'audit_logs.result / operation_logs.result', 'string', 'enum', '结果'],
].map(([outputAlias, systemField, source, dataType, renderer, cnName]) => ({ outputAlias, systemField, source, dataType, renderer, cnName }));

const logFieldsByObject: Record<string, FieldDefinition[]> = {
  登录日志: logFieldDefinitions.filter((field) => ['login_time', 'username', 'client_ip', 'result'].includes(field.outputAlias)),
  操作日志: logFieldDefinitions.filter((field) => ['operation_time', 'operator', 'module', 'action', 'result'].includes(field.outputAlias)),
  登录固定格式: logFieldDefinitions.filter((field) => ['login_time', 'username', 'client_ip', 'result'].includes(field.outputAlias)),
  操作固定格式: logFieldDefinitions.filter((field) => ['operation_time', 'operator', 'module', 'action', 'result'].includes(field.outputAlias)),
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
    登录日志: [
      ['request_id', 'log.request_id', 'audit_logs.request_id', 'string', 'quote', '请求 ID'],
      ['user_agent', 'log.user_agent', 'audit_logs.user_agent', 'string', 'quote', '客户端标识'],
    ].map(([outputAlias, systemField, source, dataType, renderer, cnName]) => ({ outputAlias, systemField, source, dataType, renderer, cnName })),
    操作日志: [
      ['request_id', 'log.request_id', 'operation_logs.request_id', 'string', 'quote', '请求 ID'],
      ['resource', 'log.resource', 'operation_logs.resource', 'string', 'quote', '资源对象'],
    ].map(([outputAlias, systemField, source, dataType, renderer, cnName]) => ({ outputAlias, systemField, source, dataType, renderer, cnName })),
    登录固定格式: [
      ['request_id', 'log.request_id', 'audit_logs.request_id', 'string', 'quote', '请求 ID'],
      ['user_agent', 'log.user_agent', 'audit_logs.user_agent', 'string', 'quote', '客户端标识'],
    ].map(([outputAlias, systemField, source, dataType, renderer, cnName]) => ({ outputAlias, systemField, source, dataType, renderer, cnName })),
    操作固定格式: [
      ['request_id', 'log.request_id', 'operation_logs.request_id', 'string', 'quote', '请求 ID'],
      ['resource', 'log.resource', 'operation_logs.resource', 'string', 'quote', '资源对象'],
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

function createDeliveryTarget(scopeKey: string, index: number): DeliveryTargetRow {
  return {
    key: `${scopeKey}-delivery-custom-${Date.now()}`,
    name: `新增传输目标 ${index}`,
    enabled: false,
    protocol: 'SFTP',
    host: '',
    port: 22,
    username: '',
    credential: '未设置',
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
    name: 'NMS V2 Trap',
    version: 'v2',
    notificationType: 'Trap',
    listenIp: '0.0.0.0',
    listenPort: 161,
    targetHost: '10.10.41.11',
    targetPort: 162,
    community: 'baicells',
    securityName: 'baicells',
    mibQueryEnabled: true,
    clearSeverityPolicy: '保留原级别',
    timeoutSeconds: 5,
    retries: 3,
  },
  {
    key: 'snmp-v3-inform',
    name: 'NMS V3 Inform',
    version: 'v3',
    notificationType: 'Inform',
    listenIp: '0.0.0.0',
    listenPort: 161,
    targetHost: '10.10.41.12',
    targetPort: 163,
    securityName: 'notifyV3',
    authProtocol: 'SHA',
    privProtocol: 'AES128',
    mibQueryEnabled: false,
    clearSeverityPolicy: '清除置 0',
    timeoutSeconds: 5,
    retries: 3,
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
      credential: '已加密存储',
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
      credential: '已加密存储',
      enabled: true,
      purpose: '登录、实时告警、历史消息同步',
    },
    {
      key: 'ftp-main',
      channel: '文件同步账号',
      username: 'north-file',
      type: 'ftp',
      credential: '已加密存储',
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
    detail: 'SNMP V2 Trap 已发送到目标 NMS。',
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
    targetSummary: 'Inform 等待响应超时，已重试 3 次',
    detail: '目标未返回 Inform ACK，建议检查 NMS 地址、安全用户和防火墙。',
    payload: 'version=v3; user=notifyV3; auth=SHA; priv=AES128; notificationID=920171; perceivedSeverity=minor; result=timeout',
  },
};

const commonApiAuth = 'JWT 或 X-API-Key；/api/v1 统一鉴权 + Casbin 端点权限';
const currentEnvelopeFields = ['ret', 'msg', 'data'];
const listResponseFields = ['ret', 'msg', 'data.items', 'data.total', 'data.page', 'data.page_size', 'data.total_pages', 'data.stats?'];

const northboundApiRows: NorthboundApiRow[] = [
  {
    key: 'auth-login',
    apiKind: '鉴权管理',
    module: '鉴权',
    name: '登录获取 JWT',
    method: 'POST',
    url: '/api/v1/auth/login',
    auth: '公开接口；登录后使用 Authorization: Bearer <access_token>',
    backendSource: 'omcgo/internal/admin/auth_handler.go',
    oldMapping: '/v1/access/token',
    coverage: '部分覆盖',
    fieldContract: '完整契约',
    responseFields: [...currentEnvelopeFields, 'data.access_token', 'data.refresh_token', 'data.expires_at', 'data.token_type', 'data.must_change_password?', 'data.password_expires_in_days?', 'data.login_notify_msg?'],
    compatibilityNote: '当前登录接口使用 /api/v1/auth/login 和 TokenPair 字段；不兼容老系统 /v1/access/token 的 Result.data.token/expires 命名。',
    requestExample: `POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "northbound_api",
  "encrypted_password": "<RSA-OAEP 密文>",
  "key_id": "login-key-20260730"
}`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_at": "2026-07-30T02:00:00+08:00",
    "token_type": "Bearer"
  }
}`,
  },
  {
    key: 'api-key-create',
    apiKind: '鉴权管理',
    module: '鉴权',
    name: '创建 API Key',
    method: 'POST',
    url: '/api/v1/api-keys',
    auth: 'JWT；创建后调用北向接口使用 X-API-Key',
    backendSource: 'omcgo/internal/admin/apikey_handler.go',
    oldMapping: '老系统无 API Key；对应 /v1/access/token 的程序化调用替代方案',
    coverage: '当前新增',
    fieldContract: '完整契约',
    responseFields: [...currentEnvelopeFields, 'data.id', 'data.name', 'data.key', 'data.key_prefix', 'data.scopes', 'data.expires_at?', 'data.created_at'],
    compatibilityNote: 'key 明文只在创建响应返回一次，后续列表只显示 key_prefix。',
    requestExample: `POST /api/v1/api-keys
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "oss-primary",
  "scopes": ["northbound:read", "northbound:write", "pm:read"],
  "expires_at": "2027-07-30T00:00:00+08:00"
}`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "id": "5f8f33aa-6e7d-4f53-a5a6-b2c100010001",
    "name": "oss-primary",
    "key": "omc_nb_xxxxxxxxxxxxxxxxx",
    "key_prefix": "omc_nb_xx",
    "scopes": ["northbound:read", "northbound:write", "pm:read"],
    "expires_at": "2027-07-30T00:00:00+08:00",
    "created_at": "2026-07-30T01:00:00+08:00"
  }
}`,
  },
  {
    key: 'auth-refresh',
    apiKind: '鉴权管理',
    module: '鉴权',
    name: '刷新 JWT',
    method: 'POST',
    url: '/api/v1/auth/refresh',
    auth: '公开接口；使用 refresh_token 换取新的 access_token',
    backendSource: 'omcgo/internal/admin/auth_handler.go',
    oldMapping: '老系统 /v1/access/token 重新登录获取 token',
    coverage: '部分覆盖',
    fieldContract: '完整契约',
    responseFields: [...currentEnvelopeFields, 'data.access_token', 'data.refresh_token', 'data.expires_at', 'data.token_type'],
    compatibilityNote: '当前 refresh 与登录拆分；老系统文档未提供独立 refresh endpoint。',
    requestExample: `POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_at": "2026-07-30T03:00:00+08:00",
    "token_type": "Bearer"
  }
}`,
  },
  {
    key: 'api-key-list',
    apiKind: '鉴权管理',
    module: '鉴权',
    name: '查询 API Key',
    method: 'GET',
    url: '/api/v1/api-keys',
    auth: commonApiAuth,
    backendSource: 'omcgo/internal/admin/apikey_handler.go',
    oldMapping: '老系统无 API Key 管理接口',
    coverage: '当前新增',
    fieldContract: '完整契约',
    responseFields: [...currentEnvelopeFields, 'data.items[].id', 'data.items[].user_id', 'data.items[].name', 'data.items[].key_prefix', 'data.items[].scopes', 'data.items[].expires_at?', 'data.items[].last_used_at?', 'data.items[].created_at', 'data.items[].updated_at', 'data.items[].revoked_at?', 'data.total'],
    compatibilityNote: '列表不会返回 key 明文。',
    requestExample: `GET /api/v1/api-keys
Authorization: Bearer <token>`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "items": [
      {
        "id": "5f8f33aa-6e7d-4f53-a5a6-b2c100010001",
        "name": "oss-primary",
        "key_prefix": "omc_nb_xx",
        "scopes": ["northbound:read"],
        "last_used_at": "2026-07-30T01:10:00+08:00",
        "created_at": "2026-07-30T01:00:00+08:00"
      }
    ],
    "total": 1
  }
}`,
  },
  {
    key: 'api-key-revoke',
    apiKind: '鉴权管理',
    module: '鉴权',
    name: '撤销 API Key',
    method: 'DELETE',
    url: '/api/v1/api-keys/{id}',
    auth: commonApiAuth,
    backendSource: 'omcgo/internal/admin/apikey_handler.go',
    oldMapping: '老系统无 API Key 管理接口',
    coverage: '当前新增',
    fieldContract: '完整契约',
    responseFields: ['ret', 'msg', 'data=null'],
    compatibilityNote: '撤销后该 key 不再可用于 X-API-Key 鉴权。',
    requestExample: `DELETE /api/v1/api-keys/5f8f33aa-6e7d-4f53-a5a6-b2c100010001
Authorization: Bearer <token>`,
    responseExample: `{
  "ret": 1,
  "msg": "api key revoked",
  "data": null
}`,
  },
  {
    key: 'nb-sync-full',
    apiKind: '正式北向',
    module: '同步',
    name: '全量同步',
    method: 'GET',
    url: '/api/v1/northbound/sync/full?data_type=device&format=json',
    auth: commonApiAuth,
    backendSource: 'omcgo/internal/northbound/router.go + internal/northbound/sync/service.go',
    oldMapping: '/v1/device/query、/v1/cpe/infos/{sn}、/v1/enodeb/infos/status/{sn}',
    coverage: '部分覆盖',
    fieldContract: '当前契约',
    responseFields: [...currentEnvelopeFields, 'data.data_type', 'data.items[]', 'data.total', 'data.synced_at', 'data.truncated?'],
    compatibilityNote: '当前只支持 data_type=device/alarm/pm；items 字段随类型返回 Device、Alarm 或 PMCounter，不兼容老系统按 sn 的 Map 透传返回。',
    requestExample: `GET /api/v1/northbound/sync/full?data_type=device&format=json
X-API-Key: <api-key>`,
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
    oldMapping: '老系统无统一增量同步接口；参数/任务类异步接口通过 /v1/job/result/{jobId} 查询状态',
    coverage: '当前新增',
    fieldContract: '当前契约',
    responseFields: [...currentEnvelopeFields, 'data.data_type', 'data.items[]', 'data.total', 'data.synced_at', 'data.truncated?'],
    compatibilityNote: '当前只支持 data_type=alarm/pm；since 必须为 RFC3339。',
    requestExample: `GET /api/v1/northbound/sync/incremental?data_type=alarm&since=2026-07-30T00:00:00Z&format=json
X-API-Key: <api-key>`,
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
    oldMapping: '老系统 northboundApi 无 PM REST 导出；PM 文件由北向文件模块生成',
    coverage: '当前新增',
    fieldContract: '完整契约',
    responseFields: [...listResponseFields, 'data.items[].time', 'data.items[].device_id', 'data.items[].oui', 'data.items[].device_sn', 'data.items[].cell_id', 'data.items[].counter_group', 'data.items[].counter_name', 'data.items[].counter_value', 'data.items[].granularity', 'data.items[].statis_type?', 'data.items[].unit?'],
    compatibilityNote: '该端点只导出 counter；KPI 导出方法在代码中存在但当前 router 未挂载，不能在页面承诺为正式北向接口。',
    requestExample: `POST /api/v1/northbound/export/pm
X-API-Key: <api-key>
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
    oldMapping: '老系统 northboundApi 无统一告警导出接口；告警外送主要在 Socket/SNMP/File',
    coverage: '当前新增',
    fieldContract: '完整契约',
    responseFields: [...listResponseFields, 'data.items[].id', 'data.items[].device_id', 'data.items[].device_sn', 'data.items[].carrier', 'data.items[].severity', 'data.items[].alarm_type', 'data.items[].alarm_identifier', 'data.items[].description', 'data.items[].status', 'data.items[].raised_at', 'data.items[].acknowledged_at?', 'data.items[].cleared_at?', 'data.items[].device_name?', 'data.items[].technology?', 'data.items[].additional_info?'],
    compatibilityNote: 'severity 支持 1-4 和 31001-31004；非超管必须指定 device_sn。',
    requestExample: `POST /api/v1/northbound/export/alarms
X-API-Key: <api-key>
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
    name: '配置快照导出',
    method: 'GET',
    url: '/api/v1/northbound/export/config/{deviceId}',
    auth: commonApiAuth,
    backendSource: 'omcgo/internal/northbound/config_handler.go',
    oldMapping: '/v1/device/parameters/{sn}',
    coverage: '部分覆盖',
    fieldContract: '完整契约',
    responseFields: [...currentEnvelopeFields, 'data.device_id', 'data.parameters[].device_id', 'data.parameters[].parameter_path', 'data.parameters[].parameter_value', 'data.parameters[].parameter_type', 'data.parameters[].writable', 'data.parameters[].last_updated_at', 'data.parameters[].fap_instance', 'data.parameters[].param_group', 'data.total'],
    compatibilityNote: '当前按 device UUID 查询，返回 device_parameters 当前快照；老系统按 sn 查询且字段命名不同。',
    requestExample: `GET /api/v1/northbound/export/config/9a4d7b2f-2b2c-4f0a-9ec5-5e9b3a8c1001
X-API-Key: <api-key>`,
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
    oldMapping: '老系统无 HTTP Push 目标管理接口',
    coverage: '当前新增',
    fieldContract: '完整契约',
    responseFields: [...currentEnvelopeFields, 'data.items[].id', 'data.items[].url', 'data.items[].auth_type', 'data.items[].auth_token', 'data.items[].data_types', 'data.items[].format', 'data.items[].batch_size', 'data.items[].retry_count', 'data.items[].enabled', 'data.total'],
    compatibilityNote: '用于 HTTP Push fanout 目标管理；与文件 FTP/SFTP 传输目标分开。',
    requestExample: `GET /api/v1/northbound/push/targets
X-API-Key: <api-key>`,
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
    oldMapping: '老系统无 HTTP Push 目标管理接口',
    coverage: '当前新增',
    fieldContract: '完整契约',
    responseFields: [...currentEnvelopeFields, 'data.message', 'data.id'],
    compatibilityNote: '目标开关默认应保持关闭，配置验证通过后再启用。',
    requestExample: `POST /api/v1/northbound/push/targets
X-API-Key: <api-key>
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
    oldMapping: '老系统无 HTTP Push 目标管理接口',
    coverage: '当前新增',
    fieldContract: '完整契约',
    responseFields: [...currentEnvelopeFields, 'data.id'],
    compatibilityNote: '目标不存在返回 404。',
    requestExample: `DELETE /api/v1/northbound/push/targets/oss-primary
X-API-Key: <api-key>`,
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
    oldMapping: '老系统无 HTTP Push 熔断状态接口',
    coverage: '当前新增',
    fieldContract: '完整契约',
    responseFields: [...currentEnvelopeFields, 'data.target_id', 'data.state', 'data.failure_count', 'data.threshold'],
    compatibilityNote: 'state 来自 reliability circuit breaker，典型值为 closed/open/half-open。',
    requestExample: `GET /api/v1/northbound/push/targets/oss-primary/circuit
X-API-Key: <api-key>`,
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
    oldMapping: '老系统无 HTTP Push 熔断状态接口',
    coverage: '当前新增',
    fieldContract: '完整契约',
    responseFields: [...currentEnvelopeFields, 'data.target_id', 'data.state'],
    compatibilityNote: '用于目标恢复后人工关闭熔断状态。',
    requestExample: `POST /api/v1/northbound/push/targets/oss-primary/circuit/reset
X-API-Key: <api-key>`,
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
    oldMapping: '老系统无 HTTP Push 死信接口',
    coverage: '当前新增',
    fieldContract: '完整契约',
    responseFields: [...currentEnvelopeFields, 'data.items[].id', 'data.items[].event_id', 'data.items[].subject', 'data.items[].payload', 'data.items[].target_id', 'data.items[].status', 'data.items[].attempts', 'data.items[].max_attempts', 'data.items[].last_error?', 'data.items[].next_retry_at', 'data.items[].created_at', 'data.items[].updated_at', 'data.total'],
    compatibilityNote: 'limit 范围 1-100，未配置 outbox 时返回 503。',
    requestExample: `GET /api/v1/northbound/push/deadletter?limit=20&offset=0
X-API-Key: <api-key>`,
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
    oldMapping: '老系统无 HTTP Push 死信接口',
    coverage: '当前新增',
    fieldContract: '完整契约',
    responseFields: [...currentEnvelopeFields, 'data.id'],
    compatibilityNote: '仅对 dead 状态 outbox 条目执行重放。',
    requestExample: `POST /api/v1/northbound/push/deadletter/4a65c2aa-086b-4c1b-a3b7-000100010001/replay
X-API-Key: <api-key>`,
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
    oldMapping: '老系统无统一北向主备服务器配置接口',
    coverage: '当前新增',
    fieldContract: '完整契约',
    responseFields: [...currentEnvelopeFields, 'data.items[].id', 'data.items[].role', 'data.items[].host', 'data.items[].port', 'data.items[].description', 'data.items[].is_active', 'data.items[].created_at', 'data.items[].updated_at'],
    compatibilityNote: '仅在 ServerService 注入后挂载；未注入时该组端点为 404。',
    requestExample: `GET /api/v1/northbound/servers
X-API-Key: <api-key>`,
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
    oldMapping: '老系统无统一北向主备服务器配置接口',
    coverage: '当前新增',
    fieldContract: '完整契约',
    responseFields: [...currentEnvelopeFields, 'data.role'],
    compatibilityNote: 'role 仅允许 primary 或 standby。',
    requestExample: `PUT /api/v1/northbound/servers/active
X-API-Key: <api-key>
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
    oldMapping: '老系统无统一北向主备服务器配置接口',
    coverage: '当前新增',
    fieldContract: '完整契约',
    responseFields: [...currentEnvelopeFields, 'data.role'],
    compatibilityNote: '只编辑 host/port/description；激活状态必须通过 /servers/active 修改。',
    requestExample: `PUT /api/v1/northbound/servers/primary
X-API-Key: <api-key>
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
    name: '设备列表查询',
    method: 'GET',
    url: '/api/v1/devices?page=1&page_size=20&technology=LTE&sn=1202000240194',
    auth: 'JWT 或 API Key，权限 scope=devices:read',
    backendSource: 'omcgo/internal/device/device_handler.go',
    requestExample: `GET /api/v1/devices?page=1&page_size=20&technology=LTE&sn=1202000240194
Authorization: Bearer <token>`,
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
    name: '设备详情查询',
    method: 'GET',
    url: '/api/v1/devices/{id}/detail',
    auth: 'JWT 或 API Key，权限 scope=devices:read',
    backendSource: 'omcgo/internal/device/device_info_handler.go',
    requestExample: `GET /api/v1/devices/9a4d7b2f-2b2c-4f0a-9ec5-5e9b3a8c1001/detail
Authorization: Bearer <token>`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "device": {
      "id": "9a4d7b2f-2b2c-4f0a-9ec5-5e9b3a8c1001",
      "serial_number": "1202000240194",
      "site_name": "SH-001",
      "firmware_version": "BaiBS_QRTB_2.9"
    },
    "device_info": {
      "op_state": "enabled",
      "mme_status": "connected",
      "ue_count": 32,
      "cell_id": "1"
    }
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
Authorization: Bearer <token>`,
    responseExample: `HTTP/1.1 200 OK
Content-Type: text/csv; charset=utf-8
Content-Disposition: attachment; filename=devices_20260730_010000.csv

serial_number,product_class,site_name,status,last_inform_at
1202000240194,FAP-LTE-100,SH-001,online,2026-07-30T01:00:00+08:00`,
  },
  {
    key: 'parameter-tree',
    module: '配置',
    name: '参数树快照',
    method: 'GET',
    url: '/api/v1/devices/{id}/parameters/tree?root=Device.DeviceInfo',
    auth: 'JWT 或 API Key，权限 scope=devices:read',
    backendSource: 'omcgo/internal/device/device_param_handler.go',
    requestExample: `GET /api/v1/devices/9a4d7b2f-2b2c-4f0a-9ec5-5e9b3a8c1001/parameters/tree?root=Device.DeviceInfo
Authorization: Bearer <token>`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "name": "DeviceInfo",
    "full_path": "Device.DeviceInfo.",
    "is_leaf": false,
    "children": [
      {
        "name": "SoftwareVersion",
        "full_path": "Device.DeviceInfo.SoftwareVersion",
        "value": "BaiBS_QRTB_2.9",
        "type": "string",
        "writable": false
      }
    ]
  }
}`,
  },
  {
    key: 'parameter-set',
    module: '配置',
    name: '参数异步下发',
    method: 'PUT',
    url: '/api/v1/devices/{id}/parameters',
    auth: 'JWT 或 API Key，权限 scope=devices:write',
    backendSource: 'omcgo/internal/device/device_param_handler.go',
    requestExample: `PUT /api/v1/devices/9a4d7b2f-2b2c-4f0a-9ec5-5e9b3a8c1001/parameters
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
    "reboot_required": false,
    "task_id": "cmd-20260730010000-0001"
  }
}`,
  },
  {
    key: 'config-pull',
    module: '配置',
    name: '配置参数拉取',
    method: 'POST',
    url: '/api/v1/config/sync/pull/{device_sn}',
    auth: 'JWT 或 API Key，权限 scope=config:write',
    backendSource: 'omcgo/internal/config/sync_handler.go',
    requestExample: `POST /api/v1/config/sync/pull/1202000240194
Idempotency-Key: nb-pull-20260730010000
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
    "device_id": "1202000240194",
    "command_id": "run-20260730010000-0001",
    "request_id": "nb-pull-20260730010000",
    "batch_count": 1,
    "status": "queued"
  }
}`,
  },
  {
    key: 'task-detail',
    module: '任务',
    name: '异步任务结果查询',
    method: 'GET',
    url: '/api/v1/devices/tasks/{task_id}',
    auth: 'JWT 或 API Key，权限 scope=devices:read',
    backendSource: 'omcgo/internal/task/handler.go',
    requestExample: `GET /api/v1/devices/tasks/cmd-20260730010000-0001
Authorization: Bearer <token>`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "id": "cmd-20260730010000-0001",
    "device_sn": "1202000240194",
    "method": "SetParameterValues",
    "status": "completed",
    "result_code": "0",
    "completed_at": "2026-07-30T01:00:04+08:00"
  }
}`,
  },
  {
    key: 'device-reboot',
    module: '设备',
    name: '设备重启',
    method: 'POST',
    url: '/api/v1/devices/{id}/reboot',
    auth: 'JWT 或 API Key，权限 scope=devices:write',
    backendSource: 'omcgo/internal/device/device_handler.go',
    requestExample: `POST /api/v1/devices/9a4d7b2f-2b2c-4f0a-9ec5-5e9b3a8c1001/reboot
Authorization: Bearer <token>`,
    responseExample: `{
  "ret": 1,
  "msg": "ok",
  "data": {
    "message": "reboot command queued"
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
Authorization: Bearer <token>`,
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
Authorization: Bearer <token>`,
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
Authorization: Bearer <token>`,
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
Authorization: Bearer <token>`,
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
    oldMapping: '/v1/device/query、/v1/cpe/infos/{sn}',
    coverage: '部分覆盖',
    fieldContract: '当前契约',
    responseFields: [...listResponseFields, 'data.items[].id', 'data.items[].serial_number', 'data.items[].oui', 'data.items[].product_class', 'data.items[].manufacturer', 'data.items[].model_name', 'data.items[].carrier', 'data.items[].technology', 'data.items[].lifecycle_state', 'data.items[].is_online', 'data.items[].firmware_version', 'data.items[].ip_address', 'data.items[].device_name', 'data.items[].site_id', 'data.items[].last_inform_at?', 'data.items[].group_name?'],
    compatibilityNote: '当前设备列表按 /api/v1/devices 返回 Device/ListResponse；老系统按 /v1/device/query 透传下游对象，URL 和字段命名不兼容。',
  },
  'device-detail': {
    apiKind: '业务复用',
    oldMapping: '/v1/cpe/infos/{sn}、/v1/enodeb/infos/status/{sn}',
    coverage: '部分覆盖',
    fieldContract: '当前契约',
    responseFields: [...currentEnvelopeFields, 'data.device', 'data.device_info', 'data.parameters?', 'data.alarms?'],
    compatibilityNote: '当前按 device UUID 查询详情；老系统多数接口按 sn 查询。',
  },
  'inventory-export': {
    apiKind: '业务复用',
    oldMapping: '老 northboundApi 无 Inventory REST；对应当前 Inventory 文件/设备清单导出能力',
    coverage: '当前新增',
    fieldContract: '当前契约',
    responseFields: ['HTTP 200 CSV stream', 'Content-Type', 'Content-Disposition', 'CSV columns depend on export service'],
    compatibilityNote: '该接口返回 CSV 文件流，不使用 ret/msg/data JSON envelope。',
  },
  'parameter-tree': {
    apiKind: '业务复用',
    oldMapping: '/v1/device/parameters/{sn}',
    coverage: '部分覆盖',
    fieldContract: '当前契约',
    responseFields: [...currentEnvelopeFields, 'data.name', 'data.full_path', 'data.is_leaf', 'data.children[]', 'data.children[].name', 'data.children[].full_path', 'data.children[].value?', 'data.children[].type?', 'data.children[].writable?'],
    compatibilityNote: '当前参数树按 device UUID + root 查询，老系统按 sn 查询参数快照。',
  },
  'parameter-set': {
    apiKind: '业务复用',
    oldMapping: '/v1/device/parameters/{sn}、/v1/device/parameters/cellname/{sn}',
    coverage: '部分覆盖',
    fieldContract: '当前契约',
    responseFields: [...currentEnvelopeFields, 'data.message', 'data.parameters', 'data.reboot_required', 'data.task_id'],
    compatibilityNote: '当前异步任务 ID 字段为 task_id；老系统通常返回 jobId、sn、waitTime。',
  },
  'config-pull': {
    apiKind: '业务复用',
    oldMapping: '/v1/device/parameters/query/{sn}',
    coverage: '部分覆盖',
    fieldContract: '当前契约',
    responseFields: [...currentEnvelopeFields, 'data.device_id', 'data.command_id', 'data.request_id', 'data.batch_count', 'data.status'],
    compatibilityNote: '当前用于触发 ACS 参数拉取；完整结果需结合任务结果/参数快照查询。',
  },
  'task-detail': {
    apiKind: '业务复用',
    oldMapping: '/v1/job/result/{jobId}',
    coverage: '部分覆盖',
    fieldContract: '当前契约',
    responseFields: [...currentEnvelopeFields, 'data.id', 'data.device_id?', 'data.device_sn?', 'data.method?', 'data.status', 'data.result_code?', 'data.error_message?', 'data.created_at?', 'data.completed_at?'],
    compatibilityNote: '当前任务模型不直接兼容老 JobInfo 字段；如果 OSS 要老字段，需要增加适配 facade。',
  },
  'device-reboot': {
    apiKind: '业务复用',
    oldMapping: '/v1/device/reboot/{sn}',
    coverage: '部分覆盖',
    fieldContract: '当前契约',
    responseFields: [...currentEnvelopeFields, 'data.message'],
    compatibilityNote: '当前按 device UUID 触发 reboot；老系统按 sn。',
  },
  'alarm-active': {
    apiKind: '业务复用',
    oldMapping: '老 northboundApi 无活动告警查询；告警北向来自 Socket/SNMP/File',
    coverage: '当前新增',
    fieldContract: '当前契约',
    responseFields: [...listResponseFields, 'data.items[].id', 'data.items[].device_sn', 'data.items[].severity', 'data.items[].alarm_type', 'data.items[].alarm_identifier', 'data.items[].description', 'data.items[].status', 'data.items[].raised_at'],
    compatibilityNote: '建议 OSS 查询告警优先使用正式北向 /api/v1/northbound/export/alarms。',
  },
  'alarm-statistics': {
    apiKind: '业务复用',
    oldMapping: '老 northboundApi 无告警统计接口',
    coverage: '当前新增',
    fieldContract: '当前契约',
    responseFields: [...currentEnvelopeFields, 'data.total', 'data.by_severity'],
    compatibilityNote: '统计字段以 alarm handler 当前 DTO 为准。',
  },
  'pm-aggregated': {
    apiKind: '业务复用',
    oldMapping: '老 northboundApi 无 PM REST 查询；PM 文件由北向文件模块生成',
    coverage: '当前新增',
    fieldContract: '当前契约',
    responseFields: [...listResponseFields, 'data.items[].device_sn?', 'data.items[].metric_path', 'data.items[].value', 'data.items[].bucket_start', 'data.items[].technology?'],
    compatibilityNote: '该接口可查 counter/KPI 聚合结果，适合作为 UI/临时查询；正式北向 counter 导出见 /api/v1/northbound/export/pm。',
  },
  'pm-export': {
    apiKind: '业务复用',
    oldMapping: '老 northboundApi 无 PM REST 导出',
    coverage: '当前新增',
    fieldContract: '当前契约',
    responseFields: [...currentEnvelopeFields, 'data.id', 'data.task_name', 'data.source_type', 'data.format', 'data.status', 'data.row_count', 'data.file_size'],
    compatibilityNote: '这是 PM 导出任务接口，和正式北向 /northbound/export/pm 的同步 JSON 响应不同。',
  },
  'mr-data': {
    apiKind: '业务复用',
    oldMapping: '老 northboundApi 无 MR REST 查询',
    coverage: '当前新增',
    fieldContract: '当前契约',
    responseFields: [...listResponseFields, 'data.items[].device_id', 'data.items[].mr_type', 'data.items[].cell_id?', 'data.items[].collect_time?'],
    compatibilityNote: 'MR 文件北向仍按文件配置输出 XML；该接口用于查询落库后的 MR 数据。',
  },
  'mr-export': {
    apiKind: '业务复用',
    oldMapping: '老 northboundApi 无 MR REST 导出',
    coverage: '当前新增',
    fieldContract: '当前契约',
    responseFields: ['HTTP 200 CSV stream', 'Content-Type', 'Content-Disposition', 'CSV columns depend on MR export service'],
    compatibilityNote: '该接口返回 CSV 文件流，不使用 ret/msg/data JSON envelope。',
  },
};

const legacySupportedApiKeys = new Set([
  'auth-login',
  'device-list',
  'device-detail',
  'nb-export-config',
  'parameter-tree',
  'parameter-set',
  'config-pull',
  'task-detail',
  'device-reboot',
]);

const legacySupportedApiRows = northboundApiRows.filter((row) => legacySupportedApiKeys.has(row.key));

const defaultApiEnabled = Object.fromEntries(
  legacySupportedApiRows.map((row) => [row.key, false]),
) as Record<string, boolean>;

const defaultEditorPeriodRows: ScenarioPeriodRow[] = [
  {
    key: 'cm-default',
    domain: 'CM',
    scope: 'LTE',
    format: 'CSV',
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

function parseObjectToken(token: string): { code: string; tech?: Tech } {
  const [code, tech] = token.split('/').map((item) => item.trim());
  return {
    code,
    tech: isTech(tech) ? tech : undefined,
  };
}

function formatObjectToken(token: string): string {
  const parsed = parseObjectToken(token);
  return parsed.tech ? `${parsed.code} / ${parsed.tech}` : parsed.code;
}

function formatFieldTargetLabel(target: FieldTarget): string {
  return target.objectCode;
}

function formatReportFieldObject(row: ReportFieldRow): string {
  return row.objectCode;
}

function resolveTechLabel(value?: string): Tech | undefined {
  return isTech(value) ? value : undefined;
}

function getTargetTech(target: FieldTarget | undefined): Tech | undefined {
  return target?.tech ?? resolveTechLabel(target?.scope);
}

function getRowTech(row: ReportFieldRow): Tech | undefined {
  return row.tech ?? resolveTechLabel(row.scope);
}

function formatReportFieldTech(row: ReportFieldRow): string {
  return getRowTech(row) ?? '通用';
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
  const scopeText = scope.toUpperCase();
  const techs: Tech[] = [];
  if (scopeText.includes('LTE')) techs.push('LTE');
  if (scopeText.includes('GNB')) techs.push('GNB');
  if (scopeText.includes('GSM')) techs.push('GSM');
  return techs.length > 0 ? [...new Set(techs)] : ['LTE'];
}

function resolveEditorProfile(row: ScenarioPeriodRow, objectCode: string, tech?: Tech): string {
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
      const techs = parsedObject.tech
        ? [parsedObject.tech]
        : row.domain === 'PM'
          ? getPmTechsFromScope(row.scope)
          : row.domain === 'CM' && isTech(row.scope)
            ? [row.scope]
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

function getPmFieldRows(target: FieldTarget, metrics: PmMetric[], limit?: number): ReportFieldRow[] {
  const tech = getTargetTech(target);
  if (!tech) return [];
  const profileMetrics = metrics.filter((metric) => metric.tech === tech && metric.profile === target.profile);
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
  return typeof limit === 'number' ? rows.slice(0, limit) : rows;
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

function dedupeReportRows(rows: ReportFieldRow[]): ReportFieldRow[] {
  const seen = new Set<string>();
  return rows.filter((row) => {
    const id = fieldIdentity(row);
    if (seen.has(id)) return false;
    seen.add(id);
    return true;
  });
}

function getAvailableReportFieldRows(target: FieldTarget | undefined, metrics: PmMetric[]): ReportFieldRow[] {
  if (!target) return [];
  if (target.domain === 'PM') return getPmFieldRows(target, metrics);
  return dedupeReportRows([...getReportFieldRows(target, metrics), ...getExtraFieldRows(target)]);
}

function formatProductClasses(productClasses?: string[]): string {
  if (!productClasses || productClasses.length === 0 || productClasses.includes(allProductClassValue)) return '通用';
  if (productClasses.length <= 2) return productClasses.join(', ');
  return `${productClasses.slice(0, 2).join(', ')} +${productClasses.length - 2}`;
}

function fieldIdentity(row: ReportFieldRow): string {
  return `${row.domain}:${row.objectCode}:${row.outputAlias}:${row.systemField}`;
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

function defaultPeriodRow(domain: Domain = 'CM'): ScenarioPeriodRow {
  const firstObjects = objectOptionsByDomain[domain].slice(0, domain === 'CM' ? 4 : 1).map((option) => option.value);
  const periodByDomain: Record<Domain, string> = {
    CM: '24H',
    PM: '15M',
    MR: '15M',
    LOG: '24H',
    INVENTORY: '24H',
  };
  const period = periodByDomain[domain];
  const cron = defaultCronForPeriod(period);
  return {
    key: `custom-${Date.now()}`,
    domain,
    scope: domain === 'CM' || domain === 'PM' ? 'LTE' : '默认',
    format: defaultFormatByDomain[domain],
    period,
    trigger: formatScheduleLabel(period, cron),
    cron,
    objects: joinObjects(firstObjects),
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
    compressionEnabled: false,
    compressionFormat: domain === 'LOG' ? 'gz' : 'zip',
  };
}

function getGroupTechLabel(groupItem: FileGroup): string {
  const techs = [...new Set(groupItem.objects.map((object) => object.tech).filter(Boolean))];
  if (techs.length > 0) return techs.join('/');
  if (groupItem.domain === 'CM' || groupItem.domain === 'PM') return 'LTE';
  return '默认';
}

function getGroupObjectsLabel(groupItem: FileGroup): string {
  return [...new Set(groupItem.objects.map((object) => object.code))].join(', ');
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
  return period.replace(/\D/g, '') || period;
}

function formatPeriodLabel(period: string): string {
  if (period === '15M') return '15 分钟';
  if (period === '60M') return '60 分钟';
  if (period === '24H') return '每日';
  return period;
}

function getPeriodMinutes(period: string): number {
  if (period === '60M') return 60;
  if (period === '15M') return 15;
  return Number(period.replace(/\D/g, '')) || 15;
}

function defaultCronForPeriod(period: string): string {
  if (period === '24H') return '0 1 0 * * ?';
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
  return `0 ${Number(startMinute)}/${getPeriodMinutes(period)} * * * ?`;
}

function formatScheduleLabel(period: string, cron: string): string {
  if (period === '24H') return `每日 ${getDailyTimeValue(cron)} 生成`;
  return `每 ${getPeriodMinutes(period)} 分钟，${getIntervalStartMinuteValue(cron)} 分开始生成`;
}

function normalizeApiPeriod(period: string | undefined): NorthboundPageConfigPeriod {
  const value = period ?? '24H';
  return ['15M', '60M', '24H', '7D', '1MO'].includes(value)
    ? value as NorthboundPageConfigPeriod
    : '24H';
}

function normalizeApiFormat(domain: Domain, format: string | undefined): NorthboundPageConfigFormat {
  return normalizeFormatForDomain(domain, (format ?? defaultFormatByDomain[domain]) as Format) as NorthboundPageConfigFormat;
}

function normalizeApiCompressionFormat(format: string | undefined): NorthboundPageConfigCompressionFormat {
  return format === 'gz' ? 'gz' : 'zip';
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
        return {
          id: item.id,
          domain,
          format: normalizeApiFormat(domain, item.format),
          period,
          cron: cronFromPeriodStartMinute(period, item.start_minute),
          path: item.path_template,
          name: item.file_name_template,
          csvSeparator: item.csv_separator,
          compressionEnabled: item.compression_enabled,
          compressionFormat: normalizeApiCompressionFormat(item.compression_format),
          objects: item.objects.map(mapApiScenarioObject),
          selectedFields: item.selected_fields,
        };
      })
    : fallback?.groups ?? [];

  return {
    code: profile.code,
    vendor: profile.vendor || fallback?.vendor || 'Baicells',
    scenarioName: profile.scenario_name || fallback?.scenarioName || profile.name,
    scenarioNameEn: profile.scenario_name_en || fallback?.scenarioNameEn || profile.code,
    description: profile.description || fallback?.description || profile.name,
    flags: profile.flags?.length ? profile.flags : fallback?.flags ?? [],
    name: profile.name || fallback?.name || profile.code,
    enabled: profile.enabled,
    groups,
    logs: fallback?.logs,
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

function scenarioEnabledMap(rows: ScenarioRow[]): Record<string, boolean> {
  return Object.fromEntries(rows.map((row) => [row.code, row.enabled]));
}

function getNextCustomScenarioCode(rows: ScenarioRow[]): string {
  const usedCodes = new Set(rows.map((row) => row.code.toUpperCase()));
  for (let index = 9001; index <= 9999; index += 1) {
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
    credential: target.credential_set ? '已加密存储' : '',
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
      credential: row.credential === '已加密存储' ? '' : row.credential,
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

function mapApiSnmpTarget(target: NorthboundSNMPAlarmTarget): SnmpAlarmTargetRow {
  const fallback = snmpAlarmTargets.find((row) => row.key === target.key);
  return {
    key: target.key,
    name: target.name || fallback?.name || target.key,
    version: target.version,
    notificationType: target.notification_type,
    listenIp: target.listen_ip,
    listenPort: target.listen_port,
    targetHost: target.target_host,
    targetPort: target.target_port,
    community: target.community_set ? '已加密存储' : target.community,
    securityName: target.security_name,
    authProtocol: target.auth_protocol,
    authCredential: target.auth_credential_set ? '已加密存储' : target.auth_credential,
    privProtocol: target.priv_protocol,
    privCredential: target.priv_credential_set ? '已加密存储' : target.priv_credential,
    mibQueryEnabled: target.version === 'v2' ? target.mib_query_enabled : false,
    clearSeverityPolicy: target.clear_severity_policy,
    timeoutSeconds: target.timeout_seconds,
    retries: target.retries,
  };
}

function serializeSnmpTarget(row: SnmpAlarmTargetRow, enabled: boolean): NorthboundSNMPAlarmTarget {
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
    community: row.community === '已加密存储' ? '' : row.community,
    security_name: row.securityName,
    auth_protocol: row.authProtocol,
    auth_credential: row.authCredential === '已加密存储' ? '' : row.authCredential,
    priv_protocol: row.privProtocol,
    priv_credential: row.privCredential === '已加密存储' ? '' : row.privCredential,
    clear_severity_policy: row.clearSeverityPolicy,
    mib_query_enabled: row.version === 'v2' ? row.mibQueryEnabled : false,
    timeout_seconds: row.timeoutSeconds,
    retries: row.retries,
  };
}

function getSnmpEnableBlocker(row: SnmpAlarmTargetRow): string | null {
  if (!row.targetHost.trim()) {
    return '请先编辑通知目标 IP/域名，再启用真实上报';
  }
  if (row.version === 'v2' && !row.community?.trim()) {
    return '请先编辑 SNMP v2 community，再启用真实上报';
  }
  if (row.version === 'v3' && !row.securityName?.trim()) {
    return '请先编辑 SNMP v3 security name，再启用真实上报';
  }
  return null;
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
    credential: account.credential_set ? '已加密存储' : '',
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
      credential: account.credential === '已加密存储' ? '' : account.credential,
      purpose: account.purpose,
    })),
  };
}

function mapApiConfig(row: NorthboundAPIConfig): NorthboundApiRow {
  const fallback = northboundApiRows.find((item) => item.key === row.key);
  return {
    key: row.key,
    apiKind: (row.kind as ApiKind) || fallback?.apiKind,
    module: fallback?.module ?? row.data_type,
    name: row.name,
    method: row.method,
    url: row.path,
    auth: fallback?.auth ?? commonApiAuth,
    backendSource: row.source || fallback?.backendSource || '',
    oldMapping: fallback?.oldMapping,
    coverage: fallback?.coverage ?? '旧能力对齐',
    fieldContract: fallback?.fieldContract ?? '当前契约',
    responseFields: (row.response_contract?.fields as string[] | undefined) ?? fallback?.responseFields,
    compatibilityNote: fallback?.compatibilityNote,
    requestExample: fallback?.requestExample ?? `${row.method} ${row.path}`,
    responseExample: fallback?.responseExample ?? '{ "ret": 1, "msg": "ok", "data": {} }',
  };
}

function mapApiClient(row: NorthboundAPIClient): ApiClientRow {
  return {
    clientKey: row.client_key,
    name: row.name || row.client_key,
    enabled: row.enabled,
    tokenSecret: row.token_set ? '已加密存储' : '',
    tokenSet: Boolean(row.token_set),
    allowedApiKeys: (row.allowed_api_keys ?? []).join('\n'),
    ipWhitelist: (row.ip_whitelist ?? []).join('\n'),
    expiresAt: row.expires_at,
  };
}

function serializeApiClients(rows: ApiClientRow[]) {
  return {
    items: rows.map((row) => ({
      client_key: row.clientKey.trim(),
      name: row.name.trim() || row.clientKey.trim(),
      enabled: row.enabled,
      token_secret: row.tokenSecret === '已加密存储' ? '' : row.tokenSecret.trim(),
      allowed_api_keys: splitTextareaList(row.allowedApiKeys),
      ip_whitelist: splitTextareaList(row.ipWhitelist),
      expires_at: row.expiresAt?.trim() || undefined,
    })),
  };
}

function splitTextareaList(value: string): string[] {
  return value
    .split(/[\n,，]+/)
    .map((item) => item.trim())
    .filter(Boolean);
}

// Derives the field-target keys (domain:objectCode:scope:profile) that belong to a
// single period row / group. Mirrors getFieldTargets so lookups into fieldRowsByTarget
// match exactly.
function periodRowTargetKeys(row: ScenarioPeriodRow): string[] {
  const keys: string[] = [];
  splitObjects(row.objects).forEach((objectToken) => {
    const parsedObject = parseObjectToken(objectToken);
    const techs = parsedObject.tech
      ? [parsedObject.tech]
      : row.domain === 'PM'
        ? getPmTechsFromScope(row.scope)
        : row.domain === 'CM' && isTech(row.scope)
          ? [row.scope]
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
    const format = normalizeApiFormat(row.domain, row.format);
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
        const tech = parsed.tech ?? (isTech(row.scope) ? row.scope : undefined);
        return {
          code: parsed.code,
          tech,
          profile: resolveEditorProfile(row, parsed.code, tech),
        };
      }),
    };
  });
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
  const normalizedFormat = normalizeFormatForDomain(row.domain, row.format);
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
    capabilityName: row.name,
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
  return {
    key: `api:${row.key}`,
    capabilityName: row.name,
    state: 'success',
    statusText: '契约正常',
    lastTime: '-',
    artifactType: 'message',
    artifactName: `${row.method} ${row.name}`,
    artifactPath: row.url,
    size: `${(row.responseFields ?? meta.responseFields ?? []).length} 字段`,
    targetSummary: meta.coverage,
    detail: meta.compatibilityNote,
    payload: JSON.stringify({
      method: row.method,
      url: row.url,
      request: normalizeLegacyApiText(row.requestExample),
      response_fields: row.responseFields ?? meta.responseFields ?? currentEnvelopeFields,
    }, null, 2),
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
    targetSummary: `生成 ${run.row_count} 行，可在传输目标中测试连接`,
    detail: run.error_message || `${run.profile_code} / ${run.group_id || run.object_code} 手动生成记录。`,
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
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
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
  const payload = event.payload || JSON.stringify(event.summary ?? {}, null, 2);
  return {
    key: `event:${event.id}`,
    capabilityName: fallbackCapabilityName,
    state,
    statusText: eventStatusText(event),
    lastTime: formatRunTime(event.created_at),
    artifactType: event.artifact_type === 'file' ? 'file' : 'message',
    artifactName: event.artifact_name || '-',
    artifactPath: event.artifact_path || '-',
    size: formatBytes(payload.length),
    targetSummary: eventTargetSummary(event),
    detail: event.error_message || eventDetailText(event),
    payload,
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
    contract_check: '契约检查',
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
  if (event.event_type === 'contract_check') return '接口契约检查已记录';
  if (event.event_type === 'run') return '运行记录已生成';
  return '状态已记录';
}

function eventDetailText(event: NorthboundPageConfigEvent): string {
  if (event.capability === 'delivery' && event.event_type === 'delivery') return 'FTP/SFTP 文件投递结果来自后端，包含远端路径、尝试次数和耗时。';
  if (event.capability === 'delivery') return 'FTP/SFTP 目标连接探测结果来自后端。';
  if (event.capability === 'snmp') return 'SNMP 告警字段和 OID 顺序按 omcAlarmMIB.mib 生成。';
  if (event.capability === 'socket') return 'Socket 服务端登录、心跳、同步和实时告警推送结果可查看。';
  if (event.capability === 'api') return '北向 API 仅展示老系统支持且当前 xomc 能满足的接口契约。';
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
  const rows: ScenarioPeriodRow[] = scenario.groups.map((groupItem) => ({
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
      objects: scenario.logs.mode === 'custom' ? '登录日志, 操作日志' : '登录固定格式, 操作固定格式',
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
    existing.scopes.push(periodRow.scope);
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
    .map((periodRow) => `${periodRow.domain} ${periodRow.scope}: ${formatPeriodLabel(periodRow.period)} / ${periodRow.trigger}`)
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
    oldMapping: row.oldMapping ?? meta.oldMapping ?? '当前 xomc 接口，老系统无直接对应',
    coverage: row.coverage ?? meta.coverage ?? '部分覆盖',
    fieldContract: row.fieldContract ?? meta.fieldContract ?? '当前契约',
    responseFields: row.responseFields ?? meta.responseFields ?? currentEnvelopeFields,
    compatibilityNote: row.compatibilityNote ?? meta.compatibilityNote ?? '以当前 xomc 后端 DTO 为准。',
  };
}

function normalizeLegacyApiText(value: string) {
  return value
    .replaceAll('JWT 或 X-API-Key；/api/v1 统一鉴权 + Casbin 端点权限', 'JWT；/api/v1 统一鉴权 + Casbin 端点权限')
    .replaceAll('JWT 或 API Key，', 'JWT，')
    .replaceAll('X-API-Key: <api-key>', 'Authorization: Bearer <token>');
}

function apiKindTag(value: ApiKind) {
  const colors: Record<ApiKind, string> = {
    正式北向: 'blue',
    业务复用: 'cyan',
    鉴权管理: 'geekblue',
  };
  return <Tag color={colors[value]}>{value}</Tag>;
}

function statusTag(enabled: boolean) {
  return enabled ? <Tag color="success">启用</Tag> : <Tag color="default">关闭</Tag>;
}

function reportStateTag(state: ReportState, label: string) {
  const colors: Record<ReportState, string> = {
    success: 'success',
    failed: 'error',
    running: 'processing',
    idle: 'default',
  };
  return <Tag color={colors[state]}>{label}</Tag>;
}

function deliveryProtocolTag(protocol: DeliveryProtocol) {
  return <Tag color={protocol === 'SFTP' ? 'blue' : 'cyan'}>{protocol}</Tag>;
}

function socketProfileTag(profile: SocketProfile) {
  return <Tag color={profile === 'CTCC' ? 'blue' : 'purple'}>{profile === 'CTCC' ? '电信' : '联通'}</Tag>;
}

function snmpVersionTag(version: SnmpVersion) {
  const colors: Record<SnmpVersion, string> = {
    v2: 'blue',
    v3: 'purple',
  };
  return <Tag color={colors[version]}>{version.toUpperCase()}</Tag>;
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
    const firstTarget = getFirstFieldTarget(periodRows);
    const [fieldConfigDomain, setFieldConfigDomain] = useState<Domain>(firstTarget?.domain ?? 'CM');
    const [fieldConfigTargetKey, setFieldConfigTargetKey] = useState(firstTarget?.key ?? '');
    const [fieldTechFilter, setFieldTechFilter] = useState<FieldTechFilter>('ALL');
    const [fieldCandidateKey, setFieldCandidateKey] = useState<string>();
    const [fieldRowsByTarget, setFieldRowsByTarget] = useState<Record<string, ReportFieldRow[]>>(initialFieldRowsByTarget);
    const [backendFieldCatalog, setBackendFieldCatalog] = useState<Record<string, ReportFieldRow[]>>({});

    useImperativeHandle(ref, () => ({ getFieldRowsByTarget: () => fieldRowsByTarget }), [fieldRowsByTarget]);

    const fieldTargets = useMemo(() => getFieldTargets(periodRows), [periodRows]);
    const fieldDomainOptions = useMemo(
      () => [...new Set(fieldTargets.map((t) => t.domain))].map((d) => ({ label: d, value: d })),
      [fieldTargets],
    );
    const visibleFieldTargets = useMemo(
      () => fieldTargets.filter((t) => t.domain === fieldConfigDomain).filter((t) => targetMatchesTechFilter(t, fieldTechFilter)),
      [fieldTargets, fieldConfigDomain, fieldTechFilter],
    );
    const fieldTargetOptions = useMemo(
      () => visibleFieldTargets.map((t) => ({ label: formatFieldTargetLabel(t), value: t.key })),
      [visibleFieldTargets],
    );
    const selectedFieldTarget = useMemo(
      () => visibleFieldTargets.find((t) => t.key === fieldConfigTargetKey) ?? visibleFieldTargets[0],
      [visibleFieldTargets, fieldConfigTargetKey],
    );
    const backendCatalogKey = selectedFieldTarget ? `${selectedFieldTarget.domain}:${selectedFieldTarget.objectCode}` : '';
    useEffect(() => {
      const target = selectedFieldTarget;
      if (!target) return;
      const cacheKey = `${target.domain}:${target.objectCode}`;
      if (backendFieldCatalog[cacheKey]) return;
      let cancelled = false;
      void northboundPageConfigApi.getFields({ domain: target.domain, object: target.objectCode })
        .then((resp) => {
          if (cancelled) return;
          const rows: ReportFieldRow[] = (resp.items ?? []).map((f) => ({
            key: `${target.domain}-${target.objectCode}-${f.system_field}`,
            domain: target.domain,
            objectCode: target.objectCode,
            outputAlias: f.output_alias,
            systemField: f.system_field,
            source: f.source,
            dataType: f.data_type,
            renderer: f.renderer,
            cnName: f.cn_name,
            enabled: true,
          }));
          setBackendFieldCatalog((prev) => (prev[cacheKey] ? prev : { ...prev, [cacheKey]: rows }));
        })
        .catch(() => { /* fall back to static catalog */ });
      return () => { cancelled = true; };
    }, [selectedFieldTarget, backendFieldCatalog]);
    const fieldBaseRows = useMemo(
      () => (backendCatalogKey && backendFieldCatalog[backendCatalogKey])
        ? backendFieldCatalog[backendCatalogKey]
        : getReportFieldRows(selectedFieldTarget, pmMetricRows),
      [backendCatalogKey, backendFieldCatalog, selectedFieldTarget, pmMetricRows],
    );
    const fieldCandidatePoolRows = useMemo(
      () => (backendCatalogKey && backendFieldCatalog[backendCatalogKey])
        ? backendFieldCatalog[backendCatalogKey]
        : getAvailableReportFieldRows(selectedFieldTarget, pmMetricRows),
      [backendCatalogKey, backendFieldCatalog, selectedFieldTarget, pmMetricRows],
    );
    const targetFieldRows = useMemo(() => {
      if (!selectedFieldTarget) return [];
      return fieldRowsByTarget[selectedFieldTarget.key] ?? fieldBaseRows;
    }, [fieldBaseRows, fieldRowsByTarget, selectedFieldTarget]);
    const fieldConfigRows = useMemo(
      () => targetFieldRows.filter((row) => rowMatchesTechFilter(row, fieldTechFilter)),
      [fieldTechFilter, targetFieldRows],
    );
    const fieldCandidateRows = useMemo(() => {
      const currentIds = new Set(targetFieldRows.map(fieldIdentity));
      return fieldCandidatePoolRows
        .filter((row) => !currentIds.has(fieldIdentity(row)))
        .filter((row) => rowMatchesTechFilter(row, fieldTechFilter));
    }, [fieldCandidatePoolRows, fieldTechFilter, targetFieldRows]);
    const fieldCandidateOptions = useMemo(
      () => fieldCandidateRows.map((row) => ({
        label: `${formatReportFieldTech(row)} ${row.cnName ?? row.outputAlias} / ${row.outputAlias} / ${row.systemField}`,
        value: row.key,
      })),
      [fieldCandidateRows],
    );

    const updateCurrentFieldRows = useCallback((rows: ReportFieldRow[]) => {
      if (!selectedFieldTarget) return;
      setFieldRowsByTarget((prev) => ({ ...prev, [selectedFieldTarget.key]: rows }));
    }, [selectedFieldTarget]);
    const addFieldConfigRow = useCallback(() => {
      const row = fieldCandidateRows.find((candidate) => candidate.key === fieldCandidateKey);
      if (!row) return;
      const baseOrder = new Map(fieldCandidatePoolRows.map((item, index) => [fieldIdentity(item), index]));
      const nextRows = [...targetFieldRows, row].sort((a, b) => (
        (baseOrder.get(fieldIdentity(a)) ?? Number.MAX_SAFE_INTEGER)
        - (baseOrder.get(fieldIdentity(b)) ?? Number.MAX_SAFE_INTEGER)
      ));
      updateCurrentFieldRows(nextRows);
      setFieldCandidateKey(undefined);
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
      setFieldConfigDomain(domain);
      setFieldTechFilter('ALL');
      const nextTarget = fieldTargets.find((t) => t.domain === domain);
      if (nextTarget) setFieldConfigTargetKey(nextTarget.key);
    }, [fieldTargets]);
    const changeFieldTechFilter = useCallback((techFilter: FieldTechFilter) => {
      setFieldTechFilter(techFilter);
      const nextTarget = fieldTargets.find((t) => t.domain === fieldConfigDomain && targetMatchesTechFilter(t, techFilter));
      setFieldConfigTargetKey(nextTarget?.key ?? '');
    }, [fieldTargets, fieldConfigDomain]);

    const editableFieldConfigColumns: ColumnsType<ReportFieldRow> = [
      {
        title: '上报',
        width: 76,
        fixed: 'left',
        render: (_, row) => <Switch size="small" checked={row.enabled} onChange={(checked) => toggleFieldEnabled(row, checked)} checkedChildren="开" unCheckedChildren="关" />,
      },
      { title: '对象', width: 90, fixed: 'left', render: (_, row) => <Tag>{formatReportFieldObject(row)}</Tag> },
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
      {
        title: '操作',
        width: 74,
        fixed: 'right',
        render: (_, row) => (
          <Tooltip title="从当前模板删除">
            <Button
              aria-label={`删除字段 ${row.outputAlias}`}
              type="text"
              danger
              size="small"
              icon={<DeleteOutlined />}
              onClick={() => removeFieldConfigRow(row)}
            />
          </Tooltip>
        ),
      },
    ];

    return (
      <div className={styles.editorSection}>
        <div className={styles.editorSectionHeader}>
          <Typography.Text strong>字段/指标配置</Typography.Text>
          <Space size={8} wrap className={styles.fieldConfigTools}>
            <Select
              value={fieldConfigDomain}
              style={{ width: 120 }}
              options={fieldDomainOptions}
              onChange={(value) => changeFieldConfigDomain(value as Domain)}
            />
            <Select
              value={fieldTechFilter}
              style={{ width: 132 }}
              options={fieldTechFilterOptions}
              onChange={(value) => changeFieldTechFilter(value as FieldTechFilter)}
            />
            <Select
              value={selectedFieldTarget?.key}
              placeholder="对象"
              style={{ width: 132 }}
              options={fieldTargetOptions}
              onChange={setFieldConfigTargetKey}
              notFoundContent="无对象"
            />
            <Select
              allowClear
              showSearch
              value={fieldCandidateKey}
              placeholder="搜索可新增字段/指标"
              optionFilterProp="label"
              style={{ width: 340 }}
              options={fieldCandidateOptions}
              onChange={setFieldCandidateKey}
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
          loading={pmLoading && selectedFieldTarget?.domain === 'PM'}
          pagination={false}
          scroll={{ x: 1820, y: 320 }}
        />
      </div>
    );
  },
));

export default function NorthboundPageConfig() {
  const [configForm] = Form.useForm();
  const [fileProfiles, setFileProfiles] = useState<ScenarioRow[]>(scenarioRows);
  const [selectedScenario, setSelectedScenario] = useState<ScenarioRow | null>(null);
  const [viewFieldConfigDomain, setViewFieldConfigDomain] = useState<Domain>(defaultFieldTarget?.domain ?? 'CM');
  const [viewFieldConfigTargetKey, setViewFieldConfigTargetKey] = useState(defaultFieldTarget?.key ?? '');
  const [viewFieldTechFilter, setViewFieldTechFilter] = useState<FieldTechFilter>('ALL');
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
  const [fileDeliveryTargets, setFileDeliveryTargets] = useState<DeliveryTargetRow[]>(
    () => cloneDeliveryTargets('file'),
  );
  const [selectedInventoryType, setSelectedInventoryType] = useState<InventoryType>('ENB');
  const [viewInventoryType, setViewInventoryType] = useState<InventoryType | null>(null);
  const [inventoryEditorOpen, setInventoryEditorOpen] = useState(false);
  const [inventoryConfigs, setInventoryConfigs] = useState<InventoryConfigRow[]>(initialInventoryConfigs);
  const [inventoryEnabled, setInventoryEnabled] = useState<Record<InventoryType, boolean>>(defaultInventoryEnabled);
  const [inventoryProfileSaving, setInventoryProfileSaving] = useState<Record<string, boolean>>({});
  const [inventoryFieldRowsByType, setInventoryFieldRowsByType] = useState<Record<InventoryType, InventoryField[]>>(defaultInventoryFieldRows);
  const [inventoryDeliveryTargetsByType, setInventoryDeliveryTargetsByType] = useState<Record<InventoryType, DeliveryTargetRow[]>>(
    () => Object.fromEntries(inventoryTypeOptions.map((option) => [
      option.value,
      cloneDeliveryTargets(`inventory-${option.value.toLowerCase()}`),
    ])) as Record<InventoryType, DeliveryTargetRow[]>,
  );
  const [inventoryCandidateKey, setInventoryCandidateKey] = useState<string>();
  const [apiEnabled, setApiEnabled] = useState<Record<string, boolean>>(defaultApiEnabled);
  const [apiRows, setApiRows] = useState<NorthboundApiRow[]>(legacySupportedApiRows);
  const [apiClients, setApiClients] = useState<ApiClientRow[]>([]);
  const [apiClientSaving, setApiClientSaving] = useState(false);
  const [selectedApi, setSelectedApi] = useState<NorthboundApiRow | null>(null);
  const [selectedReportStatus, setSelectedReportStatus] = useState<ReportStatusInfo | null>(null);
  const [reportRunList, setReportRunList] = useState<NorthboundFileRun[]>([]);
  const [reportEventList, setReportEventList] = useState<NorthboundPageConfigEvent[]>([]);
  const [reportEventTotal, setReportEventTotal] = useState(0);
  const [reportEventLoading, setReportEventLoading] = useState(false);
  const [reportCapabilityName, setReportCapabilityName] = useState<string>('');
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
  const [selectedSnmp, setSelectedSnmp] = useState<SnmpAlarmTargetRow | null>(null);
  const [snmpEditor, setSnmpEditor] = useState<SnmpAlarmTargetRow | null>(null);
  const [pmMetricRows, setPmMetricRows] = useState<PmMetric[]>([]);
  const [pmLoading, setPmLoading] = useState(true);
  const [pageConfigLoading, setPageConfigLoading] = useState(false);

  const loadPageConfig = useCallback(async (silent = false) => {
    setPageConfigLoading(true);
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
        apiClientResp,
      ] = await Promise.all([
        northboundPageConfigApi.getFileProfiles(),
        northboundPageConfigApi.getInventoryProfiles(),
        northboundPageConfigApi.getDeliveryTargets({ scope: 'file' }),
        northboundPageConfigApi.getDeliveryTargets({ scope: 'inventory' }),
        northboundPageConfigApi.getDeliveryTargets({ scope: 'socket' }),
        northboundPageConfigApi.getSNMPAlarmTargets(),
        northboundPageConfigApi.getSocketAlarmConfigs(),
        northboundPageConfigApi.getAPIConfigs(),
        northboundPageConfigApi.getAPIClients(),
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
      setSelectedScenario((current) => (
        current ? nextFileProfiles.find((row) => row.code === current.code) ?? current : current
      ));

      setInventoryConfigs(nextInventoryProfiles);
      setInventoryEnabled(inventoryEnabledMap(nextInventoryProfiles, inventoryProfileResp.items));
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

      if (fileDeliveryResp.items.length > 0) {
        setFileDeliveryTargets(fileDeliveryResp.items.map(mapApiDeliveryTarget));
      }
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

      const nextApiRows = apiResp.items.map(mapApiConfig);
      if (nextApiRows.length > 0) {
      setApiRows(nextApiRows);
      setApiEnabled(Object.fromEntries(apiResp.items.map((row) => [row.key, row.enabled])));
      setApiClients(apiClientResp.items.map(mapApiClient));
      }

      if (!silent) void message.success('北向页面配置已刷新');
    } catch {
      if (!silent) {
        void message.error('北向页面配置加载失败，已保留当前页面数据');
      }
    } finally {
      setPageConfigLoading(false);
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
        if (!cancelled) void message.error('指标目录加载失败');
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
  const inventoryCandidateRows = useMemo(() => {
    const selectedIds = new Set(currentInventoryFields.map((field) => field.exportKey));
    return inventoryFields
      .filter((field) => field.template === selectedInventoryType)
      .filter((field) => !selectedIds.has(field.exportKey));
  }, [currentInventoryFields, selectedInventoryType]);
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

  const getSocketDeliveryTargets = (config: SocketAlarmConfigRow) => (
    socketDeliveryTargetsByConfig[config.key] ?? cloneDeliveryTargets(`${config.key}-file-sync`)
  );

  const enabledDeliverySummary = (rows: DeliveryTargetRow[]) => {
    const enabledRows = rows.filter((row) => row.enabled);
    if (enabledRows.length === 0) return '未启用传输目标';
    return enabledRows.map((row) => `${row.name}(${row.protocol})`).join('、');
  };

  const updateFileDeliveryTarget = (key: string, patch: Partial<DeliveryTargetRow>) => {
    setFileDeliveryTargets((rows) => rows.map((row) => (row.key === key ? { ...row, ...patch } : row)));
  };

  const addFileDeliveryTarget = () => {
    setFileDeliveryTargets((rows) => [...rows, createDeliveryTarget('file', rows.length + 1)]);
  };

  const removeFileDeliveryTarget = (key: string) => {
    setFileDeliveryTargets((rows) => rows.filter((row) => row.key !== key));
  };

  const updateInventoryDeliveryTarget = (inventoryType: InventoryType, key: string, patch: Partial<DeliveryTargetRow>) => {
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
    setInventoryDeliveryTargetsByType((prev) => ({
      ...prev,
      [inventoryType]: (prev[inventoryType] ?? cloneDeliveryTargets(`inventory-${inventoryType.toLowerCase()}`)).filter((row) => row.key !== key),
    }));
  };

  const updateSocketEditorDeliveryTarget = (key: string, patch: Partial<DeliveryTargetRow>) => {
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
    setSocketEditorDeliveryTargets((rows) => rows.filter((row) => row.key !== key));
  };

  const testDeliveryTarget = (
    row: DeliveryTargetRow,
    scope: 'file' | 'inventory' | 'socket',
    ownerCode: string,
  ) => {
    if (!row.host || !row.username) {
      void message.warning('请先填写主机地址和用户名');
      return;
    }
	    void northboundPageConfigApi.testDeliveryTarget(serializeDeliveryTarget(scope, ownerCode, row))
	      .then((event) => {
	        openSingleEventReport(event, `${row.name} ${row.protocol}`);
        if (event.status === 'success') {
          void message.success(`${row.name} ${row.protocol} 连接测试通过`);
        } else {
          void message.warning(`${row.name} ${row.protocol} 连接测试未通过`);
        }
      })
      .catch(() => {
        void message.error(`${row.name} ${row.protocol} 连接测试失败`);
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

  const saveInventoryDraft = () => {
    if (!selectedInventoryConfig) return;
    const enabled = Boolean(inventoryEnabled[selectedInventoryConfig.key]);
    const fields = inventoryFieldRowsByType[selectedInventoryConfig.key] ?? [];
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
          getInventoryDeliveryTargets(selectedInventoryConfig.key),
        ),
      ),
    ])
      .then(([profile]) => {
        applyInventoryProfile(profile);
        void message.success(`${profile.object_code} Inventory 配置已保存`);
        setInventoryEditorOpen(false);
      })
      .catch(() => {
        void message.error(`${selectedInventoryConfig.objectCode} Inventory 配置保存失败`);
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
  const visibleViewFieldTargets = useMemo(
    () => viewFieldTargets
      .filter((target) => target.domain === viewFieldConfigDomain)
      .filter((target) => targetMatchesTechFilter(target, viewFieldTechFilter)),
    [viewFieldConfigDomain, viewFieldTargets, viewFieldTechFilter],
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
  const viewFieldRows = useMemo(
    () => getReportFieldRows(selectedViewFieldTarget, pmMetricRows)
      .filter((row) => rowMatchesTechFilter(row, viewFieldTechFilter)),
    [pmMetricRows, selectedViewFieldTarget, viewFieldTechFilter],
  );

  useEffect(() => {
    setInventoryCandidateKey(undefined);
  }, [selectedInventoryType]);

  useEffect(() => {
    if (!selectedScenario || viewFieldTargets.length === 0) return;
    const domainHasTargets = viewFieldTargets.some((target) => target.domain === viewFieldConfigDomain);
    if (!domainHasTargets) {
      const [nextTarget] = viewFieldTargets;
      setViewFieldConfigDomain(nextTarget.domain);
      setViewFieldTechFilter('ALL');
      setViewFieldConfigTargetKey(nextTarget.key);
      return;
    }
    if (visibleViewFieldTargets.length === 0) {
      if (viewFieldConfigTargetKey) setViewFieldConfigTargetKey('');
      return;
    }
    const currentTarget = visibleViewFieldTargets.find((target) => target.key === viewFieldConfigTargetKey);
    if (currentTarget) return;
    setViewFieldConfigTargetKey(visibleViewFieldTargets[0].key);
  }, [selectedScenario, viewFieldConfigDomain, viewFieldConfigTargetKey, viewFieldTargets, visibleViewFieldTargets]);

  const openViewDrawer = (row: ScenarioRow) => {
    const periodRows = getScenarioPeriodRows(row);
    const firstTarget = getFirstFieldTarget(periodRows);
    setSelectedScenario(row);
    setViewFieldTechFilter('ALL');
    if (firstTarget) {
      setViewFieldConfigDomain(firstTarget.domain);
      setViewFieldConfigTargetKey(firstTarget.key);
    }
  };

  const changeViewFieldConfigDomain = (domain: Domain) => {
    setViewFieldConfigDomain(domain);
    setViewFieldTechFilter('ALL');
    const nextTarget = viewFieldTargets.find((target) => target.domain === domain);
    setViewFieldConfigTargetKey(nextTarget?.key ?? '');
  };

  const changeViewFieldTechFilter = (techFilter: FieldTechFilter) => {
    setViewFieldTechFilter(techFilter);
    const nextTarget = viewFieldTargets.find((target) => target.domain === viewFieldConfigDomain && targetMatchesTechFilter(target, techFilter));
    setViewFieldConfigTargetKey(nextTarget?.key ?? '');
  };

  const openCreateEditor = () => {
    const periodRows = clonePeriodRows(defaultEditorPeriodRows);
    setEditorMode('create');
    setEditorPeriodRows(periodRows);
    setInitialFieldRows({});
    setEditorSession((n) => n + 1);
    const nextCode = getNextCustomScenarioCode(fileProfiles);
    configForm.setFieldsValue({
      scenarioCode: nextCode,
      vendor: 'Baicells',
      scenarioName: '自定义场景',
      scenarioNameEn: 'Custom',
      name: '自定义北向文件配置',
      description: '',
      domains: ['CM', 'PM'],
      periods: ['24H', '15M'],
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
        void message.warning('请输入配置编号');
        return;
      }
      if (!/^S\d{4}$/.test(code)) {
        void message.warning('配置编号必须使用 S0000 格式');
        return;
      }
      const enabled = Boolean(values.enabled);
      const vendor = values.vendor?.trim() || selectedScenario?.vendor || 'Baicells';
      const fieldRowsByTarget = fieldConfigRef.current?.getFieldRowsByTarget() ?? {};
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
            serializeDeliveryTargets('file', '', fileDeliveryTargets),
          ),
        ])
          .then(([profile]) => {
            applyFileProfile(profile);
            setSelectedScenario(mapApiFileProfile(profile));
            void message.success(`新增配置已保存：${profile.name}`);
            setEditorOpen(false);
          })
          .catch(() => {
            void message.error(`配置保存失败：${code}`);
          })
          .finally(() => setFileSaving(code, false));
        return;
      }

      void Promise.all([
        northboundPageConfigApi.updateFileProfile(code, request),
        northboundPageConfigApi.replaceDeliveryTargets(
          serializeDeliveryTargets('file', '', fileDeliveryTargets),
        ),
      ])
        .then(([profile]) => {
          applyFileProfile(profile);
          void message.success(`编辑配置已保存：${profile.name}`);
          setEditorOpen(false);
        })
        .catch(() => {
          void message.error(`配置保存失败：${code}`);
        })
        .finally(() => setFileSaving(code, false));
    });
  };

  const updateEditorPeriodRow = useCallback((key: string, patch: Partial<ScenarioPeriodRow>) => {
    setEditorPeriodRows((rows) => rows.map((row) => (row.key === key ? { ...row, ...patch } : row)));
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
    const nextDefault = defaultPeriodRow(domain);
    setEditorPeriodRows((rows) =>
      rows.map((row) => (row.key === key ? {
        ...row,
        domain,
        scope: nextDefault.scope,
        format: nextDefault.format,
        period: nextDefault.period,
        cron: nextDefault.cron,
        objects: nextDefault.objects,
        path: nextDefault.path,
        fileName: nextDefault.fileName,
        compressionEnabled: false,
        compressionFormat: nextDefault.compressionFormat,
      } : row)),
    );
  };

  const addEditorPeriodRow = () => {
    setEditorPeriodRows((rows) => [...rows, defaultPeriodRow()]);
  };

  const removeEditorPeriodRow = (key: string) => {
    setEditorPeriodRows((rows) => rows.filter((row) => row.key !== key));
  };

  const refreshScenarioList = () => {
    void loadPageConfig();
  };

  const renderTablePagination = (ariaLabel: string) => ({
    pageSize: 10,
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
        <Tag>{`每 ${getPeriodMinutes(period)} 分钟`}</Tag>
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
        void message.success('报文已复制');
      } else {
        void message.error('报文复制失败');
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
    setSelectedReportStatus(buildEventReportStatus(event, capabilityName));
  };

  const renderStatusCell = (info: ReportStatusInfo, enabled: boolean) => {
    const displayInfo = effectiveReportStatus(info, enabled);
    const normal = displayInfo.state === 'success' || displayInfo.state === 'running';
    return <Tag color={normal ? 'success' : 'default'}>{normal ? '正常' : '终止'}</Tag>;
  };

  const showLatestRunReport = (
    profileKind: 'file' | 'inventory',
    profileCode: string,
    fallback: ReportStatusInfo,
  ) => {
    const isFile = profileKind === 'file';
    setReportEventList([]);
    setReportEventTotal(0);
    setReportEventLoading(false);
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
        void message.warning('未读取到最近上报记录，已显示配置预览');
      });
  };

  const resolveRunReportStatus = async (
    run: NorthboundFileRun,
    fallbackCapabilityName: string,
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
	    try {
	      const events = await northboundPageConfigApi.listEvents({
	        capability: 'delivery',
	        owner_code: run.profile_code,
	        limit: 50,
	        include_payload: false,
	      });
      // Precisely match this run's delivery events by summary.run_id. Never fall
      // back to an unrelated event — the old `?? events.items[0]` crossed objects.
      const deliveries = events.items.filter((event) => event.summary?.run_id === run.id);
      return { ...runStatus, deliveryNote: summarizeDeliveryNote(deliveries) };
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
    const firstRun = items.find((item) => item.status === 'success') ?? items[0];
    if (firstRun) {
      void resolveRunReportStatus(firstRun, capabilityName).then((info) => {
        setSelectedReportStatus(info ?? fallback);
      });
    } else {
      setSelectedReportStatus(fallback);
    }
  };

  const selectReportEvent = (event: NorthboundPageConfigEvent, capabilityName: string) => {
    setSelectedReportStatus(buildEventReportStatus(event, capabilityName));
    void northboundPageConfigApi.getEvent(event.id)
      .then((fullEvent) => {
        setSelectedReportStatus(buildEventReportStatus(fullEvent, capabilityName));
      })
      .catch(() => {
        void message.warning('未读取到完整事件详情，已显示列表摘要');
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
    setReportCapabilityName(fallback.capabilityName);
    setSelectedReportStatus(fallback);
    setReportEventLoading(true);
    void northboundPageConfigApi.listEvents({
      capability,
      target_key: targetKey,
      limit: 100,
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
        void message.warning('未读取到最近事件，已显示配置预览');
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
        void message.error(`${row.code} 启停状态保存失败`);
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
        void message.error(`${row.objectCode} Inventory 启停状态保存失败`);
      })
      .finally(() => setInventorySaving(row.key, false));
  };

  const runFileProfile = (row: ScenarioRow) => {
    setFileProfileRunning((prev) => ({ ...prev, [row.code]: true }));
    void northboundPageConfigApi.runFileProfile(row.code, { limit: 200 })
      .then((result) => {
        openReportDrawer(result.items, `${row.code} 北向文件`, null);
        void message.success(`${row.code} 已生成 ${result.total} 条上报记录`);
      })
      .catch(() => {
        void message.error(`${row.code} 手动执行失败`);
      })
      .finally(() => {
        setFileProfileRunning((prev) => ({ ...prev, [row.code]: false }));
      });
  };

  const runInventoryProfile = (row: InventoryConfigRow) => {
    setInventoryProfileRunning((prev) => ({ ...prev, [row.key]: true }));
    void northboundPageConfigApi.runInventoryProfile(row.key, { limit: 200 })
      .then((run) => {
        void resolveRunReportStatus(run, `${row.objectCode} Inventory`)
          .then(setSelectedReportStatus);
        void message.success(`${row.objectCode} Inventory 已生成上报记录`);
      })
      .catch(() => {
        void message.error(`${row.objectCode} Inventory 手动执行失败`);
      })
      .finally(() => {
        setInventoryProfileRunning((prev) => ({ ...prev, [row.key]: false }));
      });
  };

  const persistSnmpEnabled = (row: SnmpAlarmTargetRow, checked: boolean) => {
    if (checked) {
      const blocker = getSnmpEnableBlocker(row);
      if (blocker) {
        void message.warning(`${row.name} ${blocker}`);
        return;
      }
    }
    const previous = Boolean(snmpEnabled[row.key]);
    setSnmpEnabled((prev) => ({ ...prev, [row.key]: checked }));
    void northboundPageConfigApi.updateSNMPAlarmTarget(row.key, serializeSnmpTarget(row, checked))
      .then((target) => {
        const next = mapApiSnmpTarget(target);
        setSnmpTargets((rows) => rows.map((item) => (item.key === next.key ? next : item)));
        setSnmpEnabled((prev) => ({ ...prev, [next.key]: target.enabled }));
      })
      .catch(() => {
        setSnmpEnabled((prev) => ({ ...prev, [row.key]: previous }));
        void message.error(`${row.name} SNMP 启停状态保存失败`);
      });
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
        void message.error(`${row.name} Socket 启停状态保存失败`);
      });
  };

  const persistApiEnabled = (row: NorthboundApiRow, checked: boolean) => {
    const previous = Boolean(apiEnabled[row.key]);
    setApiEnabled((prev) => ({ ...prev, [row.key]: checked }));
    void northboundPageConfigApi.updateAPIConfig(row.key, checked)
      .then((config) => {
        setApiRows((rows) => rows.map((item) => (item.key === config.key ? mapApiConfig(config) : item)));
        setApiEnabled((prev) => ({ ...prev, [config.key]: config.enabled }));
      })
      .catch(() => {
        setApiEnabled((prev) => ({ ...prev, [row.key]: previous }));
        void message.error(`${row.name} API 启停状态保存失败`);
      });
  };

  const patchApiClient = (clientKey: string, patch: Partial<ApiClientRow>) => {
    setApiClients((rows) => rows.map((row) => (row.clientKey === clientKey ? { ...row, ...patch } : row)));
  };

  const addApiClient = () => {
    const index = apiClients.length + 1;
    setApiClients((rows) => [
      ...rows,
      {
        clientKey: `northbound-client-${index}`,
        name: `北向客户端 ${index}`,
        enabled: false,
        tokenSecret: '',
        tokenSet: false,
        allowedApiKeys: apiRows.map((row) => row.key).join('\n'),
        ipWhitelist: '',
      },
    ]);
  };

  const removeApiClient = (clientKey: string) => {
    setApiClients((rows) => rows.filter((row) => row.clientKey !== clientKey));
  };

  const saveApiClients = () => {
    setApiClientSaving(true);
    void northboundPageConfigApi.replaceAPIClients(serializeApiClients(apiClients))
      .then((resp) => {
        setApiClients(resp.items.map(mapApiClient));
        void message.success('API client 配置已保存');
      })
      .catch(() => {
        void message.error('API client 配置保存失败');
      })
      .finally(() => setApiClientSaving(false));
  };

  const testSocketAlarm = (row: SocketAlarmConfigRow) => {
    void northboundPageConfigApi.testSocketAlarmConfig(row.key)
      .then((event) => {
        openSingleEventReport(event, row.name);
        void message.success(`${row.name} 样例报文已记录`);
      })
      .catch(() => {
        void message.error(`${row.name} 样例报文生成失败`);
      });
  };

  const testSnmpAlarm = (row: SnmpAlarmTargetRow) => {
    void northboundPageConfigApi.testSNMPAlarmTarget(row.key)
      .then((event) => {
        openSingleEventReport(event, row.name);
        if (event.status === 'success') {
          void message.success(`${row.name} 测试报文已生成`);
        } else {
          void message.warning(`${row.name} 测试报文已生成，但目标配置不完整`);
        }
      })
      .catch(() => {
        void message.error(`${row.name} 测试报文生成失败`);
      });
  };

  const testApiContract = (row: NorthboundApiRow) => {
    void northboundPageConfigApi.testAPIConfig(row.key)
      .then((event) => {
        openSingleEventReport(event, row.name);
        void message.success(`${row.name} 契约检查已记录`);
      })
      .catch(() => {
        void message.error(`${row.name} 契约检查失败`);
      });
  };

  const saveSocketEditor = (row: SocketAlarmConfigRow) => {
    const enabled = socketEditorEnabled;
    const accounts = socketEditorAccounts;
    void Promise.all([
      northboundPageConfigApi.updateSocketAlarmConfig(row.key, serializeSocketConfig(row, accounts, enabled)),
      northboundPageConfigApi.replaceDeliveryTargets(
        serializeDeliveryTargets('socket', row.key, socketEditorDeliveryTargets),
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
        void message.success(`${next.name} 已保存`);
        closeSocketEditor();
      })
      .catch(() => {
        void message.error(`${row.name} Socket 配置保存失败`);
      });
  };

  const saveSnmpEditor = (row: SnmpAlarmTargetRow) => {
    const enabled = Boolean(snmpEnabled[row.key]);
    if (enabled) {
      const blocker = getSnmpEnableBlocker(row);
      if (blocker) {
        void message.warning(`${row.name} ${blocker}`);
        return;
      }
    }
    void northboundPageConfigApi.updateSNMPAlarmTarget(row.key, serializeSnmpTarget(row, enabled))
      .then((target) => {
        const next = mapApiSnmpTarget(target);
        setSnmpTargets((rows) => rows.map((item) => (item.key === next.key ? next : item)));
        setSnmpEnabled((prev) => ({ ...prev, [next.key]: target.enabled }));
        void message.success(`${next.name} 已保存`);
        setSnmpEditor(null);
      })
      .catch(() => {
        void message.error(`${row.name} SNMP 配置保存失败`);
      });
  };

  const scenarioColumns: ColumnsType<ScenarioRow> = [
    {
      title: '操作',
      width: 104,
      fixed: 'left',
      render: (_, row) => (
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
                { key: 'run', icon: <PlayCircleOutlined />, label: '手动执行', disabled: Boolean(fileProfileRunning[row.code]) },
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
                  void message.info(`${row.code} 已复制为草稿`);
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
              icon={<MoreOutlined />}
              onClick={(event) => event.stopPropagation()}
            />
          </Dropdown>
        </div>
      ),
    },
    {
      title: '场景号',
      dataIndex: 'code',
      width: 96,
      render: (value: string) => <Typography.Text strong className={styles.scenarioCode}>{value}</Typography.Text>,
    },
    {
      title: '场景名称',
      dataIndex: 'scenarioName',
      width: 140,
      render: (value: string) => <Typography.Text ellipsis>{value}</Typography.Text>,
    },
    {
      title: '状态',
      width: 88,
      render: (_, row) => renderStatusCell(buildFileReportStatus(row), Boolean(scenarioEnabled[row.code])),
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
      title: '压缩',
      width: 100,
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
      render: (_, row) => (
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
                { key: 'run', icon: <PlayCircleOutlined />, label: '手动执行', disabled: Boolean(inventoryProfileRunning[row.key]) },
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
              icon={<MoreOutlined />}
              onClick={(event) => event.stopPropagation()}
            />
          </Dropdown>
        </div>
      ),
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
      width: 88,
      render: (_, row) => renderStatusCell(buildInventoryReportStatus(row), Boolean(inventoryEnabled[row.key])),
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
      title: '压缩',
      width: 100,
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
      width: 104,
      fixed: 'left',
      render: (_, row) => (
        <div className={styles.rowControl}>
          <Switch
            size="small"
            checked={apiEnabled[row.key]}
            checkedChildren="开"
            unCheckedChildren="关"
            onClick={(_, event) => event.stopPropagation()}
            onChange={(checked) => persistApiEnabled(row, checked)}
          />
          <Dropdown
            trigger={['click']}
            menu={{
              items: [
                { key: 'view', icon: <EyeOutlined />, label: '查看' },
                { key: 'report', icon: <FileSearchOutlined />, label: '契约结果' },
                { key: 'test', icon: <PlayCircleOutlined />, label: '检查契约' },
              ],
              onClick: ({ key, domEvent }) => {
                domEvent.stopPropagation();
                if (key === 'view') setSelectedApi(row);
                if (key === 'report') showLatestEventReport('api', row.key, effectiveReportStatus(buildApiReportStatus(row), Boolean(apiEnabled[row.key])));
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
        </div>
      ),
    },
    {
      title: '类型',
      width: 108,
      fixed: 'left',
      render: (_, row) => apiKindTag(getApiMeta(row).apiKind as ApiKind),
    },
    {
      title: '模块',
      dataIndex: 'module',
      width: 96,
      render: (value: string) => <Tag color="blue">{value}</Tag>,
    },
    {
      title: '接口名称',
      dataIndex: 'name',
      width: 180,
      render: (value: string) => <Typography.Text strong ellipsis>{value}</Typography.Text>,
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
    {
      title: '鉴权',
      dataIndex: 'auth',
      width: 240,
      render: (value: string) => <Typography.Text ellipsis>{normalizeLegacyApiText(value)}</Typography.Text>,
    },
  ];

  const apiClientColumns: ColumnsType<ApiClientRow> = [
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
          onChange={(enabled) => patchApiClient(row.clientKey, { enabled })}
        />
      ),
    },
    {
      title: 'Client Key',
      dataIndex: 'clientKey',
      width: 210,
      fixed: 'left',
      render: (value: string, row) => (
        <Input
          value={value}
          className={styles.monoText}
          onChange={(event) => patchApiClient(row.clientKey, { clientKey: event.target.value })}
        />
      ),
    },
    {
      title: '名称',
      dataIndex: 'name',
      width: 180,
      render: (value: string, row) => (
        <Input value={value} onChange={(event) => patchApiClient(row.clientKey, { name: event.target.value })} />
      ),
    },
    {
      title: 'Token',
      dataIndex: 'tokenSecret',
      width: 220,
      render: (value: string, row) => (
        <Input.Password
          placeholder={row.tokenSet ? '未修改保持原 token' : '请输入 token'}
          onChange={(event) => patchApiClient(row.clientKey, { tokenSecret: event.target.value || value })}
        />
      ),
    },
    {
      title: '允许接口',
      dataIndex: 'allowedApiKeys',
      width: 260,
      render: (value: string, row) => (
        <Input.TextArea
          value={value}
          autoSize={{ minRows: 2, maxRows: 4 }}
          className={styles.monoText}
          placeholder="留空表示允许全部已启用 API"
          onChange={(event) => patchApiClient(row.clientKey, { allowedApiKeys: event.target.value })}
        />
      ),
    },
    {
      title: 'IP 白名单',
      dataIndex: 'ipWhitelist',
      width: 240,
      render: (value: string, row) => (
        <Input.TextArea
          value={value}
          autoSize={{ minRows: 2, maxRows: 4 }}
          className={styles.monoText}
          placeholder="单 IP 或 CIDR；留空不限"
          onChange={(event) => patchApiClient(row.clientKey, { ipWhitelist: event.target.value })}
        />
      ),
    },
    {
      title: '过期时间',
      dataIndex: 'expiresAt',
      width: 210,
      render: (value: string | undefined, row) => (
        <Input
          value={value}
          className={styles.monoText}
          placeholder="2026-12-31T16:00:00Z"
          onChange={(event) => patchApiClient(row.clientKey, { expiresAt: event.target.value })}
        />
      ),
    },
    {
      title: '操作',
      width: 74,
      fixed: 'right',
      render: (_, row) => (
        <Tooltip title="删除 client">
          <Button
            danger
            size="small"
            type="text"
            icon={<DeleteOutlined />}
            onClick={() => removeApiClient(row.clientKey)}
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

  const snmpAlarmFieldColumns: ColumnsType<AlarmFieldMappingRow> = [
    { title: '上报', width: 76, fixed: 'left', render: (_, row) => <Switch size="small" checked={row.enabled} disabled checkedChildren="开" unCheckedChildren="关" /> },
    { title: '序号', dataIndex: 'order', width: 70, fixed: 'left' },
    { title: '输出字段', dataIndex: 'field', width: 220, fixed: 'left', render: (value: string) => <span className={styles.monoText}>{value}</span> },
    { title: 'OID', width: 300, render: (_, row) => <Typography.Text className={styles.monoText} ellipsis={{ tooltip: snmpAlarmOIDForField(row.field) }}>{snmpAlarmOIDForField(row.field)}</Typography.Text> },
    { title: '字段名称', dataIndex: 'cnName', width: 150 },
    { title: 'xomc 数据源', dataIndex: 'source', width: 320, render: (value: string) => <span className={styles.monoText}>{value}</span> },
    { title: '类型', dataIndex: 'dataType', width: 150, render: (value: string) => <Tag>{value}</Tag> },
  ];

  const deliveryTargetColumns: ColumnsType<DeliveryTargetRow> = [
    { title: '启用', dataIndex: 'enabled', width: 76, fixed: 'left', render: (value: boolean) => <Switch size="small" checked={value} disabled checkedChildren="开" unCheckedChildren="关" /> },
    { title: '目标名称', dataIndex: 'name', width: 150, fixed: 'left', render: (value: string) => <Typography.Text strong ellipsis>{value}</Typography.Text> },
    { title: '协议', dataIndex: 'protocol', width: 86, render: (value: DeliveryProtocol) => deliveryProtocolTag(value) },
    { title: '地址', width: 180, render: (_, row) => <span className={styles.monoText}>{endpointText(row.host || '-', row.port)}</span> },
    { title: '账号', dataIndex: 'username', width: 140, render: (value: string) => <span className={styles.monoText}>{value || '-'}</span> },
    { title: '密码', width: 150, render: (_, row) => <Tag>{row.credential || '-'}</Tag> },
    { title: '#FTPRoot#', dataIndex: 'remoteRoot', width: 220, render: (value: string) => <span className={styles.monoText}>{value}</span> },
    { title: '重试/超时', width: 130, render: (_, row) => <Tag>{row.retryTimes} 次 / {row.timeoutSeconds}s</Tag> },
  ];

  const createDeliveryTargetEditorColumns = (
    rows: DeliveryTargetRow[],
    onPatch: (key: string, patch: Partial<DeliveryTargetRow>) => void,
    onRemove: (key: string) => void,
    scope: 'file' | 'inventory' | 'socket',
    ownerCode: string,
  ): ColumnsType<DeliveryTargetRow> => [
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
          onChange={(checked) => onPatch(row.key, { enabled: checked })}
        />
      ),
    },
    {
      title: '目标名称',
      dataIndex: 'name',
      width: 160,
      fixed: 'left',
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
      render: (value: string, row) => (
        <Input value={value} className={styles.monoText} placeholder="IP/域名" onChange={(event) => onPatch(row.key, { host: event.target.value })} />
      ),
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
      render: (value: string, row) => (
        <Input value={value} className={styles.monoText} placeholder="用户名" onChange={(event) => onPatch(row.key, { username: event.target.value })} />
      ),
    },
    {
      title: '密码',
      dataIndex: 'credential',
      width: 190,
      render: (value: string, row) => (
        <Input.Password
          placeholder={value === '已加密存储' ? '未修改保持原凭据' : '请输入凭据'}
          onChange={(event) => onPatch(row.key, { credential: event.target.value || value })}
        />
      ),
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
    {
      title: '操作',
      width: 104,
      fixed: 'right',
      render: (_, row) => (
        <Space size={4}>
          <Tooltip title="测试连接">
            <Button size="small" type="text" icon={<PlayCircleOutlined />} onClick={() => testDeliveryTarget(row, scope, ownerCode)} />
          </Tooltip>
          <Tooltip title="删除目标">
            <Button
              danger
              size="small"
              type="text"
              icon={<DeleteOutlined />}
              disabled={rows.length <= 1}
              onClick={() => onRemove(row.key)}
            />
          </Tooltip>
        </Space>
      ),
    },
  ];

  const socketAccountColumns: ColumnsType<SocketAccountRow> = [
    { title: '启用', dataIndex: 'enabled', width: 76, render: (value: boolean) => <Switch size="small" checked={value} disabled checkedChildren="开" unCheckedChildren="关" /> },
    { title: '账号用途', dataIndex: 'channel', width: 150 },
    { title: '用户名', dataIndex: 'username', width: 160, render: (value: string) => <span className={styles.monoText}>{value || '-'}</span> },
    { title: '类型', dataIndex: 'type', width: 90, render: (value: string) => <Tag>{value}</Tag> },
    { title: '密码', dataIndex: 'credential', width: 120, render: (value: string) => <Tag>{value}</Tag> },
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
      width: 190,
      render: (value: string, row) => (
        <Input.Password
          placeholder={value === '已加密存储' ? '未修改保持原密码' : '请输入密码'}
          onChange={(event) => {
            if (!socketEditor) return;
            updateSocketEditorAccount(row.key, {
              credential: event.target.value || value,
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
    {
      title: '状态',
      width: 88,
      render: (_, row) => renderStatusCell(buildSocketReportStatus(row), Boolean(socketEnabled[row.key])),
    },
    { title: '协议', width: 120, render: (_, row) => <Space size={4}>{socketProfileTag(row.profile)}<Tag>{row.encoding}</Tag></Space> },
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
    { title: '目标名称', dataIndex: 'name', width: 180, fixed: 'left', render: (value: string) => <Typography.Text strong ellipsis>{value}</Typography.Text> },
    {
      title: '状态',
      width: 88,
      render: (_, row) => renderStatusCell(buildSnmpReportStatus(row), Boolean(snmpEnabled[row.key])),
    },
    { title: '版本', width: 100, render: (_, row) => snmpVersionTag(row.version) },
    { title: '通知', width: 100, render: (_, row) => snmpNotificationTag(row.notificationType) },
    { title: 'Agent 监听', width: 170, render: (_, row) => <span className={styles.monoText}>{endpointText(row.listenIp, row.listenPort)}</span> },
    { title: '通知目标', width: 170, render: (_, row) => <span className={styles.monoText}>{endpointText(row.targetHost, row.targetPort)}</span> },
    {
      title: '安全配置',
      width: 220,
      render: (_, row) => row.version === 'v2'
        ? <span className={styles.monoText}>{row.community}</span>
        : <Space size={4}><Tag>{row.securityName}</Tag><Tag>{row.authProtocol}</Tag><Tag>{row.privProtocol}</Tag></Space>,
    },
    {
      title: 'MIB/清除策略',
      width: 220,
      render: (_, row) => (
	        <Space size={4} wrap>
	          {row.version === 'v2' && row.mibQueryEnabled ? <Tag color="blue">MIB 查询</Tag> : <Tag>MIB 关闭</Tag>}
	          <Tag>{row.clearSeverityPolicy}</Tag>
	        </Space>
      ),
    },
    {
      title: '重试',
      width: 120,
      render: (_, row) => <Tag>{row.timeoutSeconds}s / {row.retries} 次</Tag>,
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
        className={styles.compactScenarioTable}
        columns={scenarioColumns}
        dataSource={fileProfiles}
        rowKey="code"
        size="small"
        loading={pageConfigLoading}
        pagination={renderTablePagination('刷新北向文件配置')}
        scroll={{ x: 1378, y: 560 }}
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
      <Table<NorthboundApiRow>
        className={styles.compactScenarioTable}
        columns={apiColumns}
        dataSource={apiRows}
        rowKey="key"
        size="small"
        pagination={renderTablePagination('刷新北向 API')}
        scroll={{ x: 1254, y: 560 }}
        rowClassName={(row) => (selectedApi?.key === row.key ? styles.selectedRow : '')}
        onRow={(row) => ({ onClick: () => setSelectedApi(row) })}
      />
      <div className={styles.editorSection}>
        <div className={styles.editorSectionHeader}>
          <Space>
            <SafetyCertificateOutlined />
            <Typography.Text strong>API client 与白名单</Typography.Text>
          </Space>
          <Space>
            <Button size="small" icon={<PlusOutlined />} onClick={addApiClient}>
              新增
            </Button>
            <Button
              size="small"
              type="primary"
              loading={apiClientSaving}
              onClick={saveApiClients}
            >
              保存
            </Button>
          </Space>
        </div>
        <Table<ApiClientRow>
          columns={apiClientColumns}
          dataSource={apiClients}
          rowKey="clientKey"
          size="small"
          pagination={false}
          scroll={{ x: 1470, y: 260 }}
          locale={{ emptyText: '未配置 API client，北向 API 使用现有兼容模式' }}
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
        scroll={{ x: 1628, y: 560 }}
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
      width: 88,
      render: (status: string) => {
        const state: ReportState = status === 'success' ? 'success' : status === 'running' ? 'running' : 'failed';
        return reportStateTag(state, state === 'success' ? '成功' : state === 'running' ? '生成中' : '失败');
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
      title: '目标/报文',
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
  const reportSuccessCount = reportRunList.filter((item) => item.status === 'success').length;
  const reportSummary = reportRunList.length > 1
    ? `共 ${reportRunList.length} 个对象：成功 ${reportSuccessCount} · 失败 ${reportRunList.length - reportSuccessCount}`
    : '';

  return (
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

      <Drawer
        title={
          reportCapabilityName || selectedReportStatus?.capabilityName
            ? `${reportCapabilityName || selectedReportStatus?.capabilityName} 上报结果`
            : '上报结果'
        }
        open={Boolean(selectedReportStatus)}
	        onClose={() => {
	          setSelectedReportStatus(null);
	          setReportRunList([]);
	          setReportEventList([]);
	          setReportEventTotal(0);
	          setReportEventLoading(false);
	          setReportCapabilityName('');
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
	                复制报文
	              </Button>
	            )}
	            {selectedReportStatus.artifactType === 'file' && (
	              <Button
	                type="primary"
	                icon={<DownloadOutlined />}
	                onClick={() => {
	                  void downloadReportArtifact(selectedReportStatus).catch(() => {
	                    void message.error('上报文件下载失败');
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
                      void resolveRunReportStatus(row, reportCapabilityName || selectedReportStatus?.capabilityName || '')
                        .then(setSelectedReportStatus);
                    },
                  })}
                />
              </div>
            )}
            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>
                  {selectedReportStatus.artifactType === 'file' ? '文件内容预览' : '上报报文'}
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
        title={selectedApi ? `${selectedApi.name} API` : '北向 API'}
        open={Boolean(selectedApi)}
        onClose={() => setSelectedApi(null)}
        size="large"
        rootClassName={styles.inventoryDrawer}
        destroyOnClose
        extra={selectedApi ? (
          <Space>
            <Typography.Text type="secondary">接口开关</Typography.Text>
            <Switch
              checked={apiEnabled[selectedApi.key]}
              checkedChildren="开"
              unCheckedChildren="关"
              onChange={(checked) => persistApiEnabled(selectedApi, checked)}
            />
          </Space>
        ) : undefined}
      >
        {selectedApi && (
          <Space orientation="vertical" size={16} className={styles.drawerBody}>
            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>基础信息</Typography.Text>
              </div>
              <Descriptions bordered size="small" column={2}>
                <Descriptions.Item label="接口名称">{selectedApi.name}</Descriptions.Item>
                <Descriptions.Item label="模块">{selectedApi.module}</Descriptions.Item>
                <Descriptions.Item label="接口类型">{apiKindTag(getApiMeta(selectedApi).apiKind as ApiKind)}</Descriptions.Item>
                <Descriptions.Item label="方法">{apiMethodTag(selectedApi.method)}</Descriptions.Item>
                <Descriptions.Item label="启用配置">{statusTag(Boolean(apiEnabled[selectedApi.key]))}</Descriptions.Item>
                <Descriptions.Item label="接口 URL" span={2}>
                  <span className={styles.monoText}>{selectedApi.url}</span>
                </Descriptions.Item>
                <Descriptions.Item label="鉴权" span={2}>{normalizeLegacyApiText(selectedApi.auth)}</Descriptions.Item>
                <Descriptions.Item label="后端实现" span={2}>
                  <span className={styles.monoText}>{selectedApi.backendSource}</span>
                </Descriptions.Item>
              </Descriptions>
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
                <Descriptions.Item label="协议场景">{socketProfileTag(selectedSocket.profile)}</Descriptions.Item>
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
                scroll={{ x: 820 }}
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
                  scroll={{ x: 1260 }}
                />
              </div>
            )}

            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>告警字段映射</Typography.Text>
              </div>
              <Table<AlarmFieldMappingRow>
                columns={alarmFieldColumns}
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
	                      {socketProfileTag(socketEditor.profile)}
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
                scroll={{ x: 930 }}
              />
            </div>

            {socketEditor.profile === 'CUCC' && (
	              <div className={styles.editorSection}>
	                <div className={styles.editorSectionHeader}>
	                  <Typography.Text strong>文件同步传输目标</Typography.Text>
	                  <Button size="small" icon={<PlusOutlined />} onClick={() => addSocketEditorDeliveryTarget(socketEditor.key)}>
	                    新增目标
	                  </Button>
	                </div>
	                <Table<DeliveryTargetRow>
	                  columns={createDeliveryTargetEditorColumns(
	                    socketEditorDeliveryTargets,
	                    updateSocketEditorDeliveryTarget,
	                    removeSocketEditorDeliveryTarget,
	                    'socket',
	                    socketEditor.key,
	                  )}
	                  dataSource={socketEditorDeliveryTargets}
                  rowKey="key"
                  size="small"
                  pagination={false}
                  scroll={{ x: 2160 }}
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
        title={selectedSnmp ? selectedSnmp.name : 'SNMP 告警'}
        open={Boolean(selectedSnmp)}
        onClose={() => setSelectedSnmp(null)}
        size="large"
        rootClassName={styles.inventoryDrawer}
        destroyOnClose
        extra={selectedSnmp ? (
          <Space>
            <Switch
              checked={snmpEnabled[selectedSnmp.key]}
              checkedChildren="开"
              unCheckedChildren="关"
              onChange={(checked) => persistSnmpEnabled(selectedSnmp, checked)}
            />
            <Button
              icon={<EditOutlined />}
              onClick={() => {
                const snmpConfig = selectedSnmp;
                setSelectedSnmp(null);
                setSnmpEditor(snmpConfig);
              }}
            >
              编辑
            </Button>
          </Space>
        ) : undefined}
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
                <Descriptions.Item label="启用配置">{statusTag(Boolean(snmpEnabled[selectedSnmp.key]))}</Descriptions.Item>
                <Descriptions.Item label="MIB 查询">{selectedSnmp.version === 'v2' && selectedSnmp.mibQueryEnabled ? '开启' : '关闭'}</Descriptions.Item>
                <Descriptions.Item label="Agent 监听">
                  <span className={styles.monoText}>{endpointText(selectedSnmp.listenIp, selectedSnmp.listenPort)}</span>
                </Descriptions.Item>
                <Descriptions.Item label="通知目标">
                  <span className={styles.monoText}>{endpointText(selectedSnmp.targetHost, selectedSnmp.targetPort)}</span>
                </Descriptions.Item>
                <Descriptions.Item label="安全配置" span={2}>
                  {selectedSnmp.version === 'v2'
                    ? <span className={styles.monoText}>{selectedSnmp.community}</span>
                    : <Space size={4}><Tag>{selectedSnmp.securityName}</Tag><Tag>{selectedSnmp.authProtocol}</Tag><Tag>{selectedSnmp.privProtocol}</Tag></Space>}
                </Descriptions.Item>
                <Descriptions.Item label="清除告警级别">{selectedSnmp.clearSeverityPolicy}</Descriptions.Item>
                <Descriptions.Item label="超时/重试">{selectedSnmp.timeoutSeconds}s / {selectedSnmp.retries} 次</Descriptions.Item>
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
                columns={snmpAlarmFieldColumns}
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
        title={snmpEditor ? `编辑 ${snmpEditor.name}` : '编辑 SNMP 告警'}
        open={Boolean(snmpEditor)}
        onClose={() => setSnmpEditor(null)}
        size="large"
        rootClassName={styles.inventoryDrawer}
        destroyOnClose
        extra={snmpEditor ? (
          <Button
            type="primary"
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
	                  <Form.Item label="版本">
	                    <Select
	                      value={snmpEditor.version}
	                      options={[{ label: 'V2', value: 'v2' }, { label: 'V3', value: 'v3' }]}
	                      onChange={(version) => setSnmpEditor((current) => (current ? {
	                        ...current,
	                        version,
	                        mibQueryEnabled: version === 'v2' ? current.mibQueryEnabled : false,
	                        authProtocol: version === 'v3' ? (current.authProtocol || 'SHA') : current.authProtocol,
	                        privProtocol: version === 'v3' ? (current.privProtocol || 'AES128') : current.privProtocol,
	                      } : current))}
	                    />
	                  </Form.Item>
                  <Form.Item label="通知类型">
                    <Select value={snmpEditor.notificationType} options={[{ label: 'Trap', value: 'Trap' }, { label: 'Inform', value: 'Inform' }]} onChange={(notificationType) => setSnmpEditor((current) => (current ? { ...current, notificationType } : current))} />
                  </Form.Item>
                  <Form.Item label="启用配置">
                    <Switch
                      checked={snmpEnabled[snmpEditor.key]}
                      checkedChildren="开"
                      unCheckedChildren="关"
                      onChange={(checked) => setSnmpEnabled((prev) => ({ ...prev, [snmpEditor.key]: checked }))}
                    />
                  </Form.Item>
	                  <Form.Item label="MIB 查询">
	                    <Tooltip title={snmpEditor.version === 'v3' ? 'v3 目标用于 Trap/Inform 发送；MIB walk/get 使用 v2c Agent' : undefined}>
	                      <Switch
	                        checked={snmpEditor.version === 'v2' && snmpEditor.mibQueryEnabled}
	                        disabled={snmpEditor.version === 'v3'}
	                        checkedChildren="开"
	                        unCheckedChildren="关"
	                        onChange={(mibQueryEnabled) => setSnmpEditor((current) => (current ? { ...current, mibQueryEnabled } : current))}
	                      />
	                    </Tooltip>
	                  </Form.Item>
                  <Form.Item label="Agent IP">
                    <Input value={snmpEditor.listenIp} onChange={(event) => setSnmpEditor((current) => (current ? { ...current, listenIp: event.target.value } : current))} />
                  </Form.Item>
                  <Form.Item label="Agent 端口">
                    <InputNumber value={snmpEditor.listenPort} min={1} max={65535} style={{ width: '100%' }} onChange={(listenPort) => setSnmpEditor((current) => (current ? { ...current, listenPort: Number(listenPort ?? 1) } : current))} />
                  </Form.Item>
                  <Form.Item label="目标 IP">
                    <Input value={snmpEditor.targetHost} onChange={(event) => setSnmpEditor((current) => (current ? { ...current, targetHost: event.target.value } : current))} />
                  </Form.Item>
                  <Form.Item label="目标端口">
                    <InputNumber value={snmpEditor.targetPort} min={1} max={65535} style={{ width: '100%' }} onChange={(targetPort) => setSnmpEditor((current) => (current ? { ...current, targetPort: Number(targetPort ?? 1) } : current))} />
                  </Form.Item>
                </div>
              </Form>
            </div>

            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>安全与运行</Typography.Text>
              </div>
              <Form layout="vertical" className={styles.compactForm}>
                <div className={styles.inventoryFormGrid}>
                  <Form.Item label="Community">
                    <Input value={snmpEditor.community} onChange={(event) => setSnmpEditor((current) => (current ? { ...current, community: event.target.value } : current))} />
                  </Form.Item>
                  <Form.Item label="安全名">
                    <Input value={snmpEditor.securityName} onChange={(event) => setSnmpEditor((current) => (current ? { ...current, securityName: event.target.value } : current))} />
                  </Form.Item>
	                  <Form.Item label="认证算法">
	                    <Select value={snmpEditor.authProtocol ?? 'SHA'} options={snmpAuthProtocolOptions} onChange={(authProtocol) => setSnmpEditor((current) => (current ? { ...current, authProtocol } : current))} />
	                  </Form.Item>
                  <Form.Item label="认证密码">
                    <Input.Password placeholder={snmpEditor.authCredential === '已加密存储' ? '未修改保持原密码' : '请输入认证密码'} onChange={(event) => setSnmpEditor((current) => (current ? { ...current, authCredential: event.target.value || current.authCredential } : current))} />
                  </Form.Item>
	                  <Form.Item label="加密算法">
	                    <Select value={snmpEditor.privProtocol ?? 'AES128'} options={snmpPrivProtocolOptions} onChange={(privProtocol) => setSnmpEditor((current) => (current ? { ...current, privProtocol } : current))} />
	                  </Form.Item>
                  <Form.Item label="加密密码">
                    <Input.Password placeholder={snmpEditor.privCredential === '已加密存储' ? '未修改保持原密码' : '请输入加密密码'} onChange={(event) => setSnmpEditor((current) => (current ? { ...current, privCredential: event.target.value || current.privCredential } : current))} />
                  </Form.Item>
                  <Form.Item label="清除告警级别">
                    <Select value={snmpEditor.clearSeverityPolicy} options={[{ label: '保留原级别', value: '保留原级别' }, { label: '清除置 0', value: '清除置 0' }]} onChange={(clearSeverityPolicy) => setSnmpEditor((current) => (current ? { ...current, clearSeverityPolicy } : current))} />
                  </Form.Item>
                  <Form.Item label="超时（秒）">
                    <InputNumber value={snmpEditor.timeoutSeconds} min={1} style={{ width: '100%' }} onChange={(timeoutSeconds) => setSnmpEditor((current) => (current ? { ...current, timeoutSeconds: Number(timeoutSeconds ?? 1) } : current))} />
                  </Form.Item>
                  <Form.Item label="重试次数">
                    <InputNumber value={snmpEditor.retries} min={0} style={{ width: '100%' }} onChange={(retries) => setSnmpEditor((current) => (current ? { ...current, retries: Number(retries ?? 0) } : current))} />
                  </Form.Item>
                </div>
              </Form>
            </div>

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
                scroll={{ x: 1260 }}
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
                <Typography.Text strong>传输目标</Typography.Text>
                <Button size="small" icon={<PlusOutlined />} onClick={() => addInventoryDeliveryTarget(selectedInventoryConfig.key)}>
                  新增目标
                </Button>
              </div>
              <Table<DeliveryTargetRow>
                columns={createDeliveryTargetEditorColumns(
                  selectedInventoryDeliveryTargets,
                  (targetKey, patch) => updateInventoryDeliveryTarget(selectedInventoryConfig.key, targetKey, patch),
                  (targetKey) => removeInventoryDeliveryTarget(selectedInventoryConfig.key, targetKey),
                  'inventory',
                  selectedInventoryConfig.key,
                )}
                dataSource={selectedInventoryDeliveryTargets}
                rowKey="key"
                size="small"
                pagination={false}
                scroll={{ x: 2160 }}
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
                  { title: '制式/模式', dataIndex: 'scope', width: 128 },
                  { title: '格式', dataIndex: 'format', width: 104, render: (value: Format) => <Tag>{value}</Tag> },
                  { title: '统计周期', dataIndex: 'period', width: 128, render: (value: string) => <Typography.Text strong>{formatPeriodLabel(value)}</Typography.Text> },
                  { title: '生成计划', width: 210, render: (_, record) => formatScheduleLabel(record.period, record.cron) },
                  {
                    title: '压缩',
                    width: 86,
                    render: (_, record) => record.compressionEnabled ? <Tag color="green">开启</Tag> : <Tag>关闭</Tag>,
                  },
                  { title: '压缩格式', dataIndex: 'compressionFormat', width: 120, render: (value: CompressionFormat) => <Tag>{value}</Tag> },
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
                scroll={{ x: 2448 }}
              />
            </div>

            <div className={styles.editorSection}>
              <div className={styles.editorSectionHeader}>
                <Typography.Text strong>传输目标</Typography.Text>
                <Typography.Text type="secondary">{enabledDeliverySummary(fileDeliveryTargets)}</Typography.Text>
              </div>
              <Table<DeliveryTargetRow>
                columns={deliveryTargetColumns}
                dataSource={fileDeliveryTargets}
                rowKey="key"
                size="small"
                pagination={false}
                scroll={{ x: 1260 }}
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
                    value={viewFieldTechFilter}
                    style={{ width: 132 }}
                    options={fieldTechFilterOptions}
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
                loading={pmLoading && selectedViewFieldTarget?.domain === 'PM'}
                pagination={false}
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
              loading={Boolean(selectedScenario && fileProfileSaving[selectedScenario.code])}
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
                      options={scopeOptions}
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
	                      value={normalizeFormatForDomain(record.domain, value)}
	                      style={{ width: 86 }}
	                      options={getFormatOptions(record.domain)}
	                      onChange={(nextFormat) => updateEditorPeriodRow(record.key, { format: normalizeFormatForDomain(record.domain, nextFormat as Format) })}
	                    />
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
                  title: '压缩',
                  width: 86,
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
                  title: '压缩格式',
                  dataIndex: 'compressionFormat',
                  width: 120,
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
	                {
	                  title: '操作',
	                  width: 80,
                  fixed: 'right',
                  render: (_, record) => (
                    <Tooltip title="删除对象">
                      <Button
                        aria-label={`删除 ${record.domain} 对象`}
                        type="text"
                        danger
                        size="small"
                        icon={<DeleteOutlined />}
                        onClick={() => removeEditorPeriodRow(record.key)}
                        disabled={editorPeriodRows.length <= 1}
                      />
                    </Tooltip>
                  ),
                },
              ]}
              dataSource={editorPeriodRows}
              rowKey="key"
	              size="small"
	              pagination={false}
	              scroll={{ x: 2528 }}
	            />
          </div>
          <div className={styles.editorSection}>
            <div className={styles.editorSectionHeader}>
              <Typography.Text strong>传输目标</Typography.Text>
              <Button size="small" icon={<PlusOutlined />} onClick={addFileDeliveryTarget}>
                新增目标
              </Button>
            </div>
            <Table<DeliveryTargetRow>
              columns={createDeliveryTargetEditorColumns(
                fileDeliveryTargets,
                updateFileDeliveryTarget,
                removeFileDeliveryTarget,
                'file',
                '',
              )}
              dataSource={fileDeliveryTargets}
              rowKey="key"
              size="small"
              pagination={false}
              scroll={{ x: 2160 }}
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
  );
}
