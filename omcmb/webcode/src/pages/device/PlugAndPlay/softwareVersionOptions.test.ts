import { describe, expect, it } from 'vitest';
import type { Device } from '@core/types/device';
import type { SoftwareVersion } from '@core/mock/data/software';
import {
  toActualSoftwareVersionOptions,
  toFirmwareVersionOptions,
} from './softwareVersionOptions';

describe('software version options', () => {
  it('lists actual versions reported by devices without demo or duplicate values', () => {
    const devices = [
      { softwareVersion: ' V1.0.0 ' },
      { softwareVersion: 'V1.0.0' },
      { firmwareVersion: 'V1.1.0' },
      { softwareVersion: '' },
    ] as Device[];

    expect(toActualSoftwareVersionOptions(devices)).toEqual([
      { label: 'V1.0.0', value: 'V1.0.0' },
      { label: 'V1.1.0', value: 'V1.1.0' },
    ]);
  });

  it('merges current-product firmware packages for the target version selector', () => {
    const byProduct = [
      { id: 'fw-1', versionCode: 'V2.0.0' },
      { id: 'fw-2', versionCode: 'V2.1.0' },
    ] as SoftwareVersion[];
    const byLegacyClass = [
      { id: 'fw-2', versionCode: 'V2.1.0' },
      { id: 'fw-3', versionCode: ' V2.2.0 ' },
    ] as SoftwareVersion[];

    expect(toFirmwareVersionOptions(byProduct, byLegacyClass)).toEqual([
      { label: 'V2.0.0', value: 'V2.0.0' },
      { label: 'V2.1.0', value: 'V2.1.0' },
      { label: 'V2.2.0', value: 'V2.2.0' },
    ]);
  });
});
