import { describe, expect, it } from 'vitest';

import { resolveLimitedTransferSelection } from './selectionLimit';

describe('resolveLimitedTransferSelection', () => {
  it('accepts transfer keys when the next selection is within the limit', () => {
    expect(resolveLimitedTransferSelection(['K0001'], ['K0001', 'K0002'], 2)).toEqual({
      next: ['K0001', 'K0002'],
      exceeded: false,
      count: 2,
    });
  });

  it('keeps the current selection when a transfer would exceed the limit', () => {
    expect(resolveLimitedTransferSelection(['K0001', 'K0002'], ['K0001', 'K0002', 'K0003'], 2)).toEqual({
      next: ['K0001', 'K0002'],
      exceeded: true,
      count: 3,
    });
  });

  it('normalizes numeric transfer keys to strings', () => {
    expect(resolveLimitedTransferSelection(['1'], [1, 2], 2)).toEqual({
      next: ['1', '2'],
      exceeded: false,
      count: 2,
    });
  });
});
