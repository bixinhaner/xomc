import type { GeofenceMapFilter } from '@core/types/geofence';
import type { MapViewport } from '@core/types/map';

export function buildGeofenceMapQuery(
  toolEnabled: boolean,
  viewport: MapViewport | null,
): {
  enabled: boolean;
  filter: GeofenceMapFilter;
} {
  const bounds = viewport?.bounds;
  return {
    enabled: toolEnabled && Boolean(bounds),
    filter: bounds
      ? {
          status: 'enabled',
          bounds: {
            minLng: bounds.minLng,
            maxLng: bounds.maxLng,
            minLat: bounds.minLat,
            maxLat: bounds.maxLat,
          },
        }
      : {},
  };
}
