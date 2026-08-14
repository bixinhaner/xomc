import { describe, expect, it } from 'vitest';
import { sanitizeRetiredParamConfigFields } from './retiredParamConfigFields';

describe('sanitizeRetiredParamConfigFields', () => {
  it('removes retired fields while preserving the supported local time zone', () => {
    expect(sanitizeRetiredParamConfigFields({
      DEVICE: [{ 'Local Time Zone': 'Asia/Shanghai', 'Time Zone Term': 'CET-1' }],
      INTERFACE: [{ 'IP Address': '192.0.2.10', 'OMC IP': '198.51.100.10' }],
      NETWORK: [{ 'WAN IP': '192.0.2.20', 'OMC IP': '198.51.100.20' }],
    })).toEqual({
      DEVICE: [{ 'Local Time Zone': 'Asia/Shanghai' }],
      INTERFACE: [{ 'IP Address': '192.0.2.10' }],
      NETWORK: [{ 'WAN IP': '192.0.2.20' }],
    });
  });

  it('removes Duplex Mode only from 5G CELL parameters', () => {
    const sheets = { CELL: [{ 'Duplex Mode': 'TDD', PCI: 10 }] };

    expect(sanitizeRetiredParamConfigFields(sheets, 'gNB')).toEqual({
      CELL: [{ PCI: 10 }],
    });
    expect(sanitizeRetiredParamConfigFields(sheets, 'eNB')).toEqual(sheets);
  });
});
