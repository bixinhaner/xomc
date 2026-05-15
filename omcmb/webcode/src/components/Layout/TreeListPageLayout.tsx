import { useRef, useState } from 'react';
import type { ReactNode } from 'react';
import { useThemeToken } from '@/hooks/useThemeToken';

interface Props {
  /** Tree panel content (left side) */
  tree: ReactNode;
  /** Main content (right side) */
  children: ReactNode;
  /** Initial width of the left tree panel in pixels. Default: 25% of container */
  defaultTreeWidth?: number;
  /** Minimum tree panel width in pixels */
  minTreeWidth?: number;
  /** Maximum tree panel width in pixels */
  maxTreeWidth?: number;
}

/**
 * TreeListPageLayout — resizable left tree panel + right content area.
 *
 * The divider between the two panels is draggable.
 */
export default function TreeListPageLayout({
  tree,
  children,
  defaultTreeWidth = 260,
  minTreeWidth = 160,
}: Props) {
  const token = useThemeToken();
  const [treeWidth] = useState(defaultTreeWidth);
  const containerRef = useRef<HTMLDivElement>(null);

  return (
    <div
      ref={containerRef}
      style={{
        display: 'flex',
        width: '100%',
        height: '100%',
        overflow: 'hidden',
        gap: 15,
      }}
    >
      {/* Left tree panel */}
      <div
        style={{
          width: treeWidth,
          minWidth: minTreeWidth,
          flexShrink: 0,
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden',
          background: token.colorBgContainer,
          borderRadius: 6,
          border: `1px solid ${token.colorBorderSecondary}`,
        }}
      >
        {tree}
      </div>

      {/* Right content panel */}
      <div
        style={{
          flex: 1,
          minWidth: 0,
          overflow: 'hidden',
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        {children}
      </div>
    </div>
  );
}
