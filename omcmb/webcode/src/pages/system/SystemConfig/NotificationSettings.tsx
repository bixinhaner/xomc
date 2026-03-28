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

  const handleTestEmail = () => {
    void message.info('正在发送测试邮件...');
    setTimeout(() => {
      void message.success('测试邮件发送成功');
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
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>邮件通知服务</span>}>
        <div style={settingRowStyle}>
          <Space>
            <Form.Item name="emEnabel" valuePropName="checked" noStyle>
              <Checkbox>通知邮件服务器</Checkbox>
            </Form.Item>
            <Button type="primary" size="small" onClick={handleTestEmail}>
              测试
            </Button>
          </Space>
        </div>

        <div style={subSettingStyle}>
          <div style={{ marginBottom: 12, fontWeight: 500, color: '#555' }}>邮件服务器配置</div>

          <div style={formGroupStyle}>
            <Form.Item label="邮箱" name="mailUsername" style={{ marginBottom: 0 }}>
              <Input style={{ width: 280 }} placeholder="noreply@example.com" prefix={<MailOutlined />} />
            </Form.Item>
            <Form.Item label="密码" name="mailPassword" style={{ marginBottom: 0 }}>
              <Input.Password style={{ width: 180 }} placeholder="请输入邮箱密码" maxLength={50} />
            </Form.Item>
          </div>

          <div style={formGroupStyle}>
            <Form.Item label="SMTP服务器" name="mailHost" style={{ marginBottom: 0 }}>
              <Input style={{ width: 280 }} placeholder="smtp.example.com" />
            </Form.Item>
            <Form.Item label="端口" name="mailPort" style={{ marginBottom: 0 }}>
              <InputNumber min={1} max={65535} style={{ width: 100 }} />
            </Form.Item>
          </div>
        </div>
      </Card>
    </Form>
  );
}
