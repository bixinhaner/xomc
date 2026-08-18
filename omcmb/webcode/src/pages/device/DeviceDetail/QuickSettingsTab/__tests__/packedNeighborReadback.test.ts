import { describe, expect, it } from 'vitest';
import { packedNeighborChangeObserved } from '../MultiInstanceTable';

describe('packed neighbor readback', () => {
  const first = ['460', '00', '1', '10', '100', '1'];
  const second = ['460', '00', '1', '11', '101', '2'];

  it('does not accept the pre-save list for an add or delete', () => {
    expect(packedNeighborChangeObserved([first], [first], second, 'add')).toBe(false);
    expect(packedNeighborChangeObserved([first, second], [first, second], second, 'del')).toBe(false);
  });

  it('accepts only the list whose target multiplicity changed', () => {
    expect(packedNeighborChangeObserved([first], [first, second], second, 'add')).toBe(true);
    expect(packedNeighborChangeObserved([first, second], [first], second, 'del')).toBe(true);
  });
});
