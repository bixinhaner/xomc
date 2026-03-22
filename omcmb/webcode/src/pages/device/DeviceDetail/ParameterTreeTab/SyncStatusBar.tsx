import React from 'react';
import { Alert, Progress, Space, Typography } from 'antd';
import { SyncOutlined, CheckCircleOutlined, CloseCircleOutlined } from '@ant-design/icons';
import type { ParameterSyncStatus } from '@/types/deviceParameter';

const { Text } = Typography;

interface SyncStatusBarProps {
  syncStatus: ParameterSyncStatus | undefined;
}

export default function SyncStatusBar({ syncStatus }: SyncStatusBarProps) {
  if (!syncStatus || syncStatus.status === 'idle') return null;

  const statusConfig = {
    syncing: {
      type: 'info' as const,
      icon: <SyncOutlined spin />,
      message: '参数同步中',
    },
    completed: {
      type: 'success' as const,
      icon: <CheckCircleOutlined />,
      message: '参数同步完成',
    },
    failed: {
      type: 'error' as const,
      icon: <CloseCircleOutlined />,
      message: '参数同步失败',
    },
  };

  const config = statusConfig[syncStatus.status as keyof typeof statusConfig];
  if (!config) return null;

  return (
    <Alert
      type={config.type}
      icon={config.icon}
      showIcon
      style={{ marginBottom: 16 }}
      message={
        <Space direction="vertical" style={{ width: '100%' }} size={4}>
          <Space>
            <Text strong>{config.message}</Text>
            <Text type="secondary">
              {syncStatus.completedBatches}/{syncStatus.totalBatches} 批次
              {' | '}
              {syncStatus.syncedParameters}/{syncStatus.totalParameters} 参数
            </Text>
          </Space>
          {syncStatus.status === 'syncing' && (
            <Progress
              percent={syncStatus.percentage}
              size="small"
              status="active"
              strokeColor="#1677ff"
            />
          )}
          {syncStatus.status === 'completed' && (
            <Progress
              percent={100}
              size="small"
              status="success"
            />
          )}
          {syncStatus.status === 'failed' && syncStatus.error && (
            <Text type="danger">{syncStatus.error}</Text>
          )}
        </Space>
      }
    />
  );
}
