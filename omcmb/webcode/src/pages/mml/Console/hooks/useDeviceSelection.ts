import { useState, useCallback, useMemo } from 'react';
import type { ConsoleDevice } from '../types';
import { DEVICE_LIST, DEVICE_PAGE_SIZE } from '../constants';

export function useDeviceSelection() {
  const [selectedDevices, setSelectedDevices] = useState<ConsoleDevice[]>([]);
  const [searchText, setSearchText] = useState('');
  const [productTypeFilter, setProductTypeFilter] = useState<string>('');
  const [currentPage, setCurrentPage] = useState(1);

  // 过滤后的设备列表
  const filteredDevices = useMemo(() => {
    return DEVICE_LIST.filter((device) => {
      const matchSearch = !searchText ||
        device.sn.toLowerCase().includes(searchText.toLowerCase()) ||
        device.name.includes(searchText);
      const matchType = !productTypeFilter || device.productType === productTypeFilter;
      return matchSearch && matchType;
    });
  }, [searchText, productTypeFilter]);

  // 分页后的设备列表
  const paginatedDevices = useMemo(() => {
    const start = (currentPage - 1) * DEVICE_PAGE_SIZE;
    return filteredDevices.slice(start, start + DEVICE_PAGE_SIZE);
  }, [filteredDevices, currentPage]);

  // 全选状态计算
  const filteredDeviceSns = useMemo(
    () => new Set(filteredDevices.map((d) => d.sn)),
    [filteredDevices]
  );
  const selectedInFiltered = useMemo(
    () => selectedDevices.filter((d) => filteredDeviceSns.has(d.sn)).length,
    [selectedDevices, filteredDeviceSns]
  );
  const isAllSelected = filteredDevices.length > 0 && selectedInFiltered === filteredDevices.length;
  const isIndeterminate = selectedInFiltered > 0 && selectedInFiltered < filteredDevices.length;

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

  // 全选/取消全选当前过滤结果
  const toggleSelectAll = useCallback((checked: boolean) => {
    setSelectedDevices((prev) => {
      if (checked) {
        const devicesToAdd = filteredDevices.filter((d) => !prev.some((p) => p.sn === d.sn));
        return [...prev, ...devicesToAdd];
      }
      const filteredSns = new Set(filteredDevices.map((d) => d.sn));
      return prev.filter((d) => !filteredSns.has(d.sn));
    });
  }, [filteredDevices]);

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
        .map((sn) => DEVICE_LIST.find((d) => d.sn === sn))
        .filter((d): d is ConsoleDevice => d !== undefined && !existingSns.has(d.sn));
      return [...prev, ...devicesToAdd];
    });
  }, []);

  return {
    // 状态
    selectedDevices,
    searchText,
    productTypeFilter,
    currentPage,
    filteredDevices,
    paginatedDevices,
    isAllSelected,
    isIndeterminate,
    selectedInFiltered,
    totalFiltered: filteredDevices.length,
    totalPages: Math.ceil(filteredDevices.length / DEVICE_PAGE_SIZE),

    // 操作
    setSearchText,
    setProductTypeFilter,
    setCurrentPage,
    toggleDevice,
    toggleSelectAll,
    removeDevice,
    clearSelection,
    addDevicesBySns,
  };
}
