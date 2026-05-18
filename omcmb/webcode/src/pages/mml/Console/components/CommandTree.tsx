import { useState, useMemo, useCallback } from 'react';
import type { Key, ReactNode } from 'react';
import { Input, Tree, Empty, Spin, message } from 'antd';
import { SearchOutlined, FolderOutlined, CodeOutlined, UserOutlined } from '@ant-design/icons';
import type { TreeDataNode } from 'antd';
import { useQueryClient, useQuery } from '@tanstack/react-query';
import { useGroupTree } from '@core/hooks/api/useMmlConsole';
import { mmlApi } from '@core/services/api/mmlApi';
import { useMmlConsoleStore } from '@core/store/mmlConsoleStore';
import type {
  GroupTreeNode,
  GroupTreeCommand,
  Statement,
  SubFieldDef,
} from '@core/types/mmlConsole';
import type { MMLCustomCommand } from '@core/types/mml';
import { useT } from '@/hooks/useT';

const GROUP_KEY_PREFIX = 'group:';
const CMD_KEY_PREFIX = 'cmd:';
const CUSTOM_KEY_PREFIX = 'custom:';

/**
 * 把后端返回的 N 层 LTREE 拍平为「一级大类 → 命令」两层。
 *
 * 用户决策（2026-05-18）："命令分组只保留一层，完全参考老 OMC 命令树"。
 * 后端 `mml_param_groups` 字典仍允许多层（admin 维护用），但 console 前端
 * 只展示**根节点 = 一级大类**，把所有后代命令收集挂到同一个根下。
 *
 * 例：原结构「设备信息 → 设备基础信息 → 基础查询」此处压平为「设备信息 → 基础查询」。
 */
function collectAllCommands(group: GroupTreeNode): GroupTreeCommand[] {
  const out: GroupTreeCommand[] = [...(group.commands ?? [])];
  for (const child of group.children ?? []) {
    out.push(...collectAllCommands(child));
  }
  return out;
}

function buildTreeData(nodes: GroupTreeNode[]): TreeDataNode[] {
  const sorted = [...nodes].sort((a, b) => a.displayOrder - b.displayOrder);
  return sorted.map((g) => ({
    key: `${GROUP_KEY_PREFIX}${g.id}`,
    title: (
      <span>
        <FolderOutlined style={{ marginRight: 4 }} />
        {g.displayName}
      </span>
    ),
    selectable: false,
    children: collectAllCommands(g)
      .sort((a, b) => a.displayName.localeCompare(b.displayName))
      .map<TreeDataNode>((c) => ({
        key: `${CMD_KEY_PREFIX}${c.id}`,
        title: (
          <span>
            <CodeOutlined style={{ marginRight: 4 }} />
            {c.displayName}
          </span>
        ),
        isLeaf: true,
      })),
  }));
}

function collectMatches(
  nodes: GroupTreeNode[],
  needle: string,
): { matchedKeys: Set<string>; expandKeys: string[] } {
  const matchedKeys = new Set<string>();
  const expandKeys: string[] = [];
  const lower = needle.toLowerCase();
  const walk = (arr: GroupTreeNode[], ancestors: string[]) => {
    arr.forEach((g) => {
      const groupKey = `${GROUP_KEY_PREFIX}${g.id}`;
      const path = [...ancestors, groupKey];
      (g.commands ?? []).forEach((c) => {
        if (c.displayName.toLowerCase().includes(lower)) {
          matchedKeys.add(`${CMD_KEY_PREFIX}${c.id}`);
          path.forEach((k) => {
            if (!expandKeys.includes(k)) expandKeys.push(k);
          });
        }
      });
      if (g.children?.length) walk(g.children, path);
    });
  };
  walk(nodes, []);
  return { matchedKeys, expandKeys };
}

function flattenCommandsById(nodes: GroupTreeNode[]): Map<string, GroupTreeCommand> {
  const map = new Map<string, GroupTreeCommand>();
  const walk = (arr: GroupTreeNode[]) => {
    arr.forEach((g) => {
      (g.commands ?? []).forEach((c) => map.set(c.id, c));
      if (g.children?.length) walk(g.children);
    });
  };
  walk(nodes);
  return map;
}

/**
 * 构造 Customized 子树（PrivateTemplate + PublicTemplate）— T-0123-P4 集成。
 * 结构：Customized > Private (admin/wangyunqi/…)> template / Public > template
 */
function buildCustomTreeData(
  customs: MMLCustomCommand[],
  t: (id: string) => string,
): TreeDataNode | null {
  if (customs.length === 0) return null;

  const privateGroup: MMLCustomCommand[] = [];
  const publicGroup: MMLCustomCommand[] = [];
  customs.forEach((c) => {
    (c.commandScope === 'public' ? publicGroup : privateGroup).push(c);
  });

  const byCreator = new Map<string, MMLCustomCommand[]>();
  privateGroup.forEach((c) => {
    const key = c.creator || 'unknown';
    const list = byCreator.get(key) ?? [];
    list.push(c);
    byCreator.set(key, list);
  });

  const renderLeaf = (cc: MMLCustomCommand): TreeDataNode => ({
    key: `${CUSTOM_KEY_PREFIX}${cc.id}`,
    title: (
      <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
        <CodeOutlined style={{ color: cc.commandScope === 'public' ? '#52c41a' : '#fa8c16' }} />
        {cc.commandName}
      </span>
    ),
    isLeaf: true,
  });

  const privateChildren: TreeDataNode[] = Array.from(byCreator.entries()).map(
    ([creator, items]) => ({
      key: `${CUSTOM_KEY_PREFIX}user:${creator}`,
      title: (
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
          <UserOutlined />
          {creator} ({items.length})
        </span>
      ),
      selectable: false,
      children: items.sort((a, b) => a.commandName.localeCompare(b.commandName)).map(renderLeaf),
    }),
  );

  const children: TreeDataNode[] = [];
  if (privateGroup.length > 0) {
    children.push({
      key: `${CUSTOM_KEY_PREFIX}root:private`,
      title: (
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
          <FolderOutlined style={{ color: '#fa8c16' }} />
          PrivateTemplate ({privateGroup.length})
        </span>
      ),
      selectable: false,
      children: privateChildren,
    });
  }
  if (publicGroup.length > 0) {
    children.push({
      key: `${CUSTOM_KEY_PREFIX}root:public`,
      title: (
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
          <FolderOutlined style={{ color: '#52c41a' }} />
          PublicTemplate ({publicGroup.length})
        </span>
      ),
      selectable: false,
      children: publicGroup
        .sort((a, b) => a.commandName.localeCompare(b.commandName))
        .map(renderLeaf),
    });
  }

  return {
    key: `${CUSTOM_KEY_PREFIX}root`,
    title: (
      <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4, fontWeight: 500 }}>
        <FolderOutlined />
        {t('mml.console.commandTree.customized')}
      </span>
    ),
    selectable: false,
    children,
  };
}

/**
 * MMLCustomCommand → Statement adapter (T-0123-P4)。
 * - parameters JSONB → values: Record<string,string>
 * - 无 catalog sub_fields，selectedSubFieldIds 空 + subFields 空
 * - commandId undefined（Customized 不属 catalog）
 */
function customCommandToStatement(cc: MMLCustomCommand): Statement {
  const values: Record<string, string> = {};
  Object.entries(cc.parameters ?? {}).forEach(([k, v]) => {
    values[k] = typeof v === 'string' ? v : String(v);
  });
  return {
    uid: crypto.randomUUID(),
    commandId: undefined,
    commandCode: cc.commandCode,
    logicalCode: cc.commandCode,
    operationType: cc.operationType,
    logicalNameI18n: {
      'zh-CN': cc.commandName,
      'en-US': cc.commandName,
    },
    subFields: [],
    selectedSubFieldIds: [],
    values,
    unknownCodes: [],
  };
}

export interface CommandTreeProps {
  lang?: 'zh-CN' | 'en-US';
}

export default function CommandTree({ lang }: CommandTreeProps) {
  const t = useT();
  const storeLang = useMmlConsoleStore((s) => s.lang);
  // 用户决策（2026-05-18）：连续点击命令是**覆盖**而非追加。
  // 老 OMC 行为：用户点一个命令 → 控制面板只显示这一条；要多语句脚本走 textbox。
  const replaceStatement = useMmlConsoleStore((s) => s.replaceStatement);
  const effectiveLang = lang ?? storeLang;

  const { data: tree = [], isLoading } = useGroupTree(undefined, effectiveLang);
  const queryClient = useQueryClient();

  // Customized PrivateTemplate / PublicTemplate (T-0123-P4 集成)
  const { data: customResp } = useQuery({
    queryKey: ['mml', 'console', 'custom-commands'],
    queryFn: () => mmlApi.getTemplates({ page: 1, pageSize: 100 }),
    staleTime: 5 * 60 * 1000,
  });
  const customCommands = useMemo(() => customResp?.items ?? [], [customResp]);

  const [searchText, setSearchText] = useState('');
  const [expandedKeys, setExpandedKeys] = useState<Key[]>([]);
  const [autoExpand, setAutoExpand] = useState(true);

  const customById = useMemo(() => {
    const m = new Map<string, MMLCustomCommand>();
    customCommands.forEach((c) => m.set(c.id, c));
    return m;
  }, [customCommands]);

  const treeData = useMemo(() => {
    const groups = buildTreeData(tree);
    const customRoot = buildCustomTreeData(customCommands, t);
    return customRoot ? [...groups, customRoot] : groups;
  }, [tree, customCommands, t]);
  const commandsById = useMemo(() => flattenCommandsById(tree), [tree]);
  const matched = useMemo(() => {
    if (!searchText.trim()) return { matchedKeys: new Set<string>(), expandKeys: [] };
    return collectMatches(tree, searchText.trim());
  }, [searchText, tree]);

  const onSearchChange = useCallback(
    (value: string) => {
      setSearchText(value);
      if (!value.trim()) {
        setExpandedKeys([]);
        setAutoExpand(true);
        return;
      }
      const { expandKeys } = collectMatches(tree, value.trim());
      setExpandedKeys(expandKeys);
      setAutoExpand(false);
    },
    [tree],
  );

  const handleSelect = useCallback(
    async (selectedKeys: Key[]) => {
      const key = selectedKeys[0];
      if (typeof key !== 'string') return;

      // Customized PrivateTemplate / PublicTemplate 加载到右栏（T-0123-P4 adapter）
      if (key.startsWith(CUSTOM_KEY_PREFIX)) {
        const customId = key.slice(CUSTOM_KEY_PREFIX.length);
        const cc = customById.get(customId);
        if (!cc) return;
        replaceStatement(customCommandToStatement(cc));
        return;
      }

      if (!key.startsWith(CMD_KEY_PREFIX)) return;
      const commandId = key.slice(CMD_KEY_PREFIX.length);
      const cmd = commandsById.get(commandId);
      if (!cmd) return;

      try {
        const subFields = await queryClient.fetchQuery<SubFieldDef[]>({
          queryKey: ['mml', 'console', 'sub-fields', commandId, effectiveLang],
          queryFn: () => mmlApi.getCommandSubFields(commandId, effectiveLang),
          staleTime: 30 * 60 * 1000,
        });

        const stmt: Statement = {
          uid: crypto.randomUUID(),
          commandId: cmd.id,
          commandCode: cmd.commandCode,
          logicalCode: cmd.logicalCode,
          operationType: cmd.operationType,
          logicalNameI18n: {
            'zh-CN': cmd.displayName,
            'en-US': cmd.displayName,
          },
          subFields,
          selectedSubFieldIds: subFields
            .filter((sf) => sf.defaultSelected)
            .map((sf) => sf.id),
          values: {},
          unknownCodes: [],
          targetObject: cmd.targetObject,
        };
        replaceStatement(stmt);
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e);
        message.error(msg);
      }
    },
    [replaceStatement, commandsById, customById, effectiveLang, queryClient],
  );

  if (isLoading) {
    return <Spin />;
  }
  if (tree.length === 0) {
    return <Empty description={t('mml.console.commandTree.empty')} />;
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      <Input
        prefix={<SearchOutlined />}
        placeholder={t('mml.console.commandTree.searchPlaceholder')}
        value={searchText}
        onChange={(e) => onSearchChange(e.target.value)}
        allowClear
      />
      <Tree
        treeData={treeData}
        expandedKeys={expandedKeys}
        autoExpandParent={autoExpand}
        onExpand={(keys) => {
          setExpandedKeys(keys);
          setAutoExpand(false);
        }}
        onSelect={handleSelect}
        selectedKeys={[]}
        showLine
        style={{ minHeight: 320, overflow: 'auto' }}
        titleRender={(node) => {
          const key = node.key;
          if (typeof key === 'string' && matched.matchedKeys.has(key)) {
            return <span style={{ background: '#fff2b8' }}>{node.title as ReactNode}</span>;
          }
          return node.title as ReactNode;
        }}
      />
    </div>
  );
}
