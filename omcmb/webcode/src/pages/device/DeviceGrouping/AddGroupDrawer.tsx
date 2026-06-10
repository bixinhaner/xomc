import { useMemo } from 'react';
import { Button, Drawer, Form, Input, TreeSelect } from 'antd';
import type { FormInstance } from 'antd';
import type { GroupItem } from './types';

export interface AddGroupDrawerProps {
  open: boolean;
  form: FormInstance<{
    name?: string;
    description?: string;
    parentId?: string;
  }>;
  groups: GroupItem[];
  onOk: () => void;
  onCancel: () => void;
  t: (id: string, values?: Record<string, string | number>) => string;
}

/**
 * 新增 L1 分组 Drawer。拆分自 GroupDialogs.tsx 以满足单文件 ≤ 400 行。
 */
export default function AddGroupDrawer({ open, form, groups, onOk, onCancel, t }: AddGroupDrawerProps) {
  const parentGroupTreeData = useMemo(() => {
    const rootGroups = (groups || []).filter((g) => !g.parentId);
    return rootGroups.map((g) => ({
      value: g.id,
      title: g.name,
      key: g.id,
    }));
  }, [groups]);

  return (
    <Drawer
      title={t('device.addGroup')}
      open={open}
      onClose={onCancel}
      size={520}
      destroyOnHidden
      footer={
        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
          <Button onClick={onCancel}>{t('common.cancel')}</Button>
          <Button type="primary" onClick={onOk}>
            {t('common.confirm')}
          </Button>
        </div>
      }
    >
      <Form form={form} layout="vertical">
        {/* Basic info */}
        <div style={{
          padding: '16px',
          background: 'var(--color-fill-quaternary)',
          borderRadius: 8,
          marginBottom: 16,
        }}>
          <div style={{ marginBottom: 12, fontWeight: 500, color: 'var(--color-text)' }}>
            {t('device.basicInfo')}
          </div>
          <Form.Item
            name="name"
            label={t('table.name')}
            required
            rules={[{ required: true, message: t('table.name') }]}
            style={{ marginBottom: 12 }}
          >
            <Input maxLength={50} placeholder={t('table.name')} />
          </Form.Item>
          <Form.Item
            name="parentId"
            label={t('device.parentGroup')}
            tooltip={t('device.parentGroupTooltip')}
            style={{ marginBottom: 0 }}
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
        </div>

        {/* Description */}
        <div style={{
          padding: '16px',
          background: 'var(--color-fill-quaternary)',
          borderRadius: 8,
        }}>
          <div style={{ marginBottom: 12, fontWeight: 500, color: 'var(--color-text)' }}>
            {t('table.description')}
          </div>
          <Form.Item name="description" style={{ marginBottom: 0 }}>
            <Input.TextArea rows={3} maxLength={200} placeholder={t('table.description')} />
          </Form.Item>
        </div>
      </Form>
    </Drawer>
  );
}
