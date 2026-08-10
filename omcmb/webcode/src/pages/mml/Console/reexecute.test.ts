import { describe, expect, it } from 'vitest';
import { buildHistoricalReexecuteRequest } from './reexecute';

describe('buildHistoricalReexecuteRequest', () => {
  it('rebuilds a selected LST history record even when the top bar has another command', () => {
    expect(buildHistoricalReexecuteRequest({
      operationType: 'LST',
      read: true,
      paths: ['Device.X.1.Name'],
    })).toEqual({
      request: {
        mode: 'raw',
        operationType: 'LST',
        rows: [{ id: 0, path: 'Device.X.1.Name', value: '' }],
        execMode: 'whole',
      },
    });
  });

  it.each(['ADD', 'RMV'] as const)('blocks unsafe %s history replay without a snapshot', (operationType) => {
    expect(buildHistoricalReexecuteRequest({
      operationType,
      read: false,
      paths: ['Device.X.1.'],
    })).toEqual({ reason: 'write-reconfigure' });
  });
});
