import type React from 'react';

/**
 * 系统设置页面统一样式常量
 * 用于保持所有设置页面的视觉一致性。
 * （样式常量与样式组件分离，满足 react-refresh/only-export-components：
 *  组件文件只导出组件，常量集中在本文件。）
 */

// 一级分组标题样式 - 带圆点标记
export const groupTitleStyle: React.CSSProperties = {
  fontSize: 14,
  fontWeight: 'bold',
  margin: '0 0 12px 0',
  paddingLeft: 12,
  position: 'relative',
  color: 'var(--color-neutral-700)',
};

// 一级分组容器样式
export const groupContainerStyle: React.CSSProperties = {
  marginBottom: 20,
};

// 二级分组标题样式
export const subGroupTitleStyle: React.CSSProperties = {
  fontSize: 13,
  fontWeight: 600,
  color: '#555',
  marginBottom: 12,
  marginTop: 16,
  paddingBottom: 8,
  borderBottom: '1px solid #f0f0f0',
};

// 分割线样式
export const dividerStyle: React.CSSProperties = {
  margin: '16px 0',
};

// 表单项行内样式
export const inlineFormItemStyle: React.CSSProperties = {
  marginBottom: 8,
};

// 描述文字样式
export const descTextStyle: React.CSSProperties = {
  color: 'rgba(0, 0, 0, 0.45)',
  fontSize: 12,
  marginTop: 4,
};
