/**
 * 地图元数据加载 Hook - 支持 TileJSON 格式
 * @module components/GISMap/useMapConfig
 *
 * 功能说明：
 * 1. 从服务端加载 Maperitive 生成的 tiles.json (TileJSON 格式)
 * 2. 自动转换为内部 MapMetadata 格式
 * 3. 异常场景自动降级到 DEFAULT_METADATA
 *
 * 异常场景处理：
 * - tiles.json 不存在（404）→ 降级到默认配置
 * - tiles.json 格式错误 → 降级到默认配置
 * - 网络请求失败 → 降级到默认配置
 * - Nginx 未配置 tiles 目录 → 降级到默认配置
 */

import { useState, useEffect } from 'react';

/**
 * TileJSON 格式（Maperitive / TileServer GL 生成）
 * 规范：https://github.com/mapbox/tilejson-spec
 *
 * 字段说明：
 * - bounds: [minLon, minLat, maxLon, maxLat] 地理边界数组
 * - center: [lon, lat, zoom] 中心点数组
 * - minzoom/maxzoom: 缩放级别范围
 */
export interface TileJSON {
  tilejson?: string;
  name?: string;
  description?: string;
  attribution?: string;
  tiles?: string[];
  minzoom?: number;
  maxzoom?: number;
  bounds?: [number, number, number, number];
  center?: [number, number, number];
}

/**
 * 内部统一格式
 */
export interface MapMetadata {
  name: string;
  region: string;
  center: { lon: number; lat: number; zoom: number };
  bounds: { minLon: number; maxLon: number; minLat: number; maxLat: number };
  zoom: { min: number; max: number; default: number };
  attribution: string;
  description: string;
}

/**
 * 默认元数据（兜底方案）
 *
 * 使用场景：
 * 1. tiles.json 文件不存在（404）
 * 2. tiles.json 格式错误
 * 3. 网络请求失败
 * 4. 服务端未配置 tiles 目录
 */
export const DEFAULT_METADATA: MapMetadata = {
  name: 'Zambia Offline Map',
  region: 'Zambia',
  // 赞比亚中心点（所有设备经纬度平均值）
  center: { lon: 28.221, lat: -14.607, zoom: 6 },
  // 赞比亚区域边界
  bounds: { minLon: 22.0, maxLon: 34.0, minLat: -18.0, maxLat: -8.0 },
  // 常用缩放范围
  zoom: { min: 6, max: 15, default: 6 },
  attribution: '© OpenStreetMap contributors',
  description: '赞比亚区域离线地图，包含 6-15 级瓦片',
};

/**
 * 安全获取数字值
 */
function safeNumber(value: unknown, defaultValue: number): number {
  return typeof value === 'number' && !isNaN(value) ? value : defaultValue;
}

/**
 * 安全获取数组值
 */
function safeArray<T>(value: unknown, length: number, defaultValue: T): T {
  if (Array.isArray(value) && value.length >= length) {
    return value as T;
  }
  return defaultValue;
}

/**
 * 安全获取字符串值
 */
function safeString(value: unknown, defaultValue: string): string {
  return typeof value === 'string' ? value : defaultValue;
}

/**
 * 将 TileJSON 转换为 MapMetadata（容错处理）
 *
 * 即使 TileJSON 格式不完整，也会尽力解析并提供可用值
 */
function tileJsonToMetadata(tilejson: TileJSON): MapMetadata {
  // 解析 bounds，容错处理
  const defaultBounds: [number, number, number, number] = [22.0, -18.0, 34.0, -8.0];
  const bounds = safeArray(tilejson.bounds, 4, defaultBounds);
  const [minLon, minLat, maxLon, maxLat] = bounds;

  // 解析 center，容错处理
  const defaultCenter: [number, number, number] = [28.221, -14.607, 6];
  const center = safeArray(tilejson.center, 3, defaultCenter);
  const [lon, lat, zoom] = center;

  // 解析缩放级别，容错处理
  const minzoom = safeNumber(tilejson.minzoom, 6);
  const maxzoom = safeNumber(tilejson.maxzoom, 15);

  // 解析字符串字段，容错处理
  const name = safeString(tilejson.name, 'Offline Map');
  const description = safeString(tilejson.description, 'Offline map tiles');
  const attribution = safeString(
    tilejson.attribution,
    '© OpenStreetMap contributors'
  );

  return {
    name,
    region: name,
    center: { lon, lat, zoom },
    bounds: { minLon, maxLon, minLat, maxLat },
    zoom: {
      min: minzoom,
      max: maxzoom,
      default: zoom,
    },
    attribution,
    description,
  };
}

/**
 * 加载状态类型
 */
export type LoadStatus =
  | 'idle' // 未开始
  | 'loading' // 加载中
  | 'success' // 成功
  | 'error'; // 失败（已降级）

/**
 * 地图元数据加载 Hook
 *
 * @returns {Object}
 * - metadata: 元数据对象（成功时来自 tiles.json，失败时为 DEFAULT_METADATA）
 * - loading: 是否正在加载
 * - error: 错误信息（仅在失败时有值）
 * - status: 加载状态
 * - isUsingDefault: 是否使用了默认配置（用于调试）
 */
export function useMapConfig() {
  const [metadata, setMetadata] = useState<MapMetadata | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [status, setStatus] = useState<LoadStatus>('idle');
  const [isUsingDefault, setIsUsingDefault] = useState(false);

  useEffect(() => {
    const fetchMetadata = async () => {
      setStatus('loading');
      setIsUsingDefault(false);

      try {
        const response = await fetch('/tiles-metadata');

        if (response.ok) {
          const tilejson: TileJSON = await response.json();

          // 验证 TileJSON 基本结构
          if (tilejson && (tilejson.bounds || tilejson.center)) {
            const converted = tileJsonToMetadata(tilejson);
            setMetadata(converted);
            setStatus('success');
          } else {
            // TileJSON 格式不完整，使用默认值
            setMetadata(DEFAULT_METADATA);
            setStatus('error');
            setIsUsingDefault(true);
            setError('TileJSON format incomplete');
          }
        } else {
          // HTTP 错误（404/500 等），使用默认值
          setMetadata(DEFAULT_METADATA);
          setStatus('error');
          setIsUsingDefault(true);
          setError(`HTTP ${response.status}: ${response.statusText}`);
        }
      } catch (err) {
        // 网络错误或其他异常，使用默认值
        const errorMessage =
          err instanceof Error ? err.message : 'Unknown error';
        setMetadata(DEFAULT_METADATA);
        setStatus('error');
        setIsUsingDefault(true);
        setError(errorMessage);
      } finally {
        setLoading(false);
      }
    };

    fetchMetadata();
  }, []);

  return {
    metadata,
    loading,
    error,
    status,
    isUsingDefault,
  };
}
