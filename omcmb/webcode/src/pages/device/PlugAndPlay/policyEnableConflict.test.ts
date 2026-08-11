import { describe, expect, it } from 'vitest';
import { findEnabledPolicyProductConflict } from './policyEnableConflict';

describe('findEnabledPolicyProductConflict', () => {
  const enabledBNQ = { policyId: 'enabled', productNames: ['BNQ'], enabled: true };

  it('finds another enabled policy sharing any product name', () => {
    expect(findEnabledPolicyProductConflict(
      { policyId: 'candidate', productNames: ['MLN', 'bnq'], enabled: false },
      [enabledBNQ],
    )).toEqual(enabledBNQ);
  });

  it('ignores disabled policies, other products and the candidate itself', () => {
    expect(findEnabledPolicyProductConflict(
      { policyId: 'candidate', productNames: ['BNQ'], enabled: false },
      [
        { policyId: 'candidate', productNames: ['BNQ'], enabled: true },
        { policyId: 'disabled', productNames: ['BNQ'], enabled: false },
        { policyId: 'other', productNames: ['MLN'], enabled: true },
      ],
    )).toBeUndefined();
  });
});
