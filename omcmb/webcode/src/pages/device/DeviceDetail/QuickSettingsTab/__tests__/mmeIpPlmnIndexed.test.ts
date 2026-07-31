import { describe, expect, it } from 'vitest';
import {
  buildMlnMmePoolUpdates,
  getMlnMmePoolSyncPaths,
  isMlnIndexedMmePoolModel,
  parseMlnMmePoolRows,
} from '../mmeIpPlmnIndexed';

const currentMmePool = Array.from({ length: 16 }, (_, offset) => {
  const index = offset + 1;
  return [
    {
      parameterPath: `Device.Services.FAPService.CellConfig.LTE.MmePoolConfigParam.${index}.MMEIp1`,
      parameterValue: index === 1 ? '172.24.224.88' : '0.0.0.0',
    },
    {
      parameterPath: `Device.Services.FAPService.CellConfig.LTE.MmePoolConfigParam.${index}.PLMNID`,
      parameterValue: index === 1 ? '222222' : '000000',
    },
  ];
}).flat();

describe('MLN indexed MME pool adapter', () => {
  it('only enables the indexed representation for MLN', () => {
    expect(isMlnIndexedMmePoolModel('MLN')).toBe(true);
    expect(isMlnIndexedMmePoolModel('mln')).toBe(true);
    expect(isMlnIndexedMmePoolModel('BLQ')).toBe(false);
  });

  it('reads configured rows from the fixed indexed leaves and ignores empty slots', () => {
    expect(parseMlnMmePoolRows(currentMmePool)).toEqual([
      {
        key: 'mme-pool-1',
        mmeIp: '172.24.224.88',
        plmn: '222222',
      },
    ]);
  });

  it('writes a newly added second row to index 2 instead of the aggregate leaf', () => {
    const updates = buildMlnMmePoolUpdates([
      { key: 'row-1', mmeIp: '172.24.224.88', plmn: '222222' },
      { key: 'row-2', mmeIp: '172.24.224.49', plmn: '33333' },
    ], currentMmePool);

    expect(updates).toEqual([
      {
        parameterPath: 'Device.Services.FAPService.CellConfig.LTE.MmePoolConfigParam.2.MMEIp1',
        parameterValue: '172.24.224.49',
        parameterType: 'string',
      },
      {
        parameterPath: 'Device.Services.FAPService.CellConfig.LTE.MmePoolConfigParam.2.PLMNID',
        parameterValue: '33333',
        parameterType: 'string',
      },
    ]);
    expect(updates.every((item) => !item.parameterPath.endsWith('.MmeIpPlmnList'))).toBe(true);
  });

  it('resets an unused fixed index when a row is removed', () => {
    const withSecondRow = currentMmePool.map((item) => {
      if (item.parameterPath.endsWith('.2.MMEIp1')) {
        return { ...item, parameterValue: '172.24.224.49' };
      }
      if (item.parameterPath.endsWith('.2.PLMNID')) {
        return { ...item, parameterValue: '33333' };
      }
      return item;
    });

    expect(buildMlnMmePoolUpdates([
      { key: 'row-1', mmeIp: '172.24.224.88', plmn: '222222' },
    ], withSecondRow)).toEqual([
      {
        parameterPath: 'Device.Services.FAPService.CellConfig.LTE.MmePoolConfigParam.2.MMEIp1',
        parameterValue: '0.0.0.0',
        parameterType: 'string',
      },
      {
        parameterPath: 'Device.Services.FAPService.CellConfig.LTE.MmePoolConfigParam.2.PLMNID',
        parameterValue: '000000',
        parameterType: 'string',
      },
    ]);
  });

  it('uses all 16 indexed leaf pairs as the MLN sync target', () => {
    const paths = getMlnMmePoolSyncPaths();

    expect(paths).toHaveLength(32);
    expect(paths[0]).toBe('Device.Services.FAPService.CellConfig.LTE.MmePoolConfigParam.1.MMEIp1');
    expect(paths[31]).toBe('Device.Services.FAPService.CellConfig.LTE.MmePoolConfigParam.16.PLMNID');
    expect(paths.some((path) => path.endsWith('.MmeIpPlmnList'))).toBe(false);
  });
});
