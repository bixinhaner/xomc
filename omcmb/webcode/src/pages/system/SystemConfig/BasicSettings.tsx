import { Form, Input, Select, Card, Space } from 'antd';
import { useT } from '@/hooks/useT';

const { Option } = Select;

interface BasicSettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
}

// 时区选项
const timezoneOptions = [
  { label: '(GMT+08:00) 北京, 重庆, 香港, 乌鲁木齐', value: 'Asia/Shanghai' },
  { label: '(GMT+00:00) 格林威治标准时间', value: 'UTC' },
  { label: '(GMT-05:00) 美国东部时间', value: 'America/New_York' },
  { label: '(GMT-08:00) 美国太平洋时间', value: 'America/Los_Angeles' },
  { label: '(GMT+01:00) 中欧时间', value: 'Europe/Paris' },
  { label: '(GMT+09:00) 日本时间', value: 'Asia/Tokyo' },
];

// 语言选项
const languageOptions = [
  { label: '中文', value: 'zh' },
  { label: 'English', value: 'en' },
];

// 设置行样式
const settingRowStyle: React.CSSProperties = {
  marginBottom: 16,
};

export default function BasicSettings({ form }: BasicSettingsProps) {
  const t = useT();

  return (
    <Form form={form} layout="vertical" size="small" initialValues={{
      mrVendor: '',
      mrOMCName: 'OMC',
      timezoneCode: 'Asia/Shanghai',
      languageCode: 'zh',
    }}>
      {/* 基本信息 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>基本信息</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Form.Item
            name="mrVendor"
            label="运营商名称"
            rules={[{ max: 50, message: '最大50个字符' }]}
            style={{ marginBottom: 0 }}
          >
            <Input placeholder="请输入运营商名称" maxLength={50} style={{ width: 300 }} />
          </Form.Item>
        </div>

        <div style={settingRowStyle}>
          <Form.Item
            name="mrOMCName"
            label="OMC名称"
            rules={[{ max: 200, message: '最大200个字符' }]}
            style={{ marginBottom: 0 }}
          >
            <Input placeholder="请输入网管系统名称" maxLength={200} style={{ width: 350 }} />
          </Form.Item>
        </div>
      </Card>

      {/* 系统设置 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>系统设置</span>}>
        <div style={settingRowStyle}>
          <Form.Item
            name="timezoneCode"
            label="时区设置"
            style={{ marginBottom: 0 }}
          >
            <Select placeholder="请选择系统时区" options={timezoneOptions} style={{ width: 400 }} />
          </Form.Item>
        </div>

        <div style={settingRowStyle}>
          <Form.Item
            name="languageCode"
            label="语言设置"
            style={{ marginBottom: 0 }}
          >
            <Select placeholder="请选择系统语言" options={languageOptions} style={{ width: 120 }} />
          </Form.Item>
        </div>
      </Card>
    </Form>
  );
}
