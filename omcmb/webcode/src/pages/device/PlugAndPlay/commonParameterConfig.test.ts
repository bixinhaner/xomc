import { describe, expect, it } from 'vitest';
import { sanitizeCommonParamConfig, withInitialCommonRadioInstance } from './commonParameterConfig';

describe('sanitizeCommonParamConfig', () => {
  it('removes excluded and retired fields from common parameters', () => {
    const result = sanitizeCommonParamConfig({
      sheetParameters: {
        DEVICE: [{ URL: 'http://acs.example.com', 'Periodic Inform Interval': 300, 'Time Zone Term': 'CET-1' }],
        INTERFACE: [{ 'Interface Name': 'eth0', 'Address Type': 'IPv4', 'IP Address': '10.0.0.2', 'Vlan Name': 'wan', 'OMC IP': '10.0.0.3' }],
        IPSEC: [{ FORCEENCAPS: '1', TUNNEL_ENABLE: '1' }],
      },
    });

    expect(result.sheetParameters).toEqual({
      DEVICE: [{ URL: 'http://acs.example.com', 'Periodic Inform Interval': 300 }],
      INTERFACE: [{ 'IP Address': '10.0.0.2' }],
      IPSEC: [{ TUNNEL_ENABLE: '1' }],
    });
  });
});

describe('withInitialCommonRadioInstance', () => {
  it('creates the first NR cell with an explicit instance index', () => {
    const result = withInitialCommonRadioInstance({ deviceType: 'gNB' }, 'gNB');
    expect(result.sheetParameters.CELL).toEqual([
      expect.objectContaining({ 'Cell Index': 1 }),
    ]);
  });

  it('uses BTS Index for a GSM BSC product', () => {
    const result = withInitialCommonRadioInstance({}, 'GSM', 'Example BSC Product');
    expect(result.sheetParameters.GSM).toEqual([
      expect.objectContaining({ 'BTS Index': 1 }),
    ]);
  });

  it('does not recreate a deliberately empty radio instance list', () => {
    const result = withInitialCommonRadioInstance({ sheetParameters: { CELL: [] } }, 'eNB');
    expect(result.sheetParameters.CELL).toEqual([]);
  });
});
