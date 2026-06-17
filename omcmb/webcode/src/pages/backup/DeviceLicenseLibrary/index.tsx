/**
 * DeviceLicenseLibrary (T-0165) — 设备 license 文件库列表页。
 *
 * 结构镜像 ConfigSnapshotLibrary：一设备一份最新 license（serial_number 主键），
 * 独立 bucket=device-licenses。LICENSE_UPGRADE 任务从这里按 SN 取文件下发。
 * 只支持手动导入（无自动 promote）。
 */
import { useMemo, useState } from 'react';
import { Button, Card, Input, Space, message, Modal } from 'antd';
import {
  PlusOutlined,
  DownloadOutlined,
  DeleteOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';
import {
  useDeviceLicenses,
  useBatchDeleteDeviceLicenses,
} from '@core/hooks/api/useDeviceLicense';
import type { DeviceLicense } from '@core/services/api/deviceLicenseApi';
import { deviceLicenseApi } from '@core/services/api/deviceLicenseApi';
import { useBatchDownloadWithMessage } from '@/hooks/useBatchDownloadWithMessage';
import ImportDrawer from './ImportDrawer';
import { formatSystemTime } from '@core/utils/systemTime';

export default function DeviceLicenseLibraryPage() {
  const t = useT();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [serialFilter, setSerialFilter] = useState('');
  const [enbFilter, setEnbFilter] = useState('');
  const [productFilter, setProductFilter] = useState('');
  const [importOpen, setImportOpen] = useState(false);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);
  const bundle = useBatchDownloadWithMessage();
  const handleBatchDownload = () => {
    if (selectedKeys.length === 0) {
      void message.warning(t('bundle.selectFiles'));
      return;
    }
    bundle.trigger({ module: 'device_license', targets: selectedKeys.map(String) });
  };

  const queryParams = useMemo(
    () => ({
      page,
      pageSize,
      serialNumber: serialFilter || undefined,
      enbName: enbFilter || undefined,
      productType: productFilter || undefined,
    }),
    [page, pageSize, serialFilter, enbFilter, productFilter],
  );

  const { data, isLoading, refetch } = useDeviceLicenses(queryParams);
  const batchDelete = useBatchDeleteDeviceLicenses();

  const columns: DataTableColumn<DeviceLicense>[] = [
    { key: 'serialNumber', title: t('transfer.fileLib.col.serialNumber'), dataIndex: 'serialNumber', width: 200, copyable: true, mono: true },
    {
      key: 'enbName',
      title: t('transfer.fileLib.col.enbName'),
      dataIndex: 'enbName',
      width: 180,
      render: (v) => (v ? String(v) : <span style={{ color: '#999' }}>—</span>),
    },
    {
      key: 'productType',
      title: t('transfer.fileLib.col.productType'),
      dataIndex: 'productType',
      width: 160,
      render: (v) => (v ? String(v) : <span style={{ color: '#999' }}>—</span>),
    },
    { key: 'fileName', title: t('transfer.fileLib.col.licenseFile'), dataIndex: 'fileName', width: 260, mono: true },
    {
      key: 'fileSize',
      title: t('transfer.fileLib.col.size'),
      dataIndex: 'fileSize',
      width: 100,
      render: (v) => formatBytes(v as number),
    },
    {
      key: 'description',
      title: t('transfer.fileLib.col.description'),
      dataIndex: 'description',
      width: 200,
      render: (v) => (v ? String(v) : <span style={{ color: '#999' }}>—</span>),
    },
    {
      key: 'updateTime',
      title: t('transfer.fileLib.col.updateTime'),
      dataIndex: 'updateTime',
      width: 180,
      sorter: true,
      render: (v) => (v ? formatSystemTime(v as string, { format: 'YYYY-MM-DD HH:mm:ss', placeholder: '-' }) : '—'),
    },
    {
      key: 'actions',
      title: t('transfer.fileLib.col.actions'),
      width: 160,
      fixed: 'right',
      render: (_, record) => (
        <Space>
          <Button
            type="link"
            size="small"
            icon={<DownloadOutlined />}
            onClick={() => { void downloadLicense(record.serialNumber); }}
          >
            {t('transfer.fileLib.action.download')}
          </Button>
          <Button
            type="link"
            danger
            size="small"
            icon={<DeleteOutlined />}
            onClick={() => confirmDelete([record.serialNumber])}
          >
            {t('transfer.fileLib.action.delete')}
          </Button>
        </Space>
      ),
    },
  ];

  async function downloadLicense(sn: string) {
    try {
      await deviceLicenseApi.download(sn);
    } catch (e) {
      message.error(t('transfer.fileLib.msg.downloadFailed', { reason: (e as Error).message ?? t('transfer.fileLib.msg.tryAgainLater') }));
    }
  }

  function confirmDelete(sns: string[]) {
    Modal.confirm({
      title: t('transfer.fileLib.msg.deleteConfirmTitle'),
      content: t('transfer.fileLib.msg.deleteLicenseConfirm', { count: sns.length }),
      okType: 'danger',
      onOk: async () => {
        try {
          const res = await batchDelete.mutateAsync(sns);
          message.success(res.failed.length
            ? t('transfer.fileLib.msg.deletePartial', { count: res.succeeded.length, failedCount: res.failed.length })
            : t('transfer.fileLib.msg.deleteSuccess', { count: res.succeeded.length }));
          setSelectedKeys((prev) => prev.filter((k) => !sns.includes(String(k))));
        } catch {
          message.error(t('transfer.fileLib.msg.deleteFailed'));
        }
      },
    });
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16, flex: 1, minHeight: 0 }}>
      <Card size="small">
        <Space wrap>
          <Input
            placeholder={t('transfer.fileLib.filter.sn')}
            allowClear
            value={serialFilter}
            onChange={(e) => setSerialFilter(e.target.value)}
            style={{ width: 180 }}
          />
          <Input
            placeholder={t('transfer.fileLib.filter.enbName')}
            allowClear
            value={enbFilter}
            onChange={(e) => setEnbFilter(e.target.value)}
            style={{ width: 180 }}
          />
          <Input
            placeholder={t('transfer.fileLib.filter.productType')}
            allowClear
            value={productFilter}
            onChange={(e) => setProductFilter(e.target.value)}
            style={{ width: 180 }}
          />
          <Button icon={<ReloadOutlined />} onClick={() => refetch()}>{t('transfer.fileLib.action.refresh')}</Button>
        </Space>
      </Card>
      <DataTable<DeviceLicense>
        tableId="device-license-library"
        columns={columns}
        dataSource={data?.items ?? []}
        loading={isLoading}
        rowKey={(r) => r.serialNumber}
        selectable
        selectedRowKeys={selectedKeys}
        onSelectionChange={setSelectedKeys}
        total={data?.total ?? 0}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, s) => {
          setPage(p);
          setPageSize(s);
        }}
        extraToolbarLeft={
          <Space>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setImportOpen(true)}>
              {t('transfer.fileLib.action.importLicense')}
            </Button>
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
              disabled={selectedKeys.length === 0}
              onClick={() => confirmDelete(selectedKeys.map(String))}
            >
              {t('transfer.fileLib.action.batchDelete', { count: selectedKeys.length })}
            </Button>
          </Space>
        }
      />
      <ImportDrawer
        open={importOpen}
        onClose={() => setImportOpen(false)}
        onSuccess={() => refetch()}
      />
    </div>
  );
}

function formatBytes(n: number): string {
  if (!n || n < 0) return '—';
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(2)} MB`;
}
