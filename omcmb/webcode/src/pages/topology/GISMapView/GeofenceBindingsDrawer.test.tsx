import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { IntlProvider } from 'react-intl';
import { getMessages } from '@core/i18n/index';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type {
  GeofenceManualBindingPreview,
  GeofenceMapDefinition,
} from '@core/types/geofence';

const mocks = vi.hoisted(() => ({
  bindingsQuery: vi.fn(),
  definitionsQuery: vi.fn(),
  actionsQuery: vi.fn(),
  preview: vi.fn(),
  createJob: vi.fn(),
  jobQuery: vi.fn(),
  itemsQuery: vi.fn(),
  exportBindings: vi.fn(),
  suspend: vi.fn(),
  resume: vi.fn(),
  remove: vi.fn(),
}));

vi.mock('@core/hooks/api/useGeofence', () => ({
  useGeofenceBindings: (...args: unknown[]) => mocks.bindingsQuery(...args),
  useGeofenceDefinitions: (...args: unknown[]) => mocks.definitionsQuery(...args),
  useGeofenceControlActions: (...args: unknown[]) => mocks.actionsQuery(...args),
  usePreviewGeofenceManualBindings: () => ({
    mutateAsync: mocks.preview,
    isPending: false,
  }),
  useCreateGeofenceManualBindJob: () => ({
    mutateAsync: mocks.createJob,
    isPending: false,
  }),
  useGeofenceManualBindJob: (...args: unknown[]) => mocks.jobQuery(...args),
  useGeofenceManualBindItems: (...args: unknown[]) => mocks.itemsQuery(...args),
  useExportGeofenceBindings: () => ({
    mutateAsync: mocks.exportBindings,
    isPending: false,
  }),
  useSuspendGeofenceBinding: () => ({
    mutateAsync: mocks.suspend,
    isPending: false,
  }),
  useResumeGeofenceBinding: () => ({
    mutateAsync: mocks.resume,
    isPending: false,
  }),
  useRemoveGeofenceBinding: () => ({
    mutateAsync: mocks.remove,
    isPending: false,
  }),
}));

import GeofenceBindingsDrawer from './GeofenceBindingsDrawer';

const fence: GeofenceMapDefinition = {
  definition: {
    id: 'fence-1',
    name: '园区围栏',
    carrier: 'cmcc',
    ruleType: 'polygon_allow_zone',
    status: 'enabled',
    currentVersionId: 'version-1',
    createdBy: 'operator',
    updatedBy: 'operator',
    createdAt: '2026-07-31T08:00:00Z',
    updatedAt: '2026-07-31T08:00:00Z',
  },
  currentVersion: null,
};

const sourceFence: GeofenceMapDefinition = {
  ...fence,
  definition: {
    ...fence.definition,
    id: 'old-fence',
    name: '原活动围栏',
  },
};

const preview: GeofenceManualBindingPreview = {
  geofenceId: 'fence-1',
  geofenceVersionId: 'version-1',
  ruleType: 'polygon_allow_zone',
  inputCount: 2,
  eligibleCount: 1,
  moveCount: 0,
  skippedCount: 1,
  items: [
    {
      inputKey: 'sn:SN001',
      input: 'SN001',
      deviceId: 'device-1',
      deviceSN: 'SN001',
      decision: 'eligible',
    },
    {
      inputKey: 'sn:SN002',
      input: 'SN002',
      deviceSN: 'SN002',
      decision: 'skipped',
      reasonCode: 'already_bound',
    },
  ],
  previewFingerprint: 'binding-preview',
};

function renderDrawer() {
  return {
    user: userEvent.setup(),
    ...render(
      <IntlProvider
        locale="zh-CN"
        defaultLocale="zh-CN"
        messages={getMessages('zh-CN')}
      >
        <GeofenceBindingsDrawer open item={fence} onClose={vi.fn()} />
      </IntlProvider>,
    ),
  };
}

describe('GeofenceBindingsDrawer', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.definitionsQuery.mockReturnValue({
      data: [fence.definition, sourceFence.definition],
    });
    mocks.bindingsQuery.mockReturnValue({
      data: {
        items: [
          {
            id: 'binding-1',
            deviceId: 'device-bound',
            geofenceId: 'fence-1',
            ruleType: 'polygon_allow_zone',
            status: 'active',
            bindSource: 'manual',
            boundBy: 'operator',
            boundAt: '2026-07-31T08:00:00Z',
            deviceSN: 'BOUND001',
            deviceName: '已绑定设备',
            deviceCarrier: 'cmcc',
            hasLocation: true,
            evaluation: {
              confirmedState: 'inside',
              candidateState: 'exit',
              candidateCount: 2,
              lastObservationVersion: 12,
              lastObservedAt: '2026-07-31T08:05:00Z',
              lastDistanceToBoundary: 35.5,
              evaluationHealth: 'healthy',
            },
          },
        ],
        total: 1,
        page: 1,
        pageSize: 50,
      },
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
    });
    mocks.actionsQuery.mockReturnValue({
      data: [],
      isLoading: false,
      isError: false,
    });
    mocks.preview.mockResolvedValue(preview);
    mocks.createJob.mockResolvedValue({ jobId: 'job-1' });
    mocks.exportBindings.mockResolvedValue(undefined);
    mocks.suspend.mockResolvedValue({});
    mocks.resume.mockResolvedValue({});
    mocks.remove.mockResolvedValue({});
    mocks.jobQuery.mockImplementation((id?: string) => ({
      data: id
        ? {
            id,
            jobType: 'manual_bind',
            status: 'succeeded',
            geofenceId: 'fence-1',
            requestedBy: 'operator',
            reason: '新站入网',
            attempt: 1,
            maxAttempts: 3,
            createdAt: '2026-07-31T08:00:00Z',
            progress: {
              total: 2,
              pending: 0,
              succeeded: 1,
              skipped: 1,
              failed: 0,
            },
          }
        : undefined,
    }));
    mocks.itemsQuery.mockImplementation((id?: string) => ({
      data: id
        ? {
            items: [],
            total: 0,
            page: 1,
            pageSize: 50,
          }
        : undefined,
    }));
  });

  it('normalizes SN separators, previews eligibility, and creates the existing batch job', async () => {
    const { user } = renderDrawer();

    expect(screen.getAllByText('已绑定设备')).toHaveLength(2);
    await user.type(
      screen.getByLabelText('输入设备 SN，支持空格、逗号或分号分隔'),
      'SN001; SN002,\nSN001',
    );
    await user.click(screen.getByRole('button', { name: '绑定设备预览' }));

    await waitFor(() =>
      expect(mocks.preview).toHaveBeenCalledWith({
        id: 'fence-1',
        inputs: { deviceSNs: ['SN001', 'SN002'] },
      }),
    );
    expect(screen.getByText(/可绑定.*1/)).toBeInTheDocument();
    expect(screen.getByText(/已跳过.*1/)).toBeInTheDocument();
    expect(screen.getByText('直接绑定')).toBeInTheDocument();
    expect(screen.getByText('设备已绑定到当前围栏')).toBeInTheDocument();
    expect(screen.getByText('请输入操作原因')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: '创建绑定任务' }),
    ).toBeDisabled();

    await user.type(screen.getByLabelText('操作原因'), '新站入网');
    await user.click(screen.getByRole('button', { name: '创建绑定任务' }));

    await waitFor(() =>
      expect(mocks.createJob).toHaveBeenCalledWith({
        id: 'fence-1',
        input: {
          deviceSNs: ['SN001', 'SN002'],
          previewFingerprint: 'binding-preview',
          reason: '新站入网',
        },
      }),
    );
    expect(mocks.jobQuery).toHaveBeenLastCalledWith(
      'job-1',
      'fence-1',
    );
    expect(await screen.findByText('已完成 2/2')).toBeInTheDocument();
  });

  it('allows an explicitly previewed move from another active fence', async () => {
    mocks.preview.mockResolvedValue({
      ...preview,
      eligibleCount: 0,
      moveCount: 1,
      skippedCount: 1,
      items: [
        {
          ...preview.items[0],
          decision: 'move',
          reasonCode: 'reassigned',
          sourceBindingId: 'old-binding',
          sourceGeofenceId: 'old-fence',
        },
        preview.items[1],
      ],
    });
    const { user } = renderDrawer();

    await user.type(
      screen.getByLabelText('输入设备 SN，支持空格、逗号或分号分隔'),
      'SN001 SN002',
    );
    await user.click(screen.getByRole('button', { name: '绑定设备预览' }));

    expect(
      await screen.findByText(/将从其他围栏移入.*1/),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        '移入会替换设备当前生效的多边形围栏；后续删除或归档当前围栏不会自动恢复原归属。',
      ),
    ).toBeInTheDocument();
    expect(screen.getByText('将从“原活动围栏”移入')).toBeInTheDocument();
    await user.type(screen.getByLabelText('操作原因'), '调整围栏归属');
    expect(
      screen.getByRole('button', { name: '创建绑定任务' }),
    ).toBeEnabled();
  });

  it('separates removed binding history from current assignments', async () => {
    const baseBinding = mocks.bindingsQuery().data.items[0];
    mocks.bindingsQuery.mockReturnValue({
      ...mocks.bindingsQuery(),
      data: {
        ...mocks.bindingsQuery().data,
        items: [
          baseBinding,
          {
            ...baseBinding,
            id: 'binding-removed',
            deviceSN: 'REMOVED001',
            deviceName: '历史设备',
            status: 'removed',
          },
        ],
        total: 2,
      },
    });
    const { user } = renderDrawer();

    expect(screen.getAllByText('已绑定设备')).toHaveLength(2);
    expect(screen.queryByText('历史设备')).not.toBeInTheDocument();

    await user.click(screen.getByText('历史记录 1'));

    expect(screen.getByText('历史设备')).toBeInTheDocument();
    expect(screen.getAllByText('已绑定设备')).toHaveLength(1);
  });

  it('shows durable device readback instead of treating task completion as success', () => {
    mocks.actionsQuery.mockReturnValue({
      data: [{
        id: 'action-1',
        deviceSN: 'BOUND001',
        commandKey: 'geofence:device:8:deactivate',
        actionType: 'deactivate',
        status: 'partial_failed',
        beforeState: [{ path: 'rf', value: '1' }],
        requestedState: [{ path: 'rf', value: '0' }],
        verifiedState: [{ path: 'rf', value: '1' }],
        lastError: 'rf expected 0 got 1',
        createdAt: '2026-08-05T08:00:00Z',
        completedAt: '2026-08-05T08:00:05Z',
      }],
      isLoading: false,
      isError: false,
    });

    renderDrawer();

    expect(screen.getByText('部分失败')).toBeInTheDocument();
    expect(screen.getByText('控制参数')).toBeInTheDocument();
    expect(screen.getAllByText('rf')).toHaveLength(2);
    expect(screen.getByText('不一致')).toBeInTheDocument();
    expect(screen.getAllByText('0').length).toBeGreaterThan(0);
    expect(screen.getAllByText('1').length).toBeGreaterThan(0);
    expect(screen.getByText('rf expected 0 got 1')).toBeInTheDocument();
  });

  it('shows the read-only Observe evaluation state without exposing coordinates', () => {
    renderDrawer();

    expect(screen.getByText('已在围栏内')).toBeInTheDocument();
    expect(screen.getByText('正在确认越界（2 次）')).toBeInTheDocument();
    expect(screen.getByText('距边界：35.5 m')).toBeInTheDocument();
    expect(
      screen.getByText('最近判定：2026/7/31 16:05:00'),
    ).toBeInTheDocument();
    expect(screen.queryByText(/longitude|latitude/i)).not.toBeInTheDocument();
  });

  it('exports all visible bindings from the server endpoint', async () => {
    const { user } = renderDrawer();

    await user.click(
      screen.getByRole('button', { name: '导出绑定设备' }),
    );

    await waitFor(() =>
      expect(mocks.exportBindings).toHaveBeenCalledWith({
        id: 'fence-1',
        filter: {},
      }),
    );
  });

  it('distinguishes a failed first evaluation from a confirmed location state', () => {
    mocks.bindingsQuery.mockReturnValue({
      data: {
        items: [
          {
            id: 'binding-failed',
            deviceId: 'device-failed',
            geofenceId: 'fence-1',
            ruleType: 'polygon_allow_zone',
            status: 'active',
            bindSource: 'manual',
            boundBy: 'operator',
            boundAt: '2026-07-31T08:00:00Z',
            deviceSN: 'FAILED001',
            deviceName: '首次判定失败设备',
            deviceCarrier: 'cmcc',
            hasLocation: true,
            evaluation: {
              confirmedState: 'unknown',
              candidateCount: 0,
              evaluationHealth: 'failed',
              errorCode: 'invalid_geometry',
            },
          },
        ],
        total: 1,
        page: 1,
        pageSize: 50,
      },
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
    });

    renderDrawer();

    expect(screen.getByText('尚无成功判定')).toBeInTheDocument();
    expect(screen.getByText('判定失败')).toBeInTheDocument();
    expect(
      screen.getByText('错误码：invalid_geometry'),
    ).toBeInTheDocument();
  });

  it('does not create a job when the server preview has no eligible devices', async () => {
    mocks.preview.mockResolvedValue({
      ...preview,
      eligibleCount: 0,
      skippedCount: 2,
      items: preview.items.map((item) => ({
        ...item,
        decision: 'skipped',
      })),
    });
    const { user } = renderDrawer();

    await user.type(
      screen.getByLabelText('输入设备 SN，支持空格、逗号或分号分隔'),
      'SN001 SN002',
    );
    await user.click(screen.getByRole('button', { name: '绑定设备预览' }));

    expect(
      await screen.findByText('没有可绑定或可移入的设备'),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: '创建绑定任务' }),
    ).toBeDisabled();
    expect(mocks.createJob).not.toHaveBeenCalled();
  });

  it('requires a reason and warns that removing a binding does not recover the device', async () => {
    const { user } = renderDrawer();

    await user.click(
      screen.getByRole('button', { name: '移除绑定' }),
    );
    expect(
      screen.getByText(
        '移除绑定只停止该规则后续检查，不会自动恢复或重新激活设备',
      ),
    ).toBeInTheDocument();
    const confirm = screen.getByRole('button', { name: /确\s*认/ });
    expect(confirm).toBeDisabled();

    await user.type(screen.getByLabelText('操作原因'), '设备迁出');
    await user.click(confirm);

    await waitFor(() =>
      expect(mocks.remove).toHaveBeenCalledWith({
        id: 'binding-1',
        reason: '设备迁出',
      }),
    );
  });

  it('requires and submits a reason when resuming a suspended binding', async () => {
    const current = mocks.bindingsQuery();
    mocks.bindingsQuery.mockReturnValue({
      ...current,
      data: {
        ...current.data,
        items: current.data.items.map((binding: { status: string }) => ({
          ...binding,
          status: 'suspended',
        })),
      },
    });
    const { user } = renderDrawer();

    await user.click(
      screen.getByRole('button', { name: '恢复绑定' }),
    );
    const confirm = screen.getByRole('button', { name: /确\s*认/ });
    expect(confirm).toBeDisabled();

    await user.type(screen.getByLabelText('操作原因'), '维护完成');
    await user.click(confirm);

    await waitFor(() =>
      expect(mocks.resume).toHaveBeenCalledWith({
        id: 'binding-1',
        reason: '维护完成',
      }),
    );
  });
});
