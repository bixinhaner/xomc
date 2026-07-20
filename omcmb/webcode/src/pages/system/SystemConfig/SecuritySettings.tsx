import { Form, Input, InputNumber, Checkbox, Card, Space, Alert, theme } from 'antd';
import { useT } from '@/hooks/useT';

interface SecuritySettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
  defaultPasswordConfigured: boolean;
}

// 设置行样式
const settingRowStyle: React.CSSProperties = {
  marginBottom: 16,
};

// 子设置区域布局（颜色随主题，见组件内 subSettingStyle）
const subSettingBaseStyle: React.CSSProperties = {
  marginLeft: 24,
  marginTop: 12,
  padding: '12px 16px',
  borderRadius: 4,
};

export default function SecuritySettings({ form, defaultPasswordConfigured }: SecuritySettingsProps) {
  const t = useT();
  const { token } = theme.useToken();
  // 子设置区域背景跟随明/暗主题，不再写死 #fafafa
  const subSettingStyle: React.CSSProperties = { ...subSettingBaseStyle, backgroundColor: token.colorFillAlter };

  return (
    <Form form={form} layout="vertical" size="small" initialValues={{
      passwordContent: false,
      pwdMinLength: 10,
      pwdMaxLength: 23,
      checkUserCodeEnable: false,
      expires: false,
      validPeriod: 70,
      promptBeforeDays: 6,
      verifyEnable: false,
      attemptTimes: 5,
      sumTimes: 8,
      unlockMinu: 2,
      limitMinus: 1,
      limitCount: 30,
      limitTimes: 120,
      userSessionExpirationMin: 0,
      isBrowserAutoRecordPass: false,
      autoLockUserDayEnable: false,
      autoLockUserDay: 90,
      isOnlyOneUserLoginEnable: false,
      enabledFlag: false,
      msg: '',
    }}>
      {/* 默认密码 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.security.defaultPassword')}</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          {/* Issue #649：modifyPWD 按设计废弃（管理员注入密码已改为后端硬规则强制首次改密）。
              下方 Alert 提示密码强度规则调整后要检查默认密码是否仍满足，避免 drift。 */}
          <Alert
            type="info"
            showIcon
            message={t('system.security.defaultPasswordDriftWarning')}
            style={{ marginBottom: 12 }}
          />
          <div style={subSettingStyle}>
            <Space>
              {t('system.security.useAsDefaultPassword')}
              <Form.Item name="defaultPasswd" noStyle>
                <Input.Password
                  style={{ width: 240 }}
                  maxLength={50}
                  autoComplete="new-password"
                  visibilityToggle={false}
                  placeholder={t(
                    defaultPasswordConfigured
                      ? 'system.security.defaultPasswordConfiguredPlaceholder'
                      : 'system.security.defaultPasswordUnsetPlaceholder',
                  )}
                />
              </Form.Item>
            </Space>
          </div>
        </div>
      </Card>

      {/* 密码强度 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.security.passwordStrength')}</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Form.Item name="passwordContent" valuePropName="checked" noStyle>
            <Checkbox>{t('system.security.passwordComplexityRequirement')}</Checkbox>
          </Form.Item>
          <div style={subSettingStyle}>
            <div style={{ marginBottom: 8 }}>{t('system.security.userPasswordLength')}</div>
            <Space size="large">
              <Space>
                {t('common.minLength')}
                <Form.Item name="pwdMinLength" noStyle>
                  <InputNumber min={6} max={32} style={{ width: 60 }} />
                </Form.Item>
              </Space>
              <Space>
                {t('common.maxLength')}
                <Form.Item name="pwdMaxLength" noStyle>
                  <InputNumber min={6} max={64} style={{ width: 60 }} />
                </Form.Item>
              </Space>
            </Space>
          </div>
        </div>
      </Card>

      {/* 用户 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.security.user')}</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <span style={{ color: token.colorTextTertiary }}>
            {t('system.security.usernameRule')}
          </span>
        </div>
      </Card>

      {/* 密码有效期 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.security.passwordExpiration')}</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space wrap>
            <Form.Item name="expires" valuePropName="checked" noStyle>
              <Checkbox>{t('system.security.passwordValidFor')}</Checkbox>
            </Form.Item>
            <Form.Item name="validPeriod" noStyle>
              <InputNumber min={1} max={365} style={{ width: 60 }} />
            </Form.Item>
            <span>{t('system.security.daysBeforeExpiry')}</span>
            <Form.Item name="promptBeforeDays" noStyle>
              <InputNumber min={1} max={30} style={{ width: 60 }} />
            </Form.Item>
            <span>{t('system.security.daysRemind')}</span>
          </Space>
        </div>
      </Card>

      {/* 登录锁定 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.security.loginLockout')}</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space wrap>
            <Form.Item name="verifyEnable" valuePropName="checked" noStyle>
              <Checkbox>{t('system.security.requireCaptchaAfterFailedAttempts')}</Checkbox>
            </Form.Item>
            <Form.Item name="attemptTimes" noStyle>
              <InputNumber min={1} max={10} style={{ width: 60 }} />
            </Form.Item>
            <span>{t('system.security.timesThenRequireCaptcha')}</span>
          </Space>
        </div>
        <div style={subSettingStyle}>
          <Space wrap>
            {t('sysconfig.security.failedAttemptsPrefix')}
            <Form.Item name="sumTimes" noStyle>
              <InputNumber min={1} max={20} style={{ width: 60 }} />
            </Form.Item>
            <span>{t('sysconfig.security.lockAccountSuffix')}</span>
            <Form.Item name="unlockMinu" noStyle>
              <InputNumber min={1} max={1440} style={{ width: 60 }} />
            </Form.Item>
            <span>{t('sysconfig.security.minutes')}</span>
          </Space>
        </div>
      </Card>

      {/* IP限流 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.security.ipRateLimit')}</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space wrap>
            {t('sysconfig.security.ifWithinPrefix')}
            <Form.Item name="limitMinus" noStyle>
              <InputNumber min={1} max={60} style={{ width: 60 }} />
            </Form.Item>
            <span>{t('sysconfig.security.passwordRetryPrefix')}</span>
            <Form.Item name="limitCount" noStyle>
              <InputNumber min={1} max={100} style={{ width: 60 }} />
            </Form.Item>
            <span>{t('sysconfig.security.ipLockSuffix')}</span>
            <Form.Item name="limitTimes" noStyle>
              <InputNumber min={1} max={1440} style={{ width: 60 }} />
            </Form.Item>
            <span>{t('sysconfig.security.autoUnlockSuffix')}</span>
          </Space>
        </div>
      </Card>

      {/* 屏幕锁定 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.security.screenLockout')}</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space>
            {t('sysconfig.security.ifUserPrefix')}
            <Form.Item name="userSessionExpirationMin" noStyle>
              <InputNumber min={0} max={480} style={{ width: 60 }} />
            </Form.Item>
            <span>{t('sysconfig.security.idleLockSuffix')}</span>
          </Space>
        </div>
      </Card>

      {/* 禁止浏览器自动记录密码 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.security.preventBrowserPasswordSave')}</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Form.Item name="isBrowserAutoRecordPass" valuePropName="checked" noStyle>
            <Checkbox>{t('system.security.preventBrowserPasswordSave')}</Checkbox>
          </Form.Item>
        </div>
      </Card>

      {/* 账户锁定 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.security.accountLockout')}</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space>
            <Form.Item name="autoLockUserDayEnable" noStyle valuePropName="checked">
              <Checkbox>{t('sysconfig.security.noLoginPrefix')}</Checkbox>
            </Form.Item>
            <Form.Item name="autoLockUserDay" noStyle>
              <InputNumber min={1} max={365} style={{ width: 60 }} />
            </Form.Item>
            <span>{t('sysconfig.security.daysLockAccount')}</span>
          </Space>
        </div>
      </Card>

      {/* 最大会话限制 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.security.maxSessionLimit')}</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Form.Item name="isOnlyOneUserLoginEnable" noStyle valuePropName="checked">
            <Checkbox>{t('system.security.allowMultipleConcurrentLogin')}</Checkbox>
          </Form.Item>
        </div>
      </Card>

      {/* 登录提示 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.security.loginNotification')}</span>}>
        <div style={settingRowStyle}>
          <Form.Item name="enabledFlag" valuePropName="checked" noStyle>
            <Checkbox>{t('system.security.pushNotificationOnLogin')}</Checkbox>
          </Form.Item>
          <div style={subSettingStyle}>
            <Form.Item name="msg" noStyle>
              <Input.TextArea rows={3} maxLength={500} showCount placeholder={t('system.security.pleaseInputLoginMessage')} />
            </Form.Item>
          </div>
        </div>
      </Card>
    </Form>
  );
}
