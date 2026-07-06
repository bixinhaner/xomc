import { Form, InputNumber, Checkbox, Radio, Card, Space, Typography, theme } from 'antd';
import { useT } from '@/hooks/useT';

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
      nameSyncMode: 'prompt',
      deviceOfflineEnable: false,
      deviceOfflineSaveDay: 90,
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

      {/* 设备名称同步 —— 四选一策略（页面选项与存储值 nameSyncMode 一一对应） */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.device.deviceNameSync')}</span>} style={{ marginBottom: 16 }}>
        <Typography.Text type="secondary" style={{ display: 'block', marginBottom: 12 }}>
          {t('system.device.nameSync.desc')}
        </Typography.Text>
        <Form.Item name="nameSyncMode" noStyle>
          <Radio.Group>
            <Space direction="vertical" size={12}>
              <Radio value="auto_lmt_to_omc">
                {t('system.device.nameSync.mode.autoLmtToOmc')}
                <Typography.Text type="secondary" style={{ display: 'block', fontSize: 12, marginTop: 2 }}>
                  {t('system.device.nameSync.mode.autoLmtToOmc.hint')}
                </Typography.Text>
              </Radio>
              <Radio value="auto_omc_to_lmt">
                {t('system.device.nameSync.mode.autoOmcToLmt')}
                <Typography.Text type="secondary" style={{ display: 'block', fontSize: 12, marginTop: 2 }}>
                  {t('system.device.nameSync.mode.autoOmcToLmt.hint')}
                </Typography.Text>
              </Radio>
              <Radio value="prompt">
                {t('system.device.nameSync.mode.prompt')}
                <Typography.Text type="secondary" style={{ display: 'block', fontSize: 12, marginTop: 2 }}>
                  {t('system.device.nameSync.mode.prompt.hint')}
                </Typography.Text>
              </Radio>
            </Space>
          </Radio.Group>
        </Form.Item>
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
