import { describe, expect, it } from 'vitest';
import zhCN from '@core/i18n/zh-CN';
import enUS from '@core/i18n/en-US';
import {
  ENB_1588_TEMPLATE_FIELDS,
  ENB_IPSEC_TEMPLATE_EXTRA_FIELDS,
  ENB_QUICK_SETTING_GROUPS,
  ENB_TEMPLATE_EXTRA_FIELDS,
} from './enbQuickSettingsFields';
import { withProductEnumOptions } from './EnbQuickSettingsCards';

describe('eNB plug-and-play quick-setting fields', () => {
  it('matches the LTE quick-settings group and field order', () => {
    expect(ENB_QUICK_SETTING_GROUPS.map((group) => group.id)).toEqual([
      'enb-cell', 'enb-plmn', 'enb-mme', 'device-time',
      'device-ipsec-control', 'device-ipsec',
    ]);
    expect(ENB_QUICK_SETTING_GROUPS[0].fields.map((field) => field.id)).toEqual([
      'ECI', 'CellName', 'TAC', 'PCI', 'MaxTxPower', 'BandSupport',
      'BandIndicator', 'DLEarfcn', 'DLBandWidth', 'SubFrameAssignment',
      'SpecialSubFramePatterns', 'RootSequenceIndex', 'CellType', 'PowerClass',
    ]);
    expect(ENB_QUICK_SETTING_GROUPS[3].fields.map((field) => field.id)).toEqual([
      'Enable', 'NTPServer1', 'NTPServer2', 'NTPServer3',
      'NTPServer4', 'NTPServer5', 'LocalTimeZoneName',
    ]);
    expect(ENB_QUICK_SETTING_GROUPS[5].fields.map((field) => field.id)).toEqual([
      'TUNNEL_ENABLE', 'TUNNEL_GATEWAY', 'LEFT_AUTH', 'RIGHT_AUTH',
      'LEFT_IDENTIFIER', 'RIGHT_IDENTIFIER', 'LEFTSOURCEIP', 'LEFT_SUBNET',
      'RIGHT_SUBNET', 'FRAGMENTATION', 'IKE_ENCRYPTION', 'IKE_DH_GROUP',
      'IKE_AUTHENTICATION', 'ESP_ENCRYPTION', 'ESP_DH_GROUP',
      'ESP_AUTHENTICATION', 'KEYLIFE', 'IKELIFETIME', 'REKEYMARGIN',
      'DPDACTION', 'DPDDELAY',
    ]);
    expect(ENB_1588_TEMPLATE_FIELDS.map((field) => field.id)).toEqual([
      'PpsTimeMode', 'SyncSource', 'PTPProfile', 'PTPDomain',
      'PTPTransmode', 'PTPInterface', 'PTPUnicastMode', 'PTPSyncInterval',
      'PTPDelayInterval',
    ]);
  });

  it('uses semantic controls and keeps workbook-only fields separate', () => {
    const fields = ENB_QUICK_SETTING_GROUPS.flatMap((group) => group.fields);
    expect(fields.find((field) => field.id === 'BandSupport')?.control).toBe('readonly');
    expect(fields.find((field) => field.id === 'DLBandWidth')?.control).toBe('select');
    expect(fields.find((field) => field.id === 'Enable')?.control).toBe('select');
    expect(fields.find((field) => field.id === 'IPSEC_ENABLE')?.control).toBe('select');
    expect(fields.find((field) => field.id === 'TUNNEL_ENABLE')?.control).toBe('switch');
    expect(ENB_TEMPLATE_EXTRA_FIELDS.map((field) => field.id)).toContain('CELL_NUMBER');
    expect(ENB_IPSEC_TEMPLATE_EXTRA_FIELDS.map((field) => field.id)).toContain('FORCEENCAPS');
  });

  it('keeps LTE bandwidth options in page/import format when product metadata uses wire values', () => {
    const groups = withProductEnumOptions(ENB_QUICK_SETTING_GROUPS, [{
      id: 'enb-cell', titleZh: '', titleEn: '', multiInstance: false,
      params: [{
        name: 'DLBandWidth', titleZh: '带宽', titleEn: 'Bandwidth', type: 'int',
        enumOptions: [
          { value: '25', label: '5MHz' },
          { value: '50', label: '10MHz' },
        ],
      }],
    }]);

    expect(groups[0].fields.find((field) => field.id === 'DLBandWidth')?.options).toEqual([
      { value: 'n25', label: '5MHz' },
      { value: 'n50', label: '10MHz' },
    ]);
  });

  it('defines every group, field and option label in both supported locales', () => {
    const fields = [
      ...ENB_QUICK_SETTING_GROUPS.flatMap((group) => group.fields),
      ...ENB_TEMPLATE_EXTRA_FIELDS,
      ...ENB_1588_TEMPLATE_FIELDS,
      ...ENB_IPSEC_TEMPLATE_EXTRA_FIELDS,
    ];
    const keys = new Set([
      ...ENB_QUICK_SETTING_GROUPS.map((group) => group.titleKey),
      ...fields.map((field) => field.labelKey),
      ...fields.flatMap((field) => field.options?.map((option) => option.labelKey).filter(Boolean) ?? []),
      'provision.lteQuick.mmeIp',
      'provision.lteQuick.tunnelTemplateExtras',
    ]);
    for (const key of keys) {
      expect(zhCN[key as string], `missing zh-CN message: ${key}`).toBeTruthy();
      expect(enUS[key as string], `missing en-US message: ${key}`).toBeTruthy();
    }
  });
});
