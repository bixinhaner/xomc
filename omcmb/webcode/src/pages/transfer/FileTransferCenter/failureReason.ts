const TERMINATED_FAILURE_VALUES = new Set(['task terminated by operator', '终止']);

export function normalizeFailureReasonCode(value: string): string {
  const trimmed = value.trim();
  if (TERMINATED_FAILURE_VALUES.has(trimmed)) {
    return 'OPERATOR_TERMINATED';
  }
  return trimmed;
}

export function formatFailureReasonDisplay(
  value: string,
  t: (id: string) => string,
): { codeOrRaw: string; display: string } {
  const codeOrRaw = normalizeFailureReasonCode(value);
  const key = `software.failureCode.${codeOrRaw}`;
  const i18nLabel = t(key);
  return {
    codeOrRaw,
    display: i18nLabel && i18nLabel !== key ? i18nLabel : value,
  };
}
