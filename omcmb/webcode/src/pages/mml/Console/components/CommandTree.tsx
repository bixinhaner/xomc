import { useState, useMemo, useCallback } from 'react';
import type { Key, ReactNode } from 'react';
import { Input, Tree, Empty, Spin, message, Tooltip, Tag } from 'antd';
import { SearchOutlined, FolderOutlined, CodeOutlined, UserOutlined, PlusOutlined } from '@ant-design/icons';
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
import type { MMLCustomCommand, MMLOperationType } from '@core/types/mml';
import { useT } from '@/hooks/useT';
import AddTemplateModal from './AddTemplateModal';

// R-2: per-op 命令树叶子统一前缀 `<OP> <Object>`。OpTagColor 按操作语义着色，
// 便于用户在密集命令列表中快速辨别 LST(读) / MOD(改) / ADD(增) / RMV(删) 的危险等级。
const OP_TAG_COLOR: Record<string, string> = {
  LST: 'blue',
  MOD: 'orange',
  ADD: 'green',
  RMV: 'red',
};

// 解析 backend displayName。后端历史格式为 "设备信息(LST DEVICE_INFO)" — 把括号尾
// 部去掉只保留对象名，与左侧 OP Tag 配合显示，避免 "LST 设备信息(LST DEVICE_INFO)"
// 这种语义重复。括号未出现时原样返回。
function stripOpSuffix(displayName: string): string {
  const i = displayName.lastIndexOf('(');
  if (i < 0) return displayName;
  const closing = displayName.lastIndexOf(')');
  if (closing < i) return displayName;
  return displayName.slice(0, i).trimEnd();
}

function renderOpLeafTitle(op: MMLOperationType, displayName: string): ReactNode {
  const color = OP_TAG_COLOR[op] ?? 'default';
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
      <Tag color={color} style={{ marginRight: 0, fontSize: 10, padding: '0 4px' }}>
        {op}
      </Tag>
      <CodeOutlined />
      {stripOpSuffix(displayName)}
    </span>
  );
}

/** Customized 子树「+」按钮的目标 scope；null = 模态关闭。 */
type AddScope = 'public' | 'private' | null;

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

// chapterSortKey 把章节码归一化为可排序字符串；空 / undefined（老 catalog 未分章）
// 映射为高位 sentinel 排末位。与后端 chapterSortKey 行为对齐。
function chapterSortKey(chapter: string | undefined): string {
  return chapter && chapter !== '' ? chapter : '~~~~~';
}

function buildTreeData(nodes: GroupTreeNode[]): TreeDataNode[] {
  // R-1/R-2 排序：主键 chapterCode (SA→SB→...→SR，空末位)，副键 displayOrder。
  // 让对象级 group 跨章节按 SA-SR 顺序排列，前端 UI 不渲染章节为节点。
  const sorted = [...nodes].sort((a, b) => {
    const ac = chapterSortKey(a.chapterCode);
    const bc = chapterSortKey(b.chapterCode);
    if (ac !== bc) return ac < bc ? -1 : 1;
    return a.displayOrder - b.displayOrder;
  });
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
      // R-2: 命令叶子按 (op_type, displayName) 双键排序，让同一对象的不同 op
      // 相邻显示（LST 设备信息 / MOD 设备信息 / ADD 设备信息 / RMV 设备信息）。
      .sort((a, b) => {
        const opOrder = ['LST', 'MOD', 'ADD', 'RMV'];
        const ao = opOrder.indexOf(a.operationType);
        const bo = opOrder.indexOf(b.operationType);
        if (ao !== bo) return ao - bo;
        return a.displayName.localeCompare(b.displayName);
      })
      .map<TreeDataNode>((c) => ({
        key: `${CMD_KEY_PREFIX}${c.id}`,
        title: renderOpLeafTitle(c.operationType, c.displayName),
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
 * 构造 Customized 子树（PrivateTemplate + PublicTemplate）。
 * 结构：Customized > PrivateTemplate (按 creator 分组) / PublicTemplate
 *
 * PrivateTemplate / PublicTemplate 两个节点**恒显示**（空集时也在），标题尾部带
 * 「+」；点击「+」经 onAdd(scope) 打开 AddTemplateModal 新增私有 / 公共命令。
 */
function buildCustomTreeData(
  customs: MMLCustomCommand[],
  t: (id: string) => string,
  onAdd: (scope: 'public' | 'private') => void,
): TreeDataNode {
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

  // R-2: Customized 叶子同样加 OP 前缀，与 standard 命令保持视觉一致。
  // 颜色边按 scope 微调（public/private 通过 OP Tag 颜色已能区分，无需额外标记）。
  const renderLeaf = (cc: MMLCustomCommand): TreeDataNode => ({
    key: `${CUSTOM_KEY_PREFIX}${cc.id}`,
    title: renderOpLeafTitle(cc.operationType, cc.commandName),
    isLeaf: true,
  });

  // 标题尾部「+」：stopPropagation 阻止冒泡触发节点展开 / 选中。
  const addBtn = (scope: 'public' | 'private') => (
    <Tooltip
      title={scope === 'public' ? t('mml.console.addPublicTemplate') : t('mml.console.addPrivateTemplate')}
    >
      <PlusOutlined
        style={{ color: '#1677ff', marginLeft: 6 }}
        onClick={(e) => {
          e.stopPropagation();
          onAdd(scope);
        }}
      />
    </Tooltip>
  );

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

  const publicLeaves = publicGroup
    .sort((a, b) => a.commandName.localeCompare(b.commandName))
    .map(renderLeaf);

  // PrivateTemplate / PublicTemplate 恒显示；空集时 isLeaf=true（无展开箭头），
  // 仅保留标题行 + 「+」，供用户添加第一条。
  const children: TreeDataNode[] = [
    {
      key: `${CUSTOM_KEY_PREFIX}root:private`,
      title: (
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
          <FolderOutlined style={{ color: '#fa8c16' }} />
          PrivateTemplate ({privateGroup.length})
          {addBtn('private')}
        </span>
      ),
      selectable: false,
      isLeaf: privateChildren.length === 0,
      children: privateChildren.length ? privateChildren : undefined,
    },
    {
      key: `${CUSTOM_KEY_PREFIX}root:public`,
      title: (
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
          <FolderOutlined style={{ color: '#52c41a' }} />
          PublicTemplate ({publicGroup.length})
          {addBtn('public')}
        </span>
      ),
      selectable: false,
      isLeaf: publicLeaves.length === 0,
      children: publicLeaves.length ? publicLeaves : undefined,
    },
  ];

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
  // Customized 子树「+」点击后打开 AddTemplateModal；null = 关闭。
  const [addScope, setAddScope] = useState<AddScope>(null);

  const customById = useMemo(() => {
    const m = new Map<string, MMLCustomCommand>();
    customCommands.forEach((c) => m.set(c.id, c));
    return m;
  }, [customCommands]);

  const treeData = useMemo(() => {
    const groups = buildTreeData(tree);
    // Customized 恒显示（含空 Private/PublicTemplate + 「+」）；setAddScope 由 useState
    // 保证引用稳定，无需进依赖数组。
    const customRoot = buildCustomTreeData(customCommands, t, setAddScope);
    return [...groups, customRoot];
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
      <AddTemplateModal
        open={addScope !== null}
        scope={addScope ?? 'private'}
        onClose={() => setAddScope(null)}
        onSuccess={() => {
          void queryClient.invalidateQueries({ queryKey: ['mml', 'console', 'custom-commands'] });
        }}
      />
    </div>
  );
}
