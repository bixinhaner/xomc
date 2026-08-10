import Feature from 'ol/Feature';
import Point from 'ol/geom/Point';
import Polygon, { circular } from 'ol/geom/Polygon';
import { fromLonLat } from 'ol/proj';
import CircleStyle from 'ol/style/Circle';
import Fill from 'ol/style/Fill';
import Stroke from 'ol/style/Stroke';
import Style from 'ol/style/Style';
import type { Extent } from 'ol/extent';
import type {
  GeofenceDefinitionStatus,
  GeofenceMapDefinition,
  GeofencePolygonGeometry,
} from '@core/types/geofence';

export type GeofenceFeature = Feature<Polygon>;

const POLYGON_COLORS: Record<
  Exclude<GeofenceDefinitionStatus, 'archived'>,
  {
    stroke: string;
    fill: string;
    lineDash?: number[];
  }
> = {
  draft: {
    stroke: '#8c8c8c',
    fill: 'rgba(140, 140, 140, 0.06)',
    lineDash: [8, 6],
  },
  enabled: {
    stroke: '#1677ff',
    fill: 'rgba(22, 119, 255, 0.12)',
  },
  disabled: {
    stroke: '#8c8c8c',
    fill: 'rgba(140, 140, 140, 0.08)',
  },
};

function projectedGeometry(
  item: GeofenceMapDefinition,
): {
  geometry: Polygon;
  center?: number[];
} | null {
  const snapshot = item.currentVersion?.geometry;
  if (!snapshot) return null;
  if (snapshot.type === 'Polygon') {
    const geometry = new Polygon(snapshot.coordinates);
    geometry.transform('EPSG:4326', 'EPSG:3857');
    return { geometry };
  }
  const geometry = circular(
    snapshot.center,
    snapshot.radiusMeters,
    64,
  );
  geometry.transform('EPSG:4326', 'EPSG:3857');
  return {
    geometry,
    center: fromLonLat(snapshot.center),
  };
}

export function createGeofenceFeature(
  item: GeofenceMapDefinition,
): GeofenceFeature | null {
  if (
    item.definition.status === 'archived' ||
    !item.currentVersion
  ) {
    return null;
  }
  const projected = projectedGeometry(item);
  if (!projected) return null;
  const feature = new Feature<Polygon>({
    geometry: projected.geometry,
  });
  feature.setId(`geofence:${item.definition.id}`);
  feature.setProperties(
    {
      geofenceId: item.definition.id,
      geofenceItem: item,
      geofenceStatus: item.definition.status,
      geofenceRuleType: item.definition.ruleType,
      geofenceCenter: projected.center,
      geofenceSelected: false,
    },
    true,
  );
  return feature;
}

export function createGeofenceFeatures(
  items: GeofenceMapDefinition[],
): GeofenceFeature[] {
  return items.flatMap((item) => {
    const feature = createGeofenceFeature(item);
    return feature ? [feature] : [];
  });
}

function shapeStyle(
  strokeColor: string,
  fillColor: string,
  lineDash: number[] | undefined,
  selected: boolean,
) {
  return new Style({
    stroke: new Stroke({
      color: strokeColor,
      width: selected ? 5 : 2,
      lineDash,
    }),
    fill: new Fill({ color: fillColor }),
  });
}

export function geofenceFeatureStyle(
  feature: GeofenceFeature,
): Style | Style[] {
  const selected = feature.get('geofenceSelected') === true;
  if (feature.get('geofenceRuleType') === 'baseline_radius') {
    const center = feature.get('geofenceCenter') as
      | number[]
      | undefined;
    const boundary = shapeStyle(
      '#722ed1',
      selected
        ? 'rgba(114, 46, 209, 0.24)'
        : 'rgba(114, 46, 209, 0.12)',
      undefined,
      selected,
    );
    if (!center) return boundary;
    return [
      boundary,
      new Style({
        geometry: new Point(center),
        image: new CircleStyle({
          radius: selected ? 6 : 5,
          fill: new Fill({ color: '#722ed1' }),
          stroke: new Stroke({
            color: '#ffffff',
            width: 2,
          }),
        }),
      }),
    ];
  }

  const status = feature.get(
    'geofenceStatus',
  ) as Exclude<GeofenceDefinitionStatus, 'archived'>;
  const colors = POLYGON_COLORS[status] ?? POLYGON_COLORS.draft;
  return shapeStyle(
    colors.stroke,
    selected
      ? colors.fill.replace(/0\.\d+\)/, '0.24)')
      : colors.fill,
    colors.lineDash,
    selected,
  );
}

export function polygonGeometryToGeofence(
  geometry: Polygon,
): GeofencePolygonGeometry {
  const wgs84 = geometry.clone();
  wgs84.transform('EPSG:3857', 'EPSG:4326');
  const rings = wgs84
    .getCoordinates()
    .slice(0, 1)
    .map((ring) =>
      ring.map((coordinate) => [
        coordinate[0],
        coordinate[1],
      ]),
    );
  const ring = rings[0] ?? [];
  if (ring.length > 0) {
    const first = ring[0];
    const last = ring[ring.length - 1];
    if (first[0] !== last[0] || first[1] !== last[1]) {
      ring.push([...first]);
    }
  }
  return {
    type: 'Polygon',
    coordinates: [ring],
  };
}

export function geofenceFeatureExtent(
  feature: GeofenceFeature | null,
): Extent | null {
  const geometry = feature?.getGeometry();
  return geometry ? geometry.getExtent() : null;
}
