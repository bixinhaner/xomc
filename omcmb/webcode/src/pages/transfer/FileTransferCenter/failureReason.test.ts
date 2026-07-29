import { describe, expect, it } from 'vitest';
import { normalizeFailureReasonCode } from './failureReason';

describe('normalizeFailureReasonCode', () => {
  it('maps terminated task fallbacks to the i18n failure code', () => {
    expect(normalizeFailureReasonCode('task terminated by operator')).toBe('OPERATOR_TERMINATED');
    expect(normalizeFailureReasonCode('终止')).toBe('OPERATOR_TERMINATED');
  });

  it('keeps existing failure codes and raw vendor messages unchanged', () => {
    expect(normalizeFailureReasonCode('DOWNLOAD_FAULT')).toBe('DOWNLOAD_FAULT');
    expect(normalizeFailureReasonCode('FaultCode: 0')).toBe('FaultCode: 0');
  });
});
