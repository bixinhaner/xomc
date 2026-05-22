/**
 * PathPicker — Bundle D：自定义命令模板的"参数路径"多选选择器
 *
 * 给定一组已选 path（受控 value: string[]），用户可通过两种交互添加 / 移除：
 *
 *   A. 搜索框（AutoComplete）— type-ahead 复用 Bundle C `useSearchCommands`：
 *      后端 ILIKE 联合搜命令 + path + description，下拉直接列匹配 path，
 *      点击即追加到已选列表（去重）。
 *
 *   B. 三级下拉（分组 → 命令 → 路径多选）：
 *      · 分组：来自 /mml/group-tree 的 18 个 chapter 节点
 *      · 命令：当前分组下的命令叶子（来自同一 group-tree 响应内 commands[]）
 *      · 路径：选定命令后调 /mml/commands/:id/sub-fields 拿 path 集合，
 *               以多选 Select 一次添加多条 path
 *
 * 已选 path 以 Tag 列表展示在下方，可点 × 单条移除。
 *
 * 复用：AddTemplateModal（新增 / 编辑公私模板）；未来 Bundle E（如需在其它
 *      命令编辑入口加 path 绑定）也可直接复用本组件。
 */

import { useEffect, useMemo, useState } from 'react';
import { AutoComplete, Empty, Select, Space, Tag, Tooltip, Typography } from 'antd';
import { useGroupTree, useCommandSubFields, useSearchCommands } from '@core/hooks/api/useMmlConsole';
import type { GroupTreeNode } from '@core/types/mmlConsole';
import { useT } from '@/hooks/useT';

export interface PathPickerProps {
  /** 已选 path 集合（受控）。 */
  value: string[];
  /** 选择变更回调（接收去重后的 path 数组）。 */
  onChange: (paths: string[]) => void;
  /** 语言；默认 'zh-CN'。 */
  lang?: 'zh-CN' | 'en-US';
  /** 整个组件禁用（如表单 readonly 时）。 */
  disabled?: boolean;
}

interface Option {
  label: string;
  value: string;
}

export default function PathPicker({
  value,
  onChange,
  lang = 'zh-CN',
  disabled,
}: PathPickerProps): JSX.Element {
  const t = useT();
  const { data: tree = [] } = useGroupTree(undefined, lang);

  // ----- Browse 模式 state -----
  const [groupCode, setGroupCode] = useState<string | undefined>();
  const [commandId, setCommandId] = useState<string | undefined>();

  const groupOptions = useMemo<Option[]>(
    () =>
      (tree as GroupTreeNode[])
        // chapter:* 顶层（v2.3 spec §R-1）；旧 catalog 顶层兜底
        .filter((g) => g.groupCode?.startsWith('chapter:') || g.source === 'synthetic')
        .map((g) => ({ label: g.displayName ?? g.groupCode, value: g.groupCode })),
    [tree],
  );

  const commandOptions = useMemo<Option[]>(() => {
    if (!groupCode) return [];
    const grp = (tree as GroupTreeNode[]).find((g) => g.groupCode === groupCode);
    if (!grp) return [];
    // chapter 直挂的 commands；同时兼容 (老 catalog) sub-group 嵌套
    const directLeaves = grp.commands ?? [];
    const nestedLeaves: typeof directLeaves = [];
    (grp.children ?? []).forEach((child) => {
      (child.commands ?? []).forEach((c) => nestedLeaves.push(c));
    });
    return [...directLeaves, ...nestedLeaves].map((c) => ({
      label: c.displayName,
      value: c.id,
    }));
  }, [tree, groupCode]);

  // 选中命令变化时拉它的 sub_fields（含 standardPath + description）
  const { data: subFields = [] } = useCommandSubFields(commandId, lang);
  const pathOptions = useMemo<Option[]>(
    () =>
      subFields.map((sf) => ({
        value: sf.tr069Path,
        // path 主文 + 中文 description 灰副文（Cascader / Select 不支持复杂 label
        // 节点，这里用 " — descr" 拼接，AutoComplete 同款）
        label: sf.description ? `${sf.tr069Path} — ${sf.description}` : sf.tr069Path,
      })),
    [subFields],
  );

  // ----- Search 模式 state -----
  const [searchText, setSearchText] = useState('');
  const [debouncedQuery, setDebouncedQuery] = useState('');
  useEffect(() => {
    const id = setTimeout(() => setDebouncedQuery(searchText.trim()), 300);
    return () => clearTimeout(id);
  }, [searchText]);
  const { data: searchResults = [] } = useSearchCommands(debouncedQuery, lang);

  // 把 search 返回的 matched_paths 摊平为 AutoComplete options：每条 path 一行，
  // 后缀附加命令名让用户辨识同 path 属于哪条命令。
  const searchPathOptions = useMemo<Option[]>(() => {
    const seen = new Set<string>();
    const out: Option[] = [];
    searchResults.forEach((r) => {
      r.matchedPaths.forEach((p) => {
        if (seen.has(p)) return;
        seen.add(p);
        out.push({ value: p, label: `${p}  —  ${r.displayName}` });
      });
    });
    return out;
  }, [searchResults]);

  // ----- 已选 path 操作 -----
  const handleAddPaths = (newPaths: string[]) => {
    if (newPaths.length === 0) return;
    const merged = Array.from(new Set([...value, ...newPaths]));
    if (merged.length !== value.length) onChange(merged);
  };
  const handleAddOne = (p: string) => {
    handleAddPaths([p]);
    setSearchText(''); // 清空搜索 input 以便连续添加
  };
  const handleRemove = (p: string) => {
    onChange(value.filter((x) => x !== p));
  };
  const handleClear = () => onChange([]);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      {/* A. 搜索模式 */}
      <AutoComplete
        size="small"
        style={{ width: '100%' }}
        placeholder={t('mml.console.pathPicker.searchPlaceholder')}
        value={searchText}
        options={searchPathOptions}
        onSearch={setSearchText}
        onSelect={handleAddOne}
        onChange={setSearchText}
        disabled={disabled}
        notFoundContent={
          debouncedQuery ? <Empty description={false} image={Empty.PRESENTED_IMAGE_SIMPLE} /> : null
        }
      />

      {/* B. 三级下拉模式（分组 → 命令 → 路径多选） */}
      <Space size="small" wrap style={{ width: '100%' }}>
        <Select
          size="small"
          style={{ width: 200 }}
          placeholder={t('mml.console.pathPicker.groupPlaceholder')}
          options={groupOptions}
          value={groupCode}
          onChange={(v) => {
            setGroupCode(v);
            setCommandId(undefined);
          }}
          disabled={disabled}
          showSearch
          optionFilterProp="label"
          allowClear
        />
        <Select
          size="small"
          style={{ width: 240 }}
          placeholder={t('mml.console.pathPicker.commandPlaceholder')}
          options={commandOptions}
          value={commandId}
          onChange={setCommandId}
          disabled={disabled || !groupCode}
          showSearch
          optionFilterProp="label"
          allowClear
        />
        <Select
          size="small"
          style={{ width: 320 }}
          mode="multiple"
          placeholder={t('mml.console.pathPicker.pathPlaceholder')}
          options={pathOptions}
          // multi-select Select 的 value 应表示"当前下拉框内已勾选"，本组件把
          // value 提到外部 Tag 列表，所以这里始终 [] —— 用户点哪条直接 add
          value={[]}
          onChange={(picked) => handleAddPaths(picked as string[])}
          disabled={disabled || !commandId}
          maxTagCount={0}
          maxTagPlaceholder={() => null}
          optionFilterProp="label"
          allowClear
        />
      </Space>

      {/* 已选 path 列表 */}
      <div
        style={{
          minHeight: 32,
          padding: '6px 8px',
          border: '1px dashed #d9d9d9',
          borderRadius: 4,
          background: '#fafafa',
          display: 'flex',
          flexWrap: 'wrap',
          gap: 4,
          alignItems: 'center',
        }}
      >
        {value.length === 0 ? (
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {t('mml.console.pathPicker.empty')}
          </Typography.Text>
        ) : (
          <>
            {value.map((p) => (
              <Tooltip key={p} title={p}>
                <Tag
                  closable={!disabled}
                  onClose={(e) => {
                    e.preventDefault();
                    handleRemove(p);
                  }}
                  style={{ fontFamily: 'monospace', fontSize: 11 }}
                >
                  {p}
                </Tag>
              </Tooltip>
            ))}
            {!disabled && (
              <Typography.Link
                style={{ fontSize: 11, marginLeft: 4 }}
                onClick={handleClear}
                aria-label="clear-all-paths"
              >
                {t('mml.console.pathPicker.clearAll')}
              </Typography.Link>
            )}
          </>
        )}
      </div>
    </div>
  );
}
