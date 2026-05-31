/**
 * Task #9: 扁平化命令树（GET /mml/group-tree?format=flat）
 *
 * 与 CommandTree.tsx 区别：
 *   - 数据源：useGroupTreeFlat → FlatGroupTreeResponse（2 层：分组 → 命令叶子）
 *   - 渲染：18 个一级分组节点（按章节码 SA-SR 排序），命令为叶子节点
 *   - 命令名直接使用 API 返回的 name（已含 "LST/MOD/ADD/RMV <中文名>" 前缀）
 *   - 选中命令时通过 onSelectCommand 把 FlatCommand 整体上抛，由父级（如 CommandPanelFlat）
 *     按命令 name 前缀派生 op_type 决定渲染表格/表单/路径展示
 *
 * 不复用旧 CommandTree 的原因：
 *   旧组件耦合 sub_fields 异步加载、Customized 模板树、章节合成节点、兼容性警告等，
 *   而 flat API 已把核心数据预聚合，前端只做纯渲染 + 派发。保留两套以平滑过渡。
 */
import { useMemo, useState, useCallback } from 'react';
import type { Key, ReactNode } from 'react';
import { Input, Tree, Empty, Spin, Tag, Tooltip } from 'antd';
import {
  SearchOutlined,
  FolderOutlined,
  CodeOutlined,
} from '@ant-design/icons';
import type { TreeDataNode } from 'antd';
import { useGroupTreeFlat } from '@core/hooks/api/useMmlConsole';
import { useI18nText } from '@/hooks/useI18nText';
import { useT } from '@/hooks/useT';
import type {
  FlatCommand,
  FlatGroup,
  FlatCommandOp,
} from '@core/types/mmlConsole';
import {
  parseOpFromCommandName,
  stripCommandNameOpPrefix,
} from '@core/types/mmlConsole';

// 与旧 CommandTree 保持一致的 OP 配色，便于跨页面视觉认知。
const OP_TAG_COLOR: Record<FlatCommandOp, string> = {
  LST: 'blue',
  MOD: 'orange',
  ADD: 'green',
  RMV: 'red',
};

const GROUP_KEY_PREFIX = 'group:';
const CMD_KEY_PREFIX = 'cmd:';

/**
 * 章节码主排序键。SA-SR 取末位字母作主序；其他/空串排末位（'~' 比 'Z' 大）。
 * 与旧 commandTreeUtils.chapterSortKey 等价但本组件自含一份，避免跨包依赖。
 */
function chapterSortKey(code: string | undefined): string {
  if (!code) return '~';
  const m = /^S([A-Z])$/i.exec(code);
  return m ? m[1].toUpperCase() : `~${code}`;
}

function renderCommandLeaf(cmd: FlatCommand): ReactNode {
  const op = parseOpFromCommandName(cmd.name);
  const display = stripCommandNameOpPrefix(cmd.name);
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
      {op && (
        <Tag
          color={OP_TAG_COLOR[op]}
          style={{ marginRight: 0, fontSize: 10, padding: '0 4px' }}
        >
          {op}
        </Tag>
      )}
      <CodeOutlined />
      {display}
    </span>
  );
}

function renderGroupTitle(g: FlatGroup): ReactNode {
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4, fontWeight: 500 }}>
      <FolderOutlined />
      <Tooltip title={g.code}>
        <span>{g.name}</span>
      </Tooltip>
      <span style={{ color: '#999', fontWeight: 400 }}>({g.commands.length})</span>
    </span>
  );
}

function buildTreeData(groups: FlatGroup[]): TreeDataNode[] {
  const sorted = [...groups].sort((a, b) => {
    const ac = chapterSortKey(a.code);
    const bc = chapterSortKey(b.code);
    if (ac !== bc) return ac < bc ? -1 : 1;
    return a.name.localeCompare(b.name);
  });

  return sorted.map<TreeDataNode>((g) => ({
    key: `${GROUP_KEY_PREFIX}${g.code}`,
    title: renderGroupTitle(g),
    selectable: false,
    children: g.commands.map<TreeDataNode>((c) => ({
      key: `${CMD_KEY_PREFIX}${c.id}`,
      title: renderCommandLeaf(c),
      isLeaf: true,
    })),
  }));
}

/** 搜索匹配：按命令 name 子串匹配，返回命中 keys + 需要展开的祖先 group keys。 */
function collectMatches(
  groups: FlatGroup[],
  needle: string,
): { matchedKeys: Set<string>; expandKeys: string[] } {
  const matchedKeys = new Set<string>();
  const expandKeys: string[] = [];
  const lower = needle.toLowerCase();
  groups.forEach((g) => {
    let groupHit = false;
    g.commands.forEach((c) => {
      if (c.name.toLowerCase().includes(lower)) {
        matchedKeys.add(`${CMD_KEY_PREFIX}${c.id}`);
        groupHit = true;
      }
    });
    if (groupHit) expandKeys.push(`${GROUP_KEY_PREFIX}${g.code}`);
  });
  return { matchedKeys, expandKeys };
}

export interface CommandTreeFlatProps {
  /** 选中命令叶子时回调，传完整 FlatCommand 给父级（CommandPanelFlat 消费） */
  onSelectCommand?: (cmd: FlatCommand) => void;
  /** 当前已选中命令 id（用于高亮）；undefined = 无选中 */
  selectedCommandId?: string;
  /** lang 透传给 API；缺省 'zh-CN'。 */
  lang?: 'zh-CN' | 'en-US';
}

/**
 * Task #9 扁平化命令树。父组件控制选中态以便联动右侧面板。
 */
export default function CommandTreeFlat({
  onSelectCommand,
  selectedCommandId,
  lang,
}: CommandTreeFlatProps) {
  const t = useT();
  const { locale: appLocale } = useI18nText();
  const effectiveLang = lang ?? appLocale;
  const { data, isLoading } = useGroupTreeFlat(effectiveLang);
  const groups = useMemo(() => data?.groups ?? [], [data]);

  const [searchText, setSearchText] = useState('');
  const [expandedKeys, setExpandedKeys] = useState<Key[]>([]);
  const [autoExpand, setAutoExpand] = useState(true);

  const treeData = useMemo(() => buildTreeData(groups), [groups]);

  const matched = useMemo(() => {
    if (!searchText.trim()) return { matchedKeys: new Set<string>(), expandKeys: [] };
    return collectMatches(groups, searchText.trim());
  }, [searchText, groups]);

  // commandsById：选中时 O(1) 取回 FlatCommand 整体
  const commandsById = useMemo(() => {
    const m = new Map<string, FlatCommand>();
    groups.forEach((g) => g.commands.forEach((c) => m.set(c.id, c)));
    return m;
  }, [groups]);

  const onSearchChange = useCallback(
    (value: string) => {
      setSearchText(value);
      if (!value.trim()) {
        setExpandedKeys([]);
        setAutoExpand(true);
        return;
      }
      const { expandKeys } = collectMatches(groups, value.trim());
      setExpandedKeys(expandKeys);
      setAutoExpand(false);
    },
    [groups],
  );

  const handleSelect = useCallback(
    (selectedKeys: Key[]) => {
      const key = selectedKeys[0];
      if (typeof key !== 'string' || !key.startsWith(CMD_KEY_PREFIX)) return;
      const id = key.slice(CMD_KEY_PREFIX.length);
      const cmd = commandsById.get(id);
      if (cmd) onSelectCommand?.(cmd);
    },
    [commandsById, onSelectCommand],
  );

  const selectedKeys = useMemo<Key[]>(
    () => (selectedCommandId ? [`${CMD_KEY_PREFIX}${selectedCommandId}`] : []),
    [selectedCommandId],
  );

  if (isLoading) {
    return <Spin />;
  }
  if (groups.length === 0) {
    return <Empty description={t('mml.console.empty.noGroups')} />;
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      <Input
        prefix={<SearchOutlined />}
        placeholder={t('mml.console.searchPlaceholder')}
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
        selectedKeys={selectedKeys}
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
