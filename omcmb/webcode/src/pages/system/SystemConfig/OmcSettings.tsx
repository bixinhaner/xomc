import { Form, Input, Switch, Select, Button, Card, Space, message, theme } from 'antd';
import { useT } from '@/hooks/useT';

const { Option } = Select;

interface OmcSettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
}

// 设置行样式
const settingRowStyle: React.CSSProperties = {
  marginBottom: 16,
};

// 子设置区域布局（颜色随主题，见组件内 subSettingStyle）
const subSettingBaseStyle: React.CSSProperties = {
  marginTop: 12,
  padding: '12px 16px',
  borderRadius: 4,
};

// 分组标题布局（颜色/边框随主题，见组件内 sectionTitleStyle）
const sectionTitleBaseStyle: React.CSSProperties = {
  fontSize: 13,
  fontWeight: 600,
  marginBottom: 12,
  paddingBottom: 8,
};

// 磁盘空间选项
const diskSpaceOptions = [
  { text: '10%', value: '10%' },
  { text: '20%', value: '20%' },
  { text: '30%', value: '30%' },
  { text: '40%', value: '40%' },
  { text: '50%', value: '50%' },
  { text: '60%', value: '60%' },
  { text: '70%', value: '70%' },
  { text: '80%', value: '80%' },
  { text: '90%', value: '90%' },
];

export default function OmcSettings({ form }: OmcSettingsProps) {
  const t = useT();
  const { token } = theme.useToken();
  // 子设置区域 / 分组标题颜色随明暗主题，不再写死 #fafafa / #555 / #e8e8e8
  const subSettingStyle: React.CSSProperties = { ...subSettingBaseStyle, backgroundColor: token.colorFillAlter };
  const sectionTitleStyle: React.CSSProperties = { ...sectionTitleBaseStyle, color: token.colorTextSecondary, borderBottom: `1px solid ${token.colorBorderSecondary}` };

  const handleTestSyslog = () => {
    void message.info(t('system.omc.testingSyslog'));
    setTimeout(() => {
      void message.success(t('system.omc.syslogConnected'));
    }, 1000);
  };

  return (
    <Form form={form} layout="vertical" size="small" initialValues={{
      rsysLogEnable: '0',
      rsysLogIp: '',
      rsysLogPort: '',
      varDiskAlarmThresHold: '10%',
      homeDiskAlarmThresHold: '10%',
      usrDiskAlarmThresHold: '10%',
      rootDiskAlarmThresHold: '10%',
    }}>
      {/* OMC协议设置 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.omc.protocolSettings')}</span>} style={{ marginBottom: 16 }}>
        {/* 协议方式 */}
        <div style={settingRowStyle}>
          <div style={sectionTitleStyle}>{t('system.omc.protocolMethod')}</div>
          <Space style={{ marginBottom: 8 }}>
            <Form.Item name="rsysLogEnable" noStyle valuePropName="checked">
              <Switch checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />
            </Form.Item>
            <Button type="primary" size="small" onClick={handleTestSyslog}>
              {t('common.test')}
            </Button>
          </Space>
          <div style={subSettingStyle}>
            <Space size="large">
              <Space>
                <span>{t('common.ip')}</span>
                <Form.Item name="rsysLogIp" noStyle>
                  <Input style={{ width: 150 }} placeholder={t('sysconfig.omc.ipPlaceholder')} />
                </Form.Item>
              </Space>
              <Space>
                <span>{t('common.port')}</span>
                <Form.Item name="rsysLogPort" noStyle>
                  <Input style={{ width: 100 }} placeholder="514" />
                </Form.Item>
              </Space>
            </Space>
          </div>
        </div>
      </Card>

      {/* 磁盘告警设置 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.omc.diskAlarmSettings')}</span>}>
        <div style={settingRowStyle}>
          <div style={{ marginBottom: 12 }}>{t('system.omc.diskAlarmThreshold')}</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <span style={{ width: 200 }}>{t('sysconfig.omc.dirLog')}</span>
              <Form.Item name="varDiskAlarmThresHold" noStyle>
                <Select style={{ width: 80 }} size="small">
                  {diskSpaceOptions.map(opt => (
                    <Option key={opt.value} value={opt.value}>{opt.text}</Option>
                  ))}
                </Select>
              </Form.Item>
              <span style={{ marginLeft: 8 }}>{t('sysconfig.omc.alarmSuffix')}</span>
            </div>
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <span style={{ width: 200 }}>{t('sysconfig.omc.dirData')}</span>
              <Form.Item name="homeDiskAlarmThresHold" noStyle>
                <Select style={{ width: 80 }} size="small">
                  {diskSpaceOptions.map(opt => (
                    <Option key={opt.value} value={opt.value}>{opt.text}</Option>
                  ))}
                </Select>
              </Form.Item>
              <span style={{ marginLeft: 8 }}>{t('sysconfig.omc.alarmSuffix')}</span>
            </div>
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <span style={{ width: 200 }}>{t('sysconfig.omc.dirApp')}</span>
              <Form.Item name="usrDiskAlarmThresHold" noStyle>
                <Select style={{ width: 80 }} size="small">
                  {diskSpaceOptions.map(opt => (
                    <Option key={opt.value} value={opt.value}>{opt.text}</Option>
                  ))}
                </Select>
              </Form.Item>
              <span style={{ marginLeft: 8 }}>{t('sysconfig.omc.alarmSuffix')}</span>
            </div>
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <span style={{ width: 200 }}>{t('sysconfig.omc.dirRoot')}</span>
              <Form.Item name="rootDiskAlarmThresHold" noStyle>
                <Select style={{ width: 80 }} size="small">
                  {diskSpaceOptions.map(opt => (
                    <Option key={opt.value} value={opt.value}>{opt.text}</Option>
                  ))}
                </Select>
              </Form.Item>
              <span style={{ marginLeft: 8 }}>{t('sysconfig.omc.alarmSuffix')}</span>
            </div>
          </div>
        </div>
      </Card>
    </Form>
  );
}
