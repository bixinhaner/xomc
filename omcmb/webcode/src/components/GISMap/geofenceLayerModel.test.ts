import { describe, expect, it } from 'vitest';
import Feature from 'ol/Feature';
import Polygon from 'ol/geom/Polygon';
import { fromLonLat, toLonLat } from 'ol/proj';
import { getDistance } from 'ol/sphere';
import type {
  GeofenceMapDefinition,
  GeofenceVersion,
} from '@core/types/geofence';
import {
  createGeofenceFeature,
  createGeofenceFeatures,
  geofenceFeatureExtent,
  geofenceFeatureStyle,
  polygonGeometryToGeofence,
} from './geofenceLayerModel';

const definition = {
  id: 'geofence-1',
  name: '杭州保护区',
  carrier: 'cmcc',
  ruleType: 'polygon_allow_zone' as const,
  status: 'enabled' as const,
  currentVersionId: 'version-1',
  createdBy: 'user-1',
  updatedBy: 'user-1',
  createdAt: '2026-07-31T08:00:00Z',
  updatedAt: '2026-07-31T08:00:00Z',
};

const polygonVersion: GeofenceVersion = {
  id: 'version-1',
  geofenceId: 'geofence-1',
  version: 1,
  status: 'published',
  geometry: {
    type: 'Polygon',
    coordinates: [
      [
        [120.1, 30.1],
        [120.2, 30.1],
        [120.2, 30.2],
        [120.1, 30.1],
      ],
    ],
  },
  boundingBox: {
    minLongitude: 120.1,
    minLatitude: 30.1,
    maxLongitude: 120.2,
    maxLatitude: 30.2,
  },
  policy: { exitAction: 'notify_only' },
  createdBy: 'user-1',
  publishedBy: 'user-1',
  createdAt: '2026-07-31T08:00:00Z',
  publishedAt: '2026-07-31T08:00:00Z',
};

function mapItem(
  overrides: Partial<GeofenceMapDefinition> = {},
): GeofenceMapDefinition {
  return {
    definition,
    currentVersion: polygonVersion,
    ...overrides,
  };
}

describe('geofence OpenLayers geometry adapter', () => {
  it('projects a WGS84 polygon and keeps the stable business identity', () => {
    const item = mapItem();
    const feature = createGeofenceFeature(item);

    expect(feature).not.toBeNull();
    expect(feature?.getId()).toBe('geofence:geofence-1');
    expect(feature?.get('geofenceId')).toBe('geofence-1');
    expect(feature?.get('geofenceItem')).toBe(item);

    const geometry = feature?.getGeometry();
    expect(geometry).toBeInstanceOf(Polygon);
    const firstProjected = (geometry as Polygon)
      .getCoordinates()[0][0];
    const firstWGS84 = toLonLat(firstProjected);
    expect(firstWGS84[0]).toBeCloseTo(120.1, 6);
    expect(firstWGS84[1]).toBeCloseTo(30.1, 6);
  });

  it('uses a geodesic polygon for baseline radius', () => {
    const center: [number, number] = [120.16, 30.16];
    const radiusMeters = 1000;
    const feature = createGeofenceFeature(
      mapItem({
        definition: {
          ...definition,
          id: 'geofence-radius',
          ruleType: 'baseline_radius',
        },
        currentVersion: {
          ...polygonVersion,
          id: 'version-radius',
          geofenceId: 'geofence-radius',
          geometry: {
            type: 'Circle',
            center,
            radiusMeters,
          },
        },
      }),
    );

    const geometry = feature?.getGeometry() as Polygon;
    const edgeWGS84 = toLonLat(geometry.getCoordinates()[0][0]);
    expect(getDistance(center, edgeWGS84)).toBeCloseTo(
      radiusMeters,
      -1,
    );
    const projectedCenter = feature?.get('geofenceCenter');
    const restoredCenter = toLonLat(projectedCenter);
    expect(restoredCenter[0]).toBeCloseTo(center[0], 6);
    expect(restoredCenter[1]).toBeCloseTo(center[1], 6);
  });

  it('skips archived definitions and definitions without a published version', () => {
    expect(
      createGeofenceFeature(
        mapItem({
          definition: {
            ...definition,
            status: 'archived',
          },
        }),
      ),
    ).toBeNull();
    expect(
      createGeofenceFeature(
        mapItem({ currentVersion: null }),
      ),
    ).toBeNull();
    expect(
      createGeofenceFeatures([
        mapItem(),
        mapItem({
          definition: {
            ...definition,
            id: 'archived',
            status: 'archived',
          },
        }),
      ]),
    ).toHaveLength(1);
  });

  it.each([
    ['draft', '#8c8c8c', [8, 6]],
    ['enabled', '#1677ff', undefined],
    ['disabled', '#8c8c8c', undefined],
  ] as const)(
    'uses the expected %s polygon status style',
    (status, color, lineDash) => {
      const feature = createGeofenceFeature(
        mapItem({
          definition: { ...definition, status },
        }),
      ) as Feature<Polygon>;
      const styles = geofenceFeatureStyle(feature);
      const shapeStyle = Array.isArray(styles)
        ? styles[0]
        : styles;

      expect(shapeStyle.getStroke()?.getColor()).toBe(color);
      expect(shapeStyle.getStroke()?.getLineDash()).toEqual(
        lineDash ?? null,
      );
    },
  );

  it('uses a purple shape and center marker for baseline radius', () => {
    const feature = createGeofenceFeature(
      mapItem({
        definition: {
          ...definition,
          ruleType: 'baseline_radius',
        },
        currentVersion: {
          ...polygonVersion,
          geometry: {
            type: 'Circle',
            center: [120.16, 30.16],
            radiusMeters: 1000,
          },
        },
      }),
    ) as Feature<Polygon>;
    const styles = geofenceFeatureStyle(feature);

    expect(Array.isArray(styles)).toBe(true);
    expect(styles).toHaveLength(2);
    expect(styles[0].getStroke()?.getColor()).toBe('#722ed1');
    expect(styles[1].getImage()).not.toBeNull();
  });

  it('converts a drawn projected polygon back to one closed WGS84 ring', () => {
    const geometry = new Polygon([
      [
        fromLonLat([120.1, 30.1]),
        fromLonLat([120.2, 30.1]),
        fromLonLat([120.2, 30.2]),
        fromLonLat([120.1, 30.1]),
      ],
    ]);

    const result = polygonGeometryToGeofence(geometry);

    expect(result.type).toBe('Polygon');
    expect(result.coordinates).toHaveLength(1);
    expect(result.coordinates[0][0]).toEqual(
      result.coordinates[0][result.coordinates[0].length - 1],
    );
    expect(result.coordinates[0][0][0]).toBeCloseTo(120.1, 6);
    expect(result.coordinates[0][0][1]).toBeCloseTo(30.1, 6);
  });

  it('returns the projected feature extent for map fitting', () => {
    const feature = createGeofenceFeature(mapItem());
    const extent = geofenceFeatureExtent(feature);

    expect(extent).not.toBeNull();
    expect(extent?.[0]).toBeLessThan(extent?.[2] ?? 0);
    expect(extent?.[1]).toBeLessThan(extent?.[3] ?? 0);
  });
});
