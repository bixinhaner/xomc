import { describe, expect, it } from 'vitest';
import * as XLSX from 'xlsx';
import ExcelJS from 'exceljs';
import {
  createParamConfigWorkbook,
  createParamConfigTemplateWorkbook,
  enrichParamConfigWorkbook,
  mergeImportedParamConfigs,
  parseParamConfigWorkbook,
  PARAM_MAPPING_SHEET,
  PARAM_TEMPLATE_EXAMPLE_SERIAL,
} from './paramConfigWorkbook';
import { getParamConfigExportFields } from './paramConfigExportFields';

describe('parameter config workbook', () => {
  it('puts a serial-number column first on every generated business sheet', () => {
    const workbook = createParamConfigTemplateWorkbook('gNB', {
      deviceType: 'gNB',
      quickSettingsGroups: [{
        id: 'device', titleZh: '设备', titleEn: 'Device', multiInstance: false,
        params: [{
          name: 'URL', titleZh: '地址', titleEn: 'URL',
          standardPath: 'Device.ManagementServer.URL',
        }],
      }, {
        id: 'cell', titleZh: '小区', titleEn: 'Cell', multiInstance: false,
        params: [{
          name: 'PCI', titleZh: 'PCI', titleEn: 'PCI',
          standardPath: 'Device.Services.FAPService.1.CellConfig.1.NR.RAN.PCI',
        }],
      }, {
        id: 'core', titleZh: '核心网', titleEn: 'Core Network', multiInstance: false,
        params: [{
          name: 'PLMNID', titleZh: 'PLMN', titleEn: 'PLMN',
          standardPath: 'Device.Services.FAPService.1.CellConfig.1.NR.CN.PLMNID',
        }],
      }, {
        id: 'sync', titleZh: '同步', titleEn: 'Synchronization', multiInstance: false,
        params: [{
          name: 'PTPDomain', titleZh: 'PTP域', titleEn: 'PTP Domain',
          standardPath: 'Device.Time.PTPDomain',
        }],
      }, {
        id: 'ipsec', titleZh: 'IPSec', titleEn: 'IPSec', multiInstance: false,
        params: [{
          name: 'Enable', titleZh: '启用', titleEn: 'Enable',
          standardPath: 'Device.FAP.IPSec.1.Enable',
        }],
      }],
    });

    for (const sheetName of workbook.SheetNames) {
      const headers = XLSX.utils.sheet_to_json<unknown[]>(workbook.Sheets[sheetName], {
        header: 1,
        defval: '',
      })[0] as string[];
      expect(headers[0], sheetName).toBe('Serial Number');
    }
  });

  it('preserves imported columns without adding fixed-template planning fields', () => {
    const workbook = createParamConfigWorkbook([{
      deviceType: 'gNB',
      serialNumber: 'SN-001',
      sheetParameters: {
        PLMN: [{
          'Serial Number': 'SN-001',
          '*NCI': '1',
          '*TAC': '2',
          '*RANAC': '3',
          '*PLMN ID': '46001',
          '*PRIMARY': '1',
        }],
      },
      workbookMappings: [],
    }]);

    const rows = XLSX.utils.sheet_to_json<unknown[]>(workbook.Sheets.PLMN, {
      header: 1,
      defval: '',
    });
    expect(rows[0]).toEqual([
      'Serial Number', '*NCI', '*TAC', '*RANAC', '*PLMN ID', '*PRIMARY',
    ]);
    expect(rows[0]).not.toEqual(expect.arrayContaining([
      'SD', 'SD Value', 'AMF IP:DEFAULT', 'NguBindInterface',
    ]));
  });

  it('removes obsolete unmapped fields and 5G Duplex Mode from downloads', () => {
    const workbook = createParamConfigWorkbook([{
      deviceType: 'gNB',
      serialNumber: 'SN-001',
      sheetParameters: {
        CELL: [{
          'Serial Number': 'SN-001',
          PCI: 10,
          DLBandwidth: '106',
          'DL Carrier Bandwidth': '25',
          'Duplex Mode': 'TDD',
        }],
      },
      workbookMappings: [
        { displayName: 'PCI', sheet: 'CELL', header: 'PCI', trPath: 'Device.Radio.1.PCI', source: 'system' },
        { displayName: 'DL Carrier Bandwidth', sheet: 'CELL', header: 'DL Carrier Bandwidth', trPath: 'Device.Radio.1.DLBandwidth', source: 'system' },
        { displayName: 'Duplex Mode', sheet: 'CELL', header: 'Duplex Mode', trPath: 'Device.Ethernet.Interface.1.DuplexMode', source: 'system' },
      ],
    }]);

    const cellRows = XLSX.utils.sheet_to_json<unknown[]>(workbook.Sheets.CELL, { header: 1, defval: '' });
    expect(cellRows[0]).toEqual(['Serial Number', 'Cell Index', 'PCI', 'DL Carrier Bandwidth']);
    expect(cellRows[1]).toEqual(['SN-001', 1, 10, '25']);
    const mappingRows = XLSX.utils.sheet_to_json<Record<string, unknown>>(
      workbook.Sheets['参数映射'], { defval: '' },
    );
    expect(mappingRows.map((row) => row['参数列名'])).toEqual(['PCI', 'DL Carrier Bandwidth']);
  });

  it('rejects placeholder serial numbers instead of guessing their target device', () => {
    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, XLSX.utils.aoa_to_sheet([
      ['Serial Number', 'URL'],
      ['REAL-SN-001', 'http://acs.example.test'],
    ]), 'DEVICE');
    XLSX.utils.book_append_sheet(workbook, XLSX.utils.aoa_to_sheet([
      ['Serial Number', 'Sync Source', 'Domain'],
      [PARAM_TEMPLATE_EXAMPLE_SERIAL, 'GPS', '24'],
    ]), '1588_CONFIGURATION');
    XLSX.utils.book_append_sheet(workbook, XLSX.utils.aoa_to_sheet([
      ['页面显示名称', '数据工作表', '参数列名', 'TRPath', '来源', '说明'],
      ['URL', 'DEVICE', 'URL', 'Device.ManagementServer.URL', '系统预置', ''],
      ['同步源', '1588_CONFIGURATION', 'Sync Source', 'Device.Time.SyncSource', '系统预置', ''],
      ['域', '1588_CONFIGURATION', 'Domain', 'Device.Time.Domain', '系统预置', ''],
    ]), PARAM_MAPPING_SHEET);

    expect(() => parseParamConfigWorkbook(
      XLSX.write(workbook, { type: 'array', bookType: 'xlsx' }),
      'gNB',
      'now',
    )).toThrowError(expect.objectContaining({
      code: 'invalid_row',
      field: '1588_CONFIGURATION.Serial Number',
    }));
  });

  it('adds only the explicit serial-number column when an imported sheet row lacks it', () => {
    const downloaded = createParamConfigWorkbook([{
      deviceType: 'gNB',
      serialNumber: 'REAL-SN-001',
      sheetParameters: { DEVICE: [{ URL: 'http://acs.example.test' }] },
      workbookMappings: [],
    }]);
    const rows = XLSX.utils.sheet_to_json<unknown[]>(downloaded.Sheets.DEVICE, {
      header: 1,
      defval: '',
    });
    expect(rows).toEqual([
      ['Serial Number', 'URL'],
      ['REAL-SN-001', 'http://acs.example.test'],
    ]);
  });

  it.each(['Serial Number', '*Serial Number', '*SERIAL_NUMBER'])(
    'keeps the %s alias as the first exported column',
    (serialHeader) => {
      const downloaded = createParamConfigWorkbook([{
        deviceType: 'gNB',
        serialNumber: 'REAL-SN-001',
        sheetParameters: {
          DEVICE: [{ URL: 'http://acs.example.test', [serialHeader]: 'REAL-SN-001' }],
        },
        workbookMappings: [{
          displayName: 'ACS URL',
          sheet: 'DEVICE',
          header: 'URL',
          trPath: 'Device.ManagementServer.URL',
          source: 'system',
        }],
      }]);
      const rows = XLSX.utils.sheet_to_json<unknown[]>(downloaded.Sheets.DEVICE, {
        header: 1,
        defval: '',
      });
      expect(rows[0]).toEqual([serialHeader, 'URL']);
      expect(rows[1]).toEqual(['REAL-SN-001', 'http://acs.example.test']);
    },
  );

  it('preserves imported column order and aliases even when they share a TRPath', () => {
    const workbook = createParamConfigWorkbook([{
      deviceType: 'gNB',
      serialNumber: 'SN-001',
      sheetParameters: {
        CELL: [{
          '*gNB Lenth': '',
          'Custom B': 'b',
          'gNB ID Length': 24,
          'Custom A': 'a',
          '*Serial Number': 'SN-001',
        }],
      },
      workbookMappings: [
        { displayName: '标识长度', sheet: 'CELL', header: '*gNB Lenth', trPath: 'Device.Cell.GNBIDLength', source: 'system' },
        { displayName: '标识长度', sheet: 'CELL', header: 'gNB ID Length', trPath: 'Device.Cell.GNBIDLength', source: 'system' },
        { displayName: '参数 A', sheet: 'CELL', header: 'Custom A', trPath: 'Device.Cell.Related.A', source: 'custom' },
        { displayName: '参数 B', sheet: 'CELL', header: 'Custom B', trPath: 'Device.Cell.Related.B', source: 'custom' },
        { displayName: '未映射 A', sheet: 'CELL', header: 'Unmapped A', trPath: '', source: 'custom' },
        { displayName: '未映射 B', sheet: 'CELL', header: 'Unmapped B', trPath: '', source: 'custom' },
      ],
    }]);

    const rows = XLSX.utils.sheet_to_json<unknown[]>(workbook.Sheets.CELL, { header: 1, defval: '' });
    const headers = rows[0] as string[];
    expect(headers).toEqual([
      '*Serial Number', 'Cell Index', '*gNB Lenth', 'Custom B', 'gNB ID Length', 'Custom A',
    ]);
    expect(rows[1]).toEqual(['SN-001', 1, '', 'b', 24, 'a']);
    const mappingRows = XLSX.utils.sheet_to_json<Record<string, string>>(
      workbook.Sheets[PARAM_MAPPING_SHEET],
      { defval: '' },
    );
    expect(mappingRows.filter((row) => row.TRPath === '').map((row) => row['参数列名']))
      .toEqual(['Unmapped A', 'Unmapped B']);
  });

  it('creates the GSM template from its current public parameters', () => {
    const workbook = createParamConfigTemplateWorkbook('GSM', {
      deviceType: 'GSM',
      quickSettingsGroups: [{
        id: 'gsm-radio', titleZh: 'GSM 无线参数', titleEn: 'GSM Radio', multiInstance: false,
        params: [
          { name: 'Band', titleZh: '频段', titleEn: 'Band', standardPath: 'Device.GSM.Radio.Band' },
          { name: 'Bsic', titleZh: 'BSIC', titleEn: 'Bsic', standardPath: 'Device.GSM.Radio.Bsic' },
        ],
      }],
    });

    expect(XLSX.utils.sheet_to_json<unknown[]>(workbook.Sheets.GSM, {
      header: 1,
      defval: '',
    })[0]).toEqual([
      'Serial Number', 'Cell Index', 'Band', 'Bsic',
    ]);
  });

  it('uses a BTS index for BSC workbooks', () => {
    const workbook = createParamConfigTemplateWorkbook('GSM', {
      deviceType: 'GSM',
      productClass: 'BSC',
      quickSettingsGroups: [{
        id: 'bsc-bts-config', titleZh: 'BTS', titleEn: 'BTS', multiInstance: false,
        params: [{
          name: 'IpaRslIp', titleZh: 'RSL IP', titleEn: 'Remote IP',
          standardPath: 'DeviceGSM.Bts.{i}.IpaRslIp',
        }],
      }],
    });

    expect(XLSX.utils.sheet_to_json<unknown[]>(workbook.Sheets.GSM, {
      header: 1,
      defval: '',
    })[0]).toEqual(['Serial Number', 'BTS Index', 'Remote IP']);
  });

  it('uses the BSC parameter-model identity when the literal product class is PGSM', () => {
    const workbook = createParamConfigTemplateWorkbook('GSM', {
      deviceType: 'GSM',
      productClass: 'FAP/PGSM BSC BSC',
      quickSettingsGroups: [{
        id: 'bsc-bts-config', titleZh: 'BTS', titleEn: 'BTS', multiInstance: false,
        params: [{
          name: 'IpaRslIp', titleZh: 'RSL IP', titleEn: 'Remote IP',
          standardPath: 'DeviceGSM.Bts.{i}.IpaRslIp',
        }],
      }],
    });

    expect(XLSX.utils.sheet_to_json<unknown[]>(workbook.Sheets.GSM, {
      header: 1,
      defval: '',
    })[0]).toEqual(['Serial Number', 'BTS Index', 'Remote IP']);
  });

  it('adds a BTS index when exporting a legacy single-row BSC config', () => {
    const workbook = createParamConfigWorkbook([{
      deviceType: 'GSM',
      serialNumber: 'BSC-001',
      sheetParameters: { GSM: [{ 'Serial Number': 'BSC-001', IPA: '6969' }] },
    }], { productClass: 'BSC' });

    expect(XLSX.utils.sheet_to_json<unknown[]>(workbook.Sheets.GSM, {
      header: 1,
      defval: '',
    })).toEqual([
      ['Serial Number', 'BTS Index', 'IPA'],
      ['BSC-001', 1, '6969'],
    ]);
  });

  it('rejects a primary instance column that does not match the GSM product', () => {
    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, XLSX.utils.aoa_to_sheet([
      ['Serial Number', 'Cell Index', 'IPA'],
      ['BSC-001', 1, '6969'],
    ]), 'GSM');
    XLSX.utils.book_append_sheet(workbook, XLSX.utils.aoa_to_sheet([
      ['页面显示名称', '数据工作表', '参数列名', 'TRPath', '来源', '说明'],
      ['IPA', 'GSM', 'IPA', 'DeviceGSM.Bts.{i}.IPA', '系统预置', ''],
    ]), PARAM_MAPPING_SHEET);

    expect(() => parseParamConfigWorkbook(
      XLSX.write(workbook, { type: 'array', bookType: 'xlsx' }),
      'GSM',
      'now',
      { productClass: 'BSC' },
    )).toThrowError(expect.objectContaining({ code: 'invalid_row', field: 'GSM.BTS Index' }));
  });

  it('imports explicit cell indexes independently of row order', () => {
    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, XLSX.utils.aoa_to_sheet([
      ['Serial Number', 'Cell Index', 'PCI'],
      ['NR-MULTI-1', 2, 202],
      ['NR-MULTI-1', 1, 101],
    ]), 'CELL');
    XLSX.utils.book_append_sheet(workbook, XLSX.utils.aoa_to_sheet([
      ['页面显示名称', '数据工作表', '参数列名', 'TRPath', '来源', '说明'],
      ['PCI', 'CELL', 'PCI', 'Device.Services.FAPService.1.CellConfig.{i}.NR.RAN.RF.PhyCellID', '系统预置', ''],
    ]), PARAM_MAPPING_SHEET);

    const [parsed] = parseParamConfigWorkbook(
      XLSX.write(workbook, { type: 'array', bookType: 'xlsx' }),
      'gNB',
      'now',
    );
    expect(parsed.sheetParameters?.CELL).toEqual([
      expect.objectContaining({ 'Cell Index': 2, PCI: 202 }),
      expect.objectContaining({ 'Cell Index': 1, PCI: 101 }),
    ]);
  });

  it('rejects ambiguous or duplicate primary instance rows', () => {
    const parseRows = (rows: unknown[][]) => {
      const workbook = XLSX.utils.book_new();
      XLSX.utils.book_append_sheet(workbook, XLSX.utils.aoa_to_sheet([
        ['Serial Number', 'Cell Index', 'PCI'],
        ...rows,
      ]), 'CELL');
      XLSX.utils.book_append_sheet(workbook, XLSX.utils.aoa_to_sheet([
        ['页面显示名称', '数据工作表', '参数列名', 'TRPath', '来源', '说明'],
        ['PCI', 'CELL', 'PCI', 'Device.Cell.{i}.PCI', '系统预置', ''],
      ]), PARAM_MAPPING_SHEET);
      return parseParamConfigWorkbook(
        XLSX.write(workbook, { type: 'array', bookType: 'xlsx' }),
        'gNB',
        'now',
      );
    };

    expect(() => parseRows([
      ['NR-MULTI-1', '', 1],
      ['NR-MULTI-1', '', 2],
    ])).toThrowError(expect.objectContaining({ code: 'invalid_row', field: 'CELL.Cell Index' }));
    expect(() => parseRows([
      ['NR-MULTI-1', 1, 1],
      ['NR-MULTI-1', 1, 2],
    ])).toThrowError(expect.objectContaining({ code: 'invalid_row', field: 'CELL.Cell Index' }));
  });

  it('adds hover notes and dropdowns without occupying a data row', async () => {
    const metadata = {
      deviceType: 'gNB' as const,
      quickSettingsGroups: [{
        id: 'device', titleZh: '设备', titleEn: 'Device', multiInstance: false,
        params: [{
          name: 'Enabled', titleZh: '启用', titleEn: 'Enabled', type: 'bool',
          standardPath: 'Device.Example.Enabled',
        }],
      }],
    };
    const workbook = await enrichParamConfigWorkbook(
      createParamConfigTemplateWorkbook('gNB', metadata), metadata,
    );
    const device = workbook.getWorksheet('DEVICE')!;

    expect(device.getCell('A1').note).toContain('参数类型：string');
    expect(device.getCell('B1').note).toContain('参数类型：bool');
    expect(device.getCell('B2').dataValidation.formulae).toEqual(['"true,false"']);
    expect(device.getRow(2).values).not.toContain('参数类型');
  });

  it('uses quick-setting constraints first and parameter-model constraints as fallback', async () => {
    const metadata = {
      deviceType: 'gNB' as const,
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
      }],
    };
    const source = createParamConfigTemplateWorkbook('gNB', metadata);
    const workbook = await enrichParamConfigWorkbook(source, metadata);
    const cell = workbook.getWorksheet('CELL')!;
    const pciColumn = cell.getRow(1).values.indexOf('PCI');

    expect(cell.getCell(1, pciColumn).note).toContain('参数类型：int');
    expect(cell.getCell(1, pciColumn).note).toContain('取值范围：0 ~ 1007');
    expect(cell.getCell(1, pciColumn).note).toContain('显示条件：射频参数启用');
    expect(cell.getCell(2, pciColumn).dataValidation.formulae).toEqual(['"10,20"']);
  });

  it('appends a final parameter mapping sheet with page labels and TRPaths', async () => {
    const metadata = {
      deviceType: 'gNB',
      quickSettingsGroups: [{
        id: 'cell', titleZh: '小区参数', titleEn: 'Cell', multiInstance: false,
        params: [{
          name: 'PCI', titleZh: '物理小区标识', titleEn: 'PCI',
          standardPath: 'Device.Cell.{i}.PCI',
        }],
      }],
    } as const;
    const workbook = await enrichParamConfigWorkbook(
      createParamConfigTemplateWorkbook('gNB', metadata), metadata,
    );
    const mapping = workbook.getWorksheet(PARAM_MAPPING_SHEET)!;
    expect(mapping.getRow(1).values).toEqual([
      undefined, '页面显示名称', '数据工作表', '参数列名', 'TRPath', '来源', '说明',
    ]);
    expect(mapping.getRow(2).values).toEqual(expect.arrayContaining([
      'PCI', 'CELL', 'PCI', 'Device.Cell.1.PCI', '系统预置',
    ]));
    const visibleSheets = workbook.worksheets.filter((sheet) => sheet.state === 'visible');
    expect(visibleSheets.at(-1)?.name).toBe(PARAM_MAPPING_SHEET);
  });

  it('generates only sheets represented by current public parameters', async () => {
    const metadata = {
      deviceType: 'gNB' as const,
      quickSettingsGroups: [{
        id: 'gnb-network-interface', titleZh: '接口', titleEn: 'Interface', multiInstance: true,
        maxInstances: 16, objectPath: 'Device.Ethernet.Interface.{i}.',
        params: [
          { name: 'Name', titleZh: '接口名称', titleEn: 'Name', leaf: 'Name' },
          { name: 'IPv4Address', titleZh: 'IPv4地址', titleEn: 'IPv4 Address', leaf: 'IPv4Address' },
        ],
      }],
    };
    const source = createParamConfigTemplateWorkbook('gNB', metadata);
    expect(source.SheetNames).toEqual(['INTERFACE']);
    expect(XLSX.utils.sheet_to_json<unknown[]>(source.Sheets.INTERFACE, { header: 1, defval: '' })[0])
      .toEqual([
        'Serial Number',
        'Name [1]', 'IPv4 Address [1]',
        'Name [2]', 'IPv4 Address [2]',
      ]);

    const workbook = await enrichParamConfigWorkbook(source, metadata);
    const mapping = workbook.getWorksheet(PARAM_MAPPING_SHEET)!;
    expect(mapping.getColumn(4).values).toEqual(expect.arrayContaining([
      'Device.Ethernet.Interface.1.Name', 'Device.Ethernet.Interface.2.Name',
      'Device.Ethernet.Interface.1.IPv4Address', 'Device.Ethernet.Interface.2.IPv4Address',
    ]));
    expect(workbook.worksheets.filter((sheet) => sheet.state === 'visible').at(-1)?.name)
      .toBe(PARAM_MAPPING_SHEET);
  });

  it('uses one IPSEC tunnel and routes gNB identity and PLMN parameters to their sheets', async () => {
    const metadata = {
      deviceType: 'gNB' as const,
      quickSettingsGroups: [{
        id: 'gnb-cell', titleZh: '小区', titleEn: 'Cell', multiInstance: true,
        objectPath: 'Device.Services.FAPService.{i}.CellConfig.NR.RAN.{j}.',
        params: [
          { name: 'GNBName', titleZh: '基站名称', titleEn: 'gNB Name', leaf: 'GNBName' },
          { name: 'GNBID', titleZh: '基站标识', titleEn: 'gNB ID', leaf: 'GNBID' },
          { name: 'GNBIDLength', titleZh: '标识长度', titleEn: 'gNB ID Length', leaf: 'GNBIDLength' },
          { name: 'PLMNID', titleZh: '运营商', titleEn: 'PLMN ID', leaf: 'PLMNID' },
        ],
      }, {
        id: 'gnb-ipsec', titleZh: '隧道', titleEn: 'IPSec Tunnel', multiInstance: true,
        objectPath: 'Device.FAP.IPSec.Tunnel.{i}.',
        params: [{ name: 'Enable', titleZh: '启用', titleEn: 'Enable', leaf: 'Enable' }],
      }],
    };
    const source = createParamConfigTemplateWorkbook('gNB', metadata);
    expect(XLSX.utils.sheet_to_json<unknown[]>(source.Sheets.DEVICE, { header: 1, defval: '' })[0])
      .toEqual(['Serial Number', 'gNB Name', 'gNB ID', 'gNB ID Length']);
    expect(XLSX.utils.sheet_to_json<unknown[]>(source.Sheets.PLMN, { header: 1, defval: '' })[0])
      .toEqual(['Serial Number', 'PLMN ID']);
    expect(XLSX.utils.sheet_to_json<unknown[]>(source.Sheets.IPSEC, { header: 1, defval: '' })[0])
      .toEqual(['Serial Number', 'Enable']);
  });

  it('keeps VLAN and IPv6 out of generated network templates', async () => {
    const metadata = {
      deviceType: 'gNB' as const,
      quickSettingsGroups: [{
        id: 'gnb-network-interface', titleZh: '网络接口', titleEn: 'Network Interface', multiInstance: false,
        objectPath: 'Device.Ethernet.Interface.1.',
        params: [
          { name: 'IPv4Address', titleZh: 'IPv4地址', titleEn: 'IPv4 Address', leaf: 'IPv4Address' },
          { name: 'IPv6Address', titleZh: 'IPv6地址', titleEn: 'IPv6 Address', leaf: 'IPv6Address' },
          { name: 'VLANID', titleZh: 'VLAN ID', titleEn: 'VLAN ID', leaf: 'VLANID' },
        ],
      }],
    };
    const source = createParamConfigTemplateWorkbook('gNB', metadata);
    expect(XLSX.utils.sheet_to_json<unknown[]>(source.Sheets.INTERFACE, { header: 1, defval: '' })[0])
      .toEqual(['Serial Number', 'IPv4 Address']);
    const workbook = await enrichParamConfigWorkbook(source, metadata);
    const mappingPaths = workbook.getWorksheet(PARAM_MAPPING_SHEET)?.getColumn(4).values;
    expect(mappingPaths).toEqual(expect.arrayContaining(['Device.Ethernet.Interface.1.IPv4Address']));
    expect(mappingPaths).not.toEqual(expect.arrayContaining([
      'Device.Ethernet.Interface.1.IPv6Address',
      'Device.Ethernet.Interface.1.VLANID',
    ]));
  });

  it('filters DSCP and static routes and limits repeated parameter groups', async () => {
    const metadata = {
      deviceType: 'gNB' as const,
      quickSettingsGroups: [{
        id: 'dscp', titleZh: 'DSCP', titleEn: 'DSCP', multiInstance: false,
        params: [{
          name: 'DSCPMark', titleZh: 'DSCP标记', titleEn: 'DSCP Mark',
          standardPath: 'Device.QoS.DSCPMark',
        }],
      }, {
        id: 'static-route', titleZh: '静态路由', titleEn: 'Static Route', multiInstance: false,
        params: [{
          name: 'Destination', titleZh: '目的地址', titleEn: 'Destination',
          standardPath: 'Device.Routing.Router.1.IPv4Forwarding.1.DestIPAddress',
        }],
      }, ...[1, 2, 3].map((instance) => ({
        id: `management-${instance}`,
        titleZh: `管理地址${instance}`,
        titleEn: `Management ${instance}`,
        multiInstance: false,
        params: [{
          name: 'URL', titleZh: '地址', titleEn: 'URL',
          standardPath: `Device.ManagementServer.${instance}.URL`,
        }],
      }))],
    };
    const source = createParamConfigTemplateWorkbook('gNB', metadata);
    const headers = source.SheetNames.flatMap((sheetName) => (
      XLSX.utils.sheet_to_json<unknown[]>(source.Sheets[sheetName], { header: 1, defval: '' })[0] as string[]
    ));
    expect(headers).toEqual(expect.arrayContaining(['URL [1]', 'URL [2]']));
    expect(headers).not.toEqual(expect.arrayContaining([
      'DSCP Mark', 'Destination', 'URL [3]',
    ]));

    const workbook = await enrichParamConfigWorkbook(source, metadata);
    const paths = workbook.getWorksheet(PARAM_MAPPING_SHEET)?.getColumn(4).values.map(String);
    expect(paths).toEqual(expect.arrayContaining([
      'Device.ManagementServer.1.URL', 'Device.ManagementServer.2.URL',
    ]));
    expect(paths).not.toEqual(expect.arrayContaining([
      'Device.QoS.DSCPMark',
      'Device.Routing.Router.1.IPv4Forwarding.1.DestIPAddress',
      'Device.ManagementServer.3.URL',
    ]));
  });

  it('keeps 4G neighbor parameters out of generated templates and mappings', async () => {
    const metadata = {
      deviceType: 'eNB' as const,
      quickSettingsGroups: [{
        id: 'enb-cell', titleZh: '小区设置', titleEn: 'Cell Settings', multiInstance: false,
        params: [{
          name: 'CellIdentity', titleZh: '小区标识', titleEn: 'Cell Identity',
          standardPath: 'Device.Services.FAPService.1.CellConfig.LTE.RAN.CellIdentity',
        }],
      }, {
        id: 'enb-neighbor-cell', titleZh: '邻区列表', titleEn: 'Neighbor Cells', multiInstance: true,
        objectPath: 'Device.Services.FAPService.1.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.',
        params: [{ name: 'PCI', titleZh: '物理小区标识', titleEn: 'PCI', leaf: 'PCI' }],
      }],
    };
    const source = createParamConfigTemplateWorkbook('eNB', metadata);
    const workbook = await enrichParamConfigWorkbook(source, metadata);
    const allHeaders = workbook.worksheets
      .filter((sheet) => sheet.name !== PARAM_MAPPING_SHEET)
      .flatMap((sheet) => sheet.getRow(1).values.map(String));
    const mappingPaths = workbook.getWorksheet(PARAM_MAPPING_SHEET)?.getColumn(4).values.map(String);

    expect(allHeaders).toContain('Cell Identity');
    expect(allHeaders).not.toContain('PCI');
    expect(mappingPaths).toEqual(expect.arrayContaining([
      'Device.Services.FAPService.1.CellConfig.LTE.RAN.CellIdentity',
    ]));
    expect(mappingPaths?.some((path) => path.includes('NeighborList'))).toBe(false);
  });

  it('keeps page enum selects as Excel dropdowns when the English label differs from the parameter name', async () => {
    const metadata = {
      deviceType: 'gNB' as const,
      quickSettingsGroups: [{
        id: 'device-network', titleZh: '网络', titleEn: 'Network', multiInstance: false,
        params: [{
          name: 'IPMode', titleZh: '地址类型', titleEn: 'Address Type',
          standardPath: 'Device.IP.Interface.1.IPMode', type: 'enum',
          enumOptions: [
            { value: 'DHCP', label: 'DHCP' },
            { value: 'Static', label: 'Static' },
          ],
        }],
      }],
    };
    const workbook = await enrichParamConfigWorkbook(
      createParamConfigTemplateWorkbook('gNB', metadata),
      metadata,
    );
    const sheet = workbook.getWorksheet('INTERFACE')!;
    const column = sheet.getRow(1).values.indexOf('Address Type');
    expect(sheet.getCell(2, column).dataValidation).toMatchObject({
      type: 'list',
      formulae: ['"DHCP,Static"'],
    });
  });

  it('moves 1588 and synchronization settings into a dedicated sheet', async () => {
    const metadata = {
      deviceType: 'gNB' as const,
      quickSettingsGroups: [{
        id: 'gnb-sync-source', titleZh: '同步设置', titleEn: 'Synchronization Settings', multiInstance: false,
        params: [
          { name: 'SyncSource', titleZh: '同步源', titleEn: 'Sync Source', standardPath: 'Device.Time.SyncSource' },
          { name: 'PTPDomain', titleZh: 'PTP域', titleEn: 'PTP Domain', standardPath: 'Device.Time.PTPDomain' },
        ],
      }],
    };
    const source = createParamConfigTemplateWorkbook('gNB', metadata);
    expect(source.SheetNames).toEqual(['1588_CONFIGURATION']);
    expect(XLSX.utils.sheet_to_json<unknown[]>(source.Sheets['1588_CONFIGURATION'], { header: 1, defval: '' })[0])
      .toEqual(['Serial Number', 'Sync Source', 'PTP Domain']);
    const workbook = await enrichParamConfigWorkbook(source, metadata);
    expect(workbook.worksheets.filter((sheet) => sheet.state === 'visible').at(-1)?.name)
      .toBe(PARAM_MAPPING_SHEET);
  });

  it('moves only 4G synchronization and 1588 parameters to the 1588 sheet', () => {
    const metadata = {
      deviceType: 'eNB' as const,
      quickSettingsGroups: [{
        id: 'device-time', titleZh: '时间设置', titleEn: 'Time Settings', multiInstance: false,
        params: [
          { name: 'NTPServer1', titleZh: 'NTP服务器', titleEn: 'NTP Server 1', standardPath: 'Device.Time.NTPServer1' },
          { name: 'SyncSource', titleZh: '同步源', titleEn: 'Sync Source', standardPath: 'Device.Time.SyncSource' },
          { name: 'PTPDomain', titleZh: 'PTP域', titleEn: 'PTP Domain', standardPath: 'Device.Time.PTPDomain' },
        ],
      }],
    };
    const source = createParamConfigTemplateWorkbook('eNB', metadata);
    expect(XLSX.utils.sheet_to_json<unknown[]>(source.Sheets['1588_CONFIGURATION'], { header: 1, defval: '' })[0])
      .toEqual(['Serial Number', 'Sync Source', 'PTP Domain']);
    expect(XLSX.utils.sheet_to_json<unknown[]>(source.Sheets.NETWORK, { header: 1, defval: '' })[0])
      .toEqual(['Serial Number', 'NTP Server 1']);
  });

  it('uses only explicitly configured defaults in the template example row', async () => {
    const metadata = {
      deviceType: 'gNB' as const,
      quickSettingsGroups: [{
        id: 'device-example', titleZh: '示例', titleEn: 'Example', multiInstance: false,
        params: [
          {
            name: 'Mode', titleZh: '模式', titleEn: 'Mode', type: 'enum',
            standardPath: 'Device.Example.Mode',
            enumOptions: [{ value: 'Auto', label: 'Auto' }, { value: 'Manual', label: 'Manual' }],
          },
          {
            name: 'Priority', titleZh: '优先级', titleEn: 'Priority', type: 'int', minValue: 1,
            standardPath: 'Device.Example.Priority',
          },
          {
            name: 'Endpoint', titleZh: '端点', titleEn: 'Endpoint', defaultValue: '192.0.2.1',
            standardPath: 'Device.Example.Endpoint',
          },
        ],
      }],
    };
    const source = createParamConfigTemplateWorkbook('gNB', metadata);
    expect(XLSX.utils.sheet_to_json<unknown[]>(source.Sheets.DEVICE, { header: 1, defval: '' })[1])
      .toEqual([PARAM_TEMPLATE_EXAMPLE_SERIAL, '', '', '192.0.2.1']);

    const enriched = await enrichParamConfigWorkbook(source, metadata);
    const parsed = parseParamConfigWorkbook(
      await enriched.xlsx.writeBuffer(),
      'gNB',
      'now',
      metadata,
    );
    expect(parsed[0]).toMatchObject({
      serialNumber: PARAM_TEMPLATE_EXAMPLE_SERIAL,
      sheetParameters: {
        DEVICE: [{ Mode: '', Priority: '', Endpoint: '192.0.2.1' }],
      },
    });
  });

  it('keeps parameters blank when only enum and range constraints are configured', async () => {
    const metadata = {
      deviceType: 'eNB' as const,
      quickSettingsGroups: [{
        id: 'enb-cell', titleZh: '小区参数', titleEn: 'Cell Parameters', multiInstance: false,
        params: [{
          name: 'DLBandWidth', titleZh: '带宽', titleEn: 'Bandwidth', type: 'enum',
          standardPath: 'Device.Cell.DLBandwidth',
          enumOptions: [{ value: 'n25', label: '5MHz' }, { value: 'n50', label: '10MHz' }],
        }, {
          name: 'CellReselectPenaltyTime', titleZh: '小区重选惩罚时间', titleEn: 'CellReselectPenaltyTime', type: 'enum',
          standardPath: 'Device.Cell.CellReselectPenaltyTime',
          enumOptions: [{ value: '0', label: '0' }, { value: '20', label: '20' }],
        }, {
          name: 'handover2.min.rxlev', titleZh: '最低 RXLEV', titleEn: 'handover2.min.rxlev',
          standardPath: 'Device.Cell.handover2.min.rxlev',
        }],
      }],
      paramMappings: [{
        id: 'bandwidth', paramModelId: 'model', standardPath: 'Device.Cell.DLBandwidth',
        privatePath: 'Device.Cell.DLBandwidth', entryType: 'parameter', access: 'readWrite',
        dataType: 'INT', changeApplies: 'Immediate', isStorable: true, isActive: true,
      }, {
        id: 'penalty', paramModelId: 'model', standardPath: 'Device.Cell.CellReselectPenaltyTime',
        privatePath: 'Device.Cell.CellReselectPenaltyTime', entryType: 'parameter', access: 'readWrite',
        dataType: 'U_INT', changeApplies: 'Immediate', minValue: '20', maxValue: '620',
        isStorable: true, isActive: true,
      }, {
        id: 'other-rxlev', paramModelId: 'model', standardPath: 'Device.Other.rxlev',
        privatePath: 'Device.Other.rxlev', entryType: 'parameter', access: 'readWrite',
        dataType: 'U_INT', changeApplies: 'Immediate', minValue: '0', maxValue: '100',
        isStorable: true, isActive: true,
      }, {
        id: 'minimum-rxlev', paramModelId: 'model', standardPath: 'Device.Cell.handover2.min.rxlev',
        privatePath: 'Device.Cell.handover2.min.rxlev', entryType: 'parameter', access: 'readWrite',
        dataType: 'U_INT', changeApplies: 'Immediate', minValue: '-110', maxValue: '-50',
        isStorable: true, isActive: true,
      }],
    };
    const downloaded = await enrichParamConfigWorkbook(
      createParamConfigTemplateWorkbook('eNB', metadata),
      metadata,
    );

    const parsed = parseParamConfigWorkbook(
      await downloaded.xlsx.writeBuffer(),
      'eNB',
      'now',
      metadata,
    );

    expect(parsed[0].sheetParameters.CELL).toEqual([
      expect.objectContaining({
        Bandwidth: '',
        CellReselectPenaltyTime: '',
        'handover2.min.rxlev': '',
      }),
    ]);
  });

  it('uses n-prefixed LTE bandwidth values for BLN, BLQ, MLN and MLQ workbook dropdowns', async () => {
    const metadata = {
      deviceType: 'eNB' as const,
      quickSettingsGroups: [{
        id: 'enb-cell', titleZh: '小区参数', titleEn: 'Cell Parameters', multiInstance: false,
        params: [
          {
            name: 'DLBandWidth', titleZh: '下行带宽', titleEn: 'BLN/BLQ/MLQ DL Bandwidth', type: 'enum',
            standardPath: 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.DLBandwidth',
            enumOptions: [{ value: '25', label: '5MHz' }, { value: '50', label: '10MHz' }],
          },
          {
            name: 'Bandwidth', titleZh: '下行带宽', titleEn: 'MLN DL Bandwidth', type: 'enum',
            standardPath: 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.DLBandwidth',
            enumOptions: [{ value: '25', label: '5MHz' }, { value: '50', label: '10MHz' }],
          },
          {
            name: 'ULBandWidth', titleZh: '上行带宽', titleEn: 'BLN/BLQ/MLN/MLQ UL Bandwidth', type: 'enum',
            standardPath: 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.ULBandwidth',
            enumOptions: [{ value: '25', label: '5MHz' }, { value: '50', label: '10MHz' }],
          },
        ],
      }],
      quickSettingFields: getParamConfigExportFields('eNB'),
    };
    const workbook = await enrichParamConfigWorkbook(
      createParamConfigTemplateWorkbook('eNB', metadata),
      metadata,
    );
    const cell = workbook.getWorksheet('CELL')!;
    const headers = cell.getRow(1).values as string[];
    for (const header of ['BLN/BLQ/MLQ DL Bandwidth', 'MLN DL Bandwidth', 'BLN/BLQ/MLN/MLQ UL Bandwidth']) {
      const column = headers.indexOf(header);
      const formula = String(cell.getCell(2, column).dataValidation.formulae?.[0] ?? '');
      const options = formula.replace(/^"|"$/g, '').split(',');
      expect(options).toContain('n50');
      expect(options.every((option) => option.startsWith('n'))).toBe(true);
    }

    const roundTrip = await workbook.xlsx.writeBuffer();
    const edited = XLSX.read(roundTrip, { type: 'array' });
    const worksheet = edited.Sheets.CELL;
    for (const [header, value] of [
      ['Serial Number', 'SN-BW-001'],
      ['BLN/BLQ/MLQ DL Bandwidth', 'n50'],
      ['MLN DL Bandwidth', 'n50'],
      ['BLN/BLQ/MLN/MLQ UL Bandwidth', 'n50'],
    ] as const) {
      worksheet[XLSX.utils.encode_cell({ r: 1, c: headers.indexOf(header) - 1 })] = {
        t: 's',
        v: value,
      };
    }

    expect(parseParamConfigWorkbook(
      XLSX.write(edited, { type: 'array', bookType: 'xlsx' }),
      'eNB',
      'now',
      metadata,
    )[0].sheetParameters?.CELL?.[0]).toMatchObject({
      'BLN/BLQ/MLQ DL Bandwidth': 'n50',
      'MLN DL Bandwidth': 'n50',
      'BLN/BLQ/MLN/MLQ UL Bandwidth': 'n50',
    });
  });

  it('rejects template generation when the product has no public parameters', () => {
    expect(() => createParamConfigTemplateWorkbook('gNB', {
      deviceType: 'gNB',
      quickSettingsGroups: [],
    })).toThrowError(expect.objectContaining({ code: 'empty_workbook' }));
  });

  it('keeps only one copy of repeated cell parameters', async () => {
    const metadata = {
      deviceType: 'gNB' as const,
      quickSettingsGroups: [{
        id: 'serving-cell', titleZh: '服务小区', titleEn: 'Serving Cell', multiInstance: false,
        params: [{ name: 'ServingPCI', titleZh: 'PCI', titleEn: 'PCI', standardPath: 'Device.Cell.1.PCI' }],
      }, {
        id: 'neighbor-cell', titleZh: '邻区', titleEn: 'Neighbor Cell', multiInstance: false,
        params: [{ name: 'NeighborPCI', titleZh: 'PCI', titleEn: 'PCI', standardPath: 'Device.Neighbor.1.PCI' }],
      }],
    };
    const source = createParamConfigTemplateWorkbook('gNB', metadata);
    expect(XLSX.utils.sheet_to_json<unknown[]>(source.Sheets.CELL, { header: 1, defval: '' })[0])
      .toEqual(['Serial Number', 'Cell Index', 'PCI']);

    const downloaded = await enrichParamConfigWorkbook(source, metadata);
    expect(downloaded.getWorksheet(PARAM_MAPPING_SHEET)?.getColumn(4).values)
      .toEqual(expect.arrayContaining(['Device.Cell.1.PCI']));
    expect(downloaded.getWorksheet(PARAM_MAPPING_SHEET)?.getColumn(4).values)
      .not.toEqual(expect.arrayContaining(['Device.Neighbor.1.PCI']));
    expect(parseParamConfigWorkbook(
      await downloaded.xlsx.writeBuffer(), 'gNB', 'now', metadata,
    )[0].serialNumber).toBe(PARAM_TEMPLATE_EXAMPLE_SERIAL);
  });

  it('uses English parameter names and keeps cell-level fields on cell instance one', async () => {
    const metadata = {
      deviceType: 'eNB' as const,
      quickSettingsGroups: [{
        id: 'enb-cell', titleZh: '小区参数', titleEn: 'Cell Parameters', multiInstance: true,
        objectPath: 'Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.{i}.',
        params: [{
          name: 'CellType', titleZh: '小区类型', titleEn: 'Cell Type', leaf: 'CellType', type: 'enum',
          enumOptions: [{ value: '0', label: 'Macro' }, { value: '1', label: 'Home' }],
        }, {
          name: 'PowerClass', titleZh: '功率等级', titleEn: 'Power Class', leaf: 'PowerClass',
        }],
      }],
    };
    const source = createParamConfigTemplateWorkbook('eNB', metadata);
    expect(XLSX.utils.sheet_to_json<unknown[]>(source.Sheets.CELL, { header: 1, defval: '' })[0])
      .toEqual(['Serial Number', '*CELL_NUMBER', 'Cell Type', 'Power Class']);

    const workbook = await enrichParamConfigWorkbook(source, metadata);
    const cell = workbook.getWorksheet('CELL')!;
    const cellTypeColumn = cell.getRow(1).values.indexOf('Cell Type');
    expect(cell.getCell(2, cellTypeColumn).dataValidation.formulae).toEqual(['"0,1"']);
    expect(workbook.getWorksheet(PARAM_MAPPING_SHEET)?.getColumn(4).values).toEqual(expect.arrayContaining([
      'Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.1.CellType',
      'Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.1.PowerClass',
    ]));
  });

  it('fixes BNQ 5G FAPService and every CellConfig child parameter to instance one', async () => {
    const metadata = {
      deviceType: 'gNB' as const,
      productClass: 'BNQ-5G',
      quickSettingsGroups: [{
        id: 'gnb-sector-carrier', titleZh: '载波', titleEn: 'Carrier', multiInstance: true,
        objectPath: 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.{j}.Carrier.{i}.',
        params: [{ name: 'Enable', titleZh: '启用', titleEn: 'Enable', leaf: 'Enable' }],
      }],
    };
    const workbook = await enrichParamConfigWorkbook(
      createParamConfigTemplateWorkbook('gNB', metadata),
      metadata,
    );
    const paths = workbook.getWorksheet(PARAM_MAPPING_SHEET)?.getColumn(4).values;
    expect(paths).toEqual(expect.arrayContaining([
      'Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.1.Carrier.1.Enable',
    ]));
    expect(paths).not.toEqual(expect.arrayContaining([
      'Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.1.Carrier.2.Enable',
      'Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.2.Carrier.1.Enable',
    ]));
  });

  it('generates one parameter for 4G and 5G groups below CellConfig', () => {
    for (const deviceType of ['eNB', 'gNB'] as const) {
      const metadata = {
        deviceType,
        quickSettingsGroups: [{
          id: 'radio-settings', titleZh: '无线参数', titleEn: 'Radio Settings', multiInstance: true,
          objectPath: 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.{j}.',
          params: [{ name: 'PowerClass', titleZh: '功率等级', titleEn: 'Power Class', leaf: 'PowerClass' }],
        }],
      };
      const source = createParamConfigTemplateWorkbook(deviceType, metadata);
      expect(XLSX.utils.sheet_to_json<unknown[]>(source.Sheets.CELL, { header: 1, defval: '' })[0])
        .toEqual([
          'Serial Number', deviceType === 'eNB' ? '*CELL_NUMBER' : 'Cell Index', 'Power Class',
        ]);
    }
  });

  it('imports user-defined column mappings from the final sheet', () => {
    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, XLSX.utils.aoa_to_sheet([
      ['Serial Number', 'Custom Header'],
      ['NR-SN-CUSTOM', 'custom-value'],
    ]), 'DEVICE');
    XLSX.utils.book_append_sheet(workbook, XLSX.utils.aoa_to_sheet([
      ['页面显示名称', '数据工作表', '参数列名', 'TRPath', '来源', '说明'],
      ['自定义参数', 'DEVICE', 'Custom Header', 'Device.Custom.1.Value', '用户自定义', ''],
    ]), PARAM_MAPPING_SHEET);

    const rows = parseParamConfigWorkbook(
      XLSX.write(workbook, { type: 'array', bookType: 'xlsx' }),
      'gNB',
      'now',
    );
    expect(rows[0].sheetParameters?.DEVICE?.[0]['Custom Header']).toBe('custom-value');
    expect(rows[0].workbookMappings).toEqual([{
      displayName: '自定义参数', sheet: 'DEVICE', header: 'Custom Header',
      trPath: 'Device.Custom.1.Value', source: 'custom',
    }]);
  });

  it('can re-import a downloaded compatible workbook without inventing partial mappings', async () => {
    const source = createParamConfigWorkbook([{
      deviceType: 'gNB',
      serialNumber: 'EXAMPLE-SN-001',
      sheetParameters: {
        DEVICE: [{
          'Serial Number': 'EXAMPLE-SN-001',
          'Periodic Inform Enable': 'true',
        }],
      },
      workbookMappings: [],
    }]);
    const downloaded = await enrichParamConfigWorkbook(source, {
      deviceType: 'gNB',
      quickSettingsGroups: [{
        id: 'device', titleZh: '设备', titleEn: 'Device', multiInstance: false,
        params: [{
          name: 'NTPEnable', titleZh: 'NTP', titleEn: 'NTP',
          standardPath: 'Device.Time.Enable',
        }],
      }],
    });

    const rows = parseParamConfigWorkbook(
      await downloaded.xlsx.writeBuffer(),
      'gNB',
      'now',
    );

    expect(rows[0].sheetParameters?.DEVICE?.[0]['Periodic Inform Enable']).toBe('true');
    expect(rows[0].workbookMappings).toEqual([]);
  });

  it('rejects retired workbooks without the parameter mapping sheet', () => {
    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, XLSX.utils.aoa_to_sheet([
      ['*SERIAL_NUMBER', '*CELL_NUMBER', 'CELL_NAME'],
      ['OLD-SN-001', 1, 'Legacy Cell'],
    ]), 'CELL');
    expect(() => parseParamConfigWorkbook(
      XLSX.write(workbook, { type: 'array', bookType: 'xlsx' }),
      'eNB',
      'now',
    )).toThrowError(expect.objectContaining({ code: 'missing_columns', field: PARAM_MAPPING_SHEET }));
  });

  it('parses mapped parameters across sheets and rejects any unmapped data column', () => {
    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, XLSX.utils.aoa_to_sheet([
      ['Serial Number', 'gNB ID'],
      ['NR-SN-001', 1001],
    ]), 'DEVICE');
    XLSX.utils.book_append_sheet(workbook, XLSX.utils.aoa_to_sheet([
      ['Serial Number', 'PLMN ID'],
      ['NR-SN-001', '46000'],
    ]), 'PLMN');
    XLSX.utils.book_append_sheet(workbook, XLSX.utils.aoa_to_sheet([
      ['页面显示名称', '数据工作表', '参数列名', 'TRPath', '来源', '说明'],
      ['gNB ID', 'DEVICE', 'gNB ID', 'Device.NR.1.GNBID', '系统预置', ''],
      ['PLMN ID', 'PLMN', 'PLMN ID', 'Device.NR.1.PLMN.1.PLMNID', '系统预置', ''],
    ]), PARAM_MAPPING_SHEET);
    const rows = parseParamConfigWorkbook(
      XLSX.write(workbook, { type: 'array', bookType: 'xlsx' }),
      'gNB',
      'now',
    );
    expect(rows).toMatchObject([{
      serialNumber: 'NR-SN-001',
      sheetParameters: {
        DEVICE: [{ 'Serial Number': 'NR-SN-001', 'gNB ID': 1001 }],
        PLMN: [{ 'Serial Number': 'NR-SN-001', 'PLMN ID': '46000' }],
      },
    }]);

    XLSX.utils.sheet_add_aoa(workbook.Sheets.DEVICE, [['Unmapped'], ['has-value']], { origin: 'C1' });
    expect(() => parseParamConfigWorkbook(
      XLSX.write(workbook, { type: 'array', bookType: 'xlsx' }),
      'gNB',
      'now',
    )).toThrowError(expect.objectContaining({ code: 'missing_columns', field: 'DEVICE.Unmapped' }));
  });

  it('ignores an unmapped column when spreadsheet editing leaves it completely empty', () => {
    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, XLSX.utils.aoa_to_sheet([
      ['Serial Number', 'Mapped', 'Empty Note Column'],
      ['SN-EDITED', 'value', ''],
    ]), 'DEVICE');
    XLSX.utils.book_append_sheet(workbook, XLSX.utils.aoa_to_sheet([
      ['页面显示名称', '数据工作表', '参数列名', 'TRPath', '来源', '说明'],
      ['Mapped', 'DEVICE', 'Mapped', 'Device.Example.Mapped', '系统预置', ''],
    ]), PARAM_MAPPING_SHEET);

    expect(parseParamConfigWorkbook(
      XLSX.write(workbook, { type: 'array', bookType: 'xlsx' }),
      'gNB',
      'now',
    )[0].sheetParameters?.DEVICE?.[0]).toEqual({
      'Serial Number': 'SN-EDITED',
      Mapped: 'value',
    });
  });

  it('validates mapped values by TRPath instead of legacy column aliases', () => {
    const metadata = {
      quickSettingsGroups: [{
        id: 'management', titleZh: '管理', titleEn: 'Management', multiInstance: false,
        params: [{
          name: 'Mode', titleZh: '模式', titleEn: 'Completely Different Label', type: 'enum',
          standardPath: 'Device.Management.Mode',
          enumOptions: [{ value: 'Auto', label: 'Auto' }, { value: 'Manual', label: 'Manual' }],
        }],
      }],
    };
    const workbook = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(workbook, XLSX.utils.aoa_to_sheet([
      ['Serial Number', 'User Chosen Header'],
      ['SN-001', 'Invalid'],
    ]), 'DEVICE');
    XLSX.utils.book_append_sheet(workbook, XLSX.utils.aoa_to_sheet([
      ['页面显示名称', '数据工作表', '参数列名', 'TRPath', '来源', '说明'],
      ['Mode', 'DEVICE', 'User Chosen Header', 'Device.Management.Mode', '用户自定义', ''],
    ]), PARAM_MAPPING_SHEET);
    expect(() => parseParamConfigWorkbook(
      XLSX.write(workbook, { type: 'array', bookType: 'xlsx' }),
      'gNB',
      'now',
      metadata,
    )).toThrowError(expect.objectContaining({ code: 'invalid_row', field: 'User Chosen Header' }));
  });

  it('uses each IPSEC TR path enum and carrier planning ranges as dropdowns', async () => {
    const metadata = {
      deviceType: 'gNB' as const,
      quickSettingsGroups: [{
        id: 'gnb-ipsec', titleZh: 'IPSec', titleEn: 'IPSec', multiInstance: true,
        objectPath: 'Device.FAP.Ipsec.{i}.',
        params: [{ name: 'IKE_ENCRYPTION', titleZh: '加密', titleEn: 'IKE_ENCRYPTION', leaf: 'IKE_ENCRYPTION' }],
      }, {
        id: 'gnb-cell', titleZh: '小区', titleEn: 'Cell', multiInstance: true,
        params: [
          {
            name: 'DLSubCarrierSpacing', titleZh: '载波间隔', titleEn: 'SubcarrierSpacing(DL)',
            standardPath: 'Device.Cell.1.DLSubCarrierSpacing',
          },
          {
            name: 'DLCarrierBandWidth', titleZh: '下行带宽', titleEn: 'DLBandwidth',
            standardPath: 'Device.Cell.1.DLCarrierBandWidth',
          },
        ],
      }, {
        id: 'device-time', titleZh: '时间', titleEn: 'Time', multiInstance: false,
        params: [{
          name: 'LocalTimeZoneName', titleZh: '时区', titleEn: 'Local Time Zone',
          standardPath: 'Device.Time.LocalTimeZoneName',
        }],
      }],
      quickSettingFields: getParamConfigExportFields('gNB'),
      paramMappings: [{
        id: 'ipsec-1', paramModelId: 'model', standardPath: 'Device.FAP.Ipsec.{i}.IKE_ENCRYPTION',
        privatePath: 'Device.FAP.Ipsec.{i}.IKE_ENCRYPTION', entryType: 'parameter', access: 'readWrite',
        dataType: 'STRING', changeApplies: 'Immediate', enumValues: 'aes128,aes256,3des',
        enumLabels: 'aes128,aes256,3des', isStorable: true, isActive: true,
      }],
    };
    const workbook = await enrichParamConfigWorkbook(
      createParamConfigTemplateWorkbook('gNB', metadata), metadata,
    );
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

  it('binds BaiBNQ carrier-bandwidth dropdowns to the generated SCS headers', async () => {
    const metadata = {
      deviceType: 'gNB' as const,
      quickSettingsGroups: [{
        id: 'gnb-cell', titleZh: '小区', titleEn: 'Cell', multiInstance: false,
        params: [
          {
            name: 'DLSubCarrierSpacing', titleZh: '下行子载波间隔', titleEn: 'DL SubCarrier Spacing',
            type: 'enum', standardPath: 'Device.Cell.1.DLSubCarrierSpacing',
            enumOptions: ['0', '1', '2'].map((value) => ({ value, label: value })),
          },
          {
            name: 'DLCarrierBandWidth', titleZh: '下行载波带宽', titleEn: 'DL Carrier Bandwidth',
            standardPath: 'Device.Cell.1.DLCarrierBandWidth',
          },
        ],
      }],
      quickSettingFields: getParamConfigExportFields('gNB'),
    };
    const workbook = await enrichParamConfigWorkbook(
      createParamConfigTemplateWorkbook('gNB', metadata), metadata,
    );
    const cell = workbook.getWorksheet('CELL')!;
    const scsColumn = cell.getRow(1).values.indexOf('DL SubCarrier Spacing');
    const bandwidthColumn = cell.getRow(1).values.indexOf('DL Carrier Bandwidth');

    expect(cell.getCell(2, bandwidthColumn).dataValidation.formulae)
      .toEqual([`INDIRECT("XOMC_BW_"&${cell.getColumn(scsColumn).letter}2)`]);
    expect(String(cell.getCell(1, bandwidthColumn).note)).not.toContain('25、52、79');
  });

  it('uses the public 5G network address-method values for Address Type', async () => {
    const metadata = {
      deviceType: 'gNB' as const,
      quickSettingsGroups: [{
        id: 'gnb-network', titleZh: '网络', titleEn: 'Network', multiInstance: false,
        params: [{
          name: 'AddressType', titleZh: '地址类型', titleEn: 'Address Type', type: 'enum',
          standardPath: 'Device.IP.Interface.1.AddressType',
          enumOptions: ['DHCP', 'Static', 'DHCPv6', 'Staticv6'].map((value) => ({ value, label: value })),
        }],
      }],
    };
    const workbook = await enrichParamConfigWorkbook(
      createParamConfigTemplateWorkbook('gNB', metadata), metadata,
    );
    const network = workbook.getWorksheet('INTERFACE')!;
    const addressTypeColumn = network.getRow(1).values.indexOf('Address Type');

    expect(network.getCell(1, addressTypeColumn).note)
      .toContain('可选值：DHCP、Static、DHCPv6、Staticv6');
    expect(network.getCell(2, addressTypeColumn).dataValidation.formulae)
      .toEqual(['"DHCP,Static,DHCPv6,Staticv6"']);
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

});
