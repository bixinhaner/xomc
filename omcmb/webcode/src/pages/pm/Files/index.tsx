/**
 * PM Files — File Management → PM Tab 主入口。
 *
 * 列表按 device_sn 聚合，每行 1 个设备 + 起止 collect_time + 文件数 + 上报状态。
 * 点 deviceSn 链接打开 DeviceFilesDrawer，按时间筛选 + 多选批量下载。
 *
 * 与 mr/Files 同构。reporting 字段后端依据 max(collect_time) 距 now() 2 小时内
 * 判定，PM 无 MR 那样的订阅任务表。
 */
import { useMemo, useState } from 'react';
import { Badge, Button, Card, Input, Space, message } from 'antd';
import { DownloadOutlined, ReloadOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';
import { usePMFileDevices } from '@core/hooks/api/usePerformance';
import { useBatchDownloadWithMessage } from '@/hooks/useBatchDownloadWithMessage';
import type { PMFileDeviceItem } from '@core/services/api/pmApi';
import DeviceFilesDrawer from './DeviceFilesDrawer';

interface Props {
  /** 嵌入到 FileManagement 时去掉外层卡片背景的留白。 */
  embedded?: boolean;
}

export default function PMFilesPage({ embedded }: Props) {
  const t = useT();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [keyword, setKeyword] = useState('');
  const [activeDevice, setActiveDevice] = useState<PMFileDeviceItem | null>(null);

  const params = useMemo(
    () => ({ page, pageSize, keyword: keyword || undefined }),
    [page, pageSize, keyword],
  );
  const { data, isLoading, refetch } = usePMFileDevices(params);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);
  const bundle = useBatchDownloadWithMessage();
  const handleBatchDownload = () => {
    if (selectedKeys.length === 0) {
      void message.warning(t('bundle.selectFiles'));
      return;
    }
    bundle.trigger({ module: 'pm', targets: selectedKeys.map(String) });
  };

  const columns: DataTableColumn<PMFileDeviceItem>[] = useMemo(
    () => [
      {
        key: 'deviceSn',
        title: t('device.sn'),
        dataIndex: 'deviceSn',
        width: 220,
        render: (_, record) => (
          <Button
            type="link"
            size="small"
            style={{ padding: 0, fontFamily: 'monospace', fontSize: 12 }}
            onClick={() => setActiveDevice(record)}
          >
            {record.deviceSn}
          </Button>
        ),
      },
      {
        // 测量周期目前是固定值：CPE 默认 PM 上传间隔 900s = 15 分钟（worker
        // config.dev.yaml 第 63 行）。后续若按设备读 Device.FAP.PerfMgmt.Config
        // 再改成动态列。
        key: 'measurementPeriod',
        title: t('pm.measurementPeriod'),
        dataIndex: 'measurementPeriod',
        width: 140,
        render: () => t('pm.measurementPeriodValue'),
      },
      {
        key: 'firstCollectTime',
        title: t('pm.startTime'),
        dataIndex: 'firstCollectTime',
        width: 180,
        render: (v) => (v ? dayjs(v as string).format('YYYY-MM-DD HH:mm:ss') : '—'),
      },
      {
        key: 'lastCollectTime',
        title: t('pm.updateTime'),
        dataIndex: 'lastCollectTime',
        width: 180,
        render: (v) => (v ? dayjs(v as string).format('YYYY-MM-DD HH:mm:ss') : '—'),
      },
      {
        key: 'fileCount',
        title: t('pm.fileCount'),
        dataIndex: 'fileCount',
        width: 100,
        render: (v) => Number(v ?? 0).toLocaleString(),
      },
      {
        key: 'reporting',
        title: t('pm.reportingStatus'),
        dataIndex: 'reporting',
        width: 110,
        fixed: 'right',
        render: (_, record) =>
          record.reporting ? (
            <Badge status="processing" text={t('pm.reporting')} />
          ) : (
            <Badge status="default" text={t('pm.reportStopped')} />
          ),
      },
    ],
    [t],
  );

  const body = (
    <>
      <Space style={{ marginBottom: 12 }}>
        <Input.Search
          allowClear
          placeholder={t('pm.searchDeviceSn')}
          style={{ width: 260 }}
          onSearch={(v) => {
            setKeyword(v.trim());
            setPage(1);
          }}
        />
        <Button icon={<ReloadOutlined />} onClick={() => void refetch()}>
          {t('common.refresh')}
        </Button>
        <Button
          type="primary"
          icon={<DownloadOutlined />}
          disabled={selectedKeys.length === 0 || bundle.isPending}
          loading={bundle.isPending}
          onClick={handleBatchDownload}
        >
          {t('bundle.batchDownload')} ({selectedKeys.length})
        </Button>
      </Space>
      <DataTable<PMFileDeviceItem>
        tableId="pm-file-devices"
        columns={columns}
        dataSource={data?.items ?? []}
        loading={isLoading}
        rowKey="deviceSn"
        total={data?.total ?? 0}
        pageSize={pageSize}
        currentPage={page}
        onPageChange={(p, s) => {
          setPage(p);
          setPageSize(s);
        }}
        selectable
        selectedRowKeys={selectedKeys}
        onSelectionChange={(keys) => setSelectedKeys(keys)}
        scroll={{ x: 900 }}
      />
      <DeviceFilesDrawer
        open={!!activeDevice}
        device={activeDevice}
        onClose={() => setActiveDevice(null)}
      />
    </>
  );

  if (embedded) {
    return <div style={{ paddingTop: 4 }}>{body}</div>;
  }
  return (
    <Card
      size="small"
      bordered
      style={{ flex: 1, display: 'flex', flexDirection: 'column' }}
      styles={{ body: { display: 'flex', flexDirection: 'column', flex: 1 } }}
    >
      {body}
    </Card>
  );
}
