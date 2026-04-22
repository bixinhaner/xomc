import { useState, useMemo } from 'react';
import { Button, Card, Tag, message } from 'antd';
import { DownloadOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';

type MRFileType = 'MRO' | 'MRE' | 'MRS';

interface MRFile {
  id: string;
  fileName: string;
  mrType: MRFileType;
  deviceSn: string;
  deviceName: string;
  fileSize: number;
  collectTime: string;
  uploadTime: string;
  recordCount: number;
}

const mockMRFiles: MRFile[] = [
  { id: 'mrf-001', fileName: 'ENB00001_MRO_20240601_0000.xml', mrType: 'MRO', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', fileSize: 1024 * 8192, collectTime: '2024-06-01T00:00:00.000Z', uploadTime: '2024-06-01T00:05:00.000Z', recordCount: 12500 },
  { id: 'mrf-002', fileName: 'ENB00001_MRO_20240601_0015.xml', mrType: 'MRO', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', fileSize: 1024 * 7800, collectTime: '2024-06-01T00:15:00.000Z', uploadTime: '2024-06-01T00:20:00.000Z', recordCount: 11800 },
  { id: 'mrf-003', fileName: 'ENB00001_MRE_20240601_0000.xml', mrType: 'MRE', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', fileSize: 1024 * 4096, collectTime: '2024-06-01T00:00:00.000Z', uploadTime: '2024-06-01T00:05:00.000Z', recordCount: 3200 },
  { id: 'mrf-004', fileName: 'ENB00002_MRO_20240601_0000.xml', mrType: 'MRO', deviceSn: 'ENB00002', deviceName: '北京-eNB-0002', fileSize: 1024 * 9500, collectTime: '2024-06-01T00:00:00.000Z', uploadTime: '2024-06-01T00:06:00.000Z', recordCount: 14200 },
  { id: 'mrf-005', fileName: 'GNB00001_MRO_20240601_0000.xml', mrType: 'MRO', deviceSn: 'GNB00001', deviceName: '北京-gNB-0001', fileSize: 1024 * 15360, collectTime: '2024-06-01T00:00:00.000Z', uploadTime: '2024-06-01T00:10:00.000Z', recordCount: 28000 },
  { id: 'mrf-006', fileName: 'ENB00001_MRS_20240601.xml', mrType: 'MRS', deviceSn: 'ENB00001', deviceName: '北京-eNB-0001', fileSize: 1024 * 1024, collectTime: '2024-06-01T01:00:00.000Z', uploadTime: '2024-06-01T01:05:00.000Z', recordCount: 96 },
  { id: 'mrf-007', fileName: 'ENB00010_MRO_20240601_0000.xml', mrType: 'MRO', deviceSn: 'ENB00010', deviceName: '上海-eNB-0001', fileSize: 1024 * 6800, collectTime: '2024-06-01T00:00:00.000Z', uploadTime: '2024-06-01T00:08:00.000Z', recordCount: 10200 },
  { id: 'mrf-008', fileName: 'ENB00010_MRE_20240601_0000.xml', mrType: 'MRE', deviceSn: 'ENB00010', deviceName: '上海-eNB-0001', fileSize: 1024 * 3200, collectTime: '2024-06-01T00:00:00.000Z', uploadTime: '2024-06-01T00:08:00.000Z', recordCount: 2500 },
];

const mrTypeColorMap: Record<MRFileType, string> = { MRO: 'blue', MRE: 'green', MRS: 'orange' };

function formatFileSize(bytes: number): string {
  if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
  return `${(bytes / 1024).toFixed(2)} KB`;
}

export default function Files() {
  const t = useT();
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('mr.fileName'), type: 'input', placeholder: t('mr.fileNamePlaceholder') },
    {
      name: 'mrType',
      label: t('mr.mrType'),
      type: 'select',
      options: [
        { label: 'MRO', value: 'MRO' },
        { label: 'MRE', value: 'MRE' },
        { label: 'MRS', value: 'MRS' },
      ],
    },
    { name: 'deviceSn', label: t('device.sn'), type: 'input', placeholder: t('mr.deviceSnPlaceholder') },
    { name: 'timeRange', label: t('mr.collectTime'), type: 'date-range' },
  ], [t]);

  const filtered = mockMRFiles.filter((f) => {
    if (filters.keyword && !f.fileName.includes(String(filters.keyword))) return false;
    if (filters.mrType && f.mrType !== filters.mrType) return false;
    if (filters.deviceSn && !f.deviceSn.includes(String(filters.deviceSn))) return false;
    return true;
  });

  const startIndex = (page - 1) * pageSize;
  const paginated = filtered.slice(startIndex, startIndex + pageSize);

  const columns: DataTableColumn<MRFile & Record<string, unknown>>[] = useMemo(() => [
    { key: 'fileName', title: t('mr.fileName'), dataIndex: 'fileName', ellipsis: true, width: 260 },
    {
      key: 'mrType', title: t('mr.mrType'), dataIndex: 'mrType', width: 90,
      render: (val) => <Tag color={mrTypeColorMap[val as MRFileType]}>{String(val)}</Tag>,
    },
    { key: 'deviceSn', title: t('device.sn'), dataIndex: 'deviceSn', width: 120, render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span> },
    { key: 'deviceName', title: t('device.name'), dataIndex: 'deviceName', ellipsis: true, width: 160 },
    { key: 'fileSize', title: t('mr.fileSize'), dataIndex: 'fileSize', width: 100, render: (val) => formatFileSize(Number(val)) },
    { key: 'recordCount', title: t('mr.recordCount'), dataIndex: 'recordCount', width: 90 },
    {
      key: 'collectTime', title: t('mr.collectTime'), dataIndex: 'collectTime', width: 160,
      render: (val) => new Date(String(val)).toLocaleString('zh-CN'),
    },
    {
      key: 'uploadTime', title: t('mr.uploadTime'), dataIndex: 'uploadTime', width: 160,
      render: (val) => new Date(String(val)).toLocaleString('zh-CN'),
    },
    {
      key: 'actions', title: t('table.operation'), dataIndex: 'id', width: 80, fixed: 'right',
      render: () => (
        <Button type="link" size="small"
          onClick={() => void message.success(t('mr.downloadTaskCreated'))}>
          {t('common.download')}
        </Button>
      ),
    },
  ], [t]);

  return (
    <ListPageLayout
      title={t('nav.mr.files')}
      subtitle={t('mr.filesSubtitle')}
      extra={
        <Button
          icon={<DownloadOutlined />}
          disabled={selectedKeys.length === 0}
          onClick={() => void message.success(t('mr.batchDownload', { count: String(selectedKeys.length) }))}
        >
          {t('common.batchExport')}
        </Button>
      }
    >
      <FilterBar
        filterId="mr-files-filter"
        fields={filterFields}
        onSearch={(vals) => { setFilters(vals); setPage(1); }}
        onReset={() => { setFilters({}); setPage(1); }}
      />
      <Card
        size="small"
        bordered
        style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
        styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
      >
        <DataTable
          tableId="mr-files-list"
          columns={columns}
          dataSource={paginated as (MRFile & Record<string, unknown>)[]}
          loading={false}
          rowKey="id"
          total={filtered.length}
          pageSize={pageSize}
          currentPage={page}
          onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
          selectable
          selectedRowKeys={selectedKeys}
          onSelectionChange={(keys) => setSelectedKeys(keys)}
          onExport={(format) => void console.log(t('common.export'), format)}
          scroll={{ x: 1200 }}
        />
      </Card>
    </ListPageLayout>
  );
}
