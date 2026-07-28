import { describe, expect, it } from 'vitest';
import { scopeSelectedDeviceKeys } from './deviceSelection';

describe('scopeSelectedDeviceKeys', () => {
  it('keeps only selected SNs that belong to the current device selection scope', () => {
    expect(
      scopeSelectedDeviceKeys(
        ['BU1810-SN', 'BSC7041-SN', 'ANOTHER-BU1810-SN'],
        ['BU1810-SN', 'ANOTHER-BU1810-SN'],
      ),
    ).toEqual(['BU1810-SN', 'ANOTHER-BU1810-SN']);
  });

  it('preserves selected order after filtering stale SNs', () => {
    expect(scopeSelectedDeviceKeys(['sn-3', 'sn-1', 'sn-2'], ['sn-1', 'sn-2', 'sn-3'])).toEqual([
      'sn-3',
      'sn-1',
      'sn-2',
    ]);
  });
});
