export type DeviceRadioField = 'pci' | 'tac' | 'band' | 'dlEarfcn' | 'ulEarfcn' | 'txPower';

interface RadioFieldDevice {
  networkType: string;
  productClass: string;
}

function normalizedProductClass(device: RadioFieldDevice): string {
  return String(device.productClass ?? '').trim().toUpperCase();
}

export function supportsOwnRF(device: RadioFieldDevice): boolean {
  return normalizedProductClass(device) !== 'FAP/PGSM';
}

export function formatDeviceRFStatus(device: RadioFieldDevice, value: unknown): string {
  if (!supportsOwnRF(device)) return '-';
  return value == null ? '' : String(value);
}

export function formatDeviceRadioField(
  device: RadioFieldDevice,
  field: DeviceRadioField,
  value: unknown,
): string {
  const productClass = normalizedProductClass(device);
  if (productClass === 'FAP/PGSM') return '-';
  if (productClass === 'FAP/BTS' && field !== 'txPower') return '-';

  if (
    device.networkType === 'GSM'
    && (field === 'pci' || field === 'tac' || field === 'dlEarfcn' || field === 'ulEarfcn')
  ) {
    return '-';
  }

  return value == null ? '' : String(value);
}
