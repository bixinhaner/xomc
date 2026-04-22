import { useState, useMemo } from 'react';
import { Button, Tabs, Tree, Tag, Space, message, Tooltip } from 'antd';
import {
  DownloadOutlined,
  ReloadOutlined,
  MinusCircleOutlined,
  ClearOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';

interface NEItem {
  id: string;
  neName: string;
  sn: string;
  neType: string;
  vendor: string;
  region: string;
}

interface RetrievalResult {
  id: string;
  neName: string;
  sn: string;
  fileName: string;
  fileSize: number;
  status: 'success' | 'failed' | 'retrieving' | 'pending';
  failReason?: string;
  retrievalTime: string;
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

const mockNEs: NEItem[] = [
  { id: 'ne-001', neName: '北京-eNB-0001', sn: 'ENB00001', neType: 'eNB', vendor: '华为', region: '北京' },
  { id: 'ne-002', neName: '北京-eNB-0002', sn: 'ENB00002', neType: 'eNB', vendor: '华为', region: '北京' },
  { id: 'ne-003', neName: '上海-gNB-0001', sn: 'GNB00001', neType: 'gNB', vendor: '中兴', region: '上海' },
  { id: 'ne-004', neName: '广州-RRU-0001', sn: 'RRU00001', neType: 'RRU', vendor: '华为', region: '广州' },
];

const mockResults: RetrievalResult[] = [
  { id: 'r-001', neName: '北京-eNB-0001', sn: 'ENB00001', fileName: 'ENB00001_config_20240601.xml', fileSize: 131072, status: 'success', retrievalTime: '2024-06-01T08:00:00.000Z' },
  { id: 'r-002', neName: '北京-eNB-0002', sn: 'ENB00002', fileName: 'ENB00002_config_20240601.xml', fileSize: 98304, status: 'failed', failReason: '设备连接超时', retrievalTime: '2024-06-01T08:01:00.000Z' },
  { id: 'r-003', neName: '上海-gNB-0001', sn: 'GNB00001', fileName: 'GNB00001_config_20240601.xml', fileSize: 262144, status: 'success', retrievalTime: '2024-06-01T08:02:00.000Z' },
];

function formatFileSize(bytes: number): string {
  if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
  return `${(bytes / 1024).toFixed(2)} KB`;
}

export default function ConfigRetrieval() {
  const t = useT();
  const [activeTreeTab, setActiveTreeTab] = useState('type');
  const [selectedNEs, setSelectedNEs] = useState<NEItem[]>(mockNEs.slice(0, 2));
  const [results, setResults] = useState<RetrievalResult[]>(mockResults);
  const [selectedNEKeys, setSelectedNEKeys] = useState<React.Key[]>([]);
  const [retrieving, setRetrieving] = useState(false);

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

  const resultColumns: DataTableColumn<RetrievalResult & Record<string, unknown>>[] = useMemo(() => [
    { key: 'neName', title: t('device.name'), dataIndex: 'neName', ellipsis: true },
    { key: 'sn', title: t('device.sn'), dataIndex: 'sn', width: 120, mono: true },
    { key: 'fileName', title: t('table.name'), dataIndex: 'fileName', ellipsis: true },
    { key: 'fileSize', title: t('table.description'), dataIndex: 'fileSize', width: 100, render: (val) => formatFileSize(Number(val)) },
    {
      key: 'status',
      title: t('table.status'),
      dataIndex: 'status',
      width: 90,
      render: (val) => {
        const colorMap: Record<string, string> = { success: 'green', failed: 'red', retrieving: 'processing', pending: 'default' };
        const labelKeyMap: Record<string, string> = { success: 'status.success', failed: 'status.failed', retrieving: 'status.running', pending: 'status.pending' };
        const s = String(val);
        return <Tag color={colorMap[s]}>{t(labelKeyMap[s])}</Tag>;
      },
    },
    {
      key: 'failReason',
      title: t('table.result'),
      dataIndex: 'failReason',
      width: 140,
      render: (val) => val ? <Tooltip title={String(val)}><span style={{ color: '#ff4d4f', cursor: 'pointer' }}>{String(val).substring(0, 12)}...</span></Tooltip> : '—',
    },
    { key: 'retrievalTime', title: t('table.time'), dataIndex: 'retrievalTime', width: 160, render: (val) => new Date(String(val)).toLocaleString('zh-CN') },
    {
      key: 'actions',
      title: t('table.operation'),
      dataIndex: 'id',
      width: 80,
      fixed: 'right',
      render: (_, record) => {
        const r = record as RetrievalResult;
        if (r.status === 'success') return <Button type="link" size="small" icon={<DownloadOutlined />}>{t('common.download')}</Button>;
        if (r.status === 'failed') return <Button type="link" size="small" icon={<ReloadOutlined />}>{t('common.refresh')}</Button>;
        return null;
      },
    },
  ], [t]);

  const handleRetrieve = () => {
    if (selectedNEs.length === 0) { void message.warning(t('common.pleaseSelect')); return; }
    setRetrieving(true);
    setTimeout(() => {
      const newResults = selectedNEs.map((ne) => ({
        id: `r-${Date.now()}-${ne.id}`,
        neName: ne.neName,
        sn: ne.sn,
        fileName: `${ne.sn}_config_${new Date().toISOString().slice(0, 10).replace(/-/g, '')}.xml`,
        fileSize: Math.floor(Math.random() * 512 * 1024) + 64 * 1024,
        status: (Math.random() > 0.2 ? 'success' : 'failed') as RetrievalResult['status'],
        failReason: Math.random() > 0.8 ? '设备连接超时' : undefined,
        retrievalTime: new Date().toISOString(),
      }));
      setResults((prev) => [...newResults, ...prev]);
      setRetrieving(false);
      void message.success(t('status.success'));
    }, 2000);
  };

  const currentTreeData = treeTabs.find((t) => t.key === activeTreeTab)?.treeData ?? [];

  const treePanel = (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Tabs
        size="small"
        activeKey={activeTreeTab}
        onChange={setActiveTreeTab}
        items={treeTabs.map((t) => ({ key: t.key, label: t.label }))}
        style={{ padding: '8px 8px 0' }}
      />
      <div style={{ flex: 1, overflow: 'auto', padding: '8px' }}>
        <Tree
          treeData={currentTreeData}
          defaultExpandAll
          checkable
          onSelect={() => {
            const sample = mockNEs.slice(0, 2);
            setSelectedNEs(sample);
          }}
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
        <div style={{ flex: 1, overflow: 'auto' }}>
          <div style={{ padding: '8px 12px', background: '#fafafa', borderBottom: '1px solid #f0f0f0' }}>
            <span style={{ fontWeight: 500 }}>{t('table.result')}: {results.length}</span>
          </div>
          <DataTable
            tableId="config-retrieval-results"
            columns={resultColumns}
            dataSource={results as (RetrievalResult & Record<string, unknown>)[]}
            loading={false}
            rowKey="id"
            showPagination={false}
            size="small"
            scroll={{ x: 900 }}
          />
        </div>
      </div>
    </TreeListPageLayout>
  );
}
