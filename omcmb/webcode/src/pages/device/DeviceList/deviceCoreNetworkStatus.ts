import type { Device } from '@core/types/device';

export function mmeStatusForDevice(device: Device): string {
  return device.networkType === 'eNB' ? device.mmeStatus : '';
}

export function amfStatusForDevice(device: Device): string {
  return device.networkType === 'gNB' ? device.amfStatus : '';
}

export function bscLinkStatusForDevice(device: Device): string {
  return device.networkType === 'GSM' ? device.bscLinkStatus : '';
}
