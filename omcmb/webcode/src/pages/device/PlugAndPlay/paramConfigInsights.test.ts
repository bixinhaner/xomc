import { describe, expect, it } from 'vitest';
import { buildParamConfigImportPreview, buildParamConfigInsights } from './paramConfigInsights';
import { PARAM_CONFIG_FIELD_MAPPING_MATRIX } from './paramConfigFieldMappings';

describe('parameter configuration insights', () => {
  it('extracts NR-specific summary fields from imported sheets', () => {
    const insights = buildParamConfigInsights([{
      serialNumber: 'SN-001',
      deviceType: 'gNB',
      updatedBy: 'import',
      sheetParameters: {
        CELL: [{ '*gNB ID': 123, '*PCI': 42, 'Freq BandIndicator': 78, NRARFCNDL: 640000, 'SSB Frequency': 630000, DLBandwidth: '100MHz' }],
        PLMN: [{ '*TAC': 1001 }],
      },
    }]);

    expect(insights.get('SN-001')).toEqual({
      source: 'import',
      validationStatus: 'valid',
      gnbId: '123',
      pci: '42',
      band: '78',
      bandwidth: '100MHz',
      frequency: '640000',
      ssbFrequency: '630000',
      tac: '1001',
    });
  });

  it('marks duplicate NR identifiers and missing required identifiers', () => {
    const insights = buildParamConfigInsights([
      { serialNumber: 'A', deviceType: 'gNB', gnbId: 1, pci: 10 },
      { serialNumber: 'B', deviceType: 'gNB', gnbId: 1, pci: 11 },
      { serialNumber: 'C', deviceType: 'gNB', gnbId: 3 },
    ]);
    expect(insights.get('A')?.validationStatus).toBe('conflict');
    expect(insights.get('B')?.validationStatus).toBe('conflict');
    expect(insights.get('C')?.validationStatus).toBe('incomplete');
  });

  it('uses workbook TRPath mappings to validate dynamically named template fields', () => {
    const insights = buildParamConfigInsights([{
      serialNumber: 'SN-DYNAMIC',
      deviceType: 'gNB',
      updatedBy: 'import',
      sheetParameters: {
        DEVICE: [{ 'Base Station Identifier': 101 }],
        CELL: [{ 'Physical Cell Identifier': 22 }],
      },
      workbookMappings: [{
        displayName: 'Base Station Identifier',
        sheet: 'DEVICE',
        header: 'Base Station Identifier',
        trPath: 'Device.Services.FAPService.1.CellConfig.NR.RAN.GNBID',
        source: 'system',
      }, {
        displayName: 'Physical Cell Identifier',
        sheet: 'CELL',
        header: 'Physical Cell Identifier',
        trPath: 'Device.Services.FAPService.1.CellConfig.NR.RAN.PCI',
        source: 'system',
      }],
    }]);

    expect(insights.get('SN-DYNAMIC')).toMatchObject({
      validationStatus: 'valid',
      gnbId: '101',
      pci: '22',
    });
  });

  it('extracts BaiBNQ summary values from the generated workbook paths', () => {
    const insights = buildParamConfigInsights([{
      serialNumber: '120200054822CHB0004',
      deviceType: 'gNB',
      updatedBy: 'import',
      sheetParameters: {
        CELL: [{
          Band: 78,
          PCI: 500,
          'DL Carrier Bandwidth': 51,
          NRARFCNDL: 377483,
          'SSB Frequency': 614522,
          TAC: 1,
        }],
        DEVICE: [{ 'gNB ID': 2451329 }],
      },
      workbookMappings: [
        { displayName: 'Band', sheet: 'CELL', header: 'Band', trPath: 'Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.FrequencyInfoDLSIB.MultiFrequencyBandListNRSIB.1.FreqBandIndicatorNR', source: 'system' },
        { displayName: 'PCI', sheet: 'CELL', header: 'PCI', trPath: 'Device.Services.FAPService.1.CellConfig.1.NR.RAN.RF.PhyCellID', source: 'system' },
        { displayName: 'DL Carrier Bandwidth', sheet: 'CELL', header: 'DL Carrier Bandwidth', trPath: 'Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.FrequencyInfoDLSIB.ScsSpecificCarrierList.1.SCSSpecificCarrier.CarrierBandwidth', source: 'system' },
        { displayName: 'TAC', sheet: 'CELL', header: 'TAC', trPath: 'Device.Services.FAPService.1.CellConfig.1.NR.CN.TA.1.TAC', source: 'system' },
        { displayName: 'gNB ID', sheet: 'DEVICE', header: 'gNB ID', trPath: 'Device.Services.FAPService.1.FAPControl.NR.RAN.Common.gNBId', source: 'system' },
      ],
    }]);

    expect(insights.get('120200054822CHB0004')).toMatchObject({
      validationStatus: 'valid',
      gnbId: '2451329',
      pci: '500',
      band: '78',
      bandwidth: '51',
      frequency: '377483',
      ssbFrequency: '614522',
      tac: '1',
    });
  });

  it('accepts legacy DL NRARFCN field spelling when summarizing persisted rows', () => {
    const insights = buildParamConfigInsights([{
      serialNumber: 'SN-LEGACY',
      deviceType: 'gNB',
      gnbId: 1001,
      pci: 10,
      nrarfcndl: 640000,
    }]);

    expect(insights.get('SN-LEGACY')).toMatchObject({
      validationStatus: 'valid',
      frequency: '640000',
    });
  });

  it('previews add, update and duplicate import actions', () => {
    expect(buildParamConfigImportPreview(
      [{ serialNumber: 'SN-001' }],
      [{ serialNumber: 'SN-001' }, { serialNumber: 'SN-002' }, { serialNumber: 'SN-002' }],
    )).toEqual([
      { serialNumber: 'SN-001', action: 'update' },
      { serialNumber: 'SN-002', action: 'add' },
      { serialNumber: 'SN-002', action: 'duplicate' },
    ]);
  });

  it('keeps critical NR fields in the executable workbook mapping contract', () => {
    expect(PARAM_CONFIG_FIELD_MAPPING_MATRIX.gNB).toEqual(expect.arrayContaining([
      { field: 'gnbId', sheet: 'CELL', header: '*gNB ID' },
      { field: 'gnbIdLength', sheet: 'CELL', header: '*gNB Lenth' },
      { field: 'pci', sheet: 'CELL', header: '*PCI' },
      { field: 'nrarfcnndl', sheet: 'CELL', header: 'NRARFCNDL' },
      { field: 'tac', sheet: 'PLMN', header: '*TAC' },
    ]));
  });
});
