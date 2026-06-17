import type { CSSProperties } from 'react';
import { useMemo, useState } from 'react';
import { Button, Card, Descriptions, Form, Input, InputNumber, Modal, Radio, Segmented, Space, Tag, Typography, message } from 'antd';
import { EyeOutlined, RadarChartOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useT } from '@/hooks/useT';
import {
  useDiagnosticPing,
  useDiagnosticThroughput,
  useDiagnosticTraceroute,
  useOpsDiagnostics,
} from '@core/hooks/api/useOpsExt';
import type { DiagnosticStatus, OpsDiagnostic } from '@core/services/api/opsExtApi';
import { formatSystemTime } from '@core/utils/systemTime';

type OpsDiagnosticRow = OpsDiagnostic & Record<string, unknown>;
type DiagKind = 'ping' | 'traceroute' | 'throughput';

const STATUS_COLOR: Record<DiagnosticStatus, string> = {
  pending: 'default',
  running: 'processing',
  complete: 'success',
  failed: 'error',
  timeout: 'warning',
};

function safeJson(v: unknown): string {
  if (v == null) return '—';
  if (typeof v === 'string') return v;
  try {
    return JSON.stringify(v, null, 2);
  } catch {
    return String(v);
  }
}

export default function NetworkDiagnosis() {
  const t = useT();
  const [filters, setFilters] = useState<{ deviceSn?: string; diagType?: string; status?: DiagnosticStatus }>({});
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [detail, setDetail] = useState<OpsDiagnostic | null>(null);
  const [launchOpen, setLaunchOpen] = useState(false);

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(filters.deviceSn ? { deviceSn: filters.deviceSn } : {}),
      ...(filters.diagType ? { diagType: filters.diagType } : {}),
      ...(filters.status ? { status: filters.status } : {}),
    }),
    [page, pageSize, filters],
  );

  const { data, isLoading, refetch } = useOpsDiagnostics(params);
  const rows = (data?.items ?? []) as OpsDiagnosticRow[];
  const total = data?.total ?? 0;

  const statusLabel = useMemo(
    (): Record<DiagnosticStatus, string> => ({
      pending: t('ops.diag.stPending'),
      running: t('ops.diag.stRunning'),
      complete: t('ops.diag.stComplete'),
      failed: t('ops.diag.stFailed'),
      timeout: t('ops.diag.stTimeout'),
    }),
    [t],
  );

  const filterFields: FilterField[] = useMemo(
    () => [
      { name: 'deviceSn', label: t('trace.column.deviceSn'), type: 'input', placeholder: t('ops.deviceSnPlaceholder') },
      {
        name: 'diagType',
        label: t('ops.diag.type'),
        type: 'select',
        placeholder: t('ops.diag.allTypes'),
        options: [
          { label: t('ops.diag.typePing'), value: 'ping' },
          { label: t('ops.diag.typeTraceroute'), value: 'traceroute' },
          { label: t('ops.diag.typeThroughput'), value: 'throughput' },
        ],
      },
      {
        name: 'status',
        label: t('table.status'),
        type: 'select',
        placeholder: t('common.all'),
        options: [
          { label: t('ops.diag.stComplete'), value: 'complete' },
          { label: t('ops.diag.stRunning'), value: 'running' },
          { label: t('ops.diag.stPending'), value: 'pending' },
          { label: t('ops.diag.stFailed'), value: 'failed' },
          { label: t('ops.diag.stTimeout'), value: 'timeout' },
        ],
      },
    ],
    [t],
  );

  const columns: DataTableColumn<OpsDiagnosticRow>[] = useMemo(
    () => [
      { key: 'diag_type', title: t('ops.diag.type'), dataIndex: 'diag_type', width: 130, render: (val) => <Tag>{String(val)}</Tag> },
      { key: 'device_sn', title: t('ops.diag.device'), dataIndex: 'device_sn', width: 160, mono: true, copyable: true, render: (val) => (val ? String(val) : '—') },
      { key: 'initiator', title: t('ops.diag.initiator'), dataIndex: 'initiator', width: 110, render: (val) => (val ? String(val) : '—') },
      {
        key: 'started_at',
        title: t('ops.diag.started'),
        dataIndex: 'started_at',
        width: 170,
        render: (val) => (val ? formatSystemTime(String(val)) : '—'),
      },
      { key: 'duration_ms', title: t('ops.diag.duration'), dataIndex: 'duration_ms', width: 100, render: (val) => (val ? `${Number(val)} ms` : '—') },
      {
        key: 'status',
        title: t('table.status'),
        dataIndex: 'status',
        width: 100,
        render: (val) => {
          const s = val as DiagnosticStatus;
          return <Tag color={STATUS_COLOR[s]}>{statusLabel[s] ?? s}</Tag>;
        },
      },
      {
        key: 'actions',
        title: t('table.operation'),
        dataIndex: 'id',
        width: 90,
        fixed: 'right',
        render: (_, record) => (
          <Button type="link" size="small" icon={<EyeOutlined />} onClick={() => setDetail(record)}>
            {t('common.detail')}
          </Button>
        ),
      },
    ],
    [t, statusLabel],
  );

  return (
    <ListPageLayout
      title={t('nav.ops.networkDiagnosis')}
      subtitle={t('ops.networkDiagnosisSubtitle')}
      extra={
        <Button type="primary" icon={<RadarChartOutlined />} onClick={() => setLaunchOpen(true)}>
          {t('ops.diag.new')}
        </Button>
      }
    >
      <FilterBar
        filterId="ops-diagnostics-filter"
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
          tableId="ops-diagnostics-list"
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
          scroll={{ x: 1020 }}
        />
      </Card>

      <DiagnosticDetailModal diag={detail} statusLabel={statusLabel} onClose={() => setDetail(null)} />

      <LaunchDiagnosticModal open={launchOpen} onClose={() => setLaunchOpen(false)} onDone={() => void refetch()} />
    </ListPageLayout>
  );
}

function DiagnosticDetailModal({
  diag,
  statusLabel,
  onClose,
}: {
  diag: OpsDiagnostic | null;
  statusLabel: Record<DiagnosticStatus, string>;
  onClose: () => void;
}) {
  const t = useT();
  return (
    <Modal
      open={diag !== null}
      onCancel={onClose}
      title={diag ? `${diag.diag_type} · ${diag.device_sn || '—'}` : ''}
      footer={<Button onClick={onClose}>{t('common.cancel')}</Button>}
      width={680}
    >
      {diag && (
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <Descriptions column={2} size="small" bordered>
            <Descriptions.Item label={t('ops.diag.type')}>{diag.diag_type}</Descriptions.Item>
            <Descriptions.Item label={t('table.status')}>
              <Tag color={STATUS_COLOR[diag.status]}>{statusLabel[diag.status] ?? diag.status}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label={t('ops.diag.device')}>{diag.device_sn || '—'}</Descriptions.Item>
            <Descriptions.Item label={t('ops.diag.initiator')}>{diag.initiator || diag.operator || '—'}</Descriptions.Item>
            <Descriptions.Item label={t('ops.diag.started')}>
              {formatSystemTime(diag.started_at)}
            </Descriptions.Item>
            <Descriptions.Item label={t('ops.diag.duration')}>
              {diag.duration_ms ? `${diag.duration_ms} ms` : '—'}
            </Descriptions.Item>
            {diag.completed_at && (
              <Descriptions.Item label={t('ops.executeTime')}>
                {formatSystemTime(diag.completed_at)}
              </Descriptions.Item>
            )}
            {diag.error_message && (
              <Descriptions.Item label={t('ops.diag.errorMessage')} span={2}>
                <span style={{ color: 'var(--color-error-500, #ff4d4f)' }}>{diag.error_message}</span>
              </Descriptions.Item>
            )}
          </Descriptions>

          <div>
            <Typography.Text strong>{t('ops.diag.request')}</Typography.Text>
            <pre style={preStyle}>{safeJson(diag.request)}</pre>
          </div>
          <div>
            <Typography.Text strong>{t('ops.diag.result')}</Typography.Text>
            <pre style={preStyle}>{safeJson(diag.result)}</pre>
          </div>
        </Space>
      )}
    </Modal>
  );
}

const preStyle: CSSProperties = {
  margin: '8px 0 0',
  padding: 12,
  maxHeight: 240,
  overflow: 'auto',
  fontSize: 12,
  background: 'var(--color-fill-quaternary, #f5f5f5)',
  borderRadius: 4,
  whiteSpace: 'pre-wrap',
  wordBreak: 'break-all',
};

function LaunchDiagnosticModal({ open, onClose, onDone }: { open: boolean; onClose: () => void; onDone: () => void }) {
  const t = useT();
  const [kind, setKind] = useState<DiagKind>('ping');
  const [deviceSn, setDeviceSn] = useState('');
  const [host, setHost] = useState('');
  const [count, setCount] = useState(4);
  const [url, setUrl] = useState('');
  const [direction, setDirection] = useState<'download' | 'upload'>('download');

  const ping = useDiagnosticPing();
  const traceroute = useDiagnosticTraceroute();
  const throughput = useDiagnosticThroughput();
  const pending = ping.isPending || traceroute.isPending || throughput.isPending;

  const close = () => {
    setKind('ping');
    setDeviceSn('');
    setHost('');
    setCount(4);
    setUrl('');
    setDirection('download');
    onClose();
  };

  const done = {
    onSuccess: () => {
      void message.success(t('ops.diag.launchSuccess'));
      close();
      onDone();
    },
    onError: () => void message.error(t('ops.diag.launchFailed')),
  };

  const submit = () => {
    const sn = deviceSn.trim();
    if (!sn) return;
    if (kind === 'ping') {
      if (!host.trim()) return;
      ping.mutate({ device_sn: sn, host: host.trim(), count }, done);
    } else if (kind === 'traceroute') {
      if (!host.trim()) return;
      traceroute.mutate({ device_sn: sn, host: host.trim() }, done);
    } else {
      if (!url.trim()) return;
      throughput.mutate({ device_sn: sn, url: url.trim(), direction }, done);
    }
  };

  const submitDisabled =
    !deviceSn.trim() || (kind === 'throughput' ? !url.trim() : !host.trim());

  return (
    <Modal
      open={open}
      title={t('ops.diag.newTitle')}
      onCancel={close}
      onOk={submit}
      okText={t('ops.diag.new')}
      cancelText={t('common.cancel')}
      confirmLoading={pending}
      okButtonProps={{ disabled: submitDisabled }}
      width={520}
    >
      <Form layout="vertical">
        <Form.Item label={t('ops.diag.type')}>
          <Segmented
            value={kind}
            onChange={(v) => setKind(v as DiagKind)}
            options={[
              { label: t('ops.diag.typePing'), value: 'ping' },
              { label: t('ops.diag.typeTraceroute'), value: 'traceroute' },
              { label: t('ops.diag.typeThroughput'), value: 'throughput' },
            ]}
          />
        </Form.Item>
        <Form.Item label={t('trace.column.deviceSn')} required>
          <Input
            placeholder={t('ops.deviceSnPlaceholder')}
            value={deviceSn}
            onChange={(e) => setDeviceSn(e.target.value)}
          />
        </Form.Item>
        {kind !== 'throughput' && (
          <Form.Item label={t('ops.diag.targetHost')} required>
            <Input
              placeholder={t('ops.diag.targetHostPlaceholder')}
              value={host}
              onChange={(e) => setHost(e.target.value)}
            />
          </Form.Item>
        )}
        {kind === 'ping' && (
          <Form.Item label={t('ops.diag.probeCount')}>
            <InputNumber min={1} max={20} value={count} onChange={(v) => setCount(Number(v) || 4)} />
          </Form.Item>
        )}
        {kind === 'throughput' && (
          <>
            <Form.Item label={t('ops.diag.speedUrl')} required>
              <Input
                placeholder="http://speedtest.example/test.bin"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
              />
            </Form.Item>
            <Form.Item label={t('ops.diag.direction')}>
              <Radio.Group value={direction} onChange={(e) => setDirection(e.target.value)}>
                <Radio.Button value="download">{t('ops.diag.directionDownload')}</Radio.Button>
                <Radio.Button value="upload">{t('ops.diag.directionUpload')}</Radio.Button>
              </Radio.Group>
            </Form.Item>
          </>
        )}
      </Form>
    </Modal>
  );
}
