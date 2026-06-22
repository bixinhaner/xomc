import { Form, Input, InputNumber, Checkbox, Select, Card, Space, theme } from 'antd';
import { useT } from '@/hooks/useT';

const { Option } = Select;

interface StorageSettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
}

// 设置行样式
const settingRowStyle: React.CSSProperties = {
  marginBottom: 16,
};

// 子设置区域布局（颜色随主题，见组件内 subSettingStyle）
const subSettingBaseStyle: React.CSSProperties = {
  marginTop: 12,
  padding: '12px 16px',
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
  const { token } = theme.useToken();
  // 子设置区域背景跟随明/暗主题，不再写死 #fafafa
  const subSettingStyle: React.CSSProperties = { ...subSettingBaseStyle, backgroundColor: token.colorFillAlter };

  return (
    <Form form={form} layout="vertical" size="small" initialValues={{
      logDataSaveDays: 90,
      rebootLogDataSaveDays: 60,
      rebootLogSaveCount: 2,
      sysOperateLogDataSaveDays: 90,
      // MinIO 对外可达 endpoint：issue #548 切片 3。空 = 走 env / 派生回退（后端订阅桥处理）
      minio_public_endpoint: '',
      alarmHisMaxHoldTime: 365,
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
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.storage.logSettings')}</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space>
            <span>{t('sysconfig.storage.rawFileLabel')}</span>
            <Form.Item name="logDataSaveDays" noStyle>
              <Select style={{ width: 70 }}>
                <Option value={30}>1</Option>
                <Option value={90}>3</Option>
                <Option value={180}>6</Option>
              </Select>
            </Form.Item>
            <span>{t('sysconfig.storage.unit.month')}</span>
          </Space>
        </div>

        <div style={settingRowStyle}>
          <Space>
            <span>{t('sysconfig.storage.errorLogLabel')}</span>
            <Form.Item name="rebootLogDataSaveDays" noStyle>
              <Select style={{ width: 70 }}>
                <Option value={1}>1</Option>
                <Option value={7}>7</Option>
                <Option value={30}>30</Option>
                <Option value={60}>60</Option>
                <Option value={90}>90</Option>
              </Select>
            </Form.Item>
            <span>{t('sysconfig.storage.unit.day')}</span>
          </Space>
        </div>

        <div style={settingRowStyle}>
          <Space>
            <span>{t('sysconfig.storage.devErrorLogPrefix')}</span>
            <Form.Item name="rebootLogSaveCount" noStyle>
              <Select style={{ width: 70 }}>
                <Option value={1}>1</Option>
                <Option value={2}>2</Option>
                <Option value={3}>3</Option>
                <Option value={4}>4</Option>
                <Option value={5}>5</Option>
              </Select>
            </Form.Item>
            <span>{t('sysconfig.storage.devErrorLogSuffix')}</span>
          </Space>
        </div>

        <div style={settingRowStyle}>
          <Space>
            <span>{t('sysconfig.storage.userOpLogLabel')}</span>
            <Form.Item name="sysOperateLogDataSaveDays" noStyle>
              <Select style={{ width: 70 }}>
                <Option value={90}>3</Option>
                <Option value={180}>6</Option>
                <Option value={360}>12</Option>
                <Option value={720}>24</Option>
                <Option value={1080}>36</Option>
              </Select>
            </Form.Item>
            <span>{t('sysconfig.storage.unit.month')}</span>
          </Space>
        </div>

      </Card>

      {/* MinIO 对象存储 — issue #548 切片 3 收敛：
           只暴露 `MinIO 对外可达 endpoint`（sys_configs.storage.minio_public_endpoint）
           一字段；其他 endpoint/port/accessKey/secret/bucket/region/pathStyle/useSSL/enable
           都是部署期决策（docker compose / yaml）不该 UI 编辑——改了不生效就是 §5
           设计原则禁止的"打字进数据库不生效"半成品。详见 issue-548-slice3-ledger.md。 */}
      <Card
        size="small"
        title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.storage.minio')}</span>}
        style={{ marginBottom: 16 }}
      >
        <div style={subSettingStyle}>
          <Form.Item
            label={t('system.storage.minioPublicEndpoint')}
            name="minio_public_endpoint"
            // 后端 storage.minio_public_endpoint validator 已对相同正则校验并返 400；
            // 此处加前端 pattern 让用户在输入即时看到错误，避免一次保存才得知。
            // 规则与后端 ValidatePublicEndpoint 同步：禁 scheme/path/IPv6，允许空。
            rules={[{
              validator: (_, value: string) => {
                if (!value) return Promise.resolve();
                if (value.includes('://')) return Promise.reject(new Error(t('system.storage.minioPublicEndpointErrScheme')));
                if (/[/?#]/.test(value)) return Promise.reject(new Error(t('system.storage.minioPublicEndpointErrPath')));
                if (/[[\]]/.test(value) || (value.match(/:/g) || []).length > 1) {
                  return Promise.reject(new Error(t('system.storage.minioPublicEndpointErrIpv6')));
                }
                const m = value.match(/^([^:]+)(?::(\d+))?$/);
                if (!m || !m[1]) return Promise.reject(new Error(t('system.storage.minioPublicEndpointErrFormat')));
                if (m[2]) {
                  const p = parseInt(m[2], 10);
                  if (!Number.isFinite(p) || p < 1 || p > 65535) {
                    return Promise.reject(new Error(t('system.storage.minioPublicEndpointErrPort')));
                  }
                }
                return Promise.resolve();
              },
            }]}
            extra={<span style={{ color: token.colorTextSecondary }}>{t('system.storage.minioPublicEndpointDesc')}</span>}
          >
            <Input
              style={{ width: 360 }}
              placeholder={t('system.storage.minioPublicEndpointPlaceholder')}
              allowClear
              maxLength={253}
            />
          </Form.Item>
        </div>
      </Card>

      {/* 告警 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.storage.alarm')}</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space>
            <span>{t('sysconfig.storage.alarmHistoryLabel')}</span>
            <Form.Item name="alarmHisMaxHoldTime" noStyle>
              <InputNumber min={1} max={365} style={{ width: 70 }} />
            </Form.Item>
            <span>{t('sysconfig.storage.unit.day')}</span>
          </Space>
        </div>
      </Card>

      {/* KPI */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.storage.kpi')}</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space>
            <span>{t('sysconfig.storage.kpiFileLabel')}</span>
            <Form.Item name="kpiFilesSaveDays" noStyle>
              <InputNumber min={1} max={365} style={{ width: 70 }} disabled />
            </Form.Item>
            <span>{t('sysconfig.storage.unit.day')}</span>
          </Space>
        </div>

        <div style={settingRowStyle}>
          <Space>
            <span>{t('sysconfig.storage.kpiReportLabel')}</span>
            <Form.Item name="kpiReportDataSaveDays" noStyle>
              <InputNumber min={1} max={365} style={{ width: 70 }} disabled />
            </Form.Item>
            <span>{t('sysconfig.storage.unit.day')}</span>
          </Space>
        </div>

        <div style={settingRowStyle}>
          <Space>
            <span>{t('sysconfig.storage.kpiRawLabel')}</span>
            <Form.Item name="kpiStorge15DataDays" noStyle>
              <InputNumber min={1} max={365} style={{ width: 70 }} disabled />
            </Form.Item>
            <span>{t('sysconfig.storage.unit.day')}</span>
          </Space>
        </div>

        <div style={settingRowStyle}>
          <Space>
            <span>{t('sysconfig.storage.kpiHourLabel')}</span>
            <Form.Item name="kpiStorge60DataDays" noStyle>
              <InputNumber min={1} max={365} style={{ width: 70 }} />
            </Form.Item>
            <span>{t('sysconfig.storage.unit.day')}</span>
          </Space>
        </div>

        <div style={settingRowStyle}>
          <Space>
            <span>{t('sysconfig.storage.kpiDayLabel')}</span>
            <Form.Item name="kpiStorge1440DataDays" noStyle>
              <InputNumber min={1} max={730} style={{ width: 70 }} disabled />
            </Form.Item>
            <span>{t('sysconfig.storage.unit.day')}</span>
          </Space>
        </div>

        <div style={settingRowStyle}>
          <Form.Item name="kpiWeekAndMonthSwitch" valuePropName="checked" noStyle>
            <Checkbox>{t('sysconfig.storage.weekMonthGranularity')}</Checkbox>
          </Form.Item>
        </div>
      </Card>

      {/* MR */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.storage.mr')}</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space>
            <span>{t('sysconfig.storage.mrRawLabel')}</span>
            <Form.Item name="mrFileSaveDays" noStyle>
              <InputNumber min={1} max={365} style={{ width: 70 }} disabled />
            </Form.Item>
            <span>{t('sysconfig.storage.unit.day')}</span>
          </Space>
        </div>
      </Card>

      {/* 信令追踪 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.storage.signalingTrace')}</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space>
            <span>{t('sysconfig.storage.traceLabel')}</span>
            <Form.Item name="signalingTraceSaveDays" noStyle>
              <InputNumber min={1} max={365} style={{ width: 70 }} disabled />
            </Form.Item>
            <span>{t('sysconfig.storage.unit.day')}</span>
          </Space>
        </div>
      </Card>

      {/* 磁盘告警 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.storage.diskAlarm')}</span>}>
        <div style={settingRowStyle}>
          <Space>
            <span>{t('sysconfig.storage.diskLog')}</span>
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
            <span>{t('sysconfig.storage.diskData')}</span>
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
            <span>{t('sysconfig.storage.diskApp')}</span>
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
            <span>{t('sysconfig.storage.diskRoot')}</span>
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
