import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { App, Button, Card, Space, Tag } from 'antd';
import {
  DeleteOutlined,
  ExportOutlined,
  ImportOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import AutoRefreshDropdown from '@/pages/alarm/components/AutoRefreshDropdown';
import { useT } from '@/hooks/useT';
import { useRecycleBinList, useRestoreDevices, usePermanentDeleteDevices, useDeviceGroups } from '@core/hooks/api/useDevices';
import { useDictionaryBatch } from '@core/hooks/api/useSystem';
import { useAppStore } from '@core/store/appStore';
import { withDeviceGroupDisplayName } from '@core/utils/deviceGroupDisplay';
import { resolveNetworkTypeLabel } from '@core/utils/networkType';
import { getI18nText } from '@core/utils/i18nText';
import type { Device } from '@core/types/device';
import ImportModal from './ImportModal';

// 基站制式 Tag 颜色映射（与设备列表一致，按 networkType 原始值取色）
const NETWORK_TYPE_COLOR: Record<string, string> = {
  eNB: 'blue',
  gNB: 'green',
  GSM: 'orange',
};

// 计算离线天数
function calcOfflineDays(lastInformTime: string, deletedAt: string): number {
  const refTime = deletedAt || lastInformTime;
  if (!refTime) return 0;
  const diff = Date.now() - new Date(refTime).getTime();
  return Math.floor(diff / (1000 * 60 * 60 * 24));
}

// 回收方式映射（暂时全部为手动）
const getMoveTypeLabel = (t: (key: string) => string) => {
  return t('recycle.manual');
};

export default function RecycleBin() {
  const t = useT();
  const appLocale = useAppStore((s) => s.locale);
  const { modal, message } = App.useApp();
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [refreshInterval, setRefreshInterval] = useState(30);
  const [importModalOpen, setImportModalOpen] = useState(false);

  const { data: groupsResp } = useDeviceGroups();

  // issue #223: 基站制式列与设备列表 / 筛选下拉同源——走 network_type 字典
  // value→label 映射，不再用 product_class 启发式推导。
  const { data: batchDicts } = useDictionaryBatch(['network_type']);
  const networkTypeDetails = batchDicts?.['network_type']?.sysDictionaryDetails;

  // 构建设备分组选项（只显示L2分组，带完整路径）
  const deviceGroupOptions: { id: string; name: string; fullName: string }[] = useMemo(() => {
    const groups = groupsResp?.groups ?? [];
    const byId = new Map(groups.map((g) => [g.id, g]));

    return groups
      .filter((g) => g.parentId !== null)
      .map((group) => {
        const parent = group.parentId ? byId.get(group.parentId) : undefined;
        const name = getI18nText(group.nameI18n, appLocale, group.name);
        const parentName = parent ? getI18nText(parent.nameI18n, appLocale, parent.name) : '';
        return {
          id: group.id,
          name,
          fullName: parentName ? `${parentName}/${name}` : name,
        };
      })
      .sort((a, b) => a.fullName.localeCompare(b.fullName, appLocale === 'zh-CN' ? 'zh-CN' : 'en-US'));
  }, [groupsResp?.groups, appLocale]);

  // 获取回收站设备列表
  const { data, isLoading, isFetching, refetch } = useRecycleBinList(
    {
      search: filterParams.searchText as string,
      group_id: filterParams.group_id as string,
      deleted_by: filterParams.deleted_by as string,
      page: currentPage,
      pageSize,
    }
  );

  // 恢复设备
  const restoreMutation = useRestoreDevices();

  // 永久删除
  const permanentDeleteMutation = usePermanentDeleteDevices();

  const handleManualRefresh = useCallback(() => {
    void refetch();
  }, [refetch]);

  useEffect(() => {
    if (!autoRefresh) return;

    void refetch();
    const timer = window.setInterval(() => {
      void refetch();
    }, refreshInterval * 1000);

    return () => window.clearInterval(timer);
  }, [autoRefresh, refreshInterval, refetch]);

  // 转换数据格式以适配表格
  const tableData = useMemo(() => {
    if (!data?.items) return [];
    const devices = withDeviceGroupDisplayName(
      data.items as Device[],
      groupsResp?.groups ?? [],
      appLocale,
    );
    return devices.map((device: Device) => ({
      ...device,
      offlineDays: calcOfflineDays(device.lastOnlineTime, device.deletedAt || ''),
      moveTime: device.deletedAt || '',
      move_author: device.deletedBy || 'system',
    }));
  }, [data, groupsResp?.groups, appLocale]);

  // 移出回收站（带确认）
  const handleRestore = useCallback(
    (ids: React.Key[]) => {
      modal.confirm({
        title: t('common.confirm'),
        content: t('recycle.restoreConfirm', { count: ids.length }),
        okText: t('common.confirm'),
        cancelText: t('common.cancel'),
        icon: <ExportOutlined style={{ color: '#52C41A' }} />,
        onOk: async () => {
          try {
            // #378: 恢复可部分成功——SN 冲突设备被后端跳过并回传 conflicts。
            const res = await restoreMutation.mutateAsync(ids.map(String));
            if (res.skipped > 0) {
              const conflictSNs = res.conflicts.map((c) => c.serialNumber).join('、');
              message.warning(
                t('recycle.restorePartial', {
                  restored: res.restored,
                  skipped: res.skipped,
                  sns: conflictSNs,
                })
              );
            } else {
              message.success(t('status.success'));
            }
            setSelectedRowKeys([]);
          } catch {
            message.error(t('common.operationFailed'));
          }
        },
      });
    },
    [t, modal, message, restoreMutation]
  );

  // 批量删除（带确认）
  const handlePermanentDelete = useCallback(
    (ids: React.Key[]) => {
      modal.confirm({
        title: t('common.confirmDelete'),
        content: t('recycle.deleteConfirm', { count: ids.length }),
        okText: t('common.confirm'),
        cancelText: t('common.cancel'),
        okType: 'danger',
        onOk: async () => {
          try {
            await permanentDeleteMutation.mutateAsync(ids.map(String));
            message.success(t('common.deleteSuccess'));
            setSelectedRowKeys([]);
          } catch {
            message.error(t('common.deleteFailed'));
          }
        },
      });
    },
    [t, modal, message, permanentDeleteMutation]
  );

  // 打开导入弹窗
  const handleOpenImportModal = useCallback(() => {
    setImportModalOpen(true);
  }, []);

  // 导入完成
  const handleImportComplete = useCallback(() => {
    void refetch();
    message.success(t('common.success'));
  }, [refetch, message, t]);

  // 筛选字段配置
  const FILTER_FIELDS: FilterField[] = useMemo(
    () => [
      {
        name: 'searchText',
        label: t('common.search'),
        type: 'input',
        placeholder: t('recycle.searchPlaceholder'),
        width: 240,
      },
      {
        name: 'group_id',
        label: t('recycle.groupName'),
        type: 'select',
        options: deviceGroupOptions.map((g) => ({
          label: g.fullName,
          value: g.id,
        })),
        placeholder: t('common.pleaseSelect'),
        width: 220,
      },
    ],
    [t, deviceGroupOptions]
  );

  // 列定义
  const columns: DataTableColumn<Device & { offlineDays: number; moveTime: string; move_author: string }>[] = useMemo(
    () => [
      {
        key: 'serial_number',
        title: t('device.serialNumber'),
        dataIndex: 'sn',
        width: 140,
        mono: true,
        copyable: true,
      },
      {
        key: 'networkType',
        title: t('device.radioMode'),
        dataIndex: 'networkType',
        width: 100,
        render: (_v, record) => {
          const label = resolveNetworkTypeLabel(record.networkType, networkTypeDetails, appLocale);
          return (
            <Tag color={NETWORK_TYPE_COLOR[record.networkType] || 'default'}>{label}</Tag>
          );
        },
      },
      {
        key: 'host_name',
        title: t('device.hostName'),
        dataIndex: 'deviceName',
        width: 140,
        ellipsis: true,
        render: (_v, record) => record.deviceName || '-',
      },
      {
        key: 'mac',
        title: t('device.macAddress'),
        dataIndex: 'macAddress',
        width: 130,
        mono: true,
        render: (_v, record) => record.macAddress || '-',
      },
      { key: 'longitude', title: t('device.longitude'), dataIndex: 'longitude', width: 90 },
      { key: 'latitude', title: t('device.latitude'), dataIndex: 'latitude', width: 90 },
      { key: 'height', title: t('recycle.height'), dataIndex: 'gpsHeight', width: 70 },
      { key: 'offlineDays', title: t('recycle.offlineDays'), dataIndex: 'offlineDays', width: 90 },
      { key: 'group_name', title: t('recycle.groupName'), dataIndex: 'groupName', width: 120 },
      {
        key: 'moveType',
        title: t('recycle.moveType'),
        dataIndex: 'moveType',
        width: 90,
        render: () => <Tag color="blue">{getMoveTypeLabel(t)}</Tag>,
      },
      { key: 'moveTime', title: t('recycle.moveTime'), dataIndex: 'moveTime', width: 160 },
      { key: 'move_author', title: t('recycle.account'), dataIndex: 'move_author', width: 90 },
    ],
    [t, networkTypeDetails, appLocale]
  );

  // 批量操作
  const batchActions: BatchAction[] = useMemo(
    () => [
      {
        key: 'batch-restore',
        label: t('recycle.restore'),
        icon: <ExportOutlined />,
        onClick: handleRestore,
      },
      {
        key: 'batch-delete',
        label: t('common.batchDelete'),
        icon: <DeleteOutlined />,
        danger: true,
        onClick: handlePermanentDelete,
      },
    ],
    [handleRestore, handlePermanentDelete, t]
  );

  return (
    <ListPageLayout>
      {/* 2026-06-03 用户决策:去掉"回收站"标题;筛选条件与「导入」同一行,
          筛选靠左(字段固定宽度)、导入按钮两端对齐推到页面最右(flex space-between)。 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 12 }}>
        <div style={{ flex: 1, minWidth: 0 }}>
          <FilterBar
            filterId="recycle-bin"
            fields={FILTER_FIELDS}
            onSearch={(v) => {
              setFilterParams(v);
              setCurrentPage(1);
            }}
            onReset={() => {
              setFilterParams({});
              setCurrentPage(1);
            }}
            collapsedRows={1}
          />
        </div>
        <Button type="primary" icon={<ImportOutlined />} onClick={handleOpenImportModal} style={{ flexShrink: 0 }}>
          {t('common.import')}
        </Button>
      </div>

      <Card
        size="small"
        variant="outlined"
        style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
        styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
      >
        <DataTable<Device & { offlineDays: number; moveTime: string; move_author: string }>
          tableId="recycle-bin-table"
          columns={columns}
          dataSource={tableData}
          loading={isLoading || restoreMutation.isPending || permanentDeleteMutation.isPending}
          rowKey="id"
          selectable
          selectedRowKeys={selectedRowKeys}
          onSelectionChange={(keys) => setSelectedRowKeys(keys)}
          total={data?.total || 0}
          pageSize={pageSize}
          currentPage={currentPage}
          onPageChange={(p, s) => {
            setCurrentPage(p);
            if (s !== pageSize) setPageSize(s);
          }}
          batchActions={batchActions}
          onRefresh={handleManualRefresh}
          extraToolbarRight={(
            <Space size={8}>
              <Button size="small" icon={<ReloadOutlined />} loading={isFetching} onClick={handleManualRefresh}>
                {t('common.refresh')}
              </Button>
              <AutoRefreshDropdown
                enabled={autoRefresh}
                intervalSeconds={refreshInterval}
                onEnabledChange={setAutoRefresh}
                onIntervalChange={setRefreshInterval}
                spinning={autoRefresh && isFetching}
                size="small"
              />
            </Space>
          )}
          defaultDensity="default"
          hideRealtime
          showRowNumber
          rowNumberTitle={t('table.rowNumber')}
          scroll={{ x: 'max-content', y: 'calc(100vh - 350px)' }}
        />
      </Card>

      <ImportModal
        open={importModalOpen}
        onClose={() => setImportModalOpen(false)}
        onConfirm={handleImportComplete}
      />
    </ListPageLayout>
  );
}
