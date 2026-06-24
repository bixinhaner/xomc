/**
 * ConfigSnapshotLibrary (T-0164 / F2) — 配置快照库列表页。
 *
 * 仿固件库，但记录的是"每设备最新一份配置文件"。serial_number 主键，覆盖式更新。
 * 写入入口：① 备份任务完成自动 promote；② 本页面"导入"按钮手动批量上传。
 * 读取入口：① 本页面列表；② Restore 创建页"按设备快照" Tab。
 *
 * 历史数据不回填（用户确认 2026-05-22）。
 */
import { useMemo, useState } from 'react';
import { Button, Card, Input, Select, Space, Tag, message, Modal } from 'antd';
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
  useConfigSnapshots,
  useBatchDeleteConfigSnapshots,
} from '@core/hooks/api/useConfigSnapshot';
import { useProductList } from '@core/hooks/api/useProducts';
import type {
  ConfigSnapshot,
  SnapshotSource,
} from '@core/services/api/configSnapshotApi';
import { configSnapshotApi } from '@core/services/api/configSnapshotApi';
import { useBatchDownloadWithMessage } from '@/hooks/useBatchDownloadWithMessage';
import ImportDrawer from './ImportDrawer';
import { formatSystemTime } from '@core/utils/systemTime';

// 来源 Tag 颜色映射；label 走 i18n key 在渲染时按当前 locale 取。
const SOURCE_TAG: Record<SnapshotSource, { color: string; labelKey: string }> = {
  backup: { color: 'blue', labelKey: 'transfer.fileLib.source.backup' },
  manual_upload: { color: 'green', labelKey: 'transfer.fileLib.source.manualUpload' },
};

export default function ConfigSnapshotLibraryPage() {
  const t = useT();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [serialFilter, setSerialFilter] = useState('');
  const [enbFilter, setEnbFilter] = useState('');
  // #602：按产品名称下拉过滤，后端按 product.id 展开 patterns IN (...)。
  const [productId, setProductId] = useState('');
  const [sourceFilter, setSourceFilter] = useState<SnapshotSource | ''>('');
  const [importOpen, setImportOpen] = useState(false);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);
  const bundle = useBatchDownloadWithMessage();
  const handleBatchDownload = () => {
    if (selectedKeys.length === 0) {
      void message.warning(t('bundle.selectFiles'));
      return;
    }
    bundle.trigger({ module: 'config_snapshot', targets: selectedKeys.map(String) });
  };

  const queryParams = useMemo(
    () => ({
      page,
      pageSize,
      serialNumber: serialFilter || undefined,
      enbName: enbFilter || undefined,
      productId: productId || undefined,
      source: sourceFilter || undefined,
    }),
    [page, pageSize, serialFilter, enbFilter, productId, sourceFilter],
  );

  const { data, isLoading, refetch } = useConfigSnapshots(queryParams);
  const batchDelete = useBatchDeleteConfigSnapshots();
  // #602：产品名称下拉 + pattern → 产品名反查，列表列以名称展示。
  const { data: productsData } = useProductList();
  const productNameOptions = useMemo(
    () => (productsData?.items ?? [])
      .map((p) => ({ label: `${p.name} (${p.tech})`, value: p.id }))
      .sort((a, b) => a.label.localeCompare(b.label, 'zh-CN')),
    [productsData],
  );
  const productNameByPattern = useMemo(() => {
    const map = new Map<string, string>();
    (productsData?.items ?? []).forEach((p) => {
      (p.patterns ?? []).forEach((pat) => map.set(pat, p.name));
    });
    return map;
  }, [productsData]);

  const columns: DataTableColumn<ConfigSnapshot>[] = [
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
      render: (v) => {
        const raw = v ? String(v) : '';
        const name = raw ? productNameByPattern.get(raw) : undefined;
        if (name) return name;
        return raw ? raw : <span style={{ color: '#999' }}>—</span>;
      },
    },
    { key: 'fileName', title: t('transfer.fileLib.col.snapshotFile'), dataIndex: 'fileName', width: 260, mono: true },
    {
      key: 'fileSize',
      title: t('transfer.fileLib.col.size'),
      dataIndex: 'fileSize',
      width: 100,
      render: (v) => formatBytes(v as number),
    },
    {
      key: 'source',
      title: t('transfer.fileLib.col.source'),
      dataIndex: 'source',
      width: 110,
      render: (v) => {
        const cfg = SOURCE_TAG[v as SnapshotSource];
        return <Tag color={cfg.color}>{t(cfg.labelKey)}</Tag>;
      },
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
            onClick={() => { void downloadSnapshot(record.serialNumber); }}
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

  async function downloadSnapshot(sn: string) {
    try {
      await configSnapshotApi.download(sn);
    } catch (e) {
      message.error(t('transfer.fileLib.msg.downloadFailed', { reason: (e as Error).message ?? t('transfer.fileLib.msg.tryAgainLater') }));
    }
  }

  function confirmDelete(sns: string[]) {
    Modal.confirm({
      title: t('transfer.fileLib.msg.deleteConfirmTitle'),
      content: t('transfer.fileLib.msg.deleteSnapshotConfirm', { count: sns.length }),
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
          <Select<string>
            showSearch
            allowClear
            optionFilterProp="label"
            placeholder={t('transfer.fileLib.filter.productType')}
            value={productId || undefined}
            onChange={(v) => setProductId(v ?? '')}
            options={productNameOptions}
            style={{ width: 220 }}
          />
          <Select<SnapshotSource | ''>
            placeholder={t('transfer.fileLib.filter.source')}
            allowClear
            value={sourceFilter || undefined}
            onChange={(v) => setSourceFilter(v ?? '')}
            style={{ width: 140 }}
            options={[
              { value: 'backup', label: t('transfer.fileLib.source.backup') },
              { value: 'manual_upload', label: t('transfer.fileLib.source.manualUpload') },
            ]}
          />
          <Button icon={<ReloadOutlined />} onClick={() => refetch()}>{t('transfer.fileLib.action.refresh')}</Button>
        </Space>
      </Card>
      <DataTable<ConfigSnapshot>
        tableId="config-snapshot-library"
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
              {t('transfer.fileLib.action.importConfig')}
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
