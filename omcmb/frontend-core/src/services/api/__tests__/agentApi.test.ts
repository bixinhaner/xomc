import { beforeEach, describe, expect, it, vi } from 'vitest';

const { postMock } = vi.hoisted(() => ({ postMock: vi.fn() }));

vi.mock('../../http', () => ({
  default: { get: vi.fn(), post: postMock, patch: vi.fn(), delete: vi.fn() },
}));

import { agentApi } from '../agentApi';

beforeEach(() => {
  postMock.mockReset();
});

describe('agentApi', () => {
  it('requests a short-lived agent delegation token from the authenticated API', async () => {
    postMock.mockResolvedValue({
      data: {
        token: 'delegated',
        expires_at: '2026-07-06T10:00:00Z',
      },
    });

    const result = await agentApi.requestDelegationToken();

    expect(postMock).toHaveBeenCalledWith('/agent/delegation');
    expect(result.token).toBe('delegated');
  });
});

