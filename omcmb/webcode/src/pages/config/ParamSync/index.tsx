import { useState, useMemo } from 'react';
import { Button, Input, Tag, Tree, Typography, Space, message } from 'antd';
import { SyncOutlined, ExportOutlined, SearchOutlined } from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useConfigParams, useCreateConfigTask } from '@core/hooks/api/useConfig';
import { useT } from '@/hooks/useT';

interface ParamRow extends Record<string, unknown> {
  id: string;
  paramName: string;
  paramCode: string;
  currentValue: string;
  defaultValue: string;
  paramType: string;
  syncStatus: 'synced' | 'pending' | 'failed' | 'unknown';
  lastSyncTime: string;
}

const NE_TREE_DATA: DataNode[] = [
  {
    title: '全部网元',
    key: 'all',
    children: [
      {
        title: '4G基站 (eNB)',
        key: 'enb',
        children: [
          { title: 'ENB-北京-001', key: 'ENB00001' },
          { title: 'ENB-北京-002', key: 'ENB00002' },
          { title: 'ENB-上海-001', key: 'ENB00003' },
        ],
      },
      {
        title: '5G基站 (gNB)',
        key: 'gnb',
        children: [
          { title: 'GNB-北京-001', key: 'GNB00001' },
          { title: 'GNB-北京-002', key: 'GNB00002' },
        ],
      },
    ],
  },
];

export default function ParamSync() {
  const t = useT();
  const [selectedNE, setSelectedNE] = useState<string>('');
  const [searchValue, setSearchValue] = useState('');
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const { data, isLoading, refetch } = useConfigParams({
    deviceSn: selectedNE || undefined,
    page,
    pageSize,
  });

  const createTask = useCreateConfigTask();

  const SYNC_STATUS_CONFIG: Record<string, { color: string; text: string }> = useMemo(() => ({
    synced: { color: 'success', text: t('status.success') },
    pending: { color: 'processing', text: t('status.pending') },
    failed: { color: 'error', text: t('status.failed') },
    unknown: { color: 'default', text: t('common.noData') },
  }), [t]);

  const mockData: ParamRow[] = [
    { id: '1', paramName: '发射功率', paramCode: 'TX_POWER', currentValue: '46', defaultValue: '43', paramType: 'number', syncStatus: 'synced', lastSyncTime: '2026-03-01 10:00:00' },
    { id: '2', paramName: '小区带宽', paramCode: 'CELL_BW', currentValue: '20MHz', defaultValue: '20MHz', paramType: 'enum', syncStatus: 'pending', lastSyncTime: '2026-02-28 08:30:00' },
    { id: '3', paramName: '参考信号功率', paramCode: 'RS_POWER', currentValue: '-3', defaultValue: '0', paramType: 'number', syncStatus: 'failed', lastSyncTime: '2026-02-27 15:20:00' },
    { id: '4', paramName: '天线端口数', paramCode: 'ANT_PORTS', currentValue: '2', defaultValue: '2', paramType: 'number', syncStatus: 'synced', lastSyncTime: '2026-03-01 10:00:00' },
    { id: '5', paramName: '子帧配置', paramCode: 'SF_CFG', currentValue: '2', defaultValue: '2', paramType: 'number', syncStatus: 'synced', lastSyncTime: '2026-03-01 10:00:00' },
  ];

  const columns: DataTableColumn<ParamRow>[] = useMemo(() => [
    { key: 'paramName', title: t('config.paramName'), dataIndex: 'paramName', width: 180, ellipsis: true },
    { key: 'paramCode', title: t('config.paramCode'), dataIndex: 'paramCode', width: 160, mono: true, copyable: true },
    { key: 'currentValue', title: t('perf.value'), dataIndex: 'currentValue', width: 120 },
    { key: 'defaultValue', title: t('config.defaultValue'), dataIndex: 'defaultValue', width: 120 },
    { key: 'paramType', title: t('config.paramType'), dataIndex: 'paramType', width: 100 },
    {
      key: 'syncStatus',
      title: t('table.status'),
      dataIndex: 'syncStatus',
      width: 100,
      render: (val) => {
        const cfg = SYNC_STATUS_CONFIG[val as string] ?? SYNC_STATUS_CONFIG.unknown;
        return <Tag color={cfg.color}>{cfg.text}</Tag>;
      },
    },
    { key: 'lastSyncTime', title: t('table.updateTime'), dataIndex: 'lastSyncTime', width: 160 },
    {
      key: 'action',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 120,
      fixed: 'right',
      render: (_, record) => (
        <Space size={4}>
          <Button
            type="link"
            size="small"
            onClick={() => void message.success(`${t('config.sync')}: ${record.paramCode as string}`)}
          >
            {t('config.sync')}
          </Button>
          <Button
            type="link"
            size="small"
            onClick={() => void message.info(`${t('common.edit')}: ${record.paramCode as string}`)}
          >
            {t('common.edit')}
          </Button>
        </Space>
      ),
    },
  ], [t, SYNC_STATUS_CONFIG]);

  const handleBatchSync = () => {
    if (selectedKeys.length === 0) {
      void message.warning(t('common.pleaseSelect'));
      return;
    }
    createTask.mutate({
      taskName: `${t('config.sync')}-${new Date().toLocaleString()}`,
      taskType: 'param-sync',
      deviceSns: selectedNE ? [selectedNE] : [],
      totalCount: selectedKeys.length,
      creator: 'admin',
    });
    void message.success(t('common.exportInProgress'));
  };

  const treePanel = (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div style={{ padding: '12px 12px 8px' }}>
        <Typography.Text strong style={{ fontSize: 13 }}>{t('table.type')}</Typography.Text>
      </div>
      <div style={{ padding: '0 12px 8px' }}>
        <Input
          size="small"
          placeholder={t('common.search')}
          prefix={<SearchOutlined />}
          value={searchValue}
          onChange={(e) => setSearchValue(e.target.value)}
          allowClear
        />
      </div>
      <div style={{ flex: 1, overflow: 'auto', padding: '0 4px' }}>
        <Tree
          treeData={NE_TREE_DATA}
          onSelect={(keys) => {
            const key = keys[0] as string;
            if (key && !['all', 'enb', 'gnb'].includes(key)) {
              setSelectedNE(key);
            } else {
              setSelectedNE('');
            }
          }}
          defaultExpandAll
          showLine
        />
      </div>
    </div>
  );

  const tableSource = (data?.items ?? mockData) as unknown as ParamRow[];

  return (
    <TreeListPageLayout tree={treePanel}>
      <div style={{ padding: '12px 16px', borderBottom: '1px solid #f0f0f0', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Typography.Text strong style={{ fontSize: 14 }}>
          {selectedNE ? `${t('nav.config.paramSync')} — ${selectedNE}` : t('nav.config.paramSync')}
        </Typography.Text>
        <Space>
          <Button
            type="primary"
            icon={<SyncOutlined />}
            onClick={handleBatchSync}
            loading={createTask.isPending}
          >
            {t('config.sync')}
          </Button>
          <Button icon={<ExportOutlined />}>{t('common.export')}</Button>
        </Space>
      </div>
      <div style={{ flex: 1, overflow: 'auto' }}>
        <DataTable<ParamRow>
          tableId="param-sync"
          columns={columns}
          dataSource={tableSource}
          loading={isLoading}
          rowKey="id"
          selectable
          selectedRowKeys={selectedKeys}
          onSelectionChange={(keys) => setSelectedKeys(keys)}
          total={data?.total ?? tableSource.length}
          currentPage={page}
          pageSize={pageSize}
          onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
          onRefresh={() => void refetch()}
          onExport={() => void message.info(t('common.exportInProgress'))}
          scroll={{ x: 1000 }}
        />
      </div>
    </TreeListPageLayout>
  );
}
