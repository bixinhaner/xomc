import { describe, expect, it } from 'vitest';
import { normalizeContactNumber, isValidContactNumber } from './validation';

describe('contact number validation', () => {
  it.each([
    '13800138000',
    '010-12345678',
    '+86 10 1234 5678',
    '+1 (212) 555-0100 x. 12',
  ])('accepts a valid contact number: %s', (value) => {
    expect(isValidContactNumber(value)).toBe(true);
  });

  it.each([
    '',
    '++--',
    '12',
    '123 abc',
    '123456789012345678901234567890123',
  ])('rejects an invalid contact number: %s', (value) => {
    expect(isValidContactNumber(value)).toBe(false);
  });

  it('trims leading and trailing whitespace before sending', () => {
    expect(normalizeContactNumber('  +86 10 1234 5678  ')).toBe('+86 10 1234 5678');
  });
});
