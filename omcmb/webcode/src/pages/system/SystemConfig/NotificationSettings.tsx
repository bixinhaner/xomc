import { Form, Input, InputNumber, Checkbox, Button, Card, Space, Select, message } from 'antd';
import { MailOutlined, MobileOutlined } from '@ant-design/icons';
import { useT } from '@/hooks/useT';

interface NotificationSettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
}

// 设置行样式
const settingRowStyle: React.CSSProperties = {
  marginBottom: 16,
};

// 子设置区域样式
const subSettingStyle: React.CSSProperties = {
  marginTop: 16,
  padding: '16px',
  backgroundColor: '#fafafa',
  borderRadius: 4,
};

// 表单项组样式
const formGroupStyle: React.CSSProperties = {
  display: 'flex',
  flexWrap: 'wrap',
  gap: 16,
  marginBottom: 8,
};

export default function NotificationSettings({ form }: NotificationSettingsProps) {
  const t = useT();

  // 监听复选框状态
  const emEnabel = Form.useWatch('emEnabel', form);
  const smsEnable = Form.useWatch('smsEnable', form);

  const handleTestEmail = () => {
    void message.info(t('system.notification.sendingTestEmail'));
    setTimeout(() => {
      void message.success(t('system.notification.testEmailSent'));
    }, 1000);
  };

  const handleTestSms = () => {
    const phone = form.getFieldValue('smsTestPhone') as string | undefined;
    if (!phone) {
      void message.warning(t('system.notification.pleaseInputSmsTestPhone'));
      return;
    }
    void message.info(t('system.notification.sendingTestSms'));
    setTimeout(() => {
      void message.success(t('system.notification.testSmsSent'));
    }, 1000);
  };

  return (
    <Form form={form} layout="vertical" size="small" initialValues={{
      emEnabel: false,
      mailUsername: '',
      mailPassword: '',
      mailHost: '',
      mailPort: '',
      smsEnable: false,
      smsProvider: 'aliyun',
      smsApiUrl: '',
      smsAccessKey: '',
      smsAccessSecret: '',
      smsSignName: '',
      smsTemplateCode: '',
      smsTestPhone: '',
    }}>
      {/* 邮件通知服务 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.notification.emailService')}</span>}>
        <div style={settingRowStyle}>
          <Form.Item name="emEnabel" valuePropName="checked" noStyle>
            <Checkbox>{t('system.notification.enableEmailServer')}</Checkbox>
          </Form.Item>
        </div>

        <div style={subSettingStyle}>
          <div style={{ marginBottom: 12, fontWeight: 500, color: '#555' }}>{t('system.notification.mailServerConfig')}</div>

          <div style={formGroupStyle}>
            <Form.Item label={t('system.notification.email')} name="mailUsername" style={{ marginBottom: 0 }}>
              <Input style={{ width: 280 }} placeholder="noreply@example.com" prefix={<MailOutlined />} disabled={!emEnabel} />
            </Form.Item>
            <Form.Item label={t('common.password')} name="mailPassword" style={{ marginBottom: 0 }}>
              <Input.Password style={{ width: 180 }} placeholder={t('system.notification.pleaseInputEmailPassword')} maxLength={50} disabled={!emEnabel} />
            </Form.Item>
          </div>

          <div style={formGroupStyle}>
            <Form.Item label={t('system.notification.smtpServer')} name="mailHost" style={{ marginBottom: 0 }}>
              <Input style={{ width: 280 }} placeholder="smtp.example.com" disabled={!emEnabel} />
            </Form.Item>
            <Space>
              <Form.Item label={t('common.port')} name="mailPort" style={{ marginBottom: 0 }}>
                <InputNumber min={1} max={65535} style={{ width: 100 }} disabled={!emEnabel} />
              </Form.Item>
              <Form.Item style={{ marginBottom: 0, marginTop: 24 }}>
                <Button type="primary" size="small" onClick={handleTestEmail} disabled={!emEnabel}>
                  {t('common.test')}
                </Button>
              </Form.Item>
            </Space>
          </div>
        </div>
      </Card>

      {/* 短信通知服务 */}
      <Card
        size="small"
        title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.notification.smsService')}</span>}
        style={{ marginTop: 16 }}
      >
        <div style={settingRowStyle}>
          <Form.Item name="smsEnable" valuePropName="checked" noStyle>
            <Checkbox>{t('system.notification.enableSmsServer')}</Checkbox>
          </Form.Item>
        </div>

        <div style={subSettingStyle}>
          <div style={{ marginBottom: 12, fontWeight: 500, color: '#555' }}>{t('system.notification.smsServerConfig')}</div>

          <div style={formGroupStyle}>
            <Form.Item label={t('system.notification.smsProvider')} name="smsProvider" style={{ marginBottom: 0 }}>
              <Select style={{ width: 160 }} disabled={!smsEnable}>
                <Select.Option value="aliyun">{t('sysconfig.notification.provider.aliyun')}</Select.Option>
                <Select.Option value="tencent">{t('sysconfig.notification.provider.tencent')}</Select.Option>
                <Select.Option value="huawei">{t('sysconfig.notification.provider.huawei')}</Select.Option>
                <Select.Option value="custom">{t('sysconfig.notification.provider.custom')}</Select.Option>
              </Select>
            </Form.Item>
            <Form.Item label={t('system.notification.smsApiUrl')} name="smsApiUrl" style={{ marginBottom: 0 }}>
              <Input
                style={{ width: 320 }}
                placeholder="https://dysmsapi.aliyuncs.com"
                prefix={<MobileOutlined />}
                disabled={!smsEnable}
              />
            </Form.Item>
          </div>

          <div style={formGroupStyle}>
            <Form.Item label={t('system.notification.smsAccessKey')} name="smsAccessKey" style={{ marginBottom: 0 }}>
              <Input
                style={{ width: 280 }}
                placeholder={t('system.notification.pleaseInputSmsAccessKey')}
                disabled={!smsEnable}
              />
            </Form.Item>
            <Form.Item label={t('system.notification.smsAccessSecret')} name="smsAccessSecret" style={{ marginBottom: 0 }}>
              <Input.Password style={{ width: 240 }} maxLength={128} disabled={!smsEnable} />
            </Form.Item>
          </div>

          <div style={formGroupStyle}>
            <Form.Item label={t('system.notification.smsSignName')} name="smsSignName" style={{ marginBottom: 0 }}>
              <Input style={{ width: 200 }} placeholder={t('sysconfig.notification.omcAlertPlaceholder')} disabled={!smsEnable} />
            </Form.Item>
            <Form.Item label={t('system.notification.smsTemplateCode')} name="smsTemplateCode" style={{ marginBottom: 0 }}>
              <Input style={{ width: 220 }} placeholder="SMS_123456789" disabled={!smsEnable} />
            </Form.Item>
          </div>

          <div style={formGroupStyle}>
            <Form.Item label={t('system.notification.smsTestPhone')} name="smsTestPhone" style={{ marginBottom: 0 }}>
              <Input
                style={{ width: 180 }}
                placeholder="13800138000"
                maxLength={11}
                disabled={!smsEnable}
              />
            </Form.Item>
            <Form.Item style={{ marginBottom: 0, marginTop: 24 }}>
              <Button type="primary" size="small" onClick={handleTestSms} disabled={!smsEnable}>
                {t('common.test')}
              </Button>
            </Form.Item>
          </div>
        </div>
      </Card>
    </Form>
  );
}
