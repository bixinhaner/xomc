import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({ execute: vi.fn() }))
vi.mock('@core/hooks/api/useMML', () => ({ useCreateMMLScriptExecution: () => ({ mutateAsync: mocks.execute, isPending: false }) }))

import ScriptExecutionDialog from '../components/ScriptExecutionDialog'

const script = { id: 'script-1', scriptName: 'demo' } as never
const validation = (issues: Array<{ code: string; severity: 'error' | 'warning' }>) => ({
  summary: { totalLines: 1, validLines: 1, effectiveLines: 1, deviceCount: 1, errorCount: issues.filter((i) => i.severity === 'error').length, warningCount: issues.filter((i) => i.severity === 'warning').length },
  issues,
  planItems: [],
})

describe('v2 ScriptExecutionDialog typed validation errors', () => {
  it('opens warning confirmation for rejected warning-only 409 and retries with confirmation', async () => {
    mocks.execute.mockRejectedValueOnce(Object.assign(new Error('warnings require confirmation'), { status: 409, validation: validation([{ code: 'MML_DEVICE_OFFLINE', severity: 'warning' }]) })).mockResolvedValueOnce({ validation: validation([]) })
    const user = userEvent.setup()
    render(<ScriptExecutionDialog open script={script} onClose={vi.fn()} />)
    await user.click(screen.getByRole('button', { name: /执行/ }))
    expect(await screen.findByRole('button', { name: '确认执行' })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: '确认执行' }))
    await waitFor(() => expect(mocks.execute).toHaveBeenCalledTimes(2))
    expect(mocks.execute.mock.calls[1][0].input.confirmWarnings).toBe(true)
  })

  it('does not open confirmation for mixed error and warning', async () => {
    mocks.execute.mockRejectedValueOnce(Object.assign(new Error('validation failed'), { status: 422, validation: validation([{ code: 'BAD', severity: 'error' }, { code: 'WARN', severity: 'warning' }]) }))
    const user = userEvent.setup()
    render(<ScriptExecutionDialog open script={script} onClose={vi.fn()} />)
    await user.click(screen.getByRole('button', { name: /执行/ }))
    expect(await screen.findByText(/执行校验失败|validation failed/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '确认执行' })).not.toBeInTheDocument()
  })

  it('normalizes periodic HTML time to backend HH:MM:SS', async () => {
    mocks.execute.mockResolvedValueOnce({ validation: validation([]) })
    const user = userEvent.setup()
    render(<ScriptExecutionDialog open script={script} onClose={vi.fn()} />)
    await user.selectOptions(screen.getByLabelText('执行方式'), 'periodic')
    await user.type(screen.getByLabelText('周期开始'), '2026-07-10')
    await user.type(screen.getByLabelText('周期结束'), '2026-07-11')
    await user.type(screen.getByLabelText('周期时间'), '08:30')
    await user.click(screen.getByRole('button', { name: /执行/ }))
    await waitFor(() => expect(mocks.execute).toHaveBeenCalled())
    expect(mocks.execute.mock.lastCall?.[0].input.periodTime).toBe('08:30:00')
    expect(mocks.execute.mock.lastCall?.[0].input.periodStart).toBe('2026-07-10T00:00:00')
    expect(mocks.execute.mock.lastCall?.[0].input.periodEnd).toBe('2026-07-11T23:59:59')
  })
})
