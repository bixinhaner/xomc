import { describe, expect, it } from 'vitest';
import { getTemplateFieldCoverage } from './paramConfigGroupedFields';

describe('grouped parameter configuration fields', () => {
  it.each(['eNB', 'gNB', 'GSM'] as const)('covers every %s template field once', (deviceType) => {
    const { expected, covered } = getTemplateFieldCoverage(deviceType);
    expect(new Set(covered).size).toBe(covered.length);
    expect([...covered].sort()).toEqual([...expected].sort());
  });
});
