import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import NotificationSettings from './NotificationSettings';

const emailMocks = vi.hoisted(() => ({
  query: vi.fn(),
  save: vi.fn(),
  sendTest: vi.fn(),
  refetch: vi.fn(),
}));

vi.mock('@core/hooks/api/useSystem', () => ({
  useSysConfigsByCategory: (category: string) => emailMocks.query(category),
  useBatchUpdateSysConfigs: () => ({ mutateAsync: emailMocks.save, isPending: false }),
  useSendTestEmail: () => ({ mutateAsync: emailMocks.sendTest, isPending: false }),
}));

vi.mock('@/hooks/useT', () => ({
  useT: () => (key: string) => key,
}));

const rows = [
  ['enabled', 'true', 'bool'],
  ['host', 'smtp.example.test', 'string'],
  ['port', '587', 'int'],
  ['security_mode', 'starttls', 'string'],
  ['auth_enabled', 'true', 'bool'],
  ['username', 'omc@example.test', 'string'],
  ['password', '', 'string'],
  ['from_address', 'omc@example.test', 'string'],
  ['from_name', 'Test OMC', 'string'],
  ['timeout_seconds', '10', 'int'],
].map(([key, value, valueType]) => ({
  id: key,
  category: 'notification.email',
  key,
  value,
  valueType,
  isSecret: key === 'password',
  isConfigured: key === 'password',
}));

describe('NotificationSettings', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    emailMocks.query.mockReturnValue({
      data: rows,
      isLoading: false,
      isFetching: false,
      isError: false,
      isSuccess: true,
      refetch: emailMocks.refetch,
    });
    emailMocks.save.mockResolvedValue({});
    emailMocks.sendTest.mockResolvedValue({ recipient: 'operator@example.test' });
    emailMocks.refetch.mockResolvedValue({});
  });

  it('只展示真实邮件通道，并且测试只发送给本次填写的单一地址', async () => {
    const user = userEvent.setup();
    render(<NotificationSettings />);

    expect(screen.queryByText('system.notification.smsService')).not.toBeInTheDocument();
    expect(screen.getByPlaceholderText('system.notification.passwordConfiguredPlaceholder')).toHaveValue('');
    expect(screen.getByTestId('email-settings-actions')).toHaveStyle({
      display: 'flex',
      justifyContent: 'flex-end',
    });

    const recipient = screen.getByPlaceholderText('system.notification.testRecipientPlaceholder');
    await user.type(recipient, 'operator@example.test');
    await user.click(screen.getByRole('button', { name: /common\.test/ }));

    await waitFor(() => expect(emailMocks.sendTest).toHaveBeenCalledWith('operator@example.test'));
  });

  it('表单有未保存修改时禁止测试，保存时不覆盖已配置密码', async () => {
    const user = userEvent.setup();
    render(<NotificationSettings />);

    const recipient = screen.getByPlaceholderText('system.notification.testRecipientPlaceholder');
    await user.type(recipient, 'operator@example.test');
    const testButton = screen.getByRole('button', { name: /common\.test/ });
    expect(testButton).toBeEnabled();

    const host = await screen.findByDisplayValue('smtp.example.test');
    await user.clear(host);
    await user.type(host, 'smtp.changed.test');
    expect(testButton).toBeDisabled();

    await user.click(screen.getByRole('button', { name: /common\.save/ }));
    await waitFor(() => expect(emailMocks.save).toHaveBeenCalledTimes(1));
    const payload = emailMocks.save.mock.calls[0][0] as {
      category: string;
      items: Array<{ key: string; value: string }>;
    };
    expect(payload.category).toBe('notification.email');
    expect(payload.items).not.toContainEqual(expect.objectContaining({ key: 'password' }));
  });

  it('配置加载失败时禁止保存并保留重试', async () => {
    const user = userEvent.setup();
    emailMocks.query.mockReturnValue({
      data: undefined,
      isLoading: false,
      isFetching: false,
      isError: true,
      isSuccess: false,
      refetch: emailMocks.refetch,
    });

    render(<NotificationSettings />);
    expect(screen.getByRole('button', { name: /common\.save/ })).toBeDisabled();
    await user.click(screen.getByRole('button', { name: 'common.retry' }));
    expect(emailMocks.refetch).toHaveBeenCalledTimes(1);
    expect(emailMocks.save).not.toHaveBeenCalled();
  });
});
