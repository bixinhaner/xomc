/**
 * OpenLayers 地图核心 Hook
 * @module components/GISMap/useOLMap
 */

import { useEffect, useRef, useCallback, useState, useMemo } from 'react';
import Map from 'ol/Map';
import View from 'ol/View';
import BaseLayer from 'ol/layer/Base';
import TileLayer from 'ol/layer/Tile';
import VectorLayer from 'ol/layer/Vector';
import VectorSource from 'ol/source/Vector';
import Cluster from 'ol/source/Cluster';
import XYZ from 'ol/source/XYZ';
import Feature from 'ol/Feature';
import Point from 'ol/geom/Point';
import LineString from 'ol/geom/LineString';
import { fromLonLat, toLonLat } from 'ol/proj';
import { containsCoordinate, buffer as bufferExtent } from 'ol/extent';
import type { Extent } from 'ol/extent';
import { defaults as defaultControls } from 'ol/control';
import { Style, Stroke, Circle, Fill, Text } from 'ol/style';
import type { StyleLike } from 'ol/style/Style';
import type { MapDevice, MapViewport, MapBounds } from '@core/types/map';
import {
  MAP_CONFIG,
  ANIMATION_CONFIG,
  CLUSTER_CONFIG,
  COLORS,
  DEVICE_STATUS_CONFIG,
  SPIDERFY_CONFIG,
  VIEWPORT_CULLING,
} from './constants';
import {
  clusterStyleFunction,
  createSpiderfyLineStyle,
  createSpiderfyPointStyle,
  createSpiderfyCenterStyle,
  createSpiderfyPointHoverStyle,
} from './styleUtils';
import { useMapConfig, type MapMetadata } from './useMapConfig';
import {
  isValidMapMetadata,
  buildSafeConfig,
  checkTileAvailability,
} from '@/utils/mapValidation';

// 用于 spiderfy 函数内部访问
const SPIDERFY_CONFIG_REF = SPIDERFY_CONFIG;

/**
 * Easing 函数集合
 * 用于地图动画的缓动效果
 */
const Easing = {
  /** easeOutCubic：快速启动，平滑结束 */
  easeOutCubic: (t: number): number => 1 - Math.pow(1 - t, 3),

  /**
   * 渐进式缩放 easing：模拟从宏观到微观的自然减速
   * - 前25%：快速启动（快速离开当前视图）
   * - 中间35%：匀速过渡（自然平滑）
   * - 后40%：平滑减速（精确定位，节点自然出现）
   */
  progressive: (t: number): number => {
    if (t < 0.25) {
      // 前25%：快速启动
      return t * t * (3 - 2 * t); // smoothstep
    } else if (t < 0.6) {
      // 中间35%：线性过渡
      return 0.0625 + (t - 0.25) * 1.15;
    } else {
      // 后40%：更长、更平缓的减速
      const u = (t - 0.6) / 0.4;
      // 使用 easeOutCubic 让减速更平滑
      return 0.46 + (1 - Math.pow(1 - u, 3)) * 0.54;
    }
  },
};

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
  /** 设备点击回调（包含鼠标位置） */
  onDeviceClick?: (device: MapDevice, pixel?: { x: number; y: number }) => void;
  /** 设备悬停回调（包含鼠标位置） */
  onDeviceHover?: (device: MapDevice | null, pixel?: { x: number; y: number }) => void;
  /** 视图变化回调 */
  onViewportChange?: (viewport: MapViewport) => void;
  /** 聚合点击回调 */
  onClusterClick?: (devices: MapDevice[]) => void;
  /** 缩放级别变化回调 */
  onZoomChange?: (zoom: number) => void;
  /** 地图点击回调（点击任意位置时触发） */
  onMapClick?: () => void;
}

interface UseOLMapReturn {
  /** 地图容器 ref */
  mapRef: React.RefObject<HTMLDivElement | null>;
  /** 地图实例 ref */
  mapInstanceRef: React.MutableRefObject<Map | null>;
  /** 更新设备数据 */
  updateDevices: (devices: MapDevice[]) => void;
  /** 获取当前视图状态 */
  getViewport: () => MapViewport | null;
  /** 飞行到指定位置 */
  flyTo: (lng: number, lat: number, zoom?: number, options?: {
    /** 是否使用渐进式缩放动画（默认根据配置决定） */
    progressive?: boolean;
    /** 瓦片服务最大zoom（用于限制动画范围） */
    maxZoom?: number;
    /** 动画完成回调 */
    onComplete?: () => void;
  }) => void;
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
  /** 高亮设备并在需要时展开聚合 */
  highlightAndSpiderfyIfNeeded: (device: MapDevice, skipFlyTo?: boolean) => void;
  /** 地图元数据 */
  metadata: MapMetadata | null;
  /** 元数据加载状态 */
  metadataLoading: boolean;
}

/**
 * OpenLayers 地图 Hook
 * 支持从服务端加载元数据（TileJSON），实现零配置切换
 */
export function useOLMap(options: UseOLMapOptions = {}): UseOLMapReturn {
  // 加载地图元数据
  const { metadata, loading: metadataLoading } = useMapConfig();
  // 瓦片可用性检查状态
  const [tilesAvailable, setTilesAvailable] = useState<boolean | null>(null);

  // 检查瓦片文件是否实际可用（仅在 metadata 验证通过后执行）
  useEffect(() => {
    // metadata 未加载完成或无效，跳过检查
    if (metadataLoading || !isValidMapMetadata(metadata)) {
      return;
    }

    let cancelled = false;

    const checkAvailability = async () => {
      try {
        const available = await checkTileAvailability(metadata);
        if (!cancelled) {
          setTilesAvailable(available);
        }
      } catch {
        // 检查失败，保守降级到在线地图
        if (!cancelled) {
          setTilesAvailable(false);
        }
      }
    };

    checkAvailability();

    return () => {
      cancelled = true;
    };
  }, [metadata, metadataLoading]);

  // 根据元数据、瓦片可用性或传入的 options 获取配置
  const config = useMemo(() => {
    // 验证 metadata 是否有效，且瓦片文件实际可用
    if (isValidMapMetadata(metadata) && tilesAvailable === true) {
      return buildSafeConfig(metadata);
    }

    // 降级到在线 OSM
    return {
      defaultCenter: MAP_CONFIG.defaultCenter,
      defaultZoom: MAP_CONFIG.defaultZoom,
      minZoom: MAP_CONFIG.minZoom,
      maxZoom: MAP_CONFIG.maxZoom,
      tileUrl: MAP_CONFIG.osmTileUrl,
      attribution: MAP_CONFIG.attribution || '© OpenStreetMap contributors',
      bounds: MAP_CONFIG.bounds,
    };
  }, [metadata, tilesAvailable]);

  const {
    tileUrl,
    center = config.defaultCenter,
    zoom = config.defaultZoom,
    minZoom = config.minZoom,
    maxZoom = config.maxZoom,
    clusterDistance = CLUSTER_CONFIG.distance,
    onDeviceClick,
    onDeviceHover,
    onViewportChange,
    onClusterClick,
    onMapClick,
  } = options;

  const mapRef = useRef<HTMLDivElement>(null);
  const mapInstanceRef = useRef<Map | null>(null);
  // 完整设备列表（性能 #15：视口裁剪时只渲染可视范围内的子集，
  // 其余设备保留在此 ref 中，平移/缩放后按新视口重算）
  const allDevicesRef = useRef<MapDevice[]>([]);
  // 视口裁剪函数引用（在 moveend 中调用，避免 bindMapEvents 签名漂移）
  const cullDevicesRef = useRef<(() => void) | null>(null);
  // 强制保留渲染的设备 id（如搜索定位目标）：即便落在视口外也始终渲染，
  // 避免裁剪把搜索高亮的目标点剔除导致定位失败。
  const pinnedDeviceIdsRef = useRef<Set<string>>(new Set());
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
  // Spiderfy 状态
  const spiderfyLayerRef = useRef<VectorLayer<VectorSource> | null>(null);
  const spiderfySourceRef = useRef<VectorSource | null>(null);
  const isSpiderfiedRef = useRef(false);
  const spiderfiedCenterRef = useRef<number[] | null>(null);
  // 高亮请求 ID（用于防止竞态条件）
  const highlightRequestIdRef = useRef(0);

  const [isReady, setIsReady] = useState(false);

  // 收起 Spiderfy 展开（必须在 useEffect 之前定义，供 bindMapEvents 使用）
  const unspiderfy = useCallback(() => {
    if (!isSpiderfiedRef.current || !spiderfySourceRef.current) return;

    // 清除 spiderfy 图层的所有 feature
    spiderfySourceRef.current.clear();
    isSpiderfiedRef.current = false;
    spiderfiedCenterRef.current = null;

    // 注意：VectorSource.clear() 会自动触发渲染，手动调用 render() 可能冗余
    // 保留此行以确保兼容性，后续可移除并测试验证
    mapInstanceRef.current?.render();
  }, []);

  // 展开 Spiderfy（扇形方式）（必须在 useEffect 之前定义，供 bindMapEvents 使用）
  const spiderfy = useCallback((_clusterFeature: Feature, center: number[], features: Feature[]) => {
    if (!mapInstanceRef.current || !spiderfySourceRef.current) return;

    // 如果已经展开，先收起
    if (isSpiderfiedRef.current) {
      unspiderfy();
    }

    const count = features.length;
    if (count <= 1) return;

    // 创建 spiderfy 图层的 feature
    const spiderfyFeatures: Feature[] = [];

    // 中心点 feature
    const centerFeature = new Feature({
      geometry: new Point(center),
      spiderfyCenter: true,
      count,
    });
    spiderfyFeatures.push(centerFeature);

    // 扇形展开点
    const angleStep = (2 * Math.PI) / count;
    const startAngle = -Math.PI / 2; // 从顶部开始

    features.forEach((f, index) => {
      const device = f.getProperties() as MapDevice;
      const angle = startAngle + index * angleStep;

      // 计算展开点的坐标（像素偏移转换为地图坐标）
      const pixelOffset = [
        Math.cos(angle) * SPIDERFY_CONFIG.radius,
        Math.sin(angle) * SPIDERFY_CONFIG.radius,
      ];

      // 将像素偏移转换为地图坐标偏移
      const map = mapInstanceRef.current!;
      const resolution = map.getView().getResolution()!;
      const coordinateOffset = [
        pixelOffset[0] * resolution,
        pixelOffset[1] * resolution,
      ];

      const pointCoordinate = [
        center[0] + coordinateOffset[0],
        center[1] - coordinateOffset[1], // Y 轴反向
      ];

      // 创建连线 feature
      const lineFeature = new Feature({
        geometry: new LineString([center, pointCoordinate]),
        spiderfyLine: true,
      });
      spiderfyFeatures.push(lineFeature);

      // 创建展开点 feature
      const pointFeature = new Feature({
        geometry: new Point(pointCoordinate),
        spiderfyPoint: true,
        device: device,
        index: index,
        total: count,
      });
      spiderfyFeatures.push(pointFeature);
    });

    // 添加所有 feature 到 spiderfy 图层
    spiderfySourceRef.current.addFeatures(spiderfyFeatures);

    isSpiderfiedRef.current = true;
    spiderfiedCenterRef.current = center;

    // 触发地图重新渲染
    mapInstanceRef.current.render();
  }, [unspiderfy]);

  // 根据缩放级别动态调整聚合距离
  const updateClusterDistance = useCallback((zoom: number) => {
    if (!clusterSourceRef.current) return;

    let newDistance: number;
    if (zoom >= CLUSTER_CONFIG.disableClusterZoom) {
      // 高缩放级别：禁用聚合（distance = 0 表示不聚合）
      newDistance = 0;
    } else if (zoom >= 12) {
      // 中等缩放级别：使用较小的聚合距离
      newDistance = CLUSTER_CONFIG.highZoomDistance;
    } else {
      // 低缩放级别：使用正常聚合距离
      newDistance = clusterDistance;
    }

    // 只有距离变化时才更新，避免不必要的重绘
    if (clusterSourceRef.current.getDistance() !== newDistance) {
      clusterSourceRef.current.setDistance(newDistance);
      // 强制刷新 Cluster source，确保重新计算聚合
      clusterSourceRef.current.refresh();
      // 触发地图重新渲染
      mapInstanceRef.current?.render();
    }
  }, [clusterDistance]);

  // 使用 ref 跟踪地图是否已初始化（避免依赖项导致的重复初始化）
  const isMapInitializedRef = useRef(false);

  // 初始化地图（等待元数据加载完成后执行一次）
  /* eslint-disable react-hooks/exhaustive-deps -- 地图初始化应只执行一次，使用 ref 防止重复初始化 */
  useEffect(() => {
    // 已经初始化过，不再重复
    if (isMapInitializedRef.current) return;
    // 容器不存在或地图已存在，跳过
    if (!mapRef.current || mapInstanceRef.current) return;
    // 等待元数据加载完成
    if (metadataLoading) return;

    // 标记初始化开始
    isMapInitializedRef.current = true;

    // 创建瓦片图层
    // 优先级：传入的 tileUrl > 环境变量 > 元数据配置 > 默认 OSM
    const finalTileUrl = tileUrl || import.meta.env.VITE_MAP_TILE_URL || config.tileUrl;
    const layers: BaseLayer[] = [];

    // 瓦片数据源
    const tileSource = new XYZ({
      url: finalTileUrl,
      crossOrigin: 'anonymous',
      projection: 'EPSG:3857',
      tileSize: 256,
      minZoom: config.minZoom || 6,
      maxZoom: config.maxZoom || 15,
    });

    const tileLayer = new TileLayer({
      source: tileSource,
      opacity: 1.0,
      zIndex: 0, // 确保瓦片层在最底层
    });
    layers.push(tileLayer);

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
      style: clusterStyleFunction as StyleLike,
      zIndex: 10,
    });
    layers.push(deviceLayerRef.current);

    // 创建 Spiderfy 图层（用于展开重叠设备点）
    spiderfySourceRef.current = new VectorSource();
    spiderfyLayerRef.current = new VectorLayer({
      source: spiderfySourceRef.current,
      style: spiderfyStyleFunction as StyleLike,
      zIndex: 11, // 确保在设备图层之上
    });
    layers.push(spiderfyLayerRef.current);

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
        onZoomChange: (zoom) => {
          updateClusterDistance(zoom);
          // 缩放级别变化时收起 spiderfy
          if (isSpiderfiedRef.current) {
            unspiderfy();
          }
        },
        onSpiderfy: spiderfy,
        onUnspiderfy: unspiderfy,
        onMapClick,
      },
      deviceLayerRef.current,
      spiderfyLayerRef.current
    );

    // 视口裁剪重算（性能 #15）：地图平移/缩放结束后，按新视口重新喂点。
    // 带防抖，避免连续 moveend 频繁重建 feature。
    let cullTimeout: ReturnType<typeof setTimeout>;
    mapInstanceRef.current.on('moveend', () => {
      clearTimeout(cullTimeout);
      cullTimeout = setTimeout(() => {
        cullDevicesRef.current?.();
      }, 150);
    });

    // 延迟设置 isReady，避免在 effect 中同步调用 setState 导致级联渲染
    // 使用 setTimeout 将状态更新推迟到下一个事件循环
    setTimeout(() => setIsReady(true), 0);

    // 清理函数
    return () => {
      if (mapInstanceRef.current) {
        mapInstanceRef.current.setTarget(undefined);
        mapInstanceRef.current = null;
      }
      deviceSourceRef.current = null;
      clusterSourceRef.current = null;
      deviceLayerRef.current = null;
      spiderfySourceRef.current = null;
      spiderfyLayerRef.current = null;
      // 重置初始化标记，允许重新初始化
      isMapInitializedRef.current = false;
    };
  }, [metadataLoading]);

  // 把一组设备渲染进 VectorSource（构造 Point feature 并替换现有数据）
  const renderDeviceFeatures = useCallback((devices: MapDevice[]) => {
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

  // 视口裁剪：只渲染当前可视范围（带缓冲）内的设备（性能 #15）
  // 设备数 <= 阈值时直接全量渲染，保持原有行为。
  const cullDevicesToViewport = useCallback(() => {
    const map = mapInstanceRef.current;
    if (!map || !deviceSourceRef.current) return;

    const all = allDevicesRef.current;

    // 数据量较小：全量渲染，无需裁剪
    if (all.length <= VIEWPORT_CULLING.enableThreshold) {
      renderDeviceFeatures(all);
      return;
    }

    const size = map.getSize();
    if (!size) {
      // 地图尺寸未就绪，保守全量渲染（不丢点）
      renderDeviceFeatures(all);
      return;
    }

    // 计算当前可视 extent 并按缓冲倍数向外扩展（投影坐标 EPSG:3857）
    const extent = map.getView().calculateExtent(size) as Extent;
    const bufferX = (extent[2] - extent[0]) * VIEWPORT_CULLING.bufferRatio;
    const bufferY = (extent[3] - extent[1]) * VIEWPORT_CULLING.bufferRatio;
    // bufferExtent 取单一边距值，取宽高缓冲的较大者以覆盖两个方向
    const bufferedExtent = bufferExtent(extent, Math.max(bufferX, bufferY));

    const pinned = pinnedDeviceIdsRef.current;
    const visible = all.filter(
      (device) =>
        pinned.has(device.id) ||
        containsCoordinate(bufferedExtent, fromLonLat([device.lng, device.lat]))
    );

    renderDeviceFeatures(visible);
  }, [renderDeviceFeatures]);

  // 把裁剪函数挂到 ref，供 moveend 监听器调用
  useEffect(() => {
    cullDevicesRef.current = cullDevicesToViewport;
  }, [cullDevicesToViewport]);

  // 更新设备数据
  const updateDevices = useCallback((devices: MapDevice[]) => {
    if (!deviceSourceRef.current) return;

    // 如果当前有 spiderfy 展开，先收起（因为设备数据已变化，展开的内容可能不再有效）
    if (isSpiderfiedRef.current) {
      unspiderfy();
    }

    // 保存完整列表，随后按当前视口裁剪渲染
    allDevicesRef.current = devices;
    cullDevicesToViewport();
  }, [unspiderfy, cullDevicesToViewport]);

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

  /**
   * 渐进式飞行到目标位置
   *
   * 实现从宏观到微观的多级缩放效果：国家→省份→市→区→街道
   * 所有中间步骤的zoom级别都被限制在瓦片服务覆盖范围内
   *
   * @param lng - 目标经度
   * @param lat - 目标纬度
   * @param targetZoom - 最终缩放级别
   * @param maxZoom - 瓦片服务最大zoom（确保不超出覆盖范围）
   * @param onComplete - 动画完成回调
   */
  const progressiveFlyTo = useCallback((
    lng: number,
    lat: number,
    targetZoom: number,
    _maxZoom: number,
    onComplete?: () => void
  ) => {
    const map = mapInstanceRef.current;
    if (!map) {
      onComplete?.();
      return;
    }

    const view = map.getView();
    const currentZoom = view.getZoom() ?? MAP_CONFIG.defaultZoom;

    // 智能判断：如果zoom差值太小，使用单次动画
    if (Math.abs(currentZoom - targetZoom) < ANIMATION_CONFIG.progressiveZoomThreshold) {
      const animateOptions = {
        center: fromLonLat([lng, lat]),
        zoom: targetZoom,
        duration: 600,
        easing: Easing.easeOutCubic,
      };

      if (onComplete) {
        view.animate(animateOptions, () => onComplete());
      } else {
        view.animate(animateOptions);
      }
      return;
    }

    // 渐进式缩放：使用单次长动画 + 平滑 easing 曲线
    // 模拟从宏观到微观的自然减速效果，避免分段感
    const targetCoord = fromLonLat([lng, lat]);
    const zoomDiff = targetZoom - currentZoom;

    // 根据缩放幅度动态调整时长，稍微慢一点
    const totalDuration = Math.min(1400, Math.max(900, zoomDiff * 90));

    const animateOptions = {
      center: targetCoord,
      zoom: targetZoom,
      duration: totalDuration,
      easing: Easing.progressive,
    };

    if (onComplete) {
      view.animate(animateOptions, () => onComplete());
    } else {
      view.animate(animateOptions);
    }
  }, [MAP_CONFIG.defaultZoom]);

  // 飞行到指定位置
  const flyTo = useCallback((lng: number, lat: number, targetZoom?: number, options?: {
    /** 是否使用渐进式缩放动画（默认根据配置决定） */
    progressive?: boolean;
    /** 瓦片服务最大zoom（用于限制动画范围） */
    maxZoom?: number;
    /** 动画完成回调 */
    onComplete?: () => void;
  }) => {
    if (!mapInstanceRef.current) return;

    const {
      progressive = ANIMATION_CONFIG.enableProgressiveZoom,
      maxZoom = MAP_CONFIG.maxZoom,
      onComplete,
    } = options ?? {};

    const finalZoom = targetZoom ?? ANIMATION_CONFIG.highlightZoom;

    // 使用渐进式缩放动画
    if (progressive) {
      progressiveFlyTo(lng, lat, finalZoom, maxZoom, onComplete);
      return;
    }

    // 使用单次动画（保持向后兼容）
    const view = mapInstanceRef.current.getView();
    if (onComplete) {
      view.animate({
        center: fromLonLat([lng, lat]),
        zoom: finalZoom,
        duration: ANIMATION_CONFIG.flyDuration,
      }, () => onComplete());
    } else {
      view.animate({
        center: fromLonLat([lng, lat]),
        zoom: finalZoom,
        duration: ANIMATION_CONFIG.flyDuration,
      });
    }
  }, [progressiveFlyTo]);

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

    // 取消搜索定位 pin（性能 #15）：高亮结束后该设备恢复受裁剪约束，
    // 避免 pin 集合无界增长。下次正常裁剪会按视口决定是否渲染。
    if (pinnedDeviceIdsRef.current.size > 0) {
      pinnedDeviceIdsRef.current.clear();
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

      // 初始化波纹状态 - 简约风格：最多2个波纹
      rippleWavesRef.current = [];

      // 创建新波纹的函数
      const createRipple = () => {
        if (!highlightFeatureRef.current) return;
        rippleWavesRef.current.push({
          radius: 0,      // 从中心开始
          opacity: 0.6,   // 初始透明度（适中）
        });
      };

      // 立即创建第一个波纹
      createRipple();

      // 定时创建新波纹（每 800ms - 更舒缓的节奏）
      rippleCreateRef.current = setInterval(() => {
        createRipple();
        // 最多同时存在 2 个波纹 - 简约风格
        if (rippleWavesRef.current.length > 2) {
          rippleWavesRef.current.shift();
        }
      }, 800);

      // 波纹扩散动画（每 40ms 更新 - 更流畅）
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
            radius: wave.radius + 0.4,     // 半径增长（更缓慢优雅）
            opacity: wave.opacity - 0.008,  // 透明度降低（更持久）
          }))
          .filter(wave => wave.opacity > 0); // 移除已消失的波纹

        // 更新 feature 的波纹数据
        highlightFeatureRef.current.set('rippleWaves', [...rippleWavesRef.current]);

        // 触发地图重新渲染
        if (mapInstanceRef.current) {
          mapInstanceRef.current.render();
        }
      }, 40); // 40ms 更新一次，更流畅的动画
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

  // 高亮设备并在需要时展开聚合（用于搜索定位）
  const highlightAndSpiderfyIfNeeded = useCallback((device: MapDevice, skipFlyTo = false) => {
    if (!mapInstanceRef.current || !clusterSourceRef.current || !spiderfySourceRef.current) return;

    // 性能 #15：把搜索/定位目标 pin 住并立即重算裁剪，
    // 确保即便目标当前落在视口外，也已渲染进 deviceSource，
    // 后续聚合查找（clusterSource）才能命中它。
    if (allDevicesRef.current.length > VIEWPORT_CULLING.enableThreshold) {
      pinnedDeviceIdsRef.current.add(device.id);
      cullDevicesToViewport();
    }

    // 递增请求 ID，用于防止竞态条件
    const currentRequestId = ++highlightRequestIdRef.current;

    // 先清除之前的高亮和展开
    clearHighlight();
    if (isSpiderfiedRef.current) {
      unspiderfy();
    }

    // 记录当前 zoom，用于判断是否需要等待聚合稳定
    const currentZoom = mapInstanceRef.current.getView().getZoom() ?? 0;
    const targetZoom = ANIMATION_CONFIG.maxHighlightZoom || 18;
    const needsZoomChange = Math.abs(currentZoom - targetZoom) > 0.5;

    // 飞行到设备位置（如果 skipFlyTo 为 true 则跳过，避免打断渐进式动画）
    if (!skipFlyTo) {
      flyTo(device.lng, device.lat, targetZoom);
    }

    // 使用 moveend 事件确保在地图完全稳定后执行
    // 这样可以避免时序问题：聚合计算完成、zoom 变化检测等都已完成
    const doHighlightAndSpiderfy = () => {
      // 检查是否是最新的请求，防止竞态条件
      if (currentRequestId !== highlightRequestIdRef.current) return;

      if (!clusterSourceRef.current || !spiderfySourceRef.current || !mapInstanceRef.current) return;

      // 强制刷新聚合源，确保获取最新的聚合结果
      clusterSourceRef.current.refresh();

      // 使用 requestAnimationFrame 确保在下一帧渲染时执行
      requestAnimationFrame(() => {
        // 再次检查是否是最新的请求
        if (currentRequestId !== highlightRequestIdRef.current) return;

        if (!clusterSourceRef.current || !spiderfySourceRef.current || !mapInstanceRef.current) return;

        // 获取所有聚合 features
        const clusterFeatures = clusterSourceRef.current.getFeatures();
        let targetClusterFeature: Feature | null = null;
        let targetDeviceFeature: Feature | null = null;

        // 查找包含目标设备的聚合
        for (const clusterFeature of clusterFeatures) {
          const features = clusterFeature.get('features');
          if (features && Array.isArray(features)) {
            for (const f of features as Feature[]) {
              const props = f.getProperties() as MapDevice;
              if (props.id === device.id) {
                targetClusterFeature = clusterFeature;
                targetDeviceFeature = f;
                break;
              }
            }
          }
          if (targetClusterFeature) break;
        }

        if (!targetClusterFeature) return;

        const featuresInCluster = targetClusterFeature.get('features') as Feature[] | undefined;
        const count = featuresInCluster?.length || 1;

        if (count > 1 && featuresInCluster) {
          // 多个设备在同一聚合中，需要展开 spiderfy
          const geometry = targetClusterFeature.getGeometry();
          if (geometry) {
            const center = (geometry as Point).getCoordinates();

            // 先展开 spiderfy
            spiderfy(targetClusterFeature, center, featuresInCluster);

            // 延迟应用高亮到 spiderfy 点上
            requestAnimationFrame(() => {
              // 检查是否是最新的请求
              if (currentRequestId !== highlightRequestIdRef.current) return;

              if (!spiderfySourceRef.current || !mapInstanceRef.current) return;

              // 找到目标设备的 spiderfy 点
              const spiderfyFeatures = spiderfySourceRef.current.getFeatures();
              for (const sf of spiderfyFeatures) {
                if (sf.get('spiderfyPoint')) {
                  const sfDevice = sf.get('device') as MapDevice;
                  if (sfDevice.id === device.id) {
                    // 在 spiderfy 点上应用高亮
                    highlightFeatureRef.current = sf as Feature;

                    // 初始化波纹状态 - 简约风格：最多2个波纹
                    rippleWavesRef.current = [];

                    // 创建新波纹的函数
                    const createRipple = () => {
                      if (!highlightFeatureRef.current) return;
                      rippleWavesRef.current.push({
                        radius: 0,
                        opacity: 0.6,
                      });
                    };

                    // 立即创建第一个波纹
                    createRipple();

                    // 定时创建新波纹（每 800ms）
                    rippleCreateRef.current = setInterval(() => {
                      createRipple();
                      if (rippleWavesRef.current.length > 2) {
                        rippleWavesRef.current.shift();
                      }
                    }, 800);

                    // 波纹扩散动画（每 40ms）
                    pulseAnimationRef.current = setInterval(() => {
                      if (!highlightFeatureRef.current) {
                        if (pulseAnimationRef.current) {
                          clearInterval(pulseAnimationRef.current);
                          pulseAnimationRef.current = null;
                        }
                        return;
                      }

                      rippleWavesRef.current = rippleWavesRef.current
                        .map(wave => ({
                          radius: wave.radius + 0.4,
                          opacity: wave.opacity - 0.008,
                        }))
                        .filter(wave => wave.opacity > 0);

                      highlightFeatureRef.current.set('rippleWaves', [...rippleWavesRef.current]);

                      if (mapInstanceRef.current) {
                        mapInstanceRef.current.render();
                      }
                    }, 40);

                    break;
                  }
                }
              }
            });
          }
        } else if (targetDeviceFeature) {
          // 单个设备，直接高亮
          targetDeviceFeature.set('highlighted', true);
          highlightFeatureRef.current = targetDeviceFeature as Feature;

          // 初始化波纹状态 - 简约风格：最多2个波纹
          rippleWavesRef.current = [];

          // 创建新波纹的函数
          const createRipple = () => {
            if (!highlightFeatureRef.current) return;
            rippleWavesRef.current.push({
              radius: 0,
              opacity: 0.6,
            });
          };

          // 立即创建第一个波纹
          createRipple();

          // 定时创建新波纹（每 800ms）
          rippleCreateRef.current = setInterval(() => {
            createRipple();
            if (rippleWavesRef.current.length > 2) {
              rippleWavesRef.current.shift();
            }
          }, 800);

          // 波纹扩散动画（每 40ms）
          pulseAnimationRef.current = setInterval(() => {
            if (!highlightFeatureRef.current) {
              if (pulseAnimationRef.current) {
                clearInterval(pulseAnimationRef.current);
                pulseAnimationRef.current = null;
              }
              return;
            }

            rippleWavesRef.current = rippleWavesRef.current
              .map(wave => ({
                radius: wave.radius + 0.4,
                opacity: wave.opacity - 0.008,
              }))
              .filter(wave => wave.opacity > 0);

            highlightFeatureRef.current.set('rippleWaves', [...rippleWavesRef.current]);

            if (mapInstanceRef.current) {
              mapInstanceRef.current.render();
            }
          }, 40);
        }
      });
    };

    if (needsZoomChange) {
      // 需要 zoom 变化，监听 moveend 事件确保地图完全稳定
      const map = mapInstanceRef.current;
      let handled = false;

      const onMoveEnd = () => {
        if (handled) return;
        handled = true;
        map.un('moveend', onMoveEnd);
        // 额外延迟确保聚合计算完成
        setTimeout(doHighlightAndSpiderfy, 150);
      };

      map.on('moveend', onMoveEnd);

      // 安全超时：如果 moveend 没触发（不应该发生），也执行
      setTimeout(() => {
        if (!handled) {
          handled = true;
          map.un('moveend', onMoveEnd);
          doHighlightAndSpiderfy();
        }
      }, ANIMATION_CONFIG.flyDuration + 500);
    } else {
      // 不需要 zoom 变化，直接执行（但稍作延迟确保状态稳定）
      setTimeout(doHighlightAndSpiderfy, 100);
    }
  }, [flyTo, clearHighlight, unspiderfy, spiderfy, cullDevicesToViewport]);

  return {
    mapRef,
    mapInstanceRef,
    updateDevices,
    getViewport,
    flyTo,
    highlightDevice,
    clearHighlight,
    isReady: isReady && !metadataLoading,
    updateSize,
    getZoom,
    fitBounds,
    highlightAndSpiderfyIfNeeded,
    metadata,
    metadataLoading,
  };
}

/**
 * Spiderfy 图层样式函数
 */
function spiderfyStyleFunction(feature: Feature): Style | Style[] {
  // 中心点样式
  if (feature.get('spiderfyCenter')) {
    const count = feature.get('count') || 1;
    return createSpiderfyCenterStyle(count);
  }

  // 连线样式
  if (feature.get('spiderfyLine')) {
    return createSpiderfyLineStyle();
  }

  // 展开点样式
  if (feature.get('spiderfyPoint')) {
    const device = feature.get('device') as MapDevice;
    const index = feature.get('index') as number;
    const total = feature.get('total') as number;
    const isHovered = feature.get('hovered');
    const rippleWaves = feature.get('rippleWaves') as { radius: number; opacity: number }[] | undefined;

    // 如果有波纹效果（高亮状态）- 简约清爽风格
    if (rippleWaves && rippleWaves.length > 0) {
      const styles: Style[] = [];
      const config = DEVICE_STATUS_CONFIG[device.status] || DEVICE_STATUS_CONFIG.offline;
      const baseRadius = SPIDERFY_CONFIG_REF.pointRadius;

      // 中心点外发光效果
      styles.push(new Style({
        image: new Circle({
          radius: baseRadius + 4,
          fill: new Fill({ color: 'rgba(24, 144, 255, 0.15)' }),
        }),
      }));

      // 波纹样式：简约清爽的圆环
      rippleWaves.forEach(wave => {
        const rippleRadius = baseRadius + wave.radius;
        if (rippleRadius > baseRadius) {
          styles.push(new Style({
            image: new Circle({
              radius: rippleRadius,
              fill: new Fill({ color: 'transparent' }),
              stroke: new Stroke({
                color: `rgba(24, 144, 255, ${wave.opacity})`,
                width: 2,
              }),
            }),
          }));
        }
      });

      // 基础样式：原始大小的节点（最上层）
      styles.push(new Style({
        image: new Circle({
          radius: baseRadius,
          fill: new Fill({ color: config.color }),
          stroke: new Stroke({
            color: COLORS.primary,
            width: 3,
          }),
        }),
        text: new Text({
          text: String(index + 1),
          fill: new Fill({ color: COLORS.white }),
          font: 'bold 10px sans-serif',
          textAlign: 'center',
          textBaseline: 'middle',
        }),
      }));

      return styles;
    }

    // 悬停时使用放大样式
    if (isHovered) {
      return createSpiderfyPointHoverStyle(device, index, total);
    }

    return createSpiderfyPointStyle(device, index, total);
  }

  // 默认样式
  return new Style({});
}

/**
 * 绑定地图事件
 */
function bindMapEvents(
  map: Map,
  _deviceSource: VectorSource,
  _clusterSource: Cluster,
  callbacks: {
    onDeviceClick?: (device: MapDevice, pixel?: { x: number; y: number }) => void;
    onDeviceHover?: (device: MapDevice | null, pixel?: { x: number; y: number }) => void;
    onViewportChange?: (viewport: MapViewport) => void;
    onClusterClick?: (devices: MapDevice[]) => void;
    onZoomChange?: (zoom: number) => void;
    onSpiderfy?: (clusterFeature: Feature, center: number[], features: Feature[]) => void;
    onUnspiderfy?: () => void;
    onMapClick?: () => void;
  },
  deviceLayer: VectorLayer<VectorSource>,
  spiderfyLayer: VectorLayer<VectorSource>
): void {
  const { onDeviceClick, onDeviceHover, onViewportChange, onClusterClick, onZoomChange, onSpiderfy, onUnspiderfy, onMapClick } = callbacks;

  // Spiderfy 状态（在 bindMapEvents 作用域内）
  let isSpiderfied = false;

  // 点击事件
  map.on('click', (evt) => {
    // 触发地图点击回调（无论点击哪里都触发）
    onMapClick?.();

    // 首先检查是否点击了 spiderfy 图层的展开点
    const spiderfyFeatures = map.getFeaturesAtPixel(evt.pixel, {
      layerFilter: (layer) => layer === spiderfyLayer,
    });

    if (spiderfyFeatures.length > 0) {
      const feature = spiderfyFeatures[0] as Feature;

      // 点击了展开的设备点
      if (feature.get('spiderfyPoint')) {
        const device = feature.get('device') as MapDevice;
        const [x, y] = evt.pixel;
        onDeviceClick?.(device, { x, y });
        return;
      }

      // 点击了中心点，收起展开
      if (feature.get('spiderfyCenter')) {
        onUnspiderfy?.();
        isSpiderfied = false;
        return;
      }
    }

    // 检查设备图层
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
          const [x, y] = evt.pixel;
          onDeviceClick?.(device, { x, y });
        } else {
          // 多个设备
          const devices = featuresProp.map((f: Feature) => f.getProperties() as MapDevice);
          onClusterClick?.(devices);

          // 获取当前 zoom 级别
          const currentZoom = map.getView().getZoom() ?? 0;

          // 如果已经展开，收起
          if (isSpiderfied) {
            onUnspiderfy?.();
            isSpiderfied = false;
          } else if (currentZoom >= SPIDERFY_CONFIG.minZoom) {
            // 高缩放级别：使用 spiderfy 展开
            const geometry = feature.getGeometry();
            if (geometry) {
              const center = (geometry as Point).getCoordinates();
              onSpiderfy?.(feature, center, featuresProp as Feature[]);
              isSpiderfied = true;
            }
          } else {
            // 低缩放级别：放大地图展开聚合
            const view = map.getView();
            view.animate({
              center: evt.coordinate,
              zoom: currentZoom + 2,
              duration: 300,
            });
          }
        }
      } else {
        // 单独设备
        const device = feature.getProperties() as MapDevice;
        const [x, y] = evt.pixel;
        onDeviceClick?.(device, { x, y });
      }
    } else if (isSpiderfied) {
      // 点击空白区域，收起展开
      onUnspiderfy?.();
      isSpiderfied = false;
    }
  });

  // 悬停事件
  let hoveredFeature: Feature | null = null;
  let hoveredSpiderfyFeature: Feature | null = null;

  map.on('pointermove', (evt) => {
    // 首先检查 spiderfy 图层的展开点
    const spiderfyFeatures = map.getFeaturesAtPixel(evt.pixel, {
      layerFilter: (layer) => layer === spiderfyLayer,
    });

    if (spiderfyFeatures.length > 0) {
      const feature = spiderfyFeatures[0] as Feature;

      // 悬停在展开的设备点上
      if (feature.get('spiderfyPoint')) {
        const device = feature.get('device') as MapDevice;

        // 清除之前在设备图层的悬停状态
        if (hoveredFeature) {
          hoveredFeature.set('hovered', false);
          hoveredFeature = null;
        }

        // 更新 spiderfy 图层的悬停状态
        if (hoveredSpiderfyFeature && hoveredSpiderfyFeature !== feature) {
          hoveredSpiderfyFeature.set('hovered', false);
        }
        feature.set('hovered', true);
        hoveredSpiderfyFeature = feature;

        onDeviceHover?.(device, { x: evt.pixel[0], y: evt.pixel[1] });
        map.getTargetElement().style.cursor = 'pointer';
        return;
      }
    }

    // 清除 spiderfy 图层的悬停状态
    if (hoveredSpiderfyFeature) {
      hoveredSpiderfyFeature.set('hovered', false);
      hoveredSpiderfyFeature = null;
    }

    // 检查设备图层
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
  let lastZoom = map.getView().getZoom() ?? 0;

  map.on('moveend', () => {
    clearTimeout(moveEndTimeout);
    moveEndTimeout = setTimeout(() => {
      const view = map.getView();
      const currentZoom = view.getZoom() ?? 0;

      // 缩放级别变化时通知（用于动态调整聚合距离）
      if (currentZoom !== lastZoom) {
        lastZoom = currentZoom;
        onZoomChange?.(currentZoom);
      }

      if (onViewportChange) {
        const center = toLonLat(view.getCenter()!);
        const extent = view.calculateExtent(map.getSize());
        const bottomLeft = toLonLat([extent[0], extent[1]]);
        const topRight = toLonLat([extent[2], extent[3]]);

        onViewportChange({
          centerLng: center[0],
          centerLat: center[1],
          zoom: currentZoom,
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
