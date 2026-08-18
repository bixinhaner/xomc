import { useState } from 'react';
import { Alert, App, Button, Select, Space, Table, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { ReloadOutlined } from '@ant-design/icons';
import { useAccessActions, useRetryAccessAction } from '@core/hooks/api/useDeviceAccess';
import type { AccessAction, ActionStatus } from '@core/services/api/deviceAccessApi';
import { formatSystemTime } from '@core/utils/systemTime';
import { ActionStatusTag, formatProductName, GuardedButton, QueryError } from './shared';

interface Props { operatorCode: string; t: (key: string) => string; allowed: boolean }

export default function AccessActionsPanel({ operatorCode, t, allowed }: Props) {
  const { message } = App.useApp();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [status, setStatus] = useState<ActionStatus>();
  const query = useAccessActions({ operatorCode, page, pageSize, status });
  const retry = useRetryAccessAction();
  const doRetry = async (row: AccessAction) => {
    try { await retry.mutateAsync({ operatorCode, actionId: row.id }); void message.success(t('deviceAccess.retryQueued')); }
    catch (error) { void message.error(error instanceof Error ? error.message : t('common.operationFailed')); }
  };

  const columns: ColumnsType<AccessAction> = [
    { title: t('deviceAccess.serialNumber'), dataIndex: 'serial_number', width: 220 },
    { title: t('deviceAccess.productName'), dataIndex: 'product_name', width: 180, render: (v: string) => formatProductName(v, t) },
    { title: t('deviceAccess.actionType'), dataIndex: 'action_type', width: 130, render: (v: string) => <Tag color={v === 'rf_off' ? 'red' : 'green'}>{t(`deviceAccess.actionType.${v}`)}</Tag> },
    { title: t('deviceAccess.direction'), dataIndex: 'direction', width: 120, render: (v: string) => t(`deviceAccess.direction.${v}`) },
    { title: t('deviceAccess.actionStatus'), dataIndex: 'status', width: 150, render: (v: ActionStatus) => <ActionStatusTag value={v} t={t} /> },
    { title: t('deviceAccess.attempts'), dataIndex: 'attempts', width: 90 },
    { title: t('deviceAccess.errorMessage'), dataIndex: 'error_message', ellipsis: true },
    { title: t('common.createdAt'), dataIndex: 'created_at', width: 190, render: (v: string) => formatSystemTime(v) },
    { title: t('common.action'), fixed: 'right', width: 240, render: (_, row) => <Space>
      {row.status === 'failed' && <GuardedButton size="small" allowed={allowed} deniedText={t('common.noPermission')} loading={retry.isPending} onClick={() => void doRetry(row)}>{t('common.retry')}</GuardedButton>}
      {row.status !== 'failed' && '—'}
    </Space> },
  ];

  return <>
    <Alert type="info" showIcon title={t('deviceAccess.actionSafetyTitle')} description={t('deviceAccess.actionSafetyDescription')} style={{ marginBottom: 16 }} />
    <Space wrap style={{ marginBottom: 16 }}><Select allowClear value={status} onChange={(v) => { setStatus(v); setPage(1); }} placeholder={t('deviceAccess.allActionStatuses')} style={{ width: 190 }} options={(['pending_dispatch', 'dispatching', 'succeeded', 'failed', 'cancelled'] as ActionStatus[]).map((v) => ({ value: v, label: t(`deviceAccess.actionStatus.${v}`) }))} /><Button icon={<ReloadOutlined />} onClick={() => void query.refetch()}>{t('common.refresh')}</Button></Space>
    <QueryError error={query.error} t={t} />
    <Table rowKey="id" columns={columns} dataSource={query.data?.items ?? []} loading={query.isLoading} scroll={{ x: 1300 }} pagination={{ current: page, pageSize, total: query.data?.total ?? 0, showSizeChanger: true, onChange: (p, size) => { setPage(p); setPageSize(size); } }} />
  </>;
}
