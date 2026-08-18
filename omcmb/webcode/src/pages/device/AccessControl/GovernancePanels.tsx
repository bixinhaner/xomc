import { useState, type Key } from 'react';
import { Alert, App, Button, Form, Input, Modal, Select, Space, Table, Tag, Tooltip } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DeleteOutlined, PlusOutlined, ReloadOutlined, StopOutlined } from '@ant-design/icons';
import {
  useAccessCandidates,
  useAccessEntries,
  useAccessPolicies,
  useAccessPolicy,
  useCreateAccessPolicyDraft,
  useDeleteAccessPolicyDraft,
  usePublishAccessPolicy,
  useReviewAccessCandidate,
  useUpsertAccessEntry,
} from '@core/hooks/api/useDeviceAccess';
import type { AccessListItem, AccessListType, CandidateItem, PolicyVersionSummary } from '@core/services/api/deviceAccessApi';
import { formatSystemTime } from '@core/utils/systemTime';
import { formatProductName, GuardedButton, QueryError } from './shared';
import { PolicyEditorModal } from './PolicyEditorModal';
import styles from './GovernancePanels.module.css';

interface Props { operatorCode: string; t: (key: string) => string }
interface PermissionProps extends Props { allowed: boolean }

export function PolicyPanel({ operatorCode, t, allowed }: PermissionProps) {
  const { message, modal } = App.useApp();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [editorOpen, setEditorOpen] = useState(false);
  const [sourceVersionID, setSourceVersionID] = useState<string>();
  const [publishingVersionID, setPublishingVersionID] = useState<string>();
  const [deletingVersionID, setDeletingVersionID] = useState<string>();
  const query = useAccessPolicies({ operatorCode, page, pageSize });
  const sourceQuery = useAccessPolicy(operatorCode, sourceVersionID);
  const createDraft = useCreateAccessPolicyDraft();
  const publish = usePublishAccessPolicy();
  const deleteDraft = useDeleteAccessPolicyDraft();
  const policySetName = query.data?.items[0]?.name;
  const doPublish = (row: PolicyVersionSummary) => modal.confirm({
    title: t('deviceAccess.publishPolicy'),
    content: t('deviceAccess.publishWarning'),
    okText: t('common.confirm'),
    cancelText: t('common.cancel'),
    onOk: async () => {
      setPublishingVersionID(row.id);
      try { await publish.mutateAsync({ operatorCode, versionId: row.id }); void message.success(t('deviceAccess.publishSuccess')); }
      catch (error) { void message.error(error instanceof Error ? error.message : t('common.operationFailed')); }
      finally { setPublishingVersionID(undefined); }
    },
  });
  const doDeleteDraft = (row: PolicyVersionSummary) => modal.confirm({
    title: t('deviceAccess.deleteDraft'),
    content: t('deviceAccess.deleteDraftWarning'),
    okText: t('common.delete'),
    okButtonProps: { danger: true },
    cancelText: t('common.cancel'),
    onOk: async () => {
      setDeletingVersionID(row.id);
      try {
        await deleteDraft.mutateAsync({ operatorCode, versionId: row.id });
        void message.success(t('deviceAccess.deleteDraftSuccess'));
      } catch (error) {
        void message.error(error instanceof Error ? error.message : t('common.operationFailed'));
        throw error;
      } finally {
        setDeletingVersionID(undefined);
      }
    },
  });
  const columns: ColumnsType<PolicyVersionSummary> = [
    { title: t('common.name'), dataIndex: 'name' },
    { title: t('deviceAccess.version'), dataIndex: 'version', width: 100 },
    { title: t('common.status'), dataIndex: 'status', width: 130, render: (v: string) => <Tag color={v === 'published' ? 'green' : v === 'draft' ? 'orange' : 'default'}>{t(`deviceAccess.policyStatus.${v}`)}</Tag> },
    { title: t('deviceAccess.defaultAction'), dataIndex: 'default_action', width: 150, render: (v: string) => t(`deviceAccess.defaultAction.${v}`) },
    { title: t('deviceAccess.publishedAt'), dataIndex: 'published_at', width: 190, render: (v?: string) => v ? formatSystemTime(v) : '—' },
    { title: t('common.createdAt'), dataIndex: 'created_at', width: 190, render: (v: string) => formatSystemTime(v) },
    { title: t('common.action'), width: 280, render: (_, row) => <Space>
      <GuardedButton type="link" allowed={allowed} deniedText={t('common.noPermission')} onClick={() => { setSourceVersionID(row.id); setEditorOpen(true); }}>{t('deviceAccess.clone')}</GuardedButton>
      {row.status === 'draft' && <GuardedButton type="link" allowed={allowed} deniedText={t('common.noPermission')} loading={publish.isPending && publishingVersionID === row.id} onClick={() => doPublish(row)}>{t('deviceAccess.publish')}</GuardedButton>}
      {row.status === 'draft' && <GuardedButton danger type="link" icon={<DeleteOutlined />} allowed={allowed} deniedText={t('common.noPermission')} loading={deleteDraft.isPending && deletingVersionID === row.id} onClick={() => doDeleteDraft(row)}>{t('common.delete')}</GuardedButton>}
    </Space> },
  ];
  const submitDraft = async (name: string, policy: Parameters<typeof createDraft.mutateAsync>[0]['policy']) => {
    try {
      await createDraft.mutateAsync({ operatorCode, name, policy });
      void message.success(t('deviceAccess.draftSaved'));
      setEditorOpen(false);
      setSourceVersionID(undefined);
    } catch (error) {
      void message.error(error instanceof Error ? error.message : t('common.operationFailed'));
    }
  };
  return <>
    <Space wrap className={styles.tableToolbar}>
      <Button icon={<ReloadOutlined />} onClick={() => void query.refetch()}>{t('common.refresh')}</Button>
      <GuardedButton type="primary" icon={<PlusOutlined />} allowed={allowed} deniedText={t('common.noPermission')} onClick={() => { setSourceVersionID(undefined); setEditorOpen(true); }}>{t('deviceAccess.createPolicyDraft')}</GuardedButton>
    </Space>
    <QueryError error={query.error ?? sourceQuery.error} t={t} />
    <Table rowKey="id" columns={columns} dataSource={query.data?.items ?? []} loading={query.isLoading} scroll={{ x: 1050 }} pagination={{ current: page, pageSize, total: query.data?.total ?? 0, showSizeChanger: true, onChange: (p, size) => { setPage(p); setPageSize(size); } }} />
    <PolicyEditorModal
      open={editorOpen}
      sourceLoading={Boolean(sourceVersionID) && sourceQuery.isLoading}
      submitting={createDraft.isPending}
      source={sourceVersionID ? sourceQuery.data : undefined}
      fixedPolicyName={policySetName}
      t={t}
      onCancel={() => { setEditorOpen(false); setSourceVersionID(undefined); }}
      onSubmit={submitDraft}
    />
  </>;
}

export function AccessListPanel({ operatorCode, t, allowed }: PermissionProps) {
  const { message } = App.useApp();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [entryType, setEntryType] = useState<AccessListType>();
  const [open, setOpen] = useState(false);
  const [disablingEntry, setDisablingEntry] = useState<AccessListItem>();
  const [selectedRowKeys, setSelectedRowKeys] = useState<Key[]>([]);
  const [batchEntries, setBatchEntries] = useState<AccessListItem[]>();
  const [form] = Form.useForm();
  const query = useAccessEntries({ operatorCode, page, pageSize, entryType });
  const upsert = useUpsertAccessEntry();
  const submit = async () => {
    try {
      const values = await form.validateFields();
      await upsert.mutateAsync({ operatorCode, entryType: values.entryType, serialNumber: values.serialNumber.trim(), reason: values.reason?.trim() });
      void message.success(t('deviceAccess.listSaved')); setOpen(false); form.resetFields();
    } catch (error) { if (error instanceof Error) void message.error(error.message); }
  };
  const disableEntry = async () => {
    if (!disablingEntry) return;
    try {
      await upsert.mutateAsync({ operatorCode, entryType: disablingEntry.entry_type, serialNumber: disablingEntry.identity_value, reason: disablingEntry.reason, status: 'disabled' });
      setSelectedRowKeys((current) => current.filter((key) => key !== disablingEntry.id));
      setDisablingEntry(undefined);
      void message.success(t('deviceAccess.disableSuccess'));
    } catch (error) {
      void message.error(error instanceof Error ? error.message : t('common.operationFailed'));
    }
  };
  const disableBatch = async () => {
    if (!batchEntries?.length) return;
    const failed: AccessListItem[] = [];
    for (const entry of batchEntries) {
      try {
        await upsert.mutateAsync({
          operatorCode,
          entryType: entry.entry_type,
          serialNumber: entry.identity_value,
          reason: entry.reason,
          status: 'disabled',
        });
      } catch {
        failed.push(entry);
      }
    }
    setBatchEntries(undefined);
    setSelectedRowKeys(failed.map((entry) => entry.id));
    if (failed.length > 0) {
      void message.error(`${t('deviceAccess.batchDisablePartial')}: ${batchEntries.length - failed.length}/${batchEntries.length}`);
    } else {
      void message.success(t('deviceAccess.batchDisableSuccess'));
    }
  };
  const columns: ColumnsType<AccessListItem> = [
    { title: t('deviceAccess.entryType'), dataIndex: 'entry_type', width: 130, render: (v: string) => <Tag color={v === 'allow' ? 'green' : v === 'deny' ? 'red' : 'volcano'}>{t(`deviceAccess.listType.${v}`)}</Tag> },
    { title: t('deviceAccess.serialNumber'), dataIndex: 'identity_value' },
    { title: t('deviceAccess.productName'), dataIndex: 'product_name', render: (v: string) => formatProductName(v, t) },
    { title: t('deviceAccess.reason'), dataIndex: 'reason', ellipsis: true },
    { title: t('common.status'), dataIndex: 'status', width: 110, render: (v: string) => t(`deviceAccess.listStatus.${v}`) },
    { title: t('deviceAccess.validFrom'), dataIndex: 'valid_from', width: 190, render: (v: string) => formatSystemTime(v) },
    { title: t('common.updatedAt'), dataIndex: 'updated_at', width: 190, render: (v: string) => formatSystemTime(v) },
    { title: t('common.action'), width: 110, align: 'center', render: (_, row) => row.status === 'active' ? <GuardedButton danger type="link" icon={<StopOutlined />} allowed={allowed} deniedText={t('common.noPermission')} loading={upsert.isPending && disablingEntry?.id === row.id} onClick={() => setDisablingEntry(row)}>{t('common.disable')}</GuardedButton> : '—' },
  ];
  return <>
    <Space wrap style={{ marginBottom: 16 }}>
      <Select allowClear value={entryType} onChange={(v) => { setEntryType(v); setPage(1); }} placeholder={t('deviceAccess.allListTypes')} style={{ width: 180 }} options={(['allow', 'deny', 'revoked'] as AccessListType[]).map((v) => ({ value: v, label: t(`deviceAccess.listType.${v}`) }))} />
      <Button icon={<ReloadOutlined />} onClick={() => void query.refetch()}>{t('common.refresh')}</Button>
      <GuardedButton
        danger
        icon={<StopOutlined />}
        allowed={allowed}
        deniedText={t('common.noPermission')}
        disabled={selectedRowKeys.length === 0}
        onClick={() => setBatchEntries((query.data?.items ?? []).filter((item) => selectedRowKeys.includes(item.id)))}
      >{t('deviceAccess.batchDisable')} ({selectedRowKeys.length})</GuardedButton>
      <GuardedButton type="primary" icon={<PlusOutlined />} allowed={allowed} deniedText={t('common.noPermission')} onClick={() => setOpen(true)}>{t('deviceAccess.addListEntry')}</GuardedButton>
    </Space>
    <Alert showIcon type="info" title={t('deviceAccess.listSemanticsHint')} style={{ marginBottom: 16 }} />
    <QueryError error={query.error} t={t} />
    <Table
      rowKey="id"
      rowSelection={{
        selectedRowKeys,
        onChange: setSelectedRowKeys,
        getCheckboxProps: (row) => ({ disabled: !allowed || row.status !== 'active' }),
      }}
      columns={columns}
      dataSource={query.data?.items ?? []}
      loading={query.isLoading}
      pagination={{ current: page, pageSize, total: query.data?.total ?? 0, showSizeChanger: true, onChange: (p, size) => { setPage(p); setPageSize(size); setSelectedRowKeys([]); } }}
    />
    <Modal open={open} title={t('deviceAccess.addListEntry')} okText={t('common.save')} cancelText={t('common.cancel')} confirmLoading={upsert.isPending} onOk={() => void submit()} onCancel={() => setOpen(false)}>
      <Form form={form} layout="vertical" initialValues={{ entryType: 'deny' }}>
        <Form.Item name="entryType" label={t('deviceAccess.entryType')} rules={[{ required: true }]}><Select options={(['allow', 'deny', 'revoked'] as AccessListType[]).map((v) => ({ value: v, label: t(`deviceAccess.listType.${v}`) }))} /></Form.Item>
        <Form.Item name="serialNumber" label={t('deviceAccess.serialNumber')} rules={[{ required: true, whitespace: true }]}><Input /></Form.Item>
        <Form.Item name="reason" label={t('deviceAccess.reason')} rules={[{ required: true, whitespace: true }]}><Input.TextArea rows={3} maxLength={500} showCount /></Form.Item>
      </Form>
    </Modal>
    <Modal
      open={Boolean(disablingEntry)}
      title={t('common.disable')}
      okText={t('common.confirm')}
      cancelText={t('common.cancel')}
      confirmLoading={upsert.isPending}
      cancelButtonProps={{ disabled: upsert.isPending }}
      closable={!upsert.isPending}
      keyboard={!upsert.isPending}
      mask={{ closable: !upsert.isPending }}
      onOk={() => void disableEntry()}
      onCancel={() => setDisablingEntry(undefined)}
    >
      <p>{t('deviceAccess.disableWarning')}</p>
      <p>{t('deviceAccess.serialNumber')}: {disablingEntry?.identity_value}；{t('deviceAccess.productName')}: {formatProductName(disablingEntry?.product_name, t)}</p>
    </Modal>
    <Modal
      open={Boolean(batchEntries?.length)}
      title={`${t('deviceAccess.batchDisable')} (${batchEntries?.length ?? 0})`}
      okText={t('common.confirm')}
      cancelText={t('common.cancel')}
      okButtonProps={{ danger: true }}
      confirmLoading={upsert.isPending}
      cancelButtonProps={{ disabled: upsert.isPending }}
      closable={!upsert.isPending}
      keyboard={!upsert.isPending}
      mask={{ closable: !upsert.isPending }}
      onOk={() => void disableBatch()}
      onCancel={() => setBatchEntries(undefined)}
    >
      <Alert showIcon type="warning" title={t('deviceAccess.batchDisableWarning')} style={{ marginBottom: 12 }} />
      <div style={{ maxHeight: 260, overflowY: 'auto' }}>
        {(batchEntries ?? []).map((entry) => <div key={entry.id} style={{ padding: '10px 0', borderBottom: '1px solid var(--ant-color-border-secondary)' }}>
          <strong>{entry.identity_value}</strong>
          <div style={{ color: 'var(--ant-color-text-secondary)' }}>{t('deviceAccess.productName')}: {formatProductName(entry.product_name, t)} · {t(`deviceAccess.listType.${entry.entry_type}`)}</div>
        </div>)}
      </div>
    </Modal>
  </>;
}

export function CandidatePanel({ operatorCode, t, allowed }: PermissionProps) {
  const { message } = App.useApp();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [status, setStatus] = useState('pending');
  const [review, setReview] = useState<{ candidate: CandidateItem; outcome: 'allow' | 'deny' }>();
  const [reason, setReason] = useState('');
  const query = useAccessCandidates({ operatorCode, page, pageSize, reviewStatus: status || undefined });
  const mutation = useReviewAccessCandidate();
  const submit = async () => {
    if (!review || !reason.trim()) return;
    try { await mutation.mutateAsync({ operatorCode, candidateId: review.candidate.id, outcome: review.outcome, reason: reason.trim() }); void message.success(t('deviceAccess.reviewSaved')); setReview(undefined); setReason(''); }
    catch (error) { void message.error(error instanceof Error ? error.message : t('common.operationFailed')); }
  };
  const columns: ColumnsType<CandidateItem> = [
    { title: t('deviceAccess.serialNumber'), dataIndex: 'serial_number', width: 220 },
    { title: 'OUI', dataIndex: 'oui', width: 110 },
    { title: t('deviceAccess.productClass'), dataIndex: 'product_class' },
    { title: t('deviceAccess.remoteIP'), dataIndex: 'observed_remote_ip', width: 150 },
    { title: t('deviceAccess.informCount'), dataIndex: 'inform_count', width: 100 },
    { title: t('deviceAccess.lastSeenAt'), dataIndex: 'last_seen_at', width: 190, render: (v: string) => formatSystemTime(v) },
    { title: t('common.status'), dataIndex: 'review_status', width: 110, render: (v: string) => t(`deviceAccess.reviewStatus.${v}`) },
    { title: t('common.action'), width: 180, render: (_, row) => row.review_status === 'pending' ? <Space><Tooltip title={allowed ? undefined : t('common.noPermission')}><Button size="small" disabled={!allowed} onClick={() => setReview({ candidate: row, outcome: 'allow' })}>{t('deviceAccess.allow')}</Button></Tooltip><Tooltip title={allowed ? undefined : t('common.noPermission')}><Button danger size="small" disabled={!allowed} onClick={() => setReview({ candidate: row, outcome: 'deny' })}>{t('deviceAccess.deny')}</Button></Tooltip></Space> : '—' },
  ];
  return <>
    <Space wrap style={{ marginBottom: 16 }}><Select value={status} onChange={(v) => { setStatus(v); setPage(1); }} style={{ width: 180 }} options={['pending', 'approved', 'rejected', 'expired'].map((v) => ({ value: v, label: t(`deviceAccess.reviewStatus.${v}`) }))} /><Button icon={<ReloadOutlined />} onClick={() => void query.refetch()}>{t('common.refresh')}</Button></Space>
    <QueryError error={query.error} t={t} />
    <Table rowKey="id" columns={columns} dataSource={query.data?.items ?? []} loading={query.isLoading} scroll={{ x: 1100 }} pagination={{ current: page, pageSize, total: query.data?.total ?? 0, showSizeChanger: true, onChange: (p, size) => { setPage(p); setPageSize(size); } }} />
    <Modal open={Boolean(review)} title={review ? t(`deviceAccess.review.${review.outcome}`) : ''} okText={t('common.confirm')} cancelText={t('common.cancel')} okButtonProps={{ danger: review?.outcome === 'deny', disabled: !reason.trim() }} confirmLoading={mutation.isPending} onOk={() => void submit()} onCancel={() => { setReview(undefined); setReason(''); }}>
      <p>{review?.candidate.serial_number}</p><Input.TextArea value={reason} onChange={(e) => setReason(e.target.value)} placeholder={t('deviceAccess.reviewReasonPlaceholder')} rows={4} maxLength={500} showCount />
    </Modal>
  </>;
}
