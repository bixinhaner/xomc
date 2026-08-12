/**
 * 地图悬浮提示卡片组件
 * @module components/GISMap/MapPopup
 * 根据 UI 设计图 GISMap_UI_Design_Main.svg 实现
 */

import React, { useState } from 'react';
import { Button, Form, InputNumber, message, Tabs } from 'antd';
import { CloseOutlined } from '@ant-design/icons';
import { useIntl } from 'react-intl';
import { useThemeToken } from '@/hooks/useThemeToken';
import type { AntennaEditableField, AntennaSector, MapDevice } from '@core/types/map';
import { DEVICE_STATUS_CONFIG, ALARM_BADGE_CONFIG } from './constants';
import type { AntennaSectorRenderMode } from './antennaSectorRender';
import styles from './styles.module.css';

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
  /** 当前选中设备已解析的天线扇区。 */
  antennaSectors?: AntennaSector[];
  /** 编辑值变更时更新地图覆盖范围预览。 */
  onAntennaPreviewChange?: (sectorNumber: number, field: AntennaEditableField, value: number | null) => void;
  /** 放弃编辑值并恢复地图中的原始覆盖范围。 */
  onAntennaCancel?: () => void;
  /** 保存 OMC 本地天线规划参数。 */
  onAntennaSave?: (sectorNumber: number) => Promise<unknown>;
  antennaSaving?: boolean;
  /** 展示模式：地图 hover 提示或右侧完整详情。 */
  variant?: 'tooltip' | 'panel';
  /** 当前活动扇区，由地图容器统一维护以同步高亮。 */
  activeSectorNumber?: number;
  onActiveSectorChange?: (sectorNumber: number) => void;
  activeSectorRenderMode?: AntennaSectorRenderMode;
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
  onClose,
  onAlarmClick,
  antennaSectors = [],
  onAntennaPreviewChange,
  onAntennaCancel,
  onAntennaSave,
  antennaSaving = false,
  variant = 'tooltip',
  activeSectorNumber,
  onActiveSectorChange,
  activeSectorRenderMode,
}) => {
  const token = useThemeToken();
  const intl = useIntl();
  const [alarmHovered, setAlarmHovered] = useState(false);
  const [editingSector, setEditingSector] = useState(false);

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
  const isPanel = variant === 'panel';
  const containerStyle: React.CSSProperties = isPanel ? {
    width: '100%',
    height: '100%',
    display: 'flex',
  } : {
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
    width: isPanel ? '100%' : 288,
    height: isPanel ? '100%' : undefined,
    maxHeight: isPanel ? 'none' : '70vh',
    background: '#FFF',
    borderRadius: isPanel ? 0 : 12,
    boxShadow: isPanel ? '-4px 0 12px rgba(0,0,0,0.08)' : '0 4px 12px rgba(0,0,0,0.12)',
    border: '1px solid #E8E8E8',
    overflowY: 'auto',
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

  const activeSector = antennaSectors.find((sector) => sector.number === activeSectorNumber) ?? antennaSectors[0];
  const sectorLabelStyle: React.CSSProperties = {
    ...labelStyle,
    width: 'auto',
    whiteSpace: 'nowrap',
  };
  const sectorDetailRowStyle: React.CSSProperties = {
    ...detailRowStyle,
    display: 'grid',
    gridTemplateColumns: 'minmax(0, 1fr) max-content',
    columnGap: 12,
    alignItems: 'baseline',
    marginBottom: 7,
    fontSize: 12,
  };
  const sectorValueStyle: React.CSSProperties = {
    ...valueStyle,
    justifySelf: 'end',
    textAlign: 'right',
    color: '#262626',
    fontWeight: 500,
  };
  const canEdit = Boolean(activeSector && onAntennaSave);
  const editorFields: Array<{
    field: AntennaEditableField;
    label: string;
    min: number;
    max: number;
    unit: string;
  }> = [
    { field: 'azimuth', label: intl.formatMessage({ id: 'gis.antenna.azimuth' }), min: 0, max: 359, unit: '°' },
    { field: 'antennaHeight', label: intl.formatMessage({ id: 'gis.antenna.height' }), min: 0.01, max: 9999, unit: 'm' },
    { field: 'mechanicalDowntilt', label: intl.formatMessage({ id: 'gis.antenna.mechanicalDowntilt' }), min: 0, max: 89.99, unit: '°' },
    { field: 'horizontalBeamwidth', label: intl.formatMessage({ id: 'gis.antenna.horizontalBeamwidth' }), min: 0.01, max: 179.99, unit: '°' },
    { field: 'verticalBeamwidth', label: intl.formatMessage({ id: 'gis.antenna.verticalBeamwidth' }), min: 0.01, max: 179.99, unit: '°' },
  ];

  const saveAntennaSector = async () => {
    if (!activeSector || !onAntennaSave) return;
    try {
      await onAntennaSave(activeSector.number);
      setEditingSector(false);
      message.success(intl.formatMessage({ id: 'gis.antenna.updateSubmitted' }));
    } catch {
      message.error(intl.formatMessage({ id: 'gis.antenna.updateFailed' }));
    }
  };

  const cancelAntennaEditing = () => {
    onAntennaCancel?.();
    setEditingSector(false);
  };

  return (
    <div style={containerStyle}>
      {/* 左侧箭头 */}
      {!isPanel && <div style={arrowStyle} />}

      {/* 卡片 */}
      <div style={cardStyle}>
        {/* Header */}
        <div style={{ ...headerStyle, display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 8 }}>
          <span style={{ minWidth: 0, fontSize: 14, fontWeight: 600, color: 'var(--color-neutral-800)' }}>
            📍 {device.name}
          </span>
          {isPanel && (
            <Button
              type="text"
              size="small"
              icon={<CloseOutlined />}
              aria-label={intl.formatMessage({ id: 'common.close' })}
              onClick={onClose}
            />
          )}
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
              ({device.highestSeverityAlarmCount || device.alarmCount})
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

          {isPanel && (
            <>
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
            </>
          )}

          {/* 位置信息 */}
          {device.groupName && (
            <div style={detailRowStyle}>
              <span style={labelStyle}>{intl.formatMessage({ id: 'gis.popup.deviceGroup' })}:</span>
              <span style={valueStyle}>{device.groupName}</span>
            </div>
          )}

          {isPanel && device.address && (
            <div style={detailRowStyle}>
              <span style={labelStyle}>{intl.formatMessage({ id: 'gis.popup.address' })}:</span>
              <span style={valueStyle}>{device.address}</span>
            </div>
          )}

          {isPanel && (
            <div style={{ ...detailRowStyle, marginBottom: 0 }}>
              <span style={labelStyle}>{intl.formatMessage({ id: 'gis.popup.coordinates' })}:</span>
              <span style={{ ...valueStyle, fontFamily: 'monospace', fontSize: 11 }}>
                {device.lng.toFixed(4)}, {device.lat.toFixed(4)}
              </span>
            </div>
          )}

          {antennaSectors.length > 0 && (
            <div style={{ borderTop: '1px solid #F0F0F0', marginTop: 12, paddingTop: 10 }}>
              <div style={{ marginBottom: 8, color: 'var(--color-neutral-800)', fontSize: 12, fontWeight: 600 }}>
                {intl.formatMessage({ id: 'gis.antenna.popupTitle' })}
                  {canEdit && (
                    <Button type="link" size="small" onClick={() => editingSector ? cancelAntennaEditing() : setEditingSector(true)} style={{ float: 'right', padding: 0 }}>
                      {editingSector ? intl.formatMessage({ id: 'common.cancel' }) : intl.formatMessage({ id: 'common.edit' })}
                    </Button>
                  )}
              </div>
              <Tabs
                className={styles.sectorTabs}
                activeKey={String(activeSector.number)}
                onChange={(key) => {
                  setEditingSector(false);
                  onActiveSectorChange?.(Number(key));
                }}
                size="small"
                items={antennaSectors.map((sector) => ({
                  key: String(sector.number),
                  label: intl.formatMessage({ id: 'gis.antenna.sector' }, { number: sector.number }),
                }))}
              />
              <div>
                <div style={sectorDetailRowStyle}>
                  <span style={sectorLabelStyle}>{intl.formatMessage({ id: 'gis.antenna.azimuth' })}:</span>
                  <span style={sectorValueStyle}>{activeSector.azimuth === undefined ? '--' : `${activeSector.azimuth}°`}</span>
                </div>
                <div style={sectorDetailRowStyle}>
                  <span style={sectorLabelStyle}>{intl.formatMessage({ id: 'gis.antenna.height' })}:</span>
                  <span style={sectorValueStyle}>{activeSector.antennaHeight === undefined ? '--' : `${activeSector.antennaHeight} m`}</span>
                </div>
                <div style={sectorDetailRowStyle}>
                  <span style={sectorLabelStyle}>{intl.formatMessage({ id: 'gis.antenna.mechanicalDowntilt' })}:</span>
                  <span style={sectorValueStyle}>{activeSector.mechanicalDowntilt === undefined ? '--' : `${activeSector.mechanicalDowntilt}°`}</span>
                </div>
                <div style={sectorDetailRowStyle}>
                  <span style={sectorLabelStyle}>{intl.formatMessage({ id: 'gis.antenna.horizontalBeamwidth' })}:</span>
                  <span style={sectorValueStyle}>{activeSector.horizontalBeamwidth === undefined ? '--' : `${activeSector.horizontalBeamwidth}°`}</span>
                </div>
                <div style={sectorDetailRowStyle}>
                  <span style={sectorLabelStyle}>{intl.formatMessage({ id: 'gis.antenna.verticalBeamwidth' })}:</span>
                  <span style={sectorValueStyle}>{activeSector.verticalBeamwidth === undefined ? '--' : `${activeSector.verticalBeamwidth}°`}</span>
                </div>
                {activeSector.coverageAvailable ? (
                  <>
                    <div style={{ ...sectorDetailRowStyle, marginBottom: 0 }}>
                      <span style={sectorLabelStyle}>{intl.formatMessage({ id: 'gis.antenna.coverageRange' })}:</span>
                      <span style={sectorValueStyle}>{Math.round(activeSector.nearRadiusMeters ?? 0)} - {Math.round(activeSector.farRadiusMeters ?? 0)} m</span>
                    </div>
                    {activeSectorRenderMode === 'narrow' && (
                      <div style={{ marginTop: 8, color: '#0958d9', fontSize: 11 }}>
                        {intl.formatMessage({ id: 'gis.antenna.narrowBeam' })}
                      </div>
                    )}
                    {activeSectorRenderMode === 'unavailable' && (
                      <div style={{ marginTop: 8, color: '#d48806', fontSize: 11 }}>
                        {intl.formatMessage({ id: 'gis.antenna.zoomRequired' })}
                      </div>
                    )}
                  </>
                ) : activeSector.coverageStatus === 'invalid_geometry' ? (
                  <div style={{ color: '#cf1322', fontSize: 11 }}>
                    {intl.formatMessage(
                      { id: 'gis.antenna.invalidGeometry' },
                      { reason: intl.formatMessage({ id: `gis.antenna.coverageIssue.${activeSector.coverageIssue ?? 'unknown'}` }) },
                    )}
                  </div>
                ) : (
                  <div style={{ color: '#d48806', fontSize: 11 }}>
                    {intl.formatMessage({ id: 'gis.antenna.incompleteFields' }, { fields: activeSector.missingFields?.join(', ') || '--' })}
                  </div>
                )}
                {editingSector && (
                  <Form className={styles.sectorEditor} layout="vertical" size="small">
                    {editorFields.map(({ field, label, min, max, unit }) => (
                      <Form.Item key={field} label={label}>
                        <div className={styles.sectorEditorInput}>
                          <InputNumber
                            controls={false}
                            min={min}
                            max={max}
                            value={activeSector[field]}
                            aria-label={label}
                            onChange={(value) => onAntennaPreviewChange?.(activeSector.number, field, value)}
                          />
                          <span>{unit}</span>
                        </div>
                      </Form.Item>
                    ))}
                    <div className={styles.sectorEditorActions}>
                      <Button
                        type="primary"
                        loading={antennaSaving}
                        disabled={activeSector.coverageStatus === 'invalid_geometry'}
                        onClick={() => void saveAntennaSector()}
                      >
                        {intl.formatMessage({ id: 'common.save' })}
                      </Button>
                    </div>
                  </Form>
                )}
              </div>
            </div>
          )}
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
