import React, { useMemo } from 'react';
import { DeleteOutlined, FolderOutlined as MoveToGroupIcon, RestOutlined } from '@ant-design/icons';
import type { App as AppNS } from 'antd';
import type { BatchAction } from '@/components/DataTable';
import type { MutationLike } from './useGroupActions';
import styles from './DeviceGrouping.module.css';

/**
 * 设备分组列表的批量操作（移动 / 回收 / 删除）。
 * 拆分自 DeviceGrouping/index.tsx（W2.C.2 / T-0054）。
 */
export function useBatchActions(deps: {
  modal: ReturnType<typeof AppNS.useApp>['modal'];
  message: ReturnType<typeof AppNS.useApp>['message'];
  t: (id: string, values?: Record<string, string | number>) => string;
  refetch: () => Promise<unknown>;
  /** 刷新分组树查询（device-groups）—— 回收/删除设备后需同步更新各分组徽标 + 「全部」总数。 */
  refetchGroups: () => Promise<unknown>;
  deleteDevicesMutation: MutationLike<string[]>;
  setSelectedDeviceIds: React.Dispatch<React.SetStateAction<React.Key[]>>;
  onMoveToGroup: (selectedKeys: React.Key[]) => void;
}): BatchAction[] {
  const { modal, message, t, refetch, refetchGroups, deleteDevicesMutation, setSelectedDeviceIds, onMoveToGroup } = deps;

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
            width: 520,
            content: (
              <div className={styles.dangerConfirmContent}>
                <div className={styles.dangerConfirmTitle}>
                  {t('device.batch.recycleMsg', { count: selectedKeys.length })}
                </div>
                <div className={styles.dangerConfirmHint}>
                  {t('common.recycleConfirmDesc')}
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
              await Promise.all([refetch(), refetchGroups()]);
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
            width: 520,
            content: (
              <div className={styles.dangerConfirmContent}>
                <div className={styles.dangerConfirmTitle}>
                  {t('common.deleteConfirmMsg', { count: selectedKeys.length })}
                </div>
                <div className={styles.dangerConfirmHint}>
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
              await Promise.all([refetch(), refetchGroups()]);
            },
          });
        },
      },
    ],
    [t, modal, message, refetch, refetchGroups, deleteDevicesMutation, setSelectedDeviceIds, onMoveToGroup]
  );
}
