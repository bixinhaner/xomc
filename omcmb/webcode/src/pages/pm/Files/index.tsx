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
import { Badge, Button, Card, Input, Modal, Select, Space, message } from 'antd';
import { DeleteOutlined, DownloadOutlined, ReloadOutlined } from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';
import {
  useBatchDeletePMFiles,
  usePMFileDevices,
} from '@core/hooks/api/usePerformance';
import { useProductList, useProductNameResolver } from '@core/hooks/api/useProducts';
import { useBatchDownloadWithMessage } from '@/hooks/useBatchDownloadWithMessage';
import type { PMFileDeviceItem } from '@core/services/api/pmApi';
import DeviceFilesDrawer from './DeviceFilesDrawer';
import { formatSystemTime } from '@core/utils/systemTime';

interface Props {
  /** 嵌入到 FileManagement 时去掉外层卡片背景的留白。 */
  embedded?: boolean;
}

export default function PMFilesPage({ embedded }: Props) {
  const t = useT();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [keyword, setKeyword] = useState('');
  const [siteName, setSiteName] = useState('');
  // #602：产品名称下拉过滤，传 product.id。
  const [productId, setProductId] = useState('');
  const [activeDevice, setActiveDevice] = useState<PMFileDeviceItem | null>(null);

  const params = useMemo(
    () => ({
      page,
      pageSize,
      keyword: keyword || undefined,
      siteName: siteName || undefined,
      productId: productId || undefined,
    }),
    [page, pageSize, keyword, siteName, productId],
  );
  const { data, isLoading, refetch } = usePMFileDevices(params);
  const { data: productsData } = useProductList();
  const productNameOptions = useMemo(
    () => (productsData?.items ?? [])
      .map((p) => ({ label: `${p.name} (${p.tech})`, value: p.id }))
      .sort((a, b) => a.label.localeCompare(b.label, 'zh-CN')),
    [productsData],
  );
  const resolveProductName = useProductNameResolver();
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);
  const bundle = useBatchDownloadWithMessage();
  const batchDelete = useBatchDeletePMFiles();
  const handleBatchDownload = () => {
    if (selectedKeys.length === 0) {
      void message.warning(t('bundle.selectFiles'));
      return;
    }
    bundle.trigger({ module: 'pm', targets: selectedKeys.map(String) });
  };

  // 与 License/Config 的 confirmDelete 模式一致：Modal.confirm 二次确认 +
  // 调 batch-delete → 按 succeeded/failed 分情况 toast。
  const confirmDelete = (sns: string[]) => {
    Modal.confirm({
      title: t('transfer.fileLib.msg.deleteConfirmTitle'),
      content: t('transfer.fileLib.msg.deletePMConfirm', { count: sns.length }),
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
        key: 'siteName',
        title: t('pm.siteName'),
        dataIndex: 'siteName',
        width: 180,
        ellipsis: true,
        render: (v) => (v ? String(v) : '—'),
      },
      {
        key: 'productClass',
        title: t('pm.productClass'),
        dataIndex: 'productClass',
        width: 160,
        ellipsis: true,
        render: (v) => {
          const raw = v ? String(v) : '';
          if (!raw) return '—';
          return resolveProductName(raw);
        },
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
        render: (v) => (v ? formatSystemTime(v as string, { format: 'YYYY-MM-DD HH:mm:ss', placeholder: '-' }) : '—'),
      },
      {
        key: 'lastCollectTime',
        title: t('pm.updateTime'),
        dataIndex: 'lastCollectTime',
        width: 180,
        render: (v) => (v ? formatSystemTime(v as string, { format: 'YYYY-MM-DD HH:mm:ss', placeholder: '-' }) : '—'),
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
    [t, resolveProductName],
  );

  const body = (
    <>
      <Card size="small" style={{ marginBottom: 12 }}>
        <Space wrap>
          <Input
            allowClear
            placeholder={t('pm.searchDeviceSn')}
            style={{ width: 180 }}
            value={keyword}
            onChange={(e) => {
              setKeyword(e.target.value);
              setPage(1);
            }}
          />
          <Input
            allowClear
            placeholder={t('pm.searchSiteName')}
            style={{ width: 180 }}
            value={siteName}
            onChange={(e) => {
              setSiteName(e.target.value);
              setPage(1);
            }}
          />
          <Select<string>
            showSearch
            allowClear
            optionFilterProp="label"
            placeholder={t('pm.searchProductClass')}
            style={{ width: 220 }}
            value={productId || undefined}
            onChange={(v) => {
              setProductId(v ?? '');
              setPage(1);
            }}
            options={productNameOptions}
          />
          <Button icon={<ReloadOutlined />} onClick={() => void refetch()}>
            {t('transfer.fileLib.action.refresh')}
          </Button>
        </Space>
      </Card>
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
