import type { DeviceParameter, ParameterSchemaItem } from '@core/types/deviceParameter';

export function isNrNetworkType(networkType: string | undefined): boolean {
  return String(networkType ?? '').trim().toLowerCase() === 'nr';
}

interface DeviceTimeModeLabels {
  nrServer: string;
  nrClient: string;
  enable: string;
  disable: string;
}

export function mapDeviceTimeModeLabel(
  value: string,
  fallbackLabel: string,
  networkType: string | undefined,
  labels: DeviceTimeModeLabels,
): string {
  const normalized = String(value ?? '').trim().toLowerCase();

  // 先按明确语义词直出，避免被 0/1/true/false 规则覆盖。
  if (normalized === 'server') return labels.nrServer;
  if (normalized === 'client') return labels.nrClient;

  const isEnabledLike = normalized === '1'
    || normalized === 'true'
    || normalized === 'on'
    || normalized === 'enable'
    || normalized === 'enabled'
    || normalized === 'client';
  const isDisabledLike = normalized === '0'
    || normalized === 'false'
    || normalized === 'off'
    || normalized === 'disable'
    || normalized === 'disabled'
    || normalized === 'server';

  if (!isEnabledLike && !isDisabledLike) {
    return fallbackLabel;
  }

  if (isNrNetworkType(networkType)) {
    // NR 设备真值表：Device.Time.Enable=1 是 Client，0 是 Server。
    return isEnabledLike ? labels.nrClient : labels.nrServer;
  }
  return isEnabledLike ? labels.enable : labels.disable;
}

export function inferDeviceTimeMode(
  rawParameterByPath: Map<string, DeviceParameter>,
  schemaByPath: Map<string, ParameterSchemaItem>,
  networkType: string | undefined,
): string {
  const current = schemaByPath.get('Device.Time.Enable')?.currentValue
    ?? rawParameterByPath.get('Device.Time.Enable')?.parameterValue;
  const normalized = String(current ?? '').trim().toLowerCase();
  if (normalized === '1' || normalized === 'true') return '1';
  if (normalized === '0' || normalized === 'false') return '0';

  const hasNtpServers = ['Device.Time.NTPServer1', 'Device.Time.NTPServer2', 'Device.Time.NTPServer3', 'Device.Time.NTPServer4', 'Device.Time.NTPServer5']
    .some((path) => String(
      schemaByPath.get(path)?.currentValue
        ?? rawParameterByPath.get(path)?.parameterValue
        ?? '',
    ).trim() !== '');

  if (isNrNetworkType(networkType)) {
    // NR 设备真值表：有 NTP Server 配置时更可能是 Client 模式（值=1）。
    return hasNtpServers ? '1' : '0';
  }
  return hasNtpServers ? '1' : '0';
}

export function shouldShowDeviceTimeNtpServerFields(
  networkType: string | undefined,
  modeValue: string | undefined,
): boolean {
  if (!isNrNetworkType(networkType)) return true;

  const normalized = String(modeValue ?? '').trim().toLowerCase();
  return normalized !== '0' && normalized !== 'server';
}
