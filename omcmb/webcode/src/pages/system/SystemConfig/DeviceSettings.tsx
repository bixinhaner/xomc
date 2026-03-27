import { Form, Input, InputNumber, Checkbox, Select, Card, Space } from 'antd';
import { useT } from '@/hooks/useT';

const { Option } = Select;

interface DeviceSettingsProps {
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

// 信号指示器样式
const signalIndicatorStyle = (color: string): React.CSSProperties => ({
  display: 'inline-flex',
  alignItems: 'center',
  padding: '4px 12px',
  backgroundColor: '#fafafa',
  borderRadius: 4,
  marginRight: 16,
});

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
      uploadSelected: '3',
      deviceOfflineEnable: false,
      deviceOfflineSaveDay: 30,
      locationDetection: false,
      latitudeToleranceRange: 100,
    }}>
      {/* 设备通信设置 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>设备通信设置</span>} style={{ marginBottom: 16 }}>
        {/* 设备Inform周期 */}
        <div style={settingRowStyle}>
          <div style={{ marginBottom: 8, fontWeight: 500 }}>设备Inform周期</div>
          <Space wrap style={{ marginBottom: 8 }}>
            <Form.Item name="enbInformPeriodAdjustEnable" valuePropName="checked" noStyle>
              <Checkbox>设备启动ENB Inform周期不符合</Checkbox>
            </Form.Item>
            <Form.Item name="enbInformPeriod" noStyle>
              <InputNumber min={60} max={3600} style={{ width: 70 }} />
            </Form.Item>
            <span>秒自动调整</span>
          </Space>
          <div style={subSettingStyle}>
            <Space style={{ marginBottom: 8 }}>
              规定时间无响应
              <Form.Item name="enbTimeout" noStyle>
                <InputNumber min={300} max={7200} style={{ width: 70 }} />
              </Form.Item>
              <span>秒设备将会关机</span>
            </Space>
          </div>
        </div>

        {/* CPE Inform周期 */}
        <div style={settingRowStyle}>
          <Space wrap style={{ marginBottom: 8 }}>
            <Form.Item name="cpeInformPeriodAdjustEnable" valuePropName="checked" noStyle>
              <Checkbox>设备启动CPE Inform周期不符合</Checkbox>
            </Form.Item>
            <Form.Item name="cpeInformPeriod" noStyle>
              <InputNumber min={60} max={3600} style={{ width: 70 }} />
            </Form.Item>
            <span>秒自动调整</span>
          </Space>
          <div style={subSettingStyle}>
            <Space>
              规定时间无响应
              <Form.Item name="cpeTimeout" noStyle>
                <InputNumber min={300} max={7200} style={{ width: 70 }} />
              </Form.Item>
              <span>秒设备将会关机</span>
            </Space>
          </div>
        </div>

        {/* 基站文件上传协议 */}
        <div style={settingRowStyle}>
          <div style={{ marginBottom: 8, fontWeight: 500 }}>基站文件上传协议</div>
          <Form.Item name="uploadSelected" noStyle>
            <Select style={{ width: 160 }}>
              <Option value="1">http</Option>
              <Option value="2">https</Option>
              <Option value="3">保持基站不变</Option>
            </Select>
          </Form.Item>
        </div>
      </Card>

      {/* 设备管理设置 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>设备管理设置</span>} style={{ marginBottom: 16 }}>
        {/* 设备名称同步设置 */}
        <div style={settingRowStyle}>
          <div style={{ marginBottom: 8, fontWeight: 500 }}>设备名称同步设置</div>
          <div style={{ marginBottom: 8 }}>
            <Form.Item name="nameSettingEnable" valuePropName="checked" noStyle>
              <Checkbox>检查和设置相同设备名称的LMT</Checkbox>
            </Form.Item>
          </div>
          <div style={{ marginLeft: 24 }}>
            <Form.Item name="prompt" valuePropName="checked" noStyle>
              <Checkbox>通知我是否手动同步</Checkbox>
            </Form.Item>
          </div>
        </div>

        {/* 设备访问控制 */}
        <div style={settingRowStyle}>
          <div style={{ marginBottom: 8, fontWeight: 500 }}>设备访问控制</div>
          <Space>
            <Form.Item name="accessContralEnable" valuePropName="checked" noStyle>
              <Checkbox>只允许符合规则</Checkbox>
            </Form.Item>
            <a href="#">规则</a>
            <span>的设备访问系统</span>
          </Space>
        </div>

        {/* 回收站 */}
        <div style={settingRowStyle}>
          <div style={{ marginBottom: 8, fontWeight: 500 }}>回收站</div>
          <Space wrap style={{ marginBottom: 4 }}>
            <Form.Item name="deviceOfflineEnable" valuePropName="checked" noStyle>
              <Checkbox>系统将离线设备移入回收站</Checkbox>
            </Form.Item>
            <span>，保留</span>
            <Form.Item name="deviceOfflineSaveDay" noStyle>
              <InputNumber min={1} max={365} style={{ width: 60 }} />
            </Form.Item>
            <span>天</span>
          </Space>
          <div style={{ marginLeft: 24, color: 'rgba(0, 0, 0, 0.45)', fontSize: 12 }}>
            勾选后，系统将离线设备移入回收站，同时清空告警
          </div>
        </div>

        {/* 基站区域活（GPS检测） */}
        <div style={settingRowStyle}>
          <div style={{ marginBottom: 8, fontWeight: 500 }}>基站区域活</div>
          <Space wrap>
            <Form.Item name="locationDetection" valuePropName="checked" noStyle>
              <Checkbox>启用设备经纬度检测</Checkbox>
            </Form.Item>
            <span>，经纬度变化超过</span>
            <Form.Item name="latitudeToleranceRange" noStyle>
              <InputNumber min={10} max={10000} style={{ width: 70 }} />
            </Form.Item>
            <span>米触发告警</span>
          </Space>
        </div>
      </Card>

      {/* 信号强度设置 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>信号强度设置</span>}>
        {/* 设备信号强度显示 */}
        <div style={settingRowStyle}>
          <div style={{ marginBottom: 12, fontWeight: 500 }}>设备信号强度显示</div>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
            <div style={signalIndicatorStyle('#E88282')}>
              <span style={{ color: '#E88282', marginRight: 4 }}>●</span>
              <span>信号弱 &lt;</span>
              <Form.Item name="rsrpVal0" noStyle style={{ marginLeft: 8 }}>
                <InputNumber style={{ width: 60 }} />
              </Form.Item>
            </div>
            <div style={signalIndicatorStyle('#F2B354')}>
              <span style={{ color: '#F2B354', marginRight: 4 }}>●</span>
              <span>信号正常 &lt;</span>
              <Form.Item name="rsrpVal1" noStyle style={{ marginLeft: 8 }}>
                <InputNumber style={{ width: 60 }} />
              </Form.Item>
            </div>
            <div style={signalIndicatorStyle('#67D972')}>
              <span style={{ color: '#67D972', marginRight: 4 }}>●</span>
              <span>信号强</span>
            </div>
          </div>
        </div>

        {/* UE设备信号强度显示 */}
        <div style={settingRowStyle}>
          <div style={{ marginBottom: 12, fontWeight: 500 }}>UE设备信号强度显示</div>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
            <div style={signalIndicatorStyle('#E88282')}>
              <span style={{ color: '#E88282', marginRight: 4 }}>●</span>
              <span>信号弱 &lt;</span>
              <Form.Item name="uersrpVal0" noStyle style={{ marginLeft: 8 }}>
                <InputNumber style={{ width: 60 }} />
              </Form.Item>
            </div>
            <div style={signalIndicatorStyle('#F2B354')}>
              <span style={{ color: '#F2B354', marginRight: 4 }}>●</span>
              <span>信号正常 &lt;</span>
              <Form.Item name="uersrpVal1" noStyle style={{ marginLeft: 8 }}>
                <InputNumber style={{ width: 60 }} />
              </Form.Item>
            </div>
            <div style={signalIndicatorStyle('#67D972')}>
              <span style={{ color: '#67D972', marginRight: 4 }}>●</span>
              <span>信号强</span>
            </div>
          </div>
        </div>
      </Card>
    </Form>
  );
}
