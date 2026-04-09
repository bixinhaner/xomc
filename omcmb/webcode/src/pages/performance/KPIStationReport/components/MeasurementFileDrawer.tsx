import { Drawer, Table, Button, Space, Tag, App, DatePicker, Alert, Divider } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DownloadOutlined, ReloadOutlined, DeleteOutlined } from '@ant-design/icons';
import { useT } from '@/hooks/useT';
import { useState, useCallback, useMemo } from 'react';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';

const { RangePicker } = DatePicker;

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
  { id: '4', fileName: 'ENB00001_20260401_001.xml.gz', fileSize: 87040, collectTime: '2026-04-01 08:00:00', status: 'success' },
  { id: '5', fileName: 'ENB00001_20260401_002.xml.gz', fileSize: 92160, collectTime: '2026-04-01 08:15:00', status: 'success' },
  { id: '6', fileName: 'ENB00001_20260401_003.xml.gz', fileSize: 0, collectTime: '2026-04-01 08:30:00', status: 'pending' },
  { id: '7', fileName: 'ENB00001_20260331_001.xml.gz', fileSize: 0, collectTime: '2026-03-31 08:00:00', status: 'failed' },
  { id: '8', fileName: 'ENB00001_20260331_002.xml.gz', fileSize: 110592, collectTime: '2026-03-31 08:15:00', status: 'success' },
  { id: '9', fileName: 'ENB00001_20260330_001.xml.gz', fileSize: 95000, collectTime: '2026-03-30 08:00:00', status: 'success' },
  { id: '10', fileName: 'ENB00001_20260330_002.xml.gz', fileSize: 88000, collectTime: '2026-03-30 08:15:00', status: 'success' },
];

// 状态颜色映射
const statusColorMap: Record<string, string> = {
  success: 'success',
  failed: 'error',
  pending: 'processing',
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
  const { message, modal } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [timeRange, setTimeRange] = useState<[Dayjs, Dayjs] | null>(null);

  // 按时间过滤文件
  const filteredFiles = useMemo(() => {
    if (!timeRange) return mockFiles;
    const start = timeRange[0].startOf('day');
    const end = timeRange[1].endOf('day');
    return mockFiles.filter((f) => {
      const fileTime = dayjs(f.collectTime);
      return fileTime.isAfter(start) && fileTime.isBefore(end);
    });
  }, [timeRange]);

  // 刷新文件列表
  const handleRefresh = useCallback(() => {
    setLoading(true);
    setTimeout(() => {
      setLoading(false);
      void message.success(t('common.success'));
    }, 500);
  }, [message, t]);

  // 下载单个文件
  const handleDownload = useCallback((record: MeasurementFile) => {
    void message.info(`${t('common.downloading')}: ${record.fileName}`);
  }, [message, t]);

  // 批量下载
  const handleBatchDownload = useCallback(() => {
    if (selectedRowKeys.length === 0) {
      void message.warning(t('common.selectAtLeastOne'));
      return;
    }
    const selected = filteredFiles.filter(
      (f) => selectedRowKeys.includes(f.id) && f.status === 'success'
    );
    if (selected.length === 0) {
      void message.warning(t('perf.measurement.noDownloadableFiles'));
      return;
    }
    void message.success(t('perf.measurement.batchDownloadSuccess', { count: selected.length }));
  }, [selectedRowKeys, filteredFiles, message, t]);

  // 批量删除
  const handleBatchDelete = useCallback(() => {
    if (selectedRowKeys.length === 0) {
      void message.warning(t('common.selectAtLeastOne'));
      return;
    }
    modal.confirm({
      title: t('common.confirmDelete'),
      content: t('perf.measurement.batchDeleteConfirm', { count: selectedRowKeys.length }),
      okType: 'danger',
      onOk: () => {
        setSelectedRowKeys([]);
        void message.success(t('common.deleteSuccess'));
      },
    });
  }, [selectedRowKeys, modal, message, t]);

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
          {t(`perf.measurement.file${status.charAt(0).toUpperCase() + status.slice(1)}`)}
        </Tag>
      ),
    },
    {
      key: 'action',
      title: t('common.operation'),
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
      onClose={() => {
        setSelectedRowKeys([]);
        setTimeRange(null);
        onClose();
      }}
      footer={
        <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
          <Button onClick={() => { setSelectedRowKeys([]); setTimeRange(null); onClose(); }}>
            {t('common.close')}
          </Button>
        </Space>
      }
    >
      <div style={{ marginBottom: 16 }}>
        <span style={{ color: '#8c8c8c' }}>
          {t('device.hostName')}: {device.hostName} | {t('device.code')}: {device.serialNumber}
        </span>
      </div>

      {/* 时间筛选 */}
      <div style={{ marginBottom: 16, display: 'flex', alignItems: 'center', gap: 8 }}>
        <span style={{ whiteSpace: 'nowrap' }}>{t('perf.measurement.timeRange')}</span>
        <RangePicker
          value={timeRange}
          onChange={(dates) => setTimeRange(dates as [Dayjs, Dayjs] | null)}
          format="YYYY-MM-DD"
          style={{ flex: 1 }}
          size="small"
          allowClear
          placeholder={[t('perf.measurement.startDate'), t('perf.measurement.endDate')]}
        />
        <Button icon={<ReloadOutlined />} onClick={handleRefresh} loading={loading} size="small">
          {t('common.refresh')}
        </Button>
      </div>

      {/* 批量操作 */}
      {selectedRowKeys.length > 0 && (
        <>
          <Alert
            type="info"
            showIcon
            message={t('perf.measurement.selectedFilesInfo', { count: selectedRowKeys.length })}
            style={{ marginBottom: 12 }}
            action={
              <Space size="small">
                <Button
                  size="small"
                  icon={<DownloadOutlined />}
                  onClick={handleBatchDownload}
                >
                  {t('perf.measurement.batchDownload')}
                </Button>
                <Button
                  size="small"
                  danger
                  icon={<DeleteOutlined />}
                  onClick={handleBatchDelete}
                >
                  {t('common.delete')}
                </Button>
              </Space>
            }
          />
          <Divider style={{ margin: '0 0 12px' }} />
        </>
      )}

      <Table<MeasurementFile>
        rowKey="id"
        columns={columns}
        dataSource={filteredFiles}
        loading={loading}
        size="small"
        rowSelection={{
          selectedRowKeys,
          onChange: (keys) => setSelectedRowKeys(keys),
        }}
        scroll={{ y: 'calc(100vh - 310px)' }}
        pagination={{ pageSize: 10, showSizeChanger: false, size: 'small' }}
      />
    </Drawer>
  );
}
