import type { ReactNode } from 'react';
import { Alert, Button, Tag, Tooltip, type ButtonProps } from 'antd';
import type { AccessState, ActionStatus } from '@core/services/api/deviceAccessApi';

const stateColors: Record<AccessState, string> = {
  review_required: 'orange',
  collecting: 'processing',
  accepted: 'green',
  rejected: 'red',
  revalidating: 'blue',
  revoked: 'volcano',
};

const actionColors: Record<ActionStatus, string> = {
  pending_dispatch: 'orange',
  dispatching: 'processing',
  succeeded: 'green',
  failed: 'red',
  cancelled: 'default',
};

export function StateTag({ value, t }: { value: AccessState; t: (key: string) => string }) {
  return <Tag color={stateColors[value]}>{t(`deviceAccess.state.${value}`)}</Tag>;
}

export function ActionStatusTag({ value, t }: { value: ActionStatus; t: (key: string) => string }) {
  return <Tag color={actionColors[value]}>{t(`deviceAccess.actionStatus.${value}`)}</Tag>;
}

export function formatProductName(value: string | null | undefined, t: (key: string) => string): string {
  return value?.trim() || t('deviceAccess.productNameUnknown');
}

export function normalTaskProtectionPresentation(frozen: boolean) {
  if (!frozen) return { color: 'green', messageKey: 'deviceAccess.allowed' } as const;
  return { color: 'red', messageKey: 'deviceAccess.frozen' } as const;
}

export function GuardedButton({ allowed, deniedText, children, ...props }: ButtonProps & {
  allowed: boolean;
  deniedText: string;
  children: ReactNode;
}) {
  return (
    <Tooltip title={allowed ? undefined : deniedText}>
      <span>
        <Button {...props} disabled={!allowed || props.disabled}>{children}</Button>
      </span>
    </Tooltip>
  );
}

export function QueryError({ error, t }: { error: unknown; t: (key: string) => string }) {
  if (!error) return null;
  return (
    <Alert
      showIcon
      type="error"
      title={t('empty.loadFailed')}
      description={error instanceof Error ? error.message : t('empty.loadFailedDesc')}
      style={{ marginBottom: 16 }}
    />
  );
}
