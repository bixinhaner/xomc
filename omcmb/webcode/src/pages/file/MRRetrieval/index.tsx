import { useState, useMemo } from 'react';
import { Button, Tabs, Tree, Tag, Space, message, Modal, Form, Checkbox, DatePicker } from 'antd';
import { DownloadOutlined, MinusCircleOutlined, ClearOutlined, SyncOutlined, ReloadOutlined } from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import type { Dayjs } from 'dayjs';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';

const { RangePicker } = DatePicker;

interface NEItem {
  id: string;
  neName: string;
  sn: string;
  neType: string;
  region: string;
}

interface MRRetrievalResult {
  id: string;
  neName: string;
  sn: string;
  fileName: string;
  fileSize: number;
  mrType: 'MRO' | 'MRE' | 'MRS';
  status: 'success' | 'failed' | 'retrieving' | 'pending';
  failReason?: string;
  retrievalTime: string;
}

const treeData: DataNode[] = [
  {
    key: 'enb', title: 'eNB设备',
    children: [
      { key: 'enb-bj', title: '北京eNB (12台)' },
      { key: 'enb-sh', title: '上海eNB (8台)' },
    ],
  },
  { key: 'gnb', title: 'gNB设备', children: [{ key: 'gnb-bj', title: '北京gNB (6台)' }] },
];

const treeTabs = [
  { key: 'type', label: '按类型', treeData },
  { key: 'subnet', label: '按子网', treeData: [{ key: 's1', title: '10.1.0.0/24' }] },
  { key: 'site', label: '按站点', treeData: [{ key: 'bj', title: '北京站点' }] },
  { key: 'region', label: '按区域', treeData: [{ key: 'bj', title: '北京区域' }] },
  { key: 'template', label: '按模板', treeData: [{ key: 'tpl-1', title: '标准模板' }] },
  { key: 'group', label: '按分组', treeData: [{ key: 'grp-1', title: '默认分组' }] },
];

const mockNEs: NEItem[] = [
  { id: 'ne-001', neName: '北京-eNB-0001', sn: 'ENB00001', neType: 'eNB', region: '北京' },
  { id: 'ne-002', neName: '北京-eNB-0002', sn: 'ENB00002', neType: 'eNB', region: '北京' },
  { id: 'ne-003', neName: '上海-gNB-0001', sn: 'GNB00001', neType: 'gNB', region: '上海' },
];

const mockResults: MRRetrievalResult[] = [
  { id: 'mr-001', neName: '北京-eNB-0001', sn: 'ENB00001', fileName: 'ENB00001_MRO_20240601.xml', fileSize: 1024 * 8192, mrType: 'MRO', status: 'success', retrievalTime: '2024-06-01T08:00:00.000Z' },
  { id: 'mr-002', neName: '北京-eNB-0001', sn: 'ENB00001', fileName: 'ENB00001_MRE_20240601.xml', fileSize: 1024 * 4096, mrType: 'MRE', status: 'success', retrievalTime: '2024-06-01T08:01:00.000Z' },
  { id: 'mr-003', neName: '北京-eNB-0002', sn: 'ENB00002', fileName: 'ENB00002_MRO_20240601.xml', fileSize: 0, mrType: 'MRO', status: 'failed', failReason: '暂无MR数据', retrievalTime: '2024-06-01T08:02:00.000Z' },
];

function formatFileSize(bytes: number): string {
  if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
  return `${(bytes / 1024).toFixed(2)} KB`;
}

const mrTypeColorMap: Record<string, string> = { MRO: 'blue', MRE: 'green', MRS: 'orange' };

export default function MRRetrieval() {
  const t = useT();
  const [activeTreeTab, setActiveTreeTab] = useState('type');
  const [selectedNEs, setSelectedNEs] = useState<NEItem[]>(mockNEs.slice(0, 2));
  const [results, setResults] = useState<MRRetrievalResult[]>(mockResults);
  const [retrieveVisible, setRetrieveVisible] = useState(false);
  const [form] = Form.useForm();
  const [retrieving, setRetrieving] = useState(false);

  const neColumns: DataTableColumn<NEItem & Record<string, unknown>>[] = useMemo(() => [
    { key: 'neName', title: t('device.name'), dataIndex: 'neName', ellipsis: true },
    { key: 'sn', title: t('device.sn'), dataIndex: 'sn', width: 120, mono: true },
    { key: 'neType', title: t('table.type'), dataIndex: 'neType', width: 80 },
    { key: 'region', title: t('table.region'), dataIndex: 'region', width: 80 },
    {
      key: 'actions', title: t('table.operation'), dataIndex: 'id', width: 80, fixed: 'right',
      render: (_, record) => {
        const ne = record as NEItem;
        return <Button type="link" size="small" danger icon={<MinusCircleOutlined />}
          onClick={() => setSelectedNEs((prev) => prev.filter((n) => n.id !== ne.id))}>{t('common.delete')}</Button>;
      },
    },
  ], [t]);

  const resultColumns: DataTableColumn<MRRetrievalResult & Record<string, unknown>>[] = useMemo(() => [
    { key: 'neName', title: t('device.name'), dataIndex: 'neName', ellipsis: true },
    { key: 'sn', title: t('device.sn'), dataIndex: 'sn', width: 120, mono: true },
    { key: 'fileName', title: t('table.name'), dataIndex: 'fileName', ellipsis: true },
    { key: 'fileSize', title: t('table.description'), dataIndex: 'fileSize', width: 100, render: (val) => formatFileSize(Number(val)) },
    {
      key: 'mrType', title: t('table.type'), dataIndex: 'mrType', width: 90,
      render: (val) => <Tag color={mrTypeColorMap[String(val)] ?? 'default'}>{String(val)}</Tag>,
    },
    {
      key: 'status', title: t('table.status'), dataIndex: 'status', width: 90,
      render: (val) => {
        const m: Record<string, [string, string]> = { success: ['green', t('status.success')], failed: ['red', t('status.failed')], retrieving: ['processing', t('status.running')], pending: ['default', t('status.pending')] };
        const [color, label] = m[String(val)] ?? ['default', String(val)];
        return <Tag color={color}>{label}</Tag>;
      },
    },
    {
      key: 'failReason', title: t('table.result'), dataIndex: 'failReason', width: 120,
      render: (val) => val ? <span style={{ color: '#ff4d4f', fontSize: 12 }}>{String(val)}</span> : '—',
    },
    { key: 'retrievalTime', title: t('table.time'), dataIndex: 'retrievalTime', width: 160, render: (val) => new Date(String(val)).toLocaleString('zh-CN') },
    {
      key: 'actions', title: t('table.operation'), dataIndex: 'id', width: 80, fixed: 'right',
      render: (_, record) => {
        const r = record as MRRetrievalResult;
        if (r.status === 'success') return <Button type="link" size="small" icon={<DownloadOutlined />}>{t('common.download')}</Button>;
        if (r.status === 'failed') return <Button type="link" size="small" icon={<ReloadOutlined />}>{t('common.refresh')}</Button>;
        return null;
      },
    },
  ], [t]);

  const handleRetrieve = () => {
    form.validateFields().then((vals) => {
      const mrTypes = vals.mrTypes as string[];
      const timeRange = vals.timeRange as [Dayjs, Dayjs];
      const days = timeRange[1].diff(timeRange[0], 'day');
      if (days > 3) { void message.error(t('common.featureInDev')); return; }
      setRetrieving(true);
      setRetrieveVisible(false);
      setTimeout(() => {
        const newResults = selectedNEs.flatMap((ne) =>
          mrTypes.map((mt) => ({
            id: `mr-${Date.now()}-${ne.id}-${mt}`,
            neName: ne.neName,
            sn: ne.sn,
            fileName: `${ne.sn}_${mt}_${timeRange[0].format('YYYYMMDD')}.xml`,
            fileSize: Math.floor(Math.random() * 1024 * 10240) + 1024 * 512,
            mrType: mt as MRRetrievalResult['mrType'],
            status: (Math.random() > 0.1 ? 'success' : 'failed') as MRRetrievalResult['status'],
            failReason: Math.random() > 0.9 ? '暂无MR数据' : undefined,
            retrievalTime: new Date().toISOString(),
          }))
        );
        setResults((prev) => [...newResults, ...prev]);
        setRetrieving(false);
        void message.success(t('status.success'));
        form.resetFields();
      }, 1500);
    });
  };

  const currentTreeData = treeTabs.find((t) => t.key === activeTreeTab)?.treeData ?? [];

  const treePanel = (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Tabs size="small" activeKey={activeTreeTab} onChange={setActiveTreeTab}
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
        <div style={{ flex: '0 0 auto', borderBottom: '1px solid #f0f0f0' }}>
          <div style={{ padding: '8px 12px', display: 'flex', justifyContent: 'space-between', alignItems: 'center', background: '#fafafa' }}>
            <span style={{ fontWeight: 500 }}>{t('table.total')}: {selectedNEs.length}</span>
            <Space>
              <Button size="small" icon={<ClearOutlined />} onClick={() => setSelectedNEs([])}>{t('common.reset')}</Button>
              <Button size="small" type="primary" icon={<SyncOutlined spin={retrieving} />} loading={retrieving}
                onClick={() => { if (selectedNEs.length === 0) { void message.warning(t('common.pleaseSelect')); return; } setRetrieveVisible(true); }}>
                {t('common.execute')}
              </Button>
            </Space>
          </div>
          <div style={{ maxHeight: 200, overflow: 'auto' }}>
            <DataTable tableId="mr-retrieval-ne-list" columns={neColumns}
              dataSource={selectedNEs as (NEItem & Record<string, unknown>)[]}
              loading={false} rowKey="id" showPagination={false} size="small" />
          </div>
        </div>
        <div style={{ flex: 1, overflow: 'auto' }}>
          <div style={{ padding: '8px 12px', background: '#fafafa', borderBottom: '1px solid #f0f0f0' }}>
            <span style={{ fontWeight: 500 }}>{t('table.result')}: {results.length}</span>
          </div>
          <DataTable tableId="mr-retrieval-results" columns={resultColumns}
            dataSource={results as (MRRetrievalResult & Record<string, unknown>)[]}
            loading={false} rowKey="id" showPagination={false} size="small" scroll={{ x: 1000 }} />
        </div>
      </div>

      <Modal
        title={`${t('common.execute')} (${selectedNEs.length})`}
        open={retrieveVisible}
        onOk={handleRetrieve}
        onCancel={() => { setRetrieveVisible(false); form.resetFields(); }}
        width={480}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="mrTypes" label={t('table.type')} initialValue={['MRO']} rules={[{ required: true }]}>
            <Checkbox.Group options={[
              { label: 'MRO', value: 'MRO' },
              { label: 'MRE', value: 'MRE' },
              { label: 'MRS', value: 'MRS' },
            ]} style={{ display: 'flex', flexDirection: 'column', gap: 8 }} />
          </Form.Item>
          <Form.Item name="timeRange" label={t('table.time')} rules={[{ required: true }]}>
            <RangePicker showTime style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>
    </TreeListPageLayout>
  );
}
