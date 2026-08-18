import { useState } from 'react';
import { App, Button, Collapse, Descriptions, Drawer, Input, Pagination, Select, Space, Table, Tag, Tooltip, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { EyeOutlined, ReloadOutlined, SearchOutlined, SyncOutlined } from '@ant-design/icons';
import { useAccessDetail, useAccessStates, useReevaluateAccessDevice } from '@core/hooks/api/useDeviceAccess';
import type { AccessState, AccessStateItem, DecisionItem, EvidenceItem } from '@core/services/api/deviceAccessApi';
import { formatSystemTime } from '@core/utils/systemTime';
import { ActionStatusTag, formatProductName, normalTaskProtectionPresentation, QueryError, StateTag } from './shared';

interface Props {
  operatorCode: string;
  t: (key: string) => string;
  canReevaluate: boolean;
  businessEnabled?: boolean;
}

const decisionHistoryPageSize = 10;

function formatValue(value: unknown): string {
  if (value === null || value === undefined || value === '') return '—';
  if (typeof value === 'string') return value;
  return JSON.stringify(value);
}

export default function AccessStatesPanel({ operatorCode, t, canReevaluate, businessEnabled = true }: Props) {
  const { message } = App.useApp();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [serialInput, setSerialInput] = useState('');
  const [serialNumber, setSerialNumber] = useState<string>();
  const [state, setState] = useState<AccessState>();
  const [selectedSN, setSelectedSN] = useState<string>();
  const [decisionHistoryPage, setDecisionHistoryPage] = useState(1);
  const [reevaluatingSN, setReevaluatingSN] = useState<string>();
  const query = useAccessStates({ operatorCode, page, pageSize, serialNumber, state });
  const detail = useAccessDetail(operatorCode, selectedSN);
  const reevaluate = useReevaluateAccessDevice();
  const decisions = detail.data?.decisions ?? [];
  const maxDecisionHistoryPage = Math.max(1, Math.ceil(decisions.length / decisionHistoryPageSize));
  const visibleDecisionHistoryPage = Math.min(decisionHistoryPage, maxDecisionHistoryPage);
  const visibleDecisions = decisions.slice(
    (visibleDecisionHistoryPage - 1) * decisionHistoryPageSize,
    visibleDecisionHistoryPage * decisionHistoryPageSize,
  );

  const columns: ColumnsType<AccessStateItem> = [
    { title: t('deviceAccess.serialNumber'), dataIndex: 'serial_number', width: 210 },
    { title: t('deviceAccess.productName'), dataIndex: 'product_name', width: 180, render: (v: string) => formatProductName(v, t) },
    { title: t('deviceAccess.accessState'), dataIndex: 'state', width: 150, render: (v: AccessState) => <StateTag value={v} t={t} /> },
    { title: t('deviceAccess.effectiveDecision'), dataIndex: 'effective_decision', width: 140 },
    { title: t('deviceAccess.reasonCode'), dataIndex: 'reason_code', ellipsis: true },
    { title: t('deviceAccess.normalTaskProtection'), dataIndex: 'normal_tasks_frozen', width: 230, render: (v: boolean) => {
      const presentation = normalTaskProtectionPresentation(v);
      return <Tag color={presentation.color}>{t(presentation.messageKey)}</Tag>;
    } },
    { title: t('deviceAccess.decisionVersion'), dataIndex: 'decision_version', width: 110 },
    { title: t('common.updatedAt'), dataIndex: 'updated_at', width: 180, render: (v: string) => formatSystemTime(v) },
    { title: t('common.action'), fixed: 'right', width: 210, align: 'center', render: (_, row) => <Space size={4}>
      <Button type="link" icon={<EyeOutlined />} onClick={() => { setDecisionHistoryPage(1); setSelectedSN(row.serial_number); }}>{t('common.detail')}</Button>
      <Tooltip title={!canReevaluate ? t('common.noPermission') : (!businessEnabled ? t('deviceAccess.switchRequired') : undefined)}>
        <Button
          type="link"
          icon={<SyncOutlined />}
          disabled={!canReevaluate || !businessEnabled || (reevaluate.isPending && reevaluatingSN !== row.serial_number)}
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
        >{t('deviceAccess.reevaluate')}</Button>
      </Tooltip>
    </Space> },
  ];

  const search = () => { setPage(1); setSerialNumber(serialInput.trim() || undefined); };
  return (
    <>
      <Space wrap style={{ marginBottom: 16 }}>
        <Input allowClear value={serialInput} onChange={(e) => setSerialInput(e.target.value)} onPressEnter={search} placeholder={t('deviceAccess.serialPlaceholder')} style={{ width: 260 }} />
        <Select allowClear value={state} onChange={(v) => { setState(v); setPage(1); }} placeholder={t('deviceAccess.allStates')} style={{ width: 180 }} options={(['review_required', 'collecting', 'accepted', 'rejected', 'revalidating', 'revoked'] as AccessState[]).map((v) => ({ value: v, label: t(`deviceAccess.state.${v}`) }))} />
        <Button type="primary" icon={<SearchOutlined />} onClick={search}>{t('common.search')}</Button>
        <Button icon={<ReloadOutlined />} onClick={() => void query.refetch()}>{t('common.refresh')}</Button>
      </Space>
      <QueryError error={query.error} t={t} />
      <Table rowKey="serial_number" columns={columns} dataSource={query.data?.items ?? []} loading={query.isLoading} scroll={{ x: 1370 }} pagination={{ current: page, pageSize, total: query.data?.total ?? 0, showSizeChanger: true, onChange: (p, size) => { setPage(p); setPageSize(size); } }} />
      <Drawer size="large" open={Boolean(selectedSN)} title={`${t('deviceAccess.decisionDetail')} — ${selectedSN ?? ''}`} onClose={() => { setSelectedSN(undefined); setDecisionHistoryPage(1); }} loading={detail.isLoading}>
        <QueryError error={detail.error} t={t} />
        {detail.data && <>
          <Descriptions bordered size="small" column={2} style={{ marginBottom: 20 }}>
            <Descriptions.Item label={t('deviceAccess.serialNumber')}>{detail.data.state.serial_number}</Descriptions.Item>
            <Descriptions.Item label={t('deviceAccess.productName')}>{formatProductName(detail.data.state.product_name, t)}</Descriptions.Item>
            <Descriptions.Item label={t('deviceAccess.accessState')}><StateTag value={detail.data.state.state} t={t} /></Descriptions.Item>
            <Descriptions.Item label={t('deviceAccess.reasonCode')}>{detail.data.state.reason_code}</Descriptions.Item>
            <Descriptions.Item label={t('deviceAccess.evidenceVersion')}>{detail.data.state.evidence_version}</Descriptions.Item>
          </Descriptions>
          <Typography.Title level={5}>{t('deviceAccess.decisionChecks')}</Typography.Title>
          <Collapse items={visibleDecisions.map((decision: DecisionItem) => ({
            key: decision.id,
            label: <Space wrap><StateTag value={decision.new_state} t={t} /><span>v{decision.decision_version}</span><span>{formatSystemTime(decision.occurred_at)}</span></Space>,
            children: <Table size="small" rowKey="id" pagination={false} dataSource={decision.checks} columns={[
              { title: t('deviceAccess.checkType'), dataIndex: 'check_type' },
              { title: t('deviceAccess.result'), dataIndex: 'result', width: 110 },
              { title: t('deviceAccess.expected'), dataIndex: 'expected_summary' },
              { title: t('deviceAccess.observed'), dataIndex: 'observed_summary' },
              { title: t('deviceAccess.source'), dataIndex: 'evidence_source' },
            ]} />,
          }))} />
          {decisions.length > decisionHistoryPageSize && <Pagination
            size="small"
            current={visibleDecisionHistoryPage}
            pageSize={decisionHistoryPageSize}
            total={decisions.length}
            showSizeChanger={false}
            onChange={setDecisionHistoryPage}
            style={{ marginTop: 12, textAlign: 'right' }}
          />}
          <Typography.Title level={5} style={{ marginTop: 20 }}>{t('deviceAccess.evidence')}</Typography.Title>
          <Table<EvidenceItem> size="small" rowKey="id" pagination={false} dataSource={detail.data.evidence} columns={[
            { title: t('deviceAccess.evidenceType'), dataIndex: 'evidence_type' },
            { title: t('deviceAccess.result'), dataIndex: 'evidence_status', width: 120 },
            { title: t('deviceAccess.observed'), dataIndex: 'normalized_value', render: formatValue },
            { title: t('deviceAccess.source'), dataIndex: 'source' },
            { title: t('deviceAccess.observedAt'), dataIndex: 'observed_at', render: (v: string) => formatSystemTime(v) },
          ]} />
          <Typography.Title level={5} style={{ marginTop: 20 }}>{t('deviceAccess.actions')}</Typography.Title>
          <Table size="small" rowKey="id" pagination={false} dataSource={detail.data.actions} columns={[
            { title: t('deviceAccess.actionType'), dataIndex: 'action_type', render: (v: string) => t(`deviceAccess.actionType.${v}`) },
            { title: t('deviceAccess.actionStatus'), dataIndex: 'status', render: (v) => <ActionStatusTag value={v} t={t} /> },
            { title: t('deviceAccess.attempts'), dataIndex: 'attempts' },
            { title: t('common.updatedAt'), dataIndex: 'updated_at', render: (v: string) => formatSystemTime(v) },
          ]} />
        </>}
      </Drawer>
    </>
  );
}
