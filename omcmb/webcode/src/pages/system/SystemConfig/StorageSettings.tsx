import { Form, Input, InputNumber, Switch, Select, Divider, Space, Collapse } from 'antd';
import { useT } from '@/hooks/useT';

const { Option } = Select;
const { Panel } = Collapse;

interface StorageSettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
}

export default function StorageSettings({ form }: StorageSettingsProps) {
  const t = useT();

  return (
    <Form form={form} layout="vertical" size="small" initialValues={{
      logDataSaveDays: 90,
      rebootLogDataSaveDays: 30,
      rebootLogSaveCount: 2,
      sysOperateLogDataSaveDays: 365,
      logFtpEnable: false,
      logFtpType: 'SFTP',
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
      <Collapse defaultActiveKey={['log', 'alarm', 'kpi', 'mr', 'signaling']} bordered={false}>
        {/* 日志设置 */}
        <Panel header="日志设置" key="log">
          <Form.Item name="logDataSaveDays" label="设备原始文件存储">
            <Select style={{ width: 200 }}>
              <Option value={30}>1个月</Option>
              <Option value={90}>3个月</Option>
              <Option value={180}>6个月</Option>
            </Select>
          </Form.Item>
          <Form.Item name="rebootLogDataSaveDays" label="异常日志存储">
            <Select style={{ width: 200 }}>
              <Option value={1}>1天</Option>
              <Option value={7}>7天</Option>
              <Option value={30}>30天</Option>
              <Option value={60}>60天</Option>
              <Option value={90}>90天</Option>
            </Select>
          </Form.Item>
          <Form.Item name="rebootLogSaveCount" label="保留异常日志次数">
            <Select style={{ width: 200 }}>
              <Option value={1}>1次</Option>
              <Option value={2}>2次</Option>
            </Select>
          </Form.Item>
          <Form.Item name="sysOperateLogDataSaveDays" label="操作日志存储时长">
            <Select style={{ width: 200 }}>
              <Option value={90}>3个月</Option>
              <Option value={180}>6个月</Option>
              <Option value={365}>12个月</Option>
              <Option value={730}>24个月</Option>
              <Option value={1095}>36个月</Option>
            </Select>
          </Form.Item>

          {/* 远程存储 */}
          <Divider orientation="left" plain>远程存储</Divider>
          <Form.Item name="logFtpEnable" label="转发到远程地址" valuePropName="checked">
            <Switch checkedChildren="开启" unCheckedChildren="关闭" />
          </Form.Item>
          <Space>
            <Form.Item name="logFtpType" label="FTP协议">
              <Select style={{ width: 120 }}>
                <Option value="SFTP">SFTP</Option>
                <Option value="FTP">FTP</Option>
              </Select>
            </Form.Item>
            <Form.Item name="logFtpPort" label="端口">
              <InputNumber min={1} max={65535} style={{ width: 120 }} />
            </Form.Item>
          </Space>
          <Form.Item name="logFtpIpAddr" label="IP地址">
            <Input placeholder="远程服务器IP" style={{ width: 200 }} />
          </Form.Item>
          <Form.Item name="logFtpSavePath" label="上传路径">
            <Input placeholder="/var/log/omc" />
          </Form.Item>
          <Space>
            <Form.Item name="logFtpUser" label="用户名">
              <Input placeholder="登录用户名" style={{ width: 200 }} />
            </Form.Item>
            <Form.Item name="logFtpPassword" label="密码">
              <Input.Password placeholder="登录密码" style={{ width: 200 }} />
            </Form.Item>
          </Space>
        </Panel>

        {/* 告警设置 */}
        <Panel header="告警设置" key="alarm">
          <Form.Item name="alarmHisMaxHoldTime" label="历史告警存储天数" extra="数据库数据存储天数">
            <InputNumber min={1} max={365} addonAfter="天" style={{ width: 150 }} />
          </Form.Item>
        </Panel>

        {/* 指标设置 */}
        <Panel header="指标设置" key="kpi">
          <Form.Item name="kpiFilesSaveDays" label="KPI文件存储天数">
            <InputNumber min={1} max={365} addonAfter="天" style={{ width: 150 }} />
          </Form.Item>
          <Form.Item name="kpiReportDataSaveDays" label="KPI报表文件存储天数">
            <InputNumber min={1} max={365} addonAfter="天" style={{ width: 150 }} />
          </Form.Item>
          <Form.Item name="kpiStorge15DataDays" label="KPI原始数据存储" extra="15分钟粒度数据存储天数">
            <InputNumber min={1} max={365} addonAfter="天" style={{ width: 150 }} />
          </Form.Item>
          <Form.Item name="kpiStorge60DataDays" label="KPI小时数据存储">
            <Select style={{ width: 200 }}>
              <Option value={30}>30天</Option>
              <Option value={60}>60天</Option>
              <Option value={90}>90天</Option>
            </Select>
          </Form.Item>
          <Form.Item name="kpiStorge1440DataDays" label="KPI天数据存储">
            <InputNumber min={1} max={365} addonAfter="天" style={{ width: 150 }} />
          </Form.Item>
          <Form.Item name="kpiWeekAndMonthSwitch" label="KPI周月查询粒度" valuePropName="checked" extra="支持周月查询粒度开关">
            <Switch checkedChildren="开启" unCheckedChildren="关闭" />
          </Form.Item>
        </Panel>

        {/* MR设置 */}
        <Panel header="MR设置" key="mr">
          <Form.Item name="mrFileSaveDays" label="MR存储天数" extra="设备报告原始文件存储天数">
            <InputNumber min={1} max={365} addonAfter="天" style={{ width: 150 }} />
          </Form.Item>
        </Panel>

        {/* 心令追踪设置 */}
        <Panel header="心令追踪设置" key="signaling">
          <Form.Item name="signalingTraceSaveDays" label="心令追踪文件存储" extra="设备报告存储天数">
            <InputNumber min={1} max={365} addonAfter="天" style={{ width: 150 }} />
          </Form.Item>
        </Panel>
      </Collapse>
    </Form>
  );
}
