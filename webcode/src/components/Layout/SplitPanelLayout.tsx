import { useState, useRef, useCallback } from 'react';
import type { ReactNode } from 'react';
import { Button } from 'antd';
import { UpOutlined, DownOutlined } from '@ant-design/icons';
import { useT } from '@/hooks/useT';
import { useThemeToken } from '@/hooks/useThemeToken';

interface Props {
  /** Top panel content */
  upper: ReactNode;
  /** Bottom panel content */
  lower: ReactNode;
  /** Initial split ratio (0–1) for the upper panel. Default: 0.6 */
  defaultSplitRatio?: number;
  /** Upper panel title for collapse toggle */
  upperTitle?: string;
  /** Lower panel title for collapse toggle */
  lowerTitle?: string;
}

/**
 * SplitPanelLayout — vertical top/bottom split with draggable divider.
 *
 * Both panels are independently collapsible.
 */
export default function SplitPanelLayout({
  upper,
  lower,
  defaultSplitRatio = 0.6,
  upperTitle = '上方面板',
  lowerTitle = '下方面板',
}: Props) {
  const t = useT();
  const token = useThemeToken();
  const [splitRatio, setSplitRatio] = useState(defaultSplitRatio);
  const [upperCollapsed, setUpperCollapsed] = useState(false);
  const [lowerCollapsed, setLowerCollapsed] = useState(false);
  // isDraggingState is used in render to disable CSS transitions while dragging.
  const [isDraggingState, setIsDraggingState] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  // isDraggingRef is a fast mutable flag used inside mousemove handlers
  // to avoid stale closure issues without triggering re-renders on every frame.
  const isDraggingRef = useRef(false);

  const handleDividerMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      isDraggingRef.current = true;
      setIsDraggingState(true);
      const startY = e.clientY;
      const startRatio = splitRatio;

      const onMouseMove = (moveEvent: MouseEvent) => {
        if (!isDraggingRef.current || !containerRef.current) return;
        const containerHeight = containerRef.current.getBoundingClientRect().height;
        const delta = moveEvent.clientY - startY;
        const newRatio = Math.min(
          0.85,
          Math.max(0.15, startRatio + delta / containerHeight),
        );
        setSplitRatio(newRatio);
      };

      const onMouseUp = () => {
        isDraggingRef.current = false;
        setIsDraggingState(false);
        document.removeEventListener('mousemove', onMouseMove);
        document.removeEventListener('mouseup', onMouseUp);
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
      };

      document.addEventListener('mousemove', onMouseMove);
      document.addEventListener('mouseup', onMouseUp);
      document.body.style.cursor = 'row-resize';
      document.body.style.userSelect = 'none';
    },
    [splitRatio],
  );

  const DIVIDER_HEIGHT = 8;
  const HEADER_HEIGHT = 32;

  return (
    <div
      ref={containerRef}
      style={{
        display: 'flex',
        flexDirection: 'column',
        width: '100%',
        height: '100%',
        overflow: 'hidden',
        gap: 0,
      }}
    >
      {/* Upper panel */}
      <div
        style={{
          height: upperCollapsed
            ? HEADER_HEIGHT
            : lowerCollapsed
              ? `calc(100% - ${HEADER_HEIGHT + DIVIDER_HEIGHT}px)`
              : `calc(${splitRatio * 100}% - ${DIVIDER_HEIGHT / 2}px)`,
          minHeight: HEADER_HEIGHT,
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden',
          background: token.colorBgContainer,
          borderRadius: 6,
          border: `1px solid ${token.colorBorderSecondary}`,
          transition: isDraggingState ? 'none' : 'height 200ms ease',
        }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            height: HEADER_HEIGHT,
            padding: '0 12px',
            borderBottom: upperCollapsed ? 'none' : `1px solid ${token.colorBorderSecondary}`,
            flexShrink: 0,
            background: token.colorBgLayout,
          }}
        >
          <span style={{ fontSize: 13, fontWeight: 500, color: token.colorTextHeading }}>
            {upperTitle}
          </span>
          <Button
            type="text"
            size="small"
            icon={upperCollapsed ? <DownOutlined /> : <UpOutlined />}
            onClick={() => setUpperCollapsed((v) => !v)}
            aria-label={upperCollapsed ? t('panel.expandTop') : t('panel.collapseTop')}
          />
        </div>
        {!upperCollapsed && (
          <div style={{ flex: 1, overflow: 'auto' }}>{upper}</div>
        )}
      </div>

      {/* Draggable divider */}
      {!upperCollapsed && !lowerCollapsed && (
        <div
          role="separator"
          aria-orientation="horizontal"
          aria-label="拖拽调整面板高度"
          onMouseDown={handleDividerMouseDown}
          style={{
            height: DIVIDER_HEIGHT,
            flexShrink: 0,
            cursor: 'row-resize',
            background: 'transparent',
            transition: 'background 150ms ease',
          }}
          onMouseEnter={(e) => {
            (e.currentTarget as HTMLElement).style.background = token.colorBorder;
          }}
          onMouseLeave={(e) => {
            if (!isDraggingRef.current) {
              (e.currentTarget as HTMLElement).style.background = 'transparent';
            }
          }}
        />
      )}

      {/* Lower panel */}
      <div
        style={{
          height: lowerCollapsed
            ? HEADER_HEIGHT
            : upperCollapsed
              ? `calc(100% - ${HEADER_HEIGHT + DIVIDER_HEIGHT}px)`
              : `calc(${(1 - splitRatio) * 100}% - ${DIVIDER_HEIGHT / 2}px)`,
          minHeight: HEADER_HEIGHT,
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden',
          background: token.colorBgContainer,
          borderRadius: 6,
          border: `1px solid ${token.colorBorderSecondary}`,
          transition: isDraggingState ? 'none' : 'height 200ms ease',
        }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            height: HEADER_HEIGHT,
            padding: '0 12px',
            borderBottom: lowerCollapsed ? 'none' : `1px solid ${token.colorBorderSecondary}`,
            flexShrink: 0,
            background: token.colorBgLayout,
          }}
        >
          <span style={{ fontSize: 13, fontWeight: 500, color: token.colorTextHeading }}>
            {lowerTitle}
          </span>
          <Button
            type="text"
            size="small"
            icon={lowerCollapsed ? <UpOutlined /> : <DownOutlined />}
            onClick={() => setLowerCollapsed((v) => !v)}
            aria-label={lowerCollapsed ? t('panel.expandBottom') : t('panel.collapseBottom')}
          />
        </div>
        {!lowerCollapsed && (
          <div style={{ flex: 1, overflow: 'auto' }}>{lower}</div>
        )}
      </div>
    </div>
  );
}
