import { describe, expect, it } from 'vitest';
import { messages } from '../index';

describe('provision i18n messages', () => {
  it('describes NR startup reports without claiming that OMC activated the cell', () => {
    expect(messages['zh-CN']['provision.startupStageReport']).toBe('开站阶段上报（Stage 3）');
    expect(messages['en-US']['provision.startupStageReport']).toBe('Startup Stage Report (Stage 3)');
    expect(messages['zh-CN']['provision.waitStartupResult']).toBe('确认开站结果');
    expect(messages['en-US']['provision.waitStartupResult']).toBe('Confirm Startup Result');
    expect(messages['zh-CN']['provision.confirmStartupResult']).toBe('确认开站结果');
    expect(messages['en-US']['provision.confirmStartupResult']).toBe('Confirm Startup Result');
    expect(messages['zh-CN']['provision.recordStartupSuccess']).toBe('记录开站成功');
    expect(messages['en-US']['provision.recordStartupSuccess']).toBe('Record Startup Success');
    expect(messages['zh-CN']['provision.verifyOnline']).toBe('记录开站成功');
    expect(messages['en-US']['provision.verifyOnline']).toBe('Record Startup Success');
  });
});
