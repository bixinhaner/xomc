import React, { useCallback, useMemo, useState } from 'react';
import { App, Button, Card, Tag } from 'antd';
import {
  DeleteOutlined,
  ExportOutlined,
  ImportOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn, BatchAction } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';
import { useRecycleBinList, useRestoreDevices, usePermanentDeleteDevices } from '@core/hooks/api/useDevices';
import { useDomainTree } from '@core/hooks/api/useTopology';
import type { Device } from '@core/types/device';
import ImportModal from './ImportModal';

// 设备类型
type DeviceType = 'eNB' | 'gNB' | 'CPE';

// 设备类型颜色映射
const DEVICE_TYPE_COLOR: Record<DeviceType, string> = {
  eNB: 'blue',
  gNB: 'green',
  CPE: 'orange',
};

// 根据 product_class 推断设备类型
function inferDeviceType(productClass: string): DeviceType {
  const pc = productClass.toLowerCase();
  if (pc.includes('gnb') || pc.includes('5g')) return 'gNB';
  if (pc.includes('enb') || pc.includes('lte')) return 'eNB';
  return 'CPE';
}

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
  const { modal, message } = App.useApp();
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [importModalOpen, setImportModalOpen] = useState(false);

  // 获取设备分组树（使用树形结构避免重复数据）
  const { data: domains } = useDomainTree();

  // 构建设备分组选项（只显示L2分组，带完整路径）
  const deviceGroupOptions: { id: string; name: string; fullName: string }[] = useMemo(() => {
    // 使用 fullName 作为去重键，确保相同路径只出现一次
    const optionsMap = new Map<string, { id: string; name: string; fullName: string }>();
    const idSet = new Set<string>();

    if (!domains || !Array.isArray(domains)) return [];

    // 辅助函数：安全获取 level 值（处理字符串和数字类型）
    const getLevel = (level: unknown): number => {
      if (typeof level === 'number') return level;
      if (typeof level === 'string') return parseInt(level, 10) || 0;
      return 0;
    };

    // 遍历分组树，只添加 L2 分组（带父级路径）
    const buildOptions = (items: unknown[], parentPath: string = '') => {
      if (!Array.isArray(items)) return;

      items.forEach((item) => {
        if (!item || typeof item !== 'object') return;

        const group = item as { id?: string; name?: string; level?: unknown; children?: unknown[] };
        const level = getLevel(group.level);
        const id = String(group.id || '');
        const name = String(group.name || '');

        if (level === 1) {
          // L1 分组：不添加到选项，只遍历其子分组
          if (group.children && Array.isArray(group.children) && group.children.length > 0) {
            buildOptions(group.children, name);
          }
        } else if (level === 2) {
          // L2 分组：添加到选项，显示完整路径（一级分组/二级分组）
          const fullName = parentPath ? `${parentPath}/${name}` : name;

          // 双重去重：先检查 id，再检查 fullName
          if (id && !idSet.has(id) && !optionsMap.has(fullName)) {
            idSet.add(id);
            optionsMap.set(fullName, {
              id: id,
              name: name,
              fullName: fullName,
            });
          }
        }
      });
    };

    buildOptions(domains as unknown[]);
    // 按 fullName 排序
    return Array.from(optionsMap.values()).sort((a, b) => a.fullName.localeCompare(b.fullName, 'zh-CN'));
  }, [domains]);

  // 获取回收站设备列表
  const { data, isLoading, refetch } = useRecycleBinList({
    search: filterParams.searchText as string,
    group_id: filterParams.group_id as string,
    deleted_by: filterParams.deleted_by as string,
    page: currentPage,
    pageSize,
  });

  // 恢复设备
  const restoreMutation = useRestoreDevices();

  // 永久删除
  const permanentDeleteMutation = usePermanentDeleteDevices();

  // 转换数据格式以适配表格
  const tableData = useMemo(() => {
    if (!data?.items) return [];
    return data.items.map((device: Device) => ({
      ...device,
      deviceType: inferDeviceType(device.productClass),
      offlineDays: calcOfflineDays(device.lastOnlineTime, device.deletedAt || ''),
      moveTime: device.deletedAt || '',
      move_author: device.deletedBy || 'system',
    }));
  }, [data]);

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
            await restoreMutation.mutateAsync(ids.map(String));
            message.success(t('status.success'));
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
  const columns: DataTableColumn<Device & { deviceType: DeviceType; offlineDays: number; moveTime: string; move_author: string }>[] = useMemo(
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
        key: 'deviceType',
        title: t('device.radioMode'),
        dataIndex: 'deviceType',
        width: 100,
        render: (v) => (
          <Tag color={DEVICE_TYPE_COLOR[v as DeviceType] || 'default'}>{String(v)}</Tag>
        ),
      },
      { key: 'host_name', title: t('device.hostName'), dataIndex: 'hostName', width: 140, ellipsis: true },
      {
        key: 'mac',
        title: t('device.macAddress'),
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
        width: 90,
        render: () => <Tag color="blue">{getMoveTypeLabel(t)}</Tag>,
      },
      { key: 'moveTime', title: t('recycle.moveTime'), dataIndex: 'moveTime', width: 160 },
      { key: 'move_author', title: t('recycle.account'), dataIndex: 'move_author', width: 90 },
    ],
    [t]
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
        <DataTable<Device & { deviceType: DeviceType; offlineDays: number; moveTime: string; move_author: string }>
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
          defaultDensity="default"
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
