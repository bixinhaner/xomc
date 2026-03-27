import { Form, Input, InputNumber, Checkbox, Select, Card, Space, Divider } from 'antd';
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
  marginLeft: 0,
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

export default function StorageSettings({ form }: StorageSettingsProps) {
  const t = useT();

  return (
    <Form form={form} layout="vertical" size="small" initialValues={{
      logDataSaveDays: 90,
      rebootLogDataSaveDays: 30,
      rebootLogSaveCount: 2,
      sysOperateLogDataSaveDays: 365,
      logFtpEnable: false,
      logFtpType: 'sftp',
      logFtpSavePath: '/var/log/omc',
      logFtpIpAddr: '',
      logFtpPort: 22,
      logFtpUser: '',
      logFtpPassword: '',
      alarmHisMaxHoldTime: 90,
      kpiFilesSaveDays: 30,
      kpiReportDataSaveDays: 90,
      kpiStorge15DataDays: 30,
      kpiStorge60DataDays: 60,
      kpiStorge1440DataDays: 90,
      kpiWeekAndMonthSwitch: false,
      mrFileSaveDays: 30,
      signalingTraceSaveDays: 30,
    }}>
      {/* 日志存储设置 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>日志存储设置</span>} style={{ marginBottom: 16 }}>
        {/* 日志存储 */}
        <div style={settingRowStyle}>
          <div style={sectionTitleStyle}>日志存储</div>
          <Space>
            <span>设备原始文件存储</span>
            <Form.Item name="logDataSaveDays" noStyle>
              <Select style={{ width: 70 }} size="small">
                <Option value={30}>1</Option>
                <Option value={90}>3</Option>
                <Option value={180}>6</Option>
              </Select>
            </Form.Item>
            <span>月</span>
          </Space>
        </div>

        {/* 异常日志存储 */}
        <div style={settingRowStyle}>
          <div style={sectionTitleStyle}>异常日志存储</div>
          <Space wrap style={{ marginBottom: 8 }}>
            <span>设备原始文件存储</span>
            <Form.Item name="rebootLogDataSaveDays" noStyle>
              <Select style={{ width: 70 }} size="small">
                <Option value={1}>1</Option>
                <Option value={7}>7</Option>
                <Option value={30}>30</Option>
                <Option value={60}>60</Option>
                <Option value={90}>90</Option>
              </Select>
            </Form.Item>
            <span>天</span>
          </Space>
          <div style={subSettingStyle}>
            <Space>
              <span>保留异常日志次数</span>
              <Form.Item name="rebootLogSaveCount" noStyle>
                <Select style={{ width: 70 }} size="small">
                  <Option value={1}>1</Option>
                  <Option value={2}>2</Option>
                </Select>
              </Form.Item>
              <span>次，更多日志将覆盖</span>
            </Space>
          </div>
        </div>

        {/* 操作日志 */}
        <div style={settingRowStyle}>
          <div style={sectionTitleStyle}>操作日志</div>
          <Space>
            <span>操作日志存储时长</span>
            <Form.Item name="sysOperateLogDataSaveDays" noStyle>
              <Select style={{ width: 70 }} size="small">
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

        {/* 远程存储 */}
        <div style={settingRowStyle}>
          <div style={sectionTitleStyle}>远程存储</div>
          <Form.Item name="logFtpEnable" valuePropName="checked" noStyle>
            <Checkbox>日志转发到远程地址</Checkbox>
          </Form.Item>
          <div style={subSettingStyle}>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16, marginBottom: 12 }}>
              <Form.Item label="FTP协议" name="logFtpType" style={{ marginBottom: 0 }}>
                <Select style={{ width: 80 }} size="small">
                  <Option value="sftp">SFTP</Option>
                  <Option value="ftp">FTP</Option>
                </Select>
              </Form.Item>
              <Form.Item label="上传路径" name="logFtpSavePath" style={{ marginBottom: 0 }}>
                <Input style={{ width: 280 }} placeholder="/" />
              </Form.Item>
            </div>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16, marginBottom: 12 }}>
              <Form.Item label="IP地址" name="logFtpIpAddr" style={{ marginBottom: 0 }}>
                <Input style={{ width: 140 }} placeholder="请输入IP地址" />
              </Form.Item>
              <Form.Item label="端口" name="logFtpPort" style={{ marginBottom: 0 }}>
                <InputNumber style={{ width: 80 }} />
              </Form.Item>
            </div>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
              <Form.Item label="用户名称" name="logFtpUser" style={{ marginBottom: 0 }}>
                <Input style={{ width: 180 }} maxLength={60} placeholder="请输入用户名" />
              </Form.Item>
              <Form.Item label="密码" name="logFtpPassword" style={{ marginBottom: 0 }}>
                <Input.Password style={{ width: 140 }} placeholder="请输入密码" />
              </Form.Item>
            </div>
          </div>
        </div>
      </Card>

      {/* 数据存储设置 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>数据存储设置</span>}>
        {/* 告警 */}
        <div style={settingRowStyle}>
          <div style={sectionTitleStyle}>告警</div>
          <Space>
            <span>历史告警存储天数，数据库数据存储</span>
            <Form.Item name="alarmHisMaxHoldTime" noStyle>
              <InputNumber style={{ width: 60 }} disabled />
            </Form.Item>
            <span>天</span>
          </Space>
        </div>

        {/* KPI指标 */}
        <div style={settingRowStyle}>
          <div style={sectionTitleStyle}>KPI指标</div>
          <div style={{ marginBottom: 12 }}>
            <Space>
              <span>KPI文件存储天数，设备报告存储</span>
              <Form.Item name="kpiFilesSaveDays" noStyle>
                <InputNumber style={{ width: 60 }} disabled />
              </Form.Item>
              <span>天</span>
            </Space>
          </div>
          <div style={{ marginBottom: 12 }}>
            <Space>
              <span>KPI报表文件存储天数，KPI报表存储</span>
              <Form.Item name="kpiReportDataSaveDays" noStyle>
                <InputNumber style={{ width: 60 }} disabled />
              </Form.Item>
              <span>天</span>
            </Space>
          </div>
          <div style={{ marginBottom: 12 }}>
            <Space>
              <span>KPI原始数据在服务器最多存储</span>
              <Form.Item name="kpiStorge15DataDays" noStyle>
                <InputNumber style={{ width: 60 }} disabled />
              </Form.Item>
              <span>天</span>
            </Space>
          </div>
          <div style={{ marginBottom: 12 }}>
            <Space>
              <span>KPI小时数据在服务器最多存储</span>
              <Form.Item name="kpiStorge60DataDays" noStyle>
                <Select style={{ width: 70 }} size="small">
                  <Option value={30}>30</Option>
                  <Option value={60}>60</Option>
                  <Option value={90}>90</Option>
                </Select>
              </Form.Item>
              <span>天</span>
            </Space>
          </div>
          <div style={{ marginBottom: 12 }}>
            <Space>
              <span>KPI天数据在服务器最多存储</span>
              <Form.Item name="kpiStorge1440DataDays" noStyle>
                <InputNumber style={{ width: 60 }} disabled />
              </Form.Item>
              <span>天</span>
            </Space>
          </div>
          <div>
            <Form.Item name="kpiWeekAndMonthSwitch" valuePropName="checked" noStyle>
              <Checkbox>支持周月查询粒度</Checkbox>
            </Form.Item>
          </div>
        </div>

        {/* MR测量报告 */}
        <div style={settingRowStyle}>
          <div style={sectionTitleStyle}>MR测量报告</div>
          <Space>
            <span>MR存储天数，设备报告原始文件存储</span>
            <Form.Item name="mrFileSaveDays" noStyle>
              <InputNumber style={{ width: 60 }} disabled />
            </Form.Item>
            <span>天</span>
          </Space>
        </div>

        {/* 信令追踪 */}
        <div style={{ ...settingRowStyle, marginBottom: 0 }}>
          <div style={sectionTitleStyle}>信令追踪</div>
          <Space>
            <span>信令追踪文件存储，设备报告存储</span>
            <Form.Item name="signalingTraceSaveDays" noStyle>
              <InputNumber style={{ width: 60 }} disabled />
            </Form.Item>
            <span>天</span>
          </Space>
        </div>
      </Card>
    </Form>
  );
}
