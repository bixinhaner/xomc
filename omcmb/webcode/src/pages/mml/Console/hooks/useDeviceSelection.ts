import { useState, useCallback, useMemo, useEffect, useRef } from 'react';
import type { ConsoleDevice } from '../types';
import { DEVICE_PAGE_SIZE } from '../constants';
import { deviceApi } from '@core/services/api/deviceApi';

export function useDeviceSelection() {
  const [selectedDevices, setSelectedDevices] = useState<ConsoleDevice[]>([]);
  const [searchText, setSearchText] = useState('');
  const [productTypeFilter, setProductTypeFilter] = useState<string>('');
  const [currentPage, setCurrentPage] = useState(1);
  const [devices, setDevices] = useState<ConsoleDevice[]>([]);
  const [total, setTotal] = useState(0);
  const [isLoadingDevices, setIsLoadingDevices] = useState(false);
  const _abortRef = useRef<AbortController | null>(null);
  void _abortRef;

  // Fetch devices from API — server-side pagination + filter
  const fetchDevices = useCallback(async (page: number, search?: string, productType?: string) => {
    setIsLoadingDevices(true);
    try {
      const result = await deviceApi.getList({
        page,
        pageSize: DEVICE_PAGE_SIZE,
        ...(search ? { searchText: search } : {}),
        ...(productType ? { productType } : {}),
      });
      const mapped: ConsoleDevice[] = result.items.map((d) => ({
        sn: d.sn,
        name: d.siteName || d.sn,
        type: d.networkType === 'nr' ? 'gNB' : 'eNB',
        productType: d.productType || '',
        status: (d.connStatus === 'online' ? 'online' : 'offline') as ConsoleDevice['status'],
      }));
      // 后端分页边界或缓存异常偶发会带回同 SN 重复项；此处按 SN 排重，
      // 避免 Checkbox/Tag 渲染两条相同设备（to-do-list #6）。
      const seen = new Set<string>();
      const deduped = mapped.filter((d) => {
        if (seen.has(d.sn)) return false;
        seen.add(d.sn);
        return true;
      });
      setDevices(deduped);
      setTotal(result.total);
    } catch {
      setDevices([]);
      setTotal(0);
    } finally {
      setIsLoadingDevices(false);
    }
  }, []);

  // 首次加载
  useEffect(() => {
    void fetchDevices(1, undefined, productTypeFilter || undefined);
  }, []);

  // 产品类型切换时重置到第 1 页并请求
  const handleFilterChange = useCallback((filter: string) => {
    setProductTypeFilter(filter);
    setCurrentPage(1);
    void fetchDevices(1, searchText || undefined, filter || undefined);
  }, [fetchDevices, searchText]);

  // 搜索文本变化时重置到第 1 页并请求（防抖在组件层处理）
  const handleSearchChange = useCallback((text: string) => {
    setSearchText(text);
    setCurrentPage(1);
    void fetchDevices(1, text || undefined, productTypeFilter || undefined);
  }, [fetchDevices, productTypeFilter]);

  // 翻页时请求真实数据
  const handlePageChange = useCallback((page: number) => {
    setCurrentPage(page);
    void fetchDevices(page, searchText || undefined, productTypeFilter || undefined);
  }, [fetchDevices, searchText, productTypeFilter]);

  // 全选状态计算（基于当前页）
  const deviceSns = useMemo(() => new Set(devices.map((d) => d.sn)), [devices]);
  const selectedInPage = useMemo(
    () => selectedDevices.filter((d) => deviceSns.has(d.sn)).length,
    [selectedDevices, deviceSns],
  );
  const isAllSelected = devices.length > 0 && selectedInPage === devices.length;
  const isIndeterminate = selectedInPage > 0 && selectedInPage < devices.length;

  const totalPages = Math.ceil(total / DEVICE_PAGE_SIZE);

  // 选择/取消选择单个设备
  const toggleDevice = useCallback((device: ConsoleDevice, selected: boolean) => {
    setSelectedDevices((prev) => {
      if (selected) {
        if (prev.some((d) => d.sn === device.sn)) return prev;
        return [...prev, device];
      }
      return prev.filter((d) => d.sn !== device.sn);
    });
  }, []);

  // 全选/取消全选当前页
  const toggleSelectAll = useCallback((checked: boolean) => {
    setSelectedDevices((prev) => {
      if (checked) {
        const devicesToAdd = devices.filter((d) => !prev.some((p) => p.sn === d.sn));
        return [...prev, ...devicesToAdd];
      }
      const pageSns = new Set(devices.map((d) => d.sn));
      return prev.filter((d) => !pageSns.has(d.sn));
    });
  }, [devices]);

  // 从已选列表移除设备
  const removeDevice = useCallback((sn: string) => {
    setSelectedDevices((prev) => prev.filter((d) => d.sn !== sn));
  }, []);

  // 清空选择
  const clearSelection = useCallback(() => {
    setSelectedDevices([]);
  }, []);

  // 批量添加设备（通过SN列表）
  const addDevicesBySns = useCallback((sns: string[]) => {
    setSelectedDevices((prev) => {
      const existingSns = new Set(prev.map((d) => d.sn));
      const devicesToAdd = sns
        .map((sn) => devices.find((d) => d.sn === sn))
        .filter((d): d is ConsoleDevice => d !== undefined && !existingSns.has(d.sn));
      return [...prev, ...devicesToAdd];
    });
  }, [devices]);

  // 所有设备SN的Set（用于批量输入验证 — 基于当前页）
  const allDeviceSns = useMemo(
    () => new Set(devices.map((d) => d.sn)),
    [devices],
  );

  return {
    // 状态
    selectedDevices,
    searchText,
    productTypeFilter,
    currentPage,
    filteredDevices: devices,
    paginatedDevices: devices,
    isAllSelected,
    isIndeterminate,
    selectedInFiltered: selectedInPage,
    totalFiltered: total,
    totalPages,
    allDeviceSns,
    isLoadingDevices,

    // 操作
    setSearchText: handleSearchChange,
    setProductTypeFilter: handleFilterChange,
    setCurrentPage: handlePageChange,
    toggleDevice,
    toggleSelectAll,
    removeDevice,
    clearSelection,
    addDevicesBySns,
    fetchDevices,
  };
}
