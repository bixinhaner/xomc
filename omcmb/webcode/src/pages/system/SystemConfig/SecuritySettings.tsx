import { Form, Input, InputNumber, Checkbox, Card, Space } from 'antd';
import { useT } from '@/hooks/useT';

interface SecuritySettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
}

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
      defaultPasswd: 'OMC@123456',
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
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>默认密码</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Form.Item name="modifyPWD" valuePropName="checked" noStyle>
            <Checkbox>用户需要在首次登录时修改默认密码</Checkbox>
          </Form.Item>
          <div style={subSettingStyle}>
            <Space>
              将
              <Form.Item name="defaultPasswd" noStyle>
                <Input style={{ width: 120 }} maxLength={50} />
              </Form.Item>
              作为默认密码登录，用户被密码重置后使用此密码
            </Space>
          </div>
        </div>
      </Card>

      {/* 密码强度 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>密码强度</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Form.Item name="passwordContent" valuePropName="checked" noStyle>
            <Checkbox>密码需要包含数字、小写字母、大写字母和特殊字符(.!@#$%^&*?)</Checkbox>
          </Form.Item>
          <div style={subSettingStyle}>
            <div style={{ marginBottom: 8 }}>用户密码长度：</div>
            <Space size="large">
              <Space>
                最小值
                <Form.Item name="pwdMinLength" noStyle>
                  <InputNumber min={6} max={32} style={{ width: 60 }} />
                </Form.Item>
              </Space>
              <Space>
                最大值
                <Form.Item name="pwdMaxLength" noStyle>
                  <InputNumber min={6} max={64} style={{ width: 60 }} />
                </Form.Item>
              </Space>
            </Space>
          </div>
        </div>
      </Card>

      {/* 用户 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>用户</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space>
            <span style={{ fontWeight: 500 }}>用户名称：</span>
            <span style={{ color: 'rgba(0, 0, 0, 0.45)' }}>
              只能包含数字、26个小写字母、大写字母和减号(-)、下划线(_)、破折号(-)
            </span>
          </Space>
        </div>
      </Card>

      {/* 密码有效期 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>密码有效期</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space wrap>
            <Form.Item name="expires" valuePropName="checked" noStyle>
              <Checkbox>密码有效期为</Checkbox>
            </Form.Item>
            <Form.Item name="validPeriod" noStyle>
              <InputNumber min={1} max={365} style={{ width: 60 }} />
            </Form.Item>
            <span>天修改一次密码，系统将在到期前</span>
            <Form.Item name="promptBeforeDays" noStyle>
              <InputNumber min={1} max={30} style={{ width: 60 }} />
            </Form.Item>
            <span>天提醒</span>
          </Space>
        </div>
      </Card>

      {/* 登录锁定 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>登录锁定</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space wrap>
            <Form.Item name="verifyEnable" valuePropName="checked" noStyle>
              <Checkbox>用户登录需要验证码验证，如果用户名或密码输入错误</Checkbox>
            </Form.Item>
            <Form.Item name="attemptTimes" noStyle>
              <InputNumber min={1} max={10} style={{ width: 60 }} />
            </Form.Item>
            <span>次，将被要求输入验证码</span>
          </Space>
        </div>
        <div style={subSettingStyle}>
          <Space wrap>
            如果用户名或密码尝试错误
            <Form.Item name="sumTimes" noStyle>
              <InputNumber min={1} max={20} style={{ width: 60 }} />
            </Form.Item>
            <span>次，将锁定账户</span>
            <Form.Item name="unlockMinu" noStyle>
              <InputNumber min={1} max={1440} style={{ width: 60 }} />
            </Form.Item>
            <span>分钟</span>
          </Space>
        </div>
      </Card>

      {/* IP限流 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>IP限流</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space wrap>
            如果你在
            <Form.Item name="limitMinus" noStyle>
              <InputNumber min={1} max={60} style={{ width: 60 }} />
            </Form.Item>
            <span>分钟内连续输入错误的密码或用户名</span>
            <Form.Item name="limitCount" noStyle>
              <InputNumber min={1} max={100} style={{ width: 60 }} />
            </Form.Item>
            <span>次，登录IP将被锁定，</span>
            <Form.Item name="limitTimes" noStyle>
              <InputNumber min={1} max={1440} style={{ width: 60 }} />
            </Form.Item>
            <span>分钟后自动解锁</span>
          </Space>
        </div>
      </Card>

      {/* 屏幕锁定 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>屏幕锁定</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space>
            如果用户
            <Form.Item name="userSessionExpirationMin" noStyle>
              <InputNumber min={0} max={480} style={{ width: 60 }} />
            </Form.Item>
            <span>分钟没有任何操作，系统将会自动锁屏</span>
          </Space>
        </div>
      </Card>

      {/* 禁止浏览器自动记录密码 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>禁止浏览器自动记录密码</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Form.Item name="isBrowserAutoRecordPass" valuePropName="checked" noStyle>
            <Checkbox>禁止浏览器自动记录密码</Checkbox>
          </Form.Item>
        </div>
      </Card>

      {/* 账户锁定 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>账户锁定</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space>
            <Form.Item name="autoLockUserDayEnable" noStyle valuePropName="checked">
              <Checkbox>连续未登录omc超过</Checkbox>
            </Form.Item>
            <Form.Item name="autoLockUserDay" noStyle>
              <InputNumber min={1} max={365} style={{ width: 60 }} />
            </Form.Item>
            <span>天，锁定账户</span>
          </Space>
        </div>
      </Card>

      {/* 最大会话限制 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>最大会话限制</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Form.Item name="isOnlyOneUserLoginEnable" noStyle valuePropName="checked">
            <Checkbox>允许用户同时在多个设备登陆</Checkbox>
          </Form.Item>
        </div>
      </Card>

      {/* 登录提示 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>登录提示</span>}>
        <div style={settingRowStyle}>
          <Form.Item name="enabledFlag" valuePropName="checked" noStyle>
            <Checkbox>用户每次登录时，系统推送消息通知</Checkbox>
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
