import { App as AntdApp } from 'antd';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { getLockedSession, useUserStore } from '@core/store/userStore';
import LoginPage from './index';

const mocks = vi.hoisted(() => ({
  login: vi.fn(),
  getMe: vi.fn(),
}));

vi.mock('@core/services/apiSwitch', () => ({ useMock: false }));
vi.mock('@core/services/api/authApi', () => ({
  authApi: {
    login: mocks.login,
    getMe: mocks.getMe,
    getCaptcha: vi.fn(),
  },
}));
vi.mock('@core/hooks/api/useSecuritySettings', () => ({
  usePublicSecuritySettings: () => ({ settings: {} }),
}));
vi.mock('@core/hooks/api/useOmcName', () => ({
  usePublicOmcName: () => ({ omcName: '' }),
  resolveOmcName: (_name: string, fallback: string) => fallback,
}));
vi.mock('@/hooks/useT', () => ({
  useT: () => (key: string) => ({
    'login.title': 'OMC网管系统',
    'login.lockedTitle': '屏幕已锁定',
    'login.unlock': '解锁',
    'login.usernameTip': '请输入用户名',
    'login.passwordTip': '请输入密码',
  }[key] ?? key),
}));

describe('LoginPage 锁屏真实鉴权边界', () => {
  beforeEach(() => {
    sessionStorage.clear();
    localStorage.clear();
    mocks.login.mockReset();
    mocks.getMe.mockReset();
    useUserStore.setState({
      currentUser: null,
      accessToken: null,
      refreshToken: null,
      tokenExpiresAt: null,
      isAuthenticated: false,
      permissions: [],
      loading: false,
      mustChangePassword: false,
    });
    sessionStorage.setItem('omc-locked-session', JSON.stringify({
      userId: 'user-1',
      username: 'admin',
      displayName: 'System Admin',
      returnPath: '/system/config',
    }));
  });

  it('忽略被篡改的表单用户名，且 /auth/me 失败时回滚临时 token', async () => {
    mocks.login.mockResolvedValue({
      access_token: 'temporary-access',
      refresh_token: 'temporary-refresh',
      expires_at: new Date(Date.now() + 60_000).toISOString(),
    });
    mocks.getMe.mockRejectedValue(new Error('getMe failed'));

    render(
      <AntdApp>
        <MemoryRouter initialEntries={['/login']}>
          <LoginPage />
        </MemoryRouter>
      </AntdApp>,
    );

    fireEvent.change(screen.getByPlaceholderText('请输入用户名'), {
      target: { value: 'attacker' },
    });
    fireEvent.change(screen.getByPlaceholderText('请输入密码'), {
      target: { value: 'secret' },
    });
    fireEvent.click(screen.getByRole('button', { name: /解\s*锁/ }));

    await waitFor(() => expect(mocks.getMe).toHaveBeenCalledOnce());
    expect(mocks.login).toHaveBeenCalledWith('admin', 'secret', undefined);
    expect(useUserStore.getState().accessToken).toBeNull();
    expect(useUserStore.getState().isAuthenticated).toBe(false);
    expect(getLockedSession()?.userId).toBe('user-1');
  });
});
