import { describe, expect, it } from 'vitest';
import { inferDeviceTimeMode, isNrNetworkType, mapDeviceTimeModeLabel, shouldShowDeviceTimeNtpServerFields } from '../deviceTimeMode';

describe('device time mode mapping', () => {
  const labels = {
    nrServer: 'NTP Server',
    nrClient: 'NTP Client',
    enable: 'Enable',
    disable: 'Disable',
  };

  it('identifies NR network type', () => {
    expect(isNrNetworkType('nr')).toBe(true);
    expect(isNrNetworkType('NR')).toBe(true);
    expect(isNrNetworkType(' lte ')).toBe(false);
  });

  it('maps labels by network type', () => {
    expect(mapDeviceTimeModeLabel('1', 'fallback', 'nr', labels)).toBe('NTP Client');
    expect(mapDeviceTimeModeLabel('0', 'fallback', 'nr', labels)).toBe('NTP Server');
    expect(mapDeviceTimeModeLabel('true', 'fallback', 'lte', labels)).toBe('Enable');
    expect(mapDeviceTimeModeLabel('false', 'fallback', 'lte', labels)).toBe('Disable');
    expect(mapDeviceTimeModeLabel('unknown', 'fallback', 'lte', labels)).toBe('fallback');
  });

  it('uses explicit Device.Time.Enable when available', () => {
    const schemaByPath = new Map<string, any>([
      ['Device.Time.Enable', { path: 'Device.Time.Enable', currentValue: 'false', type: 'string' as const }],
    ]);
    const rawByPath = new Map<string, any>();

    expect(inferDeviceTimeMode(rawByPath, schemaByPath, 'nr')).toBe('0');
    expect(inferDeviceTimeMode(rawByPath, schemaByPath, 'lte')).toBe('0');
  });

  it('falls back by NTP server presence with NR/non-NR split', () => {
    const schemaByPath = new Map<string, any>([
      ['Device.Time.NTPServer1', { path: 'Device.Time.NTPServer1', currentValue: '1.us.pool.ntp.org', type: 'string' as const }],
    ]);
    const rawByPath = new Map<string, any>();

    // NR: has server -> client(1)
    expect(inferDeviceTimeMode(rawByPath, schemaByPath, 'nr')).toBe('1');
    // non-NR: has server -> enable(1)
    expect(inferDeviceTimeMode(rawByPath, schemaByPath, 'lte')).toBe('1');
  });

  it('hides NTP server fields only for NR server mode', () => {
    expect(shouldShowDeviceTimeNtpServerFields('nr', '1')).toBe(true);
    expect(shouldShowDeviceTimeNtpServerFields('nr', '0')).toBe(false);
    expect(shouldShowDeviceTimeNtpServerFields('nr', 'server')).toBe(false);
    expect(shouldShowDeviceTimeNtpServerFields('lte', '0')).toBe(true);
  });
});
