import React, { useState, useMemo, useRef, useCallback } from 'react';
import {
  Drawer,
  Form,
  Input,
  Select,
  Radio,
  Button,
  Space,
  Divider,
  Card,
  InputNumber,
  App,
  Upload,
  Typography,
} from 'antd';
import {
  UploadOutlined,
  DownloadOutlined,
  PlusOutlined,
  DeleteOutlined,
} from '@ant-design/icons';
import type { UploadFile } from 'antd/es/upload/interface';
import ImportPanel, { type ImportPanelRef } from '@/components/ImportPanel';
import { useT } from '@/hooks/useT';

const { Text } = Typography;

type ProvMode = 'manual' | 'import';
type StationType = 'eNB' | 'gNB' | 'GSM';
type DrawerMode = 'add' | 'edit';

interface AddDrawerProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (values: Record<string, unknown>) => void;
  loading?: boolean;
  mode?: DrawerMode;
  initialValues?: {
    stationCode?: string;
    stationName?: string;
    stationType?: StationType;
    [key: string]: unknown;
  };
}

interface SliceConfig {
  id: string;
  tac: string;
  plmn: string;
  eci: string;
}

interface SharedCellConfig {
  id: string;
  name: string;
  value: string;
}

// 4G表单初始值
const initial4GValues = {
  wanIp: '',
  eci: '',
  frequency: '',
  pci: '',
  tac: '',
  plmn: '',
  bandwidth: '',
  coreIp: '',
};

// 5G表单初始值
const initial5GValues = {
  wanIp: '',
  gnbId: '',
  frequency: '',
  pci: '',
  coreIp: '',
};

// 2G表单初始值
const initial2GValues = {
  ipaUnit: '',
  bscIp: '',
  wanIp: '',
};

// IP地址验证正则
const IP_REGEX = /^((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$/;

// IP地址验证正则和规则（message 使用函数动态获取i18n）

export default function AddDrawer({ open, onClose, onSubmit, loading, mode = 'add', initialValues }: AddDrawerProps) {
  const t = useT();
  const { message } = App.useApp();

  // IP地址验证规则（依赖 t 函数）
  const ipValidationRule = {
    pattern: IP_REGEX,
    message: t('commission.inputValidIp'),
  };
  const [form] = Form.useForm();
  const [provMode, setProvMode] = useState<ProvMode>('manual');
  const [stationType, setStationType] = useState<StationType>('eNB');
  const importPanelRef = useRef<ImportPanelRef>(null);

  // 5G切片配置
  const [slices, setSlices] = useState<SliceConfig[]>([
    { id: '1', tac: '', plmn: '', eci: '' },
  ]);

  // 5G同步配置
  const [syncConfig, setSyncConfig] = useState({
    syncMode: 'gps',
    syncServer: '',
  });

  // 5G共享小区配置
  const [sharedCells, setSharedCells] = useState<SharedCellConfig[]>([]);

  // 导入文件列表
  const [fileList, setFileList] = useState<UploadFile[]>([]);

  // 当打开抽屉时设置初始值（编辑模式）
  React.useEffect(() => {
    if (open && mode === 'edit' && initialValues) {
      form.setFieldsValue({
        stationCode: initialValues.stationCode,
        stationName: initialValues.stationName,
      });
      if (initialValues.stationType) {
        setStationType(initialValues.stationType);
      }
    }
  }, [open, mode, initialValues, form]);

  // 重置表单状态
  const resetState = () => {
    form.resetFields();
    setProvMode('manual');
    setStationType('eNB');
    setSlices([{ id: '1', tac: '', plmn: '', eci: '' }]);
    setSyncConfig({ syncMode: 'gps', syncServer: '' });
    setSharedCells([]);
    setFileList([]);
  };

  const handleClose = () => {
    resetState();
    onClose();
  };

  const handleSubmit = () => {
    if (provMode === 'import') {
      if (fileList.length === 0) {
        message.warning(t('common.pleaseSelectFile'));
        return;
      }
      // 导入模式提交
      onSubmit({
        provMode,
        stationType,
        file: fileList[0],
      });
      return;
    }

    // 手动配置模式
    form.validateFields().then((values) => {
      const submitData: Record<string, unknown> = {
        provMode,
        stationType,
        ...values,
      };

      // 5G额外配置
      if (stationType === 'gNB') {
        submitData.slices = slices;
        submitData.syncConfig = syncConfig;
        submitData.sharedCells = sharedCells;
      }

      onSubmit(submitData);
    });
  };

  // 添加切片
  const addSlice = () => {
    setSlices([...slices, { id: Date.now().toString(), tac: '', plmn: '', eci: '' }]);
  };

  // 删除切片
  const removeSlice = (id: string) => {
    if (slices.length <= 1) return;
    setSlices(slices.filter((s) => s.id !== id));
  };

  // 更新切片字段
  const updateSlice = (id: string, field: keyof SliceConfig, value: string) => {
    setSlices(slices.map((s) => (s.id === id ? { ...s, [field]: value } : s)));
  };

  // 添加共享小区
  const addSharedCell = () => {
    setSharedCells([...sharedCells, { id: Date.now().toString(), name: '', value: '' }]);
  };

  // 删除共享小区
  const removeSharedCell = (id: string) => {
    setSharedCells(sharedCells.filter((c) => c.id !== id));
  };

  // 更新共享小区字段
  const updateSharedCell = (id: string, field: keyof SharedCellConfig, value: string) => {
    setSharedCells(sharedCells.map((c) => (c.id === id ? { ...c, [field]: value } : c)));
  };

  // 4G表单
  const render4GForm = () => (
    <>
      {/* WAN IP */}
      <Card title="WAN IP" size="small" style={{ marginBottom: 16 }}>
        <Form.Item
          name="wanIp4g"
          label="WAN IP"
          rules={[
            { required: true, message: t('common.pleaseInput') },
            ipValidationRule,
          ]}
        >
          <Input placeholder={t('commission.ipExample')} />
        </Form.Item>
      </Card>

      {/* 基站参数 */}
      <Card title={t('commission.stationParams')} size="small" style={{ marginBottom: 16 }}>
        <Form.Item
          name="eci"
          label="ECI"
          rules={[
            { required: true, message: t('common.pleaseInput') },
            { pattern: /^\d{1,10}$/, message: t('commission.eciRange') },
          ]}
        >
          <Input placeholder="E-UTRAN Cell Identifier" maxLength={10} />
        </Form.Item>
        <Form.Item
          name="frequency"
          label={t('commission.frequency')}
          rules={[{ required: true, message: t('common.pleaseInput') }]}
        >
          <InputNumber style={{ width: '100%' }} placeholder={t('commission.frequencyPlaceholder')} min={0} />
        </Form.Item>
        <Form.Item
          name="pci"
          label="PCI"
          rules={[
            { required: true, message: t('common.pleaseInput') },
            { type: 'number', min: 0, max: 503, message: t('commission.pciRange4g') },
          ]}
        >
          <InputNumber style={{ width: '100%' }} placeholder="Physical Cell ID" min={0} max={503} />
        </Form.Item>
        <Form.Item
          name="tac"
          label="TAC"
          rules={[
            { required: true, message: t('common.pleaseInput') },
            { pattern: /^\d{1,6}$/, message: t('commission.tacRange') },
          ]}
        >
          <Input placeholder="Tracking Area Code" maxLength={6} />
        </Form.Item>
        <Form.Item
          name="plmn"
          label="PLMN"
          rules={[
            { required: true, message: t('common.pleaseInput') },
            { pattern: /^\d{5,6}$/, message: t('commission.plmnRange') },
          ]}
        >
          <Input placeholder={t('commission.plmnExample')} maxLength={6} />
        </Form.Item>
        <Form.Item
          name="bandwidth"
          label={t('commission.bandwidth')}
          rules={[{ required: true, message: t('common.pleaseSelect') }]}
        >
          <Select placeholder={t('common.pleaseSelect')}>
            <Select.Option value="5">5 MHz</Select.Option>
            <Select.Option value="10">10 MHz</Select.Option>
            <Select.Option value="15">15 MHz</Select.Option>
            <Select.Option value="20">20 MHz</Select.Option>
          </Select>
        </Form.Item>
      </Card>

      {/* 核心网 IP */}
      <Card title={t('commission.coreIp')} size="small">
        <Form.Item
          name="coreIp4g"
          label={t('commission.coreIp')}
          rules={[
            { required: true, message: t('common.pleaseInput') },
            ipValidationRule,
          ]}
        >
          <Input placeholder={t('commission.mmeAddress')} />
        </Form.Item>
      </Card>
    </>
  );

  // 5G表单
  const render5GForm = () => (
    <>
      {/* WAN IP */}
      <Card title="WAN IP" size="small" style={{ marginBottom: 16 }}>
        <Form.Item
          name="wanIp5g"
          label="WAN IP"
          rules={[
            { required: true, message: t('common.pleaseInput') },
            ipValidationRule,
          ]}
        >
          <Input placeholder={t('commission.ipExample')} />
        </Form.Item>
      </Card>

      {/* 基站参数 */}
      <Card title={t('commission.stationParams')} size="small" style={{ marginBottom: 16 }}>
        <Form.Item
          name="gnbId"
          label="GNB ID"
          rules={[
            { required: true, message: t('common.pleaseInput') },
            { pattern: /^\d{1,10}$/, message: t('commission.gnbIdRange') },
          ]}
        >
          <Input placeholder="gNB Identifier" maxLength={10} />
        </Form.Item>
        <Form.Item
          name="frequency5g"
          label={t('commission.frequency')}
          rules={[{ required: true, message: t('common.pleaseInput') }]}
        >
          <InputNumber style={{ width: '100%' }} placeholder={t('commission.frequencyPlaceholder')} min={0} />
        </Form.Item>
        <Form.Item
          name="pci5g"
          label="PCI"
          rules={[
            { required: true, message: t('common.pleaseInput') },
            { type: 'number', min: 0, max: 1007, message: t('commission.pciRange5g') },
          ]}
        >
          <InputNumber style={{ width: '100%' }} placeholder="Physical Cell ID" min={0} max={1007} />
        </Form.Item>
      </Card>

      {/* 核心网 IP */}
      <Card title={t('commission.coreIp')} size="small" style={{ marginBottom: 16 }}>
        <Form.Item
          name="coreIp5g"
          label={t('commission.coreIp')}
          rules={[
            { required: true, message: t('common.pleaseInput') },
            ipValidationRule,
          ]}
        >
          <Input placeholder={t('commission.amfAddress')} />
        </Form.Item>
      </Card>

      {/* 切片配置 */}
      <Card
        title={t('commission.sliceConfig')}
        size="small"
        style={{ marginBottom: 16 }}
        extra={
          <Button type="link" size="small" icon={<PlusOutlined />} onClick={addSlice}>
            {t('commission.addSlice')}
          </Button>
        }
      >
        {slices.map((slice, index) => (
          <div
            key={slice.id}
            style={{
              padding: '12px',
              marginBottom: index < slices.length - 1 ? 12 : 0,
              background: '#fafafa',
              borderRadius: 4,
              position: 'relative',
            }}
          >
            {slices.length > 1 && (
              <Button
                type="text"
                danger
                size="small"
                icon={<DeleteOutlined />}
                onClick={() => removeSlice(slice.id)}
                style={{ position: 'absolute', top: 8, right: 8 }}
              />
            )}
            <Text strong style={{ display: 'block', marginBottom: 8 }}>
              {t('commission.slice')} {index + 1}
            </Text>
            <Form.Item label="TAC" style={{ marginBottom: 8 }}>
              <Input
                value={slice.tac}
                onChange={(e) => updateSlice(slice.id, 'tac', e.target.value)}
                placeholder={t('commission.sliceTac')}
              />
            </Form.Item>
            <Form.Item label="PLMN" style={{ marginBottom: 8 }}>
              <Input
                value={slice.plmn}
                onChange={(e) => updateSlice(slice.id, 'plmn', e.target.value)}
                placeholder={t('commission.slicePlmn')}
              />
            </Form.Item>
            <Form.Item label="ECI" style={{ marginBottom: 0 }}>
              <Input
                value={slice.eci}
                onChange={(e) => updateSlice(slice.id, 'eci', e.target.value)}
                placeholder={t('commission.sliceEci')}
              />
            </Form.Item>
          </div>
        ))}
      </Card>

      {/* 同步配置 */}
      <Card title={t('commission.sync')} size="small" style={{ marginBottom: 16 }}>
        <Form.Item label={t('commission.syncMode')}>
          <Radio.Group
            value={syncConfig.syncMode}
            onChange={(e) => setSyncConfig({ ...syncConfig, syncMode: e.target.value })}
          >
            <Radio value="gps">{t('commission.syncGps')}</Radio>
            <Radio value="ptp">{t('commission.syncPtp')}</Radio>
            <Radio value="none">{t('commission.syncNone')}</Radio>
          </Radio.Group>
        </Form.Item>
        {syncConfig.syncMode === 'ptp' && (
          <Form.Item label={t('commission.syncServer')}>
            <Input
              value={syncConfig.syncServer}
              onChange={(e) => setSyncConfig({ ...syncConfig, syncServer: e.target.value })}
              placeholder={t('commission.ptpServerPlaceholder')}
            />
          </Form.Item>
        )}
      </Card>

      {/* 共享小区配置 */}
      <Card
        title={t('commission.sharedCellConfig')}
        size="small"
        extra={
          <Button type="link" size="small" icon={<PlusOutlined />} onClick={addSharedCell}>
            {t('commission.addConfig')}
          </Button>
        }
      >
        {sharedCells.length === 0 ? (
          <Text type="secondary">{t('commission.noSharedCellConfig')}</Text>
        ) : (
          sharedCells.map((cell, index) => (
            <div
              key={cell.id}
              style={{
                display: 'flex',
                gap: 8,
                alignItems: 'center',
                marginBottom: index < sharedCells.length - 1 ? 8 : 0,
              }}
            >
              <Input
                value={cell.name}
                onChange={(e) => updateSharedCell(cell.id, 'name', e.target.value)}
                placeholder={t('commission.paramName')}
                style={{ flex: 1 }}
              />
              <Input
                value={cell.value}
                onChange={(e) => updateSharedCell(cell.id, 'value', e.target.value)}
                placeholder={t('commission.paramValue')}
                style={{ flex: 1 }}
              />
              <Button
                type="text"
                danger
                icon={<DeleteOutlined />}
                onClick={() => removeSharedCell(cell.id)}
              />
            </div>
          ))
        )}
      </Card>
    </>
  );

  // 2G GSM表单
  const render2GForm = () => (
    <>
      <Card title="IPA Unit" size="small" style={{ marginBottom: 16 }}>
        <Form.Item
          name="ipaUnit"
          label="IPA Unit"
          rules={[
            { required: true, message: t('common.pleaseInput') },
            { max: 50, message: t('commission.ipaUnitMax') },
          ]}
        >
          <Input placeholder={t('commission.ipaUnitPlaceholder')} maxLength={50} />
        </Form.Item>
      </Card>

      <Card title={t('commission.bscIp')} size="small" style={{ marginBottom: 16 }}>
        <Form.Item
          name="bscIp"
          label="BSC IP"
          rules={[
            { required: true, message: t('common.pleaseInput') },
            ipValidationRule,
          ]}
        >
          <Input placeholder={t('commission.bscAddressPlaceholder')} />
        </Form.Item>
      </Card>

      <Card title="WAN IP" size="small">
        <Form.Item
          name="wanIp2g"
          label="WAN IP"
          rules={[
            { required: true, message: t('common.pleaseInput') },
            ipValidationRule,
          ]}
        >
          <Input placeholder={t('commission.ipExample')} />
        </Form.Item>
      </Card>
    </>
  );

  // 渲染动态表单
  const renderDynamicForm = useMemo(() => {
    if (provMode === 'import') return null;

    switch (stationType) {
      case 'eNB':
        return render4GForm();
      case 'gNB':
        return render5GForm();
      case 'GSM':
        return render2GForm();
      default:
        return null;
    }
  }, [provMode, stationType, slices, syncConfig, sharedCells]);

  // 处理基站类型变更时重置表单
  const handleStationTypeChange = (value: StationType) => {
    setStationType(value);
    form.resetFields();
    // 重置5G特定状态
    if (value !== 'gNB') {
      setSlices([{ id: '1', tac: '', plmn: '', eci: '' }]);
      setSyncConfig({ syncMode: 'gps', syncServer: '' });
      setSharedCells([]);
    }
  };

  // 导入处理函数
  const handleImport = useCallback(async (file: File) => {
    console.log('Import file:', file.name);
    await new Promise((resolve) => setTimeout(resolve, 1000));
    return { success: true };
  }, []);

  const handleImportSuccess = useCallback(() => {
    message.success(t('common.success'));
    handleClose();
  }, [message, t]);

  const handleDownloadTemplate = useCallback(() => {
    message.info(t('user.downloadingTemplate'));
  }, [message, t]);

  return (
    <Drawer
      title={mode === 'edit' ? t('commission.editTask') : t('commission.addTask')}
      open={open}
      onClose={handleClose}
      width={640}
      destroyOnClose
      footer={
        <div style={{ textAlign: 'right' }}>
          <Button style={{ marginRight: 8 }} onClick={handleClose}>
            {t('common.cancel')}
          </Button>
          <Button type="primary" loading={loading} onClick={handleSubmit}>
            {mode === 'edit' ? t('common.save') : t('common.confirm')}
          </Button>
        </div>
      }
    >
      <Form form={form} layout="vertical" initialValues={initial4GValues}>
        {/* 编辑模式下不显示开站方式 */}
        {mode === 'add' && (
          <Form.Item label={t('commission.provisionMode')} required>
            <Radio.Group
              value={provMode}
              onChange={(e) => {
                setProvMode(e.target.value);
                form.resetFields();
              }}
            >
              <Radio value="manual">{t('commission.manualConfig')}</Radio>
              <Radio value="import">{t('commission.batchImport')}</Radio>
            </Radio.Group>
          </Form.Item>
        )}

        {(provMode === 'manual' || mode === 'edit') && (
          <>
            {/* 基站编码 */}
            <Form.Item
              name="stationCode"
              label={t('commission.stationCode')}
              rules={[{ required: true, message: t('common.pleaseInput') }]}
            >
              <Input placeholder={t('commission.stationCodePlaceholder')} maxLength={50} />
            </Form.Item>

            {/* 基站名称 */}
            <Form.Item
              name="stationName"
              label={t('commission.stationName')}
              rules={[{ required: true, message: t('common.pleaseInput') }]}
            >
              <Input placeholder={t('commission.stationNamePlaceholder')} maxLength={100} />
            </Form.Item>

            {/* 基站类型 - 编辑模式下只显示标签 */}
            <Form.Item label={t('commission.stationType')} required>
              {mode === 'edit' ? (
                <Text strong>{stationType === 'eNB' ? t('commission.stationType4g') : stationType === 'gNB' ? t('commission.stationType5g') : t('commission.stationType2g')}</Text>
              ) : (
                <Select
                  value={stationType}
                  onChange={handleStationTypeChange}
                  placeholder={t('common.pleaseSelect')}
                >
                  <Select.Option value="eNB">eNB (4G)</Select.Option>
                  <Select.Option value="gNB">gNB (5G)</Select.Option>
                  <Select.Option value="GSM">GSM (2G BTS)</Select.Option>
                </Select>
              )}
            </Form.Item>

            <Divider style={{ margin: '12px 0' }} />

            {/* 动态表单 */}
            {renderDynamicForm}
          </>
        )}

        {provMode === 'import' && mode === 'add' && (
          <>
            {/* 导入模式下仍需选择基站类型 */}
            <Form.Item label={t('commission.stationType')} required>
              <Select
                value={stationType}
                onChange={setStationType}
                placeholder={t('common.pleaseSelect')}
              >
                <Select.Option value="eNB">eNB (4G)</Select.Option>
                <Select.Option value="gNB">gNB (5G)</Select.Option>
                <Select.Option value="GSM">GSM (2G BTS)</Select.Option>
              </Select>
            </Form.Item>

            <Divider style={{ margin: '12px 0' }} />

            <ImportPanel
              ref={importPanelRef}
              accept=".xlsx,.xls,.csv"
              maxSizeMB={10}
              onImport={handleImport}
              onSuccess={handleImportSuccess}
              onDownloadTemplate={handleDownloadTemplate}
            />
          </>
        )}
      </Form>
    </Drawer>
  );
}
