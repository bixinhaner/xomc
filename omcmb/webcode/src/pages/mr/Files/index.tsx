/**
 * MR Files (Y3) — File Management → MR Tab 主入口。
 *
 * 列表按 device_sn 聚合，每行 1 个设备 + 起止 collect_time + 文件数 + 下载按钮。
 * 点"查看文件"打开 DeviceFilesDrawer，按时间筛选 + 多选批量下载。
 */
import { useMemo, useState } from 'react';
import { Button, Card, Input, Space, Tooltip } from 'antd';
import { DownloadOutlined, ReloadOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';
import { useMRFileDevices } from '@core/hooks/api/useMR';
import type { MRFileDeviceItem } from '@core/services/api/mrApi';
import DeviceFilesDrawer from './DeviceFilesDrawer';

interface Props {
  /** 嵌入到 FileManagement 时去掉外层卡片背景的留白。 */
  embedded?: boolean;
}

export default function MRFilesPage({ embedded }: Props) {
  const t = useT();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [keyword, setKeyword] = useState('');
  const [activeDevice, setActiveDevice] = useState<MRFileDeviceItem | null>(null);

  const params = useMemo(
    () => ({ page, pageSize, keyword: keyword || undefined }),
    [page, pageSize, keyword],
  );
  const { data, isLoading, refetch } = useMRFileDevices(params);

  const columns: DataTableColumn<MRFileDeviceItem>[] = useMemo(
    () => [
      {
        key: 'deviceSn',
        title: t('device.sn'),
        dataIndex: 'deviceSn',
        width: 220,
        render: (v) => (
          <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(v)}</span>
        ),
      },
      {
        key: 'firstCollectTime',
        title: t('mr.firstCollectTime'),
        dataIndex: 'firstCollectTime',
        width: 180,
        render: (v) => (v ? dayjs(v as string).format('YYYY-MM-DD HH:mm:ss') : '—'),
      },
      {
        key: 'lastCollectTime',
        title: t('mr.lastCollectTime'),
        dataIndex: 'lastCollectTime',
        width: 180,
        render: (v) => (v ? dayjs(v as string).format('YYYY-MM-DD HH:mm:ss') : '—'),
      },
      {
        key: 'fileCount',
        title: t('mr.fileCount'),
        dataIndex: 'fileCount',
        width: 100,
        render: (v) => Number(v ?? 0).toLocaleString(),
      },
      {
        key: 'actions',
        title: t('table.operation'),
        width: 120,
        fixed: 'right',
        render: (_, record) => (
          <Tooltip title={t('mr.batchDownloadHint')}>
            <Button
              type="link"
              size="small"
              icon={<DownloadOutlined />}
              onClick={() => setActiveDevice(record)}
            >
              {t('common.download')}
            </Button>
          </Tooltip>
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
          placeholder={t('mr.searchDeviceSn')}
          style={{ width: 260 }}
          onSearch={(v) => {
            setKeyword(v.trim());
            setPage(1);
          }}
        />
        <Button icon={<ReloadOutlined />} onClick={() => void refetch()}>
          {t('common.refresh')}
        </Button>
      </Space>
      <DataTable<MRFileDeviceItem>
        tableId="mr-file-devices"
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
