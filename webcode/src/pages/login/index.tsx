import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Form, Input, Button, Checkbox, message } from 'antd';
import { UserOutlined, LockOutlined } from '@ant-design/icons';
import { useUserStore } from '@/store/userStore';
import { useT } from '@/hooks/useT';
import type { User } from '@/types/system';
import styles from './Login.module.css';

interface LoginFormValues {
  username: string;
  password: string;
  remember: boolean;
}

export default function LoginPage() {
  const t = useT();
  const [loading, setLoading] = useState(false);
  const [form] = Form.useForm<LoginFormValues>();
  const navigate = useNavigate();
  const login = useUserStore((s) => s.login);

  const handleSubmit = async (values: LoginFormValues) => {
    setLoading(true);
    try {
      // Simulate async login request
      await new Promise<void>((resolve) => setTimeout(resolve, 600));

      if (values.password !== 'admin123') {
        message.error(t('login.failed'));
        return;
      }

      const mockUser: User = {
        id: '1',
        username: values.username,
        displayName: values.username === 'admin' ? 'Admin' : values.username,
        email: `${values.username}@omc.example.com`,
        phone: '18800000000',
        role: 'admin',
        status: 'active',
        lastLoginTime: new Date().toISOString(),
        createTime: '2024-01-01T00:00:00.000Z',
      };

      login(mockUser);
      message.success(t('login.success'));
      navigate('/dashboard', { replace: true });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className={styles.container}>
      <div className={styles.card}>
        {/* Logo / branding area */}
        <div className={styles.logoArea}>
          <div className={styles.logoIcon}>
            <span role="img" aria-label="network">🌐</span>
          </div>
          <h1 className={styles.title}>{t('login.title')}</h1>
          <p className={styles.subtitle}>Unified Network Management System</p>
        </div>

        {/* Login form */}
        <div className={styles.formArea}>
          <Form
            form={form}
            name="login"
            initialValues={{ remember: true }}
            onFinish={handleSubmit}
            size="large"
            autoComplete="off"
          >
            <Form.Item
              name="username"
              rules={[{ required: true, message: t('login.usernameTip') }]}
            >
              <Input
                prefix={<UserOutlined style={{ color: 'var(--login-input-icon)' }} />}
                placeholder={t('login.usernameTip')}
                autoComplete="username"
              />
            </Form.Item>

            <Form.Item
              name="password"
              rules={[{ required: true, message: t('login.passwordTip') }]}
            >
              <Input.Password
                prefix={<LockOutlined style={{ color: 'var(--login-input-icon)' }} />}
                placeholder={t('login.passwordTip')}
                autoComplete="current-password"
              />
            </Form.Item>

            <Form.Item>
              <div className={styles.rememberRow}>
                <Form.Item name="remember" valuePropName="checked" noStyle>
                  <Checkbox>{t('login.rememberMe')}</Checkbox>
                </Form.Item>
              </div>
            </Form.Item>

            <Form.Item>
              <Button
                type="primary"
                htmlType="submit"
                className={styles.loginButton}
                loading={loading}
                size="large"
                block
              >
                {t('login.submit')}
              </Button>
            </Form.Item>
          </Form>
        </div>

        {/* Footer */}
        <div className={styles.footer}>
          © 2024 OMC Network Management System
        </div>
      </div>
    </div>
  );
}
