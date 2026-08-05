import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const pageSource = readFileSync(
  resolve(process.cwd(), 'src/pages/device/PlugAndPlay/AddPolicyPage.tsx'),
  'utf8',
);
const cardsSource = readFileSync(
  resolve(process.cwd(), 'src/pages/device/PlugAndPlay/EnbQuickSettingsCards.tsx'),
  'utf8',
);
const enbSection = pageSource.slice(
  pageSource.indexOf('{/* eNB fields aligned with device quick settings; template-only fields follow. */}'),
  pageSource.indexOf('{/* gNB specific fields */}'),
);

describe('eNB parameter editor', () => {
  it('uses the LTE quick-settings cards and puts template extensions after them', () => {
    expect(enbSection).toContain('<EnbQuickSettingsCards />');
    expect(enbSection).toContain('<EnbTemplateExtraFieldGrid />');
    expect(enbSection.indexOf('<EnbTemplateExtraFieldGrid />'))
      .toBeGreaterThan(enbSection.indexOf('<EnbQuickSettingsCards />'));
  });

  it('shows sync-source mode separately and gates the LTE 1588 details by mode', () => {
    expect(cardsSource).toContain("const syncMode = Form.useWatch('PpsTimeMode', form)");
    expect(cardsSource).toContain('const show1588Settings = isPtpDetailsVisible(syncMode)');
    expect(cardsSource).toContain("group.id === 'device-time' && (");
    expect(cardsSource).toContain("title={t('provision.syncSourceConfig')}");
    expect(cardsSource).toContain('ENB_1588_TEMPLATE_FIELDS.slice(0, 2)');
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
