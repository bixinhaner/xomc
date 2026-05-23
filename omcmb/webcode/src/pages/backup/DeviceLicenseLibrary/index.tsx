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
import dayjs from 'dayjs';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import {
  useDeviceLicenses,
  useBatchDeleteDeviceLicenses,
} from '@core/hooks/api/useDeviceLicense';
import type { DeviceLicense } from '@core/services/api/deviceLicenseApi';
import { deviceLicenseApi } from '@core/services/api/deviceLicenseApi';
import ImportDrawer from './ImportDrawer';

export default function DeviceLicenseLibraryPage() {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [serialFilter, setSerialFilter] = useState('');
  const [enbFilter, setEnbFilter] = useState('');
  const [productFilter, setProductFilter] = useState('');
  const [importOpen, setImportOpen] = useState(false);
  const [selectedKeys, setSelectedKeys] = useState<React.Key[]>([]);

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
    { key: 'fileName', title: 'License 文件', dataIndex: 'fileName', width: 260, mono: true },
    {
      key: 'fileSize',
      title: '大小',
      dataIndex: 'fileSize',
      width: 100,
      render: (v) => formatBytes(v as number),
    },
    {
      key: 'description',
      title: '说明',
      dataIndex: 'description',
      width: 200,
      render: (v) => (v ? String(v) : <span style={{ color: '#999' }}>—</span>),
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
            onClick={() => { void downloadLicense(record.serialNumber); }}
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

  async function downloadLicense(sn: string) {
    try {
      await deviceLicenseApi.download(sn);
    } catch (e) {
      message.error(`下载失败：${(e as Error).message ?? '请稍后重试'}`);
    }
  }

  function confirmDelete(sns: string[]) {
    Modal.confirm({
      title: '确认删除？',
      content: `将删除 ${sns.length} 台设备的 license 文件（DB 行 + MinIO 对象），不可恢复。`,
      okType: 'danger',
      onOk: async () => {
        try {
          const res = await batchDelete.mutateAsync(sns);
          message.success(`已删除 ${res.succeeded.length} 条${res.failed.length ? `；${res.failed.length} 条失败` : ''}`);
          setSelectedKeys((prev) => prev.filter((k) => !sns.includes(String(k))));
        } catch {
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
          <Button icon={<ReloadOutlined />} onClick={() => refetch()}>刷新</Button>
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
              导入 License
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
