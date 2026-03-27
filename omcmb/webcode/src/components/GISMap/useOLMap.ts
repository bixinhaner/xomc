/**
 * OpenLayers 地图核心 Hook
 * @module components/GISMap/useOLMap
 */

import { useEffect, useRef, useCallback, useState } from 'react';
import Map from 'ol/Map';
import View from 'ol/View';
import TileLayer from 'ol/layer/Tile';
import VectorLayer from 'ol/layer/Vector';
import VectorSource from 'ol/source/Vector';
import Cluster from 'ol/source/Cluster';
import XYZ from 'ol/source/XYZ';
import Feature from 'ol/Feature';
import Point from 'ol/geom/Point';
import { fromLonLat, toLonLat } from 'ol/proj';
import { defaults as defaultControls } from 'ol/control';
import type { MapDevice, MapViewport, MapBounds } from '@/types/map';
import {
  MAP_CONFIG,
  ANIMATION_CONFIG,
  CLUSTER_CONFIG,
} from './constants';
import { deviceStyleFunction, clusterStyleFunction } from './styleUtils';

interface UseOLMapOptions {
  /** 瓦片服务地址（离线模式） */
  tileUrl?: string;
  /** 默认中心点 [lng, lat] */
  center?: [number, number];
  /** 默认缩放级别 */
  zoom?: number;
  /** 最小缩放级别 */
  minZoom?: number;
  /** 最大缩放级别 */
  maxZoom?: number;
  /** 聚合距离 */
  clusterDistance?: number;
  /** 设备点击回调 */
  onDeviceClick?: (device: MapDevice) => void;
  /** 设备悬停回调（包含鼠标位置） */
  onDeviceHover?: (device: MapDevice | null, pixel?: { x: number; y: number }) => void;
  /** 视图变化回调 */
  onViewportChange?: (viewport: MapViewport) => void;
  /** 聚合点击回调 */
  onClusterClick?: (devices: MapDevice[]) => void;
}

interface UseOLMapReturn {
  /** 地图容器 ref */
  mapRef: React.RefObject<HTMLDivElement>;
  /** 地图实例 ref */
  mapInstanceRef: React.MutableRefObject<Map | null>;
  /** 更新设备数据 */
  updateDevices: (devices: MapDevice[]) => void;
  /** 获取当前视图状态 */
  getViewport: () => MapViewport | null;
  /** 飞行到指定位置 */
  flyTo: (lng: number, lat: number, zoom?: number) => void;
  /** 高亮设备 */
  highlightDevice: (deviceId: string) => void;
  /** 取消高亮 */
  clearHighlight: () => void;
  /** 地图是否就绪 */
  isReady: boolean;
  /** 刷新地图尺寸 */
  updateSize: () => void;
  /** 获取当前 zoom 级别 */
  getZoom: () => number;
  /** 适配边界 */
  fitBounds: (bounds: MapBounds) => void;
}

/**
 * OpenLayers 地图 Hook
 */
export function useOLMap(options: UseOLMapOptions = {}): UseOLMapReturn {
  const {
    tileUrl,
    center = MAP_CONFIG.defaultCenter,
    zoom = MAP_CONFIG.defaultZoom,
    minZoom = MAP_CONFIG.minZoom,
    maxZoom = MAP_CONFIG.maxZoom,
    clusterDistance = CLUSTER_CONFIG.distance,
    onDeviceClick,
    onDeviceHover,
    onViewportChange,
    onClusterClick,
  } = options;

  const mapRef = useRef<HTMLDivElement>(null);
  const mapInstanceRef = useRef<Map | null>(null);
  const deviceSourceRef = useRef<VectorSource | null>(null);
  const clusterSourceRef = useRef<Cluster | null>(null);
  const deviceLayerRef = useRef<VectorLayer<VectorSource> | null>(null);
  const highlightFeatureRef = useRef<Feature | null>(null);
  // 水波纹动画定时器
  const pulseAnimationRef = useRef<ReturnType<typeof setInterval> | null>(null);
  // 波纹创建定时器
  const rippleCreateRef = useRef<ReturnType<typeof setInterval> | null>(null);
  // 波纹状态数组
  const rippleWavesRef = useRef<{ radius: number; opacity: number }[]>([]);

  const [isReady, setIsReady] = useState(false);

  // 初始化地图
  useEffect(() => {
    if (!mapRef.current || mapInstanceRef.current) return;

    // 创建瓦片图层（可选：如果提供了 tileUrl 则使用，否则不显示底图）
    // 地图背景使用 CSS 格条纹理，不再使用 OSM 瓦片
    const layers: any[] = [];

    // 只有提供了 tileUrl 才添加瓦片图层
    if (tileUrl) {
      const tileSource = new XYZ({ url: tileUrl });
      const tileLayer = new TileLayer({
        source: tileSource,
        opacity: 0.6, // 半透明，让格条纹理背景显示出来
      });
      layers.push(tileLayer);
    }

    // 创建设备数据源
    deviceSourceRef.current = new VectorSource();

    // 创建聚合数据源
    clusterSourceRef.current = new Cluster({
      source: deviceSourceRef.current,
      distance: clusterDistance,
    });

    // 创建设备图层
    deviceLayerRef.current = new VectorLayer({
      source: clusterSourceRef.current,
      style: clusterStyleFunction,
      zIndex: 10,
    });
    layers.push(deviceLayerRef.current);

    // 创建地图实例
    mapInstanceRef.current = new Map({
      target: mapRef.current,
      layers,
      view: new View({
        center: fromLonLat(center),
        zoom,
        minZoom,
        maxZoom,
      }),
      controls: defaultControls({
        zoom: false,
        attribution: false,
        rotate: false,
      }),
    });

    // 绑定事件
    bindMapEvents(
      mapInstanceRef.current,
      deviceSourceRef.current,
      clusterSourceRef.current,
      {
        onDeviceClick,
        onDeviceHover,
        onViewportChange,
        onClusterClick,
      },
      deviceLayerRef.current
    );

    setIsReady(true);

    // 清理函数
    return () => {
      if (mapInstanceRef.current) {
        mapInstanceRef.current.setTarget(undefined);
        mapInstanceRef.current = null;
      }
      deviceSourceRef.current = null;
      clusterSourceRef.current = null;
      deviceLayerRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []); // 仅在挂载时执行

  // 更新设备数据
  const updateDevices = useCallback((devices: MapDevice[]) => {
    if (!deviceSourceRef.current) return;

    // 清除现有数据
    deviceSourceRef.current.clear();

    // 添加新数据
    const features = devices.map((device) => {
      const feature = new Feature({
        geometry: new Point(fromLonLat([device.lng, device.lat])),
        ...device,
      });
      feature.setId(device.id);
      return feature;
    });

    deviceSourceRef.current.addFeatures(features);
  }, []);

  // 获取当前视图状态
  const getViewport = useCallback((): MapViewport | null => {
    if (!mapInstanceRef.current) return null;

    const map = mapInstanceRef.current;
    const view = map.getView();
    const center = toLonLat(view.getCenter()!);
    const extent = view.calculateExtent(map.getSize());

    // 转换 extent 坐标
    const bottomLeft = toLonLat([extent[0], extent[1]]);
    const topRight = toLonLat([extent[2], extent[3]]);

    return {
      centerLng: center[0],
      centerLat: center[1],
      zoom: view.getZoom()!,
      bounds: {
        minLng: bottomLeft[0],
        maxLng: topRight[0],
        minLat: bottomLeft[1],
        maxLat: topRight[1],
      },
    };
  }, []);

  // 飞行到指定位置
  const flyTo = useCallback((lng: number, lat: number, targetZoom?: number) => {
    if (!mapInstanceRef.current) return;

    const view = mapInstanceRef.current.getView();
    view.animate({
      center: fromLonLat([lng, lat]),
      zoom: targetZoom ?? ANIMATION_CONFIG.highlightZoom,
      duration: ANIMATION_CONFIG.flyDuration,
    });
  }, []);

  // 取消高亮（必须在 highlightDevice 之前定义）
  const clearHighlight = useCallback(() => {
    // 清除波纹动画定时器
    if (pulseAnimationRef.current) {
      clearInterval(pulseAnimationRef.current);
      pulseAnimationRef.current = null;
    }

    // 清除波纹创建定时器
    if (rippleCreateRef.current) {
      clearInterval(rippleCreateRef.current);
      rippleCreateRef.current = null;
    }

    // 清除波纹状态
    rippleWavesRef.current = [];

    if (highlightFeatureRef.current) {
      highlightFeatureRef.current.set('highlighted', false);
      highlightFeatureRef.current.set('rippleWaves', undefined);
      highlightFeatureRef.current = null;
    }
  }, []);

  // 高亮设备（水波纹动画）
  const highlightDevice = useCallback((deviceId: string) => {
    if (!deviceSourceRef.current || !mapInstanceRef.current) return;

    // 先清除之前的高亮
    clearHighlight();

    const feature = deviceSourceRef.current.getFeatureById(deviceId);
    if (feature) {
      feature.set('highlighted', true);
      highlightFeatureRef.current = feature as Feature;

      // 飞行到设备位置
      const geometry = feature.getGeometry();
      if (geometry) {
        const coordinate = (geometry as Point).getCoordinates();
        const lonLat = toLonLat(coordinate);
        flyTo(lonLat[0], lonLat[1]);
      }

      // 初始化波纹状态
      rippleWavesRef.current = [];

      // 创建新波纹的函数
      const createRipple = () => {
        if (!highlightFeatureRef.current) return;
        rippleWavesRef.current.push({
          radius: 0,      // 从中心开始
          opacity: 0.5,   // 初始透明度（更浅）
        });
      };

      // 立即创建第一个波纹
      createRipple();

      // 定时创建新波纹（每 600ms）
      rippleCreateRef.current = setInterval(() => {
        createRipple();
        // 最多同时存在 3 个波纹
        if (rippleWavesRef.current.length > 3) {
          rippleWavesRef.current.shift();
        }
      }, 600);

      // 波纹扩散动画（每 30ms 更新）
      pulseAnimationRef.current = setInterval(() => {
        if (!highlightFeatureRef.current) {
          if (pulseAnimationRef.current) {
            clearInterval(pulseAnimationRef.current);
            pulseAnimationRef.current = null;
          }
          return;
        }

        // 更新所有波纹的状态
        rippleWavesRef.current = rippleWavesRef.current
          .map(wave => ({
            radius: wave.radius + 0.5,     // 半径增长（扩散更慢）
            opacity: wave.opacity - 0.01,  // 透明度降低
          }))
          .filter(wave => wave.opacity > 0); // 移除已消失的波纹

        // 更新 feature 的波纹数据
        highlightFeatureRef.current.set('rippleWaves', [...rippleWavesRef.current]);

        // 触发地图重新渲染
        if (mapInstanceRef.current) {
          mapInstanceRef.current.render();
        }
      }, 30); // 30ms 更新一次，实现平滑动画
    }
  }, [flyTo, clearHighlight]);

  // 刷新地图尺寸
  const updateSize = useCallback(() => {
    mapInstanceRef.current?.updateSize();
  }, []);

  // 获取当前 zoom 级别
  const getZoom = useCallback((): number => {
    if (!mapInstanceRef.current) return zoom;
    return mapInstanceRef.current.getView().getZoom() ?? zoom;
  }, [zoom]);

  // 适配边界
  const fitBounds = useCallback((bounds: MapBounds) => {
    if (!mapInstanceRef.current) return;

    const view = mapInstanceRef.current.getView();
    const extent = [
      fromLonLat([bounds.minLng, bounds.minLat])[0],
      fromLonLat([bounds.minLng, bounds.minLat])[1],
      fromLonLat([bounds.maxLng, bounds.maxLat])[0],
      fromLonLat([bounds.maxLng, bounds.maxLat])[1],
    ];

    view.fit(extent, {
      padding: [50, 50, 50, 50],
      duration: ANIMATION_CONFIG.flyDuration,
    });
  }, []);

  return {
    mapRef,
    mapInstanceRef,
    updateDevices,
    getViewport,
    flyTo,
    highlightDevice,
    clearHighlight,
    isReady,
    updateSize,
    getZoom,
    fitBounds,
  };
}

/**
 * 绑定地图事件
 */
function bindMapEvents(
  map: Map,
  deviceSource: VectorSource,
  clusterSource: Cluster,
  callbacks: {
    onDeviceClick?: (device: MapDevice) => void;
    onDeviceHover?: (device: MapDevice | null, pixel?: { x: number; y: number }) => void;
    onViewportChange?: (viewport: MapViewport) => void;
    onClusterClick?: (devices: MapDevice[]) => void;
  },
  deviceLayer: VectorLayer<VectorSource>
): void {
  const { onDeviceClick, onDeviceHover, onViewportChange, onClusterClick } = callbacks;

  // 点击事件
  map.on('click', (evt) => {
    const features = map.getFeaturesAtPixel(evt.pixel, {
      layerFilter: (layer) => layer === deviceLayer, // 设备图层
    });

    if (features.length > 0) {
      const feature = features[0] as Feature;
      const featuresProp = feature.get('features');

      if (featuresProp && Array.isArray(featuresProp)) {
        // 聚合点击
        if (featuresProp.length === 1) {
          // 单个设备
          const device = featuresProp[0].getProperties() as MapDevice;
          onDeviceClick?.(device);
        } else {
          // 多个设备
          const devices = featuresProp.map((f: Feature) => f.getProperties() as MapDevice);
          onClusterClick?.(devices);
          // 放大地图展开聚合
          const view = map.getView();
          const currentZoom = view.getZoom() ?? 0;
          view.animate({
            center: evt.coordinate,
            zoom: currentZoom + 2,
            duration: 300,
          });
        }
      } else {
        // 单独设备
        const device = feature.getProperties() as MapDevice;
        onDeviceClick?.(device);
      }
    }
  });

  // 悬停事件
  let hoveredFeature: Feature | null = null;

  map.on('pointermove', (evt) => {
    const features = map.getFeaturesAtPixel(evt.pixel, {
      layerFilter: (layer) => layer === deviceLayer,
    });

    if (features.length > 0) {
      const feature = features[0] as Feature;
      const featuresProp = feature.get('features');

      // 获取单个设备
      let device: MapDevice | null = null;
      if (featuresProp && Array.isArray(featuresProp) && featuresProp.length === 1) {
        device = featuresProp[0].getProperties() as MapDevice;
      } else if (!featuresProp) {
        device = feature.getProperties() as MapDevice;
      }

      if (device) {
        // 更新悬停状态
        if (hoveredFeature && hoveredFeature !== feature) {
          hoveredFeature.set('hovered', false);
        }
        feature.set('hovered', true);
        hoveredFeature = feature;

        onDeviceHover?.(device, { x: evt.pixel[0], y: evt.pixel[1] });
        map.getTargetElement().style.cursor = 'pointer';
      } else {
        if (hoveredFeature) {
          hoveredFeature.set('hovered', false);
          hoveredFeature = null;
        }
        onDeviceHover?.(null);
        map.getTargetElement().style.cursor = '';
      }
    } else {
      if (hoveredFeature) {
        hoveredFeature.set('hovered', false);
        hoveredFeature = null;
      }
      onDeviceHover?.(null);
      map.getTargetElement().style.cursor = '';
    }
  });

  // 视图变化事件（带防抖）
  let moveEndTimeout: ReturnType<typeof setTimeout>;
  map.on('moveend', () => {
    clearTimeout(moveEndTimeout);
    moveEndTimeout = setTimeout(() => {
      if (onViewportChange) {
        const view = map.getView();
        const center = toLonLat(view.getCenter()!);
        const extent = view.calculateExtent(map.getSize());
        const bottomLeft = toLonLat([extent[0], extent[1]]);
        const topRight = toLonLat([extent[2], extent[3]]);

        onViewportChange({
          centerLng: center[0],
          centerLat: center[1],
          zoom: view.getZoom()!,
          bounds: {
            minLng: bottomLeft[0],
            maxLng: topRight[0],
            minLat: bottomLeft[1],
            maxLat: topRight[1],
          },
        });
      }
    }, 100);
  });
}

export default useOLMap;
