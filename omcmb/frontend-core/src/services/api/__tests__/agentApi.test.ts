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
  it('reads visibility config from the authenticated API', async () => {
    getMock.mockResolvedValue({
      data: {
        visible: true,
        enabled: false,
        status: 'not_configured',
        lastValidatedAt: '',
        lastError: '',
      },
    });

    const result = await agentApi.getVisibilityConfig();

    expect(getMock).toHaveBeenCalledWith('/agent/visibility');
    expect(result.visible).toBe(true);
    expect(result.enabled).toBe(false);
  });

  it('reads runtime config from the authenticated API', async () => {
    getMock.mockResolvedValue({
      data: {
        visible: true,
        enabled: true,
        endpoint: '/api/v1/agent/chat/stream',
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

  it('reads the active agent conversation', async () => {
    getMock.mockResolvedValue({ data: { conversationId: 'conversation-1' } });

    const result = await agentApi.getConversation();

    expect(getMock).toHaveBeenCalledWith('/agent/conversation');
    expect(result.conversationId).toBe('conversation-1');
  });

  it('starts a new agent conversation', async () => {
    postMock.mockResolvedValue({ data: { conversationId: 'conversation-2' } });

    const result = await agentApi.startConversation();

    expect(postMock).toHaveBeenCalledWith('/agent/conversation');
    expect(result.conversationId).toBe('conversation-2');
  });

  it('syncs admin config through the admin API', async () => {
    const payload = {
      enabled: true,
      agentStudioBaseUrl: 'https://agent.example.com',
      agentStudioServiceToken: 'secret',
      allowedMethods: ['GET'],
      blockedPathPrefixes: ['/api/v1/auth/*'],
      toolTimeoutSeconds: 30,
      maxResponseBytes: 262144,
    };
    postMock.mockResolvedValue({
      data: {
        enabled: true,
        agentStudioBaseUrl: payload.agentStudioBaseUrl,
        serviceTokenConfigured: true,
        connectorSlug: 'external-agent',
        connectorId: 'c1',
        runtimeStreamUrl: '/api/v1/agent/chat/stream',
        status: 'connected',
        lastValidatedAt: '2026-07-07T10:00:00Z',
        lastError: '',
        policy: {
          allowedMethods: payload.allowedMethods,
          blockedPathPrefixes: payload.blockedPathPrefixes,
          toolTimeoutSeconds: payload.toolTimeoutSeconds,
          maxResponseBytes: payload.maxResponseBytes,
        },
      },
    });

    const result = await agentApi.syncAdminConfig(payload);

    expect(postMock).toHaveBeenCalledWith('/admin/agent-config/sync', payload);
    expect(result.status).toBe('connected');
  });
});
