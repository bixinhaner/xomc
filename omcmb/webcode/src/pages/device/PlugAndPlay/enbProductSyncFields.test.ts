import { describe, expect, it } from 'vitest';
import { getEnbProductSyncConfig } from './enbProductSyncFields';

describe('getEnbProductSyncConfig', () => {
  it.each([
    ['MLN', ['tfcsManagerPrimsrc'], ['1', '2', '3', '4', '5', '6', '7', '8', '9', '10']],
    ['MLQ', ['tfcsManagerPrimsrc'], ['1', '2', '4', '5', '6', '7', '8', '9', '10', '11', '31', '32']],
    ['BLQ', ['tfcsManagerPrimsrc'], ['2', '3', '7', '8', '9', '10', '11']],
  ])('uses the %s synchronization parameter and enum', (paramModel, ids, values) => {
    const config = getEnbProductSyncConfig(paramModel);
    expect(config.fields.map((field) => field.id)).toEqual(ids);
    expect(config.fields[0].options?.map((option) => option.value)).toEqual(values);
  });

  it('uses BM-specific synchronization and PTP fields', () => {
    const config = getEnbProductSyncConfig('BM');
    expect(config.fields.slice(0, 2).map((field) => field.id)).toEqual(['PpsTimeMode', 'SyncSource']);
    expect(config.fields[0].options?.map((option) => option.value)).toEqual(['1', '2', '4', '8', '16']);
    expect(config.ptpModeValues).toEqual(['2']);
  });

  it.each(['BLN', 'ENB_DEFAULT_098', 'ENB_DEFAULT_181', undefined])(
    'hides synchronization settings when %s has no matching product fields',
    (paramModel) => {
      expect(getEnbProductSyncConfig(paramModel).fields).toEqual([]);
    },
  );
});
