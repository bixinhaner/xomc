import { describe, expect, it } from 'vitest';
import { disabledAgentRuntimeConfig, resolveAgentRuntimeConfig } from './agent';

describe('resolveAgentRuntimeConfig', () => {
  it('does not enable direct browser-to-Agent-Studio runtime configuration', () => {
    expect(resolveAgentRuntimeConfig()).toEqual({
      enabled: false,
      endpoint: '',
      connectorId: '',
    });
  });

  it('returns a fresh disabled config object', () => {
    expect(disabledAgentRuntimeConfig()).toEqual({
      enabled: false,
      endpoint: '',
      connectorId: '',
    });
  });
});
