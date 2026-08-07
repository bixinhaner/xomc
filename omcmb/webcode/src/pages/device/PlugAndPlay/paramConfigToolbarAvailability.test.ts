import { describe, expect, it } from 'vitest';
import { isParamConfigToolbarEnabled } from './paramConfigToolbarAvailability';

describe('plug-and-play parameter configuration toolbar availability', () => {
  it('enables toolbar actions when parameter configuration is enabled', () => {
    expect(isParamConfigToolbarEnabled(true)).toBe(true);
  });

  it('disables toolbar actions when parameter configuration is disabled', () => {
    expect(isParamConfigToolbarEnabled(false)).toBe(false);
  });
});
