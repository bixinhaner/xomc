import { Drawer, Table, Button, Space, Tag, App } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DownloadOutlined, ReloadOutlined } from '@ant-design/icons';
import { useT } from '@/hooks/useT';
import { useState, useCallback, useMemo } from 'react';

/** 测量文件行数据 */
interface MeasurementFile {
  id: string;
  fileName: string;
  fileSize: number;
  collectTime: string;
  status: 'success' | 'failed' | 'pending';
}

interface MeasurementFileDrawerProps {
  open: boolean;
  device: {
    id: string;
    serialNumber: string;
    hostName: string;
    cellId: string;
    smallCellCode: string;
  } | null;
  onClose: () => void;
}

// Mock 文件数据
const mockFiles: MeasurementFile[] = [
  { id: '1', fileName: 'ENB00001_20260402_001.xml.gz', fileSize: 102400, collectTime: '2026-04-02 08:00:00', status: 'success' },
  { id: '2', fileName: 'ENB00001_20260402_002.xml.gz', fileSize: 98304, collectTime: '2026-04-02 08:15:00', status: 'success' },
  { id: '3', fileName: 'ENB00001_20260402_003.xml.gz', fileSize: 105472, collectTime: '2026-04-02 08:30:00', status: 'success' },
  { id: '4', fileName: 'ENB00001_20260402_004.xml.gz', fileSize: 0, collectTime: '2026-04-02 08:45:00', status: 'pending' },
  { id: '5', fileName: 'ENB00001_20260402_005.xml.gz', fileSize: 0, collectTime: '2026-04-02 09:00:00', status: 'failed' },
];

// 状态颜色映射
const statusColorMap: Record<string, string> = {
  success: 'success',
  failed: 'error',
  pending: 'processing',
};

// 状态文本映射
const statusTextMap: Record<string, string> = {
  success: 'perf.measurement.fileSuccess',
  failed: 'perf.measurement.fileFailed',
  pending: 'perf.measurement.filePending',
};

// 格式化文件大小
function formatFileSize(bytes: number): string {
  if (bytes === 0) return '-';
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
}

export default function MeasurementFileDrawer({ open, device, onClose }: MeasurementFileDrawerProps) {
  const t = useT();
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);

  // 刷新文件列表
  const handleRefresh = useCallback(() => {
    setLoading(true);
    setTimeout(() => {
      setLoading(false);
      void message.success(t('common.success'));
    }, 500);
  }, [message, t]);

  // 下载文件
  const handleDownload = useCallback((record: MeasurementFile) => {
    console.log('下载测量文件:', record.fileName);
    void message.info(t('common.downloading'));
  }, [message, t]);

  // 表格列配置
  const columns: ColumnsType<MeasurementFile> = useMemo(() => [
    {
      key: 'fileName',
      title: t('perf.measurement.fileName'),
      dataIndex: 'fileName',
      ellipsis: true,
      render: (text: string) => (
        <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{text}</span>
      ),
    },
    {
      key: 'fileSize',
      title: t('perf.measurement.fileSize'),
      dataIndex: 'fileSize',
      width: 100,
      render: (size: number) => formatFileSize(size),
    },
    {
      key: 'collectTime',
      title: t('perf.measurement.collectTime'),
      dataIndex: 'collectTime',
      width: 160,
    },
    {
      key: 'status',
      title: t('common.status'),
      dataIndex: 'status',
      width: 100,
      render: (status: string) => (
        <Tag color={statusColorMap[status] || 'default'}>
          {t(statusTextMap[status] || status)}
        </Tag>
      ),
    },
    {
      key: 'action',
      title: t('common.action'),
      width: 80,
      render: (_, record) => (
        <Button
          type="link"
          size="small"
          icon={<DownloadOutlined />}
          disabled={record.status !== 'success'}
          onClick={() => handleDownload(record)}
        >
          {t('common.download')}
        </Button>
      ),
    },
  ], [t, handleDownload]);

  if (!device) return null;

  return (
    <Drawer
      title={`${t('perf.measurement.fileList')} - ${device.serialNumber}`}
      placement="right"
      width={720}
      open={open}
      onClose={onClose}
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={handleRefresh} loading={loading}>
            {t('common.refresh')}
          </Button>
        </Space>
      }
      footer={
        <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
          <Button onClick={onClose}>{t('common.close')}</Button>
        </Space>
      }
    >
      <div style={{ marginBottom: 16 }}>
        <span style={{ color: '#8c8c8c' }}>
          {t('device.hostName')}: {device.hostName} | {t('device.code')}: {device.serialNumber}
        </span>
      </div>
      <Table<MeasurementFile>
        rowKey="id"
        columns={columns}
        dataSource={mockFiles}
        loading={loading}
        size="small"
        pagination={{ pageSize: 10, showSizeChanger: false }}
      />
    </Drawer>
  );
}
