import { dataTypeRangeKind } from '@core/types/paramModel';
import type { CommandParamPath } from './types';

export type ModParamValidationCode =
  | 'required'
  | 'integer'
  | 'minValue'
  | 'maxValue'
  | 'minLength'
  | 'maxLength'
  | 'enumValue'
  | 'pattern';

export interface ModParamValidationError {
  code: ModParamValidationCode;
  bound?: number;
}

const NAMED_VALIDATION_PATTERNS: Record<string, string> = {
  // omc ship's legacy front-end rule name: reject CJK characters.
  no_zh: '^(?:(?![\\u4E00-\\u9FA5]|[\\uFE30-\\uFFA0]).)+$',
};

function isBooleanValueType(valueType?: string): boolean {
  const normalized = valueType?.trim().toLowerCase();
  return normalized === 'boolean' || normalized === 'bool';
}

function normalizeBooleanEquivalent(value: unknown): 'true' | 'false' | undefined {
  const normalized = String(value ?? '').trim().toLowerCase();
  if (normalized === 'true' || normalized === '1') return 'true';
  if (normalized === 'false' || normalized === '0') return 'false';
  return undefined;
}

function enumContainsValue(path: CommandParamPath, value: string): boolean {
  const options = path.enumOptions ?? [];
  if (options.some((option) => option.value === value)) return true;
  if (!isBooleanValueType(path.valueType)) return false;

  const normalizedValue = normalizeBooleanEquivalent(value);
  return normalizedValue != null
    && options.some((option) => normalizeBooleanEquivalent(option.value) === normalizedValue);
}

export function validateModParamValue(
  path: CommandParamPath,
  value: string,
): ModParamValidationError | null {
  if (value.trim() === '') {
    // Explicitly optional catalog fields may be omitted from MOD/ADD. Keep
    // undefined as required for compatibility with older command payloads.
    return path.isRequired === false ? null : { code: 'required' };
  }

  if (path.enumOptions?.length && !enumContainsValue(path, value)) {
    return { code: 'enumValue' };
  }

  const rangeKind = dataTypeRangeKind(path.valueType);
  if (rangeKind === 'length') {
    const length = Array.from(value).length;
    if (path.minValue != null && length < path.minValue) {
      return { code: 'minLength', bound: path.minValue };
    }
    if (path.maxValue != null && length > path.maxValue) {
      return { code: 'maxLength', bound: path.maxValue };
    }
  } else if (rangeKind !== 'none' && (path.minValue != null || path.maxValue != null)) {
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
  }

  if (path.validationPattern && !matchesValidationPattern(path.validationPattern, value)) {
    return { code: 'pattern' };
  }

  return null;
}

function matchesValidationPattern(pattern: string, value: string): boolean {
  const rawPattern = pattern.trim();
  const source = NAMED_VALIDATION_PATTERNS[rawPattern] ?? rawPattern;
  try {
    if (source.startsWith('/') && source.lastIndexOf('/') > 0) {
      const end = source.lastIndexOf('/');
      return new RegExp(source.slice(1, end), source.slice(end + 1)).test(value);
    }
    return new RegExp(source).test(value);
  } catch {
    return false;
  }
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
