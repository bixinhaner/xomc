import { act, renderHook } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import Feature from 'ol/Feature';
import Polygon from 'ol/geom/Polygon';
import { fromLonLat } from 'ol/proj';
import type Map from 'ol/Map';
import type VectorLayer from 'ol/layer/Vector';
import type VectorSource from 'ol/source/Vector';
import type {
  GeofenceMapDefinition,
  GeofencePolygonGeometry,
  GeofenceVersion,
} from '@core/types/geofence';
import {
  createGeofenceLayerController,
  useGeofenceLayer,
} from './useGeofenceLayer';

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

const version: GeofenceVersion = {
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

const item: GeofenceMapDefinition = {
  definition,
  currentVersion: version,
};

function mapHarness() {
  const layers: unknown[] = [];
  const interactions: unknown[] = [];
  const handlers = new Map<string, (event: unknown) => void>();
  const fit = vi.fn();
  let hitFeatures: Feature[] = [];
  const map = {
    addLayer: vi.fn((layer: unknown) => layers.push(layer)),
    removeLayer: vi.fn((layer: unknown) => {
      const index = layers.indexOf(layer);
      if (index >= 0) layers.splice(index, 1);
    }),
    addInteraction: vi.fn((interaction: unknown) =>
      interactions.push(interaction),
    ),
    removeInteraction: vi.fn((interaction: unknown) => {
      const index = interactions.indexOf(interaction);
      if (index >= 0) interactions.splice(index, 1);
    }),
    on: vi.fn(
      (type: string, handler: (event: unknown) => void) =>
        handlers.set(type, handler),
    ),
    un: vi.fn((type: string) => handlers.delete(type)),
    getFeaturesAtPixel: vi.fn(() => hitFeatures),
    getView: vi.fn(() => ({ fit })),
    render: vi.fn(),
  };
  return {
    map: map as unknown as Map,
    layers,
    interactions,
    handlers,
    fit,
    setHitFeatures(features: Feature[]) {
      hitFeatures = features;
    },
  };
}

describe('geofence layer controller', () => {
  it('keeps every visible fence and marks only the current selection', () => {
    const harness = mapHarness();
    const controller = createGeofenceLayerController(
      harness.map,
      {},
    );
    const secondItem: GeofenceMapDefinition = {
      definition: {
        ...definition,
        id: 'geofence-2',
        name: '杭州第二保护区',
        currentVersionId: 'version-2',
      },
      currentVersion: {
        ...version,
        id: 'version-2',
        geofenceId: 'geofence-2',
      },
    };

    controller.update([item, secondItem], 'geofence-2');

    const layer = harness.layers[0] as VectorLayer<VectorSource>;
    const features = layer.getSource()?.getFeatures() ?? [];
    expect(features).toHaveLength(2);
    expect(
      features.map((feature) => [
        feature.get('geofenceId'),
        feature.get('geofenceSelected'),
      ]),
    ).toEqual([
      ['geofence-1', false],
      ['geofence-2', true],
    ]);
  });

  it('adds one layer, replaces source items and updates selection', () => {
    const harness = mapHarness();
    const controller = createGeofenceLayerController(
      harness.map,
      {},
    );

    controller.update([item], 'geofence-1');
    controller.update([item], undefined);

    expect(harness.layers).toHaveLength(1);
    const layer = harness.layers[0] as VectorLayer<VectorSource>;
    const features = layer.getSource()?.getFeatures() ?? [];
    expect(features).toHaveLength(1);
    expect(features[0].get('geofenceSelected')).toBe(false);
  });

  it('returns the clicked business item and fits its extent', () => {
    const harness = mapHarness();
    const onSelect = vi.fn();
    const controller = createGeofenceLayerController(
      harness.map,
      { onSelect },
    );
    controller.update([item], undefined);
    const layer = harness.layers[0] as VectorLayer<VectorSource>;
    const feature = layer.getSource()?.getFeatures()[0] as Feature;
    harness.setHitFeatures([feature]);

    harness.handlers.get('singleclick')?.({
      pixel: [10, 20],
    });
    const fitted = controller.fitGeofence('geofence-1');

    expect(onSelect).toHaveBeenCalledWith(item);
    expect(fitted).toBe(true);
    expect(harness.fit).toHaveBeenCalledWith(
      expect.any(Array),
      expect.objectContaining({
        padding: [64, 64, 64, 64],
      }),
    );
  });

  it('replaces an active draw, emits WGS84 geometry and cleans up', () => {
    const harness = mapHarness();
    const onDrawComplete = vi.fn();
    const onDrawingChange = vi.fn();
    const controller = createGeofenceLayerController(
      harness.map,
      { onDrawComplete, onDrawingChange },
    );

    controller.startPolygonDraw();
    const firstDraw = harness.interactions[0] as {
      dispatchEvent: (event: unknown) => void;
    };
    controller.startPolygonDraw();
    expect(harness.map.removeInteraction).toHaveBeenCalledWith(
      firstDraw,
    );

    const activeDraw = harness.interactions[0] as {
      dispatchEvent: (event: unknown) => void;
    };
    const feature = new Feature(
      new Polygon([
        [
          fromLonLat([120.1, 30.1]),
          fromLonLat([120.2, 30.1]),
          fromLonLat([120.2, 30.2]),
          fromLonLat([120.1, 30.1]),
        ],
      ]),
    );
    activeDraw.dispatchEvent({ type: 'drawend', feature });

    const geometry = onDrawComplete.mock
      .calls[0][0] as GeofencePolygonGeometry;
    expect(geometry.coordinates[0][0][0]).toBeCloseTo(
      120.1,
      6,
    );
    expect(onDrawingChange).toHaveBeenLastCalledWith(false);

    controller.dispose();
    expect(harness.layers).toHaveLength(0);
    expect(harness.handlers.has('singleclick')).toBe(false);
    expect(harness.interactions).toHaveLength(0);
  });
});

describe('useGeofenceLayer', () => {
  it('does not add a layer before ready and disposes it on unmount', () => {
    const harness = mapHarness();
    const mapInstanceRef = { current: harness.map };
    const { rerender, unmount } = renderHook(
      ({ ready }) =>
        useGeofenceLayer({
          mapInstanceRef,
          isReady: ready,
          items: [item],
        }),
      { initialProps: { ready: false } },
    );

    expect(harness.layers).toHaveLength(0);
    act(() => rerender({ ready: true }));
    expect(harness.layers).toHaveLength(1);
    unmount();
    expect(harness.layers).toHaveLength(0);
  });
});
