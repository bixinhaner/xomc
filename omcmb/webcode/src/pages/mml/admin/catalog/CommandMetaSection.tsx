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
  Tooltip,
  message,
} from 'antd';
import { EditOutlined, DeleteOutlined } from '@ant-design/icons';
import { useUpdateCommand } from '@core/hooks/api/useMmlAdmin';
import type { GroupTreeNode, GroupTreeCommand } from '@core/types/mmlConsole';
import type { MMLOperationType } from '@core/types/mml';
import { useT } from '@/hooks/useT';

// 2026-05-27 mml-admin-catalog-redesign-20260527 §4.2-4.3：
// 右栏顶部命令元数据 Descriptions（显示态）↔ Form（编辑态）内联切换。
// 受 catalogProtected=true 保护的命令禁止进入编辑态。

const OPERATION_TYPES: MMLOperationType[] = ['LST', 'MOD', 'ADD', 'RMV', 'DSP', 'ACT', 'DEA', 'RST', 'CLR', 'UPG'];

export interface CommandMetaSectionProps {
  command: GroupTreeCommand;
  /** 命令所在分组,用于显示 + 切换分组的下拉 */
  parentGroupId: string;
  groupOptions: GroupTreeNode[];
  /** 当前是否处于编辑态（由父组件 lift up,支持 dirty 检测） */
  editing: boolean;
  onEditingChange: (editing: boolean) => void;
  onDirtyChange: (dirty: boolean) => void;
  onDeleteRequest: () => void;
  onSaved?: () => void;
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

function toFormValues(cmd: GroupTreeCommand, groupId: string): FormValues {
  return {
    groupId,
    commandCode: cmd.commandCode,
    logicalCode: cmd.logicalCode,
    operationType: cmd.operationType,
    displayNameZh:
      cmd.logicalNameI18n?.['zh-CN'] ?? cmd.displayName ?? '',
    displayNameEn:
      cmd.logicalNameI18n?.['en-US'] ?? '',
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
  onDeleteRequest,
  onSaved,
}: CommandMetaSectionProps): React.ReactElement {
  const t = useT();
  const [form] = Form.useForm<FormValues>();
  const updateMut = useUpdateCommand();
  const [submitting, setSubmitting] = useState(false);

  const isProtected = command.catalogProtected === true;

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
      await updateMut.mutateAsync({
        id: command.id,
        req: {
          groupId: values.groupId,
          commandCode: values.commandCode,
          logicalCode: values.logicalCode,
          operationType: values.operationType,
          commandNameI18n: {
            'zh-CN': values.displayNameZh,
            'en-US': values.displayNameEn,
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
            label={t('mml.admin.catalog.commands.group')}
            rules={[{ required: true }]}
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
            label="command_code"
            rules={[
              { required: true },
              { pattern: /^[A-Z0-9_]+$/, message: 'UPPER_SNAKE_CASE only' },
            ]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="logicalCode"
            label="logical_code"
            rules={[{ required: true }]}
          >
            <Input />
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
            <Input />
          </Form.Item>
          <Form.Item
            name="displayNameEn"
            label={t('mml.admin.catalog.commands.displayNameEn')}
            rules={[{ required: true }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="targetObject"
            label={t('mml.admin.catalog.commands.targetObject')}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="requireConfirm"
            label={t('mml.admin.catalog.commands.requireConfirm')}
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
  return (
    <div style={{ marginBottom: 16 }}>
      <Space
        style={{ width: '100%', justifyContent: 'space-between', marginBottom: 12 }}
        align="start"
      >
        <h3 style={{ margin: 0 }}>{command.displayName}</h3>
        <Space>
          {isProtected ? (
            <Tooltip title={t('mml.admin.catalog.common.lockedTooltip')}>
              <Button icon={<EditOutlined />} disabled>
                {t('mml.admin.catalog.commands.edit')}
              </Button>
            </Tooltip>
          ) : (
            <Button
              icon={<EditOutlined />}
              onClick={() => onEditingChange(true)}
            >
              {t('mml.admin.catalog.commands.edit')}
            </Button>
          )}
          {isProtected ? (
            <Tooltip title={t('mml.admin.catalog.common.lockedTooltip')}>
              <Button icon={<DeleteOutlined />} danger disabled>
                {t('mml.admin.catalog.common.delete')}
              </Button>
            </Tooltip>
          ) : (
            <Button icon={<DeleteOutlined />} danger onClick={onDeleteRequest}>
              {t('mml.admin.catalog.common.delete')}
            </Button>
          )}
        </Space>
      </Space>
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
        <Descriptions.Item label="source">
          <Tag color={command.source === 'admin' ? 'blue' : 'default'}>
            {command.source ?? 'standard'}
          </Tag>
        </Descriptions.Item>
        <Descriptions.Item label="catalog_protected">
          {isProtected ? <Tag color="default">locked</Tag> : '-'}
        </Descriptions.Item>
      </Descriptions>
    </div>
  );
}
