import React from 'react';
import { Badge, Tag } from 'antd';
import { useThemeToken } from '@/hooks/useThemeToken';

export type IndicatorStatus =
  | 'online'
  | 'offline'
  | 'warning'
  | 'error'
  | 'processing'
  | 'inactive'
  | 'locked';

export type IndicatorVariant = 'dot' | 'tag';

export interface StatusIndicatorProps {
  status: IndicatorStatus;
  text?: string;
  variant?: IndicatorVariant;
  style?: React.CSSProperties;
}

const STATUS_CONFIG: Record<
  IndicatorStatus,
  { color: string; tagColor: string; label: string; antdStatus?: 'success' | 'processing' | 'error' | 'warning' | 'default' }
> = {
  online: {
    color: '#52C41A',
    tagColor: 'success',
    label: '在线',
    antdStatus: 'success',
  },
  offline: {
    color: '#8C8C8C',
    tagColor: 'default',
    label: '离线',
    antdStatus: 'default',
  },
  warning: {
    color: '#FA8C16',
    tagColor: 'orange',
    label: '告警',
    antdStatus: 'warning',
  },
  error: {
    color: '#F5222D',
    tagColor: 'error',
    label: '故障',
    antdStatus: 'error',
  },
  processing: {
    color: 'var(--color-primary-600)',
    tagColor: 'processing',
    label: '处理中',
    antdStatus: 'processing',
  },
  inactive: {
    color: '#8C8C8C',
    tagColor: 'default',
    label: '未激活',
    antdStatus: 'default',
  },
  locked: {
    color: '#722ED1',
    tagColor: 'purple',
    label: '锁定',
    antdStatus: 'default',
  },
};

const StatusIndicator: React.FC<StatusIndicatorProps> = ({
  status,
  text,
  variant = 'dot',
  style,
}) => {
  const token = useThemeToken();
  const config = STATUS_CONFIG[status];
  const displayText = text ?? config.label;

  if (variant === 'tag') {
    return (
      <Tag color={config.tagColor} style={style}>
        {displayText}
      </Tag>
    );
  }

  // Dot variant
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 6,
        ...style,
      }}
    >
      {config.antdStatus === 'processing' ? (
        <Badge status="processing" color={config.color} />
      ) : (
        <span
          style={{
            display: 'inline-block',
            width: 8,
            height: 8,
            borderRadius: '50%',
            backgroundColor: config.color,
            flexShrink: 0,
          }}
        />
      )}
      <span style={{ fontSize: 13, color: token.colorText }}>{displayText}</span>
    </span>
  );
};

export default StatusIndicator;
