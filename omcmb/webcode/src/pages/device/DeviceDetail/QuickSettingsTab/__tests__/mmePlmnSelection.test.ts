import { describe, expect, it } from 'vitest';
import {
  findMmePlmnOutsideServingList,
  getServingPlmnOptionsForModel,
  getServingPlmnOptions,
  isServingPlmnRestrictedModel,
} from '../mmePlmnSelection';

describe('MME serving PLMN selection', () => {
  it('enables serving-list selection for MLN, BLQ, and BM', () => {
    expect(isServingPlmnRestrictedModel('MLN')).toBe(true);
    expect(isServingPlmnRestrictedModel('blq')).toBe(true);
    expect(isServingPlmnRestrictedModel('BM')).toBe(true);
    expect(isServingPlmnRestrictedModel('MLQ')).toBe(false);
  });

  it('normalizes and deduplicates serving PLMN candidates', () => {
    expect(getServingPlmnOptions('46000, 46001;46000\n46002')).toEqual([
      '46000',
      '46001',
      '46002',
    ]);
  });

  it('reads aggregate serving PLMNs for MLN and BM', () => {
    const parameters = [
      {
        parameterPath: 'Device.Services.FAPService.1.FAPControl.LTE.Gateway.ExistPlmnidList',
        parameterValue: '44190,44193',
      },
      {
        parameterPath: 'Device.Services.FAPService.2.FAPControl.LTE.Gateway.ExistPlmnidList',
        parameterValue: '46000',
      },
    ];
    expect(getServingPlmnOptionsForModel('BM', 1, parameters)).toEqual(['44190', '44193']);
    expect(getServingPlmnOptionsForModel('MLN', 2, parameters)).toEqual(['46000']);
  });

  it('reads BLQ serving PLMNs from the standard multi-instance PLMN list', () => {
    const parameters = [
      {
        parameterPath: 'Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.1.PLMNID',
        parameterValue: '222222',
      },
      {
        parameterPath: 'Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.2.PLMNID',
        parameterValue: '23031',
      },
      {
        parameterPath: 'Device.Services.FAPService.2.CellConfig.LTE.EPC.PLMNList.1.PLMNID',
        parameterValue: '333333',
      },
      {
        parameterPath: 'Device.Services.FAPService.1.FAPControl.LTE.Gateway.MmeIpPlmnList',
        parameterValue: '10.0.0.1+99999',
      },
    ];
    expect(getServingPlmnOptionsForModel('BLQ', 1, parameters)).toEqual([
      '222222',
      '23031',
    ]);
  });

  it('finds the first MME row whose PLMN is outside the serving list', () => {
    expect(findMmePlmnOutsideServingList([
      { mmeIp: '10.0.0.1', plmn: '46000' },
      { mmeIp: '10.0.0.2', plmn: '46099' },
    ], ['46000', '46001'])).toEqual({
      row: 2,
      plmn: '46099',
    });
  });

  it('accepts selected serving PLMNs and lets required-field validation handle blanks', () => {
    expect(findMmePlmnOutsideServingList([
      { mmeIp: '10.0.0.1', plmn: '46000' },
      { mmeIp: '', plmn: '' },
    ], ['46000'])).toBeNull();
  });
});
