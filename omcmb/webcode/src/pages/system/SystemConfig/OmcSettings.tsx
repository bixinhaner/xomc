import { Form, Input, InputNumber, Switch, Divider, Space, Button, message, Slider, Row, Col } from 'antd';
import { useT } from '@/hooks/useT';

interface OmcSettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
}

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
      rsysLogEnable: false,
      rsysLogIp: '',
      rsysLogPort: 514,
      varDiskAlarmThresHold: 80,
      homeDiskAlarmThresHold: 80,
      usrDiskAlarmThresHold: 80,
      rootDiskAlarmThresHold: 80,
    }}>
      {/* 协议方式（Syslog） */}
      <Divider orientation="left" plain>协议方式（Syslog）</Divider>
      <Form.Item name="rsysLogEnable" label="协议方式开关" valuePropName="checked">
        <Switch checkedChildren="开启" unCheckedChildren="关闭" />
      </Form.Item>
      <Space>
        <Form.Item name="rsysLogIp" label="IP地址" rules={[{ required: true, message: '请输入Syslog服务器IP' }]}>
          <Input placeholder="192.168.1.100" style={{ width: 200 }} />
        </Form.Item>
        <Form.Item name="rsysLogPort" label="端口">
          <InputNumber min={1} max={65535} style={{ width: 120 }} />
        </Form.Item>
        <Form.Item label=" ">
          <Button type="primary" ghost onClick={handleTestSyslog}>
            测试连接
          </Button>
        </Form.Item>
      </Space>

      {/* 磁盘告警 */}
      <Divider orientation="left" plain>磁盘告警</Divider>
      <Row gutter={24}>
        <Col span={12}>
          <Form.Item name="varDiskAlarmThresHold" label="/var 目录磁盘告警阈值">
            <Slider
              min={50}
              max={99}
              marks={{ 50: '50%', 70: '70%', 80: '80%', 90: '90%', 99: '99%' }}
              tooltip={{ formatter: (value) => `${value}%` }}
            />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item name="homeDiskAlarmThresHold" label="/home 目录磁盘告警阈值">
            <Slider
              min={50}
              max={99}
              marks={{ 50: '50%', 70: '70%', 80: '80%', 90: '90%', 99: '99%' }}
              tooltip={{ formatter: (value) => `${value}%` }}
            />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item name="usrDiskAlarmThresHold" label="/usr 目录磁盘告警阈值">
            <Slider
              min={50}
              max={99}
              marks={{ 50: '50%', 70: '70%', 80: '80%', 90: '90%', 99: '99%' }}
              tooltip={{ formatter: (value) => `${value}%` }}
            />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item name="rootDiskAlarmThresHold" label="/ 目录磁盘告警阈值">
            <Slider
              min={50}
              max={99}
              marks={{ 50: '50%', 70: '70%', 80: '80%', 90: '90%', 99: '99%' }}
              tooltip={{ formatter: (value) => `${value}%` }}
            />
          </Form.Item>
        </Col>
      </Row>
    </Form>
  );
}
