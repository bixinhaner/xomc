/**
 * 地图悬浮提示卡片组件
 * @module components/GISMap/MapPopup
 * 根据 UI 设计图 GISMap_UI_Design_Main.svg 实现
 */

import React, { useState } from 'react';
import { useIntl } from 'react-intl';
import { useThemeToken } from '@/hooks/useThemeToken';
import type { MapDevice } from '@core/types/map';
import { DEVICE_STATUS_CONFIG, ALARM_BADGE_CONFIG } from './constants';

/** 告警级别颜色配置（label 通过 i18n key 在组件内动态获取） */
const SEVERITY_COLOR: Record<number, { i18nKey: string; color: string }> = {
  1: { i18nKey: 'alarm.severity.critical', color: '#FF4D4F' },
  2: { i18nKey: 'alarm.severity.major',    color: '#FA8C16' },
  3: { i18nKey: 'alarm.severity.minor',    color: '#FADB14' },
  4: { i18nKey: 'alarm.severity.warning',  color: '#1677FF' },
};

interface MapPopupProps {
  /** 设备数据 */
  device: MapDevice | null;
  /** 是否可见 */
  visible?: boolean;
  /** 弹窗位置 */
  position?: { x: number; y: number };
  /** 关闭回调 */
  onClose?: () => void;
  /** 点击告警跳转回调（G-09） */
  onAlarmClick?: (sn: string) => void;
}

/**
 * 地图悬浮提示卡片
 * 按照 UI 设计图 GISMap_UI_Design_Main.svg
 * - 260x180 卡片
 * - 左侧三角形箭头指向设备
 * - 顶部 Header 带 📍 图标
 * - 状态行：绿色圆点 + 在线 + | + 告警角标
 * - 详情行：序列号、设备组、地址、坐标
 * - 底部蓝色强调线
 */
const MapPopup: React.FC<MapPopupProps> = ({
  device,
  visible = true,
  position,
  onAlarmClick,
}) => {
  const token = useThemeToken();
  const intl = useIntl();
  const [alarmHovered, setAlarmHovered] = useState(false);

  if (!visible || !device) return null;

  const statusConfig = DEVICE_STATUS_CONFIG[device.status] || DEVICE_STATUS_CONFIG.offline;
  const hasAlarm = (device.alarmCount ?? 0) > 0;
  const severityCfg = SEVERITY_COLOR[device.highestAlarmSeverity ?? 1] ?? SEVERITY_COLOR[1];

  // 告警角标样式（用于无告警时的灰色占位）
  const alarmBadgeStyle: React.CSSProperties = {
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
    minWidth: 20,
    height: 20,
    borderRadius: 10,
    padding: '0 4px',
    background: `linear-gradient(180deg, ${ALARM_BADGE_CONFIG.gradientStart} 0%, ${ALARM_BADGE_CONFIG.gradientEnd} 100%)`,
    color: '#FFF',
    fontSize: 11,
    fontWeight: 700,
  };

  // 卡片容器样式
  const containerStyle: React.CSSProperties = {
    position: 'absolute',
    left: (position?.x ?? 0) + 15, // 箭头指向设备，所以偏移
    top: position?.y ?? 0,
    transform: 'translateY(-50%)',
    zIndex: 1000,
    display: 'flex',
    alignItems: 'center',
  };

  // 左侧箭头 (三角形)
  const arrowStyle: React.CSSProperties = {
    width: 0,
    height: 0,
    borderTop: '8px solid transparent',
    borderBottom: '8px solid transparent',
    borderRight: `10px solid #FFF`,
    filter: 'drop-shadow(-2px 0 2px rgba(0,0,0,0.08))',
    flexShrink: 0,
  };

  // 卡片样式
  const cardStyle: React.CSSProperties = {
    width: 260,
    background: '#FFF',
    borderRadius: 12,
    boxShadow: '0 4px 12px rgba(0,0,0,0.12)',
    border: '1px solid #E8E8E8',
    overflow: 'hidden',
    marginLeft: -1, // 与箭头重叠消除缝隙
  };

  // Header 样式
  const headerStyle: React.CSSProperties = {
    padding: '12px 16px',
    background: '#FAFAFA',
    borderBottom: '1px solid #F0F0F0',
  };

  // 状态行样式
  const statusRowStyle: React.CSSProperties = {
    padding: '16px 16px 12px',
    display: 'flex',
    alignItems: 'center',
    gap: 8,
  };

  // 分隔线样式
  const dividerStyle: React.CSSProperties = {
    height: 1,
    background: '#F0F0F0',
    margin: '0 16px',
  };

  // 详情区域样式
  const detailsStyle: React.CSSProperties = {
    padding: '12px 16px',
  };

  // 详情行样式
  const detailRowStyle: React.CSSProperties = {
    display: 'flex',
    marginBottom: 6,
    fontSize: 11,
  };

  // 标签样式
  const labelStyle: React.CSSProperties = {
    color: '#8C8C8C',
    width: 56,
    flexShrink: 0,
  };

  // 值样式
  const valueStyle: React.CSSProperties = {
    color: 'var(--color-neutral-800)',
    flex: 1,
  };

  return (
    <div style={containerStyle}>
      {/* 左侧箭头 */}
      <div style={arrowStyle} />

      {/* 卡片 */}
      <div style={cardStyle}>
        {/* Header */}
        <div style={headerStyle}>
          <span style={{ fontSize: 14, fontWeight: 600, color: 'var(--color-neutral-800)' }}>
            📍 {device.name}
          </span>
        </div>

        {/* 状态行 */}
        <div style={statusRowStyle}>
          {/* 状态圆点 */}
          <div
            style={{
              width: 12,
              height: 12,
              borderRadius: '50%',
              background: device.status !== 'offline'
                ? `linear-gradient(180deg, ${statusConfig.gradientStart} 0%, ${statusConfig.gradientEnd} 100%)`
                : statusConfig.color,
            }}
          />
          {/* 状态文字 */}
          <span style={{ fontSize: 12, color: statusConfig.color }}>
            {device.status === 'onlineActive'
              ? intl.formatMessage({ id: 'gis.status.onlineActive' })
              : device.status === 'onlineInactive'
                ? intl.formatMessage({ id: 'gis.status.onlineInactive' })
                : intl.formatMessage({ id: 'gis.status.offline' })}
          </span>

          {/* 分隔符 */}
          <span style={{ fontSize: 12, color: '#D9D9D9' }}>|</span>

          {/* 告警 */}
          <span style={{ fontSize: 12, color: 'var(--color-neutral-600)' }}>{intl.formatMessage({ id: 'alarm.activeAlarm' })}:</span>
          {hasAlarm ? (
            <span
              style={{
                fontSize: 12,
                fontWeight: 600,
                color: severityCfg.color,
                cursor: 'pointer',
                opacity: alarmHovered ? 0.75 : 1,
                transition: 'opacity 0.15s',
              }}
              title={intl.formatMessage({ id: 'gis.popup.alarmClickTip' })}
              onClick={() => onAlarmClick?.(device.sn)}
              onMouseEnter={() => setAlarmHovered(true)}
              onMouseLeave={() => setAlarmHovered(false)}
            >
              ⚠ {intl.formatMessage({ id: severityCfg.i18nKey })}
              ({device.highestSeverityAlarmCount ?? device.alarmCount})
            </span>
          ) : (
            <span style={{ ...alarmBadgeStyle, background: '#F0F0F0', color: '#8C8C8C' }}>0</span>
          )}
        </div>

        {/* 分隔线 */}
        <div style={dividerStyle} />

        {/* 详情 */}
        <div style={detailsStyle}>
          {/* 核心标识 */}
          <div style={detailRowStyle}>
            <span style={labelStyle}>{intl.formatMessage({ id: 'table.sn' })}:</span>
            <span style={{ ...valueStyle, fontFamily: 'monospace' }}>{device.sn}</span>
          </div>

          {/* 网络信息（常用） */}
          <div style={detailRowStyle}>
            <span style={labelStyle}>IP:</span>
            <span style={{ ...valueStyle, fontFamily: 'monospace' }}>
              {device.ip_address || <span style={{ color: '#BFBFBF' }}>--</span>}
            </span>
          </div>

          <div style={detailRowStyle}>
            <span style={labelStyle}>MAC:</span>
            <span style={{ ...valueStyle, fontFamily: 'monospace' }}>
              {device.mac || <span style={{ color: '#BFBFBF' }}>--</span>}
            </span>
          </div>

          {/* 可读名称 */}
          <div style={detailRowStyle}>
            <span style={labelStyle}>{intl.formatMessage({ id: 'device.name' })}:</span>
            <span style={valueStyle}>{device.device_name || <span style={{ color: '#BFBFBF' }}>--</span>}</span>
          </div>

          {/* 无线参数 */}
          <div style={detailRowStyle}>
            <span style={labelStyle}>PCI:</span>
            <span style={{ ...valueStyle, fontFamily: 'monospace' }}>
              {device.pci || <span style={{ color: '#BFBFBF' }}>--</span>}
            </span>
          </div>

          {/* UE 数 */}
          <div style={detailRowStyle}>
            <span style={labelStyle}>{intl.formatMessage({ id: 'device.ueCount' })}:</span>
            <span style={{ ...valueStyle, fontFamily: 'monospace', color: (device.ueCount ?? 0) > 0 ? '#52C41A' : '#8C8C8C' }}>
              {device.ueCount ?? 0}
            </span>
          </div>

          {/* 位置信息 */}
          {device.groupName && (
            <div style={detailRowStyle}>
              <span style={labelStyle}>{intl.formatMessage({ id: 'gis.popup.deviceGroup' })}:</span>
              <span style={valueStyle}>{device.groupName}</span>
            </div>
          )}

          {device.address && (
            <div style={detailRowStyle}>
              <span style={labelStyle}>{intl.formatMessage({ id: 'gis.popup.address' })}:</span>
              <span style={valueStyle}>{device.address}</span>
            </div>
          )}

          <div style={{ ...detailRowStyle, marginBottom: 0 }}>
            <span style={labelStyle}>{intl.formatMessage({ id: 'gis.popup.coordinates' })}:</span>
            <span style={{ ...valueStyle, fontFamily: 'monospace', fontSize: 11 }}>
              {device.lng.toFixed(4)}, {device.lat.toFixed(4)}
            </span>
          </div>
        </div>

        {/* 底部蓝色强调线 */}
        <div
          style={{
            height: 3,
            background: `linear-gradient(90deg, ${token.colorPrimary} 0%, ${token.colorPrimaryHover} 100%)`,
            borderRadius: '0 0 3px 3px',
          }}
        />
      </div>
    </div>
  );
};

// 性能 #15：地图 hover/平移会高频驱动父组件重渲染，props（device/position）
// 不变时用 memo 跳过卡片重渲染。
export default React.memo(MapPopup);
