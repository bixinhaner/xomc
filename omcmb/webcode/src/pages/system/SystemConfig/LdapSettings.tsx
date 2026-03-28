import { Form, Input, InputNumber, Switch, Checkbox, Space, Button, Card, message } from 'antd';
import { ApiOutlined } from '@ant-design/icons';
import { useT } from '@/hooks/useT';

interface LdapSettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
}

// 设置行样式
const settingRowStyle: React.CSSProperties = {
  marginBottom: 16,
};

// 子设置区域样式
const subSettingStyle: React.CSSProperties = {
  marginTop: 12,
  padding: '16px',
  backgroundColor: '#fafafa',
  borderRadius: 4,
};

// 表单项组样式
const formGroupStyle: React.CSSProperties = {
  display: 'flex',
  flexWrap: 'wrap',
  gap: 16,
  marginBottom: 12,
};

export default function LdapSettings({ form }: LdapSettingsProps) {
  const t = useT();

  // 监听LDAP开关状态
  const ldapEnable = Form.useWatch('ldapEnable', form);

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
      {/* LDAP服务设置 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>LDAP服务设置</span>}>
        {/* LDAP Enable */}
        <div style={settingRowStyle}>
          <Space>
            <span>LDAP Enable</span>
            <Form.Item name="ldapEnable" noStyle valuePropName="checked">
              <Switch size="small" />
            </Form.Item>
            <Button type="primary" size="small" onClick={handleTestLdap} disabled={!ldapEnable}>
              测试
            </Button>
          </Space>
        </div>

        {/* 服务器配置 */}
        <div style={subSettingStyle}>
          <div style={{ marginBottom: 12, fontWeight: 500, color: '#555' }}>服务器配置</div>

          <div style={formGroupStyle}>
            <Form.Item label="LDAP IP" name="ldapIp" style={{ marginBottom: 0 }}>
              <Input style={{ width: 180 }} placeholder="192.168.1.50" disabled={!ldapEnable} />
            </Form.Item>
            <Form.Item label="LDAP Port" name="ldapPort" style={{ marginBottom: 0 }}>
              <Input style={{ width: 100 }} placeholder="389" disabled={!ldapEnable} />
            </Form.Item>
            <Form.Item name="ldapSSL" valuePropName="checked" noStyle style={{ marginTop: 30 }}>
              <Checkbox disabled={!ldapEnable}>SSL/TLS</Checkbox>
            </Form.Item>
          </div>

          <div style={formGroupStyle}>
            <Form.Item label="LDAP Base" name="ldapBase" style={{ marginBottom: 0 }}>
              <Input style={{ width: 300 }} placeholder="dc=example,dc=com" disabled={!ldapEnable} />
            </Form.Item>
          </div>

          <div style={formGroupStyle}>
            <Form.Item label="LDAP User" name="ldapUser" style={{ marginBottom: 0 }}>
              <Input style={{ width: 300 }} placeholder="cn=admin,dc=example,dc=com" disabled={!ldapEnable} />
            </Form.Item>
          </div>

          <div style={formGroupStyle}>
            <Form.Item label="LDAP Password" name="ldapPwd" style={{ marginBottom: 0 }}>
              <Input.Password style={{ width: 200 }} placeholder="请输入密码" disabled={!ldapEnable} />
            </Form.Item>
          </div>
        </div>
      </Card>
    </Form>
  );
}
