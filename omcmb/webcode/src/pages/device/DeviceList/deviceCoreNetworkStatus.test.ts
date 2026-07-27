import { describe, expect, it } from 'vitest';
import type { Device } from '@core/types/device';
import { amfStatusForDevice, bscLinkStatusForDevice, mmeStatusForDevice } from './deviceCoreNetworkStatus';

const device = (networkType: Device['networkType']): Device => ({
  id: networkType,
  sn: `SN-${networkType}`,
  networkType,
  mmeStatus: 'connected',
  amfStatus: 'connected',
  bscLinkStatus: 'connected',
} as Device);

describe('设备列表核心网状态展示', () => {
  it('MME 仅对 4G eNB 展示', () => {
    expect(mmeStatusForDevice(device('eNB'))).toBe('connected');
    expect(mmeStatusForDevice(device('gNB'))).toBe('');
    expect(mmeStatusForDevice(device('GSM'))).toBe('');
  });

  it('AMF 仅对 5G gNB 展示', () => {
    expect(amfStatusForDevice(device('eNB'))).toBe('');
    expect(amfStatusForDevice(device('gNB'))).toBe('connected');
    expect(amfStatusForDevice(device('GSM'))).toBe('');
  });

  it('BSC Link 仅对 2G GSM 展示', () => {
    expect(bscLinkStatusForDevice(device('eNB'))).toBe('');
    expect(bscLinkStatusForDevice(device('gNB'))).toBe('');
    expect(bscLinkStatusForDevice(device('GSM'))).toBe('connected');
  });
});
