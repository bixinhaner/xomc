import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

const pageSource = readFileSync(
  resolve(process.cwd(), 'src/pages/device/PlugAndPlay/AddPolicyPage.tsx'),
  'utf8',
);
const sharedPanelSource = readFileSync(
  resolve(process.cwd(), 'src/pages/device/PlugAndPlay/CommonParameterConfigPanel.tsx'),
  'utf8',
);
const networkCardsSource = readFileSync(
  resolve(process.cwd(), 'src/pages/device/PlugAndPlay/CommonQuickSettingsNetworkCards.tsx'),
  'utf8',
);
const drawerSection = pageSource.slice(
  pageSource.indexOf('{/* Config Detail/Edit Drawer */}'),
  pageSource.indexOf('{/* Firmware package import belongs to the target version. */}'),
);

describe('specified-device parameter editor', () => {
  it('uses the same parameter field panel as common parameter configuration', () => {
    expect(drawerSection).toContain('<ParameterConfigFields');
    expect(drawerSection).toContain('scope="device"');
    expect(drawerSection).not.toContain('<GnbNetworkConfigCards');
    expect(drawerSection).not.toContain('<Collapse.Panel');
    expect(sharedPanelSource).toContain('export function ParameterConfigFields');
    expect(sharedPanelSource).toContain('<CommonQuickSettingsNetworkCards paramModelName={paramModelName} onRequestEdit={onRequestEdit} />');
  });

  it('lets a delete action promote the read-only drawer into edit mode', () => {
    expect(drawerSection).toContain('onRequestEdit={() => setConfigDetailMode(\'edit\')}');
    expect(networkCardsSource).toContain('onRequestEdit?.();');
    expect(networkCardsSource).toContain('<ConfigProvider componentDisabled={onRequestEdit ? false : undefined}>');
  });

  it('offers a per-device download for the latest edited workbook', () => {
    expect(pageSource).toContain('handleDownloadParamConfig(record)');
    expect(pageSource).toContain('createParamConfigWorkbook([withTemplateSheetParameters(record)])');
    expect(pageSource).toContain("{t('common.download')}");
  });
});
