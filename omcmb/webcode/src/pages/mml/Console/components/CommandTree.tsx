import { useState, useMemo, useCallback } from 'react';
import type { Key, ReactNode } from 'react';
import { Input, Tree, Empty, Spin, message, Popconfirm, Tooltip, Tag } from 'antd';
import {
  SearchOutlined,
  FolderOutlined,
  FolderOpenOutlined,
  CodeOutlined,
  UserOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import { AxiosError } from 'axios';
import type { TreeDataNode } from 'antd';
import { useQueryClient, useQuery } from '@tanstack/react-query';
import { useGroupTree, useCommandCompatibility } from '@core/hooks/api/useMmlConsole';
import { useDeleteMMLTemplate } from '@core/hooks/api/useMML';
import { mmlApi } from '@core/services/api/mmlApi';
import { useMmlConsoleStore } from '@core/store/mmlConsoleStore';
import { useUserStore } from '@core/store/userStore';
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

// stripOpSuffix / chapterSortKey 已拆到 ./commandTreeUtils.ts；下方 import 复用。
import { chapterSortKey, stripOpSuffix } from './commandTreeUtils';

/**
 * R-8.5：命令叶子装饰上下文。在 buildTreeData 调用时由组件构造，沿渲染链传递，
 * 避免每个 helper 各自接 4-5 个独立参数。
 */
interface LeafDecor {
  /** 不兼容命令的 ID 集合；undefined = 兼容性数据未加载，全部不显示警告 */
  unsupportedSet?: Set<string>;
  /** R-8.5 Tooltip 文案（zh-CN / en-US 已 i18n 解析） */
  unsupportedTooltip: string;
}

/**
 * commandId + decor 都 optional：
 *   - 标准命令叶子（buildTreeData 链路）传完整 4 参数 → R-8.5 警告生效
 *   - Customized 模板（buildCustomTreeData 链路）只传前 2 参数 →
 *     不显示兼容性警告（R-5 customized 不在 R-8.5 检查范围内，且语义不适用）
 */
function renderOpLeafTitle(
  op: MMLOperationType,
  displayName: string,
  commandId?: string,
  decor?: LeafDecor,
): ReactNode {
  const color = OP_TAG_COLOR[op] ?? 'default';
  const isUnsupported =
    commandId != null && decor?.unsupportedSet?.has(commandId) === true;
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
      <Tag color={color} style={{ marginRight: 0, fontSize: 10, padding: '0 4px' }}>
        {op}
      </Tag>
      <CodeOutlined />
      {stripOpSuffix(displayName)}
      {isUnsupported && decor && (
        <Tooltip title={decor.unsupportedTooltip} placement="right">
          <WarningOutlined
            style={{ color: '#faad14', marginLeft: 2 }}
            aria-label="unsupported-for-product-class"
          />
        </Tooltip>
      )}
    </span>
  );
}

/** Customized 子树「+」按钮的目标 scope；null = 模态关闭。 */
type AddScope = 'public' | 'private' | null;

const GROUP_KEY_PREFIX = 'group:';
const CMD_KEY_PREFIX = 'cmd:';
const CUSTOM_KEY_PREFIX = 'custom:';
// CMCC TD-LTE v2.3：后端 wrapByChapter 把 group 折叠到合成的 SA-SR 章节父节点下。
// 章节节点的 groupCode 形如 "chapter:SA"，用此前缀识别（避免与真 group code 撞）。
const CHAPTER_GROUP_PREFIX = 'chapter:';

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

/** 把命令叶子列表转 antd TreeDataNode（统一 OP 排序）。 */
function commandsToLeafNodes(cmds: GroupTreeCommand[], decor: LeafDecor): TreeDataNode[] {
  return [...cmds]
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
      title: renderOpLeafTitle(c.operationType, c.displayName, c.id, decor),
      isLeaf: true,
    }));
}

/** 判断节点是否为后端合成的章节父节点（groupCode 形如 "chapter:SA"）。 */
function isChapterNode(g: GroupTreeNode): boolean {
  return g.groupCode.startsWith(CHAPTER_GROUP_PREFIX) || g.source === 'synthetic';
}

/** 将单个 group 节点（含其所有平铺命令）转为 antd TreeDataNode。 */
function groupToTreeDataNode(g: GroupTreeNode, decor: LeafDecor): TreeDataNode {
  return {
    key: `${GROUP_KEY_PREFIX}${g.id}`,
    title: (
      <span>
        <FolderOutlined style={{ marginRight: 4 }} />
        {g.displayName}
      </span>
    ),
    selectable: false,
    children: commandsToLeafNodes(collectAllCommands(g), decor),
  };
}

/** 将后端合成的章节节点转为 antd TreeDataNode，children 仍是 group 子节点。 */
function chapterToTreeDataNode(g: GroupTreeNode, decor: LeafDecor): TreeDataNode {
  // 章节节点 key 仍走 GROUP_KEY_PREFIX + id（章节合成 id 也是 UUID，与 group 同空间
  // 不冲突；handleSelect 通过 isLeaf=false + selectable:false 防止误触发命令加载）
  return {
    key: `${GROUP_KEY_PREFIX}${g.id}`,
    title: (
      <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4, fontWeight: 500 }}>
        <FolderOpenOutlined style={{ marginRight: 4 }} />
        {g.displayName}
      </span>
    ),
    selectable: false,
    children: (g.children ?? []).map((child) => groupToTreeDataNode(child, decor)),
  };
}

/**
 * 后端 wrapByChapter 已把 group 折叠到 SA-SR 章节父节点下；前端只递归映射为 antd 树。
 * - 章节节点（source='synthetic' 或 groupCode 以 "chapter:" 开头）→ chapterToTreeDataNode
 * - 普通 group → groupToTreeDataNode（含命令叶子）
 * - 老 catalog 空 chapter 的 group 后端未包装，仍保持顶层
 */
function buildTreeData(nodes: GroupTreeNode[], decor: LeafDecor): TreeDataNode[] {
  // 后端已按 chapter 排好序，前端再做一次防御性排序（按 chapterCode + displayOrder）
  const sorted = [...nodes].sort((a, b) => {
    const ac = chapterSortKey(a.chapterCode);
    const bc = chapterSortKey(b.chapterCode);
    if (ac !== bc) return ac < bc ? -1 : 1;
    return a.displayOrder - b.displayOrder;
  });

  return sorted.map((g) =>
    isChapterNode(g) ? chapterToTreeDataNode(g, decor) : groupToTreeDataNode(g, decor),
  );
}

function collectMatches(
  nodes: GroupTreeNode[],
  needle: string,
): { matchedKeys: Set<string>; expandKeys: string[] } {
  const matchedKeys = new Set<string>();
  const expandKeys: string[] = [];
  const lower = needle.toLowerCase();
  // 后端 wrapByChapter 后树形为：chapter → group → command（或老 catalog 的 group → command）。
  // walk 沿 children 递归把 ancestor 路径（含 chapter 父节点）加进 expandKeys，
  // 保证 antd Tree 展开命中命令的整条祖先链。
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
 *
 * Edit/Delete 入口（docs/design/mml-user-private-template-crud-20260520.md §5.1 D5）：
 * 仅在 (commandScope='private' AND (creator===currentUsername || isSuperAdmin)) 时
 * 在叶子右侧追加 ✏️ / 🗑️ 图标；其他模板（public / 非自己的 private）只显示文本。
 */
function buildCustomTreeData(
  customs: MMLCustomCommand[],
  t: (id: string) => string,
  onAdd: (scope: 'public' | 'private') => void,
  onEdit: (cc: MMLCustomCommand) => void,
  onDelete: (cc: MMLCustomCommand) => void,
  currentUsername: string,
  isSuperAdmin: boolean,
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
  const renderLeaf = (cc: MMLCustomCommand): TreeDataNode => {
    // 编辑/删除入口仅在私有 + 自己的（或 super_admin）时暴露
    const canModify =
      cc.commandScope === 'private' && (isSuperAdmin || cc.creator === currentUsername);
    const titleEl = (
      <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
        {renderOpLeafTitle(cc.operationType, cc.commandName)}
        {canModify && (
          <span style={{ display: 'inline-flex', gap: 6, marginLeft: 8 }}>
            <Tooltip title={t('mml.template.action.edit')}>
              <EditOutlined
                style={{ color: '#1677ff', cursor: 'pointer', fontSize: 12 }}
                onClick={(e) => {
                  e.stopPropagation();
                  onEdit(cc);
                }}
              />
            </Tooltip>
            <Popconfirm
              title={t('mml.template.deleteConfirm', { name: cc.commandName })}
              okText={t('common.confirm')}
              cancelText={t('common.cancel')}
              onConfirm={(e) => {
                e?.stopPropagation();
                onDelete(cc);
              }}
              onCancel={(e) => e?.stopPropagation()}
            >
              <Tooltip title={t('mml.template.action.delete')}>
                <DeleteOutlined
                  style={{ color: '#ff4d4f', cursor: 'pointer', fontSize: 12 }}
                  onClick={(e) => e.stopPropagation()}
                />
              </Tooltip>
            </Popconfirm>
          </span>
        )}
      </span>
    );
    return {
      key: `${CUSTOM_KEY_PREFIX}${cc.id}`,
      title: titleEl,
      isLeaf: true,
    };
  };

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
  const deleteMutation = useDeleteMMLTemplate();

  // 当前用户上下文（用于编辑/删除入口暴露条件 §5.1 D5）
  const currentUsername = useUserStore((s) => s.currentUser?.username ?? '');
  const isSuperAdmin = useUserStore((s) => Boolean(s.currentUser?.isSuperAdmin));

  // R-8.5：订阅当前选中 product_class（由 Console/index.tsx 单向镜像进 store），
  // 调 useCommandCompatibility 取"该 product_class 下不兼容的命令 ID 集合"。
  // productClassFilter 为空 / 加载中 → unsupportedSet 为 undefined → 不显示任何警告。
  const productClassFilter = useMmlConsoleStore((s) => s.productClassFilter);
  const { data: unsupportedSet } = useCommandCompatibility(productClassFilter);
  const unsupportedTooltip = t('mml.console.commandTree.unsupportedForProductClass');

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
  // 编辑模式：非空时 modal 打开走 update 路径（§5.2 D13）
  const [editingTemplate, setEditingTemplate] = useState<MMLCustomCommand | null>(null);

  const customById = useMemo(() => {
    const m = new Map<string, MMLCustomCommand>();
    customCommands.forEach((c) => m.set(c.id, c));
    return m;
  }, [customCommands]);

  // 删除模板：成功后失效本地 query；后端 hook 已 invalidate ['mml','templates']，
  // 这里再 invalidate ['mml','console','custom-commands']（CommandTree 用的 key）。
  const handleDeleteTemplate = useCallback(
    (cc: MMLCustomCommand) => {
      deleteMutation.mutate(cc.id, {
        onSuccess: () => {
          void queryClient.invalidateQueries({ queryKey: ['mml', 'console', 'custom-commands'] });
          void message.success(t('mml.template.deleted'));
        },
        onError: (err: unknown) => {
          if (err instanceof AxiosError && err.response?.status === 403) {
            void message.error(t('mml.template.error.notOwner'));
            return;
          }
          const msg = err instanceof Error ? err.message : String(err ?? 'Unknown');
          void message.error(msg);
        },
      });
    },
    [deleteMutation, queryClient, t],
  );

  const treeData = useMemo(() => {
    const leafDecor: LeafDecor = { unsupportedSet, unsupportedTooltip };
    const groups = buildTreeData(tree, leafDecor);
    const customRoot = buildCustomTreeData(
      customCommands,
      t,
      setAddScope,
      setEditingTemplate,
      handleDeleteTemplate,
      currentUsername,
      isSuperAdmin,
    );
    return [...groups, customRoot];
  }, [
    tree,
    customCommands,
    t,
    handleDeleteTemplate,
    currentUsername,
    isSuperAdmin,
    unsupportedSet,
    unsupportedTooltip,
  ]);
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
        open={addScope !== null || editingTemplate !== null}
        scope={addScope ?? (editingTemplate?.commandScope as 'public' | 'private') ?? 'private'}
        editingTemplate={editingTemplate}
        onClose={() => {
          setAddScope(null);
          setEditingTemplate(null);
        }}
        onSuccess={() => {
          void queryClient.invalidateQueries({ queryKey: ['mml', 'console', 'custom-commands'] });
        }}
      />
    </div>
  );
}
