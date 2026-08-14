import { useState, useMemo, useCallback, useEffect, useRef } from 'react';
import { useNavigate, useLocation, useParams } from 'react-router-dom';
import {
  Form,
  Input,
  Select,
  Switch,
  Radio,
  Button,
  Space,
  Card,
  Divider,
  Typography,
  Checkbox,
  Table,
  Upload,
  message,
  Modal,
  Drawer,
  Descriptions,
  Alert,
  Tag,
  Tabs,
  Dropdown,
  type UploadFile,
  type MenuProps,
} from 'antd';
import {
  ArrowLeftOutlined,
  UploadOutlined,
  DeleteOutlined,
  SearchOutlined,
  CheckCircleOutlined,
  DownloadOutlined,
  EyeOutlined,
  EditOutlined,
  InboxOutlined,
  MoreOutlined,
} from '@ant-design/icons';
import { useT } from '@/hooks/useT';
import {
  usePlugAndPlayPolicy,
  useProvisioningTasks,
  useSavePlugAndPlayPolicy,
} from '@core/hooks/api/useProvisioning';
import { useDeviceList } from '@core/hooks/api/useDevices';
import { useProductList, useProductMatch } from '@core/hooks/api/useProducts';
import { useSoftwareVersions, useUploadFirmware } from '@core/hooks/api/useSoftware';
import {
  useDeleteDeviceLicense,
  useDeviceLicenses,
} from '@core/hooks/api/useDeviceLicense';
import type { DeviceLicense } from '@core/services/api/deviceLicenseApi';
import { paramModelApi } from '@core/services/api/paramModelApi';
import { quicksettingsApi } from '@core/services/api/quicksettingsApi';
import LicenseImportDrawer from '@/pages/backup/DeviceLicenseLibrary/ImportDrawer';
import {
  normalizeProductTechnology,
  resolveProductClassTechnology,
  resolveProductClassForName,
  toSupportedProductNameOptions,
  type ProductTechnology,
} from './productClassOptions';
import {
  toActualSoftwareVersionOptions,
  toFirmwareVersionOptions,
} from './softwareVersionOptions';
import {
  createParamConfigWorkbook,
  createParamConfigTemplateWorkbook,
  mergeImportedParamConfigs,
  ParamConfigWorkbookError,
  parseParamConfigWorkbook,
  writeParamConfigWorkbookFile,
  type ParamConfigDeviceType,
  type ParamConfigWorkbookMapping,
} from './paramConfigWorkbook';
import { toParamConfigDeviceType } from './paramConfigTemplate';
import { getParamConfigExportFields } from './paramConfigExportFields';
import {
  mergeParamConfigFormValues,
  toParamConfigFormValues,
} from './paramConfigDetail';
import { isParamConfigToolbarEnabled } from './paramConfigToolbarAvailability';
import {
  buildParamConfigImportPreview,
  buildParamConfigInsights,
  type ParamConfigImportPreviewItem,
  type ParamConfigSource,
  type ParamConfigValidationStatus,
} from './paramConfigInsights';
import { getPolicyModuleActionAvailability } from './policyActionAvailability';
import { getPolicySubmitAvailability } from './policySubmitAvailability';
import ProductClassSelect from './components/ProductClassSelect';
import PolicyReadOnlySection from './components/PolicyReadOnlySection';
import CommonParameterConfigPanel, { ParameterConfigFields } from './CommonParameterConfigPanel';
import { sanitizeCommonParamConfig } from './commonParameterConfig';
import { buildParamConfigListPolicyUpdate } from './paramConfigPersistence';

const { Text, Title } = Typography;

// Types
type ExecuteType = '0' | '1';
type EnableType = '0' | '1';

// T-0136: 保留为 export 占位，避免 TS6196 同时不破坏未来可能复用
export interface _PolicyForm {
  selfStartEnable: EnableType;
  policyName: string;
  productClasses: string[];
  executeType: ExecuteType;
  functionModule: '0' | '1' | '2'; // 0-software upgrade, 1-license, 2-self config
  // Software Upgrade
  upgradeEnable: EnableType;
  specifyVersionType: EnableType;
  originalVersion: string[];
  targetVersion: string;
  preserveSetting: EnableType;
  // License
  licenseEnable: EnableType;
  // Self Config
  selfConfigEnable: EnableType;
  switchEnable: EnableType;
}

// Device type for param config
type DeviceType = 'eNB' | 'gNB' | 'GSM';

// IPSECS Tunnel for eNB
interface IpsecTunnel {
  key: string;
  TUNNEL_INDEX?: string;
  TUNNEL_ENABLE: '0' | '1';
  LEFT_AUTH: string;
  RIGHT_AUTH: string;
  TUNNEL_GATEWAY: string;
  LEFT_IDENTIFIER: string;
  RIGHT_IDENTIFIER: string;
  LEFT_CERT: string;
  SECRET_KEY: string;
  RIGHT_SECRET_KEY: string;
  LEFTSOURCEIP: string;
  LEFT_SUBNET: string;
  RIGHT_SUBNET: string;
  IKE_ENCRYPTION: string;
  IKE_DH_GROUP: string;
  IKE_AUTHENTICATION: string;
  ESP_ENCRYPTION: string;
  ESP_DH_GROUP: string;
  ESP_AUTHENTICATION: string;
  IKELIFETIME: string;
  KEYLIFE: string;
  REKEYMARGIN: string;
  DPDACTION: string;
  DPDDELAY: string;
  FRAGMENTATION?: string;
  LEFT_INTERFACE?: string;
  FORCEENCAPS?: string;
}

// 自定义参数项
interface CustomParam {
  name: string;
  value: string;
  trPath: string;
}

// AMF配置项（支持多个）
interface AmfItem {
  amfIp: string;
  amfPort: string;
}

// MME配置项（支持多个）
interface MmeItem {
  mmeIp: string;
}

// Param Config item for new design
interface ParamConfig {
  id: string;
  deviceType: DeviceType;
  serialNumber: string;
  cellName: string;
  updatedBy: string;
  updatedAt: string;
  sheetParameters?: Record<string, Record<string, unknown>[]>;
  workbookMappings?: ParamConfigWorkbookMapping[];
  // ========== eNB 基础配置字段 ==========
  eNodeBId?: string;
  bandsSupport?: number;
  bandWidth?: string;
  frequency?: number;
  subframeAssignment?: number;
  specialSubframePatterns?: number;
  plmnId?: string;
  tac?: number;
  cellIdentity?: number;
  phycellid?: number;
  rootSequenceIndex?: number;
  carrierType?: 'FDD' | 'TDD';  // 载波制式（仅DXDF产品）
  timeZoneUtc?: string;  // UTC时区
  // ========== eNB 核心网配置 ==========
  halobEnable?: '0' | '1';
  mmeList?: MmeItem[];
  mme?: string;
  // ========== eNB IP配置 ==========
  serviceIp?: string;
  serviceMask?: string;
  serviceGateway?: string;
  serviceGatewayMask?: string;
  serviceVlan?: number;
  mgmtMask?: string;
  mgmtGateway?: string;
  mgmtGatewayMask?: string;
  mgmtVlan?: number;
  // ========== eNB IPSECS配置 ==========
  ipsecEnable?: '0' | '1';
  leftInterface?: string;
  ipsecList?: IpsecTunnel[];
  mtu?: number;  // MTU配置
  // ========== eNB 功率控制配置 ==========
  totalTxPower?: string;
  powerRamping?: number;
  preambleInitTargetPower?: number;
  poNominalPusch?: number;
  poNominalPucch?: number;
  // ========== eNB 自定义参数 ==========
  customParams?: CustomParam[];
  // ========== gNB 基础配置字段 ==========
  gnbName?: string;  // gNB名称
  gnbId?: string;
  gnbIdLength?: number;  // gNB ID长度，范围22~32
  pci?: number;  // 物理小区标识，范围0~1007
  freqBandIndicator?: number;  // 频带指示，范围1~1024
  nrarfcnndl?: number;  // 下行NRARFCN，范围0~3279165
  dlbandwidth?: string;  // 下行带宽
  ssbFrequency?: number;  // SSB频率号，范围0~3279165
  nrarfcnul?: number;  // 上行NRARFCN
  frameOffset?: number;
  arfcn?: number;
  ssbAbsoluteFrequency?: number;
  sliceSst?: number;
  sliceSd?: string;
  prachConfigIndex?: number;
  // ========== gNB TDD Pattern配置 ==========
  subframeSlot?: string;
  subframeSlotDlUl?: string;
  slotConfig?: string;
  // ========== gNB 同步配置 ==========
  gpsSync?: '0' | '1';
  ntpSync?: '0' | '1';
  offsetToPointA?: number;
  kssb?: number;
  // ========== gNB PLMN配置 ==========
  nci?: string;
  ranac?: number;
  plmnConfigList?: PlmnConfigItem[];  // PLMN配置列表
  sliceConfigList?: SliceConfigItem[];  // 切片配置列表
  // ========== gNB AMF配置 ==========
  amfList?: AmfItem[];
  amfIp?: string;
  amfPlmnId?: string;
  amfDefault?: '0' | '1';
  // ========== gNB IP配置 ==========
  omMask?: string;
  // ========== gNB IPSec配置 ==========
  ipsecImsi?: string;
  ipsecKey?: string;
  ipsecOpc?: string;
  // ========== gNB DNS配置 ==========
  dns1?: string;
  dns2?: string;
  // ========== gNB WAN配置 ==========
  addressType?: 'DHCP' | 'Static' | 'DHCPv6' | 'Staticv6';
  bearType?: string;
  ipAddress?: string;
  subnetMask?: string;
  prefixLength?: number;
  wanGateway?: string;
  wanVlanId?: number;
  vlanName?: string;
  // ========== gNB LAN配置 ==========
  lanIp?: string;
  lanSubnetMask?: string;
  // ========== GSM 基础参数 ==========
  ipaUnitid?: string;
  bscServiceIp?: string;  // BSC服务IP
  omlRemoteIp?: string;
  omlRemoteIpBak?: string;
  rfPower?: number;
  // ========== GSM DNS/时区配置 ==========
  // dns1/dns2 与 gNB 段共用（去重 duplicate identifier）
  localTimezoneName?: string;  // 时区名称
  // ========== GSM Route配置 ==========
  onboot?: 'yes' | 'no';
  routeGateway?: string;
  netAddr?: string;
  netMask?: string;
  // ========== GSM WAN配置 ==========
  wanEnable?: '0' | '1';
  ipMode?: 'static' | 'dhcp';
  ipAddr?: string;
  wanNetMask?: string;
  gateway?: string;
  vlanId?: number;
}

// gNB PLMN配置项（支持多个）
interface PlmnConfigItem {
  plmnId: string;
  primary: '0' | '1';
}

// gNB 切片配置项（支持多个）
interface SliceConfigItem {
  sd: '0' | '1';  // 0-空，1-非空
  sdValue: string;
}

// Param Config mock data (for new design) - eNB, gNB, GSM one each
const _MOCK_PARAM_CONFIGS: ParamConfig[] = [
  {
    id: '1',
    deviceType: 'eNB',
    serialNumber: 'ENB00001',
    cellName: 'Cell-001',
    bandsSupport: 38,
    bandWidth: '20MHz',
    frequency: 36000,
    subframeAssignment: 2,
    specialSubframePatterns: 7,
    plmnId: '46001',
    tac: 1001,
    gnbId: '123456',
    gnbIdLength: 24,
    cellIdentity: 12345678,
    nci: '4600112345678901',
    arfcn: 360000,
    ssbAbsoluteFrequency: 3600000,
    frameOffset: 0,
    prachConfigIndex: 0,
    sliceSst: 1,
    sliceSd: '010203',
    phycellid: 150,
    rootSequenceIndex: 0,
    // 同步配置
    gpsSync: '1',
    ntpSync: '0',
    offsetToPointA: 0,
    kssb: 0,
    // IP配置
    serviceIp: '192.168.100.50',
    serviceMask: '255.255.255.0',
    omMask: '255.255.255.0',
    serviceGateway: '192.168.100.1',
    serviceGatewayMask: '255.255.255.0',
    mgmtGateway: '192.168.200.1',
    mgmtGatewayMask: '255.255.255.0',
    serviceVlan: 100,
    mgmtVlan: 200,
    // AMF配置
    amfList: [
      { amfIp: '192.168.1.100', amfPort: '38412' },
    ],
    halobEnable: '0',
    mme: '192.168.1.100',
    // IPSECS
    ipsecEnable: '1',
    leftInterface: 'eth0',
    ipsecList: [
      {
        key: '1',
        TUNNEL_ENABLE: '1',
        LEFT_AUTH: 'psk',
        RIGHT_AUTH: 'psk',
        TUNNEL_GATEWAY: '10.0.0.1',
        LEFT_IDENTIFIER: 'left@example.com',
        RIGHT_IDENTIFIER: 'right@example.com',
        LEFT_CERT: '',
        SECRET_KEY: 'secret123',
        RIGHT_SECRET_KEY: '',
        LEFTSOURCEIP: '%config',
        LEFT_SUBNET: '192.168.1.0/24',
        RIGHT_SUBNET: '10.0.0.0/24',
        IKE_ENCRYPTION: 'aes128',
        IKE_DH_GROUP: 'modp2048',
        IKE_AUTHENTICATION: 'sha256',
        ESP_ENCRYPTION: 'aes128',
        ESP_DH_GROUP: 'modp2048',
        ESP_AUTHENTICATION: 'sha256',
        IKELIFETIME: '8h',
        KEYLIFE: '1h',
        REKEYMARGIN: '9m',
        DPDACTION: 'restart',
        DPDDELAY: '30s',
      },
    ],
    // 自定义参数
    customParams: [
      { name: 'param1', value: 'value1', trPath: 'Device.X_CUSTOM.Param1' },
      { name: 'param2', value: 'value2', trPath: 'Device.X_CUSTOM.Param2' },
    ],
    updatedBy: 'admin',
    updatedAt: '2026-04-07 10:00:00',
  },
  {
    id: '2',
    deviceType: 'gNB',
    serialNumber: 'GNB00001',
    cellName: 'NR-Cell-001',
    gnbName: 'gNB-Beijing-001',
    gnbId: '123456',
    gnbIdLength: 24,
    pci: 150,
    plmnId: '46001',
    tac: 1001,
    nci: '12345678901234',
    ranac: 100,
    freqBandIndicator: 78,
    nrarfcnndl: 360000,
    dlbandwidth: '100MHz',
    ssbFrequency: 3600000,
    arfcn: 360000,
    ssbAbsoluteFrequency: 3600000,
    frameOffset: 0,
    prachConfigIndex: 0,
    sliceSst: 1,
    sliceSd: '010203',
    // TDD Pattern
    subframeSlot: 'DSDDU',
    subframeSlotDlUl: 'DDDDDDDSUU',
    // PLMN Config
    plmnConfigList: [
      { plmnId: '46001', primary: '1' },
    ],
    // Slice Config
    sliceConfigList: [
      { sd: '1', sdValue: '010203' },
    ],
    // Sync
    gpsSync: '1',
    ntpSync: '0',
    offsetToPointA: 0,
    kssb: 0,
    // AMF Config
    amfIp: '192.168.1.100',
    amfPlmnId: '46001',
    amfDefault: '1',
    amfList: [
      { amfIp: '192.168.1.100', amfPort: '38412' },
    ],
    // IP Config
    serviceIp: '192.168.100.50',
    serviceMask: '255.255.255.0',
    omMask: '255.255.255.0',
    serviceGateway: '192.168.100.1',
    serviceGatewayMask: '255.255.255.0',
    mgmtGateway: '192.168.200.1',
    mgmtGatewayMask: '255.255.255.0',
    serviceVlan: 100,
    mgmtVlan: 200,
    // WAN Config
    addressType: 'Static',
    bearType: 'Ethernet',
    ipAddress: '192.168.1.50',
    subnetMask: '255.255.255.0',
    wanGateway: '192.168.1.1',
    wanVlanId: 100,
    vlanName: 'WAN-VLAN',
    // LAN Config
    lanIp: '192.168.2.1',
    lanSubnetMask: '255.255.255.0',
    // DNS Config
    dns1: '8.8.8.8',
    dns2: '8.8.4.4',
    // IPSec Config
    ipsecEnable: '0',
    ipsecImsi: '',
    ipsecKey: '',
    ipsecOpc: '',
    customParams: [
      { name: 'gnbParam1', value: 'value1', trPath: 'Device.X_GNB.Param1' },
    ],
    updatedBy: 'admin',
    updatedAt: '2026-04-07 11:00:00',
  },
  {
    id: '3',
    deviceType: 'GSM',
    serialNumber: 'GSM00001',
    cellName: 'GSM-Cell-001',
    ipaUnitid: 'Ipa[65535],Id[0]',
    bscServiceIp: '192.168.1.100',
    omlRemoteIp: '192.168.1.200',
    omlRemoteIpBak: '192.168.1.201',
    rfPower: 40,
    // DNS Config
    dns1: '8.8.8.8',
    dns2: '8.8.4.4',
    // Timezone
    localTimezoneName: 'Asia/Shanghai',
    // Route Config
    onboot: 'yes',
    routeGateway: '192.168.1.1',
    netAddr: '192.168.1.0',
    netMask: '255.255.255.0',
    // WAN Config
    wanEnable: '1',
    ipMode: 'static',
    ipAddr: '192.168.1.50',
    wanNetMask: '255.255.255.0',
    gateway: '192.168.1.1',
    vlanId: 100,
    customParams: [
      { name: 'gsmParam1', value: 'value1', trPath: 'Device.X_GSM.Param1' },
    ],
    updatedBy: 'user',
    updatedAt: '2026-04-07 12:00:00',
  },
];
void _MOCK_PARAM_CONFIGS;

// _BANDWIDTH_OPTIONS_DXDF 占位常量已拆到同级 ./constants.ts
// （react-refresh/only-export-components：页面文件只导出组件）。

export default function AddPolicyPage() {
  const t = useT();
  const { data: productCatalog, isLoading: productCatalogLoading } = useProductList();
  const navigate = useNavigate();
  const location = useLocation();
  const { id = '' } = useParams();
  const [form] = Form.useForm();
  const productTechnology = Form.useWatch('productTechnology', form) as ProductTechnology | undefined;
  const productTechnologyOptions = useMemo(() => [
    { label: t('provision.productTechnology.lte'), value: 'lte' },
    { label: t('provision.productTechnology.nr'), value: 'nr' },
    { label: t('provision.productTechnology.gsm'), value: 'gsm' },
  ], [t]);
  const productClassOptions = useMemo(
    () => toSupportedProductNameOptions(
      productCatalog?.items,
      productTechnology,
    ),
    [productCatalog?.items, productTechnology],
  );
  const [firmwareImportForm] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const submittingRef = useRef(false);

  // Get mode from URL path
  const pathParts = location.pathname.split('/');
  const lastPart = pathParts[pathParts.length - 2];
  const isEdit = lastPart === 'edit';
  const isView = lastPart === 'view';
  const moduleActions = getPolicyModuleActionAvailability(
    isView ? 'view' : isEdit ? 'edit' : 'create',
  );
  const { data: persistedPolicy, isLoading: policyLoading } = usePlugAndPlayPolicy(isEdit || isView ? id : '');
  const savePolicyMutation = useSavePlugAndPlayPolicy(isEdit ? id : undefined);
  const hasPersistedPolicy = Boolean(persistedPolicy);
  const submitAvailability = getPolicySubmitAvailability({
    isEdit,
    policyLoading,
    hasPersistedPolicy,
    productCatalogLoading,
    submitting: loading || savePolicyMutation.isPending,
  });

  // State for software upgrade
  const [firmwareImportVisible, setFirmwareImportVisible] = useState(false);
  const [firmwareFileList, setFirmwareFileList] = useState<UploadFile[]>([]);

  // State for license
  const [licenseSearchText, setLicenseSearchText] = useState('');
  const [licenseImportModalVisible, setLicenseImportModalVisible] = useState(false);

  // State for self config (new design - config list)
  const [paramConfigList, setParamConfigList] = useState<ParamConfig[]>([]);
  const [configSearchText, setConfigSearchText] = useState('');
  const [configSourceFilter, setConfigSourceFilter] = useState<ParamConfigSource | undefined>();
  const [configValidationFilter, setConfigValidationFilter] = useState<ParamConfigValidationStatus | undefined>();
  const [configDetailVisible, setConfigDetailVisible] = useState(false);
  const [configDetailMode, setConfigDetailMode] = useState<'view' | 'edit'>('view');
  const [currentConfig, setCurrentConfig] = useState<ParamConfig | null>(null);
  const [configForm] = Form.useForm();
  const [commonConfigForm] = Form.useForm();
  const [importModalVisible, setImportModalVisible] = useState(false);
  const [paramImportType, setParamImportType] = useState<'append' | 'replace'>('append');
  const [paramFileList, setParamFileList] = useState<UploadFile[]>([]);
  const [importPreview, setImportPreview] = useState<ParamConfigImportPreviewItem[]>([]);
  const [pendingImportedConfigs, setPendingImportedConfigs] = useState<ParamConfig[]>([]);
  const [importPreviewError, setImportPreviewError] = useState('');
  useEffect(() => {
    if (!configDetailVisible || !currentConfig) return;
    configForm.setFieldsValue(toParamConfigFormValues(currentConfig));
  }, [configDetailVisible, configForm, currentConfig]);

  // Current function module
  const [functionModule, setFunctionModule] = useState<'0' | '1' | '2'>('0');
  const [productClasses, setProductClasses] = useState<string[]>([]);
  // Compatibility form field: values are product names. Southbound-only
  // calls still receive a literal product class resolved from the catalog.
  const productName = productClasses[0] ?? '';
  const selectedProduct = productCatalog?.items.find(
    (item) => item.name.trim().toLowerCase() === productName.trim().toLowerCase(),
  );
  const productClass = resolveProductClassForName(productName, productCatalog?.items);
  const { data: productDevicesData, isLoading: actualVersionsLoading } = useDeviceList(
    {
      page: 1,
      pageSize: 1000,
      productId: selectedProduct?.id,
      productClass: selectedProduct?.id ? undefined : productClass,
    },
    { enabled: Boolean(productName) },
  );
  const { data: parameterTaskData } = useProvisioningTasks({
    page: 1,
    pageSize: 1000,
    policyOnly: true,
    policyId: id,
    module: 'self_config',
  }, {
    enabled: Boolean(id),
    refetchInterval: isView ? 10_000 : false,
  });
  const {
    data: productMatchData,
  } = useProductMatch(selectedProduct?.id ? '' : productClass);
  const activeParamDeviceType = useMemo<ParamConfigDeviceType | undefined>(
    () => toParamConfigDeviceType(selectedProduct?.tech ?? productMatchData?.product?.tech),
    [productMatchData, selectedProduct?.tech],
  );
  useEffect(() => {
    if (!activeParamDeviceType || functionModule !== '2') return;
    const current = commonConfigForm.getFieldsValue(true);
    const saved = persistedPolicy?.config?.commonParamConfig;
    commonConfigForm.setFieldsValue({
      deviceType: activeParamDeviceType,
      ...(saved && typeof saved === 'object' ? saved : {}),
      ...(activeParamDeviceType === 'gNB' && !current.gnbIdAllocation && !saved ? {
        gnbIdAllocation: { start: 1, end: 16_777_215, step: 1, reserved: [] },
        pciAllocation: { start: 0, end: 1007, step: 1, reserved: [] },
        gnbIdLength: 24,
      } : {}),
    });
  }, [activeParamDeviceType, commonConfigForm, functionModule, persistedPolicy]);
  const matchedProductId = selectedProduct?.id ?? productMatchData?.product?.id;
  const {
    data: licenseData,
    isLoading: licensesLoading,
    refetch: refetchLicenses,
  } = useDeviceLicenses({
    page: 1,
    pageSize: 1000,
    serialNumber: licenseSearchText || undefined,
    productId: matchedProductId,
  });
  const deleteLicenseMutation = useDeleteDeviceLicense();
  const {
    data: productFirmwareData,
    isLoading: productFirmwareLoading,
  } = useSoftwareVersions(
    { page: 1, pageSize: 100, productId: matchedProductId, fileType: 0 },
    { enabled: Boolean(matchedProductId) },
  );
  const {
    data: legacyClassFirmwareData,
    isLoading: legacyFirmwareLoading,
  } = useSoftwareVersions(
    { page: 1, pageSize: 100, deviceType: productClass, fileType: 0 },
    { enabled: Boolean(productClass) && !matchedProductId },
  );
  const uploadFirmwareMutation = useUploadFirmware();
  const actualVersionOptions = useMemo(
    () => toActualSoftwareVersionOptions(productDevicesData?.items ?? []),
    [productDevicesData],
  );
  const targetVersionOptions = useMemo(
    () => toFirmwareVersionOptions(
      productFirmwareData?.items ?? [],
      legacyClassFirmwareData?.items ?? [],
    ),
    [legacyClassFirmwareData, productFirmwareData],
  );
  const targetVersionsLoading = productFirmwareLoading || legacyFirmwareLoading;

  useEffect(() => {
    if (!persistedPolicy) return;
    const config = persistedPolicy.config ?? {};
    const savedProductClasses = persistedPolicy.productClasses.length
      ? persistedPolicy.productClasses
      : [persistedPolicy.productClass].filter(Boolean);
    const savedProductNames = persistedPolicy.productNames.length
      ? persistedPolicy.productNames
      : savedProductClasses;
    const originalVersionValue = config.originalVersion;
    const originalVersions = Array.isArray(originalVersionValue)
      ? originalVersionValue.map(String).filter(Boolean)
      : String(originalVersionValue ?? '')
        .split(',')
        .map((version) => version.trim())
        .filter(Boolean);
    form.setFieldsValue({
      ...config,
      originalVersion: originalVersions,
      policyName: persistedPolicy.name,
      productTechnology: normalizeProductTechnology(String(config.productTechnology ?? ''))
        ?? resolveProductClassTechnology(
          resolveProductClassForName(savedProductNames[0] ?? '', productCatalog?.items),
          productCatalog?.items,
        ),
      productClasses: savedProductNames,
      executeType: persistedPolicy.executeType === 'auto' ? '0' : '1',
      selfStartEnable: persistedPolicy.enabled,
      upgradeEnable: persistedPolicy.upgradeEnabled,
      targetVersion: persistedPolicy.targetVersion,
      licenseEnable: persistedPolicy.licenseEnabled,
      selfConfigEnable: persistedPolicy.selfConfigEnabled,
    });
    setProductClasses(savedProductNames);
    if (Array.isArray(config.paramConfigList)) {
      setParamConfigList(config.paramConfigList as ParamConfig[]);
    }
  }, [persistedPolicy, form, productCatalog?.items]);

  // Page title
  const pageTitle = useMemo(() => {
    if (isView) return t('common.detail');
    if (isEdit) return t('common.edit') + ' Policy';
    return t('common.add') + ' Policy';
  }, [isView, isEdit, t]);

  // Handle product type change
  const handleProductClassChange = useCallback((value: string) => {
    setProductClasses(value ? [value] : []);
    form.setFieldsValue({ originalVersion: [], targetVersion: undefined });
  }, [form]);

  const handleProductTechnologyChange = useCallback(() => {
    setProductClasses([]);
    form.setFieldsValue({
      productClasses: [],
      originalVersion: [],
      targetVersion: undefined,
    });
  }, [form]);

  const handleOpenFirmwareImport = useCallback(() => {
    if (!productClass) {
      void message.warning(t('software.firmware.selectProductClass'));
      return;
    }
    firmwareImportForm.resetFields();
    setFirmwareFileList([]);
    setFirmwareImportVisible(true);
  }, [firmwareImportForm, productClass, t]);

  const handleCloseFirmwareImport = useCallback(() => {
    setFirmwareImportVisible(false);
    setFirmwareFileList([]);
    firmwareImportForm.resetFields();
  }, [firmwareImportForm]);

  const handleImportFirmware = useCallback(async () => {
    try {
      const values = await firmwareImportForm.validateFields();
      const rawFile = firmwareFileList[0]?.originFileObj as File | undefined;
      if (!rawFile) {
        void message.warning(t('software.firmware.selectFile'));
        return;
      }
      const uploaded = await uploadFirmwareMutation.mutateAsync({
        file: rawFile,
        metadata: {
          version: values.version,
          productId: matchedProductId,
          productClass,
          fileType: 0,
        },
      });
      form.setFieldValue('targetVersion', uploaded.versionCode);
      handleCloseFirmwareImport();
      void message.success(t('software.firmware.importSuccess'));
    } catch (error) {
      if (error instanceof Error && error.message) {
        void message.error(error.message);
      }
    }
  }, [
    firmwareFileList,
    firmwareImportForm,
    form,
    handleCloseFirmwareImport,
    matchedProductId,
    productClass,
    t,
    uploadFirmwareMutation,
  ]);

  // License file table columns
  const licenseColumns = [
    {
      title: t('table.operation'),
      key: 'action',
      width: 80,
      fixed: 'right' as const,
      render: (_: unknown, record: DeviceLicense) => moduleActions.delete ? (
        <Button
          type="link"
          size="small"
          danger
          icon={<DeleteOutlined />}
          loading={deleteLicenseMutation.isPending}
          onClick={() => {
            void deleteLicenseMutation.mutateAsync(record.serialNumber)
              .then(() => message.success(t('common.success')))
              .catch((error: Error) => message.error(error.message));
          }}
        />
      ) : null,
    },
    {
      title: t('provision.deviceCode'),
      dataIndex: 'serialNumber',
      key: 'serialNumber',
    },
    {
      title: 'License ' + t('common.file'),
      dataIndex: 'fileName',
      key: 'fileName',
    },
    {
      title: t('provision.uploadTime'),
      dataIndex: 'updateTime',
      key: 'updateTime',
    },
  ];

  // Parameter configuration summary, validation and latest execution context.
  const selfConfigEnabled = Form.useWatch('selfConfigEnable', form);
  const paramConfigToolbarEnabled = isParamConfigToolbarEnabled(selfConfigEnabled);
  const configInsights = useMemo(
    () => buildParamConfigInsights(paramConfigList),
    [paramConfigList],
  );
  const latestParameterTaskBySerial = useMemo(() => {
    const result = new Map<string, NonNullable<typeof parameterTaskData>['items'][number]>();
    for (const task of parameterTaskData?.items ?? []) {
      if (task.serialNumber && !result.has(task.serialNumber)) result.set(task.serialNumber, task);
    }
    return result;
  }, [parameterTaskData]);

  const handleDownloadParamConfig = useCallback(async (record: ParamConfig) => {
    const paramModelName = selectedProduct?.paramModelName;
    const [quickSettings, mappings] = paramModelName
      ? await Promise.all([
        quicksettingsApi.getGroupsByParamModel(paramModelName),
        paramModelApi.listMappings(paramModelName),
      ])
      : [{ groups: [] }, { items: [] }];
    const safeSerialNumber = record.serialNumber.replace(/[\\/:*?"<>|]+/g, '_');
    await writeParamConfigWorkbookFile(
      createParamConfigWorkbook([record]),
      `${safeSerialNumber}-${t('provision.paramConfigExportFileSuffix')}.xlsx`,
      {
        deviceType: record.deviceType as ParamConfigDeviceType,
        productClass,
        quickSettingsGroups: quickSettings.groups,
        paramMappings: mappings.items,
        quickSettingFields: getParamConfigExportFields(record.deviceType as ParamConfigDeviceType),
      },
    );
    void message.success(t('provision.paramConfigExportSuccess', { count: 1 }));
  }, [productClass, selectedProduct?.paramModelName, t]);

  const handleDeleteParamConfig = useCallback(async (record: ParamConfig) => {
    const previous = paramConfigList;
    const next = previous.filter((item) => item.id !== record.id);
    setParamConfigList(next);
    if (!isEdit || !persistedPolicy) {
      void message.success(t('common.success'));
      return;
    }
    try {
      await savePolicyMutation.mutateAsync(
        buildParamConfigListPolicyUpdate(persistedPolicy, next),
      );
      void message.success(t('common.success'));
    } catch {
      setParamConfigList(previous);
      void message.error(t('common.failed'));
    }
  }, [isEdit, paramConfigList, persistedPolicy, savePolicyMutation, t]);

  const operationColumn = {
    title: t('table.operation'),
    key: 'action',
    width: 180,
    fixed: 'right' as const,
    render: (_: unknown, record: ParamConfig) => {
      const items: MenuProps['items'] = [
        { key: 'edit', label: t('common.edit'), icon: <EditOutlined />,
          onClick: () => {
            setCurrentConfig(record);
            setConfigDetailMode('edit');
            setConfigDetailVisible(true);
          },
        },
        { key: 'delete', label: t('common.delete'), icon: <DeleteOutlined />, danger: true,
          onClick: () => { void handleDeleteParamConfig(record); },
        },
      ];
      return (
        <Space size={4}>
          <Button type="link" size="small" icon={<EyeOutlined />}
            onClick={() => {
              setCurrentConfig(record);
              setConfigDetailMode('view');
              setConfigDetailVisible(true);
            }}>
            {t('common.view')}
          </Button>
          {moduleActions.download && (
            <Button
              type="link"
              size="small"
              icon={<DownloadOutlined />}
              onClick={() => { void handleDownloadParamConfig(record); }}
            >
              {t('common.download')}
            </Button>
          )}
          {moduleActions.edit && moduleActions.delete && (
            <Dropdown menu={{ items }} trigger={['click']}>
              <Button type="text" size="small" icon={<MoreOutlined />} />
            </Dropdown>
          )}
        </Space>
      );
    },
  };
  const summaryColumn = (
    title: string,
    key: 'gnbId' | 'pci' | 'band' | 'bandwidth' | 'frequency' | 'ssbFrequency' | 'tac',
    width = 110,
  ) => ({
    title,
    key,
    width,
    render: (_: unknown, record: ParamConfig) => configInsights.get(record.serialNumber)?.[key] ?? '-',
  });
  const technologyColumns = activeParamDeviceType === 'gNB'
    ? [
      summaryColumn(t('provision.gnbId'), 'gnbId'),
      summaryColumn('PCI', 'pci', 80),
      summaryColumn(t('provision.bandsSupport'), 'band', 100),
      summaryColumn(t('provision.bandwidth'), 'bandwidth', 110),
      summaryColumn(t('provision.nrarfcnDl'), 'frequency', 130),
      summaryColumn(t('provision.ssbFrequency'), 'ssbFrequency', 130),
      summaryColumn('TAC', 'tac', 100),
    ]
    : activeParamDeviceType === 'eNB'
      ? [
        summaryColumn(t('provision.bandsSupport'), 'band', 100),
        summaryColumn(t('provision.bandwidth'), 'bandwidth', 110),
        summaryColumn(t('provision.frequency'), 'frequency', 110),
      ]
      : [];
  const paramConfigColumns = [
    operationColumn,
    {
      title: t('provision.serialNumber'),
      dataIndex: 'serialNumber',
      key: 'serialNumber',
      width: 130,
    },
    {
      title: t('provision.cellName'),
      dataIndex: 'cellName',
      key: 'cellName',
      width: 120,
    },
    ...technologyColumns,
    {
      title: t('provision.configSource'),
      key: 'configSource',
      width: 110,
      render: (_: unknown, record: ParamConfig) => {
        const source = configInsights.get(record.serialNumber)?.source ?? 'manual';
        return t(`provision.configSource.${source}`);
      },
    },
    {
      title: t('provision.validationStatus'),
      key: 'validationStatus',
      width: 110,
      render: (_: unknown, record: ParamConfig) => {
        const status = configInsights.get(record.serialNumber)?.validationStatus ?? 'incomplete';
        const color = status === 'valid' ? 'success' : status === 'conflict' ? 'error' : 'warning';
        return <Tag color={color}>{t(`provision.validationStatus.${status}`)}</Tag>;
      },
    },
    {
      title: t('provision.latestExecution'),
      key: 'latestExecution',
      width: 120,
      render: (_: unknown, record: ParamConfig) => {
        const task = latestParameterTaskBySerial.get(record.serialNumber);
        if (!task) return '-';
        const color = task.status === 'completed' ? 'success'
          : task.status === 'failed' ? 'error'
            : 'processing';
        return <Tag color={color}>{t(`provision.taskStatus.${task.status}`)}</Tag>;
      },
    },
    {
      title: t('provision.updatedAt'),
      dataIndex: 'updatedAt',
      key: 'updatedAt',
      width: 160,
    },
  ];

  // Parameter rows belong to a radio technology while the selected value is a product class.
  const filteredParamConfigList = useMemo(() => {
    let list = paramConfigList;

    if (activeParamDeviceType) {
      list = list.filter(item => item.deviceType === activeParamDeviceType);
    } else if (productClass) {
      list = [];
    }

    // Filter by search text
    if (configSearchText) {
      list = list.filter(item =>
        item.serialNumber.toLowerCase().includes(configSearchText.toLowerCase())
      );
    }

    if (configSourceFilter) {
      list = list.filter((item) => configInsights.get(item.serialNumber)?.source === configSourceFilter);
    }
    if (configValidationFilter) {
      list = list.filter((item) => (
        configInsights.get(item.serialNumber)?.validationStatus === configValidationFilter
      ));
    }

    return list;
  }, [
    activeParamDeviceType,
    configInsights,
    configSearchText,
    configSourceFilter,
    configValidationFilter,
    paramConfigList,
    productClass,
  ]);

  const handleExportConfig = useCallback(async () => {
    if (!productClass) {
      void message.warning(t('provision.selectProductClassFirst'));
      return;
    }
    if (filteredParamConfigList.length === 0) {
      void message.warning(t('provision.noParamConfigToExport'));
      return;
    }

    const paramModelName = selectedProduct?.paramModelName;
    const [quickSettings, mappings] = paramModelName
      ? await Promise.all([
        quicksettingsApi.getGroupsByParamModel(paramModelName),
        paramModelApi.listMappings(paramModelName),
      ])
      : [{ groups: [] }, { items: [] }];
    const workbook = createParamConfigWorkbook(filteredParamConfigList);
    const safeProductClass = (productClass || 'parameter-config').replace(/[\\/:*?"<>|]+/g, '_');
    await writeParamConfigWorkbookFile(
      workbook,
      `${safeProductClass}-${t('provision.paramConfigExportFileSuffix')}.xlsx`,
      {
        deviceType: activeParamDeviceType,
        productClass,
        quickSettingsGroups: quickSettings.groups,
        paramMappings: mappings.items,
        quickSettingFields: getParamConfigExportFields(activeParamDeviceType),
      },
    );
    void message.success(t('provision.paramConfigExportSuccess', {
      count: filteredParamConfigList.length,
    }));
  }, [activeParamDeviceType, filteredParamConfigList, productClass, selectedProduct?.paramModelName, t]);

  const handleDownloadParamConfigTemplate = useCallback(async () => {
    if (!productClass) {
      void message.warning(t('provision.selectProductClassFirst'));
      return;
    }
    if (!activeParamDeviceType) return;
    const paramModelName = selectedProduct?.paramModelName;
    const [quickSettings, mappings] = paramModelName
      ? await Promise.all([
        quicksettingsApi.getGroupsByParamModel(paramModelName),
        paramModelApi.listMappings(paramModelName),
      ])
      : [{ groups: [] }, { items: [] }];
    if (quickSettings.groups.length === 0) {
      void message.warning(t('provision.paramConfigTemplateUnavailable'));
      return;
    }
    await writeParamConfigWorkbookFile(
      createParamConfigTemplateWorkbook(activeParamDeviceType, {
        deviceType: activeParamDeviceType,
        productClass,
        quickSettingsGroups: quickSettings.groups,
        paramMappings: mappings.items,
        quickSettingFields: getParamConfigExportFields(activeParamDeviceType),
      }),
      `${productClass.replace(/[\\/:*?"<>|]+/g, '_')}-${t('provision.paramConfigExportFileSuffix')}.xlsx`,
      {
        deviceType: activeParamDeviceType,
        productClass,
        quickSettingsGroups: quickSettings.groups,
        paramMappings: mappings.items,
        quickSettingFields: getParamConfigExportFields(activeParamDeviceType),
      },
    );
    void message.success(t('provision.paramConfigTemplateDownloaded'));
  }, [activeParamDeviceType, productClass, selectedProduct?.paramModelName, t]);

  const handleOpenParamConfigImport = useCallback(() => {
    if (!productClass) {
      void message.warning(t('provision.selectProductClassFirst'));
      return;
    }
    setParamFileList([]);
    setImportPreview([]);
    setPendingImportedConfigs([]);
    setImportPreviewError('');
    setImportModalVisible(true);
  }, [productClass, t]);

  // Handle config form submit
  const handleConfigFormSubmit = useCallback(() => {
    configForm.validateFields().then(() => {
      const values = configForm.getFieldsValue(true);
      if (currentConfig) {
        // Edit mode
        setParamConfigList(prev => prev.map(item =>
          item.id === currentConfig.id
            ? mergeParamConfigFormValues(currentConfig, values)
            : item
        ));
      }
      setConfigDetailVisible(false);
      setCurrentConfig(null);
      configForm.resetFields();
      message.success(t('common.success'));
    });
  }, [currentConfig, configForm, t]);

  const previewImportConfig = useCallback(async (file: File) => {
    if (!activeParamDeviceType) {
      void message.warning(t('provision.paramConfigDeviceTypeUnavailable'));
      return;
    }

    try {
      const importedAt = new Date().toLocaleString();
      const paramModelName = selectedProduct?.paramModelName;
      const [quickSettings, mappings] = paramModelName
        ? await Promise.all([
          quicksettingsApi.getGroupsByParamModel(paramModelName),
          paramModelApi.listMappings(paramModelName),
        ])
        : [{ groups: [] }, { items: [] }];
      const rows = parseParamConfigWorkbook(
        await file.arrayBuffer(),
        activeParamDeviceType,
        importedAt,
        {
          deviceType: activeParamDeviceType,
          productClass,
          quickSettingsGroups: quickSettings.groups,
          paramMappings: mappings.items,
          quickSettingFields: getParamConfigExportFields(activeParamDeviceType),
        },
      );
      const importId = Date.now();
      const importedConfigs: ParamConfig[] = rows.map((row, index) => ({
        id: `import-${importId}-${index}`,
        deviceType: row.deviceType ?? activeParamDeviceType,
        serialNumber: row.serialNumber,
        cellName: row.cellName ?? '',
        bandsSupport: row.bandsSupport,
        bandWidth: row.bandWidth,
        frequency: row.frequency,
        subframeAssignment: row.subframeAssignment,
        sheetParameters: row.sheetParameters,
        workbookMappings: row.workbookMappings,
        updatedBy: row.updatedBy ?? 'import',
        updatedAt: row.updatedAt ?? importedAt,
      }));
      const preview = buildParamConfigImportPreview(paramConfigList, importedConfigs);
      setPendingImportedConfigs(importedConfigs);
      setImportPreview(preview);
      setImportPreviewError('');
    } catch (error) {
      setPendingImportedConfigs([]);
      setImportPreview([]);
      if (error instanceof ParamConfigWorkbookError) {
        const errorMessage = error.code === 'empty_workbook'
          ? t('provision.paramConfigImportEmpty')
          : error.code === 'template_no_data'
            ? t('provision.paramConfigTemplateNoData')
            : error.code === 'missing_columns'
              ? t('provision.paramConfigImportMissingColumns', { field: error.field ?? '-' })
              : t('provision.paramConfigImportInvalidRow', {
            row: error.row ?? '-',
            field: error.field ?? '-',
          });
        setImportPreviewError(errorMessage);
        return;
      }
      setImportPreviewError(t('provision.paramConfigImportFailed'));
    }
  }, [activeParamDeviceType, paramConfigList, selectedProduct?.paramModelName, t]);

  const applyImportConfig = useCallback(() => {
    if (pendingImportedConfigs.length === 0 || importPreview.some((item) => item.action === 'duplicate')) {
      void message.warning(t('provision.paramConfigImportResolveConflicts'));
      return;
    }
    setParamConfigList((previous) =>
      mergeImportedParamConfigs(previous, pendingImportedConfigs, paramImportType),
    );
    setImportModalVisible(false);
    setParamFileList([]);
    setImportPreview([]);
    setPendingImportedConfigs([]);
    void message.success(t('provision.paramConfigImportSuccess', {
      count: pendingImportedConfigs.length,
    }));
  }, [importPreview, paramImportType, pendingImportedConfigs, t]);

  // Handle submit
  const handleSubmit = useCallback(async () => {
    if (submittingRef.current || (isEdit && !hasPersistedPolicy) || productCatalogLoading) return;
    submittingRef.current = true;
    setLoading(true);
    try {
      await form.validateFields();
      // Module panels are conditionally mounted. validateFields() only returns
      // values from the currently mounted panel, while getFieldsValue(true)
      // also includes the preserved values of the other selected modules.
      const values = form.getFieldsValue(true);
      let commonParamConfig = persistedPolicy?.config?.commonParamConfig ?? {};
      if (values.selfConfigEnable && functionModule === '2') {
        await commonConfigForm.validateFields();
        commonParamConfig = sanitizeCommonParamConfig({
          ...commonConfigForm.getFieldsValue(true),
          deviceType: activeParamDeviceType,
        });
      }
      const selectedProductNames = (values.productClasses ?? []) as string[];
      const selectedProductClass = resolveProductClassForName(
        selectedProductNames[0] ?? '',
        productCatalog?.items,
      );
      await savePolicyMutation.mutateAsync({
        name: values.policyName,
        enabled: Boolean(values.selfStartEnable),
        productName: selectedProductNames[0] ?? '',
        productNames: selectedProductNames,
        productClass: selectedProductClass,
        productClasses: selectedProductNames,
        executeType: values.executeType === '0' ? 'auto' : 'manual',
        priority: Number(values.priority || 100),
        upgradeEnabled: Boolean(values.upgradeEnable),
        targetVersion: values.targetVersion || '',
        licenseEnabled: Boolean(values.licenseEnable),
        selfConfigEnabled: Boolean(values.selfConfigEnable),
        config: {
          ...values,
          productName: selectedProductNames[0] ?? '',
          productNames: selectedProductNames,
          productClass: selectedProductClass,
          productClasses: selectedProductNames,
          commonParamConfig,
          paramConfigList,
        },
      });
      message.success(t('common.success'));
      navigate('/device/plug-and-play');
    } catch (error) {
      console.error('Validation error:', error);
      const firstInvalidField = (error as { errorFields?: Array<{ name?: (string | number)[] }> })
        ?.errorFields?.[0]?.name;
      if (firstInvalidField) {
        form.scrollToField(firstInvalidField, { block: 'center' });
      }
      if ((error as { response?: { status?: number } })?.response?.status === 409) {
        message.error(t('provision.enabledPolicyProductConflict'));
      }
    } finally {
      submittingRef.current = false;
      setLoading(false);
    }
  }, [activeParamDeviceType, commonConfigForm, form, functionModule, hasPersistedPolicy, isEdit, navigate, persistedPolicy?.config?.commonParamConfig, productCatalog?.items, productCatalogLoading, t, savePolicyMutation, paramConfigList]);

  // Handle cancel
  const handleCancel = useCallback(() => {
    navigate('/device/plug-and-play');
  }, [navigate]);

  // Render software upgrade config
  const renderSoftwareUpgradeConfig = () => (
    <Card size="small" title={
      <Space>
        <span>{t('provision.softwareUpgrade')}</span>
        <Form.Item name="upgradeEnable" valuePropName="checked" noStyle>
          <Switch
            size="small"
            checkedChildren={t('common.on')}
            unCheckedChildren={t('common.off')}
          />
        </Form.Item>
      </Space>
    } style={{ marginBottom: 16 }}>
      {/* 初始版本区域 */}
      <div style={{ padding: '16px 0 0' }}>
        <div style={{ marginBottom: 12 }}>
          <Space size={16}>
            <Text>{t('provision.originalVersion')}</Text>
            <Form.Item name="specifyVersionType" noStyle>
              <Radio.Group>
                <Radio value="specify">{t('provision.specifyVersion')}</Radio>
                <Radio value="all">{t('provision.allVersions')}</Radio>
              </Radio.Group>
            </Form.Item>
          </Space>
        </div>

        <Form.Item
          noStyle
          shouldUpdate={(prev, curr) =>
            prev.specifyVersionType !== curr.specifyVersionType ||
            prev.upgradeEnable !== curr.upgradeEnable
          }
        >
          {({ getFieldValue }) => {
            const specifyVersionType = getFieldValue('specifyVersionType');

            if (specifyVersionType === 'all') {
              return (
                <div style={{
                  padding: '8px 12px',
                  background: '#f6f8fc',
                  borderRadius: 4,
                  border: '1px solid #E9EDF9',
                }}>
                  <Text type="secondary">{t('provision.allVersionsSelectedHint')}</Text>
                </div>
              );
            }

            return (
              <Form.Item
                name="originalVersion"
                style={{ marginBottom: 0 }}
                rules={[{
                  validator: (_, value) => {
                    if (!getFieldValue('upgradeEnable') || (Array.isArray(value) && value.length > 0)) {
                      return Promise.resolve();
                    }
                    return Promise.reject(new Error(t('common.pleaseSelect')));
                  },
                }]}
              >
                <Select
                  mode="multiple"
                  allowClear
                  showSearch
                  optionFilterProp="label"
                  placeholder={t('provision.selectFromList')}
                  options={actualVersionOptions}
                  loading={actualVersionsLoading}
                  notFoundContent={actualVersionsLoading ? t('common.loading') : t('common.noData')}
                  style={{ width: 360 }}
                />
              </Form.Item>
            );
          }}
        </Form.Item>
      </div>

      {/* 分隔线 */}
      <Divider style={{ margin: '8px 0 16px' }} />

      {/* 目标版本 + 保留配置 */}
      <div style={{ display: 'flex', alignItems: 'flex-end', gap: 16 }}>
        <Form.Item
          name="targetVersion"
          label={t('provision.targetVersion')}
          style={{ marginBottom: 0 }}
          rules={[{
            validator: (_, value) => {
              if (!form.getFieldValue('upgradeEnable') || value) return Promise.resolve();
              return Promise.reject(new Error(t('common.pleaseSelect')));
            },
          }]}
        >
          <Select
            placeholder={t('common.pleaseSelect')}
            style={{ width: 280 }}
            options={targetVersionOptions}
            loading={targetVersionsLoading}
            showSearch
            optionFilterProp="label"
            notFoundContent={targetVersionsLoading ? t('common.loading') : t('common.noData')}
          />
        </Form.Item>
        {moduleActions.import && (
          <Button icon={<UploadOutlined />} onClick={handleOpenFirmwareImport}>
            {t('provision.importVersionPackage')}
          </Button>
        )}
        <Form.Item name="preserveSetting" valuePropName="checked" style={{ marginBottom: 0 }}>
          <Checkbox>{t('provision.preserveConfig')}</Checkbox>
        </Form.Item>
      </div>
    </Card>
  );

  // Render license config
  const renderLicenseConfig = () => (
    <Card size="small" title={
      <Space>
        <span>License</span>
        <Form.Item name="licenseEnable" valuePropName="checked" noStyle>
          <Switch
            size="small"
            disabled={isView}
            checkedChildren={t('common.on')}
            unCheckedChildren={t('common.off')}
          />
        </Form.Item>
      </Space>
    } style={{ marginBottom: 16 }}>
      <div style={{ padding: '16px 0' }}>
        <div style={{ marginBottom: 12, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Text type="secondary">{t('provision.importLicenseFile')}</Text>
          <Space>
            <Input
              placeholder={t('provision.deviceCode')}
              prefix={<SearchOutlined />}
              value={licenseSearchText}
              onChange={(e) => setLicenseSearchText(e.target.value)}
              style={{ width: 200 }}
              size="small"
            />
            {moduleActions.import && (
              <Button size="small" type="primary" icon={<UploadOutlined />} onClick={() => setLicenseImportModalVisible(true)}>
                {t('common.import')}
              </Button>
            )}
          </Space>
        </div>
        <Table
          columns={licenseColumns}
          dataSource={licenseData?.items ?? []}
          rowKey="serialNumber"
          loading={licensesLoading}
          pagination={false}
          size="small"
          scroll={{ y: 250 }}
        />
      </div>
    </Card>
  );

  // Render self config (new design - config list)
  const renderSelfConfig = () => (
    <Card size="small" title={
      <Space>
        <span>{t('provision.selfConfig')}</span>
        <Form.Item name="selfConfigEnable" valuePropName="checked" noStyle>
          <Switch
            size="small"
            disabled={isView}
            checkedChildren={t('common.on')}
            unCheckedChildren={t('common.off')}
          />
        </Form.Item>
      </Space>
    } style={{ marginBottom: 16 }}>
      <Tabs
        defaultActiveKey="common"
        items={[
          {
            key: 'common',
            label: t('provision.commonParameters'),
            children: (
              <div style={{ paddingTop: 8 }}>
                <CommonParameterConfigPanel
                  form={commonConfigForm}
                  deviceType={activeParamDeviceType}
                  paramModelName={selectedProduct?.paramModelName}
                  disabled={isView || !paramConfigToolbarEnabled}
                />
              </div>
            ),
          },
          {
            key: 'overrides',
            label: t('provision.deviceParameterOverrides'),
            children: (
              <div style={{ padding: '8px 0' }}>
                <Alert
                  type="info"
                  showIcon
                  title={t('provision.deviceParameterOverridesHint')}
                  style={{ marginBottom: 16 }}
                />
                <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
                  <Space wrap>
                    <Input
                      placeholder={t('provision.searchBySerialNumber')}
                      prefix={<SearchOutlined />}
                      value={configSearchText}
                      onChange={(e) => setConfigSearchText(e.target.value)}
                      style={{ width: 220 }}
                      allowClear
                    />
                    <Select
                      allowClear
                      placeholder={t('provision.configSource')}
                      value={configSourceFilter}
                      onChange={setConfigSourceFilter}
                      style={{ width: 130 }}
                      options={(['import', 'batch_plan', 'manual'] as const).map((value) => ({
                        value,
                        label: t(`provision.configSource.${value}`),
                      }))}
                    />
                    <Select
                      allowClear
                      placeholder={t('provision.validationStatus')}
                      value={configValidationFilter}
                      onChange={setConfigValidationFilter}
                      style={{ width: 130 }}
                      options={(['valid', 'conflict', 'incomplete'] as const).map((value) => ({
                        value,
                        label: t(`provision.validationStatus.${value}`),
                      }))}
                    />
                  </Space>
                  <Space>
                    <Button
                      icon={<DownloadOutlined />}
                      disabled={!paramConfigToolbarEnabled}
                      onClick={handleDownloadParamConfigTemplate}
                    >
                      {t('provision.downloadTemplate')}
                    </Button>
                    {moduleActions.import && (
                      <Button
                        icon={<UploadOutlined />}
                        disabled={!paramConfigToolbarEnabled}
                        onClick={handleOpenParamConfigImport}
                      >
                        {t('common.import')}
                      </Button>
                    )}
                    <Button
                      icon={<DownloadOutlined />}
                      disabled={!paramConfigToolbarEnabled}
                      onClick={handleExportConfig}
                    >
                      {t('common.export')}
                    </Button>
                  </Space>
                </div>
                <Table
                  columns={paramConfigColumns}
                  dataSource={filteredParamConfigList}
                  rowKey="id"
                  pagination={{
                    showSizeChanger: true,
                    showQuickJumper: true,
                    showTotal: (total) => t('table.totalCount', { count: total }),
                  }}
                  size="small"
                  scroll={{ x: 1200 }}
                />
              </div>
            ),
          },
        ]}
      />
    </Card>
  );

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column', background: '#F6F7FB' }}>
      {/* Page Header */}
      <div style={{
        background: '#fff',
        padding: '16px 24px',
        borderBottom: '1px solid #f0f0f0',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <Button type="text" icon={<ArrowLeftOutlined />} onClick={handleCancel} />
          <Title level={4} style={{ margin: 0 }}>{pageTitle}</Title>
        </div>
        {!isView && (
          <Space>
            <Button onClick={handleCancel}>{t('common.cancel')}</Button>
            <Button
              type="primary"
              loading={submitAvailability.loading}
              disabled={submitAvailability.disabled}
              onClick={handleSubmit}
            >
              {t('common.confirm')}
            </Button>
          </Space>
        )}
      </div>

      <div style={{ flex: 1, overflow: 'auto', padding: 16 }}>
        <Form
          form={form}
          layout="vertical"
          initialValues={{
            executeType: '0',
            functionModule: '0',
            upgradeEnable: false,
            originalVersion: [],
            licenseEnable: false,
            selfConfigEnable: false,
            switchEnable: false,
            specifyVersionType: 'specify',
            preserveSetting: false,
          }}
        >
          {/* Basic Info Card */}
          <PolicyReadOnlySection readOnly={isView}>
            <Card size="small" title={t('common.basicInfo')} style={{ marginBottom: 16 }}>
              <Descriptions column={1} bordered size="small" labelStyle={{ width: 120 }} contentStyle={{ flex: 1 }}>
                <Descriptions.Item label={t('provision.settingSwitch')}>
                  <Form.Item name="selfStartEnable" valuePropName="checked" noStyle>
                    <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
                  </Form.Item>
                </Descriptions.Item>
                <Descriptions.Item label={t('provision.policyName')}>
                  <Form.Item name="policyName" noStyle rules={[{ required: true, message: t('common.pleaseInput') }]}>
                    <Input placeholder={t('provision.policyNamePlaceholder')} maxLength={50} />
                  </Form.Item>
                </Descriptions.Item>
                <Descriptions.Item label={t('provision.productTechnology')}>
                  <Form.Item name="productTechnology" noStyle rules={[{ required: true, message: t('common.pleaseSelect') }]}>
                    <Select
                      placeholder={t('provision.productTechnologyPlaceholder')}
                      options={productTechnologyOptions}
                      onChange={handleProductTechnologyChange}
                      style={{ width: '100%', maxWidth: 420 }}
                    />
                  </Form.Item>
                </Descriptions.Item>
                <Descriptions.Item label={t('provision.productName')}>
                  <Form.Item
                    name="productClasses"
                    noStyle
                    getValueProps={(value: string[] | undefined) => ({ value: value?.[0] })}
                    normalize={(value: string | undefined) => value ? [value] : []}
                    rules={[{ required: true, message: t('common.pleaseSelect') }]}
                  >
                    <ProductClassSelect
                      placeholder={productTechnology
                        ? t('common.pleaseSelect')
                        : t('provision.selectProductTechnologyFirst')}
                      options={productClassOptions}
                      loading={productCatalogLoading}
                      onChange={handleProductClassChange}
                      disabled={!productTechnology}
                    />
                  </Form.Item>
                </Descriptions.Item>
                <Descriptions.Item label={t('provision.executeType')}>
                  <Form.Item name="executeType" noStyle rules={[{ required: true }]}>
                    <Radio.Group>
                      <Radio value="0">{t('provision.autoExecute')}</Radio>
                      <Radio value="1">{t('provision.manualExecute')}</Radio>
                    </Radio.Group>
                  </Form.Item>
                </Descriptions.Item>
              </Descriptions>
            </Card>
          </PolicyReadOnlySection>

          {/* Function Module Selection */}
          <Card size="small" style={{ marginBottom: 16 }}>
            <Text type="secondary" style={{ marginBottom: 12, display: 'block' }}>
              {t('provision.selectModuleHint')}
            </Text>
            <Radio.Group value={functionModule} onChange={(e) => setFunctionModule(e.target.value)} style={{ width: '100%' }}>
              <Space size={16}>
                <Radio.Button value="0" style={{ width: 240, height: 50, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                  <Space>
                    <CheckCircleOutlined />
                    <span>{t('provision.softwareUpgrade')}</span>
                  </Space>
                </Radio.Button>
                <Radio.Button value="1" style={{ width: 240, height: 50, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                  <Space>
                    <CheckCircleOutlined />
                    <span>License</span>
                  </Space>
                </Radio.Button>
                <Radio.Button value="2" style={{ width: 240, height: 50, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                  <Space>
                    <CheckCircleOutlined />
                    <span>{t('provision.selfConfig')}</span>
                  </Space>
                </Radio.Button>
              </Space>
            </Radio.Group>
          </Card>

          {/* Module Config Panels */}
          {functionModule === '0' && (
            <PolicyReadOnlySection readOnly={isView}>
              {renderSoftwareUpgradeConfig()}
            </PolicyReadOnlySection>
          )}
          {functionModule === '1' && renderLicenseConfig()}
          {functionModule === '2' && renderSelfConfig()}

        </Form>
      </div>

      {/* Config Detail/Edit Drawer */}
      <Drawer
        title={configDetailMode === 'view' ? t('provision.viewConfig') : t('provision.editConfig')}
        open={configDetailVisible}
        onClose={() => {
          setConfigDetailVisible(false);
          setCurrentConfig(null);
          configForm.resetFields();
        }}
        size={720}
        destroyOnHidden
      >
        <Form form={configForm} layout="vertical" disabled={configDetailMode === 'view'} style={{ paddingBottom: 60 }}>
          <Alert
            type="info"
            showIcon
            title={t('provision.deviceParameterOverridesHint')}
            style={{ marginBottom: 16 }}
          />
          <ParameterConfigFields
            deviceType={currentConfig?.deviceType as ParamConfigDeviceType | undefined}
            paramModelName={selectedProduct?.paramModelName}
            scope="device"
            onRequestEdit={() => setConfigDetailMode('edit')}
          />
        </Form>


        {/* Drawer Footer */}
        <div style={{ position: 'absolute', bottom: 0, right: 0, width: '100%', borderTop: '1px solid #f0f0f0', background: '#fff', padding: '10px 16px', textAlign: 'right' }}>
          <Space>
            <Button onClick={() => {
              setConfigDetailVisible(false);
              setCurrentConfig(null);
              configForm.resetFields();
            }}>
              {t('common.cancel')}
            </Button>
            {configDetailMode === 'edit' && (
              <Button type="primary" onClick={handleConfigFormSubmit}>
                {t('common.save')}
              </Button>
            )}
          </Space>
        </div>
      </Drawer>

      {/* Firmware package import belongs to the target version. */}
      <Modal
        title={t('provision.importVersionPackage')}
        open={firmwareImportVisible}
        onCancel={handleCloseFirmwareImport}
        onOk={() => void handleImportFirmware()}
        confirmLoading={uploadFirmwareMutation.isPending}
        okText={t('common.import')}
        cancelText={t('common.cancel')}
        width={560}
        destroyOnHidden
      >
        <Form form={firmwareImportForm} layout="vertical">
          <Form.Item label={t('provision.productName')}>
            <Input value={productClasses.join(', ')} disabled />
          </Form.Item>
          <Form.Item
            name="version"
            label={t('software.firmware.version')}
            rules={[
              { required: true, message: t('software.firmware.inputVersion') },
              { max: 45, message: t('software.firmware.versionMaxLen') },
            ]}
          >
            <Input placeholder={t('software.firmware.inputVersion')} maxLength={45} />
          </Form.Item>
          <Form.Item
            label={t('software.firmware.fileName')}
            required
          >
            <Upload.Dragger
              accept=".img,.ext"
              fileList={firmwareFileList}
              maxCount={1}
              beforeUpload={(file) => {
                setFirmwareFileList([{
                  uid: file.uid,
                  name: file.name,
                  status: 'done',
                  originFileObj: file,
                }]);
                return false;
              }}
              onRemove={() => setFirmwareFileList([])}
            >
              <p className="ant-upload-drag-icon">
                <InboxOutlined style={{ fontSize: 40, color: 'var(--color-primary-600)' }} />
              </p>
              <p className="ant-upload-text">{t('software.firmware.clickOrDrag')}</p>
              <p className="ant-upload-hint">
                {t('software.firmware.supportFormat', { format: 'IMG / EXT' })}
              </p>
            </Upload.Dragger>
          </Form.Item>
        </Form>
      </Modal>

      <LicenseImportDrawer
        open={licenseImportModalVisible}
        onClose={() => setLicenseImportModalVisible(false)}
        onSuccess={() => { void refetchLicenses(); }}
      />

      {/* Import Config Modal */}
      <Modal
        title={t('provision.importParamConfig')}
        open={importModalVisible}
        onCancel={() => {
          setImportModalVisible(false);
          setParamFileList([]);
          setImportPreview([]);
          setPendingImportedConfigs([]);
          setImportPreviewError('');
        }}
        footer={[
          <Button key="cancel" onClick={() => {
            setImportModalVisible(false);
            setParamFileList([]);
            setImportPreview([]);
            setPendingImportedConfigs([]);
            setImportPreviewError('');
          }}>
            {t('common.cancel')}
          </Button>,
          <Button
            key="import"
            type="primary"
            disabled={pendingImportedConfigs.length === 0
              || importPreview.some((item) => item.action === 'duplicate')}
            onClick={applyImportConfig}
          >
            {t('common.import')}
          </Button>,
        ]}
        width={520}
      >
        {/* Import Type Selection */}
        <div style={{
          padding: '16px',
          background: 'var(--color-fill-quaternary)',
          borderRadius: 8,
          marginBottom: 16
        }}>
          <div style={{ marginBottom: 4, fontWeight: 500, color: 'var(--color-text)' }}>
            {t('provision.importType')}
          </div>
          <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 12 }}>
            {t('provision.importTypeDesc')}
          </Text>
          <Radio.Group
            value={paramImportType}
            onChange={(e) => setParamImportType(e.target.value)}
            style={{ marginBottom: 8 }}
          >
            <Radio value="append">{t('provision.importTypeAppend')}</Radio>
            <Radio value="replace">{t('provision.importTypeReplace')}</Radio>
          </Radio.Group>
          <Text type="secondary" style={{ fontSize: 12, display: 'block' }}>
            {paramImportType === 'append' ? t('provision.importTypeAppendHint') : t('provision.importTypeReplaceHint')}
          </Text>
        </div>

        {activeParamDeviceType === 'gNB' && (
          <Alert
            type="info"
            showIcon
            title={t('provision.paramConfig5GTemplateFillHint')}
            style={{ marginBottom: 16 }}
          />
        )}

        {/* Upload Area */}
        <Upload.Dragger
          accept=".xlsx,.xls,.csv"
          fileList={paramFileList}
          beforeUpload={(file) => {
            setParamFileList([file]);
            void previewImportConfig(file);
            return false;
          }}
          onRemove={() => {
            setParamFileList([]);
            setImportPreview([]);
            setPendingImportedConfigs([]);
            setImportPreviewError('');
          }}
        >
          <p className="ant-upload-drag-icon">
            <InboxOutlined style={{ fontSize: 40, color: 'var(--color-primary-600)' }} />
          </p>
          <p className="ant-upload-text">{t('recycle.importSelectFile')}</p>
          <p className="ant-upload-hint" style={{ fontSize: 12, color: '#8c8c8c' }}>
            {t('provision.importParamConfigHint')}
          </p>
        </Upload.Dragger>
        {importPreviewError && (
          <Alert type="error" showIcon title={importPreviewError} style={{ marginTop: 16 }} />
        )}
        {importPreview.length > 0 && (
          <div style={{ marginTop: 16 }}>
            <Alert
              type={importPreview.some((item) => item.action === 'duplicate') ? 'warning' : 'success'}
              showIcon
              title={t('provision.paramConfigImportPreviewSummary', {
                add: importPreview.filter((item) => item.action === 'add').length,
                update: importPreview.filter((item) => item.action === 'update').length,
                conflict: importPreview.filter((item) => item.action === 'duplicate').length,
              })}
              style={{ marginBottom: 12 }}
            />
            <Table
              size="small"
              rowKey={(record) => `${record.serialNumber}-${record.action}`}
              pagination={{ pageSize: 5, hideOnSinglePage: true }}
              dataSource={importPreview}
              columns={[
                { title: t('provision.serialNumber'), dataIndex: 'serialNumber' },
                {
                  title: t('provision.importAction'),
                  dataIndex: 'action',
                  width: 100,
                  render: (action: ParamConfigImportPreviewItem['action']) => (
                    <Tag color={action === 'add' ? 'success' : action === 'update' ? 'processing' : 'error'}>
                      {t(`provision.importAction.${action}`)}
                    </Tag>
                  ),
                },
              ]}
            />
          </div>
        )}
      </Modal>

    </div>
  );
}
