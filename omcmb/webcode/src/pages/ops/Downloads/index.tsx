import { useMemo, useState } from 'react';
import { Button, Card, Checkbox, Descriptions, Form, Input, Modal, Space, Tag, message } from 'antd';
import { DownloadOutlined, EyeOutlined, FileDoneOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';
import { useCollectDownload, useOpsDownloads } from '@core/hooks/api/useOpsExt';
import type { DownloadStatus, OpsDownload } from '@core/services/api/opsExtApi';

type OpsDownloadRow = OpsDownload & Record<string, unknown>;

const STATUS_COLOR: Record<DownloadStatus, string> = {
  pending: 'default',
  uploading: 'processing',
  complete: 'success',
  failed: 'error',
  expired: 'warning',
};

// 与后端 collect 入参一致的可勾选内容类型
const COLLECT_TYPES = ['config', 'log', 'pm', 'mr', 'diagnostic', 'pcap', 'gps'];

function formatFileSize(bytes: number): string {
  if (!bytes) return '—';
  if (bytes >= 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
  if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
  if (bytes >= 1024) return `${(bytes / 1024).toFixed(2)} KB`;
  return `${bytes} B`;
}

export default function Downloads() {
  const t = useT();
  const [filters, setFilters] = useState<{ deviceSn?: string; contentType?: string; status?: DownloadStatus }>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [detail, setDetail] = useState<OpsDownload | null>(null);
  const [collectOpen, setCollectOpen] = useState(false);

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(filters.deviceSn ? { deviceSn: filters.deviceSn } : {}),
      ...(filters.contentType ? { contentType: filters.contentType } : {}),
      ...(filters.status ? { status: filters.status } : {}),
    }),
    [page, pageSize, filters],
  );

  const { data, isLoading, refetch } = useOpsDownloads(params);
  const rows = (data?.items ?? []) as OpsDownloadRow[];
  const total = data?.total ?? 0;

  const contentTypeLabel = useMemo(
    (): Record<string, string> => ({
      config: t('ops.dl.ctConfig'),
      syslog: t('ops.dl.ctSyslog'),
      runlog: t('ops.dl.ctRunlog'),
      perf: t('ops.dl.ctPerf'),
      cert: t('ops.dl.ctCert'),
      firmware: t('ops.dl.ctFirmware'),
      backup: t('ops.dl.ctBackup'),
      mr: t('ops.dl.ctMr'),
      log: t('ops.dl.ctLog'),
      diagnostic: t('ops.dl.ctDiagnostic'),
      pcap: t('ops.dl.ctPcap'),
      gps: t('ops.dl.ctGps'),
    }),
    [t],
  );

  const statusLabel = useMemo(
    (): Record<DownloadStatus, string> => ({
      pending: t('ops.dl.stPending'),
      uploading: t('ops.dl.stUploading'),
      complete: t('ops.dl.stComplete'),
      failed: t('ops.dl.stFailed'),
      expired: t('ops.dl.stExpired'),
    }),
    [t],
  );

  const filterFields: FilterField[] = useMemo(
    () => [
      { name: 'deviceSn', label: t('trace.column.deviceSn'), type: 'input', placeholder: t('ops.deviceSnPlaceholder') },
      {
        name: 'contentType',
        label: t('ops.dl.contentType'),
        type: 'select',
        placeholder: t('ops.dl.allContentTypes'),
        options: [
          { label: t('ops.dl.ctConfig'), value: 'config' },
          { label: t('ops.dl.ctSyslog'), value: 'syslog' },
          { label: t('ops.dl.ctRunlog'), value: 'runlog' },
          { label: t('ops.dl.ctPerf'), value: 'perf' },
          { label: t('ops.dl.ctCert'), value: 'cert' },
          { label: t('ops.dl.ctFirmware'), value: 'firmware' },
          { label: t('ops.dl.ctBackup'), value: 'backup' },
        ],
      },
      {
        name: 'status',
        label: t('table.status'),
        type: 'select',
        placeholder: t('common.all'),
        options: [
          { label: t('ops.dl.stComplete'), value: 'complete' },
          { label: t('ops.dl.stUploading'), value: 'uploading' },
          { label: t('ops.dl.stPending'), value: 'pending' },
          { label: t('ops.dl.stFailed'), value: 'failed' },
          { label: t('ops.dl.stExpired'), value: 'expired' },
        ],
      },
    ],
    [t],
  );

  const columns: DataTableColumn<OpsDownloadRow>[] = useMemo(
    () => [
      { key: 'device_sn', title: t('trace.column.deviceSn'), dataIndex: 'device_sn', width: 160, mono: true, copyable: true },
      {
        key: 'content_type',
        title: t('ops.dl.contentType'),
        dataIndex: 'content_type',
        width: 110,
        render: (val) => <Tag color="blue">{contentTypeLabel[String(val)] ?? String(val)}</Tag>,
      },
      { key: 'file_path', title: t('ops.dl.filePath'), dataIndex: 'file_path', width: 240, ellipsis: true, mono: true, render: (val) => (val ? String(val) : '—') },
      { key: 'file_size', title: t('ops.dl.fileSize'), dataIndex: 'file_size', width: 110, render: (val) => formatFileSize(Number(val)) },
      {
        key: 'status',
        title: t('table.status'),
        dataIndex: 'status',
        width: 100,
        render: (val) => {
          const s = val as DownloadStatus;
          return <Tag color={STATUS_COLOR[s]}>{statusLabel[s] ?? s}</Tag>;
        },
      },
      { key: 'operator', title: t('table.operator'), dataIndex: 'operator', width: 120, render: (val) => (val ? String(val) : '—') },
      {
        key: 'created_at',
        title: t('table.createTime'),
        dataIndex: 'created_at',
        width: 170,
        render: (val) => (val ? new Date(String(val)).toLocaleString('zh-CN') : '—'),
      },
      {
        key: 'actions',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 130,
        fixed: 'right',
        render: (_, record) => (
          <Space size={4}>
            <Button type="link" size="small" icon={<EyeOutlined />} onClick={() => setDetail(record)}>
              {t('common.detail')}
            </Button>
            <Button
              type="link"
              size="small"
              icon={<DownloadOutlined />}
              disabled={record.status !== 'complete' || !record.file_path}
            >
              {t('common.download')}
            </Button>
          </Space>
        ),
      },
    ],
    [t, contentTypeLabel, statusLabel],
  );

  return (
    <ListPageLayout
      title={t('nav.ops.downloads')}
      subtitle={t('ops.downloadsSubtitle')}
      extra={
        <Button type="primary" icon={<FileDoneOutlined />} onClick={() => setCollectOpen(true)}>
          {t('ops.dl.collect')}
        </Button>
      }
    >
      <FilterBar
        filterId="ops-downloads-filter"
        fields={filterFields}
        onSearch={(vals) => {
          setFilters(vals as typeof filters);
          setPage(1);
        }}
        onReset={() => {
          setFilters({});
          setPage(1);
        }}
      />
      <Card
        size="small"
        variant="outlined"
        style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
        styles={{ body: { padding: 0, display: 'flex', flexDirection: 'column', flex: 1, overflow: 'hidden' } }}
      >
        <DataTable
          tableId="ops-downloads-list"
          columns={columns}
          dataSource={rows}
          loading={isLoading}
          rowKey="id"
          total={total}
          pageSize={pageSize}
          currentPage={page}
          onPageChange={(p, s) => {
            setPage(p);
            setPageSize(s);
          }}
          onRefresh={() => void refetch()}
          scroll={{ x: 1180 }}
        />
      </Card>

      <DownloadDetailModal
        download={detail}
        contentTypeLabel={contentTypeLabel}
        statusLabel={statusLabel}
        onClose={() => setDetail(null)}
      />

      <CollectModal open={collectOpen} onClose={() => setCollectOpen(false)} onDone={() => void refetch()} />
    </ListPageLayout>
  );
}

function DownloadDetailModal({
  download,
  contentTypeLabel,
  statusLabel,
  onClose,
}: {
  download: OpsDownload | null;
  contentTypeLabel: Record<string, string>;
  statusLabel: Record<DownloadStatus, string>;
  onClose: () => void;
}) {
  const t = useT();
  return (
    <Modal
      open={download !== null}
      onCancel={onClose}
      title={download ? `${contentTypeLabel[download.content_type] ?? download.content_type} · ${download.device_sn}` : ''}
      footer={<Button onClick={onClose}>{t('common.cancel')}</Button>}
      width={620}
    >
      {download && (
        <Descriptions column={1} size="small" bordered>
          <Descriptions.Item label={t('trace.column.deviceSn')}>{download.device_sn}</Descriptions.Item>
          <Descriptions.Item label={t('ops.dl.contentType')}>
            {contentTypeLabel[download.content_type] ?? download.content_type}
          </Descriptions.Item>
          <Descriptions.Item label={t('table.status')}>
            <Tag color={STATUS_COLOR[download.status]}>{statusLabel[download.status] ?? download.status}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label={t('ops.dl.filePath')}>{download.file_path || '—'}</Descriptions.Item>
          <Descriptions.Item label={t('ops.dl.fileSize')}>{formatFileSize(download.file_size)}</Descriptions.Item>
          {download.checksum && (
            <Descriptions.Item label={t('ops.dl.checksum')}>{download.checksum}</Descriptions.Item>
          )}
          <Descriptions.Item label={t('table.operator')}>{download.operator || '—'}</Descriptions.Item>
          <Descriptions.Item label={t('table.createTime')}>
            {new Date(download.created_at).toLocaleString('zh-CN')}
          </Descriptions.Item>
          {download.expires_at && (
            <Descriptions.Item label={t('ops.expireTime')}>
              {new Date(download.expires_at).toLocaleString('zh-CN')}
            </Descriptions.Item>
          )}
        </Descriptions>
      )}
    </Modal>
  );
}

function CollectModal({ open, onClose, onDone }: { open: boolean; onClose: () => void; onDone: () => void }) {
  const t = useT();
  const [deviceSn, setDeviceSn] = useState('');
  const [types, setTypes] = useState<string[]>(['config']);
  const collect = useCollectDownload();

  const close = () => {
    setDeviceSn('');
    setTypes(['config']);
    onClose();
  };

  const submit = () => {
    if (!deviceSn.trim() || types.length === 0) return;
    collect.mutate(
      { device_sn: deviceSn.trim(), content_types: types },
      {
        onSuccess: () => {
          void message.success(t('ops.dl.collectSuccess'));
          close();
          onDone();
        },
        onError: () => void message.error(t('ops.dl.collectFailed')),
      },
    );
  };

  return (
    <Modal
      open={open}
      title={t('ops.dl.collectTitle')}
      onCancel={close}
      onOk={submit}
      okText={t('ops.dl.collect')}
      cancelText={t('common.cancel')}
      confirmLoading={collect.isPending}
      okButtonProps={{ disabled: !deviceSn.trim() || types.length === 0 }}
      width={480}
    >
      <Form layout="vertical">
        <Form.Item label={t('trace.column.deviceSn')} required>
          <Input
            placeholder={t('ops.deviceSnPlaceholder')}
            value={deviceSn}
            onChange={(e) => setDeviceSn(e.target.value)}
          />
        </Form.Item>
        <Form.Item label={`${t('ops.dl.collectContent')}（${types.length}）`} required>
          <Checkbox.Group
            value={types}
            onChange={(vals) => setTypes(vals as string[])}
            options={COLLECT_TYPES.map((tp) => ({ label: tp, value: tp }))}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
}
