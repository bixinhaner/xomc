import { Form, Input } from 'antd';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import SystemConfig from './index';

const hookMocks = vi.hoisted(() => ({
  query: vi.fn(),
  mutateAsync: vi.fn(),
  refetch: vi.fn(),
  validate: vi.fn(),
}));

vi.mock('@core/hooks/api/useSystem', () => ({
  useSysConfigsByCategory: (...args: unknown[]) => hookMocks.query(...args),
  useBatchUpdateSysConfigs: () => ({
    mutateAsync: hookMocks.mutateAsync,
    isPending: false,
  }),
  useSysConfigApplyBatch: () => ({ data: undefined }),
}));

vi.mock('@/hooks/useT', () => ({
  useT: () => (key: string) => key,
}));

vi.mock('./BasicSettings', () => ({
  default: ({ form }: { form: ReturnType<typeof Form.useForm>[0] }) => (
    <Form form={form} initialValues={{ mrOMCName: '', timezoneCode: '' }}>
      <Form.Item
        name="mrOMCName"
        rules={[{ validator: () => hookMocks.validate() }]}
        validateTrigger={[]}
      >
        <Input aria-label="omc-name" />
      </Form.Item>
      <Form.Item name="timezoneCode">
        <Input aria-label="timezone" />
      </Form.Item>
    </Form>
  ),
}));

vi.mock('./SecuritySettings', () => ({ default: () => <div /> }));
vi.mock('./DeviceSettings', () => ({ default: () => <div /> }));
vi.mock('./StorageSettings', () => ({ default: () => <div /> }));
vi.mock('./TransferSettings', () => ({ default: () => <div /> }));
vi.mock('./AgentSettings', () => ({ default: () => <div /> }));
vi.mock('./PmRetentionSection', () => ({ default: () => <div /> }));
vi.mock('./RetentionBackpressureSection', () => ({ default: () => <div /> }));

function renderSystemConfig() {
  return render(
    <MemoryRouter>
      <SystemConfig />
    </MemoryRouter>,
  );
}

describe('SystemConfig load guard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    hookMocks.validate.mockResolvedValue(undefined);
  });

  it('配置加载失败时禁用保存并允许重试', () => {
    hookMocks.query.mockReturnValue({
      data: undefined,
      isFetching: false,
      isError: true,
      isSuccess: false,
      refetch: hookMocks.refetch,
    });

    renderSystemConfig();

    expect(screen.getByText('empty.loadFailed')).toBeInTheDocument();
    const saveButton = screen.getByRole('button', { name: /common\.save/ });
    expect(saveButton).toBeDisabled();
    fireEvent.click(saveButton);
    expect(hookMocks.mutateAsync).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole('button', { name: 'common.retry' }));
    expect(hookMocks.refetch).toHaveBeenCalledTimes(1);
  });

  it('配置加载成功后只保存管理员实际修改的字段', async () => {
    const user = userEvent.setup();
    hookMocks.query.mockReturnValue({
      data: [
        {
          id: '1',
          category: 'basic',
          key: 'mrOMCName',
          value: 'Original OMC',
          valueType: 'string',
          isPublic: true,
          isSecret: false,
        },
        {
          id: '2',
          category: 'basic',
          key: 'timezoneCode',
          value: 'Asia/Shanghai',
          valueType: 'string',
          isPublic: false,
          isSecret: false,
        },
      ],
      isFetching: false,
      isError: false,
      isSuccess: true,
      refetch: hookMocks.refetch,
    });
    hookMocks.mutateAsync.mockResolvedValue({
      batch: { id: 'batch-1', category: 'basic', status: 'applied', targets: [] },
    });

    renderSystemConfig();

    const nameInput = screen.getByLabelText('omc-name');
    await waitFor(() => expect(nameInput).toHaveValue('Original OMC'));
    await user.clear(nameInput);
    await user.type(nameInput, 'Updated OMC');
    await user.click(screen.getByRole('button', { name: /common\.save/ }));

    await waitFor(() => {
      expect(hookMocks.mutateAsync).toHaveBeenCalledWith({
        category: 'basic',
        items: [{ key: 'mrOMCName', value: 'Updated OMC', value_type: 'string' }],
      });
    });
  });

  it('配置分类为空时允许编辑，但未修改不提交', async () => {
    const user = userEvent.setup();
    hookMocks.query.mockReturnValue({
      data: [],
      isFetching: false,
      isError: false,
      isSuccess: true,
      refetch: hookMocks.refetch,
    });
    hookMocks.mutateAsync.mockResolvedValue({
      batch: { id: 'batch-empty', category: 'basic', status: 'applied', targets: [] },
    });

    renderSystemConfig();

    const saveButton = screen.getByRole('button', { name: /common\.save/ });
    expect(saveButton).toBeEnabled();
    await user.click(saveButton);
    expect(hookMocks.mutateAsync).not.toHaveBeenCalled();

    await user.type(screen.getByLabelText('omc-name'), 'New OMC');
    await user.click(saveButton);
    await waitFor(() => {
      expect(hookMocks.mutateAsync).toHaveBeenCalledWith({
        category: 'basic',
        items: [{ key: 'mrOMCName', value: 'New OMC', value_type: 'string' }],
      });
    });
  });

  it('表单校验期间配置进入刷新态时不提交', async () => {
    let releaseValidation!: () => void;
    hookMocks.validate.mockReturnValue(
      new Promise<void>((resolve) => {
        releaseValidation = resolve;
      }),
    );
    let queryState = {
      data: [
        {
          id: '1',
          category: 'basic',
          key: 'mrOMCName',
          value: 'Original OMC',
          valueType: 'string' as const,
          isPublic: true,
          isSecret: false,
        },
      ],
      isFetching: false,
      isError: false,
      isSuccess: true,
      refetch: hookMocks.refetch,
    };
    hookMocks.query.mockImplementation(() => queryState);

    const { rerender } = renderSystemConfig();

    const nameInput = screen.getByLabelText('omc-name');
    await waitFor(() => expect(nameInput).toHaveValue('Original OMC'));
    fireEvent.change(nameInput, { target: { value: 'Updated OMC' } });
    fireEvent.click(screen.getByRole('button', { name: /common\.save/ }));
    await waitFor(() => expect(hookMocks.validate).toHaveBeenCalled());

    queryState = { ...queryState, isFetching: true };
    rerender(
      <MemoryRouter>
        <SystemConfig />
      </MemoryRouter>,
    );
    await act(async () => releaseValidation());

    await waitFor(() => expect(hookMocks.mutateAsync).not.toHaveBeenCalled());
  });
});
