import { useState, useEffect } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { App, Form, Input, Button, Checkbox } from 'antd';
import { UserOutlined, LockOutlined } from '@ant-design/icons';
import { useUserStore } from '@core/store/userStore';
import { useT } from '@/hooks/useT';
import { useMock } from '@core/services/apiSwitch';
import { authApi } from '@core/services/api/authApi';
import type { User } from '@core/types/system';
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
  // T-0117: antd v5 静态 message 在某些场景下脱 ConfigProvider/App 上下文丢失（同 T-0097 修复模式）。
  // 切到 App.useApp().message scoped 实例后，encryptPassword 抛错的 toast 才能稳定显示给用户。
  const { message } = App.useApp();
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
      // T-0119: error 显示优先级修复（T-0117 follow-up）
      //   1. axios userMessage — 拦截器从后端 envelope 解出的友好业务文本（401/403/...）
      //   2. client-side Error.message — passwordCipher 等前端抛的中文诊断（如 T-0117
      //      "当前访问非安全上下文..."），仅当 err 不是 axios error 时使用，避免
      //      "Request failed with status code 401" 渗透到 toast
      //   3. i18n fallback 'login.failed'
      const axiosErr = err as AxiosError & { userMessage?: string };
      const isAxiosError = !!axiosErr.response;
      const clientErrMsg =
        !isAxiosError && err instanceof Error && err.message
          ? err.message
          : undefined;
      const msg = axiosErr.userMessage || clientErrMsg || t('login.failed');
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
