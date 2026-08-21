import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const panelSource = readFileSync(
  resolve(process.cwd(), 'src/pages/device/PlugAndPlay/CommonParameterConfigPanel.tsx'),
  'utf8',
);
const gnbSection = panelSource.slice(
  panelSource.indexOf("{deviceType === 'gNB'"),
  panelSource.indexOf("{deviceType === 'eNB'"),
);

describe('gNB parameter editor quick-settings layout', () => {
  it('uses the same card and three-column layout as quick settings', () => {
    expect(gnbSection).not.toContain('<Collapse.Panel');
    expect(gnbSection).toContain('<GnbQuickSettingsCards');
    expect(gnbSection).toContain('<CommonQuickSettingsNetworkCards paramModelName={paramModelName} onRequestEdit={onRequestEdit} readOnly={readOnly} />');
  });

  it('keeps batch allocation common-only and template extensions after quick settings', () => {
    expect(gnbSection).toContain('{commonScope && (');
    expect(panelSource).toContain("excludedFieldIds={commonScope && deviceType === 'gNB' ? ['PCI'] : []}");
    expect(gnbSection).toContain("excludedFieldIds={commonScope ? ['gNBId', 'PCI'] : []}");
    expect(gnbSection.indexOf('<GnbTemplateExtraFieldGrid'))
      .toBeGreaterThan(gnbSection.indexOf('<GnbQuickSettingsCards'));
  });
});
