import { describe, expect, it } from 'vitest';
import { formatFailureReasonDisplay, normalizeFailureReasonCode } from './failureReason';

describe('normalizeFailureReasonCode', () => {
  it('maps terminated task fallbacks to the i18n failure code', () => {
    expect(normalizeFailureReasonCode('task terminated by operator')).toBe('OPERATOR_TERMINATED');
    expect(normalizeFailureReasonCode('终止')).toBe('OPERATOR_TERMINATED');
  });

  it('keeps existing failure codes and raw vendor messages unchanged', () => {
    expect(normalizeFailureReasonCode('DOWNLOAD_FAULT')).toBe('DOWNLOAD_FAULT');
    expect(normalizeFailureReasonCode('FaultCode: 0')).toBe('FaultCode: 0');
  });

  it('formats failure codes through i18n messages and keeps raw details unchanged', () => {
    const messages: Record<string, string> = {
      'software.failureCode.DOWNLOAD_TIMEOUT': '下载响应超时，未收到设备 DownloadResponse',
    };
    const t = (id: string) => messages[id] ?? id;

    expect(formatFailureReasonDisplay('DOWNLOAD_TIMEOUT', t)).toEqual({
      codeOrRaw: 'DOWNLOAD_TIMEOUT',
      display: '下载响应超时，未收到设备 DownloadResponse',
    });
    expect(formatFailureReasonDisplay('FaultCode: 0', t)).toEqual({
      codeOrRaw: 'FaultCode: 0',
      display: 'FaultCode: 0',
    });
  });
});
