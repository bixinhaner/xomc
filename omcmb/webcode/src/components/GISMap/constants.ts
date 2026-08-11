/**
 * GIS 地图常量配置
 * @module components/GISMap/constants
 */

import type { DeviceStatus } from '@core/types/map';
import type { MapMetadata } from './useMapConfig';

/**
 * 设备状态配置
 * 根据 UI 设计图: GISMap_UI_Design_Markers.svg
 * 三种状态：在线激活(绿色)、在线未激活(黄色)、离线(红色)
 *
 * 颜色说明：
 * - color: 标记/图标的颜色（保持鲜艳）
 * - textColor: 文字显示颜色（加深版本，符合 WCAG AA 对比度要求 4.5:1）
 */
export const DEVICE_STATUS_CONFIG: Record<
  DeviceStatus,
  {
    color: string;
    gradientStart: string;
    gradientEnd: string;
    bgColor: string;
    borderColor: string;
    textColor: string; // 文字颜色（加深版本，提升可读性）
    text: string;
    i18nKey: string;
  }
> = {
  onlineActive: {
    color: '#52C41A',
    gradientStart: '#73D13D',
    gradientEnd: '#52C41A',
    bgColor: '#F6FFED',
    borderColor: '#B7EB8F',
    textColor: '#237804', // 深绿色，对比度 > 7:1
    text: '在线激活',
    i18nKey: 'status.onlineActive',
  },
  onlineInactive: {
    color: '#FAAD14',
    gradientStart: '#FFC53D',
    gradientEnd: '#FAAD14',
    bgColor: '#FFFBE6',
    borderColor: '#FFE58F',
    textColor: '#D48806', // 深黄色，对比度 > 5:1
    text: '在线未激活',
    i18nKey: 'status.onlineInactive',
  },
  offline: {
    color: '#b60808',
    gradientStart: '#b60808',
    gradientEnd: '#b60808',
    bgColor: '#FFF1F0',
    borderColor: '#FFA39E',
    textColor: '#8B0000', // 深红色，对比度 > 7:1
    text: '离线',
    i18nKey: 'status.offline',
  },
};

/**
 * 聚合标记配置
 * 根据 UI 设计图: GISMap_UI_Design_Markers.svg
 */
export const CLUSTER_CONFIG = {
  /** 渐变起始色 */
  gradientStart: '#40A9FF',
  /** 渐变结束色 */
  gradientEnd: '#1890FF',
  /** 最小半径 (px) */
  minRadius: 16,
  /** 最大半径 (px) */
  maxRadius: 40,
  /** 基础半径 (px) */
  baseRadius: 16,
  /** 半径计算系数 */
  radiusFactor: 10,
  /**
   * 按 zoom 分档的聚合距离 (px)
   *
   * 取第一个满足 `zoom <= maxZoom` 的档；列表必须按 maxZoom 升序。
   * 设计意图：让 zoom 5→16 每升一档都能看到聚合数变化，
   * 形成洲→国→省→市→区→街道的逐级分散视觉过渡。
   */
  distanceTiers: [
    { maxZoom: 5, distance: 90 },
    { maxZoom: 7, distance: 50 },
    { maxZoom: 9, distance: 40 },
    { maxZoom: 11, distance: 30 },
    { maxZoom: 12, distance: 22 },
    { maxZoom: 13, distance: 14 },
    { maxZoom: 14, distance: 8 },
    { maxZoom: Infinity, distance: 0 },
  ] as ReadonlyArray<{ maxZoom: number; distance: number }>,
  /** 小型聚合阈值 (10-49) */
  smallThreshold: 10,
  /** 中型聚合阈值 (50-99) */
  mediumThreshold: 50,
  /** 大型聚合阈值 (100+) */
  largeThreshold: 100,
  /**
   * 点击聚合时用于判定“散不开”的屏幕像素对角线阈值。
   * features bbox 的屏幕像素对角线 > 该值 → 说明放大能散开，走 view.fit 下钻；
   * ≤ 该值 → 说明几乎同坐标，走 spiderfy 或列表。
   */
  clickExpandThresholdPx: 50,
  /**
   * “散不开”场景下仍然合适用 spiderfy 展开的最大节点数。
   * 超过该值会优先触发 onClusterShowList 回调由父层弹列表；未接则退化为 spiderfy + maxNodes 截断。
   */
  spiderfyMaxCount: 20,
};

/**
 * 根据 zoom 查询聚合距离（像素）
 * 与 `CLUSTER_CONFIG.distanceTiers` 配套使用。
 */
export function getClusterDistanceForZoom(zoom: number): number {
  const tier = CLUSTER_CONFIG.distanceTiers.find((t) => zoom <= t.maxZoom);
  return tier ? tier.distance : 0;
}

/**
 * 视口裁剪配置（性能 #15）
 *
 * 设备列表分页 pageSize 可达 10000，全量塞进 VectorSource 会让 OpenLayers
 * 每帧对所有 feature 做聚合/命中检测，渲染耗时 500ms+。视口裁剪只把"当前
 * 可视范围 + 缓冲边距"内的设备喂给地图，其余设备保留在内存（allDevicesRef）
 * 不参与渲染；地图平移/缩放（moveend）后按新视口重算。聚合（Cluster）逻辑
 * 不变——裁剪后的子集仍照常聚合，缩放与平移交互完全保留。
 */
export const VIEWPORT_CULLING = {
  /**
   * 启用裁剪的设备数量阈值。
   * 设备数 <= 此值时全量渲染（保持原有行为，避免小数据集额外开销与
   * "平移露白"边缘情形）；超过才启用视口裁剪。
   */
  enableThreshold: 2000,
  /**
   * 视口缓冲倍数。按当前可视 extent 的宽/高各向外扩展该比例，
   * 让平移时边缘设备已预先在视口内，避免可见的"突然出现"。
   * 0.5 表示左右各扩展半个屏宽、上下各扩展半个屏高。
   */
  bufferRatio: 0.5,
};

/**
 * 告警角标配置
 */
export const ALARM_BADGE_CONFIG = {
  /** 渐变起始色 */
  gradientStart: '#FF7875',
  /** 渐变结束色 */
  gradientEnd: '#F5222D',
  /** 角标半径 (px) */
  radius: 9,
  /** 大角标半径 (99+) */
  largeRadius: 11,
  /** 字体大小 */
  fontSize: 9,
  /** 大字体大小 (99+) */
  largeFontSize: 8,
  /** 最大显示数字 */
  maxDisplay: 99,
};

/**
 * 设备标记尺寸配置
 * 根据 UI 设计图: zoom 级别对应不同尺寸
 */
export const MARKER_SIZE_CONFIG = {
  /** 小尺寸 (zoom >= 15) */
  small: {
    radius: 8,
    zoomMin: 15,
  },
  /** 中尺寸 (zoom 12-14) */
  medium: {
    radius: 12,
    zoomMin: 12,
    zoomMax: 14,
  },
  /** 大尺寸 (zoom < 12) */
  large: {
    radius: 15,
    zoomMax: 11,
  },
  /** 白色边框宽度 */
  strokeWidth: 2,
};

/**
 * 地图默认配置
 */
export const MAP_CONFIG = {
  /**
   * 默认中心点 [lng, lat]
   * 支持环境变量配置：VITE_MAP_DEFAULT_CENTER="lng,lat,zoom"
  * 优先级：环境变量 > 中国区域默认中心点
   */
  get defaultCenter(): [number, number] {
    // 尝试从环境变量读取
    const envValue = import.meta.env.VITE_MAP_DEFAULT_CENTER;
    if (envValue) {
      try {
        const parts = envValue.split(',').map((p: string) => p.trim());
        if (parts.length >= 2) {
          const lng = parseFloat(parts[0]);
          const lat = parseFloat(parts[1]);
          if (!isNaN(lng) && !isNaN(lat) && lng >= -180 && lng <= 180 && lat >= -90 && lat <= 90) {
            return [lng, lat];
          }
        }
      } catch {
        console.warn('[MAP_CONFIG] Failed to parse VITE_MAP_DEFAULT_CENTER, using default');
      }
    }
    // 默认值：没有离线元数据且没有设备坐标时使用中国区域中心
    return [104.0, 35.0];
  },

  /**
   * 默认缩放级别
   * 支持从环境变量 VITE_MAP_DEFAULT_CENTER="lng,lat,zoom" 中读取
   * 优先级：环境变量中的 zoom > 硬编码默认值 6
   */
  get defaultZoom(): number {
    // 尝试从环境变量读取
    const envValue = import.meta.env.VITE_MAP_DEFAULT_CENTER;
    if (envValue) {
      try {
        const parts = envValue.split(',').map((p: string) => p.trim());
        if (parts.length >= 3) {
          const zoom = parseInt(parts[2], 10);
          if (!isNaN(zoom) && zoom >= 1 && zoom <= 18) {
            return zoom;
          }
        }
      } catch {
        console.warn('[MAP_CONFIG] Failed to parse zoom from VITE_MAP_DEFAULT_CENTER, using default');
      }
    }
    // 默认缩放级别
    return 4;
  },
  /** 最小缩放级别 */
  minZoom: 1,
  /** 最大缩放级别 */
  maxZoom: 18,
  /** 聚合显示阈值 (zoom < 12 显示聚合) */
  clusterZoomThreshold: 12,
  /** OpenStreetMap 在线瓦片地址 */
  osmTileUrl: 'https://{a-c}.tile.openstreetmap.org/{z}/{x}/{y}.png',
  /** 默认版权信息 */
  attribution: '© OpenStreetMap contributors',
  /** 默认地理边界（全球范围，仅用于没有设备数据时的兜底） */
  bounds: { minLon: -180, maxLon: 180, minLat: -85, maxLat: 85 },
  /**
   * 瓦片服务地址（智能选择）
   * 优先级：环境变量 VITE_MAP_TILE_URL > 在线 OSM
   *
   * 配置方式：
   * - 在 .env.development 或 .env.production 中设置：
   *   VITE_MAP_TILE_URL=http://192.168.20.31:8050/{z}/{x}/{y}.png
   * - 不设置则使用在线 OSM 瓦片
   */
  get tileUrl() {
    return import.meta.env.VITE_MAP_TILE_URL || this.osmTileUrl;
  },
  /** 视图变化防抖时间 (ms) */
  viewportDebounce: 300,
  /** 搜索防抖时间 (ms) */
  searchDebounce: 300,
};

/**
 * 渐进式缩放步骤配置
 * 模拟从国家→省份→市→区→街道的视觉效果
 * 每个步骤的zoom级别都会被限制在瓦片服务范围内
 */
export interface ProgressiveZoomStep {
  /** 目标缩放级别 */
  zoom: number;
  /** 动画时长 (ms) */
  duration: number;
  /** 步骤间极短暂停，保持流畅感 */
  pause: number;
}

/**
 * 渐进式缩放配置
 * 搜索定位时使用，形成"从宏观到微观"的视觉效果
 * 流畅节奏：总时长约 1.1 秒，几乎无停顿感
 */
export const PROGRESSIVE_ZOOM_STEPS: ProgressiveZoomStep[] = [
  { zoom: 5, duration: 280, pause: 10 },   // 国家级视图（缩小显示，地图占约80%）
  { zoom: 8, duration: 240, pause: 10 },   // 省级/大区视图
  { zoom: 11, duration: 200, pause: 10 },  // 市级视图
  { zoom: 14, duration: 180, pause: 10 },  // 区级视图
  { zoom: 16, duration: 180, pause: 0 },   // 街道视图（最终定位）
];

/**
 * 动画配置
 * 根据 UI 设计图
 */
export const ANIMATION_CONFIG = {
  /** 聚合圈呼吸动画周期 (ms) */
  pulseDuration: 2000,
  /** 悬停放大比例 */
  hoverScale: 1.2,
  /** 悬停动画时间 (ms) */
  hoverDuration: 200,
  /** 点击缩放动画时间 (ms) */
  clickDuration: 150,
  /** 搜索高亮脉冲圈数量 */
  highlightPulseCount: 3,
  /** 搜索高亮脉冲周期 (ms) */
  highlightPulseDuration: 1500,
  /** 飞行动画时间 (ms) */
  flyDuration: 1000,
  /** 高亮缩放级别 */
  highlightZoom: 15,
  /** 最大放大程度（用于搜索定位） */
  maxHighlightZoom: 18,
  /**
   * 是否启用渐进式缩放动画
   * 启用后，搜索定位会使用多级缩放效果（国家→省→市→区→街道）
   * 禁用后，使用单次平滑动画
   */
  enableProgressiveZoom: true,
  /**
   * 渐进式缩放最小触发zoom差值
   * 当前zoom与目标zoom差值小于此值时，使用单次动画（避免不必要的多级动画）
   */
  progressiveZoomThreshold: 3,
  /**
   * 搜索两段定位的中间停靠 zoom。
   * 命中聚合时停留在能分辨 cluster 的高度，避免一口气飞到最大 zoom
   * 导致同坐标聚合被打散、目标 feature 找不到。
   */
  searchIntermediateZoom: 13,
};

/**
 * 从 PROGRESSIVE_ZOOM_STEPS 里挑出严格大于 currentZoom 且 < targetZoom 的中间档位。
 * 最后一步会被替换成实际的 targetZoom，从而让用户传入的目标 zoom（例如 18）保持精确。
 *
 * 返回空数组表示不需要中间分段，调用方应回退到单段动画。
 */
export function pickProgressiveSteps(
  currentZoom: number,
  targetZoom: number,
): ProgressiveZoomStep[] {
  if (!(targetZoom > currentZoom)) return [];
  const middle = PROGRESSIVE_ZOOM_STEPS.filter(
    (s) => s.zoom > currentZoom + 0.5 && s.zoom < targetZoom - 0.5,
  );
  if (middle.length === 0) return [];
  const last = PROGRESSIVE_ZOOM_STEPS[PROGRESSIVE_ZOOM_STEPS.length - 1];
  return [
    ...middle,
    { zoom: targetZoom, duration: last.duration, pause: 0 },
  ];
}

/**
 * 中国边界范围（用于初始视图）
 */
export const CHINA_BOUNDS = {
  minLng: 73,
  maxLng: 135,
  minLat: 18,
  maxLat: 53,
};

/**
 * 通用颜色配置
 * 用于地图标记、边框、文字等样式
 */
export const COLORS = {
  /** 白色 - 边框、文字 */
  white: '#FFFFFF',
  /** 主题色 - 高亮、连线 */
  primary: '#1890FF',
  /** 聚合标记结束色 */
  clusterEnd: '#1890FF',
  /** 告警渐变起始色 */
  alarmStart: '#FF7875',
  /** 告警渐变结束色 */
  alarmEnd: '#F5222D',
};

/**
 * Spiderfy 展开配置
 * 用于点击聚合标记时展开重叠设备点
 */
export const SPIDERFY_CONFIG = {
  /** 展开半径（像素） */
  radius: 80,
  /** 连线宽度 */
  lineWidth: 2,
  /** 连线颜色 */
  lineColor: 'rgba(24, 144, 255, 0.5)',
  /** 展开点半径 */
  pointRadius: 12,
  /** 触发 spiderfy 的最小 zoom 级别 */
  minZoom: 4,
  /** 展开动画时长（ms） */
  animationDuration: 300,
  /**
   * 单次最多展开的节点数（防爆屏）。
   * 同坐标 100+ 设备时全部画出会导致主线程阻塞、网络请求假死。
   * 多出的设备保留在中心 count 显示，搜索目标会被强制提权到可视集中。
   */
  maxNodes: 60,
  /** 同圈相邻节点之间的弧长间隔（像素），用于多圈分布算容量 */
  ringArcSpacing: 28,
  /** 多圈展开时每外推一圈的半径增量（像素） */
  ringRadiusStep: 45,
};

/**
 * 从元数据构建地图配置
 *
 * @deprecated 此函数已废弃，请使用 @/utils/mapValidation 中的 buildSafeConfig 代替
 * 原因：新函数包含完整的 metadata 验证逻辑，更加安全可靠
 *
 * @param metadata - 地图元数据
 * @returns 地图配置对象
 */
export function buildMapConfigFromMetadata(metadata: MapMetadata) {
  // 保留此函数仅为向后兼容，实际使用 buildSafeConfig
  return {
    defaultCenter: [metadata.center.lon, metadata.center.lat] as [number, number],
    defaultZoom: metadata.center.zoom,
    minZoom: metadata.zoom.min,
    maxZoom: metadata.zoom.max,
    tileUrl: import.meta.env.VITE_MAP_TILE_URL || MAP_CONFIG.osmTileUrl,
    attribution: metadata.attribution,
    bounds: metadata.bounds,
  };
}
