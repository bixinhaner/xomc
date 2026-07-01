import { describe, expect, it } from 'vitest';

import { buildDeviceGroupSubtreeCountMap } from '../deviceGroupCounts';

describe('buildDeviceGroupSubtreeCountMap', () => {
  it('aggregates descendant counts for parent groups', () => {
    const counts = buildDeviceGroupSubtreeCountMap([
      { id: 'root', parentId: null, deviceCount: 0 },
      { id: 'child-a', parentId: 'root', deviceCount: 2 },
      { id: 'child-b', parentId: 'root', deviceCount: 3 },
    ]);

    expect(counts.get('root')).toBe(5);
    expect(counts.get('child-a')).toBe(2);
    expect(counts.get('child-b')).toBe(3);
  });

  it('uses the filtered subtree when ancestors are kept only for search context', () => {
    const counts = buildDeviceGroupSubtreeCountMap([
      { id: 'root', parentId: null, deviceCount: 0 },
      { id: 'matched-child', parentId: 'root', deviceCount: 5 },
    ]);

    expect(counts.get('root')).toBe(5);
  });
});