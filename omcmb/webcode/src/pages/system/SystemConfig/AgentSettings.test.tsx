import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { IntlProvider } from 'react-intl';
import { getMessages } from '@core/i18n';
import AgentSettings from './AgentSettings';

const agentMocks = vi.hoisted(() => ({
  query: vi.fn(),
  refetch: vi.fn(),
  mutateAsync: vi.fn(),
}));

vi.mock('@core/hooks/api/useAgentConfig', () => ({
  useAdminAgentConfig: () => agentMocks.query(),
  useSaveAdminAgentConfig: () => ({ mutateAsync: agentMocks.mutateAsync, isPending: false }),
  useTestAdminAgentConfig: () => ({ mutateAsync: agentMocks.mutateAsync, isPending: false }),
  useSyncAdminAgentConfig: () => ({ mutateAsync: agentMocks.mutateAsync, isPending: false }),
}));

describe('AgentSettings', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    agentMocks.query.mockReturnValue({
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
      isError: false,
      isSuccess: true,
      refetch: agentMocks.refetch,
    });
  });

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

  it('配置加载失败时禁止 Agent 敏感操作并保留重试', () => {
    agentMocks.query.mockReturnValue({
      data: undefined,
      isLoading: false,
      isFetching: false,
      isError: true,
      isSuccess: false,
      refetch: agentMocks.refetch,
    });

    render(
      <IntlProvider locale="zh-CN" messages={getMessages('zh-CN')}>
        <AgentSettings />
      </IntlProvider>,
    );

    expect(screen.getByText('加载失败')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /测试连接/ })).toBeDisabled();
    expect(screen.getByRole('button', { name: /保存$/ })).toBeDisabled();
    expect(screen.getByRole('button', { name: /保存并同步/ })).toBeDisabled();

    const retryButton = screen.getByRole('button', { name: /重\s*试/ });
    expect(retryButton).toBeEnabled();
    retryButton.click();
    expect(agentMocks.refetch).toHaveBeenCalledTimes(1);
    expect(agentMocks.mutateAsync).not.toHaveBeenCalled();
  });

  it('操作校验期间配置进入刷新态时不提交 Agent 敏感操作', async () => {
    let queryState = agentMocks.query();
    agentMocks.query.mockImplementation(() => queryState);

    const renderAgent = () => (
      <IntlProvider locale="zh-CN" messages={getMessages('zh-CN')}>
        <AgentSettings />
      </IntlProvider>
    );
    const { rerender } = render(renderAgent());

    const saveButton = screen.getByRole('button', { name: /保存$/ });
    await waitFor(() => expect(saveButton).toBeEnabled());
    fireEvent.click(saveButton);
    queryState = { ...queryState, isFetching: true };
    rerender(renderAgent());

    await waitFor(() => expect(saveButton).toBeDisabled());
    await act(async () => {
      await new Promise<void>((resolve) => setTimeout(resolve, 0));
    });
    expect(agentMocks.mutateAsync).not.toHaveBeenCalled();
  });
});
