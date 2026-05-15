/**
 * GIS 地图样式工具函数
 * @module components/GISMap/styleUtils
 */

import { Style, Circle, Fill, Stroke, Text } from 'ol/style';
import type Feature from 'ol/Feature';
import type { MapDevice, DeviceStatus } from '@core/types/map';
import {
  DEVICE_STATUS_CONFIG,
  CLUSTER_CONFIG,
  ALARM_BADGE_CONFIG,
  MARKER_SIZE_CONFIG,
  COLORS,
} from './constants';

/**
 * 根据缩放级别获取标记半径
 */
export function getMarkerRadius(zoom: number): number {
  const { small, medium, large } = MARKER_SIZE_CONFIG;
  if (zoom >= small.zoomMin) return small.radius;
  if (zoom >= medium.zoomMin) return medium.radius;
  return large.radius;
}

/**
 * 根据聚合数量计算半径
 * radius = 16 + count/10, 范围 [16, 40]
 */
export function getClusterRadius(count: number): number {
  const { baseRadius, radiusFactor, minRadius, maxRadius } = CLUSTER_CONFIG;
  const radius = baseRadius + count / radiusFactor;
  return Math.max(minRadius, Math.min(maxRadius, radius));
}

/**
 * 格式化告警数量显示
 * 1-9: 直接显示数字
 * 10-99: 直接显示数字
 * 99+: 显示 "99+"
 */
export function formatAlarmCount(count: number | undefined): string | null {
  if (!count || count <= 0) return null;
  if (count > ALARM_BADGE_CONFIG.maxDisplay) return '99+';
  return String(count);
}

/**
 * 创建设备标记样式
 * 在线设备：绿色渐变圆点 + 白色边框 + 告警角标（如有）
 * 离线设备：红色实心圆点 + 白色边框（无告警角标）
 */
export function createDeviceStyle(device: MapDevice, zoom: number): Style {
  const config = DEVICE_STATUS_CONFIG[device.status] || DEVICE_STATUS_CONFIG.offline;
  const radius = getMarkerRadius(zoom);

  return new Style({
    image: new Circle({
      radius,
      fill: new Fill({ color: config.color }),
      stroke: new Stroke({
        color: COLORS.white,
        width: MARKER_SIZE_CONFIG.strokeWidth,
      }),
    }),
  });
}

/**
 * 创建聚合标记样式
 * 蓝色渐变圆圈 + 数量文字
 */
export function createClusterStyle(count: number): Style {
  const radius = getClusterRadius(count);

  return new Style({
    image: new Circle({
      radius,
      fill: new Fill({ color: COLORS.clusterEnd }),
      stroke: new Stroke({
        color: COLORS.white,
        width: 2,
      }),
    }),
    text: new Text({
      text: count.toString(),
      fill: new Fill({ color: COLORS.white }),
      font: 'bold 12px sans-serif',
      textAlign: 'center',
      textBaseline: 'middle',
    }),
  });
}

/**
 * 创建高亮设备样式（搜索定位）
 * 节点大小不变 + 水波纹扩散效果
 */
export function createHighlightStyle(
  device: MapDevice,
  zoom: number,
  rippleWaves: { radius: number; opacity: number }[] = []
): Style[] {
  const baseRadius = getMarkerRadius(zoom);
  const config = DEVICE_STATUS_CONFIG[device.status] || DEVICE_STATUS_CONFIG.offline;

  const styles: Style[] = [];

  // 水波纹样式：从中心向外扩散的圆环
  rippleWaves.forEach(wave => {
    const rippleRadius = baseRadius + wave.radius;
    if (rippleRadius > baseRadius) {
      styles.push(new Style({
        image: new Circle({
          radius: rippleRadius,
          fill: new Fill({ color: 'transparent' }),
          stroke: new Stroke({
            color: `rgba(24, 144, 255, ${wave.opacity})`, // 蓝色波纹，透明度随扩散降低
            width: 1, // 细线条
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
  }));

  return styles;
}

/**
 * 创建悬停设备样式
 * 放大 1.2 倍
 */
export function createHoverStyle(device: MapDevice, zoom: number): Style {
  const baseRadius = getMarkerRadius(zoom);
  const config = DEVICE_STATUS_CONFIG[device.status] || DEVICE_STATUS_CONFIG.offline;

  return new Style({
    image: new Circle({
      radius: baseRadius * 1.2,
      fill: new Fill({ color: config.color }),
      stroke: new Stroke({
        color: COLORS.white,
        width: MARKER_SIZE_CONFIG.strokeWidth + 1,
      }),
    }),
  });
}

/**
 * OpenLayers StyleFunction for device layer
 */
export function deviceStyleFunction(feature: Feature, resolution: number): Style | Style[] {
  const zoom = Math.round(Math.log2(40075016.686 / (resolution * 256)));
  const device = feature.getProperties() as MapDevice;

  // 检查是否高亮
  const isHighlighted = feature.get('highlighted');
  if (isHighlighted) {
    return createHighlightStyle(device, zoom);
  }

  // 检查是否悬停
  const isHovered = feature.get('hovered');
  if (isHovered) {
    return createHoverStyle(device, zoom);
  }

  return createDeviceStyle(device, zoom);
}

/**
 * OpenLayers StyleFunction for cluster layer
 */
export function clusterStyleFunction(feature: Feature, _resolution: number): Style | Style[] {
  const features = feature.get('features') as Feature[] | undefined;
  const count = features?.length || 1;

  if (count === 1 && features) {
    // 单个设备
    const singleFeature = features[0];
    const device = singleFeature.getProperties() as MapDevice;

    // 检查是否高亮
    const isHighlighted = singleFeature.get('highlighted');
    if (isHighlighted) {
      // 获取水波纹数据
      const rippleWaves = singleFeature.get('rippleWaves') || [];
      return createHighlightStyle(device, 15, rippleWaves);
    }

    // 检查是否悬停
    const isHovered = singleFeature.get('hovered');
    if (isHovered) {
      return createHoverStyle(device, 15);
    }

    return createDeviceStyle(device, 15);
  }

  return createClusterStyle(count);
}

/**
 * 生成告警角标 SVG Data URL
 * 用于在设备标记右上角显示告警数量
 */
export function createAlarmBadgeDataUrl(count: number): string {
  const display = formatAlarmCount(count);
  if (!display) return '';

  const isLarge = count > ALARM_BADGE_CONFIG.maxDisplay;
  const radius = isLarge ? ALARM_BADGE_CONFIG.largeRadius : ALARM_BADGE_CONFIG.radius;
  const fontSize = isLarge ? ALARM_BADGE_CONFIG.largeFontSize : ALARM_BADGE_CONFIG.fontSize;
  const width = radius * 2 + 4;

  const svg = `
    <svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${radius * 2}" viewBox="0 0 ${width} ${radius * 2}">
      <defs>
        <linearGradient id="alarmGrad" x1="0%" y1="0%" x2="0%" y2="100%">
          <stop offset="0%" style="stop-color:${COLORS.alarmStart}"/>
          <stop offset="100%" style="stop-color:${COLORS.alarmEnd}"/>
        </linearGradient>
      </defs>
      <circle cx="${width / 2}" cy="${radius}" r="${radius}" fill="url(#alarmGrad)"/>
      <text x="${width / 2}" y="${radius + 3}" text-anchor="middle" font-family="sans-serif" font-size="${fontSize}" font-weight="bold" fill="white">${display}</text>
    </svg>
  `;

  return `data:image/svg+xml,${encodeURIComponent(svg.trim())}`;
}

/**
 * 生成带告警角标的设备标记 SVG Data URL
 */
export function createDeviceMarkerDataUrl(
  status: DeviceStatus,
  alarmCount?: number,
  size: 'small' | 'medium' | 'large' = 'medium'
): string {
  const config = DEVICE_STATUS_CONFIG[status] || DEVICE_STATUS_CONFIG.offline;
  const radius = MARKER_SIZE_CONFIG[size].radius;
  const strokeWidth = MARKER_SIZE_CONFIG.strokeWidth;
  const showAlarm = status === 'onlineActive' && alarmCount && alarmCount > 0;

  let alarmBadgeSvg = '';
  let viewBoxWidth = radius * 2 + strokeWidth * 2;
  let viewBoxHeight = radius * 2 + strokeWidth * 2;

  if (showAlarm) {
    const badgeRadius = alarmCount! > ALARM_BADGE_CONFIG.maxDisplay
      ? ALARM_BADGE_CONFIG.largeRadius
      : ALARM_BADGE_CONFIG.radius;
    viewBoxWidth = radius * 2 + badgeRadius * 2;
    viewBoxHeight = radius * 2 + badgeRadius * 2;
    const display = formatAlarmCount(alarmCount!);
    const fontSize = alarmCount! > ALARM_BADGE_CONFIG.maxDisplay
      ? ALARM_BADGE_CONFIG.largeFontSize
      : ALARM_BADGE_CONFIG.fontSize;

    alarmBadgeSvg = `
      <circle cx="${viewBoxWidth - badgeRadius}" cy="${badgeRadius}" r="${badgeRadius}" fill="${COLORS.alarmEnd}"/>
      <text x="${viewBoxWidth - badgeRadius}" y="${badgeRadius + 3}" text-anchor="middle" font-family="sans-serif" font-size="${fontSize}" font-weight="bold" fill="white">${display}</text>
    `;
  }

  const centerOffset = showAlarm ? radius + strokeWidth : radius + strokeWidth;
  const gradientDef = status === 'onlineActive'
    ? `
      <defs>
        <linearGradient id="markerGrad" x1="0%" y1="0%" x2="0%" y2="100%">
          <stop offset="0%" style="stop-color:${config.gradientStart}"/>
          <stop offset="100%" style="stop-color:${config.gradientEnd}"/>
        </linearGradient>
      </defs>`
    : '';

  const fillColor = status === 'onlineActive' ? 'url(#markerGrad)' : config.color;

  const svg = `
    <svg xmlns="http://www.w3.org/2000/svg" width="${viewBoxWidth}" height="${viewBoxHeight}" viewBox="0 0 ${viewBoxWidth} ${viewBoxHeight}">
      ${gradientDef}
      <circle cx="${centerOffset}" cy="${centerOffset}" r="${radius}" fill="${fillColor}" stroke="${COLORS.white}" stroke-width="${strokeWidth}"/>
      ${alarmBadgeSvg}
    </svg>
  `;

  return `data:image/svg+xml,${encodeURIComponent(svg.trim())}`;
}

/**
 * 计算两点之间的距离（米）
 * 使用 Haversine 公式
 */
export function calculateDistance(
  lat1: number,
  lng1: number,
  lat2: number,
  lng2: number
): number {
  const R = 6371000; // 地球半径（米）
  const dLat = ((lat2 - lat1) * Math.PI) / 180;
  const dLng = ((lng2 - lng1) * Math.PI) / 180;
  const a =
    Math.sin(dLat / 2) * Math.sin(dLat / 2) +
    Math.cos((lat1 * Math.PI) / 180) *
      Math.cos((lat2 * Math.PI) / 180) *
      Math.sin(dLng / 2) *
      Math.sin(dLng / 2);
  const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
  return R * c;
}

/**
 * 判断点是否在边界内
 */
export function isPointInBounds(
  lng: number,
  lat: number,
  minLng: number,
  maxLng: number,
  minLat: number,
  maxLat: number
): boolean {
  return lng >= minLng && lng <= maxLng && lat >= minLat && lat <= maxLat;
}

/**
 * Spiderfy 配置
 */
export const SPIDERFY_CONFIG = {
  /** 展开半径（像素） */
  radius: 60,
  /** 连线宽度 */
  lineWidth: 2,
  /** 连线颜色 */
  lineColor: 'rgba(24, 144, 255, 0.6)',
  /** 展开点半径 */
  pointRadius: 10,
  /** 展开动画时长（ms） */
  animationDuration: 300,
};

/**
 * 创建 Spiderfy 连线样式
 */
export function createSpiderfyLineStyle(): Style {
  return new Style({
    stroke: new Stroke({
      color: SPIDERFY_CONFIG.lineColor,
      width: SPIDERFY_CONFIG.lineWidth,
    }),
  });
}

/**
 * 创建 Spiderfy 展开点样式
 */
export function createSpiderfyPointStyle(device: MapDevice, index: number, _total: number): Style {
  const config = DEVICE_STATUS_CONFIG[device.status] || DEVICE_STATUS_CONFIG.offline;

  return new Style({
    image: new Circle({
      radius: SPIDERFY_CONFIG.pointRadius,
      fill: new Fill({ color: config.color }),
      stroke: new Stroke({
        color: COLORS.white,
        width: 2,
      }),
    }),
    text: new Text({
      text: String(index + 1),
      fill: new Fill({ color: COLORS.white }),
      font: 'bold 10px sans-serif',
      textAlign: 'center',
      textBaseline: 'middle',
    }),
  });
}

/**
 * 创建 Spiderfy 展开点悬停样式
 */
export function createSpiderfyPointHoverStyle(device: MapDevice, index: number, _total: number): Style {
  const config = DEVICE_STATUS_CONFIG[device.status] || DEVICE_STATUS_CONFIG.offline;

  return new Style({
    image: new Circle({
      radius: SPIDERFY_CONFIG.pointRadius * 1.3, // 悬停时放大 1.3 倍
      fill: new Fill({ color: config.color }),
      stroke: new Stroke({
        color: COLORS.primary, // 使用主题色边框
        width: 3,
      }),
    }),
    text: new Text({
      text: String(index + 1),
      fill: new Fill({ color: COLORS.white }),
      font: 'bold 11px sans-serif', // 字体稍微增大
      textAlign: 'center',
      textBaseline: 'middle',
    }),
  });
}

/**
 * 创建 Spiderfy 中心点样式
 */
export function createSpiderfyCenterStyle(count: number): Style {
  return new Style({
    image: new Circle({
      radius: 8,
      fill: new Fill({ color: COLORS.primary }),
      stroke: new Stroke({
        color: COLORS.white,
        width: 2,
      }),
    }),
    text: new Text({
      text: String(count),
      fill: new Fill({ color: COLORS.white }),
      font: 'bold 10px sans-serif',
      textAlign: 'center',
      textBaseline: 'middle',
    }),
  });
}
