import { render, screen, waitFor } from '@testing-library/react';
import { IntlProvider } from 'react-intl';
import { getMessages } from '@core/i18n';
import AgentSettings from './AgentSettings';

const refetch = vi.fn();
const mutateAsync = vi.fn();

vi.mock('@core/hooks/api/useAgentConfig', () => ({
  useAdminAgentConfig: () => ({
    data: {
      enabled: false,
      agentStudioBaseUrl: '',
      serviceTokenConfigured: false,
      connectorSlug: '',
      connectorId: '',
      runtimeStreamUrl: '',
      status: 'disabled',
      lastValidatedAt: '',
      lastError: '',
      policy: {
        allowedMethods: ['GET'],
        blockedPathPrefixes: [],
        toolTimeoutSeconds: 30,
        maxResponseBytes: 1048576,
      },
    },
    isLoading: false,
    isFetching: false,
    refetch,
  }),
  useSaveAdminAgentConfig: () => ({ mutateAsync, isPending: false }),
  useTestAdminAgentConfig: () => ({ mutateAsync, isPending: false }),
  useSyncAdminAgentConfig: () => ({ mutateAsync, isPending: false }),
}));

describe('AgentSettings', () => {
  it('不使用 Ant Design 6 已废弃属性，并关闭非账号字段的自动填充推断', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);

    render(
      <IntlProvider locale="zh-CN" messages={getMessages('zh-CN')}>
        <AgentSettings />
      </IntlProvider>,
    );

    await waitFor(() => expect(screen.getByDisplayValue('30')).toBeInTheDocument());
    expect(screen.getByPlaceholderText('https://agent.example.com')).toHaveAttribute('autocomplete', 'off');
    expect(screen.getByPlaceholderText('external-agent-...')).toHaveAttribute('autocomplete', 'off');

    const output = consoleError.mock.calls.flat().join(' ');
    expect(output).not.toMatch(/addonBefore|addonAfter|\[antd: Alert\].*message|\[antd: Space\].*direction/);
    consoleError.mockRestore();
  });
});
