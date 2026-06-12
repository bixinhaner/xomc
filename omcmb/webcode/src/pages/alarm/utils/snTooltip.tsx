import React from 'react';
import { Tooltip } from 'antd';

/**
 * 告警列表 SN 列的规范渲染（issue #224）。
 *
 * 修复目标：超长 SN 此前用裸 <span> 直接铺开，会溢出固定宽度的 SN 列、
 * 在浅色主题下遮挡相邻列文字；而浏览器原生 title 提示在深色主题下又
 * 显示不全/不可读。
 *
 * 做法：
 * 1. 整个单元格用一个宽度受约束的 flex 容器（minWidth:0 / maxWidth:100%）承载，
 *    可选的"未读"小红点用 flex:none 固定占位，SN 文本用 flex:1 + minWidth:0
 *    使其能在 flex 行内收缩到列宽以内，超长用 CSS ellipsis 截断（不再溢出遮挡后列）；
 *    —— 关键：只有给文本元素 minWidth:0 + flex:1，flex 子项才会收缩而非按内容铺开，
 *       这是 round1 仅靠 maxWidth:100% 仍溢出的根因修复。
 * 2. 悬浮时用 antd Tooltip 渲染到 body portal 显示完整 SN —— Tooltip 跟随
 *    全局主题（浅/深）令牌，深浅主题下均完整可读，且不占据表格布局空间。
 *
 * 仅当文本非空时才挂 Tooltip，避免空 SN 出现一个空气泡。
 *
 * @param value SN 原始值
 * @param prefix 可选的前缀节点（如未读小红点 Badge），固定占位、不参与收缩
 */
export function renderSnWithTooltip(value: unknown, prefix?: React.ReactNode): React.ReactNode {
  const text = String(value ?? '');

  const textSpan = (
    <span
      style={{
        flex: '1 1 auto',
        minWidth: 0,
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        whiteSpace: 'nowrap',
      }}
    >
      {text}
    </span>
  );

  const inner = text === '' ? textSpan : <Tooltip title={text}>{textSpan}</Tooltip>;

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 4,
        maxWidth: '100%',
        minWidth: 0,
      }}
    >
      {prefix != null && <span style={{ flex: 'none', display: 'inline-flex' }}>{prefix}</span>}
      {inner}
    </div>
  );
}
