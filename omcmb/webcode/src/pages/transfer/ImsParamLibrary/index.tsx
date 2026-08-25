/**
 * ImsParamLibrary — 核心网（IMS Core）文件库。
 *
 * 结构镜像 DeviceLicenseLibrary：筛选栏 + DataTable + 上传 Modal（文件类型
 * FT_ImsCore_* 下拉 + 描述 + 拖拽批量上传）。IMS_FILE_DISTRIBUTE 任务从这里选文件下发。
 * 设计：docs/design/imscore-file-transfer.md
 */
import { useMemo, useState } from 'react';
import { Button, Card, Input, Modal, Select, Space, Tag, Upload, message } from 'antd';
import {
  PlusOutlined,
  DownloadOutlined,
  DeleteOutlined,
  ReloadOutlined,
  InboxOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';
import {
  useImsParamFiles,
  useImsParamTypes,
  useImportImsParamFiles,
  useBatchDeleteImsParamFiles,
} from '@core/hooks/api/useImsParam';
import type { ImsParamFile, ImsParamTypeOption } from '@core/services/api/imsParamApi';
import { imsParamApi } from '@core/services/api/imsParamApi';
import { formatSystemTime } from '@core/utils/systemTime';

export default function ImsParamLibraryPage() {
  const t = useT();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [paramTypeFilter, setParamTypeFilter] = useState<string>();
  const [fileNameFilter, setFileNameFilter] = useState('');
  const [deviceSnFilter, setDeviceSnFilter] = useState('');
  const [uploadOpen, setUploadOpen] = useState(false);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);

  const queryParams = useMemo(
    () => ({
      page,
      pageSize,
      paramType: paramTypeFilter || undefined,
      fileName: fileNameFilter.trim() || undefined,
      deviceSn: deviceSnFilter.trim() || undefined,
    }),
    [page, pageSize, paramTypeFilter, fileNameFilter, deviceSnFilter],
  );

  const { data, isLoading, refetch } = useImsParamFiles(queryParams);
  const { data: paramTypes = [] } = useImsParamTypes();
  const batchDelete = useBatchDeleteImsParamFiles();

  // 文件库是下发源：类型下拉只列可下发类型（参数 FT1~7 / 下载鉴权 / 备份）。
  const paramTypeOptions = useMemo(
    () => paramTypes
      .filter((item: ImsParamTypeOption) => item.downloadSupported)
      .map((item: ImsParamTypeOption) => ({
        label: item.name,
        value: item.code,
      })),
    [paramTypes],
  );

  const paramTypeLabel = (code: string) => {
    const opt = paramTypes.find((item) => item.code === code);
    return opt ? opt.name : code;
  };

  const columns: DataTableColumn<ImsParamFile>[] = [
    {
      key: 'paramType',
      title: t('transfer.imsParamLib.col.paramType'),
      dataIndex: 'paramType',
      width: 180,
      render: (v) => <Tag color="geekblue">{paramTypeLabel(String(v))}</Tag>,
    },
    { key: 'fileName', title: t('transfer.imsParamLib.col.fileName'), dataIndex: 'fileName', width: 280, mono: true, copyable: true },
    {
      key: 'deviceSn',
      title: t('transfer.imsParamLib.col.deviceSn'),
      dataIndex: 'deviceSn',
      width: 170,
      mono: true,
      render: (v) => v ? String(v) : <span style={{ color: '#999' }}>—</span>,
    },
    {
      key: 'fileSize',
      title: t('transfer.imsParamLib.col.size'),
      dataIndex: 'fileSize',
      width: 100,
      render: (v) => formatBytes(v as number),
    },
    {
      key: 'description',
      title: t('transfer.imsParamLib.col.description'),
      dataIndex: 'description',
      width: 200,
      render: (v) => (v ? String(v) : <span style={{ color: '#999' }}>—</span>),
    },
    {
      key: 'uploadedBy',
      title: t('transfer.imsParamLib.col.uploadedBy'),
      dataIndex: 'uploadedBy',
      width: 120,
      render: (v) => (v ? String(v) : '—'),
    },
    {
      key: 'createdAt',
      title: t('transfer.imsParamLib.col.createdAt'),
      dataIndex: 'createdAt',
      width: 180,
      sorter: true,
      render: (v) => formatSystemTime(v as string, { format: 'YYYY-MM-DD HH:mm:ss', placeholder: '-' }),
    },
    {
      key: 'actions',
      title: t('transfer.imsParamLib.col.actions'),
      width: 160,
      fixed: 'right',
      render: (_, record) => (
        <Space>
          <Button
            type="link"
            size="small"
            icon={<DownloadOutlined />}
            onClick={() => { void downloadFile(record); }}
          >
            {t('transfer.imsParamLib.action.download')}
          </Button>
          <Button
            type="link"
            danger
            size="small"
            icon={<DeleteOutlined />}
            onClick={() => confirmDelete([record.id])}
          >
            {t('transfer.imsParamLib.action.delete')}
          </Button>
        </Space>
      ),
    },
  ];

  async function downloadFile(record: ImsParamFile) {
    try {
      await imsParamApi.download(record.id, record.fileName);
    } catch (e) {
      message.error(t('transfer.imsParamLib.msg.downloadFailed', { reason: (e as Error).message ?? t('transfer.imsParamLib.msg.tryAgainLater') }));
    }
  }

  function confirmDelete(ids: string[]) {
    Modal.confirm({
      title: t('transfer.imsParamLib.msg.deleteConfirmTitle'),
      content: t('transfer.imsParamLib.msg.deleteConfirm', { count: ids.length }),
      okType: 'danger',
      onOk: async () => {
        try {
          const res = await batchDelete.mutateAsync(ids);
          message.success(res.failed.length
            ? t('transfer.imsParamLib.msg.deletePartial', { count: res.succeeded.length, failedCount: res.failed.length })
            : t('transfer.imsParamLib.msg.deleteSuccess', { count: res.succeeded.length }));
          setSelectedKeys((prev) => prev.filter((k) => !ids.includes(String(k))));
        } catch {
          message.error(t('transfer.imsParamLib.msg.deleteFailed'));
        }
      },
    });
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16, flex: 1, minHeight: 0 }}>
      <Card size="small">
        <Space wrap>
          <Select<string>
            allowClear
            showSearch
            optionFilterProp="label"
            placeholder={t('transfer.imsParamLib.filter.paramType')}
            value={paramTypeFilter}
            onChange={(v) => setParamTypeFilter(v)}
            options={paramTypeOptions}
            style={{ width: 220 }}
          />
          <Input
            placeholder={t('transfer.imsParamLib.filter.fileName')}
            allowClear
            value={fileNameFilter}
            onChange={(e) => setFileNameFilter(e.target.value)}
            style={{ width: 220 }}
          />
          <Input
            placeholder={t('transfer.imsParamLib.filter.deviceSn')}
            allowClear
            value={deviceSnFilter}
            onChange={(e) => setDeviceSnFilter(e.target.value)}
            style={{ width: 180 }}
          />
          <Button icon={<ReloadOutlined />} onClick={() => refetch()}>{t('transfer.imsParamLib.action.refresh')}</Button>
        </Space>
      </Card>
      <DataTable<ImsParamFile>
        tableId="ims-param-library"
        columns={columns}
        dataSource={data?.items ?? []}
        loading={isLoading}
        rowKey={(r) => r.id}
        selectable
        selectedRowKeys={selectedKeys}
        onSelectionChange={setSelectedKeys}
        total={data?.total ?? 0}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, s) => {
          setPage(p);
          setPageSize(s);
        }}
        extraToolbarLeft={(
          <Space>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setUploadOpen(true)}>
              {t('transfer.imsParamLib.action.upload')}
            </Button>
            <Button
              danger
              icon={<DeleteOutlined />}
              disabled={selectedKeys.length === 0}
              onClick={() => confirmDelete(selectedKeys.map(String))}
            >
              {t('transfer.imsParamLib.action.batchDelete', { count: selectedKeys.length })}
            </Button>
          </Space>
        )}
      />
      <UploadModal
        open={uploadOpen}
        paramTypes={paramTypes}
        onClose={() => setUploadOpen(false)}
        onSuccess={() => refetch()}
      />
    </div>
  );
}

function UploadModal({
  open,
  paramTypes,
  onClose,
  onSuccess,
}: {
  open: boolean;
  paramTypes: ImsParamTypeOption[];
  onClose: () => void;
  onSuccess?: () => void;
}) {
  const t = useT();
  const [paramType, setParamType] = useState<string>();
  const [description, setDescription] = useState('');
  const [files, setFiles] = useState<File[]>([]);
  const importMutation = useImportImsParamFiles();

  const paramTypeOptions = useMemo(
    () => paramTypes.filter((item) => item.downloadSupported).map((item) => ({ label: item.name, value: item.code })),
    [paramTypes],
  );

  function handleAdd(file: File): boolean {
    // 去重提示放 setState 外：拖拽路径上外层 div.onDrop 与 Dragger 自身处理可能
    // 双重进入本函数（事件冒泡次序不定），提示只弹一次，state 更新天然幂等。
    if (files.some((f) => f.name === file.name)) {
      void message.warning(t('transfer.imsParamLib.upload.duplicate', { name: file.name }));
      return false;
    }
    setFiles((prev) => (prev.some((f) => f.name === file.name) ? prev : [...prev, file]));
    return false;
  }

  function handleClose() {
    setFiles([]);
    setDescription('');
    onClose();
  }

  async function handleSubmit() {
    if (!paramType || files.length === 0) return;
    try {
      const result = await importMutation.mutateAsync({ paramType, files, description });
      if (result.succeeded.length > 0) {
        void message.success(t('transfer.imsParamLib.upload.successMsg', { count: result.succeeded.length }));
        setFiles([]);
        onSuccess?.();
      }
      if (result.failed.length > 0) {
        void message.warning(t('transfer.imsParamLib.upload.partialFailMsg', {
          count: result.failed.length,
          first: result.failed[0]?.message ?? '',
        }));
      }
    } catch {
      void message.error(t('transfer.imsParamLib.upload.requestFailed'));
    }
  }

  return (
    <Modal
      open={open}
      title={t('transfer.imsParamLib.upload.title')}
      onCancel={handleClose}
      onOk={() => void handleSubmit()}
      okButtonProps={{ disabled: !paramType || files.length === 0 }}
      confirmLoading={importMutation.isPending}
      okText={t('transfer.imsParamLib.upload.submit', { count: files.length })}
      destroyOnHidden
    >
      <Space direction="vertical" size={12} style={{ width: '100%' }}>
        <Select<string>
          showSearch
          optionFilterProp="label"
          placeholder={t('transfer.imsParamLib.upload.paramTypeRequired')}
          value={paramType}
          onChange={(v) => setParamType(v)}
          options={paramTypeOptions}
          style={{ width: '100%' }}
        />
        <Input
          placeholder={t('transfer.imsParamLib.upload.descriptionPlaceholder')}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          maxLength={200}
        />
        {/* 拖拽上传只由 Upload.Dragger 自身处理（beforeUpload=handleAdd）。
            外层不再绑 onDrop —— Dragger 内部会自行监听 drop，外层再手动调
            handleAdd 会同文件双触发（重复提示弹两次的根因）。 */}
        <Upload.Dragger
          multiple
          beforeUpload={handleAdd}
          showUploadList
          fileList={files.map((f) => ({ uid: f.name, name: f.name, status: 'done' as const }))}
          onRemove={(file) => setFiles((prev) => prev.filter((f) => f.name !== file.name))}
          height={120}
        >
            <Space size={10} align="center" style={{ width: '100%', justifyContent: 'center' }}>
              <InboxOutlined style={{ fontSize: 22, color: '#1677ff' }} />
              <div style={{ textAlign: 'left', fontSize: 12, color: 'rgba(0,0,0,0.45)' }}>
                {t('transfer.imsParamLib.upload.dropHint')}
              </div>
            </Space>
          </Upload.Dragger>
      </Space>
    </Modal>
  );
}

function formatBytes(n: number): string {
  if (!n || n < 0) return '—';
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(2)} MB`;
}
