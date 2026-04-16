import { Form, Input, InputNumber, Checkbox, Button, Card, Space, message } from 'antd';
import { MailOutlined } from '@ant-design/icons';
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

  const handleTestEmail = () => {
    void message.info(t('system.notification.sendingTestEmail'));
    setTimeout(() => {
      void message.success(t('system.notification.testEmailSent'));
    }, 1000);
  };

  return (
    <Form form={form} layout="vertical" size="small" initialValues={{
      emEnabel: false,
      mailUsername: '',
      mailPassword: '',
      mailHost: '',
      mailPort: '',
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
    </Form>
  );
}
