import { describe, expect, it } from 'vitest';
import type { Device } from '@core/types/device';
import {
  amfStatusForDevice,
  amfStatusMessageIdForDevice,
  bscLinkStatusForDevice,
  bscLinkStatusMessageIdForDevice,
  mmeStatusMessageIdForDevice,
  mmeStatusForDevice,
} from './deviceCoreNetworkStatus';

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

  it('历史 partial 按任一 MME 已连接归一为已连接展示', () => {
    const lte = {
      ...device('eNB'),
      mmeStatus: 'partial',
    };

    expect(mmeStatusForDevice(lte)).toBe('connected');
    expect(mmeStatusMessageIdForDevice(lte)).toBe('status.connected');
  });

  it('未连接状态映射为本地化文案 key', () => {
    const lte = { ...device('eNB'), mmeStatus: 'disconnected' };

    expect(mmeStatusMessageIdForDevice(lte)).toBe('status.disconnected');
  });

  it('AMF 仅对 5G gNB 展示', () => {
    expect(amfStatusForDevice(device('eNB'))).toBe('');
    expect(amfStatusForDevice(device('gNB'))).toBe('connected');
    expect(amfStatusForDevice(device('GSM'))).toBe('');
  });

  it('AMF 未连接状态映射为本地化文案 key', () => {
    const nr = { ...device('gNB'), amfStatus: 'disconnected' };

    expect(amfStatusMessageIdForDevice(nr)).toBe('status.disconnected');
  });

  it('BSC Link 仅对 2G GSM 展示', () => {
    expect(bscLinkStatusForDevice(device('eNB'))).toBe('');
    expect(bscLinkStatusForDevice(device('gNB'))).toBe('');
    expect(bscLinkStatusForDevice(device('GSM'))).toBe('connected');
  });

  it('BSC Link 连接状态映射为本地化文案 key', () => {
    expect(bscLinkStatusMessageIdForDevice(device('GSM'))).toBe('status.connected');
  });
});
