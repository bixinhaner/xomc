import { useCallback, useEffect, useRef, useState } from 'react';
import type React from 'react';
import Feature from 'ol/Feature';
import type Map from 'ol/Map';
import Draw from 'ol/interaction/Draw';
import VectorLayer from 'ol/layer/Vector';
import VectorSource from 'ol/source/Vector';
import type {
  GeofenceMapDefinition,
  GeofencePolygonGeometry,
} from '@core/types/geofence';
import {
  createGeofenceFeatures,
  geofenceFeatureExtent,
  geofenceFeatureStyle,
  polygonGeometryToGeofence,
  type GeofenceFeature,
} from './geofenceLayerModel';

interface GeofenceLayerCallbacks {
  onSelect?: (item: GeofenceMapDefinition) => void;
  onDrawComplete?: (
    geometry: GeofencePolygonGeometry,
  ) => void;
  onDrawingChange?: (drawing: boolean) => void;
}

export interface GeofenceLayerController {
  update(
    items: GeofenceMapDefinition[],
    selectedId: string | undefined,
  ): void;
  startPolygonDraw(): void;
  stopGeofenceDraw(): void;
  fitGeofence(id: string): boolean;
  dispose(): void;
}

export function createGeofenceLayerController(
  map: Map,
  callbacks: GeofenceLayerCallbacks,
): GeofenceLayerController {
  const source = new VectorSource();
  const layer = new VectorLayer({
    source,
    style: (feature) =>
      feature instanceof Feature
        ? geofenceFeatureStyle(feature as GeofenceFeature)
        : undefined,
    zIndex: 6,
  });
  let draw: Draw | null = null;
  let draftFeature: Feature | null = null;

  map.addLayer(layer);

  const handleSingleClick = (event: {
    pixel: number[];
  }) => {
    if (draw) return;
    const feature = map.getFeaturesAtPixel(event.pixel, {
      layerFilter: (candidate) => candidate === layer,
    })[0];
    if (!(feature instanceof Feature)) return;
    const item = feature.get(
      'geofenceItem',
    ) as GeofenceMapDefinition | undefined;
    if (item) callbacks.onSelect?.(item);
  };
  map.on('singleclick', handleSingleClick);

  const stopGeofenceDraw = () => {
    if (draw) {
      try {
        draw.abortDrawing();
      } catch {
        // Draw may already have completed.
      }
      map.removeInteraction(draw);
      draw = null;
    }
    if (draftFeature) {
      source.removeFeature(draftFeature);
      draftFeature = null;
    }
    callbacks.onDrawingChange?.(false);
  };

  return {
    update(items, selectedId) {
      if (draftFeature) {
        source.removeFeature(draftFeature);
        draftFeature = null;
      }
      source.clear();
      const features = createGeofenceFeatures(items);
      for (const feature of features) {
        feature.set(
          'geofenceSelected',
          feature.get('geofenceId') === selectedId,
          true,
        );
      }
      source.addFeatures(features);
      map.render();
    },

    startPolygonDraw() {
      stopGeofenceDraw();
      draw = new Draw({
        source,
        type: 'Polygon',
      });
      const activeDraw = draw;
      activeDraw.on('drawend', (event) => {
        const geometry = event.feature.getGeometry();
        if (!geometry || geometry.getType() !== 'Polygon') return;
        draftFeature = event.feature;
        draftFeature.setProperties(
          {
            geofenceId: '__drawing__',
            geofenceStatus: 'draft',
            geofenceRuleType: 'polygon_allow_zone',
            geofenceSelected: true,
          },
          true,
        );
        callbacks.onDrawComplete?.(
          polygonGeometryToGeofence(
            geometry as import('ol/geom/Polygon').default,
          ),
        );
        map.removeInteraction(activeDraw);
        if (draw === activeDraw) draw = null;
        callbacks.onDrawingChange?.(false);
        map.render();
      });
      map.addInteraction(activeDraw);
      callbacks.onDrawingChange?.(true);
    },

    stopGeofenceDraw,

    fitGeofence(id) {
      const feature = source.getFeatureById(`geofence:${id}`);
      const extent = geofenceFeatureExtent(
        feature as GeofenceFeature | null,
      );
      if (!extent) return false;
      map.getView().fit(extent, {
        padding: [64, 64, 64, 64],
        duration: 400,
        maxZoom: 16,
      });
      return true;
    },

    dispose() {
      stopGeofenceDraw();
      map.un('singleclick', handleSingleClick);
      source.clear();
      map.removeLayer(layer);
    },
  };
}

interface UseGeofenceLayerOptions
  extends Omit<GeofenceLayerCallbacks, 'onDrawingChange'> {
  mapInstanceRef: React.MutableRefObject<Map | null>;
  isReady: boolean;
  items: GeofenceMapDefinition[];
  selectedId?: string;
}

export function useGeofenceLayer({
  mapInstanceRef,
  isReady,
  items,
  selectedId,
  onSelect,
  onDrawComplete,
}: UseGeofenceLayerOptions) {
  const controllerRef = useRef<GeofenceLayerController | null>(
    null,
  );
  const callbackRef = useRef({ onSelect, onDrawComplete });
  const [isGeofenceDrawing, setIsGeofenceDrawing] =
    useState(false);

  callbackRef.current = { onSelect, onDrawComplete };

  useEffect(() => {
    const map = mapInstanceRef.current;
    if (!isReady || !map) return;
    const controller = createGeofenceLayerController(map, {
      onSelect: (item) => callbackRef.current.onSelect?.(item),
      onDrawComplete: (geometry) =>
        callbackRef.current.onDrawComplete?.(geometry),
      onDrawingChange: setIsGeofenceDrawing,
    });
    controllerRef.current = controller;
    return () => {
      controller.dispose();
      if (controllerRef.current === controller) {
        controllerRef.current = null;
      }
    };
  }, [isReady, mapInstanceRef]);

  useEffect(() => {
    controllerRef.current?.update(items, selectedId);
  }, [items, selectedId]);

  const startPolygonDraw = useCallback(() => {
    controllerRef.current?.startPolygonDraw();
  }, []);

  const stopGeofenceDraw = useCallback(() => {
    controllerRef.current?.stopGeofenceDraw();
  }, []);

  const fitGeofence = useCallback((id: string) => {
    return controllerRef.current?.fitGeofence(id) ?? false;
  }, []);

  return {
    startPolygonDraw,
    stopGeofenceDraw,
    fitGeofence,
    isGeofenceDrawing,
  };
}
