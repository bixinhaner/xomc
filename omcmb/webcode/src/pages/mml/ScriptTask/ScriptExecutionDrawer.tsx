import { useEffect, useRef, useState } from 'react';
import { Button, Checkbox, DatePicker, Drawer, Form, Input, InputNumber, Modal, Radio, Space, TimePicker, Typography, message } from 'antd';
import type { Dayjs } from 'dayjs';
import type { MMLExecuteType, MMLScript, MMLScriptImportValidation, MMLScriptExecutionInput, MMLTask } from '@core/types/mml';
import { useCreateMMLScriptExecution } from '@core/hooks/api/useMML';
import { useSystemTimezoneValue } from '@core/hooks/api/useSystemTimezone';
import { toSystemTimezoneRFC3339 } from '@core/utils/systemTime';
import { useT } from '@/hooks/useT';
import ScriptImportPreview from './ScriptImportPreview';
import { buildMmlScriptExecutionTaskName } from '../utils/defaultTaskName';

export interface ScriptExecutionDrawerProps {
  open: boolean;
  script: MMLScript | null;
  onClose: () => void;
  onSuccess?: (task: MMLTask) => void;
}

interface ExecutionForm {
  taskName: string;
  executeType: MMLExecuteType;
  scheduledAt?: Dayjs;
  periodRange?: [Dayjs, Dayjs];
  periodTime?: Dayjs;
  offlineRetry: boolean;
  offlineRetryWait: number;
  failedRetry: boolean;
  failedRetryCount: number;
  failedRetryInterval: number;
}

function createRequestId(): string {
  return globalThis.crypto?.randomUUID?.() ?? `mml-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

function validationFromError(error: unknown): MMLScriptImportValidation | undefined {
  if (!error || typeof error !== 'object') return undefined;
  const candidate = (error as { validation?: unknown }).validation;
  return candidate && typeof candidate === 'object' ? candidate as MMLScriptImportValidation : undefined;
}

export default function ScriptExecutionDrawer({ open, script, onClose, onSuccess }: ScriptExecutionDrawerProps) {
  const t = useT();
  const [form] = Form.useForm<ExecutionForm>();
  const [validation, setValidation] = useState<MMLScriptImportValidation | null>(null);
  const [errorMessages, setErrorMessages] = useState<string[]>([]);
  const [warningValues, setWarningValues] = useState<MMLScriptExecutionInput | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const submittingRef = useRef(false);
  const requestIdRef = useRef('');
  const executionMutation = useCreateMMLScriptExecution();
  const systemTimezone = useSystemTimezoneValue();
  const executeType = Form.useWatch('executeType', form);
  const offlineRetry = Form.useWatch('offlineRetry', form);
  const failedRetry = Form.useWatch('failedRetry', form);
  const isScheduledExecution = executeType === 'scheduled';
  const isPeriodicExecution = executeType === 'periodic';
  const showOfflineRetryWait = Boolean(offlineRetry);
  const showFailedRetryInputs = Boolean(failedRetry);
  const executeTypeOptions = [
    { value: 'immediate', label: t('mml.scriptExecution.executeType.immediate') },
    { value: 'suspended', label: t('mml.scriptExecution.executeType.suspended') },
    { value: 'scheduled', label: t('mml.scriptExecution.executeType.scheduled') },
    { value: 'periodic', label: t('mml.scriptExecution.executeType.periodic') },
  ];

  useEffect(() => {
    if (!open) return;
    const taskName = script ? buildMmlScriptExecutionTaskName(script.scriptName) : '';
    form.setFieldsValue({ taskName, executeType: 'immediate', offlineRetry: false, offlineRetryWait: 60, failedRetry: false, failedRetryCount: 3, failedRetryInterval: 5 });
    setValidation(null); setWarningValues(null); setErrorMessages([]);
    setSubmitting(false); submittingRef.current = false; requestIdRef.current = '';
  }, [form, open, script]);

  const execute = async (input: MMLScriptExecutionInput) => {
    if (!script) return;
    if (submittingRef.current) return;
    submittingRef.current = true;
    setSubmitting(true);
    const requestId = input.requestId || requestIdRef.current || createRequestId();
    requestIdRef.current = requestId;
    try {
      const result = await executionMutation.mutateAsync({ id: script.id, input: { ...input, requestId } });
      const resultValidation = result.validation;
      if (resultValidation.summary.warningCount > 0 || resultValidation.issues.some((issue) => issue.severity === 'warning')) {
        setValidation(resultValidation); setWarningValues({ ...input, requestId }); return;
      }
      setValidation(null); onSuccess?.(result.task); onClose();
    } catch (error) {
      const errorValidation = validationFromError(error);
      if (errorValidation) {
        setValidation(errorValidation);
        setErrorMessages(errorValidation.issues.filter((issue) => issue.severity === 'error').map((issue) => issue.displayMessage || issue.message || t('mml.scriptImport.issueFallback')));
        const hasValidationErrors = errorValidation.summary.errorCount > 0 || errorValidation.issues.some((issue) => issue.severity === 'error');
        if (!hasValidationErrors && (errorValidation.summary.warningCount > 0 || errorValidation.issues.some((issue) => issue.severity === 'warning'))) {
          setWarningValues({ ...input, requestId });
          return;
        }
      }
      void message.error(error instanceof Error ? error.message : t('mml.scriptExecution.validationFailed'));
    } finally {
      submittingRef.current = false;
      setSubmitting(false);
    }
  };

  const submit = async () => {
    const values = form.getFieldsValue();
    if (!values.taskName?.trim()) {
      await form.validateFields(['taskName']);
      return;
    }
    if (values.executeType === 'scheduled') {
      try {
        await form.validateFields(['scheduledAt']);
      } catch {
        return;
      }
    }
    if (values.executeType === 'periodic') {
      try {
        await form.validateFields(['periodRange', 'periodTime']);
      } catch {
        return;
      }
    }
    const input: MMLScriptExecutionInput = {
      taskName: values.taskName.trim(),
      executeType: values.executeType,
      scheduledAt: values.executeType === 'scheduled'
        ? toSystemTimezoneRFC3339(values.scheduledAt, systemTimezone) ?? values.scheduledAt?.toISOString()
        : undefined,
      periodStart: values.executeType === 'periodic'
        ? toSystemTimezoneRFC3339(values.periodRange?.[0], systemTimezone) ?? values.periodRange?.[0]?.toISOString()
        : undefined,
      periodEnd: values.executeType === 'periodic'
        ? toSystemTimezoneRFC3339(values.periodRange?.[1], systemTimezone) ?? values.periodRange?.[1]?.toISOString()
        : undefined,
      periodTime: values.executeType === 'periodic' ? values.periodTime?.format('HH:mm:ss') : undefined,
      offlineRetry: values.offlineRetry,
      offlineRetryWait: values.offlineRetryWait,
      failedRetry: values.failedRetry,
      failedRetryCount: values.failedRetryCount,
      failedRetryInterval: values.failedRetryInterval,
    };
    await execute(input);
  };

  const confirmWarnings = () => {
    if (!warningValues) return;
    const input = { ...warningValues, confirmWarnings: true };
    setWarningValues(null); void execute(input);
  };

  return (
    <Drawer open={open} onClose={onClose} title={script ? t('mml.scriptExecution.titleWithName', { name: script.scriptName }) : t('mml.scriptExecution.title')} width={520} destroyOnHidden>
      <Form form={form} layout="vertical">
        <Form.Item label={t('mml.taskName')} name="taskName" rules={[{ required: true, message: t('mml.inputTaskNameRequired') }]}><Input /></Form.Item>
        <Form.Item label={t('mml.executeMethod')} name="executeType"><Radio.Group options={executeTypeOptions} /></Form.Item>
        {isScheduledExecution ? (
          <Form.Item label={t('mml.executionTime')} name="scheduledAt" rules={[{ required: true, message: t('mml.scriptExecution.selectExecutionTime') }]}><DatePicker showTime style={{ width: '100%' }} /></Form.Item>
        ) : null}
        {isPeriodicExecution ? <>
          <Form.Item label={t('mml.scriptExecution.periodDate')} name="periodRange" rules={[{ required: true, message: t('mml.scriptExecution.selectPeriodDateRange') }]}><DatePicker.RangePicker style={{ width: '100%' }} /></Form.Item>
          <Form.Item label={t('mml.periodTime')} name="periodTime" rules={[{ required: true, message: t('mml.scriptExecution.selectPeriodTime') }]}><TimePicker style={{ width: '100%' }} /></Form.Item>
        </> : null}
        <Space direction="vertical" style={{ width: '100%' }}>
          <div
            aria-label={t('mml.scriptExecution.retryOptions')}
            role="group"
            style={{ display: 'flex', gap: 24, alignItems: 'center', flexWrap: 'wrap' }}
          >
            <Form.Item name="offlineRetry" valuePropName="checked" noStyle><Checkbox>{t('mml.offlineRetryPolicy')}</Checkbox></Form.Item>
            <Form.Item name="failedRetry" valuePropName="checked" noStyle><Checkbox>{t('mml.failedRetryPolicy')}</Checkbox></Form.Item>
          </div>
          {showOfflineRetryWait ? (
            <Form.Item name="offlineRetryWait" label={t('mml.scriptExecution.offlineRetryWaitSeconds')}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item>
          ) : null}
          {showFailedRetryInputs ? <>
            <Form.Item name="failedRetryCount" label={t('mml.scriptExecution.failedRetryCount')}><InputNumber min={0} style={{ width: '100%' }} /></Form.Item>
            <Form.Item name="failedRetryInterval" label={t('mml.scriptExecution.failedRetryIntervalSeconds')}><InputNumber min={1} style={{ width: '100%' }} /></Form.Item>
          </> : null}
        </Space>
        <div
          aria-label={t('mml.scriptExecution.executionActions')}
          role="group"
          style={{
            marginTop: 20,
            paddingTop: 16,
            borderTop: '1px solid rgba(5, 5, 5, 0.06)',
          }}
        >
          <Button aria-label={t('mml.script.action.execute')} type="primary" htmlType="button" loading={executionMutation.isPending || submitting} disabled={executionMutation.isPending || submitting} onClick={() => void submit()}>{t('mml.script.action.execute')}</Button>
        </div>
      </Form>
      {errorMessages.length ? <Typography.Text type="danger">{errorMessages.join(', ')}</Typography.Text> : null}
      {validation ? <ScriptImportPreview validation={validation} /> : null}
      <Modal open={Boolean(warningValues)} title={t('mml.scriptImport.warningTitle')} onCancel={() => setWarningValues(null)} onOk={confirmWarnings} okText={t('mml.console.confirmExecute')} cancelText={t('common.cancel')}>{t('mml.scriptExecution.confirmWarnings')}</Modal>
    </Drawer>
  );
}
