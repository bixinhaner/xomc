import { describe, expect, it } from 'vitest';

import { buildMetricBatchSelection, parseMetricBatchIds } from './metricBatchSelection';

const indicators = [
  { id: 'K0001' },
  { id: 'K0002' },
  { id: 'C0001' },
  { id: 'C0002' },
];

describe('metricBatchSelection', () => {
  it('parses metric IDs separated by commas, Chinese commas, semicolons, spaces, and lines', () => {
    expect(parseMetricBatchIds('K0001, K0002\nC0001；C0002， K0003')).toEqual([
      'K0001',
      'K0002',
      'C0001',
      'C0002',
      'K0003',
    ]);
  });

  it('selects valid IDs, reports missing IDs, and deduplicates repeated IDs', () => {
    const result = buildMetricBatchSelection({
      text: 'K0001, K0002, K0001, C9999',
      indicators,
      selected: ['C0001'],
    });

    expect(result.nextSelected).toEqual(['C0001', 'K0001', 'K0002']);
    expect(result.validIds).toEqual(['K0001', 'K0002']);
    expect(result.addedIds).toEqual(['K0001', 'K0002']);
    expect(result.invalidIds).toEqual(['C9999']);
  });

  it('keeps existing selection unchanged when all IDs are invalid', () => {
    const result = buildMetricBatchSelection({
      text: 'K404 C404',
      indicators,
      selected: ['C0001'],
    });

    expect(result.nextSelected).toEqual(['C0001']);
    expect(result.validIds).toEqual([]);
    expect(result.invalidIds).toEqual(['K404', 'C404']);
  });

  it('respects max selected count without clearing current selection', () => {
    const result = buildMetricBatchSelection({
      text: 'K0001 K0002 C0002',
      indicators,
      selected: ['C0001'],
      maxSelected: 3,
    });

    expect(result.nextSelected).toEqual(['C0001', 'K0001', 'K0002']);
    expect(result.omittedValidIds).toEqual(['C0002']);
    expect(result.limitExceeded).toBe(true);
  });
});
