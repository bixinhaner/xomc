/**
 * 地图元数据验证工具
 * @module utils/mapValidation
 *
 * 用于验证从 tiles.json 加载的地图元数据是否有效
 * 防止无效数据导致地图渲染异常
 */

import type { MapMetadata } from '@/components/GISMap/useMapConfig';

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
