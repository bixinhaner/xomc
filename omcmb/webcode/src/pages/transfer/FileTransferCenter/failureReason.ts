const TERMINATED_FAILURE_VALUES = new Set(['task terminated by operator', '终止']);

export function normalizeFailureReasonCode(value: string): string {
  const trimmed = value.trim();
  if (TERMINATED_FAILURE_VALUES.has(trimmed)) {
    return 'OPERATOR_TERMINATED';
  }
  return trimmed;
}
