import { Form, Input, InputNumber, Checkbox, Card, Space } from 'antd';
import { useT } from '@/hooks/useT';

interface SecuritySettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
}

// 分组标题样式
const sectionTitleStyle: React.CSSProperties = {
  fontSize: 14,
  fontWeight: 600,
  marginBottom: 16,
  paddingBottom: 8,
  borderBottom: '1px solid #f0f0f0',
  color: '#333',
};

// 设置行样式
const settingRowStyle: React.CSSProperties = {
  marginBottom: 16,
};

// 子设置区域样式
const subSettingStyle: React.CSSProperties = {
  marginLeft: 24,
  marginTop: 12,
  padding: '12px 16px',
  backgroundColor: '#fafafa',
  borderRadius: 4,
};

export default function SecuritySettings({ form }: SecuritySettingsProps) {
  const t = useT();

  return (
    <Form form={form} layout="vertical" size="small" initialValues={{
      modifyPWD: false,
      defaultPasswd: '',
      passwordContent: false,
      pwdMinLength: 8,
      pwdMaxLength: 32,
      checkUserCodeEnable: false,
      expires: false,
      validPeriod: 90,
      promptBeforeDays: 7,
      verifyEnable: false,
      attemptTimes: 3,
      sumTimes: 5,
      unlockMinu: 30,
      limitMinus: 5,
      limitCount: 10,
      limitTimes: 30,
      userSessionExpirationMin: 30,
      isBrowserAutoRecordPass: false,
      autoLockUserDayEnable: false,
      autoLockUserDay: 90,
      isOnlyOneUserLoginEnable: false,
      enabledFlag: false,
      msg: '',
    }}>
      {/* 密码策略 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>密码策略</span>} style={{ marginBottom: 16 }}>
        {/* 默认密码 */}
        <div style={settingRowStyle}>
          <Form.Item name="modifyPWD" valuePropName="checked" noStyle>
            <Checkbox>首次登录修改密码</Checkbox>
          </Form.Item>
          <div style={subSettingStyle}>
            <Space>
              将
              <Form.Item name="defaultPasswd" noStyle>
                <Input style={{ width: 100 }} maxLength={50} placeholder="请输入默认密码" />
              </Form.Item>
              作为密码重置后的默认密码
            </Space>
          </div>
        </div>

        {/* 密码强度 */}
        <div style={settingRowStyle}>
          <Form.Item name="passwordContent" valuePropName="checked" noStyle>
            <Checkbox>密码必须两种类型</Checkbox>
          </Form.Item>
          <div style={subSettingStyle}>
            <div style={{ marginBottom: 8 }}>用户密码长度：</div>
            <Space size="large">
              <Space>
                <span>最小值</span>
                <Form.Item name="pwdMinLength" noStyle>
                  <InputNumber min={6} max={32} style={{ width: 60 }} />
                </Form.Item>
              </Space>
              <Space>
                <span>最大值</span>
                <Form.Item name="pwdMaxLength" noStyle>
                  <InputNumber min={6} max={64} style={{ width: 60 }} />
                </Form.Item>
              </Space>
            </Space>
          </div>
        </div>

        {/* 密码有效期 */}
        <div style={settingRowStyle}>
          <Form.Item name="expires" valuePropName="checked" noStyle>
            <Checkbox>用户修改密码频率</Checkbox>
          </Form.Item>
          <div style={subSettingStyle}>
            <Space>
              <Form.Item name="validPeriod" noStyle>
                <InputNumber min={1} max={365} style={{ width: 60 }} />
              </Form.Item>
              <span>天修改一次密码，系统会在到期前</span>
              <Form.Item name="promptBeforeDays" noStyle>
                <InputNumber min={1} max={30} style={{ width: 60 }} />
              </Form.Item>
              <span>天提示</span>
            </Space>
          </div>
        </div>
      </Card>

      {/* 登录安全 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>登录安全</span>} style={{ marginBottom: 16 }}>
        {/* 最大允许登录错误次数 */}
        <div style={settingRowStyle}>
          <div style={{ marginBottom: 8, fontWeight: 500 }}>最大允许登录错误次数</div>
          <Space wrap>
            <Form.Item name="verifyEnable" valuePropName="checked" noStyle>
              <Checkbox>验证码验证时的用户名或密码提示</Checkbox>
            </Form.Item>
            <Space>
              <Form.Item name="attemptTimes" noStyle>
                <InputNumber min={1} max={10} style={{ width: 60 }} />
              </Form.Item>
              <span>次，锁定时间</span>
            </Space>
          </Space>
          <div style={{ ...subSettingStyle, marginTop: 12 }}>
            <Space>
              登录失败
              <Form.Item name="sumTimes" noStyle>
                <InputNumber min={1} max={20} style={{ width: 60 }} />
              </Form.Item>
              <span>次后用户锁定</span>
              <Form.Item name="unlockMinu" noStyle>
                <InputNumber min={1} max={1440} style={{ width: 60 }} />
              </Form.Item>
              <span>分钟</span>
            </Space>
          </div>
        </div>

        {/* IP限流 */}
        <div style={settingRowStyle}>
          <div style={{ marginBottom: 8, fontWeight: 500 }}>IP限流</div>
          <Space wrap>
            当在
            <Form.Item name="limitMinus" noStyle>
              <InputNumber min={1} max={60} style={{ width: 60 }} />
            </Form.Item>
            <span>分钟内连续错误</span>
            <Form.Item name="limitCount" noStyle>
              <InputNumber min={1} max={100} style={{ width: 60 }} />
            </Form.Item>
            <span>次，IP加入黑名单，</span>
            <Form.Item name="limitTimes" noStyle>
              <InputNumber min={1} max={1440} style={{ width: 60 }} />
            </Form.Item>
            <span>分钟后自动释放</span>
          </Space>
        </div>

        {/* 锁屏时间 */}
        <div style={settingRowStyle}>
          <div style={{ marginBottom: 8, fontWeight: 500 }}>锁屏时间</div>
          <Space>
            用户非活动状态
            <Form.Item name="userSessionExpirationMin" noStyle>
              <InputNumber min={1} max={480} style={{ width: 60 }} />
            </Form.Item>
            <span>分钟后请锁屏</span>
          </Space>
        </div>

        {/* 浏览器记录密码 */}
        <div style={settingRowStyle}>
          <Form.Item name="isBrowserAutoRecordPass" valuePropName="checked" noStyle>
            <Checkbox>开启浏览器记录密码</Checkbox>
          </Form.Item>
        </div>
      </Card>

      {/* 用户管理 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>用户管理</span>} style={{ marginBottom: 16 }}>
        {/* 用户 */}
        <div style={settingRowStyle}>
          <div style={{ marginBottom: 8 }}>
            <span style={{ fontWeight: 500 }}>用户名称：</span>
            <span style={{ color: 'rgba(0, 0, 0, 0.45)' }}>用户名必含字符提示</span>
          </div>
        </div>

        {/* 自动锁定 */}
        <div style={settingRowStyle}>
          <div style={{ marginBottom: 8, fontWeight: 500 }}>自动锁定</div>
          <Space>
            <Form.Item name="autoLockUserDayEnable" valuePropName="checked" noStyle>
              <Checkbox>自动锁定超过</Checkbox>
            </Form.Item>
            <Form.Item name="autoLockUserDay" noStyle>
              <InputNumber min={1} max={365} style={{ width: 60 }} />
            </Form.Item>
            <span>天未登录的用户</span>
          </Space>
        </div>

        {/* 最大会话限制 */}
        <div style={settingRowStyle}>
          <Form.Item name="isOnlyOneUserLoginEnable" valuePropName="checked" noStyle>
            <Checkbox>同一用户只允许一个会话登录</Checkbox>
          </Form.Item>
        </div>
      </Card>

      {/* 登录提示 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>登录提示</span>}>
        <div style={settingRowStyle}>
          <Form.Item name="enabledFlag" valuePropName="checked" noStyle>
            <Checkbox>通知用户消息</Checkbox>
          </Form.Item>
          <div style={subSettingStyle}>
            <Form.Item name="msg" noStyle>
              <Input.TextArea rows={3} maxLength={500} showCount placeholder="请输入登录后提示消息" />
            </Form.Item>
          </div>
        </div>
      </Card>
    </Form>
  );
}
