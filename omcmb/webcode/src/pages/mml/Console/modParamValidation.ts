import { dataTypeRangeKind } from '@core/types/paramModel';
import type { CommandParamPath } from './types';

export type ModParamValidationCode =
  | 'required'
  | 'integer'
  | 'minValue'
  | 'maxValue'
  | 'minLength'
  | 'maxLength';

export interface ModParamValidationError {
  code: ModParamValidationCode;
  bound?: number;
}

export function validateModParamValue(
  path: CommandParamPath,
  value: string,
): ModParamValidationError | null {
  if (value.trim() === '') return { code: 'required' };
  if (path.minValue == null && path.maxValue == null) return null;

  const rangeKind = dataTypeRangeKind(path.valueType);
  if (rangeKind === 'none') return null;
  if (rangeKind === 'length') {
    const length = Array.from(value).length;
    if (path.minValue != null && length < path.minValue) {
      return { code: 'minLength', bound: path.minValue };
    }
    if (path.maxValue != null && length > path.maxValue) {
      return { code: 'maxLength', bound: path.maxValue };
    }
    return null;
  }

  const trimmed = value.trim();
  if (!/^[+-]?\d+$/.test(trimmed)) return { code: 'integer' };
  const numericValue = Number(trimmed);
  if (!Number.isSafeInteger(numericValue)) return { code: 'integer' };
  if (path.minValue != null && numericValue < path.minValue) {
    return { code: 'minValue', bound: path.minValue };
  }
  if (path.maxValue != null && numericValue > path.maxValue) {
    return { code: 'maxValue', bound: path.maxValue };
  }
  return null;
}

export function getModParamValidationErrors(
  paths: CommandParamPath[],
  values: Record<string, string>,
): Record<string, ModParamValidationError> {
  const errors: Record<string, ModParamValidationError> = {};

  for (const path of paths) {
    const error = validateModParamValue(path, values[path.path] ?? '');
    if (error) errors[path.path] = error;
  }

  return errors;
}
