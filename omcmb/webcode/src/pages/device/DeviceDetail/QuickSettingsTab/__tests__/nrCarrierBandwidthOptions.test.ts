import { describe, expect, it } from 'vitest';
import { getNrCarrierBandwidthOptions } from '../CellParameterForm';

describe('NR quick-settings carrier bandwidth options', () => {
  it('does not expose unsupported 5MHz, 15MHz or 25MHz options for 30kHz SCS', () => {
    const labels = getNrCarrierBandwidthOptions('DLCarrierBandWidth', '1', '1')
      .map((option) => option.label);

    expect(labels).not.toContain('5MHz(11RB)');
    expect(labels).not.toContain('15MHz(38RB)');
    expect(labels).not.toContain('25MHz(65RB)');
  });
});
