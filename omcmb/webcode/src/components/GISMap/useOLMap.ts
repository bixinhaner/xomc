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
import MultiPoint from 'ol/geom/MultiPoint';
import Polygon from 'ol/geom/Polygon';
import Draw from 'ol/interaction/Draw';
import Overlay from 'ol/Overlay';
import { getLength, offset as offsetCoordinate } from 'ol/sphere';
import { fromLonLat, toLonLat } from 'ol/proj';
import { containsCoordinate, buffer as bufferExtent, boundingExtent } from 'ol/extent';
import type { Extent } from 'ol/extent';
import { defaults as defaultControls } from 'ol/control';
import { Style, Stroke, Circle, Fill, Text } from 'ol/style';
import type { StyleLike } from 'ol/style/Style';
import type { AntennaSector, MapDevice, MapViewport, MapBounds } from '@core/types/map';
import {
  MAP_CONFIG,
  ANIMATION_CONFIG,
  COLORS,
  CLUSTER_CONFIG,
  DEVICE_STATUS_CONFIG,
  SPIDERFY_CONFIG,
  VIEWPORT_CULLING,
  getClusterDistanceForZoom,
  pickProgressiveSteps,
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
} from '@/utils/mapValidation';
import {
  DIRECTION_INDICATOR_LENGTH_PX,
  formatAntennaCoverageRange,
  resolveAntennaSectorRenderMode,
} from './antennaSectorRender';

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

/**
 * 将米数格式化为可读距离字符串
 * < 1000m 显示米，>= 1000m 显示千米
 */
function formatMeasureDistance(meters: number): string {
  if (meters >= 1000) return `${(meters / 1000).toFixed(2)} km`;
  return `${Math.round(meters)} m`;
}

interface UseOLMapOptions {
  /** 瓦片服务地址（离线模式） */
  tileUrl?: string;
  /** 默认中心点 [lng, lat] - 传入后作为备选中心点，优先级低于 center */
  defaultCenter?: [number, number];
  /** 默认缩放级别 - 传入后作为备选缩放，优先级低于 zoom */
  defaultZoom?: number;
  /** 中心点 [lng, lat] - 优先级最高，用于动态更新中心点 */
  center?: [number, number];
  /** 缩放级别 - 优先级最高，用于动态更新缩放 */
  zoom?: number;
  /** 最小缩放级别 */
  minZoom?: number;
  /** 最大缩放级别 */
  maxZoom?: number;
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
  /** 清除所有设备数据并重置图层 */
  clearDevices: () => void;
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
  /**
   * 动态调整瓦片并发上限（0 = 暂停队列，正常值建议 3-6）
   * 搜索期间调低可为 API 请求让出 TCP 连接；搜索完成后恢复
   * 注：当前为占位实现，OL 瓦片并发控制待后续版本落地
   */
  setTileConcurrency: (n: number) => void;
  /** 开启测距模式：地图进入划线量距交互，鼠标变十字 */
  startMeasure: () => void;
  /** 退出测距模式：清除折线和标注 */
  stopMeasure: () => void;
  updateAntennaSectors: (
    device: MapDevice | null,
    sectors: AntennaSector[],
    activeSectorNumber?: number,
  ) => void;
}

/**
 * OpenLayers 地图 Hook
 * 支持从服务端加载元数据（TileJSON），实现零配置切换
 */
export function useOLMap(options: UseOLMapOptions = {}): UseOLMapReturn {
  // 加载地图元数据
  const {
    metadata,
    loading: metadataLoading,
    tilesAvailable,
  } = useMapConfig();

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
    center,
    zoom,
    minZoom = config.minZoom,
    maxZoom = config.maxZoom,
    onDeviceClick,
    onDeviceHover,
    onViewportChange,
    onClusterClick,
    onMapClick,
    defaultCenter,
    defaultZoom,
  } = options;

  const mapRef = useRef<HTMLDivElement>(null);
  const mapInstanceRef = useRef<Map | null>(null);
  // 完整设备列表（性能 #15：视口裁剪时只渲染可视范围内的子集，
  // 其余设备保留在此 ref 中，平移/缩放后按新视口重算）
  const allDevicesRef = useRef<MapDevice[]>([]);
  // 视口裁剪函数引用（在 moveend 中调用，避免 bindMapEvents 签名漂移）
  const cullDevicesRef = useRef<(() => void) | null>(null);
  // clearHighlight 的 ref，供 moveend 等闭包内安全调用（避免 stale closure）
  const clearHighlightRef = useRef<(() => void) | null>(null);
  // 批量渲染 ID（用于取消上一批未完成的 rAF 渲染，防止并发写入 VectorSource）
  const renderBatchIdRef = useRef(0);
  // 强制保留渲染的设备 id（如搜索定位目标）：即便落在视口外也始终渲染，
  // 避免裁剪把搜索高亮的目标点剔除导致定位失败。
  const pinnedDeviceIdsRef = useRef<Set<string>>(new Set());
  const deviceSourceRef = useRef<VectorSource | null>(null);
  const clusterSourceRef = useRef<Cluster | null>(null);
  const deviceLayerRef = useRef<VectorLayer<VectorSource> | null>(null);
  const antennaSectorSourceRef = useRef<VectorSource | null>(null);
  const highlightFeatureRef = useRef<Feature | null>(null);
  // 水波纹动画定时器
  const pulseAnimationRef = useRef<ReturnType<typeof setInterval> | null>(null);
  // 波纹创建定时器
  const rippleCreateRef = useRef<ReturnType<typeof setInterval> | null>(null);
  // 搜索定位高亮自动清除定时器
  const highlightAutoClearRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  // 波纹状态数组
  const rippleWavesRef = useRef<{ radius: number; opacity: number }[]>([]);
  // 当前高亮设备的 ID（用于 renderDeviceFeatures 重建 feature 时恢复高亮状态）
  const highlightedDeviceIdRef = useRef<string | null>(null);
  // Spiderfy 状态
  const spiderfyLayerRef = useRef<VectorLayer<VectorSource> | null>(null);
  const spiderfySourceRef = useRef<VectorSource | null>(null);
  const isSpiderfiedRef = useRef(false);
  const spiderfiedCenterRef = useRef<number[] | null>(null);
  // 测距状态
  const measureSourceRef = useRef<VectorSource | null>(null);
  const measureLayerRef = useRef<VectorLayer<VectorSource> | null>(null);
  const measureDrawRef = useRef<Draw | null>(null);
  const isMeasuringRef = useRef(false);
  const measureOverlaysRef = useRef<Overlay[]>([]);
  // 新 API 数据到达标志：区分「视口移动触发的 cull（可能过渡态）」和「新数据权威 cull」
  // - true  → updateDevices 刚更新 allDevicesRef，必须执行全量重算（权威态）
  // - false → 仅视口移动，若 visible=0 说明是过渡态，保留旧 feature 不闪白
  const allDevicesChangedRef = useRef(false);
  // 高亮请求 ID（用于防止竞态条件）
  const highlightRequestIdRef = useRef(0);
  // “程序化飞行”计数器：progressiveFlyTo / flyTo / view.fit 起始 +1、结束 -1。
  // 在 bindMapEvents 的 moveend 处理里用它跳过中间档位的 onViewportChange，
  // 避免一次下钻发 N 个 /devices/geo 请求。用计数器而非布尔是为了能背丝安全地处理嵌套/重入。
  const isProgrammaticFlyingRef = useRef(0);

  const [isReady, setIsReady] = useState(false);

  const updateAntennaSectors = useCallback((
    device: MapDevice | null,
    sectors: AntennaSector[],
    activeSectorNumber?: number,
  ) => {
    const source = antennaSectorSourceRef.current;
    const map = mapInstanceRef.current;
    if (!source) return;
    source.clear();
    if (!device || !map || (map.getView().getZoom() ?? 0) < 13) return;

    const deviceCoordinate = fromLonLat([device.lng, device.lat]);
    const devicePixel = map.getPixelFromCoordinate(deviceCoordinate);

    for (const sector of sectors) {
      if (!sector.directionAvailable || sector.azimuth === undefined) continue;
      const bearing = sector.azimuth * Math.PI / 180;
      const isActive = sector.number === (activeSectorNumber ?? sectors[0]?.number);
      const referenceEnd = fromLonLat(offsetCoordinate([device.lng, device.lat], 1000, bearing));
      const referencePixel = map.getPixelFromCoordinate(referenceEnd);
      let directionEnd = referenceEnd;
      if (devicePixel && referencePixel) {
        const deltaX = referencePixel[0] - devicePixel[0];
        const deltaY = referencePixel[1] - devicePixel[1];
        const pixelLength = Math.hypot(deltaX, deltaY);
        if (pixelLength > 0) {
          directionEnd = map.getCoordinateFromPixel([
            devicePixel[0] + deltaX / pixelLength * DIRECTION_INDICATOR_LENGTH_PX,
            devicePixel[1] + deltaY / pixelLength * DIRECTION_INDICATOR_LENGTH_PX,
          ]);
        }
      }
      const direction = new Feature(new LineString([
        deviceCoordinate,
        directionEnd,
      ]));
      const directionColor = sector.coverageStatus === 'invalid_geometry'
        ? '#d48806'
        : sector.coverageStatus === 'incomplete' ? '#8c8c8c' : '#1677ff';
      direction.setStyle(new Style({
        stroke: new Stroke({
          color: directionColor,
          width: isActive ? 3 : 1.5,
          lineDash: sector.coverageAvailable ? undefined : [6, 4],
        }),
      }));
      source.addFeature(direction);

      if (!sector.coverageAvailable || sector.nearRadiusMeters === undefined || sector.farRadiusMeters === undefined) continue;
      if (sector.horizontalBeamwidth === undefined) continue;
      const halfBeam = sector.horizontalBeamwidth * Math.PI / 360;
      const outer: number[][] = [];
      const inner: number[][] = [];
      for (let step = 0; step <= 20; step++) {
        const angle = bearing - halfBeam + (2 * halfBeam * step) / 20;
        outer.push(fromLonLat(offsetCoordinate([device.lng, device.lat], sector.farRadiusMeters, angle)));
        inner.push(fromLonLat(offsetCoordinate([device.lng, device.lat], sector.nearRadiusMeters, angle)));
      }
      const outerStartPixel = map.getPixelFromCoordinate(outer[0]);
      const outerEndPixel = map.getPixelFromCoordinate(outer[outer.length - 1]);
      const renderMode = resolveAntennaSectorRenderMode(
        sector,
        outerStartPixel ? [outerStartPixel[0], outerStartPixel[1]] : undefined,
        outerEndPixel ? [outerEndPixel[0], outerEndPixel[1]] : undefined,
      );

      if (renderMode === 'polygon') {
        const ring = [...outer, ...inner.reverse()];
        ring.push(ring[0]);
        const coverage = new Feature(new Polygon([ring]));
        coverage.setStyle(new Style({
          fill: new Fill({ color: isActive ? 'rgba(22, 119, 255, 0.18)' : 'rgba(22, 119, 255, 0.07)' }),
          stroke: new Stroke({ color: '#1677ff', width: isActive ? 1.8 : 1 }),
        }));
        source.addFeature(coverage);
      }

      if (!isActive) continue;

      const nearCenter = fromLonLat(offsetCoordinate([device.lng, device.lat], sector.nearRadiusMeters, bearing));
      const farCenter = fromLonLat(offsetCoordinate([device.lng, device.lat], sector.farRadiusMeters, bearing));
      const radiusLine = new Feature(new LineString([nearCenter, farCenter]));
      radiusLine.setStyle(new Style({
        stroke: new Stroke({
          color: '#1677ff',
          width: renderMode === 'narrow' ? 2.5 : 1.5,
          lineDash: renderMode === 'narrow' ? [7, 5] : undefined,
        }),
      }));
      source.addFeature(radiusLine);

      const endpointStyle = (filled: boolean) => new Style({
        image: new Circle({
          radius: 4,
          fill: new Fill({ color: filled ? '#1677ff' : '#ffffff' }),
          stroke: new Stroke({ color: '#1677ff', width: 2 }),
        }),
      });
      const nearPoint = new Feature(new Point(nearCenter));
      nearPoint.setStyle(endpointStyle(false));
      source.addFeature(nearPoint);

      const rangeText = formatAntennaCoverageRange(sector);
      const farPoint = new Feature(new Point(farCenter));
      farPoint.setStyle(new Style({
        image: endpointStyle(true).getImage() ?? undefined,
        text: rangeText ? new Text({
          text: rangeText,
          offsetY: -16,
          font: '12px sans-serif',
          fill: new Fill({ color: '#0958d9' }),
          backgroundFill: new Fill({ color: 'rgba(255,255,255,0.92)' }),
          padding: [3, 5, 3, 5],
        }) : undefined,
      }));
      source.addFeature(farPoint);
    }
  }, []);

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

  // 展开 Spiderfy（多圈螺旋方式）（必须在 useEffect 之前定义，供 bindMapEvents 使用）
  // 同坐标设备数超过 SPIDERFY_CONFIG.maxNodes 时仅画前 N 个，防止数千个 Canvas 节点冻住主线程。
  // targetDeviceId 存在时，会被强制提权进入可视集（搜索定位场景）。
  const spiderfy = useCallback((
    _clusterFeature: Feature,
    center: number[],
    features: Feature[],
    targetDeviceId?: string,
  ) => {
    if (!mapInstanceRef.current || !spiderfySourceRef.current) return;

    // 如果已经展开，先收起
    if (isSpiderfiedRef.current) {
      unspiderfy();
    }

    const count = features.length;
    if (count <= 1) return;

    // 防爆屏：最多只展开 maxNodes 个节点
    const maxNodes = SPIDERFY_CONFIG.maxNodes ?? 60;
    let displayFeatures = features;
    if (count > maxNodes) {
      displayFeatures = features.slice(0, maxNodes);
      if (targetDeviceId) {
        const inVisible = displayFeatures.some(
          (f) => (f.getProperties() as MapDevice).id === targetDeviceId,
        );
        if (!inVisible) {
          const targetIdx = features.findIndex(
            (f) => (f.getProperties() as MapDevice).id === targetDeviceId,
          );
          if (targetIdx >= 0) {
            displayFeatures = displayFeatures.slice();
            displayFeatures[maxNodes - 1] = features[targetIdx];
          }
        }
      }
    }

    // 创建 spiderfy 图层的 feature
    const spiderfyFeatures: Feature[] = [];

    // 中心点 feature（count 仍使用真实总数，如 "2000"）
    const centerFeature = new Feature({
      geometry: new Point(center),
      spiderfyCenter: true,
      count,
    });
    spiderfyFeatures.push(centerFeature);

    // 按弧长间隔生成多圈分布表
    const displayCount = displayFeatures.length;
    const ringArcSpacing = SPIDERFY_CONFIG.ringArcSpacing ?? 28;
    const ringRadiusStep = SPIDERFY_CONFIG.ringRadiusStep ?? 45;
    const rings: { radius: number; count: number; angleStep: number }[] = [];
    let remaining = displayCount;
    let currentRadius = SPIDERFY_CONFIG.radius;
    while (remaining > 0) {
      const capacity = Math.max(
        12,
        Math.floor((2 * Math.PI * currentRadius) / ringArcSpacing),
      );
      const pts = Math.min(remaining, capacity);
      rings.push({
        radius: currentRadius,
        count: pts,
        angleStep: (2 * Math.PI) / pts,
      });
      remaining -= pts;
      currentRadius += ringRadiusStep;
    }

    const startAngle = -Math.PI / 2; // 从顶部开始
    const map = mapInstanceRef.current;
    const resolution = map.getView().getResolution()!;

    let cursor = 0;
    rings.forEach((ring) => {
      for (let k = 0; k < ring.count; k++) {
        const f = displayFeatures[cursor];
        const device = f.getProperties() as MapDevice;
        const angle = startAngle + k * ring.angleStep;
        const pixelOffset = [
          Math.cos(angle) * ring.radius,
          Math.sin(angle) * ring.radius,
        ];
        const pointCoordinate = [
          center[0] + pixelOffset[0] * resolution,
          center[1] - pixelOffset[1] * resolution, // Y 轴反向
        ];

        const lineFeature = new Feature({
          geometry: new LineString([center, pointCoordinate]),
          spiderfyLine: true,
        });
        spiderfyFeatures.push(lineFeature);

        const pointFeature = new Feature({
          geometry: new Point(pointCoordinate),
          spiderfyPoint: true,
          device: device,
          index: cursor,
          total: count,
        });
        spiderfyFeatures.push(pointFeature);

        cursor++;
      }
    });

    // 添加所有 feature 到 spiderfy 图层
    spiderfySourceRef.current.addFeatures(spiderfyFeatures);

    isSpiderfiedRef.current = true;
    spiderfiedCenterRef.current = center;

    // 触发地图重新渲染
    mapInstanceRef.current.render();
  }, [unspiderfy]);

  // 根据缩放级别动态调整聚合距离（按 CLUSTER_CONFIG.distanceTiers 查表）
  const updateClusterDistance = useCallback((zoom: number) => {
    if (!clusterSourceRef.current) return;

    const newDistance = getClusterDistanceForZoom(zoom);

    if (clusterSourceRef.current.getDistance() !== newDistance) {
      clusterSourceRef.current.setDistance(newDistance);
      clusterSourceRef.current.refresh();
      mapInstanceRef.current?.render();
    }
  }, []);

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
    // 等待瓦片可用性检查完成（null = 检查进行中）
    // 需要确认结果后再初始化，才能根据可用性选择正确的瓦片 URL
    if (tilesAvailable === null) return;

    // 标记初始化开始
    isMapInitializedRef.current = true;

    // 瓦片地址决策：只有确认离线瓦片可用时才走离线路径
    // 否则强制使用在线 OSM，避免 VITE_MAP_TILE_URL 的高优先级覆盖降级逻辑导致 502 报错
    const finalTileUrl = tilesAvailable === true
      ? (tileUrl || import.meta.env.VITE_MAP_TILE_URL || '/tiles/{z}/{x}/{y}.png')
      : MAP_CONFIG.osmTileUrl;
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

    antennaSectorSourceRef.current = new VectorSource();
    layers.push(new VectorLayer({ source: antennaSectorSourceRef.current, zIndex: 5 }));

    // 创建设备数据源
    deviceSourceRef.current = new VectorSource();

    // 创建聚合数据源（初始 distance 按初始 zoom 查表，避免首帧聚合距离不匹配）
    clusterSourceRef.current = new Cluster({
      source: deviceSourceRef.current,
      distance: getClusterDistanceForZoom(zoom ?? defaultZoom ?? config.defaultZoom),
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
        center: fromLonLat(center ?? defaultCenter ?? config.defaultCenter),
        zoom: zoom ?? defaultZoom ?? config.defaultZoom,
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
        isProgrammaticFlyingRef,
        isMeasuringRef,
      },
      deviceLayerRef.current,
      spiderfyLayerRef.current
    );

    // 视口裁剪重算（性能 #15）：地图平移/缩放结束后，按新视口重新喂点。
    // 带防抖，避免连续 moveend 频繁重建 feature。
    let cullTimeout: ReturnType<typeof setTimeout>;
    // 记录上一次 moveend 时的中心点，用于判断是否发生了「主动拖动」
    let lastMoveCenter: number[] | null = null;
    // 记录高亮时的 zoom，用于判断缩放幅度
    let highlightZoomLevel: number | null = null;
    mapInstanceRef.current.on('moveend', () => {
      clearTimeout(cullTimeout);
      // 程序化飞行中跳过 cull，飞行结束后 moveend（counter=0）自然触发一次，避免双重 cull
      if (isProgrammaticFlyingRef.current > 0) return;
      cullTimeout = setTimeout(() => {
        cullDevicesRef.current?.();
      }, 150);

      // 仅「用户手动拖动/缩放」时清除搜索高亮，程序化 flyTo 产生的 moveend 不处理
      const view = mapInstanceRef.current?.getView();
      if (view && highlightedDeviceIdRef.current && !(isProgrammaticFlyingRef.current > 0)) {
        const currentCenter = view.getCenter();
        const currentZoom = view.getZoom() ?? 0;

        // 记录高亮时的 zoom（第一次非程序化 moveend 时记录）
        if (highlightZoomLevel === null) {
          highlightZoomLevel = currentZoom;
        }

        // 条件1：中心点位移 > 120px（主动平移）
        let shouldClear = false;
        if (lastMoveCenter && currentCenter) {
          const dx = currentCenter[0] - lastMoveCenter[0];
          const dy = currentCenter[1] - lastMoveCenter[1];
          const resolution = view.getResolution() ?? 1;
          const pixelDist = Math.sqrt(dx * dx + dy * dy) / resolution;
          if (pixelDist > 120) shouldClear = true;
        }

        // 条件2：从高亮时的 zoom 缩小超过 2 级（用户明显缩小了地图）
        if (currentZoom < (highlightZoomLevel ?? currentZoom) - 2) {
          shouldClear = true;
        }

        if (shouldClear) {
          clearHighlightRef.current?.();
          highlightZoomLevel = null;
        }

        lastMoveCenter = currentCenter ? [...currentCenter] : null;
      } else if (view && !(isProgrammaticFlyingRef.current > 0)) {
        // 无高亮时重置 zoom 记录
        highlightZoomLevel = null;
        lastMoveCenter = view.getCenter() ? [...(view.getCenter()!)] : null;
      }
    });

    // 延迟设置 isReady，避免在 effect 中同步调用 setState 导致级联渲染
    // 使用 setTimeout 将状态更新推迟到下一个事件循环
    setTimeout(() => setIsReady(true), 0);

    // 清理函数
    return () => {
      if (mapInstanceRef.current) {
        // 清除测量图层和 Overlay
        if (measureDrawRef.current) {
          try { measureDrawRef.current.abortDrawing(); } catch { /* ignore */ }
          mapInstanceRef.current.removeInteraction(measureDrawRef.current);
          measureDrawRef.current = null;
        }
        measureOverlaysRef.current.forEach(o => mapInstanceRef.current?.removeOverlay(o));
        measureOverlaysRef.current = [];
        measureSourceRef.current?.clear();
        isMeasuringRef.current = false;

        mapInstanceRef.current.setTarget(undefined);
        mapInstanceRef.current = null;
      }
      deviceSourceRef.current = null;
      clusterSourceRef.current = null;
      deviceLayerRef.current = null;
      spiderfySourceRef.current = null;
      spiderfyLayerRef.current = null;
      measureSourceRef.current = null;
      measureLayerRef.current = null;
      // 重置初始化标记，允许重新初始化
      isMapInitializedRef.current = false;
    };
  }, [metadataLoading, tilesAvailable]);

  // 把一组设备渲染进 VectorSource
  // 分批策略：设备数 > BATCH_SIZE 时用 rAF 逐批写入，避免单帧构造大量 Feature 阻塞主线程。
  // RENDER_BATCH_SIZE 设为 3000，覆盖典型 pageSize（≤2000），使其走同步路径，
  // 避免 rAF 批次被 moveend/exitFlying 等多个触发点取消导致节点反复消失。
  const RENDER_BATCH_SIZE = 3000;
  const renderDeviceFeatures = useCallback((devices: MapDevice[]) => {
    if (!deviceSourceRef.current) return;

    // 递增 batchId，旧批次检测到 id 不匹配时自动放弃
    const batchId = ++renderBatchIdRef.current;

    // 在 clear() 之前快照高亮信息：clear() 会移除旧 feature，
    // 重建后需要把 highlighted/rippleWaves 恢复到新 feature 上，
    // 否则 moveend 触发的 cullDevicesToViewport 会让水波纹凭空消失。
    const highlightedId = highlightedDeviceIdRef.current;
    const currentWaves = highlightedId ? [...rippleWavesRef.current] : [];

    deviceSourceRef.current.clear();
    if (devices.length === 0) return;

    const makeFeature = (device: MapDevice) => {
      const feature = new Feature({
        geometry: new Point(fromLonLat([device.lng, device.lat])),
        ...device,
      });
      feature.setId(device.id);
      // 恢复高亮状态：moveend 重建 feature 后脉冲 interval 仍能正确写入
      if (highlightedId && device.id === highlightedId) {
        feature.set('highlighted', true, true);
        if (currentWaves.length > 0) {
          feature.set('rippleWaves', currentWaves, true);
        }
        // 更新引用，下一个 pulse tick 直接写入新 feature
        highlightFeatureRef.current = feature;
      }
      return feature;
    };

    // 小数据集：同步写入，无额外调度开销
    if (devices.length <= RENDER_BATCH_SIZE) {
      deviceSourceRef.current.addFeatures(devices.map(makeFeature));
      return;
    }

    // 大数据集：rAF 分批写入，每帧处理 RENDER_BATCH_SIZE 条
    let offset = 0;
    const source = deviceSourceRef.current;
    const flush = () => {
      if (renderBatchIdRef.current !== batchId || !source) return; // 已被新批次取消
      const slice = devices.slice(offset, offset + RENDER_BATCH_SIZE);
      source.addFeatures(slice.map(makeFeature));
      offset += RENDER_BATCH_SIZE;
      if (offset < devices.length) {
        requestAnimationFrame(flush);
      }
    };
    requestAnimationFrame(flush);
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

    const source = deviceSourceRef.current;
    const currentFeatures = source.getFeatures();

    // ── 过渡态保护 ──────────────────────────────────────────────────────────
    // 区分两种调用场景：
    //   A. 视口移动（moveend）触发：allDevicesChangedRef=false
    //      旧数据是宽视口的稀疏采样，缩放到新区域后可能 visible=0。
    //      此时不清空 source，保留旧 feature 显示，等待新 API 数据到达。
    //   B. updateDevices（新 API 数据）触发：allDevicesChangedRef=true
    //      数据是权威的，必须执行全量重算，不跳过。
    const isDataUpdate = allDevicesChangedRef.current;
    allDevicesChangedRef.current = false;

    if (!isDataUpdate && visible.length === 0 && currentFeatures.length > 0) {
      // 过渡态：视口已移动但新数据未到，保留旧 feature 防止空白闪烁
      return;
    }

    // visible=0 且是权威数据（新 API 确认该区域无设备）：直接全量清空
    if (visible.length === 0) {
      renderDeviceFeatures([]);
      return;
    }
    // ─────────────────────────────────────────────────────────────────────────

    // 增量更新策略：避免 clear() 产生空帧闪烁
    // 仅在删除量较小时使用增量删除（每次 removeFeature 触发一次 Cluster 重聚合）；
    // 删除量超过阈值时回退到全量 renderDeviceFeatures
    const INCREMENTAL_REMOVE_THRESHOLD = 200;
    if (currentFeatures.length > 0) {
      const visibleIds = new Set(visible.map(d => d.id));
      const currentIds = new Set(currentFeatures.map(f => String(f.getId())));

      const toAdd = visible.filter(d => !currentIds.has(d.id));
      const toRemove = currentFeatures.filter(f => !visibleIds.has(String(f.getId())));

      if (toAdd.length === 0 && toRemove.length <= INCREMENTAL_REMOVE_THRESHOLD) {
        // 纯删除且量小：逐个 removeFeature，不经过 clear()，无空帧
        toRemove.forEach(f => source.removeFeature(f));
        return;
      }
    }

    // 有新增 feature、删除量大、或 source 还是空的：走全量渲染
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
    // 标记为权威数据更新：cullDevicesToViewport 必须全量重算，不跳过
    allDevicesRef.current = devices;
    allDevicesChangedRef.current = true;
    cullDevicesToViewport();
  }, [unspiderfy, cullDevicesToViewport]);

  /** 清除所有设备数据并重置图层（外部调用：切换群组/全量刷新前先清空，避免残影） */
  const clearDevices = useCallback(() => {
    // 先收起 Spiderfy 展开状态
    if (isSpiderfiedRef.current) {
      unspiderfy();
    }

    // 停止水波纹动画，防止动画 tick 在 clear 之后继续读取已删除的 feature
    if (pulseAnimationRef.current) {
      clearInterval(pulseAnimationRef.current);
      pulseAnimationRef.current = null;
    }
    if (rippleCreateRef.current) {
      clearInterval(rippleCreateRef.current);
      rippleCreateRef.current = null;
    }
    rippleWavesRef.current = [];
    highlightFeatureRef.current = null;
    highlightedDeviceIdRef.current = null;

    // 取消进行中的批量渲染
    renderBatchIdRef.current++;

    // 清空设备数据和图层
    allDevicesRef.current = [];
    pinnedDeviceIdsRef.current.clear();
    deviceSourceRef.current?.clear();

    mapInstanceRef.current?.render();
  }, [unspiderfy]);

  /**
   * 动态调整瓦片并发上限
   * 占位实现：当前 OpenLayers XYZ source 不暴露并发数 API，此处记录期望值，
   * 待后续接入自定义 TileQueue 时落地真实控制逻辑。
   */
  const tileConcurrencyRef = useRef(6);
  const setTileConcurrency = useCallback((n: number) => {
    tileConcurrencyRef.current = Math.max(0, n);
  }, []);

  // 退出测距模式（先定义，供 startMeasure 调用）
  const stopMeasure = useCallback(() => {
    const map = mapInstanceRef.current;
    if (!map) return;

    // 放弃正在绘制中的线段
    if (measureDrawRef.current) {
      try { measureDrawRef.current.abortDrawing(); } catch { /* ignore */ }
      map.removeInteraction(measureDrawRef.current);
      measureDrawRef.current = null;
    }

    // 清除所有 Overlay 标注
    measureOverlaysRef.current.forEach(o => map.removeOverlay(o));
    measureOverlaysRef.current = [];

    // 清除测量折线
    measureSourceRef.current?.clear();

    isMeasuringRef.current = false;
    map.getTargetElement().style.cursor = '';
  }, []);

  // 开启测距模式
  const startMeasure = useCallback(() => {
    const map = mapInstanceRef.current;
    if (!map) return;

    // 如已在测距模式，先退出再重新开始
    if (isMeasuringRef.current) stopMeasure();

    // 创建/复用测量图层
    if (!measureSourceRef.current) {
      measureSourceRef.current = new VectorSource();
    } else {
      measureSourceRef.current.clear();
    }

    if (!measureLayerRef.current) {
      measureLayerRef.current = new VectorLayer({
        source: measureSourceRef.current,
        style: (feature) => {
          const geom = feature.getGeometry();
          if (geom instanceof LineString) {
            return [
              new Style({
                stroke: new Stroke({ color: '#1677ff', width: 2, lineDash: [6, 4] }),
              }),
              new Style({
                image: new Circle({
                  radius: 4,
                  fill: new Fill({ color: '#fff' }),
                  stroke: new Stroke({ color: '#1677ff', width: 2 }),
                }),
                geometry: () => new MultiPoint((geom as LineString).getCoordinates()),
              }),
            ];
          }
          return [];
        },
        zIndex: 200,
      });
      map.addLayer(measureLayerRef.current);
    }

    isMeasuringRef.current = true;
    map.getTargetElement().style.cursor = 'crosshair';

    // 累积已完成段的总距离（支持多段折线）
    let accumulatedDistance = 0;

    // 浮动 tooltip（跟随鼠标）
    const tooltipEl = document.createElement('div');
    tooltipEl.style.cssText =
      'background:#fff;border:1px solid #d9d9d9;border-radius:4px;padding:2px 8px;font-size:12px;' +
      'white-space:nowrap;pointer-events:none;box-shadow:0 2px 6px rgba(0,0,0,0.15);color:#333;';
    const tooltipOverlay = new Overlay({
      element: tooltipEl,
      offset: [14, -14],
      positioning: 'bottom-left' as const,
    });
    map.addOverlay(tooltipOverlay);
    measureOverlaysRef.current.push(tooltipOverlay);

    const draw = new Draw({
      source: measureSourceRef.current,
      type: 'LineString',
      style: [
        new Style({
          stroke: new Stroke({ color: 'rgba(22,119,255,0.6)', width: 2, lineDash: [6, 4] }),
        }),
        new Style({
          image: new Circle({
            radius: 4,
            fill: new Fill({ color: '#fff' }),
            stroke: new Stroke({ color: '#1677ff', width: 2 }),
          }),
        }),
      ],
    });

    // 绘制中：实时更新浮动 tooltip
    draw.on('drawstart', (evt) => {
      const sketchFeature = evt.feature;
      const sketchGeom = sketchFeature.getGeometry() as LineString;
      sketchGeom.on('change', () => {
        const coords = sketchGeom.getCoordinates();
        if (coords.length < 2) return;
        tooltipOverlay.setPosition(coords[coords.length - 1]);
        const currentLen = getLength(sketchGeom);
        const total = accumulatedDistance + currentLen;
        tooltipEl.textContent = total > 0
          ? `${formatMeasureDistance(total)}${accumulatedDistance > 0 ? ' (累计)' : ''}`
          : '';
      });
    });

    // 双击完成一段折线
    draw.on('drawend', (evt) => {
      const geom = evt.feature.getGeometry() as LineString;
      const segmentLength = getLength(geom);
      accumulatedDistance += segmentLength;

      // 在折线终点打固定标注
      const endCoord = geom.getLastCoordinate();
      const labelEl = document.createElement('div');
      labelEl.style.cssText =
        'background:#1677ff;color:#fff;border-radius:3px;padding:1px 7px;font-size:12px;' +
        'white-space:nowrap;pointer-events:none;transform:translateX(-50%);margin-bottom:4px;';
      labelEl.textContent = formatMeasureDistance(accumulatedDistance);
      const labelOverlay = new Overlay({
        element: labelEl,
        positioning: 'bottom-center' as const,
        stopEvent: false,
        offset: [0, -4],
      });
      labelOverlay.setPosition(endCoord);
      map.addOverlay(labelOverlay);
      measureOverlaysRef.current.push(labelOverlay);

      // 更新 tooltip 清空，等待下一段开始
      tooltipEl.textContent = '';
      tooltipOverlay.setPosition(undefined);
    });

    map.addInteraction(draw);
    measureDrawRef.current = draw;
  }, [stopMeasure]);

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

    // 进入程序化飞行：防止中间 moveend 触发一串 /devices/geo 请求
    isProgrammaticFlyingRef.current += 1;
    let flyingExited = false;
    const exitFlying = () => {
      if (flyingExited) return;
      flyingExited = true;
      isProgrammaticFlyingRef.current = Math.max(0, isProgrammaticFlyingRef.current - 1);
    };

    // 智能判断：如果zoom差值太小，使用单次动画
    if (Math.abs(currentZoom - targetZoom) < ANIMATION_CONFIG.progressiveZoomThreshold) {
      const animateOptions = {
        center: fromLonLat([lng, lat]),
        zoom: targetZoom,
        duration: 600,
        easing: Easing.easeOutCubic,
      };

      view.animate(animateOptions, () => {
        exitFlying();
        onComplete?.();
      });
      return;
    }

    // 渐进式缩放：按 PROGRESSIVE_ZOOM_STEPS 挑出的中间档位串接多段动画。
    // 如果中间档位为空（跳跃太小），则退化为单次动画。
    const targetCoord = fromLonLat([lng, lat]);
    const zoomDiff = targetZoom - currentZoom;
    const steps = pickProgressiveSteps(currentZoom, targetZoom);

    if (steps.length === 0) {
      const fallbackDuration = Math.min(1400, Math.max(900, zoomDiff * 90));
      const animateOptions = {
        center: targetCoord,
        zoom: targetZoom,
        duration: fallbackDuration,
        easing: Easing.progressive,
      };
      view.animate(animateOptions, () => {
        exitFlying();
        onComplete?.();
      });
      return;
    }

    let i = 0;
    let finished = false;
    const runStep = () => {
      if (finished) return;
      if (i >= steps.length) {
        finished = true;
        exitFlying();
        onComplete?.();
        return;
      }
      const step = steps[i++];
      view.animate(
        {
          center: targetCoord,
          zoom: step.zoom,
          duration: step.duration,
          easing: Easing.easeOutCubic,
        },
        (complete: boolean) => {
          if (!complete) {
            // 被用户拖动 / 新 animate 打断：放弃后续 step，但仍触发 onComplete
            // 让外层（如 search 探测流程）能拿到回调继续推进。
            if (finished) return;
            finished = true;
            exitFlying();
            onComplete?.();
            return;
          }
          runStep();
        },
      );
    };
    runStep();
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
    isProgrammaticFlyingRef.current += 1;
    let exited = false;
    const exitFlying = () => {
      if (exited) return;
      exited = true;
      isProgrammaticFlyingRef.current = Math.max(0, isProgrammaticFlyingRef.current - 1);
    };
    view.animate(
      {
        center: fromLonLat([lng, lat]),
        zoom: finalZoom,
        duration: ANIMATION_CONFIG.flyDuration,
      },
      () => {
        exitFlying();
        onComplete?.();
      },
    );
  }, [progressiveFlyTo]);

  // 取消高亮（必须在 highlightDevice 之前定义）
  const clearHighlight = useCallback(() => {
    // 清除自动清除定时器
    if (highlightAutoClearRef.current) {
      clearTimeout(highlightAutoClearRef.current);
      highlightAutoClearRef.current = null;
    }
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

    // 清除波纹状态 & 设备 ID 记录
    rippleWavesRef.current = [];
    highlightedDeviceIdRef.current = null;

    if (highlightFeatureRef.current) {
      // silent=true：不触发 Cluster 重聚合
      highlightFeatureRef.current.set('highlighted', false, true);
      highlightFeatureRef.current.set('rippleWaves', undefined, true);
      highlightFeatureRef.current = null;
    }

    // 取消搜索定位 pin（性能 #15）：高亮结束后该设备恢复受裁剪约束，
    // 避免 pin 集合无界增长。下次正常裁剪会按视口决定是否渲染。
    if (pinnedDeviceIdsRef.current.size > 0) {
      pinnedDeviceIdsRef.current.clear();
    }

    // 触发一次重绘，确保高亮立即消失（silent 不会自动触发 Cluster 重绘）
    mapInstanceRef.current?.render();
  }, []);

  // 挂到 ref，让 moveend 闭包可安全调用最新版本（避免 stale closure）
  clearHighlightRef.current = clearHighlight;

  // 高亮设备（水波纹动画）
  const highlightDevice = useCallback((deviceId: string) => {
    if (!deviceSourceRef.current || !mapInstanceRef.current) return;

    // 先清除之前的高亮
    clearHighlight();

    const feature = deviceSourceRef.current.getFeatureById(deviceId);
    if (feature) {
      // silent=true：不触发 Cluster 重聚合；flyTo 动画会在每帧调 render，高亮会立即可见
      feature.set('highlighted', true, true);
      highlightFeatureRef.current = feature as Feature;
      highlightedDeviceIdRef.current = deviceId;

      // 飞行到设备位置
      const geometry = feature.getGeometry();
      if (geometry) {
        const coordinate = (geometry as Point).getCoordinates();
        const lonLat = toLonLat(coordinate);
        flyTo(lonLat[0], lonLat[1]);
      }

      // 初始化波纹状态：最多 3 个波纹
      rippleWavesRef.current = [];

      // 创建新波纹的函数
      const createRipple = () => {
        if (!highlightFeatureRef.current) return;
        rippleWavesRef.current.push({
          radius: 0,    // 从中心开始
          opacity: 0.9, // 高初始透明度，让波纹一出现就明显
        });
      };

      // 立即创建第一个波纹
      createRipple();

      // 定时创建新波纹（每 600ms）
      // 最多创建 3 个波纹（1.8s），之后停止创建，让最后一个波纹自然消散（~2.3s）
      // 总体动画约 3-4s，符合「搜索定位高亮」的最佳实践时长。
      const MAX_RIPPLE_CREATE_COUNT = 3;
      let rippleCreateCount = 0;
      rippleCreateRef.current = setInterval(() => {
        rippleCreateCount++;
        createRipple();
        // 最多同时存在 4 个波纹
        if (rippleWavesRef.current.length > 4) {
          rippleWavesRef.current.shift();
        }
        // 达到上限后停止创建，让最后一批波纹自然衰减消失
        if (rippleCreateCount >= MAX_RIPPLE_CREATE_COUNT) {
          if (rippleCreateRef.current) {
            clearInterval(rippleCreateRef.current);
            rippleCreateRef.current = null;
          }
        }
      }, 600);

      // 波纹扩散动画（每 40ms 更新）
      pulseAnimationRef.current = setInterval(() => {
        if (!highlightFeatureRef.current) {
          if (pulseAnimationRef.current) {
            clearInterval(pulseAnimationRef.current);
            pulseAnimationRef.current = null;
          }
          return;
        }

        // 更新所有波纹的状态（半径缩小：max ~36px，衰减加快避免视觉过大）
        rippleWavesRef.current = rippleWavesRef.current
          .map(wave => ({
            radius: wave.radius + 0.6,
            opacity: wave.opacity - 0.017,
          }))
          .filter(wave => wave.opacity > 0);

        // silent=true：不触发 source change 事件，避免 Cluster 每帧重新聚合
        // （map.render() 会直接重绘图层，style 函数会读到最新的 rippleWaves 值）
        highlightFeatureRef.current.set('rippleWaves', [...rippleWavesRef.current], true);

        if (mapInstanceRef.current) {
          mapInstanceRef.current.render();
        }
      }, 40);
    }
  }, [flyTo, clearHighlight]);

  // 刷新地图尺寸
  const updateSize = useCallback(() => {
    mapInstanceRef.current?.updateSize();
  }, []);

  // 获取当前 zoom 级别
  const getZoom = useCallback((): number => {
    const fallbackZoom = zoom ?? defaultZoom ?? config.defaultZoom;
    if (!mapInstanceRef.current) return fallbackZoom;
    return mapInstanceRef.current.getView().getZoom() ?? fallbackZoom;
  }, [zoom, defaultZoom, config.defaultZoom]);

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
  //
  // 设计要点（修复 Issue D：search-progressive-locate 主线程冻结）：
  //   1. 两段飞行：先 flyTo 到 searchIntermediateZoom（约 13）让聚合可分辨，
  //      再决定是否飞到 maxHighlightZoom。一口气到 18 会跨过聚合可见区间，
  //      clusterSource 在该 zoom 下可能完全没有目标 feature，落到下面的 "找不到"
  //      分支导致静默失败。
  //   2. 全程用 flyTo onComplete 串接，废弃旧的 moveend + setTimeout(150) 模式，
  //      避免 moveend 因为渐进多段动画连发或被拖拽抢占而错过时机。
  //   3. clusterSource.getFeatures() 不是同步保证最新的，所以加重试循环：
  //      最多 12 次 × 50ms。每次 tick 前用 highlightRequestIdRef 校验请求是否过期。
  const highlightAndSpiderfyIfNeeded = useCallback((device: MapDevice, skipFlyTo = false) => {
    if (!mapInstanceRef.current || !clusterSourceRef.current || !spiderfySourceRef.current) return;

    // 校验坐标，避免 NaN/null 直接喂给 fromLonLat
    if (
      typeof device.lng !== 'number' ||
      typeof device.lat !== 'number' ||
      !Number.isFinite(device.lng) ||
      !Number.isFinite(device.lat)
    ) {
      return;
    }

    // 性能 #15：把搜索/定位目标 pin 住并立即重算裁剪，
    // 确保即便目标当前落在视口外，也已渲染进 deviceSource，
    // 后续聚合查找（clusterSource）才能命中它。
    if (allDevicesRef.current.length > VIEWPORT_CULLING.enableThreshold) {
      pinnedDeviceIdsRef.current.add(device.id);
      cullDevicesToViewport();
    }

    // 递增请求 ID，用于防止竞态条件
    const currentRequestId = ++highlightRequestIdRef.current;
    const isStale = () => currentRequestId !== highlightRequestIdRef.current;

    // 先清除之前的高亮和展开
    clearHighlight();
    if (isSpiderfiedRef.current) {
      unspiderfy();
    }

    const intermediateZoom = ANIMATION_CONFIG.searchIntermediateZoom ?? 13;
    const maxZoom = ANIMATION_CONFIG.maxHighlightZoom || 18;

    // 给目标 feature 上水波纹高亮的副作用
    const applyRippleHighlight = (feature: Feature) => {
      // 防御性清理：确保没有旧定时器残留（调用方通常已 clearHighlight，此处双重保险）
      if (pulseAnimationRef.current) {
        clearInterval(pulseAnimationRef.current);
        pulseAnimationRef.current = null;
      }
      if (rippleCreateRef.current) {
        clearInterval(rippleCreateRef.current);
        rippleCreateRef.current = null;
      }
      highlightFeatureRef.current = feature;
      // 记录设备 ID：renderDeviceFeatures 重建 feature 时用此 ID 恢复高亮引用
      highlightedDeviceIdRef.current = device.id;
      rippleWavesRef.current = [];

      const createRipple = () => {
        if (!highlightFeatureRef.current) return;
        rippleWavesRef.current.push({ radius: 0, opacity: 1.0 });
      };
      createRipple();

      // 只再创建 1 个额外波纹，共 2 个波，之后停止
      let extraCreated = 0;
      rippleCreateRef.current = setInterval(() => {
        extraCreated++;
        createRipple();
        if (rippleWavesRef.current.length > 4) {
          rippleWavesRef.current.shift();
        }
        if (extraCreated >= 1) {
          clearInterval(rippleCreateRef.current!);
          rippleCreateRef.current = null;
        }
      }, 480);

      pulseAnimationRef.current = setInterval(() => {
        if (!highlightFeatureRef.current) {
          if (pulseAnimationRef.current) {
            clearInterval(pulseAnimationRef.current);
            pulseAnimationRef.current = null;
          }
          return;
        }
        rippleWavesRef.current = rippleWavesRef.current
          .map((wave) => ({
            radius: wave.radius + 0.4,    // 扩散速度放慢：10px/s，波纹更小
            opacity: wave.opacity - 0.030, // 衰减加快：~1.3s 消失，不长期驻留
          }))
          .filter((wave) => wave.opacity > 0);
        highlightFeatureRef.current.set('rippleWaves', [...rippleWavesRef.current], true);
        if (mapInstanceRef.current) {
          mapInstanceRef.current.render();
        }
        // 所有波纹消散后自动停止 pulseAnimation
        if (rippleWavesRef.current.length === 0 && !rippleCreateRef.current) {
          clearInterval(pulseAnimationRef.current!);
          pulseAnimationRef.current = null;
        }
      }, 40);

      // 2s 后强制清除高亮（无论波纹是否消散完毕）
      if (highlightAutoClearRef.current) clearTimeout(highlightAutoClearRef.current);
      highlightAutoClearRef.current = setTimeout(() => {
        highlightAutoClearRef.current = null;
        clearHighlight();
      }, 2000);
    };

    // 在 cluster source 中重试查找目标设备所在聚合
    // 重试是必要的：渐进式 flyTo 完成时 clusterSource 可能尚未发出 'change'
    const detectClusterAndAct = (
      onClusterFound: (clusterFeature: Feature, deviceFeature: Feature) => void,
    ) => {
      let attempts = 0;
      const maxAttempts = 12;

      const tick = () => {
        if (isStale()) return;
        if (!clusterSourceRef.current || !mapInstanceRef.current) return;

        const clusterFeatures = clusterSourceRef.current.getFeatures();
        let targetClusterFeature: Feature | null = null;
        let targetDeviceFeature: Feature | null = null;

        for (const cf of clusterFeatures) {
          const features = cf.get('features');
          if (features && Array.isArray(features)) {
            for (const f of features as Feature[]) {
              const props = f.getProperties() as MapDevice;
              if (props.id === device.id) {
                targetClusterFeature = cf;
                targetDeviceFeature = f;
                break;
              }
            }
          }
          if (targetClusterFeature) break;
        }

        if (targetClusterFeature && targetDeviceFeature) {
          onClusterFound(targetClusterFeature, targetDeviceFeature);
          return;
        }

        attempts++;
        if (attempts < maxAttempts) {
          setTimeout(tick, 50);
        }
      };
      tick();
    };

    // 阶段二：到位后判断聚合 / 单点，再决定 spiderfy 或直接高亮
    const onArrivedAtFinalZoom = () => {
      if (isStale()) return;
      detectClusterAndAct((clusterFeature, deviceFeature) => {
        if (isStale()) return;
        const featuresInCluster = clusterFeature.get('features') as Feature[] | undefined;
        const count = featuresInCluster?.length ?? 1;

        if (count > 1 && featuresInCluster) {
          const geometry = clusterFeature.getGeometry();
          if (!geometry) return;
          const center = (geometry as Point).getCoordinates();
          spiderfy(clusterFeature, center, featuresInCluster, device.id);

          requestAnimationFrame(() => {
            if (isStale() || !spiderfySourceRef.current) return;
            const spiderfyFeatures = spiderfySourceRef.current.getFeatures();
            for (const sf of spiderfyFeatures) {
              if (sf.get('spiderfyPoint')) {
                const sfDevice = sf.get('device') as MapDevice;
                if (sfDevice.id === device.id) {
                  applyRippleHighlight(sf as Feature);
                  break;
                }
              }
            }
          });
        } else {
          // silent=true + 立即 render：不触发 Cluster 重聚合，且选中环在波纹出现之前就可见
          deviceFeature.set('highlighted', true, true);
          mapInstanceRef.current?.render();
          applyRippleHighlight(deviceFeature);
        }
      });
    };

    // 阶段一：飞到中间 zoom，看是否仍然落在聚合中
    const onArrivedAtIntermediate = () => {
      if (isStale() || !mapInstanceRef.current) return;
      detectClusterAndAct((clusterFeature, deviceFeature) => {
        if (isStale()) return;
        const featuresInCluster = clusterFeature.get('features') as Feature[] | undefined;
        const count = featuresInCluster?.length ?? 1;

        if (count > 1 && featuresInCluster) {
          // 仍在聚合中：直接 spiderfy，无需再爬到 zoom 18，
          // 避免把同坐标设备拆散后反而找不到。
          const geometry = clusterFeature.getGeometry();
          if (!geometry) return;
          const center = (geometry as Point).getCoordinates();
          spiderfy(clusterFeature, center, featuresInCluster, device.id);

          requestAnimationFrame(() => {
            if (isStale() || !spiderfySourceRef.current) return;
            const spiderfyFeatures = spiderfySourceRef.current.getFeatures();
            for (const sf of spiderfyFeatures) {
              if (sf.get('spiderfyPoint')) {
                const sfDevice = sf.get('device') as MapDevice;
                if (sfDevice.id === device.id) {
                  applyRippleHighlight(sf as Feature);
                  break;
                }
              }
            }
          });
          return;
        }

        // 单点：可以放心继续推到最大 zoom；但若调用方明确 skipFlyTo，则尊重契约直接高亮。
        const view = mapInstanceRef.current!.getView();
        const nowZoom = view.getZoom() ?? intermediateZoom;
        if (!skipFlyTo && nowZoom < maxZoom - 0.5) {
          flyTo(device.lng, device.lat, maxZoom, { onComplete: onArrivedAtFinalZoom });
        } else {
          // silent=true + 立即 render：不触发 Cluster 重聚合，且选中环立即可见
          deviceFeature.set('highlighted', true, true);
          mapInstanceRef.current?.render();
          applyRippleHighlight(deviceFeature);
        }
      });
    };

    if (skipFlyTo) {
      // 由调用方负责飞行（如轨迹播放），这里只跑探测/高亮逻辑
      onArrivedAtIntermediate();
      return;
    }

    const currentZoom = mapInstanceRef.current.getView().getZoom() ?? 0;
    if (currentZoom >= intermediateZoom - 0.5) {
      // 已经在中间 zoom 或更高：跳过第一段，直接探测/上推
      onArrivedAtIntermediate();
    } else {
      flyTo(device.lng, device.lat, intermediateZoom, { onComplete: onArrivedAtIntermediate });
    }
  }, [flyTo, clearHighlight, unspiderfy, spiderfy, cullDevicesToViewport]);

  return {
    mapRef,
    mapInstanceRef,
    updateDevices,
    clearDevices,
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
    setTileConcurrency,
    startMeasure,
    stopMeasure,
    updateAntennaSectors,
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

    // 如果有波纹效果（高亮状态）
    if (rippleWaves && rippleWaves.length > 0) {
      const styles: Style[] = [];
      const config = DEVICE_STATUS_CONFIG[device.status] || DEVICE_STATUS_CONFIG.offline;
      const baseRadius = SPIDERFY_CONFIG_REF.pointRadius;

      // 波纹样式（按半径大到小压栈，小波纹在最上层）
      const sortedSpiderfyWaves = [...rippleWaves].sort((a, b) => b.radius - a.radius);
      sortedSpiderfyWaves.forEach(wave => {
        const rippleRadius = baseRadius + wave.radius;
        if (rippleRadius > baseRadius) {
          styles.push(new Style({
            image: new Circle({
              radius: rippleRadius,
              fill: new Fill({ color: 'transparent' }),
              stroke: new Stroke({
                color: `rgba(24, 144, 255, ${Math.max(0, Math.min(1, wave.opacity))})`,
                width: 4,
              }),
            }),
          }));
        }
      });

      // 选中环：白色衬底
      styles.push(new Style({
        image: new Circle({
          radius: baseRadius + 14,
          fill: new Fill({ color: 'transparent' }),
          stroke: new Stroke({
            color: 'rgba(255, 255, 255, 0.85)',
            width: 6,
          }),
        }),
      }));

      // 选中环：蓝色
      styles.push(new Style({
        image: new Circle({
          radius: baseRadius + 14,
          fill: new Fill({ color: 'transparent' }),
          stroke: new Stroke({
            color: 'rgba(24, 144, 255, 0.95)',
            width: 2.5,
          }),
        }),
      }));

      // 外发光晕
      styles.push(new Style({
        image: new Circle({
          radius: baseRadius + 7,
          fill: new Fill({ color: 'rgba(24, 144, 255, 0.30)' }),
        }),
      }));

      // 内发光晕
      styles.push(new Style({
        image: new Circle({
          radius: baseRadius + 3,
          fill: new Fill({ color: 'rgba(24, 144, 255, 0.50)' }),
        }),
      }));

      // 白色衬底
      styles.push(new Style({
        image: new Circle({
          radius: baseRadius + 1,
          fill: new Fill({ color: 'rgba(255, 255, 255, 0.95)' }),
        }),
      }));

      // 基础样式：原始大小的节点（最上层）
      styles.push(new Style({
        image: new Circle({
          radius: baseRadius,
          fill: new Fill({ color: config.color }),
          stroke: new Stroke({
            color: COLORS.primary,
            width: 4,
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
    onSpiderfy?: (clusterFeature: Feature, center: number[], features: Feature[], targetDeviceId?: string) => void;
    onUnspiderfy?: () => void;
    onMapClick?: () => void;
    /**
     * “程序化飞行中”计数器的 ref。为 >0 时，表示中间动画档位，
     * moveend 不应在此时上报 viewport，避免引起外部重复拉取 /devices/geo。
     */
    isProgrammaticFlyingRef?: React.MutableRefObject<number>;
    /** 测距模式中时为 true，此时屏蔽设备点击/hover，避免误触 */
    isMeasuringRef?: React.MutableRefObject<boolean>;
  },
  deviceLayer: VectorLayer<VectorSource>,
  spiderfyLayer: VectorLayer<VectorSource>
): void {
  const { onDeviceClick, onDeviceHover, onViewportChange, onClusterClick, onZoomChange, onSpiderfy, onUnspiderfy, onMapClick, isProgrammaticFlyingRef, isMeasuringRef } = callbacks;

  // Spiderfy 状态（在 bindMapEvents 作用域内）
  let isSpiderfied = false;

  // 点击事件
  map.on('click', (evt) => {
    // 测距模式：屏蔽设备点击交互，由 Draw interaction 处理
    if (isMeasuringRef?.current) return;

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

          // 如果已经展开，收起
          if (isSpiderfied) {
            onUnspiderfy?.();
            isSpiderfied = false;
            return;
          }

          // 智能路由：看 features 的屏幕 bbox 对角线
          //   1) > clickExpandThresholdPx → 散得开，view.fit 下钻（类 Google Maps）
          //   2) ≤ 阈值 → spiderfy
          const ptsArr = featuresProp as Feature[];
          const coords: number[][] = [];
          for (const f of ptsArr) {
            const g = f.getGeometry();
            if (g) coords.push((g as Point).getCoordinates());
          }
          let pixelDiagonal = 0;
          if (coords.length > 1) {
            let minX = Infinity;
            let minY = Infinity;
            let maxX = -Infinity;
            let maxY = -Infinity;
            for (const c of coords) {
              const px = map.getPixelFromCoordinate(c);
              if (!px) continue;
              if (px[0] < minX) minX = px[0];
              if (px[0] > maxX) maxX = px[0];
              if (px[1] < minY) minY = px[1];
              if (px[1] > maxY) maxY = px[1];
            }
            if (Number.isFinite(minX)) {
              const dx = maxX - minX;
              const dy = maxY - minY;
              pixelDiagonal = Math.sqrt(dx * dx + dy * dy);
            }
          }

          const currentZoom = map.getView().getZoom() ?? 0;
          const canExpand = pixelDiagonal > CLUSTER_CONFIG.clickExpandThresholdPx && coords.length > 1;

          if (canExpand) {
            // 下钻：适配到 bbox，限上不超过 maxHighlightZoom，避免一口气捆到最大级
            const extent = boundingExtent(coords);
            const view = map.getView();
            const targetMaxZoom = Math.min(
              (view.getMaxZoom?.() ?? 20),
              ANIMATION_CONFIG.maxHighlightZoom ?? 18,
            );
            // 与 flyTo/progressiveFlyTo 同机制：进入“程序化飞行”计数，
            // 让 moveend 只在最后一个档位上报 viewport。双插锐拍防护用 setTimeout 兑底。
            if (isProgrammaticFlyingRef) {
              isProgrammaticFlyingRef.current += 1;
              const decay = () => {
                isProgrammaticFlyingRef.current = Math.max(
                  0,
                  isProgrammaticFlyingRef.current - 1,
                );
              };
              // view.fit 没有 callback，按动画 duration + 小量 buffer 释放计数
              setTimeout(decay, 450);
            }
            view.fit(extent, {
              padding: [80, 80, 80, 80],
              duration: 400,
              maxZoom: targetMaxZoom,
              easing: Easing.easeOutCubic,
            });
            return;
          }

          // 散不开：走 spiderfy
          if (currentZoom >= SPIDERFY_CONFIG.minZoom) {
            const geometry = feature.getGeometry();
            if (geometry) {
              const center = (geometry as Point).getCoordinates();
              onSpiderfy?.(feature, center, featuresProp as Feature[]);
              isSpiderfied = true;
            }
          } else {
            // zoom 太低，不适合 spiderfy，先推一档
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
    // 测距模式：保持十字光标，不处理设备 hover
    if (isMeasuringRef?.current) {
      // 只在有悬停状态需要清除时才回调，避免每帧无条件触发
      if (hoveredFeature) { hoveredFeature.set('hovered', false); hoveredFeature = null; onDeviceHover?.(null); }
      if (hoveredSpiderfyFeature) { hoveredSpiderfyFeature.set('hovered', false); hoveredSpiderfyFeature = null; }
      return;
    }

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
      // 即使处于程序化飞行中也需要触发，这样聚合距离才能随 zoom 缩放同步变化
      if (currentZoom !== lastZoom) {
        lastZoom = currentZoom;
        onZoomChange?.(currentZoom);
      }

      // 中间档位跳过 viewport 上报：避免一次下钻/飞行发 N 个 /devices/geo
      if (isProgrammaticFlyingRef && isProgrammaticFlyingRef.current > 0) {
        return;
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
