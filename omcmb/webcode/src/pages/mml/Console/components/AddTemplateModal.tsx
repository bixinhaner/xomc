import { useEffect } from 'react';
import { App, Modal, Form, Input, Button } from 'antd';
import { AxiosError } from 'axios';
import { useCreateMMLTemplate, useUpdateMMLTemplate } from '@core/hooks/api/useMML';
import type { MMLCustomCommand } from '@core/types/mml';
import { useT } from '@/hooks/useT';
import OperationTypeWithModify from '../../components/OperationTypeWithModify';
import PathPicker from './PathPicker';

interface AddTemplateModalProps {
  open: boolean;
  scope: 'public' | 'private';
  /**
   * 非空时进入编辑模式：表单回填、提交走 update mutation、scope 锁定不可改。
   * 关联：docs/design/mml-user-private-template-crud-20260520.md §5.2 D13
   */
  editingTemplate?: MMLCustomCommand | null;
  onClose: () => void;
  onSuccess: () => void;
  onSaveAndExecute?: (template: Omit<MMLCustomCommand, 'id' | 'creator' | 'createdAt' | 'updatedAt'>) => void;
}

// 业务错误码（与 omcgo/global/errors.go 对齐）
const ERR_CODE_TEMPLATE_NAME_DUPLICATED = 17008;

interface BackendErrorBody {
  code?: number;
  biz_code?: number;
  msg?: string;
  message?: string;
}

function extractErrorCode(err: unknown): number | undefined {
  if (err instanceof AxiosError && err.response?.data) {
    const body = err.response.data as BackendErrorBody;
    return body.code ?? body.biz_code;
  }
  return undefined;
}

export default function AddTemplateModal({
  open,
  scope,
  editingTemplate,
  onClose,
  onSuccess,
  onSaveAndExecute,
}: AddTemplateModalProps) {
  const t = useT();
  // antd v5：Modal 嵌套场景下静态 message 调用会脱离 ConfigProvider/App 上下文导致提示丢失（T-0097）。
  // 统一改走 App.useApp() 的 scoped messageApi。
  const { message } = App.useApp();

  const [form] = Form.useForm();
  const createMutation = useCreateMMLTemplate();
  const updateMutation = useUpdateMMLTemplate();

  const isEdit = Boolean(editingTemplate);
  // 编辑时 scope 强制取 editingTemplate.commandScope；add 时取 prop。
  const effectiveScope = (editingTemplate?.commandScope as 'public' | 'private') ?? scope;

  useEffect(() => {
    if (!open) return;
    if (editingTemplate) {
      form.setFieldsValue({
        templateName: editingTemplate.commandName,
        commandCode: editingTemplate.commandCode,
        operationType: editingTemplate.operationType,
        description: editingTemplate.description ?? '',
        paramPaths: editingTemplate.paramPaths ?? [],
      });
    } else {
      form.resetFields();
      form.setFieldsValue({ paramPaths: [] });
    }
  }, [open, editingTemplate, form]);

  const handleError = (err: unknown, fallbackKey: string) => {
    const code = extractErrorCode(err);
    if (code === ERR_CODE_TEMPLATE_NAME_DUPLICATED) {
      void message.error(t('mml.template.error.nameDuplicated'));
      return;
    }
    // 403 — 创建者/super_admin 之外的越权写
    if (err instanceof AxiosError && err.response?.status === 403) {
      void message.error(t('mml.template.error.notOwner'));
      return;
    }
    void message.error(
      t(fallbackKey, {
        error: err instanceof Error ? err.message : String(err ?? 'Unknown'),
      }),
    );
  };

  const handleSubmit = async (andExecute: boolean) => {
    try {
      const values = await form.validateFields();

      // 命令定义不再录入修改值（已移除「修改值入口」）：新增恒为空，编辑保留既有 parameters
      // 不被清空（历史命令可能烘焙过值，避免编辑误删）。具体修改值在 console 执行时录入。
      const parameters = isEdit && editingTemplate ? (editingTemplate.parameters ?? {}) : {};

      // T-0090 子项 ②：UI 删 categoryGroup / productClasses / 参数配置 section；
      // productClasses column 已由 T-0090-b 真删；categoryGroup schema 仍 required，
      // 提交时传 empty default。paramPaths 由 Bundle D 通过 PathPicker 收集。
      const template: Omit<MMLCustomCommand, 'id' | 'creator' | 'createdAt' | 'updatedAt'> = {
        commandName: values.templateName,
        commandCode: values.commandCode,
        operationType: values.operationType,
        commandScope: effectiveScope,
        categoryGroup: '',
        parameters,
        paramPaths: Array.isArray(values.paramPaths) ? values.paramPaths : [],
        description: values.description ?? '',
      };

      try {
        if (isEdit && editingTemplate) {
          await updateMutation.mutateAsync({ id: editingTemplate.id, data: template });
          void message.success(t('mml.template.updated'));
        } else {
          await createMutation.mutateAsync(template);
          void message.success(
            effectiveScope === 'public'
              ? t('mml.console.publicTemplateCreated')
              : t('mml.console.privateTemplateCreated'),
          );
        }
      } catch (err) {
        handleError(err, isEdit ? 'common.updateFailed' : 'common.addFailed');
        return;
      }
      onSuccess();

      if (!isEdit && andExecute && onSaveAndExecute) {
        onSaveAndExecute(template);
      }

      onClose();
    } catch {
      // 表单 validateFields 失败：inline 错误已展示在各字段下方
    }
  };

  const titleKey = isEdit
    ? effectiveScope === 'public'
      ? 'mml.template.editPublicTitle'
      : 'mml.template.editPrivateTitle'
    : effectiveScope === 'public'
      ? 'mml.console.addPublicTemplate'
      : 'mml.console.addPrivateTemplate';

  const isPending = isEdit ? updateMutation.isPending : createMutation.isPending;

  return (
    <Modal
      title={t(titleKey)}
      open={open}
      onCancel={onClose}
      width={720}
      destroyOnHidden
      footer={
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
          <Button onClick={onClose}>{t('common.cancel')}</Button>
          <Button onClick={() => void handleSubmit(false)} loading={isPending}>
            {isEdit ? t('common.save') : t('mml.console.saveOnly')}
          </Button>
          {!isEdit && onSaveAndExecute && (
            <Button type="primary" onClick={() => void handleSubmit(true)} loading={isPending}>
              {t('mml.console.saveAndExecute')}
            </Button>
          )}
        </div>
      }
    >
      <Form form={form} layout="vertical" size="small">
        <Form.Item
          name="templateName"
          label={t('mml.console.commandName')}
          rules={[{ required: true, message: t('mml.console.inputCommandName') }]}
        >
          <Input placeholder={t('mml.console.inputCommandName')} maxLength={200} />
        </Form.Item>

        <Form.Item
          name="commandCode"
          label={t('mml.console.commandCode')}
          rules={[{ required: true, message: t('mml.console.commandCodeRequired') }]}
        >
          <Input
            placeholder={t('mml.console.commandCodeTextareaPlaceholder')}
            maxLength={200}
            style={{ fontFamily: "'SFMono-Regular', Consolas, monospace" }}
          />
        </Form.Item>

        <OperationTypeWithModify />

        <Form.Item
          name="paramPaths"
          label={t('mml.console.pathPicker.label')}
          valuePropName="value"
          trigger="onChange"
          rules={[
            { required: true, type: 'array', min: 1, message: t('mml.console.pathPicker.required') },
          ]}
        >
          <PathPicker value={[]} onChange={() => undefined} />
        </Form.Item>

        <Form.Item name="description" label={t('common.description')}>
          <Input.TextArea rows={2} placeholder={t('mml.console.commandDescriptionOptional')} maxLength={500} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
