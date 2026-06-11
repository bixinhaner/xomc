import { useCallback, useEffect, useState } from 'react';
import type { Key } from 'react';
import type { MessageInstance } from 'antd/es/message/interface';
import type { Alarm, AlarmFilter } from '@core/types/alarm';
import type { AlarmExportFieldKey } from '../utils/alarmExportFields';

interface AlarmScopedExportParams {
  deviceGroupIds: string[];
  timeRange?: [string, string];
  fieldKeys: AlarmExportFieldKey[];
}

interface UseAlarmListExportOptions {
  exportMessageKey: string;
  filterParams: AlarmFilter;
  selectedRowKeys: Key[];
  setSelectedRowKeys: (keys: Key[]) => void;
  fetchAllAlarmsForExport: (filters: AlarmFilter) => Promise<Alarm[]>;
  fetchDeviceSnsByGroups: (groupIds: string[]) => Promise<Set<string>>;
  downloadAlarmCsv: (items: Alarm[], fieldKeys: AlarmExportFieldKey[]) => void;
  message: MessageInstance;
  t: (id: string, values?: Record<string, unknown>) => string;
}

export function useAlarmListExport({
  exportMessageKey,
  filterParams,
  selectedRowKeys,
  setSelectedRowKeys,
  fetchAllAlarmsForExport,
  fetchDeviceSnsByGroups,
  downloadAlarmCsv,
  message,
  t,
}: UseAlarmListExportOptions) {
  const [exportOpen, setExportOpen] = useState(false);
  const [exportLoading, setExportLoading] = useState(false);
  const [selectAllLoading, setSelectAllLoading] = useState(false);

  const clearSelection = useCallback(() => {
    setSelectedRowKeys([]);
  }, [setSelectedRowKeys]);

  useEffect(() => {
    clearSelection();
  }, [filterParams, clearSelection]);

  const handleExportConfirm = useCallback(async (params: AlarmScopedExportParams) => {
    setExportLoading(true);
    message.open({
      key: exportMessageKey,
      type: 'loading',
      content: t('common.exportInProgress'),
      duration: 0,
    });

    try {
      if (params.fieldKeys.length === 0) {
        message.open({
          key: exportMessageKey,
          type: 'warning',
          content: t('export.selectFields'),
        });
        return;
      }

      const effectiveFilter: AlarmFilter = params.timeRange
        ? { ...filterParams, timeRange: params.timeRange }
        : filterParams;

      const alarmsForExport = await fetchAllAlarmsForExport(effectiveFilter);
      let exportItems: Alarm[] = [];

      if (selectedRowKeys.length > 0) {
        const selectedIdSet = new Set(selectedRowKeys.map((key) => String(key)));
        exportItems = alarmsForExport.filter((alarm) => selectedIdSet.has(alarm.id));
      } else {
        if (params.deviceGroupIds.length === 0) {
          message.open({
            key: exportMessageKey,
            type: 'warning',
            content: t('export.selectDeviceGroup'),
          });
          return;
        }

        const allowedDeviceSns = await fetchDeviceSnsByGroups(params.deviceGroupIds);
        if (allowedDeviceSns.size === 0) {
          message.open({
            key: exportMessageKey,
            type: 'warning',
            content: t('common.noDataToExport'),
          });
          return;
        }

        exportItems = alarmsForExport.filter((alarm) => allowedDeviceSns.has(alarm.deviceSn));
      }

      if (exportItems.length === 0) {
        message.open({
          key: exportMessageKey,
          type: 'warning',
          content: t('common.noDataToExport'),
        });
        return;
      }

      downloadAlarmCsv(exportItems, params.fieldKeys);
      message.open({
        key: exportMessageKey,
        type: 'success',
        content: t('common.exportSuccess', { count: exportItems.length }),
      });
      setExportOpen(false);
    } catch {
      message.open({
        key: exportMessageKey,
        type: 'error',
        content: t('common.exportFailed', { reason: t('common.exportUnavailable') }),
      });
    } finally {
      setExportLoading(false);
    }
  }, [
    downloadAlarmCsv,
    exportMessageKey,
    fetchAllAlarmsForExport,
    fetchDeviceSnsByGroups,
    filterParams,
    message,
    selectedRowKeys,
    t,
  ]);

  const handleExportTrigger = useCallback(() => {
    setExportOpen(true);
  }, []);

  const handleSelectAllFiltered = useCallback(async () => {
    const selectAllMessageKey = `${exportMessageKey}-select-all`;
    setSelectAllLoading(true);
    message.open({
      key: selectAllMessageKey,
      type: 'loading',
      content: t('alarm.selectAllFilteredLoading'),
      duration: 0,
    });

    try {
      const alarmsForSelection = await fetchAllAlarmsForExport(filterParams);
      const allIds = Array.from(new Set(alarmsForSelection.map((alarm) => alarm.id)));

      if (allIds.length === 0) {
        message.open({
          key: selectAllMessageKey,
          type: 'warning',
          content: t('common.noDataToExport'),
        });
        return;
      }

      setSelectedRowKeys(allIds);
      message.open({
        key: selectAllMessageKey,
        type: 'success',
        content: t('alarm.selectAllFilteredSuccess', { count: allIds.length }),
      });
    } catch {
      message.open({
        key: selectAllMessageKey,
        type: 'error',
        content: t('alarm.selectAllFilteredFailed'),
      });
    } finally {
      setSelectAllLoading(false);
    }
  }, [exportMessageKey, fetchAllAlarmsForExport, filterParams, message, setSelectedRowKeys, t]);

  return {
    clearSelection,
    exportLoading,
    exportOpen,
    handleExportConfirm,
    handleExportTrigger,
    handleSelectAllFiltered,
    selectAllLoading,
    setExportOpen,
  };
}