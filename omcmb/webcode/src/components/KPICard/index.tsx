import React from 'react';
import { Card, Skeleton, Typography } from 'antd';
import { ArrowDownOutlined, ArrowUpOutlined } from '@ant-design/icons';
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
  /** false 表示没有有效历史基线，此时仅显示 N/A，不显示伪造趋势 */
  hasComparison?: boolean;
  unavailableText?: string;
  unit?: string;
  onClick?: () => void;
  loading?: boolean;
  minHeight?: number;
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
  hasComparison,
  unavailableText = 'N/A',
  unit,
  onClick,
  loading,
  minHeight,
}) => {
  const token = useThemeToken();
  const trendColor =
    trend === 'up' ? '#52C41A' : trend === 'down' ? '#F5222D' : token.colorTextSecondary;

  const TrendIcon =
    trend === 'up' ? ArrowUpOutlined : trend === 'down' ? ArrowDownOutlined : null;

  return (
    <TiltCard
      shadow
      className="omc-kpi-card"
      style={{ width: '100%', height: '100%' }}
    >
      <Card
        hoverable={Boolean(onClick)}
        onClick={onClick}
        style={{
          borderRadius: 12,
          height: '100%',
          cursor: onClick ? 'pointer' : 'default',
          userSelect: 'none',
        }}
        styles={{
          body: { padding: '16px 20px' },
        }}
      >
        <div style={{ display: 'flex', alignItems: 'flex-start', gap: 16 }}>
          {/* Icon area — floating */}
          <FloatingElement amplitude={4} speed={3000}>
            <div
              style={{
                width: 48,
                height: 48,
                borderRadius: 10,
                background: iconBgColor,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: 22,
                color: iconColor ?? token.colorPrimary,
                flexShrink: 0,
              }}
            >
              {icon}
            </div>
          </FloatingElement>

        {/* Content */}
        <div style={{ flex: 1, minWidth: 0 }}>
          {/* Metric name */}
          <Typography.Text
            type="secondary"
            style={{ fontSize: 14, display: 'block', marginBottom: 4 }}
          >
            {title}
          </Typography.Text>

          {/* Value row */}
          <div
            style={{
              display: 'flex',
              alignItems: 'baseline',
              gap: 4,
              flexWrap: 'wrap',
            }}
          >
            {/* 加载期显示骨架占位，避免硬刷新（缓存为空）首帧闪现占位/旧数字。issue #370 */}
            {loading ? (
              <Skeleton.Button active size="small" style={{ width: 80, height: 36 }} />
            ) : (
              <span
                style={{
                  fontSize: 32,
                  fontWeight: 700,
                  lineHeight: 1.2,
                  color: token.colorTextHeading,
                }}
              >
                {value}
              </span>
            )}
            {!loading && unit && (
              <Typography.Text type="secondary" style={{ fontSize: 13 }}>
                {unit}
              </Typography.Text>
            )}
          </div>

          {/* Trend —— 加载期一并隐藏趋势/增量行（参照 v2），避免闪现旧 delta。issue #370 */}
          {!loading && (hasComparison === false || trend || delta !== undefined) && (
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 4,
                marginTop: 4,
                minHeight: minHeight,
              }}
            >
              {hasComparison === false ? (
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  {unavailableText}
                </Typography.Text>
              ) : (
                <>
                  {TrendIcon && (
                    <TrendIcon style={{ fontSize: 12, color: trendColor }} />
                  )}
                  {delta !== undefined && (
                    <Typography.Text style={{ fontSize: 12, color: trendColor }}>
                      {delta}
                    </Typography.Text>
                  )}
                  {deltaLabel && (
                    <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                      {deltaLabel}
                    </Typography.Text>
                  )}
                </>
              )}
            </div>
          )}
        </div>
      </div>
      </Card>
    </TiltCard>
  );
};

export default KPICard;
