import { describe, expect, it } from 'vitest';
import {
  getParamConfigTemplate,
  toParamConfigDeviceType,
} from './paramConfigTemplate';

describe('parameter configuration default templates', () => {
  it('uses the supplied 4G workbook for LTE products', () => {
    expect(getParamConfigTemplate('eNB')).toEqual({
      url: '/templates/selfConfiguration_qa.xlsx',
      fileName: 'selfConfiguration_qa.xlsx',
    });
  });

  it('uses the supplied 5G workbook for NR products', () => {
    expect(getParamConfigTemplate('gNB')).toEqual({
      url: '/templates/5G_Autonomous_Self-Deployment_Template.xlsx',
      fileName: '5G_Autonomous_Self-Deployment_Template.xlsx',
    });
  });

  it('provides a dedicated GSM planning workbook', () => {
    expect(getParamConfigTemplate('GSM')).toEqual({
      url: '/templates/GSM_Autonomous_Self-Deployment_Template.xlsx',
      fileName: 'GSM_Autonomous_Self-Deployment_Template.xlsx',
    });
    expect(getParamConfigTemplate(undefined)).toBeUndefined();
  });

  it('infers the parameter configuration type from product technology metadata', () => {
    expect(toParamConfigDeviceType('lte')).toBe('eNB');
    expect(toParamConfigDeviceType('nr')).toBe('gNB');
    expect(toParamConfigDeviceType('gsm')).toBe('GSM');
    expect(toParamConfigDeviceType('lte', 'BM')).toBe('GSM');
    expect(toParamConfigDeviceType(undefined)).toBeUndefined();
  });
});
