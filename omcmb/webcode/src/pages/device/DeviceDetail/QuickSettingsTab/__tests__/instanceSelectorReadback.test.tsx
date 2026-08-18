import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { QuickSettingsGroup } from '@core/types/quicksettings';
import {
  feedbackKey,
  useQuickSettingsFeedbackStore,
} from '@core/store/quickSettingsFeedbackStore';
import InstanceSelectorForm from '../InstanceSelectorForm';

const objectPath = 'Device.Test.';
const valuePath = `${objectPath}1.Name`;

const mocks = vi.hoisted(() => ({
  refetchSchema: vi.fn(),
  invalidateParameterSchemaCache: vi.fn(),
}));

function schema(currentValue: string) {
  return {
    parameters: [{
      path: valuePath,
      type: 'string',
      writable: true,
      currentValue,
    }],
    objects: [{
      path: objectPath,
      currentInstances: [1],
      canAdd: true,
      canDeleteAny: true,
    }],
    total: 1,
  };
}

vi.mock('@core/hooks/api/useDeviceParameters', () => ({
  useParameterSchema: () => ({
    data: schema('old'),
    isLoading: false,
    isFetching: false,
    refetch: mocks.refetchSchema,
  }),
  useUpdateParameters: () => ({ isPending: false, mutateAsync: vi.fn() }),
  useAddObject: () => ({ isPending: false, mutateAsync: vi.fn() }),
  useDeleteObject: () => ({ isPending: false, mutateAsync: vi.fn() }),
}));

vi.mock('@core/hooks/api/useDeviceTask', () => ({
  useDeviceTaskStatus: (taskId?: string) => ({
    data: taskId ? { id: taskId, status: 'completed', result: {} } : undefined,
  }),
}));

vi.mock('@core/services/api/deviceParameterApi', () => ({
  deviceParameterApi: {
    invalidateParameterSchemaCache: mocks.invalidateParameterSchemaCache,
  },
}));

vi.mock('@core/services/api/deviceTaskApi', () => ({
  deviceTaskApi: { getTask: vi.fn() },
}));

const selectorGroup: QuickSettingsGroup = {
  id: 'test-selector',
  titleZh: '测试实例',
  titleEn: 'Test Instances',
  multiInstance: true,
  style: 'table',
  objectPath: 'Device.Test.{i}.',
  params: [{
    name: 'Name',
    leaf: 'Name',
    titleZh: '名称',
    titleEn: 'Name',
    type: 'string',
  }],
};

describe('InstanceSelectorForm terminal readback', () => {
  beforeEach(() => {
    mocks.refetchSchema.mockReset();
    mocks.refetchSchema
      .mockResolvedValueOnce({ data: schema('old') })
      .mockResolvedValue({ data: schema('new') });
    mocks.invalidateParameterSchemaCache.mockReset();
    useQuickSettingsFeedbackStore.setState({
      entries: {},
      drafts: {},
      draftRevisions: {},
    });
  });

  it('keeps polling until the submitted instance value is visible', async () => {
    const fbKey = feedbackKey('device-1', selectorGroup.id, 1);
    useQuickSettingsFeedbackStore.getState().setFeedback(fbKey, {
      kind: 'multi',
      action: 'save',
      submitStatus: 'queued',
      taskId: 'task-1',
      savedInstId: '1',
      expectedReadback: { [valuePath]: 'new' },
      detail: '实例 1',
      at: Date.now(),
    });

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    render(
      <QueryClientProvider client={queryClient}>
        <InstanceSelectorForm
          deviceId="device-1"
          selectorGroup={selectorGroup}
          childGroups={[]}
          instanceContext={{ networkType: 'gsm', fapInstance: 1 }}
          locale="zh-CN"
        />
      </QueryClientProvider>,
    );

    await waitFor(() => expect(mocks.refetchSchema.mock.calls.length).toBeGreaterThanOrEqual(2), {
      timeout: 2_000,
    });
    await waitFor(() => {
      expect(useQuickSettingsFeedbackStore.getState().entries[fbKey]).toMatchObject({
        syncedForTaskId: 'task-1',
      });
    });
  });

  it('keeps polling a completed delete until the instance disappears', async () => {
    mocks.refetchSchema.mockReset();
    mocks.refetchSchema
      .mockResolvedValueOnce({ data: schema('old') })
      .mockResolvedValue({
        data: {
          ...schema('old'),
          objects: [{ ...schema('old').objects[0], currentInstances: [] }],
        },
      });
    const fbKey = feedbackKey('device-1', selectorGroup.id, 1);
    useQuickSettingsFeedbackStore.getState().setFeedback(fbKey, {
      kind: 'multi',
      action: 'delete',
      submitStatus: 'queued',
      taskId: 'task-2',
      deletedInstIds: ['1'],
      readbackObjectPath: objectPath,
      detail: '实例 1',
      at: Date.now(),
    });

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    render(
      <QueryClientProvider client={queryClient}>
        <InstanceSelectorForm
          deviceId="device-1"
          selectorGroup={selectorGroup}
          childGroups={[]}
          instanceContext={{ networkType: 'gsm', fapInstance: 1 }}
          locale="zh-CN"
        />
      </QueryClientProvider>,
    );

    await waitFor(() => expect(mocks.refetchSchema.mock.calls.length).toBeGreaterThanOrEqual(2), {
      timeout: 2_000,
    });
    await waitFor(() => {
      expect(useQuickSettingsFeedbackStore.getState().entries[fbKey]).toMatchObject({
        syncedForTaskId: 'task-2',
      });
    });
  });
});
