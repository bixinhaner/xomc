import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, Card, Form, Input, Select, Switch, Typography } from 'antd';
import { getTimezoneAliasOptions } from '@core/utils/timezoneAliasConfig';
import { useT } from '@/hooks/useT';
import {
  GNB_QUICK_SETTING_GROUPS,
  GNB_TEMPLATE_EXTRA_FIELDS,
  NR_CARRIER_BANDWIDTH_OPTIONS_BY_SCS,
  type GnbQuickSettingField,
  type GnbQuickSettingOption,
} from './gnbQuickSettingsFields';
import { isIpsecParametersVisible, isPtpDetailsVisible } from './quickSettingsVisibility';

const { Text } = Typography;
const DL_SCS_PATH = ['sheetParameters', 'CELL', 0, 'SubcarrierSpacing(DL)'];
const UL_SCS_PATH = ['sheetParameters', 'CELL', 0, 'SubcarrierSpacing(UL)'];

function localizedOptions(
  options: GnbQuickSettingOption[] | undefined,
  t: ReturnType<typeof useT>,
) {
  return (options ?? []).map((option) => ({
    value: option.value,
    label: option.labelKey ? t(option.labelKey) : option.label ?? option.value,
  }));
}

function appendCurrentOption(
  options: Array<{ value: string; label: string }>,
  current: unknown,
) {
  const value = current == null ? '' : String(current);
  if (!value || options.some((option) => option.value === value)) return options;
  return [...options, { value, label: value }];
}

function FieldLabel({ field }: { field: GnbQuickSettingField }) {
  const t = useT();
  return (
    <span>
      {t(field.labelKey, field.labelValues)}
      {field.range && (
        <Text type="secondary" style={{ marginLeft: 4, fontSize: 12 }}>
          {t('provision.nrQuick.rangeHint', { range: field.range })}
        </Text>
      )}
      {field.control === 'readonly' && (
        <Text type="secondary" style={{ marginLeft: 4, fontSize: 12 }}>
          {t('device.cell.readonly')}
        </Text>
      )}
    </span>
  );
}

function QuickSettingControl({
  field,
  value,
  checked,
  onChange,
  dlScs,
  ulScs,
}: {
  field: GnbQuickSettingField;
  value: unknown;
  checked?: boolean;
  onChange?: (value: unknown) => void;
  dlScs: unknown;
  ulScs: unknown;
}) {
  const t = useT();
  if (field.control === 'readonly') {
    return <Input value={String(value ?? '')} disabled />;
  }
  if (field.control === 'switch') {
    return <Switch checked={checked} onChange={onChange} checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />;
  }
  if (field.control === 'timezone') {
    const options = getTimezoneAliasOptions().map((option) => ({
      value: option.value,
      label: option.label,
    }));
    return (
      <Select
        showSearch
        optionFilterProp="label"
        options={appendCurrentOption(options, value)}
        value={value as string | undefined}
        onChange={onChange}
      />
    );
  }
  if (field.control === 'select' || field.control === 'multi-select') {
    return (
      <Select
        mode={field.control === 'multi-select' ? 'multiple' : undefined}
        showSearch
        optionFilterProp="label"
        options={localizedOptions(field.options, t)}
        value={value as string | string[] | undefined}
        onChange={onChange}
      />
    );
  }
  if (field.control === 'dl-bandwidth' || field.control === 'ul-bandwidth') {
    const scs = field.control === 'dl-bandwidth' ? dlScs : ulScs;
    const options = localizedOptions(NR_CARRIER_BANDWIDTH_OPTIONS_BY_SCS[String(scs ?? '')], t);
    return <Select options={appendCurrentOption(options, value)} value={value as string | undefined} onChange={onChange} />;
  }
  return <Input value={String(value ?? '')} onChange={onChange} />;
}

function QuickSettingFieldItem({
  field,
  name = field.name,
  dlScs,
  ulScs,
}: {
  field: GnbQuickSettingField;
  name?: string | number | Array<string | number>;
  dlScs: unknown;
  ulScs: unknown;
}) {
  const control = field.control ?? 'input';
  return (
    <Form.Item
      name={name}
      label={<FieldLabel field={field} />}
      getValueProps={(rawValue) => {
        if (control === 'switch') return { checked: rawValue === true || String(rawValue ?? '') === '1' };
        if (control === 'multi-select') {
          return { value: String(rawValue ?? '').split(',').map((item) => item.trim()).filter(Boolean) };
        }
        if (control === 'select' || control === 'timezone' || control.endsWith('-bandwidth')) {
          return { value: rawValue == null || rawValue === '' ? undefined : String(rawValue) };
        }
        return { value: rawValue == null ? '' : String(rawValue) };
      }}
      normalize={(nextValue) => {
        if (control === 'switch') return nextValue ? '1' : '0';
        if (control === 'multi-select') return Array.isArray(nextValue) ? nextValue.join(',') : nextValue;
        return nextValue;
      }}
    >
      <QuickSettingControl field={field} value={undefined} dlScs={dlScs} ulScs={ulScs} />
    </Form.Item>
  );
}

export function GnbQuickSettingFieldGrid({
  fields,
  namePrefix,
}: {
  fields: GnbQuickSettingField[];
  namePrefix?: Array<string | number>;
}) {
  const form = Form.useFormInstance();
  const dlScs = Form.useWatch(DL_SCS_PATH, form);
  const ulScs = Form.useWatch(UL_SCS_PATH, form);
  return (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', columnGap: 16 }}>
      {fields.map((field) => (
        <QuickSettingFieldItem
          key={field.id}
          field={field}
          name={namePrefix ? [...namePrefix, field.name as string] : field.name}
          dlScs={dlScs}
          ulScs={ulScs}
        />
      ))}
    </div>
  );
}

function IpsecListCard({ fields }: { fields: GnbQuickSettingField[] }) {
  const t = useT();
  return (
    <Form.List name="ipsecList">
      {(listFields, { add, remove }) => (
        <>
          {listFields.map(({ key, name }) => (
            <Card
              key={key}
              size="small"
              title={`${t('provision.ipsecTunnel')} ${name + 1}`}
              extra={<Button type="link" danger icon={<DeleteOutlined />} onClick={() => remove(name)} />}
              style={{ marginBottom: 12 }}
            >
              <GnbQuickSettingFieldGrid fields={fields} namePrefix={[name]} />
            </Card>
          ))}
          <Button type="dashed" onClick={() => add()} block icon={<PlusOutlined />}>
            {t('provision.addTunnel')}
          </Button>
        </>
      )}
    </Form.List>
  );
}

export default function GnbQuickSettingsCards() {
  const t = useT();
  const form = Form.useFormInstance();
  const ipsecEnable = Form.useWatch('IPSEC_ENABLE', form);
  const syncMode = Form.useWatch(['sheetParameters', 'DEVICE', 0, 'PpsTimeMode'], form);
  return (
    <>
      {GNB_QUICK_SETTING_GROUPS
        .filter((group) => group.id !== 'gnb-ipsec' || isIpsecParametersVisible(ipsecEnable))
        .map((group) => (
        <Card
          key={group.id}
          size="small"
          title={t(group.titleKey)}
          style={{ marginBottom: 16 }}
        >
          {group.multiInstance
            ? <IpsecListCard fields={group.fields} />
            : <GnbQuickSettingFieldGrid fields={
              group.id === 'gnb-sync-source' && !isPtpDetailsVisible(syncMode)
                ? group.fields.slice(0, 3)
                : group.fields
            } />}
        </Card>
        ))}
    </>
  );
}

export function GnbTemplateExtraFieldGrid() {
  return <GnbQuickSettingFieldGrid fields={GNB_TEMPLATE_EXTRA_FIELDS} />;
}
