import { fireEvent, render, screen } from '@testing-library/react';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import UserDropdown from './UserDropdown';

const mocks = vi.hoisted(() => ({
  logout: vi.fn(),
  mustChangePassword: false,
  navigate: vi.fn(),
  setMustChangePassword: vi.fn(),
  toggleLocale: vi.fn(),
}));

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mocks.navigate,
  };
});

vi.mock('@core/store/userStore', () => ({
  useUserStore: (selector: (state: Record<string, unknown>) => unknown) =>
    selector({
      currentUser: {
        id: 'admin-id',
        username: 'admin',
        displayName: 'System Admin',
        role: 'admin',
        status: 'active',
        createTime: '2026-01-01T00:00:00Z',
      },
      logout: mocks.logout,
      mustChangePassword: mocks.mustChangePassword,
      setMustChangePassword: mocks.setMustChangePassword,
    }),
}));

vi.mock('@core/store/appStore', () => ({
  useAppStore: (selector: (state: Record<string, unknown>) => unknown) =>
    selector({ locale: 'zh-CN', toggleLocale: mocks.toggleLocale }),
}));

vi.mock('@tanstack/react-query', () => ({
  useMutation: () => ({ isPending: false, mutate: vi.fn() }),
}));

vi.mock('@/hooks/useT', () => ({
  useT: () => (key: string) => key,
}));

vi.mock('@/hooks/useThemeToken', () => ({
  useThemeToken: () => ({
    colorPrimary: '#1677ff',
    colorTextSecondary: '#999',
    zIndexPopupBase: 1000,
  }),
}));

vi.mock('antd', async () => {
  const actual = await vi.importActual<typeof import('antd')>('antd');
  type ModalFooter = (
    originNode: ReactNode,
    extra: { CancelBtn: () => ReactNode; OkBtn: () => ReactNode },
  ) => ReactNode;
  return {
    ...actual,
    App: {
      ...actual.App,
      useApp: () => ({ message: { error: vi.fn(), success: vi.fn() } }),
    },
    Modal: ({
      children,
      footer,
      open,
    }: {
      children?: ReactNode;
      footer?: ModalFooter | ReactNode;
      open?: boolean;
    }) => {
      if (!open) return null;
      const renderedFooter = typeof footer === 'function'
        ? footer(null, { CancelBtn: () => null, OkBtn: () => null })
        : footer;
      return <div role="dialog">{children}{renderedFooter}</div>;
    },
  };
});

describe('UserDropdown', () => {
  beforeEach(() => {
    mocks.logout.mockReset();
    mocks.navigate.mockReset();
    mocks.setMustChangePassword.mockReset();
    mocks.mustChangePassword = false;
  });

  it('主动退出时清理认证并进入不带返回路由的登录页', () => {
    render(<UserDropdown />);

    fireEvent.click(screen.getByRole('button', { name: /System Admin/ }));
    fireEvent.click(screen.getByRole('menuitem', { name: /user\.logout/ }));

    expect(mocks.logout).toHaveBeenCalledOnce();
    expect(mocks.navigate).toHaveBeenCalledWith('/login', {
      flushSync: true,
      replace: true,
      state: null,
    });
    expect(mocks.navigate.mock.invocationCallOrder[0]).toBeLessThan(
      mocks.logout.mock.invocationCallOrder[0],
    );
  });

  it('强制改密弹窗主动退出时复用无返回路由的登录流程', () => {
    mocks.mustChangePassword = true;
    render(<UserDropdown />);

    fireEvent.click(screen.getByRole('button', { name: 'user.logout' }));

    expect(mocks.setMustChangePassword).toHaveBeenCalledWith(false);
    expect(mocks.navigate).toHaveBeenCalledWith('/login', {
      flushSync: true,
      replace: true,
      state: null,
    });
    expect(mocks.logout).toHaveBeenCalledOnce();
  });
});
