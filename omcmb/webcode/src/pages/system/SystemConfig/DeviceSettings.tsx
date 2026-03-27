import { Form, Input, InputNumber, Switch, Select, Divider, Space, Radio } from 'antd';
import { useT } from '@/hooks/useT';

const { Option } = Select;

interface DeviceSettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
}

export default function DeviceSettings({ form }: DeviceSettingsProps) {
  const t = useT();

  return (
    <Form form={form} layout="vertical" size="small" initialValues={{
      enbInformPeriodAdjustEnable: false,
      enbInformPeriod: 300,
      enbTimeout: 900,
      cpeInformPeriodAdjustEnable: false,
      cpeInformPeriod: 300,
      cpeTimeout: 900,
      nameSettingEnable: false,
      prompt: false,
      accessContralEnable: false,
      rsrpVal0: -100,
      rsrpVal1: -80,
      uersrpVal0: -100,
      uersrpVal1: -80,
      uploadSelected: 'keep',
      deviceOfflineEnable: false,
      deviceOfflineSaveDay: 30,
      locationDetection: false,
      latitudeToleranceRange: 100,
    }}>
      {/* 设备Inform周期 */}
      <Divider orientation="left" plain>设备Inform周期</Divider>
      <Form.Item name="enbInformPeriodAdjustEnable" label="ENB心跳周期检测" valuePropName="checked">
        <Switch checkedChildren="开启" unCheckedChildren="关闭" />
      </Form.Item>
      <Space>
        <Form.Item name="enbInformPeriod" label="ENB Inform周期">
          <InputNumber min={60} max={3600} addonAfter="秒" style={{ width: 140 }} />
        </Form.Item>
        <Form.Item name="enbTimeout" label="ENB超时时间">
          <InputNumber min={300} max={7200} addonAfter="秒" style={{ width: 140 }} />
        </Form.Item>
      </Space>
      <Form.Item name="cpeInformPeriodAdjustEnable" label="CPE心跳周期检测" valuePropName="checked">
        <Switch checkedChildren="开启" unCheckedChildren="关闭" />
      </Form.Item>
      <Space>
        <Form.Item name="cpeInformPeriod" label="CPE Inform周期">
          <InputNumber min={60} max={3600} addonAfter="秒" style={{ width: 140 }} />
        </Form.Item>
        <Form.Item name="cpeTimeout" label="CPE超时时间">
          <InputNumber min={300} max={7200} addonAfter="秒" style={{ width: 140 }} />
        </Form.Item>
      </Space>

      {/* 设备名称同步 */}
      <Divider orientation="left" plain>设备名称同步</Divider>
      <Form.Item name="nameSettingEnable" label="检查相同设备名称" valuePropName="checked" extra="检查和设置相同设备名称的LMT">
        <Switch checkedChildren="开启" unCheckedChildren="关闭" />
      </Form.Item>
      <Form.Item name="prompt" label="通知手动同步" valuePropName="checked">
        <Switch checkedChildren="开启" unCheckedChildren="关闭" />
      </Form.Item>

      {/* 设备访问控制 */}
      <Divider orientation="left" plain>设备访问控制</Divider>
      <Form.Item name="accessContralEnable" label="访问控制开关" valuePropName="checked" extra="只允许符合规则的设备访问系统">
        <Switch checkedChildren="开启" unCheckedChildren="关闭" />
      </Form.Item>

      {/* 设备信号强度显示 */}
      <Divider orientation="left" plain>设备信号强度显示</Divider>
      <Space>
        <Form.Item name="rsrpVal0" label="信号弱阈值" extra="小于此值显示为弱">
          <InputNumber addonAfter="dBm" style={{ width: 120 }} />
        </Form.Item>
        <Form.Item name="rsrpVal1" label="信号正常阈值" extra="小于此值显示为正常，大于等于显示为强">
          <InputNumber addonAfter="dBm" style={{ width: 120 }} />
        </Form.Item>
      </Space>

      {/* UE设备信号强度 */}
      <Divider orientation="left" plain>UE设备信号强度</Divider>
      <Space>
        <Form.Item name="uersrpVal0" label="UE信号弱阈值">
          <InputNumber style={{ width: 120 }} />
        </Form.Item>
        <Form.Item name="uersrpVal1" label="UE信号正常阈值">
          <InputNumber style={{ width: 120 }} />
        </Form.Item>
      </Space>

      {/* 基站文件上传协议 */}
      <Divider orientation="left" plain>基站文件上传协议</Divider>
      <Form.Item name="uploadSelected" label="上传协议选择">
        <Select style={{ width: 200 }}>
          <Option value="http">HTTP</Option>
          <Option value="https">HTTPS</Option>
          <Option value="keep">保持基站不变</Option>
        </Select>
      </Form.Item>

      {/* 回收站 */}
      <Divider orientation="left" plain>回收站</Divider>
      <Form.Item name="deviceOfflineEnable" label="离线设备移入回收站" valuePropName="checked">
        <Switch checkedChildren="开启" unCheckedChildren="关闭" />
      </Form.Item>
      <Form.Item name="deviceOfflineSaveDay" label="保存天数">
        <InputNumber min={1} max={365} addonAfter="天" style={{ width: 140 }} />
      </Form.Item>

      {/* 基站位置移动检测 */}
      <Divider orientation="left" plain>基站位置移动检测（GPS检测）</Divider>
      <Form.Item name="locationDetection" label="位置检测开关" valuePropName="checked" extra="启用设备经纬度变化检测">
        <Switch checkedChildren="开启" unCheckedChildren="关闭" />
      </Form.Item>
      <Form.Item name="latitudeToleranceRange" label="经纬度容差范围" extra="经纬度变化超过此范围触发告警">
        <InputNumber min={10} max={10000} addonAfter="米" style={{ width: 150 }} />
      </Form.Item>
    </Form>
  );
}
