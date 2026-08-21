import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { Alert, Button, Card, Form, Input, InputNumber, Space, Typography, type FormInstance } from 'antd';
import { useT } from '@/hooks/useT';
import type { ParamConfigDeviceType } from './paramConfigWorkbook';
import GnbQuickSettingsCards, { GnbTemplateExtraFieldGrid } from './GnbQuickSettingsCards';
import EnbQuickSettingsCards, { EnbTemplateExtraFieldGrid } from './EnbQuickSettingsCards';
import { GNB_COMMON_EXCLUDED_EXTRA_FIELD_IDS } from './gnbQuickSettingsFields';
import CommonQuickSettingsNetworkCards from './CommonQuickSettingsNetworkCards';
import PrimaryRadioInstanceEditor from './PrimaryRadioInstanceEditor';

const { Text } = Typography;

interface AllocationRuleFieldsProps {
  name: 'gnbIdAllocation' | 'pciAllocation';
  title: string;
  maximum: number;
  readOnly?: boolean;
}

function AllocationRuleFields({ name, title, maximum, readOnly = false }: AllocationRuleFieldsProps) {
  const t = useT();
  return (
    <Card size="small" title={title} style={{ marginBottom: 16 }}>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: 16 }}>
        <Form.Item
          name={[name, 'start']}
          label={t('provision.rangeStart')}
          rules={[{ required: true }]}
        >
          <InputNumber min={0} max={maximum} readOnly={readOnly} style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item
          name={[name, 'end']}
          label={t('provision.rangeEnd')}
          rules={[{ required: true }]}
        >
          <InputNumber min={0} max={maximum} readOnly={readOnly} style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item
          name={[name, 'step']}
          label={t('provision.rangeStep')}
          rules={[{ required: true }]}
        >
          <InputNumber min={1} readOnly={readOnly} style={{ width: '100%' }} />
        </Form.Item>
      </div>
      <Form.List name={[name, 'reserved']}>
        {(fields, { add, remove }) => (
          <>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: fields.length ? 8 : 0 }}>
              <Text type="secondary">{t('provision.reservedRanges', { field: title })}</Text>
              {!readOnly && (
                <Button type="link" icon={<PlusOutlined />} onClick={() => add({ start: 0, end: 0 })}>
                  {t('common.add')}
                </Button>
              )}
            </div>
            {fields.length === 0 && <Text type="secondary">{t('provision.noReservedRanges')}</Text>}
            {fields.map((field) => (
              <Space key={field.key} align="baseline" style={{ marginRight: 16 }}>
                <Form.Item {...field} name={[field.name, 'start']} rules={[{ required: true }]}>
                  <InputNumber min={0} max={maximum} placeholder={t('provision.rangeStart')} readOnly={readOnly} />
                </Form.Item>
                <span>—</span>
                <Form.Item {...field} name={[field.name, 'end']} rules={[{ required: true }]}>
                  <InputNumber min={0} max={maximum} placeholder={t('provision.rangeEnd')} readOnly={readOnly} />
                </Form.Item>
                {!readOnly && <Button type="text" danger icon={<DeleteOutlined />} onClick={() => remove(field.name)} />}
              </Space>
            ))}
          </>
        )}
      </Form.List>
    </Card>
  );
}

function CustomParameters({ readOnly = false }: { readOnly?: boolean }) {
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
                extra={readOnly ? undefined : <Button type="link" danger icon={<DeleteOutlined />} onClick={() => remove(name)} />}
                style={{ marginBottom: 12 }}
              >
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: 16 }}>
                  <Form.Item {...restField} name={[name, 'name']} label={t('provision.nrQuick.name')}><Input readOnly={readOnly} /></Form.Item>
                  <Form.Item {...restField} name={[name, 'value']} label={t('provision.nrQuick.value')}><Input readOnly={readOnly} /></Form.Item>
                  <Form.Item {...restField} name={[name, 'trPath']} label={t('provision.nrQuick.trPath')}><Input readOnly={readOnly} /></Form.Item>
                </div>
              </Card>
            ))}
            {!readOnly && (
              <Button type="dashed" onClick={() => add()} block icon={<PlusOutlined />}>
                {t('provision.addCustomParam')}
              </Button>
            )}
          </>
        )}
      </Form.List>
    </Card>
  );
}

export interface CommonParameterConfigPanelProps {
  form: FormInstance;
  deviceType?: ParamConfigDeviceType;
  paramModelName?: string;
  productClass?: string;
  disabled?: boolean;
  readOnly?: boolean;
}

export interface ParameterConfigFieldsProps {
  deviceType?: ParamConfigDeviceType;
  paramModelName?: string;
  productClass?: string;
  scope?: 'common' | 'device';
  onRequestEdit?: () => void;
  readOnly?: boolean;
}

export function ParameterConfigFields({
  deviceType,
  paramModelName,
  productClass,
  scope = 'common',
  onRequestEdit,
  readOnly = false,
}: ParameterConfigFieldsProps) {
  const t = useT();
  const commonScope = scope === 'common';
  return (
    <>
      {deviceType && (
        <PrimaryRadioInstanceEditor
          deviceType={deviceType}
          productClass={productClass}
          excludedFieldIds={commonScope && deviceType === 'gNB' ? ['PCI'] : []}
          readOnly={readOnly}
        />
      )}
      {deviceType === 'gNB' && (
        <>
          {commonScope && (
            <>
              <Card size="small" title={t('provision.batchAllocationRules')} style={{ marginBottom: 16 }}>
                <Text type="secondary">{t('provision.batchAllocationRulesHint')}</Text>
              </Card>
              <AllocationRuleFields name="gnbIdAllocation" title="gNB ID" maximum={4_294_967_295} readOnly={readOnly} />
              <AllocationRuleFields name="pciAllocation" title="PCI" maximum={1007} readOnly={readOnly} />
            </>
          )}
          <GnbQuickSettingsCards
            excludedFieldIds={commonScope ? ['gNBId', 'PCI'] : []}
            excludedGroupIds={['gnb-cell', 'gnb-tdd']}
            beforeIpsec={<CommonQuickSettingsNetworkCards paramModelName={paramModelName} onRequestEdit={onRequestEdit} readOnly={readOnly} />}
            readOnly={readOnly}
          />
          <Card size="small" title={t('provision.otherTemplateParams')} style={{ marginBottom: 16 }}>
            <GnbTemplateExtraFieldGrid
              readOnly={readOnly}
              excludedFieldIds={commonScope
                ? [...GNB_COMMON_EXCLUDED_EXTRA_FIELD_IDS, 'PrachRootSequenceIndex', 'PrachRootSequenceValue']
                : ['PrachRootSequenceIndex', 'PrachRootSequenceValue']}
            />
          </Card>
          <CustomParameters readOnly={readOnly} />
        </>
      )}
      {deviceType === 'eNB' && (
        <>
          <EnbQuickSettingsCards
            paramModelName={paramModelName}
            excludedGroupIds={['enb-cell']}
            beforeIpsec={<CommonQuickSettingsNetworkCards paramModelName={paramModelName} onRequestEdit={onRequestEdit} readOnly={readOnly} />}
            readOnly={readOnly}
          />
          <Card size="small" title={t('provision.otherTemplateParams')} style={{ marginBottom: 16 }}>
            <EnbTemplateExtraFieldGrid excludedFieldIds={['CELL_NUMBER']} readOnly={readOnly} />
          </Card>
          <CustomParameters readOnly={readOnly} />
        </>
      )}
      {deviceType === 'GSM' && (
        <>
          <CommonQuickSettingsNetworkCards paramModelName={paramModelName} onRequestEdit={onRequestEdit} readOnly={readOnly} />
          <CustomParameters readOnly={readOnly} />
        </>
      )}
      {!deviceType && <Alert type="warning" showIcon title={t('provision.paramConfigDeviceTypeUnavailable')} />}
    </>
  );
}

export default function CommonParameterConfigPanel({
  form,
  deviceType,
  paramModelName,
  productClass,
  disabled = false,
  readOnly = false,
}: CommonParameterConfigPanelProps) {
  const t = useT();
  return (
    <Form form={form} component={false} disabled={disabled && !readOnly}>
      <Alert
        type="info"
        showIcon
        title={t('provision.commonParamConfigHint')}
        style={{ marginBottom: 16 }}
      />
      <ParameterConfigFields
        deviceType={deviceType}
        paramModelName={paramModelName}
        productClass={productClass}
        scope="common"
        readOnly={readOnly}
      />
    </Form>
  );
}
