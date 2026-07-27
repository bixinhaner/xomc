import { describe, expect, it } from 'vitest';
import { sysConfigApplyRefetchInterval } from './useSystem';

describe('sysConfigApplyRefetchInterval', () => {
  it('continues observing failed batches because the backend retries them', () => {
    expect(sysConfigApplyRefetchInterval('failed')).toBe(15_000);
  });

  it('stops only after a successful terminal state', () => {
    expect(sysConfigApplyRefetchInterval('applied')).toBe(false);
    expect(sysConfigApplyRefetchInterval('pending')).toBe(2_000);
    expect(sysConfigApplyRefetchInterval('applying')).toBe(2_000);
  });
});
