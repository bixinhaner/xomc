import { useEffect, useState } from 'react';
import { App, Button, Card, Col, DatePicker, Input, Modal, Row, Select, Space, Statistic, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { AuditOutlined, CheckCircleOutlined, EyeOutlined, FilterOutlined, ReloadOutlined, SearchOutlined, StopOutlined, SyncOutlined } from '@ant-design/icons';
import { useAccessStateSummary, useAccessStates, useReevaluateAccessDevice, useUpsertAccessEntry } from '@core/hooks/api/useDeviceAccess';
import type { AccessListType, AccessState, AccessStateItem, ActionStatus, RuleDimension } from '@core/services/api/deviceAccessApi';
import { formatSystemTime } from '@core/utils/systemTime';
import { formatProductName, IconActionButton, normalTaskProtectionPresentation, QueryError, StateTag } from './shared';
import AccessDetailDrawer from './AccessDetailDrawer';
import styles from './AccessControl.module.css';

interface Props {
  operatorCode: string;
  t: (key: string) => string;
  canReevaluate: boolean;
  canManageList: boolean;
  canArchive?: boolean;
  businessEnabled?: boolean;
  ruleDrilldown?: { policyVersionId: string; matchedRuleId: string };
  onReviewCandidate?: (serialNumber: string) => void;
}

const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

export default function AccessStatesPanel({ operatorCode, t, canReevaluate, canManageList, canArchive = false, businessEnabled = true, ruleDrilldown, onReviewCandidate }: Props) {
  const { message } = App.useApp();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [serialInput, setSerialInput] = useState('');
  const [serialNumber, setSerialNumber] = useState<string>();
  const [state, setState] = useState<AccessState>();
  const [decision, setDecision] = useState<string>();
  const [reasonCode, setReasonCode] = useState<string>();
  const [reasonInput, setReasonInput] = useState('');
  const [actionStatus, setActionStatus] = useState<ActionStatus>();
  const [dimension, setDimension] = useState<RuleDimension>();
  const [policyVersionInput, setPolicyVersionInput] = useState('');
  const [policyVersionId, setPolicyVersionId] = useState<string>();
  const [matchedRuleInput, setMatchedRuleInput] = useState('');
  const [matchedRuleId, setMatchedRuleId] = useState<string>();
  const [startedAt, setStartedAt] = useState<string>();
  const [endedAt, setEndedAt] = useState<string>();
  const [selectedSN, setSelectedSN] = useState<string>();
  const [advancedFiltersVisible, setAdvancedFiltersVisible] = useState(false);
  const [datePickerKey, setDatePickerKey] = useState(0);
  const [reevaluatingSN, setReevaluatingSN] = useState<string>();
  const [quickListTarget, setQuickListTarget] = useState<{ row: AccessStateItem; type: AccessListType }>();
  const [quickListReason, setQuickListReason] = useState('');
  const filters = { operatorCode, serialNumber, state, decision, reasonCode, actionStatus, dimension, policyVersionId, matchedRuleId, startedAt, endedAt };
  const query = useAccessStates({ ...filters, page, pageSize });
  const summary = useAccessStateSummary(filters);
  const reevaluate = useReevaluateAccessDevice();
  const upsertEntry = useUpsertAccessEntry();

  useEffect(() => {
    if (!ruleDrilldown) return;
    setPolicyVersionInput(ruleDrilldown.policyVersionId);
    setMatchedRuleInput(ruleDrilldown.matchedRuleId);
    setPolicyVersionId(ruleDrilldown.policyVersionId);
    setMatchedRuleId(ruleDrilldown.matchedRuleId);
    setPage(1);
  }, [ruleDrilldown]);

  const columns: ColumnsType<AccessStateItem> = [
    { title: t('deviceAccess.serialNumber'), dataIndex: 'serial_number', width: 200, ellipsis: true },
    { title: t('deviceAccess.productName'), dataIndex: 'product_name', width: 150, ellipsis: true, render: (v: string) => formatProductName(v, t) },
    { title: t('deviceAccess.accessState'), dataIndex: 'state', width: 130, render: (v: AccessState) => <StateTag value={v} t={t} /> },
    { title: t('deviceAccess.effectiveDecision'), dataIndex: 'effective_decision', width: 120, render: (v: string) => t(`deviceAccess.decision.${v}`) },
    { title: t('deviceAccess.reasonCode'), dataIndex: 'reason_code', width: 180, ellipsis: true, render: (v: string) => t(`deviceAccess.reason.${v}`) },
    { title: t('deviceAccess.normalTaskProtection'), dataIndex: 'normal_tasks_frozen', width: 180, render: (v: boolean) => {
      const presentation = normalTaskProtectionPresentation(Boolean(businessEnabled && v));
      return <Tag color={presentation.color}>{t(presentation.messageKey)}</Tag>;
    } },
    { title: t('deviceAccess.decisionVersion'), dataIndex: 'decision_version', width: 110 },
    { title: t('common.updatedAt'), dataIndex: 'updated_at', width: 170, render: (v: string) => formatSystemTime(v) },
    { title: t('common.action'), fixed: 'right', width: 160, align: 'center', render: (_, row) => {
      const unknownCandidate = Boolean(row.candidate_id && !row.device_id);
      const requiresCandidateReview = unknownCandidate && row.state === 'review_required';
      const revoked = row.state === 'revoked' || row.reason_code === 'revoked_list_matched';
      return <div className={styles.iconActions}>
      <IconActionButton label={t('common.detail')} icon={<EyeOutlined />} onClick={() => setSelectedSN(row.serial_number)} />
      {!unknownCandidate && <IconActionButton
        label={t('deviceAccess.reevaluate')}
        icon={<SyncOutlined />}
        allowed={canReevaluate && businessEnabled}
        deniedText={!canReevaluate ? t('common.noPermission') : t('deviceAccess.switchRequired')}
        disabled={reevaluate.isPending && reevaluatingSN !== row.serial_number}
        loading={reevaluate.isPending && reevaluatingSN === row.serial_number}
        onClick={async () => {
            setReevaluatingSN(row.serial_number);
            try {
              await reevaluate.mutateAsync({ operatorCode, serialNumber: row.serial_number });
              void message.success(t('deviceAccess.reevaluateRequested'));
            } catch (error) {
              void message.error(error instanceof Error ? error.message : t('common.operationFailed'));
            } finally {
              setReevaluatingSN(undefined);
            }
          }}
      />}
      {requiresCandidateReview && onReviewCandidate && <IconActionButton label={t('deviceAccess.reviewCandidate')} icon={<AuditOutlined />} onClick={() => onReviewCandidate(row.serial_number)} />}
      {!unknownCandidate && !revoked && row.reason_code !== 'allowlist_matched' && <IconActionButton label={t('deviceAccess.addToAllowlist')} tooltip={!canManageList ? t('common.noPermission') : t('deviceAccess.quickAllowConsequence')} icon={<CheckCircleOutlined />} allowed={canManageList} deniedText={t('common.noPermission')} onClick={() => { setQuickListReason(''); setQuickListTarget({ row, type: 'allow' }); }} />}
      {!unknownCandidate && !revoked && row.reason_code !== 'denylist_matched' && <IconActionButton danger label={t('deviceAccess.addToDenylist')} tooltip={!canManageList ? t('common.noPermission') : t('deviceAccess.quickDenyConsequence')} icon={<StopOutlined />} allowed={canManageList} deniedText={t('common.noPermission')} onClick={() => { setQuickListReason(''); setQuickListTarget({ row, type: 'deny' }); }} />}
    </div>;
    } },
  ];

  const search = () => {
    const rawPolicyVersionId = policyVersionInput.trim();
    const rawMatchedRuleId = matchedRuleInput.trim();
    if ((rawPolicyVersionId && !uuidPattern.test(rawPolicyVersionId)) || (rawMatchedRuleId && !uuidPattern.test(rawMatchedRuleId))) {
      void message.error(t('deviceAccess.invalidUuidFilter'));
      return;
    }
    setPage(1);
    setSerialNumber(serialInput.trim() || undefined);
    setReasonCode(reasonInput.trim() || undefined);
    setPolicyVersionId(policyVersionInput.trim() || undefined);
    setMatchedRuleId(matchedRuleInput.trim() || undefined);
  };
  const resetFilters = () => {
    setSerialInput(''); setSerialNumber(undefined); setState(undefined); setDecision(undefined);
    setReasonInput(''); setReasonCode(undefined); setActionStatus(undefined); setDimension(undefined);
    setPolicyVersionInput(''); setPolicyVersionId(undefined); setMatchedRuleInput(''); setMatchedRuleId(undefined);
    setStartedAt(undefined); setEndedAt(undefined); setDatePickerKey((value) => value + 1); setPage(1);
  };
  return (
    <>
      <Space wrap className={styles.filterBar}>
        <Input allowClear value={serialInput} onChange={(e) => setSerialInput(e.target.value)} onPressEnter={search} placeholder={t('deviceAccess.serialPlaceholder')} style={{ width: 260 }} />
        <Select allowClear value={state} onChange={(v) => { setState(v); setPage(1); }} placeholder={t('deviceAccess.allStates')} style={{ width: 180 }} options={(['review_required', 'collecting', 'accepted', 'rejected', 'revalidating', 'revoked'] as AccessState[]).map((v) => ({ value: v, label: t(`deviceAccess.state.${v}`) }))} />
        <DatePicker.RangePicker key={datePickerKey} showTime onChange={(range) => {
          setStartedAt(range?.[0]?.toISOString());
          setEndedAt(range?.[1]?.toISOString());
          setPage(1);
        }} />
        <Button type="primary" icon={<SearchOutlined />} onClick={search}>{t('common.search')}</Button>
        <Button onClick={resetFilters}>{t('common.reset')}</Button>
        <Button icon={<FilterOutlined />} onClick={() => setAdvancedFiltersVisible((visible) => !visible)}>{t('deviceAccess.advancedFilters')}</Button>
        <Button icon={<ReloadOutlined />} onClick={() => { void query.refetch(); void summary.refetch(); }}>{t('common.refresh')}</Button>
      </Space>
      {advancedFiltersVisible && <Space wrap className={styles.filterBar}>
        <Select allowClear value={decision} onChange={(value) => { setDecision(value); setPage(1); }} placeholder={t('deviceAccess.allDecisions')} style={{ width: 170 }} options={['accept', 'reject', 'review', 'revoke', 'bypass'].map((value) => ({ value, label: t(`deviceAccess.decision.${value}`) }))} />
        <Input allowClear value={reasonInput} onChange={(event) => setReasonInput(event.target.value)} onPressEnter={search} placeholder={t('deviceAccess.reasonCode')} style={{ width: 190 }} />
        <Select allowClear value={dimension} onChange={(value) => { setDimension(value); setPage(1); }} placeholder={t('deviceAccess.dimension')} style={{ width: 150 }} options={(['sn', 'tac', 'ecgi', 'ip', 'gps'] as RuleDimension[]).map((value) => ({ value, label: value.toUpperCase() }))} />
        <Select allowClear value={actionStatus} onChange={(value) => { setActionStatus(value); setPage(1); }} placeholder={t('deviceAccess.actionStatus')} style={{ width: 180 }} options={(['pending_dispatch', 'dispatching', 'verifying', 'retry_wait', 'succeeded', 'failed', 'dead', 'cancelled'] as ActionStatus[]).map((value) => ({ value, label: t(`deviceAccess.actionStatus.${value}`) }))} />
        <Input allowClear value={policyVersionInput} onChange={(event) => setPolicyVersionInput(event.target.value)} onPressEnter={search} placeholder={t('deviceAccess.policyVersionId')} style={{ width: 240 }} />
        <Input allowClear value={matchedRuleInput} onChange={(event) => setMatchedRuleInput(event.target.value)} onPressEnter={search} placeholder={t('deviceAccess.matchedRuleId')} style={{ width: 240 }} />
      </Space>}
      <QueryError error={summary.error} t={t} />
      <Row gutter={12} style={{ marginBottom: 16 }}>
        <Col xs={12} sm={8} md={4}><Card size="small"><Statistic title={t('deviceAccess.summary.total')} value={summary.data?.total ?? 0} /></Card></Col>
        <Col xs={12} sm={8} md={5}><Card size="small"><Statistic title={t('deviceAccess.summary.accepted')} value={summary.data?.accepted ?? 0} /></Card></Col>
        <Col xs={12} sm={8} md={5}><Card size="small"><Statistic title={t('deviceAccess.summary.rejected')} value={summary.data?.rejected ?? 0} /></Card></Col>
        <Col xs={12} sm={8} md={5}><Card size="small"><Statistic title={t('deviceAccess.summary.reviewRequired')} value={summary.data?.review_required ?? 0} /></Card></Col>
        <Col xs={12} sm={8} md={5}><Card size="small"><Statistic title={t('deviceAccess.summary.revoked')} value={summary.data?.revoked ?? 0} /></Card></Col>
      </Row>
      <QueryError error={query.error} t={t} />
      <Table className={styles.dataTable} rowKey="serial_number" columns={columns} dataSource={query.data?.items ?? []} loading={query.isLoading} scroll={{ x: 1390 }} pagination={{ current: page, pageSize, total: query.data?.total ?? 0, showSizeChanger: true, onChange: (p, size) => { setPage(p); setPageSize(size); } }} />
      <Modal
        open={Boolean(quickListTarget)}
        title={quickListTarget?.type === 'deny' ? t('deviceAccess.confirmAddToDenylist') : t('deviceAccess.confirmAddToAllowlist')}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        okButtonProps={{ danger: quickListTarget?.type === 'deny', disabled: !quickListReason.trim() }}
        confirmLoading={upsertEntry.isPending}
        closable={!upsertEntry.isPending}
        mask={{ closable: !upsertEntry.isPending }}
        onCancel={() => { if (!upsertEntry.isPending) setQuickListTarget(undefined); }}
        onOk={async () => {
          if (!quickListTarget || !quickListReason.trim()) return;
          try {
            await upsertEntry.mutateAsync({ operatorCode, entryType: quickListTarget.type, serialNumber: quickListTarget.row.serial_number, reason: quickListReason.trim() });
            void message.success(t(businessEnabled ? 'deviceAccess.quickListSubmitted' : 'deviceAccess.quickListSavedWhileDisabled'));
            setQuickListTarget(undefined);
          } catch (error) {
            void message.error(error instanceof Error ? error.message : t('common.operationFailed'));
          }
        }}
      >
        <Typography.Paragraph strong>{quickListTarget?.row.serial_number} · {quickListTarget ? formatProductName(quickListTarget.row.product_name, t) : ''}</Typography.Paragraph>
        <Typography.Paragraph type={quickListTarget?.type === 'deny' ? 'danger' : undefined}>
          {quickListTarget?.type === 'deny' ? t('deviceAccess.quickDenyConsequence') : t('deviceAccess.quickAllowConsequence')}
        </Typography.Paragraph>
        <Input.TextArea value={quickListReason} onChange={(event) => setQuickListReason(event.target.value)} maxLength={500} showCount autoSize={{ minRows: 3, maxRows: 6 }} placeholder={t('deviceAccess.quickListReasonRequired')} />
      </Modal>
      {selectedSN && <AccessDetailDrawer operatorCode={operatorCode} serialNumber={selectedSN} open canArchive={canArchive} t={t} onClose={() => setSelectedSN(undefined)} />}
    </>
  );
}
