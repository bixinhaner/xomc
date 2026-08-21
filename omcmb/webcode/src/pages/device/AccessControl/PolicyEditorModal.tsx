import { useEffect, useState } from 'react';
import {
  Alert,
  App,
  Button,
  Card,
  Checkbox,
  Col,
  Collapse,
  Divider,
  Dropdown,
  Form,
  Input,
  InputNumber,
  Modal,
  Row,
  Select,
  Space,
  Spin,
  Statistic,
  Switch,
  Table,
  Tag,
  Typography,
  Upload,
} from 'antd';
import {
  DeleteOutlined,
  DownOutlined,
  DownloadOutlined,
  EnvironmentOutlined,
  HistoryOutlined,
  PlusOutlined,
  UploadOutlined,
} from '@ant-design/icons';
import { useClearRuleDimension, useCommitAccessListImport, usePreviewRuleDimensionImport, useRollbackAccessListImport, useRuleDimensionImports } from '@core/hooks/api/useDeviceAccess';
import {
  deviceAccessApi,
  type AccessConditionOperator,
  type AccessConditionType,
  type CompiledAccessPolicy,
  type ImportFailurePolicy,
  type ImportMode,
  type PolicyVersion,
  type RuleDimension,
  type RuleDimensionImportPreview,
  type SerialScopeType,
} from '@core/services/api/deviceAccessApi';
import styles from './PolicyEditorModal.module.css';

interface Props {
  operatorCode: string;
  open: boolean;
  sourceLoading?: boolean;
  submitting?: boolean;
  source?: PolicyVersion;
  readOnly?: boolean;
  fixedPolicyName?: string;
  t: (key: string) => string;
  onCancel: () => void;
  onSubmit: (name: string, policy: CompiledAccessPolicy) => Promise<void>;
  onDrilldownRule?: (policyVersionId: string, matchedRuleId: string) => void;
}

interface DimensionImportState {
  ruleID: string;
  dimension: RuleDimension;
  file?: File;
  mode: ImportMode;
  failurePolicy: ImportFailurePolicy;
  preview?: RuleDimensionImportPreview;
}

interface ConditionFormValue {
  id?: string;
  type: AccessConditionType;
  operator: AccessConditionOperator;
  expected?: string;
  latitude?: number;
  longitude?: number;
  radiusMeters?: number;
  startIP?: string;
  endIP?: string;
  ipRanges?: Array<{ start: string; end: string }>;
  minLatitude?: number;
  maxLatitude?: number;
  minLongitude?: number;
  maxLongitude?: number;
  geoBoundsAny?: Array<{ min_latitude: number; max_latitude: number; min_longitude: number; max_longitude: number; allow_missing?: boolean }>;
  allowMissing?: boolean;
  required?: boolean;
  evidenceTTLSeconds?: number;
}

interface RuleFormValue {
  id?: string;
  name: string;
  enabled?: boolean;
  priority: number;
  scopeType: SerialScopeType;
  scopeValues?: string;
  scopePrefix?: string;
  scopeStart?: string;
  scopeEnd?: string;
  conditions?: ConditionFormValue[];
}

interface PolicyFormValue {
  name: string;
  collectionTimeoutSeconds: number;
  bypassProfiles?: BypassProfileFormValue[];
  rules?: RuleFormValue[];
}

interface BypassProfileFormValue {
  id?: string;
  name: string;
  enabled?: boolean;
  priority: number;
  scopeType?: SerialScopeType;
  scopeValues?: string;
  scopePrefix?: string;
  scopeStart?: string;
  scopeEnd?: string;
  ouis?: string;
  productClasses?: string;
  reason: string;
  validFrom?: string;
  validUntil?: string;
}

const newID = () => globalThis.crypto?.randomUUID?.() ?? `draft-${Date.now()}-${Math.random().toString(16).slice(2)}`;
const splitValues = (value?: string) => value?.split(/[\n,]/).map((item) => item.trim()).filter(Boolean) ?? [];
const defaultCondition = (type: AccessConditionType): ConditionFormValue => ({
  type,
  operator: type === 'gps' ? 'within_radius' : 'equal',
  required: true,
  allowMissing: false,
  evidenceTTLSeconds: 900,
});

function initialValues(source?: PolicyVersion, fixedPolicyName?: string): PolicyFormValue {
  if (!source) return { name: fixedPolicyName ?? '', collectionTimeoutSeconds: 900, bypassProfiles: [], rules: [] };
  return {
    name: source.name,
    collectionTimeoutSeconds: (source.policy.collection_timeout ?? 900_000_000_000) / 1_000_000_000,
    bypassProfiles: (source.policy.bypass_profiles ?? []).map((profile) => ({
      id: profile.id, name: profile.name, enabled: profile.enabled, priority: profile.priority,
      scopeType: profile.serial_scope?.type,
      scopeValues: profile.serial_scope?.values?.join('\n'), scopePrefix: profile.serial_scope?.prefix,
      scopeStart: profile.serial_scope?.start, scopeEnd: profile.serial_scope?.end,
      ouis: profile.ouis?.join('\n'), productClasses: profile.product_classes?.join('\n'),
      reason: profile.reason,
      validFrom: profile.valid_from?.slice(0, 16), validUntil: profile.valid_until?.slice(0, 16),
    })),
    rules: (source.policy.rules ?? []).map((rule) => ({
      id: rule.id,
      name: rule.name,
      enabled: rule.enabled,
      priority: rule.priority,
      scopeType: rule.serial_scope.type,
      scopeValues: rule.serial_scope.values?.join('\n'),
      scopePrefix: rule.serial_scope.prefix,
      scopeStart: rule.serial_scope.start,
      scopeEnd: rule.serial_scope.end,
      conditions: (rule.conditions ?? []).map((condition) => ({
        id: condition.id,
        type: condition.type,
        operator: condition.operator,
        expected: condition.operator === 'in' ? condition.expected_any?.join(', ') : condition.expected,
        latitude: condition.geo_fence?.center.latitude,
        longitude: condition.geo_fence?.center.longitude,
        radiusMeters: condition.geo_fence?.radius_meters,
        startIP: condition.ip_range?.start ?? condition.ip_ranges?.[0]?.start,
        endIP: condition.ip_range?.end ?? condition.ip_ranges?.[0]?.end,
        ipRanges: condition.ip_ranges,
        minLatitude: condition.geo_bounds?.min_latitude ?? condition.geo_bounds_any?.[0]?.min_latitude,
        maxLatitude: condition.geo_bounds?.max_latitude ?? condition.geo_bounds_any?.[0]?.max_latitude,
        minLongitude: condition.geo_bounds?.min_longitude ?? condition.geo_bounds_any?.[0]?.min_longitude,
        maxLongitude: condition.geo_bounds?.max_longitude ?? condition.geo_bounds_any?.[0]?.max_longitude,
        geoBoundsAny: condition.geo_bounds_any,
        allowMissing: condition.geo_fence?.allow_missing ?? condition.geo_bounds?.allow_missing ?? condition.geo_bounds_any?.some((bounds) => bounds.allow_missing),
        required: condition.required,
        evidenceTTLSeconds: condition.evidence_ttl / 1_000_000_000,
      })),
    })),
  };
}

export function PolicyEditorModal({ operatorCode, open, sourceLoading = false, submitting = false, source, readOnly = false, fixedPolicyName, t, onCancel, onSubmit, onDrilldownRule }: Props) {
  const { message, modal } = App.useApp();
  const [form] = Form.useForm<PolicyFormValue>();
  const [dimensionImport, setDimensionImport] = useState<DimensionImportState>();
  const previewDimension = usePreviewRuleDimensionImport();
  const commitImport = useCommitAccessListImport();
  const policyNameLocked = Boolean(source || fixedPolicyName);

  useEffect(() => {
    if (open) form.setFieldsValue(initialValues(source, fixedPolicyName));
  }, [fixedPolicyName, form, open, source]);

  const submit = async () => {
    const values = await form.validateFields();
    const preserveIDs = source?.status === 'draft';
    const policy: CompiledAccessPolicy = {
      // The legacy access-control contract has no generic manual-review queue:
      // no applicable rule and exhausted evidence collection therefore both
      // fail closed. Unknown devices continue through the candidate workflow.
      default_action: 'reject',
      failure_mode: 'fail_closed',
      collection_timeout: values.collectionTimeoutSeconds * 1_000_000_000,
      bypass_profiles: (values.bypassProfiles ?? []).map((profile) => ({
        id: preserveIDs && profile.id ? profile.id : newID(),
        name: profile.name.trim(), enabled: Boolean(profile.enabled), priority: profile.priority,
        serial_scope: profile.scopeType ? {
          type: profile.scopeType,
          values: profile.scopeType === 'list' ? splitValues(profile.scopeValues) : undefined,
          prefix: profile.scopeType === 'prefix' ? profile.scopePrefix?.trim() : undefined,
          start: profile.scopeType === 'range' ? profile.scopeStart?.trim() : undefined,
          end: profile.scopeType === 'range' ? profile.scopeEnd?.trim() : undefined,
        } : undefined,
        ouis: splitValues(profile.ouis).map((value) => value.toUpperCase()),
        product_classes: splitValues(profile.productClasses), reason: profile.reason.trim(),
        valid_from: profile.validFrom ? new Date(profile.validFrom).toISOString() : undefined,
        valid_until: profile.validUntil ? new Date(profile.validUntil).toISOString() : undefined,
      })),
      rules: (values.rules ?? []).map((rule) => ({
        id: preserveIDs && rule.id ? rule.id : newID(),
        name: rule.name.trim(),
        enabled: Boolean(rule.enabled),
        priority: rule.priority,
        serial_scope: {
          type: rule.scopeType,
          values: rule.scopeType === 'list' ? splitValues(rule.scopeValues) : undefined,
          prefix: rule.scopeType === 'prefix' ? rule.scopePrefix?.trim() : undefined,
          start: rule.scopeType === 'range' ? rule.scopeStart?.trim() : undefined,
          end: rule.scopeType === 'range' ? rule.scopeEnd?.trim() : undefined,
        },
        conditions: (rule.conditions ?? []).map((condition) => ({
          id: preserveIDs && condition.id ? condition.id : newID(),
          type: condition.type,
          operator: condition.operator,
          expected: !['in', 'within_radius', 'ip_range', 'within_bounds'].includes(condition.operator) ? condition.expected?.trim() : undefined,
          expected_any: condition.operator === 'in' ? splitValues(condition.expected) : undefined,
          ip_range: condition.operator === 'ip_range' && !condition.ipRanges?.length ? {
            start: condition.startIP!.trim(),
            end: condition.endIP!.trim(),
          } : undefined,
          ip_ranges: condition.operator === 'ip_range' && condition.ipRanges?.length ? condition.ipRanges.map((range, index) => index === 0 ? {
            start: condition.startIP!.trim(), end: condition.endIP!.trim(),
          } : range) : undefined,
          geo_fence: condition.operator === 'within_radius' ? {
            center: { latitude: condition.latitude!, longitude: condition.longitude! },
            radius_meters: condition.radiusMeters!,
            allow_missing: Boolean(condition.allowMissing),
          } : undefined,
          geo_bounds: condition.operator === 'within_bounds' && !condition.geoBoundsAny?.length ? {
            min_latitude: condition.minLatitude!,
            max_latitude: condition.maxLatitude!,
            min_longitude: condition.minLongitude!,
            max_longitude: condition.maxLongitude!,
            allow_missing: Boolean(condition.allowMissing),
          } : undefined,
          geo_bounds_any: condition.operator === 'within_bounds' && condition.geoBoundsAny?.length ? condition.geoBoundsAny.map((bounds, index) => ({
            ...(index === 0 ? {
              min_latitude: condition.minLatitude!, max_latitude: condition.maxLatitude!,
              min_longitude: condition.minLongitude!, max_longitude: condition.maxLongitude!,
            } : bounds),
            allow_missing: Boolean(condition.allowMissing),
          })) : undefined,
          required: Boolean(condition.required),
          evidence_ttl: (condition.evidenceTTLSeconds ?? 0) * 1_000_000_000,
        })),
      })),
    };
    await onSubmit(values.name.trim(), policy);
  };

  const previewSelectedDimension = async () => {
    if (!source || !dimensionImport?.file) return;
    try {
      const preview = await previewDimension.mutateAsync({
        operatorCode, policyVersionId: source.id, ruleId: dimensionImport.ruleID,
        dimension: dimensionImport.dimension, mode: dimensionImport.mode,
        failurePolicy: dimensionImport.failurePolicy, file: dimensionImport.file,
        idempotencyKey: globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${dimensionImport.file.name}`,
      });
      setDimensionImport((current) => current ? { ...current, preview } : current);
    } catch (error) {
      void message.error(error instanceof Error ? error.message : t('common.operationFailed'));
    }
  };

  const performDimensionCommit = async () => {
    if (!dimensionImport?.preview) return;
    try {
      await commitImport.mutateAsync({ operatorCode, batchId: dimensionImport.preview.batch.id });
      void message.success(t('deviceAccess.dimensionImportCommitted'));
      setDimensionImport(undefined);
    } catch (error) {
      void message.error(error instanceof Error ? error.message : t('common.operationFailed'));
    }
  };

  const commitSelectedDimension = () => {
    if (dimensionImport?.mode === 'replace' && dimensionImport.failurePolicy === 'valid_only') {
      modal.confirm({
        title: t('deviceAccess.highRiskReplaceImport'),
        content: t('deviceAccess.replaceValidOnlyWarning'),
        okText: t('common.confirm'), cancelText: t('common.cancel'), okButtonProps: { danger: true },
        onOk: performDimensionCommit,
      });
      return;
    }
    void performDimensionCommit();
  };

  return <><Modal
    className={styles.policyEditorModal}
    open={open}
    width={1120}
    footer={readOnly ? null : undefined}
    title={<Space size={10}>
      <EnvironmentOutlined className={styles.titleIcon} />
      <span>{readOnly ? t('deviceAccess.viewPolicy') : source?.status === 'draft' ? t('deviceAccess.editPolicyDraft') : source ? t('deviceAccess.clonePolicy') : t('deviceAccess.createPolicyDraft')}</span>
      <Tag color="blue">{t('deviceAccess.policyEditorTag')}</Tag>
    </Space>}
    okText={readOnly ? undefined : t('deviceAccess.saveDraft')}
    cancelText={t('common.cancel')}
    confirmLoading={submitting}
    okButtonProps={{ disabled: readOnly || sourceLoading }}
    cancelButtonProps={{ disabled: submitting }}
    closable={!submitting}
    keyboard={!submitting}
    mask={{ closable: !submitting }}
    styles={{ body: { maxHeight: 'calc(100vh - 190px)', overflowY: 'auto', padding: '20px 24px 28px' } }}
    onOk={readOnly ? undefined : () => void submit()}
    onCancel={onCancel}
    destroyOnHidden
  >
    <Spin spinning={sourceLoading}>
      <Form form={form} layout="vertical" initialValues={initialValues(source, fixedPolicyName)} requiredMark="optional" disabled={readOnly}>
        <Alert
          showIcon
          type="info"
          title={t('deviceAccess.policyEditorHint')}
          className={styles.editorHint}
        />

        <Card size="small" className={styles.summaryCard}>
          <Row gutter={[20, 0]} align="top">
            <Col xs={24} lg={8}>
              <Form.Item
                name="name"
                label={t('deviceAccess.policySetName')}
                extra={policyNameLocked ? t('deviceAccess.policySetNameFixed') : t('deviceAccess.policySetNameFirstDraft')}
                rules={[{ required: true, whitespace: true }]}
              ><Input maxLength={128} disabled={policyNameLocked} /></Form.Item>
            </Col>
            <Col xs={12} lg={5}>
              <Form.Item required label={t('deviceAccess.defaultAction')}>
                <Typography.Text strong>{t('deviceAccess.defaultAction.reject')}</Typography.Text>
              </Form.Item>
            </Col>
            <Col xs={12} lg={5}>
              <Form.Item required label={t('deviceAccess.failureMode')}>
                <Typography.Text strong>{t('deviceAccess.failureMode.fail_closed')}</Typography.Text>
              </Form.Item>
            </Col>
            <Col xs={12} lg={6}><Form.Item name="collectionTimeoutSeconds" label={t('deviceAccess.collectionTimeout')} rules={[{ required: true }]}><InputNumber className={styles.fullWidth} min={60} max={86400} precision={0} suffix={t('common.seconds')} /></Form.Item></Col>
          </Row>
        </Card>

        <div className={styles.sectionHeader}><div><Typography.Title level={5}>{t('deviceAccess.bypassProfiles')}</Typography.Title><Typography.Text type="secondary">{t('deviceAccess.bypassProfilesHint')}</Typography.Text></div></div>
        <Form.List name="bypassProfiles">
          {(fields, { add, remove }) => <>
            {fields.map((field, index) => <BypassProfileFields key={field.key} profileName={field.name} index={index} readOnly={readOnly} t={t} onRemove={() => remove(field.name)} />)}
            {!readOnly && <Button block type="dashed" icon={<PlusOutlined />} onClick={() => add({ name: '', enabled: true, priority: (fields.length + 1) * 100, reason: '' })}>{t('deviceAccess.addBypassProfile')}</Button>}
          </>}
        </Form.List>

        <div className={styles.sectionHeader}>
          <div>
            <Typography.Title level={5}>{t('deviceAccess.ruleConfiguration')}</Typography.Title>
            <Typography.Text type="secondary">{t('deviceAccess.ruleConfigurationHint')}</Typography.Text>
          </div>
        </div>

        <Form.List name="rules">
          {(fields, { add, remove }) => <>
            {fields.map((field, ruleIndex) => <Card
              key={field.key}
              className={styles.ruleCard}
              title={<Space><span className={styles.indexBadge}>{ruleIndex + 1}</span>{t('deviceAccess.accessRule')}</Space>}
              extra={readOnly && source && onDrilldownRule && source.policy.rules?.[ruleIndex]?.id
                ? <Button type="link" disabled={false} onClick={() => { onCancel(); onDrilldownRule(source.id, source.policy.rules[ruleIndex].id); }}>{t('deviceAccess.viewMatchingDevices')}</Button>
                : !readOnly && <Button danger type="text" aria-label={t('common.delete')} icon={<DeleteOutlined />} onClick={() => remove(field.name)}>{t('common.delete')}</Button>}
            >
              <Form.Item name={[field.name, 'id']} hidden><Input /></Form.Item>
              <Row gutter={[20, 0]} align="top">
                <Col xs={24} lg={14}>
                  <Form.Item name={[field.name, 'name']} label={t('deviceAccess.ruleName')} rules={[{ required: true, whitespace: true }]}>
                    <Input maxLength={128} />
                  </Form.Item>
                </Col>
                <Col xs={12} lg={5}>
                  <Form.Item name={[field.name, 'priority']} label={t('deviceAccess.priority')} rules={[{ required: true }]}>
                    <InputNumber className={styles.fullWidth} min={1} precision={0} />
                  </Form.Item>
                </Col>
                <Col xs={12} lg={5}>
                  <Form.Item name={[field.name, 'enabled']} label={t('deviceAccess.ruleEnabled')} valuePropName="checked">
                    <Switch checkedChildren={t('common.enabled')} unCheckedChildren={t('common.disabled')} />
                  </Form.Item>
                </Col>
              </Row>
              {!readOnly && source?.status === 'draft' && <Form.Item noStyle shouldUpdate={(previous, current) => previous.rules?.[field.name]?.id !== current.rules?.[field.name]?.id}>
                {({ getFieldValue }) => {
                  const ruleID = getFieldValue(['rules', field.name, 'id']) as string | undefined;
                  return ruleID ? <RuleDimensionActions
                    operatorCode={operatorCode}
                    policyVersionID={source.id}
                    ruleID={ruleID}
                    t={t}
                    onImport={(dimension) => setDimensionImport({ ruleID, dimension, mode: 'append', failurePolicy: 'strict' })}
                  /> : <Alert showIcon type="info" title={t('deviceAccess.saveRuleBeforeDimensionImport')} />;
                }}
              </Form.Item>}
              <ScopeFields ruleName={field.name} t={t} />
              <Divider titlePlacement="start" plain>{t('deviceAccess.conditions')}</Divider>
              <Form.List name={[field.name, 'conditions']}>
                {(conditionFields, conditionOps) => <div className={styles.conditionsStack}>
                  {conditionFields.map((conditionField, conditionIndex) => <ConditionFields
                    key={conditionField.key}
                    index={conditionIndex}
                    ruleName={field.name}
                    conditionName={conditionField.name}
                    defaultExpanded={!source?.policy.rules?.[ruleIndex]?.conditions?.[conditionIndex]?.id}
                    onRemove={readOnly ? undefined : () => conditionOps.remove(conditionField.name)}
                    t={t}
                  />)}
                  {!readOnly && <Dropdown
                    trigger={['click']}
                    menu={{
                      items: (['tac', 'ecgi', 'observed_ip', 'gps'] as AccessConditionType[]).map((type) => ({ key: type, label: t(`deviceAccess.conditionType.${type}`) })),
                      onClick: ({ key }) => conditionOps.add(defaultCondition(key as AccessConditionType)),
                    }}
                  >
                    <Button block type="dashed" icon={<PlusOutlined />}>
                      {t('deviceAccess.addCondition')} <DownOutlined />
                    </Button>
                  </Dropdown>}
                </div>}
              </Form.List>
            </Card>)}
            {!readOnly && <Button
              className={styles.addRuleButton}
              block
              type="dashed"
              icon={<PlusOutlined />}
              onClick={() => add({ name: '', enabled: true, priority: (fields.length + 1) * 100, scopeType: 'all', conditions: [] })}
            >{t('deviceAccess.addRule')}</Button>}
          </>}
        </Form.List>
      </Form>
    </Spin>
  </Modal>
  <Modal
    open={Boolean(dimensionImport)}
    width={900}
    title={dimensionImport ? `${t('deviceAccess.importRuleDimension')} · ${t(`deviceAccess.dimension.${dimensionImport.dimension}`)}` : ''}
    okText={dimensionImport?.preview ? t('deviceAccess.commitImport') : t('deviceAccess.previewImport')}
    cancelText={t('common.cancel')}
    okButtonProps={{
      disabled: dimensionImport?.preview
        ? dimensionImport.preview.batch.failure_policy === 'strict' && dimensionImport.preview.batch.invalid_count > 0
        : !dimensionImport?.file,
    }}
    confirmLoading={previewDimension.isPending || commitImport.isPending}
    onOk={() => dimensionImport?.preview ? commitSelectedDimension() : void previewSelectedDimension()}
    onCancel={() => setDimensionImport(undefined)}
  >
    {dimensionImport && !dimensionImport.preview && <Space wrap align="start">
      <Select value={dimensionImport.mode} onChange={(mode) => setDimensionImport({ ...dimensionImport, mode })} style={{ width: 160 }} options={(['append', 'replace'] as ImportMode[]).map((value) => ({ value, label: t(`deviceAccess.importMode.${value}`) }))} />
      <Select value={dimensionImport.failurePolicy} onChange={(failurePolicy) => setDimensionImport({ ...dimensionImport, failurePolicy })} style={{ width: 230 }} options={(['strict', 'valid_only'] as ImportFailurePolicy[]).map((value) => ({ value, label: t(`deviceAccess.failurePolicy.${value}`) }))} />
      <Upload accept=".csv,.xlsx,text/csv,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" maxCount={1} beforeUpload={(file) => { setDimensionImport({ ...dimensionImport, file }); return false; }} onRemove={() => { setDimensionImport({ ...dimensionImport, file: undefined }); }}>
        <Button icon={<UploadOutlined />}>{t('deviceAccess.selectImportFile')}</Button>
      </Upload>
    </Space>}
    {dimensionImport?.preview && <>
      {dimensionImport.mode === 'replace' && dimensionImport.failurePolicy === 'valid_only' && <Alert showIcon type="warning" title={t('deviceAccess.replaceValidOnlyWarning')} style={{ marginBottom: 12 }} />}
      <Space wrap size="large" style={{ marginBottom: 16 }}>
        <Statistic title={t('deviceAccess.importTotal')} value={dimensionImport.preview.batch.total_count} />
        <Statistic title={t('deviceAccess.importValid')} value={dimensionImport.preview.batch.valid_count} />
        <Statistic title={t('deviceAccess.importInvalid')} value={dimensionImport.preview.batch.invalid_count} />
        <Statistic title={t('deviceAccess.importChanged')} value={dimensionImport.preview.batch.changed_count} />
        <Statistic title={t('deviceAccess.importWillDisable')} value={dimensionImport.preview.disable_count} />
      </Space>
      <Table size="small" rowKey="row_number" pagination={{ pageSize: 10 }} dataSource={dimensionImport.preview.rows} columns={[
        { title: t('deviceAccess.rowNumber'), dataIndex: 'row_number', width: 80 },
        { title: t('deviceAccess.importValue'), render: (_, row) => JSON.stringify(row.normalized_value ?? row.raw_value ?? {}) },
        { title: t('common.status'), dataIndex: 'validation_status', width: 130, render: (value: string) => t(`deviceAccess.importRowStatus.${value}`) },
        { title: t('deviceAccess.errorMessage'), dataIndex: 'error_message' },
      ]} />
    </>}
  </Modal></>;
}

function BypassProfileFields({ profileName, index, readOnly, t, onRemove }: { profileName: number; index: number; readOnly: boolean; t: (key: string) => string; onRemove: () => void }) {
  return <Card className={styles.ruleCard} title={<Space><span className={styles.indexBadge}>{index + 1}</span>{t('deviceAccess.bypassProfile')}</Space>} extra={!readOnly && <Button danger type="text" icon={<DeleteOutlined />} onClick={onRemove}>{t('common.delete')}</Button>}>
    <Form.Item name={[profileName, 'id']} hidden><Input /></Form.Item>
    <Row gutter={[16, 0]}>
      <Col xs={24} lg={10}><Form.Item name={[profileName, 'name']} label={t('common.name')} rules={[{ required: true, whitespace: true }]}><Input maxLength={128} /></Form.Item></Col>
      <Col xs={12} lg={5}><Form.Item name={[profileName, 'priority']} label={t('deviceAccess.priority')} rules={[{ required: true }]}><InputNumber className={styles.fullWidth} min={1} precision={0} /></Form.Item></Col>
      <Col xs={12} lg={5}><Form.Item name={[profileName, 'enabled']} label={t('common.status')} valuePropName="checked"><Switch checkedChildren={t('common.enabled')} unCheckedChildren={t('common.disabled')} /></Form.Item></Col>
      <Col xs={24} lg={4}><Form.Item name={[profileName, 'scopeType']} label={t('deviceAccess.serialScope')}><Select allowClear options={(['list', 'prefix', 'range'] as SerialScopeType[]).map((value) => ({ value, label: t(`deviceAccess.serialScope.${value}`) }))} /></Form.Item></Col>
      <Col xs={24} lg={8}><Form.Item name={[profileName, 'ouis']} label={t('deviceAccess.bypassOUIs')}><Input.TextArea autoSize={{ minRows: 2, maxRows: 4 }} /></Form.Item></Col>
      <Col xs={24} lg={8}><Form.Item name={[profileName, 'productClasses']} label={t('deviceAccess.bypassProductClasses')}><Input.TextArea autoSize={{ minRows: 2, maxRows: 4 }} /></Form.Item></Col>
      <Col xs={24} lg={8}><Form.Item name={[profileName, 'scopeValues']} label={t('deviceAccess.bypassSerialValues')}><Input.TextArea autoSize={{ minRows: 2, maxRows: 4 }} /></Form.Item></Col>
      <Col xs={12} lg={6}><Form.Item name={[profileName, 'scopePrefix']} label={t('deviceAccess.serialPrefix')}><Input /></Form.Item></Col>
      <Col xs={12} lg={6}><Form.Item name={[profileName, 'scopeStart']} label={t('deviceAccess.rangeStart')}><Input /></Form.Item></Col>
      <Col xs={12} lg={6}><Form.Item name={[profileName, 'scopeEnd']} label={t('deviceAccess.rangeEnd')}><Input /></Form.Item></Col>
      <Col xs={12} lg={6}><Form.Item name={[profileName, 'validFrom']} label={t('deviceAccess.validFrom')}><Input type="datetime-local" /></Form.Item></Col>
      <Col xs={12} lg={6}><Form.Item name={[profileName, 'validUntil']} label={t('deviceAccess.validUntil')} rules={[{ required: true }]}><Input type="datetime-local" /></Form.Item></Col>
      <Col xs={24} lg={18}><Form.Item name={[profileName, 'reason']} label={t('deviceAccess.operationReason')} rules={[{ required: true, whitespace: true }]}><Input maxLength={512} /></Form.Item></Col>
    </Row>
    <Alert showIcon type="warning" title={t('deviceAccess.bypassProfileWarning')} />
  </Card>;
}

function RuleDimensionActions({ operatorCode, policyVersionID, ruleID, t, onImport }: {
  operatorCode: string;
  policyVersionID: string;
  ruleID: string;
  t: (key: string) => string;
  onImport: (dimension: RuleDimension) => void;
}) {
  const { message, modal } = App.useApp();
  const [dimension, setDimension] = useState<RuleDimension>('sn');
  const [historyOpen, setHistoryOpen] = useState(false);
  const clearDimension = useClearRuleDimension();
  const rollbackImport = useRollbackAccessListImport();
  const imports = useRuleDimensionImports({ operatorCode, policyVersionId: policyVersionID, ruleId: ruleID, dimension, page: 1, pageSize: 50, enabled: historyOpen });
  const downloadTemplate = async () => {
    try {
      downloadBlob(await deviceAccessApi.downloadRuleDimensionTemplate(operatorCode, dimension), `device-access-${dimension}-template.xlsx`);
    } catch (error) {
      void message.error(error instanceof Error ? error.message : t('common.operationFailed'));
    }
  };
  const exportDimension = async () => {
    try {
      downloadBlob(await deviceAccessApi.exportRuleDimension({ operatorCode, policyVersionId: policyVersionID, ruleId: ruleID, dimension }), `device-access-${dimension}.csv`);
    } catch (error) {
      void message.error(error instanceof Error ? error.message : t('common.operationFailed'));
    }
  };
  const confirmClear = () => modal.confirm({
    title: `${t('deviceAccess.clearRuleDimension')} · ${t(`deviceAccess.dimension.${dimension}`)}`,
    content: t('deviceAccess.clearRuleDimensionWarning'),
    okText: t('common.confirm'), cancelText: t('common.cancel'), okButtonProps: { danger: true },
    onOk: async () => {
      try {
        await clearDimension.mutateAsync({ operatorCode, policyVersionId: policyVersionID, ruleId: ruleID, dimension });
        void message.success(t('deviceAccess.ruleDimensionCleared'));
      } catch (error) {
        void message.error(error instanceof Error ? error.message : t('common.operationFailed'));
        throw error;
      }
    },
  });
  const confirmRollback = (batchID: string) => modal.confirm({
    title: t('deviceAccess.rollbackImport'), content: t('deviceAccess.rollbackImportWarning'),
    okText: t('common.confirm'), cancelText: t('common.cancel'), okButtonProps: { danger: true },
    onOk: async () => {
      try {
        await rollbackImport.mutateAsync({ operatorCode, batchId: batchID });
        void message.success(t('deviceAccess.importRolledBack'));
      } catch (error) {
        void message.error(error instanceof Error ? error.message : t('common.operationFailed'));
        throw error;
      }
    },
  });
  return <><Alert
    showIcon
    type="info"
    title={t('deviceAccess.ruleDimensionGovernance')}
    style={{ marginBottom: 16 }}
    action={<Space wrap>
      <Select value={dimension} onChange={setDimension} style={{ width: 120 }} options={(['sn', 'tac', 'ecgi', 'ip', 'gps'] as RuleDimension[]).map((value) => ({ value, label: t(`deviceAccess.dimension.${value}`) }))} />
      <Button size="small" icon={<DownloadOutlined />} onClick={() => void downloadTemplate()}>{t('deviceAccess.downloadTemplate')}</Button>
      <Button size="small" icon={<UploadOutlined />} onClick={() => onImport(dimension)}>{t('deviceAccess.importList')}</Button>
      <Button size="small" icon={<DownloadOutlined />} onClick={() => void exportDimension()}>{t('common.export')}</Button>
      <Button size="small" icon={<HistoryOutlined />} onClick={() => setHistoryOpen(true)}>{t('deviceAccess.importHistory')}</Button>
      <Button size="small" danger icon={<DeleteOutlined />} loading={clearDimension.isPending} onClick={confirmClear}>{t('deviceAccess.clearRuleDimension')}</Button>
    </Space>}
  />
  <Modal open={historyOpen} width={900} title={`${t('deviceAccess.importHistory')} · ${t(`deviceAccess.dimension.${dimension}`)}`} footer={null} onCancel={() => setHistoryOpen(false)}>
    <Table size="small" rowKey="id" loading={imports.isLoading} pagination={false} dataSource={imports.data?.items ?? []} columns={[
      { title: t('common.createdAt'), dataIndex: 'created_at', width: 190 },
      { title: t('deviceAccess.importModeLabel'), dataIndex: 'mode', width: 100, render: (value: string) => t(`deviceAccess.importMode.${value}`) },
      { title: t('common.status'), dataIndex: 'status', width: 120, render: (value: string) => t(`deviceAccess.importStatus.${value}`) },
      { title: t('deviceAccess.importCounts'), render: (_, row) => `${row.valid_count}/${row.total_count} · ${row.invalid_count}` },
      { title: t('common.action'), width: 100, render: (_, row) => row.status === 'committed' ? <Button danger type="link" loading={rollbackImport.isPending} onClick={() => confirmRollback(row.id)}>{t('deviceAccess.rollbackImport')}</Button> : '—' },
    ]} />
  </Modal></>;
}

function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}

function ScopeFields({ ruleName, t }: { ruleName: number; t: (key: string) => string }) {
  return <Row gutter={[20, 0]} align="top">
    <Col xs={24} md={8}>
      <Form.Item name={[ruleName, 'scopeType']} label={t('deviceAccess.serialScope')} rules={[{ required: true }]}>
        <Select options={(['all', 'list', 'prefix', 'range'] as SerialScopeType[]).map((value) => ({ value, label: t(`deviceAccess.serialScope.${value}`) }))} />
      </Form.Item>
    </Col>
    <Form.Item noStyle shouldUpdate={(previous, current) => previous.rules?.[ruleName]?.scopeType !== current.rules?.[ruleName]?.scopeType}>
      {({ getFieldValue }) => {
        const scopeType = getFieldValue(['rules', ruleName, 'scopeType']) as SerialScopeType | undefined;
        if (scopeType === 'list') return <Col xs={24} md={16}><Form.Item name={[ruleName, 'scopeValues']} label={t('deviceAccess.serialList')} rules={[{ required: true, whitespace: true }]}><Input.TextArea autoSize={{ minRows: 2, maxRows: 4 }} /></Form.Item></Col>;
        if (scopeType === 'prefix') return <Col xs={24} md={16}><Form.Item name={[ruleName, 'scopePrefix']} label={t('deviceAccess.serialPrefix')} rules={[{ required: true, whitespace: true }]}><Input /></Form.Item></Col>;
        if (scopeType === 'range') return <>
          <Col xs={24} md={8}><Form.Item name={[ruleName, 'scopeStart']} label={t('deviceAccess.rangeStart')} rules={[{ required: true, whitespace: true }]}><Input /></Form.Item></Col>
          <Col xs={24} md={8}><Form.Item name={[ruleName, 'scopeEnd']} label={t('deviceAccess.rangeEnd')} rules={[{ required: true, whitespace: true }]}><Input /></Form.Item></Col>
        </>;
        return <Col xs={24} md={16} className={styles.scopeHintColumn}>
          <Alert className={styles.scopeHint} type="success" showIcon title={t('deviceAccess.allSerialsHint')} />
        </Col>;
      }}
    </Form.Item>
  </Row>;
}

function conditionSummaryValue(condition?: ConditionFormValue): string {
  if (!condition) return '—';
  if (condition.operator === 'ip_range') return `${condition.startIP || '—'} – ${condition.endIP || '—'}`;
  if (condition.operator === 'within_radius') return `${condition.latitude ?? '—'}, ${condition.longitude ?? '—'} · ${condition.radiusMeters ?? '—'} m`;
  if (condition.operator === 'within_bounds') return `${condition.minLongitude ?? '—'}, ${condition.minLatitude ?? '—'} – ${condition.maxLongitude ?? '—'}, ${condition.maxLatitude ?? '—'}`;
  return condition.expected?.trim() || '—';
}

function ConditionSummary({ index, ruleName, conditionName, t }: { index: number; ruleName: number; conditionName: number; t: (key: string) => string }) {
  const form = Form.useFormInstance<PolicyFormValue>();
  const condition = Form.useWatch(['rules', ruleName, 'conditions', conditionName], form) as ConditionFormValue | undefined;
  return <div className={styles.conditionSummary}>
    <span className={styles.conditionIndex}>{index + 1}</span>
    <Typography.Text strong>{condition?.type ? t(`deviceAccess.conditionType.${condition.type}`) : t('deviceAccess.accessCondition')}</Typography.Text>
    {condition?.operator && <Tag color="blue">{t(`deviceAccess.operator.${condition.operator}`)}</Tag>}
    <Typography.Text className={styles.conditionSummaryValue} type="secondary" ellipsis={{ tooltip: conditionSummaryValue(condition) }}>
      {conditionSummaryValue(condition)}
    </Typography.Text>
  </div>;
}

function ConditionFields({ index, ruleName, conditionName, defaultExpanded, onRemove, t }: { index: number; ruleName: number; conditionName: number; defaultExpanded: boolean; onRemove?: () => void; t: (key: string) => string }) {
  const form = Form.useFormInstance<PolicyFormValue>();
  const operators: Record<AccessConditionType, AccessConditionOperator[]> = {
    tac: ['equal', 'in'],
    ecgi: ['equal', 'in'],
    observed_ip: ['equal', 'in', 'cidr', 'ip_range'],
    gps: ['within_radius', 'within_bounds'],
  };
  return <Collapse
    className={styles.conditionCollapse}
    defaultActiveKey={defaultExpanded ? ['condition'] : []}
    items={[{
      key: 'condition',
      forceRender: true,
      label: <ConditionSummary index={index} ruleName={ruleName} conditionName={conditionName} t={t} />,
      extra: onRemove && <Button danger type="text" aria-label={t('common.delete')} icon={<DeleteOutlined />} onClick={(event) => { event.stopPropagation(); onRemove(); }} />,
      children: <>
    <Row gutter={[16, 0]} align="top">
      <Col xs={24} md={8} lg={6}>
        <Form.Item name={[conditionName, 'type']} label={t('deviceAccess.conditionType')} rules={[{ required: true }]}>
          <Select
            options={(['tac', 'ecgi', 'observed_ip', 'gps'] as AccessConditionType[]).map((value) => ({ value, label: t(`deviceAccess.conditionType.${value}`) }))}
            onChange={(value: AccessConditionType) => {
              form.setFieldValue(['rules', ruleName, 'conditions', conditionName, 'operator'], operators[value][0]);
              form.setFieldValue(['rules', ruleName, 'conditions', conditionName, 'allowMissing'], false);
            }}
          />
        </Form.Item>
      </Col>
      <Form.Item noStyle shouldUpdate={(previous, current) => {
        const previousCondition = previous.rules?.[ruleName]?.conditions?.[conditionName];
        const currentCondition = current.rules?.[ruleName]?.conditions?.[conditionName];
        return previousCondition?.type !== currentCondition?.type || previousCondition?.operator !== currentCondition?.operator;
      }}>
        {({ getFieldValue }) => {
          const condition = getFieldValue(['rules', ruleName, 'conditions', conditionName]) as ConditionFormValue | undefined;
          const type = condition?.type;
          const operator = condition?.operator;
          return <>
            <Col xs={24} md={8} lg={6}>
              <Form.Item name={[conditionName, 'operator']} label={t('deviceAccess.conditionOperator')} rules={[{ required: true }]}>
                <Select options={(type ? operators[type] : []).map((value) => ({ value, label: t(`deviceAccess.operator.${value}`) }))} />
              </Form.Item>
            </Col>
            {!['within_radius', 'ip_range', 'within_bounds'].includes(operator ?? '') && <Col xs={24} md={8} lg={12}>
              <Form.Item name={[conditionName, 'expected']} label={operator === 'in' ? t('deviceAccess.expectedValues') : t('deviceAccess.expected')} rules={[{ required: true, whitespace: true }]}>
                <Input />
              </Form.Item>
            </Col>}
            {operator === 'ip_range' && <>
              <Col xs={24} md={8} lg={6}><Form.Item name={[conditionName, 'startIP']} label={t('deviceAccess.startIP')} rules={[{ required: true, whitespace: true }]}><Input /></Form.Item></Col>
              <Col xs={24} md={8} lg={6}><Form.Item name={[conditionName, 'endIP']} label={t('deviceAccess.endIP')} rules={[{ required: true, whitespace: true }]}><Input /></Form.Item></Col>
            </>}
            {operator === 'within_radius' && <>
              <Col xs={24} md={8} lg={4}><Form.Item name={[conditionName, 'latitude']} label={t('deviceAccess.latitude')} rules={[{ required: true }]}><InputNumber className={styles.fullWidth} min={-90} max={90} precision={6} /></Form.Item></Col>
              <Col xs={24} md={8} lg={4}><Form.Item name={[conditionName, 'longitude']} label={t('deviceAccess.longitude')} rules={[{ required: true }]}><InputNumber className={styles.fullWidth} min={-180} max={180} precision={6} /></Form.Item></Col>
              <Col xs={24} md={8} lg={4}><Form.Item name={[conditionName, 'radiusMeters']} label={t('deviceAccess.radiusMeters')} rules={[{ required: true }]}><InputNumber className={styles.fullWidth} min={1} precision={0} /></Form.Item></Col>
            </>}
            {operator === 'within_bounds' && <>
              <Col xs={24} md={6} lg={3}><Form.Item name={[conditionName, 'minLongitude']} label={t('deviceAccess.minLongitude')} rules={[{ required: true }]}><InputNumber className={styles.fullWidth} min={-180} max={180} precision={6} /></Form.Item></Col>
              <Col xs={24} md={6} lg={3}><Form.Item name={[conditionName, 'maxLongitude']} label={t('deviceAccess.maxLongitude')} rules={[{ required: true }]}><InputNumber className={styles.fullWidth} min={-180} max={180} precision={6} /></Form.Item></Col>
              <Col xs={24} md={6} lg={3}><Form.Item name={[conditionName, 'minLatitude']} label={t('deviceAccess.minLatitude')} rules={[{ required: true }]}><InputNumber className={styles.fullWidth} min={-90} max={90} precision={6} /></Form.Item></Col>
              <Col xs={24} md={6} lg={3}><Form.Item name={[conditionName, 'maxLatitude']} label={t('deviceAccess.maxLatitude')} rules={[{ required: true }]}><InputNumber className={styles.fullWidth} min={-90} max={90} precision={6} /></Form.Item></Col>
            </>}
          </>;
        }}
      </Form.Item>
    </Row>
    <Collapse
      ghost
      className={styles.advancedSettings}
      items={[{
        key: 'advanced',
        forceRender: true,
        label: t('deviceAccess.conditionAdvancedSettings'),
        children: <div className={styles.conditionOptions}>
          <Row gutter={[20, 0]} align="middle">
            <Col xs={24} md={8}>
              <Form.Item name={[conditionName, 'evidenceTTLSeconds']} label={t('deviceAccess.evidenceTTL')}>
                <InputNumber className={styles.fullWidth} min={0} precision={0} />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item name={[conditionName, 'required']} valuePropName="checked" label={t('deviceAccess.missingEvidenceHandling')}>
                <Checkbox>{t('deviceAccess.requiredEvidence')}</Checkbox>
              </Form.Item>
            </Col>
            <Form.Item noStyle shouldUpdate={(previous, current) => previous.rules?.[ruleName]?.conditions?.[conditionName]?.type !== current.rules?.[ruleName]?.conditions?.[conditionName]?.type}>
              {({ getFieldValue }) => getFieldValue(['rules', ruleName, 'conditions', conditionName, 'type']) === 'gps' ? <Col xs={24} md={8}>
                <Form.Item name={[conditionName, 'allowMissing']} valuePropName="checked" label={t('deviceAccess.gpsMissingPolicy')}>
                  <Checkbox>{t('deviceAccess.gpsAllowMissing')}</Checkbox>
                </Form.Item>
              </Col> : null}
            </Form.Item>
          </Row>
        </div>,
      }]}
    />
      </>,
    }]}
  />;
}
