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
const gnbQuickSettingsFieldsSource = readFileSync(
  resolve(process.cwd(), 'src/pages/device/PlugAndPlay/gnbQuickSettingsFields.ts'),
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
    expect(pageSource).toContain("const submittedFunctionModule = isFunctionModule(values.functionModule)");
    expect(pageSource).toContain("if (submittedFunctionModule === '2' && values.selfConfigEnable)");
    expect(pageSource).toContain('let commonParamConfig = {};');
    expect(pageSource).toContain('let submittedParamConfigList: ParamConfig[] = [];');
    expect(pageSource).toContain('commonParamConfig = {};');
    expect(pageSource).toContain('paramConfigMode: submittedParamConfigMode');
    expect(pageSource).toContain('paramConfigList: submittedParamConfigList');
  });

  it('keeps module selection in the form snapshot before submit', () => {
    expect(pageSource).toContain('function isFunctionModule');
    expect(pageSource).toContain('<Form.Item name="functionModule" noStyle>');
    expect(pageSource).toContain('<Radio.Group style={{ width: \'100%\' }}>');
    expect(pageSource).not.toContain('handleFunctionModuleChange');
  });

  it('recovers the selected module from legacy policy payloads', () => {
    expect(pageSource).toContain('function inferFunctionModule');
    expect(pageSource).toContain('flags.selfConfigEnabled === true');
    expect(pageSource).toContain('hasObjectContent(config.commonParamConfig)');
    expect(pageSource).toContain('config.paramConfigList.length > 0');
    expect(pageSource).toContain('licenseEnabled: persistedPolicy.licenseEnabled');
    expect(pageSource).toContain('selfConfigEnabled: persistedPolicy.selfConfigEnabled');
  });

  it('recovers specified-device mode from existing per-device configs when mode is missing', () => {
    expect(pageSource).toContain('function inferParamConfigMode');
    expect(pageSource).toContain("config.paramConfigMode === 'specified' || config.paramConfigMode === 'common'");
    expect(pageSource).toContain('Array.isArray(config.paramConfigList) && config.paramConfigList.length > 0');
    expect(pageSource).toContain('paramConfigMode: inferParamConfigMode(config)');
  });

  it('initializes common parameter form before switching to parameter module', () => {
    expect(pageSource).toContain('withInitialCommonParamConfig(');
    expect(pageSource).not.toContain("functionModule !== '2'");
  });

  it('allows parameter templates to be imported from multiple files at once', () => {
    expect(pageSource).toContain('const previewImportConfigFiles = useCallback(async (files: UploadFile[]) => {');
    expect(pageSource).toContain('for (const [fileIndex, uploadFile] of files.entries())');
    expect(pageSource).toContain('const importedConfigs: ParamConfig[] = [];');
    expect(pageSource).toContain('importedConfigs.push(...rows.map((row, index) => (');
    expect(pageSource).toContain('materializeParamConfigDisplayValues({');
    expect(pageSource).toContain('multiple');
    expect(pageSource).toContain('beforeUpload={(_, fileList) => {');
    expect(pageSource).toContain('const nextFiles = toSelectedParamImportFiles(fileList as UploadFile[]);');
    expect(pageSource).not.toContain('void previewImportConfig(file);');
  });

  it('uses the same parameter field panel as common parameter configuration', () => {
    expect(drawerSection).toContain('<ParameterConfigFields');
    expect(drawerSection).toContain('scope="device"');
    expect(drawerSection).not.toContain('<GnbNetworkConfigCards');
    expect(drawerSection).not.toContain('<Collapse.Panel');
    expect(sharedPanelSource).toContain('export function ParameterConfigFields');
    expect(sharedPanelSource).toContain('<PrimaryRadioInstanceEditor');
    expect(sharedPanelSource).toContain('deviceType={deviceType}');
    expect(sharedPanelSource).toContain('productClass={productClass}');
    expect(sharedPanelSource).toContain("excludedFieldIds={commonScope && deviceType === 'gNB' ? ['PCI'] : []}");
    expect(sharedPanelSource).toContain('readOnly={readOnly}');
    expect(sharedPanelSource).toContain('<CommonQuickSettingsNetworkCards paramModelName={paramModelName} onRequestEdit={onRequestEdit} readOnly={readOnly} />');
  });

  it('keeps the view drawer readable while making parameter fields read-only', () => {
    expect(drawerSection).toContain('<Form form={configForm} layout="vertical" style={{ paddingBottom: 60 }}>');
    expect(drawerSection).not.toContain('disabled={configDetailMode === \'view\'}');
    expect(drawerSection).toContain("readOnly={configDetailMode === 'view'}");
    expect(pageSource).toContain('readOnly={isView}');
  });

  it('lets a delete action promote the read-only drawer into edit mode', () => {
    expect(drawerSection).toContain('onRequestEdit={() => setConfigDetailMode(\'edit\')}');
    expect(networkCardsSource).toContain('onRequestEdit?.();');
    expect(networkCardsSource).toContain('<ConfigProvider componentDisabled={onRequestEdit ? false : undefined}>');
  });

  it('shows an explicit edit action in the read-only config drawer before saving', () => {
    expect(drawerSection).toContain("configDetailMode === 'view' && moduleActions.edit");
    expect(drawerSection).toContain("onClick={() => setConfigDetailMode('edit')}");
    expect(drawerSection).toContain("configDetailMode === 'edit'");
    expect(drawerSection).toContain('handleConfigFormSubmit');
  });

  it('persists edits made in the device config drawer immediately for existing policies', () => {
    const submitSection = pageSource.slice(
      pageSource.indexOf('const handleConfigFormSubmit'),
      pageSource.indexOf('const toSelectedParamImportFiles'),
    );

    expect(submitSection).toContain('buildParamConfigListPolicyUpdate(persistedPolicy, next)');
    expect(submitSection).toContain('savePolicyMutation.mutateAsync');
    expect(submitSection).toContain('setParamConfigList(previous)');
  });

  it('offers a per-device download for the latest edited workbook', () => {
    expect(pageSource).toContain('handleDownloadParamConfig(record)');
    expect(pageSource).toContain('createParamConfigWorkbook([record], exportContext)');
    expect(pageSource).not.toContain('withTemplateSheetParameters(record)');
    expect(pageSource).toContain("{t('common.download')}");
  });

  it('maps imported workbook columns into the existing device detail fields', () => {
    expect(drawerSection).toContain('<ParameterConfigFields');
    expect(drawerSection).toContain('scope="device"');
    expect(sharedPanelSource).not.toContain('function ImportedSheetParameterFields');
    expect(sharedPanelSource).not.toContain('provision.importedWorkbookFields');
    expect(gnbQuickSettingsFieldsSource).toContain("name: sheetField('DEVICE', `NTP Server${index}`)");
  });
});
