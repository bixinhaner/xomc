import type { Device } from '@core/types/device';
import type { SoftwareVersion } from '@core/mock/data/software';

export interface SoftwareVersionOption {
  label: string;
  value: string;
}

function toOptions(versions: Array<string | undefined>): SoftwareVersionOption[] {
  return Array.from(
    new Set(versions.map((version) => version?.trim() ?? '').filter(Boolean)),
  ).map((version) => ({ label: version, value: version }));
}

export function toActualSoftwareVersionOptions(devices: readonly Device[]): SoftwareVersionOption[] {
  return toOptions(
    devices.map((device) => device.softwareVersion || device.firmwareVersion),
  );
}

export function toFirmwareVersionOptions(
  ...firmwareGroups: Array<readonly SoftwareVersion[]>
): SoftwareVersionOption[] {
  return toOptions(
    firmwareGroups.flatMap((firmware) => firmware.map((item) => item.versionCode)),
  );
}
