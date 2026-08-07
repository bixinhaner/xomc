export type DeviceRadioField = 'pci' | 'tac' | 'band' | 'dlEarfcn' | 'ulEarfcn' | 'txPower';

interface RadioFieldDevice {
  networkType: string;
  productClass: string;
  arfcn?: unknown;
}

function normalizedProductClass(device: RadioFieldDevice): string {
  return String(device.productClass ?? '').trim().toUpperCase();
}

function firstNumericValue(value: unknown): number | null {
  if (typeof value === 'number' && Number.isFinite(value)) return value;
  const raw = String(value ?? '').trim();
  if (!raw) return null;
  for (const segment of raw.split(/[,/\s]+/)) {
    const parsed = Number(segment);
    if (Number.isFinite(parsed)) return parsed;
  }
  return null;
}

function isBandwidthLikeGsmBand(value: unknown): boolean {
  return /^(?:GSM\s*)?200\s*K?$/i.test(String(value ?? '').trim());
}

function formatGsmBandFromArfcn(arfcn: unknown): string | null {
  const value = firstNumericValue(arfcn);
  if (value == null) return null;

  if ((value >= 1 && value <= 124) || (value >= 975 && value <= 1023)) return 'GSM900';
  if (value >= 128 && value <= 251) return 'GSM850';
  if (value >= 259 && value <= 293) return 'GSM450';
  if (value >= 306 && value <= 340) return 'GSM480';
  if (value >= 512 && value <= 885) return 'DCS1800';
  if (value >= 512 && value <= 810) return 'PCS1900';
  return null;
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

  if (device.networkType === 'GSM' && field === 'band') {
    if (isBandwidthLikeGsmBand(value)) {
      return formatGsmBandFromArfcn(device.arfcn) ?? '-';
    }
    return value == null ? '' : String(value);
  }

  return value == null ? '' : String(value);
}
