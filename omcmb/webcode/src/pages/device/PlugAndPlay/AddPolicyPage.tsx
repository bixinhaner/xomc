import { useState, useMemo, useCallback } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
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
  Tag,
  Upload,
  InputNumber,
  message,
  Modal,
  Drawer,
  Collapse,
  Descriptions,
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
import { useT } from '@/hooks/useT';

const { Text, Title } = Typography;

// Types
type ExecuteType = '0' | '1';
type EnableType = '0' | '1';

// T-0136: 保留为 export 占位，避免 TS6196 同时不破坏未来可能复用
// eslint-disable-next-line @typescript-eslint/no-unused-vars
export interface _PolicyForm {
  selfStartEnable: EnableType;
  policyName: string;
  productClass: string;
  executeType: ExecuteType;
  functionModule: '0' | '1' | '2'; // 0-software upgrade, 1-license, 2-self config
  // Software Upgrade
  upgradeEnable: EnableType;
  specifyVersionType: EnableType;
  originalVersion: string;
  targetVersion: string;
  preserveSetting: EnableType;
  // License
  licenseEnable: EnableType;
  // Self Config
  selfConfigEnable: EnableType;
  switchEnable: EnableType;
}

interface OriginalVersion {
  originalVersion: string;
}

interface LicenseFile {
  serial_number: string;
  file_name: string;
  upload_time: string;
  execute_status: '0' | '1' | '2' | '3';
}

// Device type for param config
type DeviceType = 'eNB' | 'gNB' | 'GSM';

// IPSECS Tunnel for eNB
interface IpsecTunnel {
  key: string;
  TUNNEL_ENABLE: '0' | '1';
  authBy: string;
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
}

// WAN Binding item for eNB
interface WanBindingItem {
  ipAddress: string;
  netmask: string;
  gateway: string;
  vlanId: string;
  binding: string;
}

// eNB WAN固定分组类型 (T-0136: 保留为 export 占位避免 TS6196)
// eslint-disable-next-line @typescript-eslint/no-unused-vars
export type _EnbWanGroup = 'wanOamTr069' | 'wanS1c' | 'wanS1u' | 'wanX2ap';

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
  mmePort: number;
}

// Param Config item for new design
interface ParamConfig {
  id: string;
  deviceType: DeviceType;
  serialNumber: string;
  cellName: string;
  updatedBy: string;
  updatedAt: string;
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
  ipsecSwitch?: '0' | '1';
  ipsecEnable?: '0' | '1';
  ipsecRightIkePort?: string;
  leftInterface?: string;
  ipsecList?: IpsecTunnel[];
  // ========== eNB WAN配置 ==========
  wanSendEnable?: '0' | '1';
  wanOamTr069?: WanBindingItem;
  wanS1c?: WanBindingItem;
  wanS1u?: WanBindingItem;
  wanX2ap?: WanBindingItem;
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

// Product types - 设备类型
const PRODUCT_TYPES = [
  { label: 'eNB', value: 'eNB' },
  { label: 'gNB', value: 'gNB' },
  { label: 'GSM', value: 'GSM' },
];

// Target versions (mock)
const TARGET_VERSIONS = [
  { label: 'V2.2.0', value: 'V2.2.0' },
  { label: 'V2.1.0', value: 'V2.1.0' },
  { label: 'V2.0.5', value: 'V2.0.5' },
  { label: 'V2.0.0', value: 'V2.0.0' },
  { label: 'V1.5.0', value: 'V1.5.0' },
];

// Original versions (mock for selection)
const AVAILABLE_ORIGINAL_VERSIONS = [
  { originalVersion: 'V1.0.0' },
  { originalVersion: 'V1.1.0' },
  { originalVersion: 'V1.2.0' },
  { originalVersion: 'V1.5.0' },
  { originalVersion: 'V2.0.0' },
];

// License files (mock)
const MOCK_LICENSE_FILES: LicenseFile[] = [
  {
    serial_number: 'ENB00001',
    file_name: 'license_001.lic',
    upload_time: '2026-04-07 10:00:00',
    execute_status: '0',
  },
  {
    serial_number: 'ENB00002',
    file_name: 'license_002.lic',
    upload_time: '2026-04-07 11:00:00',
    execute_status: '1',
  },
];

// Param Config mock data (for new design) - eNB, gNB, GSM one each
const MOCK_PARAM_CONFIGS: ParamConfig[] = [
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
    ipsecSwitch: '1',
    ipsecEnable: '1',
    ipsecRightIkePort: '500',
    leftInterface: 'eth0',
    ipsecList: [
      {
        key: '1',
        TUNNEL_ENABLE: '1',
        authBy: 'psk',
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
    // WAN Config
    wanSendEnable: '1',
    wanOamTr069: {
      ipAddress: '192.168.10.50',
      netmask: '255.255.255.0',
      gateway: '192.168.10.1',
      vlanId: '100',
      binding: 'TR069-1',
    },
    wanS1c: {
      ipAddress: '192.168.20.50',
      netmask: '255.255.255.0',
      gateway: '192.168.20.1',
      vlanId: '200',
      binding: 'S1-C-1',
    },
    wanS1u: {
      ipAddress: '192.168.30.50',
      netmask: '255.255.255.0',
      gateway: '192.168.30.1',
      vlanId: '300',
      binding: 'S1-U-1',
    },
    wanX2ap: {
      ipAddress: '192.168.40.50',
      netmask: '255.255.255.0',
      gateway: '192.168.40.1',
      vlanId: '400',
      binding: 'X2-AP-1',
    },
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

// Bandwidth options (T-0136: 保留为 export 占位避免 TS6133)
// eslint-disable-next-line @typescript-eslint/no-unused-vars
export const _BANDWIDTH_OPTIONS_DXDF = [
  { label: '6', value: '6' },
  { label: '15', value: '15' },
  { label: '25', value: '25' },
  { label: '50', value: '50' },
  { label: '75', value: '75' },
  { label: '100', value: '100' },
];

const BANDWIDTH_OPTIONS_OTHER = [
  { label: '5MHz', value: 'n25' },
  { label: '10MHz', value: 'n50' },
  { label: '15MHz', value: 'n75' },
  { label: '20MHz', value: 'n100' },
];

// Subframe assignment options
const SUBFRAME_OPTIONS = [
  { label: '0 (DL:UL = 1:3)', value: '0' },
  { label: '1 (DL:UL = 2:2)', value: '1' },
  { label: '2 (DL:UL = 3:1)', value: '2' },
  { label: '6 (DL:UL = 3:5)', value: '6' },
];

// Special subframe options
const SPECIAL_SUBFRAME_OPTIONS = [
  { label: '5', value: '5' },
  { label: '7', value: '7' },
];

// IPSEC options
const AUTH_BY_OPTIONS = [
  { label: 'PSK', value: 'psk' },
  { label: 'Cert', value: 'cert' },
  { label: 'AKA PSK', value: 'aka_psk' },
  { label: 'AKA Cert', value: 'aka_cert' },
];
const AUTH_OPTIONS = [
  { label: 'PSK', value: 'psk' },
  { label: 'Pubkey', value: 'pubkey' },
  { label: 'EAP-AKA', value: 'eap-aka' },
];
const ENCRYPTION_OPTIONS = [
  { label: 'AES128', value: 'aes128' },
  { label: 'AES256', value: 'aes256' },
  { label: '3DES', value: '3des' },
  { label: 'DES', value: 'des' },
];
const DH_GROUP_OPTIONS = [
  { label: 'MODP768', value: 'modp768' },
  { label: 'MODP1024', value: 'modp1024' },
  { label: 'MODP1536', value: 'modp1536' },
  { label: 'MODP2048', value: 'modp2048' },
  { label: 'MODP4096', value: 'modp4096' },
];
const AUTH_ALGORITHM_OPTIONS = [
  { label: 'SHA1', value: 'sha1' },
  { label: 'SHA1_160', value: 'sha1_160' },
  { label: 'SHA256_96', value: 'sha256_96' },
  { label: 'SHA256', value: 'sha256' },
];
const DPD_ACTION_OPTIONS = [
  { label: 'None', value: 'none' },
  { label: 'Clear', value: 'clear' },
  { label: 'Hold', value: 'hold' },
  { label: 'Restart', value: 'restart' },
];

export default function AddPolicyPage() {
  const t = useT();
  const navigate = useNavigate();
  const location = useLocation();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);

  // License status config with translations
  const LICENSE_STATUS_CONFIG = useMemo(() => ({
    '0': { label: t('provision.licenseStatusPending'), color: 'default' },
    '1': { label: t('provision.licenseStatusSuccess'), color: 'success' },
    '2': { label: t('provision.licenseStatusFailed'), color: 'error' },
    '3': { label: t('provision.licenseStatusRunning'), color: 'processing' },
  }), [t]);

  // Get mode from URL path
  const pathParts = location.pathname.split('/');
  const lastPart = pathParts[pathParts.length - 2];
  const isEdit = lastPart === 'edit';
  const isView = lastPart === 'view';

  // State for software upgrade
  const [selectedOriginalVersions, setSelectedOriginalVersions] = useState<OriginalVersion[]>([]);
  const [manualVersion, setManualVersion] = useState('');
  const [selectedAvailableVersions, setSelectedAvailableVersions] = useState<string[]>([]);
  const [showVersionList, setShowVersionList] = useState(false);

  // State for license
  const [licenseFiles, setLicenseFiles] = useState<LicenseFile[]>(MOCK_LICENSE_FILES);
  const [licenseSearchText, setLicenseSearchText] = useState('');
  const [licenseImportModalVisible, setLicenseImportModalVisible] = useState(false);
  const [licenseFileList, setLicenseFileList] = useState<UploadFile[]>([]);

  // State for self config (new design - config list)
  const [paramConfigList, setParamConfigList] = useState<ParamConfig[]>(MOCK_PARAM_CONFIGS);
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
  const [productClass, setProductClass] = useState<string>('');

  // Page title
  const pageTitle = useMemo(() => {
    if (isView) return t('common.detail');
    if (isEdit) return t('common.edit') + ' Policy';
    return t('common.add') + ' Policy';
  }, [isView, isEdit, t]);

  // Handle product type change
  const handleProductClassChange = useCallback((value: string) => {
    setProductClass(value);
  }, []);

  // Handle add version from list
  const handleAddVersionsFromList = useCallback(() => {
    const versionsToAdd = selectedAvailableVersions.filter(
      v => !selectedOriginalVersions.find(sv => sv.originalVersion === v)
    );
    if (versionsToAdd.length > 0) {
      setSelectedOriginalVersions(prev => [
        ...prev,
        ...versionsToAdd.map(v => ({ originalVersion: v })),
      ]);
    }
    setSelectedAvailableVersions([]);
    setShowVersionList(false);
  }, [selectedAvailableVersions, selectedOriginalVersions]);

  // Handle manual add version
  const handleManualAddVersion = useCallback(() => {
    if (manualVersion && !selectedOriginalVersions.find(v => v.originalVersion === manualVersion)) {
      setSelectedOriginalVersions(prev => [...prev, { originalVersion: manualVersion }]);
      setManualVersion('');
    }
  }, [manualVersion, selectedOriginalVersions]);

  // Handle delete version
  const handleDeleteVersion = useCallback((version: string) => {
    setSelectedOriginalVersions(prev => {
      const newVersions = prev.filter(v => v.originalVersion !== version);
      form.setFieldValue('originalVersion', newVersions.map(v => v.originalVersion).join(','));
      return newVersions;
    });
  }, [form]);

  // Handle clear versions
  const handleClearVersions = useCallback(() => {
    setSelectedOriginalVersions([]);
    form.setFieldValue('originalVersion', '');
  }, [form]);

  // Handle license delete
  const handleDeleteLicense = useCallback((fileName: string) => {
    setLicenseFiles(prev => prev.filter(f => f.file_name !== fileName));
  }, []);

  // Available versions list
  const availableVersions = AVAILABLE_ORIGINAL_VERSIONS;

  // Filtered license files
  const filteredLicenseFiles = useMemo(() => {
    if (!licenseSearchText) return licenseFiles;
    return licenseFiles.filter(f =>
      f.serial_number.toLowerCase().includes(licenseSearchText.toLowerCase())
    );
  }, [licenseFiles, licenseSearchText]);

  // License file table columns
  const licenseColumns = [
    {
      title: t('table.operation'),
      key: 'action',
      width: 80,
      fixed: 'right' as const,
      render: (_: unknown, record: LicenseFile) => (
        <Button
          type="link"
          size="small"
          danger
          icon={<DeleteOutlined />}
          onClick={() => handleDeleteLicense(record.file_name)}
        />
      ),
    },
    {
      title: t('provision.deviceCode'),
      dataIndex: 'serial_number',
      key: 'serial_number',
    },
    {
      title: 'License ' + t('common.file'),
      dataIndex: 'file_name',
      key: 'file_name',
    },
    {
      title: t('provision.uploadTime'),
      dataIndex: 'upload_time',
      key: 'upload_time',
    },
    {
      title: t('table.status'),
      dataIndex: 'execute_status',
      key: 'execute_status',
      render: (status: string) => {
        const cfg = LICENSE_STATUS_CONFIG[status as keyof typeof LICENSE_STATUS_CONFIG];
        return <Tag color={cfg?.color || 'default'}>{cfg?.label || status}</Tag>;
      },
    },
  ];

  // Param config table columns (simplified - only basic fields)
  const selfConfigEnabled = Form.useWatch('selfConfigEnable', form);

  // Simplified columns for all device types
  const paramConfigColumns = [
    {
      title: t('table.operation'),
      key: 'action',
      width: 100,
      fixed: 'right' as const,
      render: (_: unknown, record: ParamConfig) => {
        const items: MenuProps['items'] = [
          { key: 'edit', label: t('common.edit'), icon: <EditOutlined />, disabled: !!selfConfigEnabled,
            onClick: () => { setCurrentConfig(record); setConfigDetailMode('edit'); configForm.setFieldsValue(record); setConfigDetailVisible(true); },
          },
          { key: 'delete', label: t('common.delete'), icon: <DeleteOutlined />, danger: true, disabled: !!selfConfigEnabled,
            onClick: () => { setParamConfigList(prev => prev.filter(item => item.id !== record.id)); message.success(t('common.success')); },
          },
        ];
        return (
          <Space size={4}>
            <Button type="link" size="small" icon={<EyeOutlined />}
              onClick={() => { setCurrentConfig(record); setConfigDetailMode('view'); configForm.setFieldsValue(record); setConfigDetailVisible(true); }}>
              {t('common.view')}
            </Button>
            <Dropdown menu={{ items }} trigger={['click']}>
              <Button type="text" size="small" icon={<MoreOutlined />} />
            </Dropdown>
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

  // Filtered param config list - filter by productClass and search text
  const filteredParamConfigList = useMemo(() => {
    let list = paramConfigList;

    // Filter by product type (device type)
    if (productClass) {
      list = list.filter(item => item.deviceType === productClass);
    }

    // Filter by search text
    if (configSearchText) {
      list = list.filter(item =>
        item.serialNumber.toLowerCase().includes(configSearchText.toLowerCase())
      );
    }

    return list;
  }, [paramConfigList, productClass, configSearchText]);

  // Handle export
  const handleExportConfig = useCallback(() => {
    const data = filteredParamConfigList.map(item => ({
      serialNumber: item.serialNumber,
      cellName: item.cellName,
      bandsSupport: item.bandsSupport,
      bandWidth: item.bandWidth,
      frequency: item.frequency,
      subframeAssignment: item.subframeAssignment,
    }));
    console.log('Export data:', data);
    message.success(t('common.success'));
  }, [filteredParamConfigList, t]);

  // Handle config form submit
  const handleConfigFormSubmit = useCallback(() => {
    configForm.validateFields().then(values => {
      if (currentConfig) {
        // Edit mode
        setParamConfigList(prev => prev.map(item =>
          item.id === currentConfig.id ? { ...item, ...values } : item
        ));
      }
      setConfigDetailVisible(false);
      setCurrentConfig(null);
      configForm.resetFields();
      message.success(t('common.success'));
    });
  }, [currentConfig, configForm, t]);

  // Handle import
  const handleImportConfig = useCallback((_file: File) => {
    // Simulate import
    const newConfig: ParamConfig = {
      id: Date.now().toString(),
      deviceType: 'eNB',
      serialNumber: `ENB${Date.now().toString().slice(-5)}`,
      cellName: `Cell-${Date.now().toString().slice(-4)}`,
      bandsSupport: 38,
      bandWidth: '20MHz',
      frequency: 36000,
      subframeAssignment: 2,
      updatedBy: 'import',
      updatedAt: new Date().toLocaleString(),
    };
    if (paramImportType === 'replace') {
      setParamConfigList([newConfig]);
    } else {
      setParamConfigList(prev => [...prev, newConfig]);
    }
    setImportModalVisible(false);
    message.success(t('common.success'));
    return false;
  }, [paramImportType, t]);

  // Handle submit
  const handleSubmit = useCallback(async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);

      // Simulate API call
      await new Promise(resolve => setTimeout(resolve, 1000));

      console.log('Submit values:', values);
      message.success(t('common.success'));
      navigate('/device/plug-and-play');
    } catch (error) {
      console.error('Validation error:', error);
    } finally {
      setLoading(false);
    }
  }, [form, navigate, t]);

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
          <Switch size="small" checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
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

        <Form.Item noStyle shouldUpdate={(prev, curr) => prev.specifyVersionType !== curr.specifyVersionType}>
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
              <div>
                {/* Manual input row */}
                <div style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
                  <Input
                    value={manualVersion}
                    onChange={(e) => setManualVersion(e.target.value)}
                    placeholder={t('provision.enterVersion')}
                    style={{ width: 240 }}
                    onPressEnter={handleManualAddVersion}
                  />
                  <Button type="primary" icon={<PlusOutlined />} onClick={handleManualAddVersion}>
                    {t('common.add')}
                  </Button>
                  <Button
                    type={showVersionList ? 'primary' : 'default'}
                    ghost={showVersionList}
                    onClick={() => setShowVersionList(!showVersionList)}
                  >
                    {t('provision.selectFromList')}
                  </Button>
                </div>

                {/* Selected versions as tags */}
                <div style={{ marginBottom: showVersionList ? 12 : 0 }}>
                  {selectedOriginalVersions.length > 0 ? (
                    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4, alignItems: 'center' }}>
                      <Text type="secondary" style={{ marginRight: 4 }}>
                        {t('provision.selectedVersions')} ({selectedOriginalVersions.length}):
                      </Text>
                      {selectedOriginalVersions.map(v => (
                        <Tag
                          key={v.originalVersion}
                          closable
                          color="blue"
                          onClose={() => handleDeleteVersion(v.originalVersion)}
                        >
                          {v.originalVersion}
                        </Tag>
                      ))}
                      <Button type="link" size="small" danger onClick={handleClearVersions}>
                        {t('provision.clearAll')}
                      </Button>
                    </div>
                  ) : (
                    <Text type="secondary">{t('provision.noVersionAdded')}</Text>
                  )}
                </div>

                {/* Available versions list (collapsible) */}
                {showVersionList && (
                  <div style={{
                    border: '1px solid #f0f0f0',
                    borderRadius: 4,
                    marginTop: 8,
                    maxHeight: 220,
                    overflow: 'auto',
                    display: 'flex',
                    flexDirection: 'column',
                  }}>
                    <div style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0', background: '#fafafa', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <Checkbox
                        checked={availableVersions.length > 0 && availableVersions.every(v => selectedAvailableVersions.includes(v.originalVersion))}
                        indeterminate={selectedAvailableVersions.length > 0 && !availableVersions.every(v => selectedAvailableVersions.includes(v.originalVersion))}
                        onChange={(e) => {
                          if (e.target.checked) {
                            setSelectedAvailableVersions(availableVersions.map(v => v.originalVersion));
                          } else {
                            setSelectedAvailableVersions([]);
                          }
                        }}
                      >
                        {t('provision.selectAll')}
                      </Checkbox>
                      <Button
                        size="small"
                        type="primary"
                        disabled={selectedAvailableVersions.length === 0}
                        onClick={handleAddVersionsFromList}
                      >
                        {t('common.confirm')} ({selectedAvailableVersions.length})
                      </Button>
                    </div>
                    {availableVersions.map(v => (
                      <div key={v.originalVersion} style={{
                        padding: '6px 12px',
                        borderBottom: '1px solid #f5f5f5',
                        display: 'flex',
                        alignItems: 'center',
                      }}>
                        <Checkbox
                          checked={selectedAvailableVersions.includes(v.originalVersion)}
                          onChange={(e) => {
                            if (e.target.checked) {
                              setSelectedAvailableVersions(prev => [...prev, v.originalVersion]);
                            } else {
                              setSelectedAvailableVersions(prev => prev.filter(id => id !== v.originalVersion));
                            }
                          }}
                        >
                          {v.originalVersion}
                        </Checkbox>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            );
          }}
        </Form.Item>
      </div>

      {/* 分隔线 */}
      <Divider style={{ margin: '8px 0 16px' }} />

      {/* 目标版本 + 保留配置 */}
      <div style={{ display: 'flex', alignItems: 'flex-end', gap: 32 }}>
        <Form.Item name="targetVersion" label={t('provision.targetVersion')} style={{ marginBottom: 0 }}>
          <Select placeholder={t('common.pleaseSelect')} style={{ width: 240 }} options={TARGET_VERSIONS} />
        </Form.Item>
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
          <Switch size="small" checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
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
            <Button size="small" type="primary" icon={<UploadOutlined />} onClick={() => setLicenseImportModalVisible(true)}>
              {t('common.import')}
            </Button>
          </Space>
        </div>
        <Table
          columns={licenseColumns}
          dataSource={filteredLicenseFiles}
          rowKey="file_name"
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
          <Switch size="small" checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
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
            <Button icon={<UploadOutlined />} onClick={() => setImportModalVisible(true)}>
              {t('common.import')}
            </Button>
            <Button icon={<DownloadOutlined />} onClick={handleExportConfig}>
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
          disabled={isView}
          initialValues={{
            executeType: '0',
            functionModule: '0',
            upgradeEnable: false,
            licenseEnable: false,
            selfConfigEnable: false,
            switchEnable: false,
            specifyVersionType: 'specify',
            preserveSetting: false,
          }}
        >
          {/* Basic Info Card */}
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
              <Descriptions.Item label={t('provision.productClass')}>
                <Form.Item name="productClass" noStyle rules={[{ required: true, message: t('common.pleaseSelect') }]}>
                  <Select placeholder={t('common.pleaseSelect')} options={PRODUCT_TYPES} onChange={handleProductClassChange} />
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
          {functionModule === '0' && renderSoftwareUpgradeConfig()}
          {functionModule === '1' && renderLicenseConfig()}
          {functionModule === '2' && renderSelfConfig()}

          {/* Hidden field for original version */}
          <Form.Item name="originalVersion" hidden>
            <Input />
          </Form.Item>
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
        width={720}
        destroyOnClose
      >
        <Form form={configForm} layout="vertical" disabled={configDetailMode === 'view'} style={{ paddingBottom: 60 }}>
          {/* eNB specific fields */}
          {currentConfig?.deviceType === 'eNB' && (
            <Collapse defaultActiveKey={['enb-basic', 'enb-core', 'enb-ip', 'enb-ipsec', 'enb-wan', 'enb-custom']} ghost>
              <Collapse.Panel key="enb-basic" header={t('provision.enbBasicConfig')}>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                  <Form.Item name="eNodeBId" label="eNodeB ID" style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="plmnId" label="PLMN ID" style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="tac" label="TAC" tooltip={`${t('provision.integer')}, ${t('provision.range')}: 0~65535`} style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} min={0} max={65535} />
                  </Form.Item>
                  <Form.Item name="cellIdentity" label="ECI" tooltip={`${t('provision.integer')}, ${t('provision.range')}: 0~268435455`} style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} min={0} max={268435455} />
                  </Form.Item>
                  <Form.Item name="phycellid" label="PCI" tooltip={`${t('provision.integer')}, ${t('provision.range')}: 0~503`} style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} min={0} max={503} />
                  </Form.Item>
                  <Form.Item name="bandsSupport" label={t('provision.bandsSupport')} rules={[{ required: true }]} tooltip={`${t('provision.integer')}, ${t('provision.range')}: 1~62`} style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} min={1} max={62} />
                  </Form.Item>
                  <Form.Item name="bandWidth" label={t('provision.bandwidth')} rules={[{ required: true }]} style={{ flex: '1 1 200px' }}>
                    <Select options={BANDWIDTH_OPTIONS_OTHER} />
                  </Form.Item>
                  <Form.Item name="frequency" label={t('provision.frequency')} rules={[{ required: true }]} tooltip={`${t('provision.integer')}, ${t('provision.range')}: 1~65535`} style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} min={1} max={65535} />
                  </Form.Item>
                  <Form.Item name="subframeAssignment" label={t('provision.subframeAssignment')} rules={[{ required: true }]} style={{ flex: '1 1 200px' }}>
                    <Select options={SUBFRAME_OPTIONS} />
                  </Form.Item>
                  <Form.Item name="specialSubframePatterns" label={t('provision.specialSubframePatterns')} style={{ flex: '1 1 200px' }}>
                    <Select options={SPECIAL_SUBFRAME_OPTIONS} />
                  </Form.Item>
                  <Form.Item name="rootSequenceIndex" label={t('provision.rootSequenceIndex')} tooltip={`${t('provision.integer')}, ${t('provision.range')}: 0~837`} style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} min={0} max={837} />
                  </Form.Item>
                </div>
              </Collapse.Panel>
              <Collapse.Panel key="enb-core" header={t('provision.coreNetworkConfig')}>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                  <Form.Item name="halobEnable" label={t('provision.halobSwitch')} style={{ flex: '1 1 200px' }}>
                    <Select options={[
                      { label: t('provision.halobOn'), value: '1' },
                      { label: t('provision.halobOff'), value: '0' },
                    ]} />
                  </Form.Item>
                </div>
                <Divider orientation="left" style={{ margin: '12px 0 16px' }}>{t('provision.mmeList')}</Divider>
                <Form.List name="mmeList">
                  {(fields, { add, remove }) => (
                    <>
                      {fields.map(({ key, name, ...restField }) => (
                        <Card key={key} size="small" style={{ marginBottom: 12 }} title={`${t('provision.mmeItem')} ${name + 1}`} extra={
                          <Button type="link" danger icon={<DeleteOutlined />} onClick={() => remove(name)} />
                        }>
                          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                            <Form.Item {...restField} name={[name, 'mmeIp']} label="MME IP" style={{ flex: '1 1 200px' }}>
                              <Input style={{ width: '100%' }} placeholder={t('provision.mmePlaceholder')} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'mmePort']} label="MME Port" style={{ flex: '1 1 200px' }}>
                              <InputNumber style={{ width: '100%' }} min={1} max={65535} />
                            </Form.Item>
                          </div>
                        </Card>
                      ))}
                      <Button type="dashed" onClick={() => add()} block icon={<PlusOutlined />}>
                        {t('provision.addMme')}
                      </Button>
                    </>
                  )}
                </Form.List>
              </Collapse.Panel>
              <Collapse.Panel key="enb-ip" header={t('provision.ipConfig')}>
                <Card size="small" style={{ marginBottom: 12 }} title={t('provision.serviceIpConfig')}>
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                    <Form.Item name="serviceIp" label={t('provision.serviceIp')} style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name="serviceMask" label={t('provision.serviceMask')} style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name="serviceGateway" label={t('provision.serviceGateway')} style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name="serviceGatewayMask" label={t('provision.serviceGatewayMask')} style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name="serviceVlan" label={t('provision.serviceVlan')} style={{ flex: '1 1 200px' }}>
                      <InputNumber style={{ width: '100%' }} min={0} max={4095} />
                    </Form.Item>
                  </div>
                </Card>
                <Card size="small" title={t('provision.mgmtIpConfig')}>
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                    <Form.Item name="mgmtIp" label={t('provision.mgmtIp')} style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name="mgmtMask" label={t('provision.mgmtMask')} style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name="mgmtGateway" label={t('provision.mgmtGateway')} style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name="mgmtGatewayMask" label={t('provision.mgmtGatewayMask')} style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name="mgmtVlan" label={t('provision.mgmtVlan')} style={{ flex: '1 1 200px' }}>
                      <InputNumber style={{ width: '100%' }} min={0} max={4095} />
                    </Form.Item>
                  </div>
                </Card>
              </Collapse.Panel>
              <Collapse.Panel key="enb-ipsec" header={t('provision.ipsecConfig')}>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                  <Form.Item name="ipsecSwitch" label={t('provision.ipsecSwitch')} valuePropName="checked" style={{ flex: '1 1 200px' }}>
                    <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
                  </Form.Item>
                  <Form.Item name="ipsecEnable" label={t('provision.ipsecEnable')} style={{ flex: '1 1 200px' }}>
                    <Select options={[
                      { label: t('common.enable'), value: '1' },
                      { label: t('common.disable'), value: '0' },
                    ]} />
                  </Form.Item>
                  <Form.Item name="ipsecRightIkePort" label="Right IKE Port" style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="leftInterface" label="Left Interface" style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                </div>
                <Divider orientation="left" style={{ margin: '12px 0 16px' }}>{t('provision.ipsecTunnelList')}</Divider>
                <Form.List name="ipsecList">
                  {(fields, { add, remove }) => (
                    <>
                      {fields.map(({ key, name, ...restField }) => (
                        <Card key={key} size="small" style={{ marginBottom: 12 }} title={`${t('provision.ipsecTunnel')} ${name + 1}`} extra={
                          <Button type="link" danger icon={<DeleteOutlined />} onClick={() => remove(name)} />
                        }>
                          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                            <Form.Item {...restField} name={[name, 'TUNNEL_ENABLE']} label={t('provision.tunnelEnable')} valuePropName="checked" style={{ flex: '1 1 200px' }}>
                              <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'authBy']} label="AuthBy" style={{ flex: '1 1 200px' }}>
                              <Select options={AUTH_BY_OPTIONS} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'LEFT_AUTH']} label="Left Auth" style={{ flex: '1 1 200px' }}>
                              <Select options={AUTH_OPTIONS} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'RIGHT_AUTH']} label="Right Auth" style={{ flex: '1 1 200px' }}>
                              <Select options={AUTH_OPTIONS} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'TUNNEL_GATEWAY']} label={t('provision.tunnelGateway')} style={{ flex: '1 1 200px' }}>
                              <Input style={{ width: '100%' }} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'LEFT_IDENTIFIER']} label="Left ID" style={{ flex: '1 1 200px' }}>
                              <Input style={{ width: '100%' }} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'RIGHT_IDENTIFIER']} label="Right ID" style={{ flex: '1 1 200px' }}>
                              <Input style={{ width: '100%' }} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'LEFT_CERT']} label="Left Cert" style={{ flex: '1 1 200px' }}>
                              <Input style={{ width: '100%' }} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'SECRET_KEY']} label="Secret Key" style={{ flex: '1 1 200px' }}>
                              <Input style={{ width: '100%' }} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'RIGHT_SECRET_KEY']} label="Right Secret Key" style={{ flex: '1 1 200px' }}>
                              <Input style={{ width: '100%' }} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'LEFTSOURCEIP']} label="Left Source IP" style={{ flex: '1 1 200px' }}>
                              <Input style={{ width: '100%' }} placeholder="%config" />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'LEFT_SUBNET']} label="Left Subnet" style={{ flex: '1 1 200px' }}>
                              <Input style={{ width: '100%' }} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'RIGHT_SUBNET']} label="Right Subnet" style={{ flex: '1 1 200px' }}>
                              <Input style={{ width: '100%' }} />
                            </Form.Item>
                          </div>
                          <Divider orientation="left" style={{ margin: '8px 0 12px', fontSize: 12 }}>IKE</Divider>
                          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                            <Form.Item {...restField} name={[name, 'IKE_ENCRYPTION']} label="IKE Encryption" style={{ flex: '1 1 200px' }}>
                              <Select options={ENCRYPTION_OPTIONS} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'IKE_DH_GROUP']} label="IKE DH Group" style={{ flex: '1 1 200px' }}>
                              <Select options={DH_GROUP_OPTIONS} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'IKE_AUTHENTICATION']} label="IKE Auth" style={{ flex: '1 1 200px' }}>
                              <Select options={AUTH_ALGORITHM_OPTIONS} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'IKELIFETIME']} label="IKE Life Time" style={{ flex: '1 1 200px' }}>
                              <Input style={{ width: '100%' }} placeholder="e.g. 8h" />
                            </Form.Item>
                          </div>
                          <Divider orientation="left" style={{ margin: '8px 0 12px', fontSize: 12 }}>ESP</Divider>
                          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                            <Form.Item {...restField} name={[name, 'ESP_ENCRYPTION']} label="ESP Encryption" style={{ flex: '1 1 200px' }}>
                              <Select options={ENCRYPTION_OPTIONS} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'ESP_DH_GROUP']} label="ESP DH Group" style={{ flex: '1 1 200px' }}>
                              <Select options={DH_GROUP_OPTIONS} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'ESP_AUTHENTICATION']} label="ESP Auth" style={{ flex: '1 1 200px' }}>
                              <Select options={AUTH_ALGORITHM_OPTIONS} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'KEYLIFE']} label="Key Life" style={{ flex: '1 1 200px' }}>
                              <Input style={{ width: '100%' }} placeholder="e.g. 1h" />
                            </Form.Item>
                          </div>
                          <Divider orientation="left" style={{ margin: '8px 0 12px', fontSize: 12 }}>DPD / Rekey</Divider>
                          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                            <Form.Item {...restField} name={[name, 'REKEYMARGIN']} label="Rekey Margin" style={{ flex: '1 1 200px' }}>
                              <Input style={{ width: '100%' }} placeholder="e.g. 9m" />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'DPDACTION']} label="DPD Action" style={{ flex: '1 1 200px' }}>
                              <Select options={DPD_ACTION_OPTIONS} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'DPDDELAY']} label="DPD Delay" style={{ flex: '1 1 200px' }}>
                              <Input style={{ width: '100%' }} placeholder="e.g. 30s" />
                            </Form.Item>
                          </div>
                        </Card>
                      ))}
                      <Button type="dashed" onClick={() => add()} block icon={<PlusOutlined />}>
                        {t('provision.addTunnel')}
                      </Button>
                    </>
                  )}
                </Form.List>
              </Collapse.Panel>
              <Collapse.Panel key="enb-wan" header={t('provision.enbWanConfig')}>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                  <Form.Item name="wanSendEnable" label={t('provision.wanSendEnable')} valuePropName="checked" style={{ flex: '1 1 200px' }}>
                    <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
                  </Form.Item>
                </div>

                {/* WAN(OAM-TR069) */}
                <Card size="small" style={{ marginBottom: 12 }} title={t('provision.wanOamTr069')}>
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                    <Form.Item name={['wanOamTr069', 'ipAddress']} label={t('provision.ipAddr')} style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name={['wanOamTr069', 'netmask']} label={t('provision.subnetMask')} style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name={['wanOamTr069', 'gateway']} label={t('provision.gateway')} style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name={['wanOamTr069', 'vlanId']} label="VLAN ID" style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name={['wanOamTr069', 'binding']} label="TR069 Binding" style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                  </div>
                </Card>

                {/* WAN(S1-C) */}
                <Card size="small" style={{ marginBottom: 12 }} title={t('provision.wanS1c')}>
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                    <Form.Item name={['wanS1c', 'ipAddress']} label={t('provision.ipAddr')} style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name={['wanS1c', 'netmask']} label={t('provision.subnetMask')} style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name={['wanS1c', 'gateway']} label={t('provision.gateway')} style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name={['wanS1c', 'vlanId']} label="VLAN ID" style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name={['wanS1c', 'binding']} label="S1-C Binding" style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                  </div>
                </Card>

                {/* WAN(S1-U) */}
                <Card size="small" style={{ marginBottom: 12 }} title={t('provision.wanS1u')}>
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                    <Form.Item name={['wanS1u', 'ipAddress']} label={t('provision.ipAddr')} style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name={['wanS1u', 'netmask']} label={t('provision.subnetMask')} style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name={['wanS1u', 'gateway']} label={t('provision.gateway')} style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name={['wanS1u', 'vlanId']} label="VLAN ID" style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name={['wanS1u', 'binding']} label="S1-U Binding" style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                  </div>
                </Card>

                {/* WAN(X2AP) */}
                <Card size="small" style={{ marginBottom: 12 }} title={t('provision.wanX2ap')}>
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                    <Form.Item name={['wanX2ap', 'ipAddress']} label={t('provision.ipAddr')} style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name={['wanX2ap', 'netmask']} label={t('provision.subnetMask')} style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name={['wanX2ap', 'gateway']} label={t('provision.gateway')} style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name={['wanX2ap', 'vlanId']} label="VLAN ID" style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name={['wanX2ap', 'binding']} label="X2-ap Binding" style={{ flex: '1 1 200px' }}>
                      <Input style={{ width: '100%' }} />
                    </Form.Item>
                  </div>
                </Card>
              </Collapse.Panel>

              {/* 自定义参数 */}
              <Collapse.Panel key="enb-custom" header={t('provision.customParams')}>
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

          {/* gNB specific fields */}
          {currentConfig?.deviceType === 'gNB' && (
            <Collapse defaultActiveKey={['gnb-basic', 'gnb-sync', 'gnb-amf', 'gnb-ip', 'gnb-dns', 'gnb-ipsec', 'gnb-custom']} ghost>
              <Collapse.Panel key="gnb-basic" header={t('provision.gnbBasicConfig')}>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                  <Form.Item name="gnbName" label="gNB Name" style={{ flex: '1 1 300px' }}>
                    <Input style={{ width: '100%' }} placeholder={t('provision.gnbNamePlaceholder')} maxLength={150} />
                  </Form.Item>
                  <Form.Item name="plmnId" label="PLMN ID" style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="gnbId" label="gNB ID" style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="gnbIdLength" label="gNB ID Length" tooltip={`${t('provision.integer')}, ${t('provision.range')}: 22~32`} style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} min={22} max={32} />
                  </Form.Item>
                  <Form.Item name="nci" label="NCI" tooltip={`${t('provision.range')}: 0~68719476735`} style={{ flex: '1 1 300px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="tac" label="TAC" tooltip={`${t('provision.integer')}, ${t('provision.range')}: 0~16777215`} style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} min={0} max={16777215} />
                  </Form.Item>
                  <Form.Item name="ranac" label={t('provision.ranac')} tooltip={`${t('provision.integer')}, ${t('provision.range')}: 0~255`} style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} min={0} max={255} />
                  </Form.Item>
                </div>
                <Divider orientation="left" style={{ margin: '16px 0 12px', fontSize: 12 }}>Radio Config</Divider>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                  <Form.Item name="pci" label="PCI" tooltip={`${t('provision.integer')}, ${t('provision.range')}: 0~1007`} style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} min={0} max={1007} />
                  </Form.Item>
                  <Form.Item name="freqBandIndicator" label={t('provision.freqBandIndicator')} tooltip={`${t('provision.integer')}, ${t('provision.range')}: 1~1024`} style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} min={1} max={1024} />
                  </Form.Item>
                  <Form.Item name="nrarfcnndl" label={t('provision.nrarfcnndl')} tooltip={`${t('provision.integer')}, ${t('provision.range')}: 0~3279165`} style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} min={0} max={3279165} />
                  </Form.Item>
                  <Form.Item name="dlbandwidth" label={t('provision.dlbandwidth')} style={{ flex: '1 1 200px' }}>
                    <Select options={[
                      { label: '5 MHz', value: '5' },
                      { label: '10 MHz', value: '10' },
                      { label: '15 MHz', value: '15' },
                      { label: '20 MHz', value: '20' },
                      { label: '25 MHz', value: '25' },
                      { label: '30 MHz', value: '30' },
                      { label: '40 MHz', value: '40' },
                      { label: '50 MHz', value: '50' },
                      { label: '60 MHz', value: '60' },
                      { label: '80 MHz', value: '80' },
                      { label: '90 MHz', value: '90' },
                      { label: '100 MHz', value: '100' },
                      { label: '200 MHz', value: '200' },
                      { label: '400 MHz', value: '400' },
                    ]} />
                  </Form.Item>
                  <Form.Item name="ssbFrequency" label={t('provision.ssbFrequency')} tooltip={`${t('provision.integer')}, ${t('provision.range')}: 0~3279165`} style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} min={0} max={3279165} />
                  </Form.Item>
                  <Form.Item name="nrarfcnul" label={t('provision.nrarfcnul')} tooltip={`${t('provision.integer')}, ${t('provision.range')}: 0~3279165`} style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} min={0} max={3279165} />
                  </Form.Item>
                  <Form.Item name="duplexMode" label={t('provision.duplexMode')} style={{ flex: '1 1 200px' }}>
                    <Select options={[
                      { label: 'TDD', value: 'TDD' },
                      { label: 'FDD', value: 'FDD' },
                    ]} />
                  </Form.Item>
                  <Form.Item name="arfcn" label="ARFCN" style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="ssbAbsoluteFrequency" label={t('provision.ssbAbsoluteFrequency')} style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="frameOffset" label={t('provision.frameOffset')} style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="prachConfigIndex" label="PRACH Config Index" style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} />
                  </Form.Item>
                </div>
                <Divider orientation="left" style={{ margin: '16px 0 12px', fontSize: 12 }}>Slice Config</Divider>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                  <Form.Item name="sliceSst" label={t('provision.sliceSst')} style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="sliceSd" label={t('provision.sliceSd')} style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                </div>
              </Collapse.Panel>
              <Collapse.Panel key="gnb-sync" header={t('provision.syncConfig')}>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                  <Form.Item name="gpsSync" label={t('provision.gpsSync')} valuePropName="checked" style={{ flex: '1 1 200px' }}>
                    <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
                  </Form.Item>
                  <Form.Item name="ntpSync" label={t('provision.ntpSync')} valuePropName="checked" style={{ flex: '1 1 200px' }}>
                    <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
                  </Form.Item>
                  <Form.Item name="offsetToPointA" label={t('provision.offsetToPointA')} style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="kssb" label="Kssb" style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} />
                  </Form.Item>
                </div>
              </Collapse.Panel>
              <Collapse.Panel key="gnb-amf" header={t('provision.amfConfig')}>
                <Form.List name="amfList">
                  {(fields, { add, remove }) => (
                    <>
                      {fields.map(({ key, name, ...restField }) => (
                        <Card key={key} size="small" style={{ marginBottom: 12 }} title={`${t('provision.amfItem')} ${name + 1}`} extra={
                          <Button type="link" danger icon={<DeleteOutlined />} onClick={() => remove(name)} />
                        }>
                          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                            <Form.Item {...restField} name={[name, 'amfIp']} label="AMF IP" style={{ flex: '1 1 200px' }}>
                              <Input style={{ width: '100%' }} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'amfPort']} label="AMF Port" style={{ flex: '1 1 200px' }}>
                              <Input style={{ width: '100%' }} />
                            </Form.Item>
                          </div>
                        </Card>
                      ))}
                      <Button type="dashed" onClick={() => add()} block icon={<PlusOutlined />}>
                        {t('provision.addAmf')}
                      </Button>
                    </>
                  )}
                </Form.List>
              </Collapse.Panel>
              <Collapse.Panel key="gnb-ip" header={t('provision.ipConfig')}>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                  <Form.Item name="serviceIp" label={t('provision.serviceIp')} style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="serviceMask" label={t('provision.serviceMask')} style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="omIp" label={t('provision.omIp')} style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="omMask" label={t('provision.omMask')} style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="serviceGateway" label={t('provision.serviceGateway')} style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="serviceGatewayMask" label={t('provision.serviceGatewayMask')} style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="mgmtGateway" label={t('provision.mgmtGateway')} style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="mgmtGatewayMask" label={t('provision.mgmtGatewayMask')} style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="serviceVlan" label={t('provision.serviceVlan')} style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} min={0} max={4095} />
                  </Form.Item>
                  <Form.Item name="mgmtVlan" label={t('provision.mgmtVlan')} style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} min={0} max={4095} />
                  </Form.Item>
                </div>
              </Collapse.Panel>
              <Collapse.Panel key="gnb-tdd" header={t('provision.tddPatternConfig')}>
                <Form.Item noStyle shouldUpdate={(prev, curr) => prev.duplexMode !== curr.duplexMode}>
                  {({ getFieldValue }) => {
                    const duplexMode = getFieldValue('duplexMode');
                    if (duplexMode !== 'TDD') {
                      return <Text type="secondary">{t('provision.tddPatternHint')}</Text>;
                    }
                    return (
                      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                        <Form.Item name="subframeSlot" label={t('provision.subframeSlot')} style={{ flex: '1 1 200px' }}>
                          <Select options={[
                            { label: 'DSDDU', value: 'DSDDU' },
                            { label: 'DDDSU', value: 'DDDSU' },
                            { label: 'DDSUU', value: 'DDSUU' },
                          ]} />
                        </Form.Item>
                        <Form.Item name="subframeSlotDlUl" label={t('provision.subframeSlotDlUl')} style={{ flex: '1 1 200px' }}>
                          <Select options={[
                            { label: 'DDDDDDDSUU', value: 'DDDDDDDSUU' },
                            { label: 'DDDDDDDSUD', value: 'DDDDDDDSUD' },
                            { label: 'DDDUUDDDUU', value: 'DDDUUDDDUU' },
                          ]} />
                        </Form.Item>
                        <Form.Item name="slotConfig" label={t('provision.slotConfig')} style={{ flex: '1 1 200px' }}>
                          <Input style={{ width: '100%' }} />
                        </Form.Item>
                      </div>
                    );
                  }}
                </Form.Item>
              </Collapse.Panel>
              <Collapse.Panel key="gnb-plmn" header={t('provision.plmnConfigList')}>
                <Form.List name="plmnConfigList">
                  {(fields, { add, remove }) => (
                    <>
                      {fields.map(({ key, name, ...restField }) => (
                        <Card key={key} size="small" style={{ marginBottom: 12 }} title={`${t('provision.plmnConfig')} ${name + 1}`} extra={
                          <Button type="link" danger icon={<DeleteOutlined />} onClick={() => remove(name)} />
                        }>
                          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                            <Form.Item {...restField} name={[name, 'plmnId']} label="PLMN ID" style={{ flex: '1 1 200px' }}>
                              <Input style={{ width: '100%' }} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'primary']} label={t('provision.primary')} style={{ flex: '1 1 200px' }}>
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
              </Collapse.Panel>
              <Collapse.Panel key="gnb-slice" header={t('provision.sliceConfigList')}>
                <Form.List name="sliceConfigList">
                  {(fields, { add, remove }) => (
                    <>
                      {fields.map(({ key, name, ...restField }) => (
                        <Card key={key} size="small" style={{ marginBottom: 12 }} title={`${t('provision.sliceConfig')} ${name + 1}`} extra={
                          <Button type="link" danger icon={<DeleteOutlined />} onClick={() => remove(name)} />
                        }>
                          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                            <Form.Item {...restField} name={[name, 'sd']} label="SD" style={{ flex: '1 1 200px' }}>
                              <Select options={[
                                { label: t('provision.sdEmpty'), value: '0' },
                                { label: t('provision.sdNotEmpty'), value: '1' },
                              ]} />
                            </Form.Item>
                            <Form.Item {...restField} name={[name, 'sdValue']} label="SD Value" style={{ flex: '1 1 200px' }}>
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
              </Collapse.Panel>
              <Collapse.Panel key="gnb-dns" header={t('provision.dnsConfig')}>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                  <Form.Item name="dns1" label="DNS1" style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} placeholder="e.g. 8.8.8.8" />
                  </Form.Item>
                  <Form.Item name="dns2" label="DNS2" style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} placeholder="e.g. 8.8.4.4" />
                  </Form.Item>
                </div>
              </Collapse.Panel>
              <Collapse.Panel key="gnb-ipsec" header={t('provision.ipsecConfig')}>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                  <Form.Item name="ipsecEnable" label={t('provision.ipsecEnable')} valuePropName="checked" style={{ flex: '1 1 200px' }}>
                    <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
                  </Form.Item>
                </div>
                <Form.Item noStyle shouldUpdate={(prev, curr) => prev.ipsecEnable !== curr.ipsecEnable}>
                  {({ getFieldValue }) => {
                    const ipsecEnable = getFieldValue('ipsecEnable');
                    if (!ipsecEnable) return null;
                    return (
                      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                        <Form.Item name="ipsecImsi" label="IMSI" style={{ flex: '1 1 200px' }}>
                          <Input style={{ width: '100%' }} />
                        </Form.Item>
                        <Form.Item name="ipsecKey" label="Key" style={{ flex: '1 1 200px' }}>
                          <Input style={{ width: '100%' }} />
                        </Form.Item>
                        <Form.Item name="ipsecOpc" label="OPC" style={{ flex: '1 1 200px' }}>
                          <Input style={{ width: '100%' }} />
                        </Form.Item>
                      </div>
                    );
                  }}
                </Form.Item>
              </Collapse.Panel>
              <Collapse.Panel key="gnb-custom" header={t('provision.customParams')}>
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

          {/* GSM specific fields */}
          {currentConfig?.deviceType === 'GSM' && (
            <Collapse defaultActiveKey={['gsm-basic', 'gsm-route', 'gsm-wan', 'gsm-custom']} ghost>
              <Collapse.Panel key="gsm-basic" header={t('provision.gsmBasicConfig')}>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                  <Form.Item name="ipaUnitid" label="IPA Unit ID" style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="bscServiceIp" label={t('provision.bscServiceIp')} style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="omlRemoteIp" label="OML Remote IP" style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="omlRemoteIpBak" label="OML Remote IP (Backup)" style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="rfPower" label="RF Power" style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} />
                  </Form.Item>
                </div>
              </Collapse.Panel>
              <Collapse.Panel key="gsm-route" header={t('provision.routeConfig')}>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                  <Form.Item name="onboot" label={t('provision.onboot')} style={{ flex: '1 1 200px' }}>
                    <Select options={[
                      { label: 'Yes', value: 'yes' },
                      { label: 'No', value: 'no' },
                    ]} />
                  </Form.Item>
                  <Form.Item name="routeGateway" label={t('provision.gateway')} style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="netAddr" label={t('provision.netAddr')} style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="netMask" label={t('provision.netMask')} style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                </div>
              </Collapse.Panel>
              <Collapse.Panel key="gsm-wan" header={t('provision.wanConfig')}>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                  <Form.Item name="wanEnable" label={t('provision.enable')} valuePropName="checked" style={{ flex: '1 1 200px' }}>
                    <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
                  </Form.Item>
                  <Form.Item name="ipMode" label={t('provision.ipMode')} style={{ flex: '1 1 200px' }}>
                    <Select options={[
                      { label: 'Static', value: 'static' },
                      { label: 'DHCP', value: 'dhcp' },
                    ]} />
                  </Form.Item>
                  <Form.Item name="ipAddr" label={t('provision.ipAddr')} style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="wanNetMask" label={t('provision.subnetMask')} style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="gateway" label={t('provision.gateway')} style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="vlanId" label="VLAN ID" style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} min={0} max={4095} />
                  </Form.Item>
                </div>
              </Collapse.Panel>
              <Collapse.Panel key="gsm-dns" header={t('provision.dnsTimezoneConfig')}>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                  <Form.Item name="dns1" label="DNS1" style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} placeholder="e.g. 8.8.8.8" />
                  </Form.Item>
                  <Form.Item name="dns2" label="DNS2" style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} placeholder="e.g. 8.8.4.4" />
                  </Form.Item>
                  <Form.Item name="localTimezoneName" label={t('provision.timezone')} style={{ flex: '1 1 200px' }}>
                    <Select showSearch options={[
                      { label: 'Asia/Shanghai', value: 'Asia/Shanghai' },
                      { label: 'Asia/Hong_Kong', value: 'Asia/Hong_Kong' },
                      { label: 'Asia/Tokyo', value: 'Asia/Tokyo' },
                      { label: 'America/New_York', value: 'America/New_York' },
                      { label: 'America/Los_Angeles', value: 'America/Los_Angeles' },
                      { label: 'Europe/London', value: 'Europe/London' },
                      { label: 'Europe/Paris', value: 'Europe/Paris' },
                      { label: 'UTC', value: 'UTC' },
                    ]} />
                  </Form.Item>
                </div>
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

      {/* License Import Modal */}
      <Modal
        title={t('provision.importLicenseFile')}
        open={licenseImportModalVisible}
        onCancel={() => {
          setLicenseImportModalVisible(false);
          setLicenseFileList([]);
        }}
        footer={[
          <Button key="cancel" onClick={() => {
            setLicenseImportModalVisible(false);
            setLicenseFileList([]);
          }}>
            {t('common.cancel')}
          </Button>,
          <Button
            key="download"
            onClick={() => {
              // Download template logic
              void message.success(t('common.success'));
            }}
          >
            {t('provision.downloadTemplate')}
          </Button>,
          <Button
            key="import"
            type="primary"
            onClick={() => {
              if (licenseFileList.length === 0) {
                void message.warning(t('common.pleaseSelect'));
                return;
              }
              // Process license files
              licenseFileList.forEach(file => {
                const newFile: LicenseFile = {
                  serial_number: `NEW-${Date.now()}`,
                  file_name: file.name,
                  upload_time: new Date().toLocaleString(),
                  execute_status: '0',
                };
                setLicenseFiles(prev => [...prev, newFile]);
              });
              setLicenseImportModalVisible(false);
              setLicenseFileList([]);
              void message.success(t('common.success'));
            }}
          >
            {t('common.import')}
          </Button>,
        ]}
        width={520}
      >
        <Upload.Dragger
          accept=".lic"
          fileList={licenseFileList}
          beforeUpload={(file) => {
            setLicenseFileList([file]);
            return false;
          }}
          onRemove={() => {
            setLicenseFileList([]);
          }}
        >
          <p className="ant-upload-drag-icon">
            <InboxOutlined style={{ fontSize: 40, color: 'var(--color-primary-600)' }} />
          </p>
          <p className="ant-upload-text">{t('recycle.importSelectFile')}</p>
          <p className="ant-upload-hint" style={{ fontSize: 12, color: '#8c8c8c' }}>
            {t('provision.importLicenseHint')}
          </p>
        </Upload.Dragger>
      </Modal>

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
            key="download"
            onClick={() => {
              // Download template logic
              void message.success(t('common.success'));
            }}
          >
            {t('provision.downloadTemplate')}
          </Button>,
          <Button
            key="import"
            type="primary"
            onClick={() => {
              if (paramFileList.length === 0) {
                void message.warning(t('common.pleaseSelect'));
                return;
              }
              // Process import based on import type
              paramFileList.forEach((file) => {
                // antd UploadFile carries originFileObj: File for actual upload
                const realFile = (file.originFileObj ?? file) as File;
                handleImportConfig(realFile);
              });
              setImportModalVisible(false);
              setParamFileList([]);
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
