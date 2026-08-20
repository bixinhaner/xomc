import { useState } from 'react';
import { Alert, App, Button, Input, Modal, Select, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { RedoOutlined, ReloadOutlined } from '@ant-design/icons';
import { useAccessActionAttempts, useAccessActions, useRetryAccessAction } from '@core/hooks/api/useDeviceAccess';
import type { AccessAction, AccessActionAttempt, ActionStatus } from '@core/services/api/deviceAccessApi';
import { formatSystemTime } from '@core/utils/systemTime';
import { ActionStatusTag, formatProductName, IconActionButton, QueryError } from './shared';
import styles from './AccessControl.module.css';

interface Props { operatorCode: string; t: (key: string) => string; allowed: boolean }

function ActionAttempts({ operatorCode, action, t }: { operatorCode: string; action: AccessAction; t: Props['t'] }) {
  const query = useAccessActionAttempts(operatorCode, action.id);
  const columns: ColumnsType<AccessActionAttempt> = [
    { title: t('deviceAccess.attempts'), dataIndex: 'attempt_no', width: 90 },
    { title: t('deviceAccess.attemptPhase'), dataIndex: 'phase', width: 150, render: (value: string) => t(`deviceAccess.attemptPhase.${value}`) },
    { title: t('deviceAccess.actionStatus'), dataIndex: 'status', width: 130, render: (value: string) => t(`deviceAccess.attemptStatus.${value}`) },
    { title: t('deviceAccess.lastFailureCode'), width: 180, render: (_, row) => row.failure_code || row.fault_code || '—' },
    { title: t('deviceAccess.errorMessage'), dataIndex: 'error_message', ellipsis: true },
    { title: t('common.createdAt'), dataIndex: 'started_at', width: 190, render: (value: string) => formatSystemTime(value) },
  ];
  const evidenceMissing = !query.isLoading && !query.error
    && (action.status === 'succeeded' || action.last_failure_code === 'evidence_missing')
    && (query.data?.length ?? 0) === 0;
  return <>
    <QueryError error={query.error} t={t} />
    {evidenceMissing && <Alert type="error" showIcon title={t('deviceAccess.actionEvidenceMissing')} style={{ marginBottom: 12 }} />}
    <Table
      rowKey="id"
      size="small"
      pagination={false}
      loading={query.isLoading}
      columns={columns}
      dataSource={query.data ?? []}
      locale={{ emptyText: evidenceMissing ? t('deviceAccess.actionEvidenceMissing') : t('common.noData') }}
    />
  </>;
}

export default function AccessActionsPanel({ operatorCode, t, allowed }: Props) {
  const { message } = App.useApp();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [status, setStatus] = useState<ActionStatus>();
  const [retryingAction, setRetryingAction] = useState<AccessAction>();
  const [retryReason, setRetryReason] = useState('');
  const query = useAccessActions({ operatorCode, page, pageSize, status });
  const retry = useRetryAccessAction();
  const doRetry = async () => {
    if (!retryingAction) return;
    const reason = retryReason.trim();
    if (!reason) { void message.error(t('deviceAccess.retryReasonRequired')); return; }
    try {
      await retry.mutateAsync({ operatorCode, actionId: retryingAction.id, reason });
      setRetryingAction(undefined); setRetryReason('');
      void message.success(t('deviceAccess.retryQueued'));
    }
    catch (error) { void message.error(error instanceof Error ? error.message : t('common.operationFailed')); }
  };

  const columns: ColumnsType<AccessAction> = [
    { title: t('deviceAccess.serialNumber'), dataIndex: 'serial_number', width: 220 },
    { title: t('deviceAccess.productName'), dataIndex: 'product_name', width: 180, render: (v: string) => formatProductName(v, t) },
    { title: t('deviceAccess.actionType'), dataIndex: 'action_type', width: 130, render: (v: string) => <Tag color={v === 'rf_off' ? 'red' : 'green'}>{t(`deviceAccess.actionType.${v}`)}</Tag> },
    { title: t('deviceAccess.direction'), dataIndex: 'direction', width: 120, render: (v: string) => t(`deviceAccess.direction.${v}`) },
    { title: t('deviceAccess.actionStatus'), dataIndex: 'status', width: 150, render: (v: ActionStatus) => <ActionStatusTag value={v} t={t} /> },
    { title: t('deviceAccess.attempts'), width: 90, render: (_, row) => `${row.attempts}/${row.max_attempts}` },
    { title: t('deviceAccess.lastFailureCode'), dataIndex: 'last_failure_code', width: 150, render: (v?: string) => v || '—' },
    { title: t('deviceAccess.nextAttemptAt'), dataIndex: 'next_attempt_at', width: 190, render: (v?: string) => v ? formatSystemTime(v) : '—' },
    { title: t('deviceAccess.errorMessage'), dataIndex: 'error_message', ellipsis: true },
    { title: t('common.createdAt'), dataIndex: 'created_at', width: 190, render: (v: string) => formatSystemTime(v) },
    { title: t('common.action'), fixed: 'right', width: 76, align: 'center', render: (_, row) => <div className={styles.iconActions}>
      {(row.status === 'failed' || row.status === 'dead') && <IconActionButton label={t('common.retry')} icon={<RedoOutlined />} allowed={allowed} deniedText={t('common.noPermission')} loading={retry.isPending} onClick={() => setRetryingAction(row)} />}
      {row.status !== 'failed' && row.status !== 'dead' && '—'}
    </div> },
  ];

  return <>
    <Alert type="info" showIcon title={t('deviceAccess.actionSafetyTitle')} description={t('deviceAccess.actionSafetyDescription')} style={{ marginBottom: 16 }} />
    <Space wrap className={styles.filterBar}><Select allowClear value={status} onChange={(v) => { setStatus(v); setPage(1); }} placeholder={t('deviceAccess.allActionStatuses')} style={{ width: 190 }} options={(['pending_dispatch', 'dispatching', 'verifying', 'retry_wait', 'succeeded', 'failed', 'dead', 'cancelled'] as ActionStatus[]).map((v) => ({ value: v, label: t(`deviceAccess.actionStatus.${v}`) }))} /><Button icon={<ReloadOutlined />} onClick={() => void query.refetch()}>{t('common.refresh')}</Button></Space>
    <QueryError error={query.error} t={t} />
    <Table className={styles.dataTable} rowKey="id" columns={columns} dataSource={query.data?.items ?? []} loading={query.isLoading} scroll={{ x: 1490 }} expandable={{ expandedRowRender: (row) => <ActionAttempts operatorCode={operatorCode} action={row} t={t} /> }} pagination={{ current: page, pageSize, total: query.data?.total ?? 0, showSizeChanger: true, onChange: (p, size) => { setPage(p); setPageSize(size); } }} />
    <Modal title={t('deviceAccess.retryConfirmTitle')} open={Boolean(retryingAction)} confirmLoading={retry.isPending} onOk={() => void doRetry()} onCancel={() => { setRetryingAction(undefined); setRetryReason(''); }}>
      <Typography.Paragraph>{t('deviceAccess.retryConfirmDescription')}</Typography.Paragraph>
      <Input.TextArea value={retryReason} onChange={(event) => setRetryReason(event.target.value)} rows={4} maxLength={500} showCount placeholder={t('deviceAccess.retryReasonRequired')} aria-label={t('deviceAccess.retryReason')} />
    </Modal>
  </>;
}
