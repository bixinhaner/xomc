import { beforeEach, describe, expect, it, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { useQuickSettingsFeedbackStore } from '@core/store/quickSettingsFeedbackStore';
import { feedbackKey } from '@core/store/quickSettingsFeedbackStore';
import type { QuickSettingsGroup } from '@core/types/quicksettings';
import CellParameterForm from '../CellParameterForm';
import MultiInstanceTable from '../MultiInstanceTable';

const mocks = vi.hoisted(() => ({
  globalCurrentValue: '0' as string | undefined,
  updateParameters: vi.fn(),
  addObject: vi.fn(),
  deleteObject: vi.fn(),
  getTask: vi.fn(),
  syncDeviceParams: vi.fn(),
  readbackGlobalValue: '1',
  searchParameters: vi.fn(),
  taskStatus: 'completed',
  refetchSchema: vi.fn(),
  invalidateParameterSchemaCache: vi.fn(),
}));

const ipsecSchema = {
  parameters: [
    {
      path: 'Device.Services.FAPService.Ipsec.IPSEC_ENABLE',
      type: 'BOOLEAN',
      writable: true,
      currentValue: '0',
    },
    {
      path: 'Device.FAP.Ipsec.1.TUNNEL_ENABLE',
      type: 'BOOLEAN',
      writable: true,
      currentValue: 'true',
    },
  ],
  objects: [{
    path: 'Device.FAP.Ipsec.',
    currentInstances: [1],
    canAdd: true,
    canDeleteAny: true,
  }],
  total: 3,
};

vi.mock('@core/hooks/api/useDeviceParameters', () => ({
  useParameterSchema: () => ({
    data: ipsecSchema,
    isLoading: false,
    refetch: mocks.refetchSchema,
  }),
  useSearchParameters: () => ({
    data: [{
      parameterPath: 'Device.Services.FAPService.Ipsec.IPSEC_ENABLE',
      parameterValue: mocks.globalCurrentValue,
      parameterType: 'BOOLEAN',
      writable: true,
    }],
  }),
  useUpdateParameters: () => ({
    isPending: false,
    mutateAsync: mocks.updateParameters,
  }),
  useAddObject: () => ({ isPending: false, mutateAsync: mocks.addObject }),
  useDeleteObject: () => ({ isPending: false, mutateAsync: mocks.deleteObject }),
}));

vi.mock('@core/hooks/api/useDeviceTask', () => ({
  useDeviceTaskStatus: (taskId?: string) => ({
    data: taskId ? { id: taskId, status: mocks.taskStatus, result: {} } : undefined,
  }),
}));

vi.mock('@core/hooks/api/useDevices', () => ({
  useSyncDeviceParams: () => ({
    mutateAsync: mocks.syncDeviceParams,
  }),
}));

vi.mock('@core/services/api/deviceTaskApi', () => ({
  deviceTaskApi: {
    getTask: mocks.getTask,
  },
}));

vi.mock('@core/services/api/deviceParameterApi', () => ({
  deviceParameterApi: {
    invalidateParameterSchemaCache: mocks.invalidateParameterSchemaCache,
    searchParameters: mocks.searchParameters,
  },
}));

vi.mock('@/hooks/useT', () => ({
  useT: () => (id: string, values?: Record<string, string | number>) => (
    values?.count === undefined ? id : `${id}:${values.count}`
  ),
}));

const ipsecControlGroup: QuickSettingsGroup = {
  id: 'device-ipsec-control',
  titleZh: 'IPSec 配置',
  titleEn: 'IPSec Config',
  multiInstance: false,
  params: [{
    name: 'IPSEC_ENABLE',
    titleZh: 'IPSec 开关',
    titleEn: 'IPSec Enable',
    standardPath: 'Device.Services.FAPService.Ipsec.IPSEC_ENABLE',
    enumOptions: [
      { value: '1', label: '开启' },
      { value: '0', label: '关闭' },
    ],
  }],
};

const ipsecTunnelGroup: QuickSettingsGroup = {
  id: 'device-ipsec',
  titleZh: 'IPSec 参数',
  titleEn: 'IPSec Settings',
  multiInstance: true,
  maxInstances: 4,
  objectPath: 'Device.FAP.Ipsec.{i}.',
  params: [{
    name: 'TUNNEL_ENABLE',
    titleZh: 'TunnelEnable',
    titleEn: 'Tunnel Enable',
    leaf: 'TUNNEL_ENABLE',
    defaultValue: 'true',
  }],
};

function renderControl(actionMode: 'standalone' | 'staged') {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );

  return render(
    <CellParameterForm
      deviceId="device-1"
      group={ipsecControlGroup}
      instanceContext={{ networkType: 'lte', fapInstance: 1 }}
      locale="zh-CN"
      actionMode={actionMode}
    />,
    { wrapper },
  );
}

describe('IPSec unified submit actions', () => {
  beforeEach(() => {
    mocks.globalCurrentValue = '0';
    mocks.updateParameters.mockReset();
    mocks.addObject.mockReset();
    mocks.deleteObject.mockReset();
    mocks.getTask.mockReset();
    mocks.syncDeviceParams.mockReset();
    mocks.searchParameters.mockReset();
    mocks.readbackGlobalValue = '1';
    mocks.taskStatus = 'completed';
    mocks.refetchSchema.mockReset();
    mocks.refetchSchema.mockResolvedValue({
      data: {
        ...ipsecSchema,
        parameters: ipsecSchema.parameters.map((item) => (
          item.path === 'Device.FAP.Ipsec.1.TUNNEL_ENABLE'
            ? { ...item, currentValue: 'false' }
            : item
        )),
      },
    });
    mocks.invalidateParameterSchemaCache.mockReset();
    mocks.searchParameters.mockImplementation(async () => [{
      parameterPath: 'Device.Services.FAPService.Ipsec.IPSEC_ENABLE',
      parameterValue: mocks.readbackGlobalValue,
      parameterType: 'BOOLEAN',
      writable: true,
    }]);
    mocks.syncDeviceParams.mockResolvedValue({ targetCount: 1, gpvTaskCount: 0 });
    mocks.updateParameters
      .mockResolvedValueOnce({ taskId: 'task-1' })
      .mockResolvedValueOnce({ taskId: 'task-2' });
    mocks.addObject.mockResolvedValue({ taskId: 'add-task' });
    mocks.deleteObject.mockResolvedValue({ taskId: 'rollback-task' });
    mocks.getTask.mockImplementation(async (taskId: string) => ({
      id: taskId,
      status: 'completed',
      result: {},
    }));
    useQuickSettingsFeedbackStore.setState({
      entries: {},
      drafts: {},
      draftRevisions: {},
    });
  });

  it('hides the independent save action when the control form is staged', () => {
    renderControl('staged');

    expect(screen.queryByRole('button', { name: 'common.save' })).not.toBeInTheDocument();
  });

  it('keeps the save action for standalone forms', () => {
    renderControl('standalone');

    expect(screen.getByRole('button', { name: 'common.save' })).toBeInTheDocument();
  });

  it('refreshes related quick-settings lists after a failed device response', async () => {
    mocks.taskStatus = 'failed';
    const fbKey = feedbackKey('device-1', 'device-ipsec-control', 1);
    useQuickSettingsFeedbackStore.getState().setDraftField(fbKey, 'IPSEC_ENABLE', '1');
    const submittedDraftRevision = useQuickSettingsFeedbackStore.getState().draftRevisions[fbKey];
    useQuickSettingsFeedbackStore.getState().setFeedback(fbKey, {
      kind: 'cell',
      submitStatus: 'queued',
      taskId: 'failed-task',
      count: 1,
      at: Date.now(),
      expectedReadback: {
        'Device.Services.FAPService.Ipsec.IPSEC_ENABLE': '1',
      },
      submittedDraftRevision,
    });
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    const invalidateQueries = vi.spyOn(queryClient, 'invalidateQueries');

    render(
      <QueryClientProvider client={queryClient}>
        <CellParameterForm
          deviceId="device-1"
          group={ipsecControlGroup}
          instanceContext={{ networkType: 'lte', fapInstance: 1 }}
          locale="zh-CN"
          actionMode="standalone"
        />
      </QueryClientProvider>,
    );

    await waitFor(() => {
      expect(mocks.invalidateParameterSchemaCache).toHaveBeenCalledWith('device-1');
      expect(invalidateQueries).toHaveBeenCalledWith(expect.objectContaining({
        queryKey: ['devices', 'parameters', 'device-1'],
      }));
      expect(invalidateQueries).toHaveBeenCalledWith(expect.objectContaining({
        queryKey: ['devices', 'parameter-schema', 'device-1'],
      }));
      expect(invalidateQueries).toHaveBeenCalledWith(expect.objectContaining({
        queryKey: ['devices', 'parameters', 'search', 'device-1'],
      }));
      expect(invalidateQueries).toHaveBeenCalledWith(expect.objectContaining({
        queryKey: ['devices', 'parameter-tree', 'device-1'],
      }));
    });
    await waitFor(() => expect(mocks.refetchSchema).toHaveBeenCalled());
    await waitFor(() => {
      expect(useQuickSettingsFeedbackStore.getState().drafts[fbKey]).toBeUndefined();
      expect(screen.getByText('关闭')).toBeInTheDocument();
    });
  });

  it('shows the unified action when only the global switch changed', () => {
    const controlDraftKey = feedbackKey('device-1', 'device-ipsec-control', 1);
    useQuickSettingsFeedbackStore.getState().setDraftField(controlDraftKey, 'IPSEC_ENABLE', '1');
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <MultiInstanceTable
          deviceId="device-1"
          group={ipsecTunnelGroup}
          instanceContext={{ networkType: 'lte', fapInstance: 1 }}
          locale="zh-CN"
          ipsecControlValue="1"
        />
      </QueryClientProvider>,
    );

    expect(screen.getByRole('button', {
      name: /device\.ipsec\.unifiedSubmitWithCount:1/,
    })).toBeInTheDocument();
  });

  it('does not submit tunnel changes before the global switch readback is available', () => {
    mocks.globalCurrentValue = undefined;
    const tunnelDraftKey = feedbackKey('device-1', 'device-ipsec', 1);
    useQuickSettingsFeedbackStore.getState().setDraftField(tunnelDraftKey, '1.TUNNEL_ENABLE', 'false');
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <MultiInstanceTable
          deviceId="device-1"
          group={ipsecTunnelGroup}
          instanceContext={{ networkType: 'lte', fapInstance: 1 }}
          locale="zh-CN"
        />
      </QueryClientProvider>,
    );

    const submit = screen.getByRole('button', {
      name: /device\.ipsec\.unifiedSubmitWithCount:1/,
    });
    expect(submit).toBeDisabled();
    fireEvent.click(submit);
    expect(mocks.updateParameters).not.toHaveBeenCalled();
  });

  it('waits for global enable before submitting tunnel edits', async () => {
    const controlDraftKey = feedbackKey('device-1', 'device-ipsec-control', 1);
    const tunnelDraftKey = feedbackKey('device-1', 'device-ipsec', 1);
    useQuickSettingsFeedbackStore.getState().setDraftField(controlDraftKey, 'IPSEC_ENABLE', '1');
    useQuickSettingsFeedbackStore.getState().setDraftField(tunnelDraftKey, '1.TUNNEL_ENABLE', 'false');
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <MultiInstanceTable
          deviceId="device-1"
          group={ipsecTunnelGroup}
          instanceContext={{ networkType: 'lte', fapInstance: 1 }}
          locale="zh-CN"
          ipsecControlValue="1"
        />
      </QueryClientProvider>,
    );

    const submit = await screen.findByRole('button', {
      name: /device\.ipsec\.unifiedSubmitWithCount:2/,
    });
    fireEvent.click(submit);

    await waitFor(() => expect(mocks.updateParameters).toHaveBeenCalledTimes(2));
    expect(mocks.updateParameters.mock.calls[0][0].parameters).toEqual([{
      parameterPath: 'Device.Services.FAPService.Ipsec.IPSEC_ENABLE',
      parameterValue: '1',
      parameterType: 'boolean',
    }]);
    expect(mocks.updateParameters.mock.calls[1][0].parameters).toEqual([{
      parameterPath: 'Device.FAP.Ipsec.1.TUNNEL_ENABLE',
      parameterValue: 'false',
      parameterType: 'boolean',
    }]);
    expect(mocks.getTask).toHaveBeenCalledWith('task-1');
    await waitFor(() => {
      expect(useQuickSettingsFeedbackStore.getState().entries[tunnelDraftKey]).toMatchObject({
        ipsecTaskIds: ['task-1', 'task-2'],
        ipsecTargetEnabled: true,
      });
    });
  });

  it('persists the running operation across remounts and blocks duplicate submit', async () => {
    const taskResolvers = new Map<string, (task: {
      id: string;
      status: string;
      result: Record<string, unknown>;
    }) => void>();
    mocks.getTask.mockImplementation((taskId: string) => new Promise((resolve) => {
      taskResolvers.set(taskId, resolve);
    }));
    const controlDraftKey = feedbackKey('device-1', 'device-ipsec-control', 1);
    const tunnelDraftKey = feedbackKey('device-1', 'device-ipsec', 1);
    useQuickSettingsFeedbackStore.getState().setDraftField(controlDraftKey, 'IPSEC_ENABLE', '1');
    useQuickSettingsFeedbackStore.getState().setDraftField(tunnelDraftKey, '1.TUNNEL_ENABLE', 'false');
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    const props = {
      deviceId: 'device-1',
      group: ipsecTunnelGroup,
      instanceContext: { networkType: 'lte' as const, fapInstance: 1 },
      locale: 'zh-CN' as const,
      ipsecControlValue: '1',
    };

    const first = render(
      <QueryClientProvider client={queryClient}>
        <MultiInstanceTable {...props} />
      </QueryClientProvider>,
    );
    fireEvent.click(await screen.findByRole('button', {
      name: /device\.ipsec\.unifiedSubmitWithCount:2/,
    }));

    await waitFor(() => expect(mocks.updateParameters).toHaveBeenCalledTimes(1));
    expect(useQuickSettingsFeedbackStore.getState().entries[tunnelDraftKey]).toMatchObject({
      ipsecOperationStatus: 'running',
      ipsecActivePhase: 'enable-global',
      ipsecTaskIds: ['task-1'],
    });

    first.unmount();
    const remounted = render(
      <QueryClientProvider client={queryClient}>
        <MultiInstanceTable {...props} />
      </QueryClientProvider>,
    );
    const remountedSubmit = await screen.findByRole('button', {
      name: /device\.ipsec\.unifiedSubmitWithCount:2/,
    });
    expect(remountedSubmit).toBeDisabled();
    fireEvent.click(remountedSubmit);
    expect(mocks.updateParameters).toHaveBeenCalledTimes(1);

    await act(async () => {
      taskResolvers.get('task-1')?.({ id: 'task-1', status: 'completed', result: {} });
    });
    await waitFor(() => expect(mocks.updateParameters).toHaveBeenCalledTimes(2));
    await act(async () => {
      taskResolvers.get('task-2')?.({ id: 'task-2', status: 'completed', result: {} });
    });
    await waitFor(() => {
      expect(useQuickSettingsFeedbackStore.getState().entries[tunnelDraftKey]).toMatchObject({
        ipsecOperationStatus: 'completed',
      });
    });
    mocks.globalCurrentValue = '1';
    remounted.rerender(
      <QueryClientProvider client={queryClient}>
        <MultiInstanceTable {...props} ipsecControlValue={undefined} />
      </QueryClientProvider>,
    );
    await waitFor(() => {
      expect(screen.queryByRole('button', {
        name: /device\.ipsec\.unifiedSubmitWithCount/,
      })).not.toBeInTheDocument();
    });
  });

  it('keeps edits made while the unified operation is running', async () => {
    const taskResolvers = new Map<string, (task: {
      id: string;
      status: string;
      result: Record<string, unknown>;
    }) => void>();
    mocks.getTask.mockImplementation((taskId: string) => new Promise((resolve) => {
      taskResolvers.set(taskId, resolve);
    }));
    const controlDraftKey = feedbackKey('device-1', 'device-ipsec-control', 1);
    const tunnelDraftKey = feedbackKey('device-1', 'device-ipsec', 1);
    useQuickSettingsFeedbackStore.getState().setDraftField(controlDraftKey, 'IPSEC_ENABLE', '1');
    useQuickSettingsFeedbackStore.getState().setDraftField(tunnelDraftKey, '1.TUNNEL_ENABLE', 'false');
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <MultiInstanceTable
          deviceId="device-1"
          group={ipsecTunnelGroup}
          instanceContext={{ networkType: 'lte', fapInstance: 1 }}
          locale="zh-CN"
          ipsecControlValue="1"
        />
      </QueryClientProvider>,
    );
    fireEvent.click(await screen.findByRole('button', {
      name: /device\.ipsec\.unifiedSubmitWithCount:2/,
    }));

    await waitFor(() => expect(mocks.updateParameters).toHaveBeenCalledTimes(1));
    useQuickSettingsFeedbackStore.getState().setDraftField(
      tunnelDraftKey,
      '1.TUNNEL_ENABLE',
      'true',
    );
    await act(async () => {
      taskResolvers.get('task-1')?.({ id: 'task-1', status: 'completed', result: {} });
    });
    await waitFor(() => expect(mocks.updateParameters).toHaveBeenCalledTimes(2));
    await act(async () => {
      taskResolvers.get('task-2')?.({ id: 'task-2', status: 'completed', result: {} });
    });

    await waitFor(() => {
      expect(useQuickSettingsFeedbackStore.getState().entries[tunnelDraftKey]).toMatchObject({
        ipsecOperationStatus: 'completed',
      });
    });
    expect(useQuickSettingsFeedbackStore.getState().drafts[tunnelDraftKey]).toEqual({
      '1.TUNNEL_ENABLE': 'true',
    });
    expect(useQuickSettingsFeedbackStore.getState().drafts[controlDraftKey]).toBeUndefined();
  });

  it('clears confirmed tunnel drafts while preserving a newer global edit', async () => {
    const taskResolvers = new Map<string, (task: {
      id: string;
      status: string;
      result: Record<string, unknown>;
    }) => void>();
    mocks.getTask.mockImplementation((taskId: string) => new Promise((resolve) => {
      taskResolvers.set(taskId, resolve);
    }));
    const controlDraftKey = feedbackKey('device-1', 'device-ipsec-control', 1);
    const tunnelDraftKey = feedbackKey('device-1', 'device-ipsec', 1);
    useQuickSettingsFeedbackStore.getState().setDraftField(controlDraftKey, 'IPSEC_ENABLE', '1');
    useQuickSettingsFeedbackStore.getState().setDraftField(tunnelDraftKey, '1.TUNNEL_ENABLE', 'false');
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <MultiInstanceTable
          deviceId="device-1"
          group={ipsecTunnelGroup}
          instanceContext={{ networkType: 'lte', fapInstance: 1 }}
          locale="zh-CN"
          ipsecControlValue="1"
        />
      </QueryClientProvider>,
    );
    fireEvent.click(await screen.findByRole('button', {
      name: /device\.ipsec\.unifiedSubmitWithCount:2/,
    }));

    await waitFor(() => expect(mocks.updateParameters).toHaveBeenCalledTimes(1));
    useQuickSettingsFeedbackStore.getState().setDraftField(controlDraftKey, 'IPSEC_ENABLE', '0');
    await act(async () => {
      taskResolvers.get('task-1')?.({ id: 'task-1', status: 'completed', result: {} });
    });
    await waitFor(() => expect(mocks.updateParameters).toHaveBeenCalledTimes(2));
    await act(async () => {
      taskResolvers.get('task-2')?.({ id: 'task-2', status: 'completed', result: {} });
    });

    await waitFor(() => {
      expect(useQuickSettingsFeedbackStore.getState().entries[tunnelDraftKey]).toMatchObject({
        ipsecOperationStatus: 'completed',
      });
    });
    expect(useQuickSettingsFeedbackStore.getState().drafts[tunnelDraftKey]).toBeUndefined();
    expect(useQuickSettingsFeedbackStore.getState().drafts[controlDraftKey]).toEqual({
      IPSEC_ENABLE: '0',
    });
  });

  it('does not report completion or clear drafts before readback confirmation', async () => {
    mocks.refetchSchema.mockReset();
    mocks.refetchSchema
      .mockResolvedValueOnce({ data: ipsecSchema })
      .mockResolvedValue({
        data: {
          ...ipsecSchema,
          parameters: ipsecSchema.parameters.map((item) => (
            item.path === 'Device.FAP.Ipsec.1.TUNNEL_ENABLE'
              ? { ...item, currentValue: 'false' }
              : item
          )),
        },
      });
    let readbackResolved = false;
    let resolveReadback: ((value: Array<{
      parameterPath: string;
      parameterValue: string;
      parameterType: string;
      writable: boolean;
    }>) => void) | undefined;
    const confirmedReadback = [{
      parameterPath: 'Device.Services.FAPService.Ipsec.IPSEC_ENABLE',
      parameterValue: '1',
      parameterType: 'BOOLEAN',
      writable: true,
    }];
    mocks.searchParameters.mockImplementation(() => {
      if (readbackResolved) return Promise.resolve(confirmedReadback);
      return new Promise((resolve) => {
        resolveReadback = resolve;
      });
    });
    const controlDraftKey = feedbackKey('device-1', 'device-ipsec-control', 1);
    const tunnelDraftKey = feedbackKey('device-1', 'device-ipsec', 1);
    useQuickSettingsFeedbackStore.getState().setDraftField(controlDraftKey, 'IPSEC_ENABLE', '1');
    useQuickSettingsFeedbackStore.getState().setDraftField(tunnelDraftKey, '1.TUNNEL_ENABLE', 'false');
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <MultiInstanceTable
          deviceId="device-1"
          group={ipsecTunnelGroup}
          instanceContext={{ networkType: 'lte', fapInstance: 1 }}
          locale="zh-CN"
          ipsecControlValue="1"
        />
      </QueryClientProvider>,
    );
    fireEvent.click(await screen.findByRole('button', {
      name: /device\.ipsec\.unifiedSubmitWithCount:2/,
    }));

    await waitFor(() => expect(mocks.searchParameters).toHaveBeenCalled());
    expect(useQuickSettingsFeedbackStore.getState().entries[tunnelDraftKey]).toMatchObject({
      ipsecOperationStatus: 'awaiting-readback',
    });
    expect(useQuickSettingsFeedbackStore.getState().drafts[controlDraftKey]).toBeDefined();
    expect(useQuickSettingsFeedbackStore.getState().drafts[tunnelDraftKey]).toBeDefined();

    await act(async () => {
      readbackResolved = true;
      resolveReadback?.(confirmedReadback);
    });
    await waitFor(() => {
      expect(useQuickSettingsFeedbackStore.getState().entries[tunnelDraftKey]).toMatchObject({
        ipsecOperationStatus: 'completed',
        expectedReadback: {
          'Device.Services.FAPService.Ipsec.IPSEC_ENABLE': '1',
          'Device.FAP.Ipsec.1.TUNNEL_ENABLE': 'false',
        },
      });
    });
    expect(mocks.refetchSchema.mock.calls.length).toBeGreaterThanOrEqual(2);
  });

  it('waits for tunnel edits before disabling global IPSec', async () => {
    mocks.globalCurrentValue = '1';
    mocks.readbackGlobalValue = '0';
    const controlDraftKey = feedbackKey('device-1', 'device-ipsec-control', 1);
    const tunnelDraftKey = feedbackKey('device-1', 'device-ipsec', 1);
    useQuickSettingsFeedbackStore.getState().setDraftField(controlDraftKey, 'IPSEC_ENABLE', '0');
    useQuickSettingsFeedbackStore.getState().setDraftField(tunnelDraftKey, '1.TUNNEL_ENABLE', 'false');
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <MultiInstanceTable
          deviceId="device-1"
          group={ipsecTunnelGroup}
          instanceContext={{ networkType: 'lte', fapInstance: 1 }}
          locale="zh-CN"
          ipsecControlValue="0"
        />
      </QueryClientProvider>,
    );

    const submit = await screen.findByRole('button', {
      name: /device\.ipsec\.unifiedSubmitWithCount:2/,
    });
    fireEvent.click(submit);

    await waitFor(() => expect(mocks.updateParameters).toHaveBeenCalledTimes(2));
    expect(mocks.updateParameters.mock.calls[0][0].parameters[0].parameterPath)
      .toBe('Device.FAP.Ipsec.1.TUNNEL_ENABLE');
    expect(mocks.getTask).toHaveBeenCalledWith('task-1');
    expect(mocks.updateParameters.mock.calls[1][0].parameters).toEqual([{
      parameterPath: 'Device.Services.FAPService.Ipsec.IPSEC_ENABLE',
      parameterValue: '0',
      parameterType: 'boolean',
    }]);
  });

  it('does not submit tunnel edits when enabling IPSec fails', async () => {
    mocks.updateParameters.mockReset();
    mocks.updateParameters.mockResolvedValue({ taskId: 'enable-task' });
    mocks.getTask.mockResolvedValue({
      id: 'enable-task',
      status: 'failed',
      errorMessage: 'enable rejected',
    });
    const controlDraftKey = feedbackKey('device-1', 'device-ipsec-control', 1);
    const tunnelDraftKey = feedbackKey('device-1', 'device-ipsec', 1);
    useQuickSettingsFeedbackStore.getState().setDraftField(controlDraftKey, 'IPSEC_ENABLE', '1');
    useQuickSettingsFeedbackStore.getState().setDraftField(tunnelDraftKey, '1.TUNNEL_ENABLE', 'false');
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <MultiInstanceTable
          deviceId="device-1"
          group={ipsecTunnelGroup}
          instanceContext={{ networkType: 'lte', fapInstance: 1 }}
          locale="zh-CN"
          ipsecControlValue="1"
        />
      </QueryClientProvider>,
    );

    fireEvent.click(await screen.findByRole('button', {
      name: /device\.ipsec\.unifiedSubmitWithCount:2/,
    }));

    await waitFor(() => {
      const feedback = useQuickSettingsFeedbackStore.getState().entries[tunnelDraftKey];
      expect(feedback).toMatchObject({
        submitStatus: 'failed_to_queue',
        ipsecTaskIds: ['enable-task'],
        ipsecFailedPhase: 'enable-global',
      });
    });
    expect(mocks.updateParameters).toHaveBeenCalledTimes(1);
    expect(useQuickSettingsFeedbackStore.getState().drafts[controlDraftKey]).toBeDefined();
    expect(useQuickSettingsFeedbackStore.getState().drafts[tunnelDraftKey]).toBeDefined();
  });

  it('does not disable IPSec when tunnel submission fails', async () => {
    mocks.globalCurrentValue = '1';
    mocks.updateParameters.mockReset();
    mocks.updateParameters.mockResolvedValue({ taskId: 'tunnel-task' });
    mocks.getTask.mockResolvedValue({
      id: 'tunnel-task',
      status: 'failed',
      errorMessage: 'tunnel rejected',
    });
    const controlDraftKey = feedbackKey('device-1', 'device-ipsec-control', 1);
    const tunnelDraftKey = feedbackKey('device-1', 'device-ipsec', 1);
    useQuickSettingsFeedbackStore.getState().setDraftField(controlDraftKey, 'IPSEC_ENABLE', '0');
    useQuickSettingsFeedbackStore.getState().setDraftField(tunnelDraftKey, '1.TUNNEL_ENABLE', 'false');
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <MultiInstanceTable
          deviceId="device-1"
          group={ipsecTunnelGroup}
          instanceContext={{ networkType: 'lte', fapInstance: 1 }}
          locale="zh-CN"
          ipsecControlValue="0"
        />
      </QueryClientProvider>,
    );

    fireEvent.click(await screen.findByRole('button', {
      name: /device\.ipsec\.unifiedSubmitWithCount:2/,
    }));

    await waitFor(() => {
      const feedback = useQuickSettingsFeedbackStore.getState().entries[tunnelDraftKey];
      expect(feedback).toMatchObject({
        submitStatus: 'failed_to_queue',
        ipsecTaskIds: ['tunnel-task'],
        ipsecFailedPhase: 'apply-tunnels',
      });
    });
    expect(mocks.updateParameters).toHaveBeenCalledTimes(1);
    expect(mocks.updateParameters.mock.calls[0][0].parameters[0].parameterPath)
      .toBe('Device.FAP.Ipsec.1.TUNNEL_ENABLE');
    expect(useQuickSettingsFeedbackStore.getState().drafts[controlDraftKey]).toBeDefined();
    expect(useQuickSettingsFeedbackStore.getState().drafts[tunnelDraftKey]).toBeDefined();
  });

  it('keeps a pending tunnel add available after its parameter task fails', async () => {
    mocks.globalCurrentValue = '1';
    mocks.readbackGlobalValue = '1';
    mocks.updateParameters.mockReset();
    mocks.updateParameters.mockResolvedValue({ taskId: 'add-params-task' });
    mocks.getTask.mockImplementation(async (taskId: string) => {
      if (taskId === 'add-task') {
        return {
          id: taskId,
          status: 'completed',
          result: { instance_number: 2 },
        };
      }
      if (taskId === 'add-params-task') {
        return {
          id: taskId,
          status: 'failed',
          errorMessage: 'tunnel parameters rejected',
          result: {},
        };
      }
      return { id: taskId, status: 'completed', result: {} };
    });
    const tunnelDraftKey = feedbackKey('device-1', 'device-ipsec', 1);
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <MultiInstanceTable
          deviceId="device-1"
          group={ipsecTunnelGroup}
          instanceContext={{ networkType: 'lte', fapInstance: 1 }}
          locale="zh-CN"
          ipsecControlValue="1"
        />
      </QueryClientProvider>,
    );

    fireEvent.click(screen.getByRole('button', { name: /common\.add/ }));
    fireEvent.click(await screen.findByRole('button', { name: 'device.multi.confirmAdd' }));
    fireEvent.click(await screen.findByRole('button', {
      name: /device\.ipsec\.unifiedSubmitWithCount:1/,
    }));

    await waitFor(() => {
      expect(useQuickSettingsFeedbackStore.getState().entries[tunnelDraftKey]).toMatchObject({
        ipsecOperationStatus: 'failed',
        ipsecFailedPhase: 'apply-tunnels',
        pendingAddRows: [expect.objectContaining({
          values: { TUNNEL_ENABLE: '1' },
        })],
      });
    });
    expect(mocks.deleteObject).toHaveBeenCalledWith({
      deviceId: 'device-1',
      objectPath: 'Device.FAP.Ipsec.2.',
    });
    expect(screen.getByRole('button', {
      name: /device\.ipsec\.unifiedSubmitWithCount:1/,
    })).toBeEnabled();

    fireEvent.click(screen.getByRole('button', { name: 'device.multi.batchClear' }));
    await waitFor(() => {
      expect(useQuickSettingsFeedbackStore.getState().entries[tunnelDraftKey]).toMatchObject({
        pendingAddRows: [],
      });
      expect(screen.queryByRole('button', {
        name: /device\.ipsec\.unifiedSubmitWithCount/,
      })).not.toBeInTheDocument();
    });
  });

  it('restores an unconfirmed delete snapshot after remount', async () => {
    mocks.globalCurrentValue = '1';
    const tunnelDraftKey = feedbackKey('device-1', 'device-ipsec', 1);
    useQuickSettingsFeedbackStore.getState().setFeedback(tunnelDraftKey, {
      kind: 'multi',
      action: 'delete',
      submitStatus: 'failed_to_queue',
      pendingDeleteInstIds: ['1'],
      ipsecOperationStatus: 'failed',
      detail: 'failed delete',
      at: Date.now(),
    });
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <MultiInstanceTable
          deviceId="device-1"
          group={ipsecTunnelGroup}
          instanceContext={{ networkType: 'lte', fapInstance: 1 }}
          locale="zh-CN"
          ipsecControlValue="1"
        />
      </QueryClientProvider>,
    );

    expect(await screen.findByRole('button', {
      name: /device\.ipsec\.unifiedSubmitWithCount:1/,
    })).toBeEnabled();
  });

  it('clears global and tunnel drafts together without an API request', async () => {
    const controlDraftKey = feedbackKey('device-1', 'device-ipsec-control', 1);
    const tunnelDraftKey = feedbackKey('device-1', 'device-ipsec', 1);
    useQuickSettingsFeedbackStore.getState().setDraftField(controlDraftKey, 'IPSEC_ENABLE', '1');
    useQuickSettingsFeedbackStore.getState().setDraftField(tunnelDraftKey, '1.TUNNEL_ENABLE', 'false');
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <MultiInstanceTable
          deviceId="device-1"
          group={ipsecTunnelGroup}
          instanceContext={{ networkType: 'lte', fapInstance: 1 }}
          locale="zh-CN"
          ipsecControlValue="1"
        />
      </QueryClientProvider>,
    );

    fireEvent.click(await screen.findByRole('button', {
      name: 'device.multi.batchClear',
    }));

    await waitFor(() => {
      expect(useQuickSettingsFeedbackStore.getState().drafts[controlDraftKey]).toBeUndefined();
      expect(useQuickSettingsFeedbackStore.getState().drafts[tunnelDraftKey]).toBeUndefined();
    });
    expect(mocks.updateParameters).not.toHaveBeenCalled();
  });

  it('clears the global draft after the unified task completes and readback refreshes', async () => {
    const controlDraftKey = feedbackKey('device-1', 'device-ipsec-control', 1);
    const tunnelDraftKey = feedbackKey('device-1', 'device-ipsec', 1);
    useQuickSettingsFeedbackStore.getState().setDraftField(controlDraftKey, 'IPSEC_ENABLE', '1');
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <MultiInstanceTable
          deviceId="device-1"
          group={ipsecTunnelGroup}
          instanceContext={{ networkType: 'lte', fapInstance: 1 }}
          locale="zh-CN"
          ipsecControlValue="1"
        />
      </QueryClientProvider>,
    );

    fireEvent.click(await screen.findByRole('button', {
      name: /device\.ipsec\.unifiedSubmitWithCount:1/,
    }));

    await waitFor(() => {
      expect(useQuickSettingsFeedbackStore.getState().drafts[controlDraftKey]).toBeUndefined();
      expect(useQuickSettingsFeedbackStore.getState().entries[tunnelDraftKey]).toMatchObject({
        taskId: 'task-1',
        syncedForTaskId: 'task-1',
        ipsecTargetEnabled: true,
      });
    });
    expect(mocks.syncDeviceParams).toHaveBeenCalledWith(expect.objectContaining({
      parameterPaths: expect.arrayContaining([
        'Device.Services.FAPService.Ipsec.IPSEC_ENABLE',
      ]),
    }));
    expect(mocks.searchParameters).toHaveBeenCalledWith(
      'device-1',
      'IPSEC_ENABLE',
      20,
    );
    expect(queryClient.getQueryData([
      'devices',
      'parameters',
      'search',
      'device-1',
      'IPSEC_ENABLE',
      20,
    ])).toEqual(expect.arrayContaining([
      expect.objectContaining({
        parameterPath: 'Device.Services.FAPService.Ipsec.IPSEC_ENABLE',
        parameterValue: '1',
      }),
    ]));
  });

  it('keeps the global draft when readback does not match the submitted target', async () => {
    mocks.readbackGlobalValue = '0';
    const controlDraftKey = feedbackKey('device-1', 'device-ipsec-control', 1);
    const tunnelDraftKey = feedbackKey('device-1', 'device-ipsec', 1);
    useQuickSettingsFeedbackStore.getState().setDraftField(controlDraftKey, 'IPSEC_ENABLE', '1');
    useQuickSettingsFeedbackStore.getState().setDraftField(tunnelDraftKey, '1.TUNNEL_ENABLE', 'false');
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <MultiInstanceTable
          deviceId="device-1"
          group={ipsecTunnelGroup}
          instanceContext={{ networkType: 'lte', fapInstance: 1 }}
          locale="zh-CN"
          ipsecControlValue="1"
        />
      </QueryClientProvider>,
    );

    fireEvent.click(await screen.findByRole('button', {
      name: /device\.ipsec\.unifiedSubmitWithCount:2/,
    }));

    await waitFor(() => expect(mocks.searchParameters).toHaveBeenCalled());
    expect(useQuickSettingsFeedbackStore.getState().drafts[controlDraftKey]).toBeDefined();
    expect(useQuickSettingsFeedbackStore.getState().drafts[tunnelDraftKey]).toBeDefined();
    expect(useQuickSettingsFeedbackStore.getState().entries[tunnelDraftKey]).not.toMatchObject({
      syncedForTaskId: 'task-1',
    });
  });
});
