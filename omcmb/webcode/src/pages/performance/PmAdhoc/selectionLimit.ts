import type { Key } from 'react';

export type TransferKey = Key;

export interface LimitedTransferSelectionResult {
  next: string[];
  exceeded: boolean;
  count: number;
}

export function resolveLimitedTransferSelection(
  current: readonly string[],
  nextKeys: readonly TransferKey[],
  max: number,
): LimitedTransferSelectionResult {
  const next = nextKeys.map(String);
  if (next.length <= max) {
    return { next, exceeded: false, count: next.length };
  }
  return { next: [...current], exceeded: true, count: next.length };
}
