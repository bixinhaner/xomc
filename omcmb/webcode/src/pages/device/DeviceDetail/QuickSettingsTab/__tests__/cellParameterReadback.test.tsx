import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { QuickSettingsGroup } from '@core/types/quicksettings';
import {
  feedbackKey,
  useQuickSettingsFeedbackStore,
} from '@core/store/quickSettingsFeedbackStore';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import CellParameterForm from '../CellParameterForm';

const DLBW_PATH = 'Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.DLBandwidth';

const mocks = vi.hoisted(() => ({
  refetchSchema: vi.fn(),
  invalidateParameterSchemaCache: vi.fn(),
  updateParameters: vi.fn(),
  schemaData: undefined as ReturnType<typeof schema> | undefined,
}));

function schema(currentValue: string) {
  return {
    parameters: [{
      path: DLBW_PATH,
      type: 'unsignedInt',
      writable: true,
      currentValue,
      constraints: { enumValues: ['25', '50', '75', '100'] },
    }],
    objects: [],
    total: 1,
  };
}

vi.mock('@core/hooks/api/useDeviceParameters', () => ({
  useParameterSchema: () => ({
    data: mocks.schemaData ?? schema('25'),
    isLoading: false,
    refetch: mocks.refetchSchema,
  }),
  useSearchParameters: () => ({ data: [] }),
  useUpdateParameters: () => ({ isPending: false, mutateAsync: mocks.updateParameters }),
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

vi.mock('@/hooks/useT', () => ({
  useT: () => (id: string) => id,
}));

const cellGroup: QuickSettingsGroup = {
  id: 'enb-cell',
  titleZh: '小区参数',
  titleEn: 'Cell Parameters',
  multiInstance: false,
  params: [{
    name: 'DLBandwidth',
    titleZh: '下行带宽',
    titleEn: 'Downlink Bandwidth',
    standardPath: DLBW_PATH,
    type: 'enum',
    enumOptions: [
      { value: '25', label: '5M' },
      { value: '50', label: '10M' },
      { value: '75', label: '15M' },
      { value: '100', label: '20M' },
    ],
  }],
};

describe('cell parameter terminal readback', () => {
  beforeEach(() => {
    mocks.refetchSchema.mockReset();
    mocks.invalidateParameterSchemaCache.mockReset();
    mocks.updateParameters.mockReset();
    mocks.updateParameters.mockResolvedValue({ taskId: 'task-2' });
    mocks.schemaData = undefined;
    mocks.refetchSchema
      .mockResolvedValueOnce({ data: schema('25') })
      .mockResolvedValue({ data: schema('50') });
    useQuickSettingsFeedbackStore.setState({
      entries: {},
      drafts: {},
      draftRevisions: {},
    });
  });

  it.each(['enb-cell', 'gnb-cell'])(
    'keeps polling %s when the first post-save schema still contains the old value',
    async (groupId) => {
      const fbKey = feedbackKey('device-1', groupId, 1);
      useQuickSettingsFeedbackStore.getState().setDraftField(fbKey, 'DLBandwidth', '50');
      const submittedDraftRevision = useQuickSettingsFeedbackStore.getState().draftRevisions[fbKey];
      useQuickSettingsFeedbackStore.getState().setFeedback(fbKey, {
        kind: 'cell',
        submitStatus: 'queued',
        taskId: 'task-1',
        count: 1,
        at: Date.now(),
        expectedReadback: { [DLBW_PATH]: '50' },
        submittedDraftRevision,
      });

      const queryClient = new QueryClient({
        defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
      });
      render(
        <QueryClientProvider client={queryClient}>
          <CellParameterForm
            deviceId="device-1"
            group={{ ...cellGroup, id: groupId }}
            instanceContext={{ networkType: 'lte', fapInstance: 1 }}
            locale="zh-CN"
          />
        </QueryClientProvider>,
      );

      await waitFor(() => expect(mocks.refetchSchema).toHaveBeenCalledTimes(2), { timeout: 2_000 });
      await waitFor(() => {
        expect(useQuickSettingsFeedbackStore.getState().entries[fbKey]).toMatchObject({
          syncedForTaskId: 'task-1',
        });
        expect(useQuickSettingsFeedbackStore.getState().drafts[fbKey]).toBeUndefined();
      });
    },
  );

  it('compares a quick revert with the pending submitted value instead of the stale schema value', async () => {
    const pciPath = 'Device.Services.FAPService.1.CellConfig.1.NR.RAN.RF.PhyCellID';
    mocks.schemaData = {
      parameters: [{
        path: pciPath,
        type: 'unsignedInt',
        writable: true,
        currentValue: '23',
        constraints: { minValue: 0, maxValue: 1007 },
      }],
      objects: [],
      total: 1,
    };
    const fbKey = feedbackKey('device-1', 'gnb-cell', 1, 1);
    useQuickSettingsFeedbackStore.getState().setDraftField(fbKey, 'PCI', '22');
    const submittedDraftRevision = useQuickSettingsFeedbackStore.getState().draftRevisions[fbKey];
    useQuickSettingsFeedbackStore.getState().setFeedback(fbKey, {
      kind: 'cell',
      submitStatus: 'queued',
      taskId: 'task-1',
      count: 1,
      at: Date.now(),
      expectedReadback: { [pciPath]: '22' },
      submittedDraftRevision,
    });
    // The user changes back to the old schema value before task-1 has been read back.
    useQuickSettingsFeedbackStore.getState().setDraftField(fbKey, 'PCI', '23');

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    render(
      <QueryClientProvider client={queryClient}>
        <CellParameterForm
          deviceId="device-1"
          group={{
            id: 'gnb-cell',
            titleZh: '小区参数',
            titleEn: 'Cell Parameters',
            multiInstance: false,
            params: [{
              name: 'PCI',
              titleZh: 'PCI',
              titleEn: 'PCI',
              standardPath: pciPath,
            }],
          }}
          instanceContext={{ networkType: 'nr', fapInstance: 1, cellInstance: 1 }}
          locale="zh-CN"
        />
      </QueryClientProvider>,
    );

    await waitFor(() => expect(screen.getByRole('textbox')).toHaveValue('23'));
    fireEvent.click(screen.getByRole('button', { name: 'common.save' }));

    await waitFor(() => expect(mocks.updateParameters).toHaveBeenCalledWith({
      deviceId: 'device-1',
      parameters: [{
        parameterPath: pciPath,
        parameterValue: '23',
        parameterType: 'unsignedInt',
      }],
    }));
  });
});
