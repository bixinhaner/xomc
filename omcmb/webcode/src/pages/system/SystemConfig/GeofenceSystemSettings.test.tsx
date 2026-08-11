import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { IntlProvider } from 'react-intl';
import { getMessages } from '@core/i18n/index';
import { beforeEach, describe, expect, it, vi } from 'vitest';

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
  }),
  useUpdateGeofenceSettings: () => ({
    mutateAsync: mocks.update,
    isPending: false,
  }),
}));

import GeofenceSystemSettings from './GeofenceSystemSettings';

const settings = {
  systemMode: 'off' as const,
  carriers: [{
    carrier: 'cmcc',
    mode: 'off' as const,
    effectiveMode: 'off' as const,
    defaultBaselineRadiusMeters: 1000,
    updatedAt: '2026-08-05T08:00:00Z',
  }],
};

function renderSettings() {
  return {
    user: userEvent.setup(),
    ...render(
      <IntlProvider
        locale="zh-CN"
        defaultLocale="zh-CN"
        messages={getMessages('zh-CN')}
      >
        <GeofenceSystemSettings />
      </IntlProvider>,
    ),
  };
}

describe('GeofenceSystemSettings', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.settingsQuery.mockReturnValue({
      data: settings,
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
    });
    mocks.preview.mockResolvedValue({
      current: settings,
      proposed: { ...settings, systemMode: 'enforce' },
      enabledGeofences: 2,
      activeBindings: 3,
      newlyObservedDevices: 4,
    });
    mocks.update.mockResolvedValue({
      ...settings,
      systemMode: 'enforce',
    });
  });

  it('previews and confirms the system switch without changing carrier settings', async () => {
    const { user } = renderSettings();

    await user.click(
      screen.getByRole('switch', { name: '电子围栏总开关' }),
    );

    await waitFor(() => expect(mocks.preview).toHaveBeenCalledWith({
      systemMode: 'enforce',
      carriers: [{
        carrier: 'cmcc',
        mode: 'off',
        defaultBaselineRadiusMeters: 1000,
      }],
    }));
    expect(screen.getByText('活动绑定')).toBeInTheDocument();

    await user.click(
      screen.getByRole('button', { name: /确\s*认/ }),
    );
    await waitFor(() => expect(mocks.update).toHaveBeenCalledTimes(1));
  });
});
