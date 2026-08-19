import { beforeEach, describe, expect, it, vi } from 'vitest';

const { postMock, putMock } = vi.hoisted(() => ({
  postMock: vi.fn(),
  putMock: vi.fn(),
}));

vi.mock('../../http', () => ({
  default: { get: vi.fn(), post: postMock, put: putMock },
}));

import { storageProtectionApi, type StorageProtectionPolicyPayload } from '../storageProtectionApi';

const backendPolicy = {
  id: 'policy-data',
  target_type: 'filesystem',
  target_id: 'mount-data',
  write_scope: 'all',
  enabled: true,
  warn_used_percent: 70,
  recover_used_percent: 80,
  block_used_percent: 90,
  check_interval_seconds: 30,
  unknown_behavior: 'allow_with_alarm',
  current_state: 'normal',
  state_observations: 0,
};

const payload: StorageProtectionPolicyPayload = {
  targetType: 'filesystem',
  targetId: 'mount-data',
  writeScope: 'all',
  enabled: true,
  warnUsedPercent: 70,
  recoverUsedPercent: 80,
  blockUsedPercent: 90,
  checkIntervalSeconds: 30,
  unknownBehavior: 'allow_with_alarm',
};

beforeEach(() => {
  postMock.mockReset();
  putMock.mockReset();
});

describe('storageProtectionApi', () => {
  it('preserves directory target when creating a policy', async () => {
    postMock.mockResolvedValue({ data: backendPolicy });

    await storageProtectionApi.savePolicy(payload);

    expect(postMock).toHaveBeenCalledWith('/admin/storage-protection/policies', {
      target_type: 'filesystem',
      target_id: 'mount-data',
      write_scope: 'all',
      enabled: true,
      warn_used_percent: 70,
      recover_used_percent: 80,
      block_used_percent: 90,
      check_interval_seconds: 30,
      unknown_behavior: 'allow_with_alarm',
      updated_by: undefined,
    });
  });

  it('preserves directory target when updating a policy', async () => {
    putMock.mockResolvedValue({ data: backendPolicy });

    await storageProtectionApi.updatePolicy('policy-data', payload);

    expect(putMock).toHaveBeenCalledWith('/admin/storage-protection/policies/policy-data', expect.objectContaining({
      target_type: 'filesystem',
      target_id: 'mount-data',
      write_scope: 'all',
    }));
  });
});
