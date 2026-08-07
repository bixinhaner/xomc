import { describe, expect, it } from 'vitest';
import * as XLSX from 'xlsx';
import {
  createParamConfigWorkbook,
  createParamConfigTemplateWorkbook,
  mergeImportedParamConfigs,
  ParamConfigWorkbookError,
  parseParamConfigWorkbook,
} from './paramConfigWorkbook';

describe('parameter config workbook', () => {
  it('creates the GSM template with spreadsheet defaults', () => {
    const workbook = createParamConfigTemplateWorkbook('GSM');
    expect(workbook.SheetNames).toEqual(['GSM']);
    expect(XLSX.utils.sheet_to_json<unknown[]>(workbook.Sheets.GSM, {
      header: 1,
      defval: '',
    })).toEqual([
      ['Serial Number', 'IPA', 'Unit ID', 'Remote IP', 'Bind IP', 'WAN IP', 'Synchronization', 'OMC'],
      ['', '6969', '', '', '', '', 'GNSS', ''],
    ]);
  });

  it('includes the supported LTE and NR planning fields in generated templates', () => {
    const lte = createParamConfigTemplateWorkbook('eNB');
    const nr = createParamConfigTemplateWorkbook('gNB');
    expect(XLSX.utils.sheet_to_json<unknown[]>(lte.Sheets.NETWORK, { header: 1 })[0])
      .toEqual(['*SERIAL_NUMBER', 'WAN IP', 'NTP Enable', 'NTP Server1', 'Local Time Zone']);
    expect(XLSX.utils.sheet_to_json<unknown[]>(nr.Sheets.IPSEC, { header: 1 })[0])
      .toContain('LEFT_INTERFACE');
    expect(XLSX.utils.sheet_to_json<unknown[]>(nr.Sheets.DEVICE, { header: 1 })[0])
      .not.toContain('Time Zone Term');
    expect(XLSX.utils.sheet_to_json<unknown[]>(nr.Sheets.INTERFACE, { header: 1 })[0])
      .not.toContain('OMC IP');
    expect(XLSX.utils.sheet_to_json<unknown[]>(nr.Sheets.CELL, { header: 1 })[0])
      .toEqual(expect.arrayContaining(['PowerModify', 'OffsetToPointA', 'SsbSubcarrierOffset']));
  });

  it('round-trips exported parameter configuration rows', () => {
    const workbook = createParamConfigWorkbook([{
      serialNumber: 'SN-001',
      cellName: 'Cell A',
      bandsSupport: 38,
      bandWidth: '20MHz',
      frequency: 36000,
      subframeAssignment: 2,
      updatedBy: 'admin',
      updatedAt: '2026-07-30 11:30:00',
    }]);
    const bytes = XLSX.write(workbook, { type: 'array', bookType: 'xlsx' }) as ArrayBuffer;

    expect(parseParamConfigWorkbook(bytes, 'eNB', '2026-07-30 11:31:00')).toEqual([{
      deviceType: 'eNB',
      serialNumber: 'SN-001',
      cellName: 'Cell A',
      bandsSupport: 38,
      bandWidth: '20MHz',
      frequency: 36000,
      subframeAssignment: 2,
      updatedBy: 'admin',
      updatedAt: '2026-07-30 11:30:00',
    }]);
  });

  it('rejects a row that does not contain the required station code', () => {
    const worksheet = XLSX.utils.aoa_to_sheet([
      ['基站编码', '支持频段', '带宽', '频点', '子帧配比'],
      ['', 38, '20MHz', 36000, 2],
    ]);
    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, worksheet, '参数配置');
    const bytes = XLSX.write(workbook, { type: 'array', bookType: 'xlsx' }) as ArrayBuffer;

    expect(() => parseParamConfigWorkbook(bytes, 'eNB', '2026-07-30 11:31:00')).toThrow(
      expect.objectContaining<Partial<ParamConfigWorkbookError>>({
        code: 'invalid_row',
        row: 2,
        field: 'serialNumber',
      }),
    );
  });

  it('appends new stations and updates an existing station by station code', () => {
    const existing = [
      { serialNumber: 'SN-001', cellName: 'Old name' },
      { serialNumber: 'SN-002', cellName: 'Kept' },
    ];
    const imported = [
      { serialNumber: 'SN-001', cellName: 'New name' },
      { serialNumber: 'SN-003', cellName: 'Added' },
    ];

    expect(mergeImportedParamConfigs(existing, imported, 'append')).toEqual([
      { serialNumber: 'SN-001', cellName: 'New name' },
      { serialNumber: 'SN-002', cellName: 'Kept' },
      { serialNumber: 'SN-003', cellName: 'Added' },
    ]);
    expect(mergeImportedParamConfigs(existing, imported, 'replace')).toEqual(imported);
  });

  it('imports rows from the supplied 4G default template structure', () => {
    const worksheet = XLSX.utils.aoa_to_sheet([
      ['eNB', '', 'Cell'],
      ['*SERIAL_NUMBER', '*CELL_NUMBER', 'CELL_NAME', '*ECI', '*BAND', '*EARFCN_DL', '*BANDWIDTH_DL', '*PCI', 'SPECIAL_SUBFRAME_PATTERNS', 'SUBFRAME_ASSIGNMENT'],
      ['4G-SN-001', 1, 'LTE Cell', 222218, 42, 43000, 'n50', 218, 5, 1],
    ]);
    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, worksheet, 'CELL');
    const bytes = XLSX.write(workbook, { type: 'array', bookType: 'xlsx' }) as ArrayBuffer;

    expect(parseParamConfigWorkbook(bytes, 'eNB', '2026-07-30 14:00:00')).toMatchObject([{
      deviceType: 'eNB',
      serialNumber: '4G-SN-001',
      cellName: 'LTE Cell',
      bandsSupport: 42,
      bandWidth: 'n50',
      frequency: 43000,
      subframeAssignment: 1,
      updatedBy: 'import',
      updatedAt: '2026-07-30 14:00:00',
    }]);
  });

  it('imports rows from the supplied 5G default template structure', () => {
    const worksheet = XLSX.utils.aoa_to_sheet([
      ['*Serial Number', 'gNB Name', '*gNB ID', '*gNB Lenth', '*PCI', 'SSB Frequency', 'Freq BandIndicator', 'NRARFCNDL', 'NRARFCNUL', 'DLBandwidth'],
      ['length:1-64', 'length:1-150'],
      ['5G-SN-001', 'NR Cell', 123456, 24, 150, 3600000, 78, 360000, 360000, '100MHz'],
    ]);
    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, worksheet, 'CELL');
    const bytes = XLSX.write(workbook, { type: 'array', bookType: 'xlsx' }) as ArrayBuffer;

    expect(parseParamConfigWorkbook(bytes, 'gNB', '2026-07-30 14:00:00')).toMatchObject([{
      deviceType: 'gNB',
      serialNumber: '5G-SN-001',
      cellName: 'NR Cell',
      bandsSupport: 78,
      bandWidth: '100MHz',
      frequency: 360000,
      updatedBy: 'import',
      updatedAt: '2026-07-30 14:00:00',
    }]);
  });

  it('identifies a default template that has no data in any sheet', () => {
    const worksheet = XLSX.utils.aoa_to_sheet([
      ['*Serial Number', 'gNB Name', '*gNB ID', '*gNB Lenth', '*PCI', 'SSB Frequency', 'Freq BandIndicator', 'NRARFCNDL', 'NRARFCNUL', 'DLBandwidth'],
      ['length:1-64', 'length:1-150', 'unsignedInt', 'unsignedInt', 'long', 'unsignedInt', 'unsignedInt', 'unsignedInt', 'unsignedInt', 'MHz'],
    ]);
    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, worksheet, 'CELL');
    const bytes = XLSX.write(workbook, { type: 'array', bookType: 'xlsx' }) as ArrayBuffer;

    expect(() => parseParamConfigWorkbook(bytes, 'gNB', '2026-07-30 14:00:00')).toThrow(
      expect.objectContaining<Partial<ParamConfigWorkbookError>>({
        code: 'template_no_data',
      }),
    );
  });

  it('imports a 5G station when CELL is empty but another sheet contains parameters', () => {
    const cellWorksheet = XLSX.utils.aoa_to_sheet([
      ['*Serial Number', 'gNB Name', '*gNB ID', 'Freq BandIndicator', 'NRARFCNDL', 'DLBandwidth'],
      ['length:1-64', 'length:1-150', 'unsignedInt', 'unsignedInt', 'unsignedInt', 'MHz'],
    ]);
    const deviceWorksheet = XLSX.utils.aoa_to_sheet([
      ['Serial Number', 'Periodic Inform Enable', 'NTP Enable'],
      ['length:1-64', 'boolean', 'boolean'],
      ['5G-SN-DEVICE-001', true, true],
    ]);
    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, deviceWorksheet, 'DEVICE');
    XLSX.utils.book_append_sheet(workbook, cellWorksheet, 'CELL');
    const bytes = XLSX.write(workbook, { type: 'array', bookType: 'xlsx' }) as ArrayBuffer;

    expect(parseParamConfigWorkbook(bytes, 'gNB', '2026-07-30 14:00:00')).toEqual([{
      deviceType: 'gNB',
      serialNumber: '5G-SN-DEVICE-001',
      updatedBy: 'import',
      updatedAt: '2026-07-30 14:00:00',
      sheetParameters: {
        DEVICE: [{
          'Serial Number': '5G-SN-DEVICE-001',
          'Periodic Inform Enable': true,
          'NTP Enable': true,
        }],
      },
    }]);
  });

  it('preserves empty template columns so every parameter can be edited later', () => {
    const worksheet = XLSX.utils.aoa_to_sheet([
      ['Serial Number', 'NTP Enable', 'NTP Server1'],
      ['length:1-64', 'boolean', 'string'],
      ['5G-SN-EMPTY-001', true, ''],
    ]);
    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, worksheet, 'DEVICE');
    const bytes = XLSX.write(workbook, { type: 'array', bookType: 'xlsx' }) as ArrayBuffer;

    expect(parseParamConfigWorkbook(bytes, 'gNB', '2026-07-30 14:00:00'))
      .toMatchObject([{
        sheetParameters: {
          DEVICE: [{
            'Serial Number': '5G-SN-EMPTY-001',
            'NTP Enable': true,
            'NTP Server1': '',
          }],
        },
      }]);
  });

  it('merges parameters for the same station across multiple sheets', () => {
    const cellWorksheet = XLSX.utils.aoa_to_sheet([
      ['*Serial Number', 'gNB Name', '*gNB ID', 'Freq BandIndicator', 'NRARFCNDL', 'DLBandwidth'],
      ['length:1-64', 'length:1-150', 'unsignedInt', 'unsignedInt', 'unsignedInt', 'MHz'],
      ['5G-SN-MERGED-001', 'Merged NR Cell', 123, 78, 360000, '100MHz'],
    ]);
    const plmnWorksheet = XLSX.utils.aoa_to_sheet([
      ['Serial Number', '*NCI', '*PLMN ID'],
      ['length:1-64', 'unsignedInt', '5-6 Digit Integer'],
      ['5G-SN-MERGED-001', 123456, 46000],
    ]);
    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, cellWorksheet, 'CELL');
    XLSX.utils.book_append_sheet(workbook, plmnWorksheet, 'PLMN');
    const bytes = XLSX.write(workbook, { type: 'array', bookType: 'xlsx' }) as ArrayBuffer;

    expect(parseParamConfigWorkbook(bytes, 'gNB', '2026-07-30 14:00:00')).toEqual([{
      deviceType: 'gNB',
      serialNumber: '5G-SN-MERGED-001',
      cellName: 'Merged NR Cell',
      bandsSupport: 78,
      bandWidth: '100MHz',
      frequency: 360000,
      updatedBy: 'import',
      updatedAt: '2026-07-30 14:00:00',
      sheetParameters: {
        CELL: [expect.objectContaining({
          '*Serial Number': '5G-SN-MERGED-001',
          'gNB Name': 'Merged NR Cell',
        })],
        PLMN: [{
          'Serial Number': '5G-SN-MERGED-001',
          '*NCI': 123456,
          '*PLMN ID': 46000,
        }],
      },
    }]);
  });

  it('exports and re-imports parameters that came from a non-CELL sheet', () => {
    const sourceWorksheet = XLSX.utils.aoa_to_sheet([
      ['Serial Number', 'Periodic Inform Enable'],
      ['5G-SN-DEVICE-EXPORT', true],
    ]);
    const sourceWorkbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(sourceWorkbook, sourceWorksheet, 'DEVICE');
    const sourceBytes = XLSX.write(sourceWorkbook, { type: 'array', bookType: 'xlsx' }) as ArrayBuffer;
    const imported = parseParamConfigWorkbook(sourceBytes, 'gNB', '2026-07-30 14:00:00');

    const exportedWorkbook = createParamConfigWorkbook(imported);
    expect(exportedWorkbook.SheetNames).toEqual(['DEVICE']);
    const exportedBytes = XLSX.write(
      exportedWorkbook,
      { type: 'array', bookType: 'xlsx' },
    ) as ArrayBuffer;

    expect(parseParamConfigWorkbook(
      exportedBytes,
      'gNB',
      '2026-07-30 14:01:00',
    )).toMatchObject([{
      serialNumber: '5G-SN-DEVICE-EXPORT',
      sheetParameters: {
        DEVICE: [{
          'Serial Number': '5G-SN-DEVICE-EXPORT',
          'Periodic Inform Enable': true,
        }],
      },
    }]);
  });
});
