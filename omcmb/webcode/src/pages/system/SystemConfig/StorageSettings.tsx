import { Form, Input, InputNumber, Checkbox, Select, Card, Space } from 'antd';
import { useT } from '@/hooks/useT';

const { Option } = Select;

interface StorageSettingsProps {
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

export default function StorageSettings({ form }: StorageSettingsProps) {
  const t = useT();

  // 监听FTP转发复选框状态
  const logFtpEnable = Form.useWatch('logFtpEnable', form);

  return (
    <Form form={form} layout="vertical" size="small" initialValues={{
      logDataSaveDays: 90,
      rebootLogDataSaveDays: 60,
      rebootLogSaveCount: 2,
      sysOperateLogDataSaveDays: 90,
      logFtpEnable: false,
      logFtpType: 'ftp',
      logFtpSavePath: '/',
      logFtpIpAddr: '127.0.0.1',
      logFtpPort: 21,
      logFtpUser: 'ftpuser',
      logFtpPassword: '',
      alarmHisMaxHoldTime: 90,
      kpiFilesSaveDays: 7,
      kpiReportDataSaveDays: 7,
      kpiStorge15DataDays: 30,
      kpiStorge60DataDays: 30,
      kpiStorge1440DataDays: 365,
      kpiWeekAndMonthSwitch: true,
      mrFileSaveDays: 3,
      signalingTraceSaveDays: 7,
      varDiskAlarmThresHold: '10%',
      homeDiskAlarmThresHold: '10%',
      usrDiskAlarmThresHold: '10%',
      rootDiskAlarmThresHold: '10%',
    }}>
      {/* 日志设置 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>日志设置</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space>
            <span>原始文件：设备上报的源文件将存储</span>
            <Form.Item name="logDataSaveDays" noStyle>
              <Select style={{ width: 70 }}>
                <Option value={30}>1</Option>
                <Option value={90}>3</Option>
                <Option value={180}>6</Option>
              </Select>
            </Form.Item>
            <span>月</span>
          </Space>
        </div>

        <div style={settingRowStyle}>
          <Space>
            <span>异常日志存储：设备上报的源文件将存储</span>
            <Form.Item name="rebootLogDataSaveDays" noStyle>
              <Select style={{ width: 70 }}>
                <Option value={1}>1</Option>
                <Option value={7}>7</Option>
                <Option value={30}>30</Option>
                <Option value={60}>60</Option>
                <Option value={90}>90</Option>
              </Select>
            </Form.Item>
            <span>天</span>
          </Space>
        </div>

        <div style={settingRowStyle}>
          <Space>
            <span>每个设备最多保留最近</span>
            <Form.Item name="rebootLogSaveCount" noStyle>
              <Select style={{ width: 70 }}>
                <Option value={1}>1</Option>
                <Option value={2}>2</Option>
                <Option value={3}>3</Option>
                <Option value={4}>4</Option>
                <Option value={5}>5</Option>
              </Select>
            </Form.Item>
            <span>次异常日志，更多的日志则将覆盖最早的那次</span>
          </Space>
        </div>

        <div style={settingRowStyle}>
          <Space>
            <span>用户操作日志将存储</span>
            <Form.Item name="sysOperateLogDataSaveDays" noStyle>
              <Select style={{ width: 70 }}>
                <Option value={90}>3</Option>
                <Option value={180}>6</Option>
                <Option value={360}>12</Option>
                <Option value={720}>24</Option>
                <Option value={1080}>36</Option>
              </Select>
            </Form.Item>
            <span>月</span>
          </Space>
        </div>

        <div style={settingRowStyle}>
          <Form.Item name="logFtpEnable" valuePropName="checked" noStyle>
            <Checkbox>日志文件将转发到远端地址</Checkbox>
          </Form.Item>
          <div style={subSettingStyle}>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16, marginBottom: 12 }}>
              <Form.Item label="FTP协议" name="logFtpType" style={{ marginBottom: 0 }}>
                <Select style={{ width: 100 }} disabled={!logFtpEnable}>
                  <Option value="ftp">FTP</Option>
                  <Option value="sftp">SFTP</Option>
                </Select>
              </Form.Item>
              <Form.Item label="上传路径" name="logFtpSavePath" style={{ marginBottom: 0 }}>
                <Input style={{ width: 200 }} disabled={!logFtpEnable} />
              </Form.Item>
            </div>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16, marginBottom: 12 }}>
              <Form.Item label="IP地址" name="logFtpIpAddr" style={{ marginBottom: 0 }}>
                <Input style={{ width: 140 }} disabled={!logFtpEnable} />
              </Form.Item>
              <Form.Item label="端口" name="logFtpPort" style={{ marginBottom: 0 }}>
                <InputNumber style={{ width: 80 }} disabled={!logFtpEnable} />
              </Form.Item>
            </div>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
              <Form.Item label="用户名" name="logFtpUser" style={{ marginBottom: 0 }}>
                <Input style={{ width: 160 }} disabled={!logFtpEnable} />
              </Form.Item>
              <Form.Item label="密码" name="logFtpPassword" style={{ marginBottom: 0 }}>
                <Input.Password style={{ width: 140 }} disabled={!logFtpEnable} />
              </Form.Item>
            </div>
          </div>
        </div>
      </Card>

      {/* 告警 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>告警</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space>
            <span>历史告警存储：历史告警在数据库最多存储</span>
            <Form.Item name="alarmHisMaxHoldTime" noStyle>
              <InputNumber min={1} max={365} style={{ width: 70 }} />
            </Form.Item>
            <span>天</span>
          </Space>
        </div>
      </Card>

      {/* KPI */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>KPI</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space>
            <span>KPI文件存储：设备上报的原始文件在服务器最多存储</span>
            <Form.Item name="kpiFilesSaveDays" noStyle>
              <InputNumber min={1} max={365} style={{ width: 70 }} />
            </Form.Item>
            <span>天</span>
          </Space>
        </div>

        <div style={settingRowStyle}>
          <Space>
            <span>KPI报表文件存储：根据KPI查询模板生成的报表文件将在服务器最多存储</span>
            <Form.Item name="kpiReportDataSaveDays" noStyle>
              <InputNumber min={1} max={365} style={{ width: 70 }} />
            </Form.Item>
            <span>天</span>
          </Space>
        </div>

        <div style={settingRowStyle}>
          <Space>
            <span>KPI原始数据存储：KPI原始数据在服务器最多存储</span>
            <Form.Item name="kpiStorge15DataDays" noStyle>
              <InputNumber min={1} max={365} style={{ width: 70 }} />
            </Form.Item>
            <span>天</span>
          </Space>
        </div>

        <div style={settingRowStyle}>
          <Space>
            <span>KPI小时数据存储：KPI小时数据在服务器最多存储</span>
            <Form.Item name="kpiStorge60DataDays" noStyle>
              <InputNumber min={1} max={365} style={{ width: 70 }} />
            </Form.Item>
            <span>天</span>
          </Space>
        </div>

        <div style={settingRowStyle}>
          <Space>
            <span>KPI天数据存储：KPI天数据在服务器最多存储</span>
            <Form.Item name="kpiStorge1440DataDays" noStyle>
              <InputNumber min={1} max={730} style={{ width: 70 }} />
            </Form.Item>
            <span>天</span>
          </Space>
        </div>

        <div style={settingRowStyle}>
          <Form.Item name="kpiWeekAndMonthSwitch" valuePropName="checked" noStyle>
            <Checkbox>支持周和月统计粒度</Checkbox>
          </Form.Item>
        </div>
      </Card>

      {/* MR */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>MR</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space>
            <span>MR原始文件存储：设备上报的原始文件将在服务器最多存储</span>
            <Form.Item name="mrFileSaveDays" noStyle>
              <InputNumber min={1} max={365} style={{ width: 70 }} />
            </Form.Item>
            <span>天</span>
          </Space>
        </div>
      </Card>

      {/* 信令追踪 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>信令追踪</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space>
            <span>信令追踪文件存储：设备上报的原始文件在服务器最多存储</span>
            <Form.Item name="signalingTraceSaveDays" noStyle>
              <InputNumber min={1} max={365} style={{ width: 70 }} />
            </Form.Item>
            <span>天</span>
          </Space>
        </div>
      </Card>

      {/* 磁盘告警 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>磁盘告警</span>}>
        <div style={settingRowStyle}>
          <Space>
            <span>日志目录磁盘可用存储百分比</span>
            <Form.Item name="varDiskAlarmThresHold" noStyle>
              <Select style={{ width: 100 }}>
                {diskSpaceOptions.map(opt => (
                  <Option key={opt.value} value={opt.value}>{opt.text}</Option>
                ))}
              </Select>
            </Form.Item>
          </Space>
        </div>

        <div style={settingRowStyle}>
          <Space>
            <span>数据目录磁盘可用存储百分比</span>
            <Form.Item name="homeDiskAlarmThresHold" noStyle>
              <Select style={{ width: 100 }}>
                {diskSpaceOptions.map(opt => (
                  <Option key={opt.value} value={opt.value}>{opt.text}</Option>
                ))}
              </Select>
            </Form.Item>
          </Space>
        </div>

        <div style={settingRowStyle}>
          <Space>
            <span>应用目录磁盘可用存储百分比</span>
            <Form.Item name="usrDiskAlarmThresHold" noStyle>
              <Select style={{ width: 100 }}>
                {diskSpaceOptions.map(opt => (
                  <Option key={opt.value} value={opt.value}>{opt.text}</Option>
                ))}
              </Select>
            </Form.Item>
          </Space>
        </div>

        <div style={settingRowStyle}>
          <Space>
            <span>根目录磁盘可用存储百分比</span>
            <Form.Item name="rootDiskAlarmThresHold" noStyle>
              <Select style={{ width: 100 }}>
                {diskSpaceOptions.map(opt => (
                  <Option key={opt.value} value={opt.value}>{opt.text}</Option>
                ))}
              </Select>
            </Form.Item>
          </Space>
        </div>
      </Card>
    </Form>
  );
}
