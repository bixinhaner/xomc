import { describe, expect, it } from 'vitest';
import zhCN from '@core/i18n/zh-CN';
import enUS from '@core/i18n/en-US';
import {
  GNB_QUICK_SETTING_GROUPS,
  GNB_TEMPLATE_EXTRA_FIELDS,
} from './gnbQuickSettingsFields';

describe('gNB plug-and-play quick-setting fields', () => {
  it('matches the NR quick-settings group and field order through TDD', () => {
    expect(GNB_QUICK_SETTING_GROUPS.map((group) => group.id)).toEqual([
      'device-time',
      'gnb-sync-source',
      'device-ipsec-control',
      'gnb-ipsec',
      'gnb-cell',
      'gnb-core',
      'gnb-tdd',
    ]);
    const expectedFields = {
      'device-time': ['Enable', 'LocalTimeZoneName', 'NTPServer1', 'NTPServer2', 'NTPServer3', 'NTPServer4', 'NTPServer5'],
      'gnb-sync-source': ['PpsTimeMode', 'SyncSource', 'ForcedSync', 'PTPProfile', 'PTPDomain', 'PTPTransmode', 'PTPInterface', 'PTPUnicastMode', 'PTPSyncInterval', 'PTPDelayInterval'],
      'device-ipsec-control': ['IPSEC_ENABLE'],
      'gnb-ipsec': ['TUNNEL_ENABLE', 'TUNNEL_GATEWAY', 'TUNNEL_LEFT_AUTH', 'TUNNEL_RIGHT_AUTH', 'LEFT_IDENTIFIER', 'RIGHT_IDENTIFIER', 'LEFTSOURCEIP', 'LEFTSUBNET', 'RIGHT_SUBNET', 'TUNNEL_FRAGMENTATION', 'IKE_ENCRYPTION', 'IKE_DH_GROUP', 'IKE_AUTHENTICATION', 'ESP_ENCRYPTION', 'ESP_DH_GROUP', 'ESP_AUTHENTICATION', 'KEYLIFE', 'IKELIFETIME', 'REKEYMARGIN', 'DPDACTION', 'DPDDELAY', 'LEFT_INTERFACE'],
      'gnb-cell': ['Band', 'SsbFrequency', 'DLSubCarrierSpacing', 'ULSubCarrierSpacing', 'DLCarrierBandWidth', 'ULCarrierBandWidth', 'NRARFCNDL', 'NRARFCNUL', 'NumOfRxAntenna', 'NumOfTxAntenna', 'OffsetToPointA', 'PCI', 'PowerModify', 'RFEnable', 'SsbSubcarrierOffset'],
      'gnb-core': ['PLMNID', 'TAC', 'AmfIP1', 'gNBName', 'NguBindInterface', 'gNBId', 'gNBIdLength', 'NrcellIdentity'],
      'gnb-tdd': ['ReferenceSubcarrierSpacing', 'DlULTransmissionPeriodicity', 'NrofDownlinkSlots', 'NrofDownlinkSymbols', 'NrofUplinkSlots', 'NrofUplinkSymbols', 'Pat2DlULTransmissionPeriodicity', 'Pat2NrofDownlinkSlots', 'Pat2NrofDownlinkSymbols', 'Pat2NrofUplinkSlots', 'Pat2NrofUplinkSymbols'],
    };
    for (const group of GNB_QUICK_SETTING_GROUPS) {
      expect(group.fields.map((field) => field.id)).toEqual(
        expectedFields[group.id as keyof typeof expectedFields],
      );
    }
  });

  it('uses localized labels and quick-settings control types instead of raw workbook headers', () => {
    const fields = GNB_QUICK_SETTING_GROUPS.flatMap((group) => group.fields);
    expect(fields.every((field) => field.labelKey.startsWith('provision.nrQuick.'))).toBe(true);
    expect(fields.find((field) => field.id === 'DLSubCarrierSpacing')?.control).toBe('select');
    expect(fields.find((field) => field.id === 'DLCarrierBandWidth')?.control).toBe('dl-bandwidth');
    expect(fields.find((field) => field.id === 'RFEnable')?.control).toBe('select');
    expect(fields.map((field) => field.labelKey)).not.toContain('SubcarrierSpacing(DL)');
  });

  it('defines every group, field and option label in both supported locales', () => {
    const fields = [
      ...GNB_QUICK_SETTING_GROUPS.flatMap((group) => group.fields),
      ...GNB_TEMPLATE_EXTRA_FIELDS,
    ];
    const keys = new Set([
      ...GNB_QUICK_SETTING_GROUPS.map((group) => group.titleKey),
      ...fields.map((field) => field.labelKey),
      ...fields.flatMap((field) => field.options?.map((option) => option.labelKey).filter(Boolean) ?? []),
      'provision.nrQuick.rangeHint',
    ]);
    for (const key of keys) {
      expect(zhCN[key as string], `missing zh-CN message: ${key}`).toBeTruthy();
      expect(enUS[key as string], `missing en-US message: ${key}`).toBeTruthy();
    }
  });
});
