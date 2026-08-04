/**
 * 地图元数据加载 Hook - 支持 TileJSON 格式
 * @module components/GISMap/useMapConfig
 *
 * 功能说明：
 * 1. 从服务端加载 Maperitive 生成的 tiles.json (TileJSON 格式)
 * 2. 自动转换为内部 MapMetadata 格式
 * 3. 异常场景返回无元数据状态，由调用方继续走设备/环境变量兜底
 * 4. 全局缓存避免重复请求（React StrictMode 双重渲染）
 *
 * 异常场景处理：
 * - tiles.json 不存在（404）→ 返回无元数据状态
 * - tiles.json 格式错误 → 返回无元数据状态
 * - 网络请求失败 → 返回无元数据状态
 * - Nginx 未配置 tiles 目录 → 返回无元数据状态
 */

import { useState, useEffect } from 'react';
import { checkTileAvailability, isValidMapMetadata } from '@/utils/mapValidation';

// 全局缓存状态，避免 React StrictMode 双重渲染导致重复请求
let globalMetadataCache: MapMetadata | null = null;
let globalFetchPromise: Promise<void> | null = null;
let globalLastFetchFailed = false;
let globalLastError: string | null = null;
let globalTilesAvailable: boolean | null = null;
let globalTileCheckPromise: Promise<boolean> | null = null;
// 失败时间戳：用于在短时间内复用上次的错误结果，避免每次组件挂载都重新等满超时
let globalLastFailedAt: number | null = null;

const DEFAULT_METADATA_TIMEOUT_MS = 2500;
// 失败冷却：短时间内（默认 30s）跳过重新 fetch，直接复用 DEFAULT_METADATA
// 避免用户在 metadata 持续不可用时每次进入地图页都转一次满超时的圈
const FAILURE_COOLDOWN_MS = 30_000;

function getMetadataTimeoutMs(): number {
  const rawEnv = import.meta.env.VITE_MAP_METADATA_TIMEOUT_MS;
  // 空串 / 纯空白 / undefined 都等同于未配置，避免 Number('') === 0 / Number(' ') === 0 被 clamp 到 500
  const trimmed = typeof rawEnv === 'string' ? rawEnv.trim() : '';
  if (!trimmed) {
    return DEFAULT_METADATA_TIMEOUT_MS;
  }
  const raw = Number(trimmed);

  if (!Number.isFinite(raw)) {
    return DEFAULT_METADATA_TIMEOUT_MS;
  }

  // 保护性夹取，避免配置异常导致过小或过大超时
  return Math.min(10000, Math.max(500, Math.floor(raw)));
}

// 处在失败冷却窗内 → 直接复用上次错误，不再发请求
function isInFailureCooldown(): boolean {
  return (
    globalLastFetchFailed &&
    globalLastFailedAt !== null &&
    Date.now() - globalLastFailedAt < FAILURE_COOLDOWN_MS
  );
}

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
 * 兼容保留的默认元数据。
 *
 * 中心点决策不再使用它作为失败兜底，避免离线地图不可用时误跳到某个
 * 历史部署区域。实际失败状态由 metadata=null 表示。
 *
 * 使用场景：
 * 1. tiles.json 文件不存在（404）
 * 2. tiles.json 格式错误
 * 3. 网络请求失败
 * 4. 服务端未配置 tiles 目录
 */
export const DEFAULT_METADATA: MapMetadata = {
  name: 'No Offline Map Metadata',
  region: 'default',
  center: { lon: 104.0, lat: 35.0, zoom: 4 },
  bounds: { minLon: -180, maxLon: 180, minLat: -85, maxLat: 85 },
  zoom: { min: 1, max: 18, default: 4 },
  attribution: '© OpenStreetMap contributors',
  description: 'Offline map metadata unavailable',
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
  // 三种初始情形：
  // a) 命中成功缓存 → 立即返回缓存的 metadata，不 loading
  // b) 处在失败冷却窗 → 立即返回无元数据状态，不 loading（避免再转 2.5s 圈）
  // c) 首次或冷却已过期 → metadata=null + loading=true，等 fetch
  const inCooldown = isInFailureCooldown();

  const [metadata, setMetadata] = useState<MapMetadata | null>(() => {
    if (globalMetadataCache) return globalMetadataCache;
    if (inCooldown) return null;
    return null;
  });
  const [loading, setLoading] = useState(() => !globalMetadataCache && !inCooldown);
  const [error, setError] = useState<string | null>(() => {
    if (globalMetadataCache) return null;
    if (inCooldown) return globalLastError;
    return null;
  });
  const [status, setStatus] = useState<LoadStatus>(() => {
    if (globalMetadataCache) return 'success';
    if (inCooldown) return 'error';
    return 'idle';
  });
  const [isUsingDefault, setIsUsingDefault] = useState(() => inCooldown);
  const [tilesAvailable, setTilesAvailable] = useState<boolean | null>(() => globalTilesAvailable);

  useEffect(() => {
    // a) 命中成功缓存 / b) 处在失败冷却窗：初始化器已置好所有状态，无需任何 setState
    if (globalMetadataCache || isInFailureCooldown()) {
      return;
    }

    // 正在获取中：等待同一个 Promise，避免并发挂载触发多次请求
    if (globalFetchPromise) {
      globalFetchPromise.then(() => {
        if (globalMetadataCache) {
          setMetadata(globalMetadataCache);
          setLoading(false);
          setStatus('success');
          setIsUsingDefault(false);
          setError(null);
        } else if (globalLastFetchFailed) {
          setMetadata(null);
          setLoading(false);
          setStatus('error');
          setIsUsingDefault(true);
          setError(globalLastError);
        }
      });
      return;
    }

    const fetchMetadata = async () => {
      // 覆盖 render 阶段初始化器可能留下的“冷却未过期”状态：
      // 在 render 与 effect 之间冷却刚好过期时，state 可能仍是 loading=false/status='error'，
      // 这里重新开始 fetch 前重置为 loading，避免 UI 与后台请求不一致。
      setLoading(true);
      setError(null);
      setStatus('loading');
      setIsUsingDefault(false);
      globalLastFetchFailed = false;
      globalLastError = null;
      globalLastFailedAt = null;
      globalTilesAvailable = null;
      setTilesAvailable(null);

      let timeoutId: ReturnType<typeof setTimeout> | null = null;
      let timedOut = false;

      try {
        // 创建 AbortController 用于超时控制
        const controller = new AbortController();

        // tiles-metadata 超时支持环境变量配置，默认 2.5 秒
        const TIMEOUT_MS = getMetadataTimeoutMs();

        timeoutId = setTimeout(() => {
          timedOut = true;
          controller.abort();
        }, TIMEOUT_MS);

        const response = await fetch('/tiles-metadata', {
          signal: controller.signal,
        });

        if (response.ok) {
          let tilejson: TileJSON;
          try {
            tilejson = await response.json();
          } catch {
            const parseError = 'json_parse_error:invalid_tiles_metadata_json';
            console.warn('[MapConfig] ⚠', parseError);
            setMetadata(null);
            setStatus('error');
            setIsUsingDefault(true);
            setError(parseError);
            globalLastFetchFailed = true;
            globalLastError = parseError;
            return;
          }

          // 验证 TileJSON 基本结构
          if (tilejson && (tilejson.bounds || tilejson.center)) {
            const converted = tileJsonToMetadata(tilejson);
            globalMetadataCache = converted; // 缓存结果
            setMetadata(converted);
            setStatus('success');
            setIsUsingDefault(false);
            setError(null);
          } else {
            // TileJSON 格式不完整，使用默认值
            setMetadata(null);
            setStatus('error');
            setIsUsingDefault(true);
            const parseError = 'json_parse_error:tilejson_format_incomplete';
            console.warn('[MapConfig] ⚠', parseError);
            setError(parseError);
            globalLastFetchFailed = true;
            globalLastError = parseError;
          }
        } else {
          // HTTP 错误（404/500 等），使用默认值
          setMetadata(null);
          setStatus('error');
          setIsUsingDefault(true);
          const httpError = `http_${response.status}:${response.statusText || 'unknown'}`;
          console.warn('[MapConfig] ⚠', httpError);
          setError(httpError);
          globalLastFetchFailed = true;
          globalLastError = httpError;
        }
      } catch (err) {
        // 网络错误、超时或其他异常，使用默认值
        const errorMessage = err instanceof Error ? err.message : 'Unknown error';
        const timeoutMs = getMetadataTimeoutMs();
        let classifiedError = `network_error_tiles_metadata:${errorMessage}`;

        // 判断是否是超时导致的中止
        if (err instanceof Error && err.name === 'AbortError') {
          if (timedOut) {
            classifiedError = `timeout_tiles_metadata:${timeoutMs}ms`;
            console.warn('[MapConfig] ⚠', classifiedError, 'offline map not available, using default config');
          } else {
            classifiedError = 'network_error_abort:tiles_metadata_request_aborted';
            console.warn('[MapConfig] ⚠', classifiedError, 'using default config');
          }
        } else {
          console.warn('[MapConfig] ⚠', classifiedError);
        }

        setMetadata(null);
        setStatus('error');
        setIsUsingDefault(true);
        setError(classifiedError);
        globalLastFetchFailed = true;
        globalLastError = classifiedError;
      } finally {
        if (timeoutId) {
          clearTimeout(timeoutId);
        }
        // 集中记录失败时间戳，触发后续 FAILURE_COOLDOWN_MS 窗口内的快速降级
        if (globalLastFetchFailed) {
          globalLastFailedAt = Date.now();
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

  useEffect(() => {
    if (loading) return;

    if (!isValidMapMetadata(metadata)) {
      globalTilesAvailable = false;
      setTilesAvailable(false);
      return;
    }

    if (globalTilesAvailable !== null) {
      setTilesAvailable(globalTilesAvailable);
      return;
    }

    if (!globalTileCheckPromise) {
      globalTileCheckPromise = checkTileAvailability(metadata).then((available) => {
        globalTilesAvailable = available;
        return available;
      }).finally(() => {
        globalTileCheckPromise = null;
      });
    }

    globalTileCheckPromise.then(setTilesAvailable);
  }, [metadata, loading]);

  return {
    metadata,
    loading,
    error,
    status,
    isUsingDefault,
    tilesAvailable,
  };
}
