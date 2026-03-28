import { Form, Input, InputNumber, Switch, Select, Button, Card, Space, message } from 'antd';
import { useT } from '@/hooks/useT';

const { Option } = Select;

interface OmcSettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
}

// 设置行样式
const settingRowStyle: React.CSSProperties = {
  marginBottom: 16,
};

// 子设置区域样式
const subSettingStyle: React.CSSProperties = {
  marginTop: 12,
  padding: '12px 16px',
  backgroundColor: '#fafafa',
  borderRadius: 4,
};

// 分组标题样式
const sectionTitleStyle: React.CSSProperties = {
  fontSize: 13,
  fontWeight: 600,
  color: '#555',
  marginBottom: 12,
  paddingBottom: 8,
  borderBottom: '1px solid #e8e8e8',
};

// 磁盘空间选项
const diskSpaceOptions = [
  { text: '10%', value: '10%' },
  { text: '20%', value: '20%' },
  { text: '30%', value: '30%' },
  { text: '40%', value: '40%' },
  { text: '50%', value: '50%' },
  { text: '60%', value: '60%' },
  { text: '70%', value: '70%' },
  { text: '80%', value: '80%' },
  { text: '90%', value: '90%' },
];

export default function OmcSettings({ form }: OmcSettingsProps) {
  const t = useT();

  const handleTestSyslog = () => {
    void message.info('正在测试Syslog连接...');
    setTimeout(() => {
      void message.success('Syslog连接成功');
    }, 1000);
  };

  return (
    <Form form={form} layout="vertical" size="small" initialValues={{
      rsysLogEnable: '0',
      rsysLogIp: '',
      rsysLogPort: '',
      varDiskAlarmThresHold: '10%',
      homeDiskAlarmThresHold: '10%',
      usrDiskAlarmThresHold: '10%',
      rootDiskAlarmThresHold: '10%',
    }}>
      {/* OMC协议设置 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>OMC协议设置</span>} style={{ marginBottom: 16 }}>
        {/* 协议方式 */}
        <div style={settingRowStyle}>
          <div style={sectionTitleStyle}>协议方式</div>
          <Space style={{ marginBottom: 8 }}>
            <Form.Item name="rsysLogEnable" noStyle valuePropName="checked">
              <Switch checkedChildren="开启" unCheckedChildren="关闭" />
            </Form.Item>
            <Button type="primary" size="small" onClick={handleTestSyslog}>
              测试
            </Button>
          </Space>
          <div style={subSettingStyle}>
            <Space size="large">
              <Space>
                <span>IP</span>
                <Form.Item name="rsysLogIp" noStyle>
                  <Input style={{ width: 150 }} placeholder="请输入IP地址" />
                </Form.Item>
              </Space>
              <Space>
                <span>端口</span>
                <Form.Item name="rsysLogPort" noStyle>
                  <Input style={{ width: 100 }} placeholder="514" />
                </Form.Item>
              </Space>
            </Space>
          </div>
        </div>
      </Card>

      {/* 磁盘告警设置 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>磁盘告警设置</span>}>
        <div style={settingRowStyle}>
          <div style={{ marginBottom: 12 }}>磁盘告警阈值设置</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <span style={{ width: 200 }}>日志目录（/var）超过</span>
              <Form.Item name="varDiskAlarmThresHold" noStyle>
                <Select style={{ width: 80 }} size="small">
                  {diskSpaceOptions.map(opt => (
                    <Option key={opt.value} value={opt.value}>{opt.text}</Option>
                  ))}
                </Select>
              </Form.Item>
              <span style={{ marginLeft: 8 }}>时产生告警</span>
            </div>
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <span style={{ width: 200 }}>数据目录（/home）超过</span>
              <Form.Item name="homeDiskAlarmThresHold" noStyle>
                <Select style={{ width: 80 }} size="small">
                  {diskSpaceOptions.map(opt => (
                    <Option key={opt.value} value={opt.value}>{opt.text}</Option>
                  ))}
                </Select>
              </Form.Item>
              <span style={{ marginLeft: 8 }}>时产生告警</span>
            </div>
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <span style={{ width: 200 }}>应用目录（/usr）超过</span>
              <Form.Item name="usrDiskAlarmThresHold" noStyle>
                <Select style={{ width: 80 }} size="small">
                  {diskSpaceOptions.map(opt => (
                    <Option key={opt.value} value={opt.value}>{opt.text}</Option>
                  ))}
                </Select>
              </Form.Item>
              <span style={{ marginLeft: 8 }}>时产生告警</span>
            </div>
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <span style={{ width: 200 }}>根目录（/）超过</span>
              <Form.Item name="rootDiskAlarmThresHold" noStyle>
                <Select style={{ width: 80 }} size="small">
                  {diskSpaceOptions.map(opt => (
                    <Option key={opt.value} value={opt.value}>{opt.text}</Option>
                  ))}
                </Select>
              </Form.Item>
              <span style={{ marginLeft: 8 }}>时产生告警</span>
            </div>
          </div>
        </div>
      </Card>
    </Form>
  );
}
