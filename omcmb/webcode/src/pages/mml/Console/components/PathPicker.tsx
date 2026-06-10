/**
 * PathPicker — 自定义命令（私有 / 公共）的"参数路径"多选选择器
 *
 * 单一标准 PATH 下拉：复用 `useStandardParamsList`（standard_params 字典）。
 *   - 空查询即返回前 N 条标准 PATH，用户不输入也能直接下拉浏览/勾选；
 *   - 输入关键字则走后端 ILIKE 过滤（path + description）。
 * 勾选即追加到已选列表（去重）；已选 path 以 Tag 列表展示在下方，可点 × 单条移除。
 *
 * 复用：AddTemplateModal（新增 / 编辑公私命令）。
 */

import { useEffect, useMemo, useState } from 'react';
import { Empty, Select, Spin, Tag, Tooltip, Typography } from 'antd';
import { useStandardParamsList } from '@core/hooks/api/useMmlAdmin';
import { useT } from '@/hooks/useT';

export interface PathPickerProps {
  /** 已选 path 集合（受控）。 */
  value: string[];
  /** 选择变更回调（接收去重后的 path 数组）。 */
  onChange: (paths: string[]) => void;
  /** 语言；保留以兼容旧调用，当前不再使用（标准 PATH 字典与语言无关）。 */
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
  disabled,
}: PathPickerProps): JSX.Element {
  const t = useT();

  // 受控搜索词 + 300ms debounce，降低后端 RTT。
  const [searchText, setSearchText] = useState('');
  const [debouncedQuery, setDebouncedQuery] = useState('');
  useEffect(() => {
    const id = setTimeout(() => setDebouncedQuery(searchText.trim()), 300);
    return () => clearTimeout(id);
  }, [searchText]);

  // 标准 PATH 列表：空查询返回前 50 条（可不输入直接浏览选择），输入则后端 ILIKE 过滤。
  const { data, isLoading } = useStandardParamsList({ q: debouncedQuery, pageSize: 50 });

  const pathOptions = useMemo<Option[]>(() => {
    const seen = new Set<string>();
    const out: Option[] = [];
    (data?.items ?? []).forEach((it) => {
      if (seen.has(it.standardPath)) return;
      seen.add(it.standardPath);
      out.push({
        value: it.standardPath,
        label: it.description ? `${it.standardPath}  —  ${it.description}` : it.standardPath,
      });
    });
    return out;
  }, [data]);

  // 下拉 value = 已选 path ∩ 当前候选；保证候选里命中项带 ✓。其它已选 path 保留在下方 Tag。
  const selectValue = useMemo(
    () => value.filter((p) => pathOptions.some((o) => o.value === p)),
    [value, pathOptions],
  );
  const handleSelectChange = (picked: string[]) => {
    const optionValues = new Set(pathOptions.map((o) => o.value));
    const otherPaths = value.filter((p) => !optionValues.has(p));
    onChange(Array.from(new Set([...otherPaths, ...picked])));
  };

  // ----- 已选 path 操作 -----
  const handleRemove = (p: string) => {
    onChange(value.filter((x) => x !== p));
  };
  const handleClear = () => onChange([]);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      {/* 标准 PATH 下拉（多选 + 后端模糊搜索；空查询亦返回候选，可直接浏览选择） */}
      <Select
        size="small"
        style={{ width: '100%' }}
        mode="multiple"
        placeholder={t('mml.console.pathPicker.searchPlaceholder')}
        options={pathOptions}
        // 受控搜索词，结合 debounce 把网络请求降到 300ms 后才发
        searchValue={searchText}
        onSearch={setSearchText}
        // 多选 value = 已选 path ∩ 候选；保证下拉里命中项带 ✓
        value={selectValue}
        onChange={(picked) => handleSelectChange(picked as string[])}
        // 关闭客户端二次过滤：所有匹配由后端 ILIKE 完成
        filterOption={false}
        disabled={disabled}
        notFoundContent={
          isLoading ? (
            <Spin size="small" />
          ) : debouncedQuery && pathOptions.length === 0 ? (
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
