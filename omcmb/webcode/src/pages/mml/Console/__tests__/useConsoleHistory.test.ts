import { describe, expect, it } from 'vitest';
import { isMissingTaskError } from '../useConsoleHistory';

describe('useConsoleHistory', () => {
  it('only treats 404/410 as removable missing history tasks', () => {
    expect(isMissingTaskError({ response: { status: 404 } })).toBe(true);
    expect(isMissingTaskError({ response: { status: 410 } })).toBe(true);
    expect(isMissingTaskError({ response: { status: 401 } })).toBe(false);
    expect(isMissingTaskError({ response: { status: 500 } })).toBe(false);
    expect(isMissingTaskError(new Error('network'))).toBe(false);
  });
});
