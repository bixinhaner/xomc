import { describe, expect, it } from 'vitest';
import {
  normalizeProductTechnology,
  resolveProductClassTechnology,
  toSupportedProductClassOptions,
  toSupportedProductNameOptions,
} from './productClassOptions';

describe('toSupportedProductClassOptions', () => {
  it('merges observed classes with literal classes supported by the product catalog', () => {
    expect(toSupportedProductClassOptions(
      [' BLQ ', 'QAFA', '', 'BLQ'],
      [
        { patterns: ['^FAP/BU1810$', 'FAP/BU1521'] },
        { patterns: ['QAFA'] },
      ],
    )).toEqual([
      { label: 'BLQ', value: 'BLQ' },
      { label: 'FAP/BU1521', value: 'FAP/BU1521' },
      { label: 'FAP/BU1810', value: 'FAP/BU1810' },
      { label: 'QAFA', value: 'QAFA' },
    ]);
  });

  it('does not inject demo product classes when both APIs return no data', () => {
    expect(toSupportedProductClassOptions(undefined)).toEqual([]);
    expect(toSupportedProductClassOptions([])).toEqual([]);
  });

  it('does not expose wildcard product matching expressions as selectable classes', () => {
    expect(toSupportedProductClassOptions([], [
      {
        patterns: [
          String.raw`FAP/\w*BSC\w+`,
          'FAP/(QAFA|QAFB)',
          'FAP/BU1520',
        ],
      },
    ])).toEqual([
      { label: 'FAP/BU1520', value: 'FAP/BU1520' },
    ]);
  });

  it('only exposes product classes belonging to the selected technology', () => {
    const products = [
      { tech: '4G', patterns: ['^FAP/BU1810$', String.raw`FAP/\w*MLN\w+`] },
      { tech: '5G', patterns: ['^FAP/NR100$', String.raw`FAP/\w*GNB\w+`] },
      { tech: '2G', patterns: ['^FAP/BSC100$'] },
    ];

    expect(toSupportedProductClassOptions(
      ['FAP/BU1810', 'FAP/ABCMLN01', 'FAP/NR100', 'FAP/XYZGNB01', 'FAP/BSC100'],
      products,
      'lte',
    )).toEqual([
      { label: 'FAP/ABCMLN01', value: 'FAP/ABCMLN01' },
      { label: 'FAP/BU1810', value: 'FAP/BU1810' },
    ]);
  });

  it('normalizes catalog technology aliases and resolves an existing class', () => {
    expect(normalizeProductTechnology('4G')).toBe('lte');
    expect(normalizeProductTechnology('5G (NR)')).toBe('nr');
    expect(normalizeProductTechnology('GSM')).toBe('gsm');
    expect(resolveProductClassTechnology('FAP/BU1810', [
      { tech: '4G', patterns: ['^FAP/BU1810$'] },
      { tech: '5G', patterns: ['^FAP/NR100$'] },
    ])).toBe('lte');
  });
});

describe('toSupportedProductNameOptions', () => {
  it('only exposes catalog product names belonging to the selected technology', () => {
    const products = [
      { name: 'LTE Product', tech: '4G', patterns: ['^FAP/LTE$'] },
      { name: 'NR Product', tech: '5G', patterns: ['^FAP/NR$'] },
    ];

    expect(toSupportedProductNameOptions(
      products,
      'lte',
    )).toEqual([
      { label: 'LTE Product', value: 'LTE Product' },
    ]);
  });

  it('does not expose an unmatched product class as a product name', () => {
    expect(toSupportedProductNameOptions(
      [{ name: 'LTE Product', tech: '4G', patterns: ['^FAP/LTE$'] }],
    )).toEqual([
      { label: 'LTE Product', value: 'LTE Product' },
    ]);
  });
});
