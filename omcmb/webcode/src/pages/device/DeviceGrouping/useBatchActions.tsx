import React, { useMemo } from 'react';
import { DeleteOutlined, FolderOutlined as MoveToGroupIcon, RestOutlined } from '@ant-design/icons';
import type { App as AppNS } from 'antd';
import type { BatchAction } from '@/components/DataTable';
import type { MutationLike } from './useGroupActions';

/**
 * 设备分组列表的批量操作（移动 / 回收 / 删除）。
 * 拆分自 DeviceGrouping/index.tsx（W2.C.2 / T-0054）。
 */
export function useBatchActions(deps: {
  modal: ReturnType<typeof AppNS.useApp>['modal'];
  message: ReturnType<typeof AppNS.useApp>['message'];
  t: (id: string, values?: Record<string, string | number>) => string;
  refetch: () => Promise<unknown>;
  deleteDevicesMutation: MutationLike<string[]>;
  setSelectedDeviceIds: React.Dispatch<React.SetStateAction<React.Key[]>>;
  onMoveToGroup: (selectedKeys: React.Key[]) => void;
}): BatchAction[] {
  const { modal, message, t, refetch, deleteDevicesMutation, setSelectedDeviceIds, onMoveToGroup } = deps;

  return useMemo<BatchAction[]>(
    () => [
      {
        key: 'moveToGroup',
        label: t('device.batch.moveToGroup'),
        icon: <MoveToGroupIcon />,
        onClick: onMoveToGroup,
      },
      {
        key: 'recycle',
        label: t('device.batch.recycle'),
        icon: <RestOutlined />,
        onClick: (selectedKeys: React.Key[]) => {
          modal.confirm({
            title: t('device.batch.recycleConfirm'),
            width: 480,
            content: (
              <div>
                <div>{t('device.batch.recycleMsg', { count: selectedKeys.length })}</div>
                <div style={{ marginTop: 8, color: 'var(--color-text-secondary)', fontSize: 13, whiteSpace: 'nowrap' }}>
                  {t('device.batch.deleteWarning')}
                </div>
              </div>
            ),
            okText: t('common.confirm'),
            okType: 'danger',
            onOk: async () => {
              try {
                await deleteDevicesMutation.mutateAsync(selectedKeys as string[]);
                void message.success(t('common.success'));
              } catch {
                void message.error(t('common.operationFailed'));
              }
              setSelectedDeviceIds([]);
              await refetch();
            },
          });
        },
      },
      {
        key: 'delete',
        label: t('common.delete'),
        icon: <DeleteOutlined />,
        danger: true,
        onClick: (selectedKeys: React.Key[]) => {
          modal.confirm({
            title: t('common.confirmDelete'),
            width: 480,
            content: (
              <div>
                <div>{t('common.deleteConfirmMsg', { count: selectedKeys.length })}</div>
                <div style={{ marginTop: 8, color: 'var(--color-text-secondary)', fontSize: 13, whiteSpace: 'nowrap' }}>
                  {t('device.batch.deleteWarning')}
                </div>
              </div>
            ),
            okText: t('common.confirmDelete'),
            okType: 'danger',
            onOk: async () => {
              try {
                await deleteDevicesMutation.mutateAsync(selectedKeys as string[]);
                void message.success(t('common.deleteSuccess'));
              } catch {
                void message.error(t('common.operationFailed'));
              }
              setSelectedDeviceIds([]);
              await refetch();
            },
          });
        },
      },
    ],
    [t, modal, message, refetch, deleteDevicesMutation, setSelectedDeviceIds, onMoveToGroup]
  );
}
