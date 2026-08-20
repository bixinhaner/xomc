import { useState } from 'react';
import { Alert, App, Button, Collapse, Descriptions, Drawer, Input, Modal, Pagination, Select, Space, Table, Tag, Typography } from 'antd';
import {
  useAccessActionAttempts,
  useAccessActions,
  useAccessDecisions,
  useAccessDetail,
  useAccessEvidence,
  useAccessIdentitySnapshots,
  useAccessManualOperations,
  useAccessNotifications,
  useArchiveAccessDecision,
  useRestoreAccessDecision,
} from '@core/hooks/api/useDeviceAccess';
import type { AccessAction, AccessState, DecisionItem, EvidenceItem } from '@core/services/api/deviceAccessApi';
import { formatSystemTime } from '@core/utils/systemTime';
import { ActionStatusTag, formatProductName, QueryError, StateTag } from './shared';

interface Props {
  operatorCode: string;
  serialNumber?: string;
  open: boolean;
  canArchive: boolean;
  t: (key: string) => string;
  onClose: () => void;
}

const sectionPageSize = 10;

function formatValue(value: unknown): string {
  if (value === null || value === undefined || value === '') return '—';
  if (typeof value === 'string') return value;
  return JSON.stringify(value);
}

function SectionPager({ page, total, onChange }: { page: number; total: number; onChange: (page: number) => void }) {
  if (total <= sectionPageSize) return null;
  return <Pagination size="small" current={page} pageSize={sectionPageSize} total={total} showSizeChanger={false} onChange={onChange} style={{ marginTop: 12, textAlign: 'right' }} />;
}

function DetailActionAttempts({ operatorCode, action, t }: { operatorCode: string; action: AccessAction; t: (key: string) => string }) {
  const attempts = useAccessActionAttempts(operatorCode, action.id);
  const evidenceMissing = !attempts.isLoading && !attempts.error
    && (action.status === 'succeeded' || action.last_failure_code === 'evidence_missing')
    && (attempts.data?.length ?? 0) === 0;
  return <>
    <QueryError error={attempts.error} t={t} />
    {evidenceMissing && <Alert type="error" showIcon title={t('deviceAccess.actionEvidenceMissing')} style={{ marginBottom: 12 }} />}
    <Table size="small" rowKey="id" loading={attempts.isLoading} pagination={false} dataSource={attempts.data ?? []} locale={{ emptyText: evidenceMissing ? t('deviceAccess.actionEvidenceMissing') : t('common.noData') }} columns={[
      { title: t('deviceAccess.attempts'), dataIndex: 'attempt_no', width: 90 },
      { title: t('deviceAccess.attemptPhase'), dataIndex: 'phase', width: 150 },
      { title: t('deviceAccess.result'), dataIndex: 'status', width: 120 },
      { title: t('deviceAccess.lastFailureCode'), dataIndex: 'failure_code', render: formatValue },
      { title: t('deviceAccess.errorMessage'), dataIndex: 'error_message', render: formatValue },
      { title: t('deviceAccess.observedAt'), dataIndex: 'started_at', render: (value: string) => formatSystemTime(value) },
    ]} />
  </>;
}

export default function AccessDetailDrawer({ operatorCode, serialNumber, open, canArchive, t, onClose }: Props) {
  const { message } = App.useApp();
  const [decisionPage, setDecisionPage] = useState(1);
  const [identityPage, setIdentityPage] = useState(1);
  const [evidencePage, setEvidencePage] = useState(1);
  const [actionPage, setActionPage] = useState(1);
  const [notificationPage, setNotificationPage] = useState(1);
  const [manualPage, setManualPage] = useState(1);
  const [archiveStatus, setArchiveStatus] = useState<'visible' | 'archived' | 'all'>('visible');
  const [archiveTarget, setArchiveTarget] = useState<DecisionItem>();
  const [archiveReason, setArchiveReason] = useState('');

  const detail = useAccessDetail(operatorCode, serialNumber);
  const decisions = useAccessDecisions({ operatorCode, serialNumber, page: decisionPage, pageSize: sectionPageSize, archiveStatus });
  const identities = useAccessIdentitySnapshots({ operatorCode, serialNumber, page: identityPage, pageSize: sectionPageSize });
  const evidence = useAccessEvidence({ operatorCode, serialNumber, page: evidencePage, pageSize: sectionPageSize });
  const actions = useAccessActions({ operatorCode, serialNumber, page: actionPage, pageSize: sectionPageSize });
  const notifications = useAccessNotifications({ operatorCode, serialNumber, page: notificationPage, pageSize: sectionPageSize });
  const manualOperations = useAccessManualOperations({ operatorCode, serialNumber, page: manualPage, pageSize: sectionPageSize });
  const archiveDecision = useArchiveAccessDecision();
  const restoreDecision = useRestoreAccessDecision();

  const resetAndClose = () => {
    setDecisionPage(1);
    setIdentityPage(1);
    setEvidencePage(1);
    setActionPage(1);
    setNotificationPage(1);
    setManualPage(1);
    setArchiveStatus('visible');
    onClose();
  };

  const currentDecisionVersion = detail.data?.state.decision_version;
  const decisionItems = decisions.data?.items ?? [];
  return <>
    <Drawer size="large" open={open} title={`${t('deviceAccess.decisionDetail')} — ${serialNumber ?? ''}`} onClose={resetAndClose} loading={detail.isLoading}>
      <QueryError error={detail.error} t={t} />
      {detail.data && <Descriptions bordered size="small" column={2} style={{ marginBottom: 20 }}>
        <Descriptions.Item label={t('deviceAccess.serialNumber')}>{detail.data.state.serial_number}</Descriptions.Item>
        <Descriptions.Item label={t('deviceAccess.productName')}>{formatProductName(detail.data.state.product_name, t)}</Descriptions.Item>
        <Descriptions.Item label={t('deviceAccess.accessState')}><StateTag value={detail.data.state.state} t={t} /></Descriptions.Item>
        <Descriptions.Item label={t('deviceAccess.reasonCode')}>{detail.data.state.reason_code}</Descriptions.Item>
        <Descriptions.Item label={t('deviceAccess.evidenceVersion')}>{detail.data.state.evidence_version}</Descriptions.Item>
        <Descriptions.Item label={t('deviceAccess.decisionVersion')}>{detail.data.state.decision_version}</Descriptions.Item>
        <Descriptions.Item label={t('deviceAccess.policyVersionId')}>{formatValue(detail.data.state.policy_version_id)}</Descriptions.Item>
        {detail.data.state.decision_expires_at && <Descriptions.Item label={t('deviceAccess.collectionDeadlineAt')}>{formatSystemTime(detail.data.state.decision_expires_at)}</Descriptions.Item>}
      </Descriptions>}

      <Typography.Title level={5}>{t('deviceAccess.auditTimeline')}</Typography.Title>
      <Collapse defaultActiveKey={['decisions']} items={[
        {
          key: 'identity',
          label: `${t('deviceAccess.identitySnapshots')} (${identities.data?.total ?? 0})`,
          children: <>
            <QueryError error={identities.error} t={t} />
            <Table size="small" rowKey="id" loading={identities.isLoading} pagination={false} dataSource={identities.data?.items ?? []} columns={[
              { title: t('deviceAccess.observedAt'), dataIndex: 'inform_time', render: (value: string) => formatSystemTime(value) },
              { title: t('deviceAccess.informEvent'), dataIndex: 'inform_event', render: formatValue },
              { title: t('deviceAccess.identityStatus'), dataIndex: 'identity_status', render: (value: string) => <Tag>{value}</Tag> },
              { title: 'OUI', dataIndex: 'oui', render: formatValue },
              { title: 'ProductClass', dataIndex: 'product_class', render: formatValue },
              { title: 'CloudKey', dataIndex: 'cloud_key', render: formatValue },
              { title: t('deviceAccess.sourceIP'), dataIndex: 'observed_remote_ip', render: formatValue },
            ]} />
            <SectionPager page={identityPage} total={identities.data?.total ?? 0} onChange={setIdentityPage} />
          </>,
        },
        {
          key: 'decisions',
          label: `${t('deviceAccess.decisions')} (${decisions.data?.total ?? 0})`,
          children: <>
            <Space style={{ marginBottom: 12 }}>
              <Select value={archiveStatus} onChange={(value) => { setArchiveStatus(value); setDecisionPage(1); }} style={{ width: 180 }} options={[
                { value: 'visible', label: t('deviceAccess.archive.visible') },
                { value: 'archived', label: t('deviceAccess.archive.archived') },
                { value: 'all', label: t('deviceAccess.archive.all') },
              ]} />
            </Space>
            <QueryError error={decisions.error} t={t} />
            <Collapse items={decisionItems.map((decision) => ({
              key: decision.id,
              label: <Space wrap>
                <StateTag value={decision.new_state as AccessState} t={t} />
                <span>v{decision.decision_version}</span>
                <span>{decision.reason_code}</span>
                <span>{formatSystemTime(decision.occurred_at)}</span>
                {decision.archive && <Tag>{t('deviceAccess.archive.archived')}</Tag>}
              </Space>,
              extra: decision.archive
                ? <Button type="link" disabled={!canArchive || restoreDecision.isPending} onClick={(event) => { event.stopPropagation(); void restoreDecision.mutateAsync({ operatorCode, decisionId: decision.id }).then(() => message.success(t('deviceAccess.archive.restored'))).catch((error) => message.error(error instanceof Error ? error.message : t('common.operationFailed'))); }}>{t('deviceAccess.archive.restore')}</Button>
                : <Button type="link" danger disabled={!canArchive || decision.decision_version === currentDecisionVersion} onClick={(event) => { event.stopPropagation(); setArchiveReason(''); setArchiveTarget(decision); }}>{t('deviceAccess.archive.action')}</Button>,
              children: <>
                <Descriptions bordered size="small" column={1} style={{ marginBottom: 12 }}>
                  <Descriptions.Item label={t('deviceAccess.decisionId')}>{decision.id}</Descriptions.Item>
                  <Descriptions.Item label={t('deviceAccess.policyVersionId')}>{formatValue(decision.policy_version_id)}</Descriptions.Item>
                  <Descriptions.Item label={t('deviceAccess.matchedRuleId')}>{formatValue(decision.matched_rule_id)}</Descriptions.Item>
                </Descriptions>
                <Table size="small" rowKey="id" pagination={false} dataSource={decision.checks} columns={[
                  { title: t('deviceAccess.checkType'), dataIndex: 'check_type' },
                  { title: t('deviceAccess.result'), dataIndex: 'result', width: 110 },
                  { title: t('deviceAccess.expected'), dataIndex: 'expected_summary' },
                  { title: t('deviceAccess.observed'), dataIndex: 'observed_summary' },
                  { title: t('deviceAccess.source'), dataIndex: 'evidence_source' },
                ]} />
              </>,
            }))} />
            <SectionPager page={decisionPage} total={decisions.data?.total ?? 0} onChange={setDecisionPage} />
          </>,
        },
        {
          key: 'evidence',
          label: `${t('deviceAccess.evidence')} (${evidence.data?.total ?? 0})`,
          children: <>
            <QueryError error={evidence.error} t={t} />
            <Table<EvidenceItem> size="small" rowKey="id" loading={evidence.isLoading} pagination={false} dataSource={evidence.data?.items ?? []} columns={[
              { title: t('deviceAccess.evidenceVersion'), dataIndex: 'evidence_version' },
              { title: t('deviceAccess.evidenceType'), dataIndex: 'evidence_type' },
              { title: t('deviceAccess.result'), dataIndex: 'evidence_status' },
              { title: t('deviceAccess.observed'), dataIndex: 'normalized_value', render: formatValue },
              { title: t('deviceAccess.source'), dataIndex: 'source' },
              { title: t('deviceAccess.observedAt'), dataIndex: 'observed_at', render: (value: string) => formatSystemTime(value) },
            ]} />
            <SectionPager page={evidencePage} total={evidence.data?.total ?? 0} onChange={setEvidencePage} />
          </>,
        },
        {
          key: 'actions',
          label: `${t('deviceAccess.actions')} (${actions.data?.total ?? 0})`,
          children: <>
            <QueryError error={actions.error} t={t} />
            <Table size="small" rowKey="id" loading={actions.isLoading} pagination={false} dataSource={actions.data?.items ?? []} expandable={{ expandedRowRender: (action) => <DetailActionAttempts operatorCode={operatorCode} action={action} t={t} /> }} columns={[
              { title: t('deviceAccess.actionType'), dataIndex: 'action_type', render: (value: string) => t(`deviceAccess.actionType.${value}`) },
              { title: t('deviceAccess.actionStatus'), dataIndex: 'status', render: (value) => <ActionStatusTag value={value} t={t} /> },
              { title: t('deviceAccess.attempts'), render: (_, action: AccessAction) => `${action.attempts}/${action.max_attempts}` },
              { title: t('deviceAccess.lastFailureCode'), dataIndex: 'last_failure_code', render: formatValue },
              { title: t('common.updatedAt'), dataIndex: 'updated_at', render: (value: string) => formatSystemTime(value) },
            ]} />
            <SectionPager page={actionPage} total={actions.data?.total ?? 0} onChange={setActionPage} />
          </>,
        },
        {
          key: 'notifications',
          label: `${t('deviceAccess.notifications')} (${notifications.data?.total ?? 0})`,
          children: <>
            <QueryError error={notifications.error} t={t} />
            <Table size="small" rowKey="id" loading={notifications.isLoading} pagination={false} dataSource={notifications.data?.items ?? []} columns={[
              { title: t('deviceAccess.notificationChannel'), dataIndex: 'channel', width: 110 },
              { title: t('deviceAccess.notificationSubject'), dataIndex: 'subject', render: formatValue },
              { title: t('deviceAccess.notificationStatus'), dataIndex: 'status', width: 150, render: (value: string) => <Tag color={value === 'sent' ? 'success' : value === 'failed' || value === 'dead_letter' ? 'error' : 'default'}>{t(`deviceAccess.notificationStatus.${value}`)}</Tag> },
              { title: t('deviceAccess.notificationError'), dataIndex: 'error_message', render: formatValue },
              { title: t('deviceAccess.correlationId'), dataIndex: 'correlation_id', render: formatValue },
              { title: t('common.createdAt'), dataIndex: 'created_at', render: (value: string) => formatSystemTime(value) },
            ]} />
            <SectionPager page={notificationPage} total={notifications.data?.total ?? 0} onChange={setNotificationPage} />
          </>,
        },
        {
          key: 'manual',
          label: `${t('deviceAccess.manualOperations')} (${manualOperations.data?.total ?? 0})`,
          children: <>
            <QueryError error={manualOperations.error} t={t} />
            <Table size="small" rowKey="id" loading={manualOperations.isLoading} pagination={false} dataSource={manualOperations.data?.items ?? []} columns={[
              { title: t('deviceAccess.operation'), dataIndex: 'operation' },
              { title: t('deviceAccess.operatorUser'), dataIndex: 'username', render: formatValue },
              { title: t('deviceAccess.reason'), dataIndex: ['details', 'reason'], render: formatValue },
              { title: t('common.createdAt'), dataIndex: 'created_at', render: (value: string) => formatSystemTime(value) },
            ]} />
            <SectionPager page={manualPage} total={manualOperations.data?.total ?? 0} onChange={setManualPage} />
          </>,
        },
      ]} />
    </Drawer>
    <Modal
      open={Boolean(archiveTarget)}
      title={t('deviceAccess.archive.confirmTitle')}
      okText={t('common.confirm')}
      cancelText={t('common.cancel')}
      okButtonProps={{ danger: true, disabled: !archiveReason.trim() }}
      confirmLoading={archiveDecision.isPending}
      onCancel={() => { if (!archiveDecision.isPending) setArchiveTarget(undefined); }}
      onOk={async () => {
        if (!archiveTarget || !archiveReason.trim()) return;
        try {
          await archiveDecision.mutateAsync({ operatorCode, decisionId: archiveTarget.id, reason: archiveReason.trim() });
          void message.success(t('deviceAccess.archive.completed'));
          setArchiveTarget(undefined);
        } catch (error) {
          void message.error(error instanceof Error ? error.message : t('common.operationFailed'));
        }
      }}
    >
      <Typography.Paragraph>{t('deviceAccess.archive.explanation')}</Typography.Paragraph>
      <Input.TextArea value={archiveReason} onChange={(event) => setArchiveReason(event.target.value)} maxLength={500} showCount placeholder={t('deviceAccess.archive.reasonRequired')} />
    </Modal>
  </>;
}
