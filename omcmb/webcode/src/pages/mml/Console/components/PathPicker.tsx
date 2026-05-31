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
import { Empty, Select, Tag, Tooltip, Typography } from 'antd';
import { useGroupTree, useCommandSubFields, useSearchCommands } from '@core/hooks/api/useMmlConsole';
import type { GroupTreeNode } from '@core/types/mmlConsole';
import { useT } from '@/hooks/useT';
import { useI18nText } from '@/hooks/useI18nText';

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
  lang,
  disabled,
}: PathPickerProps): JSX.Element {
  const t = useT();
  const { locale: appLocale } = useI18nText();
  const effectiveLang = lang ?? appLocale;
  const { data: tree = [] } = useGroupTree(undefined, effectiveLang);

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
      subFields.map((sf) => {
        // path 主文 + 中文 description 灰副文（Cascader / Select 不支持复杂 label
        // 节点，这里用 " — descr" 拼接，AutoComplete 同款）
        // 2026-05-27 后端 ListEnrichedByCommand 已物理过滤 is_supported=false 行,
        // 不再需要 "[不支持]" 后缀。
        const label = sf.description ? `${sf.tr069Path} — ${sf.description}` : sf.tr069Path;
        return {
          value: sf.tr069Path,
          label,
        };
      }),
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
  const handleRemove = (p: string) => {
    onChange(value.filter((x) => x !== p));
  };
  const handleClear = () => onChange([]);

  // 路径多选框的当前 value：当前命令下已被外层 value 选中的 path 子集
  // —— 用来在下拉里把已勾选项打 ✓，同时关闭下拉时 onChange 会拿到新增/减少的全量
  const pathOptionValues = useMemo(() => pathOptions.map((o) => o.value), [pathOptions]);
  const selectedInCurrentCommand = useMemo(
    () => pathOptionValues.filter((p) => value.includes(p)),
    [pathOptionValues, value],
  );
  const handlePathSelectChange = (picked: string[]) => {
    // picked = 当前命令下用户最新勾选的全集。
    // 把"其它命令的已选 path"（外层 value 减去当前命令所有 path 候选）保留，
    // 再追加当前 picked，构成新的总 value。
    const otherCommandPaths = value.filter((p) => !pathOptionValues.includes(p));
    const merged = Array.from(new Set([...otherCommandPaths, ...picked]));
    onChange(merged);
  };

  // 搜索框是「跨命令」的全局多选：value 反映外层 value 中命中过搜索结果的子集，
  // 让用户能再次 reopen 时看到已勾选状态；options 始终是后端搜索响应聚合后的 path。
  const searchSelectValue = useMemo(
    () => value.filter((p) => searchPathOptions.some((o) => o.value === p)),
    [value, searchPathOptions],
  );
  const handleSearchSelectChange = (picked: string[]) => {
    // picked = 当前搜索结果集合内最新勾选的全集；其它不在搜索候选里的 path 保留。
    const searchPathValues = new Set(searchPathOptions.map((o) => o.value));
    const otherPaths = value.filter((p) => !searchPathValues.has(p));
    onChange(Array.from(new Set([...otherPaths, ...picked])));
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      {/* Row 1：搜索 path（多选，跨命令） — 置顶，用户优先输入关键词找 path */}
      <Select
        size="small"
        style={{ width: '100%' }}
        mode="multiple"
        placeholder={t('mml.console.pathPicker.searchPlaceholder')}
        options={searchPathOptions}
        // 受控搜索词，结合 debounce 把网络请求降到 300ms 后才发
        searchValue={searchText}
        onSearch={setSearchText}
        // 多选 value = 已选 path ∩ 搜索结果候选；保证下拉里命中项带 ✓
        value={searchSelectValue}
        onChange={(picked) => handleSearchSelectChange(picked as string[])}
        // 关闭客户端二次过滤：所有匹配由后端 ILIKE 完成
        filterOption={false}
        disabled={disabled}
        notFoundContent={
          debouncedQuery && searchPathOptions.length === 0 ? (
            <Empty description={false} image={Empty.PRESENTED_IMAGE_SIMPLE} />
          ) : null
        }
        maxTagCount={0}
        maxTagPlaceholder={(omitted) =>
          omitted.length > 0
            ? `${t('mml.console.pathPicker.searchPlaceholder')} · ${omitted.length}`
            : null
        }
        allowClear
      />

      {/* Row 2：分组 + 命令 两个独立选择器并排（三级筛选起点） */}
      <div style={{ display: 'flex', gap: 8, width: '100%' }}>
        <Select
          size="small"
          style={{ flex: 1, minWidth: 0 }}
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
          style={{ flex: 1, minWidth: 0 }}
          placeholder={t('mml.console.pathPicker.commandPlaceholder')}
          options={commandOptions}
          value={commandId}
          onChange={setCommandId}
          disabled={disabled || !groupCode}
          showSearch
          optionFilterProp="label"
          allowClear
        />
      </div>

      {/* Row 3：当前命令下的路径多选（已选 path 默认带 ✓） */}
      {commandId && (
        <Select
          size="small"
          style={{ width: '100%' }}
          mode="multiple"
          placeholder={t('mml.console.pathPicker.pathPlaceholder')}
          options={pathOptions}
          // value 反映当前命令下已选 path 子集 → 下拉列表里这些项默认带 ✓
          value={selectedInCurrentCommand}
          onChange={(picked) => handlePathSelectChange(picked as string[])}
          disabled={disabled}
          optionFilterProp="label"
          // 不在 selector 里塞重复 Tag —— 已选展示统一交给下方 Tag 区域
          maxTagCount={0}
          maxTagPlaceholder={(omitted) =>
            omitted.length > 0
              ? `${t('mml.console.pathPicker.pathPlaceholder')} · ${omitted.length}`
              : null
          }
          allowClear
        />
      )}

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
