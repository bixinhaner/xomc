import { useMemo, useState, useCallback } from 'react';
import { Tree, Empty, Spin, Alert, Button, Dropdown, Modal, message, Space, Tag, Tooltip } from 'antd';
import {
  FolderOutlined,
  CodeOutlined,
  PlusOutlined,
  EllipsisOutlined,
  LockOutlined,
} from '@ant-design/icons';
import type { TreeDataNode, MenuProps } from 'antd';
import type { Key } from 'react';
import { useGroupTree } from '@core/hooks/api/useMmlConsole';
import type { GroupTreeNode } from '@core/types/mmlConsole';
import { useDeleteGroup, useUpdateGroup } from '@core/hooks/api/useMmlAdmin';
import { useT } from '@/hooks/useT';
import GroupEditorModal, { type GroupEditorMode } from './GroupEditorModal';

interface GroupTargetForEdit {
  id: string;
  groupCode: string;
  displayNameI18n: Record<string, string>;
  displayOrder: number;
}

interface EditState {
  open: boolean;
  mode: GroupEditorMode;
  target?: GroupTargetForEdit;
}

const initialEdit: EditState = { open: false, mode: 'create-root' };

function findFlatGroupById(nodes: GroupTreeNode[], id: string): GroupTreeNode | null {
  for (const g of nodes) {
    if (g.id === id) return g;
    const inChild = findFlatGroupById(g.children ?? [], id);
    if (inChild) return inChild;
  }
  return null;
}

function buildAdminTreeData(
  nodes: GroupTreeNode[],
  onAction: (key: string, group: GroupTreeNode) => void,
  t: (id: string) => string,
): TreeDataNode[] {
  const sorted = [...nodes].sort((a, b) => a.displayOrder - b.displayOrder);
  return sorted.map((g) => {
    const locked = g.catalogProtected === true;
    const items: MenuProps['items'] = [
      { key: 'addChild', label: t('mml.admin.catalog.groups.addChild') },
      { key: 'rename', label: t('mml.admin.catalog.groups.rename'), disabled: locked },
      { type: 'divider' },
      { key: 'delete', label: t('mml.admin.catalog.groups.delete'), danger: true, disabled: locked },
    ];
    return {
      key: `group:${g.id}`,
      title: (
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
          <FolderOutlined />
          <span>{g.displayName}</span>
          {locked && (
            <Tooltip title={t('mml.admin.catalog.common.lockedTooltip')}>
              <LockOutlined style={{ color: '#999' }} />
            </Tooltip>
          )}
          <Tag style={{ marginLeft: 2 }}>{g.path}</Tag>
          <Dropdown
            menu={{ items, onClick: ({ key, domEvent }) => { domEvent.stopPropagation(); onAction(key, g); } }}
            trigger={['click']}
          >
            <Button
              type="text"
              size="small"
              icon={<EllipsisOutlined />}
              onClick={(e) => e.stopPropagation()}
            />
          </Dropdown>
        </span>
      ),
      selectable: false,
      children: [
        ...buildAdminTreeData(g.children ?? [], onAction, t),
        ...[...(g.commands ?? [])]
          .sort((a, b) => a.displayName.localeCompare(b.displayName))
          .map<TreeDataNode>((c) => ({
            key: `cmd:${c.id}`,
            title: (
              <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
                <CodeOutlined />
                {c.displayName}
              </span>
            ),
            isLeaf: true,
            selectable: false,
          })),
      ],
    };
  });
}

export default function GroupsTab() {
  const t = useT();
  const { data: tree = [], isLoading, refetch } = useGroupTree(undefined, 'zh-CN');
  const updateMut = useUpdateGroup();
  const deleteMut = useDeleteGroup();

  const [edit, setEdit] = useState<EditState>(initialEdit);
  const [expandedKeys, setExpandedKeys] = useState<Key[]>([]);

  const handleAction = useCallback(
    (action: string, g: GroupTreeNode) => {
      // catalog_protected 锁定 rename/delete（addChild 仍允许）
      if (g.catalogProtected && (action === 'rename' || action === 'delete')) {
        message.warning(t('mml.admin.catalog.common.lockedTooltip'));
        return;
      }
      const target: GroupTargetForEdit = {
        id: g.id,
        groupCode: g.groupCode,
        displayNameI18n: g.displayNameI18n ?? {
          'zh-CN': g.displayName,
          'en-US': g.displayName,
        },
        displayOrder: g.displayOrder,
      };
      if (action === 'addChild') {
        setEdit({ open: true, mode: 'create-child', target });
      } else if (action === 'rename') {
        setEdit({ open: true, mode: 'rename', target });
      } else if (action === 'delete') {
        Modal.confirm({
          title: t('mml.admin.catalog.groups.deleteConfirm'),
          okText: t('mml.admin.catalog.common.confirm'),
          okButtonProps: { danger: true },
          cancelText: t('mml.admin.catalog.common.cancel'),
          onOk: async () => {
            try {
              await deleteMut.mutateAsync(g.id);
              message.success(t('mml.admin.catalog.common.deleteSuccess'));
              void refetch();
            } catch (e) {
              if (e instanceof Error) message.error(e.message);
            }
          },
        });
      }
    },
    [deleteMut, refetch, t],
  );

  const treeData = useMemo(
    () => buildAdminTreeData(tree, handleAction, t),
    [tree, handleAction, t],
  );

  const handleDrop = useCallback<NonNullable<React.ComponentProps<typeof Tree>['onDrop']>>(
    async (info) => {
      const dragKey = String(info.dragNode.key);
      if (!dragKey.startsWith('group:')) return;
      const dropKey = String(info.node.key);
      if (!dropKey.startsWith('group:')) return;

      const dragGroupId = dragKey.slice('group:'.length);
      const dropGroupId = dropKey.slice('group:'.length);

      const dragGroup = findFlatGroupById(tree, dragGroupId);
      const dropGroup = findFlatGroupById(tree, dropGroupId);
      if (!dragGroup || !dropGroup) return;

      // dropPosition < 0 → drop 在 node 之前（同级）
      // dropPosition > 0 && dropToGap → drop 在 node 之后（同级）
      // dropToGap=false → drop 成为 node 的 child
      const dropToGap = info.dropToGap;
      let parentId: string | null = null;
      let newOrder = dropGroup.displayOrder;

      if (dropToGap) {
        // 同级：parent 与 drop 节点的 parent 相同
        // GroupTreeNode 不暴露 parentId — 我们退化为：以 drop 节点 displayOrder ± 5 调整 + parent 用 path 推断
        // PRD §Q.3 拖拽是简化语义，本期同级排序为主，跨 parent 移动留 P4 改进
        const path = dropGroup.path;
        const lastDot = path.lastIndexOf('.');
        const parentPath = lastDot > 0 ? path.slice(0, lastDot) : '';
        const parentNode = parentPath
          ? findFlatGroupByPath(tree, parentPath)
          : null;
        parentId = parentNode?.id ?? null;
        newOrder = Math.max(0, dropGroup.displayOrder + (info.dropPosition > 0 ? 5 : -5));
      } else {
        // 成为 child
        parentId = dropGroup.id;
        newOrder = (dropGroup.children?.length ?? 0) * 10 + 10;
      }

      try {
        await updateMut.mutateAsync({
          id: dragGroupId,
          req: { parentId, displayOrder: newOrder },
        });
        message.success(t('mml.admin.catalog.common.saveSuccess'));
        void refetch();
      } catch (e) {
        if (e instanceof Error) message.error(e.message);
      }
    },
    [tree, updateMut, refetch, t],
  );

  if (isLoading) return <Spin />;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <Space>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => setEdit({ open: true, mode: 'create-root' })}
        >
          {t('mml.admin.catalog.groups.addRoot')}
        </Button>
      </Space>
      <Alert type="info" showIcon message={t('mml.admin.catalog.groups.dragHint')} />
      {tree.length === 0 ? (
        <Empty />
      ) : (
        <Tree
          treeData={treeData}
          draggable={{ icon: false, nodeDraggable: (n) => String(n.key).startsWith('group:') }}
          onDrop={handleDrop}
          expandedKeys={expandedKeys}
          onExpand={setExpandedKeys}
          showLine
        />
      )}
      <GroupEditorModal
        open={edit.open}
        mode={edit.mode}
        targetGroup={edit.target}
        onClose={() => setEdit(initialEdit)}
        onSuccess={() => void refetch()}
      />
    </div>
  );
}

function findFlatGroupByPath(nodes: GroupTreeNode[], path: string): GroupTreeNode | null {
  for (const g of nodes) {
    if (g.path === path) return g;
    const inChild = findFlatGroupByPath(g.children ?? [], path);
    if (inChild) return inChild;
  }
  return null;
}
