import { useState, useMemo } from 'react';
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
} from '@ant-design/icons';
import type { UploadFile, RcFile } from 'antd/es/upload';
import type { MenuProps } from 'antd';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';

const { Dragger } = Upload;
const { TextArea } = Input;

// 文件类型枚举
type FileType = 'upgrade' | 'ca' | 'fpga' | 'ap';
// 版本类型枚举
type VersionType = 'all' | 'none' | 'beta';

interface FirmwareFile {
  id: string;
  version: string;
  product?: string; // AP类型没有产品类型
  size: number;
  toWho?: VersionType;
  uploadTime: string;
  recommend: boolean; // 是否推荐
  fileName: string;
  description?: string;
}

// Mock 数据
const mockUpgradeFiles: FirmwareFile[] = [
  { id: '1', version: 'V1.3.0', product: 'PM-B4860,QAFA,QATA', size: 512 * 1024 * 1024, toWho: 'all', uploadTime: '2026-03-25 10:00:00', recommend: true, fileName: 'eNB_V1.3.0_full.tar.gz', description: 'eNB主版本升级包' },
  { id: '2', version: 'V2.1.0', product: 'BaiBNX,BaiBNQ', size: 768 * 1024 * 1024, toWho: 'all', uploadTime: '2026-03-24 14:30:00', recommend: true, fileName: 'gNB_V2.1.0_full.tar.gz', description: 'gNB 5G版本升级包' },
  { id: '3', version: 'V1.2.5', product: 'PM-B4860', size: 128 * 1024 * 1024, toWho: 'none', uploadTime: '2026-03-20 09:15:00', recommend: false, fileName: 'PM-B4860_V1.2.5.img', description: '测试版本' },
];

const mockCaFiles: FirmwareFile[] = [
  { id: '4', version: 'CA-V1.0.2', product: 'PM-B4860', size: 32 * 1024 * 1024, toWho: 'all', uploadTime: '2026-03-22 11:00:00', recommend: false, fileName: 'CA_V1.0.2.patch', description: 'CA补丁包' },
  { id: '5', version: 'CA-V1.0.1', product: 'QAFA', size: 28 * 1024 * 1024, toWho: 'all', uploadTime: '2026-03-18 16:20:00', recommend: true, fileName: 'CA_V1.0.1.patch', description: 'CA补丁包' },
];

const mockFpgaFiles: FirmwareFile[] = [
  { id: '6', version: 'FPGA-V2.0', product: 'BaiBNX', size: 16 * 1024 * 1024, toWho: 'all', uploadTime: '2026-03-21 08:45:00', recommend: true, fileName: 'FPGA_V2.0.img', description: 'FPGA固件升级' },
];

const mockApFiles: FirmwareFile[] = [
  { id: '7', version: 'AP-V3.0', size: 24 * 1024 * 1024, toWho: 'all', uploadTime: '2026-03-23 13:30:00', recommend: true, fileName: 'AP_V3.0.img', description: 'AP无线升级包' },
  { id: '8', version: 'AP-V2.5', size: 20 * 1024 * 1024, toWho: 'beta', uploadTime: '2026-03-15 10:00:00', recommend: false, fileName: 'AP_V2.5.img', description: 'AP测试版本' },
];

// 产品类型列表
const productTypeOptions = [
  { label: 'PM-B4860', value: 'PM-B4860' },
  { label: 'QAFA', value: 'QAFA' },
  { label: 'QATA', value: 'QATA' },
  { label: 'QAFB', value: 'QAFB' },
  { label: 'RTD', value: 'RTD' },
  { label: 'BaiBNX', value: 'BaiBNX' },
  { label: 'BaiBNQ', value: 'BaiBNQ' },
];

function formatFileSize(bytes: number): string {
  if (bytes >= 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
  if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
  return `${(bytes / 1024).toFixed(2)} KB`;
}

export default function FirmwareUpload() {
  const t = useT();
  const [form] = Form.useForm();
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [uploading, setUploading] = useState(false);

  // 文件类型状态
  const [fileType, setFileType] = useState<FileType>('upgrade');
  // 搜索条件
  const [filters, setFilters] = useState<Record<string, unknown>>({});
  // 导入文件抽屉
  const [importDrawerVisible, setImportDrawerVisible] = useState(false);
  const [importMode, setImportMode] = useState<'add' | 'view' | 'modify'>('add');
  const [selectedFile, setSelectedFile] = useState<FirmwareFile | null>(null);
  // 删除确认
  const [deleteFile, setDeleteFile] = useState<FirmwareFile | null>(null);


  // 获取当前文件类型的数据
  const fileData = useMemo(() => {
    const dataMap: Record<FileType, FirmwareFile[]> = {
      upgrade: mockUpgradeFiles,
      ca: mockCaFiles,
      fpga: mockFpgaFiles,
      ap: mockApFiles,
    };
    return dataMap[fileType];
  }, [fileType]);

  // 过滤后的数据
  const filteredData = useMemo(() => {
    return fileData.filter((row) => {
      if (filters.keyword && typeof filters.keyword === 'string') {
        const keyword = filters.keyword.toLowerCase();
        if (!row.version.toLowerCase().includes(keyword)) {
          return false;
        }
      }
      return true;
    });
  }, [fileData, filters]);

  // 搜索字段
  const filterFields: FilterField[] = useMemo(() => [
    { name: 'keyword', label: t('software.firmware.version'), type: 'input', placeholder: t('software.firmware.inputVersion') },
  ], [t]);

  // 打开导入抽屉
  const handleOpenImportDrawer = (mode: 'add' | 'view' | 'modify', file?: FirmwareFile) => {
    setImportMode(mode);
    setSelectedFile(file ?? null);
    if (file) {
      form.setFieldsValue({
        product: file.product?.split(',') ?? [],
        version: file.version,
        toWho: file.toWho ?? 'all',
        recommend: file.recommend ? '1' : '0',
        description: file.description ?? '',
      });
    } else {
      form.resetFields();
    }
    setFileList([]);
    setImportDrawerVisible(true);
  };

  // 关闭导入抽屉
  const handleCloseImportDrawer = () => {
    setImportDrawerVisible(false);
    setSelectedFile(null);
    form.resetFields();
    setFileList([]);
  };

  // 提交导入
  const handleImportSubmit = () => {
    if (importMode === 'view') {
      handleCloseImportDrawer();
      return;
    }

    form.validateFields().then(() => {
      if (fileList.length === 0 && importMode === 'add') {
        void message.warning(t('software.firmware.selectFile'));
        return;
      }

      setUploading(true);
      setUploadProgress(0);

      const timer = setInterval(() => {
        setUploadProgress((prev) => {
          if (prev >= 100) {
            clearInterval(timer);
            setUploading(false);
            void message.success(importMode === 'add' ? t('software.firmware.importSuccess') : t('software.firmware.modifySuccess'));
            handleCloseImportDrawer();
            return 100;
          }
          return prev + 10;
        });
      }, 150);
    });
  };

  // 删除文件
  const handleDeleteFile = () => {
    if (deleteFile) {
      void message.success(t('software.firmware.deleted', { name: deleteFile.fileName }));
      setDeleteFile(null);
    }
  };

  // 切换推荐状态
  const handleToggleRecommend = (file: FirmwareFile) => {
    void message.success(file.recommend ? t('software.firmware.recommendUnset', { version: file.version }) : t('software.firmware.recommendSet', { version: file.version }));
  };

  // 表格列定义
  const columns: DataTableColumn<FirmwareFile>[] = [
    {
      key: 'operation',
      title: t('common.operation'),
      width: 100,
      fixed: 'right',
      render: (_: unknown, record: FirmwareFile) => {
        const items: MenuProps['items'] = [
          {
            key: 'download',
            label: t('common.download'),
            icon: <DownloadOutlined />,
            onClick: () => void message.success(t('software.firmware.deleted', { name: record.fileName })),
          },
          {
            key: 'modify',
            label: t('common.edit'),
            icon: <EditOutlined />,
            onClick: () => handleOpenImportDrawer('modify', record),
          },
          {
            key: 'delete',
            label: t('common.delete'),
            icon: <DeleteOutlined />,
            danger: true,
            onClick: () => setDeleteFile(record),
          },
          { type: 'divider' },
          {
            key: 'recommend',
            label: record.recommend ? t('software.firmware.cancelRecommend') : t('software.firmware.setRecommend'),
            icon: record.recommend ? <StarFilled style={{ color: '#faad14' }} /> : <StarOutlined />,
            onClick: () => handleToggleRecommend(record),
          },
        ];
        return (
          <Space size={4}>
            <Button type="link" size="small" onClick={() => handleOpenImportDrawer('view', record)}>{t('common.info')}</Button>
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
      dataIndex: 'version',
      width: 300,
      ellipsis: true,
      render: (val: string, record: FirmwareFile) => (
        <Space>
          <Typography.Text style={{ fontFamily: 'monospace', fontSize: 13 }}>{val}</Typography.Text>
          {record.recommend && <StarFilled style={{ color: '#faad14' }} />}
        </Space>
      ),
    },
    {
      key: 'product',
      title: t('software.firmware.productType'),
      dataIndex: 'product',
      width: 250,
      ellipsis: true,
      render: (val: string | undefined) => val ?? '-',
    },
    {
      key: 'size',
      title: t('table.fileSize') ?? '文件大小',
      dataIndex: 'size',
      width: 120,
      render: (val: number) => formatFileSize(val),
    },
    {
      key: 'uploadTime',
      title: t('table.uploadTime') ?? '上传时间',
      dataIndex: 'uploadTime',
      width: 180,
    },
  ];

  // 获取当前文件类型的中文名称
  const fileTypeName = useMemo(() => {
    const nameMap: Record<FileType, string> = {
      upgrade: 'IMAGE',
      ca: t('software.firmware.caVersion'),
      fpga: t('software.firmware.fpgaFile'),
      ap: t('software.firmware.apFile'),
    };
    return nameMap[fileType];
  }, [fileType, t]);

  return (
    <ListPageLayout title={t('software.firmware.title')}>
      {/* 文件类型选择 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
        <Radio.Group
          value={fileType}
          onChange={(e) => {
            setFileType(e.target.value);
            setFilters({});
          }}
          optionType="button"
          buttonStyle="solid"
        >
          <Radio.Button value="upgrade">IMAGE</Radio.Button>
          <Radio.Button value="ca">{t('software.firmware.caVersion')}</Radio.Button>
          <Radio.Button value="fpga">{t('software.firmware.fpgaFile')}</Radio.Button>
          <Radio.Button value="ap">{t('software.firmware.apFile')}</Radio.Button>
        </Radio.Group>
        <Button type="primary" icon={<InboxOutlined />} onClick={() => handleOpenImportDrawer('add')}>
          {t('software.firmware.importFile')}
        </Button>
      </div>

      {/* 搜索表单 */}
      <FilterBar
        filterId="firmware-filter"
        fields={filterFields}
        onSearch={(vals) => setFilters(vals)}
        onReset={() => setFilters({})}
      />

      {/* 文件列表 */}
      <Card
        size="small"
        bordered
        style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
        styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
      >
        <DataTable<FirmwareFile>
          tableId="firmware-list"
          columns={columns}
          dataSource={filteredData}
          rowKey="id"
          total={filteredData.length}
          currentPage={1}
          pageSize={20}
          onPageChange={() => {}}
          scroll={{ x: 'max-content', y: 'calc(100vh - 400px)' }}
          showRowNumber
          rowNumberTitle={t('table.rowNumber')}
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
                loading={uploading}
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
          {/* 产品类型标识 - AP类型不显示 */}
          {fileType !== 'ap' && (
            <Form.Item
              name="product"
              label={t('software.firmware.productType')}
              rules={[{ required: true, message: t('software.firmware.selectProductType') }]}
            >
              {fileType === 'upgrade' ? (
                <Select
                  mode="multiple"
                  maxTagCount="responsive"
                  placeholder={t('software.firmware.selectProductType')}
                  options={productTypeOptions}
                />
              ) : (
                <Select
                  placeholder={t('software.firmware.selectProductType')}
                  options={productTypeOptions}
                />
              )}
            </Form.Item>
          )}

          {/* 文件名 */}
          <Form.Item
            label={
              <Space>
                {t('software.firmware.fileName')}
                {importMode === 'add' && (
                  <span style={{ color: '#999', fontSize: 12 }}>
                    ({t('software.firmware.supportFormat', { format: fileType === 'upgrade' ? 'IMG / EXT' : fileType === 'ca' ? 'Patch' : 'IMG' })})
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
                    setFileList([file]);
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
                {uploading && (
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
        okButtonProps={{ danger: true }}
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
                <strong>{t('software.firmware.versionLabel')}</strong>{deleteFile?.version}<br />
                <strong>{t('software.firmware.fileNameLabel')}</strong>{deleteFile?.fileName}<br />
                <strong>{t('software.firmware.fileSizeLabel')}</strong>{deleteFile ? formatFileSize(deleteFile.size) : '-'}
              </p>
            </div>
          }
        />
      </Modal>
    </ListPageLayout>
  );
}
