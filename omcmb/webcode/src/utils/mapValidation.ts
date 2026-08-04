/**
 * 地图元数据验证工具
 * @module utils/mapValidation
 *
 * 用于验证从 tiles.json 加载的地图元数据是否有效
 * 防止无效数据导致地图渲染异常
 */

import type { MapMetadata } from '@/components/GISMap/useMapConfig';

export type CenterPointSource = 'metadata' | 'device_data' | 'env_config' | 'default';

export interface MapInitialView {
  center: [number, number];
  zoom: number;
  source: CenterPointSource;
}

interface MapInitialViewOptions {
  metadata: MapMetadata | null;
  metadataAvailable: boolean;
  devices: DeviceCoordinate[];
  envCenter: { center: [number, number]; zoom: number } | null;
  defaultCenter: [number, number];
  defaultZoom: number;
}

/**
 * 验证地图元数据是否有效
 *
 * 验证规则：
 * 1. metadata 不为 null/undefined
 * 2. center 存在且 lon/lat/zoom 为有效数字
 * 3. bounds 存在且四个边界值为有效数字
 * 4. zoom 存在且 min/max 为有效数字
 *
 * @param data - 待验证的元数据
 * @returns 类型守卫，验证通过时 data 为 MapMetadata 类型
 *
 * @example
 * ```ts
 * if (isValidMapMetadata(metadata)) {
 *   // TypeScript 知道 metadata 是有效 MapMetadata
 *   const lon = metadata.center.lon;
 * }
 * ```
 */
export function isValidMapMetadata(data: MapMetadata | null): data is MapMetadata {
  if (!data) {
    return false;
  }

  const { center, bounds, zoom } = data;

  // 验证 center
  const validCenter = center &&
    typeof center.lon === 'number' && !isNaN(center.lon) &&
    typeof center.lat === 'number' && !isNaN(center.lat) &&
    typeof center.zoom === 'number' && !isNaN(center.zoom);

  if (!validCenter) {
    return false;
  }

  // 验证 bounds
  const validBounds = bounds &&
    typeof bounds.minLon === 'number' && !isNaN(bounds.minLon) &&
    typeof bounds.maxLon === 'number' && !isNaN(bounds.maxLon) &&
    typeof bounds.minLat === 'number' && !isNaN(bounds.minLat) &&
    typeof bounds.maxLat === 'number' && !isNaN(bounds.maxLat);

  if (!validBounds) {
    return false;
  }

  // 验证 zoom
  const validZoom = zoom &&
    typeof zoom.min === 'number' && !isNaN(zoom.min) &&
    typeof zoom.max === 'number' && !isNaN(zoom.max);

  if (!validZoom) {
    return false;
  }

  return true;
}

/**
 * 从验证通过的 metadata 构建安全的地图配置
 *
 * @param metadata - 已验证的元数据
 * @returns 地图配置对象
 *
 * @example
 * ```ts
 * if (isValidMapMetadata(metadata)) {
 *   const config = buildSafeConfig(metadata);
 * }
 * ```
 */
export function buildSafeConfig(metadata: MapMetadata) {
  return {
    /** 默认中心点 [经度, 纬度] */
    defaultCenter: [metadata.center.lon, metadata.center.lat] as [number, number],
    /** 默认缩放级别 */
    defaultZoom: metadata.center.zoom,
    /** 最小缩放级别 */
    minZoom: metadata.zoom.min,
    /** 最大缩放级别 */
    maxZoom: metadata.zoom.max,
    /**
     * 瓦片服务地址
     * 优先使用环境变量 VITE_MAP_TILE_URL，否则使用默认离线路径
     */
    tileUrl: import.meta.env.VITE_MAP_TILE_URL || '/tiles/{z}/{x}/{y}.png',
    /** 版权信息 */
    attribution: metadata.attribution || '© OpenStreetMap contributors',
    /** 地理边界 */
    bounds: metadata.bounds,
  };
}

/**
 * 将经纬度转换为瓦片坐标（XYZ tile scheme）
 *
 * @param lon - 经度
 * @param lat - 纬度
 * @param zoom - 缩放级别
 * @returns [x, y] 瓦片坐标
 */
function lonLatToTileXY(lon: number, lat: number, zoom: number): [number, number] {
  // 限制纬度在有效范围内，避免极地计算问题（Web Mercator 覆盖范围：±85.0511°）
  const clampedLat = Math.max(-85, Math.min(85, lat));

  const n = Math.pow(2, zoom);
  const x = Math.floor((lon + 180) / 360 * n);
  const latRad = (clampedLat * Math.PI) / 180;
  const y = Math.floor((1 - Math.log(Math.tan(latRad) + 1 / Math.cos(latRad)) / Math.PI) / 2 * n);
  return [x, y];
}

/**
 * 检查离线瓦片文件是否实际可用
 *
 * 通过请求一个具体的瓦片文件来验证瓦片服务是否正常工作
 * 采样策略：使用 metadata.center 对应的瓦片坐标进行检查
 *
 * @param metadata - 地图元数据
 * @param timeout - 超时时间（毫秒），默认 800ms
 * @returns Promise<boolean> 瓦片是否可用
 *
 * @example
 * ```ts
 * const available = await checkTileAvailability(metadata);
 * if (!available) {
 *   console.warn('离线瓦片不可用，降级到在线地图');
 * }
 * ```
 */
export async function checkTileAvailability(
  metadata: MapMetadata | null,
  timeout = 800
): Promise<boolean> {
  if (!metadata) {
    return false;
  }

  // 使用 metadata 的默认缩放级别作为采样级别（通常是中等级别，更有可能有瓦片）
  const sampleZoom = metadata.zoom.default || Math.floor((metadata.zoom.min + metadata.zoom.max) / 2);

  // 从 metadata.center 计算瓦片坐标（确保采样点在覆盖区域内）
  const [x, y] = lonLatToTileXY(
    metadata.center.lon,
    metadata.center.lat,
    sampleZoom
  );

  const tileUrl = `/tiles/${sampleZoom}/${x}/${y}.png`;

  try {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), timeout);

    const response = await fetch(tileUrl, {
      method: 'HEAD',
      signal: controller.signal,
    });

    clearTimeout(timeoutId);

    if (response.ok) {
      return true;
    }

    return false;
  } catch {
    return false;
  }
}

/**
 * 从设备数据计算地图中心点和缩放级别
 *
 * 算法：
 * 1. 计算所有设备的经纬度范围（minLng, maxLng, minLat, maxLat）
 * 2. 中心点 = [(minLng+maxLng)/2, (minLat+maxLat)/2]
 * 3. 根据设备分布范围自动选择缩放级别
 *
 * 缩放级别选择规则：
 * - 范围 > 50°：zoom = 4（全国级别，如中国）
 * - 范围 > 20°：zoom = 6（省级别）
 * - 范围 > 5°：zoom = 8（市级别）
 * - 范围 > 1°：zoom = 11（县级别）
 * - 其他：zoom = 13（详细级别）
 *
 * @param devices 设备地理位置数据数组，每个设备必须有有效的 longitude 和 latitude
 * @returns 中心点、缩放级别和边界范围；如果设备列表为空则返回全球默认值
 *
 * @example
 * ```ts
 * const result = calculateCenterFromDevices(devices);
 * const [lng, lat] = result.center;
 * map.setCenter([lng, lat], result.zoom);
 * ```
 */
interface DeviceCoordinate {
  longitude: number | null | undefined;
  latitude: number | null | undefined;
}

interface ValidDeviceCoordinate {
  longitude: number;
  latitude: number;
}

/**
 * 按统一优先级解析地图初始视图。
 *
 * metadataAvailable 必须由调用方根据离线元数据和瓦片健康状态确认，
 * 避免把请求失败时的兼容默认值误认为有效离线地图中心点。
 */
export function resolveMapInitialView(options: MapInitialViewOptions): MapInitialView {
  const {
    metadata,
    metadataAvailable,
    devices,
    envCenter,
    defaultCenter,
    defaultZoom,
  } = options;

  if (metadataAvailable && isValidMapMetadata(metadata)) {
    return {
      center: [metadata.center.lon, metadata.center.lat],
      zoom: metadata.center.zoom,
      source: 'metadata',
    };
  }

  const deviceView = calculateCenterFromDevices(devices);
  if (deviceView.hasDevices) {
    return {
      center: deviceView.center,
      zoom: deviceView.zoom,
      source: 'device_data',
    };
  }

  if (envCenter) {
    return {
      center: envCenter.center,
      zoom: envCenter.zoom,
      source: 'env_config',
    };
  }

  return {
    center: defaultCenter,
    zoom: defaultZoom,
    source: 'default',
  };
}

export function calculateCenterFromDevices(devices: DeviceCoordinate[]): {
  center: [number, number];
  zoom: number;
  bounds: { minLng: number; maxLng: number; minLat: number; maxLat: number };
  hasDevices: boolean;
} {
  const isValidCoordinate = (device: DeviceCoordinate): device is ValidDeviceCoordinate =>
    Number.isFinite(device.longitude) && Number.isFinite(device.latitude);

  // 过滤掉无效坐标，避免 NaN 传播导致中心点异常
  const validDevices = (devices || []).filter(isValidCoordinate);

  // 处理空列表
  if (validDevices.length === 0) {
    return {
      center: [0, 20],
      zoom: 2,
      bounds: { minLng: -180, maxLng: 180, minLat: -90, maxLat: 90 },
      hasDevices: false,
    };
  }

  // 初始化边界范围
  let minLng = validDevices[0].longitude;
  let maxLng = validDevices[0].longitude;
  let minLat = validDevices[0].latitude;
  let maxLat = validDevices[0].latitude;

  // 遍历计算最小/最大值
  for (const device of validDevices) {
    minLng = Math.min(minLng, device.longitude);
    maxLng = Math.max(maxLng, device.longitude);
    minLat = Math.min(minLat, device.latitude);
    maxLat = Math.max(maxLat, device.latitude);
  }

  // 计算中心点
  const centerLng = (minLng + maxLng) / 2;
  const centerLat = (minLat + maxLat) / 2;

  // 计算范围
  const lngRange = maxLng - minLng;
  const latRange = maxLat - minLat;
  const maxRange = Math.max(lngRange, latRange);

  // 根据范围自动选择缩放级别
  let zoom = 13; // 默认值
  if (maxRange > 50) {
    zoom = 4;
  } else if (maxRange > 20) {
    zoom = 6;
  } else if (maxRange > 5) {
    zoom = 8;
  } else if (maxRange > 1) {
    zoom = 11;
  }

  return {
    center: [centerLng, centerLat],
    zoom,
    bounds: { minLng, maxLng, minLat, maxLat },
    hasDevices: true,
  };
}

/**
 * 解析和验证环境变量中的默认中心点配置
 *
 * 格式：VITE_MAP_DEFAULT_CENTER="lng,lat,zoom"
 * 示例：
 * - "28.221,-14.607,6"  (赞比亚)
 * - "104.0,35.0,4"      (中国)
 * - "-3.436,55.378,6"   (英国)
 * - "-95.713,37.090,4"  (美国)
 *
 * @returns 中心点和缩放级别，或 null 如果未配置或格式错误
 *
 * @example
 * ```ts
 * const envCenter = parseEnvCenter();
 * if (envCenter) {
 *   const [lng, lat] = envCenter.center;
 *   map.setCenter([lng, lat], envCenter.zoom);
 * }
 * ```
 */
export function parseEnvCenter(): {
  center: [number, number];
  zoom: number;
} | null {
  const envValue = import.meta.env.VITE_MAP_DEFAULT_CENTER;

  // 未配置
  if (!envValue) {
    return null;
  }

  // 解析逗号分隔的值
  const parts = envValue.split(',').map((p: string) => p.trim());

  // 必须至少有经度和纬度
  if (parts.length < 2) {
    console.warn('[mapValidation] Invalid VITE_MAP_DEFAULT_CENTER format, expected "lng,lat[,zoom]"');
    return null;
  }

  try {
    const lng = parseFloat(parts[0]);
    const lat = parseFloat(parts[1]);
    const zoom = parts.length > 2 ? parseInt(parts[2], 10) : 13;

    // 验证值的有效性
    if (isNaN(lng) || isNaN(lat) || isNaN(zoom)) {
      console.warn('[mapValidation] VITE_MAP_DEFAULT_CENTER contains non-numeric values');
      return null;
    }

    // 验证坐标范围
    if (lng < -180 || lng > 180) {
      console.warn(`[mapValidation] Longitude ${lng} out of range [-180, 180]`);
      return null;
    }

    if (lat < -90 || lat > 90) {
      console.warn(`[mapValidation] Latitude ${lat} out of range [-90, 90]`);
      return null;
    }

    // 验证缩放级别范围
    if (zoom < 1 || zoom > 18) {
      console.warn(`[mapValidation] Zoom ${zoom} out of range [1, 18]`);
      return null;
    }

    console.info(`[mapValidation] Parsed VITE_MAP_DEFAULT_CENTER: center=[${lng},${lat}], zoom=${zoom}`);

    return {
      center: [lng, lat],
      zoom
    };
  } catch (err) {
    console.warn('[mapValidation] Error parsing VITE_MAP_DEFAULT_CENTER:', err);
    return null;
  }
}
