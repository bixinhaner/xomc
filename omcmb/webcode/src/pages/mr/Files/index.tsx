/**
 * MR Files (Y3) — File Management → MR Tab 主入口。
 *
 * 列表按 device_sn 聚合，每行 1 个设备 + 起止 collect_time + 文件数 + 下载按钮。
 * 点"查看文件"打开 DeviceFilesDrawer，按时间筛选 + 多选批量下载。
 */
import { useMemo, useState } from 'react';
import { Badge, Button, Card, Input, Modal, Space, message } from 'antd';
import { DeleteOutlined, DownloadOutlined, ReloadOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';
import { useBatchDeleteMRFiles, useMRFileDevices } from '@core/hooks/api/useMR';
import { useBatchDownloadWithMessage } from '@/hooks/useBatchDownloadWithMessage';
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
  const [siteName, setSiteName] = useState('');
  const [productClass, setProductClass] = useState('');
  const [activeDevice, setActiveDevice] = useState<MRFileDeviceItem | null>(null);

  const params = useMemo(
    () => ({
      page,
      pageSize,
      keyword: keyword || undefined,
      siteName: siteName || undefined,
      productClass: productClass || undefined,
    }),
    [page, pageSize, keyword, siteName, productClass],
  );
  const { data, isLoading, refetch } = useMRFileDevices(params);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);
  const bundle = useBatchDownloadWithMessage();
  const batchDelete = useBatchDeleteMRFiles();
  const handleBatchDownload = () => {
    if (selectedKeys.length === 0) {
      void message.warning(t('bundle.selectFiles'));
      return;
    }
    bundle.trigger({ module: 'mr', targets: selectedKeys.map(String) });
  };

  // 与 License/Config 的 confirmDelete 模式一致。
  const confirmDelete = (sns: string[]) => {
    Modal.confirm({
      title: t('transfer.fileLib.msg.deleteConfirmTitle'),
      content: t('transfer.fileLib.msg.deleteMRConfirm', { count: sns.length }),
      okType: 'danger',
      onOk: async () => {
        try {
          const res = await batchDelete.mutateAsync(sns);
          if (res.failed.length > 0) {
            void message.warning(
              t('transfer.fileLib.msg.deletePartial', {
                count: res.succeeded.length,
                failedCount: res.failed.length,
              }),
            );
          } else {
            void message.success(
              t('transfer.fileLib.msg.deleteSuccess', { count: res.succeeded.length }),
            );
          }
          setSelectedKeys([]);
        } catch {
          void message.error(t('transfer.fileLib.msg.deleteFailed'));
        }
      },
    });
  };

  const columns: DataTableColumn<MRFileDeviceItem>[] = useMemo(
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
        key: 'siteName',
        title: t('mr.siteName'),
        dataIndex: 'siteName',
        width: 180,
        ellipsis: true,
        render: (v) => v || '—',
      },
      {
        key: 'productClass',
        title: t('mr.productClass'),
        dataIndex: 'productClass',
        width: 160,
        ellipsis: true,
        render: (v) => v || '—',
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
        key: 'reporting',
        title: t('mr.reportingStatus'),
        dataIndex: 'reporting',
        width: 110,
        fixed: 'right',
        render: (_, record) =>
          record.reporting ? (
            <Badge status="processing" text={t('mr.reporting')} />
          ) : (
            <Badge status="default" text={t('mr.reportStopped')} />
          ),
      },
    ],
    [t],
  );

  const body = (
    <>
      <Card size="small" style={{ marginBottom: 12 }}>
        <Space wrap>
          <Input
            allowClear
            placeholder={t('mr.searchDeviceSn')}
            style={{ width: 180 }}
            value={keyword}
            onChange={(e) => {
              setKeyword(e.target.value);
              setPage(1);
            }}
          />
          <Input
            allowClear
            placeholder={t('mr.searchSiteName')}
            style={{ width: 180 }}
            value={siteName}
            onChange={(e) => {
              setSiteName(e.target.value);
              setPage(1);
            }}
          />
          <Input
            allowClear
            placeholder={t('mr.searchProductClass')}
            style={{ width: 180 }}
            value={productClass}
            onChange={(e) => {
              setProductClass(e.target.value);
              setPage(1);
            }}
          />
          <Button icon={<ReloadOutlined />} onClick={() => void refetch()}>
            {t('transfer.fileLib.action.refresh')}
          </Button>
        </Space>
      </Card>
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
        selectable
        selectedRowKeys={selectedKeys}
        onSelectionChange={(keys) => setSelectedKeys(keys)}
        scroll={{ x: 900 }}
        extraToolbarLeft={
          <Space>
            <Button
              icon={<DownloadOutlined />}
              disabled={selectedKeys.length === 0 || bundle.isPending}
              loading={bundle.isPending}
              onClick={handleBatchDownload}
            >
              {t('bundle.batchDownload')} ({selectedKeys.length})
            </Button>
            <Button
              danger
              icon={<DeleteOutlined />}
              disabled={selectedKeys.length === 0 || batchDelete.isPending}
              loading={batchDelete.isPending}
              onClick={() => confirmDelete(selectedKeys.map(String))}
            >
              {t('transfer.fileLib.action.batchDelete', { count: selectedKeys.length })}
            </Button>
          </Space>
        }
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
      variant="outlined"
      style={{ flex: 1, display: 'flex', flexDirection: 'column' }}
      styles={{ body: { display: 'flex', flexDirection: 'column', flex: 1 } }}
    >
      {body}
    </Card>
  );
}
