import { useState, useMemo, useCallback } from 'react';
import {
  Button,
  Card,
  Form,
  Input,
  Select,
  Progress,
  Space,
  Upload,
  message,
  Typography,
  Radio,
  Drawer,
  Modal,
  Alert,
  Dropdown,
} from 'antd';
import {
  InboxOutlined,
  DeleteOutlined,
  DownloadOutlined,
  StarOutlined,
  StarFilled,
  MoreOutlined,
  EditOutlined,
  WarningOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons';
import { useSearchParams } from 'react-router-dom';
import type { UploadFile, RcFile } from 'antd/es/upload';
import type { MenuProps } from 'antd';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';
import {
  useSoftwareVersions,
  useUploadFirmware,
  useDeleteSoftwareVersions,
  useToggleRecommend,
  useDownloadFirmware,
  useUpdateFirmware,
} from '@core/hooks/api/useSoftware';
import { useProductClasses } from '@core/hooks/api/useDevices';
import { useBatchDownloadWithMessage } from '@/hooks/useBatchDownloadWithMessage';
import type { SoftwareVersion } from '@core/mock/data/software';

const { Dragger } = Upload;
const { TextArea } = Input;

// 产品类型列表 — 从后端动态获取，此常量仅作为 fallback
const fallbackProductClassOptions = [
  { label: 'PM-B4860', value: 'PM-B4860' },
  { label: 'QAFA', value: 'QAFA' },
  { label: 'QAFB', value: 'QAFB' },
  { label: 'FAP/BU1810', value: 'FAP/BU1810' },
];

// 文件类型枚举
type FileType = 'upgrade' | 'patch' | 'fpga';

// 文件类型 tab 到后端 fileType 的映射
const fileTypeParamMap: Record<FileType, 0 | 1 | 6> = {
  upgrade: 0,
  patch: 1,
  fpga: 6,
};

function formatFileSize(bytes: number): string {
  if (bytes >= 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
  if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
  return `${(bytes / 1024).toFixed(2)} KB`;
}

interface FirmwareUploadProps {
  /** 嵌入 Tab/弹窗使用时隐藏页面级标题，避免与父级标题重复 */
  embedded?: boolean;
}

export default function FirmwareUpload({ embedded = false }: FirmwareUploadProps = {}) {
  const t = useT();
  const [form] = Form.useForm();
  // 当 URL 带 ?return=ufte 时，认为是 UFTE 任务创建弹窗里"维护升级文件"开的新 tab。
  // 顶部展示提示 + "完成并关闭窗口"按钮，关掉新 tab 自动回原弹窗（焦点切换触发
  // React Query refetch，固件列表自动更新）。
  const [searchParams] = useSearchParams();
  const fromUFTE = searchParams.get('return') === 'ufte';
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [uploadProgress, setUploadProgress] = useState(0);

  // Dynamic product type options from API
  const { data: productClassesData } = useProductClasses();
  const productClassOptions = useMemo(() => {
    if (productClassesData && productClassesData.length > 0) {
      return productClassesData.map((c) => ({ label: c, value: c }));
    }
    return fallbackProductClassOptions;
  }, [productClassesData]);

  // 文件类型状态
  const [fileType, setFileType] = useState<FileType>('upgrade');
  // 分页
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  // 搜索条件
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  // 导入文件抽屉
  const [importDrawerVisible, setImportDrawerVisible] = useState(false);
  const [importMode, setImportMode] = useState<'add' | 'view' | 'modify'>('add');
  const [selectedFile, setSelectedFile] = useState<SoftwareVersion | null>(null);
  // 删除确认
  const [deleteFile, setDeleteFile] = useState<SoftwareVersion | null>(null);

  // API hooks
  const { data, isLoading, refetch } = useSoftwareVersions({
    fileType: fileTypeParamMap[fileType],
    page,
    pageSize,
  });

  const uploadMutation = useUploadFirmware();
  const deleteMutation = useDeleteSoftwareVersions();
  const toggleRecommendMutation = useToggleRecommend();
  const downloadMutation = useDownloadFirmware();
  const updateMutation = useUpdateFirmware();

  // 批量下载 — 同步流式 POST,axios 拿 blob 自动触发浏览器下载。
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);
  const bundle = useBatchDownloadWithMessage();
  const handleBatchDownload = () => {
    if (selectedKeys.length === 0) {
      void message.warning(t('bundle.selectFiles'));
      return;
    }
    bundle.trigger({ module: 'firmware', targets: selectedKeys.map(String) });
  };

  // 获取列表数据
  const tableData = useMemo(() => {
    const items = data?.items ?? [];
    if (filters.keyword && typeof filters.keyword === 'string') {
      const keyword = (filters.keyword as string).toLowerCase();
      return items.filter(
        (item: SoftwareVersion) =>
          item.versionCode?.toLowerCase().includes(keyword) ||
          item.versionName?.toLowerCase().includes(keyword),
      );
    }
    return items;
  }, [data?.items, filters.keyword]);

  // 搜索字段
  const filterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('software.firmware.version'), type: 'input', placeholder: t('software.firmware.inputVersion') },
  ], [t]);

  // 打开导入抽屉
  const handleOpenImportDrawer = useCallback((mode: 'add' | 'view' | 'modify', file?: SoftwareVersion) => {
    setImportMode(mode);
    setSelectedFile(file ?? null);
    if (file) {
      form.setFieldsValue({
        product: file.deviceType?.split(',') ?? [],
        version: file.versionCode,
        recommend: file.recommend ? '1' : '0',
        description: file.description ?? '',
      });
    } else {
      form.resetFields();
    }
    setFileList([]);
    setImportDrawerVisible(true);
  }, [form]);

  // 关闭导入抽屉
  const handleCloseImportDrawer = useCallback(() => {
    setImportDrawerVisible(false);
    setSelectedFile(null);
    form.resetFields();
    setFileList([]);
    setUploadProgress(0);
  }, [form]);

  // 提交导入
  const handleImportSubmit = useCallback(() => {
    if (importMode === 'view') {
      handleCloseImportDrawer();
      return;
    }

    form.validateFields().then((values) => {
      if (importMode === 'modify') {
        // 编辑模式：只更新元数据
        if (!selectedFile) return;
        updateMutation.mutate(
          {
            id: selectedFile.id,
            metadata: {
              productClass: Array.isArray(values.product) ? values.product.join(',') : values.product,
              version: values.version ?? '',
              recommend: values.recommend === '1',
              description: values.description ?? '',
            },
          },
          {
            onSuccess: () => {
              void message.success(t('software.firmware.modifySuccess'));
              handleCloseImportDrawer();
            },
            onError: () => {
              void message.error(t('software.firmware.modifyFailed'));
            },
          },
        );
        return;
      }

      // 新增模式：上传文件
      if (fileList.length === 0) {
        void message.warning(t('software.firmware.selectFile'));
        return;
      }

      const rawFile = fileList[0]?.originFileObj as File | undefined;
      if (!rawFile) {
        void message.warning(t('software.firmware.selectFile'));
        return;
      }

      setUploadProgress(0);
      uploadMutation.mutate(
        {
          file: rawFile,
          metadata: {
            version: values.version ?? '',
            productClass: Array.isArray(values.product) ? values.product.join(',') : values.product,
            releaseNotes: values.description ?? '',
            fileType: fileTypeParamMap[fileType],
            recommend: values.recommend === '1',
            description: values.description ?? '',
          },
        },
        {
          onSuccess: () => {
            void message.success(t('software.firmware.importSuccess'));
            handleCloseImportDrawer();
          },
          onError: () => {
            void message.error(t('software.firmware.importFailed'));
            setUploadProgress(0);
          },
        },
      );
    });
  }, [importMode, selectedFile, fileList, fileType, uploadMutation, updateMutation, form, t, handleCloseImportDrawer]);

  // 删除文件
  const handleDeleteFile = useCallback(() => {
    if (deleteFile) {
      deleteMutation.mutate([deleteFile.id], {
        onSuccess: () => {
          void message.success(t('software.firmware.deleted', { name: deleteFile.versionCode }));
          setDeleteFile(null);
        },
      });
    }
  }, [deleteFile, deleteMutation, t]);

  // 切换推荐状态
  const handleToggleRecommend = useCallback((file: SoftwareVersion) => {
    toggleRecommendMutation.mutate(file.id, {
      onSuccess: () => {
        void message.success(
          file.recommend
            ? t('software.firmware.recommendUnset', { version: file.versionCode })
            : t('software.firmware.recommendSet', { version: file.versionCode }),
        );
      },
    });
  }, [toggleRecommendMutation, t]);

  // 表格列定义
  const columns: DataTableColumn<SoftwareVersion & Record<string, unknown>>[] = useMemo(() => [
    {
      key: 'operation',
      title: t('common.operation'),
      width: 100,
      fixed: 'right',
      render: (_: unknown, record: SoftwareVersion & Record<string, unknown>) => {
        const sv = record as SoftwareVersion;
        const items: MenuProps['items'] = [
          {
            key: 'download',
            label: t('common.download'),
            icon: <DownloadOutlined />,
            onClick: () => {
              downloadMutation.mutate({ id: sv.id, fileName: sv.fileName });
            },
          },
          {
            key: 'modify',
            label: t('common.edit'),
            icon: <EditOutlined />,
            onClick: () => handleOpenImportDrawer('modify', sv),
          },
          {
            key: 'delete',
            label: t('common.delete'),
            icon: <DeleteOutlined />,
            danger: true,
            onClick: () => setDeleteFile(sv),
          },
          { type: 'divider' },
          {
            key: 'recommend',
            label: sv.recommend ? t('software.firmware.cancelRecommend') : t('software.firmware.setRecommend'),
            icon: sv.recommend ? <StarFilled style={{ color: '#faad14' }} /> : <StarOutlined />,
            onClick: () => handleToggleRecommend(sv),
          },
        ];
        return (
          <Space size={4}>
            <Button type="link" size="small" onClick={() => handleOpenImportDrawer('view', sv)}>{t('common.info')}</Button>
            <Dropdown menu={{ items }} trigger={['click']}>
              <Button type="text" size="small" icon={<MoreOutlined />} onClick={(e) => e.stopPropagation()} />
            </Dropdown>
          </Space>
        );
      },
    },
    {
      key: 'version',
      title: t('software.firmware.version'),
      dataIndex: 'versionCode',
      width: 300,
      ellipsis: true,
      render: (val: unknown, record: SoftwareVersion & Record<string, unknown>) => {
        const sv = record as SoftwareVersion;
        return (
          <Space>
            <Typography.Text style={{ fontFamily: 'monospace', fontSize: 13 }}>{String(val)}</Typography.Text>
            {sv.recommend && <StarFilled style={{ color: '#faad14' }} />}
          </Space>
        );
      },
    },
    {
      key: 'product',
      title: t('software.firmware.productClass'),
      dataIndex: 'deviceType',
      width: 250,
      ellipsis: true,
      render: (val: unknown) => val ? String(val) : '-',
    },
    {
      key: 'size',
      title: t('table.fileSize') ?? t('transfer.fileLib.firmware.col.fileSize'),
      dataIndex: 'fileSize',
      width: 120,
      render: (val: unknown) => formatFileSize(Number(val)),
    },
    {
      key: 'uploadTime',
      title: t('table.uploadTime') ?? t('transfer.fileLib.firmware.col.uploadTime'),
      dataIndex: 'releaseDate',
      width: 180,
      render: (val: unknown) => val ? String(val) : '-',
    },
  ], [t, handleOpenImportDrawer, handleToggleRecommend]);

  // 获取当前文件类型的中文名称
  const fileTypeName = useMemo(() => {
    const nameMap: Record<FileType, string> = {
      upgrade: 'IMAGE',
      patch: 'PATCH',
      fpga: t('software.firmware.fpgaFile'),
    };
    return nameMap[fileType];
  }, [fileType, t]);

  const handleFileTypeChange = useCallback((newType: FileType) => {
    setFileType(newType);
    setFilters({});
    setPage(1);
  }, []);

  return (
    <ListPageLayout title={embedded ? undefined : t('software.firmware.title')}>
      {/* embedded 模式（嵌入到「文件管理」tab）下，FileManagement 父组件已经
          在页面顶部统一显示「您来自任务创建 / 完成并关闭」Alert，这里就不重复显示。
          只在直链 /software/firmware?return=ufte 的旧入口才走这条 Alert。 */}
      {fromUFTE && !embedded ? (
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 12 }}
          message={t('transfer.fileLib.firmware.fromUfteTitle')}
          description={t('transfer.fileLib.firmware.fromUfteDesc')}
          action={(
            <Button
              size="small"
              type="primary"
              icon={<CloseCircleOutlined />}
              onClick={() => {
                // self-close — 由 window.open 打开的同源窗口允许自闭
                try {
                  window.close();
                } catch {
                  void message.info(t('transfer.fileLib.firmware.closeBlocked'));
                }
              }}
            >
              {t('transfer.fileLib.firmware.close')}
            </Button>
          )}
        />
      ) : null}

      {/* 文件类型选择 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
        <Radio.Group
          value={fileType}
          onChange={(e) => handleFileTypeChange(e.target.value)}
          optionType="button"
          buttonStyle="solid"
        >
          <Radio.Button value="upgrade">IMAGE</Radio.Button>
          <Radio.Button value="patch">PATCH</Radio.Button>
          <Radio.Button value="fpga">{t('software.firmware.fpgaFile')}</Radio.Button>
        </Radio.Group>
        <Button type="primary" icon={<InboxOutlined />} onClick={() => handleOpenImportDrawer('add')}>
          {t('software.firmware.importFile')}
        </Button>
      </div>

      {/* 搜索表单 */}
      <FilterBar
        filterId="firmware-filter"
        fields={filterFields}
        onSearch={(vals) => { setFilters(vals); setPage(1); }}
        onReset={() => { setFilters({}); setPage(1); }}
      />

      {/* 文件列表 */}
      <Card
        size="small"
        bordered
        style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
        styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
      >
        <DataTable<SoftwareVersion & Record<string, unknown>>
          tableId="firmware-list"
          columns={columns}
          dataSource={tableData as (SoftwareVersion & Record<string, unknown>)[]}
          loading={isLoading}
          rowKey="id"
          total={data?.total ?? 0}
          currentPage={page}
          pageSize={pageSize}
          onPageChange={(p, s) => { setPage(p); setPageSize(s); }}
          onRefresh={() => void refetch()}
          selectable
          selectedRowKeys={selectedKeys}
          onSelectionChange={(keys) => setSelectedKeys(keys)}
          batchActions={[{
            key: 'batch-download',
            label: t('bundle.batchDownload'),
            icon: <DownloadOutlined />,
            disabled: bundle.isPending,
            onClick: () => handleBatchDownload(),
          }]}
          scroll={{ x: 'max-content', y: 'calc(100vh - 400px)' }}
        />
      </Card>

      {/* 导入/查看/修改文件抽屉 */}
      <Drawer
        title={
          importMode === 'add' ? `${t('software.firmware.importFile')}${fileTypeName}` :
          importMode === 'view' ? t('software.firmware.fileInfo') : t('software.firmware.modifyFile')
        }
        placement="right"
        width={400}
        open={importDrawerVisible}
        onClose={handleCloseImportDrawer}
        footer={
          importMode === 'view' ? null : (
            <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
              <Button onClick={handleCloseImportDrawer}>{t('common.cancel')}</Button>
              <Button
                type="primary"
                onClick={handleImportSubmit}
                loading={uploadMutation.isPending}
              >
                {t('common.confirm')}
              </Button>
            </Space>
          )
        }
      >
        <Form
          form={form}
          layout="vertical"
          size="small"
          disabled={importMode === 'view'}
        >
          {/* 产品类型标识 */}
          <Form.Item
              name="product"
              label={t('software.firmware.productClass')}
              rules={[{ required: true, message: t('software.firmware.selectProductClass') }]}
            >
              {fileType === 'upgrade' ? (
                <Select
                  mode="multiple"
                  maxTagCount="responsive"
                  placeholder={t('software.firmware.selectProductClass')}
                  options={productClassOptions}
                />
              ) : (
                <Select
                  placeholder={t('software.firmware.selectProductClass')}
                  options={productClassOptions}
                />
              )}
            </Form.Item>

          {/* 文件名 */}
          <Form.Item
            label={
              <Space>
                {t('software.firmware.fileName')}
                {importMode === 'add' && (
                  <span style={{ color: '#999', fontSize: 12 }}>
                    ({t('software.firmware.supportFormat', { format: fileType === 'upgrade' ? 'IMG / EXT' : fileType === 'patch' ? 'Patch' : 'IMG' })})
                  </span>
                )}
              </Space>
            }
            required={importMode === 'add'}
          >
            {importMode === 'add' ? (
              <>
                <Dragger
                  fileList={fileList}
                  beforeUpload={(file: RcFile) => {
                    setFileList([{
                      uid: file.uid || '-1',
                      name: file.name,
                      status: 'done',
                      originFileObj: file,
                    }]);
                    return false;
                  }}
                  onRemove={() => setFileList([])}
                  maxCount={1}
                >
                  <p className="ant-upload-drag-icon">
                    <InboxOutlined />
                  </p>
                  <p className="ant-upload-text">{t('software.firmware.clickOrDrag')}</p>
                </Dragger>
                {uploadMutation.isPending && (
                  <Progress
                    percent={uploadProgress}
                    status={uploadProgress < 100 ? 'active' : 'success'}
                    style={{ marginTop: 12 }}
                  />
                )}
              </>
            ) : (
              <Input value={selectedFile?.fileName} disabled />
            )}
          </Form.Item>

          {/* 版本 */}
          <Form.Item
            name="version"
            label={t('software.firmware.version')}
            rules={[
              { required: true, message: t('software.firmware.inputVersion') },
              { max: 45, message: t('software.firmware.versionMaxLen') },
            ]}
          >
            <Input placeholder={t('software.firmware.inputVersion')} maxLength={45} />
          </Form.Item>

          {/* 推荐 */}
          <Form.Item name="recommend" label={t('software.firmware.recommend')}>
            <Select
              placeholder={t('common.pleaseSelect')}
              options={[
                { label: t('common.yes'), value: '1' },
                { label: t('common.no'), value: '0' },
              ]}
            />
          </Form.Item>

          {/* 描述 */}
          <Form.Item name="description" label={t('software.firmware.description')}>
            <TextArea rows={3} placeholder={t('software.firmware.inputDescription')} />
          </Form.Item>
        </Form>
      </Drawer>

      {/* 删除确认弹窗 */}
      <Modal
        title={t('software.firmware.confirmDelete')}
        open={!!deleteFile}
        onCancel={() => setDeleteFile(null)}
        onOk={handleDeleteFile}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        okButtonProps={{ danger: true, loading: deleteMutation.isPending }}
      >
        <Alert
          type="warning"
          showIcon
          icon={<WarningOutlined />}
          message={
            <div>
              <p style={{ marginBottom: 8 }}>
                {t('software.firmware.confirmDeleteMsg')}
              </p>
              <p style={{ marginBottom: 0 }}>
                <strong>{t('software.firmware.versionLabel')}</strong>{deleteFile?.versionCode}<br />
                <strong>{t('software.firmware.fileNameLabel')}</strong>{deleteFile?.fileName}<br />
                <strong>{t('software.firmware.fileSizeLabel')}</strong>{deleteFile ? formatFileSize(deleteFile.fileSize) : '-'}
              </p>
            </div>
          }
        />
      </Modal>
    </ListPageLayout>
  );
}
