import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { Alert, Button, Card, Form, Input, InputNumber, Space, Typography, type FormInstance } from 'antd';
import { useT } from '@/hooks/useT';
import type { ParamConfigDeviceType } from './paramConfigWorkbook';
import GnbQuickSettingsCards, { GnbTemplateExtraFieldGrid } from './GnbQuickSettingsCards';
import EnbQuickSettingsCards, { EnbTemplateExtraFieldGrid } from './EnbQuickSettingsCards';
import { GSM_GROUPED_TEMPLATE_FIELDS, type TemplateFieldRef } from './paramConfigGroupedFields';
import { GNB_COMMON_EXCLUDED_EXTRA_FIELD_IDS } from './gnbQuickSettingsFields';

const { Text } = Typography;

interface AllocationRuleFieldsProps {
  name: 'gnbIdAllocation' | 'pciAllocation';
  title: string;
  maximum: number;
}

function AllocationRuleFields({ name, title, maximum }: AllocationRuleFieldsProps) {
  const t = useT();
  return (
    <Card size="small" title={title} style={{ marginBottom: 16 }}>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: 16 }}>
        <Form.Item
          name={[name, 'start']}
          label={t('provision.rangeStart')}
          rules={[{ required: true }]}
        >
          <InputNumber min={0} max={maximum} style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item
          name={[name, 'end']}
          label={t('provision.rangeEnd')}
          rules={[{ required: true }]}
        >
          <InputNumber min={0} max={maximum} style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item
          name={[name, 'step']}
          label={t('provision.rangeStep')}
          rules={[{ required: true }]}
        >
          <InputNumber min={1} style={{ width: '100%' }} />
        </Form.Item>
      </div>
      <Form.List name={[name, 'reserved']}>
        {(fields, { add, remove }) => (
          <>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: fields.length ? 8 : 0 }}>
              <Text type="secondary">{t('provision.reservedRanges', { field: title })}</Text>
              <Button type="link" icon={<PlusOutlined />} onClick={() => add({ start: 0, end: 0 })}>
                {t('common.add')}
              </Button>
            </div>
            {fields.length === 0 && <Text type="secondary">{t('provision.noReservedRanges')}</Text>}
            {fields.map((field) => (
              <Space key={field.key} align="baseline" style={{ marginRight: 16 }}>
                <Form.Item {...field} name={[field.name, 'start']} rules={[{ required: true }]}>
                  <InputNumber min={0} max={maximum} placeholder={t('provision.rangeStart')} />
                </Form.Item>
                <span>—</span>
                <Form.Item {...field} name={[field.name, 'end']} rules={[{ required: true }]}>
                  <InputNumber min={0} max={maximum} placeholder={t('provision.rangeEnd')} />
                </Form.Item>
                <Button type="text" danger icon={<DeleteOutlined />} onClick={() => remove(field.name)} />
              </Space>
            ))}
          </>
        )}
      </Form.List>
    </Card>
  );
}

function CustomParameters() {
  const t = useT();
  return (
    <Card size="small" title={t('provision.customParams')} style={{ marginBottom: 16 }}>
      <Form.List name="customParams">
        {(fields, { add, remove }) => (
          <>
            {fields.map(({ key, name, ...restField }) => (
              <Card
                key={key}
                size="small"
                title={`${t('provision.customParam')} ${name + 1}`}
                extra={<Button type="link" danger icon={<DeleteOutlined />} onClick={() => remove(name)} />}
                style={{ marginBottom: 12 }}
              >
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: 16 }}>
                  <Form.Item {...restField} name={[name, 'name']} label={t('provision.nrQuick.name')}><Input /></Form.Item>
                  <Form.Item {...restField} name={[name, 'value']} label={t('provision.nrQuick.value')}><Input /></Form.Item>
                  <Form.Item {...restField} name={[name, 'trPath']} label={t('provision.nrQuick.trPath')}><Input /></Form.Item>
                </div>
              </Card>
            ))}
            <Button type="dashed" onClick={() => add()} block icon={<PlusOutlined />}>
              {t('provision.addCustomParam')}
            </Button>
          </>
        )}
      </Form.List>
    </Card>
  );
}

function GsmFields({ fields }: { fields: readonly TemplateFieldRef[] }) {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: 16 }}>
      {fields.map(({ sheet, header }) => (
        <Form.Item key={`${sheet}.${header}`} name={['sheetParameters', sheet, 0, header]} label={header.replace(/^\*/, '')}>
          <Input />
        </Form.Item>
      ))}
    </div>
  );
}

export interface CommonParameterConfigPanelProps {
  form: FormInstance;
  deviceType?: ParamConfigDeviceType;
  paramModelName?: string;
  disabled?: boolean;
}

export default function CommonParameterConfigPanel({
  form,
  deviceType,
  paramModelName,
  disabled = false,
}: CommonParameterConfigPanelProps) {
  const t = useT();
  return (
    <Form form={form} component={false} disabled={disabled}>
      <Alert
        type="info"
        showIcon
        title={t('provision.commonParamConfigHint')}
        style={{ marginBottom: 16 }}
      />
      {deviceType === 'gNB' && (
        <>
          <Card size="small" title={t('provision.batchAllocationRules')} style={{ marginBottom: 16 }}>
            <Text type="secondary">{t('provision.batchAllocationRulesHint')}</Text>
          </Card>
          <AllocationRuleFields name="gnbIdAllocation" title="gNB ID" maximum={4_294_967_295} />
          <AllocationRuleFields name="pciAllocation" title="PCI" maximum={1007} />
          <GnbQuickSettingsCards excludedFieldIds={['gNBId', 'PCI']} />
          <Card size="small" title={t('provision.otherTemplateParams')} style={{ marginBottom: 16 }}>
            <GnbTemplateExtraFieldGrid excludedFieldIds={GNB_COMMON_EXCLUDED_EXTRA_FIELD_IDS} />
          </Card>
          <CustomParameters />
        </>
      )}
      {deviceType === 'eNB' && (
        <>
          <EnbQuickSettingsCards paramModelName={paramModelName} />
          <Card size="small" title={t('provision.otherTemplateParams')} style={{ marginBottom: 16 }}>
            <EnbTemplateExtraFieldGrid />
          </Card>
          <CustomParameters />
        </>
      )}
      {deviceType === 'GSM' && (
        <>
          <Card size="small" title={t('provision.gsmBasicConfig')} style={{ marginBottom: 16 }}>
            <GsmFields fields={GSM_GROUPED_TEMPLATE_FIELDS.quickAbis} />
          </Card>
          <Card size="small" title={t('provision.otherParams')} style={{ marginBottom: 16 }}>
            <GsmFields fields={GSM_GROUPED_TEMPLATE_FIELDS.other} />
          </Card>
          <CustomParameters />
        </>
      )}
      {!deviceType && <Alert type="warning" showIcon title={t('provision.paramConfigDeviceTypeUnavailable')} />}
    </Form>
  );
}
