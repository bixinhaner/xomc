import type { MMLCommand, MMLOperationType } from '@core/types/mml';

/**
 * Resolve the operation type from a command.
 * Uses explicit operationType if present, otherwise falls back to the
 * first token of commandCode (e.g. "LST" from "LST CELL").
 */
export function resolveOperationType(command: MMLCommand | null): MMLOperationType {
  const operationType = command?.operationType?.trim().toUpperCase();
  if (operationType) {
    return operationType as MMLOperationType;
  }

  const prefix = command?.commandCode?.trim().split(/\s+/)[0]?.toUpperCase();
  return (prefix || 'LST') as MMLOperationType;
}
