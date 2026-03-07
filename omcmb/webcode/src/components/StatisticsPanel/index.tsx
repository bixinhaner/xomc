import React from 'react';
import { Divider, Typography } from 'antd';
import { useThemeToken } from '@/hooks/useThemeToken';

export interface StatisticItem {
  label: string;
  value: string | number;
  color?: string;
  onClick?: () => void;
}

export interface StatisticsPanelProps {
  items: StatisticItem[];
  style?: React.CSSProperties;
  bordered?: boolean;
}

const StatisticsPanel: React.FC<StatisticsPanelProps> = ({
  items,
  style,
  bordered = true,
}) => {
  const token = useThemeToken();

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'stretch',
        background: token.colorBgContainer,
        borderRadius: 8,
        border: bordered ? `1px solid ${token.colorBorderSecondary}` : 'none',
        padding: '8px 0',
        flexWrap: 'wrap',
        ...style,
      }}
    >
      {items.map((item, index) => (
        <React.Fragment key={`${item.label}-${index}`}>
          {index > 0 && (
            <Divider
              type="vertical"
              style={{ height: 'auto', margin: '4px 0' }}
            />
          )}
          <div
            style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              padding: '4px 20px',
              cursor: item.onClick ? 'pointer' : 'default',
              minWidth: 80,
              flex: 1,
              transition: 'background 0.2s',
              borderRadius: 4,
            }}
            onClick={item.onClick}
            onMouseEnter={(e) => {
              if (item.onClick) {
                (e.currentTarget as HTMLDivElement).style.background = token.colorBgLayout;
              }
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLDivElement).style.background = 'transparent';
            }}
          >
            {/* Value */}
            <span
              style={{
                fontSize: 20,
                fontWeight: 700,
                lineHeight: 1.3,
                color: item.color ?? token.colorTextHeading,
                display: 'block',
              }}
            >
              {item.value}
            </span>

            {/* Label */}
            <Typography.Text
              type="secondary"
              style={{ fontSize: 12, marginTop: 2, whiteSpace: 'nowrap' }}
            >
              {item.label}
            </Typography.Text>
          </div>
        </React.Fragment>
      ))}
    </div>
  );
};

export default StatisticsPanel;
