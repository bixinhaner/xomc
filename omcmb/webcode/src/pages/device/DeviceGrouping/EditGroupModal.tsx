import { useMemo } from 'react';
import { Form, Input, Modal, TreeSelect } from 'antd';
import type { FormInstance } from 'antd';
import type { GroupItem } from './types';

export interface EditGroupModalProps {
  open: boolean;
  form: FormInstance<{ name: string; parentId?: string; description: string }>;
  groups: GroupItem[];
  editingGroupId?: string;
  onOk: () => void;
  onCancel: () => void;
  t: (id: string, values?: Record<string, unknown>) => string;
}

/**
 * 编辑 L1 分组 Modal。拆分自 GroupDialogs.tsx 以满足单文件 ≤ 400 行。
 */
export default function EditGroupModal({
  open,
  form,
  groups,
  editingGroupId,
  onOk,
  onCancel,
  t,
}: EditGroupModalProps) {
  // 编辑时构建父级分组选择器的树形数据（排除当前编辑的分组，防止循环引用）
  const parentGroupTreeData = useMemo(() => {
    const rootGroups = (groups || []).filter((g) => !g.parentId && g.id !== editingGroupId);
    return rootGroups.map((g) => ({
      value: g.id,
      title: g.name,
      key: g.id,
    }));
  }, [groups, editingGroupId]);

  return (
    <Modal
      title={t('common.edit')}
      open={open}
      onOk={onOk}
      onCancel={onCancel}
      okText={t('common.save')}
    >
      <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
        <Form.Item
          name="name"
          label={t('table.name')}
          rules={[{ required: true, message: t('common.placeholder') }]}
        >
          <Input placeholder={t('common.placeholder')} />
        </Form.Item>
        <Form.Item
          name="parentId"
          label={t('device.parentGroup')}
          tooltip={t('device.parentGroupEditTooltip')}
        >
          <TreeSelect
            treeData={parentGroupTreeData}
            placeholder={t('device.selectParentGroup')}
            allowClear
            showSearch
            treeNodeFilterProp="title"
            treeDefaultExpandAll
          />
        </Form.Item>
        <Form.Item name="description" label={t('table.description')}>
          <Input.TextArea rows={3} placeholder={t('common.placeholder')} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
