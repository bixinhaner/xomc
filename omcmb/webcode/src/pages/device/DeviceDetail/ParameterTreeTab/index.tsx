import React, { useCallback, useState } from 'react';
import { Button, Input, Space, Tooltip, message } from 'antd';
import { SearchOutlined, SyncOutlined } from '@ant-design/icons';
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
import SyncStatusBar from './SyncStatusBar';

interface ParameterTreeTabProps {
  deviceId: string;
}

export default function ParameterTreeTab({ deviceId }: ParameterTreeTabProps) {
  const [selectedPath, setSelectedPath] = useState<string>('');
  const [searchKeyword, setSearchKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [isSyncing, setIsSyncing] = useState(false);

  const treeQuery = useObjectTree(deviceId);
  const childrenQuery = useDirectChildren(deviceId, selectedPath, page, pageSize);
  const { data: syncStatus } = useSyncStatus(deviceId, isSyncing);

  // Stop polling when sync completes or fails
  React.useEffect(() => {
    if (
      syncStatus &&
      (syncStatus.status === 'completed' || syncStatus.status === 'failed')
    ) {
      const timer = setTimeout(() => setIsSyncing(false), 5000);
      return () => clearTimeout(timer);
    }
  }, [syncStatus]);

  // Mutations
  // T-0126: 切换到 useSyncDeviceParams（Path B + reason="manual"），替代旧 useSyncParameters (Path A)
  const syncMutation = useSyncDeviceParams();
  const addObjectMutation = useAddObject();
  const deleteObjectMutation = useDeleteObject();

  // 同步参数 - 触发 Path B 全量同步（完整接入 F09 触发链：差异日志 + 回写口径 + Translator）
  const handleSync = useCallback(() => {
    syncMutation.mutate(
      { deviceId },
      {
        onSuccess: (data) => {
          message.success(`参数同步已入队（${data.sourceId}）`);
          setIsSyncing(true);
        },
        onError: (err) => {
          // Path B 不可用（503）/ 设备 404 / starter nil（500）— 显示明确错误
          const errorMsg = err instanceof Error ? err.message : '参数同步触发失败';
          message.error(errorMsg);
        },
      }
    );
  }, [deviceId, syncMutation]);

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
        <Space>
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

      {/* Sync Progress */}
      <SyncStatusBar syncStatus={syncStatus} />

      {/* Split Layout: Tree + Table */}
      <div style={{ display: 'flex', gap: 16, minHeight: 500 }}>
        <div
          style={{
            width: 320,
            flexShrink: 0,
            border: '1px solid #e8e8e8',
            borderRadius: 8,
            overflow: 'hidden',
            maxHeight: 700,
            boxShadow: '0 1px 4px rgba(0,0,0,0.04)',
            background: '#fff',
          }}
        >
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
