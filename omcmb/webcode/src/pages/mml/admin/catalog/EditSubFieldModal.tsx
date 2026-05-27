import { useEffect } from 'react';
import { Modal, Form, Input, InputNumber, Switch, message } from 'antd';
import { useUpdateSubField } from '@core/hooks/api/useMmlAdmin';
import type { AdminSubFieldEnriched } from '@core/types/mmlAdmin';
import { useT } from '@/hooks/useT';

// 2026-05-27 mml-admin-catalog-redesign-20260527 §5.2：行内编辑 path。
// 仅改 labelI18n / defaultSelected / displayOrder；mmlCode 与 tr069Path 只读
// （tr069Path 是 standard_params 真值源，mmlCode 由 batch 时派生且后端持久化）。
export interface EditSubFieldModalProps {
  open: boolean;
  commandId: string;
  subField: AdminSubFieldEnriched | undefined;
  onClose: () => void;
  onSuccess?: () => void;
}

interface FormValues {
  labelZh: string;
  labelEn: string;
  defaultSelected: boolean;
  sortOrder: number;
}

export default function EditSubFieldModal({
  open,
  commandId,
  subField,
  onClose,
  onSuccess,
}: EditSubFieldModalProps) {
  const t = useT();
  const [form] = Form.useForm<FormValues>();
  const updateMut = useUpdateSubField();

  useEffect(() => {
    if (!open || !subField) return;
    form.setFieldsValue({
      labelZh: subField.labelI18n?.['zh-CN'] ?? '',
      labelEn: subField.labelI18n?.['en-US'] ?? '',
      defaultSelected: subField.defaultSelected,
      sortOrder: subField.sortOrder,
    });
  }, [open, subField, form]);

  const handleOk = async () => {
    if (!subField) return;
    try {
      const values = await form.validateFields();
      await updateMut.mutateAsync({
        commandId,
        subFieldId: subField.id,
        req: {
          labelI18n: {
            'zh-CN': values.labelZh,
            'en-US': values.labelEn,
          },
          defaultSelected: values.defaultSelected,
          sortOrder: values.sortOrder,
        },
      });
      message.success(t('mml.admin.catalog.common.saveSuccess'));
      onSuccess?.();
      onClose();
    } catch (e) {
      if (e instanceof Error) message.error(e.message);
    }
  };

  return (
    <Modal
      open={open}
      title={t('mml.admin.catalog.subField.editTitle')}
      onCancel={onClose}
      onOk={handleOk}
      confirmLoading={updateMut.isPending}
      okText={t('mml.admin.catalog.common.save')}
      cancelText={t('mml.admin.catalog.common.cancel')}
      destroyOnHidden
      width={520}
    >
      <Form form={form} layout="vertical">
        <Form.Item label="MML Code">
          <Input value={subField?.mmlCode ?? ''} disabled />
        </Form.Item>
        <Form.Item label={t('mml.admin.catalog.subField.editPathReadonly')}>
          <Input value={subField?.tr069Path ?? ''} disabled />
        </Form.Item>
        <Form.Item
          name="labelZh"
          label={t('mml.admin.catalog.subField.labelZh')}
          rules={[{ required: true }]}
        >
          <Input placeholder="小区 ID" />
        </Form.Item>
        <Form.Item
          name="labelEn"
          label={t('mml.admin.catalog.subField.labelEn')}
          rules={[{ required: true }]}
        >
          <Input placeholder="Cell ID" />
        </Form.Item>
        <Form.Item
          name="defaultSelected"
          label={t('mml.admin.catalog.subField.defaultSelected')}
          valuePropName="checked"
        >
          <Switch />
        </Form.Item>
        <Form.Item
          name="sortOrder"
          label={t('mml.admin.catalog.subField.sortOrder')}
        >
          <InputNumber min={0} step={10} style={{ width: 160 }} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
