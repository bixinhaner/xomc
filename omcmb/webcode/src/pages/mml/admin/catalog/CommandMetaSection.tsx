import { useEffect, useState } from 'react';
import {
  Descriptions,
  Tag,
  Button,
  Space,
  Form,
  Input,
  Select,
  Switch,
  message,
} from 'antd';
import { useUpdateCommand } from '@core/hooks/api/useMmlAdmin';
import type { GroupTreeNode, GroupTreeCommand } from '@core/types/mmlConsole';
import type { MMLOperationType } from '@core/types/mml';
import { useT } from '@/hooks/useT';

// 2026-05-27 mml-admin-catalog-redesign-20260527 §4.2-4.3：
// 右栏顶部命令元数据 Descriptions（显示态）↔ Form（编辑态）内联切换。
//
// 2026-05-27 用户决策:取消 catalog_protected 在 UI 上的锁定逻辑。
// 是否可编辑/删除由路由层 RBAC(withAdminRole)+ 后端 API 权限校验决定,
// 不再在前端基于 catalogProtected 拦截。
//
// 编辑态:command_code / operation_type 后端 PATCH 不允许改 → readonly 展示。

export interface CommandMetaSectionProps {
  command: GroupTreeCommand;
  /** 命令所在分组,用于显示 + 切换分组的下拉 */
  parentGroupId: string;
  groupOptions: GroupTreeNode[];
  /** 当前是否处于编辑态（由父组件 lift up,支持 dirty 检测） */
  editing: boolean;
  onEditingChange: (editing: boolean) => void;
  onDirtyChange: (dirty: boolean) => void;
  onSaved?: () => void;
}

interface FormValues {
  groupId: string;
  commandCode: string;
  logicalCode: string;
  operationType: MMLOperationType;
  displayName: string;
  targetObject?: string;
  requireConfirm: boolean;
}

function toFormValues(cmd: GroupTreeCommand, groupId: string): FormValues {
  return {
    groupId,
    commandCode: cmd.commandCode,
    logicalCode: cmd.logicalCode,
    operationType: cmd.operationType,
    displayName:
      cmd.logicalNameI18n?.['zh-CN'] ||
      cmd.logicalNameI18n?.['en-US'] ||
      cmd.displayName ||
      '',
    targetObject: cmd.targetObject ?? undefined,
    requireConfirm: cmd.requireConfirm,
  };
}

export default function CommandMetaSection({
  command,
  parentGroupId,
  groupOptions,
  editing,
  onEditingChange,
  onDirtyChange,
  onSaved,
}: CommandMetaSectionProps): React.ReactElement {
  const t = useT();
  const [form] = Form.useForm<FormValues>();
  const updateMut = useUpdateCommand();
  const [submitting, setSubmitting] = useState(false);

  // 进入编辑态时填表 + 退出时清表
  useEffect(() => {
    if (editing) {
      form.setFieldsValue(toFormValues(command, parentGroupId));
      onDirtyChange(false);
    }
  }, [editing, command, parentGroupId, form, onDirtyChange]);

  const handleSave = async () => {
    try {
      setSubmitting(true);
      const values = await form.validateFields();
      // 后端 PATCH /commands/:id 不接受 command_code / operation_type(创建后不可改),
      // 由 toBackendUpdateCommand 兜底剔除,这里同样不传以保持意图清晰。
      await updateMut.mutateAsync({
        id: command.id,
        req: {
          groupId: values.groupId,
          logicalCode: values.logicalCode,
          // 去多语言:单值显示名同时写 zh-CN / en-US 两路(沿用后端契约字段)。
          commandNameI18n: {
            'zh-CN': values.displayName,
            'en-US': values.displayName,
          },
          targetObject: values.targetObject || null,
          requireConfirm: values.requireConfirm,
        },
      });
      message.success(t('mml.admin.catalog.common.saveSuccess'));
      onDirtyChange(false);
      onEditingChange(false);
      onSaved?.();
    } catch (e) {
      if (e instanceof Error) message.error(e.message);
    } finally {
      setSubmitting(false);
    }
  };

  const handleCancel = () => {
    onDirtyChange(false);
    onEditingChange(false);
  };

  if (editing) {
    return (
      <div style={{ marginBottom: 16 }}>
        <h3 style={{ marginTop: 0 }}>{t('mml.admin.catalog.commands.edit')}</h3>
        <Form
          form={form}
          layout="vertical"
          onValuesChange={() => onDirtyChange(true)}
        >
          <Form.Item
            name="groupId"
            label={t('mml.admin.catalog.form.group')}
            rules={[{ required: true, message: t('mml.admin.catalog.validation.required') }]}
          >
            <Select
              options={groupOptions.map((g) => ({
                label: g.displayName,
                value: g.id,
              }))}
              showSearch
              optionFilterProp="label"
            />
          </Form.Item>
          <Form.Item
            name="commandCode"
            label={t('mml.admin.catalog.form.commandCode')}
            extra={t('mml.admin.catalog.form.codeNotEditableHint')}
          >
            <Input disabled />
          </Form.Item>
          <Form.Item
            name="logicalCode"
            label={t('mml.admin.catalog.form.logicalCode')}
            rules={[{ required: true, message: t('mml.admin.catalog.validation.required') }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="operationType"
            label={t('mml.admin.catalog.form.opType')}
          >
            <Input disabled />
          </Form.Item>
          <Form.Item
            name="displayName"
            label={t('mml.admin.catalog.commands.displayName')}
            rules={[{ required: true, message: t('mml.admin.catalog.validation.required') }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="targetObject"
            label={t('mml.admin.catalog.form.targetObject')}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="requireConfirm"
            label={t('mml.admin.catalog.form.requireConfirm')}
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>
          <Space>
            <Button onClick={handleCancel}>
              {t('mml.admin.catalog.common.cancel')}
            </Button>
            <Button type="primary" loading={submitting} onClick={handleSave}>
              {t('mml.admin.catalog.common.save')}
            </Button>
          </Space>
        </Form>
      </div>
    );
  }

  // 显示态
  // 2026-06-05 用户决策：移除右栏命令「编辑/删除」按钮，与左侧命令树的编辑/删除重复。
  // 编辑入口统一走左侧命令树（其 editCommand 动作驱动本组件 editing 进入内联编辑表单）。
  return (
    <div style={{ marginBottom: 16 }}>
      <h3 style={{ margin: '0 0 12px' }}>{command.displayName}</h3>
      <Descriptions column={2} size="small" bordered>
        <Descriptions.Item label={t('mml.admin.catalog.commands.code')}>
          {command.commandCode}
        </Descriptions.Item>
        <Descriptions.Item label={t('mml.admin.catalog.commands.logicalCode')}>
          {command.logicalCode}
        </Descriptions.Item>
        <Descriptions.Item label={t('mml.admin.catalog.commands.op')}>
          <Tag>{command.operationType}</Tag>
        </Descriptions.Item>
        <Descriptions.Item label={t('mml.admin.catalog.commands.displayName')}>
          {command.displayName}
        </Descriptions.Item>
        {command.targetObject ? (
          <Descriptions.Item
            label={t('mml.admin.catalog.commands.targetObject')}
            span={2}
          >
            {command.targetObject}
          </Descriptions.Item>
        ) : null}
        <Descriptions.Item
          label={t('mml.admin.catalog.commands.requireConfirm')}
          span={2}
        >
          {command.requireConfirm ? (
            <Tag color="warning">true</Tag>
          ) : (
            'false'
          )}
        </Descriptions.Item>
        <Descriptions.Item label="source" span={2}>
          <Tag color={command.source === 'admin' ? 'blue' : 'default'}>
            {command.source ?? 'standard'}
          </Tag>
        </Descriptions.Item>
      </Descriptions>
    </div>
  );
}
