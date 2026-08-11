import type { MMLOperationType } from '@core/types/mml';
import type { ExecRequest } from './types';

interface HistoricalReexecuteInput {
  operationType?: MMLOperationType;
  read: boolean;
  paths: string[];
  values?: Record<string, string>;
}

export type HistoricalReexecuteDecision =
  | { request: ExecRequest; reason?: never }
  | { request?: never; reason: 'missing' | 'write-reconfigure' };

/**
 * Rebuild exclusively from the selected history record. Never consult the
 * command currently selected in the top bar: it may be a different command.
 */
export function buildHistoricalReexecuteRequest(
  input: HistoricalReexecuteInput,
): HistoricalReexecuteDecision {
  const paths = input.paths.filter(Boolean);
  if (!input.operationType || paths.length === 0) return { reason: 'missing' };

  if (input.operationType === 'MOD' && input.values) {
    return {
      request: {
        mode: 'raw',
        operationType: 'MOD',
        rows: paths.map((path, index) => ({ id: index, path, value: input.values?.[path] ?? '' })),
        execMode: 'whole',
      },
    };
  }

  if (!input.read) return { reason: 'write-reconfigure' };
  return {
    request: {
      mode: 'raw',
      operationType: input.operationType,
      rows: paths.map((path, index) => ({ id: index, path, value: '' })),
      execMode: 'whole',
    },
  };
}
