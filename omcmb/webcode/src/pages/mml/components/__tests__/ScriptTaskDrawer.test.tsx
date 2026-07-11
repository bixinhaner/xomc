import { fireEvent, render, screen } from '@testing-library/react';
import { IntlProvider } from 'react-intl';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import zhCN from '@core/i18n/zh-CN';
import enUS from '@core/i18n/en-US';
import { useUserStore } from '@core/store/userStore';

const mocks = vi.hoisted(() => ({
  createTask: vi.fn(),
}));

vi.mock('@core/hooks/api/useMML', () => ({
  useCreateMMLTask: () => ({ mutateAsync: mocks.createTask, isPending: false }),
}));

vi.mock('../../Console/components/DeviceSelectModal', () => ({
  default: () => null,
}));

import ScriptTaskDrawer from '../ScriptTaskDrawer';

const adminUser = {
  id: 'user-1',
  username: 'admin',
  displayName: 'Admin',
  email: 'admin@omc.example.com',
  role: 'admin' as const,
  status: 'active' as const,
  createTime: '2026-07-10T00:00:00Z',
};

function renderDrawer(locale: 'zh-CN' | 'en-US' = 'zh-CN') {
  return render(
    <IntlProvider locale={locale} defaultLocale="zh-CN" messages={locale === 'zh-CN' ? zhCN : enUS}>
      <ScriptTaskDrawer open onClose={vi.fn()} prefillContent="LST DEVICE_INFO;SN1" />
    </IntlProvider>,
  );
}

describe('ScriptTaskDrawer default task name', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-07-10T13:38:30+08:00'));
    useUserStore.setState({ currentUser: adminUser, isAuthenticated: true });
  });

  afterEach(() => {
    vi.useRealTimers();
    useUserStore.setState({ currentUser: null, isAuthenticated: false });
  });

  it('auto-generates an editable localized Chinese task name when opened', async () => {
    renderDrawer();

    const input = screen.getByLabelText('任务名称');
    expect(input).toHaveValue('MML脚本任务_admin_2026-07-10 13:38:30');

    fireEvent.change(input, { target: { value: '人工修改后的任务名' } });

    expect(input).toHaveValue('人工修改后的任务名');
  });

  it('auto-generates an editable localized English task name when opened', () => {
    renderDrawer('en-US');

    expect(screen.getByLabelText('Task Name')).toHaveValue('MML Script Task_admin_2026-07-10 13:38:30');
  });
});
