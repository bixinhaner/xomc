import React from 'react';
import { Card } from 'antd';

export type SkeletonType = 'table' | 'card' | 'chart' | 'form';

export interface LoadingSkeletonProps {
  type?: SkeletonType;
  rows?: number;
  style?: React.CSSProperties;
}

const shimmerKeyframes = `
@keyframes omc-shimmer {
  0% { background-position: -200% 0; }
  100% { background-position: 200% 0; }
}
`;

const shimmerStyle: React.CSSProperties = {
  background: 'linear-gradient(90deg, #f5f5f5 25%, #ebebeb 50%, #f5f5f5 75%)',
  backgroundSize: '200% 100%',
  animation: 'omc-shimmer 1.5s infinite',
  borderRadius: 4,
};

// Inject shimmer animation once
if (typeof document !== 'undefined' && !document.getElementById('omc-shimmer-style')) {
  const style = document.createElement('style');
  style.id = 'omc-shimmer-style';
  style.textContent = shimmerKeyframes;
  document.head.appendChild(style);
}

const TableSkeleton: React.FC<{ rows: number }> = ({ rows }) => {
  return (
    <div style={{ padding: '0 0' }}>
      {/* Table header */}
      <div
        style={{
          display: 'flex',
          gap: 12,
          padding: '12px 16px',
          background: '#fafafa',
          borderBottom: '1px solid #f0f0f0',
          marginBottom: 0,
        }}
      >
        {[20, 140, 120, 100, 80, 100, 60].map((w, i) => (
          <div
            key={i}
            style={{ ...shimmerStyle, width: w, height: 16, flexShrink: 0 }}
          />
        ))}
      </div>

      {/* Table rows */}
      {Array.from({ length: rows }, (_, rowIdx) => (
        <div
          key={rowIdx}
          style={{
            display: 'flex',
            gap: 12,
            padding: '12px 16px',
            borderBottom: '1px solid #f9f9f9',
            background: rowIdx % 2 === 0 ? '#fff' : '#fafafa',
            alignItems: 'center',
          }}
        >
          {[20, 140, 120, 100, 80, 100, 60].map((w, i) => (
            <div
              key={i}
              style={{
                ...shimmerStyle,
                width: i === 0 ? 16 : w * (0.7 + Math.random() * 0.3),
                height: i === 0 ? 16 : 14,
                flexShrink: 0,
                borderRadius: i === 0 ? 2 : 4,
              }}
            />
          ))}
        </div>
      ))}
    </div>
  );
};

const CardSkeleton: React.FC = () => {
  return (
    <Card styles={{ body: { padding: 20 } }}>
      <div style={{ display: 'flex', gap: 16, alignItems: 'flex-start' }}>
        <div
          style={{
            ...shimmerStyle,
            width: 48,
            height: 48,
            borderRadius: 10,
            flexShrink: 0,
          }}
        />
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 8 }}>
          <div style={{ ...shimmerStyle, width: 80, height: 14 }} />
          <div style={{ ...shimmerStyle, width: 120, height: 28 }} />
          <div style={{ ...shimmerStyle, width: 60, height: 12 }} />
        </div>
      </div>
    </Card>
  );
};

const ChartSkeleton: React.FC = () => {
  return (
    <div style={{ padding: 16 }}>
      {/* Title */}
      <div style={{ ...shimmerStyle, width: 120, height: 16, marginBottom: 16 }} />

      {/* Chart area */}
      <div
        style={{
          position: 'relative',
          height: 200,
          display: 'flex',
          alignItems: 'flex-end',
          gap: 12,
          padding: '0 8px',
        }}
      >
        {/* Y-axis */}
        <div
          style={{
            position: 'absolute',
            left: 0,
            top: 0,
            bottom: 0,
            width: 2,
            background: '#f0f0f0',
          }}
        />

        {/* Bars */}
        {Array.from({ length: 8 }, (_, i) => {
          const h = 40 + Math.sin(i * 0.8) * 80 + 60;
          return (
            <div
              key={i}
              style={{
                ...shimmerStyle,
                flex: 1,
                height: h,
                borderRadius: '4px 4px 0 0',
              }}
            />
          );
        })}
      </div>

      {/* X-axis */}
      <div
        style={{
          height: 2,
          background: '#f0f0f0',
          marginBottom: 8,
          marginLeft: 8,
        }}
      />

      {/* X labels */}
      <div style={{ display: 'flex', gap: 12, paddingLeft: 8 }}>
        {Array.from({ length: 8 }, (_, i) => (
          <div key={i} style={{ ...shimmerStyle, flex: 1, height: 12 }} />
        ))}
      </div>
    </div>
  );
};

const FormSkeleton: React.FC<{ rows: number }> = ({ rows }) => {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 20, padding: 16 }}>
      {Array.from({ length: rows }, (_, i) => (
        <div key={i} style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          <div style={{ ...shimmerStyle, width: 80, height: 14 }} />
          <div style={{ ...shimmerStyle, width: '100%', height: 32, borderRadius: 6 }} />
        </div>
      ))}
    </div>
  );
};

const LoadingSkeleton: React.FC<LoadingSkeletonProps> = ({
  type = 'table',
  rows = 5,
  style,
}) => {
  return (
    <div style={style}>
      {type === 'table' && <TableSkeleton rows={rows} />}
      {type === 'card' && <CardSkeleton />}
      {type === 'chart' && <ChartSkeleton />}
      {type === 'form' && <FormSkeleton rows={rows} />}
    </div>
  );
};

export default LoadingSkeleton;
