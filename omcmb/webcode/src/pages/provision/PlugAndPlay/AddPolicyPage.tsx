import React, { useState, useMemo, useCallback } from 'react';
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
  App,
  Typography,
  Checkbox,
  Table,
  Tag,
  Upload,
  Popconfirm,
  InputNumber,
  message,
  Modal,
  Tooltip,
  Drawer,
  Collapse,
} from 'antd';
import {
  ArrowLeftOutlined,
  UploadOutlined,
  PlusOutlined,
  DeleteOutlined,
  SearchOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  DownloadOutlined,
  EyeOutlined,
  EditOutlined,
} from '@ant-design/icons';
import { useT } from '@/hooks/useT';

const { Text, Title } = Typography;

// Types
type ExecuteType = '0' | '1';
type EnableType = '0' | '1';

interface PolicyForm {
  selfStartEnable: EnableType;
  policyName: string;
  productType: string;
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

// Param Config item for new design
interface ParamConfig {
  id: string;
  deviceType: DeviceType;
  serialNumber: string;
  cellName: string;
  updatedBy: string;
  updatedAt: string;
  // eNB 基础配置字段
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
  // eNB 核心网配置
  halobEnable?: '0' | '1';
  mme?: string;
  // gNB PLMN配置字段
  nci?: string;
  ranac?: number;
  // gNB AMF配置
  amfIp?: string;
  amfPlmnId?: string;
  amfDefault?: '0' | '1';
  // gNB WAN配置
  addressType?: 'IPv4' | 'IPv6';
  bearType?: string;
  ipAddress?: string;
  subnetMask?: string;
  prefixLength?: number;
  wanGateway?: string;
  wanVlanId?: number;
  vlanName?: string;
  // gNB LAN配置
  lanIp?: string;
  lanSubnetMask?: string;
  // GSM 基础参数
  ipaUnitid?: string;
  omlRemoteIp?: string;
  omlRemoteIpBak?: string;
  rfPower?: number;
  // GSM Route配置
  onboot?: 'yes' | 'no';
  routeGateway?: string;
  netAddr?: string;
  netMask?: string;
  // GSM WAN配置
  wanEnable?: '0' | '1';
  ipMode?: 'static' | 'dhcp';
  ipAddr?: string;
  wanNetMask?: string;
  gateway?: string;
  vlanId?: number;
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
    cellIdentity: 12345678,
    phycellid: 150,
    rootSequenceIndex: 0,
    halobEnable: '0',
    mme: '192.168.1.100',
    updatedBy: 'admin',
    updatedAt: '2026-04-07 10:00:00',
  },
  {
    id: '2',
    deviceType: 'gNB',
    serialNumber: 'GNB00001',
    cellName: 'NR-Cell-001',
    nci: '12345678901234',
    ranac: 100,
    amfIp: '192.168.1.100',
    amfPlmnId: '46001',
    amfDefault: '1',
    addressType: 'IPv4',
    bearType: 'Ethernet',
    ipAddress: '192.168.1.50',
    subnetMask: '255.255.255.0',
    wanGateway: '192.168.1.1',
    wanVlanId: 100,
    vlanName: 'WAN-VLAN',
    lanIp: '192.168.2.1',
    lanSubnetMask: '255.255.255.0',
    updatedBy: 'admin',
    updatedAt: '2026-04-07 11:00:00',
  },
  {
    id: '3',
    deviceType: 'GSM',
    serialNumber: 'GSM00001',
    cellName: 'GSM-Cell-001',
    ipaUnitid: 'IPA001',
    omlRemoteIp: '192.168.1.200',
    omlRemoteIpBak: '192.168.1.201',
    rfPower: 40,
    onboot: 'yes',
    routeGateway: '192.168.1.1',
    netAddr: '192.168.1.0',
    netMask: '255.255.255.0',
    wanEnable: '1',
    ipMode: 'static',
    ipAddr: '192.168.1.50',
    wanNetMask: '255.255.255.0',
    gateway: '192.168.1.1',
    vlanId: 100,
    updatedBy: 'user',
    updatedAt: '2026-04-07 12:00:00',
  },
];

// Bandwidth options
const BANDWIDTH_OPTIONS_DXDF = [
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

export default function AddPolicyPage() {
  const t = useT();
  const navigate = useNavigate();
  const location = useLocation();
  const { modal } = App.useApp();
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
  const policyId = pathParts[pathParts.length - 1];
  const isAdd = lastPart === 'add';
  const isEdit = lastPart === 'edit';
  const isView = lastPart === 'view';
  const mode = isAdd ? 'add' : isEdit ? 'edit' : 'view';

  // State for software upgrade
  const [selectedOriginalVersions, setSelectedOriginalVersions] = useState<OriginalVersion[]>([]);
  const [addVersionModalOpen, setAddVersionModalOpen] = useState(false);
  const [manualVersion, setManualVersion] = useState('');
  const [selectedAvailableVersions, setSelectedAvailableVersions] = useState<string[]>([]);
  const [versionSearchText, setVersionSearchText] = useState('');

  // State for license
  const [licenseFiles, setLicenseFiles] = useState<LicenseFile[]>(MOCK_LICENSE_FILES);
  const [licenseSearchText, setLicenseSearchText] = useState('');

  // State for self config (new design - config list)
  const [paramConfigList, setParamConfigList] = useState<ParamConfig[]>(MOCK_PARAM_CONFIGS);
  const [configSearchText, setConfigSearchText] = useState('');
  const [configDetailVisible, setConfigDetailVisible] = useState(false);
  const [configDetailMode, setConfigDetailMode] = useState<'view' | 'edit'>('view');
  const [currentConfig, setCurrentConfig] = useState<ParamConfig | null>(null);
  const [configForm] = Form.useForm();
  const [importModalVisible, setImportModalVisible] = useState(false);

  // Current function module
  const [functionModule, setFunctionModule] = useState<'0' | '1' | '2'>('0');
  const [productType, setProductType] = useState<string>('');

  // Page title
  const pageTitle = useMemo(() => {
    if (isView) return t('common.detail');
    if (isEdit) return t('common.edit') + ' Policy';
    return t('common.add') + ' Policy';
  }, [isView, isEdit, t]);

  // Handle product type change
  const handleProductTypeChange = useCallback((value: string) => {
    setProductType(value);
  }, []);

  // Handle add version
  const handleAddVersion = useCallback(() => {
    const versionsToAdd: OriginalVersion[] = [];

    // Add manual version
    if (manualVersion && !selectedOriginalVersions.find(v => v.originalVersion === manualVersion)) {
      versionsToAdd.push({ originalVersion: manualVersion });
    }

    // Add selected available versions
    selectedAvailableVersions.forEach(v => {
      if (!selectedOriginalVersions.find(sv => sv.originalVersion === v)) {
        versionsToAdd.push({ originalVersion: v });
      }
    });

    if (versionsToAdd.length > 0) {
      setSelectedOriginalVersions(prev => [...prev, ...versionsToAdd]);
      form.setFieldValue('originalVersion', [...selectedOriginalVersions, ...versionsToAdd].map(v => v.originalVersion).join(','));
    }

    setManualVersion('');
    setSelectedAvailableVersions([]);
    setAddVersionModalOpen(false);
  }, [manualVersion, selectedAvailableVersions, selectedOriginalVersions, form]);

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

  // Filtered available versions
  const filteredAvailableVersions = useMemo(() => {
    if (!versionSearchText) return AVAILABLE_ORIGINAL_VERSIONS;
    return AVAILABLE_ORIGINAL_VERSIONS.filter(v =>
      v.originalVersion.toLowerCase().includes(versionSearchText.toLowerCase())
    );
  }, [versionSearchText]);

  // Filtered license files
  const filteredLicenseFiles = useMemo(() => {
    if (!licenseSearchText) return licenseFiles;
    return licenseFiles.filter(f =>
      f.serial_number.toLowerCase().includes(licenseSearchText.toLowerCase())
    );
  }, [licenseFiles, licenseSearchText]);

  // Original version table columns
  const originalVersionColumns = [
    {
      title: t('provision.originalVersion'),
      dataIndex: 'originalVersion',
      key: 'originalVersion',
    },
    {
      title: t('table.operation'),
      key: 'action',
      width: 60,
      render: (_: unknown, record: OriginalVersion) => (
        <Button
          type="text"
          size="small"
          danger
          icon={<DeleteOutlined />}
          onClick={() => handleDeleteVersion(record.originalVersion)}
        />
      ),
    },
  ];

  // License file table columns
  const licenseColumns = [
    {
      title: t('table.operation'),
      key: 'action',
      width: 60,
      render: (_: unknown, record: LicenseFile) => (
        <Button
          type="text"
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
        const cfg = LICENSE_STATUS_CONFIG[status];
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
      width: 140,
      fixed: 'left' as const,
      render: (_: unknown, record: ParamConfig) => (
        <Space size={4}>
          <Tooltip title={t('common.view')}>
            <Button
              type="text"
              size="small"
              icon={<EyeOutlined />}
              onClick={() => {
                setCurrentConfig(record);
                setConfigDetailMode('view');
                configForm.setFieldsValue(record);
                setConfigDetailVisible(true);
              }}
            />
          </Tooltip>
          <Tooltip title={t('common.edit')}>
            <Button
              type="text"
              size="small"
              icon={<EditOutlined />}
              disabled={selfConfigEnabled}
              onClick={() => {
                setCurrentConfig(record);
                setConfigDetailMode('edit');
                configForm.setFieldsValue(record);
                setConfigDetailVisible(true);
              }}
            />
          </Tooltip>
          <Popconfirm
            title={t('provision.confirmDeleteConfig')}
            onConfirm={() => {
              setParamConfigList(prev => prev.filter(item => item.id !== record.id));
              message.success(t('common.success'));
            }}
            disabled={selfConfigEnabled}
          >
            <Tooltip title={t('common.delete')}>
              <Button
                type="text"
                size="small"
                danger
                icon={<DeleteOutlined />}
                disabled={selfConfigEnabled}
              />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
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

  // Filtered param config list - filter by productType and search text
  const filteredParamConfigList = useMemo(() => {
    let list = paramConfigList;

    // Filter by product type (device type)
    if (productType) {
      list = list.filter(item => item.deviceType === productType);
    }

    // Filter by search text
    if (configSearchText) {
      list = list.filter(item =>
        item.serialNumber.toLowerCase().includes(configSearchText.toLowerCase())
      );
    }

    return list;
  }, [paramConfigList, productType, configSearchText]);

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
  const handleImportConfig = useCallback((file: File) => {
    // Simulate import
    const newConfig: ParamConfig = {
      id: Date.now().toString(),
      serialNumber: `ENB${Date.now().toString().slice(-5)}`,
      cellName: `Cell-${Date.now().toString().slice(-4)}`,
      bandsSupport: 38,
      bandWidth: '20MHz',
      frequency: 36000,
      subframeAssignment: 2,
      updatedBy: 'import',
      updatedAt: new Date().toLocaleString(),
    };
    setParamConfigList(prev => [...prev, newConfig]);
    setImportModalVisible(false);
    message.success(t('common.success'));
    return false;
  }, []);

  // Handle submit
  const handleSubmit = useCallback(async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);

      // Simulate API call
      await new Promise(resolve => setTimeout(resolve, 1000));

      console.log('Submit values:', values);
      message.success(t('common.success'));
      navigate('/provision/plug-and-play');
    } catch (error) {
      console.error('Validation error:', error);
    } finally {
      setLoading(false);
    }
  }, [form, navigate, t]);

  // Handle cancel
  const handleCancel = useCallback(() => {
    navigate('/provision/plug-and-play');
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
      <div style={{ padding: '16px 0' }}>
        <Text type="secondary">{t('provision.selectOriginalVersionHint')}</Text>

        <div style={{ marginTop: 16, display: 'flex', alignItems: 'center', gap: 8 }}>
          <Text>{t('provision.originalVersion')}</Text>
          <Form.Item name="specifyVersionType" valuePropName="checked" noStyle>
            <Checkbox onChange={(e) => {
              if (e.target.checked) {
                handleClearVersions();
              }
            }}>{t('provision.all')}</Checkbox>
          </Form.Item>
        </div>

        <Form.Item noStyle shouldUpdate={(prev, curr) => prev.specifyVersionType !== curr.specifyVersionType}>
          {({ getFieldValue }) => {
            const specifyVersionType = getFieldValue('specifyVersionType');
            if (specifyVersionType) {
              return (
                <div style={{
                  width: 400,
                  height: 254,
                  border: '1px solid #E9EDF9',
                  borderRadius: 4,
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  justifyContent: 'center',
                  marginTop: 16,
                }}>
                  <PlusOutlined style={{ fontSize: 20, color: '#BFBFBF', marginBottom: 10 }} />
                  <Text type="secondary">{t('provision.allVersionsSelected')}</Text>
                </div>
              );
            }

            return (
              <div style={{ marginTop: 16 }}>
                {selectedOriginalVersions.length === 0 ? (
                  <div
                    onClick={() => setAddVersionModalOpen(true)}
                    style={{
                      width: 400,
                      height: 254,
                      border: '1px solid #E9EDF9',
                      borderRadius: 4,
                      display: 'flex',
                      flexDirection: 'column',
                      alignItems: 'center',
                      justifyContent: 'center',
                      cursor: 'pointer',
                    }}
                  >
                    <PlusOutlined style={{ fontSize: 20, color: '#4D84FF', marginBottom: 10 }} />
                    <Text>{t('provision.canAddOriginalVersion')}</Text>
                  </div>
                ) : (
                  <div style={{ border: '1px solid #E9EDF9', borderRadius: 4 }}>
                    <div style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <Input
                        placeholder={t('provision.originalVersion')}
                        prefix={<SearchOutlined />}
                        value={versionSearchText}
                        onChange={(e) => setVersionSearchText(e.target.value)}
                        style={{ width: 220 }}
                        size="small"
                      />
                      <Space>
                        <Button size="small" icon={<DeleteOutlined />} onClick={handleClearVersions} />
                        <Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => setAddVersionModalOpen(true)} />
                      </Space>
                    </div>
                    <Table
                      columns={originalVersionColumns}
                      dataSource={selectedOriginalVersions.filter(v =>
                        !versionSearchText || v.originalVersion.toLowerCase().includes(versionSearchText.toLowerCase())
                      )}
                      rowKey="originalVersion"
                      pagination={false}
                      size="small"
                      style={{ maxHeight: 180, overflow: 'auto' }}
                    />
                  </div>
                )}

                <div style={{ marginTop: 24, display: 'flex', gap: 60 }}>
                  <Form.Item name="targetVersion" label={t('provision.targetVersion')} style={{ marginBottom: 0 }}>
                    <Select placeholder={t('common.pleaseSelect')} style={{ width: 300 }} options={TARGET_VERSIONS} />
                  </Form.Item>
                  <div style={{ display: 'flex', alignItems: 'center', paddingTop: 30 }}>
                    <Form.Item name="preserveSetting" valuePropName="checked" noStyle>
                      <Checkbox>{t('provision.preserveConfig')}</Checkbox>
                    </Form.Item>
                  </div>
                </div>
              </div>
            );
          }}
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
            <Upload
              accept=".lic"
              showUploadList={false}
              beforeUpload={(file) => {
                const newFile: LicenseFile = {
                  serial_number: `NEW-${Date.now()}`,
                  file_name: file.name,
                  upload_time: new Date().toLocaleString(),
                  execute_status: '0',
                };
                setLicenseFiles(prev => [...prev, newFile]);
                message.success(t('common.success'));
                return false;
              }}
            >
              <Button size="small" type="primary" icon={<UploadOutlined />}>{t('common.import')}</Button>
            </Upload>
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
            specifyVersionType: false,
            preserveSetting: false,
          }}
        >
          {/* Basic Info Card */}
          <Card size="small" title={t('common.basicInfo')} style={{ marginBottom: 16 }}>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 24 }}>
              <Form.Item name="selfStartEnable" label={t('provision.settingSwitch')} valuePropName="checked">
                <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
              </Form.Item>

              <Form.Item
                name="policyName"
                label={t('provision.policyName')}
                rules={[{ required: true, message: t('common.pleaseInput') }]}
              >
                <Input style={{ width: 300 }} placeholder={t('provision.policyNamePlaceholder')} maxLength={50} />
              </Form.Item>

              <Form.Item
                name="productType"
                label={t('provision.productType')}
                rules={[{ required: true, message: t('common.pleaseSelect') }]}
              >
                <Select style={{ width: 150 }} placeholder={t('common.pleaseSelect')} options={PRODUCT_TYPES} onChange={handleProductTypeChange} />
              </Form.Item>

              <Form.Item name="executeType" label={t('provision.executeType')} rules={[{ required: true }]}>
                <Radio.Group>
                  <Radio value="0">{t('provision.autoExecute')}</Radio>
                  <Radio value="1">{t('provision.manualExecute')}</Radio>
                </Radio.Group>
              </Form.Item>
            </div>
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
          {functionModule === '2' && renderSelfConfig()}

          {/* Hidden field for original version */}
          <Form.Item name="originalVersion" hidden>
            <Input />
          </Form.Item>
        </Form>
      </div>

      {/* Add Version Modal */}
      {addVersionModalOpen && (
        <Card
          title={t('provision.addVersion')}
          extra={<Button type="text" icon={<CloseCircleOutlined />} onClick={() => setAddVersionModalOpen(false)} />}
          style={{
            position: 'fixed',
            right: 0,
            top: 0,
            bottom: 0,
            width: 360,
            zIndex: 1000,
            borderRadius: 0,
            boxShadow: '-2px 0 8px rgba(0,0,0,0.1)',
          }}
        >
          <div style={{ marginBottom: 16 }}>
            <Text>{t('provision.originalVersion')}</Text>
            <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
              <Input
                value={manualVersion}
                onChange={(e) => setManualVersion(e.target.value)}
                placeholder={t('provision.enterVersion')}
                style={{ flex: 1 }}
              />
              <Button type="primary" icon={<PlusOutlined />} onClick={() => {
                if (manualVersion && !selectedOriginalVersions.find(v => v.originalVersion === manualVersion)) {
                  setSelectedOriginalVersions(prev => [...prev, { originalVersion: manualVersion }]);
                  setManualVersion('');
                }
              }} />
            </div>
          </div>

          <Divider style={{ margin: '12px 0' }} />

          <Text>{t('provision.originalVersionList')}</Text>
          <div style={{ marginTop: 8, border: '1px solid #f0f0f0', borderRadius: 4, maxHeight: 280, overflow: 'auto' }}>
            <Table
              columns={[
                {
                  title: '',
                  width: 40,
                  render: (_: unknown, record: OriginalVersion) => (
                    <Checkbox
                      checked={selectedAvailableVersions.includes(record.originalVersion)}
                      onChange={(e) => {
                        if (e.target.checked) {
                          setSelectedAvailableVersions(prev => [...prev, record.originalVersion]);
                        } else {
                          setSelectedAvailableVersions(prev => prev.filter(v => v !== record.originalVersion));
                        }
                      }}
                    />
                  ),
                },
                {
                  title: t('provision.originalVersion'),
                  dataIndex: 'originalVersion',
                },
              ]}
              dataSource={filteredAvailableVersions}
              rowKey="originalVersion"
              pagination={false}
              size="small"
              showHeader={false}
            />
          </div>

          <div style={{ marginTop: 16, display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <Button onClick={() => setAddVersionModalOpen(false)}>{t('common.cancel')}</Button>
            <Button type="primary" onClick={handleAddVersion}>{t('common.confirm')}</Button>
          </div>
        </Card>
      )}

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
        <Form form={configForm} layout="vertical" disabled={configDetailMode === 'view'}>
          {/* Common fields */}
          <div style={{ display: 'flex', gap: 16, marginBottom: 8 }}>
            <Form.Item name="deviceType" label={t('provision.deviceType')} style={{ flex: 1 }}>
              <Select disabled options={[
                { label: 'eNB', value: 'eNB' },
                { label: 'gNB', value: 'gNB' },
                { label: 'GSM', value: 'GSM' },
              ]} />
            </Form.Item>
            <Form.Item name="serialNumber" label={t('provision.serialNumber')} rules={[{ required: true }]} style={{ flex: 1 }}>
              <Input disabled />
            </Form.Item>
          </div>
          <Form.Item name="cellName" label={t('provision.cellName')}>
            <Input />
          </Form.Item>

          {/* eNB specific fields */}
          {currentConfig?.deviceType === 'eNB' && (
            <Collapse defaultActiveKey={['enb-basic', 'enb-core']} ghost>
              <Collapse.Panel key="enb-basic" header={t('provision.enbBasicConfig')}>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
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
                  <Form.Item name="mme" label="MME" style={{ flex: '1 1 400px' }}>
                    <Input style={{ width: '100%' }} placeholder={t('provision.mmePlaceholder')} />
                  </Form.Item>
                </div>
              </Collapse.Panel>
            </Collapse>
          )}

          {/* gNB specific fields */}
          {currentConfig?.deviceType === 'gNB' && (
            <Collapse defaultActiveKey={['gnb-plmn', 'gnb-amf', 'gnb-wan', 'gnb-lan']} ghost>
              <Collapse.Panel key="gnb-plmn" header={t('provision.plmnConfig')}>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                  <Form.Item name="nci" label="NCI" tooltip={`${t('provision.integer')}, ${t('provision.range')}: 0~68719476735`} style={{ flex: '1 1 300px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="tac" label="TAC" tooltip={`${t('provision.integer')}, ${t('provision.range')}: 0~16777215`} style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} min={0} max={16777215} />
                  </Form.Item>
                  <Form.Item name="ranac" label={t('provision.ranac')} tooltip={`${t('provision.integer')}, ${t('provision.range')}: 0~255`} style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} min={0} max={255} />
                  </Form.Item>
                </div>
              </Collapse.Panel>
              <Collapse.Panel key="gnb-amf" header={t('provision.amfConfig')}>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                  <Form.Item name="amfIp" label="AMF IP" style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="amfPlmnId" label="PLMN ID" style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="amfDefault" label={t('provision.defaultAmf')} style={{ flex: '1 1 200px' }}>
                    <Select options={[
                      { label: t('common.yes'), value: '1' },
                      { label: t('common.no'), value: '0' },
                    ]} />
                  </Form.Item>
                </div>
              </Collapse.Panel>
              <Collapse.Panel key="gnb-wan" header={t('provision.wanConfig')}>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                  <Form.Item name="addressType" label={t('provision.addressType')} style={{ flex: '1 1 200px' }}>
                    <Select options={[
                      { label: 'IPv4', value: 'IPv4' },
                      { label: 'IPv6', value: 'IPv6' },
                    ]} />
                  </Form.Item>
                  <Form.Item name="bearType" label={t('provision.bearType')} style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="ipAddress" label={t('provision.ipAddr')} style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="subnetMask" label={t('provision.subnetMask')} style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="wanGateway" label={t('provision.gateway')} style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="wanVlanId" label="VLAN ID" style={{ flex: '1 1 200px' }}>
                    <InputNumber style={{ width: '100%' }} min={0} max={4095} />
                  </Form.Item>
                  <Form.Item name="vlanName" label={t('provision.vlanName')} style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                </div>
              </Collapse.Panel>
              <Collapse.Panel key="gnb-lan" header={t('provision.lanConfig')}>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                  <Form.Item name="lanIp" label={t('provision.lanIp')} style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                  <Form.Item name="lanSubnetMask" label={t('provision.lanSubnetMask')} style={{ flex: '1 1 200px' }}>
                    <Input style={{ width: '100%' }} />
                  </Form.Item>
                </div>
              </Collapse.Panel>
            </Collapse>
          )}

          {/* GSM specific fields */}
          {currentConfig?.deviceType === 'GSM' && (
            <Collapse defaultActiveKey={['gsm-basic', 'gsm-route', 'gsm-wan']} ghost>
              <Collapse.Panel key="gsm-basic" header={t('provision.gsmBasicConfig')}>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
                  <Form.Item name="ipaUnitid" label="IPA Unit ID" style={{ flex: '1 1 200px' }}>
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
    );

      {/* Import Config Modal */}
      <Modal
        title={t('provision.importParamConfig')}
        open={importModalVisible}
        onCancel={() => setImportModalVisible(false)}
        footer={[
          <Button key="cancel" onClick={() => setImportModalVisible(false)}>
            {t('common.cancel')}
          </Button>,
          <Button
            key="download"
            onClick={() => {
              // Download template logic
              message.success(t('common.success'));
            }}
          >
            {t('provision.downloadTemplate')}
          </Button>,
          <Upload
            key="upload"
            accept=".xlsx,.xls,.csv"
            showUploadList={false}
            beforeUpload={(file) => {
              handleImportConfig(file);
              return false;
            }}
          >
            <Button key="import" type="primary">
              {t('common.import')}
            </Button>
          </Upload>,
        ]}
        width={500}
      >
        <div style={{ marginBottom: 16 }}>
          <Text type="secondary">{t('provision.importParamConfigHint')}</Text>
        </div>
        <div style={{ marginBottom: 16 }}>
          <Text strong>{t('provision.importTemplateFields')}:</Text>
          <ul style={{ marginTop: 8, paddingLeft: 20 }}>
            <li><Text>{t('provision.serialNumber')} ({t('common.required')})</Text></li>
            <li><Text>{t('provision.cellName')}</Text></li>
            <li><Text>{t('provision.bandsSupport')} ({t('common.required')})</Text></li>
            <li><Text>{t('provision.bandwidth')} ({t('common.required')})</Text></li>
            <li><Text>{t('provision.frequency')} ({t('common.required')})</Text></li>
            <li><Text>{t('provision.subframeAssignment')} ({t('common.required')})</Text></li>
          </ul>
        </div>
      </Modal>
    </div>
  );
}
