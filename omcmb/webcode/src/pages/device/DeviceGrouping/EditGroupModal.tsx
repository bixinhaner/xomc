import { useMemo } from 'react';
import { Form, Modal, TreeSelect } from 'antd';
import type { FormInstance } from 'antd';
import type { GroupItem } from './types';
import I18nInput from '@/components/I18nInput';

export interface EditGroupModalProps {
  open: boolean;
  form: FormInstance<{
    name_i18n?: Record<string, string>;
    description_i18n?: Record<string, string>;
    parentId?: string;
  }>;
  groups: GroupItem[];
  editingGroupId?: string;
  onOk: () => void;
  onCancel: () => void;
  t: (id: string, values?: Record<string, string | number>) => string;
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
        <Form.Item label={t('table.name')} required>
          <I18nInput name="name_i18n" required maxLength={128} />
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
        <Form.Item label={t('table.description')}>
          <I18nInput name="description_i18n" required={false} textarea rows={3} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
