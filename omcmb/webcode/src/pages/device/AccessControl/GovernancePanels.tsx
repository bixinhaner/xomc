import { useState, type Key } from 'react';
import { Alert, App, Button, Descriptions, Form, Input, Modal, Select, Space, Statistic, Table, Tag, Typography, Upload } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { CheckOutlined, CloseOutlined, CopyOutlined, DeleteOutlined, DiffOutlined, DownloadOutlined, EditOutlined, EyeOutlined, HistoryOutlined, PlusOutlined, ReloadOutlined, RollbackOutlined, SearchOutlined, SendOutlined, StopOutlined, UploadOutlined } from '@ant-design/icons';
import {
  useAccessCandidates,
  useAccessEntries,
  useAccessPolicies,
  useAccessPolicy,
  useAccessPolicyDifference,
  useAccessListImports,
  useCommitAccessListImport,
  useCreateAccessPolicyDraft,
  useDeleteAccessPolicyDraft,
  useDisableAccessEntries,
  usePreviewAccessListImport,
  usePublishAccessPolicy,
  useReviewAccessCandidate,
  useRollbackAccessListImport,
  useRollbackAccessPolicy,
  useUpdateAccessPolicyDraft,
  useUpsertAccessEntry,
  useUpsertAccessEntries,
} from '@core/hooks/api/useDeviceAccess';
import { deviceAccessApi, type AccessListImportPreview, type AccessListItem, type AccessListType, type CandidateItem, type ImportFailurePolicy, type ImportMode, type PolicyVersionSummary } from '@core/services/api/deviceAccessApi';
import { formatSystemTime } from '@core/utils/systemTime';
import { formatProductName, GuardedButton, IconActionButton, QueryError } from './shared';
import { PolicyEditorModal } from './PolicyEditorModal';
import styles from './AccessControl.module.css';

interface Props { operatorCode: string; t: (key: string) => string }
interface PermissionProps extends Props { allowed: boolean }
interface PolicyPermissionProps extends PermissionProps { onDrilldownRule?: (policyVersionId: string, matchedRuleId: string) => void }
interface RuntimePermissionProps extends PermissionProps { businessEnabled: boolean; initialSerialNumber?: string }
interface CandidatePanelProps extends RuntimePermissionProps {
  candidateId?: string;
  reviewStatus: string;
  onReviewStatusChange: (status: string) => void;
  onCandidateHandled: () => void;
}

export function PolicyPanel({ operatorCode, t, allowed, onDrilldownRule }: PolicyPermissionProps) {
  const { message, modal } = App.useApp();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [editorOpen, setEditorOpen] = useState(false);
  const [sourceVersionID, setSourceVersionID] = useState<string>();
  const [editorMode, setEditorMode] = useState<'edit' | 'clone' | 'view'>('edit');
  const [publishingVersionID, setPublishingVersionID] = useState<string>();
  const [deletingVersionID, setDeletingVersionID] = useState<string>();
  const [draftSubmitting, setDraftSubmitting] = useState(false);
	const [comparingVersionID, setComparingVersionID] = useState<string>();
	const [rollingBackVersionID, setRollingBackVersionID] = useState<string>();
  const query = useAccessPolicies({ operatorCode, page, pageSize });
  const sourceQuery = useAccessPolicy(operatorCode, sourceVersionID);
  const createDraft = useCreateAccessPolicyDraft();
  const updateDraft = useUpdateAccessPolicyDraft();
  const publish = usePublishAccessPolicy();
  const deleteDraft = useDeleteAccessPolicyDraft();
	const difference = useAccessPolicyDifference();
	const rollbackPolicy = useRollbackAccessPolicy();
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
	const showDifference = async (row: PolicyVersionSummary) => {
		setComparingVersionID(row.id);
		try {
			const result = await difference.mutateAsync({ operatorCode, versionId: row.id });
			modal.info({
				title: t('deviceAccess.policyDifference'),
				content: <Space orientation="vertical">
					<span>{t('deviceAccess.policyDifferenceVersions')}: v{result.base_version ?? '—'} → v{result.target_version}</span>
					<span>{t('deviceAccess.policyDifferenceSettings')}: {Number(result.default_action_changed) + Number(result.failure_mode_changed) + Number(result.collection_timeout_changed)}</span>
					<span>{t('deviceAccess.policyDifferenceBypass')}: +{result.bypass_profiles_added} / -{result.bypass_profiles_removed} / ~{result.bypass_profiles_changed}</span>
					<span>{t('deviceAccess.policyDifferenceRules')}: +{result.rules_added} / -{result.rules_removed} / ~{result.rules_changed}</span>
					<span>{t('deviceAccess.policyDifferenceConditions')}: +{result.conditions_added} / -{result.conditions_removed}</span>
					{result.no_changes && <span>{t('deviceAccess.policyDifferenceNone')}</span>}
				</Space>,
			});
		} catch (error) {
			void message.error(error instanceof Error ? error.message : t('common.operationFailed'));
		} finally {
			setComparingVersionID(undefined);
		}
	};
	const doRollback = (row: PolicyVersionSummary) => modal.confirm({
		title: t('deviceAccess.rollbackPolicy'),
		content: t('deviceAccess.rollbackPolicyWarning'),
		okText: t('common.confirm'),
		cancelText: t('common.cancel'),
		onOk: async () => {
			setRollingBackVersionID(row.id);
			try {
				await rollbackPolicy.mutateAsync({ operatorCode, versionId: row.id });
				void message.success(t('deviceAccess.rollbackPolicySuccess'));
			} catch (error) {
				void message.error(error instanceof Error ? error.message : t('common.operationFailed'));
				throw error;
			} finally {
				setRollingBackVersionID(undefined);
			}
		},
	});
  const columns: ColumnsType<PolicyVersionSummary> = [
    {
      title: t('common.name'),
      dataIndex: 'name',
      width: 320,
      render: (value: string) => <Typography.Text className={styles.policyName} ellipsis={{ tooltip: value }}>{value}</Typography.Text>,
    },
    { title: t('deviceAccess.version'), dataIndex: 'version', width: 100 },
    { title: t('common.status'), dataIndex: 'status', width: 130, render: (v: string) => <Tag color={v === 'published' ? 'green' : v === 'draft' ? 'orange' : 'default'}>{t(`deviceAccess.policyStatus.${v}`)}</Tag> },
    { title: t('deviceAccess.defaultAction'), dataIndex: 'default_action', width: 150, render: (v: string) => t(`deviceAccess.defaultAction.${v}`) },
    { title: t('deviceAccess.failureMode'), dataIndex: 'failure_mode', width: 160, render: (v?: string) => v ? t(`deviceAccess.failureMode.${v}`) : '—' },
    { title: t('deviceAccess.bypassProfiles'), dataIndex: 'bypass_profile_count', width: 140, render: (v?: number) => v ?? 0 },
    { title: t('deviceAccess.publishedAt'), dataIndex: 'published_at', width: 190, render: (v?: string) => v ? formatSystemTime(v) : '—' },
    { title: t('common.createdAt'), dataIndex: 'created_at', width: 190, render: (v: string) => formatSystemTime(v) },
    { title: t('common.action'), fixed: 'right', width: 230, align: 'center', render: (_, row) => {
      const legacyReviewPolicy = row.default_action === 'review' || row.failure_mode === 'review_hold';
      return <div className={styles.iconActions}>
      <IconActionButton label={t('common.view')} icon={<EyeOutlined />} onClick={() => { setEditorMode('view'); setSourceVersionID(row.id); setEditorOpen(true); }} />
      {row.version > 1 && <IconActionButton label={t('deviceAccess.policyDifference')} icon={<DiffOutlined />} loading={difference.isPending && comparingVersionID === row.id} onClick={() => void showDifference(row)} />}
      <IconActionButton label={row.status === 'draft' ? t('common.edit') : t('deviceAccess.clone')} icon={row.status === 'draft' ? <EditOutlined /> : <CopyOutlined />} allowed={allowed} deniedText={t('common.noPermission')} onClick={() => { setEditorMode(row.status === 'draft' ? 'edit' : 'clone'); setSourceVersionID(row.id); setEditorOpen(true); }} />
      {row.status === 'draft' && <IconActionButton label={t('deviceAccess.publish')} icon={<SendOutlined />} allowed={allowed} deniedText={t('common.noPermission')} loading={publish.isPending && publishingVersionID === row.id} onClick={() => doPublish(row)} />}
      {row.status === 'retired' && <IconActionButton label={t('deviceAccess.rollbackPolicy')} icon={<RollbackOutlined />} allowed={allowed && !legacyReviewPolicy} deniedText={!allowed ? t('common.noPermission') : t('deviceAccess.legacyReviewRollbackBlocked')} loading={rollbackPolicy.isPending && rollingBackVersionID === row.id} onClick={() => doRollback(row)} />}
      {row.status === 'draft' && <IconActionButton danger label={t('common.delete')} icon={<DeleteOutlined />} allowed={allowed} deniedText={t('common.noPermission')} loading={deleteDraft.isPending && deletingVersionID === row.id} onClick={() => doDeleteDraft(row)} />}
    </div>;
    } },
  ];
  const submitDraft = async (name: string, policy: Parameters<typeof createDraft.mutateAsync>[0]['policy']) => {
    setDraftSubmitting(true);
    try {
      if (sourceQuery.data?.status === 'draft') {
        await updateDraft.mutateAsync({ operatorCode, versionId: sourceQuery.data.id, policy });
      } else {
        await createDraft.mutateAsync({ operatorCode, name, policy });
      }
      void message.success(t('deviceAccess.draftSaved'));
      setEditorOpen(false);
      setSourceVersionID(undefined);
    } catch (_error) {
      void message.error(t('deviceAccess.policySaveFailed'));
    } finally {
      // Keep the modal loading state owned by this submission. React Query
      // mutation state can otherwise remain visually stale after a rejected
      // request while the editor stays mounted for correction and retry.
      createDraft.reset();
      updateDraft.reset();
      setDraftSubmitting(false);
    }
  };
  return <>
    <Space wrap className={styles.tableToolbar}>
      <Button icon={<ReloadOutlined />} onClick={() => void query.refetch()}>{t('common.refresh')}</Button>
      <GuardedButton type="primary" icon={<PlusOutlined />} allowed={allowed} deniedText={t('common.noPermission')} onClick={() => { setEditorMode('edit'); setSourceVersionID(undefined); setEditorOpen(true); }}>{t('deviceAccess.createPolicyDraft')}</GuardedButton>
    </Space>
    <QueryError error={query.error ?? sourceQuery.error} t={t} />
    <Table className={styles.dataTable} rowKey="id" columns={columns} dataSource={query.data?.items ?? []} loading={query.isLoading} scroll={{ x: 1750 }} pagination={{ current: page, pageSize, total: query.data?.total ?? 0, showSizeChanger: true, onChange: (p, size) => { setPage(p); setPageSize(size); } }} />
    <PolicyEditorModal
      operatorCode={operatorCode}
      open={editorOpen}
      sourceLoading={Boolean(sourceVersionID) && sourceQuery.isLoading}
      submitting={draftSubmitting}
      source={sourceVersionID ? sourceQuery.data : undefined}
      readOnly={editorMode === 'view'}
      fixedPolicyName={policySetName}
      t={t}
      onCancel={() => { setEditorOpen(false); setSourceVersionID(undefined); setEditorMode('edit'); }}
      onSubmit={submitDraft}
      onDrilldownRule={onDrilldownRule}
    />
  </>;
}

export function AccessListPanel({ operatorCode, t, allowed, businessEnabled }: RuntimePermissionProps) {
  const { message, modal } = App.useApp();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [entryType, setEntryType] = useState<AccessListType>();
  const [status, setStatus] = useState<'active' | 'disabled'>();
  const [serialNumber, setSerialNumber] = useState('');
  const [productName, setProductName] = useState('');
  const [open, setOpen] = useState(false);
  const [disablingEntry, setDisablingEntry] = useState<AccessListItem>();
	const [disableReason, setDisableReason] = useState('');
  const [selectedRowKeys, setSelectedRowKeys] = useState<Key[]>([]);
  const [batchEntries, setBatchEntries] = useState<AccessListItem[]>();
  const [batchReason, setBatchReason] = useState('');
  const [importOpen, setImportOpen] = useState(false);
  const [historyOpen, setHistoryOpen] = useState(false);
  const [importFile, setImportFile] = useState<File>();
  const [importEntryType, setImportEntryType] = useState<AccessListType>('deny');
  const [importMode, setImportMode] = useState<ImportMode>('append');
  const [failurePolicy, setFailurePolicy] = useState<ImportFailurePolicy>('strict');
  const [preview, setPreview] = useState<AccessListImportPreview>();
	const [errorRows, setErrorRows] = useState<AccessListImportPreview['rows']>();
	const [errorRowsLoading, setErrorRowsLoading] = useState(false);
	const [errorBatchID, setErrorBatchID] = useState('');
  const [form] = Form.useForm();
  const query = useAccessEntries({ operatorCode, page, pageSize, entryType, status, serialNumber: serialNumber || undefined, productName: productName || undefined });
  const upsert = useUpsertAccessEntry();
  const upsertMany = useUpsertAccessEntries();
  const disableMany = useDisableAccessEntries();
  const previewImport = usePreviewAccessListImport();
  const commitImport = useCommitAccessListImport();
  const rollbackImport = useRollbackAccessListImport();
  const imports = useAccessListImports({ operatorCode, page: 1, pageSize: 50 });
  const submit = async () => {
    try {
      const values = await form.validateFields();
      const serialNumbers = splitSerialNumbers(values.serialNumbers);
      if (serialNumbers.length === 0) throw new Error(t('deviceAccess.serialRequired'));
      await upsertMany.mutateAsync({ operatorCode, entryType: values.entryType, serialNumbers, reason: values.reason.trim() });
      void message.success(t(businessEnabled ? 'deviceAccess.listSaved' : 'deviceAccess.listSavedWhileDisabled')); setOpen(false); form.resetFields();
    } catch (error) { if (error instanceof Error) void message.error(error.message); }
  };
  const disableEntry = async () => {
    if (!disablingEntry) return;
    try {
	  if (!disableReason.trim()) return;
	  await upsert.mutateAsync({ operatorCode, entryType: disablingEntry.entry_type, serialNumber: disablingEntry.identity_value, reason: disableReason.trim(), status: 'disabled' });
      setSelectedRowKeys((current) => current.filter((key) => key !== disablingEntry.id));
      setDisablingEntry(undefined);
	  setDisableReason('');
      void message.success(t(businessEnabled ? 'deviceAccess.disableSuccess' : 'deviceAccess.disableSavedWhileDisabled'));
    } catch (error) {
      void message.error(error instanceof Error ? error.message : t('common.operationFailed'));
    }
  };
	const showImportErrors = async (batchId: string) => {
	  setErrorBatchID(batchId);
	  setErrorRowsLoading(true);
	  try {
		setErrorRows(await deviceAccessApi.listAccessListImportErrors(operatorCode, batchId));
	  } catch (error) {
		void message.error(error instanceof Error ? error.message : t('common.operationFailed'));
	  } finally {
		setErrorRowsLoading(false);
	  }
	};
	const downloadImportErrors = () => {
	  if (!errorRows?.length) return;
	  const lines = [
		['Row Number', 'Error Code', 'Error Message', 'Raw Value'],
		...errorRows.map((row) => [String(row.row_number), row.error_code ?? '', row.error_message ?? '', JSON.stringify(row.raw_value ?? {})]),
	  ].map((row) => row.map(csvCell).join(','));
	  downloadBlob(new Blob([`\ufeff${lines.join('\r\n')}\r\n`], { type: 'text/csv;charset=utf-8' }), `device-access-import-${errorBatchID}-errors.csv`);
	};
  const disableBatch = async () => {
    if (!batchEntries?.length || !batchReason.trim()) return;
    try {
      await disableMany.mutateAsync({ operatorCode, entryType: batchEntries[0].entry_type, serialNumbers: batchEntries.map((entry) => entry.identity_value), reason: batchReason.trim() });
      setBatchEntries(undefined); setBatchReason(''); setSelectedRowKeys([]);
      void message.success(t(businessEnabled ? 'deviceAccess.batchDisableSuccess' : 'deviceAccess.batchDisableSavedWhileDisabled'));
    } catch (error) {
      void message.error(error instanceof Error ? error.message : t('common.operationFailed'));
    }
  };
  const openBatchDisable = () => {
    const selected = (query.data?.items ?? []).filter((item) => selectedRowKeys.includes(item.id));
    if (new Set(selected.map((item) => item.entry_type)).size > 1) {
      void message.error(t('deviceAccess.batchDisableSameType'));
      return;
    }
    setBatchEntries(selected);
  };
  const downloadTemplate = async () => {
    try {
      const blob = await deviceAccessApi.downloadAccessListTemplate(operatorCode);
      downloadBlob(blob, 'device-access-list-template.xlsx');
    } catch (error) { void message.error(error instanceof Error ? error.message : t('common.operationFailed')); }
  };
  const previewSelectedImport = async () => {
    if (!importFile) { void message.error(t('deviceAccess.importFileRequired')); return; }
    try {
      const result = await previewImport.mutateAsync({
        operatorCode, entryType: importEntryType, mode: importMode, failurePolicy, file: importFile,
        idempotencyKey: globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${importFile.name}`,
      });
      setPreview(result);
    } catch (error) { void message.error(error instanceof Error ? error.message : t('common.operationFailed')); }
  };
  const commitSelectedImport = async () => {
    if (!preview) return;
    try {
	  const riskyReplace = preview.batch.mode === 'replace' && preview.batch.failure_policy === 'valid_only' && preview.batch.invalid_count > 0;
	  if (riskyReplace) {
		const confirmed = await new Promise<boolean>((resolve) => modal.confirm({
		  title: t('deviceAccess.replaceValidOnlyConfirmTitle'),
		  content: t('deviceAccess.replaceValidOnlyConfirmContent'),
		  okText: t('common.confirm'), cancelText: t('common.cancel'), okButtonProps: { danger: true },
		  onOk: () => resolve(true), onCancel: () => resolve(false),
		}));
		if (!confirmed) return;
	  }
	  await commitImport.mutateAsync({ operatorCode, batchId: preview.batch.id, confirmReplaceWithInvalid: riskyReplace });
      void message.success(t('deviceAccess.importCommitted'));
      setImportOpen(false); setPreview(undefined); setImportFile(undefined);
    } catch (error) { void message.error(error instanceof Error ? error.message : t('common.operationFailed')); }
  };
  const rollbackBatch = (batchId: string) => modal.confirm({
    title: t('deviceAccess.rollbackImport'),
    content: t('deviceAccess.rollbackImportWarning'),
    okText: t('common.confirm'),
    cancelText: t('common.cancel'),
    okButtonProps: { danger: true },
    onOk: async () => {
      try {
        await rollbackImport.mutateAsync({ operatorCode, batchId });
        void message.success(t('deviceAccess.importRolledBack'));
      } catch (error) {
        void message.error(error instanceof Error ? error.message : t('common.operationFailed'));
        throw error;
      }
    },
  });
  const columns: ColumnsType<AccessListItem> = [
    { title: t('deviceAccess.entryType'), dataIndex: 'entry_type', width: 130, render: (v: string) => <Tag color={v === 'allow' ? 'green' : v === 'deny' ? 'red' : 'volcano'}>{t(`deviceAccess.listType.${v}`)}</Tag> },
    { title: t('deviceAccess.serialNumber'), dataIndex: 'identity_value' },
    { title: t('deviceAccess.productName'), dataIndex: 'product_name', render: (v: string) => formatProductName(v, t) },
    { title: t('deviceAccess.reason'), dataIndex: 'reason', ellipsis: true },
    { title: t('common.status'), dataIndex: 'status', width: 110, render: (v: string) => t(`deviceAccess.listStatus.${v}`) },
    { title: t('deviceAccess.validFrom'), dataIndex: 'valid_from', width: 190, render: (v: string) => formatSystemTime(v) },
    { title: t('common.updatedAt'), dataIndex: 'updated_at', width: 190, render: (v: string) => formatSystemTime(v) },
    { title: t('common.action'), fixed: 'right', width: 76, align: 'center', render: (_, row) => row.status === 'active' ? <IconActionButton danger label={t('common.disable')} icon={<StopOutlined />} allowed={allowed} deniedText={t('common.noPermission')} loading={upsert.isPending && disablingEntry?.id === row.id} onClick={() => setDisablingEntry(row)} /> : '—' },
  ];
  return <>
    <Space wrap className={styles.filterBar}>
      <Input.Search allowClear value={serialNumber} onChange={(event) => setSerialNumber(event.target.value)} onSearch={() => setPage(1)} placeholder={t('deviceAccess.serialNumber')} style={{ width: 220 }} />
      <Input.Search allowClear value={productName} onChange={(event) => setProductName(event.target.value)} onSearch={() => setPage(1)} placeholder={t('deviceAccess.productName')} style={{ width: 200 }} />
      <Select allowClear value={entryType} onChange={(v) => { setEntryType(v); setPage(1); }} placeholder={t('deviceAccess.allListTypes')} style={{ width: 180 }} options={(['allow', 'deny', 'revoked'] as AccessListType[]).map((v) => ({ value: v, label: t(`deviceAccess.listType.${v}`) }))} />
      <Select allowClear value={status} onChange={(value) => { setStatus(value); setPage(1); }} placeholder={t('deviceAccess.allListStatuses')} style={{ width: 150 }} options={(['active', 'disabled'] as const).map((value) => ({ value, label: t(`deviceAccess.listStatus.${value}`) }))} />
      <Button icon={<ReloadOutlined />} onClick={() => void query.refetch()}>{t('common.refresh')}</Button>
      <Button icon={<DownloadOutlined />} onClick={() => void downloadTemplate()}>{t('deviceAccess.downloadTemplate')}</Button>
      <GuardedButton icon={<UploadOutlined />} allowed={allowed} deniedText={t('common.noPermission')} onClick={() => setImportOpen(true)}>{t('deviceAccess.importList')}</GuardedButton>
      <Button icon={<HistoryOutlined />} onClick={() => setHistoryOpen(true)}>{t('deviceAccess.importHistory')}</Button>
      <GuardedButton
        danger
        icon={<StopOutlined />}
        allowed={allowed}
        deniedText={t('common.noPermission')}
        disabled={selectedRowKeys.length === 0}
        onClick={openBatchDisable}
      >{t('deviceAccess.batchDisable')} ({selectedRowKeys.length})</GuardedButton>
      <GuardedButton type="primary" icon={<PlusOutlined />} allowed={allowed} deniedText={t('common.noPermission')} onClick={() => setOpen(true)}>{t('deviceAccess.addListEntry')}</GuardedButton>
    </Space>
    <Alert showIcon type="info" title={t('deviceAccess.listSemanticsHint')} style={{ marginBottom: 16 }} />
    <QueryError error={query.error} t={t} />
    <Table
      className={styles.dataTable}
      rowKey="id"
      rowSelection={{
        selectedRowKeys,
        onChange: setSelectedRowKeys,
        getCheckboxProps: (row) => ({ disabled: !allowed || row.status !== 'active' }),
      }}
      columns={columns}
      dataSource={query.data?.items ?? []}
      loading={query.isLoading}
      scroll={{ x: 1280 }}
      pagination={{ current: page, pageSize, total: query.data?.total ?? 0, showSizeChanger: true, onChange: (p, size) => { setPage(p); setPageSize(size); setSelectedRowKeys([]); } }}
    />
    <Modal open={open} title={t('deviceAccess.addListEntry')} okText={t('common.save')} cancelText={t('common.cancel')} confirmLoading={upsertMany.isPending} onOk={() => void submit()} onCancel={() => setOpen(false)}>
      <Form form={form} layout="vertical" initialValues={{ entryType: 'deny' }}>
        <Form.Item name="entryType" label={t('deviceAccess.entryType')} rules={[{ required: true }]}><Select options={(['allow', 'deny', 'revoked'] as AccessListType[]).map((v) => ({ value: v, label: t(`deviceAccess.listType.${v}`) }))} /></Form.Item>
        <Form.Item name="serialNumbers" label={t('deviceAccess.serialNumbers')} rules={[{ required: true, whitespace: true }]}><Input.TextArea rows={5} placeholder={t('deviceAccess.serialNumbersHint')} /></Form.Item>
        <Form.Item name="reason" label={t('deviceAccess.reason')} rules={[{ required: true, whitespace: true }]}><Input.TextArea rows={3} maxLength={500} showCount /></Form.Item>
      </Form>
    </Modal>
    <Modal
      open={Boolean(disablingEntry)}
      title={t('common.disable')}
      okText={t('common.confirm')}
      cancelText={t('common.cancel')}
      confirmLoading={upsert.isPending}
	  okButtonProps={{ danger: true, disabled: !disableReason.trim() }}
      cancelButtonProps={{ disabled: upsert.isPending }}
      closable={!upsert.isPending}
      keyboard={!upsert.isPending}
      mask={{ closable: !upsert.isPending }}
      onOk={() => void disableEntry()}
	  onCancel={() => { setDisablingEntry(undefined); setDisableReason(''); }}
    >
      <p>{t('deviceAccess.disableWarning')}</p>
      <p>{t('deviceAccess.serialNumber')}: {disablingEntry?.identity_value}；{t('deviceAccess.productName')}: {formatProductName(disablingEntry?.product_name, t)}</p>
	  <Input.TextArea value={disableReason} onChange={(event) => setDisableReason(event.target.value)} placeholder={t('deviceAccess.disableReason')} rows={3} maxLength={500} showCount />
    </Modal>
    <Modal
      open={Boolean(batchEntries?.length)}
      title={`${t('deviceAccess.batchDisable')} (${batchEntries?.length ?? 0})`}
      okText={t('common.confirm')}
      cancelText={t('common.cancel')}
      okButtonProps={{ danger: true, disabled: !batchReason.trim() }}
      confirmLoading={disableMany.isPending}
      cancelButtonProps={{ disabled: disableMany.isPending }}
      closable={!disableMany.isPending}
      keyboard={!disableMany.isPending}
      mask={{ closable: !disableMany.isPending }}
      onOk={() => void disableBatch()}
      onCancel={() => { setBatchEntries(undefined); setBatchReason(''); }}
    >
      <Alert showIcon type="warning" title={t('deviceAccess.batchDisableWarning')} style={{ marginBottom: 12 }} />
      <div style={{ maxHeight: 260, overflowY: 'auto' }}>
        {(batchEntries ?? []).map((entry) => <div key={entry.id} style={{ padding: '10px 0', borderBottom: '1px solid var(--ant-color-border-secondary)' }}>
          <strong>{entry.identity_value}</strong>
          <div style={{ color: 'var(--ant-color-text-secondary)' }}>{t('deviceAccess.productName')}: {formatProductName(entry.product_name, t)} · {t(`deviceAccess.listType.${entry.entry_type}`)}</div>
        </div>)}
      </div>
      <Input.TextArea value={batchReason} onChange={(event) => setBatchReason(event.target.value)} placeholder={t('deviceAccess.batchDisableReason')} rows={3} maxLength={500} showCount style={{ marginTop: 12 }} />
    </Modal>
    <Modal
      open={importOpen}
      width={900}
      title={t('deviceAccess.importList')}
      okText={preview ? t('deviceAccess.commitImport') : t('deviceAccess.previewImport')}
      cancelText={t('common.cancel')}
      okButtonProps={{ disabled: preview ? preview.batch.failure_policy === 'strict' && preview.batch.invalid_count > 0 : !importFile }}
      confirmLoading={previewImport.isPending || commitImport.isPending}
      onOk={() => void (preview ? commitSelectedImport() : previewSelectedImport())}
      onCancel={() => { setImportOpen(false); setPreview(undefined); setImportFile(undefined); }}
    >
      {!preview && <Space wrap align="start">
        <Select value={importEntryType} onChange={setImportEntryType} style={{ width: 160 }} options={(['allow', 'deny', 'revoked'] as AccessListType[]).map((value) => ({ value, label: t(`deviceAccess.listType.${value}`) }))} />
        <Select value={importMode} onChange={setImportMode} style={{ width: 160 }} options={(['append', 'replace'] as ImportMode[]).map((value) => ({ value, label: t(`deviceAccess.importMode.${value}`) }))} />
        <Select value={failurePolicy} onChange={setFailurePolicy} style={{ width: 160 }} options={(['strict', 'valid_only'] as ImportFailurePolicy[]).map((value) => ({ value, label: t(`deviceAccess.failurePolicy.${value}`) }))} />
        <Upload accept=".csv,.xlsx,text/csv,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" maxCount={1} beforeUpload={(file) => { setImportFile(file); setPreview(undefined); return false; }} onRemove={() => { setImportFile(undefined); setPreview(undefined); }}>
          <Button icon={<UploadOutlined />}>{t('deviceAccess.selectImportFile')}</Button>
        </Upload>
      </Space>}
      {preview && <>
        {preview.batch.mode === 'replace' && preview.batch.failure_policy === 'valid_only' && <Alert showIcon type="warning" title={t('deviceAccess.replaceValidOnlyWarning')} style={{ marginBottom: 12 }} />}
        <Space wrap size="large" style={{ marginBottom: 16 }}>
          <Statistic title={t('deviceAccess.importTotal')} value={preview.batch.total_count} />
          <Statistic title={t('deviceAccess.importValid')} value={preview.batch.valid_count} />
          <Statistic title={t('deviceAccess.importInvalid')} value={preview.batch.invalid_count} />
          <Statistic title={t('deviceAccess.importChanged')} value={preview.batch.changed_count} />
          <Statistic title={t('deviceAccess.importWillDisable')} value={preview.disable_count} />
        </Space>
        <Table size="small" rowKey="row_number" pagination={{ pageSize: 10 }} dataSource={preview.rows} columns={[
          { title: t('deviceAccess.rowNumber'), dataIndex: 'row_number', width: 90 },
          { title: t('deviceAccess.serialNumber'), render: (_, row) => String(row.normalized_value?.serial_number ?? row.raw_value?.['Serial Number'] ?? '') },
          { title: t('common.status'), dataIndex: 'validation_status', width: 130, render: (value: string) => t(`deviceAccess.importRowStatus.${value}`) },
          { title: t('deviceAccess.errorMessage'), dataIndex: 'error_message' },
        ]} />
      </>}
    </Modal>
    <Modal open={historyOpen} width={980} title={t('deviceAccess.importHistory')} footer={null} onCancel={() => setHistoryOpen(false)}>
      <QueryError error={imports.error} t={t} />
      <Table size="small" rowKey="id" loading={imports.isLoading} pagination={false} dataSource={imports.data?.items ?? []} columns={[
        { title: t('common.createdAt'), dataIndex: 'created_at', width: 180, render: (value: string) => formatSystemTime(value) },
        { title: t('deviceAccess.entryType'), dataIndex: 'entry_type', width: 110, render: (value: string) => t(`deviceAccess.listType.${value}`) },
      { title: t('deviceAccess.importModeLabel'), dataIndex: 'mode', width: 150, render: (value: string, row) => <Space size={4}>{t(`deviceAccess.importMode.${value}`)}{row.reversal_of_batch_id && <Tag color="purple">{t('deviceAccess.reverseBatch')}</Tag>}</Space> },
        { title: t('common.status'), dataIndex: 'status', width: 120, render: (value: string) => t(`deviceAccess.importStatus.${value}`) },
		{ title: t('deviceAccess.importCounts'), render: (_, row) => <Space>{`${row.valid_count}/${row.total_count}`} {row.invalid_count > 0 && <Button type="link" size="small" onClick={() => void showImportErrors(row.id)}>{t('deviceAccess.viewImportErrors')} ({row.invalid_count})</Button>}</Space> },
        { title: t('common.action'), width: 110, render: (_, row) => row.status === 'committed' ? <GuardedButton danger type="link" allowed={allowed} deniedText={t('common.noPermission')} loading={rollbackImport.isPending} onClick={() => rollbackBatch(row.id)}>{t('deviceAccess.rollbackImport')}</GuardedButton> : '—' },
      ]} />
    </Modal>
	<Modal open={Boolean(errorRows) || errorRowsLoading} width={900} title={t('deviceAccess.importErrorDetails')} footer={<Button icon={<DownloadOutlined />} disabled={!errorRows?.length} onClick={downloadImportErrors}>{t('deviceAccess.downloadImportErrors')}</Button>} onCancel={() => { setErrorRows(undefined); setErrorBatchID(''); }}>
	  <Table size="small" rowKey="row_number" loading={errorRowsLoading} pagination={{ pageSize: 10 }} dataSource={errorRows ?? []} columns={[
		{ title: t('deviceAccess.rowNumber'), dataIndex: 'row_number', width: 90 },
		{ title: t('deviceAccess.serialNumber'), render: (_, row) => String(row.normalized_value?.serial_number ?? row.raw_value?.['Serial Number'] ?? '') },
		{ title: t('deviceAccess.errorCode'), dataIndex: 'error_code', width: 210 },
		{ title: t('deviceAccess.errorMessage'), dataIndex: 'error_message' },
	  ]} />
	</Modal>
  </>;
}

function splitSerialNumbers(value: string): string[] {
  return [...new Set(value.split(/[\s,;]+/).map((item) => item.trim()).filter(Boolean))];
}

function csvCell(value: string): string {
	return `"${value.replaceAll('"', '""')}"`;
}

function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url; anchor.download = filename; anchor.click();
  URL.revokeObjectURL(url);
}

export function CandidatePanel({
  operatorCode,
  t,
  allowed,
  businessEnabled,
  initialSerialNumber,
  candidateId,
  reviewStatus,
  onReviewStatusChange,
  onCandidateHandled,
}: CandidatePanelProps) {
  const { message } = App.useApp();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [serialInput, setSerialInput] = useState(initialSerialNumber ?? '');
  const [serialNumber, setSerialNumber] = useState<string | undefined>(initialSerialNumber);
  const [selectedCandidate, setSelectedCandidate] = useState<CandidateItem>();
  const [review, setReview] = useState<{ candidate: CandidateItem; outcome: 'allow' | 'deny' }>();
  const [reason, setReason] = useState('');
  const query = useAccessCandidates({ operatorCode, page, pageSize, serialNumber, reviewStatus: reviewStatus || undefined, candidateId });
  const mutation = useReviewAccessCandidate();
  const submit = async () => {
    if (!review || !reason.trim()) return;
    try { await mutation.mutateAsync({ operatorCode, candidateId: review.candidate.id, outcome: review.outcome, reason: reason.trim() }); void message.success(t(businessEnabled ? 'deviceAccess.reviewSaved' : 'deviceAccess.reviewSavedWhileDisabled')); setReview(undefined); setReason(''); onCandidateHandled(); }
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
    { title: t('common.action'), fixed: 'right', width: 118, align: 'center', render: (_, row) => <div className={styles.iconActions}>
      <IconActionButton label={t('common.detail')} icon={<EyeOutlined />} onClick={() => setSelectedCandidate(row)} />
      {row.review_status === 'pending' && <>
        <IconActionButton label={t('deviceAccess.review.allow')} icon={<CheckOutlined />} allowed={allowed} deniedText={t('common.noPermission')} onClick={() => setReview({ candidate: row, outcome: 'allow' })} />
        <IconActionButton danger label={t('deviceAccess.review.deny')} icon={<CloseOutlined />} allowed={allowed} deniedText={t('common.noPermission')} onClick={() => setReview({ candidate: row, outcome: 'deny' })} />
      </>}
    </div> },
  ];
  const candidateDescriptionItems = (candidate?: CandidateItem) => candidate ? [
    { key: 'serial', label: t('deviceAccess.serialNumber'), children: candidate.serial_number },
    { key: 'oui', label: 'OUI', children: candidate.oui || '—' },
    { key: 'product', label: t('deviceAccess.productClass'), children: candidate.product_class || '—' },
    { key: 'software', label: t('deviceAccess.softwareVersion'), children: candidate.software_version || '—' },
    { key: 'ip', label: t('deviceAccess.remoteIP'), children: candidate.observed_remote_ip || '—' },
    { key: 'inform', label: t('deviceAccess.informCount'), children: candidate.inform_count },
    { key: 'first', label: t('deviceAccess.firstSeenAt'), children: formatSystemTime(candidate.first_seen_at) },
    { key: 'last', label: t('deviceAccess.lastSeenAt'), children: formatSystemTime(candidate.last_seen_at) },
    { key: 'expires', label: t('deviceAccess.expiresAt'), children: formatSystemTime(candidate.expires_at) },
    { key: 'status', label: t('common.status'), children: t(`deviceAccess.reviewStatus.${candidate.review_status}`) },
  ] : [];
  return <>
    <Space wrap className={styles.filterBar}>
      <Input allowClear value={serialInput} onChange={(event) => setSerialInput(event.target.value)} onPressEnter={() => { setSerialNumber(serialInput.trim() || undefined); setPage(1); }} placeholder={t('deviceAccess.serialNumber')} style={{ width: 240 }} />
      <Select value={reviewStatus} onChange={(v) => { onReviewStatusChange(v); setPage(1); }} style={{ width: 180 }} options={['pending', 'approved', 'rejected', 'expired'].map((v) => ({ value: v, label: t(`deviceAccess.reviewStatus.${v}`) }))} />
      <Button type="primary" icon={<SearchOutlined />} onClick={() => { setSerialNumber(serialInput.trim() || undefined); setPage(1); }}>{t('common.search')}</Button>
      <Button icon={<ReloadOutlined />} onClick={() => void query.refetch()}>{t('common.refresh')}</Button>
    </Space>
    {candidateId && !query.isLoading && !query.error && (query.data?.items.length ?? 0) === 0 && <Alert showIcon type="warning" title={t('deviceAccess.candidateUnavailable')} style={{ marginBottom: 12 }} />}
    <QueryError error={query.error} t={t} />
    <Table className={styles.dataTable} rowKey="id" rowClassName={(row) => row.id === candidateId ? styles.deepLinkedRow : ''} columns={columns} dataSource={query.data?.items ?? []} loading={query.isLoading} scroll={{ x: 1100 }} pagination={{ current: page, pageSize, total: query.data?.total ?? 0, showSizeChanger: true, onChange: (p, size) => { setPage(p); setPageSize(size); } }} />
    <Modal open={Boolean(selectedCandidate)} width={760} title={t('deviceAccess.candidateDetail')} footer={null} onCancel={() => setSelectedCandidate(undefined)}>
      <Descriptions bordered size="small" column={2} items={candidateDescriptionItems(selectedCandidate)} />
    </Modal>
    <Modal open={Boolean(review)} width={760} title={review ? t(`deviceAccess.review.${review.outcome}`) : ''} okText={t('common.confirm')} cancelText={t('common.cancel')} okButtonProps={{ danger: review?.outcome === 'deny', disabled: !reason.trim() }} confirmLoading={mutation.isPending} onOk={() => void submit()} onCancel={() => { setReview(undefined); setReason(''); }}>
      <Alert showIcon type={review?.outcome === 'deny' ? 'warning' : 'info'} title={t('deviceAccess.candidateReviewImpact')} style={{ marginBottom: 12 }} />
      <Descriptions bordered size="small" column={2} items={candidateDescriptionItems(review?.candidate)} style={{ marginBottom: 16 }} />
      <Input.TextArea value={reason} onChange={(e) => setReason(e.target.value)} placeholder={t('deviceAccess.reviewReasonPlaceholder')} rows={4} maxLength={500} showCount />
    </Modal>
  </>;
}
