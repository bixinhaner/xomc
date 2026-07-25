import type { TranslateFn } from '@/hooks/useT';

type ApiErrorBody = { msg?: string; message?: string };

function rawErrorMessage(error: unknown): string {
  const candidate = error as {
    response?: { data?: ApiErrorBody };
    message?: string;
  };
  return (
    candidate.response?.data?.msg ??
    candidate.response?.data?.message ??
    (error instanceof Error ? error.message : String(error))
  );
}

export function enabledIndicatorErrorMessage(error: unknown, t: TranslateFn): string {
  const raw = rawErrorMessage(error);
  if (/indicator is used by dashboard KPI layout/i.test(raw)) {
    const ids = raw.match(/\[([^\]]+)\]/)?.[1] ?? '';
    return t('product.kpi.enabled.dashboardRefError', { ids });
  }
  return t('common.operationFailed');
}
