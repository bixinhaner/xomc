import {
  ENB_1588_TEMPLATE_FIELDS,
  ENB_IPSEC_TEMPLATE_EXTRA_FIELDS,
  ENB_QUICK_SETTING_GROUPS,
  ENB_TEMPLATE_EXTRA_FIELDS,
} from './enbQuickSettingsFields';
import {
  GNB_QUICK_SETTING_GROUPS,
  GNB_TEMPLATE_EXTRA_FIELDS,
  NR_CARRIER_BANDWIDTH_OPTIONS_BY_SCS,
  type GnbQuickSettingField,
} from './gnbQuickSettingsFields';
import {
  ENB_SHEET_FIELD_MAPPINGS,
  GNB_SHEET_FIELD_MAPPINGS,
} from './paramConfigFieldMappings';
import type { ParamConfigDeviceType } from './paramConfigWorkbook';
import { getTimezoneAliasOptions } from '@core/utils/timezoneAliasConfig';

export type ParamConfigExportField = GnbQuickSettingField & { condition?: string };

function sheetHeader(
  field: GnbQuickSettingField,
  mappings: readonly { field: string; header: string }[],
): GnbQuickSettingField['name'] {
  if (Array.isArray(field.name)) return field.name;
  return mappings.find((mapping) => mapping.field === field.name)?.header ?? field.name;
}

function withCondition(
  field: GnbQuickSettingField,
  groupId: string,
): ParamConfigExportField {
  if (groupId === 'gnb-sync-source' && !['PpsTimeMode', 'SyncSource'].includes(field.id)) {
    return { ...field, condition: 'PpsTimeMode = 1588_PPS' };
  }
  return field;
}

export function getParamConfigExportFields(
  deviceType: ParamConfigDeviceType | undefined,
): ParamConfigExportField[] {
  const groups = deviceType === 'eNB' ? ENB_QUICK_SETTING_GROUPS
    : deviceType === 'gNB' ? GNB_QUICK_SETTING_GROUPS : [];
  const mappings = deviceType === 'eNB' ? ENB_SHEET_FIELD_MAPPINGS : GNB_SHEET_FIELD_MAPPINGS;
  const carrierBandwidthOptions = Array.from(new Map(
    Object.values(NR_CARRIER_BANDWIDTH_OPTIONS_BY_SCS).flat().map((option) => [option.value, option]),
  ).values());
  const timezoneOptions = getTimezoneAliasOptions();
  const fields = groups.flatMap((group) => group.fields.map((field) => {
    const options = field.control === 'dl-bandwidth' || field.control === 'ul-bandwidth'
      ? carrierBandwidthOptions
      : field.control === 'timezone' ? timezoneOptions : field.options;
    return withCondition({ ...field, options, name: sheetHeader(field, mappings) }, group.id);
  }));
  if (deviceType === 'eNB') {
    return [...fields, ...ENB_TEMPLATE_EXTRA_FIELDS, ...ENB_1588_TEMPLATE_FIELDS.map((field) => ({
      ...field,
      condition: '同步源选择 PTP/1588',
    })), ...ENB_IPSEC_TEMPLATE_EXTRA_FIELDS];
  }
  return deviceType === 'gNB' ? [...fields, ...GNB_TEMPLATE_EXTRA_FIELDS] : fields;
}
