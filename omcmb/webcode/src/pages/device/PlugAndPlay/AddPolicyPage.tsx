import { useState, useMemo, useCallback, useEffect } from 'react';
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
  InputNumber,
  message,
  Modal,
  Drawer,
  Collapse,
  Descriptions,
  Alert,
  Dropdown,
  type UploadFile,
  type MenuProps,
} from 'antd';
import {
  ArrowLeftOutlined,
  UploadOutlined,
  PlusOutlined,
  DeleteOutlined,
  SearchOutlined,
  CheckCircleOutlined,
  DownloadOutlined,
  EyeOutlined,
  EditOutlined,
  InboxOutlined,
  MoreOutlined,
} from '@ant-design/icons';
import * as XLSX from 'xlsx';
import { useT } from '@/hooks/useT';
import {
  usePlugAndPlayPolicy,
  useSavePlugAndPlayPolicy,
} from '@core/hooks/api/useProvisioning';
import { useDeviceList, useProductClasses } from '@core/hooks/api/useDevices';
import { useProductList, useProductMatch } from '@core/hooks/api/useProducts';
import { useSoftwareVersions, useUploadFirmware } from '@core/hooks/api/useSoftware';
import {
  useDeleteDeviceLicense,
  useDeviceLicenses,
} from '@core/hooks/api/useDeviceLicense';
import type { DeviceLicense } from '@core/services/api/deviceLicenseApi';
import LicenseImportDrawer from '@/pages/backup/DeviceLicenseLibrary/ImportDrawer';
import {
  normalizeProductTechnology,
  resolveProductClassTechnology,
  toSupportedProductClassOptions,
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
  type ParamConfigDeviceType,
} from './paramConfigWorkbook';
import { getParamConfigTemplate, toParamConfigDeviceType } from './paramConfigTemplate';
import {
  mergeParamConfigFormValues,
  toParamConfigFormValues,
  withTemplateSheetParameters,
} from './paramConfigDetail';
import {
  GSM_GROUPED_TEMPLATE_FIELDS,
  type TemplateFieldRef,
} from './paramConfigGroupedFields';
import { isParamConfigToolbarEnabled } from './paramConfigToolbarAvailability';
import { getPolicyModuleActionAvailability } from './policyActionAvailability';
import ProductClassMultiSelect from './components/ProductClassMultiSelect';
import PolicyReadOnlySection from './components/PolicyReadOnlySection';
import GnbQuickSettingsCards, { GnbTemplateExtraFieldGrid } from './GnbQuickSettingsCards';
import EnbQuickSettingsCards, { EnbTemplateExtraFieldGrid } from './EnbQuickSettingsCards';

const { Text, Title } = Typography;

function TemplateFieldGrid({ fields }: { fields: readonly TemplateFieldRef[] }) {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', columnGap: 16 }}>
      {fields.map(({ sheet, header }) => (
        <Form.Item
          key={`${sheet}.${header}`}
          name={['sheetParameters', sheet, 0, header]}
          label={header.replace(/^\*/, '')}
          getValueProps={(inputValue) => ({
            value: inputValue === undefined || inputValue === null ? '' : String(inputValue),
          })}
        >
          <Input style={{ width: '100%' }} />
        </Form.Item>
      ))}
    </div>
  );
}

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
  mgmtIp?: string;
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
  duplexMode?: 'TDD' | 'FDD';  // 基站制式
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
  omIp?: string;
  omMask?: string;
  // ========== gNB IPSec配置 ==========
  ipsecImsi?: string;
  ipsecKey?: string;
  ipsecOpc?: string;
  // ========== gNB DNS配置 ==========
  dns1?: string;
  dns2?: string;
  // ========== gNB WAN配置 ==========
  addressType?: 'IPv4' | 'IPv6';
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
    omIp: '192.168.200.50',
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
    duplexMode: 'TDD',
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
    omIp: '192.168.200.50',
    omMask: '255.255.255.0',
    serviceGateway: '192.168.100.1',
    serviceGatewayMask: '255.255.255.0',
    mgmtGateway: '192.168.200.1',
    mgmtGatewayMask: '255.255.255.0',
    serviceVlan: 100,
    mgmtVlan: 200,
    // WAN Config
    addressType: 'IPv4',
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

// Subframe assignment options
const SUBFRAME_OPTIONS = [
  { label: '0 (DL:UL = 1:3)', value: '0' },
  { label: '1 (DL:UL = 2:2)', value: '1' },
  { label: '2 (DL:UL = 3:1)', value: '2' },
  { label: '6 (DL:UL = 3:5)', value: '6' },
];

export default function AddPolicyPage() {
  const t = useT();
  const { data: supportedProductClasses, isLoading: productClassesLoading } = useProductClasses();
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
    () => toSupportedProductClassOptions(
      supportedProductClasses,
      productCatalog?.items,
      productTechnology,
    ),
    [productCatalog?.items, productTechnology, supportedProductClasses],
  );
  const [firmwareImportForm] = Form.useForm();
  const [loading, setLoading] = useState(false);

  // Get mode from URL path
  const pathParts = location.pathname.split('/');
  const lastPart = pathParts[pathParts.length - 2];
  const isEdit = lastPart === 'edit';
  const isView = lastPart === 'view';
  const moduleActions = getPolicyModuleActionAvailability(
    isView ? 'view' : isEdit ? 'edit' : 'create',
  );
  const { data: persistedPolicy } = usePlugAndPlayPolicy(isEdit || isView ? id : '');
  const savePolicyMutation = useSavePlugAndPlayPolicy(isEdit ? id : undefined);

  // State for software upgrade
  const [firmwareImportVisible, setFirmwareImportVisible] = useState(false);
  const [firmwareFileList, setFirmwareFileList] = useState<UploadFile[]>([]);

  // State for license
  const [licenseSearchText, setLicenseSearchText] = useState('');
  const [licenseImportModalVisible, setLicenseImportModalVisible] = useState(false);

  // State for self config (new design - config list)
  const [paramConfigList, setParamConfigList] = useState<ParamConfig[]>([]);
  const [configSearchText, setConfigSearchText] = useState('');
  const [configDetailVisible, setConfigDetailVisible] = useState(false);
  const [configDetailMode, setConfigDetailMode] = useState<'view' | 'edit'>('view');
  const [currentConfig, setCurrentConfig] = useState<ParamConfig | null>(null);
  const [configForm] = Form.useForm();
  const [importModalVisible, setImportModalVisible] = useState(false);
  const [paramImportType, setParamImportType] = useState<'append' | 'replace'>('append');
  const [paramFileList, setParamFileList] = useState<UploadFile[]>([]);

  // Current function module
  const [functionModule, setFunctionModule] = useState<'0' | '1' | '2'>('0');
  const [productClasses, setProductClasses] = useState<string[]>([]);
  const productClass = productClasses[0] ?? '';
  const { data: productDevicesData, isLoading: actualVersionsLoading } = useDeviceList(
    { page: 1, pageSize: 1000, productClass },
    { enabled: Boolean(productClass) },
  );
  const {
    data: productMatchData,
  } = useProductMatch(productClass);
  const activeParamDeviceType = useMemo<ParamConfigDeviceType | undefined>(
    () => toParamConfigDeviceType(productMatchData?.product?.tech),
    [productMatchData],
  );
  const matchedProductId = productMatchData?.product?.id;
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
    { enabled: Boolean(productClass) },
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
        ?? resolveProductClassTechnology(savedProductClasses[0] ?? '', productCatalog?.items),
      productClasses: savedProductClasses,
      executeType: persistedPolicy.executeType === 'auto' ? '0' : '1',
      selfStartEnable: persistedPolicy.enabled,
      upgradeEnable: persistedPolicy.upgradeEnabled,
      targetVersion: persistedPolicy.targetVersion,
      licenseEnable: persistedPolicy.licenseEnabled,
      selfConfigEnable: persistedPolicy.selfConfigEnabled,
    });
    setProductClasses(savedProductClasses);
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
  const handleProductClassChange = useCallback((values: string[]) => {
    setProductClasses(values);
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

  // Param config table columns (simplified - only basic fields)
  const selfConfigEnabled = Form.useWatch('selfConfigEnable', form);
  const paramConfigToolbarEnabled = isParamConfigToolbarEnabled(selfConfigEnabled);

  // Simplified columns for all device types
  const paramConfigColumns = [
    {
      title: t('table.operation'),
      key: 'action',
      width: 100,
      fixed: 'right' as const,
      render: (_: unknown, record: ParamConfig) => {
        const items: MenuProps['items'] = [
          { key: 'edit', label: t('common.edit'), icon: <EditOutlined />,
            onClick: () => {
              const hydrated = withTemplateSheetParameters(record);
              setCurrentConfig(hydrated);
              setConfigDetailMode('edit');
              configForm.setFieldsValue(toParamConfigFormValues(hydrated));
              setConfigDetailVisible(true);
            },
          },
          { key: 'delete', label: t('common.delete'), icon: <DeleteOutlined />, danger: true,
            onClick: () => { setParamConfigList(prev => prev.filter(item => item.id !== record.id)); message.success(t('common.success')); },
          },
        ];
        return (
          <Space size={4}>
            <Button type="link" size="small" icon={<EyeOutlined />}
              onClick={() => {
                const hydrated = withTemplateSheetParameters(record);
                setCurrentConfig(hydrated);
                setConfigDetailMode('view');
                configForm.setFieldsValue(toParamConfigFormValues(hydrated));
                setConfigDetailVisible(true);
              }}>
              {t('common.view')}
            </Button>
            {moduleActions.edit && moduleActions.delete && (
              <Dropdown menu={{ items }} trigger={['click']}>
                <Button type="text" size="small" icon={<MoreOutlined />} />
              </Dropdown>
            )}
          </Space>
        );
      },
    },
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
    {
      title: t('provision.bandsSupport'),
      dataIndex: 'bandsSupport',
      key: 'bandsSupport',
      width: 100,
    },
    {
      title: t('provision.bandwidth'),
      dataIndex: 'bandWidth',
      key: 'bandWidth',
      width: 100,
    },
    {
      title: t('provision.frequency'),
      dataIndex: 'frequency',
      key: 'frequency',
      width: 100,
    },
    {
      title: t('provision.subframeAssignment'),
      dataIndex: 'subframeAssignment',
      key: 'subframeAssignment',
      width: 140,
      render: (val: number) => {
        const option = SUBFRAME_OPTIONS.find(o => o.value === String(val));
        return option?.label || val;
      },
    },
    {
      title: t('provision.updatedBy'),
      dataIndex: 'updatedBy',
      key: 'updatedBy',
      width: 100,
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

    return list;
  }, [activeParamDeviceType, paramConfigList, productClass, configSearchText]);

  const handleExportConfig = useCallback(() => {
    if (!productClass) {
      void message.warning(t('provision.selectProductClassFirst'));
      return;
    }
    if (filteredParamConfigList.length === 0) {
      void message.warning(t('provision.noParamConfigToExport'));
      return;
    }

    const workbook = createParamConfigWorkbook(filteredParamConfigList);
    const safeProductClass = (productClass || 'parameter-config').replace(/[\\/:*?"<>|]+/g, '_');
    XLSX.writeFile(workbook, `${safeProductClass}-参数配置.xlsx`);
    void message.success(t('provision.paramConfigExportSuccess', {
      count: filteredParamConfigList.length,
    }));
  }, [filteredParamConfigList, productClass, t]);

  const handleDownloadParamConfigTemplate = useCallback(() => {
    if (!productClass) {
      void message.warning(t('provision.selectProductClassFirst'));
      return;
    }
    const template = getParamConfigTemplate(activeParamDeviceType);
    if (!template) {
      void message.warning(t('provision.paramConfigTemplateUnavailable'));
      return;
    }

    if (!activeParamDeviceType) return;
    XLSX.writeFile(createParamConfigTemplateWorkbook(activeParamDeviceType), template.fileName);
    void message.success(t('provision.paramConfigTemplateDownloaded'));
  }, [activeParamDeviceType, productClass, t]);

  const handleOpenParamConfigImport = useCallback(() => {
    if (!productClass) {
      void message.warning(t('provision.selectProductClassFirst'));
      return;
    }
    setImportModalVisible(true);
  }, [productClass, t]);

  // Handle config form submit
  const handleConfigFormSubmit = useCallback(() => {
    configForm.validateFields().then(values => {
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

  const handleImportConfig = useCallback(async (file: File) => {
    if (!activeParamDeviceType) {
      void message.warning(t('provision.paramConfigDeviceTypeUnavailable'));
      return;
    }

    try {
      const importedAt = new Date().toLocaleString();
      const rows = parseParamConfigWorkbook(
        await file.arrayBuffer(),
        activeParamDeviceType,
        importedAt,
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
        updatedBy: row.updatedBy ?? 'import',
        updatedAt: row.updatedAt ?? importedAt,
      }));

      setParamConfigList((previous) =>
        mergeImportedParamConfigs(previous, importedConfigs, paramImportType),
      );
      setImportModalVisible(false);
      setParamFileList([]);
      void message.success(t('provision.paramConfigImportSuccess', {
        count: importedConfigs.length,
      }));
    } catch (error) {
      if (error instanceof ParamConfigWorkbookError) {
        if (error.code === 'empty_workbook') {
          void message.error(t('provision.paramConfigImportEmpty'));
        } else if (error.code === 'template_no_data') {
          void message.error(t('provision.paramConfigTemplateNoData'));
        } else if (error.code === 'missing_columns') {
          void message.error(t('provision.paramConfigImportMissingColumns'));
        } else {
          void message.error(t('provision.paramConfigImportInvalidRow', {
            row: error.row ?? '-',
            field: error.field ?? '-',
          }));
        }
        return;
      }
      void message.error(t('provision.paramConfigImportFailed'));
    }
  }, [activeParamDeviceType, paramImportType, t]);

  // Handle submit
  const handleSubmit = useCallback(async () => {
    try {
      await form.validateFields();
      // Module panels are conditionally mounted. validateFields() only returns
      // values from the currently mounted panel, while getFieldsValue(true)
      // also includes the preserved values of the other selected modules.
      const values = form.getFieldsValue(true);
      const selectedProductClasses = (values.productClasses ?? []) as string[];
      setLoading(true);

      await savePolicyMutation.mutateAsync({
        name: values.policyName,
        enabled: Boolean(values.selfStartEnable),
        productClass: selectedProductClasses[0] ?? '',
        productClasses: selectedProductClasses,
        executeType: values.executeType === '0' ? 'auto' : 'manual',
        priority: Number(values.priority || 100),
        upgradeEnabled: Boolean(values.upgradeEnable),
        targetVersion: values.targetVersion || '',
        licenseEnabled: Boolean(values.licenseEnable),
        selfConfigEnabled: Boolean(values.selfConfigEnable),
        config: {
          ...values,
          productClass: selectedProductClasses[0] ?? '',
          productClasses: selectedProductClasses,
          paramConfigList,
        },
      });
      message.success(t('common.success'));
      navigate('/device/plug-and-play');
    } catch (error) {
      console.error('Validation error:', error);
    } finally {
      setLoading(false);
    }
  }, [form, navigate, t, savePolicyMutation, paramConfigList]);

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
      <div style={{ padding: '16px 0' }}>
        {/* Toolbar */}
        <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Input
            placeholder={t('provision.searchBySerialNumber')}
            prefix={<SearchOutlined />}
            value={configSearchText}
            onChange={(e) => setConfigSearchText(e.target.value)}
            style={{ width: 240 }}
            allowClear
          />
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

        {/* Config List Table */}
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
            <Button type="primary" loading={loading} onClick={handleSubmit}>
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
                <Descriptions.Item label={t('provision.productClass')}>
                  <Form.Item name="productClasses" noStyle rules={[{ required: true, message: t('common.pleaseSelect') }]}>
                    <ProductClassMultiSelect
                      placeholder={productTechnology
                        ? t('common.pleaseSelect')
                        : t('provision.selectProductTechnologyFirst')}
                      options={productClassOptions}
                      loading={productClassesLoading || productCatalogLoading}
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
          {/* eNB fields aligned with device quick settings; template-only fields follow. */}
          {currentConfig?.deviceType === 'eNB' && (
            <>
              <EnbQuickSettingsCards />
              <Card size="small" title={t('provision.otherTemplateParams')} style={{ marginBottom: 16 }}>
                <EnbTemplateExtraFieldGrid />
              </Card>
              <Card size="small" title={t('provision.customParams')} style={{ marginBottom: 16 }}>
                <Form.List name="customParams">
                  {(fields, { add, remove }) => (
                    <>
                      {fields.map(({ key, name, ...restField }) => (
                        <Card key={key} size="small" style={{ marginBottom: 12 }} title={`${t('provision.customParam')} ${name + 1}`} extra={
                          <Button type="link" danger icon={<DeleteOutlined />} onClick={() => remove(name)} />
                        }>
                          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', columnGap: 16 }}>
                            <Form.Item {...restField} name={[name, 'name']} label={t('provision.nrQuick.name')}><Input /></Form.Item>
                            <Form.Item {...restField} name={[name, 'value']} label={t('provision.nrQuick.value')}><Input /></Form.Item>
                            <Form.Item {...restField} name={[name, 'trPath']} label={t('provision.nrQuick.trPath')}><Input /></Form.Item>
                          </div>
                        </Card>
                      ))}
                      <Button type="dashed" onClick={() => add()} block icon={<PlusOutlined />}>
                        {t('provision.addCustomParam')}
                      </Button>
                    </>
                  )}
                </Form.List>
              </Card>
            </>
          )}
          {/* gNB specific fields */}
          {currentConfig?.deviceType === 'gNB' && (
            <div style={{ display: 'flex', flexDirection: 'column' }}>
              <GnbQuickSettingsCards />
              <Card key="gnb-ip" size="small" title={t('provision.ipConfig')} style={{ marginBottom: 16, order: 10 }}>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', columnGap: 16 }}>
                  <Form.Item name="serviceIp" label={t('provision.serviceIp')} style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="serviceMask" label={t('provision.serviceMask')} style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="omIp" label={t('provision.omIp')} style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="serviceGateway" label={t('provision.serviceGateway')} style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="serviceVlan" label={t('provision.serviceVlan')} style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} min={0} max={4095} />
                  </Form.Item>
                </div>
              </Card>
              <Card key="gnb-plmn-extra" size="small" title={t('provision.plmnConfigList')} style={{ marginBottom: 16, order: 9 }}>
                <Form.List name="plmnConfigList">
                  {(fields, { add, remove }) => (
                    <>
                      {fields.map(({ key, name, ...restField }) => (
                        <Card key={key} size="small" style={{ marginBottom: 12 }} title={`${t('provision.plmnConfig')} ${name + 1}`} extra={
                          <Button type="link" danger icon={<DeleteOutlined />} onClick={() => remove(name)} />
                        }>
                          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', columnGap: 16 }}>
                            <Form.Item {...restField} name={[name, 'plmnId']} label={t('provision.nrQuick.plmn')}>
                              <Input style={{ width: '100%' }} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'primary']} label={t('provision.primary')}>
                              <Select options={[
                                { label: t('common.yes'), value: '1' },
                                { label: t('common.no'), value: '0' },
                              ]} />
                            </Form.Item>
                          </div>
                        </Card>
                      ))}
                      <Button type="dashed" onClick={() => add()} block icon={<PlusOutlined />}>
                        {t('provision.addPlmnConfig')}
                      </Button>
                    </>
                  )}
                </Form.List>
              </Card>
              <Card key="gnb-slice" size="small" title={t('provision.sliceConfigList')} style={{ marginBottom: 16, order: 11 }}>
                <Form.List name="sliceConfigList">
                  {(fields, { add, remove }) => (
                    <>
                      {fields.map(({ key, name, ...restField }) => (
                        <Card key={key} size="small" style={{ marginBottom: 12 }} title={`${t('provision.sliceConfig')} ${name + 1}`} extra={
                          <Button type="link" danger icon={<DeleteOutlined />} onClick={() => remove(name)} />
                        }>
                          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                            <Form.Item {...restField} name={[name, 'sd']} label={t('provision.nrQuick.sd')} style={{ flex: '1 1 200px' }}>
                              <Select options={[
                                { label: t('provision.sdEmpty'), value: '0' },
                                { label: t('provision.sdNotEmpty'), value: '1' },
                              ]} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'sdValue']} label={t('provision.nrQuick.sdValue')} style={{ flex: '1 1 200px' }}>
                              <Input style={{ width: '100%' }} placeholder="e.g. 010203" />
                            </Form.Item>
                          </div>
                        </Card>
                      ))}
                      <Button type="dashed" onClick={() => add()} block icon={<PlusOutlined />}>
                        {t('provision.addSliceConfig')}
                      </Button>
                    </>
                  )}
                </Form.List>
              </Card>
              <Card key="gnb-other" size="small" title={t('provision.otherTemplateParams')} style={{ marginBottom: 16, order: 8 }}>
                <GnbTemplateExtraFieldGrid />
              </Card>
              <Card key="gnb-custom" size="small" title={t('provision.customParams')} style={{ marginBottom: 16, order: 12 }}>
                <Form.List name="customParams">
                  {(fields, { add, remove }) => (
                    <>
                      {fields.map(({ key, name, ...restField }) => (
                        <Card key={key} size="small" style={{ marginBottom: 12 }} title={`${t('provision.customParam')} ${name + 1}`} extra={
                          <Button type="link" danger icon={<DeleteOutlined />} onClick={() => remove(name)} />
                        }>
                          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                            <Form.Item {...restField} name={[name, 'name']} label={t('provision.nrQuick.name')} style={{ flex: '1 1 200px' }}>
                              <Input style={{ width: '100%' }} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'value']} label={t('provision.nrQuick.value')} style={{ flex: '1 1 200px' }}>
                              <Input style={{ width: '100%' }} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'trPath']} label={t('provision.nrQuick.trPath')} style={{ flex: '1 1 300px' }}>
                              <Input style={{ width: '100%' }} />
                            </Form.Item>
                          </div>
                        </Card>
                      ))}
                      <Button type="dashed" onClick={() => add()} block icon={<PlusOutlined />}>
                        {t('provision.addCustomParam')}
                      </Button>
                    </>
                  )}
                </Form.List>
              </Card>
            </div>
          )}

          {/* GSM specific fields */}
          {currentConfig?.deviceType === 'GSM' && (
            <Collapse defaultActiveKey={['gsm-basic', 'gsm-other', 'gsm-custom']} ghost>
              <Collapse.Panel key="gsm-basic" header={t('provision.gsmBasicConfig')}>
                <TemplateFieldGrid fields={GSM_GROUPED_TEMPLATE_FIELDS.quickAbis} />
              </Collapse.Panel>
              <Collapse.Panel key="gsm-other" header="其他参数">
                <TemplateFieldGrid fields={GSM_GROUPED_TEMPLATE_FIELDS.other} />
              </Collapse.Panel>
              <Collapse.Panel key="gsm-custom" header={t('provision.customParams')}>
                <Form.List name="customParams">
                  {(fields, { add, remove }) => (
                    <>
                      {fields.map(({ key, name, ...restField }) => (
                        <Card key={key} size="small" style={{ marginBottom: 12 }} title={`${t('provision.customParam')} ${name + 1}`} extra={
                          <Button type="link" danger icon={<DeleteOutlined />} onClick={() => remove(name)} />
                        }>
                          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                            <Form.Item {...restField} name={[name, 'name']} label="Name" style={{ flex: '1 1 200px' }}>
                              <Input style={{ width: '100%' }} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'value']} label="Value" style={{ flex: '1 1 200px' }}>
                              <Input style={{ width: '100%' }} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'trPath']} label="TR Path" style={{ flex: '1 1 300px' }}>
                              <Input style={{ width: '100%' }} />
                            </Form.Item>
                          </div>
                        </Card>
                      ))}
                      <Button type="dashed" onClick={() => add()} block icon={<PlusOutlined />}>
                        {t('provision.addCustomParam')}
                      </Button>
                    </>
                  )}
                </Form.List>
              </Collapse.Panel>
            </Collapse>
          )}
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
          <Form.Item label={t('provision.productClass')}>
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
        }}
        footer={[
          <Button key="cancel" onClick={() => {
            setImportModalVisible(false);
            setParamFileList([]);
          }}>
            {t('common.cancel')}
          </Button>,
          <Button
            key="import"
            type="primary"
            onClick={() => {
              if (paramFileList.length === 0) {
                void message.warning(t('common.pleaseSelect'));
                return;
              }
              const selectedFile = paramFileList[0];
              const realFile = (selectedFile.originFileObj ?? selectedFile) as File;
              void handleImportConfig(realFile);
            }}
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
            message={t('provision.paramConfig5GTemplateFillHint')}
            style={{ marginBottom: 16 }}
          />
        )}

        {/* Upload Area */}
        <Upload.Dragger
          accept=".xlsx,.xls,.csv"
          fileList={paramFileList}
          beforeUpload={(file) => {
            setParamFileList([file]);
            return false;
          }}
          onRemove={() => {
            setParamFileList([]);
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
      </Modal>
    </div>
  );
}
