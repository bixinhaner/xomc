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

import { useEffect, useMemo, useState, type UIEventHandler } from 'react';
import type React from 'react';
import { Empty, Select, Spin, Tag, Tooltip, Typography } from 'antd';
import {
  useStandardParamsInfiniteList,
  flattenStandardParamPages,
} from '@core/hooks/api/useMmlAdmin';
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
}: PathPickerProps): React.JSX.Element {
  const t = useT();

  // 受控搜索词 + 300ms debounce，降低后端 RTT。
  const [searchText, setSearchText] = useState('');
  const [debouncedQuery, setDebouncedQuery] = useState('');
  useEffect(() => {
    const id = setTimeout(() => setDebouncedQuery(searchText.trim()), 300);
    return () => clearTimeout(id);
  }, [searchText]);

  // 标准 PATH 列表：触底分页 load-more（修 #105：原单页 50 条数据不全）；输入则后端 ILIKE 过滤。
  const { data, isLoading, fetchNextPage, hasNextPage, isFetchingNextPage } =
    useStandardParamsInfiniteList({ q: debouncedQuery });

  // 候选 = 搜索结果（按 standardPath 去重）− 已选（#106 隐藏已选：选过的不再出现在下拉）。
  const pathOptions = useMemo<Option[]>(() => {
    const selected = new Set(value);
    const seen = new Set<string>();
    const out: Option[] = [];
    for (const it of flattenStandardParamPages(data?.pages)) {
      if (selected.has(it.standardPath) || seen.has(it.standardPath)) continue;
      seen.add(it.standardPath);
      out.push({
        value: it.standardPath,
        label: it.description ? `${it.standardPath}  —  ${it.description}` : it.standardPath,
      });
    }
    return out;
  }, [data, value]);

  // 选中即追加到已选（#105/#106 一词多选：autoClearSearchValue=false 下可从同一关键字
  // 结果连续选多个；候选已排除已选，故 picked 均为新选）。
  const handleSelectChange = (picked: string[]) => {
    onChange(Array.from(new Set([...value, ...picked])));
  };

  // 下拉触底续拉下一页（load-more）。
  const handlePopupScroll: UIEventHandler<HTMLDivElement> = (e) => {
    const el = e.currentTarget;
    if (
      hasNextPage &&
      !isFetchingNextPage &&
      el.scrollHeight - el.scrollTop - el.clientHeight < 32
    ) {
      void fetchNextPage();
    }
  };

  // ----- 已选 path 操作（逐个 Tag × 删除；#106 已去掉「一键清除」）-----
  const handleRemove = (p: string) => {
    onChange(value.filter((x) => x !== p));
  };

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
        onPopupScroll={handlePopupScroll}
        // 一词多选：选中不清搜索框，可从同一关键字结果连续选多个（#105/#106）；
        // 候选已排除已选（隐藏已选）。Select 自身不留标签（value 置空），已选在下方 Tag 逐个删。
        autoClearSearchValue={false}
        value={[]}
        onChange={(picked) => handleSelectChange(picked as string[])}
        // 关闭客户端二次过滤：所有匹配由后端 ILIKE 完成
        filterOption={false}
        disabled={disabled}
        notFoundContent={
          isLoading || isFetchingNextPage ? (
            <Spin size="small" />
          ) : debouncedQuery && pathOptions.length === 0 ? (
            <Empty description={false} image={Empty.PRESENTED_IMAGE_SIMPLE} />
          ) : null
        }
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
          </>
        )}
      </div>
    </div>
  );
}
