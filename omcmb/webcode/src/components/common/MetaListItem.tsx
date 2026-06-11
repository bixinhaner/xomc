/**
 * MetaListItem —— antd6 `List` / `List.Item` / `List.Item.Meta` 已废弃后的等价替身。
 *
 * 仅复刻本项目实际用到的「水平 Meta 列表行」形态：左侧 avatar + 中间 title/description
 * + 右侧 actions。DOM 与间距取自 antd6 List 样式实现（item flex 两端对齐、meta
 * flex:1、avatar marginInlineEnd=padding、title h4 下边距 marginXXS、description 用
 * colorTextDescription、action marginInlineStart=marginXXL），保证迁移后视觉一致。
 *
 * 配套 MetaList 提供 size=small + 可选 bordered/header 的外框，复刻原 List 容器外观。
 */
import type { ReactNode } from 'react';
import { theme } from 'antd';

interface MetaListItemProps {
  avatar?: ReactNode;
  title?: ReactNode;
  description?: ReactNode;
  /** 右侧操作区（复刻 List.Item 的 actions：横排、首项无前导分隔） */
  actions?: ReactNode[];
  /** 行级内联样式（复刻原 List.Item 的 style，如背景高亮） */
  style?: React.CSSProperties;
  /** 是否末行 —— 末行不画底部分隔线（复刻 List split 默认行为） */
  last?: boolean;
}

/** 单行：复刻 size=small 的 .ant-list-item + .ant-list-item-meta + actions。 */
export function MetaListItem({ avatar, title, description, actions, style, last }: MetaListItemProps) {
  const { token } = theme.useToken();
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        padding: `${token.paddingContentVerticalSM}px ${token.paddingContentHorizontal}px`,
        color: token.colorText,
        borderBottom: last ? 'none' : `1px solid ${token.colorSplit}`,
        ...style,
      }}
    >
      <div style={{ display: 'flex', flex: 1, alignItems: 'flex-start', maxWidth: '100%' }}>
        {avatar != null && <div style={{ marginInlineEnd: token.padding }}>{avatar}</div>}
        {(title != null || description != null) && (
          <div style={{ flex: '1 0', width: 0, color: token.colorText }}>
            {title != null && (
              <h4
                style={{
                  margin: `0 0 ${token.marginXXS}px 0`,
                  color: token.colorText,
                  fontSize: token.fontSize,
                  lineHeight: token.lineHeight,
                }}
              >
                {title}
              </h4>
            )}
            {description != null && (
              <div style={{ color: token.colorTextDescription, fontSize: token.fontSize }}>
                {description}
              </div>
            )}
          </div>
        )}
      </div>
      {actions && actions.length > 0 && (
        <ul
          style={{
            display: 'flex',
            flex: '0 0 auto',
            margin: 0,
            marginInlineStart: token.marginXXL,
            padding: 0,
            listStyle: 'none',
            alignItems: 'center',
          }}
        >
          {actions.map((action, i) => (
            // 行内 key 仅用于复刻 antd 的 action li 包裹结构
            <li key={i} style={{ padding: 0 }}>
              {action}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

interface MetaListProps {
  /** 是否带外框 + 圆角（复刻 List bordered） */
  bordered?: boolean;
  /** 顶部 header（复刻 List header：paddingSM + headerBg 背景） */
  header?: ReactNode;
  style?: React.CSSProperties;
  children?: ReactNode;
}

/** 容器：复刻 List 外框（bordered/header）。条目分隔线由 MetaListItem 自带。 */
export function MetaList({ bordered, header, style, children }: MetaListProps) {
  const { token } = theme.useToken();
  return (
    <div
      style={{
        ...(bordered
          ? { border: `1px solid ${token.colorBorder}`, borderRadius: token.borderRadiusLG }
          : {}),
        ...style,
      }}
    >
      {header != null && (
        <div
          style={{
            padding: `${token.paddingSM}px ${token.paddingContentHorizontal}px`,
            borderBottom: `1px solid ${token.colorSplit}`,
          }}
        >
          {header}
        </div>
      )}
      {children}
    </div>
  );
}

export default MetaListItem;
