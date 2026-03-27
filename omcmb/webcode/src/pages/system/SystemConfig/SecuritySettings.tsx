import { Form, Input, InputNumber, Switch, Select, Divider, Space, Alert } from 'antd';
import { useT } from '@/hooks/useT';

const { Option } = Select;

interface SecuritySettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
}

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
      {/* 默认密码 */}
      <Divider orientation="left" plain>默认密码</Divider>
      <Form.Item name="modifyPWD" label="首次登录修改密码" valuePropName="checked">
        <Switch checkedChildren="开启" unCheckedChildren="关闭" />
      </Form.Item>
      <Form.Item name="defaultPasswd" label="默认密码值">
        <Input.Password placeholder="设置密码重置后的默认密码" />
      </Form.Item>

      {/* 密码强度 */}
      <Divider orientation="left" plain>密码强度</Divider>
      <Form.Item name="passwordContent" label="密码必须两种类型" valuePropName="checked" extra="密码必须包含数字、字母、特殊字符中的至少两种">
        <Switch checkedChildren="开启" unCheckedChildren="关闭" />
      </Form.Item>
      <Space>
        <Form.Item name="pwdMinLength" label="密码最小长度">
          <InputNumber min={6} max={32} addonAfter="位" style={{ width: 120 }} />
        </Form.Item>
        <Form.Item name="pwdMaxLength" label="密码最大长度">
          <InputNumber min={6} max={64} addonAfter="位" style={{ width: 120 }} />
        </Form.Item>
      </Space>

      {/* 用户名称 */}
      <Divider orientation="left" plain>用户名称</Divider>
      <Form.Item name="checkUserCodeEnable" label="用户名必含字符提示" valuePropName="checked">
        <Switch checkedChildren="开启" unCheckedChildren="关闭" />
      </Form.Item>

      {/* 密码有效期 */}
      <Divider orientation="left" plain>密码有效期</Divider>
      <Form.Item name="expires" label="用户修改密码频率" valuePropName="checked">
        <Switch checkedChildren="开启" unCheckedChildren="关闭" />
      </Form.Item>
      <Space>
        <Form.Item name="validPeriod" label="有效期天数">
          <InputNumber min={1} max={365} addonAfter="天" style={{ width: 120 }} />
        </Form.Item>
        <Form.Item name="promptBeforeDays" label="提示到期天数">
          <InputNumber min={1} max={30} addonAfter="天前提示" style={{ width: 140 }} />
        </Form.Item>
      </Space>

      {/* 登录错误限制 */}
      <Divider orientation="left" plain>登录错误限制</Divider>
      <Form.Item name="verifyEnable" label="验证码验证" valuePropName="checked" extra="登录错误N次后需要验证码">
        <Switch checkedChildren="开启" unCheckedChildren="关闭" />
      </Form.Item>
      <Space>
        <Form.Item name="attemptTimes" label="错误次数阈值">
          <InputNumber min={1} max={10} addonAfter="次" style={{ width: 120 }} />
        </Form.Item>
        <Form.Item name="sumTimes" label="登录失败锁定次数">
          <InputNumber min={1} max={20} addonAfter="次" style={{ width: 140 }} />
        </Form.Item>
        <Form.Item name="unlockMinu" label="锁定时间">
          <InputNumber min={1} max={1440} addonAfter="分钟" style={{ width: 120 }} />
        </Form.Item>
      </Space>

      {/* IP限流 */}
      <Divider orientation="left" plain>IP限流</Divider>
      <Space>
        <Form.Item name="limitMinus" label="限流时间窗口">
          <InputNumber min={1} max={60} addonAfter="分钟内" style={{ width: 120 }} />
        </Form.Item>
        <Form.Item name="limitCount" label="连续错误次数">
          <InputNumber min={1} max={100} addonAfter="次" style={{ width: 120 }} />
        </Form.Item>
        <Form.Item name="limitTimes" label="自动释放时间">
          <InputNumber min={1} max={1440} addonAfter="分钟后释放" style={{ width: 140 }} />
        </Form.Item>
      </Space>

      {/* 锁屏时间 */}
      <Divider orientation="left" plain>锁屏时间</Divider>
      <Form.Item name="userSessionExpirationMin" label="用户无操作锁屏时间">
        <InputNumber min={1} max={480} addonAfter="分钟" style={{ width: 150 }} />
      </Form.Item>

      {/* 浏览器记录密码 */}
      <Divider orientation="left" plain>浏览器记录密码</Divider>
      <Form.Item name="isBrowserAutoRecordPass" label="开启浏览器记录密码" valuePropName="checked">
        <Switch checkedChildren="开启" unCheckedChildren="关闭" />
      </Form.Item>

      {/* 自动锁定 */}
      <Divider orientation="left" plain>自动锁定</Divider>
      <Form.Item name="autoLockUserDayEnable" label="自动锁定开关" valuePropName="checked" extra="启用自动锁定长期未登录用户">
        <Switch checkedChildren="开启" unCheckedChildren="关闭" />
      </Form.Item>
      <Form.Item name="autoLockUserDay" label="锁定天数阈值">
        <InputNumber min={1} max={365} addonAfter="天未登录自动锁定" style={{ width: 180 }} />
      </Form.Item>

      {/* 最大会话限制 */}
      <Divider orientation="left" plain>最大会话限制</Divider>
      <Form.Item name="isOnlyOneUserLoginEnable" label="限制单一会话" valuePropName="checked" extra="同一用户只允许一个会话登录">
        <Switch checkedChildren="开启" unCheckedChildren="关闭" />
      </Form.Item>

      {/* 登录提示 */}
      <Divider orientation="left" plain>登录提示</Divider>
      <Form.Item name="enabledFlag" label="启用登录后提示" valuePropName="checked">
        <Switch checkedChildren="开启" unCheckedChildren="关闭" />
      </Form.Item>
      <Form.Item name="msg" label="提示消息内容">
        <Input.TextArea rows={3} placeholder="请输入登录后显示的通知消息文本" maxLength={500} showCount />
      </Form.Item>
    </Form>
  );
}
