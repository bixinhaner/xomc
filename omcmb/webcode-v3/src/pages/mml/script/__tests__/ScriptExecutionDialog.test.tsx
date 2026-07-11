import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({ execute: vi.fn(), pending: false }))
vi.mock('@core/hooks/api/useMML', () => ({ useCreateMMLScriptExecution: () => ({ mutateAsync: mocks.execute, isPending: mocks.pending }) }))

beforeEach(() => { vi.clearAllMocks(); mocks.pending = false })

import ScriptExecutionDialog from '../ScriptExecutionDialog'

const script = { id: 'script-1', scriptName: 'demo' } as never
const validation = (issues: Array<{ code: string; severity: 'error' | 'warning'; lineNo?: number }>) => ({
  summary: { totalLines: 1, validLines: 1, effectiveLines: 1, deviceCount: 1, errorCount: issues.filter((i) => i.severity === 'error').length, warningCount: issues.filter((i) => i.severity === 'warning').length },
  issues,
  planItems: [],
})

describe('v3 ScriptExecutionDialog', () => {
  it('confirms warning-only 409 and retries with confirmation', async () => {
    mocks.execute.mockRejectedValueOnce(Object.assign(new Error('warnings require confirmation'), { status: 409, validation: validation([{ code: 'MML_DEVICE_OFFLINE', severity: 'warning' }]) })).mockResolvedValueOnce({ validation: validation([]) })
    const user = userEvent.setup()
    render(<ScriptExecutionDialog open script={script} onClose={vi.fn()} />)
    await user.click(screen.getByRole('button', { name: /执行/ }))
    expect(await screen.findByRole('button', { name: '确认执行' })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: '确认执行' }))
    await waitFor(() => expect(mocks.execute).toHaveBeenCalledTimes(2))
    expect(mocks.execute.mock.calls[1][0].input.confirmWarnings).toBe(true)
  })

  it('keeps mixed 422 errors visible without opening warning confirmation', async () => {
    mocks.execute.mockRejectedValueOnce(Object.assign(new Error('validation failed'), { status: 422, validation: validation([{ code: 'MML_LINE_FORMAT_INVALID', severity: 'error', lineNo: 7 }, { code: 'MML_DEVICE_OFFLINE', severity: 'warning', lineNo: 8 }]) }))
    const user = userEvent.setup()
    render(<ScriptExecutionDialog open script={script} onClose={vi.fn()} />)
    await user.click(screen.getByRole('button', { name: /执行/ }))
    expect(await screen.findByText(/MML_LINE_FORMAT_INVALID/)).toBeInTheDocument()
    expect(screen.getByText(/第 7 行/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: '确认执行' })).not.toBeInTheDocument()
  })

  it('supports immediate, suspended, scheduled, periodic, offline and failed retry controls', async () => {
    mocks.execute.mockResolvedValueOnce({ validation: validation([]) })
    const user = userEvent.setup()
    render(<ScriptExecutionDialog open script={script} onClose={vi.fn()} />)
    await user.selectOptions(screen.getByLabelText('执行方式'), 'periodic')
    await user.type(screen.getByLabelText('周期开始'), '2026-07-10')
    await user.type(screen.getByLabelText('周期结束'), '2026-07-11')
    await user.type(screen.getByLabelText('周期时间'), '08:30')
    await user.click(screen.getByLabelText('离线等待重试'))
    await user.click(screen.getByLabelText('失败重试'))
    await user.click(screen.getByRole('button', { name: /执行/ }))
    await waitFor(() => expect(mocks.execute).toHaveBeenCalled())
    expect(mocks.execute.mock.lastCall?.[0].input.periodTime).toBe('08:30:00')
    expect(mocks.execute.mock.lastCall?.[0].input.periodStart).toBe('2026-07-10T00:00:00')
    expect(mocks.execute.mock.lastCall?.[0].input.periodEnd).toBe('2026-07-11T23:59:59')
    expect(mocks.execute.mock.lastCall?.[0].input.offlineRetry).toBe(true)
    expect(mocks.execute.mock.lastCall?.[0].input.failedRetry).toBe(true)
  })

  it('normalizes scheduled datetime-local values to backend seconds', async () => {
    mocks.execute.mockResolvedValueOnce({ validation: validation([]) })
    const user = userEvent.setup()
    render(<ScriptExecutionDialog open script={script} onClose={vi.fn()} />)
    await user.selectOptions(screen.getByLabelText('执行方式'), 'scheduled')
    await user.type(screen.getByLabelText('执行时间'), '2026-07-10T08:30')
    await user.click(screen.getByRole('button', { name: /执行/ }))
    await waitFor(() => expect(mocks.execute).toHaveBeenCalled())
    expect(mocks.execute.mock.lastCall?.[0].input.scheduledAt).toBe('2026-07-10T08:30:00')
  })

  it('exposes accessible execution loading text while submission is pending', async () => {
    let release!: (value: { validation: ReturnType<typeof validation> }) => void
    mocks.execute.mockImplementationOnce(() => {
      mocks.pending = true
      return new Promise((resolve) => { release = resolve })
    })
    const user = userEvent.setup()
    render(<ScriptExecutionDialog open script={script} onClose={vi.fn()} />)
    await user.click(screen.getByRole('button', { name: /执行/ }))
    expect(screen.getByRole('status')).toHaveTextContent('正在提交执行…')
    release({ validation: validation([]) })
  })

  it('closes on Escape', async () => {
    const onClose = vi.fn()
    render(<ScriptExecutionDialog open script={script} onClose={onClose} />)
    await userEvent.setup().keyboard('{Escape}')
    expect(onClose).toHaveBeenCalled()
  })
})
