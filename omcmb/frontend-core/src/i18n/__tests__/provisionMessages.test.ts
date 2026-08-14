import { describe, expect, it } from 'vitest';
import { messages } from '../index';

describe('provision i18n messages', () => {
  it('describes verify_online as a cell activation check', () => {
    expect(messages['zh-CN']['provision.verifyOnline']).toBe('校验小区激活');
    expect(messages['en-US']['provision.verifyOnline']).toBe('Verify Cell Activation');
  });
});
