import type { ReactNode } from 'react';
import { CONTENT_PADDING_H, CONTENT_PADDING_V } from '@/theme/tokens';

interface Props {
  children: ReactNode;
  /** Override default padding. Useful for full-bleed pages like maps. */
  padding?: string | number;
  className?: string;
  style?: React.CSSProperties;
}

/**
 * ContentArea — scrollable wrapper for page content.
 *
 * In the AppShell, the <main> element already provides scrolling and default
 * padding. This component exists for cases where a sub-section of a page
 * needs its own independent scroll container with token-based padding.
 */
export default function ContentArea({ children, padding, className, style }: Props) {
  const defaultPadding = `${CONTENT_PADDING_V}px ${CONTENT_PADDING_H}px`;

  return (
    <div
      className={className}
      style={{
        width: '100%',
        height: '100%',
        overflowY: 'auto',
        overflowX: 'hidden',
        padding: padding !== undefined ? padding : defaultPadding,
        boxSizing: 'border-box',
        ...style,
      }}
    >
      {children}
    </div>
  );
}
