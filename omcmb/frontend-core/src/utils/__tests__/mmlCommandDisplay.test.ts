import { describe, expect, it } from 'vitest';

import { parseMmlCommandDisplay } from '../mmlCommandDisplay';

describe('parseMmlCommandDisplay', () => {
  it('splits a long MOD command into operation target and parameters', () => {
    const parsed = parseMmlCommandDisplay(
      'MOD MANAGEMENT_SERVER:URL=http://localhost:8080/smallcell/AcsService,USERNAME=acs,PASSWORD=acs'
    );

    expect(parsed.operation).toBe('MOD');
    expect(parsed.target).toBe('MANAGEMENT_SERVER');
    expect(parsed.params).toEqual([
      { key: 'URL', value: 'http://localhost:8080/smallcell/AcsService' },
      { key: 'USERNAME', value: 'acs' },
      { key: 'PASSWORD', value: 'acs' },
    ]);
    expect(parsed.parameterCount).toBe(3);
  });

  it('keeps non-parameter commands readable', () => {
    const parsed = parseMmlCommandDisplay('LST DEVICE_INFO');

    expect(parsed.operation).toBe('LST');
    expect(parsed.target).toBe('DEVICE_INFO');
    expect(parsed.params).toEqual([]);
    expect(parsed.parameterCount).toBe(0);
  });
});
