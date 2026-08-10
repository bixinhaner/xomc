import { describe, expect, it } from 'vitest';
import type {
  GeofenceSettings,
  UpdateGeofenceSettingsInput,
} from '../../types/geofence';
import {
  effectiveGeofenceMode,
  geofenceSettingsToInput,
  validateGeofenceSettingsInput,
} from '../geofenceSettings';

const settings: GeofenceSettings = {
  systemMode: 'observe',
  carriers: [
    {
      carrier: 'cmcc',
      mode: 'observe',
      effectiveMode: 'observe',
      defaultBaselineRadiusMeters: 1000,
      updatedBy: 'user-1',
      updatedAt: '2026-07-31T08:00:00Z',
    },
    {
      carrier: 'ctcc',
      mode: 'off',
      effectiveMode: 'off',
      defaultBaselineRadiusMeters: 800,
      updatedBy: 'user-2',
      updatedAt: '2026-07-31T09:00:00Z',
    },
  ],
};

describe('geofence settings form model', () => {
  it('keeps only editable fields from fetched settings', () => {
    expect(geofenceSettingsToInput(settings)).toEqual({
      systemMode: 'observe',
      carriers: [
        {
          carrier: 'cmcc',
          mode: 'observe',
          defaultBaselineRadiusMeters: 1000,
        },
        {
          carrier: 'ctcc',
          mode: 'off',
          defaultBaselineRadiusMeters: 800,
        },
      ],
    });
  });

  it.each([
    ['off', 'off', 'off'],
    ['observe', 'off', 'off'],
    ['off', 'observe', 'off'],
    ['observe', 'observe', 'observe'],
    ['enforce', 'enforce', 'enforce'],
  ] as const)(
    'derives effective mode from system %s and carrier %s',
    (systemMode, carrierMode, expected) => {
      expect(
        effectiveGeofenceMode(systemMode, carrierMode),
      ).toBe(expected);
    },
  );

  it('accepts enforce settings', () => {
    const input: UpdateGeofenceSettingsInput = {
      systemMode: 'enforce',
      carriers: [
        {
          carrier: 'cmcc',
          mode: 'observe',
          defaultBaselineRadiusMeters: 1000,
        },
      ],
    };

    expect(validateGeofenceSettingsInput(input)).toEqual([]);
  });

  it.each([0, -1, 50001])(
    'rejects baseline radius %s outside the backend contract',
    (radius) => {
      const input = geofenceSettingsToInput(settings);
      input.carriers[0].defaultBaselineRadiusMeters = radius;

      expect(validateGeofenceSettingsInput(input)).toContainEqual({
        code: 'radius_out_of_range',
        carrier: 'cmcc',
      });
    },
  );
});
