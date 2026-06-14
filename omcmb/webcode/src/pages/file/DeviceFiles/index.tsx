import { useState, useMemo } from 'react';
import { Button, Tabs, Tag, Space, Input, message } from 'antd';
import { DownloadOutlined, DeleteOutlined, SearchOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useFileList, useDownloadFile, useDeleteFiles } from '@core/hooks/api/useFiles';
import type { ManagedFile, FileType, FileStatus } from '@core/mock/data/fileManagement';
import { useT } from '@/hooks/useT';

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
  available: 'status.online', uploading: 'status.running', processing: 'status.running', expired: 'status.failed', deleted: 'status.disabled',
};

type DeviceFileRow = ManagedFile & Record<string, unknown>;

// 网元文件 = 设备维度的文件资产（按 file_type 分页签：配置 / 日志），真实接口 GET /files
function FileListTab({ fileType }: { fileType: FileType }) {
  const t = useT();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(PAGE_SIZE);
  const [keyword, setKeyword] = useState('');
  const [deviceSn, setDeviceSn] = useState('');

  const params = useMemo(
    () => ({
      fileType,
      page,
      pageSize,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(deviceSn.trim() ? { deviceSn: deviceSn.trim() } : {}),
    }),
    [fileType, page, pageSize, keyword, deviceSn]
  );

  const { data, isLoading, refetch } = useFileList(params);
  const download = useDownloadFile();
  const deleteFiles = useDeleteFiles();

  const columns: DataTableColumn<DeviceFileRow>[] = useMemo(() => [
    { key: 'fileName', title: t('table.name'), dataIndex: 'fileName', ellipsis: true },
    { key: 'deviceSn', title: t('device.sn'), dataIndex: 'deviceSn', width: 140, mono: true, render: (val) => (val ? String(val) : '—') },
    { key: 'fileSize', title: t('table.description'), dataIndex: 'fileSize', width: 100, render: (val) => formatFileSize(Number(val)) },
    {
      key: 'fileType', title: t('table.type'), dataIndex: 'fileType', width: 90,
      render: (val) => <Tag color={fileTypeColorMap[String(val)] ?? 'default'}>{String(val)}</Tag>,
    },
    {
      key: 'status', title: t('table.status'), dataIndex: 'status', width: 90,
      render: (val) => <Tag color={statusColorMap[val as FileStatus] ?? 'default'}>{t(statusLabelKeyMap[val as FileStatus] ?? 'status.pending')}</Tag>,
    },
    { key: 'uploader', title: t('table.operator'), dataIndex: 'uploader', width: 100, render: (val) => (val ? String(val) : '—') },
    { key: 'uploadTime', title: t('table.time'), dataIndex: 'uploadTime', width: 160, render: (val) => new Date(String(val)).toLocaleString('zh-CN') },
    {
      key: 'actions', title: t('table.operation'), dataIndex: 'id', width: 120, fixed: 'right' as const,
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

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <Space>
        <Input
          allowClear
          prefix={<SearchOutlined />}
          placeholder={t('table.name')}
          value={keyword}
          onChange={(e) => { setKeyword(e.target.value); setPage(1); }}
          style={{ width: 220 }}
        />
        <Input
          allowClear
          placeholder={t('device.sn')}
          value={deviceSn}
          onChange={(e) => { setDeviceSn(e.target.value); setPage(1); }}
          style={{ width: 200 }}
        />
      </Space>
      <DataTable
        tableId={`device-files-${fileType}`}
        columns={columns}
        dataSource={(data?.items ?? []) as DeviceFileRow[]}
        loading={isLoading}
        rowKey="id"
        total={data?.total ?? 0}
        pageSize={pageSize}
        currentPage={page}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
        onRefresh={() => void refetch()}
        scroll={{ x: 1000 }}
      />
    </div>
  );
}

export default function DeviceFiles() {
  const t = useT();
  const [activeTab, setActiveTab] = useState('config');

  return (
    <ListPageLayout title={t('nav.file.deviceFiles')}>
      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={[
          {
            key: 'config',
            label: t('nav.file.configRetrieval'),
            children: <FileListTab fileType="config" />,
          },
          {
            key: 'log',
            label: t('nav.file.logRetrieval'),
            children: <FileListTab fileType="log" />,
          },
        ]}
      />
    </ListPageLayout>
  );
}
