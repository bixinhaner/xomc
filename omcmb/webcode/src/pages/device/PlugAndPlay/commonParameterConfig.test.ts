import { describe, expect, it } from 'vitest';
import {
  sanitizeCommonParamConfig,
  withInitialCommonParamConfig,
  withInitialCommonRadioInstance,
} from './commonParameterConfig';

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

describe('withInitialCommonParamConfig', () => {
  it('backfills required gNB allocation defaults for legacy common configs', () => {
    const result = withInitialCommonParamConfig({
      deviceType: 'gNB',
      sheetParameters: { CELL: [{ 'Cell Index': 1 }] },
    }, 'gNB');

    expect(result.gnbIdLength).toBe(24);
    expect(result.gnbIdAllocation).toEqual({
      start: 1,
      end: 16_777_215,
      step: 1,
      reserved: [],
    });
    expect(result.pciAllocation).toEqual({
      start: 0,
      end: 1007,
      step: 1,
      reserved: [],
    });
    expect(result.sheetParameters.CELL).toEqual([{ 'Cell Index': 1 }]);
  });

  it('keeps saved gNB allocation values while filling missing fields', () => {
    const result = withInitialCommonParamConfig({
      gnbIdAllocation: { start: 10, reserved: [{ start: 20, end: 30 }] },
      pciAllocation: { end: 500 },
      gnbIdLength: 28,
    }, 'gNB');

    expect(result.gnbIdLength).toBe(28);
    expect(result.gnbIdAllocation).toEqual({
      start: 10,
      end: 16_777_215,
      step: 1,
      reserved: [{ start: 20, end: 30 }],
    });
    expect(result.pciAllocation).toEqual({
      start: 0,
      end: 500,
      step: 1,
      reserved: [],
    });
  });
});
