import { describe, expect, it } from 'vitest';
import { sanitizeCommonParamConfig } from './commonParameterConfig';

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
