// T-0137 / M1: TR069 报文跟踪页面
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Button,
  Tag,
  Space,
  Modal,
  Form,
  Input,
  InputNumber,
  Drawer,
  message,
  Typography,
  Tooltip,
  Popconfirm,
} from 'antd';
import { PlusOutlined, EyeOutlined, StopOutlined, DownloadOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import {
  useTraceTasks,
  useCreateTraceTask,
  useStopTraceTask,
  useTraceMessages,
  useRequestTraceExport,
  useTraceExportJob,
  useTraceSseRefresh,
} from '@core/hooks/api/useTrace';
import { traceApi } from '@core/services/api/traceApi';
import type {
  TraceTask,
  TraceMessage,
  TraceTaskStatus,
} from '@core/types/trace';
import { useT } from '@/hooks/useT';

const STATUS_COLOR: Record<TraceTaskStatus, string> = {
  running: 'processing',
  stopped: 'default',
  purged: 'warning',
};

const DIRECTION_COLOR: Record<TraceMessage['direction'], string> = {
  in: 'cyan',
  out: 'blue',
};

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(2)} MB`;
}

export default function MessageTrace() {
  const t = useT();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [createOpen, setCreateOpen] = useState(false);
  const [drawerTask, setDrawerTask] = useState<TraceTask | null>(null);
  const [detailMsg, setDetailMsg] = useState<TraceMessage | null>(null);

  const { data, isLoading, refetch } = useTraceTasks({ page, pageSize });
  const createMut = useCreateTraceTask();
  const stopMut = useStopTraceTask();
  const exportMut = useRequestTraceExport();
  const [activeExportId, setActiveExportId] = useState<string | undefined>(undefined);
  // SSE 实时刷新：trace.task.started/stopped/purged 事件触发 React Query invalidate
  useTraceSseRefresh();

  const [form] = Form.useForm<{ deviceSn: string; durationMinutes?: number }>();

  const handleCreate = async () => {
    const values = await form.validateFields();
    try {
      await createMut.mutateAsync(values);
      void message.success(t('common.success'));
      form.resetFields();
      setCreateOpen(false);
    } catch (e) {
      void message.error(e instanceof Error ? e.message : 'create failed');
    }
  };

  const handleStop = useCallback(
    (record: TraceTask, purge = false) => {
      stopMut.mutate(
        { id: record.id, purge },
        {
          onSuccess: () => void message.success(t('common.success')),
          onError: (err) =>
            void message.error(err instanceof Error ? err.message : 'stop failed'),
        }
      );
    },
    [stopMut, t]
  );

  const handleExport = useCallback(
    (record: TraceTask) => {
      exportMut.mutate(record.id, {
        onSuccess: (job) => {
          setActiveExportId(job.id);
          void message.info(t('trace.export.queued'));
        },
        onError: (err) =>
          void message.error(err instanceof Error ? err.message : 'export request failed'),
      });
    },
    [exportMut, t]
  );

  const columns: DataTableColumn<TraceTask & Record<string, unknown>>[] = useMemo(
    () => [
      {
        key: 'deviceSn',
        title: t('trace.column.deviceSn'),
        dataIndex: 'deviceSn',
        width: 180,
        mono: true,
      },
      {
        key: 'status',
        title: t('trace.column.status'),
        dataIndex: 'status',
        width: 100,
        render: (_v, r) => (
          <Tag color={STATUS_COLOR[r.status] || 'default'}>
            {t(`trace.status.${r.status}` as never)}
          </Tag>
        ),
      },
      {
        key: 'startTime',
        title: t('trace.column.startTime'),
        dataIndex: 'startTime',
        width: 180,
        render: (v) => (v ? new Date(v as string).toLocaleString() : '-'),
      },
      {
        key: 'expiresAt',
        title: t('trace.column.expiresAt'),
        dataIndex: 'expiresAt',
        width: 180,
        render: (v) => (v ? new Date(v as string).toLocaleString() : '-'),
      },
      {
        key: 'messageCount',
        title: t('trace.column.messageCount'),
        dataIndex: 'messageCount',
        width: 100,
      },
      {
        key: 'createdBy',
        title: t('trace.column.createdBy'),
        dataIndex: 'createdBy',
        width: 120,
      },
      {
        key: 'actions',
        title: t('table.action'),
        fixed: 'right',
        width: 280,
        render: (_v, r) => (
          <Space size="small">
            <Button
              size="small"
              icon={<EyeOutlined />}
              onClick={() => setDrawerTask(r)}
            >
              {t('trace.action.viewMessages')}
            </Button>
            {r.status === 'running' && (
              <Popconfirm
                title={t('trace.confirm.stop')}
                onConfirm={() => handleStop(r, false)}
              >
                <Button size="small" icon={<StopOutlined />}>
                  {t('trace.action.stop')}
                </Button>
              </Popconfirm>
            )}
            {r.status !== 'purged' && (r.messageCount ?? 0) > 0 && (
              <Tooltip title={t('trace.action.exportXml')}>
                <Button
                  size="small"
                  icon={<DownloadOutlined />}
                  onClick={() => handleExport(r)}
                />
              </Tooltip>
            )}
          </Space>
        ),
      },
    ],
    [t, handleStop, handleExport]
  );

  return (
    <ListPageLayout title={t('trace.title')} subtitle={t('trace.desc')}>
      <DataTable
        tableId="trace-tasks"
        rowKey="id"
        loading={isLoading}
        columns={columns}
        dataSource={(data?.items ?? []) as (TraceTask & Record<string, unknown>)[]}
        total={data?.total ?? 0}
        currentPage={page}
        pageSize={pageSize}
        onPageChange={(p, s) => {
          setPage(p);
          setPageSize(s);
        }}
        onRefresh={() => void refetch()}
        scroll={{ x: 1200 }}
        extraToolbarLeft={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setCreateOpen(true)}
          >
            {t('trace.action.create')}
          </Button>
        }
      />

      <Modal
        title={t('trace.action.create')}
        open={createOpen}
        onCancel={() => {
          form.resetFields();
          setCreateOpen(false);
        }}
        onOk={handleCreate}
        confirmLoading={createMut.isPending}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="deviceSn"
            label={t('trace.column.deviceSn')}
            rules={[{ required: true }]}
          >
            <Input placeholder="例如 BLQ-001" />
          </Form.Item>
          <Form.Item
            name="durationMinutes"
            label={t('trace.field.duration')}
            tooltip={t('trace.field.duration.tip')}
            initialValue={5}
          >
            <InputNumber min={1} max={60} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        title={drawerTask?.deviceSn}
        open={Boolean(drawerTask)}
        onClose={() => setDrawerTask(null)}
        width={960}
        destroyOnClose
      >
        {drawerTask && (
          <MessageList task={drawerTask} onSelect={setDetailMsg} t={t} />
        )}
      </Drawer>

      <Modal
        title={t('trace.message.detail.title')}
        open={Boolean(detailMsg)}
        footer={null}
        width={900}
        onCancel={() => setDetailMsg(null)}
        destroyOnClose
      >
        {detailMsg && <MessageDetail msg={detailMsg} t={t} />}
      </Modal>

      <ExportProgressModal
        jobId={activeExportId}
        onClose={() => setActiveExportId(undefined)}
        t={t}
      />
    </ListPageLayout>
  );
}

// ExportProgressModal 异步导出进度对话框：轮询 job，done 时自动 window.open 预签名 URL
interface ExportProgressModalProps {
  jobId: string | undefined;
  onClose: () => void;
  t: ReturnType<typeof useT>;
}

function ExportProgressModal({ jobId, onClose, t }: ExportProgressModalProps) {
  const { data: job } = useTraceExportJob(jobId);
  // 用 ref 跟踪已自动打开的 job_id — 避免 setState in effect 的 lint 警告
  const autoOpenedRef = useRef<string | undefined>(undefined);
  useEffect(() => {
    if (
      job?.status === 'done' &&
      job.downloadUrl &&
      autoOpenedRef.current !== job.id
    ) {
      autoOpenedRef.current = job.id;
      window.open(job.downloadUrl, '_blank');
    }
  }, [job]);
  return (
    <Modal
      title={t('trace.export.title')}
      open={Boolean(jobId)}
      onCancel={onClose}
      footer={null}
      width={520}
      destroyOnClose
    >
      {!job && <Typography.Paragraph>{t('trace.export.queued')}</Typography.Paragraph>}
      {job?.status === 'queued' && <Typography.Paragraph>{t('trace.export.queued')}</Typography.Paragraph>}
      {job?.status === 'running' && (
        <Typography.Paragraph>{t('trace.export.running')}</Typography.Paragraph>
      )}
      {job?.status === 'done' && (
        <Space direction="vertical">
          <Typography.Paragraph type="success">
            {t('trace.export.done')} — {job.messageCount} {t('trace.column.messageCount')}
          </Typography.Paragraph>
          {job.downloadUrl && (
            <Button
              type="primary"
              icon={<DownloadOutlined />}
              onClick={() => window.open(job.downloadUrl, '_blank')}
            >
              {t('trace.action.exportXml')}
            </Button>
          )}
        </Space>
      )}
      {job?.status === 'failed' && (
        <Typography.Paragraph type="danger">
          {t('trace.export.failed')}: {job.errorMessage}
        </Typography.Paragraph>
      )}
    </Modal>
  );
}

// MessageDetail 报文详情；外置报文 lazy fetch
interface MessageDetailProps {
  msg: TraceMessage;
  t: ReturnType<typeof useT>;
}

function MessageDetail({ msg, t }: MessageDetailProps) {
  // external 报文（inline 空 + 有 object_key）→ React Query 自动 lazy fetch
  const isExternal = !msg.payloadInline && Boolean(msg.payloadObjectKey);
  const { data: fetched, isLoading, error } = useQuery({
    queryKey: ['trace', 'message-payload', msg.taskId, msg.id],
    queryFn: () => traceApi.getMessagePayload(msg.taskId, msg.id),
    enabled: isExternal,
  });
  const payload = msg.payloadInline || fetched || '';
  const errorMsg = error instanceof Error ? error.message : null;
  return (
    <Space direction="vertical" style={{ width: '100%' }} size="small">
      <div>
        <Tag color={DIRECTION_COLOR[msg.direction]}>
          {t(`trace.message.direction.${msg.direction}` as never)}
        </Tag>
        <Typography.Text type="secondary">
          {new Date(msg.capturedAt).toLocaleString()}
        </Typography.Text>
        {msg.rpcMethod && <Tag style={{ marginLeft: 8 }}>{msg.rpcMethod}</Tag>}
        {msg.payloadObjectKey && <Tag color="gold">external</Tag>}
      </div>
      <Typography.Paragraph>
        <pre
          style={{
            background: 'var(--color-surface-2, #f5f5f5)',
            padding: 12,
            borderRadius: 4,
            maxHeight: 560,
            overflow: 'auto',
            fontSize: 12,
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-all',
          }}
        >
          {isLoading ? t('trace.export.running') : errorMsg ? errorMsg : payload || '(empty)'}
        </pre>
      </Typography.Paragraph>
    </Space>
  );
}

interface MessageListProps {
  task: TraceTask;
  onSelect: (m: TraceMessage) => void;
  t: ReturnType<typeof useT>;
}

function MessageList({ task, onSelect, t }: MessageListProps) {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const { data, isLoading } = useTraceMessages(task.id, { page, pageSize });

  const columns: DataTableColumn<TraceMessage & Record<string, unknown>>[] = useMemo(
    () => [
      {
        key: 'capturedAt',
        title: t('trace.message.column.capturedAt'),
        dataIndex: 'capturedAt',
        width: 200,
        render: (v) => new Date(v as string).toLocaleString(),
      },
      {
        key: 'direction',
        title: t('trace.message.column.direction'),
        dataIndex: 'direction',
        width: 110,
        render: (_v, r) => (
          <Tag color={DIRECTION_COLOR[r.direction]}>
            {t(`trace.message.direction.${r.direction}` as never)}
          </Tag>
        ),
      },
      {
        key: 'rpcMethod',
        title: t('trace.message.column.rpcMethod'),
        dataIndex: 'rpcMethod',
        width: 180,
        render: (v) => (v ? <Tag>{String(v)}</Tag> : '-'),
      },
      {
        key: 'size',
        title: t('trace.message.column.size'),
        dataIndex: 'payloadSizeBytes',
        width: 90,
        render: (v) => formatBytes(Number(v ?? 0)),
      },
      {
        key: 'actions',
        title: t('table.action'),
        fixed: 'right',
        width: 100,
        render: (_v, r) => (
          <Button size="small" type="link" onClick={() => onSelect(r)}>
            {t('common.detail')}
          </Button>
        ),
      },
    ],
    [t, onSelect]
  );

  return (
    <DataTable
      tableId="trace-messages"
      rowKey="id"
      loading={isLoading}
      columns={columns}
      dataSource={(data?.items ?? []) as (TraceMessage & Record<string, unknown>)[]}
      total={data?.total ?? 0}
      currentPage={page}
      pageSize={pageSize}
      onPageChange={(p, s) => {
        setPage(p);
        setPageSize(s);
      }}
      defaultDensity="compact"
    />
  );
}
