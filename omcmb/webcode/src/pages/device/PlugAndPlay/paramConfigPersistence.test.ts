import { describe, expect, it } from 'vitest';
import { buildParamConfigListPolicyUpdate } from './paramConfigPersistence';

describe('parameter configuration persistence', () => {
  it('updates only paramConfigList while preserving the persisted policy', () => {
    const policy = {
      id: 'policy-1',
      name: 'Policy 1',
      enabled: true,
      productName: 'BNQ',
      productNames: ['BNQ'],
      productClass: 'FAP',
      productClasses: ['FAP'],
      executeType: 'auto' as const,
      priority: 100,
      upgradeEnabled: false,
      targetVersion: '',
      licenseEnabled: false,
      selfConfigEnabled: true,
      config: {
        untouched: 'keep-me',
        paramConfigList: [{ id: 'remove-me', serialNumber: 'SN-1' }],
      },
      createdAt: 'created',
      updatedAt: 'updated',
    };

    expect(buildParamConfigListPolicyUpdate(policy, [])).toEqual({
      name: 'Policy 1',
      enabled: true,
      productName: 'BNQ',
      productNames: ['BNQ'],
      productClass: 'FAP',
      productClasses: ['FAP'],
      executeType: 'auto',
      priority: 100,
      upgradeEnabled: false,
      targetVersion: '',
      licenseEnabled: false,
      selfConfigEnabled: true,
      config: {
        untouched: 'keep-me',
        paramConfigList: [],
      },
    });
  });
});
