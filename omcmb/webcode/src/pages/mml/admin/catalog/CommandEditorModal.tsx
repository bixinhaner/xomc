import { useEffect } from 'react';
import { Modal, Form, Input, Select, Switch, message } from 'antd';
import { useCreateCommand } from '@core/hooks/api/useMmlAdmin';
import type { GroupTreeNode } from '@core/types/mmlConsole';
import type { MMLOperationType } from '@core/types/mml';
import { useT } from '@/hooks/useT';

// 2026-05-27 mml-admin-catalog-redesign-20260527 §5.2：分组旁 [⋯] → 新增命令的入口。
// 编辑命令通过 CommandMetaSection 内联表单完成（不复用此 Modal）。
const OPERATION_TYPES: MMLOperationType[] = ['LST', 'MOD', 'ADD', 'RMV', 'DSP', 'ACT', 'DEA', 'RST', 'CLR', 'UPG'];

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
  operationType: MMLOperationType;
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
          label={t('mml.admin.catalog.commands.group')}
          rules={[{ required: true }]}
        >
          <Select
            placeholder={t('mml.admin.catalog.commands.group')}
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
          label="command_code"
          rules={[
            { required: true },
            { pattern: /^[A-Z0-9_]+$/, message: 'UPPER_SNAKE_CASE only' },
          ]}
        >
          <Input placeholder="QUERY_CELL_STATUS" />
        </Form.Item>
        <Form.Item
          name="logicalCode"
          label="logical_code"
          rules={[{ required: true }]}
        >
          <Input placeholder="MML_QRY_CELL" />
        </Form.Item>
        <Form.Item
          name="operationType"
          label={t('mml.admin.catalog.commands.op')}
          rules={[{ required: true }]}
        >
          <Select options={OPERATION_TYPES.map((op) => ({ label: op, value: op }))} />
        </Form.Item>
        <Form.Item
          name="displayNameZh"
          label={t('mml.admin.catalog.commands.displayNameZh')}
          rules={[{ required: true }]}
        >
          <Input placeholder="查询小区状态" />
        </Form.Item>
        <Form.Item
          name="displayNameEn"
          label={t('mml.admin.catalog.commands.displayNameEn')}
          rules={[{ required: true }]}
        >
          <Input placeholder="Query Cell Status" />
        </Form.Item>
        <Form.Item
          name="targetObject"
          label={t('mml.admin.catalog.commands.targetObject')}
        >
          <Input placeholder="Device.Services.X_CMCC_LTE.Cell.{i}." />
        </Form.Item>
        <Form.Item
          name="requireConfirm"
          label={t('mml.admin.catalog.commands.requireConfirm')}
          valuePropName="checked"
        >
          <Switch />
        </Form.Item>
      </Form>
    </Modal>
  );
}
