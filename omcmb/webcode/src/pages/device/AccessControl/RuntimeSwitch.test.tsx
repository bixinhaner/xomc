import { App } from 'antd';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useDeviceAccessRuntimeSettings, useUpdateDeviceAccessRuntimeSettings } from '@core/hooks/api/useDeviceAccess';
import RuntimeSwitch from './RuntimeSwitch';

vi.mock('@core/hooks/api/useDeviceAccess', () => ({
  useDeviceAccessRuntimeSettings: vi.fn(),
  useUpdateDeviceAccessRuntimeSettings: vi.fn(),
}));

const t = (key: string) => key;

beforeEach(() => {
  vi.mocked(useUpdateDeviceAccessRuntimeSettings).mockReturnValue({
    isPending: false,
    mutateAsync: vi.fn(),
  } as never);
});

describe('RuntimeSwitch', () => {
  it('shows the effective operator switch', () => {
    vi.mocked(useDeviceAccessRuntimeSettings).mockReturnValue({
      data: { carrier: 'cmcc', enabled: true },
      error: null,
      isLoading: false,
    } as never);

    render(<App><RuntimeSwitch operatorCode="cmcc" allowed t={t} /></App>);

    expect(screen.getByRole('switch', { name: 'deviceAccess.businessSwitch' })).toBeChecked();
    expect(screen.getByText('deviceAccess.enabledDescription')).toBeInTheDocument();
  });

  it('keeps the switch read-only without the settings permission', () => {
    vi.mocked(useDeviceAccessRuntimeSettings).mockReturnValue({
      data: { carrier: 'ctcc', enabled: false },
      error: null,
      isLoading: false,
    } as never);

    render(<App><RuntimeSwitch operatorCode="ctcc" allowed={false} t={t} /></App>);

    expect(screen.getByRole('switch', { name: 'deviceAccess.businessSwitch' })).toBeDisabled();
    expect(screen.getByText('deviceAccess.disabledDescription')).toBeInTheDocument();
  });

  it('requires confirmation before enabling the operator switch', async () => {
    const mutateAsync = vi.fn().mockResolvedValue(undefined);
    vi.mocked(useUpdateDeviceAccessRuntimeSettings).mockReturnValue({
      isPending: false,
      mutateAsync,
    } as never);
    vi.mocked(useDeviceAccessRuntimeSettings).mockReturnValue({
      data: { carrier: 'cmcc', enabled: false },
      error: null,
      isLoading: false,
    } as never);
    const user = userEvent.setup();

    render(<App><RuntimeSwitch operatorCode="cmcc" allowed t={t} /></App>);
    await user.click(screen.getByRole('switch', { name: 'deviceAccess.businessSwitch' }));

    expect(screen.getByRole('dialog', { name: 'deviceAccess.enableConfirmTitle' })).toBeInTheDocument();
    expect(mutateAsync).not.toHaveBeenCalled();
    await user.click(screen.getByRole('button', { name: 'common.confirm' }));
    expect(mutateAsync).toHaveBeenCalledWith({ operatorCode: 'cmcc', enabled: true });
  });
});
