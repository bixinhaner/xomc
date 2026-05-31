import { useEffect } from 'react';
import { Modal, Form, Input, Select, Switch, message } from 'antd';
import { useCreateCommand } from '@core/hooks/api/useMmlAdmin';
import {
  BACKEND_ALLOWED_OPERATION_TYPES,
  type BackendOperationType,
} from '@core/types/mmlAdmin';
import type { GroupTreeNode } from '@core/types/mmlConsole';
import { useT } from '@/hooks/useT';

// 2026-05-27 mml-admin-catalog-redesign-20260527 §5.2：分组旁 [⋯] → 新增命令的入口。
// 编辑命令通过 CommandMetaSection 内联表单完成（不复用此 Modal）。
//
// 2026-05-27 修复:operationType 收窄到后端 binding 允许的 4 种(LST/MOD/ADD/RMV),
// MML 其余 op 当前没有 admin 写端点支持。
const OPERATION_TYPES = BACKEND_ALLOWED_OPERATION_TYPES;

export interface CommandEditorModalProps {
  open: boolean;
  /** 触发新增的分组（new command 会预填 groupId） */
  parentGroup: GroupTreeNode | undefined;
  /** 全部顶层分组,作 group 下拉选项（含 parentGroup 本身） */
  groupOptions: GroupTreeNode[];
  onClose: () => void;
  onSuccess?: (createdCommandId: string) => void;
}

interface FormValues {
  groupId: string;
  commandCode: string;
  logicalCode: string;
  operationType: BackendOperationType;
  displayNameZh: string;
  displayNameEn: string;
  targetObject?: string;
  requireConfirm: boolean;
}

export default function CommandEditorModal({
  open,
  parentGroup,
  groupOptions,
  onClose,
  onSuccess,
}: CommandEditorModalProps) {
  const t = useT();
  const [form] = Form.useForm<FormValues>();
  const createMut = useCreateCommand();

  useEffect(() => {
    if (!open) return;
    form.resetFields();
    form.setFieldsValue({
      groupId: parentGroup?.id ?? '',
      requireConfirm: false,
      operationType: 'LST',
    });
  }, [open, parentGroup, form]);

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      const created = await createMut.mutateAsync({
        groupId: values.groupId,
        commandCode: values.commandCode,
        logicalCode: values.logicalCode,
        operationType: values.operationType,
        commandNameI18n: {
          'zh-CN': values.displayNameZh,
          'en-US': values.displayNameEn,
        },
        targetObject: values.targetObject || undefined,
        requireConfirm: values.requireConfirm,
      });
      message.success(t('mml.admin.catalog.common.saveSuccess'));
      onSuccess?.(created.id);
      onClose();
    } catch (e) {
      if (e instanceof Error) message.error(e.message);
    }
  };

  return (
    <Modal
      open={open}
      title={t('mml.admin.catalog.commands.add')}
      onCancel={onClose}
      onOk={handleOk}
      confirmLoading={createMut.isPending}
      okText={t('mml.admin.catalog.common.save')}
      cancelText={t('mml.admin.catalog.common.cancel')}
      destroyOnHidden
      width={560}
    >
      <Form form={form} layout="vertical">
        <Form.Item
          name="groupId"
          label={t('mml.admin.catalog.form.group')}
          rules={[{ required: true, message: t('mml.admin.catalog.validation.required') }]}
        >
          <Select
            placeholder={t('mml.admin.catalog.form.group')}
            options={groupOptions.map((g) => ({
              label: g.displayName,
              value: g.id,
            }))}
            disabled={!!parentGroup}
            showSearch
            optionFilterProp="label"
          />
        </Form.Item>
        <Form.Item
          name="commandCode"
          label={t('mml.admin.catalog.form.commandCode')}
          extra={t('mml.admin.catalog.form.codeNotEditableHint')}
          rules={[
            { required: true, message: t('mml.admin.catalog.validation.required') },
            {
              pattern: /^[A-Z0-9_]+$/,
              message: t('mml.admin.catalog.validation.upperSnakeCase'),
            },
          ]}
        >
          <Input placeholder={t('mml.admin.catalog.form.commandCodePlaceholder')} />
        </Form.Item>
        <Form.Item
          name="logicalCode"
          label={t('mml.admin.catalog.form.logicalCode')}
          rules={[{ required: true, message: t('mml.admin.catalog.validation.required') }]}
        >
          <Input placeholder={t('mml.admin.catalog.form.logicalCodePlaceholder')} />
        </Form.Item>
        <Form.Item
          name="operationType"
          label={t('mml.admin.catalog.form.opType')}
          rules={[{ required: true, message: t('mml.admin.catalog.validation.required') }]}
        >
          <Select options={OPERATION_TYPES.map((op) => ({ label: op, value: op }))} />
        </Form.Item>
        <Form.Item
          name="displayNameZh"
          label={t('mml.admin.catalog.form.nameZh')}
          rules={[{ required: true, message: t('mml.admin.catalog.validation.required') }]}
        >
          <Input placeholder={t('mml.admin.catalog.placeholder.queryCellStatus')} />
        </Form.Item>
        <Form.Item
          name="displayNameEn"
          label={t('mml.admin.catalog.form.nameEn')}
          rules={[{ required: true, message: t('mml.admin.catalog.validation.required') }]}
        >
          <Input placeholder="Query Cell Status" />
        </Form.Item>
        <Form.Item
          name="targetObject"
          label={t('mml.admin.catalog.form.targetObject')}
        >
          <Input placeholder="Device.Services.X_CMCC_LTE.Cell.{i}." />
        </Form.Item>
        <Form.Item
          name="requireConfirm"
          label={t('mml.admin.catalog.form.requireConfirm')}
          valuePropName="checked"
        >
          <Switch />
        </Form.Item>
      </Form>
    </Modal>
  );
}
