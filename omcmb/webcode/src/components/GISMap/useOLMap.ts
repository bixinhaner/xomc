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
import type ImageTile from 'ol/ImageTile';
import Feature from 'ol/Feature';
import Point from 'ol/geom/Point';
import LineString from 'ol/geom/LineString';
import { fromLonLat, toLonLat } from 'ol/proj';
import { containsCoordinate, buffer as bufferExtent, boundingExtent } from 'ol/extent';
import type { Extent } from 'ol/extent';
import { defaults as defaultControls } from 'ol/control';
import { unByKey } from 'ol/Observable';
import type { EventsKey } from 'ol/events';
import { Style, Stroke, Circle, Fill, Text } from 'ol/style';
import type { StyleLike } from 'ol/style/Style';
import type { MapDevice, MapViewport, MapBounds } from '@core/types/map';
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
  /**
   * 中心点就绪信号（默认 true，向后兼容）。
   * false 时 OL 延迟初始化，等待可靠中心点到达（stats/离线地图元数据加载完成）后才创建地图实例。
   * 这样可保证 OL 首次以正确坐标初始化，设备首屏即在可视范围内，无需 flyTo 修正。
   */
  centerReady?: boolean;
}

interface UseOLMapReturn {
  /** 地图容器 ref */
  mapRef: React.RefObject<HTMLDivElement | null>;
  /** 地图实例 ref */
  mapInstanceRef: React.MutableRefObject<Map | null>;
  /** 更新设备数据 */
  updateDevices: (devices: MapDevice[]) => void;
  /** 清除所有设备数据（筛选条件变化时调用） */
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
   * 动态调整瓦片并发数（0 = 暂停队列，正常为 3）
   * 搜索时可临时降低，为 API 请求让出连接
   */
  setTileConcurrency: (n: number) => void;
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
    centerReady = true,
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
  // 波纹状态数组（已废弃，保留字段以防旧代码引用）
  const rippleWavesRef = useRef<{ radius: number; opacity: number }[]>([]);
  // rAF 水波纹动画 ID（新实现）
  const rippleRafRef = useRef<number | null>(null);
  // 水波纹开始时间戳
  const rippleStartTimeRef = useRef<number>(0);
  // Spiderfy 状态
  const spiderfyLayerRef = useRef<VectorLayer<VectorSource> | null>(null);
  const spiderfySourceRef = useRef<VectorSource | null>(null);
  const isSpiderfiedRef = useRef(false);
  const spiderfiedCenterRef = useRef<number[] | null>(null);
  // zoom 刷新 spiderfy 所需的快照：展开时保存，zoom 变化时用新 resolution 重算坐标
  const spiderfyDisplayFeaturesRef = useRef<Feature[]>([]);
  // 真实聚合总数（含截断部分）：refresh 时保持中心球数字与展示总数一致
  const spiderfyTotalCountRef = useRef<number>(0);
  // 瓦片并发控制（可动态调整，搜索时降低以让出连接给 API）
  const tileConcurrencyRef = useRef<number>(3);
  const activeTileCountRef = useRef<number>(0);
  const tileLoadQueueRef = useRef<Array<() => void>>([]);
  // drainTileQueue 用 ref 保存，供瓦片加载回调和 setTileConcurrency 共用，避免逻辑重复
  const drainTileQueueRef = useRef<() => void>(() => {});
  // 高亮请求 ID（用于防止竞态条件）
  const highlightRequestIdRef = useRef(0);
  // “程序化飞行”计数器：progressiveFlyTo / flyTo / view.fit 起始 +1、结束 -1。
  // 在 bindMapEvents 的 moveend 处理里用它跳过中间档位的 onViewportChange，
  // 避免一次下钻发 N 个 /devices/geo 请求。用计数器而非布尔是为了能背丝安全地处理嵌套/重入。
  const isProgrammaticFlyingRef = useRef(0);
  // Cluster distance 过渡动画 rAF ID（进行中时非 null）
  const clusterAnimRafRef = useRef<number | null>(null);
  // true = 正在执行 cluster distance 过渡动画，change 监听器期间跳过 birthTime 标记
  const clusterDistAnimActiveRef = useRef(false);

  const [isReady, setIsReady] = useState(false);

  // 收起 Spiderfy 展开（必须在 useEffect 之前定义，供 bindMapEvents 使用）
  const unspiderfy = useCallback(() => {
    if (!isSpiderfiedRef.current || !spiderfySourceRef.current) return;

    // 清除 spiderfy 图层的所有 feature
    spiderfySourceRef.current.clear();
    isSpiderfiedRef.current = false;
    spiderfiedCenterRef.current = null;
    // 同步清理快照，避免内存泄漏
    spiderfyDisplayFeaturesRef.current = [];
    spiderfyTotalCountRef.current = 0;

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
    // 保存快照，供 zoom 变化时 refreshSpiderfy 重算坐标
    // totalCount 保留真实聚合总数（含截断部分），保证中心球数字始终正确
    spiderfyDisplayFeaturesRef.current = displayFeatures;
    spiderfyTotalCountRef.current = count;

    // 触发地图重新渲染
    mapInstanceRef.current.render();
  }, [unspiderfy]);

  /**
   * zoom 变化时，用新的 resolution 重算 spiderfy 节点坐标并刷新图层，
   * 使子节点的屏幕间距保持不变（始终约 SPIDERFY_CONFIG.radius 像素）。
   * 替代原先的 unspiderfy()，避免用户放大查看节点时展开被销毁。
   */
  const refreshSpiderfy = useCallback(() => {
    if (
      !isSpiderfiedRef.current ||
      !spiderfySourceRef.current ||
      !spiderfiedCenterRef.current ||
      !mapInstanceRef.current
    ) return;

    const center = spiderfiedCenterRef.current;
    const displayFeatures = spiderfyDisplayFeaturesRef.current;
    if (!displayFeatures.length) return;

    const displayCount = displayFeatures.length;
    // totalCount = 真实聚合总数（含截断部分），用于中心球数字和 pointFeature.total，
    // 保证 zoom 变化后语义与初次展开完全一致。
    const totalCount = spiderfyTotalCountRef.current || displayCount;
    // 注意：center 是 spiderfy 展开时的聚合中心，不变；
    // 只有 resolution 随 zoom 变化，重算各节点的地图坐标。
    const resolution = mapInstanceRef.current.getView().getResolution()!;

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

    const newFeatures: Feature[] = [];
    const startAngle = -Math.PI / 2;

    // 中心点 count 始终用真实总数，保证放大/缩小时蓝球数字不变
    const centerFeature = new Feature({
      geometry: new Point(center),
      spiderfyCenter: true,
      count: totalCount,
    });
    newFeatures.push(centerFeature);

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
          center[1] - pixelOffset[1] * resolution,
        ];

        newFeatures.push(new Feature({
          geometry: new LineString([center, pointCoordinate]),
          spiderfyLine: true,
        }));

        newFeatures.push(new Feature({
          geometry: new Point(pointCoordinate),
          spiderfyPoint: true,
          device,
          index: cursor,
          total: totalCount,
        }));

        cursor++;
      }
    });

    // 原子替换：clear + addFeatures 在同一同步帧内，避免闪烁
    spiderfySourceRef.current.clear();
    spiderfySourceRef.current.addFeatures(newFeatures);
    mapInstanceRef.current.render();
  }, []);

  // 根据缩放级别动态调整聚合距离（缓动过渡，避免瞬间跳变）
  const updateClusterDistance = useCallback((zoom: number) => {
    if (!clusterSourceRef.current) return;

    const newDistance = getClusterDistanceForZoom(zoom);
    const fromDistance = clusterSourceRef.current.getDistance();
    if (fromDistance === newDistance) return;

    // 取消上一帧动画，从当前中间值重新开始（防止快速缩放时动画叠加）
    if (clusterAnimRafRef.current != null) {
      cancelAnimationFrame(clusterAnimRafRef.current);
      clusterAnimRafRef.current = null;
    }

    const DURATION = 250; // ms，easeOutCubic
    const startTime = performance.now();
    clusterDistAnimActiveRef.current = true;

    const step = (now: number) => {
      const t = Math.min(1, (now - startTime) / DURATION);
      const eased = 1 - Math.pow(1 - t, 3); // easeOutCubic
      const currentDistance = fromDistance + (newDistance - fromDistance) * eased;

      clusterSourceRef.current?.setDistance(currentDistance);
      mapInstanceRef.current?.render();

      if (t < 1) {
        clusterAnimRafRef.current = requestAnimationFrame(step);
      } else {
        clusterAnimRafRef.current = null;
        clusterDistAnimActiveRef.current = false;
        // 确保最终值精确，消除浮点误差
        clusterSourceRef.current?.setDistance(newDistance);
      }
    };

    clusterAnimRafRef.current = requestAnimationFrame(step);
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
    // 等待可靠中心点就绪（stats 或离线地图元数据），保证 OL 以正确坐标初始化。
    // 默认 true（向后兼容），GISMapView 在有设备中心点数据后才传 true。
    if (!centerReady) return;

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
      // wrapX 保持默认 true：瓦片背景横向连续，缩小时地图不出现空白区域
    });

    // 限制瓦片并发数，避免 OpenLayers 一次打满浏览器 6 个 HTTP/1.1 连接，
    // 留出至少 2~3 个连接给 API 请求（设备搜索、geo 数据等）。
    // 并发上限由 tileConcurrencyRef 控制，可在搜索时动态降低。
    drainTileQueueRef.current = () => {
      while (
        activeTileCountRef.current < tileConcurrencyRef.current &&
        tileLoadQueueRef.current.length > 0
      ) {
        const load = tileLoadQueueRef.current.shift()!;
        activeTileCountRef.current++;
        load();
      }
    };
    tileSource.setTileLoadFunction((tile, src) => {
      const img = (tile as ImageTile).getImage() as HTMLImageElement;
      const doLoad = () => {
        img.onload = () => { activeTileCountRef.current--; drainTileQueueRef.current(); };
        img.onerror = () => { activeTileCountRef.current--; drainTileQueueRef.current(); };
        img.src = src;
      };
      if (activeTileCountRef.current < tileConcurrencyRef.current) {
        activeTileCountRef.current++;
        doLoad();
      } else {
        tileLoadQueueRef.current.push(doLoad);
      }
    });

    const tileLayer = new TileLayer({
      source: tileSource,
      opacity: 1.0,
      zIndex: 0, // 确保瓦片层在最底层
    });
    layers.push(tileLayer);

    // 创建设备数据源（wrapX: false 防止 feature 在多个世界副本中重复显示）
    deviceSourceRef.current = new VectorSource({ wrapX: false });

    // 创建聚合数据源（初始 distance 按初始 zoom 查表，避免首帧聚合距离不匹配）
    // wrapX 不在此设置：VectorSource 已有 wrapX:false，Cluster 基于其 features 计算，
    // 对 Cluster 设 wrapX:false 会影响动态 distance 更新时的空间索引重建，影响缩放体验
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

    // Cluster feature 出生动画：为新形成的 cluster feature 打时间戳
    // distance 动画进行时跳过，避免每帧重打导致 birthTime 被刷新、弹出动画无法完成
    const clusterChangeKey: EventsKey = clusterSourceRef.current.on('change', () => {
      if (clusterDistAnimActiveRef.current) return;
      const now = performance.now();
      for (const f of (clusterSourceRef.current?.getFeatures() ?? [])) {
        if (f.get('_birthTime') === undefined) {
          f.set('_birthTime', now, true); // silent=true：不触发 feature 自身 change 事件
        }
      }
    }) as EventsKey;

    // postrender 驱动：只要有 cluster 还在弹出动画窗口内就持续触发下一帧
    const BIRTH_ANIM_DURATION = 250;
    const postrenderKey: EventsKey = deviceLayerRef.current.on('postrender', () => {
      const now = performance.now();
      const needsFrame = (clusterSourceRef.current?.getFeatures() ?? []).some(f => {
        const birth = f.get('_birthTime') as number | undefined;
        return birth !== undefined && (now - birth) < BIRTH_ANIM_DURATION;
      });
      if (needsFrame) mapInstanceRef.current?.render();
    }) as EventsKey;

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
          // zoom 变化时用新 resolution 重算 spiderfy 节点坐标，
          // 保持子节点的屏幕间距稳定，让用户放大/缩小后仍能看到展开状态。
          if (isSpiderfiedRef.current) {
            refreshSpiderfy();
          }
        },
        onSpiderfy: spiderfy,
        onUnspiderfy: unspiderfy,
        onMapClick,
        isProgrammaticFlyingRef,
        isSpiderfiedRef,
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
      unByKey(clusterChangeKey);
      unByKey(postrenderKey);
      if (clusterAnimRafRef.current != null) {
        cancelAnimationFrame(clusterAnimRafRef.current);
        clusterAnimRafRef.current = null;
      }
      clusterDistAnimActiveRef.current = false;
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
  // centerReady：从 false→true 时触发，确保 OL 以正确中心点创建（而非赞比亚默认坐标）
  }, [metadataLoading, centerReady]);

  // 把一组设备渲染进 VectorSource（增量 diff：只删除消失的、只添加新增的）
  // 避免 clear() → addFeatures() 中间的单帧空白，消除聚合数字跳变和节点闪烁。
  const renderDeviceFeatures = useCallback((devices: MapDevice[]) => {
    const source = deviceSourceRef.current;
    if (!source) return;

    // 快照当前已渲染的 features（在任何增删之前）
    const existingFeatures = source.getFeatures();
    const existingIdSet = new Set(existingFeatures.map(f => f.getId() as string));
    const newIdSet = new Set(devices.map(d => d.id));

    // 1. 移除不再出现在新集合里的 feature
    for (const f of existingFeatures) {
      if (!newIdSet.has(f.getId() as string)) {
        source.removeFeature(f);
      }
    }

    // 2. 添加尚未渲染的新 feature
    const toAdd = devices.filter(d => !existingIdSet.has(d.id));
    if (toAdd.length > 0) {
      const features = toAdd.map((device) => {
        const feature = new Feature({
          geometry: new Point(fromLonLat([device.lng, device.lat])),
          ...device,
        });
        feature.setId(device.id);
        return feature;
      });
      source.addFeatures(features);
    }
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

  // 清除所有设备数据（筛选条件变化时调用，彻底重置）
  const clearDevices = useCallback(() => {
    allDevicesRef.current = [];
    deviceSourceRef.current?.clear();
  }, []);

  // 更新设备数据
  const updateDevices = useCallback((devices: MapDevice[]) => {
    if (!deviceSourceRef.current) return;

    // 如果当前有 spiderfy 展开，先收起（因为设备数据已变化，展开的内容可能不再有效）
    if (isSpiderfiedRef.current) {
      unspiderfy();
    }

    // 累积策略：将新数据与 allDevicesRef 合并（按 id 去重，更新已有设备数据）。
    // 目的：zoom-out 时新 geo 请求尚未返回期间，allDevicesRef 仍持有之前宽视口的数据，
    // 避免"节点消失 → 等待 → 重新出现"的抖动和聚合数字跳变。
    // 只有调用 clearDevices()（筛选条件真实变化）时才真正清空。
    const existing = allDevicesRef.current;
    if (existing.length === 0) {
      allDevicesRef.current = devices;
    } else {
      const idMap: Record<string, MapDevice> = {};
      for (const d of existing) { idMap[d.id] = d; }
      for (const d of devices) { idMap[d.id] = d; }
      allDevicesRef.current = Object.keys(idMap).map(k => idMap[k]);
    }

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
    // 取消 rAF 水波纹动画
    if (rippleRafRef.current != null) {
      cancelAnimationFrame(rippleRafRef.current);
      rippleRafRef.current = null;
    }

    // 兼容旧 setInterval 路径（如 highlightDevice 还在用）
    if (pulseAnimationRef.current) {
      clearInterval(pulseAnimationRef.current);
      pulseAnimationRef.current = null;
    }
    if (rippleCreateRef.current) {
      clearInterval(rippleCreateRef.current);
      rippleCreateRef.current = null;
    }
    rippleWavesRef.current = [];

    if (highlightFeatureRef.current) {
      highlightFeatureRef.current.set('highlighted', false);
      highlightFeatureRef.current.set('rippleWaves', undefined);
      highlightFeatureRef.current.set('_rippleStart', undefined);
      highlightFeatureRef.current = null;
    }

    // 取消搜索定位 pin（性能 #15）：高亮结束后该设备恢复受裁剪约束
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
      // 飞行到设备位置
      const geometry = feature.getGeometry();
      if (geometry) {
        const coordinate = (geometry as Point).getCoordinates();
        const lonLat = toLonLat(coordinate);
        flyTo(lonLat[0], lonLat[1]);
      }

      // rAF 水波纹动画（与 highlightAndSpiderfyIfNeeded 共用同一逻辑）
      const WAVE_COUNT = 3;
      const WAVE_INTERVAL = 600;
      const WAVE_LIFETIME = 1800;
      const WAVE_MAX_RADIUS = 28;

      feature.set('highlighted', true);
      highlightFeatureRef.current = feature as Feature;
      rippleStartTimeRef.current = performance.now();
      feature.set('_rippleStart', rippleStartTimeRef.current);

      const animate = () => {
        if (!highlightFeatureRef.current) return;
        const now = performance.now();
        const elapsed = now - rippleStartTimeRef.current;
        const waves: { radius: number; opacity: number }[] = [];
        for (let i = 0; i < WAVE_COUNT; i++) {
          const offset = i * WAVE_INTERVAL;
          const cycle = WAVE_COUNT * WAVE_INTERVAL;
          const waveAge = ((elapsed - offset) % cycle + cycle) % cycle;
          if (waveAge < WAVE_LIFETIME) {
            const progress = waveAge / WAVE_LIFETIME;
            const eased = 1 - Math.pow(1 - progress, 2);
            waves.push({ radius: eased * WAVE_MAX_RADIUS, opacity: 0.75 * (1 - progress) });
          }
        }
        highlightFeatureRef.current.set('rippleWaves', waves);
        mapInstanceRef.current?.render();
        rippleRafRef.current = requestAnimationFrame(animate);
      };

      if (rippleRafRef.current != null) cancelAnimationFrame(rippleRafRef.current);
      rippleRafRef.current = requestAnimationFrame(animate);
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
  const fitBounds = useCallback((bounds: MapBounds, options?: { duration?: number }) => {
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
      duration: options?.duration ?? ANIMATION_CONFIG.flyDuration,
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
    // 全新 rAF 驱动的水波纹动画
    // 3 个波纹交错，间隔 600ms；每个波纹独立生命周期 1800ms：
    //   0ms 出生，半径 0 → 30px，透明度 0.7 → 0（基于真实时间，帧率无关）
    const WAVE_COUNT = 3;
    const WAVE_INTERVAL = 600;  // ms，两波之间的间隔
    const WAVE_LIFETIME = 1800; // ms，单波扩散总时长
    const WAVE_MAX_RADIUS = 28; // px，波纹最大半径（相对节点外边）

    const applyRippleHighlight = (feature: Feature) => {
      feature.set('highlighted', true);
      highlightFeatureRef.current = feature;
      rippleStartTimeRef.current = performance.now();
      feature.set('_rippleStart', rippleStartTimeRef.current);

      const animate = () => {
        if (!highlightFeatureRef.current) return; // 已被 clearHighlight
        const now = performance.now();
        const elapsed = now - rippleStartTimeRef.current;

        // 计算 3 个波的当前状态（交错偏移）
        const waves: { radius: number; opacity: number }[] = [];
        for (let i = 0; i < WAVE_COUNT; i++) {
          const offset = i * WAVE_INTERVAL;
          // 每个波在 elapsed 时间轴上的位置（循环周期 = WAVE_INTERVAL * WAVE_COUNT）
          const cycle = WAVE_COUNT * WAVE_INTERVAL;
          const waveAge = ((elapsed - offset) % cycle + cycle) % cycle; // 0 ~ cycle
          if (waveAge < WAVE_LIFETIME) {
            const progress = waveAge / WAVE_LIFETIME; // 0 → 1
            const eased = 1 - Math.pow(1 - progress, 2); // easeOutQuad：快扩慢收
            waves.push({
              radius: eased * WAVE_MAX_RADIUS,
              opacity: 0.75 * (1 - progress),
            });
          }
        }

        highlightFeatureRef.current.set('rippleWaves', waves);
        mapInstanceRef.current?.render();
        rippleRafRef.current = requestAnimationFrame(animate);
      };

      // 取消之前可能残留的 rAF
      if (rippleRafRef.current != null) {
        cancelAnimationFrame(rippleRafRef.current);
      }
      rippleRafRef.current = requestAnimationFrame(animate);
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
          deviceFeature.set('highlighted', true);
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
          deviceFeature.set('highlighted', true);
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
    setTileConcurrency: (n: number) => {
      // 更新并发上限，立即触发排队中的任务以填满新空余连接
      tileConcurrencyRef.current = Math.max(0, n);
      drainTileQueueRef.current();
    },
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
    onSpiderfy?: (clusterFeature: Feature, center: number[], features: Feature[], targetDeviceId?: string) => void;
    onUnspiderfy?: () => void;
    onMapClick?: () => void;
    /**
     * “散不开 + 数量太多”时触发，由父层弹出设备列表 Drawer。
     * 未提供时退化为 spiderfy（受 SPIDERFY_CONFIG.maxNodes 截断保护）。
     */
    onClusterShowList?: (devices: MapDevice[], pixel: { x: number; y: number }) => void;
    /**
     * “程序化飞行中”计数器的 ref。为 >0 时，表示中间动画档位，
     * moveend 不应在此时上报 viewport，避免引起外部重复拉取 /devices/geo。
     */
    isProgrammaticFlyingRef?: React.MutableRefObject<number>;
    /**
     * useOLMap 维护的 spiderfy 状态 ref。传入后 bindMapEvents 直接读写该 ref，
     * 消除闭包局部变量与外层 ref 双写不同步的竞态问题。
     */
    isSpiderfiedRef: React.MutableRefObject<boolean>;
  },
  deviceLayer: VectorLayer<VectorSource>,
  spiderfyLayer: VectorLayer<VectorSource>
): void {
  const { onDeviceClick, onDeviceHover, onViewportChange, onClusterClick, onZoomChange, onSpiderfy, onUnspiderfy, onMapClick, onClusterShowList, isProgrammaticFlyingRef, isSpiderfiedRef } = callbacks;

  // isSpiderfied 直接读写外层 ref，消除闭包局部变量与 ref 双写竞态

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
        isSpiderfiedRef.current = false;
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
          if (isSpiderfiedRef.current) {
            onUnspiderfy?.();
            isSpiderfiedRef.current = false;
            return;
          }

          // 智能路由：看 features 的屏幕 bbox 对角线
          //   1) > clickExpandThresholdPx → 散得开，view.fit 下钻（类 Google Maps）
          //   2) ≤ 阈值 且 count ≤ spiderfyMaxCount → spiderfy
          //   3) ≤ 阈值 且 count > spiderfyMaxCount → onClusterShowList 回调（弹 Drawer）
          //      未接时退化到 spiderfy + maxNodes 截断保证不崩
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
            // 让 moveend 只在最后一个档位上报 viewport。
            if (isProgrammaticFlyingRef) {
              isProgrammaticFlyingRef.current += 1;
            }
            view.fit(extent, {
              padding: [80, 80, 80, 80],
              duration: 400,
              maxZoom: targetMaxZoom,
              easing: Easing.easeOutCubic,
              callback: () => {
                if (isProgrammaticFlyingRef) {
                  isProgrammaticFlyingRef.current = Math.max(
                    0,
                    isProgrammaticFlyingRef.current - 1,
                  );
                }
              },
            });
            return;
          }

          // 散不开：判断走 spiderfy 还是列表
          if (
            featuresProp.length > CLUSTER_CONFIG.spiderfyMaxCount &&
            onClusterShowList
          ) {
            const [x, y] = evt.pixel;
            onClusterShowList(devices, { x, y });
            return;
          }

          if (currentZoom >= SPIDERFY_CONFIG.minZoom) {
            const geometry = feature.getGeometry();
            if (geometry) {
              const center = (geometry as Point).getCoordinates();
              onSpiderfy?.(feature, center, featuresProp as Feature[]);
              isSpiderfiedRef.current = true;
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
    } else if (isSpiderfiedRef.current) {
      // 点击空白区域，收起展开
      onUnspiderfy?.();
      isSpiderfiedRef.current = false;
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
    }, 250);
  });
}

export default useOLMap;
