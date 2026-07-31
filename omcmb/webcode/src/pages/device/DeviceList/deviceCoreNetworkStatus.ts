import type { Device } from '@core/types/device';
import { connectionStatusMessageId, normalizeConnectionStatus } from '@core/utils/connectionStatus';

export function mmeStatusForDevice(device: Device): string {
  if (device.networkType !== 'eNB') return '';
  // 兼容升级前已落库的 partial：设备级口径是任一 MME 已连接即 connected。
  return normalizeConnectionStatus(device.mmeStatus, true);
}

export function mmeStatusMessageIdForDevice(device: Device): string {
  return device.networkType === 'eNB' ? connectionStatusMessageId(device.mmeStatus, true) : '';
}

export function amfStatusForDevice(device: Device): string {
  return device.networkType === 'gNB' ? normalizeConnectionStatus(device.amfStatus) : '';
}

export function amfStatusMessageIdForDevice(device: Device): string {
  return device.networkType === 'gNB' ? connectionStatusMessageId(device.amfStatus) : '';
}

export function bscLinkStatusForDevice(device: Device): string {
  return device.networkType === 'GSM' ? normalizeConnectionStatus(device.bscLinkStatus) : '';
}

export function bscLinkStatusMessageIdForDevice(device: Device): string {
  return device.networkType === 'GSM' ? connectionStatusMessageId(device.bscLinkStatus) : '';
}
