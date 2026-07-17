import { describe, expect, it } from 'vitest';
import { shouldShowLocationSyncIndicator } from './deviceGpsSyncIndicator';

describe('shouldShowLocationSyncIndicator', () => {
  it('shows an indicator for every concrete coordinate cell', () => {
    expect(shouldShowLocationSyncIndicator(115)).toBe(true);
    expect(shouldShowLocationSyncIndicator(25)).toBe(true);
    expect(shouldShowLocationSyncIndicator(0)).toBe(true);
  });

  it('does not show an indicator for an empty coordinate cell', () => {
    expect(shouldShowLocationSyncIndicator(null)).toBe(false);
    expect(shouldShowLocationSyncIndicator(undefined)).toBe(false);
  });

  it('shows an indicator even when there is no pending report', () => {
    expect(shouldShowLocationSyncIndicator(115)).toBe(true);
    expect(shouldShowLocationSyncIndicator(25)).toBe(true);
  });
});
