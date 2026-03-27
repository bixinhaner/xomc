import { Form, Input, Select, Divider } from 'antd';
import { useT } from '@/hooks/useT';

const { Option } = Select;

// 时区选项
const timezoneOptions = [
  { label: 'UTC+8 北京时间', value: 'Asia/Shanghai' },
  { label: 'UTC+0 格林威治时间', value: 'UTC' },
  { label: 'UTC-5 美国东部时间', value: 'America/New_York' },
  { label: 'UTC-8 美国太平洋时间', value: 'America/Los_Angeles' },
  { label: 'UTC+1 中欧时间', value: 'Europe/Paris' },
  { label: 'UTC+9 日本时间', value: 'Asia/Tokyo' },
];

// 语言选项
const languageOptions = [
  { label: '中文', value: 'zh_CN' },
  { label: 'English', value: 'en_US' },
];

interface BasicSettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
}

export default function BasicSettings({ form }: BasicSettingsProps) {
  const t = useT();

  return (
    <Form form={form} layout="vertical" initialValues={{
      mrVendor: '',
      mrOMCName: 'OMC',
      timezoneCode: 'Asia/Shanghai',
      languageCode: 'zh_CN',
    }}>
      <Divider orientation="left" plain>基本设置</Divider>
      <Form.Item name="mrVendor" label="运营商名称" rules={[{ max: 50, message: '最大50个字符' }]}>
        <Input placeholder="请输入运营商名称" maxLength={50} />
      </Form.Item>
      <Form.Item name="mrOMCName" label="OMC名称" rules={[{ max: 200, message: '最大200个字符' }]}>
        <Input placeholder="请输入网管系统名称" maxLength={200} />
      </Form.Item>
      <Form.Item name="timezoneCode" label="时区设置">
        <Select placeholder="请选择系统时区" options={timezoneOptions} />
      </Form.Item>
      <Form.Item name="languageCode" label="语言设置">
        <Select placeholder="请选择系统语言" options={languageOptions} />
      </Form.Item>
    </Form>
  );
}
