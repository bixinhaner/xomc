export type DeviceRadioField = 'pci' | 'tac' | 'band' | 'dlEarfcn' | 'ulEarfcn' | 'txPower';

interface RadioFieldDevice {
  networkType: string;
  productClass: string;
}

export function formatDeviceRadioField(
  device: RadioFieldDevice,
  field: DeviceRadioField,
  value: unknown,
): string {
  const productClass = device.productClass.trim().toUpperCase();
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
