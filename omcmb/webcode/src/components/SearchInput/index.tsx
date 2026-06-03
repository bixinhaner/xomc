/**
 * SearchInput — 全站统一的列表搜索框(2026-06-03 用户决策)。
 *
 * 统一 UI:对齐 product/products 的 Input.Search 样式 —— allowClear + 默认宽度 320。
 * 统一提示:placeholder 过长被截断时,**hover 和 focus 都用 Tooltip 显示完整文案**
 *   (原生 title 只在 hover 生效,focus 不弹,故用 antd Tooltip trigger=['hover','focus'])。
 *
 * 用法:把页面里的 `<Input.Search .../>` 直接换成 `<SearchInput .../>`,其余 props 透传
 * (onSearch / value / onChange / enterButton 等)。宽度强制 320 以保证全站统一,
 * 调用处原有的 width 会被忽略(无需逐个删除)。
 */
import { Input, Tooltip } from 'antd';
import type { SearchProps } from 'antd/es/input/Search';

const DEFAULT_WIDTH = 320;

export default function SearchInput({
  placeholder,
  allowClear = true,
  style,
  ...rest
}: SearchProps) {
  const search = (
    <Input.Search
      placeholder={placeholder}
      allowClear={allowClear}
      {...rest}
      style={{ ...style, width: DEFAULT_WIDTH }}
    />
  );

  // 无 placeholder 时无需 Tooltip。
  if (placeholder == null || placeholder === '') return search;

  return (
    <Tooltip title={placeholder} trigger={['hover', 'focus']}>
      {search}
    </Tooltip>
  );
}
