import type { GnbQuickSettingField, GnbQuickSettingOption } from './gnbQuickSettingsFields';
import { ENB_1588_TEMPLATE_FIELDS } from './enbQuickSettingsFields';

export interface EnbProductSyncConfig {
  fields: GnbQuickSettingField[];
  modeFieldName?: string;
  ptpModeValues: readonly string[];
  collapsedFieldCount: number;
}

const options = (entries: ReadonlyArray<readonly [string, string]>): GnbQuickSettingOption[] => (
  entries.map(([value, label]) => ({ value, label }))
);

const tfcsField = (entries: ReadonlyArray<readonly [string, string]>): GnbQuickSettingField => ({
  id: 'tfcsManagerPrimsrc',
  name: 'tfcsManagerPrimsrc',
  labelKey: 'provision.nrQuick.syncSource',
  control: 'select',
  options: options(entries),
});

const TFCS_FIELDS: Record<string, GnbQuickSettingField> = {
  BLQ: tfcsField([
    ['2', 'PTP'], ['3', 'GPS'], ['7', 'Free'], ['8', 'Single Beidou'],
    ['9', 'GLONASS'], ['10', 'GPS+Beidou'], ['11', 'GPS+GLONASS'],
  ]),
  MLN: tfcsField([
    ['1', 'NTP'], ['2', 'PTP'], ['3', 'GNSS'], ['4', 'NL'], ['5', 'EXT_CLK'],
    ['6', 'EXT_PPS'], ['7', 'Free'], ['8', 'Single Beidou'], ['9', 'GPS+Beidou'],
    ['10', 'GLONASS'],
  ]),
  MLQ: tfcsField([
    ['1', 'NTP'], ['2', 'PTP'], ['4', 'NL'], ['5', 'EXT_CLK'], ['6', 'EXT_PPS'],
    ['7', 'Free'], ['8', 'Single Beidou'], ['9', 'GLONASS'], ['10', 'GPS+Beidou'],
    ['11', 'GPS+GLONASS'], ['31', 'Single GPS'], ['32', 'Single GLONASS'],
  ]),
};

export function getEnbProductSyncConfig(paramModelName: string | undefined): EnbProductSyncConfig {
  const normalized = paramModelName?.trim().toUpperCase() ?? '';
  if (normalized === 'BM') {
    return {
      fields: ENB_1588_TEMPLATE_FIELDS,
      modeFieldName: 'PpsTimeMode',
      ptpModeValues: ['2'],
      collapsedFieldCount: 2,
    };
  }
  const field = TFCS_FIELDS[normalized];
  return {
    fields: field ? [field] : [],
    modeFieldName: field ? 'tfcsManagerPrimsrc' : undefined,
    ptpModeValues: [],
    collapsedFieldCount: field ? 1 : 0,
  };
}
