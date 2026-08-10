import { createRef } from 'react';
import { act, render } from '@testing-library/react';
import { IntlProvider } from 'react-intl';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type {
  GeofenceMapDefinition,
  GeofencePolygonGeometry,
} from '@core/types/geofence';
import type { GISMapRef } from '@core/types/map';

const {
  useOLMapMock,
  useGeofenceLayerMock,
  startMeasure,
  stopMeasure,
  startPolygonDraw,
  stopGeofenceDraw,
  fitGeofence,
} = vi.hoisted(() => ({
  useOLMapMock: vi.fn(),
  useGeofenceLayerMock: vi.fn(),
  startMeasure: vi.fn(),
  stopMeasure: vi.fn(),
  startPolygonDraw: vi.fn(),
  stopGeofenceDraw: vi.fn(),
  fitGeofence: vi.fn(() => true),
}));

vi.mock('./useOLMap', () => ({
  useOLMap: useOLMapMock,
}));

vi.mock('./useGeofenceLayer', () => ({
  useGeofenceLayer: useGeofenceLayerMock,
}));

vi.mock('@/hooks/useThemeToken', () => ({
  useThemeToken: () => ({
    colorBgLayout: '#fff',
    colorPrimary: '#1677ff',
  }),
}));

vi.mock('./MapPopup', () => ({ default: () => null }));
vi.mock('./MapControls', () => ({ default: () => null }));
vi.mock('./MapStatsPanel', () => ({ default: () => null }));

import GISMap from './index';

const polygonGeometry: GeofencePolygonGeometry = {
  type: 'Polygon',
  coordinates: [
    [
      [120.1, 30.1],
      [120.2, 30.1],
      [120.2, 30.2],
      [120.1, 30.1],
    ],
  ],
};

const item: GeofenceMapDefinition = {
  definition: {
    id: 'geofence-1',
    name: '杭州保护区',
    carrier: 'cmcc',
    ruleType: 'polygon_allow_zone',
    status: 'enabled',
    currentVersionId: 'version-1',
    createdBy: 'user-1',
    updatedBy: 'user-1',
    createdAt: '2026-07-31T08:00:00Z',
    updatedAt: '2026-07-31T08:00:00Z',
  },
  currentVersion: {
    id: 'version-1',
    geofenceId: 'geofence-1',
    version: 1,
    status: 'published',
    geometry: polygonGeometry,
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
  },
};

beforeEach(() => {
  vi.clearAllMocks();
  useOLMapMock.mockReturnValue({
    mapRef: { current: null },
    mapInstanceRef: { current: null },
    updateDevices: vi.fn(),
    clearDevices: vi.fn(),
    getViewport: vi.fn(() => null),
    flyTo: vi.fn(),
    fitBounds: vi.fn(),
    clearHighlight: vi.fn(),
    isReady: true,
    updateSize: vi.fn(),
    highlightAndSpiderfyIfNeeded: vi.fn(),
    metadata: null,
    setTileConcurrency: vi.fn(),
    startMeasure,
    stopMeasure,
    updateAntennaSectors: vi.fn(),
  });
  useGeofenceLayerMock.mockReturnValue({
    startPolygonDraw,
    stopGeofenceDraw,
    fitGeofence,
    isGeofenceDrawing: false,
  });
});

describe('GISMap geofence integration', () => {
  it('passes geofence state and callbacks into the layer hook', () => {
    const onGeofenceClick = vi.fn();
    const onGeofenceDrawComplete = vi.fn();

    render(
      <IntlProvider locale="zh-CN" messages={{}}>
        <GISMap
          geofences={[item]}
          selectedGeofenceId="geofence-1"
          onGeofenceClick={onGeofenceClick}
          onGeofenceDrawComplete={onGeofenceDrawComplete}
          showControls={false}
          showStats={false}
        />
      </IntlProvider>,
    );

    const options = useGeofenceLayerMock.mock.calls[0][0];
    expect(options.items).toEqual([item]);
    expect(options.selectedId).toBe('geofence-1');
    options.onSelect(item);
    options.onDrawComplete(polygonGeometry);
    expect(onGeofenceClick).toHaveBeenCalledWith(item);
    expect(onGeofenceDrawComplete).toHaveBeenCalledWith(
      polygonGeometry,
    );
  });

  it('exposes mutually exclusive measure and geofence draw methods', () => {
    const ref = createRef<GISMapRef>();
    render(
      <IntlProvider locale="zh-CN" messages={{}}>
        <GISMap
          ref={ref}
          showControls={false}
          showStats={false}
        />
      </IntlProvider>,
    );

    act(() => ref.current?.startGeofencePolygonDraw());
    expect(stopMeasure).toHaveBeenCalledBefore(startPolygonDraw);

    act(() => ref.current?.startMeasure());
    expect(stopGeofenceDraw).toHaveBeenCalledBefore(startMeasure);

    expect(ref.current?.fitGeofence('geofence-1')).toBe(true);
    act(() => ref.current?.stopGeofenceDraw());
    expect(stopGeofenceDraw).toHaveBeenCalled();
  });
});
