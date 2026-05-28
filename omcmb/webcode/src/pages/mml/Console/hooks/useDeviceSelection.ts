import { useState, useCallback, useMemo, useEffect, useRef } from 'react';
import type { ConsoleDevice } from '../types';
import { DEVICE_PAGE_SIZE } from '../constants';
import { deviceApi } from '@core/services/api/deviceApi';

// 2026-05-28 用户决策:产品类型选择 localStorage 持久化,刷新页面后默认选用户上次的选择。
// key 命名沿用 mml-console 业务域前缀,与 useMmlConsoleTerminalStore 等其它 storage 区分。
const PRODUCT_CLASS_FILTER_STORAGE_KEY = 'mml-console.productClassFilter';

function readPersistedProductClass(): string {
  try {
    return localStorage.getItem(PRODUCT_CLASS_FILTER_STORAGE_KEY) || '';
  } catch {
    return '';
  }
}

function writePersistedProductClass(value: string): void {
  try {
    if (value) localStorage.setItem(PRODUCT_CLASS_FILTER_STORAGE_KEY, value);
    else localStorage.removeItem(PRODUCT_CLASS_FILTER_STORAGE_KEY);
  } catch {
    // 隐私模式 / 配额满 — 静默忽略,不影响功能
  }
}

export function useDeviceSelection() {
  const [selectedDevices, setSelectedDevices] = useState<ConsoleDevice[]>([]);
  const [searchText, setSearchText] = useState('');
  // useState lazy initializer 读 localStorage;DeviceTree useEffect 仍兜底:
  // 字典加载后若此值不在 options 里(如被删除),会被覆盖为 options[0],
  // 同时 handleFilterChange 重新写入 localStorage 保持一致。
  const [productClassFilter, setProductClassFilter] = useState<string>(
    () => readPersistedProductClass(),
  );
  const [currentPage, setCurrentPage] = useState(1);
  const [devices, setDevices] = useState<ConsoleDevice[]>([]);
  const [total, setTotal] = useState(0);
  const [isLoadingDevices, setIsLoadingDevices] = useState(false);
  const _abortRef = useRef<AbortController | null>(null);
  void _abortRef;

  // Fetch devices from API — server-side pagination + filter
  const fetchDevices = useCallback(async (page: number, search?: string, productClass?: string) => {
    setIsLoadingDevices(true);
    try {
      const result = await deviceApi.getList({
        page,
        pageSize: DEVICE_PAGE_SIZE,
        ...(search ? { searchText: search } : {}),
        ...(productClass ? { productClass } : {}),
      });
      const mapped: ConsoleDevice[] = result.items.map((d) => ({
        sn: d.sn,
        name: d.deviceName || d.sn,
        type: d.networkType === 'nr' ? 'gNB' : 'eNB',
        productClass: d.productClass || '',
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

  // R-8.1: 首次加载延后到产品类型默认值就绪（由组件传入 setProductClassFilter 触发）。
  // R-8.2: 不再用空 productClassFilter 首拉，避免回到"未筛选 = 跨类型"语义。
  useEffect(() => {
    if (productClassFilter) {
      void fetchDevices(1, undefined, productClassFilter);
    }
    // 依赖 productClassFilter：首次默认值就绪即触发一次加载
  }, [productClassFilter, fetchDevices]);

  // R-8.3: 产品类型切换时除重置分页外，**清空已选设备**（不允许跨类型累积）。
  // 2026-05-28 同步把选择写入 localStorage,刷新页面后默认恢复。
  const handleFilterChange = useCallback((filter: string) => {
    setProductClassFilter(filter);
    writePersistedProductClass(filter);
    setCurrentPage(1);
    setSelectedDevices([]); // R-8.3 清空已选
    if (filter) {
      void fetchDevices(1, searchText || undefined, filter);
    }
  }, [fetchDevices, searchText]);

  // 搜索文本变化时重置到第 1 页并请求（防抖在组件层处理）
  const handleSearchChange = useCallback((text: string) => {
    setSearchText(text);
    setCurrentPage(1);
    void fetchDevices(1, text || undefined, productClassFilter || undefined);
  }, [fetchDevices, productClassFilter]);

  // 翻页时请求真实数据
  const handlePageChange = useCallback((page: number) => {
    setCurrentPage(page);
    void fetchDevices(page, searchText || undefined, productClassFilter || undefined);
  }, [fetchDevices, searchText, productClassFilter]);

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
    productClassFilter,
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
    setProductClassFilter: handleFilterChange,
    setCurrentPage: handlePageChange,
    toggleDevice,
    toggleSelectAll,
    removeDevice,
    clearSelection,
    addDevicesBySns,
    fetchDevices,
  };
}
