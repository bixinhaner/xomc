/**
 * 地图统计面板组件 - 精致优化版
 * @module components/GISMap/MapStatsPanel
 *
 * 设计特点：
 * - 毛玻璃效果 + 饱和度增强
 * - 弹性动画效果
 * - 精致的装饰元素
 * - 流畅的交互反馈
 */

import React, { useState } from 'react';
import { useT } from '@/hooks/useT';
import type { MapStats } from '@core/types/map';

interface MapStatsPanelProps {
  /** 统计数据 */
  stats?: MapStats;
  /** 是否可见 */
  visible?: boolean;
  /** 点击「UE=0 基站」统计行的回调 */
  onUEZeroClick?: () => void;
}

/**
 * 状态数据类型 */
interface StatusItem {
  key: string;
  label: string;
  count: number;
  color: string;
  gradient: string;
  bg: string;
  textColor: string;
}

// ========== 样式函数（移到组件外部避免重复创建） ==========

/** 状态项样式 */
const getStatusItemStyle = (item: StatusItem): React.CSSProperties => ({
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
  padding: '12px 14px',
  marginBottom: 8,
  background: item.bg,
  borderRadius: 12,
  border: '1px solid rgba(0, 0, 0, 0.02)',
  transition: 'all 0.25s cubic-bezier(0.34, 1.56, 0.64, 1)',
  cursor: 'default',
});

/** 状态圆点样式 */
const getStatusDotStyle = (color: string): React.CSSProperties => ({
  width: 10,
  height: 10,
  borderRadius: '50%',
  background: color,
  boxShadow: `0 0 0 4px ${color}12, 0 2px 6px ${color}25`,
});

/** 图标按钮样式 */
const iconButtonStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  width: 28,
  height: 28,
  borderRadius: 8,
  background: 'rgba(0, 0, 0, 0.03)',
  transition: 'all 0.2s ease',
};

/** 箭头样式 */
const arrowStyle: React.CSSProperties = {
  fontSize: 9,
  color: '#8C8C8C',
  transition: 'transform 0.35s cubic-bezier(0.34, 1.56, 0.64, 1)',
};

/** 内容区域样式 */
const bodyStyle: React.CSSProperties = {
  padding: '14px 18px 18px',
};

/** 总数卡片样式 */
const totalCardStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
  padding: '18px 20px',
  marginBottom: 14,
  background: 'linear-gradient(145deg, #40A9FF 0%, #1890FF 50%, #096DD9 100%)',
  borderRadius: 14,
  boxShadow: '0 6px 20px rgba(24, 144, 255, 0.3), inset 0 1px 0 rgba(255, 255, 255, 0.2)',
  position: 'relative',
  overflow: 'hidden',
};

/** 装饰圆环样式 */
const decorRingStyle: React.CSSProperties = {
  position: 'absolute',
  top: -20,
  right: -20,
  width: 80,
  height: 80,
  borderRadius: '50%',
  background: 'radial-gradient(circle, rgba(255,255,255,0.15) 0%, transparent 70%)',
};

/** 图标容器样式 */
const iconContainerStyle: React.CSSProperties = {
  width: 48,
  height: 48,
  borderRadius: 14,
  background: 'rgba(255,255,255,0.18)',
  backdropFilter: 'blur(10px)',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  position: 'relative',
  zIndex: 1,
};

/** 折叠状态容器样式 */
const collapsedBadgeStyle: React.CSSProperties = {
  position: 'absolute',
  top: -5,
  right: -5,
  minWidth: 22,
  height: 22,
  borderRadius: 11,
  background: 'white',
  color: '#1890FF',
  fontSize: 11,
  fontWeight: 700,
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  padding: '0 7px',
  boxShadow: '0 2px 8px rgba(0, 0, 0, 0.15), 0 0 0 1px rgba(24, 144, 255, 0.1)',
  border: '2px solid white',
};

/**
 * 地图统计面板 - 现代化设计
 */
const MapStatsPanel: React.FC<MapStatsPanelProps> = ({
  stats,
  visible = true,
  onUEZeroClick,
}) => {
  const t = useT();
  const [isExpanded, setIsExpanded] = useState(true);

  if (!visible) return null;

  const totalCount = stats?.total ?? 0;
  const onlineActiveCount = stats?.statusCount?.onlineActive ?? 0;
  const onlineInactiveCount = stats?.statusCount?.onlineInactive ?? 0;
  const onlineCount = onlineActiveCount + onlineInactiveCount;
  const offlineCount = stats?.statusCount?.offline ?? 0;
  const alarmCount = stats?.alarmCount ?? 0;

  // 状态列表数据
  const statusItems: StatusItem[] = [
    {
      key: 'online',
      label: t('status.online'),
      count: onlineCount,
      color: '#52C41A',
      gradient: 'linear-gradient(135deg, #73D13D 0%, #52C41A 100%)',
      bg: 'linear-gradient(135deg, rgba(115, 209, 61, 0.08) 0%, rgba(82, 196, 26, 0.06) 100%)',
      textColor: '#389e0d',
    },
    {
      key: 'offline',
      label: t('status.offline'),
      count: offlineCount,
      color: '#8C8C8C',
      gradient: 'linear-gradient(135deg, #BFBFBF 0%, #8C8C8C 100%)',
      bg: 'linear-gradient(135deg, rgba(140, 140, 140, 0.08) 0%, rgba(107, 107, 107, 0.06) 100%)',
      textColor: '#595959',
    },
  ];

  // 添加告警（如果有）
  if (alarmCount > 0) {
    statusItems.push({
      key: 'alarm',
      label: t('alarm.count'),
      count: alarmCount,
      color: '#FF4D4F',
      gradient: 'linear-gradient(135deg, #FF7875 0%, #F5222D 100%)',
      bg: 'linear-gradient(135deg, rgba(255, 120, 117, 0.08) 0%, rgba(245, 34, 45, 0.06) 100%)',
      textColor: '#cf1322',
    });
  }

  const formatNumber = (num: number): string => {
    if (num >= 10000) return (num / 10000).toFixed(1) + 'w';
    if (num >= 1000) return (num / 1000).toFixed(1) + 'k';
    return num.toString();
  };

  // 在线比例（用于显示）
  const onlinePercent = totalCount > 0
    ? Math.round((onlineActiveCount + onlineInactiveCount) / totalCount * 100)
    : 0;

  // 容器样式（依赖 isExpanded 状态，保留在组件内）
  const containerStyle: React.CSSProperties = {
    position: 'absolute',
    right: 20,
    bottom: 24,
    zIndex: 100,
    background: 'rgba(255, 255, 255, 0.92)',
    backdropFilter: 'blur(20px) saturate(180%)',
    borderRadius: 16,
    boxShadow: '0 8px 32px rgba(0, 0, 0, 0.1), 0 1px 4px rgba(0, 0, 0, 0.04), inset 0 1px 0 rgba(255, 255, 255, 0.8)',
    border: '1px solid rgba(255, 255, 255, 0.6)',
    minWidth: isExpanded ? 248 : 52,
    maxWidth: isExpanded ? 288 : 52,
    overflow: 'hidden',
    transition: 'all 0.35s cubic-bezier(0.34, 1.56, 0.64, 1)',
  };

  // 头部样式（依赖 isExpanded 状态，保留在组件内）
  const headerStyle: React.CSSProperties = {
    padding: isExpanded ? '18px 20px 14px' : '14px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    cursor: 'pointer',
    userSelect: 'none',
    borderBottom: isExpanded ? '1px solid rgba(0, 0, 0, 0.04)' : 'none',
    background: isExpanded
      ? 'linear-gradient(180deg, rgba(24, 144, 255, 0.04) 0%, rgba(24, 144, 255, 0.01) 50%, transparent 100%)'
      : 'transparent',
  };

  return (
    <div
      style={containerStyle}
      className="map-stats-panel"
      onMouseEnter={(e) => {
        if (!isExpanded) {
          Object.assign(e.currentTarget.style, {
            transform: 'scale(1.08) rotate(2deg)',
            boxShadow: '0 16px 48px rgba(24, 144, 255, 0.15), 0 2px 8px rgba(0, 0, 0, 0.06)',
          });
        }
      }}
      onMouseLeave={(e) => {
        if (!isExpanded) {
          Object.assign(e.currentTarget.style, {
            transform: 'scale(1) rotate(0deg)',
            boxShadow: '0 8px 32px rgba(0, 0, 0, 0.1), 0 1px 4px rgba(0, 0, 0, 0.04)',
          });
        }
      }}
    >
      {/* Header */}
      <div
        style={headerStyle}
        onClick={() => setIsExpanded(!isExpanded)}
        title={isExpanded ? t('common.collapse') : t('common.expand')}
      >
        {isExpanded ? (
          <>
            <div style={{ fontSize: 14, fontWeight: 600, color: '#1a1a1a', display: 'flex', alignItems: 'center', gap: 8, letterSpacing: '0.2px' }}>
              <div style={{
                width: 20,
                height: 20,
                borderRadius: 8,
                background: 'linear-gradient(135deg, #E6F7FF 0%, #BAE7FF 100%)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}>
                <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
                  <rect x="1" y="1" width="10" height="10" rx="2" stroke="#1890FF" strokeWidth="1.5"/>
                  <path d="M4 6H8M6 4V8" stroke="#1890FF" strokeWidth="1.5" strokeLinecap="round"/>
                </svg>
              </div>
              {t('device.statistics')}
            </div>
            <div style={iconButtonStyle}>
              <span style={arrowStyle}>▼</span>
            </div>
          </>
        ) : (
          <div style={{ position: 'relative', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <div style={{
              width: 28,
              height: 28,
              borderRadius: 10,
              background: 'linear-gradient(135deg, #1890FF 0%, #096DD9 100%)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              boxShadow: '0 4px 12px rgba(24, 144, 255, 0.35), inset 0 1px 0 rgba(255, 255, 255, 0.3)',
            }}>
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                <path d="M4 7L8 11L12 7M4 4L8 8L12 4" stroke="white" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
              </svg>
            </div>
            <div style={collapsedBadgeStyle}>
              {totalCount > 999 ? '999+' : totalCount}
            </div>
          </div>
        )}
      </div>

      {/* Body */}
      {isExpanded && (
        <div style={bodyStyle}>
          {/* 总数卡片 */}
          <div style={totalCardStyle}>
            {/* 装饰圆环 */}
            <div style={decorRingStyle} />
            <div style={{ position: 'relative', zIndex: 1 }}>
              <div style={{ fontSize: 12, color: 'rgba(255,255,255,0.85)', marginBottom: 4, fontWeight: 500 }}>
                {t('device.totalDevices')}
              </div>
              <div style={{ fontSize: 32, fontWeight: 700, color: '#FFF', lineHeight: 1, letterSpacing: '-0.5px' }}>
                {formatNumber(totalCount)}
              </div>
              <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.6)', marginTop: 4 }}>
                在线 {onlinePercent}%
              </div>
            </div>
            <div style={iconContainerStyle}>
              <svg width="26" height="26" viewBox="0 0 26 26" fill="none">
                <circle cx="13" cy="13" r="10" stroke="rgba(255,255,255,0.9)" strokeWidth="2"/>
                <circle cx="13" cy="13" r="4" fill="rgba(255,255,255,0.9)"/>
                <circle cx="20" cy="7" r="3" fill="rgba(255,255,255,0.6)"/>
              </svg>
            </div>
          </div>

          {/* 状态列表 */}
          {statusItems.map((item) => (
            <div
              key={item.key}
              style={getStatusItemStyle(item)}
              onMouseEnter={(e) => {
                e.currentTarget.style.transform = 'translateX(6px) scale(1.02)';
                e.currentTarget.style.boxShadow = `0 4px 12px ${item.color}15`;
                e.currentTarget.style.borderColor = `${item.color}20`;
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.transform = 'translateX(0) scale(1)';
                e.currentTarget.style.boxShadow = 'none';
                e.currentTarget.style.borderColor = 'rgba(0, 0, 0, 0.02)';
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                <div style={getStatusDotStyle(item.color)} />
                <span style={{ fontSize: 14, color: '#4a4a4a', fontWeight: 500 }}>{item.label}</span>
              </div>
              <span style={{ fontSize: 16, fontWeight: 700, color: item.textColor }}>
                {formatNumber(item.count)}
              </span>
            </div>
          ))}

          {/* UE=0 基站统计行（可点击触发过滤） */}
          {(stats?.ueZeroCount ?? 0) > 0 && (
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                padding: '10px 14px',
                marginTop: 6,
                background: 'linear-gradient(135deg, rgba(250, 173, 20, 0.08) 0%, rgba(250, 173, 20, 0.04) 100%)',
                borderRadius: 12,
                border: '1px solid rgba(250, 173, 20, 0.15)',
                cursor: onUEZeroClick ? 'pointer' : 'default',
                transition: 'all 0.2s ease',
              }}
              onClick={onUEZeroClick}
              onMouseEnter={(e) => {
                if (onUEZeroClick) {
                  e.currentTarget.style.transform = 'translateX(4px)';
                  e.currentTarget.style.borderColor = 'rgba(250, 173, 20, 0.35)';
                }
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.transform = 'translateX(0)';
                e.currentTarget.style.borderColor = 'rgba(250, 173, 20, 0.15)';
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <div style={{ width: 8, height: 8, borderRadius: '50%', background: '#FAAD14' }} />
                <span style={{ fontSize: 13, color: '#7c5b00', fontWeight: 500 }}>UE=0 基站</span>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                <span style={{ fontSize: 15, fontWeight: 700, color: '#D48806' }}>
                  {formatNumber(stats?.ueZeroCount ?? 0)}
                </span>
                {onUEZeroClick && (
                  <span style={{ fontSize: 10, color: '#FAAD14' }}>›</span>
                )}
              </div>
            </div>
          )}

          {/* UE=0 基站统计行 */}
          {(stats?.ueZeroCount ?? 0) >= 0 && (
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                padding: '10px 14px',
                marginTop: 6,
                background: 'linear-gradient(135deg, rgba(250,219,20,0.08) 0%, rgba(250,173,20,0.06) 100%)',
                borderRadius: 10,
                border: '1px solid rgba(250,173,20,0.15)',
                cursor: onUEZeroClick ? 'pointer' : 'default',
                transition: 'all 0.2s ease',
              }}
              onClick={onUEZeroClick}
              onMouseEnter={(e) => {
                if (onUEZeroClick) {
                  e.currentTarget.style.transform = 'translateX(4px)';
                  e.currentTarget.style.borderColor = 'rgba(250,173,20,0.4)';
                }
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.transform = 'translateX(0)';
                e.currentTarget.style.borderColor = 'rgba(250,173,20,0.15)';
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                <div style={{ width: 10, height: 10, borderRadius: '50%', background: '#FAAD14', boxShadow: '0 0 0 3px rgba(250,173,20,0.2)' }} />
                <span style={{ fontSize: 13, color: '#4a4a4a', fontWeight: 500 }}>UE为0基站</span>
                {onUEZeroClick && (
                  <span style={{ fontSize: 11, color: '#FAAD14' }}>→</span>
                )}
              </div>
              <span style={{ fontSize: 15, fontWeight: 700, color: '#D48806' }}>
                {formatNumber(stats?.ueZeroCount ?? 0)}
              </span>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

// 性能 #15：stats 由 useMemo 派生、引用稳定，用 memo 避免地图 hover/平移
// 触发父组件重渲染时一并重算面板。
export default React.memo(MapStatsPanel);
