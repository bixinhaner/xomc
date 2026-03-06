import { useState, useMemo } from 'react';
import {
  Button,
  Card,
  Form,
  Input,
  Select,
  Progress,
  Table,
  Tag,
  Space,
  Upload,
  message,
  Typography,
} from 'antd';
import { InboxOutlined, DeleteOutlined, DownloadOutlined } from '@ant-design/icons';
import type { UploadFile, RcFile } from 'antd/es/upload';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useUploadSoftwareVersion } from '@/hooks/api/useSoftware';
import { useT } from '@/hooks/useT';

const { Dragger } = Upload;
const { TextArea } = Input;

interface UploadedFirmware {
  id: string;
  fileName: string;
  deviceType: string;
  versionCode: string;
  description: string;
  fileSize: number;
  uploadTime: string;
  status: 'uploaded' | 'processing' | 'ready';
}

const mockUploaded: UploadedFirmware[] = [
  {
    id: 'fw-001',
    fileName: 'eNB_V100R011C10SPC200.tar.gz',
    deviceType: 'eNB',
    versionCode: 'V100R011C10SPC200',
    description: '华为eNB主流版本升级包',
    fileSize: 1024 * 1024 * 512,
    uploadTime: '2024-06-01T09:00:00.000Z',
    status: 'ready',
  },
  {
    id: 'fw-002',
    fileName: 'gNB_V200R001C10SPC100.tar.gz',
    deviceType: 'gNB',
    versionCode: 'V200R001C10SPC100',
    description: '华为gNB 5G版本升级包',
    fileSize: 1024 * 1024 * 768,
    uploadTime: '2024-06-05T14:00:00.000Z',
    status: 'ready',
  },
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
  const [firmware, setFirmware] = useState<UploadedFirmware[]>(mockUploaded);
  const uploadVersion = useUploadSoftwareVersion();

  const handleUpload = () => {
    if (fileList.length === 0) {
      void message.warning(t('common.pleaseSelect'));
      return;
    }
    form.validateFields().then((vals) => {
      setUploading(true);
      setUploadProgress(0);
      const timer = setInterval(() => {
        setUploadProgress((prev) => {
          if (prev >= 100) {
            clearInterval(timer);
            setUploading(false);
            const file = fileList[0];
            const newFirmware: UploadedFirmware = {
              id: `fw-${Date.now()}`,
              fileName: file.name,
              deviceType: vals.deviceType as string,
              versionCode: vals.versionCode as string,
              description: (vals.description as string) ?? '',
              fileSize: file.size ?? 0,
              uploadTime: new Date().toISOString(),
              status: 'ready',
            };
            setFirmware((prev) => [newFirmware, ...prev]);
            setFileList([]);
            form.resetFields();
            void message.success(t('common.upload'));
            return 100;
          }
          return prev + 10;
        });
      }, 200);

      uploadVersion.mutate({
        versionName: vals.versionCode as string,
        versionCode: vals.versionCode as string,
        deviceType: vals.deviceType as string,
        vendor: '华为',
        status: 'beta',
        fileSize: fileList[0]?.size ?? 0,
        checksum: 'sha256:pending',
        downloadUrl: `/files/firmware/${fileList[0]?.name ?? ''}`,
        releaseNotes: (vals.description as string) ?? '',
        minHardwareVersion: '',
        features: [],
        bugFixes: [],
      });
    });
  };

  const uploadColumns = useMemo(() => [
    {
      title: t('table.name'),
      dataIndex: 'fileName',
      ellipsis: true,
      render: (val: string) => <Typography.Text style={{ fontFamily: 'monospace', fontSize: 12 }}>{val}</Typography.Text>,
    },
    { title: t('device.productType'), dataIndex: 'deviceType', width: 100 },
    {
      title: t('table.version'),
      dataIndex: 'versionCode',
      width: 200,
      render: (val: string) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{val}</span>,
    },
    {
      title: t('table.description'),
      dataIndex: 'fileSize',
      width: 110,
      render: (val: number) => formatFileSize(val),
    },
    {
      title: t('table.createTime'),
      dataIndex: 'uploadTime',
      width: 160,
      render: (val: string) => new Date(val).toLocaleString('zh-CN'),
    },
    {
      title: t('table.status'),
      dataIndex: 'status',
      width: 90,
      render: (val: string) => (
        <Tag color={val === 'ready' ? 'green' : val === 'processing' ? 'processing' : 'default'}>
          {val === 'ready' ? t('status.success') : val === 'processing' ? t('status.running') : t('status.pending')}
        </Tag>
      ),
    },
    {
      title: t('table.operation'),
      width: 120,
      render: (_: unknown, record: UploadedFirmware) => (
        <Space size="small">
          <Button type="link" size="small" icon={<DownloadOutlined />}>{t('common.download')}</Button>
          <Button
            type="link"
            size="small"
            danger
            icon={<DeleteOutlined />}
            onClick={() => setFirmware((prev) => prev.filter((f) => f.id !== record.id))}
          >
            {t('common.delete')}
          </Button>
        </Space>
      ),
    },
  ], [t]);

  return (
    <ListPageLayout title={t('nav.software.firmware')}>
      <Card title={t('common.upload')} style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', gap: 24 }}>
          <div style={{ flex: 1 }}>
            <Dragger
              fileList={fileList}
              beforeUpload={(file: RcFile) => {
                const allowed = ['.tar.gz', '.zip', '.bin', '.img'];
                const valid = allowed.some((ext) => file.name.endsWith(ext));
                if (!valid) {
                  void message.error(t('common.featureInDev'));
                  return Upload.LIST_IGNORE;
                }
                setFileList([file]);
                return false;
              }}
              onRemove={() => setFileList([])}
              maxCount={1}
              accept=".tar.gz,.zip,.bin,.img"
            >
              <p className="ant-upload-drag-icon">
                <InboxOutlined />
              </p>
              <p className="ant-upload-text">{t('common.upload')}</p>
              <p className="ant-upload-hint">.tar.gz / .zip / .bin / .img</p>
            </Dragger>
            {uploading && (
              <div style={{ marginTop: 12 }}>
                <Progress
                  percent={uploadProgress}
                  status={uploadProgress < 100 ? 'active' : 'success'}
                />
              </div>
            )}
          </div>
          <div style={{ width: 340 }}>
            <Form form={form} layout="vertical">
              <Form.Item
                name="deviceType"
                label={t('device.productType')}
                rules={[{ required: true }]}
              >
                <Select
                  placeholder={t('common.pleaseSelect')}
                  options={[
                    { label: 'eNB (4G基站)', value: 'eNB' },
                    { label: 'gNB (5G基站)', value: 'gNB' },
                    { label: 'RRU (射频单元)', value: 'RRU' },
                    { label: 'AAU (有源天线)', value: 'AAU' },
                    { label: 'BBU (基带单元)', value: 'BBU' },
                  ]}
                />
              </Form.Item>
              <Form.Item
                name="versionCode"
                label={t('table.version')}
                rules={[
                  { required: true },
                  { pattern: /^V\d+/ },
                ]}
              >
                <Input placeholder={t('common.placeholder')} />
              </Form.Item>
              <Form.Item name="description" label={t('table.description')}>
                <TextArea rows={3} placeholder={t('common.placeholder')} />
              </Form.Item>
              <Form.Item name="compatibleVersions" label={t('table.version')}>
                <TextArea rows={2} placeholder={t('common.placeholder')} />
              </Form.Item>
              <Button
                type="primary"
                onClick={handleUpload}
                loading={uploading}
                disabled={fileList.length === 0}
                block
              >
                {uploading ? `${t('common.loading')} ${uploadProgress}%` : t('common.upload')}
              </Button>
            </Form>
          </div>
        </div>
      </Card>

      <Card title={t('table.total')}>
        <Table
          dataSource={firmware}
          columns={uploadColumns}
          rowKey="id"
          size="small"
          pagination={{ pageSize: 10, showSizeChanger: true }}
          scroll={{ x: 900 }}
        />
      </Card>
    </ListPageLayout>
  );
}
