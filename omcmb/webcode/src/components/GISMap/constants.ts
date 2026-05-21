/**
 * GIS 地图常量配置
 * @module components/GISMap/constants
 */

import type { DeviceStatus } from '@core/types/map';

/**
 * 设备状态配置
 * 根据 UI 设计图: GISMap_UI_Design_Markers.svg
 * 三种状态：在线激活(绿色)、在线未激活(黄色)、离线(红色)
 */
export const DEVICE_STATUS_CONFIG: Record<
  DeviceStatus,
  {
    color: string;
    gradientStart: string;
    gradientEnd: string;
    bgColor: string;
    borderColor: string;
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
    text: '在线激活',
    i18nKey: 'status.onlineActive',
  },
  onlineInactive: {
    color: '#FAAD14',
    gradientStart: '#FFC53D',
    gradientEnd: '#FAAD14',
    bgColor: '#FFFBE6',
    borderColor: '#FFE58F',
    text: '在线未激活',
    i18nKey: 'status.onlineInactive',
  },
  offline: {
    color: '#b60808',
    gradientStart: '#b60808',
    gradientEnd: '#b60808',
    bgColor: '#FFF1F0',
    borderColor: '#FFA39E',
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
  /** 聚合距离 (px) - 低缩放级别时使用 */
  distance: 40,
  /** 高缩放级别时的聚合距离 (px) - 更小的值让设备更容易分散 */
  highZoomDistance: 10,
  /** 禁用聚合的缩放阈值 - 超过此级别完全禁用聚合 */
  disableClusterZoom: 15,
  /** 小型聚合阈值 (10-49) */
  smallThreshold: 10,
  /** 中型聚合阈值 (50-99) */
  mediumThreshold: 50,
  /** 大型聚合阈值 (100+) */
  largeThreshold: 100,
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
  /** 默认中心点 [lng, lat] - 赞比亚中心 (所有设备经纬度平均值) */
  defaultCenter: [28.221, -14.607] as [number, number],
  /** 默认缩放级别 */
  defaultZoom: 6,
  /** 最小缩放级别 */
  minZoom: 1,
  /** 最大缩放级别 */
  maxZoom: 18,
  /** 聚合显示阈值 (zoom < 12 显示聚合) */
  clusterZoomThreshold: 12,
  /** OpenStreetMap 在线瓦片地址 */
  osmTileUrl: 'https://{a-c}.tile.openstreetmap.org/{z}/{x}/{y}.png',
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
};

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
};
