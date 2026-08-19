import type { Device } from '@core/types/device';
import { connectionStatusMessageId, normalizeConnectionStatus } from '@core/utils/connectionStatus';

export interface MMEPoolSummary {
  status: string;
  connectedCount: number;
  total: number;
}

const GATEWAY_MME_STATUS_PRODUCTS = new Set(['ENB_DEFAULT_098', 'ENB_DEFAULT_181']);

export function mmePoolSummaryForDevice(device: Device): MMEPoolSummary {
  if (device.networkType !== 'eNB') {
    return { status: '', connectedCount: 0, total: 0 };
  }

  if (GATEWAY_MME_STATUS_PRODUCTS.has((device.productClass ?? '').trim().toUpperCase())) {
    return {
      status: normalizeConnectionStatus(device.mmeStatus, true),
      connectedCount: 0,
      total: 0,
    };
  }

  const observed = (device.mmePool || []).filter((entry) => entry.status.trim() !== '');
  if (observed.length === 0) {
    return {
      status: normalizeConnectionStatus(device.mmeStatus, true),
      connectedCount: 0,
      total: 0,
    };
  }

  const connectedCount = observed.filter((entry) => {
    switch (entry.status.trim().toLowerCase()) {
      case '1':
      case 'true':
      case 'connected':
      case 'active':
      case 'up':
      case 'on':
        return true;
      default:
        return false;
    }
  }).length;
  return {
    status: connectedCount > 0 ? 'connected' : 'disconnected',
    connectedCount,
    total: observed.length,
  };
}

export function mmeStatusForDevice(device: Device): string {
  return mmePoolSummaryForDevice(device).status;
}

export function mmeStatusMessageIdForDevice(device: Device): string {
  return connectionStatusMessageId(mmePoolSummaryForDevice(device).status, true);
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
