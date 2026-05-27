import { useMemo, useState, useCallback } from 'react';
import {
  Card,
  Button,
  Input,
  Space,
  Spin,
  Modal,
  message,
} from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { useGroupTree } from '@core/hooks/api/useMmlConsole';
import {
  useDeleteGroup,
  useDeleteCommand,
} from '@core/hooks/api/useMmlAdmin';
import type { GroupTreeNode, GroupTreeCommand } from '@core/types/mmlConsole';
import { useT } from '@/hooks/useT';
import LeftNavTree, {
  type GroupAction,
  type CommandAction,
} from './LeftNavTree';
import RightDetailPanel from './RightDetailPanel';
import GroupEditorModal, { type GroupEditorMode } from './GroupEditorModal';
import CommandEditorModal from './CommandEditorModal';

// 2026-05-27 mml-admin-catalog-redesign-20260527：移除 Tabs，改成左右两栏。
// 选中态语义：
//   selectedKey 形如 "group:<uuid>" 或 "cmd:<uuid>" 或 null
//   切命令前若处于 editing+dirty 状态弹 Modal.confirm

type SelectionKind = 'group' | 'cmd';

function parseSelection(
  key: string | null,
): { kind: SelectionKind; id: string } | null {
  if (!key) return null;
  if (key.startsWith('group:')) return { kind: 'group', id: key.slice(6) };
  if (key.startsWith('cmd:')) return { kind: 'cmd', id: key.slice(4) };
  return null;
}

function findCommand(
  groups: GroupTreeNode[],
  commandId: string,
): { command: GroupTreeCommand; groupId: string } | null {
  for (const g of groups) {
    for (const c of g.commands ?? []) {
      if (c.id === commandId) return { command: c, groupId: g.id };
    }
  }
  return null;
}

function findGroup(
  groups: GroupTreeNode[],
  groupId: string,
): GroupTreeNode | null {
  return groups.find((g) => g.id === groupId) ?? null;
}

export default function MMLAdminCatalog() {
  const t = useT();
  const { data: tree = [], isLoading, refetch, isFetching } = useGroupTree(
    undefined,
    'zh-CN',
  );
  const deleteGroupMut = useDeleteGroup();
  const deleteCommandMut = useDeleteCommand();

  const [selectedKey, setSelectedKey] = useState<string | null>(null);
  const [expandedKeys, setExpandedKeys] = useState<string[]>([]);
  const [search, setSearch] = useState('');

  // 命令编辑态 + dirty 检测（提供给 CommandMetaSection）
  const [editing, setEditing] = useState(false);
  const [dirty, setDirty] = useState(false);

  // Modal 状态
  const [groupEditor, setGroupEditor] = useState<{
    open: boolean;
    mode: GroupEditorMode;
    target?: {
      id: string;
      groupCode: string;
      displayNameI18n: Record<string, string>;
      displayOrder: number;
    };
  }>({ open: false, mode: 'create' });

  const [commandEditor, setCommandEditor] = useState<{
    open: boolean;
    parent: GroupTreeNode | undefined;
  }>({ open: false, parent: undefined });

  // 只渲染顶层分组（一级分组语义）
  const topGroups = useMemo<GroupTreeNode[]>(
    () => tree.filter((g) => !g.path.includes('.')),
    [tree],
  );

  const selection = useMemo(() => parseSelection(selectedKey), [selectedKey]);

  const selectedCommandCtx = useMemo(() => {
    if (selection?.kind !== 'cmd') return null;
    return findCommand(tree, selection.id);
  }, [tree, selection]);

  const selectedGroup = useMemo(() => {
    if (selection?.kind !== 'group') return null;
    return findGroup(topGroups, selection.id);
  }, [topGroups, selection]);

  // 切换命令前的 dirty 检测
  const handleSelect = useCallback(
    (key: string) => {
      if (editing && dirty) {
        Modal.confirm({
          title: t('mml.admin.catalog.commands.discardDirty'),
          okButtonProps: { danger: true },
          okText: t('mml.admin.catalog.common.confirm'),
          cancelText: t('mml.admin.catalog.common.cancel'),
          onOk: () => {
            setEditing(false);
            setDirty(false);
            setSelectedKey(key);
          },
        });
        return;
      }
      setSelectedKey(key);
      // 切换到新命令时退出编辑态
      setEditing(false);
    },
    [editing, dirty, t],
  );

  // ---------- 分组操作 ----------

  const handleGroupAction = useCallback(
    (action: GroupAction, group: GroupTreeNode) => {
      if (action === 'editGroup') {
        setGroupEditor({
          open: true,
          mode: 'rename',
          target: {
            id: group.id,
            groupCode: group.groupCode,
            displayNameI18n: group.displayNameI18n ?? {
              'zh-CN': group.displayName,
              'en-US': group.displayName,
            },
            displayOrder: group.displayOrder,
          },
        });
      } else if (action === 'addCommand') {
        setCommandEditor({ open: true, parent: group });
      } else if (action === 'deleteGroup') {
        Modal.confirm({
          title: t('mml.admin.catalog.groups.deleteGroupConfirm', {
            name: group.displayName,
          }),
          okButtonProps: { danger: true },
          okText: t('mml.admin.catalog.common.confirm'),
          cancelText: t('mml.admin.catalog.common.cancel'),
          onOk: async () => {
            try {
              await deleteGroupMut.mutateAsync(group.id);
              message.success(t('mml.admin.catalog.common.deleteSuccess'));
              // 若被删分组当前选中,清空选中
              if (selection?.kind === 'group' && selection.id === group.id) {
                setSelectedKey(null);
              }
              void refetch();
            } catch (e) {
              // 后端 ErrGroupNotEmpty → 409,提示统一文案
              const msg =
                e instanceof Error ? e.message : 'delete group failed';
              if (msg.toLowerCase().includes('group_not_empty') || msg.includes('409')) {
                message.warning(
                  t('mml.admin.catalog.groups.deleteDisabledTip'),
                );
              } else {
                message.error(msg);
              }
            }
          },
        });
      }
    },
    [deleteGroupMut, refetch, selection, t],
  );

  // ---------- 命令操作 ----------

  const handleDeleteCommand = useCallback(
    (cmd: GroupTreeCommand) => {
      Modal.confirm({
        title: t('mml.admin.catalog.commands.deleteConfirm', {
          name: cmd.displayName,
        }),
        okButtonProps: { danger: true },
        okText: t('mml.admin.catalog.common.confirm'),
        cancelText: t('mml.admin.catalog.common.cancel'),
        onOk: async () => {
          try {
            await deleteCommandMut.mutateAsync(cmd.id);
            message.success(t('mml.admin.catalog.common.deleteSuccess'));
            if (selection?.kind === 'cmd' && selection.id === cmd.id) {
              setSelectedKey(null);
              setEditing(false);
              setDirty(false);
            }
            void refetch();
          } catch (e) {
            if (e instanceof Error) message.error(e.message);
          }
        },
      });
    },
    [deleteCommandMut, refetch, selection, t],
  );

  const handleCommandAction = useCallback(
    (action: CommandAction, command: GroupTreeCommand) => {
      if (action === 'editCommand') {
        // 选中并进入编辑态
        const newKey = `cmd:${command.id}`;
        // 如果当前在另一命令的 dirty 状态,handleSelect 会拦截
        if (selectedKey !== newKey) {
          if (editing && dirty) {
            Modal.confirm({
              title: t('mml.admin.catalog.commands.discardDirty'),
              okButtonProps: { danger: true },
              okText: t('mml.admin.catalog.common.confirm'),
              cancelText: t('mml.admin.catalog.common.cancel'),
              onOk: () => {
                setEditing(false);
                setDirty(false);
                setSelectedKey(newKey);
                setTimeout(() => setEditing(true), 0);
              },
            });
            return;
          }
          setSelectedKey(newKey);
          setTimeout(() => setEditing(true), 0);
        } else {
          setEditing(true);
        }
      } else if (action === 'deleteCommand') {
        handleDeleteCommand(command);
      }
    },
    [selectedKey, editing, dirty, handleDeleteCommand, t],
  );

  // ---------- 视图 ----------

  if (isLoading) {
    return (
      <Card variant="borderless" title={t('mml.admin.catalog.title')}>
        <Spin />
      </Card>
    );
  }

  return (
    <Card
      variant="borderless"
      title={t('mml.admin.catalog.title')}
      styles={{ body: { padding: 16 } }}
    >
      <Space style={{ width: '100%', marginBottom: 12 }} size="middle">
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() =>
            setGroupEditor({ open: true, mode: 'create' })
          }
        >
          {t('mml.admin.catalog.groups.add')}
        </Button>
        <Input.Search
          placeholder={t('mml.admin.catalog.groups.searchPlaceholder')}
          allowClear
          style={{ width: 320 }}
          onSearch={setSearch}
          onChange={(e) => !e.target.value && setSearch('')}
        />
        <Button
          icon={<ReloadOutlined />}
          loading={isFetching && !isLoading}
          onClick={() => void refetch()}
        />
      </Space>

      <div
        style={{
          display: 'flex',
          gap: 16,
          minHeight: 'calc(100vh - 280px)',
          alignItems: 'stretch',
        }}
      >
        <div
          style={{
            width: 320,
            flexShrink: 0,
            borderRight: '1px solid #f0f0f0',
            paddingRight: 12,
            overflowY: 'auto',
          }}
        >
          <LeftNavTree
            groups={topGroups}
            selectedKey={selectedKey}
            expandedKeys={expandedKeys}
            search={search}
            onSelect={handleSelect}
            onExpand={setExpandedKeys}
            onGroupAction={handleGroupAction}
            onCommandAction={handleCommandAction}
          />
        </div>
        <div style={{ flex: 1, overflowY: 'auto' }}>
          <RightDetailPanel
            command={selectedCommandCtx?.command}
            parentGroupId={selectedCommandCtx?.groupId}
            groupOptions={topGroups}
            selectedGroup={selectedGroup ?? undefined}
            editing={editing}
            onEditingChange={setEditing}
            onDirtyChange={setDirty}
            onDeleteCommand={() => {
              if (selectedCommandCtx) {
                handleDeleteCommand(selectedCommandCtx.command);
              }
            }}
          />
        </div>
      </div>

      <GroupEditorModal
        open={groupEditor.open}
        mode={groupEditor.mode}
        targetGroup={groupEditor.target}
        onClose={() => setGroupEditor({ open: false, mode: 'create' })}
        onSuccess={() => void refetch()}
      />
      <CommandEditorModal
        open={commandEditor.open}
        parentGroup={commandEditor.parent}
        groupOptions={topGroups}
        onClose={() => setCommandEditor({ open: false, parent: undefined })}
        onSuccess={(newCommandId) => {
          void refetch();
          setSelectedKey(`cmd:${newCommandId}`);
          if (commandEditor.parent) {
            setExpandedKeys((prev) =>
              prev.includes(`group:${commandEditor.parent!.id}`)
                ? prev
                : [...prev, `group:${commandEditor.parent!.id}`],
            );
          }
        }}
      />
    </Card>
  );
}
