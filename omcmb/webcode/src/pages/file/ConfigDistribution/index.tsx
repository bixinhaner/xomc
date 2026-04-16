import { useState, useMemo } from 'react';
import { Button, Tabs, Tree, Tag, Space, message, Radio, Upload } from 'antd';
import {
  UploadOutlined,
  MinusCircleOutlined,
  ClearOutlined,
  SendOutlined,
  DownloadOutlined,
  ReloadOutlined,
  InboxOutlined,
} from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import type { UploadFile, RcFile } from 'antd/es/upload';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';

const { Dragger } = Upload;

interface NEItem {
  id: string;
  neName: string;
  sn: string;
  neType: string;
  region: string;
}

interface DistributionResult {
  id: string;
  neName: string;
  sn: string;
  fileName: string;
  fileSize: number;
  status: 'success' | 'failed' | 'distributing' | 'pending';
  failReason?: string;
  distributionTime: string;
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

const mockNEs: NEItem[] = [
  { id: 'ne-001', neName: '北京-eNB-0001', sn: 'ENB00001', neType: 'eNB', region: '北京' },
  { id: 'ne-002', neName: '北京-eNB-0002', sn: 'ENB00002', neType: 'eNB', region: '北京' },
  { id: 'ne-003', neName: '上海-gNB-0001', sn: 'GNB00001', neType: 'gNB', region: '上海' },
];

const mockResults: DistributionResult[] = [
  { id: 'dr-001', neName: '北京-eNB-0001', sn: 'ENB00001', fileName: 'standard_config_v2.xml', fileSize: 131072, status: 'success', distributionTime: '2024-06-01T10:00:00.000Z' },
  { id: 'dr-002', neName: '北京-eNB-0002', sn: 'ENB00002', fileName: 'standard_config_v2.xml', fileSize: 131072, status: 'failed', failReason: '配置格式不兼容', distributionTime: '2024-06-01T10:01:00.000Z' },
];

function formatFileSize(bytes: number): string {
  if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
  return `${(bytes / 1024).toFixed(2)} KB`;
}

export default function ConfigDistribution() {
  const t = useT();
  const [activeTreeTab, setActiveTreeTab] = useState('type');
  const [selectedNEs, setSelectedNEs] = useState<NEItem[]>(mockNEs.slice(0, 2));
  const [results, setResults] = useState<DistributionResult[]>(mockResults);
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [executeMethod, setExecuteMethod] = useState<'immediate' | 'manual'>('immediate');
  const [distributing, setDistributing] = useState(false);

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

  const resultColumns: DataTableColumn<DistributionResult & Record<string, unknown>>[] = useMemo(() => [
    { key: 'neName', title: t('device.name'), dataIndex: 'neName', ellipsis: true },
    { key: 'sn', title: t('device.sn'), dataIndex: 'sn', width: 120, mono: true },
    { key: 'fileName', title: t('table.name'), dataIndex: 'fileName', ellipsis: true },
    { key: 'fileSize', title: t('table.description'), dataIndex: 'fileSize', width: 100, render: (val) => formatFileSize(Number(val)) },
    {
      key: 'status', title: t('table.status'), dataIndex: 'status', width: 90,
      render: (val) => {
        const m: Record<string, [string, string]> = { success: ['green', t('status.success')], failed: ['red', t('status.failed')], distributing: ['processing', t('status.running')], pending: ['default', t('status.pending')] };
        const [color, label] = m[String(val)] ?? ['default', String(val)];
        return <Tag color={color}>{label}</Tag>;
      },
    },
    {
      key: 'failReason', title: t('table.result'), dataIndex: 'failReason', width: 130,
      render: (val) => val ? <span style={{ color: '#ff4d4f', fontSize: 12 }}>{String(val)}</span> : '—',
    },
    { key: 'distributionTime', title: t('table.time'), dataIndex: 'distributionTime', width: 160, render: (val) => new Date(String(val)).toLocaleString('zh-CN') },
    {
      key: 'actions', title: t('table.operation'), dataIndex: 'id', width: 80, fixed: 'right',
      render: (_, record) => {
        const r = record as DistributionResult;
        if (r.status === 'success') return <Button type="link" size="small" icon={<DownloadOutlined />}>{t('common.download')}</Button>;
        if (r.status === 'failed') return <Button type="link" size="small" icon={<ReloadOutlined />}>{t('common.refresh')}</Button>;
        return null;
      },
    },
  ], [t]);

  const handleDistribute = () => {
    if (selectedNEs.length === 0) { void message.warning(t('common.pleaseSelect')); return; }
    if (fileList.length === 0) { void message.warning(t('common.pleaseSelect')); return; }
    setDistributing(true);
    setTimeout(() => {
      const newResults = selectedNEs.map((ne) => ({
        id: `dr-${Date.now()}-${ne.id}`,
        neName: ne.neName,
        sn: ne.sn,
        fileName: fileList[0]?.name ?? 'config.xml',
        fileSize: fileList[0]?.size ?? 0,
        status: (Math.random() > 0.15 ? 'success' : 'failed') as DistributionResult['status'],
        failReason: Math.random() > 0.85 ? '配置格式不兼容' : undefined,
        distributionTime: new Date().toISOString(),
      }));
      setResults((prev) => [...newResults, ...prev]);
      setDistributing(false);
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
      <div style={{ flex: 1, overflow: 'auto', padding: 8 }}>
        <Tree treeData={currentTreeData} defaultExpandAll checkable
          onSelect={() => setSelectedNEs(mockNEs.slice(0, 2))} />
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
              <Button
                type="primary"
                icon={<SendOutlined />}
                loading={distributing}
                onClick={handleDistribute}
              >
                {t('common.deploy')}
              </Button>
            </div>
          </div>
        </div>
        <div style={{ flex: 1, overflow: 'auto' }}>
          <div style={{ padding: '8px 12px', background: '#fafafa', borderBottom: '1px solid #f0f0f0' }}>
            <span style={{ fontWeight: 500 }}>{t('table.result')}: {results.length}</span>
          </div>
          <DataTable
            tableId="config-dist-results"
            columns={resultColumns}
            dataSource={results as (DistributionResult & Record<string, unknown>)[]}
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
