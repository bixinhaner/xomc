import { useState, useMemo } from 'react';
import { Button, Tabs, Tree, Tag, Space, Input, message, Radio, Upload } from 'antd';
import {
  UploadOutlined,
  MinusCircleOutlined,
  ClearOutlined,
  SendOutlined,
  DownloadOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import type { UploadFile, RcFile } from 'antd/es/upload';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useFileList, useDownloadFile, useDistributeFile } from '@core/hooks/api/useFiles';
import type { ManagedFile, FileStatus } from '@core/mock/data/fileManagement';
import { useT } from '@/hooks/useT';

const { Dragger } = Upload;

interface NEItem {
  id: string;
  neName: string;
  sn: string;
  neType: string;
  region: string;
}

const treeData: DataNode[] = [
  {
    key: 'enb', title: 'eNB设备',
    children: [
      { key: 'enb-bj', title: '北京eNB (12台)' },
      { key: 'enb-sh', title: '上海eNB (8台)' },
    ],
  },
  {
    key: 'gnb', title: 'gNB设备',
    children: [
      { key: 'gnb-bj', title: '北京gNB (6台)' },
    ],
  },
];

const treeTabs = [
  { key: 'type', label: '按类型', treeData },
  { key: 'subnet', label: '按子网', treeData: [{ key: 's1', title: '10.1.0.0/24' }] },
  { key: 'site', label: '按站点', treeData: [{ key: 'bj', title: '北京站点' }, { key: 'sh', title: '上海站点' }] },
  { key: 'region', label: '按区域', treeData: [{ key: 'bj', title: '北京区域' }, { key: 'sh', title: '上海区域' }] },
  { key: 'template', label: '按模板', treeData: [{ key: 'tpl-1', title: '标准模板' }] },
  { key: 'group', label: '按分组', treeData: [{ key: 'grp-1', title: '默认分组' }] },
];

const initialNEs: NEItem[] = [
  { id: 'ne-001', neName: '北京-eNB-0001', sn: 'ENB00001', neType: 'eNB', region: '北京' },
  { id: 'ne-002', neName: '北京-eNB-0002', sn: 'ENB00002', neType: 'eNB', region: '北京' },
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

// 配置下发 = 把配置文件库中的文件分发到目标设备，真实接口 GET /files?file_type=config + POST /files/:id/distribute
export default function ConfigDistribution() {
  const t = useT();
  const [activeTreeTab, setActiveTreeTab] = useState('type');
  const [selectedNEs, setSelectedNEs] = useState<NEItem[]>(initialNEs);
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [executeMethod, setExecuteMethod] = useState<'immediate' | 'manual'>('immediate');
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
  const distribute = useDistributeFile();

  const neColumns: DataTableColumn<NEItem & Record<string, unknown>>[] = useMemo(() => [
    { key: 'neName', title: t('device.name'), dataIndex: 'neName', ellipsis: true },
    { key: 'sn', title: t('device.sn'), dataIndex: 'sn', width: 120, mono: true },
    { key: 'neType', title: t('table.type'), dataIndex: 'neType', width: 80 },
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
          <Button type="link" size="small" danger icon={<MinusCircleOutlined />}
            onClick={() => setSelectedNEs((prev) => prev.filter((n) => n.id !== ne.id))}>
            {t('common.delete')}
          </Button>
        );
      },
    },
  ], [t]);

  const handleDistributeFile = (file: ManagedFile) => {
    if (selectedNEs.length === 0) { void message.warning(t('common.pleaseSelect')); return; }
    const deviceSns = selectedNEs.map((ne) => ne.sn);
    distribute.mutate(
      { fileId: file.id, deviceSns },
      {
        onSuccess: () => { void message.success(t('status.success')); void refetch(); },
        onError: () => void message.error(t('common.failed')),
      }
    );
  };

  const resultColumns: DataTableColumn<ConfigFileRow>[] = useMemo(() => [
    { key: 'fileName', title: t('table.name'), dataIndex: 'fileName', ellipsis: true },
    { key: 'deviceSn', title: t('device.sn'), dataIndex: 'deviceSn', width: 140, mono: true, render: (val) => (val ? String(val) : '—') },
    { key: 'fileSize', title: t('table.description'), dataIndex: 'fileSize', width: 100, render: (val) => formatFileSize(Number(val)) },
    {
      key: 'status', title: t('table.status'), dataIndex: 'status', width: 90,
      render: (val) => <Tag color={statusColorMap[val as FileStatus] ?? 'default'}>{t(statusLabelKeyMap[val as FileStatus] ?? 'status.pending')}</Tag>,
    },
    { key: 'uploadTime', title: t('table.time'), dataIndex: 'uploadTime', width: 160, render: (val) => new Date(String(val)).toLocaleString('zh-CN') },
    {
      key: 'actions', title: t('table.operation'), dataIndex: 'id', width: 160, fixed: 'right',
      render: (_, record) => {
        const file = record as ManagedFile;
        return (
          <Space size={4}>
            <Button
              type="link"
              size="small"
              icon={<SendOutlined />}
              disabled={distribute.isPending}
              onClick={() => handleDistributeFile(file)}
            >
              {t('common.deploy')}
            </Button>
            <Button
              type="link"
              size="small"
              icon={<DownloadOutlined />}
              disabled={file.status !== 'available' || download.isPending}
              onClick={() => download.mutate(file.id)}
            >
              {t('common.download')}
            </Button>
          </Space>
        );
      },
    },
  ], [t, download, distribute, selectedNEs]);

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
      <div style={{ flex: 1, overflow: 'auto', padding: 8 }}>
        <Tree treeData={currentTreeData} defaultExpandAll checkable
          onSelect={() => setSelectedNEs(initialNEs)} />
      </div>
    </div>
  );

  return (
    <TreeListPageLayout tree={treePanel}>
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
        <div style={{ flex: '0 0 auto', borderBottom: '1px solid #f0f0f0', padding: '8px 12px' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8, background: '#fafafa', padding: '6px 0' }}>
            <span style={{ fontWeight: 500 }}>{t('table.total')}: {selectedNEs.length}</span>
            <Button size="small" icon={<ClearOutlined />} onClick={() => setSelectedNEs([])}>{t('common.reset')}</Button>
          </div>
          <div style={{ maxHeight: 160, overflow: 'auto', marginBottom: 8 }}>
            <DataTable
              tableId="config-dist-ne-list"
              columns={neColumns}
              dataSource={selectedNEs as (NEItem & Record<string, unknown>)[]}
              loading={false}
              rowKey="id"
              showPagination={false}
              size="small"
            />
          </div>
          <div style={{ display: 'flex', gap: 16, alignItems: 'flex-start' }}>
            <div style={{ flex: 1 }}>
              <Dragger
                fileList={fileList}
                beforeUpload={(file: RcFile) => { setFileList([file]); return false; }}
                onRemove={() => setFileList([])}
                maxCount={1}
                accept=".xml,.cfg,.json,.txt"
                style={{ padding: '8px' }}
              >
                <p style={{ margin: 0 }}><UploadOutlined /> {t('common.upload')} (.xml / .cfg / .json)</p>
              </Dragger>
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
              <div>
                <span style={{ fontSize: 13, color: '#666', marginRight: 8 }}>{t('table.type')}:</span>
                <Radio.Group value={executeMethod} onChange={(e) => setExecuteMethod(e.target.value as 'immediate' | 'manual')}>
                  <Radio value="immediate">{t('common.execute')}</Radio>
                  <Radio value="manual">{t('common.confirm')}</Radio>
                </Radio.Group>
              </div>
            </div>
          </div>
        </div>
        <div style={{ flex: 1, overflow: 'auto', display: 'flex', flexDirection: 'column' }}>
          <div style={{ padding: '8px 12px', background: '#fafafa', borderBottom: '1px solid #f0f0f0', display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 12 }}>
            <span style={{ fontWeight: 500 }}>{t('table.total')}: {data?.total ?? 0}</span>
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
            tableId="config-dist-results"
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
