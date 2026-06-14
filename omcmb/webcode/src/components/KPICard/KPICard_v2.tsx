/**
 * KPICard v2.0 - UI/UX优化版本
 *
 * 优化内容：
 * 1. 告警卡片视觉优先级提升
 * 2. 添加脉冲动画（新告警时）
 * 3. 增强数值显示可读性
 * 4. 添加点击反馈动画
 * 5. 优化视觉层次
 */

import React, { useState } from 'react';
import { Card, Typography, Tooltip } from 'antd';
import { ArrowDownOutlined, ArrowUpOutlined, InfoCircleOutlined } from '@ant-design/icons';
import { useThemeToken } from '@/hooks/useThemeToken';
import { TiltCard, FloatingElement } from '@/components/Effects';

export interface KPICardProps {
  title: string;
  value: string | number;
  icon: React.ReactNode;
  iconBgColor?: string;
  iconColor?: string;
  trend?: 'up' | 'down' | 'stable';
  delta?: string | number;
  deltaLabel?: string;
  unit?: string;
  onClick?: () => void;
  loading?: boolean;
  minHeight?: number;
  /** 是否显示脉冲动画（用于新告警等场景） */
  pulse?: boolean;
  /** 额外的提示信息 */
  tooltip?: string;
  /** 状态标记（normal/warning/error） */
  status?: 'normal' | 'warning' | 'error';
}

const KPICard: React.FC<KPICardProps> = ({
  title,
  value,
  icon,
  iconBgColor = '#e6f7ff',
  iconColor,
  trend,
  delta,
  deltaLabel,
  unit,
  onClick,
  loading = false,
  minHeight,
  pulse = false,
  tooltip,
  status = 'normal',
}) => {
  const token = useThemeToken();
  const [isPressed, setIsPressed] = useState(false);

  // 根据状态确定趋势颜色
  const getTrendColor = () => {
    if (trend === 'up') return status === 'error' ? '#F5222D' : '#52C41A';
    if (trend === 'down') return status === 'error' ? '#52C41A' : '#F5222D';
    return token.colorTextSecondary;
  };

  const trendColor = getTrendColor();
  const TrendIcon = trend === 'up' ? ArrowUpOutlined : trend === 'down' ? ArrowDownOutlined : null;

  // 状态样式映射
  const statusStyles = {
    normal: {
      borderColor: 'transparent',
      boxShadow: undefined,
    },
    warning: {
      borderColor: '#FAAD14',
      boxShadow: `0 0 0 1px #FAAD1440`,
    },
    error: {
      borderColor: '#F5222D',
      boxShadow: `0 0 0 1px #F5222D40`,
    },
  };

  const currentStatusStyle = statusStyles[status];

  const cardContent = (
    <div
      style={{
        display: 'flex',
        alignItems: 'flex-start',
        gap: 16,
        opacity: loading ? 0.6 : 1,
      }}
    >
      {/* Icon area — floating with optional pulse */}
      <div style={{ position: 'relative' }}>
        {pulse && (
          <div
            style={{
              position: 'absolute',
              inset: -4,
              borderRadius: '50%',
              background: iconColor || token.colorPrimary,
              opacity: 0.2,
              animation: 'pulse 2s ease-in-out infinite',
            }}
          />
        )}
        <FloatingElement amplitude={4} speed={3000}>
          <div
            style={{
              width: 48,
              height: 48,
              borderRadius: 12,
              background: iconBgColor,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontSize: 22,
              color: iconColor ?? token.colorPrimary,
              flexShrink: 0,
              position: 'relative',
              zIndex: 1,
              border: `2px solid ${iconColor ?? token.colorPrimary}20`,
            }}
          >
            {icon}
          </div>
        </FloatingElement>
      </div>

      {/* Content */}
      <div style={{ flex: 1, minWidth: 0 }}>
        {/* Metric name with optional tooltip */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 4, marginBottom: 4 }}>
          <Typography.Text
            type="secondary"
            style={{ fontSize: 13, display: 'block', fontWeight: 500 }}
          >
            {title}
          </Typography.Text>
          {tooltip && (
            <Tooltip title={tooltip}>
              <InfoCircleOutlined style={{ fontSize: 12, color: token.colorTextSecondary }} />
            </Tooltip>
          )}
        </div>

        {/* Value row */}
        <div
          style={{
            display: 'flex',
            alignItems: 'baseline',
            gap: 6,
            flexWrap: 'wrap',
          }}
        >
          <span
            style={{
              fontSize: 28,
              fontWeight: 700,
              lineHeight: 1.1,
              color: token.colorTextHeading,
              fontFamily: '"SF Pro Display", -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
              fontFeatureSettings: '"tnum"',
            }}
          >
            {value}
          </span>
          {unit && (
            <Typography.Text type="secondary" style={{ fontSize: 13, fontWeight: 500 }}>
              {unit}
            </Typography.Text>
          )}
        </div>

        {/* Trend area */}
        {(trend || delta !== undefined) && (
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 4,
              marginTop: 6,
              minHeight: minHeight,
            }}
          >
            {TrendIcon && (
              <TrendIcon style={{ fontSize: 11, color: trendColor }} />
            )}
            {delta !== undefined && (
              <Typography.Text style={{ fontSize: 12, fontWeight: 600, color: trendColor }}>
                {delta}
              </Typography.Text>
            )}
            {deltaLabel && (
              <Typography.Text type="secondary" style={{ fontSize: 11 }}>
                {deltaLabel}
              </Typography.Text>
            )}
          </div>
        )}
      </div>
    </div>
  );

  return (
    <TiltCard shadow>
      <Card
        hoverable={Boolean(onClick)}
        onClick={onClick}
        style={{
          borderRadius: 12,
          cursor: onClick ? 'pointer' : 'default',
          userSelect: 'none',
          border: `1.5px solid ${currentStatusStyle.borderColor}`,
          boxShadow: currentStatusStyle.boxShadow,
          transition: 'all 0.2s cubic-bezier(0.4, 0, 0.2, 1)',
          transform: isPressed ? 'scale(0.98)' : undefined,
        }}
        styles={{
          body: { padding: '16px 20px' },
        }}
        onMouseDown={() => onClick && setIsPressed(true)}
        onMouseUp={() => setIsPressed(false)}
        onMouseLeave={() => setIsPressed(false)}
      >
        {cardContent}
      </Card>

      {/* Pulse animation keyframes */}
      <style>{`
        @keyframes pulse {
          0% { transform: scale(1); opacity: 0.2; }
          50% { transform: scale(1.3); opacity: 0.1; }
          100% { transform: scale(1); opacity: 0.2; }
        }
      `}</style>
    </TiltCard>
  );
};

/**
 * 告警专用 KPICard - 自动应用告警样式
 */
export function AlarmKPICard(props: Omit<KPICardProps, 'status' | 'pulse'>) {
  const hasActiveAlarms = Number(props.value) > 0;
  return (
    <KPICard
      {...props}
      status={hasActiveAlarms ? 'error' : 'normal'}
      pulse={hasActiveAlarms}
    />
  );
}

/**
 * 在线设备专用 KPICard - 自动应用健康样式
 */
export function OnlineDeviceKPICard(props: Omit<KPICardProps, 'status'>) {
  return <KPICard {...props} status="normal" />;
}

export default KPICard;
