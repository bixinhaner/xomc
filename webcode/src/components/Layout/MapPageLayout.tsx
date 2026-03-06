import { useState, useCallback, useRef } from 'react';
import type { ReactNode } from 'react';
import { useThemeToken } from '@/hooks/useThemeToken';
import { useResponsive } from '@/hooks/useResponsive';

interface Props {
  /** Left panel content (device list, filters, etc.) */
  panel: ReactNode;
  /** Right map/canvas area */
  children: ReactNode;
  /** Initial panel width in pixels. Default: 320 */
  defaultPanelWidth?: number;
  /** Minimum panel width */
  minPanelWidth?: number;
  /** Maximum panel width */
  maxPanelWidth?: number;
  /** Whether to show the left panel. Default: true */
  panelVisible?: boolean;
}

/**
 * MapPageLayout — left panel (device list/filters) + right full-height map area.
 *
 * The divider is draggable; the panel can be hidden for a full-screen map view.
 */
export default function MapPageLayout({
  panel,
  children,
  defaultPanelWidth = 320,
  minPanelWidth = 200,
  maxPanelWidth = 600,
  panelVisible = true,
}: Props) {
  const token = useThemeToken();
  const { isMobile } = useResponsive();
  const [panelWidth, setPanelWidth] = useState(defaultPanelWidth);
  const isDragging = useRef(false);

  const handleDividerMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      isDragging.current = true;
      const startX = e.clientX;
      const startWidth = panelWidth;

      const onMouseMove = (moveEvent: MouseEvent) => {
        if (!isDragging.current) return;
        const delta = moveEvent.clientX - startX;
        const newWidth = Math.min(maxPanelWidth, Math.max(minPanelWidth, startWidth + delta));
        setPanelWidth(newWidth);
      };

      const onMouseUp = () => {
        isDragging.current = false;
        document.removeEventListener('mousemove', onMouseMove);
        document.removeEventListener('mouseup', onMouseUp);
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
      };

      document.addEventListener('mousemove', onMouseMove);
      document.addEventListener('mouseup', onMouseUp);
      document.body.style.cursor = 'col-resize';
      document.body.style.userSelect = 'none';
    },
    [panelWidth, minPanelWidth, maxPanelWidth],
  );

  const handleDividerTouchStart = useCallback(
    (e: React.TouchEvent) => {
      const touch = e.touches[0];
      if (!touch) return;
      isDragging.current = true;
      const startX = touch.clientX;
      const startWidth = panelWidth;

      const onTouchMove = (moveEvent: TouchEvent) => {
        if (!isDragging.current) return;
        const t = moveEvent.touches[0];
        if (!t) return;
        const delta = t.clientX - startX;
        const newWidth = Math.min(maxPanelWidth, Math.max(minPanelWidth, startWidth + delta));
        setPanelWidth(newWidth);
      };

      const onTouchEnd = () => {
        isDragging.current = false;
        document.removeEventListener('touchmove', onTouchMove);
        document.removeEventListener('touchend', onTouchEnd);
      };

      document.addEventListener('touchmove', onTouchMove, { passive: true });
      document.addEventListener('touchend', onTouchEnd);
    },
    [panelWidth, minPanelWidth, maxPanelWidth],
  );

  // Mobile: vertical stacking layout
  if (isMobile) {
    return (
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          width: '100%',
          height: '100%',
          overflow: 'hidden',
          background: token.colorBgLayout,
        }}
      >
        {panelVisible && (
          <div
            style={{
              maxHeight: 200,
              overflow: 'auto',
              borderBottom: `1px solid ${token.colorBorderSecondary}`,
              background: token.colorBgContainer,
              flexShrink: 0,
            }}
          >
            {panel}
          </div>
        )}
        <div
          style={{
            flex: 1,
            minHeight: 0,
            overflow: 'hidden',
            position: 'relative',
            borderRadius: 6,
            border: `1px solid ${token.colorBorderSecondary}`,
          }}
        >
          {children}
        </div>
      </div>
    );
  }

  return (
    <div
      style={{
        display: 'flex',
        width: '100%',
        height: '100%',
        overflow: 'hidden',
        background: token.colorBgLayout,
        gap: 0,
      }}
    >
      {/* Left panel */}
      {panelVisible && (
        <>
          <div
            style={{
              width: panelWidth,
              minWidth: minPanelWidth,
              flexShrink: 0,
              display: 'flex',
              flexDirection: 'column',
              overflow: 'hidden',
              background: token.colorBgContainer,
              borderRadius: '6px 0 0 6px',
              border: `1px solid ${token.colorBorderSecondary}`,
              borderRight: 'none',
            }}
          >
            {panel}
          </div>

          {/* Draggable divider */}
          <div
            role="separator"
            aria-orientation="vertical"
            aria-label="拖拽调整面板宽度"
            onMouseDown={handleDividerMouseDown}
            onTouchStart={handleDividerTouchStart}
            style={{
              width: 4,
              flexShrink: 0,
              cursor: 'col-resize',
              background: token.colorBorderSecondary,
              transition: 'background 150ms ease',
              zIndex: 1,
              touchAction: 'none',
            }}
            onMouseEnter={(e) => {
              (e.currentTarget as HTMLElement).style.background = token.colorPrimary;
              (e.currentTarget as HTMLElement).style.opacity = '0.6';
            }}
            onMouseLeave={(e) => {
              if (!isDragging.current) {
                (e.currentTarget as HTMLElement).style.background = '#f0f0f0';
                (e.currentTarget as HTMLElement).style.opacity = '1';
              }
            }}
          />
        </>
      )}

      {/* Map / canvas area — takes remaining space */}
      <div
        style={{
          flex: 1,
          minWidth: 0,
          overflow: 'hidden',
          position: 'relative',
          borderRadius: panelVisible ? '0 6px 6px 0' : 6,
          border: `1px solid ${token.colorBorderSecondary}`,
        }}
      >
        {children}
      </div>
    </div>
  );
}
