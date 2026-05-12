import { useEffect } from 'react';
import { App, Modal, Form, Input, Button } from 'antd';
import { useCreateMMLTemplate } from '@core/hooks/api/useMML';
import type { MMLCustomCommand } from '@core/types/mml';
import { useT } from '@/hooks/useT';
import CommandCodeTextarea from '../../components/CommandCodeTextarea';
import OperationTypeWithModify from '../../components/OperationTypeWithModify';

interface AddTemplateModalProps {
  open: boolean;
  scope: 'public' | 'private';
  onClose: () => void;
  onSuccess: () => void;
  onSaveAndExecute?: (template: Omit<MMLCustomCommand, 'id' | 'creator' | 'createdAt' | 'updatedAt'>) => void;
}

// 把「K=V 一行一对」字符串解析为参数 dict。空行与无 '=' 的行被丢弃。
function parseModifyValues(text: string): Record<string, string> {
  const out: Record<string, string> = {};
  text.split(/\r?\n/).forEach((line) => {
    const trimmed = line.trim();
    if (!trimmed) return;
    const eq = trimmed.indexOf('=');
    if (eq <= 0) return;
    const key = trimmed.slice(0, eq).trim();
    const value = trimmed.slice(eq + 1).trim();
    if (key) out[key] = value;
  });
  return out;
}

export default function AddTemplateModal({ open, scope, onClose, onSuccess, onSaveAndExecute }: AddTemplateModalProps) {
  const t = useT();
  // antd v5：Modal 嵌套场景下静态 message 调用会脱离 ConfigProvider/App 上下文导致提示丢失（T-0097）。
  // 统一改走 App.useApp() 的 scoped messageApi。
  const { message } = App.useApp();

  const [form] = Form.useForm();
  const createMutation = useCreateMMLTemplate();

  useEffect(() => {
    if (open) {
      form.resetFields();
    }
  }, [form, open]);

  const handleSubmit = async (andExecute: boolean) => {
    try {
      const values = await form.validateFields();

      // T-0090 子项 ①：MOD 操作下把「修改值入口」TextArea 内容解析为 parameters dict；
      // 其它操作类型 parameters 留空。
      const parameters =
        values.operationType === 'MOD' && typeof values.modifyValues === 'string'
          ? parseModifyValues(values.modifyValues)
          : {};

      // T-0090 子项 ②：UI 删 categoryGroup / productTypes / 参数配置 section；
      // 但 MMLCustomCommand schema 仍 required（T-0090-b 才真删 column），
      // 提交时传 empty default 保持后端兼容。
      const template: Omit<MMLCustomCommand, 'id' | 'creator' | 'createdAt' | 'updatedAt'> = {
        commandName: values.templateName,
        commandCode: values.commandCode,
        operationType: values.operationType,
        commandScope: scope,
        categoryGroup: '',
        parameters,
        paramPaths: [],
        description: values.description ?? '',
        productTypes: [],
      };

      try {
        await createMutation.mutateAsync(template);
      } catch (err) {
        // API 调用失败：必须显式告知用户，不能吞错（否则界面静默关闭看起来成功）
        void message.error(
          t('common.addFailed', {
            error: err instanceof Error ? err.message : String(err ?? 'Unknown'),
          }),
        );
        return;
      }
      message.success(
        scope === 'public' ? t('mml.console.publicTemplateCreated') : t('mml.console.privateTemplateCreated'),
      );
      onSuccess();

      if (andExecute && onSaveAndExecute) {
        onSaveAndExecute(template);
      }

      onClose();
    } catch {
      // 表单 validateFields 失败：inline 错误已展示在各字段下方
    }
  };

  return (
    <Modal
      title={scope === 'public' ? t('mml.console.addPublicTemplate') : t('mml.console.addPrivateTemplate')}
      open={open}
      onCancel={onClose}
      width={600}
      destroyOnClose
      footer={
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
          <Button onClick={onClose}>{t('common.cancel')}</Button>
          <Button onClick={() => void handleSubmit(false)} loading={createMutation.isPending}>
            {t('mml.console.saveOnly')}
          </Button>
          {onSaveAndExecute && (
            <Button type="primary" onClick={() => void handleSubmit(true)} loading={createMutation.isPending}>
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
          extra={t('mml.console.commandCodeTextareaTip')}
        >
          <CommandCodeTextarea />
        </Form.Item>

        <OperationTypeWithModify form={form} />

        <Form.Item name="description" label={t('common.description')}>
          <Input.TextArea rows={2} placeholder={t('mml.console.commandDescriptionOptional')} maxLength={500} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
