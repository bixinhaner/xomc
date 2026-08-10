import { describe, expect, it } from 'vitest';
import type { MapViewport } from '@core/types/map';
import { buildGeofenceMapQuery } from './geofenceViewModel';

const viewport: MapViewport = {
  centerLng: 120.15,
  centerLat: 30.15,
  zoom: 12,
  bounds: {
    minLng: 120,
    maxLng: 121,
    minLat: 30,
    maxLat: 31,
  },
};

describe('GIS geofence view query model', () => {
  it('does not query while the tool is disabled', () => {
    expect(buildGeofenceMapQuery(false, viewport)).toEqual({
      enabled: false,
      filter: {
        status: 'enabled',
        bounds: viewport.bounds,
      },
    });
  });

  it('waits for the first real viewport before querying', () => {
    expect(buildGeofenceMapQuery(true, null)).toEqual({
      enabled: false,
      filter: {},
    });
  });

  it('uses the current GIS viewport bounds when enabled', () => {
    const result = buildGeofenceMapQuery(true, viewport);

    expect(result).toEqual({
      enabled: true,
      filter: {
        status: 'enabled',
        bounds: {
          minLng: 120,
          maxLng: 121,
          minLat: 30,
          maxLat: 31,
        },
      },
    });
    expect(result.filter.bounds).not.toBe(viewport.bounds);
  });
});
