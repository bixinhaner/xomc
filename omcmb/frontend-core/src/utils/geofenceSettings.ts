import type {
  GeofenceRuntimeMode,
  GeofenceSettings,
  UpdateGeofenceSettingsInput,
} from '../types/geofence';

export type GeofenceSettingsValidationError =
  | { code: 'carriers_required' }
  | { code: 'duplicate_carrier'; carrier: string }
  | { code: 'radius_out_of_range'; carrier: string };

export function geofenceSettingsToInput(
  settings: GeofenceSettings,
): UpdateGeofenceSettingsInput {
  return {
    systemMode: settings.systemMode,
    carriers: settings.carriers.map((setting) => ({
      carrier: setting.carrier,
      mode: setting.mode,
      defaultBaselineRadiusMeters:
        setting.defaultBaselineRadiusMeters,
    })),
  };
}

export function effectiveGeofenceMode(
  systemMode: GeofenceRuntimeMode,
  carrierMode: GeofenceRuntimeMode,
): GeofenceRuntimeMode {
  const rank: Record<GeofenceRuntimeMode, number> = {
    off: 0,
    observe: 1,
    enforce: 2,
  };
  return rank[systemMode] <= rank[carrierMode]
    ? systemMode
    : carrierMode;
}

export function validateGeofenceSettingsInput(
  input: UpdateGeofenceSettingsInput,
): GeofenceSettingsValidationError[] {
  const errors: GeofenceSettingsValidationError[] = [];
  if (input.carriers.length === 0) {
    errors.push({ code: 'carriers_required' });
    return errors;
  }
  const carriers = new Set<string>();
  for (const setting of input.carriers) {
    const carrier = setting.carrier.trim().toLowerCase();
    if (carriers.has(carrier)) {
      errors.push({ code: 'duplicate_carrier', carrier });
    } else {
      carriers.add(carrier);
    }
    if (
      setting.defaultBaselineRadiusMeters <= 0 ||
      setting.defaultBaselineRadiusMeters > 50000
    ) {
      errors.push({
        code: 'radius_out_of_range',
        carrier,
      });
    }
  }
  return errors;
}
