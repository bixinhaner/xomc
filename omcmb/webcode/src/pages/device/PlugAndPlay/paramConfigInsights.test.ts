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
