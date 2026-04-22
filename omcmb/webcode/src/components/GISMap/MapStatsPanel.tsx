/**
 * 地图统计面板组件
 * @module components/GISMap/MapStatsPanel
 */

import React from 'react';
import { useThemeToken } from '@/hooks/useThemeToken';
import { useT } from '@/hooks/useT';
import type { MapStats, DeviceStatus } from '@core/types/map';
import { DEVICE_STATUS_CONFIG, COLORS } from './constants';
import styles from './styles.module.css';

interface MapStatsPanelProps {
  /** 统计数据 */
  stats?: MapStats;
  /** 是否可见 */
  visible?: boolean;
}

/**
 * 地图统计面板
 * 根据 UI 设计图 GISMap_UI_Design_Main.svg
 */
const MapStatsPanel: React.FC<MapStatsPanelProps> = ({
  stats,
  visible = true,
}) => {
  const t = useT();
  const token = useThemeToken();

  if (!visible) return null;

  const totalCount = stats?.total ?? 0;
  const onlineActiveCount = stats?.statusCount?.onlineActive ?? 0;
  const onlineInactiveCount = stats?.statusCount?.onlineInactive ?? 0;
  const onlineCount = onlineActiveCount + onlineInactiveCount;
  const offlineCount = stats?.statusCount?.offline ?? 0;
  const alarmCount = stats?.alarmCount ?? 0;

  const containerStyle: React.CSSProperties = {
    position: 'absolute',
    right: 20,
    bottom: 24,
    zIndex: 100,
    background: token.colorBgContainer,
    borderRadius: 12,
    boxShadow: '0 4px 12px rgba(0,0,0,0.08)',
    border: `1px solid ${token.colorBorderSecondary}`,
    minWidth: 200,
    overflow: 'hidden',
  };

  const headerStyle: React.CSSProperties = {
    padding: '12px 16px',
    borderBottom: `1px solid ${token.colorBorderSecondary}`,
  };

  const bodyStyle: React.CSSProperties = {
    padding: '12px 16px',
  };

  const rowStyle: React.CSSProperties = {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 8,
  };

  const totalRowStyle: React.CSSProperties = {
    ...rowStyle,
    marginBottom: 12,
    paddingBottom: 12,
    borderBottom: `1px solid ${token.colorBorderSecondary}`,
  };

  const statusDotStyle = (color: string): React.CSSProperties => ({
    width: 12,
    height: 12,
    borderRadius: '50%',
    background: color,
    flexShrink: 0,
  });

  const formatNumber = (num: number): string => {
    if (num >= 10000) {
      return (num / 10000).toFixed(1) + 'w';
    }
    return num.toLocaleString();
  };

  return (
    <div style={containerStyle} className={styles.mapStatsPanel}>
      {/* Header */}
      <div style={headerStyle}>
        <span style={{ fontWeight: 600, fontSize: 14, color: token.colorTextHeading }}>
          {t('device.statistics')}
        </span>
      </div>

      {/* Body */}
      <div style={bodyStyle}>
        {/* Total */}
        <div style={totalRowStyle}>
          <span style={{ color: COLORS.textSecondary, fontSize: 12 }}>
            {t('device.totalDevices')}
          </span>
          <span style={{ fontSize: 22, fontWeight: 700, color: token.colorTextHeading }}>
            {formatNumber(totalCount)}
          </span>
        </div>

        {/* Online */}
        <div style={rowStyle}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <div style={statusDotStyle(DEVICE_STATUS_CONFIG.onlineActive.color)} />
            <span style={{ fontSize: 12, color: token.colorText }}>{t('status.online')}</span>
          </div>
          <span style={{ fontSize: 14, fontWeight: 600, color: DEVICE_STATUS_CONFIG.onlineActive.color }}>
            {formatNumber(onlineCount)}
          </span>
        </div>

        {/* Offline */}
        <div style={{ ...rowStyle, marginBottom: 0 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <div style={statusDotStyle(DEVICE_STATUS_CONFIG.offline.color)} />
            <span style={{ fontSize: 12, color: token.colorText }}>{t('status.offline')}</span>
          </div>
          <span style={{ fontSize: 14, fontWeight: 600, color: token.colorTextSecondary }}>
            {formatNumber(offlineCount)}
          </span>
        </div>

        {/* Alarms (if any) */}
        {alarmCount > 0 && (
          <div style={{ ...rowStyle, marginTop: 8, marginBottom: 0 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <div style={statusDotStyle(COLORS.alarmEnd)} />
              <span style={{ fontSize: 12, color: token.colorText }}>{t('alarm.count')}</span>
            </div>
            <span style={{ fontSize: 14, fontWeight: 600, color: COLORS.alarmEnd }}>
              {formatNumber(alarmCount)}
            </span>
          </div>
        )}
      </div>
    </div>
  );
};

export default MapStatsPanel;
