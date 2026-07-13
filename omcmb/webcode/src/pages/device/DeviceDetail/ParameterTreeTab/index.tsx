import { useCallback, useMemo, useState } from 'react';
import { Alert, Button, Input, Space, Tooltip, Typography, message, theme } from 'antd';
import { SearchOutlined, SyncOutlined, LoadingOutlined } from '@ant-design/icons';
import {
  useObjectTree,
  useDirectChildren,
  useSyncStatus,
  useAddObject,
  useDeleteObject,
} from '@core/hooks/api/useDeviceParameters';
import { useSyncDeviceParams } from '@core/hooks/api/useDevices';
import ObjectTreePanel from './ObjectTreePanel';
import ChildParamTable from './ChildParamTable';
import { useT } from '@/hooks/useT';
import { formatTimeAgo } from '@core/utils/format';

function formatDuration(seconds?: number) {
  if (seconds === undefined || !Number.isFinite(seconds) || seconds < 0) return undefined;
  if (seconds < 10) return `${seconds.toFixed(1)}s`;
  return `${Math.round(seconds)}s`;
}

interface ParameterTreeTabProps {
  deviceId: string;
  lastScopedSync?: { targetCount: number; gpvTaskCount: number; completedAt?: string; wallClockSeconds?: number } | null;
  syncBusy?: boolean;
  onFullSyncStarted?: () => void;
}

export default function ParameterTreeTab({ deviceId, lastScopedSync, syncBusy: externalSyncBusy = false, onFullSyncStarted }: ParameterTreeTabProps) {
  const t = useT();
  const { token } = theme.useToken();
  const [selectedPath, setSelectedPath] = useState<string>('');
  const [searchKeyword, setSearchKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);

  const treePanelStyle = useMemo<React.CSSProperties>(
    () => ({
      width: 320,
      flexShrink: 0,
      border: `1px solid ${token.colorBorderSecondary}`,
      borderRadius: 8,
      overflow: 'hidden',
      maxHeight: 700,
      boxShadow: '0 1px 4px rgba(0,0,0,0.04)',
      background: token.colorBgContainer,
      ['--otp-bg' as string]: token.colorBgContainer,
      ['--otp-bg-soft' as string]: token.colorFillAlter,
      ['--otp-bg-hover' as string]: token.colorBgTextHover,
      ['--otp-bg-selected' as string]: token.controlItemBgActive,
      ['--otp-border' as string]: token.colorBorderSecondary,
      ['--otp-border-soft' as string]: token.colorSplit,
      ['--otp-text' as string]: token.colorText,
      ['--otp-text-secondary' as string]: token.colorTextSecondary,
      ['--otp-text-disabled' as string]: token.colorTextTertiary,
      ['--otp-primary' as string]: token.colorPrimary,
      ['--otp-primary-border' as string]: token.colorPrimaryBorder,
      ['--otp-success' as string]: token.colorSuccess,
      ['--otp-success-bg' as string]: token.colorSuccessBg,
      ['--otp-error' as string]: token.colorError,
      ['--otp-error-bg' as string]: token.colorErrorBg,
      ['--otp-scrollbar' as string]: token.colorFill,
    }),
    [token]
  );

  const treeQuery = useObjectTree(deviceId);
  const childrenQuery = useDirectChildren(deviceId, selectedPath, page, pageSize);
  const { data: syncStatus, refetch: refetchSyncStatus } = useSyncStatus(deviceId);
  const isSyncing = syncStatus?.status === 'syncing';
  const effectiveLastParamSyncAt = syncStatus?.lastParamSyncAt ?? syncStatus?.lastSyncGpv?.lastCompletedAt;
  const formatSyncAge = useCallback((value: string) => formatTimeAgo(new Date(value), t), [t]);

  // Mutations
  // T-0126: 切换到 useSyncDeviceParams（Path B + reason="manual"），替代旧 useSyncParameters (Path A)
  const syncMutation = useSyncDeviceParams();
  const syncBusy = externalSyncBusy || isSyncing || syncMutation.isPending;
  const syncControlLoading = syncMutation.isPending;
  const isLastScopedSync = Boolean(
    lastScopedSync?.targetCount && lastScopedSync.completedAt === effectiveLastParamSyncAt,
  );
  const addObjectMutation = useAddObject();
  const deleteObjectMutation = useDeleteObject();

  // 同步参数 - 触发 Path B 全量同步（完整接入 F09 触发链：差异日志 + 回写口径 + Translator）。
  // 入队成功后立即 refetch 一次 sync-status,让 status 尽快变 syncing
  // (避免按钮 disabled 跟 sync-status 联动期间的微小迟滞)。
  const handleSync = useCallback(() => {
    onFullSyncStarted?.();
    syncMutation.mutate(
      { deviceId },
      {
        onSuccess: (data) => {
          message.success(t('device.paramTree.syncQueued', { id: data.sourceId }));
          refetchSyncStatus();
        },
        onError: (err) => {
          // Path B unavailable (503) / device 404 / starter nil (500) — show explicit error
          const errorMsg = err instanceof Error ? err.message : t('device.paramTree.syncTriggerFailed');
          message.error(errorMsg);
        },
      }
    );
  }, [deviceId, syncMutation, refetchSyncStatus, onFullSyncStarted, t]);

  const handleSelectNode = useCallback((path: string) => {
    setSelectedPath(path);
    setPage(1);
  }, []);

  const handlePageChange = useCallback((newPage: number, newPageSize: number) => {
    setPage(newPage);
    setPageSize(newPageSize);
  }, []);

  const handleAddObject = useCallback(
    (objectPath: string) => {
      addObjectMutation.mutate(
        { deviceId, objectPath },
        {
          onSuccess: () => message.success(t('device.paramTree.addInstanceSent', { path: objectPath })),
          onError: () => message.error(t('device.paramTree.addInstanceFailed')),
        }
      );
    },
    [deviceId, addObjectMutation, t]
  );

  const handleDeleteObject = useCallback(
    (objectPath: string) => {
      deleteObjectMutation.mutate(
        { deviceId, objectPath },
        {
          onSuccess: () => message.success(t('device.paramTree.deleteInstanceSent', { path: objectPath })),
          onError: () => message.error(t('device.paramTree.deleteInstanceFailed')),
        }
      );
    },
    [deviceId, deleteObjectMutation, t]
  );

  return (
    <div style={{ padding: '16px 0' }}>
      {/* Toolbar */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 16,
          flexWrap: 'wrap',
          gap: 8,
        }}
      >
        <Input.Search
          placeholder={t('device.paramTree.searchPlaceholder')}
          allowClear
          onSearch={setSearchKeyword}
          style={{ width: 280 }}
          prefix={<SearchOutlined />}
        />
        <Space size={12}>
          {/* 持久状态小字: syncing 显示进行中+待处理批次, idle 显示上次同步相对时间。
              设备级共享状态,任何会话/任何用户打开都看到一致结果(后端 status 来自
              redis 队列长度 + db.last_param_sync_at)。 */}
          {/* toolbar 内联仅承载 syncing / succeeded / never 三态;失败态因错误信息
              可能很长,移到 toolbar 下方独立 Alert 渲染(下面 `{syncStatus?.lastParamSyncFailedAt && ...}`),
              避免挤压搜索框和按钮位置。 */}
          {syncStatus && (
            isSyncing ? (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                <LoadingOutlined style={{ marginRight: 4 }} />
                {t('device.paramTree.syncing')}
                {syncStatus.pendingCommands > 0
                  ? t('device.paramTree.syncPending', { count: syncStatus.pendingCommands })
                  : ''}
              </Typography.Text>
            ) : syncStatus.lastParamSyncFailedAt ? null : effectiveLastParamSyncAt ? (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {isLastScopedSync
                  ? t('device.paramTree.lastScopedSync', {
                    time: formatSyncAge(effectiveLastParamSyncAt),
                    count: lastScopedSync?.targetCount ?? 0,
                    gpvCount: lastScopedSync?.gpvTaskCount ?? 0,
                    duration: formatDuration(lastScopedSync?.wallClockSeconds) ?? '-',
                  })
                  : t('device.paramTree.lastSync', { time: formatSyncAge(effectiveLastParamSyncAt) })}
                {!isLastScopedSync && syncStatus.lastSyncGpv?.taskCount
                  ? t('device.paramTree.lastSyncGpvSummary', {
                    count: syncStatus.lastSyncGpv.taskCount,
                    duration: formatDuration(syncStatus.lastSyncGpv.wallClockSeconds) ?? '-',
                  })
                  : ''}
              </Typography.Text>
            ) : (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {t('device.paramTree.neverSynced')}
              </Typography.Text>
            )
          )}
          <Tooltip title={t('device.paramTree.syncTooltip')}>
            <Button
              icon={<SyncOutlined />}
              onClick={handleSync}
              loading={syncControlLoading}
              disabled={syncBusy}
            >
              {t('device.paramTree.syncParams')}
            </Button>
          </Tooltip>
        </Space>
      </div>

      {/* 失败态独立警示条:错误信息可能很长(完整 CPE Fault path + 文案),放 toolbar 内会
          挤压搜索框/按钮布局且影响阅读。用 Alert banner 形式占整宽,完整显示。 */}
      {syncStatus?.lastParamSyncFailedAt && !isSyncing && (
        <Alert
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
          message={t('device.paramTree.lastSyncFailed', { time: formatSyncAge(syncStatus.lastParamSyncFailedAt) })}
          description={syncStatus.lastParamSyncError || t('device.paramTree.noErrorDetail')}
        />
      )}

      {/* Split Layout: Tree + Table */}
      <div style={{ display: 'flex', gap: 16, minHeight: 500 }}>
        <div style={treePanelStyle}>
          <ObjectTreePanel
            treeData={treeQuery.data}
            loading={treeQuery.isLoading}
            selectedPath={selectedPath}
            searchKeyword={searchKeyword}
            onSelect={handleSelectNode}
            onAddObject={handleAddObject}
            onDeleteObject={handleDeleteObject}
          />
        </div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <ChildParamTable
            deviceId={deviceId}
            pathPrefix={selectedPath}
            data={childrenQuery.data}
            loading={childrenQuery.isLoading}
            page={page}
            pageSize={pageSize}
            onPageChange={handlePageChange}
            onNavigate={handleSelectNode}
          />
        </div>
      </div>
    </div>
  );
}
