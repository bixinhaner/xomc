import { useEffect, useRef, useState } from 'react';
import { Button, Form, Input, Modal, Space, Spin, message } from 'antd';
import { DownloadOutlined, UploadOutlined } from '@ant-design/icons';
import type { MMLScript, MMLScriptImportValidation } from '@core/types/mml';
import { useCreateImportedMMLScript, useReplaceImportedMMLScript, useValidateMMLScriptImport } from '@core/hooks/api/useMML';
import { mmlApi } from '@core/services/api/mmlApi';
import ScriptImportPreview from './ScriptImportPreview';

export interface ScriptImportModalProps {
  open: boolean;
  onClose: () => void;
  script?: MMLScript | null;
  onSaved?: (script: MMLScript) => void;
}

interface ImportForm { scriptName: string; description: string; }

function createRequestId(): string {
  return globalThis.crypto?.randomUUID?.() ?? `mml-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

function validationFromError(error: unknown): MMLScriptImportValidation | undefined {
  if (!error || typeof error !== 'object') return undefined;
  const candidate = (error as { validation?: unknown }).validation;
  return candidate && typeof candidate === 'object' ? candidate as MMLScriptImportValidation : undefined;
}

/** Import or re-import a TXT script. Content is never editable in the browser. */
export default function ScriptImportModal({ open, onClose, script, onSaved }: ScriptImportModalProps) {
  const [form] = Form.useForm<ImportForm>();
  const [validation, setValidation] = useState<MMLScriptImportValidation | null>(null);
  const [uploading, setUploading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const submittingRef = useRef(false);
  const requestIdRef = useRef('');
  const inputRef = useRef<HTMLInputElement | null>(null);
  const validateMutation = useValidateMMLScriptImport();
  const createMutation = useCreateImportedMMLScript();
  const replaceMutation = useReplaceImportedMMLScript();
  const saving = createMutation.isPending || replaceMutation.isPending || uploading || submitting;
  const hasErrors = Boolean(validation?.issues.some((issue) => issue.severity === 'error') || validation?.summary.errorCount);

  useEffect(() => {
    if (!open) return;
    form.setFieldsValue({
      scriptName: script?.scriptName ?? '',
      description: script?.description ?? '',
    });
    setValidation(null);
    setSubmitting(false);
    submittingRef.current = false;
    requestIdRef.current = '';
  }, [form, open, script]);

  const validateFile = async (file: File) => {
    setValidation(null);
    requestIdRef.current = '';
    setUploading(true);
    try {
      const next = script?.id
        ? await mmlApi.validateScriptReplacement(script.id, file)
        : await validateMutation.mutateAsync(file);
      setValidation(next);
    } catch (error) {
      const errorValidation = validationFromError(error);
      setValidation(errorValidation ?? null);
      void message.error(error instanceof Error ? error.message : 'TXT 校验失败');
    } finally {
      setUploading(false);
    }
  };

  const handleSave = async () => {
    if (!validation?.validationToken || hasErrors) return;
    const values = await form.validateFields();
    if (validation.summary.warningCount > 0 || validation.issues.some((issue) => issue.severity === 'warning')) {
      Modal.confirm({
        title: '校验发现警告',
        content: '脚本包含警告，确认后继续保存。',
        okText: '继续保存',
        cancelText: '取消',
        onOk: () => doSave(values),
      });
      return;
    }
    await doSave(values);
  };

  const doSave = async (values: ImportForm) => {
    if (!validation?.validationToken) return;
    if (submittingRef.current) return;
    submittingRef.current = true;
    setSubmitting(true);
    const requestId = requestIdRef.current || createRequestId();
    requestIdRef.current = requestId;
    try {
      const result = script?.id
        ? await replaceMutation.mutateAsync({ id: script.id, input: { validationToken: validation.validationToken, scriptName: values.scriptName.trim(), description: values.description ?? '', tags: [], expectedUpdatedAt: script.updateTime, requestId } })
        : await createMutation.mutateAsync({ validationToken: validation.validationToken, scriptName: values.scriptName.trim(), description: values.description ?? '', tags: [], requestId });
      onSaved?.(result as MMLScript);
      onClose();
    } catch (error) {
      const errorValidation = validationFromError(error);
      if (errorValidation) setValidation(errorValidation);
      void message.error(error instanceof Error ? error.message : '保存失败');
    } finally {
      submittingRef.current = false;
      setSubmitting(false);
    }
  };

  const downloadTemplate = async () => {
    try {
      const result = await mmlApi.downloadScriptImportTemplate();
      const url = URL.createObjectURL(result.blob);
      const anchor = document.createElement('a'); anchor.href = url; anchor.download = result.filename; anchor.click(); URL.revokeObjectURL(url);
    } catch (error) { void message.error(error instanceof Error ? error.message : '模板下载失败'); }
  };

  return (
    <Modal open={open} onCancel={onClose} title={script ? '重新导入 MML TXT 脚本' : '导入 MML TXT 脚本'} width={900} footer={[
      <Button key="cancel" onClick={onClose}>取消</Button>,
      <Button key="save" aria-label="确认保存" type="primary" loading={saving} onClick={() => void handleSave()} disabled={saving || !validation || !validation.validationToken || hasErrors}>确认保存</Button>,
    ]} destroyOnHidden>
      <Form form={form} layout="vertical">
        <Form.Item label="脚本名称" name="scriptName" rules={[{ required: true, message: '请输入脚本名称' }]}><Input maxLength={128} /></Form.Item>
        <Form.Item label="描述" name="description"><Input maxLength={256} /></Form.Item>
      </Form>
      <Space style={{ marginBottom: 12 }}>
        <Button icon={<UploadOutlined />} onClick={() => inputRef.current?.click()} disabled={uploading}>选择 TXT</Button>
        <input id="script-txt-input" ref={inputRef} type="file" accept=".txt,text/plain" aria-label="选择 TXT" hidden onChange={(event) => { const file = event.target.files?.[0]; if (file) void validateFile(file); event.currentTarget.value = ''; }} />
        <Button icon={<DownloadOutlined />} onClick={() => void downloadTemplate()}>下载模板</Button>
      </Space>
      {uploading ? <Spin tip="正在校验脚本…" /> : null}
      {validation ? <ScriptImportPreview validation={validation} /> : null}
    </Modal>
  );
}
