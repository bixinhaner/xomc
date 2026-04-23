import { useEffect } from 'react';
import { Form, Input, Modal } from 'antd';

import { useCreateMMLScript } from '@core/hooks/api/useMML';
import { useT } from '@/hooks/useT';
import { toast } from '@/utils/toast';

// -----------------------------------------------------------------------------
// 保存脚本 —— 面向 mml_scripts 表。控制台只记"脚本名 + 描述 + 命令内容"，
// 不涉及执行策略；执行策略属于 mml_tasks，走 ScriptTaskDrawer。
// 历史上这里复用了 ScriptTaskDrawer，导致脚本被误写到 mml_tasks（to-do-list #3）。
//
// 提示稳定性（to-do-list 本轮 #2）：handleSubmit 返回 Promise，让 antd Modal
// 在 onOk 期间挂起并保持 loading 状态；完成后才 onClose。这样可以避免
// 调用方过早卸载导致 useMutation 的 onSuccess 回调所在组件尚未挂载 message
// 上下文。toast 反馈统一走 `@/utils/toast`。
// -----------------------------------------------------------------------------

interface SaveScriptModalProps {
  open: boolean;
  onClose: () => void;
  defaultName?: string;
  /** 命令行内容，来自 Console 已选择的命令 + 参数装配。只读展示 */
  content: string;
  onSuccess?: () => void;
}

interface SaveScriptForm {
  scriptName: string;
  description?: string;
}

export default function SaveScriptModal({
  open,
  onClose,
  defaultName,
  content,
  onSuccess,
}: SaveScriptModalProps) {
  const t = useT();
  const [form] = Form.useForm<SaveScriptForm>();
  const createScriptMutation = useCreateMMLScript();

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    form.setFieldsValue({
      scriptName: defaultName ?? '',
      description: '',
    });
  }, [open, defaultName, form]);

  const handleSubmit = async () => {
    let values: SaveScriptForm;
    try {
      values = await form.validateFields();
    } catch {
      // 校验失败：antd 已在字段下方展示错误，不需要 toast。抛回让 Modal 保持打开。
      throw new Error('VALIDATION_FAILED');
    }

    if (!content.trim()) {
      toast.warning(t('mml.console.saveScriptEmptyContent'));
      throw new Error('EMPTY_CONTENT');
    }

    try {
      await createScriptMutation.mutateAsync({
        scriptName: values.scriptName.trim(),
        description: values.description?.trim() ?? '',
        content,
        creator: '',
        tags: [],
        status: 'active',
        type: 'manual',
        progress: 0,
      });
    } catch (err) {
      toast.error(err, t('mml.console.saveScriptFailedPrefix'));
      throw err;
    }

    toast.success(t('mml.console.saveScriptSuccess'));
    onSuccess?.();
    onClose();
  };

  return (
    <Modal
      title={t('mml.console.saveScript')}
      open={open}
      onCancel={onClose}
      onOk={handleSubmit}
      okText={t('common.confirm')}
      cancelText={t('common.cancel')}
      confirmLoading={createScriptMutation.isPending}
      destroyOnClose
      width={540}
    >
      <Form form={form} layout="vertical">
        <Form.Item
          name="scriptName"
          label={t('mml.scriptName')}
          rules={[{ required: true, message: t('mml.inputScriptName') }]}
        >
          <Input maxLength={100} placeholder={t('mml.inputScriptName')} />
        </Form.Item>

        <Form.Item name="description" label={t('mml.description')}>
          <Input.TextArea rows={2} maxLength={200} placeholder={t('mml.description')} />
        </Form.Item>

        <Form.Item label={t('mml.scriptContent')}>
          <Input.TextArea
            readOnly
            rows={4}
            value={content}
            style={{ fontFamily: "'SFMono-Regular', Consolas, monospace", fontSize: 12 }}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
}
