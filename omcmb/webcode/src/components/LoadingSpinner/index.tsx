import React from 'react';
import { Spin } from 'antd';
import type { SpinProps } from 'antd/es/spin';

/**
 * LoadingSpinner - 统一的加载状态组件
 *
 * 用法：
 * 1. 空白区域加载（显示 tip + indicator）：
 *    <LoadingSpinner tip="加载中..." />
 *
 * 2. 嵌套模式（包裹实际内容）：
 *    <LoadingSpinner spinning={isLoading} tip="加载中...">
 *      {content}
 *    </LoadingSpinner>
 *
 * 3. 自定义容器样式：
 *    <LoadingSpinner tip="加载中..." style={{ height: 200 }} />
 *
 * 4. 自定义 indicator：
 *    <LoadingSpinner tip="加载中..." indicator={<LoadingOutlined spin />} />
 */
export interface LoadingSpinnerProps extends Omit<SpinProps, 'children'> {
  /** 加载提示文本，空字符串时不显示 tip */
  tip?: string;
  /** 是否显示加载状态（嵌套模式） */
  spinning?: boolean;
  /** 加载状态时显示的内容（可选，默认为空） */
  children?: React.ReactNode;
  /** 容器自定义类名（仅空区域模式有效） */
  className?: string;
  /** 容器自定义样式（仅空区域模式有效） */
  style?: React.CSSProperties;
}

const DEFAULT_CONTAINER_STYLE: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  width: '100%',
  height: '100%',
};

export function LoadingSpinner({
  tip,
  spinning = true,
  children,
  className,
  style,
  ...spinProps
}: LoadingSpinnerProps) {
  // 如果有子内容，使用嵌套模式
  if (children !== undefined) {
    return (
      <Spin spinning={spinning} description={tip || undefined} {...spinProps}>
        {children}
      </Spin>
    );
  }

  // 没有子内容时，使用空区域加载模式
  // 使用 <div> 而非 <span /> 以保持更好的语义
  const containerStyle = style ? { ...DEFAULT_CONTAINER_STYLE, ...style } : DEFAULT_CONTAINER_STYLE;

  return (
    <div className={className} style={containerStyle}>
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 8 }}>
        <Spin {...spinProps} />
        {tip ? <span style={{ whiteSpace: 'nowrap' }}>{tip}</span> : null}
      </div>
    </div>
  );
}

/**
 * 用法示例：
 *
 * // 全屏覆盖（替代 LoadingOverlay）
 * <LoadingSpinner tip="加载中..." style={{ minHeight: 200 }} />
 *
 * // 行内紧凑（替代 CompactLoading）
 * <Spin size="small" description="加载中..." />
 */
