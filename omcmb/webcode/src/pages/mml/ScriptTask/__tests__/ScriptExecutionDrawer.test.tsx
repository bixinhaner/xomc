import { screen, render, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { IntlProvider } from 'react-intl';
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import zhCN from '@core/i18n/zh-CN';
import enUS from '@core/i18n/en-US';
import { useUserStore } from '@core/store/userStore';

const mocks = vi.hoisted(() => ({ execute: vi.fn() }));

vi.mock('@core/hooks/api/useMML', () => ({
  useCreateMMLScriptExecution: () => ({ mutateAsync: mocks.execute, isPending: false }),
}));

import ScriptExecutionDrawer from '../ScriptExecutionDrawer';
import { MMLScriptImportApiError } from '@core/services/api/mmlApi';

const script = {
  id: 'script-1', scriptName: '巡检脚本', description: '', content: '', creator: 'admin',
  createTime: '', updateTime: '2026-07-10T00:00:00Z', tags: [], status: 'active', type: 'batch', progress: 0,
};

const adminUser = {
  id: 'user-1',
  username: 'admin',
  displayName: 'Admin',
  email: 'admin@omc.example.com',
  role: 'admin' as const,
  status: 'active' as const,
  createTime: '2026-07-10T00:00:00Z',
};

function renderDrawer(
  props: Partial<React.ComponentProps<typeof ScriptExecutionDrawer>> = {},
  locale: 'zh-CN' | 'en-US' = 'zh-CN',
) {
  return render(
    <IntlProvider locale={locale} defaultLocale="zh-CN" messages={locale === 'zh-CN' ? zhCN : enUS}>
      <ScriptExecutionDrawer open script={script} onClose={vi.fn()} {...props} />
    </IntlProvider>,
  );
}

describe('ScriptExecutionDrawer', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useUserStore.setState({ currentUser: adminUser, isAuthenticated: true });
  });

  afterEach(() => {
    vi.useRealTimers();
    useUserStore.setState({ currentUser: null, isAuthenticated: false });
  });

  it('generates a default task name from script name and current time in Chinese', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-07-10T13:38:30+08:00'));
    renderDrawer();

    expect(screen.getByLabelText('任务名称')).toHaveValue('巡检脚本_2026-07-10 13:38:30');
  });

  it('uses the same script-name based default task name in English', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-07-10T13:38:30+08:00'));
    renderDrawer({}, 'en-US');

    expect(screen.getByLabelText('任务名称')).toHaveValue('巡检脚本_2026-07-10 13:38:30');
  });

  it('submits execution only once while the request is in flight', async () => {
    let resolveExecution: (value: unknown) => void = () => {};
    mocks.execute.mockReturnValue(new Promise((resolve) => { resolveExecution = resolve; }));
    renderDrawer();
    const user = userEvent.setup();

    await user.click(screen.getByRole('button', { name: '执行' }));
    await user.click(screen.getByRole('button', { name: '执行' }));

    expect(mocks.execute).toHaveBeenCalledTimes(1);
    expect(mocks.execute).toHaveBeenCalledWith(expect.objectContaining({
      id: 'script-1',
      input: expect.objectContaining({ requestId: expect.any(String) }),
    }));
    resolveExecution({
      task: { id: 'task-1' },
      validation: {
        planItems: [],
        summary: { totalLines: 1, validLines: 1, effectiveLines: 1, deviceCount: 1, errorCount: 0, warningCount: 0 },
        issues: [],
      },
    });
  });

  it('confirms server warnings and retries with confirmWarnings', async () => {
    mocks.execute.mockResolvedValueOnce({
      task: { id: 'task-1' },
      validation: {
        planItems: [],
        summary: { totalLines: 1, validLines: 1, effectiveLines: 1, deviceCount: 1, errorCount: 0, warningCount: 1 },
        issues: [{ code: 'MML_DEVICE_OFFLINE', severity: 'warning', lineNo: 1 }],
      },
    }).mockResolvedValueOnce({ task: { id: 'task-1' }, validation: { planItems: [], summary: { totalLines: 1, validLines: 1, effectiveLines: 1, deviceCount: 1, errorCount: 0, warningCount: 0 }, issues: [] } });
    renderDrawer();
    const user = userEvent.setup();
    await user.type(screen.getByLabelText('任务名称'), '巡检任务');
    await user.click(screen.getByRole('button', { name: '执行' }));
    expect(await screen.findByText(/校验发现警告/)).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: '确认执行' }));
    await waitFor(() => expect(mocks.execute).toHaveBeenLastCalledWith(expect.objectContaining({
      id: 'script-1', input: expect.objectContaining({ confirmWarnings: true }),
    })));
  });

  it('keeps the drawer open and displays line issues when execution fails validation', async () => {
    mocks.execute.mockRejectedValue({
      message: 'validation failed',
      validation: {
        planItems: [],
        summary: { totalLines: 1, validLines: 0, effectiveLines: 0, deviceCount: 0, errorCount: 1, warningCount: 0 },
        issues: [{ code: 'MML_PARAMETER_UNKNOWN', severity: 'error', lineNo: 1 }],
      },
    });
    renderDrawer();
    const user = userEvent.setup();
    await user.type(screen.getByLabelText('任务名称'), '失败任务');
    await user.click(screen.getByRole('button', { name: '执行' }));
    await waitFor(() => expect(mocks.execute).toHaveBeenCalled());
    expect(await screen.findByText('MML_PARAMETER_UNKNOWN')).toBeInTheDocument();
    expect(screen.getByRole('dialog')).toBeInTheDocument();
  });

  it('confirms a typed 409 warning rejection and retries with confirmWarnings', async () => {
    const warningValidation = {
      planItems: [],
      summary: { totalLines: 1, validLines: 1, effectiveLines: 1, deviceCount: 1, errorCount: 0, warningCount: 1 },
      issues: [{ code: 'MML_DEVICE_OFFLINE', severity: 'warning' as const, lineNo: 1 }],
    };
    mocks.execute
      .mockRejectedValueOnce(new MMLScriptImportApiError({ status: 409, code: 'MML_SCRIPT_WARNINGS', message: 'warnings', validation: warningValidation }))
      .mockResolvedValueOnce({ task: { id: 'task-2' }, validation: { ...warningValidation, summary: { ...warningValidation.summary, warningCount: 0 }, issues: [] } });
    renderDrawer();
    const user = userEvent.setup();
    await user.type(screen.getByLabelText('任务名称'), 'typed-warning-task');
    await user.click(screen.getByRole('button', { name: '执行' }));
    expect(await screen.findByRole('button', { name: '确认执行' })).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: '确认执行' }));
    await waitFor(() => expect(mocks.execute).toHaveBeenLastCalledWith(expect.objectContaining({ input: expect.objectContaining({ confirmWarnings: true }) })));
  });

  it('requires periodic date and time fields before submitting', async () => {
    renderDrawer();
    const user = userEvent.setup();
    await user.click(screen.getByRole('radio', { name: '周期' }));
    expect(screen.getByLabelText('周期日期')).toBeInTheDocument();
    expect(screen.getByLabelText('周期时间')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: '执行' }));
    expect(await screen.findByText('请选择周期日期范围')).toBeInTheDocument();
    expect(mocks.execute).not.toHaveBeenCalled();
  });

  it('requires a scheduled time before submitting scheduled execution', async () => {
    renderDrawer();
    const user = userEvent.setup();
    await user.click(screen.getByRole('radio', { name: '定时' }));
    await user.click(screen.getByRole('button', { name: '执行' }));
    expect(await screen.findByText('请选择执行时间')).toBeInTheDocument();
    expect(mocks.execute).not.toHaveBeenCalled();
  });

  it('does not confirm when typed validation contains both errors and warnings', async () => {
    const mixedValidation = {
      planItems: [],
      summary: { totalLines: 2, validLines: 0, effectiveLines: 0, deviceCount: 1, errorCount: 1, warningCount: 1 },
      issues: [
        { code: 'MML_LINE_FORMAT_INVALID', severity: 'error' as const, lineNo: 1 },
        { code: 'MML_DEVICE_OFFLINE', severity: 'warning' as const, lineNo: 2 },
      ],
    };
    mocks.execute.mockRejectedValueOnce({ message: 'mixed validation', validation: mixedValidation });
    renderDrawer();
    const user = userEvent.setup();
    await user.type(screen.getByLabelText('任务名称'), 'mixed-task');
    await user.click(screen.getByRole('button', { name: '执行' }));
    expect((await screen.findAllByText(/MML_LINE_FORMAT_INVALID/)).length).toBeGreaterThan(0);
    expect(screen.queryByRole('button', { name: '确认执行' })).not.toBeInTheDocument();
  });
});
