import { describe, expect, it } from 'vitest';
import { resolveAgentRuntimeConfig } from './agent';

describe('resolveAgentRuntimeConfig', () => {
  it('builds stream endpoint from runtime base URL and connector ID', () => {
    expect(
      resolveAgentRuntimeConfig({
        VITE_AGENT_RUNTIME_BASE_URL: 'http://localhost:4000/',
        VITE_AGENT_ACTION_CONNECTOR_ID: 'connector 1',
      })
    ).toEqual({
      enabled: true,
      endpoint: 'http://localhost:4000/api/action-connectors/connector%201/chat/stream',
      connectorId: 'connector 1',
    });
  });

  it('uses direct stream URL when provided', () => {
    expect(
      resolveAgentRuntimeConfig({
        VITE_AGENT_RUNTIME_STREAM_URL: 'http://localhost:4000/custom/stream',
        VITE_AGENT_RUNTIME_BASE_URL: 'http://localhost:4000',
        VITE_AGENT_ACTION_CONNECTOR_ID: 'connector-1',
      })
    ).toEqual({
      enabled: true,
      endpoint: 'http://localhost:4000/custom/stream',
      connectorId: 'connector-1',
    });
  });

  it('disables runtime when base URL or connector ID is missing', () => {
    expect(resolveAgentRuntimeConfig({ VITE_AGENT_RUNTIME_BASE_URL: 'http://localhost:4000' })).toEqual({
      enabled: false,
      endpoint: '',
      connectorId: '',
    });
  });
});

