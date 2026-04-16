import { useState, useMemo } from 'react';
import {
  Button,
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
  InfoCircleOutlined,
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
    { name: 'keyword', label: '版本', type: 'input', placeholder: '请输入版本号' },
  ], []);

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
        void message.warning('请选择文件');
        return;
      }

      setUploading(true);
      setUploadProgress(0);

      const timer = setInterval(() => {
        setUploadProgress((prev) => {
          if (prev >= 100) {
            clearInterval(timer);
            setUploading(false);
            void message.success(importMode === 'add' ? '文件导入成功' : '文件修改成功');
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
      void message.success(`已删除文件: ${deleteFile.fileName}`);
      setDeleteFile(null);
    }
  };

  // 切换推荐状态
  const handleToggleRecommend = (file: FirmwareFile) => {
    void message.success(`已${file.recommend ? '取消' : '设置'}推荐: ${file.version}`);
  };

  // 表格列定义
  const columns: DataTableColumn<FirmwareFile>[] = [
    {
      key: 'operation',
      title: '操作',
      width: 100,
      fixed: 'right',
      render: (_: unknown, record: FirmwareFile) => {
        const items: MenuProps['items'] = [
          {
            key: 'download',
            label: '下载',
            icon: <DownloadOutlined />,
            onClick: () => void message.success(`下载文件: ${record.fileName}`),
          },
          {
            key: 'modify',
            label: '修改',
            icon: <EditOutlined />,
            onClick: () => handleOpenImportDrawer('modify', record),
          },
          {
            key: 'delete',
            label: '删除',
            icon: <DeleteOutlined />,
            danger: true,
            onClick: () => setDeleteFile(record),
          },
          { type: 'divider' },
          {
            key: 'recommend',
            label: record.recommend ? '取消推荐' : '设为推荐',
            icon: record.recommend ? <StarFilled style={{ color: '#faad14' }} /> : <StarOutlined />,
            onClick: () => handleToggleRecommend(record),
          },
        ];
        return (
          <Space size={4}>
            <Button type="link" size="small" onClick={() => handleOpenImportDrawer('view', record)}>信息</Button>
            <Dropdown menu={{ items }} trigger={['click']}>
              <Button type="text" size="small" icon={<MoreOutlined />} onClick={(e) => e.stopPropagation()} />
            </Dropdown>
          </Space>
        );
      },
    },
    {
      key: 'version',
      title: '版本',
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
      title: '产品类型标识',
      dataIndex: 'product',
      width: 250,
      ellipsis: true,
      render: (val: string | undefined) => val ?? '-',
    },
    {
      key: 'size',
      title: '文件大小',
      dataIndex: 'size',
      width: 120,
      render: (val: number) => formatFileSize(val),
    },
    {
      key: 'uploadTime',
      title: '上传时间',
      dataIndex: 'uploadTime',
      width: 180,
    },
  ];

  // 获取当前文件类型的中文名称
  const fileTypeName = useMemo(() => {
    const nameMap: Record<FileType, string> = {
      upgrade: 'IMAGE',
      ca: 'CA版本',
      fpga: 'FPGA升级文件',
      ap: 'AP升级文件',
    };
    return nameMap[fileType];
  }, [fileType]);

  return (
    <ListPageLayout title="升级文件管理">
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
          <Radio.Button value="ca">CA版本</Radio.Button>
          <Radio.Button value="fpga">FPGA升级文件</Radio.Button>
          <Radio.Button value="ap">AP升级文件</Radio.Button>
        </Radio.Group>
        <Button type="primary" icon={<InboxOutlined />} onClick={() => handleOpenImportDrawer('add')}>
          导入文件
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
        rowNumberTitle="序号"
      />

      {/* 导入/查看/修改文件抽屉 */}
      <Drawer
        title={
          importMode === 'add' ? `导入${fileTypeName}` :
          importMode === 'view' ? '文件信息' : '修改文件'
        }
        placement="right"
        width={400}
        open={importDrawerVisible}
        onClose={handleCloseImportDrawer}
        footer={
          importMode === 'view' ? null : (
            <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
              <Button onClick={handleCloseImportDrawer}>取消</Button>
              <Button
                type="primary"
                onClick={handleImportSubmit}
                loading={uploading}
              >
                确定
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
              label="产品类型标识"
              rules={[{ required: true, message: '请选择产品类型' }]}
            >
              {fileType === 'upgrade' ? (
                <Select
                  mode="multiple"
                  maxTagCount="responsive"
                  placeholder="请选择产品类型"
                  options={productTypeOptions}
                />
              ) : (
                <Select
                  placeholder="请选择产品类型"
                  options={productTypeOptions}
                />
              )}
            </Form.Item>
          )}

          {/* 文件名 */}
          <Form.Item
            label={
              <Space>
                文件名
                {importMode === 'add' && (
                  <span style={{ color: '#999', fontSize: 12 }}>
                    (支持 {fileType === 'upgrade' ? 'IMG / EXT' : fileType === 'ca' ? 'Patch' : 'IMG'} 格式)
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
                  <p className="ant-upload-text">点击或拖拽文件到此区域</p>
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
            label="版本"
            rules={[
              { required: true, message: '请输入版本号' },
              { max: 45, message: '版本号不能超过45个字符' },
            ]}
          >
            <Input placeholder="请输入版本号" maxLength={45} />
          </Form.Item>

          {/* 推荐 */}
          <Form.Item name="recommend" label="推荐">
            <Select
              placeholder="请选择"
              options={[
                { label: '是', value: '1' },
                { label: '否', value: '0' },
              ]}
            />
          </Form.Item>

          {/* 描述 */}
          <Form.Item name="description" label="描述">
            <TextArea rows={3} placeholder="请输入描述信息" />
          </Form.Item>
        </Form>
      </Drawer>

      {/* 删除确认弹窗 */}
      <Modal
        title="确认删除"
        open={!!deleteFile}
        onCancel={() => setDeleteFile(null)}
        onOk={handleDeleteFile}
        okText="确认删除"
        cancelText="取消"
        okButtonProps={{ danger: true }}
      >
        <Alert
          type="warning"
          showIcon
          icon={<WarningOutlined />}
          message={
            <div>
              <p style={{ marginBottom: 8 }}>
                确定要删除以下升级文件吗？此操作不可恢复。
              </p>
              <p style={{ marginBottom: 0 }}>
                <strong>版本：</strong>{deleteFile?.version}<br />
                <strong>文件名：</strong>{deleteFile?.fileName}<br />
                <strong>文件大小：</strong>{deleteFile ? formatFileSize(deleteFile.size) : '-'}
              </p>
            </div>
          }
        />
      </Modal>
    </ListPageLayout>
  );
}
