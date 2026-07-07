import { beforeEach, describe, expect, it, vi } from 'vitest';

const { getMock, postMock } = vi.hoisted(() => ({ getMock: vi.fn(), postMock: vi.fn() }));

vi.mock('../../http', () => ({
  default: { get: getMock, post: postMock, patch: vi.fn(), delete: vi.fn() },
}));

import { agentApi } from '../agentApi';

beforeEach(() => {
  getMock.mockReset();
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

  it('reads runtime config from the authenticated API', async () => {
    getMock.mockResolvedValue({
      data: {
        enabled: true,
        endpoint: 'https://agent.example.com/api/action-connectors/c1/chat/stream',
        connectorId: 'c1',
        status: 'connected',
        lastValidatedAt: '2026-07-07T10:00:00Z',
        lastError: '',
        configuredSource: 'server',
      },
    });

    const result = await agentApi.getRuntimeConfig();

    expect(getMock).toHaveBeenCalledWith('/agent/config');
    expect(result.connectorId).toBe('c1');
  });

  it('syncs admin config through the admin API', async () => {
    const payload = {
      enabled: true,
      agentStudioBaseUrl: 'https://agent.example.com',
      agentStudioServiceToken: 'secret',
      omcPublicBaseUrl: 'https://ops.example.com',
    };
    postMock.mockResolvedValue({
      data: {
        enabled: true,
        agentStudioBaseUrl: payload.agentStudioBaseUrl,
        serviceTokenConfigured: true,
        omcPublicBaseUrl: payload.omcPublicBaseUrl,
        connectorSlug: 'external-agent',
        connectorId: 'c1',
        runtimeStreamUrl: 'https://agent.example.com/api/action-connectors/c1/chat/stream',
        status: 'connected',
        lastValidatedAt: '2026-07-07T10:00:00Z',
        lastError: '',
        healthPath: '/api/v1/agent/health',
        actionListPath: '/api/v1/agent-actions/actions',
        actionSearchPath: '/api/v1/agent-actions/actions/search',
        actionDescribePath: '/api/v1/agent-actions/actions/describe',
        actionPreviewPath: '/api/v1/agent-actions/actions/preview',
        actionExecutePath: '/api/v1/agent-actions/actions/execute',
      },
    });

    const result = await agentApi.syncAdminConfig(payload);

    expect(postMock).toHaveBeenCalledWith('/admin/agent-config/sync', payload);
    expect(result.status).toBe('connected');
  });
});
