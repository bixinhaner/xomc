import { describe, expect, it } from 'vitest';
import {
  buildEffectivePlmnRows,
  parsePlmnList,
  serializePlmnList,
  validatePlmnList,
} from '../plmnList';

describe('parsePlmnList', () => {
  it('parses a comma-separated device value into editable rows', () => {
    expect(parsePlmnList('46000, 46001')).toEqual([
      { key: 'plmn-0-46000', plmn: '46000' },
      { key: 'plmn-1-46001', plmn: '46001' },
    ]);
  });
});

describe('serializePlmnList', () => {
  it('normalizes editable rows into the device comma-separated value', () => {
    expect(serializePlmnList([
      { key: '1', plmn: ' 46000 ' },
      { key: '2', plmn: '' },
      { key: '3', plmn: '46001' },
    ])).toBe('46000,46001');
  });
});

describe('validatePlmnList', () => {
  it('rejects a PLMN that is not five or six digits', () => {
    expect(validatePlmnList([{ key: '1', plmn: '4600' }], 6)).toBe('format');
  });

  it('rejects duplicate PLMNs', () => {
    expect(validatePlmnList([
      { key: '1', plmn: '46000' },
      { key: '2', plmn: '46000' },
    ], 6)).toBe('duplicate');
  });

  it('rejects more rows than the device limit', () => {
    expect(validatePlmnList(
      Array.from({ length: 7 }, (_, index) => ({
        key: String(index),
        plmn: `4600${index}`,
      })),
      6,
    )).toBe('limit');
  });
});

describe('buildEffectivePlmnRows', () => {
  it('combines retained instances, staged edits, and staged additions', () => {
    expect(buildEffectivePlmnRows({
      existingRows: [
        { key: '1', plmn: '46000' },
        { key: '2', plmn: '46001' },
      ],
      deletedKeys: new Set(['2']),
      editedValues: new Map([['1', '46002']]),
      addedRows: [{ key: 'new:1', plmn: '46002' }],
    })).toEqual([
      { key: '1', plmn: '46002' },
      { key: 'new:1', plmn: '46002' },
    ]);
  });
});
