import { Form, InputNumber, Checkbox, Select, Card, Space, Typography, theme } from 'antd';
import { useT } from '@/hooks/useT';

const { Option } = Select;

interface DeviceSettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
}

// 设置行样式
const settingRowStyle: React.CSSProperties = {
  marginBottom: 16,
};

export default function DeviceSettings({ form }: DeviceSettingsProps) {
  const t = useT();
  const { token } = theme.useToken();

  return (
    <Form form={form} layout="vertical" size="small" initialValues={{
      enbInformPeriodAdjustEnable: true,
      enbInformPeriod: 60,
      enbTimeout: 100,
      nameSettingEnable: true,
      prompt: false,
      accessContralEnable: false,
      rsrpVal0: -100,
      rsrpVal1: -80,
      uersrpVal0: -100,
      uersrpVal1: -80,
      uploadSelected: '3',
      deviceOfflineEnable: false,
      deviceOfflineSaveDay: 90,
      locationDetection: true,
      latitudeToleranceRange: 1000,
      // 设备参数同步设置（与后端 internal/provision/periodic_sync_policy.go default 对齐）
      periodicSyncEnabled: false,
      periodicSyncIntervalHours: 24,
      periodicSyncBatchSize: 200,
      periodicSyncMaxConcurrent: 10,
      periodicSyncStaggerWindowMinutes: 0,
    }}>
      {/* 设备Inform周期 —— 按"基站类 / CPE 类"分组排版（issue #238） */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.device.informPeriod')}</span>} style={{ marginBottom: 16 }}>
        {/* 基站类（覆盖 2G/4G/5G，统称"基站"，不再硬编码 eNB） */}
        <Typography.Text strong style={{ display: 'block', marginBottom: 12 }}>
          {t('system.device.informGroup.baseStation')}
        </Typography.Text>
        <div style={settingRowStyle}>
          <Space wrap>
            <Form.Item name="enbInformPeriodAdjustEnable" valuePropName="checked" noStyle>
              <Checkbox>{t('system.device.adjustEnbPrefix')}</Checkbox>
            </Form.Item>
            <Form.Item name="enbInformPeriod" noStyle>
              <InputNumber min={60} max={3600} style={{ width: 70 }} />
            </Form.Item>
            <span>{t('system.device.adjustSuffix')}</span>
          </Space>
        </div>
        <div style={settingRowStyle}>
          <Space wrap>
            <Form.Item name="enbTimeoutEnable" valuePropName="checked" noStyle>
              <Checkbox>{t('system.device.ifSystemIn')}</Checkbox>
            </Form.Item>
            <Form.Item name="enbTimeout" noStyle>
              <InputNumber min={60} max={7200} style={{ width: 70 }} />
            </Form.Item>
            <span>{t('system.device.noHeartbeatThenOffline')}</span>
          </Space>
        </div>
      </Card>

      {/* 设备名称同步 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.device.deviceNameSync')}</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Form.Item name="nameSettingEnable" valuePropName="checked" noStyle>
            <Checkbox>{t('system.device.checkAndSetDeviceName')}</Checkbox>
          </Form.Item>
        </div>
        <div style={settingRowStyle}>
          <div style={{ marginLeft: 24 }}>
            <Form.Item name="prompt" valuePropName="checked" noStyle>
              <Checkbox>{t('system.device.promptManualSync')}</Checkbox>
            </Form.Item>
          </div>
        </div>
      </Card>

      {/* 设备接入控制 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.device.accessControl')}</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space>
            <Form.Item name="accessContralEnable" valuePropName="checked" noStyle>
              <Checkbox>{t('system.device.onlyAllowMatching')}</Checkbox>
            </Form.Item>
            <a href="#">{t('common.rules')}</a>
            <span>{t('system.device.connectToOmc')}</span>
          </Space>
        </div>
      </Card>

      {/* CPE信号强度 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.device.cpeSignalStrength')}</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <span style={{ marginRight: 16 }}>{t('system.device.displaySignalByRange')}</span>
          <span style={{ marginRight: 16, display: 'inline-flex', alignItems: 'center', padding: '4px 12px', backgroundColor: token.colorErrorBg, borderRadius: 4 }}>
            <span style={{ width: 8, height: 8, borderRadius: '50%', backgroundColor: token.colorError, marginRight: 8 }} />
            {t('system.device.signal.weak')}(&lt;
            <Form.Item name="rsrpVal0" noStyle style={{ marginLeft: 4, marginRight: 4 }}>
              <InputNumber min={-150} max={0} style={{ width: 60 }} />
            </Form.Item>
          </span>
          <span style={{ marginRight: 16, display: 'inline-flex', alignItems: 'center', padding: '4px 12px', backgroundColor: token.colorWarningBg, borderRadius: 4 }}>
            <span style={{ width: 8, height: 8, borderRadius: '50%', backgroundColor: token.colorWarning, marginRight: 8 }} />
            {t('system.device.signal.normal')}&lt;
            <Form.Item name="rsrpVal1" noStyle style={{ marginLeft: 4, marginRight: 4 }}>
              <InputNumber min={-150} max={0} style={{ width: 60 }} />
            </Form.Item>
          </span>
          <span style={{ display: 'inline-flex', alignItems: 'center', padding: '4px 12px', backgroundColor: token.colorSuccessBg, borderRadius: 4 }}>
            <span style={{ width: 8, height: 8, borderRadius: '50%', backgroundColor: token.colorSuccess, marginRight: 8 }} />
            {t('system.device.signal.strong')}
          </span>
        </div>
      </Card>

      {/* UE信号强度 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.device.ueSignalStrength')}</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <span style={{ marginRight: 16 }}>{t('system.device.displaySignalByRange')}</span>
          <span style={{ marginRight: 16, display: 'inline-flex', alignItems: 'center', padding: '4px 12px', backgroundColor: token.colorErrorBg, borderRadius: 4 }}>
            <span style={{ width: 8, height: 8, borderRadius: '50%', backgroundColor: token.colorError, marginRight: 8 }} />
            {t('system.device.signal.weak')}&lt;
            <Form.Item name="uersrpVal0" noStyle style={{ marginLeft: 4, marginRight: 4 }}>
              <InputNumber min={-150} max={0} style={{ width: 60 }} />
            </Form.Item>
          </span>
          <span style={{ marginRight: 16, display: 'inline-flex', alignItems: 'center', padding: '4px 12px', backgroundColor: token.colorWarningBg, borderRadius: 4 }}>
            <span style={{ width: 8, height: 8, borderRadius: '50%', backgroundColor: token.colorWarning, marginRight: 8 }} />
            {t('system.device.signal.normal')}&lt;
            <Form.Item name="uersrpVal1" noStyle style={{ marginLeft: 4, marginRight: 4 }}>
              <InputNumber min={-150} max={0} style={{ width: 60 }} />
            </Form.Item>
          </span>
          <span style={{ display: 'inline-flex', alignItems: 'center', padding: '4px 12px', backgroundColor: token.colorSuccessBg, borderRadius: 4 }}>
            <span style={{ width: 8, height: 8, borderRadius: '50%', backgroundColor: token.colorSuccess, marginRight: 8 }} />
            {t('system.device.signal.strong')}
          </span>
        </div>
      </Card>

      {/* 基站文件上传协议 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.device.baseStationUploadProtocol')}</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Form.Item name="uploadSelected" noStyle>
            <Select style={{ width: 160 }}>
              <Option value="1">{t('common.protocol.http')}</Option>
              <Option value="2">{t('common.protocol.https')}</Option>
              <Option value="3">{t('system.device.keepBaseStationUnchanged')}</Option>
            </Select>
          </Form.Item>
        </div>
      </Card>

      {/* 回收站 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.device.recycleBin')}</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space wrap>
            <Form.Item name="deviceOfflineEnable" valuePropName="checked" noStyle>
              <Checkbox>{t('system.device.deviceDefaultConnect')}</Checkbox>
            </Form.Item>
            <Form.Item name="deviceOfflineSaveDay" noStyle>
              <InputNumber min={1} max={365} style={{ width: 60 }} />
            </Form.Item>
            <span>{t('system.device.autoAddToRecycleBin')}</span>
          </Space>
        </div>
        <div style={settingRowStyle}>
          <span style={{ color: token.colorTextTertiary }}>{t('system.device.dailyCheckOfflineTime')}</span>
        </div>
      </Card>

      {/* eNB位置移动检测 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.device.enbLocationDetection')}</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space wrap>
            <Form.Item name="locationDetection" valuePropName="checked" noStyle>
              <Checkbox>{t('system.device.ifEnbLocationExceeds')}</Checkbox>
            </Form.Item>
            <Form.Item name="latitudeToleranceRange" noStyle>
              <InputNumber min={10} max={10000} style={{ width: 70 }} />
            </Form.Item>
            <span>{t('system.device.thenEnbWillBeLocked')}</span>
          </Space>
        </div>
      </Card>

      {/* 设备参数同步设置（T-0124 周期性参数同步兜底） */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.device.periodicSync.title')}</span>}>
        <Typography.Paragraph type="secondary" style={{ marginBottom: 12, fontSize: 12 }}>
          {t('system.device.periodicSync.intro')}
        </Typography.Paragraph>
        <div style={settingRowStyle}>
          <Form.Item name="periodicSyncEnabled" valuePropName="checked" noStyle>
            <Checkbox>{t('system.device.periodicSync.enabledLabel')}</Checkbox>
          </Form.Item>
        </div>
        <div style={settingRowStyle}>
          <Space wrap>
            <span>{t('system.device.periodicSync.intervalPrefix')}</span>
            <Form.Item name="periodicSyncIntervalHours" noStyle>
              <InputNumber min={1} max={168} style={{ width: 80 }} />
            </Form.Item>
            <span>{t('system.device.periodicSync.intervalSuffix')}</span>
          </Space>
        </div>
        <div style={settingRowStyle}>
          <Space wrap>
            <span>{t('system.device.periodicSync.batchSizePrefix')}</span>
            <Form.Item name="periodicSyncBatchSize" noStyle>
              <InputNumber min={1} max={1000} style={{ width: 80 }} />
            </Form.Item>
            <span>{t('system.device.periodicSync.batchSizeSuffix')}</span>
          </Space>
        </div>
        <div style={settingRowStyle}>
          <Space wrap>
            <span>{t('system.device.periodicSync.maxConcurrentPrefix')}</span>
            <Form.Item name="periodicSyncMaxConcurrent" noStyle>
              <InputNumber min={1} max={50} style={{ width: 80 }} />
            </Form.Item>
            <span>{t('system.device.periodicSync.maxConcurrentSuffix')}</span>
          </Space>
        </div>
        <div style={settingRowStyle}>
          <Space wrap>
            <span>{t('system.device.periodicSync.staggerPrefix')}</span>
            <Form.Item name="periodicSyncStaggerWindowMinutes" noStyle>
              <InputNumber min={0} max={120} style={{ width: 80 }} />
            </Form.Item>
            <span>{t('system.device.periodicSync.staggerSuffix')}</span>
          </Space>
        </div>
      </Card>
    </Form>
  );
}
