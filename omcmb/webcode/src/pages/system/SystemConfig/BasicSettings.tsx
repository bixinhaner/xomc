import { Form, Input, Select, Card } from 'antd';
import { useT } from '@/hooks/useT';

interface BasicSettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
}

// Row spacing for settings
const settingRowStyle: React.CSSProperties = {
  marginBottom: 16,
};

export default function BasicSettings({ form }: BasicSettingsProps) {
  const t = useT();

  // Timezone options
  const timezoneOptions = [
    { label: t('system.basic.tz.shanghai'), value: 'Asia/Shanghai' },
    { label: t('system.basic.tz.utc'), value: 'UTC' },
    { label: t('system.basic.tz.newYork'), value: 'America/New_York' },
    { label: t('system.basic.tz.losAngeles'), value: 'America/Los_Angeles' },
    { label: t('system.basic.tz.paris'), value: 'Europe/Paris' },
    { label: t('system.basic.tz.tokyo'), value: 'Asia/Tokyo' },
  ];

  return (
    <Form form={form} layout="vertical" size="small" initialValues={{
      mrVendor: '',
      mrOMCName: '',
      timezoneCode: '',
    }}>
      {/* 基本信息 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.basic.info')}</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Form.Item
            name="mrVendor"
            label={t('system.basic.operatorName')}
            rules={[{ max: 50, message: t('common.max50Chars') }]}
            style={{ marginBottom: 0 }}
          >
            <Input placeholder={t('system.basic.pleaseInputOperatorName')} maxLength={50} style={{ width: 300 }} />
          </Form.Item>
        </div>

        <div style={settingRowStyle}>
          <Form.Item
            name="mrOMCName"
            label={t('system.basic.omcName')}
            rules={[{ max: 200, message: t('common.max200Chars') }]}
            style={{ marginBottom: 0 }}
          >
            <Input placeholder={t('system.basic.pleaseInputOmcName')} maxLength={200} style={{ width: 350 }} />
          </Form.Item>
        </div>
      </Card>

      {/* 系统设置 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.basic.systemSettings')}</span>}>
        <Form.Item
          name="timezoneCode"
          label={t('system.basic.timezoneSetting')}
          style={{ marginBottom: 0 }}
        >
          <Select placeholder={t('system.basic.pleaseSelectTimezone')} options={timezoneOptions} style={{ width: 400 }} />
        </Form.Item>
      </Card>
    </Form>
  );
}
