import type { MMLOperationType } from '@core/types/mml';
import { isReadOp } from './constants';
import type { CommandParamPath } from './types';

export function commandUsesPathSelection(op: MMLOperationType | string | undefined): boolean {
  return isReadOp(op) || op === 'MOD';
}

export function getSelectableCommandPaths(
  op: MMLOperationType | string | undefined,
  paths: CommandParamPath[],
): CommandParamPath[] {
  if (isReadOp(op)) return paths;
  if (op === 'MOD') return paths.filter((path) => path.writable);
  return [];
}

export function getOrderedSelectedCommandPaths(
  paths: CommandParamPath[],
  selectedPathKeys: string[],
): CommandParamPath[] {
  const selected = new Set(selectedPathKeys);
  return paths.filter((path) => selected.has(path.path));
}

export function getOrderedSelectedPathKeys(
  paths: CommandParamPath[],
  selectedPathKeys: string[],
): string[] {
  return getOrderedSelectedCommandPaths(paths, selectedPathKeys).map((path) => path.path);
}

export function areSelectedPathValuesComplete(
  paths: CommandParamPath[],
  values: Record<string, string>,
): boolean {
  return paths.length > 0 && paths.every((path) => (values[path.path] ?? '').trim() !== '');
}
