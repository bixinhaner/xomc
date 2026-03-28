import { Form, InputNumber, Checkbox, Select, Card, Space } from 'antd';
import { useT } from '@/hooks/useT';

const { Option } = Select;

interface DeviceSettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
}

// 设置行样式
const settingRowStyle: React.CSSProperties = {
  marginBottom: 16,
};

export default function DeviceSettings({ form }: DeviceSettingsProps) {
  const t = useT();

  return (
    <Form form={form} layout="vertical" size="small" initialValues={{
      enbInformPeriodAdjustEnable: true,
      enbInformPeriod: 60,
      enbTimeout: 100,
      cpeInformPeriodAdjustEnable: false,
      cpeInformPeriod: 60,
      cpeTimeout: 300,
      nameSettingEnable: true,
      prompt: false,
      accessContralEnable: false,
      uploadSelected: '3',
      deviceOfflineEnable: false,
      deviceOfflineSaveDay: 90,
      locationDetection: true,
      latitudeToleranceRange: 1000,
    }}>
      {/* 设备Inform周期 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>设备Inform周期</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space wrap>
            <Form.Item name="enbInformPeriodAdjustEnable" valuePropName="checked" noStyle>
              <Checkbox>OMC在设备启动时，检测eNB Inform周期不符合</Checkbox>
            </Form.Item>
            <Form.Item name="enbInformPeriod" noStyle>
              <InputNumber min={60} max={3600} style={{ width: 70 }} />
            </Form.Item>
            <span>秒，则自动调整</span>
          </Space>
        </div>
        <div style={settingRowStyle}>
          <Space wrap>
            <Form.Item name="enbTimeoutEnable" valuePropName="checked" noStyle>
              <Checkbox>如果系统在</Checkbox>
            </Form.Item>
            <Form.Item name="enbTimeout" noStyle>
              <InputNumber min={60} max={7200} style={{ width: 70 }} />
            </Form.Item>
            <span>秒内没有收到设备心跳消息，则显示设备状态为离线</span>
          </Space>
        </div>
        <div style={settingRowStyle}>
          <Space wrap>
            <Form.Item name="cpeInformPeriodAdjustEnable" valuePropName="checked" noStyle>
              <Checkbox>OMC在设备启动时，检测CPE Inform周期不符合</Checkbox>
            </Form.Item>
            <Form.Item name="cpeInformPeriod" noStyle>
              <InputNumber min={60} max={3600} style={{ width: 70 }} />
            </Form.Item>
            <span>秒，则自动调整</span>
          </Space>
        </div>
        <div style={settingRowStyle}>
          <Space wrap>
            <Form.Item name="cpeTimeoutEnable" valuePropName="checked" noStyle>
              <Checkbox>如果系统在</Checkbox>
            </Form.Item>
            <Form.Item name="cpeTimeout" noStyle>
              <InputNumber min={60} max={7200} style={{ width: 70 }} />
            </Form.Item>
            <span>秒内没有收到设备心跳消息，则显示设备状态为离线</span>
          </Space>
        </div>
      </Card>

      {/* 设备名称同步 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>设备名称同步</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Form.Item name="nameSettingEnable" valuePropName="checked" noStyle>
            <Checkbox>检查并设置网管上的设备名称与LMT上设备名称一致。</Checkbox>
          </Form.Item>
        </div>
        <div style={settingRowStyle}>
          <div style={{ marginLeft: 24 }}>
            <Form.Item name="prompt" valuePropName="checked" noStyle>
              <Checkbox>提示我人工处理同步</Checkbox>
            </Form.Item>
          </div>
        </div>
      </Card>

      {/* 设备接入控制 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>设备接入控制</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space>
            <Form.Item name="accessContralEnable" valuePropName="checked" noStyle>
              <Checkbox>只允许符合接入控制</Checkbox>
            </Form.Item>
            <a href="#">规则</a>
            <span>的设备连接到网管。</span>
          </Space>
        </div>
      </Card>

      {/* CPE信号强度 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>CPE信号强度</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <span style={{ marginRight: 16 }}>按照设定的范围显示信号强度</span>
          <span style={{ marginRight: 16, display: 'inline-flex', alignItems: 'center', padding: '4px 12px', backgroundColor: '#fff1f0', borderRadius: 4 }}>
            <span style={{ width: 8, height: 8, borderRadius: '50%', backgroundColor: '#ff4d4f', marginRight: 8 }} />
            弱&lt; -100
          </span>
          <span style={{ marginRight: 16, display: 'inline-flex', alignItems: 'center', padding: '4px 12px', backgroundColor: '#fff7e6', borderRadius: 4 }}>
            <span style={{ width: 8, height: 8, borderRadius: '50%', backgroundColor: '#fa8c16', marginRight: 8 }} />
            正常&lt; -80
          </span>
          <span style={{ display: 'inline-flex', alignItems: 'center', padding: '4px 12px', backgroundColor: '#f6ffed', borderRadius: 4 }}>
            <span style={{ width: 8, height: 8, borderRadius: '50%', backgroundColor: '#52c41a', marginRight: 8 }} />
            强
          </span>
        </div>
      </Card>

      {/* UE信号强度 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>UE信号强度</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <span style={{ marginRight: 16 }}>按照设定的范围显示信号强度</span>
          <span style={{ marginRight: 16, display: 'inline-flex', alignItems: 'center', padding: '4px 12px', backgroundColor: '#fff1f0', borderRadius: 4 }}>
            <span style={{ width: 8, height: 8, borderRadius: '50%', backgroundColor: '#ff4d4f', marginRight: 8 }} />
            弱&lt; -100
          </span>
          <span style={{ marginRight: 16, display: 'inline-flex', alignItems: 'center', padding: '4px 12px', backgroundColor: '#fff7e6', borderRadius: 4 }}>
            <span style={{ width: 8, height: 8, borderRadius: '50%', backgroundColor: '#fa8c16', marginRight: 8 }} />
            正常&lt; -80
          </span>
          <span style={{ display: 'inline-flex', alignItems: 'center', padding: '4px 12px', backgroundColor: '#f6ffed', borderRadius: 4 }}>
            <span style={{ width: 8, height: 8, borderRadius: '50%', backgroundColor: '#52c41a', marginRight: 8 }} />
            强
          </span>
        </div>
      </Card>

      {/* 基站文件上传协议 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>基站文件上传协议</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Form.Item name="uploadSelected" noStyle>
            <Select style={{ width: 160 }}>
              <Option value="1">http</Option>
              <Option value="2">https</Option>
              <Option value="3">保持基站不变</Option>
            </Select>
          </Form.Item>
        </div>
      </Card>

      {/* 回收站 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>回收站</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space wrap>
            <Form.Item name="deviceOfflineEnable" valuePropName="checked" noStyle>
              <Checkbox>设备默认连接</Checkbox>
            </Form.Item>
            <Form.Item name="deviceOfflineSaveDay" noStyle>
              <InputNumber min={1} max={365} style={{ width: 60 }} />
            </Form.Item>
            <span>天不在线自动加入回收站</span>
          </Space>
        </div>
        <div style={settingRowStyle}>
          <span style={{ color: 'rgba(0, 0, 0, 0.45)' }}>每天00:10检查设备离线时间</span>
        </div>
      </Card>

      {/* eNB位置移动检测 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>eNB位置移动检测</span>}>
        <div style={settingRowStyle}>
          <Space wrap>
            <Form.Item name="locationDetection" valuePropName="checked" noStyle>
              <Checkbox>如果eNB位置移动超过</Checkbox>
            </Form.Item>
            <Form.Item name="latitudeToleranceRange" noStyle>
              <InputNumber min={10} max={10000} style={{ width: 70 }} />
            </Form.Item>
            <span>米，eNB将被锁定</span>
          </Space>
        </div>
      </Card>
    </Form>
  );
}
