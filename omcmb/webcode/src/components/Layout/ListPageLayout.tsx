import type { ReactNode } from 'react';
import { Typography } from 'antd';

interface Props {
  /** Optional page title shown at the top of the content area */
  title?: string;
  /** Page subtitle or description */
  subtitle?: string;
  children: ReactNode;
  /** Extra content placed in the top-right of the title bar (e.g. action buttons) */
  extra?: ReactNode;
}

/**
 * ListPageLayout — vertical stack used by list/table pages.
 *
 * Structure:
 *   [optional title bar]
 *   [children — typically a toolbar + table]
 */
export default function ListPageLayout({ title, subtitle, children, extra }: Props) {
  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        gap: 16,
        flex: 1,
        minHeight: 0,
      }}
    >
      {(title ?? extra) && (
        <div
          style={{
            display: 'flex',
            alignItems: 'flex-start',
            justifyContent: 'space-between',
            gap: 16,
            flexShrink: 0,
          }}
        >
          {title && (
            <div>
              <Typography.Title level={4} style={{ margin: 0, lineHeight: '32px' }}>
                {title}
              </Typography.Title>
              {subtitle && (
                <Typography.Text type="secondary" style={{ fontSize: 13 }}>
                  {subtitle}
                </Typography.Text>
              )}
            </div>
          )}
          {extra && <div style={{ flexShrink: 0 }}>{extra}</div>}
        </div>
      )}
      <div style={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>{children}</div>
    </div>
  );
}
