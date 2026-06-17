/**
 * customizedSubtree — MML「自定义命令」(Customized) 子树共享模块。
 *
 * 由 Console 命令树（Console/components/CommandTree.tsx）与 admin catalog 左栏
 * （admin/catalog/index.tsx）共用。两者复用同一套：
 *   - 自定义命令数据加载（useCustomCommands，共享 query key）
 *   - PrivateTemplate / PublicTemplate 拆分 + 按 creator 分组
 *   - Customized / PrivateTemplate / PublicTemplate 骨架 + 「+」按钮 + i18n 标签
 *
 * 差异点（每个页面自带）通过参数注入：
 *   - Console：叶子带行内 ✏️ / 🗑️ 图标，点击触发模态编辑 / 删除
 *   - catalog：叶子为可选中节点（key `custom:<id>`），无行内图标，详情走 RightDetailPanel
 *
 * 设计决策：docs/design/mml-console-redesign-20260603.md（提取共享 Customized 子树）。
 *
 * 注：本文件是「子树构造工具 + 数据 hook」模块，不导出 React 组件；Fast Refresh 的
 * 单组件约束不适用，故按项目惯例（参见 alarm/AlarmStatistics/index.tsx）关闭该规则。
 */
/* eslint-disable react-refresh/only-export-components */
import type { ReactNode } from 'react';
import { useQuery } from '@tanstack/react-query';
import { FolderOutlined, UserOutlined, PlusOutlined } from '@ant-design/icons';
import { Tag, Tooltip, type TreeDataNode } from 'antd';
import { mmlApi } from '@core/services/api/mmlApi';
import type { MMLCustomCommand, MMLOperationType } from '@core/types/mml';

// 自定义命令叶子 key 前缀，Console 与 catalog 共用（两页 handleSelect 据此识别）。
export const CUSTOM_KEY_PREFIX = 'custom:';

// Customized 子树骨架的稳定 key（root / PrivateTemplate / PublicTemplate / creator 分组）。
export const CUSTOM_ROOT_KEY = `${CUSTOM_KEY_PREFIX}root`;
export const CUSTOM_PRIVATE_ROOT_KEY = `${CUSTOM_KEY_PREFIX}root:private`;
export const CUSTOM_PUBLIC_ROOT_KEY = `${CUSTOM_KEY_PREFIX}root:public`;

// 自定义命令共享 query key —— Console 与 catalog 用同一把 key，
// 任一页的增删改 invalidate 后两页同时刷新。
export const CUSTOM_COMMANDS_QUERY_KEY = ['mml', 'custom-commands'] as const;

/** 从 key 反解自定义命令 id；非 custom 叶子返回 null。 */
export function parseCustomLeafId(key: string): string | null {
  if (!key.startsWith(CUSTOM_KEY_PREFIX)) return null;
  const rest = key.slice(CUSTOM_KEY_PREFIX.length);
  // 排除骨架节点（root / root:private / root:public / user:<creator>）。
  if (rest === 'root' || rest.startsWith('root:') || rest.startsWith('user:')) {
    return null;
  }
  return rest;
}

/**
 * 加载自定义命令（PrivateTemplate + PublicTemplate）。
 *
 * 复用与 Console 原实现一致的 getTemplates 查询，但统一到共享 query key
 * CUSTOM_COMMANDS_QUERY_KEY，使 Console / catalog 两页的失效互相联动。
 */
export function useCustomCommands() {
  const query = useQuery({
    queryKey: CUSTOM_COMMANDS_QUERY_KEY,
    queryFn: () => mmlApi.getTemplates({ page: 1, pageSize: 100 }),
    staleTime: 5 * 60 * 1000,
  });
  return {
    ...query,
    commands: query.data?.items ?? [],
  };
}

// R-2：Customized 叶子与 standard 命令一致加 OP 前缀。OP Tag 颜色按操作语义着色。
const OP_TAG_COLOR: Record<string, string> = {
  LST: 'blue',
  MOD: 'orange',
  ADD: 'green',
  RMV: 'red',
};

/** OP Tag + 命令名，供叶子标题复用（不含行内操作图标）。 */
export function renderCustomLeafLabel(
  op: MMLOperationType,
  commandName: string,
): ReactNode {
  const color = OP_TAG_COLOR[op] ?? 'default';
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
      <Tag color={color} style={{ marginRight: 0, fontSize: 10, padding: '0 4px' }}>
        {op}
      </Tag>
      {commandName}
    </span>
  );
}

export interface BuildCustomizedSubtreeOptions {
  customs: MMLCustomCommand[];
  t: (id: string, values?: Record<string, string | number>) => string;
  /**
   * 「+」按钮点击：打开 AddTemplateModal（private / public）。
   * 不传则不渲染「+」（console「选择命令」只读复用，仅选不增）。
   */
  onAdd?: (scope: 'public' | 'private') => void;
  /**
   * 叶子标题渲染器，由调用方决定：
   *   - Console：renderCustomLeafLabel + 行内 ✏️ / 🗑️ 图标
   *   - catalog：renderCustomLeafLabel（纯文本，可选中）
   */
  renderLeafTitle: (cc: MMLCustomCommand) => ReactNode;
  /** 叶子是否可选中（catalog=true 走 RightDetailPanel；Console 走自身 onSelect 逻辑）。 */
  leafSelectable?: boolean;
}

/**
 * 构造 Customized 子树（PrivateTemplate + PublicTemplate）。
 * 结构：Customized > PrivateTemplate (按 creator 分组) / PublicTemplate
 *
 * PrivateTemplate / PublicTemplate 两节点恒显示（空集时也在），标题尾部带「+」；
 * 点击「+」经 onAdd(scope) 打开 AddTemplateModal 新增私有 / 公共命令。
 */
export function buildCustomizedSubtree(
  opts: BuildCustomizedSubtreeOptions,
): TreeDataNode {
  const { customs, t, onAdd, renderLeafTitle, leafSelectable = false } = opts;

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
    title: renderLeafTitle(cc),
    isLeaf: true,
    selectable: leafSelectable,
  });

  // 标题尾部「+」：stopPropagation 阻止冒泡触发节点展开 / 选中。
  // onAdd 缺省（只读场景，如 console 选择命令）时返回 null，不渲染「+」。
  const addBtn = (scope: 'public' | 'private') =>
    onAdd ? (
      <Tooltip
        title={
          scope === 'public'
            ? t('mml.console.addPublicTemplate')
            : t('mml.console.addPrivateTemplate')
        }
      >
        <PlusOutlined
          style={{ color: '#1677ff', marginLeft: 6 }}
          onClick={(e) => {
            e.stopPropagation();
            onAdd(scope);
          }}
        />
      </Tooltip>
    ) : null;

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
      children: items
        .slice()
        .sort((a, b) => a.commandName.localeCompare(b.commandName))
        .map(renderLeaf),
    }),
  );

  const publicLeaves = publicGroup
    .slice()
    .sort((a, b) => a.commandName.localeCompare(b.commandName))
    .map(renderLeaf);

  // PrivateTemplate / PublicTemplate 恒显示；空集时 isLeaf=true（无展开箭头），
  // 仅保留标题行 + 「+」，供用户添加第一条。
  const children: TreeDataNode[] = [
    {
      key: CUSTOM_PRIVATE_ROOT_KEY,
      title: (
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
          <FolderOutlined style={{ color: '#fa8c16' }} />
          {t('mml.console.commandTree.privateTemplate')} ({privateGroup.length})
          {addBtn('private')}
        </span>
      ),
      selectable: false,
      isLeaf: privateChildren.length === 0,
      children: privateChildren.length ? privateChildren : undefined,
    },
    {
      key: CUSTOM_PUBLIC_ROOT_KEY,
      title: (
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
          <FolderOutlined style={{ color: '#52c41a' }} />
          {t('mml.console.commandTree.publicTemplate')} ({publicGroup.length})
          {addBtn('public')}
        </span>
      ),
      selectable: false,
      isLeaf: publicLeaves.length === 0,
      children: publicLeaves.length ? publicLeaves : undefined,
    },
  ];

  return {
    key: CUSTOM_ROOT_KEY,
    title: (
      <span
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 4,
          fontWeight: 500,
        }}
      >
        <FolderOutlined />
        {t('mml.console.commandTree.customized')}
      </span>
    ),
    selectable: false,
    children,
  };
}
