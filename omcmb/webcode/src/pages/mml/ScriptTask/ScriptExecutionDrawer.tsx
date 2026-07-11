import { useEffect, useRef, useState } from 'react';
import { Button, Checkbox, DatePicker, Drawer, Form, Input, InputNumber, Modal, Radio, Space, TimePicker, Typography, message } from 'antd';
import type { Dayjs } from 'dayjs';
import type { MMLExecuteType, MMLScript, MMLScriptImportValidation, MMLScriptExecutionInput } from '@core/types/mml';
import { useCreateMMLScriptExecution } from '@core/hooks/api/useMML';
import ScriptImportPreview from './ScriptImportPreview';
import { buildMmlScriptExecutionTaskName } from '../utils/defaultTaskName';

export interface ScriptExecutionDrawerProps {
  open: boolean;
  script: MMLScript | null;
  onClose: () => void;
  onSuccess?: () => void;
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
  const [form] = Form.useForm<ExecutionForm>();
  const [validation, setValidation] = useState<MMLScriptImportValidation | null>(null);
  const [errorCodes, setErrorCodes] = useState<string[]>([]);
  const [warningValues, setWarningValues] = useState<MMLScriptExecutionInput | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const submittingRef = useRef(false);
  const requestIdRef = useRef('');
  const executionMutation = useCreateMMLScriptExecution();
  const executeType = Form.useWatch('executeType', form);

  useEffect(() => {
    if (!open) return;
    const taskName = script ? buildMmlScriptExecutionTaskName(script.scriptName) : '';
    form.setFieldsValue({ taskName, executeType: 'immediate', offlineRetry: false, offlineRetryWait: 60, failedRetry: false, failedRetryCount: 3, failedRetryInterval: 5 });
    setValidation(null); setWarningValues(null); setErrorCodes([]);
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
      setValidation(null); onSuccess?.(); onClose();
    } catch (error) {
      const errorValidation = validationFromError(error);
      if (errorValidation) {
        setValidation(errorValidation);
        setErrorCodes(errorValidation.issues.filter((issue) => issue.severity === 'error').map((issue) => issue.code));
        const hasValidationErrors = errorValidation.summary.errorCount > 0 || errorValidation.issues.some((issue) => issue.severity === 'error');
        if (!hasValidationErrors && (errorValidation.summary.warningCount > 0 || errorValidation.issues.some((issue) => issue.severity === 'warning'))) {
          setWarningValues({ ...input, requestId });
          return;
        }
      }
      void message.error(error instanceof Error ? error.message : '执行校验失败');
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
      taskName: values.taskName.trim(), executeType: values.executeType, scheduledAt: values.scheduledAt?.toISOString(),
      periodStart: values.periodRange?.[0]?.toISOString(), periodEnd: values.periodRange?.[1]?.toISOString(), periodTime: values.periodTime?.format('HH:mm:ss'),
      offlineRetry: values.offlineRetry, offlineRetryWait: values.offlineRetryWait, failedRetry: values.failedRetry,
      failedRetryCount: values.failedRetryCount, failedRetryInterval: values.failedRetryInterval,
    };
    await execute(input);
  };

  const confirmWarnings = () => {
    if (!warningValues) return;
    const input = { ...warningValues, confirmWarnings: true };
    setWarningValues(null); void execute(input);
  };

  return (
    <Drawer open={open} onClose={onClose} title={script ? `执行：${script.scriptName}` : '执行脚本'} width={520} destroyOnHidden>
      <Form form={form} layout="vertical">
        <Form.Item label="任务名称" name="taskName" rules={[{ required: true, message: '请输入任务名称' }]}><Input /></Form.Item>
        <Form.Item label="执行方式" name="executeType"><Radio.Group options={[{ value: 'immediate', label: '立即' }, { value: 'suspended', label: '挂起' }, { value: 'scheduled', label: '定时' }, { value: 'periodic', label: '周期' }]} /></Form.Item>
        <Form.Item label="执行时间" name="scheduledAt" rules={executeType === 'scheduled' ? [{ required: true, message: '请选择执行时间' }] : []}><DatePicker showTime style={{ width: '100%' }} /></Form.Item>
        {executeType === 'periodic' ? <>
          <Form.Item label="周期日期" name="periodRange" rules={[{ required: true, message: '请选择周期日期范围' }]}><DatePicker.RangePicker style={{ width: '100%' }} /></Form.Item>
          <Form.Item label="周期时间" name="periodTime" rules={[{ required: true, message: '请选择周期执行时间' }]}><TimePicker style={{ width: '100%' }} /></Form.Item>
        </> : null}
        <Space direction="vertical" style={{ width: '100%' }}>
          <Form.Item name="offlineRetry" valuePropName="checked" noStyle><Checkbox>离线等待重试</Checkbox></Form.Item>
          <Form.Item name="offlineRetryWait" label="离线等待（秒）"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item>
          <Form.Item name="failedRetry" valuePropName="checked" noStyle><Checkbox>失败重试</Checkbox></Form.Item>
          <Form.Item name="failedRetryCount" label="失败重试次数"><InputNumber min={0} style={{ width: '100%' }} /></Form.Item>
          <Form.Item name="failedRetryInterval" label="失败重试间隔（秒）"><InputNumber min={1} style={{ width: '100%' }} /></Form.Item>
        </Space>
        <Button aria-label="执行" type="primary" htmlType="button" loading={executionMutation.isPending || submitting} disabled={executionMutation.isPending || submitting} onClick={() => void submit()}>执行</Button>
      </Form>
      {errorCodes.length ? <Typography.Text type="danger">{errorCodes.join(', ')}</Typography.Text> : null}
      {validation ? <ScriptImportPreview validation={validation} /> : null}
      <Modal open={Boolean(warningValues)} title="校验发现警告" onCancel={() => setWarningValues(null)} onOk={confirmWarnings} okText="确认执行" cancelText="取消">请确认后继续执行。</Modal>
    </Drawer>
  );
}
