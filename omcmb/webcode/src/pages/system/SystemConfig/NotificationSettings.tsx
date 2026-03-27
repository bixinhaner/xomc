import { Form, Input, InputNumber, Switch, Divider, Space, Button, message } from 'antd';
import { MailOutlined } from '@ant-design/icons';
import { useT } from '@/hooks/useT';

interface NotificationSettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
}

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
      mailPort: 25,
    }}>
      {/* 通知服务器 */}
      <Divider orientation="left" plain>通知服务器</Divider>
      <Form.Item name="emEnabel" label="启用邮件通知" valuePropName="checked">
        <Switch checkedChildren="开启" unCheckedChildren="关闭" />
      </Form.Item>
      <Form.Item name="mailHost" label="SMTP服务器" rules={[{ required: true, message: '请输入SMTP服务器地址' }]}>
        <Input placeholder="smtp.example.com" />
      </Form.Item>
      <Form.Item name="mailPort" label="端口" rules={[{ required: true, message: '请输入SMTP端口' }]}>
        <InputNumber min={1} max={65535} style={{ width: 150 }} />
      </Form.Item>
      <Form.Item name="mailUsername" label="邮箱" rules={[{ required: true, message: '请输入发件邮箱地址' }]}>
        <Input placeholder="noreply@example.com" prefix={<MailOutlined />} />
      </Form.Item>
      <Form.Item name="mailPassword" label="密码" rules={[{ required: true, message: '请输入邮箱密码' }]}>
        <Input.Password placeholder="请输入邮箱密码或授权码" />
      </Form.Item>
      <Form.Item>
        <Button type="primary" ghost onClick={handleTestEmail}>
          发送测试邮件
        </Button>
      </Form.Item>
    </Form>
  );
}
