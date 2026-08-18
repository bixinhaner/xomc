import { useEffect } from 'react';
import {
  Alert,
  App,
  Button,
  Card,
  Checkbox,
  Col,
  Divider,
  Form,
  Input,
  InputNumber,
  Modal,
  Row,
  Select,
  Space,
  Spin,
  Switch,
  Tag,
  Typography,
  Upload,
} from 'antd';
import {
  DeleteOutlined,
  DownloadOutlined,
  EnvironmentOutlined,
  PlusOutlined,
  UploadOutlined,
} from '@ant-design/icons';
import type {
  AccessConditionOperator,
  AccessConditionType,
  CompiledAccessPolicy,
  PolicyVersion,
  SerialScopeType,
} from '@core/services/api/deviceAccessApi';
import styles from './PolicyEditorModal.module.css';

interface Props {
  open: boolean;
  sourceLoading?: boolean;
  submitting?: boolean;
  source?: PolicyVersion;
  fixedPolicyName?: string;
  t: (key: string) => string;
  onCancel: () => void;
  onSubmit: (name: string, policy: CompiledAccessPolicy) => Promise<void>;
}

interface ConditionFormValue {
  type: AccessConditionType;
  operator: AccessConditionOperator;
  expected?: string;
  latitude?: number;
  longitude?: number;
  radiusMeters?: number;
  allowMissing?: boolean;
  required?: boolean;
  evidenceTTLSeconds?: number;
}

interface RuleFormValue {
  name: string;
  enabled?: boolean;
  scopeType: SerialScopeType;
  scopeValues?: string;
  scopePrefix?: string;
  scopeStart?: string;
  scopeEnd?: string;
  conditions?: ConditionFormValue[];
}

interface PolicyFormValue {
  name: string;
  rules?: RuleFormValue[];
}

const GPS_CSV_HEADERS = [
  'rule_name',
  'serial_number',
  'latitude',
  'longitude',
  'radius_meters',
  'allow_missing',
  'enabled',
] as const;

const newID = () => globalThis.crypto?.randomUUID?.() ?? `draft-${Date.now()}-${Math.random().toString(16).slice(2)}`;
const splitValues = (value?: string) => value?.split(/[\n,]/).map((item) => item.trim()).filter(Boolean) ?? [];

function initialValues(source?: PolicyVersion, fixedPolicyName?: string): PolicyFormValue {
  if (!source) return { name: fixedPolicyName ?? '', rules: [] };
  return {
    name: source.name,
    rules: (source.policy.rules ?? []).map((rule) => ({
      name: rule.name,
      enabled: rule.enabled,
      scopeType: rule.serial_scope.type,
      scopeValues: rule.serial_scope.values?.join('\n'),
      scopePrefix: rule.serial_scope.prefix,
      scopeStart: rule.serial_scope.start,
      scopeEnd: rule.serial_scope.end,
      conditions: (rule.conditions ?? []).map((condition) => ({
        type: condition.type,
        operator: condition.operator,
        expected: condition.operator === 'in' ? condition.expected_any?.join(', ') : condition.expected,
        latitude: condition.geo_fence?.center.latitude,
        longitude: condition.geo_fence?.center.longitude,
        radiusMeters: condition.geo_fence?.radius_meters,
        allowMissing: condition.geo_fence?.allow_missing,
        required: condition.required,
        evidenceTTLSeconds: condition.evidence_ttl / 1_000_000_000,
      })),
    })),
  };
}

function parseCSV(text: string): string[][] {
  const rows: string[][] = [];
  let row: string[] = [];
  let cell = '';
  let quoted = false;
  for (let index = 0; index < text.length; index += 1) {
    const character = text[index];
    if (character === '"') {
      if (quoted && text[index + 1] === '"') {
        cell += '"';
        index += 1;
      } else {
        quoted = !quoted;
      }
    } else if (character === ',' && !quoted) {
      row.push(cell.trim());
      cell = '';
    } else if ((character === '\n' || character === '\r') && !quoted) {
      if (character === '\r' && text[index + 1] === '\n') index += 1;
      row.push(cell.trim());
      if (row.some(Boolean)) rows.push(row);
      row = [];
      cell = '';
    } else {
      cell += character;
    }
  }
  row.push(cell.trim());
  if (row.some(Boolean)) rows.push(row);
  return rows;
}

function csvBoolean(value: string, defaultValue: boolean): boolean {
  if (!value) return defaultValue;
  const normalized = value.trim().toLowerCase();
  if (['true', '1', 'yes', 'y'].includes(normalized)) return true;
  if (['false', '0', 'no', 'n'].includes(normalized)) return false;
  throw new Error(`invalid boolean: ${value}`);
}

function gpsRulesFromCSV(text: string): RuleFormValue[] {
  const rows = parseCSV(text.replace(/^\uFEFF/, ''));
  if (rows.length < 2) throw new Error('CSV contains no data rows');
  const headers = rows[0].map((header) => header.toLowerCase());
  const positions = new Map(headers.map((header, index) => [header, index]));
  for (const header of GPS_CSV_HEADERS) {
    if (!positions.has(header)) throw new Error(`missing column: ${header}`);
  }
  return rows.slice(1).map((row, offset) => {
    const value = (header: typeof GPS_CSV_HEADERS[number]) => row[positions.get(header)!]?.trim() ?? '';
    const latitude = Number(value('latitude'));
    const longitude = Number(value('longitude'));
    const radiusMeters = Number(value('radius_meters'));
    if (!value('rule_name')) throw new Error(`line ${offset + 2}: rule_name is required`);
    if (!Number.isFinite(latitude) || latitude < -90 || latitude > 90) throw new Error(`line ${offset + 2}: invalid latitude`);
    if (!Number.isFinite(longitude) || longitude < -180 || longitude > 180) throw new Error(`line ${offset + 2}: invalid longitude`);
    if (!Number.isFinite(radiusMeters) || radiusMeters <= 0) throw new Error(`line ${offset + 2}: invalid radius_meters`);
    const serialNumber = value('serial_number');
    return {
      name: value('rule_name'),
      enabled: csvBoolean(value('enabled'), true),
      scopeType: serialNumber ? 'list' : 'all',
      scopeValues: serialNumber,
      conditions: [{
        type: 'gps',
        operator: 'within_radius',
        latitude,
        longitude,
        radiusMeters,
        allowMissing: csvBoolean(value('allow_missing'), false),
        required: true,
        evidenceTTLSeconds: 900,
      }],
    };
  });
}

function csvCell(value: string | number | boolean | undefined): string {
  const text = String(value ?? '');
  return /[",\n\r]/.test(text) ? `"${text.replace(/"/g, '""')}"` : text;
}

function gpsCSVFromRules(rules: RuleFormValue[]): string {
  const rows: Array<Array<string | number | boolean | undefined>> = [GPS_CSV_HEADERS.slice()];
  for (const rule of rules) {
    const serials = rule.scopeType === 'list' ? splitValues(rule.scopeValues) : [''];
    for (const condition of rule.conditions ?? []) {
      if (condition.type !== 'gps') continue;
      for (const serial of serials.length > 0 ? serials : ['']) {
        rows.push([
          rule.name,
          serial,
          condition.latitude,
          condition.longitude,
          condition.radiusMeters,
          Boolean(condition.allowMissing),
          Boolean(rule.enabled),
        ]);
      }
    }
  }
  return `\uFEFF${rows.map((row) => row.map(csvCell).join(',')).join('\r\n')}\r\n`;
}

export function PolicyEditorModal({ open, sourceLoading = false, submitting = false, source, fixedPolicyName, t, onCancel, onSubmit }: Props) {
  const { message } = App.useApp();
  const [form] = Form.useForm<PolicyFormValue>();
  const policyNameLocked = Boolean(source || fixedPolicyName);

  useEffect(() => {
    if (open) form.setFieldsValue(initialValues(source, fixedPolicyName));
  }, [fixedPolicyName, form, open, source]);

  const submit = async () => {
    const values = await form.validateFields();
    const policy: CompiledAccessPolicy = {
      default_action: 'reject',
      rules: (values.rules ?? []).map((rule) => ({
        id: newID(),
        name: rule.name.trim(),
        enabled: Boolean(rule.enabled),
        serial_scope: {
          type: rule.scopeType,
          values: rule.scopeType === 'list' ? splitValues(rule.scopeValues) : undefined,
          prefix: rule.scopeType === 'prefix' ? rule.scopePrefix?.trim() : undefined,
          start: rule.scopeType === 'range' ? rule.scopeStart?.trim() : undefined,
          end: rule.scopeType === 'range' ? rule.scopeEnd?.trim() : undefined,
        },
        conditions: (rule.conditions ?? []).map((condition) => ({
          id: newID(),
          type: condition.type,
          operator: condition.operator,
          expected: condition.operator !== 'in' && condition.operator !== 'within_radius' ? condition.expected?.trim() : undefined,
          expected_any: condition.operator === 'in' ? splitValues(condition.expected) : undefined,
          geo_fence: condition.operator === 'within_radius' ? {
            center: { latitude: condition.latitude!, longitude: condition.longitude! },
            radius_meters: condition.radiusMeters!,
            allow_missing: Boolean(condition.allowMissing),
          } : undefined,
          required: Boolean(condition.required),
          evidence_ttl: (condition.evidenceTTLSeconds ?? 0) * 1_000_000_000,
        })),
      })),
    };
    await onSubmit(values.name.trim(), policy);
  };

  const importGPS = async (file: File) => {
    try {
      const imported = gpsRulesFromCSV(await file.text());
      const current = form.getFieldValue('rules') ?? [];
      form.setFieldValue('rules', [...current, ...imported]);
      void message.success(`${imported.length} ${t('deviceAccess.gpsRowsImported')}`);
    } catch (error) {
      void message.error(`${t('deviceAccess.gpsImportFailed')}: ${error instanceof Error ? error.message : String(error)}`);
    }
  };

  const exportGPS = () => {
    const csv = gpsCSVFromRules(form.getFieldValue('rules') ?? []);
    const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8' }));
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = 'device-access-gps.csv';
    anchor.click();
    URL.revokeObjectURL(url);
  };

  return <Modal
    className={styles.policyEditorModal}
    open={open}
    width={1120}
    title={<Space size={10}>
      <EnvironmentOutlined className={styles.titleIcon} />
      <span>{source ? t('deviceAccess.clonePolicy') : t('deviceAccess.createPolicyDraft')}</span>
      <Tag color="blue">{t('deviceAccess.policyEditorTag')}</Tag>
    </Space>}
    okText={t('deviceAccess.saveDraft')}
    cancelText={t('common.cancel')}
    confirmLoading={submitting}
    okButtonProps={{ disabled: sourceLoading }}
    cancelButtonProps={{ disabled: submitting }}
    closable={!submitting}
    keyboard={!submitting}
    mask={{ closable: !submitting }}
    styles={{ body: { maxHeight: 'calc(100vh - 190px)', overflowY: 'auto', padding: '20px 24px 28px' } }}
    onOk={() => void submit()}
    onCancel={onCancel}
    destroyOnHidden
  >
    <Spin spinning={sourceLoading}>
      <Form form={form} layout="vertical" initialValues={initialValues(source, fixedPolicyName)} requiredMark="optional">
        <Alert
          showIcon
          type="info"
          title={t('deviceAccess.policyEditorHint')}
          className={styles.editorHint}
          action={<Space wrap>
            <Upload
              accept=".csv,text/csv"
              showUploadList={false}
              beforeUpload={(file) => { void importGPS(file); return false; }}
            >
              <Button icon={<UploadOutlined />}>{t('deviceAccess.importGPS')}</Button>
            </Upload>
            <Button icon={<DownloadOutlined />} onClick={exportGPS}>{t('deviceAccess.exportGPS')}</Button>
          </Space>}
        />

        <Card size="small" className={styles.summaryCard}>
          <Row gutter={[20, 0]} align="top">
            <Col span={24}>
              <Form.Item
                name="name"
                label={t('deviceAccess.policySetName')}
                extra={policyNameLocked ? t('deviceAccess.policySetNameFixed') : t('deviceAccess.policySetNameFirstDraft')}
                rules={[{ required: true, whitespace: true }]}
              ><Input maxLength={128} disabled={policyNameLocked} /></Form.Item>
            </Col>
          </Row>
        </Card>

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
              extra={<Button danger type="text" aria-label={t('common.delete')} icon={<DeleteOutlined />} onClick={() => remove(field.name)}>{t('common.delete')}</Button>}
            >
              <Row gutter={[20, 0]} align="top">
                <Col xs={24} lg={18}>
                  <Form.Item name={[field.name, 'name']} label={t('deviceAccess.ruleName')} rules={[{ required: true, whitespace: true }]}>
                    <Input maxLength={128} />
                  </Form.Item>
                </Col>
                <Col xs={24} lg={6}>
                  <Form.Item name={[field.name, 'enabled']} label={t('deviceAccess.ruleEnabled')} valuePropName="checked">
                    <Switch checkedChildren={t('common.enabled')} unCheckedChildren={t('common.disabled')} />
                  </Form.Item>
                </Col>
              </Row>
              <ScopeFields ruleName={field.name} t={t} />
              <Divider titlePlacement="start" plain>{t('deviceAccess.conditions')}</Divider>
              <Form.List name={[field.name, 'conditions']}>
                {(conditionFields, conditionOps) => <div className={styles.conditionsStack}>
                  {conditionFields.map((conditionField, conditionIndex) => <ConditionFields
                    key={conditionField.key}
                    index={conditionIndex}
                    ruleName={field.name}
                    conditionName={conditionField.name}
                    onRemove={() => conditionOps.remove(conditionField.name)}
                    t={t}
                  />)}
                  <Button
                    block
                    type="dashed"
                    icon={<PlusOutlined />}
                    onClick={() => conditionOps.add({ type: 'tac', operator: 'equal', required: true, allowMissing: false, evidenceTTLSeconds: 900 })}
                  >{t('deviceAccess.addCondition')}</Button>
                </div>}
              </Form.List>
            </Card>)}
            <Button
              className={styles.addRuleButton}
              block
              type="dashed"
              icon={<PlusOutlined />}
              onClick={() => add({ name: '', enabled: true, scopeType: 'all', conditions: [] })}
            >{t('deviceAccess.addRule')}</Button>
          </>}
        </Form.List>
      </Form>
    </Spin>
  </Modal>;
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

function ConditionFields({ index, ruleName, conditionName, onRemove, t }: { index: number; ruleName: number; conditionName: number; onRemove: () => void; t: (key: string) => string }) {
  const form = Form.useFormInstance<PolicyFormValue>();
  const operators: Record<AccessConditionType, AccessConditionOperator[]> = {
    tac: ['equal', 'in'],
    ecgi: ['equal', 'in'],
    observed_ip: ['equal', 'in', 'cidr'],
    gps: ['within_radius'],
  };
  return <Card
    size="small"
    className={styles.conditionCard}
    title={<Space><span className={styles.conditionIndex}>{index + 1}</span>{t('deviceAccess.accessCondition')}</Space>}
    extra={<Button danger type="text" aria-label={t('common.delete')} icon={<DeleteOutlined />} onClick={onRemove} />}
  >
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
            {operator !== 'within_radius' && <Col xs={24} md={8} lg={12}>
              <Form.Item name={[conditionName, 'expected']} label={operator === 'in' ? t('deviceAccess.expectedValues') : t('deviceAccess.expected')} rules={[{ required: true, whitespace: true }]}>
                <Input />
              </Form.Item>
            </Col>}
            {operator === 'within_radius' && <>
              <Col xs={24} md={8} lg={4}><Form.Item name={[conditionName, 'latitude']} label={t('deviceAccess.latitude')} rules={[{ required: true }]}><InputNumber className={styles.fullWidth} min={-90} max={90} precision={6} /></Form.Item></Col>
              <Col xs={24} md={8} lg={4}><Form.Item name={[conditionName, 'longitude']} label={t('deviceAccess.longitude')} rules={[{ required: true }]}><InputNumber className={styles.fullWidth} min={-180} max={180} precision={6} /></Form.Item></Col>
              <Col xs={24} md={8} lg={4}><Form.Item name={[conditionName, 'radiusMeters']} label={t('deviceAccess.radiusMeters')} rules={[{ required: true }]}><InputNumber className={styles.fullWidth} min={1} precision={0} /></Form.Item></Col>
            </>}
          </>;
        }}
      </Form.Item>
    </Row>
    <div className={styles.conditionOptions}>
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
    </div>
  </Card>;
}
