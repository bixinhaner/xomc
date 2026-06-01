import React from 'react';
import { Card } from 'antd';
import {
  groupContainerStyle,
  groupTitleStyle,
  subGroupTitleStyle,
} from './styles.constants';

/**
 * 系统设置页面统一样式组件
 * 用于保持所有设置页面的视觉一致性。
 * 样式常量见 ./styles.constants.ts（本文件只导出组件）。
 */

// 一级分组组件 Props
interface SettingsGroupProps {
  title: string;
  children: React.ReactNode;
  style?: React.CSSProperties;
}

/**
 * 一级分组组件 - 带圆点标记的标题 + 内容区域
 */
export function SettingsGroup({ title, children, style }: SettingsGroupProps) {
  return (
    <div style={{ ...groupContainerStyle, ...style }}>
      <div style={groupTitleStyle}>
        <span style={{
          position: 'absolute',
          left: 0,
          top: '50%',
          transform: 'translateY(-50%)',
          width: 4,
          height: 14,
          backgroundColor: '#1890ff',
          borderRadius: 2,
        }} />
        {title}
      </div>
      <div style={{ paddingLeft: 12 }}>
        {children}
      </div>
    </div>
  );
}

// 二级分组组件 Props
interface SubGroupProps {
  title?: string;
  children: React.ReactNode;
  style?: React.CSSProperties;
  bordered?: boolean;
}

/**
 * 二级分组组件 - 可选标题 + 带边框的内容区域
 */
export function SubGroup({ title, children, style, bordered = false }: SubGroupProps) {
  return (
    <div style={{
      ...style,
      padding: bordered ? '16px 16px 16px 20px' : undefined,
      border: bordered ? '1px solid #f0f0f0' : undefined,
      borderRadius: bordered ? 4 : undefined,
      marginBottom: 16,
    }}>
      {title && <div style={subGroupTitleStyle}>{title}</div>}
      {children}
    </div>
  );
}

// 设置卡片组件 Props
interface SettingsCardProps {
  title?: string;
  children: React.ReactNode;
  style?: React.CSSProperties;
}

/**
 * 设置卡片组件 - 用于主要分组区域的卡片容器
 */
export function SettingsCard({ title, children, style }: SettingsCardProps) {
  return (
    <Card
      title={title ? <span style={{ fontSize: 14, fontWeight: 'bold' }}>{title}</span> : undefined}
      size="small"
      style={{ marginBottom: 16, ...style }}
      headStyle={{ borderBottom: '1px solid #f0f0f0', padding: '12px 16px' }}
      bodyStyle={{ padding: 16 }}
    >
      {children}
    </Card>
  );
}
