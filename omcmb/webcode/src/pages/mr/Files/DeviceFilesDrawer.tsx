/**
 * DeviceFilesDrawer (Y3) — 单台设备的 MR 文件抽屉。
 *
 * 顶部按 collect_time 范围筛选；中间表格按时间倒序展示，可多选；
 * 底部按钮触发逐个 download（复用 useDownloadMRFile）。
 */
import { useEffect, useMemo, useState } from 'react';
import { Button, DatePicker, Drawer, Space, Table, Tag, message } from 'antd';
import { DownloadOutlined } from '@ant-design/icons';
import dayjs, { type Dayjs } from 'dayjs';
import type { ColumnsType } from 'antd/es/table';
import { useT } from '@/hooks/useT';
import { useMRFiles, useDownloadMRFile } from '@core/hooks/api/useMR';
import type { MRFileItem, MRFileDeviceItem } from '@core/services/api/mrApi';

const mrTypeColor: Record<string, string> = { MRO: 'blue', MRE: 'green', MRS: 'orange' };

function formatBytes(n: number) {
  if (!n) return '—';
  if (n >= 1024 * 1024) return `${(n / 1024 / 1024).toFixed(2)} MB`;
  return `${(n / 1024).toFixed(2)} KB`;
}

interface Props {
  open: boolean;
  device: MRFileDeviceItem | null;
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

  const { data, isLoading } = useMRFiles(
    queryParams ?? { page: 1, pageSize: 20, deviceSn: '' },
  );
  const download = useDownloadMRFile();

  const handleBatchDownload = async () => {
    const ids = selectedKeys.map(String);
    if (!ids.length) return;
    for (const id of ids) {
      try {
        await download.mutateAsync(id);
      } catch (err) {
        void message.error(String(err));
      }
    }
    void message.success(t('mr.batchDownload', { count: String(ids.length) }));
  };

  const columns: ColumnsType<MRFileItem> = [
    {
      title: t('mr.mrType'),
      dataIndex: 'mrType',
      width: 80,
      render: (v) => <Tag color={mrTypeColor[v] || 'default'}>{v}</Tag>,
    },
    {
      title: t('mr.fileName'),
      dataIndex: 'fileName',
      ellipsis: true,
      render: (v) => (
        <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{v}</span>
      ),
    },
    {
      title: t('mr.fileSize'),
      dataIndex: 'fileSize',
      width: 110,
      render: (v) => formatBytes(Number(v)),
    },
    {
      title: t('mr.collectTime'),
      dataIndex: 'collectTime',
      width: 170,
      render: (v) => (v ? dayjs(v).format('YYYY-MM-DD HH:mm:ss') : '—'),
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
      title={device ? t('mr.deviceFilesTitle', { sn: device.deviceSn }) : ''}
      open={open}
      onClose={onClose}
      width={920}
      destroyOnClose
      extra={
        <Button
          type="primary"
          icon={<DownloadOutlined />}
          disabled={!selectedKeys.length}
          loading={download.isPending}
          onClick={() => void handleBatchDownload()}
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
            t('mr.firstCollectTime'),
            t('mr.lastCollectTime'),
          ]}
          value={range as unknown as [Dayjs, Dayjs] | null}
          onChange={(vals) => {
            setRange(vals as [Dayjs, Dayjs] | null);
            setPage(1);
          }}
        />
      </Space>
      <Table<MRFileItem>
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
