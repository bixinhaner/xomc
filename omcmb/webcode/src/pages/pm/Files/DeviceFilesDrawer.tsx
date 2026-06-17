/**
 * DeviceFilesDrawer — 单台设备的 PM 文件抽屉。
 *
 * 顶部按 collect_time 范围筛选；中间表格按时间倒序展示，可多选；
 * 底部按钮触发 bundle('pm_files') 打包下载，单个文件用 useDownloadPMFile。
 */
import { useEffect, useMemo, useState } from 'react';
import { Button, DatePicker, Drawer, Space, Table, Tag } from 'antd';
import { DownloadOutlined } from '@ant-design/icons';
import { type Dayjs } from 'dayjs';
import type { ColumnsType } from 'antd/es/table';
import { useT } from '@/hooks/useT';
import { usePMFiles, useDownloadPMFile } from '@core/hooks/api/usePerformance';
import type { PMFileItem, PMFileDeviceItem } from '@core/services/api/pmApi';
import { useBatchDownloadWithMessage } from '@/hooks/useBatchDownloadWithMessage';
import { formatSystemTime } from '@core/utils/systemTime';

function formatBytes(n: number) {
  if (!n) return '—';
  if (n >= 1024 * 1024) return `${(n / 1024 / 1024).toFixed(2)} MB`;
  return `${(n / 1024).toFixed(2)} KB`;
}

interface Props {
  open: boolean;
  device: PMFileDeviceItem | null;
  onClose: () => void;
}

export default function DeviceFilesDrawer({ open, device, onClose }: Props) {
  const t = useT();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [range, setRange] = useState<[Dayjs, Dayjs] | null>(null);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);

  // 抽屉关闭/换设备时重置
  useEffect(() => {
    if (!open) return;
    setPage(1);
    setRange(null);
    setSelectedKeys([]);
  }, [open, device?.deviceSn]);

  const queryParams = useMemo(() => {
    if (!device) return null;
    return {
      page,
      pageSize,
      deviceSn: device.deviceSn,
      timeRange: range
        ? ([range[0].toISOString(), range[1].toISOString()] as [string, string])
        : undefined,
    };
  }, [device, page, pageSize, range]);

  const { data, isLoading } = usePMFiles(
    queryParams ?? { page: 1, pageSize: 20, deviceSn: '' },
  );
  const download = useDownloadPMFile();
  const bundle = useBatchDownloadWithMessage();

  const handleBatchDownload = () => {
    const ids = selectedKeys.map(String);
    if (!ids.length) return;
    bundle.trigger({ module: 'pm_files', targets: ids });
  };

  const columns: ColumnsType<PMFileItem> = [
    {
      title: t('pm.fileName'),
      dataIndex: 'fileName',
      ellipsis: true,
      render: (v) => (
        <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{v}</span>
      ),
    },
    {
      title: t('pm.fileSize'),
      dataIndex: 'fileSize',
      width: 110,
      render: (v) => formatBytes(Number(v)),
    },
    {
      title: t('pm.counterCount'),
      dataIndex: 'counterCount',
      width: 110,
      render: (v) => Number(v ?? 0).toLocaleString(),
    },
    {
      title: t('pm.parsed'),
      dataIndex: 'parsed',
      width: 90,
      render: (v) =>
        v ? (
          <Tag color="green">{t('pm.parsed')}</Tag>
        ) : (
          <Tag color="default">{t('pm.parsing')}</Tag>
        ),
    },
    {
      title: t('pm.collectTime'),
      dataIndex: 'collectTime',
      width: 170,
      render: (v) => (v ? formatSystemTime(v, { format: 'YYYY-MM-DD HH:mm:ss', placeholder: '-' }) : '—'),
    },
    {
      title: t('table.operation'),
      width: 90,
      fixed: 'right',
      render: (_, row) => (
        <Button
          type="link"
          size="small"
          icon={<DownloadOutlined />}
          loading={download.isPending}
          onClick={() => void download.mutate(row.id)}
        >
          {t('common.download')}
        </Button>
      ),
    },
  ];

  return (
    <Drawer
      title={device ? t('pm.deviceFilesTitle', { sn: device.deviceSn }) : ''}
      open={open}
      onClose={onClose}
      size={920}
      destroyOnHidden
      extra={
        <Button
          type="primary"
          icon={<DownloadOutlined />}
          disabled={!selectedKeys.length || bundle.isPending}
          loading={bundle.isPending}
          onClick={handleBatchDownload}
        >
          {t('common.batchExport')} ({selectedKeys.length})
        </Button>
      }
    >
      <Space style={{ marginBottom: 12 }}>
        <DatePicker.RangePicker
          showTime
          allowClear
          placeholder={[
            t('pm.firstCollectTime'),
            t('pm.lastCollectTime'),
          ]}
          value={range as unknown as [Dayjs, Dayjs] | null}
          onChange={(vals) => {
            setRange(vals as [Dayjs, Dayjs] | null);
            setPage(1);
          }}
        />
      </Space>
      <Table<PMFileItem>
        rowKey="id"
        size="small"
        loading={isLoading}
        columns={columns}
        dataSource={data?.items ?? []}
        rowSelection={{
          selectedRowKeys: selectedKeys,
          onChange: (keys) => setSelectedKeys(keys),
        }}
        pagination={{
          current: page,
          pageSize,
          total: data?.total ?? 0,
          showSizeChanger: true,
          onChange: (p, s) => {
            setPage(p);
            setPageSize(s);
          },
        }}
        scroll={{ x: 700 }}
      />
    </Drawer>
  );
}
