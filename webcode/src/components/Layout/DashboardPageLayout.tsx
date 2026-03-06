import type { ReactNode } from 'react';
import { useResponsive } from '@/hooks/useResponsive';

interface Props {
  children: ReactNode;
  /** Minimum column width for auto grid. Default: 320px */
  minColumnWidth?: number;
  /** Gap between grid cells. Default: 16px */
  gap?: number;
}

/**
 * DashboardPageLayout — full-width auto-grid for widget/chart pages.
 *
 * Uses CSS Grid auto-fill with a configurable min column width so widgets
 * reflow naturally as the viewport resizes.
 */
export default function DashboardPageLayout({
  children,
  minColumnWidth = 320,
  gap = 16,
}: Props) {
  const { isMobile } = useResponsive();
  const effectiveMinWidth = isMobile ? Math.min(minColumnWidth, 280) : minColumnWidth;
  const effectiveGap = isMobile ? 8 : gap;

  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: `repeat(auto-fill, minmax(${effectiveMinWidth}px, 1fr))`,
        gap: effectiveGap,
        width: '100%',
        alignItems: 'start',
      }}
    >
      {children}
    </div>
  );
}
