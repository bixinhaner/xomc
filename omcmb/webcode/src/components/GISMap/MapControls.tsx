/**
 * 地图控制按钮组件
 * @module components/GISMap/MapControls
 * 根据 UI 设计图 GISMap_UI_Design_Main.svg 实现
 */

import React from 'react';
import styles from './styles.module.css';

interface MapControlsProps {
  /** 放大回调 */
  onZoomIn: () => void;
  /** 缩小回调 */
  onZoomOut: () => void;
  /** 是否禁用放大 */
  zoomInDisabled?: boolean;
  /** 是否禁用缩小 */
  zoomOutDisabled?: boolean;
  /** 是否处于测距模式 */
  isMeasuring?: boolean;
  /** 测距按钮点击回调 */
  onMeasureToggle?: () => void;
}

/**
 * 地图缩放控制按钮
 * 按照 UI 设计图 GISMap_UI_Design_Main.svg:
 * - 44x88 容器
 * - 圆形按钮背景
 * - + 按钮在上，- 按钮在下
 */
const MapControls: React.FC<MapControlsProps> = ({
  onZoomIn,
  onZoomOut,
  zoomInDisabled = false,
  zoomOutDisabled = false,
  isMeasuring = false,
  onMeasureToggle,
}) => {
  const containerStyle: React.CSSProperties = {
    position: 'absolute',
    right: 24,
    top: 100,
    width: 44,
    height: 88,
    zIndex: 100,
    background: '#FFF',
    borderRadius: 12,
    boxShadow: '0 2px 8px rgba(0,0,0,0.08)',
    border: '1px solid #E8E8E8',
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    padding: '6px 0',
  };

  const buttonStyle = (disabled: boolean): React.CSSProperties => ({
    width: 32,
    height: 32,
    borderRadius: '50%',
    background: '#F5F5F5',
    border: 'none',
    cursor: disabled ? 'not-allowed' : 'pointer',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    margin: '6px 0',
    transition: 'background 0.2s',
    opacity: disabled ? 0.5 : 1,
  });

  const dividerStyle: React.CSSProperties = {
    width: 24,
    height: 1,
    background: '#F0F0F0',
  };

  return (
    <React.Fragment>
    <div style={containerStyle} className={styles.mapControls}>
      {/* Zoom In */}
      <button
        style={buttonStyle(zoomInDisabled)}
        onClick={zoomInDisabled ? undefined : onZoomIn}
        disabled={zoomInDisabled}
        onMouseEnter={(e) => {
          if (!zoomInDisabled) {
            e.currentTarget.style.background = '#E8E8E8';
          }
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.background = '#F5F5F5';
        }}
      >
        <span style={{ fontSize: 16, fontWeight: 600, color: 'var(--color-neutral-800)' }}>+</span>
      </button>

      <div style={dividerStyle} />

      {/* Zoom Out */}
      <button
        style={buttonStyle(zoomOutDisabled)}
        onClick={zoomOutDisabled ? undefined : onZoomOut}
        disabled={zoomOutDisabled}
        onMouseEnter={(e) => {
          if (!zoomOutDisabled) {
            e.currentTarget.style.background = '#E8E8E8';
          }
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.background = '#F5F5F5';
        }}
      >
        <span style={{ fontSize: 16, fontWeight: 600, color: 'var(--color-neutral-800)' }}>−</span>
      </button>
    </div>

    {/* 测距工具按钮 */}
    {onMeasureToggle && (
      <div
        style={{
          position: 'absolute',
          right: 24,
          top: 204,
          width: 44,
          height: 44,
          zIndex: 100,
          background: isMeasuring ? '#1677ff' : '#FFF',
          borderRadius: 12,
          boxShadow: '0 2px 8px rgba(0,0,0,0.08)',
          border: isMeasuring ? '1px solid #1677ff' : '1px solid #E8E8E8',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        <button
          title={isMeasuring ? '退出测距 (ESC)' : '测距工具'}
          style={{
            width: 32,
            height: 32,
            borderRadius: '50%',
            background: 'transparent',
            border: 'none',
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: isMeasuring ? '#fff' : 'var(--color-neutral-800)',
          }}
          onClick={onMeasureToggle}
          onMouseEnter={(e) => {
            if (!isMeasuring) e.currentTarget.style.background = 'rgba(0,0,0,0.06)';
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.background = 'transparent';
          }}
        >
          {/* 尺子图标（Material Design straighten，带刻度线） */}
          <svg viewBox="0 0 24 24" width="17" height="17" fill="currentColor">
            <path d="M21 6H3c-1.1 0-2 .9-2 2v8c0 1.1.9 2 2 2h18c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2zm0 10H3V8h2v4h2V8h2v4h2V8h2v4h2V8h2v4h2V8h2v8z"/>
          </svg>
        </button>
      </div>
    )}
  </React.Fragment>
  );
};

// 性能 #15：缩放按钮回调来自 useCallback、props 稳定，用 memo 避免随地图
// 视口状态频繁重渲染。
export default React.memo(MapControls);
