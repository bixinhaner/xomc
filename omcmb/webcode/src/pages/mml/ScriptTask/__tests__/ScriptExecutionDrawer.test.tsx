import { screen, render, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi, beforeEach } from 'vitest';

const mocks = vi.hoisted(() => ({ execute: vi.fn() }));

vi.mock('@core/hooks/api/useMML', () => ({
  useCreateMMLScriptExecution: () => ({ mutateAsync: mocks.execute, isPending: false }),
}));

import ScriptExecutionDrawer from '../ScriptExecutionDrawer';

const script = {
  id: 'script-1', scriptName: '巡检脚本', description: '', content: '', creator: 'admin',
  createTime: '', updateTime: '2026-07-10T00:00:00Z', tags: [], status: 'active', type: 'batch', progress: 0,
};

describe('ScriptExecutionDrawer', () => {
  beforeEach(() => vi.clearAllMocks());

  it('confirms server warnings and retries with confirmWarnings', async () => {
    mocks.execute.mockResolvedValueOnce({
      task: { id: 'task-1' },
      validation: {
        planItems: [],
        summary: { totalLines: 1, validLines: 1, effectiveLines: 1, deviceCount: 1, errorCount: 0, warningCount: 1 },
        issues: [{ code: 'MML_DEVICE_OFFLINE', severity: 'warning', lineNo: 1 }],
      },
    }).mockResolvedValueOnce({ task: { id: 'task-1' }, validation: { planItems: [], summary: { totalLines: 1, validLines: 1, effectiveLines: 1, deviceCount: 1, errorCount: 0, warningCount: 0 }, issues: [] } });
    render(<ScriptExecutionDrawer open script={script} onClose={vi.fn()} />);
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
    render(<ScriptExecutionDrawer open script={script} onClose={vi.fn()} />);
    const user = userEvent.setup();
    await user.type(screen.getByLabelText('任务名称'), '失败任务');
    await user.click(screen.getByRole('button', { name: '执行' }));
    await waitFor(() => expect(mocks.execute).toHaveBeenCalled());
    expect(await screen.findByText('MML_PARAMETER_UNKNOWN')).toBeInTheDocument();
    expect(screen.getByRole('dialog')).toBeInTheDocument();
  });
});
