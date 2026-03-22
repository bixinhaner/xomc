import React, { useCallback, useMemo, useState } from 'react';
import { Button, Input, Radio, Space, Tooltip, message } from 'antd';
import {
  SearchOutlined,
  SyncOutlined,
  ApartmentOutlined,
  TableOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
import {
  useDeviceParameters,
  useDeviceParameterTree,
  useSyncParameters,
  useDiscoverParameters,
  useSyncStatus,
} from '@/hooks/api/useDeviceParameters';
import TreeView from './TreeView';
import TableView from './TableView';
import SyncStatusBar from './SyncStatusBar';

type ViewMode = 'tree' | 'table';

interface ParameterTreeTabProps {
  deviceId: string;
}

export default function ParameterTreeTab({ deviceId }: ParameterTreeTabProps) {
  const [viewMode, setViewMode] = useState<ViewMode>('tree');
  const [searchKeyword, setSearchKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [isSyncing, setIsSyncing] = useState(false);

  // Queries
  const parametersQuery = useDeviceParameters(
    deviceId,
    useMemo(
      () => ({
        page,
        pageSize,
        search: searchKeyword || undefined,
      }),
      [page, pageSize, searchKeyword]
    )
  );

  const treeQuery = useDeviceParameterTree(deviceId);

  const { data: syncStatus } = useSyncStatus(deviceId, isSyncing);

  // Stop polling when sync completes or fails
  React.useEffect(() => {
    if (
      syncStatus &&
      (syncStatus.status === 'completed' || syncStatus.status === 'failed')
    ) {
      // Keep visible for a moment then stop polling
      const timer = setTimeout(() => setIsSyncing(false), 5000);
      return () => clearTimeout(timer);
    }
  }, [syncStatus]);

  // Mutations
  const syncMutation = useSyncParameters();
  const discoverMutation = useDiscoverParameters();

  const handleSync = useCallback(() => {
    syncMutation.mutate(
      { deviceId },
      {
        onSuccess: () => {
          message.success('参数同步已触发');
          setIsSyncing(true);
        },
        onError: () => {
          message.error('参数同步触发失败');
        },
      }
    );
  }, [deviceId, syncMutation]);

  const handleDiscover = useCallback(() => {
    discoverMutation.mutate(deviceId, {
      onSuccess: () => {
        message.success('参数发现已触发');
        setIsSyncing(true);
      },
      onError: () => {
        message.error('参数发现触发失败');
      },
    });
  }, [deviceId, discoverMutation]);

  const handlePageChange = useCallback((newPage: number, newPageSize: number) => {
    setPage(newPage);
    setPageSize(newPageSize);
  }, []);

  const handleSearch = useCallback((value: string) => {
    setSearchKeyword(value);
    setPage(1);
  }, []);

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
        <Space wrap>
          <Input.Search
            placeholder="搜索参数路径或值"
            allowClear
            onSearch={handleSearch}
            style={{ width: 280 }}
            prefix={<SearchOutlined />}
          />
          <Radio.Group
            value={viewMode}
            onChange={(e) => setViewMode(e.target.value as ViewMode)}
            optionType="button"
            buttonStyle="solid"
            size="middle"
          >
            <Radio.Button value="tree">
              <ApartmentOutlined /> 树形
            </Radio.Button>
            <Radio.Button value="table">
              <TableOutlined /> 列表
            </Radio.Button>
          </Radio.Group>
        </Space>

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
          <Tooltip title="发现设备参数树结构（异步执行）">
            <Button
              type="primary"
              icon={<ThunderboltOutlined />}
              onClick={handleDiscover}
              loading={discoverMutation.isPending}
              disabled={isSyncing}
            >
              参数发现
            </Button>
          </Tooltip>
        </Space>
      </div>

      {/* Sync Progress */}
      <SyncStatusBar syncStatus={syncStatus} />

      {/* Content */}
      {viewMode === 'tree' ? (
        <TreeView
          deviceId={deviceId}
          treeData={treeQuery.data}
          loading={treeQuery.isLoading}
          searchKeyword={searchKeyword}
        />
      ) : (
        <TableView
          deviceId={deviceId}
          data={parametersQuery.data}
          loading={parametersQuery.isLoading}
          page={page}
          pageSize={pageSize}
          onPageChange={handlePageChange}
        />
      )}
    </div>
  );
}
