import { describe, it, expect } from 'vitest';
import type { StandardParamView } from '@core/types/mmlAdmin';
import {
  deriveMmlCode,
  buildRow,
  appendRows,
  removeRow,
  computeExcludeIds,
} from '../subFieldRows';

function sp(id: string, standardPath: string, description = ''): StandardParamView {
  return {
    id,
    standardPath,
    entryType: 'parameter',
    access: 'READ_WRITE',
    dataType: 'string',
    changeApplies: '',
    description,
  };
}

describe('subFieldRows.deriveMmlCode', () => {
  it('handles CamelCase / underscore / digits', () => {
    expect(deriveMmlCode('Device.DeviceInfo.UserLabel')).toBe('USER_LABEL');
    expect(deriveMmlCode('Device.X.AntennaAzimuth')).toBe('ANTENNA_AZIMUTH');
    expect(deriveMmlCode('Device.X.1588_Status')).toBe('1588_STATUS');
    expect(deriveMmlCode('foo.bar.simple_path')).toBe('SIMPLE_PATH');
    expect(deriveMmlCode('NoDot')).toBe('NO_DOT');
    expect(deriveMmlCode('')).toBe('');
  });
});

describe('subFieldRows.buildRow', () => {
  it('derives mmlCode and prefers description for label', () => {
    const row = buildRow(sp('p1', 'Device.X.UserLabel', '用户标签'));
    expect(row.paramId).toBe('p1');
    expect(row.mmlCode).toBe('USER_LABEL');
    expect(row.label).toBe('用户标签');
  });
  it('falls back to humanized leaf when no description', () => {
    const row = buildRow(sp('p2', 'Device.X.AntennaAzimuth'));
    expect(row.label).toBe('Antenna Azimuth');
  });
});

describe('subFieldRows.appendRows (规则2：连续添加)', () => {
  it('appends newly picked records preserving order', () => {
    const r0 = appendRows([], [sp('a', 'Device.A'), sp('b', 'Device.B')]);
    expect(r0.map((r) => r.paramId)).toEqual(['a', 'b']);
    const r1 = appendRows(r0, [sp('c', 'Device.C')]);
    expect(r1.map((r) => r.paramId)).toEqual(['a', 'b', 'c']);
  });
  it('dedups already-present paramId (防御性)', () => {
    const r0 = appendRows([], [sp('a', 'Device.A')]);
    const r1 = appendRows(r0, [sp('a', 'Device.A')]);
    expect(r1).toBe(r0); // 无新增时原样返回（引用相等，省 re-render）
    expect(r1.map((r) => r.paramId)).toEqual(['a']);
  });
  it('preserves existing rows (含用户 inline 编辑) when appending', () => {
    const r0 = appendRows([], [sp('a', 'Device.A')]);
    const edited = [{ ...r0[0], mmlCode: 'EDITED_CODE', label: '改过的' }];
    const r1 = appendRows(edited, [sp('b', 'Device.B')]);
    expect(r1[0].mmlCode).toBe('EDITED_CODE');
    expect(r1[0].label).toBe('改过的');
    expect(r1[1].paramId).toBe('b');
  });
});

describe('subFieldRows.removeRow (规则3：单独删除)', () => {
  it('removes the matching paramId only', () => {
    const r0 = appendRows([], [sp('a', 'Device.A'), sp('b', 'Device.B'), sp('c', 'Device.C')]);
    const r1 = removeRow(r0, 'b');
    expect(r1.map((r) => r.paramId)).toEqual(['a', 'c']);
  });
  it('is a no-op for unknown id', () => {
    const r0 = appendRows([], [sp('a', 'Device.A')]);
    expect(removeRow(r0, 'zzz').map((r) => r.paramId)).toEqual(['a']);
  });
});

describe('subFieldRows.computeExcludeIds (规则1：隐藏已选)', () => {
  it('unions already-saved and already-in-table ids', () => {
    const rows = appendRows([], [sp('x', 'Device.X'), sp('y', 'Device.Y')]);
    expect(computeExcludeIds(['saved1', 'saved2'], rows)).toEqual([
      'saved1',
      'saved2',
      'x',
      'y',
    ]);
  });
  it('handles empty inputs', () => {
    expect(computeExcludeIds([], [])).toEqual([]);
  });
});
