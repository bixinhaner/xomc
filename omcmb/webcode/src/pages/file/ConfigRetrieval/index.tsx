import { useState, useMemo } from 'react';
import { Button, Tabs, Tree, Tag, Space, Input, message } from 'antd';
import {
  DownloadOutlined,
  DeleteOutlined,
  MinusCircleOutlined,
  ClearOutlined,
  SyncOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useFileList, useDownloadFile, useDeleteFiles } from '@core/hooks/api/useFiles';
import type { ManagedFile, FileStatus } from '@core/mock/data/fileManagement';
import { useT } from '@/hooks/useT';

interface NEItem {
  id: string;
  neName: string;
  sn: string;
  neType: string;
  vendor: string;
  region: string;
}

const treeDataByType: DataNode[] = [
  {
    key: 'enb', title: 'eNB设备',
    children: [
      { key: 'enb-bj', title: '北京eNB (12台)' },
      { key: 'enb-sh', title: '上海eNB (8台)' },
      { key: 'enb-gz', title: '广州eNB (15台)' },
    ],
  },
  {
    key: 'gnb', title: 'gNB设备',
    children: [
      { key: 'gnb-bj', title: '北京gNB (6台)' },
      { key: 'gnb-sh', title: '上海gNB (4台)' },
    ],
  },
  { key: 'rru', title: 'RRU设备', children: [{ key: 'rru-all', title: '全部RRU (48台)' }] },
];

const treeDataBySubnet: DataNode[] = [
  { key: 'subnet-10', title: '10.x.x.x 网段', children: [{ key: 'subnet-10-1', title: '10.1.0.0/24' }] },
  { key: 'subnet-172', title: '172.x.x.x 网段', children: [{ key: 'subnet-172-1', title: '172.16.0.0/24' }] },
];

const treeDataBySite: DataNode[] = [
  { key: 'site-bj', title: '北京', children: [{ key: 'site-bj-1', title: '北京站点A' }, { key: 'site-bj-2', title: '北京站点B' }] },
  { key: 'site-sh', title: '上海', children: [{ key: 'site-sh-1', title: '上海站点A' }] },
];

const treeTabs = [
  { key: 'type', label: '按类型', treeData: treeDataByType },
  { key: 'subnet', label: '按子网', treeData: treeDataBySubnet },
  { key: 'site', label: '按站点', treeData: treeDataBySite },
  { key: 'region', label: '按区域', treeData: [{ key: 'bj', title: '北京区域' }, { key: 'sh', title: '上海区域' }] },
  { key: 'template', label: '按模板', treeData: [{ key: 'tpl-1', title: '标准eNB模板' }, { key: 'tpl-2', title: '高性能gNB模板' }] },
  { key: 'group', label: '按分组', treeData: [{ key: 'grp-1', title: '默认分组' }, { key: 'grp-2', title: '测试分组' }] },
];

const initialNEs: NEItem[] = [
  { id: 'ne-001', neName: '北京-eNB-0001', sn: 'ENB00001', neType: 'eNB', vendor: '华为', region: '北京' },
  { id: 'ne-002', neName: '北京-eNB-0002', sn: 'ENB00002', neType: 'eNB', vendor: '华为', region: '北京' },
];

function formatFileSize(bytes: number): string {
  if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
  return `${(bytes / 1024).toFixed(2)} KB`;
}

const PAGE_SIZE = 20;

const statusColorMap: Record<FileStatus, string> = {
  available: 'green', uploading: 'blue', processing: 'processing', expired: 'red', deleted: 'default',
};
const statusLabelKeyMap: Record<FileStatus, string> = {
  available: 'status.success', uploading: 'status.running', processing: 'status.running', expired: 'status.failed', deleted: 'status.disabled',
};

type ConfigFileRow = ManagedFile & Record<string, unknown>;

// 配置文件拉取结果 = 已采集到 ACS 的设备配置文件库，真实接口 GET /files?file_type=config
export default function ConfigRetrieval() {
  const t = useT();
  const [activeTreeTab, setActiveTreeTab] = useState('type');
  const [selectedNEs, setSelectedNEs] = useState<NEItem[]>(initialNEs);
  const [selectedNEKeys, setSelectedNEKeys] = useState<React.Key[]>([]);
  const [retrieving, setRetrieving] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(PAGE_SIZE);
  const [keyword, setKeyword] = useState('');

  const params = useMemo(
    () => ({
      fileType: 'config' as const,
      page,
      pageSize,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
    }),
    [page, pageSize, keyword]
  );

  const { data, isLoading, refetch } = useFileList(params);
  const download = useDownloadFile();
  const deleteFiles = useDeleteFiles();

  const neColumns: DataTableColumn<NEItem & Record<string, unknown>>[] = useMemo(() => [
    { key: 'neName', title: t('device.name'), dataIndex: 'neName', ellipsis: true },
    { key: 'sn', title: t('device.sn'), dataIndex: 'sn', width: 120, mono: true },
    { key: 'neType', title: t('table.type'), dataIndex: 'neType', width: 80 },
    { key: 'vendor', title: t('device.vendor'), dataIndex: 'vendor', width: 80 },
    { key: 'region', title: t('table.region'), dataIndex: 'region', width: 80 },
    {
      key: 'actions',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 80,
      fixed: 'right',
      render: (_, record) => {
        const ne = record as NEItem;
        return (
          <Button
            type="link"
            size="small"
            danger
            icon={<MinusCircleOutlined />}
            onClick={() => setSelectedNEs((prev) => prev.filter((n) => n.id !== ne.id))}
          >
            {t('common.delete')}
          </Button>
        );
      },
    },
  ], [t]);

  const resultColumns: DataTableColumn<ConfigFileRow>[] = useMemo(() => [
    { key: 'fileName', title: t('table.name'), dataIndex: 'fileName', ellipsis: true },
    { key: 'deviceSn', title: t('device.sn'), dataIndex: 'deviceSn', width: 140, mono: true, render: (val) => (val ? String(val) : '—') },
    { key: 'fileSize', title: t('table.description'), dataIndex: 'fileSize', width: 100, render: (val) => formatFileSize(Number(val)) },
    {
      key: 'status',
      title: t('table.status'),
      dataIndex: 'status',
      width: 90,
      render: (val) => <Tag color={statusColorMap[val as FileStatus] ?? 'default'}>{t(statusLabelKeyMap[val as FileStatus] ?? 'status.pending')}</Tag>,
    },
    { key: 'uploader', title: t('table.operator'), dataIndex: 'uploader', width: 100, render: (val) => (val ? String(val) : '—') },
    { key: 'uploadTime', title: t('table.time'), dataIndex: 'uploadTime', width: 160, render: (val) => new Date(String(val)).toLocaleString('zh-CN') },
    {
      key: 'actions',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 120,
      fixed: 'right',
      render: (_, record) => {
        const file = record as ManagedFile;
        return (
          <Space size={4}>
            <Button
              type="link"
              size="small"
              icon={<DownloadOutlined />}
              disabled={file.status !== 'available' || download.isPending}
              onClick={() => download.mutate(file.id)}
            >
              {t('common.download')}
            </Button>
            <Button
              type="link"
              size="small"
              danger
              icon={<DeleteOutlined />}
              onClick={() => deleteFiles.mutate([file.id], { onSuccess: () => void message.success(t('common.deleteSuccess')) })}
            >
              {t('common.delete')}
            </Button>
          </Space>
        );
      },
    },
  ], [t, download, deleteFiles]);

  const handleRetrieve = () => {
    if (selectedNEs.length === 0) { void message.warning(t('common.pleaseSelect')); return; }
    setRetrieving(true);
    // 触发拉取后刷新真实列表（拉取动作本身为后端任务，UI 仅刷新结果库）
    void refetch().finally(() => {
      setRetrieving(false);
      void message.success(t('status.success'));
    });
  };

  const currentTreeData = treeTabs.find((tab) => tab.key === activeTreeTab)?.treeData ?? [];

  const treePanel = (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Tabs
        size="small"
        activeKey={activeTreeTab}
        onChange={setActiveTreeTab}
        items={treeTabs.map((tab) => ({ key: tab.key, label: tab.label }))}
        style={{ padding: '8px 8px 0' }}
      />
      <div style={{ flex: 1, overflow: 'auto', padding: '8px' }}>
        <Tree
          treeData={currentTreeData}
          defaultExpandAll
          checkable
          onSelect={() => setSelectedNEs(initialNEs)}
        />
      </div>
    </div>
  );

  return (
    <TreeListPageLayout tree={treePanel}>
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
        <div style={{ flex: '0 0 auto', borderBottom: '1px solid #f0f0f0' }}>
          <div style={{ padding: '8px 12px', display: 'flex', justifyContent: 'space-between', alignItems: 'center', background: '#fafafa' }}>
            <span style={{ fontWeight: 500 }}>{t('table.total')}: {selectedNEs.length}</span>
            <Space>
              <Button size="small" icon={<ClearOutlined />} onClick={() => setSelectedNEs([])}>{t('common.reset')}</Button>
              <Button
                size="small"
                type="primary"
                icon={<SyncOutlined spin={retrieving} />}
                loading={retrieving}
                onClick={handleRetrieve}
              >
                {t('common.execute')}
              </Button>
            </Space>
          </div>
          <div style={{ maxHeight: 220, overflow: 'auto' }}>
            <DataTable
              tableId="config-retrieval-ne-list"
              columns={neColumns}
              dataSource={selectedNEs as (NEItem & Record<string, unknown>)[]}
              loading={false}
              rowKey="id"
              selectedRowKeys={selectedNEKeys}
              onSelectionChange={(keys) => setSelectedNEKeys(keys)}
              selectable
              showPagination={false}
              size="small"
            />
          </div>
        </div>
        <div style={{ flex: 1, overflow: 'auto', display: 'flex', flexDirection: 'column' }}>
          <div style={{ padding: '8px 12px', background: '#fafafa', borderBottom: '1px solid #f0f0f0', display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 12 }}>
            <span style={{ fontWeight: 500 }}>{t('table.result')}: {data?.total ?? 0}</span>
            <Input
              allowClear
              prefix={<SearchOutlined />}
              placeholder={t('table.name')}
              value={keyword}
              onChange={(e) => { setKeyword(e.target.value); setPage(1); }}
              style={{ width: 220 }}
            />
          </div>
          <DataTable
            tableId="config-retrieval-results"
            columns={resultColumns}
            dataSource={(data?.items ?? []) as ConfigFileRow[]}
            loading={isLoading}
            rowKey="id"
            total={data?.total ?? 0}
            pageSize={pageSize}
            currentPage={page}
            onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
            onRefresh={() => void refetch()}
            size="small"
            scroll={{ x: 900 }}
          />
        </div>
      </div>
    </TreeListPageLayout>
  );
}
