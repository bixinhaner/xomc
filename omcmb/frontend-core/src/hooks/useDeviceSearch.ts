import { useState, useCallback, useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import type { DeviceSearchResult } from '../types/map';

/**
 * 设备搜索 Hook 配置选项
 */
export interface UseDeviceSearchOptions {
  /** 最小输入长度，默认 2 */
  minLength?: number;
  /** 最大值数量，默认 50 */
  maxValues?: number;
  /** 防抖延迟（毫秒），0 表示不防抖，默认 300 */
  debounce?: number;
  /** 是否自动展开结果列表，默认 true */
  autoExpand?: boolean;
}

/**
 * 设备搜索 Hook
 *
 * 统一的搜索逻辑，支持：
 * - 逗号分隔多值输入
 * - 最大值数量验证
 * - 防抖处理
 * - 自动展开/收起结果列表
 *
 * @param searchFn 搜索函数
 * @param options 配置选项
 *
 * @example
 * ```tsx
 * const { keyword, handleChange, results, isLoading, expanded, setExpanded } =
 *   useDeviceSearch((kw) => topologyApi.searchDevices(kw), {
 *     minLength: 2,
 *     maxValues: 50,
 *     debounce: 300,
 *   });
 * ```
 */
export function useDeviceSearch(
  searchFn: (keyword: string) => Promise<DeviceSearchResult[]>,
  options: UseDeviceSearchOptions = {}
) {
  const {
    minLength = 2,
    maxValues = 50,
    debounce = 300,
    autoExpand = true,
  } = options;

  const [keyword, setKeyword] = useState('');
  const [debouncedKeyword, setDebouncedKeyword] = useState(keyword);
  const [expanded, setExpanded] = useState(false);

  // 防抖处理
  useEffect(() => {
    if (debounce <= 0) {
      setDebouncedKeyword(keyword);
      return;
    }

    const timer = setTimeout(() => {
      setDebouncedKeyword(keyword);
    }, debounce);

    return () => clearTimeout(timer);
  }, [keyword, debounce]);

  // 输入处理（含验证）
  const handleChange = useCallback(
    (value: string) => {
      // 验证逗号分隔的数量
      if (value.includes(',')) {
        const values = value
          .split(',')
          .map((v) => v.trim())
          .filter(Boolean);

        if (values.length > maxValues) {
          // 通过 message API 显示警告（需要从外部传入或使用全局消息）
          console.warn(`最多支持${maxValues}个搜索值，已截断`);
          value = values.slice(0, maxValues).join(',');
        }
      }

      setKeyword(value);

      // 自动展开/收起
      if (autoExpand) {
        if (value.length >= minLength) {
          setExpanded(true);
        } else {
          setExpanded(false);
        }
      }
    },
    [maxValues, minLength, autoExpand]
  );

  // 搜索请求
  const { data: results, isLoading } = useQuery({
    queryKey: ['device-search', debouncedKeyword],
    queryFn: () => searchFn(debouncedKeyword),
    enabled: debouncedKeyword.length >= minLength,
  });

  // 清空搜索
  const handleClear = useCallback(() => {
    const empty = '';
    setKeyword(empty);
    setDebouncedKeyword(empty);
    if (autoExpand) {
      setExpanded(false);
    }
  }, [autoExpand]);

  return {
    /** 当前输入值（未防抖） */
    keyword,
    /** 防抖后的输入值 */
    debouncedKeyword,
    /** 输入处理函数（含自动展开） */
    handleChange,
    /** 清空输入 */
    handleClear,
    /** 搜索结果 */
    results: results ?? [],
    /** 加载状态 */
    isLoading,
    /** 结果列表是否展开 */
    expanded,
    /** 设置展开状态 */
    setExpanded,
  };
}
