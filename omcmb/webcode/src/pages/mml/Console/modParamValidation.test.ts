import { describe, expect, it } from 'vitest';
import type { CommandParamPath } from './types';
import {
  getModParamValidationErrors,
  validateModParamValue,
} from './modParamValidation';

function param(overrides: Partial<CommandParamPath> = {}): CommandParamPath {
  return {
    path: 'Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity',
    label: 'CellIdentity',
    writable: true,
    isObject: false,
    ...overrides,
  };
}

describe('validateModParamValue', () => {
  it('returns required before considering range configuration', () => {
    expect(validateModParamValue(param(), '   ')).toEqual({ code: 'required' });
    expect(
      validateModParamValue(
        param({ valueType: 'unsignedInt', minValue: 1, maxValue: 13 }),
        '',
      ),
    ).toEqual({ code: 'required' });
  });

  it('skips type and range checks when both bounds are missing', () => {
    expect(validateModParamValue(param({ valueType: 'unsignedInt' }), 'not-a-number')).toBeNull();
  });

  it.each([
    ['a', 'minLength', 2, 4],
    ['abcde', 'maxLength', 2, 4],
  ])('validates string Unicode length for %s', (value, code, minValue, maxValue) => {
    expect(
      validateModParamValue(
        param({ valueType: 'string', minValue, maxValue }),
        value,
      ),
    ).toEqual(expect.objectContaining({ code }));
  });

  it('counts an emoji as one Unicode character', () => {
    expect(
      validateModParamValue(
        param({ valueType: 'string', minValue: 1, maxValue: 1 }),
        '😀',
      ),
    ).toBeNull();
  });

  it('counts original string whitespace toward length', () => {
    expect(
      validateModParamValue(
        param({ valueType: 'string', minValue: 3, maxValue: 3 }),
        ' a ',
      ),
    ).toBeNull();
  });

  it('accepts inclusive integer boundaries', () => {
    const path = param({ valueType: 'unsignedInt', minValue: 1, maxValue: 13 });
    expect(validateModParamValue(path, '1')).toBeNull();
    expect(validateModParamValue(path, '13')).toBeNull();
  });

  it.each([
    ['x', 'integer'],
    ['1.2', 'integer'],
    ['1e3', 'integer'],
    ['0', 'minValue'],
    ['14', 'maxValue'],
    ['9007199254740992', 'integer'],
  ])('rejects invalid ranged integer %s', (value, code) => {
    expect(
      validateModParamValue(
        param({ valueType: 'unsignedInt', minValue: 1, maxValue: 13 }),
        value,
      ),
    ).toEqual(expect.objectContaining({ code }));
  });

  it('checks only the configured lower bound', () => {
    const path = param({ valueType: 'int', minValue: -3 });
    expect(validateModParamValue(path, '-3')).toBeNull();
    expect(validateModParamValue(path, '-4')).toEqual({ code: 'minValue', bound: -3 });
  });

  it('checks only the configured upper bound', () => {
    const path = param({ valueType: 'int', maxValue: 5 });
    expect(validateModParamValue(path, '5')).toBeNull();
    expect(validateModParamValue(path, '6')).toEqual({ code: 'maxValue', bound: 5 });
  });

  it.each(['boolean', 'dateTime'])('does not range-check %s', (valueType) => {
    expect(
      validateModParamValue(
        param({ valueType, minValue: 1, maxValue: 1 }),
        'anything',
      ),
    ).toBeNull();
  });
});

describe('getModParamValidationErrors', () => {
  it('returns only errors keyed by Path', () => {
    const invalidPath = param({
      path: 'Device.Cell.Invalid',
      valueType: 'unsignedInt',
      minValue: 1,
    });
    const validPath = param({
      path: 'Device.Cell.Valid',
      valueType: 'string',
      maxValue: 3,
    });

    expect(
      getModParamValidationErrors([invalidPath, validPath], {
        'Device.Cell.Invalid': '0',
        'Device.Cell.Valid': 'ok',
        'Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.Unselected': 'not-a-number',
      }),
    ).toEqual({
      'Device.Cell.Invalid': { code: 'minValue', bound: 1 },
    });
  });
});
