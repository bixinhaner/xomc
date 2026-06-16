import { useMemo } from 'react';
import { Form, Input, Select, Card } from 'antd';
import { useT } from '@/hooks/useT';
import { buildTimezoneOptions } from '@core/utils/timezoneOptions';

interface BasicSettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
}

// Row spacing for settings
const settingRowStyle: React.CSSProperties = {
  marginBottom: 16,
};

export default function BasicSettings({ form }: BasicSettingsProps) {
  const t = useT();

  // 时区下拉：完整 IANA 列表（带 GMT 偏移、按偏移排序），生成逻辑在 frontend-core 共享。
  // 字段仍是 timezoneCode（IANA 名），保存链路不变。
  const timezoneOptions = useMemo(() => buildTimezoneOptions(), []);

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
          <Select
            placeholder={t('system.basic.pleaseSelectTimezone')}
            options={timezoneOptions}
            style={{ width: 400 }}
            showSearch
            optionFilterProp="label"
          />
        </Form.Item>
      </Card>
    </Form>
  );
}
