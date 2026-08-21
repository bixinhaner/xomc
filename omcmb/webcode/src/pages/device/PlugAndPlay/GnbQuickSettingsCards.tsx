import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, Card, Form, Input, Select, Switch, Typography } from 'antd';
import { Fragment, type ReactNode } from 'react';
import type { NamePath } from 'antd/es/form/interface';
import { getTimezoneAliasOptions } from '@core/utils/timezoneAliasConfig';
import { useT } from '@/hooks/useT';
import {
  GNB_QUICK_SETTING_GROUPS,
  GNB_TEMPLATE_EXTRA_FIELDS,
  getNrCarrierBandwidthOptionsForScs,
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

function normalizeBinaryOptionValue(
  field: GnbQuickSettingField,
  value: unknown,
): unknown {
  if (field.control !== 'select' || !field.options?.some((option) => option.value === '0')
    || !field.options.some((option) => option.value === '1')) {
    return value;
  }
  const normalized = String(value ?? '').trim().toLowerCase();
  if (['true', 'yes', 'on'].includes(normalized)) return '1';
  if (['false', 'no', 'off'].includes(normalized)) return '0';
  return value;
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
  readOnly,
}: {
  field: GnbQuickSettingField;
  value: unknown;
  checked?: boolean;
  onChange?: (value: unknown) => void;
  dlScs: unknown;
  ulScs: unknown;
  readOnly?: boolean;
}) {
  const t = useT();
  const renderReadOnly = (displayValue: unknown) => (
    <Input value={displayValue == null ? '' : String(displayValue)} readOnly />
  );
  if (field.control === 'readonly') {
    return renderReadOnly(value);
  }
  if (field.control === 'switch') {
    if (readOnly) return renderReadOnly(checked ? t('common.on') : t('common.off'));
    return <Switch checked={checked} onChange={onChange} checkedChildren={t('common.on')} unCheckedChildren={t('common.off')} />;
  }
  if (field.control === 'timezone') {
    const options = getTimezoneAliasOptions().map((option) => ({
      value: option.value,
      label: option.label,
    }));
    if (readOnly) {
      return renderReadOnly(appendCurrentOption(options, value).find((option) => option.value === value)?.label ?? value);
    }
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
    const currentValues = (Array.isArray(value) ? value : [value])
      .map((current) => normalizeBinaryOptionValue(field, current));
    const options = localizedOptions(field.options, t);
    currentValues.forEach((current) => {
      const normalized = current == null ? '' : String(current);
      if (normalized && !options.some((option) => option.value === normalized)) {
        options.push({ value: normalized, label: normalized });
      }
    });
    if (readOnly) {
      const labels = currentValues
        .map((current) => {
          const normalized = current == null ? '' : String(current);
          return options.find((option) => option.value === normalized)?.label ?? normalized;
        })
        .filter(Boolean);
      return renderReadOnly(labels.join(', '));
    }
    return (
      <Select
        mode={field.control === 'multi-select' ? 'multiple' : undefined}
        showSearch
        optionFilterProp="label"
        options={options}
        value={value as string | string[] | undefined}
        onChange={onChange}
      />
    );
  }
  if (field.control === 'dl-bandwidth' || field.control === 'ul-bandwidth') {
    const scs = field.control === 'dl-bandwidth' ? dlScs : ulScs;
    const options = localizedOptions(getNrCarrierBandwidthOptionsForScs(scs), t);
    if (readOnly) {
      return renderReadOnly(appendCurrentOption(options, value).find((option) => option.value === value)?.label ?? value);
    }
    return <Select options={appendCurrentOption(options, value)} value={value as string | undefined} onChange={onChange} />;
  }
  return <Input value={String(value ?? '')} onChange={onChange} readOnly={readOnly} />;
}

function QuickSettingFieldItem({
  field,
  name = field.name,
  dlScs,
  ulScs,
  readOnly,
}: {
  field: GnbQuickSettingField;
  name?: string | number | Array<string | number>;
  dlScs: unknown;
  ulScs: unknown;
  readOnly?: boolean;
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
          return {
            value: rawValue == null || rawValue === ''
              ? undefined
              : String(normalizeBinaryOptionValue(field, rawValue)),
          };
        }
        return { value: rawValue == null ? '' : String(rawValue) };
      }}
      normalize={(nextValue) => {
        if (control === 'switch') return nextValue ? '1' : '0';
        if (control === 'multi-select') return Array.isArray(nextValue) ? nextValue.join(',') : nextValue;
        return nextValue;
      }}
    >
      <QuickSettingControl field={field} value={undefined} dlScs={dlScs} ulScs={ulScs} readOnly={readOnly} />
    </Form.Item>
  );
}

function namePathParts(name: GnbQuickSettingField['name']): Array<string | number> {
  return Array.isArray(name) ? name : [name as string | number];
}

export function GnbQuickSettingFieldGrid({
  fields,
  namePrefix,
  nameResolver,
  dlScsName = DL_SCS_PATH,
  ulScsName = UL_SCS_PATH,
  readOnly,
}: {
  fields: GnbQuickSettingField[];
  namePrefix?: Array<string | number>;
  nameResolver?: (field: GnbQuickSettingField) => NamePath;
  dlScsName?: NamePath;
  ulScsName?: NamePath;
  readOnly?: boolean;
}) {
  const form = Form.useFormInstance();
  const dlScs = Form.useWatch(dlScsName, form);
  const ulScs = Form.useWatch(ulScsName, form);
  return (
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', columnGap: 16 }}>
      {fields.map((field) => (
        <QuickSettingFieldItem
          key={field.id}
          field={field}
          name={nameResolver?.(field) ?? (namePrefix ? [...namePrefix, ...namePathParts(field.name)] : field.name)}
          dlScs={dlScs}
          ulScs={ulScs}
          readOnly={readOnly}
        />
      ))}
    </div>
  );
}

function IpsecListCard({ fields, readOnly }: { fields: GnbQuickSettingField[]; readOnly?: boolean }) {
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
              extra={readOnly ? undefined : <Button type="link" danger icon={<DeleteOutlined />} onClick={() => remove(name)} />}
              style={{ marginBottom: 12 }}
            >
              <GnbQuickSettingFieldGrid fields={fields} namePrefix={[name]} readOnly={readOnly} />
            </Card>
          ))}
          {!readOnly && (
            <Button type="dashed" onClick={() => add()} block icon={<PlusOutlined />}>
              {t('provision.addTunnel')}
            </Button>
          )}
        </>
      )}
    </Form.List>
  );
}

export default function GnbQuickSettingsCards({
  excludedFieldIds = [],
  excludedGroupIds = [],
  beforeIpsec,
  readOnly = false,
}: {
  excludedFieldIds?: string[];
  excludedGroupIds?: string[];
  beforeIpsec?: ReactNode;
  readOnly?: boolean;
}) {
  const t = useT();
  const form = Form.useFormInstance();
  const ipsecEnable = Form.useWatch('IPSEC_ENABLE', form);
  const syncMode = Form.useWatch(['sheetParameters', 'DEVICE', 0, 'PpsTimeMode'], form);
  return (
    <>
      {GNB_QUICK_SETTING_GROUPS
        .filter((group) => !excludedGroupIds.includes(group.id))
        .filter((group) => group.id !== 'gnb-ipsec' || isIpsecParametersVisible(ipsecEnable))
        .map((group) => (
          <Fragment key={group.id}>
            {group.id === 'device-ipsec-control' && beforeIpsec}
            <Card
              size="small"
              title={t(group.titleKey)}
              style={{ marginBottom: 16 }}
            >
              {group.multiInstance
                ? <IpsecListCard fields={group.fields} readOnly={readOnly} />
                : <GnbQuickSettingFieldGrid fields={
                  (group.id === 'gnb-sync-source' && !isPtpDetailsVisible(syncMode)
                    ? group.fields.slice(0, 3)
                    : group.fields).filter((field) => !excludedFieldIds.includes(field.id))
                } readOnly={readOnly} />}
            </Card>
          </Fragment>
        ))}
    </>
  );
}

export function GnbTemplateExtraFieldGrid({
  excludedFieldIds = [],
  readOnly = false,
}: {
  excludedFieldIds?: readonly string[];
  readOnly?: boolean;
}) {
  return (
    <GnbQuickSettingFieldGrid
      fields={GNB_TEMPLATE_EXTRA_FIELDS.filter((field) => !excludedFieldIds.includes(field.id))}
      readOnly={readOnly}
    />
  );
}
