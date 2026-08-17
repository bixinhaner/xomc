import { useState, useEffect, useCallback } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { App, Form, Input, Button, Checkbox, Modal, Spin } from 'antd';
import { UserOutlined, LockOutlined, SafetyCertificateOutlined, ReloadOutlined } from '@ant-design/icons';
import { getLockedSession, useUserStore } from '@core/store/userStore';
import { useT } from '@/hooks/useT';
import { useMock } from '@core/services/apiSwitch';
import { authApi } from '@core/services/api/authApi';
import type { CaptchaChallenge, CaptchaCredentials } from '@core/services/api/authApi';
import { usePublicSecuritySettings } from '@core/hooks/api/useSecuritySettings';
import { usePublicOmcName, resolveOmcName } from '@core/hooks/api/useOmcName';
import { getI18nKeyByBizCode } from '@core/i18n/bizCodeMessages';
import type { User } from '@core/types/system';
import type { AxiosError } from 'axios';
import { BrowserPasswordInput } from './BrowserPasswordInput';
import styles from './Login.module.css';

interface LoginFormValues {
  username: string;
  password: string;
  remember: boolean;
  captcha?: string; // Issue #687: 验证码输入
}

const REMEMBER_KEY = 'omc-remember-credentials';

// 不能作为登录后"返回目的地"的路径：错误页 / 登录页本身。
// 例：PrivateRoute 在 /403 上发现未登录 → 跳 /login 并把 from=/403 塞进 state；
// 若不过滤，登录成功后会回到 /403，用户体感"admin 登录后被踢到 403"。
const FROM_PATH_BLOCKLIST = new Set<string>(['/403', '/404', '/login']);

// 安全（#3）：只记住用户名，**绝不**把明文密码写入 localStorage。
// localStorage 任何 XSS 脚本可读，持久化明文密码 = 一次低级 XSS 即泄露管理员凭证。
// 旧版本可能写过 {username, password}，这里只读 username，旧 password 字段被忽略；
// 用户下次勾选记住时会以 username-only 覆盖旧值。
function getRememberedUsername(): string | null {
  try {
    const raw = localStorage.getItem(REMEMBER_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as { username?: unknown };
    return typeof parsed.username === 'string' ? parsed.username : null;
  } catch {
    return null;
  }
}

function setRememberedUsername(username: string) {
  localStorage.setItem(REMEMBER_KEY, JSON.stringify({ username }));
}

function clearRemembered() {
  localStorage.removeItem(REMEMBER_KEY);
}

export default function LoginPage() {
  const t = useT();
  // antd v5 静态 message 在某些场景下脱 ConfigProvider/App 上下文丢失，
  // 切到 App.useApp().message scoped 实例后 toast 才能稳定显示给用户。
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [form] = Form.useForm<LoginFormValues>();
  const navigate = useNavigate();
  const location = useLocation();
  const login = useUserStore((s) => s.login);
  const setTokenPair = useUserStore((s) => s.setTokenPair);
  const setMustChangePassword = useUserStore((s) => s.setMustChangePassword);
  const [lockedSession] = useState(getLockedSession);
  const isUnlock = lockedSession !== null;

  // Issue #687: 图形验证码状态
  const [captchaRequired, setCaptchaRequired] = useState(false);
  const [captchaData, setCaptchaData] = useState<CaptchaChallenge | null>(null);
  const [captchaLoading, setCaptchaLoading] = useState(false);
  const [captchaError, setCaptchaError] = useState(false);

  // PrivateRoute 把未登录用户从任意路径（含 /403 错误页）弹到 /login 时会把
  // 原 location 塞进 state.from。错误页不是合法的登录返回目的地，直接降级到
  // /dashboard，避免"admin 登录后又被踢回 /403"的体感 bug。
  const fromPath = (location.state as { from?: { pathname: string } })?.from?.pathname;
  const from = lockedSession?.returnPath
    ?? (fromPath && !FROM_PATH_BLOCKLIST.has(fromPath) ? fromPath : '/dashboard');
  const destinationForUser = (user: User): string =>
    lockedSession && lockedSession.userId !== user.id ? '/dashboard' : from;

  // P2-⑧ 浏览器记密：拉公开 security 配置。开启时不渲染原生 password 字段，
  // 避免 Chromium 忽略 autocomplete 后继续弹出保存密码提示。
  const { settings: publicSettings } = usePublicSecuritySettings();
  // 登录页大标题跟随「OMC 名称」配置（走免登录公开通道，登录前可读）；空回退 login.title。
  const { omcName } = usePublicOmcName();
  const loginTitle = resolveOmcName(omcName, t('login.title'));
  const usernameAutocomplete = publicSettings?.preventBrowserAutofill ? 'off' : 'username';

  // 锁屏模式固定为原用户，只允许输入密码解锁；普通登录仅回填记住的用户名。
  useEffect(() => {
    if (lockedSession) {
      form.setFieldsValue({
        username: lockedSession.username,
        remember: false,
      });
      return;
    }
    const username = getRememberedUsername();
    if (username) {
      form.setFieldsValue({
        username,
        remember: true,
      });
    }
  }, [form, lockedSession]);

  // Issue #687: 加载验证码图片
  const loadCaptcha = useCallback(async () => {
    setCaptchaLoading(true);
    setCaptchaError(false);
    try {
      const data = await authApi.getCaptcha();
      setCaptchaData(data);
      form.setFieldValue('captcha', ''); // 清空旧输入
    } catch {
      setCaptchaError(true);
      setCaptchaData(null);
    } finally {
      setCaptchaLoading(false);
    }
  }, [form]);

  // 验证码状态变为“需要”时自动拉取
  useEffect(() => {
    if (captchaRequired && !captchaData && !captchaLoading) {
      loadCaptcha();
    }
  }, [captchaRequired, captchaData, captchaLoading, loadCaptcha]);

  const handleMockLogin = async (values: LoginFormValues) => {
    await new Promise<void>((resolve) => setTimeout(resolve, 600));

    if (values.password !== 'admin123') {
      message.error(t('login.failed'));
      return;
    }

    const username = lockedSession?.username ?? values.username;
    const mockUser: User = {
      id: '1',
      username,
      displayName: username === 'admin' ? 'Admin' : username,
      email: `${username}@omc.example.com`,
      phone: '18800000000',
      role: 'admin',
      isSuperAdmin: username === 'superadmin',
      status: 'active',
      lastLoginTime: new Date().toISOString(),
      createTime: '2024-01-01T00:00:00.000Z',
    };

    const tokenPair = {
      access_token: `mock-access-token-${username}-${Date.now()}`,
      refresh_token: `mock-refresh-token-${username}-${Date.now()}`,
      expires_at: new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString(),
      token_type: 'Bearer',
    };

    setTokenPair(tokenPair);

    login(mockUser);
    message.success(t(isUnlock ? 'login.unlockSuccess' : 'login.success'));
    navigate(destinationForUser(mockUser), { replace: true });
  };

  const handleRealLogin = async (values: LoginFormValues) => {
    // Issue #687: 若验证码已触发，附带 captcha 参数
    let captchaCreds: CaptchaCredentials | undefined;
    if (captchaRequired && captchaData && values.captcha) {
      captchaCreds = {
        captchaId: captchaData.captchaId,
        captchaAnswer: values.captcha,
      };
    }

    // Step 1: Authenticate and get token pair
    const username = lockedSession?.username ?? values.username;
    const tokenPair = await authApi.login(username, values.password, captchaCreds);
    setTokenPair(tokenPair);

    // 登录成功后清除验证码状态
    setCaptchaRequired(false);
    setCaptchaData(null);

    // Step 2: Fetch current user info
    const user = await authApi.getMe();
    const destination = destinationForUser(user);
    login(user);

    // P2-⑪ 登录提示：后端响应附 login_notify_msg → 弹 Modal（用户确认后才进首页）
    if (tokenPair.login_notify_msg) {
      Modal.info({
        title: t('login.notifyTitle'),
        content: tokenPair.login_notify_msg,
        okText: t('common.confirm'),
        onOk: () => navigate(destination, { replace: true }),
      });
      message.success(t(isUnlock ? 'login.unlockSuccess' : 'login.success'));
      return;
    }

    // P1-④ 密码即将过期 — 仅提示不阻塞
    if (typeof tokenPair.password_expires_in_days === 'number') {
      message.warning(
        t('login.passwordExpiringSoon', { days: tokenPair.password_expires_in_days }),
        6,
      );
    }
    // Issue #649：必须改密 — 后端 must_change_password=true 时用阻塞 Modal 拦住，
    // 用户必须点确认；Modal 关闭后进入首页时 UserDropdown 会读 store.mustChangePassword
    // 自动弹出改密 Modal（不可关闭，必须改完才能用系统）。
    // 旧实现 message.warning + navigate('/change-password') 是死路由 + toast 一闪
    // 即逝，等于把硬规则降级成软提示。
    if (tokenPair.must_change_password) {
      setMustChangePassword(true);
      Modal.warning({
        title: t('login.mustChangePassword.title'),
        content: t('login.mustChangePassword.content'),
        okText: t('login.mustChangePassword.confirm'),
        closable: false,
        mask: { closable: false },
        keyboard: false,
        onOk: () => navigate(destination, { replace: true }),
      });
      return;
    }

    message.success(t(isUnlock ? 'login.unlockSuccess' : 'login.success'));
    navigate(destination, { replace: true });
  };

  const handleSubmit = async (values: LoginFormValues) => {
    setLoading(true);
    try {
      // 记住/清除用户名（不再持久化密码）
      if (!isUnlock) {
        if (values.remember) {
          setRememberedUsername(values.username);
        } else {
          clearRemembered();
        }
      }

      if (useMock) {
        await handleMockLogin(values);
      } else {
        await handleRealLogin(values);
      }
    } catch (err) {
      // 登录只有在 token + /auth/me 两步都成功后才算完成。若第二步失败，
      // 立即回滚临时 token；锁屏模式保留原锁屏快照供用户重试。
      const authState = useUserStore.getState();
      if (!authState.currentUser && authState.accessToken) {
        if (lockedSession) {
          authState.lock(lockedSession.returnPath);
        } else {
          authState.clearAuth();
        }
      }

      // Issue #687: 检测 biz_code=7010（需要验证码）或 7011（验证码错误）
      // http 拦截器将 biz_code 暴露为 err.bizCode（而非 response.data.biz_code）
      const axiosErr = err as AxiosError<{ biz_code?: number }> & { bizCode?: number; userMessage?: string };
      const bizCode = axiosErr.bizCode ?? axiosErr.response?.data?.biz_code;

      if (bizCode === 7010) {
        // 需要验证码 — 触发验证码流程
        setCaptchaRequired(true);
        message.warning(t('login.captcha.required'));
        return;
      }
      if (bizCode === 7011) {
        // 验证码错误 — 刷新验证码重试
        loadCaptcha();
        message.error(t('login.captcha.invalid'));
        return;
      }

      // Issue #730: 按 biz_code 查语料，fallback 后端 msg 或默认语料
      const i18nKey = getI18nKeyByBizCode(bizCode);
      if (i18nKey) {
        // 带参数的错误码（7012 账号临时锁定 / 7014 IP 限流）：从后端 msg 提取数字
        if (bizCode === 7012) {
          const match = axiosErr.userMessage?.match(/(\d+)/);
          message.error(t(i18nKey, { minutes: match ? match[1] : '?' }));
          return;
        }
        if (bizCode === 7014) {
          const match = axiosErr.userMessage?.match(/(\d+)/);
          message.error(t(i18nKey, { seconds: match ? match[1] : '?' }));
          return;
        }
        // 无参数的错误码 — 直接用前端语料
        message.error(t(i18nKey));
        return;
      }

      // T-0119: error 显示优先级修复（T-0117 follow-up）
      //   1. axios userMessage — 拦截器从后端 envelope 解出的友好业务文本（401/403/...）
      //   2. client-side Error.message — passwordCipher 等前端抛的中文诊断（如 T-0117
      //      "当前访问非安全上下文..."），仅当 err 不是 axios error 时使用，避免
      //      "Request failed with status code 401" 渗透到 toast
      //   3. i18n fallback 'login.failed'
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
          <h1 className={styles.title}>{isUnlock ? t('login.lockedTitle') : loginTitle}</h1>
          <p className={styles.subtitle}>
            {isUnlock ? lockedSession.displayName : 'Unified Network Management System'}
          </p>
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
                autoComplete={usernameAutocomplete}
                readOnly={isUnlock}
              />
            </Form.Item>

            <Form.Item
              name="password"
              rules={[{ required: true, message: t('login.passwordTip') }]}
            >
              {publicSettings?.preventBrowserAutofill ? (
                <BrowserPasswordInput
                  prefix={<LockOutlined style={{ color: 'var(--login-input-icon)' }} />}
                  placeholder={t('login.passwordTip')}
                  showPasswordLabel={t('login.showPassword')}
                  hidePasswordLabel={t('login.hidePassword')}
                />
              ) : (
                <Input.Password
                  prefix={<LockOutlined style={{ color: 'var(--login-input-icon)' }} />}
                  placeholder={t('login.passwordTip')}
                  autoComplete="current-password"
                />
              )}
            </Form.Item>

            {/* Issue #687: 验证码（仅当后端要求时显示） */}
            {captchaRequired && (
              <div style={{ display: 'flex', gap: 8, marginBottom: 24 }}>
                <Form.Item
                  name="captcha"
                  rules={[{ required: true, message: t('login.captcha.required') }]}
                  style={{ flex: 1, marginBottom: 0 }}
                >
                  <Input
                    prefix={<SafetyCertificateOutlined style={{ color: 'var(--login-input-icon)' }} />}
                    placeholder={t('login.captcha.placeholder')}
                    maxLength={5}
                  />
                </Form.Item>
                <div
                  style={{
                    width: 120,
                    height: 40,
                    border: '1px solid var(--login-input-border, #d9d9d9)',
                    borderRadius: 6,
                    overflow: 'hidden',
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    background: 'var(--login-captcha-bg, #f5f5f5)',
                  }}
                  onClick={loadCaptcha}
                  title={t('login.captcha.refresh')}
                >
                  {captchaLoading ? (
                    <Spin size="small" />
                  ) : captchaError ? (
                    <ReloadOutlined style={{ fontSize: 18, color: '#ff4d4f' }} />
                  ) : captchaData ? (
                    <img
                      src={captchaData.image}
                      alt="captcha"
                      style={{ width: '100%', height: '100%', objectFit: 'contain' }}
                    />
                  ) : (
                    <ReloadOutlined style={{ fontSize: 18, color: '#999' }} />
                  )}
                </div>
              </div>
            )}

            {!isUnlock && (
              <Form.Item>
                <div className={styles.rememberRow}>
                  <Form.Item name="remember" valuePropName="checked" noStyle>
                    <Checkbox>{t('login.rememberMe')}</Checkbox>
                  </Form.Item>
                </div>
              </Form.Item>
            )}

            <Form.Item>
              <Button
                type="primary"
                htmlType="submit"
                className={styles.loginButton}
                loading={loading}
                size="large"
                block
              >
                {t(isUnlock ? 'login.unlock' : 'login.submit')}
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
