import { describe, expect, it } from 'vitest';
import {
  mergeParamConfigFormValues,
  toParamConfigFormValues,
  withTemplateSheetParameters,
} from './paramConfigDetail';

describe('parameter configuration detail mapping', () => {
  it('maps 5G CELL and PLMN sheet values into the detail form', () => {
    expect(toParamConfigFormValues({
      deviceType: 'gNB',
      serialNumber: '5G-SN-001',
      sheetParameters: {
        CELL: [{
          '*Serial Number': '5G-SN-001',
          'gNB Name': 'NR Cell',
          '*gNB ID': 123456,
          '*gNB Lenth': 24,
          '*PCI': 150,
          'SSB Frequency': 3600000,
          'Freq BandIndicator': 78,
          NRARFCNDL: 360000,
          NRARFCNUL: 361000,
          DLBandwidth: '100MHz',
        }],
        PLMN: [{
          'Serial Number': '5G-SN-001',
          '*NCI': 123456789,
          '*TAC': 100,
          '*RANAC': 2,
          '*PLMN ID': 46000,
          '*PRIMARY': 1,
        }],
      },
    })).toMatchObject({
      gnbName: 'NR Cell',
      gnbId: 123456,
      gnbIdLength: 24,
      pci: 150,
      ssbFrequency: 3600000,
      freqBandIndicator: 78,
      nrarfcnndl: 360000,
      nrarfcnul: 361000,
      dlbandwidth: '100',
      nci: 123456789,
      tac: 100,
      ranac: 2,
      plmnId: 46000,
      plmnConfigList: [{ plmnId: 46000, primary: '1' }],
    });
  });

  it('shows populated imported 5G device, interface and IPsec values', () => {
    expect(toParamConfigFormValues({
      deviceType: 'gNB',
      sheetParameters: {
        DEVICE: [{
          'NTP Enable': true,
          'NTP Server1': '192.0.2.1',
          'Local Time Zone': 'UTC0',
        }],
        INTERFACE: [{
          'IP Address': '192.0.2.10',
          'Subnet Mask': '255.255.255.0',
          'Vlan ID': 100,
        }],
        IPSEC: [{
          TUNNEL_ENABLE: '1',
          TUNNEL_GATEWAY: '198.51.100.1',
          LEFT_AUTH: 'psk',
        }],
      },
    })).toMatchObject({
      ntpSync: '1',
      networkConfigList: [{
        'IP Address': '192.0.2.10',
        'Subnet Mask': '255.255.255.0',
        'Vlan ID': 100,
      }],
      ipsecList: [{
        TUNNEL_ENABLE: '1',
        TUNNEL_GATEWAY: '198.51.100.1',
        LEFT_AUTH: 'psk',
      }],
      IPSEC_ENABLE: '1',
      serviceIp: '192.0.2.10',
      serviceMask: '255.255.255.0',
      serviceVlan: 100,
    });
  });

  it('maps 4G CELL and NETWORK_ENABLE values into the detail form', () => {
    expect(toParamConfigFormValues({
      deviceType: 'eNB',
      serialNumber: '4G-SN-001',
      sheetParameters: {
        CELL: [{
          '*SERIAL_NUMBER': '4G-SN-001',
          CELL_NAME: 'LTE Cell',
          '*ECI': 222218,
          '*BAND': 42,
          '*EARFCN_DL': 43000,
          '*BANDWIDTH_DL': 'n50',
          '*PCI': 218,
          SPECIAL_SUBFRAME_PATTERNS: 5,
          SUBFRAME_ASSIGNMENT: 1,
          '*ROOT_SEQUENCE_INDEX': 30,
          '*TAC': 2,
        }],
        NETWORK_ENABLE: [{
          '*SERIAL_NUMBER': '4G-SN-001',
          '*PLMN': 46000,
          MME_IP: '10.0.0.1',
          IPSEC_ENABLE: 1,
          HALOB_ENABLE: 0,
        }],
        NETWORK: [{
          '*SERIAL_NUMBER': '4G-SN-001',
          'NTP Enable': 0,
        }],
      },
    })).toMatchObject({
      cellName: 'LTE Cell',
      cellIdentity: 222218,
      bandsSupport: 42,
      frequency: 43000,
      bandWidth: 'n50',
      phycellid: 218,
      specialSubframePatterns: '5',
      subframeAssignment: '1',
      rootSequenceIndex: 30,
      tac: 2,
      plmnId: 46000,
      plmnConfigList: [{ plmnId: 46000 }],
      ntpSync: '0',
      mmeList: [{ mmeIp: '10.0.0.1' }],
      ipsecEnable: '1',
      halobEnable: '0',
    });
  });

  it('defaults a blank eNB IPsec switch to off', () => {
    expect(toParamConfigFormValues({
      deviceType: 'eNB',
      sheetParameters: {
        NETWORK_ENABLE: [{ IPSEC_ENABLE: '' }],
      },
    })).toMatchObject({ ipsecEnable: '0' });
  });

  it('defaults the gNB IPsec switch to off', () => {
    expect(toParamConfigFormValues({
      deviceType: 'gNB',
      sheetParameters: {},
    })).toMatchObject({ IPSEC_ENABLE: '0' });
  });

  it('round-trips multiple LTE serving PLMNs and the NTP switch', () => {
    const current = withTemplateSheetParameters({
      deviceType: 'eNB',
      serialNumber: '4G-SN-PLMN',
      plmnConfigList: [],
      sheetParameters: {
        NETWORK_ENABLE: [{ '*SERIAL_NUMBER': '4G-SN-PLMN', '*PLMN': '46000' }],
        NETWORK: [{ '*SERIAL_NUMBER': '4G-SN-PLMN', 'NTP Enable': '1' }],
      },
    });
    const form = toParamConfigFormValues(current);

    expect(form).toMatchObject({
      plmnConfigList: [{ plmnId: '46000' }],
      ntpSync: '1',
    });

    expect(mergeParamConfigFormValues(current, {
      ...form,
      plmnConfigList: [{ plmnId: '46000' }, { plmnId: '46001' }],
      ntpSync: '0',
    })).toMatchObject({
      plmnConfigList: [{ plmnId: '46000' }, { plmnId: '46001' }],
      ntpSync: '0',
      sheetParameters: {
        NETWORK_ENABLE: [{ '*PLMN': '46000' }],
        NETWORK: [{ 'NTP Enable': '0' }],
      },
    });
  });

  it('writes edited structured fields back to imported sheet parameters', () => {
    const current = {
      deviceType: 'eNB',
      serialNumber: '4G-SN-001',
      sheetParameters: {
        CELL: [{
          '*SERIAL_NUMBER': '4G-SN-001',
          CELL_NAME: 'Old Cell',
          '*PCI': 100,
        }],
      },
    };

    expect(mergeParamConfigFormValues(current, {
      ...toParamConfigFormValues(current),
      cellName: 'New Cell',
      phycellid: 200,
    })).toMatchObject({
      cellName: 'New Cell',
      phycellid: 200,
      sheetParameters: {
        CELL: [{
          '*SERIAL_NUMBER': '4G-SN-001',
          CELL_NAME: 'New Cell',
          '*PCI': 200,
        }],
      },
    });
  });

  it('keeps direct imported-sheet edits and refreshes structured values', () => {
    const current = {
      deviceType: 'gNB',
      serialNumber: '5G-SN-001',
      sheetParameters: {
        CELL: [{
          '*Serial Number': '5G-SN-001',
          'gNB Name': 'Old NR Cell',
          '*PCI': 100,
        }],
      },
    };
    const initialValues = toParamConfigFormValues(current);

    expect(mergeParamConfigFormValues(current, {
      ...initialValues,
      sheetParameters: {
        CELL: [{
          '*Serial Number': '5G-SN-001',
          'gNB Name': 'Raw Edited NR Cell',
          '*PCI': 300,
        }],
      },
    })).toMatchObject({
      gnbName: 'Raw Edited NR Cell',
      pci: 300,
      sheetParameters: {
        CELL: [{ 'gNB Name': 'Raw Edited NR Cell', '*PCI': 300 }],
      },
    });
  });

  it('preserves the complete imported workbook when the editor submits only mounted fields', () => {
    const current = {
      deviceType: 'gNB',
      serialNumber: 'EXAMPLE-SN-001',
      sheetParameters: {
        DEVICE: [{ 'Serial Number': 'EXAMPLE-SN-001', 'gNB Name': 'Original', 'gNB ID': '22' }],
        CELL: [{ 'Serial Number': 'EXAMPLE-SN-001', Band: '1', PCI: '10' }],
        PLMN: [{ 'Serial Number': 'EXAMPLE-SN-001', PLMN: '46000' }],
        INTERFACE: [{ 'Serial Number': 'EXAMPLE-SN-001', 'IP Type [1.1]': 'DHCP' }],
        '1588_CONFIGURATION': [{ 'Serial Number': 'EXAMPLE-SN-001', Mode: 'FREE_OSCILLATION' }],
        IPSEC: [{ 'Serial Number': 'EXAMPLE-SN-001', 'IPSec Enable': '1' }],
      },
    };

    const merged = mergeParamConfigFormValues(current, {
      ...toParamConfigFormValues(current),
      sheetParameters: {
        DEVICE: [{ 'gNB Name': 'Edited', URL: 'must-not-be-added' }],
        CELL: [{ PCI: '20', ULBandwidth: 'must-not-be-added' }],
        PLMN: [{ PLMN: '46001' }],
        IPSEC: [{ 'IPSec Enable': '0' }],
      },
    });

    expect(Object.keys(merged.sheetParameters)).toEqual([
      'DEVICE', 'CELL', 'PLMN', 'INTERFACE', '1588_CONFIGURATION', 'IPSEC',
    ]);
    expect(merged.sheetParameters).toMatchObject({
      DEVICE: [{ 'gNB Name': 'Edited', 'gNB ID': '22' }],
      CELL: [{ Band: '1', PCI: '20' }],
      INTERFACE: [{ 'IP Type [1.1]': 'DHCP' }],
      '1588_CONFIGURATION': [{ Mode: 'FREE_OSCILLATION' }],
    });
    expect(merged.sheetParameters.DEVICE[0]).not.toHaveProperty('URL');
    expect(merged.sheetParameters.CELL[0]).not.toHaveProperty('ULBandwidth');
  });

  it('writes both edited carrier bandwidths back to their imported dynamic columns', () => {
    const current = {
      deviceType: 'gNB',
      serialNumber: 'NR-SN-001',
      sheetParameters: {
        CELL: [{
          'Serial Number': 'NR-SN-001',
          'DL Carrier Bandwidth': '',
          'UL Carrier Bandwidth': '',
        }],
      },
    };

    const merged = mergeParamConfigFormValues(current, {
      ...toParamConfigFormValues(current),
      dlbandwidth: '106',
      ulbandwidth: '106',
    });

    expect(merged.sheetParameters.CELL[0]).toEqual({
      'Serial Number': 'NR-SN-001',
      'DL Carrier Bandwidth': '106',
      'UL Carrier Bandwidth': '106',
    });
    expect(merged.sheetParameters.CELL[0]).not.toHaveProperty('DLBandwidth');
    expect(merged.sheetParameters.CELL[0]).not.toHaveProperty('ULBandwidth');
  });

  it('preserves all imported eNB sheets when the editor submits only mounted fields', () => {
    const current = {
      deviceType: 'eNB',
      serialNumber: 'LTE-SN-001',
      sheetParameters: {
        CELL: [{ '*SERIAL_NUMBER': 'LTE-SN-001', CELL_NAME: 'Original', '*PCI': '10' }],
        NETWORK_ENABLE: [{ '*SERIAL_NUMBER': 'LTE-SN-001', '*PLMN': '46000' }],
        NETWORK_IPSEC: [{ '*SERIAL_NUMBER': 'LTE-SN-001', '*TUNNEL_INDEX': '1' }],
        '1588_CONFIGURATION': [{ '*SERIAL_NUMBER': 'LTE-SN-001', '*DOMAIN': '24' }],
        NETWORK: [{ '*SERIAL_NUMBER': 'LTE-SN-001', 'WAN IP': '192.0.2.10' }],
      },
    };

    const merged = mergeParamConfigFormValues(current, {
      ...toParamConfigFormValues(current),
      sheetParameters: {
        CELL: [{ CELL_NAME: 'Edited', '*PCI': '20', URL: 'must-not-be-added' }],
      },
    });

    expect(Object.keys(merged.sheetParameters)).toEqual([
      'CELL', 'NETWORK_ENABLE', 'NETWORK_IPSEC', '1588_CONFIGURATION', 'NETWORK',
    ]);
    expect(merged.sheetParameters).toMatchObject({
      CELL: [{ CELL_NAME: 'Edited', '*PCI': '20' }],
      NETWORK_ENABLE: [{ '*PLMN': '46000' }],
      NETWORK_IPSEC: [{ '*TUNNEL_INDEX': '1' }],
      '1588_CONFIGURATION': [{ '*DOMAIN': '24' }],
      NETWORK: [{ 'WAN IP': '192.0.2.10' }],
    });
    expect(merged.sheetParameters.CELL[0]).not.toHaveProperty('URL');
  });

  it('preserves unmounted imported GSM columns after a partial editor save', () => {
    const current = {
      deviceType: 'GSM',
      serialNumber: 'GSM-SN-001',
      sheetParameters: {
        GSM: [{
          'Serial Number': 'GSM-SN-001',
          IPA: '6969',
          'Unit ID': '7',
          'Remote IP': '192.0.2.20',
          OMC: '192.0.2.30',
        }],
      },
    };

    const merged = mergeParamConfigFormValues(current, {
      ...toParamConfigFormValues(current),
      sheetParameters: {
        GSM: [{ IPA: '6970', URL: 'must-not-be-added' }],
      },
    });

    expect(merged.sheetParameters.GSM[0]).toEqual({
      'Serial Number': 'GSM-SN-001',
      IPA: '6970',
      'Unit ID': '7',
      'Remote IP': '192.0.2.20',
      OMC: '192.0.2.30',
    });
  });

  it('persists added and deleted primary radio instances without renumbering', () => {
    const current = {
      deviceType: 'gNB',
      serialNumber: 'NR-MULTI-001',
      sheetParameters: {
        CELL: [
          { '*Serial Number': 'NR-MULTI-001', 'Cell Index': 1, '*PCI': '10' },
          { '*Serial Number': 'NR-MULTI-001', 'Cell Index': 3, '*PCI': '30' },
        ],
      },
    };

    const merged = mergeParamConfigFormValues(current, {
      ...toParamConfigFormValues(current),
      sheetParameters: {
        CELL: [
          { 'Cell Index': 3, '*PCI': '31' },
          { 'Cell Index': 4, '*PCI': '40' },
        ],
      },
    });

    expect(merged.sheetParameters.CELL).toEqual([
      { '*Serial Number': 'NR-MULTI-001', 'Cell Index': 3, '*PCI': '31' },
      { 'Cell Index': 4, '*PCI': '40' },
    ]);
  });

  it('backfills all current 5G template fields without Duplex Mode', () => {
    const hydrated = withTemplateSheetParameters({
      deviceType: 'gNB',
      serialNumber: '5G-HISTORY-001',
      sheetParameters: {
        CELL: [{ '*Serial Number': '5G-HISTORY-001', '*PCI': 321, 'Duplex Mode': 'TDD' }],
      },
    });

    expect(Object.keys(hydrated.sheetParameters ?? {})).toEqual([
      'DEVICE', 'CELL', 'PLMN', 'INTERFACE', 'IPSEC',
    ]);
    expect(Object.values(hydrated.sheetParameters ?? {})
      .flatMap((rows) => Object.keys(rows[0] ?? {}))).toHaveLength(87);
    expect(hydrated.sheetParameters?.CELL[0]).not.toHaveProperty('Duplex Mode');
    expect(hydrated.sheetParameters?.CELL[0]).toMatchObject({
      '*Serial Number': '5G-HISTORY-001',
      '*PCI': 321,
      ULBandwidth: '',
      'SubcarrierSpacing(UL)': '',
    });
  });

  it('backfills all LTE template fields when an old row has no sheet data', () => {
    const hydrated = withTemplateSheetParameters({
      deviceType: 'eNB',
      serialNumber: '4G-HISTORY-001',
    });

    expect(Object.keys(hydrated.sheetParameters ?? {})).toEqual([
      'CELL', 'NETWORK_ENABLE', 'NETWORK_IPSEC', '1588_CONFIGURATION', 'NETWORK',
    ]);
    expect(Object.values(hydrated.sheetParameters ?? {})
      .flatMap((rows) => Object.keys(rows[0] ?? {}))).toHaveLength(62);
    expect(hydrated.sheetParameters?.CELL[0]['*SERIAL_NUMBER'])
      .toBe('4G-HISTORY-001');
    expect(hydrated.sheetParameters?.['1588_CONFIGURATION'][0])
      .toHaveProperty('UNICAST_SERVER_IP_ADDRESS', '');
    expect(hydrated.sheetParameters?.NETWORK[0]).toMatchObject({
      'NTP Enable': '0',
    });
    expect(hydrated.sheetParameters?.NETWORK_ENABLE[0]).toMatchObject({
      IPSEC_ENABLE: '0',
    });
  });

  it('backfills the GSM workbook fields and spreadsheet defaults', () => {
    const hydrated = withTemplateSheetParameters({
      deviceType: 'GSM',
      serialNumber: '2G-HISTORY-001',
    });

    expect(hydrated.sheetParameters).toEqual({
      GSM: [{
        'Serial Number': '2G-HISTORY-001',
        IPA: '6969',
        'Unit ID': '',
        'Remote IP': '',
        'Bind IP': '',
        'WAN IP': '',
        Synchronization: 'GNSS',
        OMC: '',
      }],
    });
  });

  it('maps grouped 5G interface, AMF and slice fields without a second raw editor', () => {
    const current = withTemplateSheetParameters({
      deviceType: 'gNB',
      serialNumber: '5G-SN-002',
      sheetParameters: {
        PLMN: [{ 'Serial Number': '5G-SN-002', 'AMF IP:DEFAULT': '10.0.0.1', SD: '1', 'SD Value': '010203' }],
        INTERFACE: [{ 'Serial Number': '5G-SN-002', 'IP Address': '192.0.2.10', 'Subnet Mask': '255.255.255.0' }],
      },
    });
    const form = toParamConfigFormValues(current);

    expect(form).toMatchObject({
      serviceIp: '192.0.2.10',
      serviceMask: '255.255.255.0',
      amfList: [{ amfIp: '10.0.0.1', amfPort: '' }],
      sliceConfigList: [{ sd: '1', sdValue: '010203' }],
    });

    expect(mergeParamConfigFormValues(current, {
      ...form,
      networkConfigList: [{
        ...(form.networkConfigList as Array<Record<string, unknown>>)[0],
        'IP Address': '192.0.2.20',
      }],
      amfList: [{ amfIp: '10.0.0.2', amfPort: '38412' }],
      sliceConfigList: [{ sd: '1', sdValue: '112233' }],
    })).toMatchObject({
      sheetParameters: {
        INTERFACE: [{ 'IP Address': '192.0.2.20' }],
        PLMN: [{ 'AMF IP:DEFAULT': '10.0.0.2', 'SD Value': '112233' }],
      },
    });
  });

  it('round-trips multiple network instances', () => {
    const current = withTemplateSheetParameters({
      deviceType: 'gNB',
      serialNumber: '5G-NETWORK-001',
      sheetParameters: {
        INTERFACE: [
          { 'Serial Number': '5G-NETWORK-001', 'Interface Name': 'eth0', 'IP Address': '192.0.2.10' },
          { 'Serial Number': '5G-NETWORK-001', 'Interface Name': 'eth1', 'IP Address': '192.0.2.20' },
        ],
      },
    });
    const form = toParamConfigFormValues(current);

    expect(form).toMatchObject({
      networkConfigList: [
        { 'Interface Name': 'eth0', 'IP Address': '192.0.2.10' },
        { 'Interface Name': 'eth1', 'IP Address': '192.0.2.20' },
      ],
    });

    expect(mergeParamConfigFormValues(current, {
      ...form,
      networkConfigList: [
        { 'Interface Name': 'eth1', 'IP Address': '192.0.2.21' },
        { 'Interface Name': 'eth2', 'IP Address': '192.0.2.30' },
      ],
    })).toMatchObject({
      sheetParameters: {
        INTERFACE: [
          { 'Interface Name': 'eth1', 'IP Address': '192.0.2.21' },
          { 'Interface Name': 'eth2', 'IP Address': '192.0.2.30' },
        ],
      },
    });
  });

  it('round-trips LTE IPsec template rows through the grouped tunnel list', () => {
    const current = withTemplateSheetParameters({
      deviceType: 'eNB',
      serialNumber: '4G-SN-002',
      sheetParameters: {
        NETWORK_IPSEC: [{
          '*SERIAL_NUMBER': '4G-SN-002',
          '*TUNNEL_INDEX': '1',
          '*TUNNEL_ENABLE': '1',
          '*TUNNEL_GATEWAY': '10.0.0.1',
          FRAGMENTATION: 'yes',
        }],
      },
    });
    const form = toParamConfigFormValues(current);
    expect(form).toMatchObject({
      ipsecList: [{
        TUNNEL_INDEX: '1',
        TUNNEL_ENABLE: true,
        TUNNEL_GATEWAY: '10.0.0.1',
        FRAGMENTATION: 'yes',
      }],
    });

    const edited = (form.ipsecList as Array<Record<string, unknown>>).map((row) => ({
      ...row,
      TUNNEL_GATEWAY: '10.0.0.2',
    }));
    expect(mergeParamConfigFormValues(current, { ...form, ipsecList: edited })).toMatchObject({
      sheetParameters: {
        NETWORK_IPSEC: [{
          '*SERIAL_NUMBER': '4G-SN-002',
          '*TUNNEL_INDEX': '1',
          '*TUNNEL_GATEWAY': '10.0.0.2',
          FRAGMENTATION: 'yes',
        }],
      },
    });
  });

  it('preserves LTE quick-setting-only fields while refreshing workbook fields', () => {
    const current = withTemplateSheetParameters({
      deviceType: 'eNB',
      serialNumber: '4G-SN-003',
      BandSupport: 'Band 40',
      tfcsManagerPrimsrc: '3',
      NTPServer2: '192.0.2.2',
    });
    const form = toParamConfigFormValues(current);
    const merged = mergeParamConfigFormValues(current, {
      ...form,
      BandSupport: 'Band 41',
      tfcsManagerPrimsrc: '10',
      NTPServer2: '192.0.2.22',
      cellName: 'LTE Cell Updated',
    });

    expect(merged).toMatchObject({
      BandSupport: 'Band 41',
      tfcsManagerPrimsrc: '10',
      NTPServer2: '192.0.2.22',
      sheetParameters: {
        CELL: [{ CELL_NAME: 'LTE Cell Updated' }],
        '1588_CONFIGURATION': [{ '*SYNCHRONIZATION_MODE': '10' }],
      },
    });
  });

  it('preserves quick-settings-only values when saving a gNB policy', () => {
    const current = withTemplateSheetParameters({
      deviceType: 'gNB',
      serialNumber: '5G-SN-QUICK',
      SyncSource: 'GPS',
      RFEnable: '0',
    });
    const form = toParamConfigFormValues(current);

    expect(mergeParamConfigFormValues(current, {
      ...form,
      SyncSource: 'GPS,GLONASS',
      RFEnable: '1',
      IPSEC_ENABLE: '1',
    })).toMatchObject({
      SyncSource: 'GPS,GLONASS',
      RFEnable: '1',
      IPSEC_ENABLE: '1',
    });
  });

  it('loads and saves product-model network parameters through workbook mappings', () => {
    const path = 'Device.IP.Interface.1.IPv4Address.1.IPAddress';
    const current = withTemplateSheetParameters({
      deviceType: 'gNB',
      serialNumber: '5G-SN-NETWORK',
      workbookMappings: [{ sheet: 'INTERFACE', header: 'IP Address', trPath: path }],
      sheetParameters: {
        INTERFACE: [{ 'Serial Number': '5G-SN-NETWORK', 'IP Address': '192.0.2.10' }],
      },
    });
    const form = toParamConfigFormValues(current);

    expect(form.networkParameterValues).toEqual({ [path]: '192.0.2.10' });
    const firstSave = mergeParamConfigFormValues(current, {
      ...form,
      networkParameterValues: { [path]: '192.0.2.20' },
    });
    expect(firstSave.sheetParameters.INTERFACE[0]['IP Address']).toBe('192.0.2.20');
    expect(firstSave).not.toHaveProperty('networkParameterValues');

    const secondForm = toParamConfigFormValues(firstSave);
    const secondSave = mergeParamConfigFormValues(firstSave, {
      ...secondForm,
      networkParameterValues: { [path]: '192.0.2.30' },
    });
    expect(secondSave.sheetParameters.INTERFACE[0]['IP Address']).toBe('192.0.2.30');
    expect(secondSave).not.toHaveProperty('networkParameterValues');
  });
});
