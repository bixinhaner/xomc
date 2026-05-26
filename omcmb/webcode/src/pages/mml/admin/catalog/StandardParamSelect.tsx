/**
 * StandardParamSelect — MML 配置管理"path 必须从 standard_params 选择"用户规则 #3 入口。
 *
 * 单选 / 多选两种模式（multiple prop）：
 *   - 单选：用于"新建命令 target_object" 等需要单条 path 的场景
 *   - 多选：用于"按 path 列表批量创建 sub_field"（用户规则 #4）
 *
 * 后端 search 走 ILIKE %q% on standard_path + description，前端 debounce 300ms
 * 减少 RTT。AntD Select 的 showSearch 配合 onSearch 触发后端检索；options 渲染
 * "path · description" 两行，方便 admin 辨识。
 *
 * 选中后回调 onChange(value, selectedRecords) —— records 含 standard_params
 * 全字段，调用方可直接用来做 autofill（用户规则 #4）。
 */
import { useState, useMemo } from 'react';
import { Select, Spin, Typography } from 'antd';
import type { SelectProps } from 'antd';
import { useStandardParamsList } from '@core/hooks/api/useMmlAdmin';
import type { StandardParamView } from '@core/types/mmlAdmin';
import { useT } from '@/hooks/useT';
import { useDebounce } from 'ahooks';

const { Text } = Typography;

export interface StandardParamSelectProps {
  value?: string | string[];
  onChange?: (value: string | string[], records: StandardParamView[]) => void;
  multiple?: boolean;
  /** 仅显示 entry_type='parameter' 还是包括 object。默认 'parameter'（sub_field 场景）。 */
  entryType?: 'parameter' | 'object' | '';
  placeholder?: string;
  disabled?: boolean;
  /** 排除已选 path ID（避免同命令下重复绑定）。可选。 */
  excludeIds?: string[];
}

export default function StandardParamSelect({
  value,
  onChange,
  multiple = false,
  entryType = 'parameter',
  placeholder,
  disabled,
  excludeIds = [],
}: StandardParamSelectProps) {
  const t = useT();
  const [search, setSearch] = useState('');
  const debouncedSearch = useDebounce(search, 300);

  const { data, isLoading } = useStandardParamsList({
    q: debouncedSearch,
    entryType: entryType || undefined,
    pageSize: 50,
  });

  const items = useMemo(() => {
    const all = data?.items ?? [];
    if (excludeIds.length === 0) return all;
    const excl = new Set(excludeIds);
    return all.filter((it) => !excl.has(it.id));
  }, [data, excludeIds]);

  // 记录 id → 完整 StandardParamView 映射，方便 onChange 回传完整记录给上层 autofill。
  const recordById = useMemo(() => {
    const m = new Map<string, StandardParamView>();
    for (const it of items) m.set(it.id, it);
    return m;
  }, [items]);

  const options: SelectProps['options'] = items.map((it) => ({
    value: it.id,
    label: (
      <div style={{ display: 'flex', flexDirection: 'column' }}>
        <Text code style={{ fontSize: 12 }}>{it.standardPath}</Text>
        {it.description && (
          <Text type="secondary" style={{ fontSize: 11 }}>{it.description}</Text>
        )}
      </div>
    ),
    // 给 search filter 用的纯文本（Select 默认按 label 文本匹配；这里 label 是 ReactNode，
    // 必须显式给 filterOption.text 或在 onSearch 走后端搜索时不依赖前端 filterOption）。
    title: `${it.standardPath} ${it.description ?? ''}`,
  }));

  const handleChange = (next: string | string[]) => {
    const arr = Array.isArray(next) ? next : [next];
    const records: StandardParamView[] = arr
      .map((id) => recordById.get(id))
      .filter((r): r is StandardParamView => r != null);
    onChange?.(next, records);
  };

  return (
    <Select
      mode={multiple ? 'multiple' : undefined}
      value={value}
      onChange={handleChange}
      onSearch={setSearch}
      filterOption={false} // 后端已过滤
      showSearch
      allowClear
      placeholder={placeholder ?? t('mml.admin.catalog.path.searchPlaceholder')}
      disabled={disabled}
      options={options}
      notFoundContent={isLoading ? <Spin size="small" /> : null}
      style={{ width: '100%' }}
      optionLabelProp="title" // 选中后输入框只显示 path 字符串，避免选 N 条后宽度爆炸
      maxTagCount={multiple ? 3 : undefined}
    />
  );
}
