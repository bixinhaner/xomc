/**
 * PathPicker — 自定义命令（私有 / 公共）的"参数路径"多选选择器
 *
 * 用户可通过两种方式维护已选 path（受控 value: string[]）：
 *
 *   A. 从标准 PATH 列表选（模糊搜索）— 复用 `useSearchCommands`：
 *      后端 ILIKE 联合搜命令 + path + description，下拉直接列匹配 path，
 *      勾选即追加到已选列表（去重）。
 *
 *   B. 手动输入完整 PATH — 直接键入完整 path 路径，回车或点「添加」追加，
 *      不受标准 PATH 列表约束（用于厂商私有扩展等标准库未收录的 path）。
 *
 * 顶部以 Segmented 在两种方式间切换；已选 path 以 Tag 列表展示在下方，可点 × 单条移除。
 *
 * 复用：AddTemplateModal（新增 / 编辑公私命令）。
 */

import { useEffect, useMemo, useState } from 'react';
import { Empty, Input, Segmented, Select, Tag, Tooltip, Typography } from 'antd';
import { useSearchCommands } from '@core/hooks/api/useMmlConsole';
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

/** PATH 录入方式：从标准列表选 / 手动输入完整路径。 */
type InputMode = 'standard' | 'manual';

export default function PathPicker({
  value,
  onChange,
  lang,
  disabled,
}: PathPickerProps): JSX.Element {
  const t = useT();

  const [mode, setMode] = useState<InputMode>('standard');

  // ----- 标准列表（模糊搜索）模式 state -----
  const [searchText, setSearchText] = useState('');
  const [debouncedQuery, setDebouncedQuery] = useState('');
  useEffect(() => {
    const id = setTimeout(() => setDebouncedQuery(searchText.trim()), 300);
    return () => clearTimeout(id);
  }, [searchText]);
  const { data: searchResults = [] } = useSearchCommands(debouncedQuery, lang);

  // 把 search 返回的 matched_paths 摊平为下拉 options：每条 path 一行，
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

  // ----- 手动输入模式 state -----
  const [manualText, setManualText] = useState('');
  const handleManualAdd = () => {
    const p = manualText.trim();
    if (!p) return;
    if (!value.includes(p)) onChange([...value, p]);
    setManualText('');
  };

  // ----- 已选 path 操作 -----
  const handleRemove = (p: string) => {
    onChange(value.filter((x) => x !== p));
  };
  const handleClear = () => onChange([]);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      {/* 录入方式切换：从标准列表选 / 手动输入 */}
      <Segmented<InputMode>
        size="small"
        value={mode}
        onChange={(v) => setMode(v)}
        disabled={disabled}
        options={[
          { label: t('mml.console.pathPicker.modeStandard'), value: 'standard' },
          { label: t('mml.console.pathPicker.modeManual'), value: 'manual' },
        ]}
      />

      {mode === 'standard' ? (
        /* 从标准 PATH 列表选（多选 + 后端模糊搜索） */
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
      ) : (
        /* 手动输入完整 PATH（回车或点「添加」追加） */
        <Input.Search
          size="small"
          style={{ width: '100%' }}
          placeholder={t('mml.console.pathPicker.manualPlaceholder')}
          value={manualText}
          onChange={(e) => setManualText(e.target.value)}
          onSearch={handleManualAdd}
          enterButton={t('mml.console.pathPicker.manualAdd')}
          disabled={disabled}
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
