/**
 * GroupTreeSelect — KPI 指标分组「真·树形选择器」(v1 webcode/Antd5)。
 *
 * 2026-06-22：把所有「按分组筛选 / 归属分组 / 父分组」下拉从 Antd `<Select>` +
 *   全角空格缩进的拍平选项,升级为 Antd `<TreeSelect>`,与「分组管理」表格的展示
 *   方式保持一致(树结构、可展开收起)。默认全部展开,与现状 `defaultExpandAllRows`
 *   表格行为对齐。
 *
 * 数据契约不变:value 仍是 group.id 字符串;后端 listGroups 返回的嵌套 children
 *   结构由 frontend-core 的 groupTreeData() 转为 TreeSelect treeData 格式。
 *
 * 复用范围:
 *   · index.tsx       —— 详情态顶部「分组筛选」(无 excludeId,allowClear)
 *   · IndicatorFormModal.tsx —— 「归属分组」必填字段(无 excludeId,disabled 时只读)
 *   · GroupsManageModal.tsx  —— 编辑分组时「父分组」字段(传 excludeId 排除自身子树)
 */
import { useEffect, useMemo, useRef, useState } from 'react';
import { TreeSelect } from 'antd';

import { useIndicatorGroups } from '@core/hooks/api/useIndicatorsLibrary';
import { groupTreeData } from '@core/services/api/indicatorLibraryApi';
import type { GroupTreeNode } from '@core/services/api/indicatorLibraryApi';
import type { DeviceType } from '@core/types/indicatorLibrary';

interface Props {
  deviceType: DeviceType;
  operatorCode?: string;
  /** 仅 v1 详情态分组筛选用:按 platform 过滤分组(只显示当前 platform 涉及的分组)。 */
  platform?: string;
  /** 受控值;用在 Antd Form.Item 里时 Form.Item 会经 cloneElement 注入,可省。 */
  value?: string;
  /** Antd TreeSelect 在清空时回传 undefined,父可用 ''/undefined 都行。用在 Form.Item 里时可省。 */
  onChange?: (next: string | undefined) => void;
  /** 编辑「父分组」时传被编辑分组的 id,排除自身及整棵子树(避免成环)。 */
  excludeId?: string;
  disabled?: boolean;
  allowClear?: boolean;
  placeholder?: string;
  /** 透传给 Antd TreeSelect,宽度等用 style 控制(与现有 Select 调用点保持一致)。 */
  style?: React.CSSProperties;
}

/** 收集所有非叶节点 key — 用于"默认全部展开"。 */
function collectExpandableKeys(nodes: GroupTreeNode[] | undefined, into: Set<string>): void {
  (nodes || []).forEach((n) => {
    if (n.children && n.children.length > 0) {
      into.add(n.key);
      collectExpandableKeys(n.children, into);
    }
  });
}

export default function GroupTreeSelect({
  deviceType,
  operatorCode,
  platform,
  value,
  onChange,
  excludeId,
  disabled,
  allowClear,
  placeholder,
  style,
}: Props) {
  const { data, isLoading } = useIndicatorGroups(deviceType, operatorCode, platform);
  const treeData = useMemo(
    () => groupTreeData(data?.items, { excludeId }),
    [data, excludeId],
  );

  // Antd TreeSelect 的 `treeDefaultExpandAll` 仅在**初次渲染**时生效;若 treeData 异步加载
  // (本组件经 useIndicatorGroups + React Query),首屏空 → 树到来后不会再"默认展开"。
  // 改用受控 treeExpandedKeys:新出现的可展开节点自动加入(保留用户手动收起的状态)。
  // 注:Antd `SafeKey = string | number`,这里用 string[] 即可(group.id 都是字符串)。
  const [expandedKeys, setExpandedKeys] = useState<string[]>([]);
  const seenExpandable = useRef<Set<string>>(new Set());
  useEffect(() => {
    const allExpandable = new Set<string>();
    collectExpandableKeys(treeData, allExpandable);
    const toAdd: string[] = [];
    allExpandable.forEach((k) => {
      if (!seenExpandable.current.has(k)) toAdd.push(k);
    });
    if (toAdd.length > 0) {
      setExpandedKeys((prev) => Array.from(new Set([...prev, ...toAdd])));
    }
    seenExpandable.current = allExpandable;
    // 故意只依赖 treeData,不响应 expandedKeys —— 否则用户每次手动收起会被回放为"展开"。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [treeData]);

  return (
    <TreeSelect
      value={value || undefined}
      onChange={(v) => onChange?.(v ?? undefined)}
      treeData={treeData}
      treeExpandedKeys={expandedKeys}
      onTreeExpand={(keys) => setExpandedKeys(keys.map(String))}
      treeNodeFilterProp="title"
      showSearch
      allowClear={allowClear}
      disabled={disabled}
      loading={isLoading}
      placeholder={placeholder}
      style={style}
      popupMatchSelectWidth={false}
      dropdownStyle={{ maxHeight: 400, overflow: 'auto' }}
    />
  );
}
