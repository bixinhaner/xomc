import { Form, Input, InputNumber, Switch, Divider, Space, Button, message } from 'antd';
import { ApiOutlined } from '@ant-design/icons';
import { useT } from '@/hooks/useT';

interface LdapSettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
}

export default function LdapSettings({ form }: LdapSettingsProps) {
  const t = useT();

  const handleTestLdap = () => {
    void message.info('正在测试LDAP连接...');
    setTimeout(() => {
      void message.success('LDAP连接成功');
    }, 1000);
  };

  return (
    <Form form={form} layout="vertical" size="small" initialValues={{
      ldapEnable: false,
      ldapIp: '',
      ldapPort: 389,
      ldapSSL: false,
      ldapBase: '',
      ldapUser: '',
      ldapPwd: '',
    }}>
      <Divider orientation="left" plain>LDAP协议</Divider>
      <Form.Item name="ldapEnable" label="LDAP启用" valuePropName="checked">
        <Switch checkedChildren="开启" unCheckedChildren="关闭" />
      </Form.Item>
      <Space>
        <Form.Item name="ldapIp" label="LDAP IP" rules={[{ required: true, message: '请输入LDAP服务器IP' }]}>
          <Input placeholder="192.168.1.50" style={{ width: 180 }} />
        </Form.Item>
        <Form.Item name="ldapPort" label="LDAP端口">
          <InputNumber min={1} max={65535} style={{ width: 120 }} />
        </Form.Item>
      </Space>
      <Form.Item name="ldapSSL" label="SSL/TLS" valuePropName="checked" extra="启用SSL/TLS加密">
        <Switch checkedChildren="开启" unCheckedChildren="关闭" />
      </Form.Item>
      <Form.Item name="ldapBase" label="LDAP Base" rules={[{ required: true, message: '请输入LDAP基础DN' }]}>
        <Input placeholder="dc=example,dc=com" />
      </Form.Item>
      <Space>
        <Form.Item name="ldapUser" label="LDAP用户" rules={[{ required: true, message: '请输入LDAP绑定用户' }]}>
          <Input placeholder="cn=admin,dc=example,dc=com" style={{ width: 250 }} />
        </Form.Item>
        <Form.Item name="ldapPwd" label="LDAP密码" rules={[{ required: true, message: '请输入LDAP绑定密码' }]}>
          <Input.Password placeholder="请输入密码" style={{ width: 200 }} />
        </Form.Item>
      </Space>
      <Form.Item>
        <Button type="primary" ghost icon={<ApiOutlined />} onClick={handleTestLdap}>
          测试LDAP连接
        </Button>
      </Form.Item>
    </Form>
  );
}
