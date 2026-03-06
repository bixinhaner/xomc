import { useState, useMemo } from 'react';
import { Button, Tabs, Tag, Space, Tooltip, message, Modal, Upload } from 'antd';
import { UploadOutlined, DownloadOutlined, DeleteOutlined, PlusOutlined, InboxOutlined } from '@ant-design/icons';
import type { UploadFile, RcFile } from 'antd/es/upload';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useFileList, useDeleteFiles } from '@/hooks/api/useFiles';
import type { ManagedFile, FileType } from '@/mock/data/fileManagement';
import { useT } from '@/hooks/useT';

const { Dragger } = Upload;

function formatFileSize(bytes: number): string {
  if (bytes >= 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
  if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
  return `${(bytes / 1024).toFixed(2)} KB`;
}

const fileTypeColorMap: Record<string, string> = {
  config: 'blue', log: 'orange', firmware: 'purple', backup: 'cyan', report: 'green', certificate: 'gold',
};
const fileTypeLabelMap: Record<string, string> = {
  config: 'config', log: 'log', firmware: 'firmware', backup: 'backup', report: 'report', certificate: 'certificate',
};

const reviewStatusColorMap: Record<string, string> = {
  approved: 'green', pending: 'orange', rejected: 'red', draft: 'default',
};
const reviewStatusLabelKeyMap: Record<string, string> = {
  approved: 'status.success', pending: 'status.pending', rejected: 'status.failed', draft: 'status.disabled',
};

interface FirmwareItem {
  id: string;
  fileName: string;
  fileSize: number;
  deviceType: string;
  version: string;
  uploadTime: string;
  uploader: string;
}

const mockFirmware: FirmwareItem[] = [
  { id: 'fw-001', fileName: 'eNB_V100R011C10SPC200.tar.gz', fileSize: 1024 * 1024 * 512, deviceType: 'eNB', version: 'V100R011C10SPC200', uploadTime: '2024-06-01T09:00:00.000Z', uploader: 'admin' },
  { id: 'fw-002', fileName: 'gNB_V200R001C10SPC100.tar.gz', fileSize: 1024 * 1024 * 768, deviceType: 'gNB', version: 'V200R001C10SPC100', uploadTime: '2024-06-05T14:00:00.000Z', uploader: 'admin' },
  { id: 'fw-003', fileName: 'RRU_V300R002C00SPC050.bin', fileSize: 1024 * 1024 * 128, deviceType: 'RRU', version: 'V300R002C00SPC050', uploadTime: '2024-05-20T10:00:00.000Z', uploader: 'operator1' },
];

export default function UserFiles() {
  const t = useT();
  const [activeTab, setActiveTab] = useState('user');
  const [userFilters, setUserFilters] = useState<Record<string, unknown>>({});
  const [fwFilters, setFwFilters] = useState<Record<string, unknown>>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);
  const [uploadVisible, setUploadVisible] = useState(false);
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [firmware, setFirmware] = useState<FirmwareItem[]>(mockFirmware);

  const { data, isLoading, refetch } = useFileList({
    ...userFilters,
    fileType: userFilters.fileType as FileType | undefined,
    page,
    pageSize,
  });
  const deleteFiles = useDeleteFiles();

  const mockReviewStatus = (id: string) => {
    const statuses = ['approved', 'pending', 'rejected', 'draft'];
    return statuses[id.charCodeAt(id.length - 1) % statuses.length];
  };

  const userFileFilterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('table.name'), type: 'input', placeholder: t('common.placeholder') },
    {
      name: 'fileType',
      label: t('table.type'),
      type: 'select',
      options: [
        { label: 'config', value: 'config' },
        { label: 'log', value: 'log' },
        { label: 'firmware', value: 'firmware' },
        { label: 'backup', value: 'backup' },
        { label: 'report', value: 'report' },
      ],
    },
    {
      name: 'reviewStatus',
      label: t('table.status'),
      type: 'select',
      options: [
        { label: t('status.success'), value: 'approved' },
        { label: t('status.pending'), value: 'pending' },
        { label: t('status.failed'), value: 'rejected' },
      ],
    },
  ], [t]);

  const firmwareFilterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('table.name'), type: 'input', placeholder: t('common.placeholder') },
    {
      name: 'deviceType',
      label: t('device.productType'),
      type: 'select',
      options: [
        { label: 'eNB', value: 'eNB' },
        { label: 'gNB', value: 'gNB' },
        { label: 'RRU', value: 'RRU' },
      ],
    },
  ], [t]);

  const userFileColumns: DataTableColumn<ManagedFile & Record<string, unknown>>[] = useMemo(() => [
    { key: 'fileName', title: t('table.name'), dataIndex: 'fileName', ellipsis: true },
    { key: 'fileSize', title: t('table.description'), dataIndex: 'fileSize', width: 100, render: (val) => formatFileSize(Number(val)) },
    {
      key: 'fileType', title: t('table.type'), dataIndex: 'fileType', width: 100,
      render: (val) => <Tag color={fileTypeColorMap[String(val)] ?? 'default'}>{fileTypeLabelMap[String(val)] ?? String(val)}</Tag>,
    },
    {
      key: 'reviewStatus', title: t('table.status'), dataIndex: 'id', width: 100,
      render: (val) => {
        const status = mockReviewStatus(String(val));
        return <Tag color={reviewStatusColorMap[status]}>{t(reviewStatusLabelKeyMap[status])}</Tag>;
      },
    },
    { key: 'uploader', title: t('table.operator'), dataIndex: 'uploader', width: 100 },
    { key: 'uploadTime', title: t('table.createTime'), dataIndex: 'uploadTime', width: 160, render: (val) => new Date(String(val)).toLocaleString('zh-CN') },
    {
      key: 'actions', title: t('table.operation'), dataIndex: 'id', width: 120, fixed: 'right',
      render: (_, record) => {
        const file = record as ManagedFile;
        return (
          <Space size="small">
            <Button type="link" size="small" icon={<DownloadOutlined />}>{t('common.download')}</Button>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}
              onClick={() => deleteFiles.mutate([file.id], { onSuccess: () => void message.success(t('common.deleteSuccess')) })}>
              {t('common.delete')}
            </Button>
          </Space>
        );
      },
    },
  ], [t]);

  const firmwareColumns: DataTableColumn<FirmwareItem & Record<string, unknown>>[] = useMemo(() => [
    { key: 'fileName', title: t('table.name'), dataIndex: 'fileName', ellipsis: true },
    { key: 'fileSize', title: t('table.description'), dataIndex: 'fileSize', width: 110, render: (val) => formatFileSize(Number(val)) },
    { key: 'deviceType', title: t('device.productType'), dataIndex: 'deviceType', width: 100 },
    { key: 'version', title: t('table.version'), dataIndex: 'version', width: 200, render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span> },
    { key: 'uploadTime', title: t('table.createTime'), dataIndex: 'uploadTime', width: 160, render: (val) => new Date(String(val)).toLocaleString('zh-CN') },
    { key: 'uploader', title: t('table.operator'), dataIndex: 'uploader', width: 100 },
    {
      key: 'actions', title: t('table.operation'), dataIndex: 'id', width: 120, fixed: 'right',
      render: (_, record) => {
        const fw = record as FirmwareItem;
        return (
          <Space size="small">
            <Button type="link" size="small" icon={<DownloadOutlined />}>{t('common.download')}</Button>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}
              onClick={() => setFirmware((prev) => prev.filter((f) => f.id !== fw.id))}>
              {t('common.delete')}
            </Button>
          </Space>
        );
      },
    },
  ], [t]);

  const filteredFirmware = firmware.filter((f) => {
    if (fwFilters.keyword && !f.fileName.includes(String(fwFilters.keyword))) return false;
    if (fwFilters.deviceType && f.deviceType !== fwFilters.deviceType) return false;
    return true;
  });

  return (
    <ListPageLayout title={t('nav.file.userFiles')}>
      <Tabs
        activeKey={activeTab}
        onChange={(k) => { setActiveTab(k); setPage(1); }}
        items={[
          {
            key: 'user',
            label: t('nav.file.userFiles'),
            children: (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                <FilterBar
                  filterId="user-files-filter"
                  fields={userFileFilterFields}
                  onSearch={(vals) => { setUserFilters(vals); setPage(1); }}
                  onReset={() => { setUserFilters({}); setPage(1); }}
                  extra={
                    <Button type="primary" icon={<UploadOutlined />} onClick={() => setUploadVisible(true)}>
                      {t('common.upload')}
                    </Button>
                  }
                />
                <DataTable
                  tableId="user-files-list"
                  columns={userFileColumns}
                  dataSource={(data?.list ?? []) as (ManagedFile & Record<string, unknown>)[]}
                  loading={isLoading}
                  rowKey="id"
                  total={data?.total ?? 0}
                  pageSize={pageSize}
                  currentPage={page}
                  onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
                  onRefresh={() => void refetch()}
                  selectable
                  selectedRowKeys={selectedKeys}
                  onSelectionChange={(keys) => setSelectedKeys(keys)}
                  batchActions={[
                    {
                      key: 'delete',
                      label: t('common.delete'),
                      danger: true,
                      onClick: (keys) => {
                        deleteFiles.mutate(keys.map(String), {
                          onSuccess: () => { void message.success(t('common.deleteSuccess')); setSelectedKeys([]); },
                        });
                      },
                    },
                  ]}
                  scroll={{ x: 900 }}
                />
              </div>
            ),
          },
          {
            key: 'firmware',
            label: t('nav.software.firmware'),
            children: (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                <FilterBar
                  filterId="firmware-files-filter"
                  fields={firmwareFilterFields}
                  onSearch={(vals) => { setFwFilters(vals); setPage(1); }}
                  onReset={() => { setFwFilters({}); setPage(1); }}
                  extra={
                    <Button type="primary" icon={<PlusOutlined />}>
                      {t('common.add')}
                    </Button>
                  }
                />
                <DataTable
                  tableId="firmware-files-list"
                  columns={firmwareColumns}
                  dataSource={filteredFirmware as (FirmwareItem & Record<string, unknown>)[]}
                  loading={false}
                  rowKey="id"
                  total={filteredFirmware.length}
                  pageSize={pageSize}
                  currentPage={page}
                  onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
                  scroll={{ x: 900 }}
                />
              </div>
            ),
          },
        ]}
      />

      <Modal
        title={t('common.upload')}
        open={uploadVisible}
        onOk={() => {
          if (fileList.length === 0) { void message.warning(t('common.pleaseSelect')); return; }
          void message.success(t('status.success'));
          setUploadVisible(false);
          setFileList([]);
          void refetch();
        }}
        onCancel={() => { setUploadVisible(false); setFileList([]); }}
        width={480}
      >
        <Dragger
          fileList={fileList}
          multiple
          beforeUpload={(file: RcFile) => { setFileList((prev) => [...prev, file]); return false; }}
          onRemove={(file) => setFileList((prev) => prev.filter((f) => f.uid !== file.uid))}
        >
          <p className="ant-upload-drag-icon"><InboxOutlined /></p>
          <p className="ant-upload-text">{t('common.upload')}</p>
          <Tooltip title=".xml / .cfg / .json / .txt / .tar.gz / .zip">
            <p className="ant-upload-hint">{t('common.upload')}</p>
          </Tooltip>
        </Dragger>
      </Modal>
    </ListPageLayout>
  );
}
