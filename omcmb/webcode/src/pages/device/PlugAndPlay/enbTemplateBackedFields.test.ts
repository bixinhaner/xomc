import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const panelSource = readFileSync(
  resolve(process.cwd(), 'src/pages/device/PlugAndPlay/CommonParameterConfigPanel.tsx'),
  'utf8',
);
const cardsSource = readFileSync(
  resolve(process.cwd(), 'src/pages/device/PlugAndPlay/EnbQuickSettingsCards.tsx'),
  'utf8',
);
const enbSection = panelSource.slice(
  panelSource.indexOf("{deviceType === 'eNB'"),
  panelSource.indexOf("{deviceType === 'GSM'"),
);

describe('eNB parameter editor', () => {
  it('uses the LTE quick-settings cards and puts template extensions after them', () => {
    expect(enbSection).toContain('<EnbQuickSettingsCards');
    expect(enbSection).toContain('<CommonQuickSettingsNetworkCards paramModelName={paramModelName} onRequestEdit={onRequestEdit} />');
    expect(enbSection).toContain('<EnbTemplateExtraFieldGrid />');
    expect(enbSection.indexOf('<EnbTemplateExtraFieldGrid />'))
      .toBeGreaterThan(enbSection.indexOf('<EnbQuickSettingsCards'));
  });

  it('shows product-specific sync-source fields and gates supported PTP details by mode', () => {
    expect(cardsSource).toContain('getEnbProductSyncConfig(paramModelName)');
    expect(cardsSource).toContain("syncConfig.ptpModeValues.includes(String(syncMode ?? ''))");
    expect(cardsSource).toContain("group.id === 'device-time' && visibleSyncFields.length > 0 && (");
    expect(cardsSource).toContain("title={t('provision.syncSourceConfig')}");
    expect(cardsSource).toContain('syncConfig.fields.slice(0, syncConfig.collapsedFieldCount)');
  });

  it('omits legacy unsupported controls', () => {
    for (const field of [
      'ipsecSwitch',
      'ipsecRightIkePort',
      'mmePort',
      'authBy',
      'serviceMask',
      'serviceGateway',
      'serviceGatewayMask',
      'serviceVlan',
      'mgmtMask',
      'mgmtGateway',
      'mgmtGatewayMask',
      'mgmtVlan',
      'wanSendEnable',
      'wanOamTr069',
      'wanS1c',
      'wanS1u',
      'wanX2ap',
    ]) {
      expect(enbSection).not.toContain(field);
    }
  });
});
