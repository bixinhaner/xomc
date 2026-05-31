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
  displayNameZh: string;
  displayNameEn: string;
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
        displayNameZh: targetGroup.displayNameI18n['zh-CN'] ?? '',
        displayNameEn: targetGroup.displayNameI18n['en-US'] ?? '',
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
      const displayNameI18n: Record<string, string> = {
        'zh-CN': values.displayNameZh,
        'en-US': values.displayNameEn,
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
          name="displayNameZh"
          label={t('mml.admin.catalog.form.nameZh')}
          rules={[{ required: true, message: t('mml.admin.catalog.validation.required') }]}
        >
          <Input placeholder={t('mml.admin.catalog.placeholder.bsConfig')} />
        </Form.Item>
        <Form.Item
          name="displayNameEn"
          label={t('mml.admin.catalog.form.nameEn')}
          rules={[{ required: true, message: t('mml.admin.catalog.validation.required') }]}
        >
          <Input placeholder="BSC Configuration" />
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
