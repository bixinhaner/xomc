import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { IntlProvider } from 'react-intl';
import { getMessages } from '@core/i18n/index';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type {
  GeofenceSettings,
  GeofenceSettingsPreview,
} from '@core/types/geofence';

const mocks = vi.hoisted(() => ({
  settingsQuery: vi.fn(),
  preview: vi.fn(),
  update: vi.fn(),
}));

vi.mock('@core/hooks/api/useGeofence', () => ({
  useGeofenceSettings: () => mocks.settingsQuery(),
  usePreviewGeofenceSettings: () => ({
    mutateAsync: mocks.preview,
    isPending: false,
    reset: vi.fn(),
  }),
  useUpdateGeofenceSettings: () => ({
    mutateAsync: mocks.update,
    isPending: false,
  }),
}));

import GeofenceSettingsDrawer from './GeofenceSettingsDrawer';

const settings: GeofenceSettings = {
  systemMode: 'observe',
  carriers: [
    {
      carrier: 'cmcc',
      mode: 'observe',
      effectiveMode: 'observe',
      defaultBaselineRadiusMeters: 1000,
      updatedAt: '2026-07-31T08:00:00Z',
    },
    {
      carrier: 'ctcc',
      mode: 'off',
      effectiveMode: 'off',
      defaultBaselineRadiusMeters: 800,
      updatedAt: '2026-07-31T08:00:00Z',
    },
    {
      carrier: 'cucc',
      mode: 'off',
      effectiveMode: 'off',
      defaultBaselineRadiusMeters: 600,
      updatedAt: '2026-07-31T08:00:00Z',
    },
  ],
};

const preview: GeofenceSettingsPreview = {
  current: settings,
  proposed: settings,
  enabledGeofences: 2,
  activeBindings: 26,
  newlyObservedDevices: 11,
};

function renderDrawer(onClose = vi.fn()) {
  return {
    onClose,
    user: userEvent.setup(),
    ...render(
      <IntlProvider
        locale="zh-CN"
        defaultLocale="zh-CN"
        messages={getMessages('zh-CN')}
      >
        <GeofenceSettingsDrawer open onClose={onClose} />
      </IntlProvider>,
    ),
  };
}

describe('GeofenceSettingsDrawer', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.settingsQuery.mockReturnValue({
      data: settings,
      isLoading: false,
      isFetching: false,
      isError: false,
      refetch: vi.fn(),
    });
    mocks.preview.mockResolvedValue(preview);
    mocks.update.mockResolvedValue(settings);
  });

  it('previews the fetched settings before saving', async () => {
    const { user, onClose } = renderDrawer();

    expect(screen.getByText('电子围栏设置')).toBeInTheDocument();
    expect(await screen.findByText('中国移动')).toBeInTheDocument();
    expect(screen.queryByLabelText('系统运行模式')).not.toBeInTheDocument();
    expect(screen.getByText('已开启，按运营商开关生效')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '预览影响' }));

    await waitFor(() => expect(mocks.preview).toHaveBeenCalledWith({
      systemMode: 'observe',
      carriers: [
        {
          carrier: 'cmcc',
          mode: 'observe',
          defaultBaselineRadiusMeters: 1000,
        },
        {
          carrier: 'ctcc',
          mode: 'off',
          defaultBaselineRadiusMeters: 800,
        },
        {
          carrier: 'cucc',
          mode: 'off',
          defaultBaselineRadiusMeters: 600,
        },
      ],
    }));
    expect(screen.getByText('26')).toBeInTheDocument();
    expect(screen.getByText('活动绑定')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '确认保存' }));

    await waitFor(() => expect(mocks.update).toHaveBeenCalledTimes(1));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('does not expose the system master switch from the GIS drawer', async () => {
    renderDrawer();

    expect(await screen.findByText('中国移动')).toBeInTheDocument();
    expect(screen.queryByLabelText('系统运行模式')).not.toBeInTheDocument();
    expect(
      screen.getByText(/该系统级开关决定 GIS 地图是否呈现电子围栏入口/),
    ).toBeInTheDocument();
  });

  it('keeps the drawer open when the save request fails', async () => {
    mocks.update.mockRejectedValue(new Error('save failed'));
    const { user, onClose } = renderDrawer();

    await user.click(screen.getByRole('button', { name: '预览影响' }));
    await screen.findByText('活动绑定');
    await user.click(screen.getByRole('button', { name: '确认保存' }));

    await waitFor(() => expect(mocks.update).toHaveBeenCalledTimes(1));
    expect(onClose).not.toHaveBeenCalled();
    expect(screen.getByText('电子围栏设置')).toBeInTheDocument();
  });

  it('blocks operations when the authoritative settings fail to load', async () => {
    const refetch = vi.fn();
    mocks.settingsQuery.mockReturnValue({
      data: undefined,
      isLoading: false,
      isFetching: false,
      isError: true,
      refetch,
    });
    const { user } = renderDrawer();

    expect(screen.getByText('电子围栏数据加载失败')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '预览影响' })).toBeDisabled();
    await user.click(screen.getByRole('button', { name: /重\s*试/ }));
    expect(refetch).toHaveBeenCalledTimes(1);
    expect(mocks.preview).not.toHaveBeenCalled();
  });
});
