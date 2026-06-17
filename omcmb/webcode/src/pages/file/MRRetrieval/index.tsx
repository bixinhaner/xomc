import { useState, useMemo } from 'react';
import { Button, Tabs, Tree, Tag, Space, Input, message, Modal, Form, Checkbox, DatePicker } from 'antd';
import {
  DownloadOutlined,
  DeleteOutlined,
  MinusCircleOutlined,
  ClearOutlined,
  SyncOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import type { Dayjs } from 'dayjs';
import TreeListPageLayout from '@/components/Layout/TreeListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useFileList, useDownloadFile, useDeleteFiles } from '@core/hooks/api/useFiles';
import type { ManagedFile, FileType, FileStatus } from '@core/mock/data/fileManagement';
import { useT } from '@/hooks/useT';
import { formatSystemTime } from '@core/utils/systemTime';

const { RangePicker } = DatePicker;

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

const initialNEs: NEItem[] = [
  { id: 'ne-001', neName: '北京-eNB-0001', sn: 'ENB00001', neType: 'eNB', region: '北京' },
  { id: 'ne-002', neName: '北京-eNB-0002', sn: 'ENB00002', neType: 'eNB', region: '北京' },
];

function formatFileSize(bytes: number): string {
  if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
  return `${(bytes / 1024).toFixed(2)} KB`;
}

const PAGE_SIZE = 20;

const fileTypeColorMap: Record<string, string> = {
  config: 'blue', log: 'orange', firmware: 'purple', backup: 'cyan', report: 'green', certificate: 'gold',
};
const statusColorMap: Record<FileStatus, string> = {
  available: 'green', uploading: 'blue', processing: 'processing', expired: 'red', deleted: 'default',
};
const statusLabelKeyMap: Record<FileStatus, string> = {
  available: 'status.success', uploading: 'status.running', processing: 'status.running', expired: 'status.failed', deleted: 'status.disabled',
};

type MRFileRow = ManagedFile & Record<string, unknown>;

// 测量报告获取结果 = MR（MRO/MRS/MRE）文件库。后端 managed_files 暂无独立 mr 类型，
// 因此按设备维度展示全部已采集文件，真实接口 GET /files。
export default function MRRetrieval() {
  const t = useT();
  const [activeTreeTab, setActiveTreeTab] = useState('type');
  const [selectedNEs, setSelectedNEs] = useState<NEItem[]>(initialNEs);
  const [retrieveVisible, setRetrieveVisible] = useState(false);
  const [form] = Form.useForm();
  const [retrieving, setRetrieving] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(PAGE_SIZE);
  const [keyword, setKeyword] = useState('');

  const params = useMemo(
    () => ({
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

  const resultColumns: DataTableColumn<MRFileRow>[] = useMemo(() => [
    { key: 'fileName', title: t('table.name'), dataIndex: 'fileName', ellipsis: true },
    { key: 'deviceSn', title: t('device.sn'), dataIndex: 'deviceSn', width: 140, mono: true, render: (val) => (val ? String(val) : '—') },
    { key: 'fileSize', title: t('table.description'), dataIndex: 'fileSize', width: 100, render: (val) => formatFileSize(Number(val)) },
    {
      key: 'fileType', title: t('table.type'), dataIndex: 'fileType', width: 90,
      render: (val) => <Tag color={fileTypeColorMap[val as FileType] ?? 'default'}>{String(val)}</Tag>,
    },
    {
      key: 'status', title: t('table.status'), dataIndex: 'status', width: 90,
      render: (val) => <Tag color={statusColorMap[val as FileStatus] ?? 'default'}>{t(statusLabelKeyMap[val as FileStatus] ?? 'status.pending')}</Tag>,
    },
    { key: 'uploadTime', title: t('table.time'), dataIndex: 'uploadTime', width: 160, render: (val) => formatSystemTime(String(val)) },
    {
      key: 'actions', title: t('table.operation'), dataIndex: 'id', width: 120, fixed: 'right',
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
    form.validateFields().then((vals) => {
      const timeRange = vals.timeRange as [Dayjs, Dayjs];
      const days = timeRange[1].diff(timeRange[0], 'day');
      if (days > 3) { void message.error(t('common.featureInDev')); return; }
      setRetrieving(true);
      setRetrieveVisible(false);
      // 触发拉取后刷新真实文件列表
      void refetch().finally(() => {
        setRetrieving(false);
        void message.success(t('status.success'));
        form.resetFields();
      });
    });
  };

  const currentTreeData = treeTabs.find((tab) => tab.key === activeTreeTab)?.treeData ?? [];

  const treePanel = (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Tabs size="small" activeKey={activeTreeTab} onChange={setActiveTreeTab}
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
          <DataTable tableId="mr-retrieval-results" columns={resultColumns}
            dataSource={(data?.items ?? []) as MRFileRow[]}
            loading={isLoading} rowKey="id"
            total={data?.total ?? 0}
            pageSize={pageSize}
            currentPage={page}
            onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
            onRefresh={() => void refetch()}
            size="small" scroll={{ x: 1000 }} />
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
