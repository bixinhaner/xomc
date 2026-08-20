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
  it('renders common and specified-device choices as mutually exclusive checkboxes instead of tabs', () => {
    const selfConfigStart = pageSource.indexOf('const renderSelfConfig = () => (');
    const selfConfigSection = pageSource.slice(
      selfConfigStart,
      pageSource.indexOf('  return (', selfConfigStart),
    );

    expect(selfConfigSection).toContain("<Checkbox");
    expect(selfConfigSection).toContain("checked={paramConfigMode === 'common'}");
    expect(selfConfigSection).toContain("checked={paramConfigMode === 'specified'}");
    expect(selfConfigSection).toContain("form.setFieldsValue({ paramConfigMode: 'common' })");
    expect(selfConfigSection).toContain("form.setFieldsValue({ paramConfigMode: 'specified' })");
    expect(selfConfigSection).not.toContain('<Tabs');
  });

  it('submits only the selected parameter configuration mode', () => {
    expect(pageSource).toContain("const submittedParamConfigMode: ParamConfigMode = values.paramConfigMode === 'specified' ? 'specified' : 'common';");
    expect(pageSource).toContain("submittedParamConfigList = [];");
    expect(pageSource).toContain('commonParamConfig = {};');
    expect(pageSource).toContain('paramConfigMode: submittedParamConfigMode');
    expect(pageSource).toContain('paramConfigList: submittedParamConfigList');
  });

  it('uses the same parameter field panel as common parameter configuration', () => {
    expect(drawerSection).toContain('<ParameterConfigFields');
    expect(drawerSection).toContain('scope="device"');
    expect(drawerSection).not.toContain('<GnbNetworkConfigCards');
    expect(drawerSection).not.toContain('<Collapse.Panel');
    expect(sharedPanelSource).toContain('export function ParameterConfigFields');
    expect(sharedPanelSource).toContain('<PrimaryRadioInstanceEditor deviceType={deviceType} productClass={productClass} />');
    expect(sharedPanelSource).toContain('<CommonQuickSettingsNetworkCards paramModelName={paramModelName} onRequestEdit={onRequestEdit} />');
  });

  it('lets a delete action promote the read-only drawer into edit mode', () => {
    expect(drawerSection).toContain('onRequestEdit={() => setConfigDetailMode(\'edit\')}');
    expect(networkCardsSource).toContain('onRequestEdit?.();');
    expect(networkCardsSource).toContain('<ConfigProvider componentDisabled={onRequestEdit ? false : undefined}>');
  });

  it('offers a per-device download for the latest edited workbook', () => {
    expect(pageSource).toContain('handleDownloadParamConfig(record)');
    expect(pageSource).toContain('createParamConfigWorkbook([record], { productClass })');
    expect(pageSource).not.toContain('withTemplateSheetParameters(record)');
    expect(pageSource).toContain("{t('common.download')}");
  });
});
