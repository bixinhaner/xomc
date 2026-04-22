import React from 'react';
import { Alert, Progress, Space, Typography } from 'antd';
import { SyncOutlined, CheckCircleOutlined, CloseCircleOutlined } from '@ant-design/icons';
import type { ParameterSyncStatus } from '@core/types/deviceParameter';

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

  const hasBatchInfo = syncStatus.totalBatches > 0;
  const hasPercentage = syncStatus.percentage > 0 && syncStatus.percentage < 100;

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
              {hasBatchInfo
                ? `${syncStatus.completedBatches}/${syncStatus.totalBatches} 批次 | `
                : ''}
              {syncStatus.totalParameters > 0
                ? `已同步 ${syncStatus.totalParameters} 参数`
                : '等待设备响应...'}
              {syncStatus.status === 'syncing' && syncStatus.totalBatches > 0 && (
                ` | 待处理 ${syncStatus.totalBatches} 条命令`
              )}
            </Text>
          </Space>
          {syncStatus.status === 'syncing' && (
            hasPercentage ? (
              <Progress
                percent={syncStatus.percentage}
                size="small"
                status="active"
                strokeColor="#1677ff"
              />
            ) : (
              <div style={{
                height: 8,
                borderRadius: 4,
                background: '#f0f0f0',
                overflow: 'hidden',
                position: 'relative',
              }}>
                <div style={{
                  position: 'absolute',
                  height: '100%',
                  width: '30%',
                  background: 'linear-gradient(90deg, #1677ff, #69b1ff)',
                  borderRadius: 4,
                  animation: 'syncIndeterminate 1.5s ease-in-out infinite',
                }} />
                <style>{`
                  @keyframes syncIndeterminate {
                    0% { left: -30%; }
                    100% { left: 100%; }
                  }
                `}</style>
              </div>
            )
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
