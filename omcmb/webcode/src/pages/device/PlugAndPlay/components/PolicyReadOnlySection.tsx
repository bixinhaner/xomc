import type { PropsWithChildren } from 'react';

interface PolicyReadOnlySectionProps extends PropsWithChildren {
  readOnly: boolean;
}

/**
 * Prevents editing in policy detail mode without applying Ant Design's
 * disabled appearance. Keep navigation controls outside this boundary.
 */
export default function PolicyReadOnlySection({
  children,
  readOnly,
}: PolicyReadOnlySectionProps) {
  return (
    <div
      aria-readonly={readOnly || undefined}
      inert={readOnly || undefined}
    >
      {children}
    </div>
  );
}
