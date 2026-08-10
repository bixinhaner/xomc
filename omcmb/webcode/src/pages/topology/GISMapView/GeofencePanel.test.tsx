import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { IntlProvider } from 'react-intl';
import { getMessages } from '@core/i18n/index';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type {
  GeofenceLifecycleImpact,
  GeofenceMapDefinition,
} from '@core/types/geofence';

const mocks = vi.hoisted(() => ({
  mapQuery: vi.fn(),
  settingsQuery: vi.fn(),
  previewSettings: vi.fn(),
  updateSettings: vi.fn(),
  preview: vi.fn(),
  previewCandidates: vi.fn(),
  transition: vi.fn(),
}));

vi.mock('@core/hooks/api/useGeofence', () => ({
  useGeofenceMap: (...args: unknown[]) => mocks.mapQuery(...args),
  useGeofenceSettings: () => mocks.settingsQuery(),
  usePreviewGeofenceSettings: () => ({
    mutateAsync: mocks.previewSettings,
    isPending: false,
  }),
  useUpdateGeofenceSettings: () => ({
    mutateAsync: mocks.updateSettings,
    isPending: false,
  }),
  usePreviewGeofenceLifecycle: () => ({
    mutateAsync: mocks.preview,
    isPending: false,
  }),
  usePreviewGeofenceCandidates: () => ({
    mutateAsync: mocks.previewCandidates,
    isPending: false,
  }),
  useTransitionGeofenceLifecycle: () => ({
    mutateAsync: mocks.transition,
    isPending: false,
  }),
}));

import GeofencePanel from './GeofencePanel';

const enabledFence: GeofenceMapDefinition = {
  definition: {
    id: 'fence-enabled',
    name: '园区围栏',
    carrier: 'cmcc',
    ruleType: 'polygon_allow_zone',
    status: 'enabled',
    currentVersionId: 'version-enabled',
    createdBy: 'operator',
    updatedBy: 'operator',
    createdAt: '2026-07-31T08:00:00Z',
    updatedAt: '2026-07-31T08:00:00Z',
  },
  currentVersion: {
    id: 'version-enabled',
    geofenceId: 'fence-enabled',
    version: 1,
    status: 'published',
    geometry: {
      type: 'Polygon',
      coordinates: [[[120, 30], [121, 30], [121, 31], [120, 30]]],
    },
    boundingBox: {
      minLongitude: 120,
      minLatitude: 30,
      maxLongitude: 121,
      maxLatitude: 31,
    },
    policy: { exitAction: 'notify_only' },
    createdBy: 'operator',
    publishedBy: 'operator',
    createdAt: '2026-07-31T08:00:00Z',
    publishedAt: '2026-07-31T08:00:00Z',
  },
};

const disabledFence: GeofenceMapDefinition = {
  definition: {
    ...enabledFence.definition,
    id: 'fence-disabled',
    name: '停用围栏',
    status: 'disabled',
    currentVersionId: 'version-disabled',
  },
  currentVersion: {
    ...enabledFence.currentVersion!,
    id: 'version-disabled',
    geofenceId: 'fence-disabled',
  },
};

const archivedFence: GeofenceMapDefinition = {
  definition: {
    ...enabledFence.definition,
    id: 'fence-archived',
    name: '归档围栏',
    status: 'archived',
  },
  currentVersion: enabledFence.currentVersion,
};

function impact(
  targetStatus: 'enabled' | 'disabled' | 'archived',
): GeofenceLifecycleImpact {
  return {
    geofenceId:
      targetStatus === 'disabled' ? 'fence-enabled' : 'fence-disabled',
    currentVersionId: 'version-enabled',
    currentStatus:
      targetStatus === 'disabled' ? 'enabled' : 'disabled',
    targetStatus,
    bindingCount: 3,
    deviceCount: 2,
    activeBatchJobCount: 0,
    previewFingerprint: `preview-${targetStatus}`,
  };
}

function renderPanel(onLocate = vi.fn()) {
  const onCreate = vi.fn();
  const onEdit = vi.fn();
  const onBind = vi.fn();
  return {
    onLocate,
    onCreate,
    onEdit,
    onBind,
    user: userEvent.setup(),
    ...render(
      <IntlProvider
        locale="zh-CN"
        defaultLocale="zh-CN"
        messages={getMessages('zh-CN')}
      >
        <GeofencePanel
          open
          canManage
          onLocate={onLocate}
          onCreate={onCreate}
          onEdit={onEdit}
          onBind={onBind}
        />
      </IntlProvider>,
    ),
  };
}

describe('GeofencePanel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.mapQuery.mockReturnValue({
      data: {
        items: [enabledFence, disabledFence, archivedFence],
      },
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
    });
    mocks.preview.mockImplementation(
      ({ target }: { target: 'enabled' | 'disabled' | 'archived' }) =>
        Promise.resolve(impact(target)),
    );
    mocks.transition.mockResolvedValue({ status: 'executed' });
    mocks.settingsQuery.mockReturnValue({
      data: {
        systemMode: 'observe',
        carriers: [
          {
            carrier: 'cmcc',
            mode: 'observe',
            effectiveMode: 'observe',
            defaultBaselineRadiusMeters: 1000,
            updatedAt: '2026-07-31T08:00:00Z',
          },
        ],
      },
      isLoading: false,
      isError: false,
    });
    mocks.previewSettings.mockResolvedValue({
      current: { systemMode: 'observe', carriers: [] },
      proposed: { systemMode: 'off', carriers: [] },
      enabledGeofences: 1,
      activeBindings: 2,
      newlyObservedDevices: 0,
    });
    mocks.updateSettings.mockResolvedValue({
      systemMode: 'off',
      carriers: [],
    });
    mocks.previewCandidates.mockResolvedValue({
      geofenceId: 'fence-enabled',
      versionId: 'version-enabled',
      inside: [],
      outside: [],
      noLocation: [],
    });
  });

  it('searches by name and only locates an enabled fence', async () => {
    const { user, onLocate, onCreate, onEdit, onBind } = renderPanel();

    expect(
      screen.getByRole('switch', { name: '运营商围栏开关' }),
    ).toHaveClass('geofence-master-toggle');
    expect(screen.getByText('园区围栏')).toBeInTheDocument();
    expect(screen.getByText('停用围栏')).toBeInTheDocument();
    expect(screen.getAllByText('多边形允许区域')).toHaveLength(2);
    expect(
      screen.queryByTestId('geofence-row-fence-archived'),
    ).not.toBeInTheDocument();

    const enabledRow = screen.getByTestId('geofence-row-fence-enabled');
    await user.click(
      within(enabledRow).getByRole('button', { name: '定位到围栏' }),
    );
    expect(onLocate).toHaveBeenCalledWith(enabledFence);

    const disabledRow = screen.getByTestId('geofence-row-fence-disabled');
    expect(
      within(disabledRow).getByRole('button', { name: '定位到围栏' }),
    ).toBeDisabled();

    const createButton = screen.getByRole('button', { name: '新增围栏' });
    await user.hover(createButton);
    expect(
      screen.queryByRole('tooltip', { name: '新增围栏' }),
    ).not.toBeInTheDocument();
    expect(within(createButton).getByText('新增围栏')).toBeInTheDocument();
    await user.click(createButton);
    await user.click(
      within(enabledRow).getByRole('button', { name: '编辑围栏' }),
    );
    await user.hover(
      within(enabledRow).getByRole('button', { name: '编辑围栏' }),
    );
    await waitFor(() =>
      expect(
        screen
          .getAllByRole('tooltip')
          .some((tooltip) => tooltip.textContent === '编辑围栏'),
      ).toBe(true),
    );
    await user.click(
      within(enabledRow).getByRole('button', { name: '绑定设备' }),
    );
    expect(onCreate).toHaveBeenCalledTimes(1);
    expect(onEdit).toHaveBeenCalledWith(enabledFence);
    expect(onBind).toHaveBeenCalledWith(enabledFence);

    await user.type(
      screen.getByLabelText('按围栏名称搜索'),
      '园区',
    );
    await user.keyboard('{Enter}');
    expect(mocks.mapQuery).toHaveBeenLastCalledWith(
      { name: '园区', carrier: 'cmcc' },
      { enabled: true },
    );
  });

  it('previews impact and records a reason before disabling', async () => {
    const { user } = renderPanel();
    const row = screen.getByTestId('geofence-row-fence-enabled');

    await user.click(
      within(row).getByRole('button', { name: '禁用围栏' }),
    );

    await waitFor(() =>
      expect(mocks.preview).toHaveBeenCalledWith({
        id: 'fence-enabled',
        target: 'disabled',
      }),
    );
    expect(screen.getByText('受影响设备数')).toBeInTheDocument();
    expect(
      screen.getByText(
        '禁用围栏只停止后续检查，不会自动恢复或重新激活设备',
      ),
    ).toBeInTheDocument();

    const confirm = screen.getByRole('button', { name: /确\s*认/ });
    expect(confirm).toBeDisabled();
    await user.type(screen.getByLabelText('操作原因'), '维护窗口');
    await user.click(confirm);

    await waitFor(() =>
      expect(mocks.transition).toHaveBeenCalledWith({
        id: 'fence-enabled',
        target: 'disabled',
        input: {
          reason: '维护窗口',
          previewFingerprint: 'preview-disabled',
        },
      }),
    );
  });

  it('blocks archive confirmation while a batch binding job is active', async () => {
    mocks.preview.mockResolvedValueOnce({
      ...impact('archived'),
      activeBatchJobCount: 1,
    });
    const { user } = renderPanel();
    const row = screen.getByTestId('geofence-row-fence-disabled');

    await user.click(
      within(row).getByRole('button', { name: '归档围栏' }),
    );

    expect(
      await screen.findByText(
        '存在尚未结束的批量绑定任务，任务进入终态后才能归档',
      ),
    ).toBeInTheDocument();
    await user.type(screen.getByLabelText('操作原因'), '验收清理');
    expect(screen.getByRole('button', { name: /确\s*认/ })).toBeDisabled();
    expect(mocks.transition).not.toHaveBeenCalled();
  });

  it('previews and confirms the selected carrier switch', async () => {
    const { user } = renderPanel();

    await user.click(
      screen.getByRole('switch', { name: '运营商围栏开关' }),
    );
    await waitFor(() =>
      expect(mocks.previewSettings).toHaveBeenCalledWith({
        systemMode: 'observe',
        carriers: [
          {
            carrier: 'cmcc',
            mode: 'off',
            defaultBaselineRadiusMeters: 1000,
          },
        ],
      }),
    );
    expect(screen.getByText('运营商围栏开关确认')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /确\s*认/ }));
    await waitFor(() =>
      expect(mocks.updateSettings).toHaveBeenCalledWith({
        systemMode: 'observe',
        carriers: [
          {
            carrier: 'cmcc',
            mode: 'off',
            defaultBaselineRadiusMeters: 1000,
          },
        ],
      }),
    );
  });

  it('passes selected in-fence candidates to the manual binding flow once', async () => {
    mocks.previewCandidates.mockResolvedValueOnce({
      geofenceId: 'fence-enabled',
      versionId: 'version-enabled',
      inside: [
        {
          id: 'device-1',
          serialNumber: '1202000240194DP0015',
          name: '172.24.224.27',
        },
      ],
      outside: [],
      noLocation: [],
    });
    const { user, onBind } = renderPanel();
    const row = screen.getByTestId('geofence-row-fence-enabled');

    await user.click(
      within(row).getByRole('button', { name: '区域设备预览' }),
    );
    await user.click(
      await screen.findByRole('checkbox', {
        name: '1202000240194DP0015 · 172.24.224.27',
      }),
    );
    await user.click(screen.getByRole('button', { name: '绑定所选设备' }));

    expect(onBind).toHaveBeenCalledTimes(1);
    expect(onBind).toHaveBeenCalledWith(enabledFence, [
      '1202000240194DP0015',
    ]);
  });

  it('only offers archive from disabled and requires preview confirmation', async () => {
    const { user } = renderPanel();
    const enabledRow = screen.getByTestId('geofence-row-fence-enabled');
    const disabledRow = screen.getByTestId('geofence-row-fence-disabled');

    expect(
      within(enabledRow).queryByRole('button', { name: '归档围栏' }),
    ).not.toBeInTheDocument();
    await user.click(
      within(disabledRow).getByRole('button', { name: '归档围栏' }),
    );
    await waitFor(() =>
      expect(mocks.preview).toHaveBeenCalledWith({
        id: 'fence-disabled',
        target: 'archived',
      }),
    );
    expect(
      screen.getByText(
        '归档会移除当前绑定且不会恢复设备原围栏归属，也不会自动恢复或重新激活设备',
      ),
    ).toBeInTheDocument();

    await user.type(screen.getByLabelText('操作原因'), '围栏已废弃');
    await user.click(screen.getByRole('button', { name: /确\s*认/ }));
    await waitFor(() =>
      expect(mocks.transition).toHaveBeenCalledWith({
        id: 'fence-disabled',
        target: 'archived',
        input: {
          reason: '围栏已废弃',
          previewFingerprint: 'preview-archived',
        },
      }),
    );
  });
});
