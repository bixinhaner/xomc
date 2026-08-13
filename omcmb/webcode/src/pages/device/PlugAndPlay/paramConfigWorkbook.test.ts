import { describe, expect, it } from 'vitest';
import * as XLSX from 'xlsx';
import ExcelJS from 'exceljs';
import {
  createParamConfigWorkbook,
  createParamConfigTemplateWorkbook,
  enrichParamConfigWorkbook,
  mergeImportedParamConfigs,
  ParamConfigWorkbookError,
  parseParamConfigWorkbook,
} from './paramConfigWorkbook';
import { getParamConfigExportFields } from './paramConfigExportFields';

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

  it('adds hover notes and dropdowns without occupying a data row', async () => {
    const workbook = await enrichParamConfigWorkbook(createParamConfigTemplateWorkbook('gNB'));
    const device = workbook.getWorksheet('DEVICE')!;
    const ipsec = workbook.getWorksheet('IPSEC')!;
    const cell = workbook.getWorksheet('CELL')!;

    expect(device.getCell('A1').note).toContain('参数类型：string');
    expect(device.getCell('A1').note).toContain('取值范围：长度 1-64');
    expect(device.getCell('C1').note).toContain('参数类型：bool');
    expect(device.getCell('C2').dataValidation.formulae).toEqual(['"true,false"']);
    expect(cell.getCell('J2').dataValidation.type).toBe('list');
    const leftIdentifierColumn = ipsec.getRow(1).values.indexOf('LEFT_IDENTIFIER');
    expect(ipsec.getCell(1, leftIdentifierColumn).note).toContain('参数类型：string');
    expect(device.getRow(2).values).not.toContain('参数类型');
  });

  it('uses quick-setting constraints first and parameter-model constraints as fallback', async () => {
    const source = createParamConfigTemplateWorkbook('gNB');
    const workbook = await enrichParamConfigWorkbook(source, {
      quickSettingsGroups: [{
        id: 'cell', titleZh: '小区', titleEn: 'Cell', multiInstance: false,
        params: [{
          name: 'PCI', titleZh: 'PCI', titleEn: 'PCI', type: 'enum',
          standardPath: 'Device.Cell.PCI',
          hint: '快速设置：规划值',
          enumOptions: [{ value: '10', label: '10' }, { value: '20', label: '20' }],
        }],
      }],
      quickSettingFields: [{
        id: 'PCI', name: '*PCI', labelKey: 'pci', control: 'select',
        range: '0 ~ 1007', condition: '射频参数启用',
        options: [{ value: '10' }, { value: '20' }],
      }],
      paramMappings: [{
        id: '1', paramModelId: 'model', standardPath: 'Device.Cell.PCI',
        privatePath: 'Device.Cell.PCI', entryType: 'parameter', access: 'readWrite',
        dataType: 'unsignedInt', changeApplies: 'Immediate', minValue: '0', maxValue: '3279165',
        isStorable: true, isActive: true,
      }, {
        id: '2', paramModelId: 'model', standardPath: 'Device.Cell.NRARFCNDL',
        privatePath: 'Device.Cell.NRARFCNDL', entryType: 'parameter', access: 'readWrite',
        dataType: 'unsignedInt', changeApplies: 'Immediate', minValue: '0', maxValue: '3279165',
        isStorable: true, isActive: true,
      }],
    });
    const cell = workbook.getWorksheet('CELL')!;
    const pciColumn = cell.getRow(1).values.indexOf('*PCI');
    const arfcnColumn = cell.getRow(1).values.indexOf('NRARFCNDL');

    expect(cell.getCell(1, pciColumn).note).toContain('参数类型：int');
    expect(cell.getCell(1, pciColumn).note).toContain('取值范围：0 ~ 1007');
    expect(cell.getCell(1, pciColumn).note).toContain('显示条件：射频参数启用');
    expect(cell.getCell(2, pciColumn).dataValidation.formulae).toEqual(['"10,20"']);
    expect(cell.getCell(1, arfcnColumn).note).toContain('参数类型：int');
    expect(cell.getCell(1, arfcnColumn).note).toContain('取值范围：0 ~ 3279165');
    expect(cell.getCell(1, arfcnColumn).note).not.toContain('设备数据模型为准');
  });

  it('uses each IPSEC TR path enum and carrier planning ranges as dropdowns', async () => {
    const workbook = await enrichParamConfigWorkbook(createParamConfigTemplateWorkbook('gNB'), {
      quickSettingsGroups: [{
        id: 'gnb-ipsec', titleZh: 'IPSec', titleEn: 'IPSec', multiInstance: true,
        objectPath: 'Device.FAP.Ipsec.{i}.',
        params: [{ name: 'IKE_ENCRYPTION', titleZh: '加密', titleEn: 'Encryption', leaf: 'IKE_ENCRYPTION' }],
      }, {
        id: 'gnb-cell', titleZh: '小区', titleEn: 'Cell', multiInstance: true,
        params: [
          { name: 'DLSubCarrierSpacing', titleZh: '载波间隔', titleEn: 'SCS' },
          { name: 'DLCarrierBandWidth', titleZh: '下行带宽', titleEn: 'DL bandwidth' },
        ],
      }, {
        id: 'device-time', titleZh: '时间', titleEn: 'Time', multiInstance: false,
        params: [{ name: 'LocalTimeZoneName', titleZh: '时区', titleEn: 'Timezone' }],
      }],
      quickSettingFields: getParamConfigExportFields('gNB'),
      paramMappings: [{
        id: 'ipsec-1', paramModelId: 'model', standardPath: 'Device.FAP.Ipsec.{i}.IKE_ENCRYPTION',
        privatePath: 'Device.FAP.Ipsec.{i}.IKE_ENCRYPTION', entryType: 'parameter', access: 'readWrite',
        dataType: 'STRING', changeApplies: 'Immediate', enumValues: 'aes128,aes256,3des',
        enumLabels: 'aes128,aes256,3des', isStorable: true, isActive: true,
      }],
    });
    const ipsec = workbook.getWorksheet('IPSEC')!;
    const cell = workbook.getWorksheet('CELL')!;
    const encryptionColumn = ipsec.getRow(1).values.indexOf('IKE_ENCRYPTION');
    const scsColumn = cell.getRow(1).values.indexOf('SubcarrierSpacing(DL)');
    const bandwidthColumn = cell.getRow(1).values.indexOf('DLBandwidth');
    const device = workbook.getWorksheet('DEVICE')!;
    const timezoneColumn = device.getRow(1).values.indexOf('Local Time Zone');

    expect(ipsec.getCell(1, encryptionColumn).note).toContain('可选值：aes128、aes256、3des');
    expect(ipsec.getCell(1, encryptionColumn).note).not.toContain('显示条件');
    expect(ipsec.getCell(2, encryptionColumn).dataValidation.formulae).toEqual(['"aes128,aes256,3des"']);
    expect(cell.getCell(2, scsColumn).dataValidation.formulae).toEqual(['"0,1,2"']);
    expect(cell.getCell(2, bandwidthColumn).dataValidation.type).toBe('list');
    expect(cell.getCell(2, bandwidthColumn).dataValidation.formulae)
      .toEqual([`INDIRECT("XOMC_BW_"&${cell.getColumn(scsColumn).letter}2)`]);
    expect(device.getCell(2, timezoneColumn).dataValidation.type).toBe('list');
    expect(device.getCell(2, timezoneColumn).dataValidation.formulae?.[0]).toMatch(/^XOMC_OPTIONS_/);
    expect(workbook.getWorksheet('__XOMC_OPTIONS')?.state).toBe('veryHidden');

    const serialized = await workbook.xlsx.writeBuffer();
    const reopened = new ExcelJS.Workbook();
    await reopened.xlsx.load(serialized);
    expect(reopened.getWorksheet('__XOMC_OPTIONS')?.state).toBe('veryHidden');
    expect(reopened.getWorksheet('DEVICE')?.getCell(2, timezoneColumn).dataValidation.type).toBe('list');
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

  it('uses the 5G network address-method values for Address Type', async () => {
    const workbook = await enrichParamConfigWorkbook(createParamConfigTemplateWorkbook('gNB'));
    const network = workbook.getWorksheet('INTERFACE')!;
    const addressTypeColumn = network.getRow(1).values.indexOf('Address Type');

    expect(network.getCell(1, addressTypeColumn).note)
      .toContain('可选值：DHCP、Static、DHCPv6、Staticv6');
    expect(network.getCell(2, addressTypeColumn).dataValidation.formulae)
      .toEqual(['"DHCP,Static,DHCPv6,Staticv6"']);
  });

  it('imports 5G static network rows using the new Address Type semantics', () => {
    const workbook = createParamConfigTemplateWorkbook('gNB');
    XLSX.utils.sheet_add_aoa(workbook.Sheets.INTERFACE, [[
      'NR-SN-001', 'eth0', 'Static', '192.0.2.10', '255.255.255.0', '',
      '192.0.2.1', 'OAM', 'wan', '100',
    ]], { origin: 'A2' });

    const rows = parseParamConfigWorkbook(
      XLSX.write(workbook, { type: 'array', bookType: 'xlsx' }),
      'gNB',
      'now',
    );
    expect(rows[0].sheetParameters?.INTERFACE?.[0]).toMatchObject({
      'Address Type': 'Static',
      'IP Address': '192.0.2.10',
      'Subnet Mask': '255.255.255.0',
      Gateway: '192.0.2.1',
    });
  });

  it.each([
    ['IPv4', '192.0.2.10', '255.255.255.0', '', '192.0.2.1', 'Address Type'],
    ['Static', '192.0.2.10', '', '', '192.0.2.1', 'Subnet Mask'],
    ['Staticv6', '2001:db8::10', '', '129', '2001:db8::1', 'Prefix Length'],
  ])('rejects invalid 5G network import values for %s', (
    addressType, ip, mask, prefix, gateway, field,
  ) => {
    const workbook = createParamConfigTemplateWorkbook('gNB');
    XLSX.utils.sheet_add_aoa(workbook.Sheets.INTERFACE, [[
      'NR-SN-001', 'eth0', addressType, ip, mask, prefix, gateway, 'OAM', 'wan', '100',
    ]], { origin: 'A2' });

    expect(() => parseParamConfigWorkbook(
      XLSX.write(workbook, { type: 'array', bookType: 'xlsx' }),
      'gNB',
      'now',
    )).toThrowError(expect.objectContaining({ code: 'invalid_row', field }));
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

  it.each([
    ['abc', 'Periodic Inform Interval'],
    ['86401', 'Periodic Inform Interval'],
  ])('validates imported values against selected product metadata: %s', (value, field) => {
    const worksheet = XLSX.utils.aoa_to_sheet([
      ['Serial Number', 'Periodic Inform Interval'],
      ['5G-SN-MODEL-001', value],
    ]);
    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, worksheet, 'DEVICE');
    const bytes = XLSX.write(workbook, { type: 'array', bookType: 'xlsx' }) as ArrayBuffer;

    expect(() => parseParamConfigWorkbook(bytes, 'gNB', 'now', {
      quickSettingsGroups: [{
        id: 'management', titleZh: '管理', titleEn: 'Management', multiInstance: false,
        params: [{
          name: 'PeriodicInformInterval', titleZh: '上报周期', titleEn: 'Interval',
          standardPath: 'Device.ManagementServer.PeriodicInformInterval',
        }],
      }],
      quickSettingFields: [{
        id: 'PeriodicInformInterval', name: ['sheetParameters', 'DEVICE', 0, field],
        labelKey: 'interval', range: '1 ~ 86400',
      }],
      paramMappings: [{
        id: 'interval', paramModelId: 'model',
        standardPath: 'Device.ManagementServer.PeriodicInformInterval',
        privatePath: 'Device.ManagementServer.PeriodicInformInterval',
        entryType: 'parameter', access: 'readWrite', dataType: 'unsignedInt',
        changeApplies: 'Immediate', minValue: '1', maxValue: '86400',
        isStorable: true, isActive: true,
      }],
    })).toThrowError(expect.objectContaining({ code: 'invalid_row', field }));
  });

  it('rejects a carrier bandwidth that is invalid for the imported SCS', () => {
    const worksheet = XLSX.utils.aoa_to_sheet([
      ['*Serial Number', 'SubcarrierSpacing(DL)', 'DLBandwidth'],
      ['5G-SN-BW-001', '0', '273'],
    ]);
    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, worksheet, 'CELL');
    const bytes = XLSX.write(workbook, { type: 'array', bookType: 'xlsx' }) as ArrayBuffer;

    expect(() => parseParamConfigWorkbook(bytes, 'gNB', 'now', {
      quickSettingsGroups: [{
        id: 'cell', titleZh: '小区', titleEn: 'Cell', multiInstance: true,
        params: [
          { name: 'DLSubCarrierSpacing', titleZh: '间隔', titleEn: 'SCS' },
          { name: 'DLCarrierBandWidth', titleZh: '带宽', titleEn: 'Bandwidth' },
        ],
      }],
      quickSettingFields: getParamConfigExportFields('gNB'),
    })).toThrowError(expect.objectContaining({ code: 'invalid_row', field: 'DLBandwidth' }));
  });

  it.each([
    ['A', 'string length'],
    ['abc-123', 'validation pattern'],
  ])('validates imported strings against product model %s', (value) => {
    const worksheet = XLSX.utils.aoa_to_sheet([
      ['Serial Number', 'SiteCode'],
      ['5G-SN-MODEL-002', value],
    ]);
    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, worksheet, 'DEVICE');
    const bytes = XLSX.write(workbook, { type: 'array', bookType: 'xlsx' }) as ArrayBuffer;

    expect(() => parseParamConfigWorkbook(bytes, 'gNB', 'now', {
      paramMappings: [{
        id: 'site-code', paramModelId: 'model',
        standardPath: 'Device.DeviceInfo.SiteCode', privatePath: 'Device.DeviceInfo.SiteCode',
        entryType: 'parameter', access: 'readWrite', dataType: 'string',
        changeApplies: 'Immediate', minValue: '2', maxValue: '6',
        validationPattern: '/^[A-Z]+$/', isStorable: true, isActive: true,
      }],
    })).toThrowError(expect.objectContaining({ code: 'invalid_row', field: 'SiteCode' }));
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
