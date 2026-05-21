import { useCallback, useMemo, useState } from 'react';
import { Alert, Button, Input, Space, Tooltip, Typography, message, theme } from 'antd';
import { SearchOutlined, SyncOutlined, LoadingOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import 'dayjs/locale/zh-cn';
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

dayjs.extend(relativeTime);
dayjs.locale('zh-cn');

interface ParameterTreeTabProps {
  deviceId: string;
}

export default function ParameterTreeTab({ deviceId }: ParameterTreeTabProps) {
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

  // Mutations
  // T-0126: 切换到 useSyncDeviceParams（Path B + reason="manual"），替代旧 useSyncParameters (Path A)
  const syncMutation = useSyncDeviceParams();
  const addObjectMutation = useAddObject();
  const deleteObjectMutation = useDeleteObject();

  // 同步参数 - 触发 Path B 全量同步（完整接入 F09 触发链：差异日志 + 回写口径 + Translator）。
  // 入队成功后立即 refetch 一次 sync-status,让 status 尽快变 syncing
  // (避免按钮 disabled 跟 sync-status 联动期间的微小迟滞)。
  const handleSync = useCallback(() => {
    syncMutation.mutate(
      { deviceId },
      {
        onSuccess: (data) => {
          message.success(`参数同步已入队（${data.sourceId}）`);
          refetchSyncStatus();
        },
        onError: (err) => {
          // Path B 不可用（503）/ 设备 404 / starter nil（500）— 显示明确错误
          const errorMsg = err instanceof Error ? err.message : '参数同步触发失败';
          message.error(errorMsg);
        },
      }
    );
  }, [deviceId, syncMutation, refetchSyncStatus]);

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
          onSuccess: () => message.success(`实例添加命令已下发: ${objectPath}`),
          onError: () => message.error('添加实例失败'),
        }
      );
    },
    [deviceId, addObjectMutation]
  );

  const handleDeleteObject = useCallback(
    (objectPath: string) => {
      deleteObjectMutation.mutate(
        { deviceId, objectPath },
        {
          onSuccess: () => message.success(`实例删除命令已下发: ${objectPath}`),
          onError: () => message.error('删除实例失败'),
        }
      );
    },
    [deviceId, deleteObjectMutation]
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
          placeholder="搜索参数路径或值"
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
          {syncStatus && !syncStatus.lastParamSyncFailedAt && (
            isSyncing ? (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                <LoadingOutlined style={{ marginRight: 4 }} />
                同步中
                {syncStatus.pendingCommands > 0
                  ? ` · 待处理 ${syncStatus.pendingCommands} 条命令`
                  : ''}
              </Typography.Text>
            ) : syncStatus.lastParamSyncAt ? (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                上次同步：{dayjs(syncStatus.lastParamSyncAt).fromNow()}
                {syncStatus.totalParameters > 0
                  ? ` · 共 ${syncStatus.totalParameters} 参数`
                  : ''}
              </Typography.Text>
            ) : (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                从未同步
              </Typography.Text>
            )
          )}
          <Tooltip title="从设备同步全部参数值（异步执行）">
            <Button
              icon={<SyncOutlined />}
              onClick={handleSync}
              loading={syncMutation.isPending}
              disabled={isSyncing}
            >
              同步参数
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
          message={`上次同步失败：${dayjs(syncStatus.lastParamSyncFailedAt).fromNow()}`}
          description={syncStatus.lastParamSyncError || '（无错误详情）'}
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
