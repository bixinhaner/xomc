import { useMemo, useCallback } from 'react';
import { Tree, Empty, Button, Dropdown, Tooltip } from 'antd';
import {
  FolderOutlined,
  CodeOutlined,
  EllipsisOutlined,
  EditOutlined,
  DeleteOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import type { TreeDataNode, MenuProps } from 'antd';
import type { GroupTreeNode, GroupTreeCommand } from '@core/types/mmlConsole';
import { useT } from '@/hooks/useT';

// 2026-05-27 mml-admin-catalog-redesign-20260527 §3：左栏导航树。
// 渲染顶层 group 及其 children/commands；group children 用于展示二级标签。
// 命令节点行内含 编辑/删除 链接（hover 浮现）；分组节点用 [⋯] dropdown 暴露
// 编辑 / 新增命令 / 删除（删除受 commands.length===0 条件约束）。

export type GroupAction = 'editGroup' | 'addCommand' | 'deleteGroup';
export type CommandAction = 'editCommand' | 'deleteCommand';

export interface LeftNavTreeProps {
  groups: GroupTreeNode[];
  selectedKey: string | null;
  expandedKeys: string[];
  search: string;
  onSelect: (key: string) => void;
  onExpand: (keys: string[]) => void;
  onGroupAction: (action: GroupAction, group: GroupTreeNode) => void;
  onCommandAction: (action: CommandAction, command: GroupTreeCommand) => void;
  /**
   * 追加到分组节点之后的额外顶层节点（mml-console-redesign-20260603：
   * Customized「自定义命令」子树）。搜索激活时不参与 group 过滤，恒显示在树底部，
   * 让用户始终能进入 PrivateTemplate / PublicTemplate 增删改。
   */
  extraNodes?: TreeDataNode[];
}

/** 仅挑出根分组；根分组下面的 children 会递归渲染。 */
function filterTopLevelGroups(groups: GroupTreeNode[]): GroupTreeNode[] {
  return groups.filter((g) => !g.path.includes('.'));
}

/** 在分组及其命令中查找包含搜索词的子集。 */
function filterBySearch(
  groups: GroupTreeNode[],
  search: string,
): { groups: GroupTreeNode[]; expandedHits: string[] } {
  if (!search.trim()) return { groups, expandedHits: [] };
  const lower = search.toLowerCase();
  const hits: GroupTreeNode[] = [];
  const expandedHits: string[] = [];

  const visit = (g: GroupTreeNode): GroupTreeNode | null => {
    const groupHit =
      g.displayName.toLowerCase().includes(lower) ||
      g.groupCode.toLowerCase().includes(lower);
    const matchedCmds = (g.commands ?? []).filter(
      (c) =>
        c.displayName.toLowerCase().includes(lower) ||
        c.commandCode.toLowerCase().includes(lower) ||
        c.logicalCode.toLowerCase().includes(lower),
    );
    const matchedChildren = (g.children ?? [])
      .map(visit)
      .filter((child): child is GroupTreeNode => Boolean(child));

    if (groupHit || matchedCmds.length > 0 || matchedChildren.length > 0) {
      expandedHits.push(`group:${g.id}`);
      return {
        ...g,
        commands: groupHit ? g.commands : matchedCmds,
        children: groupHit ? g.children : matchedChildren,
      };
    }
    return null;
  };

  groups.forEach((g) => {
    const hit = visit(g);
    if (hit) hits.push(hit);
  });
  return { groups: hits, expandedHits };
}

const OPERATION_ORDER: Record<string, number> = {
  LST: 1,
  MOD: 2,
  ADD: 3,
  RMV: 4,
};

const DISPLAY_VERB_PREFIX = /^(查询|修改|添加|新增|删除|Query|Modify|Add|Delete)\s+/i;

function operationOrder(op: string): number {
  return OPERATION_ORDER[op] ?? 99;
}

function commandSectionKey(command: GroupTreeCommand): string {
  return (
    command.logicalCode ||
    command.logicalName ||
    command.displayName.replace(DISPLAY_VERB_PREFIX, '') ||
    command.commandCode
  );
}

function compareCommandsBySection(a: GroupTreeCommand, b: GroupTreeCommand): number {
  const sectionCompare = commandSectionKey(a).localeCompare(
    commandSectionKey(b),
    'zh-CN',
  );
  if (sectionCompare !== 0) return sectionCompare;

  const opCompare = operationOrder(a.operationType) - operationOrder(b.operationType);
  if (opCompare !== 0) return opCompare;

  return a.commandCode.localeCompare(b.commandCode);
}

export default function LeftNavTree({
  groups,
  selectedKey,
  expandedKeys,
  search,
  onSelect,
  onExpand,
  onGroupAction,
  onCommandAction,
  extraNodes,
}: LeftNavTreeProps): React.ReactElement {
  const t = useT();

  // 从根分组开始渲染；子分组保留在 children 中递归展示。
  const topGroups = useMemo(() => filterTopLevelGroups(groups), [groups]);
  const { groups: visibleGroups, expandedHits } = useMemo(
    () => filterBySearch(topGroups, search),
    [topGroups, search],
  );

  // 搜索激活时强制展开命中分组（不覆盖用户手动展开状态当无搜索时）
  const effectiveExpanded = useMemo(() => {
    if (!search.trim()) return expandedKeys;
    return Array.from(new Set([...expandedKeys, ...expandedHits]));
  }, [expandedKeys, expandedHits, search]);

  const buildGroupMenu = useCallback(
    (g: GroupTreeNode): MenuProps['items'] => {
      // 2026-05-27 用户决策:取消 catalog_protected 在 UI 上的锁定逻辑。
      // 权限由路由层 RBAC + 后端 API 校验决定,前端不再因 catalogProtected 拦截。
      // 删除按钮仍保留"分组含命令不能删除"约束(后端 ErrGroupNotEmpty 兜底)。
      const hasCommands =
        (g.commands?.length ?? 0) > 0 || (g.children?.length ?? 0) > 0;
      return [
        {
          key: 'editGroup',
          label: t('mml.admin.catalog.groups.editGroup'),
          icon: <EditOutlined />,
        },
        {
          key: 'addCommand',
          label: t('mml.admin.catalog.groups.addCommand'),
          icon: <PlusOutlined />,
        },
        { type: 'divider' },
        {
          key: 'deleteGroup',
          label: hasCommands ? (
            <Tooltip title={t('mml.admin.catalog.groups.deleteDisabledTip')}>
              <span style={{ color: '#bfbfbf' }}>
                {t('mml.admin.catalog.common.delete')}
              </span>
            </Tooltip>
          ) : (
            t('mml.admin.catalog.common.delete')
          ),
          icon: <DeleteOutlined />,
          disabled: hasCommands,
          danger: !hasCommands,
        },
      ];
    },
    [t],
  );

  const groupTreeData = useMemo<TreeDataNode[]>(() => {
    const sortedGroups = [...visibleGroups].sort(
      (a, b) => a.displayOrder - b.displayOrder,
    );
    const buildCommandNode = (c: GroupTreeCommand): TreeDataNode => ({
      key: `cmd:${c.id}`,
      title: (
        <span
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 6,
            width: '100%',
          }}
        >
          <CodeOutlined />
          <span
            style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis' }}
          >
            {c.displayName}
          </span>
          <Tooltip title={t('mml.admin.catalog.commands.edit')}>
            <a
              onClick={(e) => {
                e.stopPropagation();
                onCommandAction('editCommand', c);
              }}
              style={{ fontSize: 14, display: 'inline-flex' }}
            >
              <EditOutlined />
            </a>
          </Tooltip>
          <Tooltip title={t('mml.admin.catalog.common.delete')}>
            <a
              onClick={(e) => {
                e.stopPropagation();
                onCommandAction('deleteCommand', c);
              }}
              style={{ fontSize: 14, color: '#ff4d4f', display: 'inline-flex' }}
            >
              <DeleteOutlined />
            </a>
          </Tooltip>
        </span>
      ),
      isLeaf: true,
    });

    const buildGroupNode = (g: GroupTreeNode): TreeDataNode => {
      const commands = [...(g.commands ?? [])].sort(compareCommandsBySection);
      const childGroups = [...(g.children ?? [])].sort(
        (a, b) => a.displayOrder - b.displayOrder,
      );
      return {
        key: `group:${g.id}`,
        title: (
          <span
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 6,
              width: '100%',
            }}
          >
            <FolderOutlined />
            <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis' }}>
              {g.displayName}
            </span>
            <Dropdown
              menu={{
                items: buildGroupMenu(g),
                onClick: ({ key, domEvent }) => {
                  domEvent.stopPropagation();
                  onGroupAction(key as GroupAction, g);
                },
              }}
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
        children: [
          ...childGroups.map(buildGroupNode),
          ...commands.map(buildCommandNode),
        ],
      };
    };

    return sortedGroups.map(buildGroupNode);
  }, [visibleGroups, buildGroupMenu, onGroupAction, onCommandAction, t]);

  // Customized 子树恒追加在分组之后；搜索时不参与 group 过滤（始终可达增删改入口）。
  const treeData = useMemo<TreeDataNode[]>(
    () => [...groupTreeData, ...(extraNodes ?? [])],
    [groupTreeData, extraNodes],
  );

  if (treeData.length === 0) {
    return <Empty description={search.trim() ? 'No match' : undefined} />;
  }

  return (
    <Tree
      treeData={treeData}
      selectedKeys={selectedKey ? [selectedKey] : []}
      expandedKeys={effectiveExpanded}
      onExpand={(keys) => onExpand(keys as string[])}
      onSelect={(keys) => {
        if (keys.length > 0) onSelect(String(keys[0]));
      }}
      blockNode
      showLine={{ showLeafIcon: false }}
    />
  );
}
