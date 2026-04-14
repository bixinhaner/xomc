import { useState, useEffect } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { Form, Input, Button, Checkbox, message } from 'antd';
import { UserOutlined, LockOutlined } from '@ant-design/icons';
import { useUserStore } from '@/store/userStore';
import { useT } from '@/hooks/useT';
import { useMock } from '@/services/apiSwitch';
import { authApi } from '@/services/api/authApi';
import type { User } from '@/types/system';
import type { AxiosError } from 'axios';
import styles from './Login.module.css';

interface LoginFormValues {
  username: string;
  password: string;
  remember: boolean;
}

const REMEMBER_KEY = 'omc-remember-credentials';

function getRemembered(): { username: string; password: string } | null {
  try {
    const raw = localStorage.getItem(REMEMBER_KEY);
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

function setRemembered(username: string, password: string) {
  localStorage.setItem(REMEMBER_KEY, JSON.stringify({ username, password }));
}

function clearRemembered() {
  localStorage.removeItem(REMEMBER_KEY);
}

export default function LoginPage() {
  const t = useT();
  const [loading, setLoading] = useState(false);
  const [form] = Form.useForm<LoginFormValues>();
  const navigate = useNavigate();
  const location = useLocation();
  const { login, setTokenPair } = useUserStore();

  const from = (location.state as { from?: { pathname: string } })?.from?.pathname || '/dashboard';

  // 页面加载时填充记住的凭据
  useEffect(() => {
    const saved = getRemembered();
    if (saved) {
      form.setFieldsValue({
        username: saved.username,
        password: saved.password,
        remember: true,
      });
    }
  }, [form]);

  const handleMockLogin = async (values: LoginFormValues) => {
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

    const tokenPair = {
      access_token: `mock-access-token-${values.username}-${Date.now()}`,
      refresh_token: `mock-refresh-token-${values.username}-${Date.now()}`,
      expires_at: new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString(),
      token_type: 'Bearer',
    };

    setTokenPair(tokenPair);

    login(mockUser);
    message.success(t('login.success'));
    navigate(from, { replace: true });
  };

  const handleRealLogin = async (values: LoginFormValues) => {
    // Step 1: Authenticate and get token pair
    const tokenPair = await authApi.login(values.username, values.password);
    setTokenPair(tokenPair);

    // Step 2: Fetch current user info
    const user = await authApi.getMe();
    login(user);

    message.success(t('login.success'));
    navigate(from, { replace: true });
  };

  const handleSubmit = async (values: LoginFormValues) => {
    setLoading(true);
    try {
      // 记住/清除密码
      if (values.remember) {
        setRemembered(values.username, values.password);
      } else {
        clearRemembered();
      }

      if (useMock) {
        await handleMockLogin(values);
      } else {
        await handleRealLogin(values);
      }
    } catch (err) {
      const axiosErr = err as AxiosError & { userMessage?: string };
      const msg = axiosErr.userMessage || t('login.failed');
      message.error(msg);
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
