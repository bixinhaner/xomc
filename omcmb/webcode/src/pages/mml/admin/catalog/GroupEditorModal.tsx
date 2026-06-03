import { useEffect } from 'react';
import { Modal, Form, Input, InputNumber, message } from 'antd';
import { useCreateGroup, useUpdateGroup } from '@core/hooks/api/useMmlAdmin';
import { useT } from '@/hooks/useT';

// 2026-05-27 重构（mml-admin-catalog-redesign-20260527）：移除 create-child mode。
// 一级分组语义下，create 永远 parentId=null；rename 仅改 group_code / 显示名 / 排序。
export type GroupEditorMode = 'create' | 'rename';

export interface GroupEditorModalProps {
  open: boolean;
  mode: GroupEditorMode;
  /** rename 时必填 — 自身 id；create 时忽略 */
  targetGroup?: {
    id: string;
    groupCode: string;
    displayNameI18n: Record<string, string>;
    displayOrder: number;
  };
  onClose: () => void;
  onSuccess?: () => void;
}

interface FormValues {
  groupCode: string;
  /** 单值显示名；提交时 zh-CN / en-US 两路同值（复用 displayNameI18n transform）。 */
  displayName: string;
  displayOrder: number;
}

export default function GroupEditorModal({
  open,
  mode,
  targetGroup,
  onClose,
  onSuccess,
}: GroupEditorModalProps) {
  const t = useT();
  const [form] = Form.useForm<FormValues>();
  const createMut = useCreateGroup();
  const updateMut = useUpdateGroup();

  useEffect(() => {
    if (!open) return;
    if (mode === 'rename' && targetGroup) {
      form.setFieldsValue({
        groupCode: targetGroup.groupCode,
        displayName: targetGroup.displayNameI18n?.['zh-CN'] || '',
        displayOrder: targetGroup.displayOrder,
      });
    } else {
      form.resetFields();
      form.setFieldsValue({ displayOrder: 100 });
    }
  }, [open, mode, targetGroup, form]);

  const titleKey =
    mode === 'rename'
      ? 'mml.admin.catalog.groups.editGroup'
      : 'mml.admin.catalog.groups.add';

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      // 去多语言:单值显示名同时写入 zh-CN / en-US 两路,保证后端读哪路都拿到该值。
      const displayNameI18n: Record<string, string> = {
        'zh-CN': values.displayName,
        'en-US': values.displayName,
      };
      if (mode === 'rename') {
        if (!targetGroup) return;
        await updateMut.mutateAsync({
          id: targetGroup.id,
          req: {
            groupCode: values.groupCode,
            displayNameI18n,
            displayOrder: values.displayOrder,
          },
        });
      } else {
        // R1: 新建分组永远顶层,parentId 不传（后端默认 null）
        await createMut.mutateAsync({
          groupCode: values.groupCode,
          displayNameI18n,
          displayOrder: values.displayOrder,
        });
      }
      message.success(t('mml.admin.catalog.common.saveSuccess'));
      onSuccess?.();
      onClose();
    } catch (e) {
      if (e instanceof Error) {
        message.error(e.message);
      }
    }
  };

  return (
    <Modal
      open={open}
      title={t(titleKey)}
      onCancel={onClose}
      onOk={handleOk}
      confirmLoading={createMut.isPending || updateMut.isPending}
      okText={t('mml.admin.catalog.common.save')}
      cancelText={t('mml.admin.catalog.common.cancel')}
      destroyOnHidden
    >
      <Form form={form} layout="vertical">
        <Form.Item
          name="groupCode"
          label={t('mml.admin.catalog.form.groupCode')}
          rules={[
            { required: true, message: t('mml.admin.catalog.validation.required') },
            {
              pattern: /^[A-Z0-9_]+$/,
              message: t('mml.admin.catalog.validation.upperSnakeCase'),
            },
          ]}
        >
          <Input
            placeholder={t('mml.admin.catalog.form.groupCodePlaceholder')}
            disabled={mode === 'rename'}
          />
        </Form.Item>
        <Form.Item
          name="displayName"
          label={t('mml.admin.catalog.form.name')}
          rules={[{ required: true, message: t('mml.admin.catalog.validation.required') }]}
        >
          <Input maxLength={128} />
        </Form.Item>
        <Form.Item
          name="displayOrder"
          label={t('mml.admin.catalog.form.sortOrder')}
        >
          <InputNumber min={0} step={10} style={{ width: 160 }} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
