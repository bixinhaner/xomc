import { describe, expect, it } from 'vitest';

import { normalizeRecycleOfflineDays, resolveRecycleType } from './recycleBin';

describe('recycle bin display metadata', () => {
  it('uses the backend offline-day snapshot instead of deriving it from the current time', () => {
    expect(normalizeRecycleOfflineDays(6)).toBe(6);
    expect(normalizeRecycleOfflineDays(undefined)).toBe(0);
  });

  it('uses the explicit recycle type for new records', () => {
    expect(resolveRecycleType('auto', 'admin')).toBe('auto');
    expect(resolveRecycleType('manual', 'alice')).toBe('manual');
  });

  it('recognizes historical system executor values as automatic recycle', () => {
    expect(resolveRecycleType(undefined, 'system:auto_recycle')).toBe('auto');
    expect(resolveRecycleType(undefined, 'bob')).toBe('manual');
  });
});
