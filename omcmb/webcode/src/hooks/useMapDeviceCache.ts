/**
 * 地图设备数据 LRU 缓存 Hook
 *
 * 用于缓存已加载的设备数据，避免重复请求相同区域的数据。
 *
 * 缓存键格式：视口边界字符串 "minLng,maxLng,minLat,maxLat"
 * 缓存值：设备数据数组
 *
 * @module hooks/useMapDeviceCache
 */

import { useCallback, useRef, useState, useEffect } from 'react';
import type { MapDevice } from '@core/types/map';

/**
 * 缓存条目
 */
interface CacheEntry {
  /** 设备数据 */
  data: MapDevice[];
  /** 访问时间戳 */
  timestamp: number;
  /** 访问次数 */
  accessCount: number;
}

/**
 * LRU 缓存配置
 */
interface LRUCacheOptions {
  /** 最大缓存条目数，默认 10 */
  maxSize?: number;
  /** 缓存过期时间（毫秒），默认 5 分钟 */
  ttl?: number;
}

/**
 * 地图设备数据 LRU 缓存 Hook
 *
 * @example
 * ```tsx
 * const { get, set, clear, has } = useMapDeviceCache({ maxSize: 10 });
 *
 * // 获取缓存
 * const devices = get(boundsKey);
 * if (devices) {
 *   // 使用缓存数据
 * } else {
 *   // 请求数据并缓存
 *   const data = await fetchDevices(bounds);
 *   set(boundsKey, data);
 * }
 * ```
 */
export function useMapDeviceCache(options: LRUCacheOptions = {}) {
  const {
    maxSize = 10,
    ttl = 5 * 60 * 1000, // 5 分钟
  } = options;

  // 使用 state 来跟踪缓存大小（必须在 useCallback 之前声明）
  const [cacheSize, setCacheSize] = useState(0);

  // 使用 Map 存储缓存，保持插入顺序
  const cacheRef = useRef<Map<string, CacheEntry>>(new Map());

  /**
   * 生成缓存键
   * 将视口边界转换为字符串作为缓存键
   *
   * @param bounds - 视口边界
   * @returns 缓存键字符串
   */
  const generateKey = useCallback((bounds: string): string => {
    return bounds;
  }, []);

  /**
   * 标准化缓存键（处理浮点数精度问题）
   * 将边界值保留 4 位小数，避免因微小差异导致缓存未命中
   *
   * @param bounds - 视口边界字符串 "minLng,maxLng,minLat,maxLat"
   * @returns 标准化后的缓存键
   */
  const normalizeKey = useCallback((bounds: string): string => {
    const parts = bounds.split(',').map(Number);
    if (parts.length !== 4 || parts.some(isNaN)) {
      // 边界格式无效，返回原始键
      return bounds;
    }
    const [minLng, maxLng, minLat, maxLat] = parts;
    // 使用 toFixed 进行四舍五入，避免精度丢失
    return [
      Number(minLng.toFixed(4)),
      Number(maxLng.toFixed(4)),
      Number(minLat.toFixed(4)),
      Number(maxLat.toFixed(4)),
    ].join(',');
  }, []);

  /**
   * 获取缓存数据
   *
   * @param key - 缓存键
   * @returns 缓存的设备数据，如果不存在或已过期返回 null
   */
  const get = useCallback((key: string): MapDevice[] | null => {
    const normalizedKey = normalizeKey(generateKey(key));
    const entry = cacheRef.current.get(normalizedKey);

    if (!entry) {
      return null;
    }

    // 检查是否过期
    const now = Date.now();
    if (now - entry.timestamp > ttl) {
      cacheRef.current.delete(normalizedKey);
      setCacheSize(cacheRef.current.size); // 更新缓存大小
      return null;
    }

    // 更新访问时间和次数（LRU 策略）
    entry.timestamp = now;
    entry.accessCount += 1;

    return entry.data;
  }, [generateKey, normalizeKey, ttl]);

  /**
   * 设置缓存数据
   *
   * 如果缓存已满，删除最久未使用的条目（LRU 策略）
   *
   * @param key - 缓存键
   * @param data - 设备数据
   */
  const set = useCallback((key: string, data: MapDevice[]): void => {
    const normalizedKey = normalizeKey(generateKey(key));
    const now = Date.now();
    const isUpdate = cacheRef.current.has(normalizedKey);

    // 如果缓存已满且不是更新操作，删除最久未使用的条目
    if (!isUpdate && cacheRef.current.size >= maxSize) {
      let oldestKey: string | null = null;
      let oldestTimestamp = now;

      for (const [k, entry] of cacheRef.current.entries()) {
        if (entry.timestamp < oldestTimestamp) {
          oldestTimestamp = entry.timestamp;
          oldestKey = k;
        }
      }

      if (oldestKey) {
        cacheRef.current.delete(oldestKey);
      }
    }

    // 设置新缓存
    cacheRef.current.set(normalizedKey, {
      data,
      timestamp: now,
      accessCount: 1,
    });

    // 更新缓存大小状态
    setCacheSize(cacheRef.current.size);
  }, [generateKey, normalizeKey, maxSize]);

  /**
   * 检查缓存是否存在
   *
   * @param key - 缓存键
   * @returns 是否存在有效缓存
   */
  const has = useCallback((key: string): boolean => {
    const normalizedKey = normalizeKey(generateKey(key));
    const entry = cacheRef.current.get(normalizedKey);

    if (!entry) {
      return false;
    }

    // 检查是否过期
    const now = Date.now();
    if (now - entry.timestamp > ttl) {
      cacheRef.current.delete(normalizedKey);
      setCacheSize(cacheRef.current.size); // 更新缓存大小
      return false;
    }

    return true;
  }, [generateKey, normalizeKey, ttl]);

  /**
   * 清除指定缓存
   *
   * @param key - 缓存键，如果不提供则清除所有缓存
   */
  const clear = useCallback((key?: string): void => {
    if (key) {
      const normalizedKey = normalizeKey(generateKey(key));
      cacheRef.current.delete(normalizedKey);
    } else {
      cacheRef.current.clear();
    }
    // 更新缓存大小状态
    setCacheSize(cacheRef.current.size);
  }, [generateKey, normalizeKey]);

  /**
   * 获取缓存统计信息
   *
   * @returns 缓存统计数据
   */
  const getStats = useCallback(() => {
    const entries = Array.from(cacheRef.current.entries());
    return {
      size: cacheRef.current.size,
      maxSize,
      entries: entries.map(([key, entry]) => ({
        key,
        dataCount: entry.data.length,
        timestamp: entry.timestamp,
        accessCount: entry.accessCount,
        age: Date.now() - entry.timestamp,
      })),
    };
  }, [maxSize]);

  // 更新缓存大小的 effect
  useEffect(() => {
    setCacheSize(cacheRef.current.size);
  }, []); // 只在组件挂载时执行一次，后续由 set 操作通过其他方式更新

  return {
    get,
    set,
    has,
    clear,
    getStats,
    size: cacheSize,
  };
}
