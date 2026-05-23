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
import dayjs from 'dayjs';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import {
  useConfigSnapshots,
  useBatchDeleteConfigSnapshots,
} from '@core/hooks/api/useConfigSnapshot';
import type {
  ConfigSnapshot,
  SnapshotSource,
} from '@core/services/api/configSnapshotApi';
import { configSnapshotApi } from '@core/services/api/configSnapshotApi';
import ImportDrawer from './ImportDrawer';

const SOURCE_TAG: Record<SnapshotSource, { color: string; label: string }> = {
  backup: { color: 'blue', label: '备份任务' },
  manual_upload: { color: 'green', label: '手动导入' },
};

export default function ConfigSnapshotLibraryPage() {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [serialFilter, setSerialFilter] = useState('');
  const [enbFilter, setEnbFilter] = useState('');
  const [productFilter, setProductFilter] = useState('');
  const [sourceFilter, setSourceFilter] = useState<SnapshotSource | ''>('');
  const [importOpen, setImportOpen] = useState(false);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);

  const queryParams = useMemo(
    () => ({
      page,
      pageSize,
      serialNumber: serialFilter || undefined,
      enbName: enbFilter || undefined,
      productType: productFilter || undefined,
      source: sourceFilter || undefined,
    }),
    [page, pageSize, serialFilter, enbFilter, productFilter, sourceFilter],
  );

  const { data, isLoading, refetch } = useConfigSnapshots(queryParams);
  const batchDelete = useBatchDeleteConfigSnapshots();

  const columns: DataTableColumn<ConfigSnapshot>[] = [
    { key: 'serialNumber', title: '设备序列号', dataIndex: 'serialNumber', width: 200, copyable: true, mono: true },
    {
      key: 'enbName',
      title: '基站名称',
      dataIndex: 'enbName',
      width: 180,
      render: (v) => (v ? String(v) : <span style={{ color: '#999' }}>—</span>),
    },
    {
      key: 'productType',
      title: 'Product Type',
      dataIndex: 'productType',
      width: 160,
      render: (v) => (v ? String(v) : <span style={{ color: '#999' }}>—</span>),
    },
    { key: 'fileName', title: '最新配置文件', dataIndex: 'fileName', width: 260, mono: true },
    {
      key: 'fileSize',
      title: '大小',
      dataIndex: 'fileSize',
      width: 100,
      render: (v) => formatBytes(v as number),
    },
    {
      key: 'source',
      title: '来源',
      dataIndex: 'source',
      width: 110,
      render: (v) => {
        const cfg = SOURCE_TAG[v as SnapshotSource];
        return <Tag color={cfg.color}>{cfg.label}</Tag>;
      },
    },
    {
      key: 'updateTime',
      title: '最新更新时间',
      dataIndex: 'updateTime',
      width: 180,
      sorter: true,
      render: (v) => (v ? dayjs(v as string).format('YYYY-MM-DD HH:mm:ss') : '—'),
    },
    {
      key: 'actions',
      title: '操作',
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
            下载
          </Button>
          <Button
            type="link"
            danger
            size="small"
            icon={<DeleteOutlined />}
            onClick={() => confirmDelete([record.serialNumber])}
          >
            删除
          </Button>
        </Space>
      ),
    },
  ];

  async function downloadSnapshot(sn: string) {
    try {
      await configSnapshotApi.download(sn);
    } catch (e) {
      message.error(`下载失败：${(e as Error).message ?? '请稍后重试'}`);
    }
  }

  function confirmDelete(sns: string[]) {
    Modal.confirm({
      title: '确认删除？',
      content: `将删除 ${sns.length} 台设备的配置快照（DB 行 + MinIO 对象），不可恢复。`,
      okType: 'danger',
      onOk: async () => {
        try {
          const res = await batchDelete.mutateAsync(sns);
          message.success(`已删除 ${res.succeeded.length} 条${res.failed.length ? `；${res.failed.length} 条失败` : ''}`);
          setSelectedKeys((prev) => prev.filter((k) => !sns.includes(String(k))));
        } catch (e) {
          message.error('删除失败');
        }
      },
    });
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16, flex: 1, minHeight: 0 }}>
      <Card size="small">
        <Space wrap>
          <Input
            placeholder="设备 SN"
            allowClear
            value={serialFilter}
            onChange={(e) => setSerialFilter(e.target.value)}
            style={{ width: 180 }}
          />
          <Input
            placeholder="基站名称"
            allowClear
            value={enbFilter}
            onChange={(e) => setEnbFilter(e.target.value)}
            style={{ width: 180 }}
          />
          <Input
            placeholder="Product Type"
            allowClear
            value={productFilter}
            onChange={(e) => setProductFilter(e.target.value)}
            style={{ width: 180 }}
          />
          <Select<SnapshotSource | ''>
            placeholder="来源"
            allowClear
            value={sourceFilter || undefined}
            onChange={(v) => setSourceFilter(v ?? '')}
            style={{ width: 140 }}
            options={[
              { value: 'backup', label: '备份任务' },
              { value: 'manual_upload', label: '手动导入' },
            ]}
          />
          <Button icon={<ReloadOutlined />} onClick={() => refetch()}>刷新</Button>
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
              导入配置
            </Button>
            <Button
              danger
              icon={<DeleteOutlined />}
              disabled={selectedKeys.length === 0}
              onClick={() => confirmDelete(selectedKeys.map(String))}
            >
              批量删除（{selectedKeys.length}）
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
