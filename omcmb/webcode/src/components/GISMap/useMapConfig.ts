/**
 * 地图元数据加载 Hook - 支持 TileJSON 格式
 * @module components/GISMap/useMapConfig
 *
 * 功能说明：
 * 1. 从服务端加载 Maperitive 生成的 tiles.json (TileJSON 格式)
 * 2. 自动转换为内部 MapMetadata 格式
 * 3. 异常场景自动降级到 DEFAULT_METADATA
 * 4. 全局缓存避免重复请求（React StrictMode 双重渲染）
 *
 * 异常场景处理：
 * - tiles.json 不存在（404）→ 降级到默认配置
 * - tiles.json 格式错误 → 降级到默认配置
 * - 网络请求失败 → 降级到默认配置
 * - Nginx 未配置 tiles 目录 → 降级到默认配置
 */

import { useState, useEffect } from 'react';

// 全局缓存状态，避免 React StrictMode 双重渲染导致重复请求
let globalMetadataCache: MapMetadata | null = null;
let globalFetchPromise: Promise<void> | null = null;
let globalCacheIsDefault = false;
let globalCacheError: string | null = null;

// tiles-metadata 最大等待时间（ms）
const METADATA_TIMEOUT_MS = 5000;

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
  const [metadata, setMetadata] = useState<MapMetadata | null>(() => globalMetadataCache);
  const [loading, setLoading] = useState(() => globalMetadataCache === null);
  const [error, setError] = useState<string | null>(() => globalCacheError);
  const [status, setStatus] = useState<LoadStatus>(() => {
    if (!globalMetadataCache) return 'idle';
    return globalCacheIsDefault ? 'error' : 'success';
  });
  const [isUsingDefault, setIsUsingDefault] = useState(() => globalCacheIsDefault);

  useEffect(() => {
    // 如果已有缓存，直接使用
    if (globalMetadataCache) {
      // 初始化 state 已从缓存读取，这里无需再次 setState，避免级联渲染
      return;
    }

    // 如果正在获取中，等待同一个 Promise
    if (globalFetchPromise) {
      globalFetchPromise.then(() => {
        if (globalMetadataCache) {
          setMetadata(globalMetadataCache);
          setLoading(false);
          setStatus(globalCacheIsDefault ? 'error' : 'success');
          setIsUsingDefault(globalCacheIsDefault);
          setError(globalCacheError);
        }
      });
      return;
    }

    const fetchMetadata = async () => {
      setStatus('loading');
      setIsUsingDefault(false);
      setError(null);

      let timeoutId: ReturnType<typeof setTimeout> | null = null;
      const controller = new AbortController();

      const applyDefaultFallback = (reason: string, logMessage: string) => {
        console.warn(`[MapConfig] ${logMessage}`);
        globalMetadataCache = DEFAULT_METADATA;
        globalCacheIsDefault = true;
        globalCacheError = reason;
        setMetadata(DEFAULT_METADATA);
        setStatus('error');
        setIsUsingDefault(true);
        setError(reason);
      };

      try {
        timeoutId = setTimeout(() => {
          controller.abort();
        }, METADATA_TIMEOUT_MS);

        const response = await fetch('/tiles-metadata', {
          signal: controller.signal,
        });

        if (response.ok) {
          let tilejson: TileJSON;
          try {
            tilejson = await response.json();
          } catch {
            applyDefaultFallback('json_parse_error', 'tiles-metadata parse failed, fallback to defaults');
            return;
          }

          // 验证 TileJSON 基本结构
          if (tilejson && (tilejson.bounds || tilejson.center)) {
            const converted = tileJsonToMetadata(tilejson);
            globalMetadataCache = converted; // 缓存结果
            globalCacheIsDefault = false;
            globalCacheError = null;
            setMetadata(converted);
            setStatus('success');
            setIsUsingDefault(false);
            setError(null);
          } else {
            // TileJSON 格式不完整，使用默认值
            applyDefaultFallback('tilejson_incomplete', 'tiles-metadata incomplete, fallback to defaults');
          }
        } else {
          // HTTP 错误（404/500 等），使用默认值
          applyDefaultFallback(
            `http_${response.status}`,
            `tiles-metadata http error ${response.status}, fallback to defaults`
          );
        }
      } catch (err) {
        // 网络错误或其他异常，使用默认值
        const isTimeoutAbort = err instanceof Error && err.name === 'AbortError';
        if (isTimeoutAbort) {
          applyDefaultFallback(
            `timeout_${METADATA_TIMEOUT_MS}ms`,
            `tiles-metadata timeout after ${METADATA_TIMEOUT_MS}ms, fallback to defaults`
          );
        } else {
          const errorMessage = err instanceof Error ? err.message : 'unknown_error';
          applyDefaultFallback(`network_error_${errorMessage}`, 'tiles-metadata network error, fallback to defaults');
        }
      } finally {
        if (timeoutId) {
          clearTimeout(timeoutId);
        }
        setLoading(false);
        globalFetchPromise = null; // 清空 Promise
      }
    };

    globalFetchPromise = fetchMetadata();
    globalFetchPromise.catch(() => {
      // 错误已在 fetchMetadata 中处理
    });
  }, []);

  return {
    metadata,
    loading,
    error,
    status,
    isUsingDefault,
  };
}
