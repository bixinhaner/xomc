import type { ReactNode } from 'react';
import { useThemeToken } from '@/hooks/useThemeToken';

interface Props {
  /** Left column content — typically device tree or command list (25%) */
  left: ReactNode;
  /** Center column content — command input / script editor (25%) */
  center: ReactNode;
  /** Right column content — output / results display (50%) */
  right: ReactNode;
  /** Gap between columns. Default: 12px */
  gap?: number;
}

/**
 * CommandConsoleLayout — three-column layout for MML console and similar tools.
 *
 * Column widths: 25% + 25% + 50%.
 *
 * All columns have independent scroll containers and occupy the full height.
 */
export default function CommandConsoleLayout({
  left,
  center,
  right,
  gap = 12,
}: Props) {
  const token = useThemeToken();
  const COLUMN_STYLE: React.CSSProperties = {
    display: 'flex',
    flexDirection: 'column',
    overflow: 'hidden',
    background: token.colorBgContainer,
    borderRadius: 6,
    border: `1px solid ${token.colorBorderSecondary}`,
    minHeight: 0,
  };

  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: '1fr 1fr 2fr',
        gap,
        width: '100%',
        height: '100%',
        overflow: 'hidden',
      }}
    >
      {/* Left column — device tree / command catalog */}
      <div style={{ ...COLUMN_STYLE, overflow: 'auto' }}>
        {left}
      </div>

      {/* Center column — command input / script editor */}
      <div style={{ ...COLUMN_STYLE, overflow: 'auto' }}>
        {center}
      </div>

      {/* Right column — output / results */}
      <div style={{ ...COLUMN_STYLE, overflow: 'auto' }}>
        {right}
      </div>
    </div>
  );
}
