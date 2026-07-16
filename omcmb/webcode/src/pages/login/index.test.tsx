import { App as AntdApp } from 'antd';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useUserStore } from '@core/store/userStore';
import LoginPage from './index';

vi.mock('@core/services/apiSwitch', () => ({ useMock: true }));

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
    'login.unlockSuccess': '解锁成功',
    'login.usernameTip': '请输入用户名',
    'login.passwordTip': '请输入密码',
    'login.rememberMe': '记住我',
    'login.submit': '登录',
  }[key] ?? key),
}));

describe('LoginPage 锁屏模式', () => {
  beforeEach(() => {
    sessionStorage.clear();
    localStorage.clear();
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
      userId: '1',
      username: 'admin',
      displayName: 'System Admin',
      returnPath: '/system/config?tab=security',
    }));
  });

  it('显示解锁界面并在同一用户解锁后返回锁屏前页面', async () => {
    render(
      <AntdApp>
        <MemoryRouter initialEntries={['/login']}>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/system/config" element={<div>system config page</div>} />
            <Route path="/dashboard" element={<div>dashboard page</div>} />
          </Routes>
        </MemoryRouter>
      </AntdApp>,
    );

    expect(screen.getByRole('heading', { name: '屏幕已锁定' })).toBeVisible();
    expect(screen.getByPlaceholderText('请输入用户名')).toHaveValue('admin');
    expect(screen.getByPlaceholderText('请输入用户名')).toHaveAttribute('readonly');
    expect(screen.queryByText('记住我')).not.toBeInTheDocument();

    fireEvent.change(screen.getByPlaceholderText('请输入密码'), {
      target: { value: 'admin123' },
    });
    fireEvent.click(screen.getByRole('button', { name: /解\s*锁/ }));

    await waitFor(() => {
      expect(screen.getByText('system config page')).toBeVisible();
    });
  });

  it('同名账号身份已变化时不恢复旧用户页面', async () => {
    sessionStorage.setItem('omc-locked-session', JSON.stringify({
      userId: 'deleted-user-id',
      username: 'admin',
      displayName: 'Old Admin',
      returnPath: '/system/config?tab=security',
    }));

    render(
      <AntdApp>
        <MemoryRouter initialEntries={['/login']}>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/system/config" element={<div>system config page</div>} />
            <Route path="/dashboard" element={<div>dashboard page</div>} />
          </Routes>
        </MemoryRouter>
      </AntdApp>,
    );

    fireEvent.change(screen.getByPlaceholderText('请输入密码'), {
      target: { value: 'admin123' },
    });
    fireEvent.click(screen.getByRole('button', { name: /解\s*锁/ }));

    await waitFor(() => {
      expect(screen.getByText('dashboard page')).toBeVisible();
    });
    expect(screen.queryByText('system config page')).not.toBeInTheDocument();
  });
});
