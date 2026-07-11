import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  validate: vi.fn(),
  create: vi.fn(),
  replace: vi.fn(),
  template: vi.fn(),
  validateReplacement: vi.fn(),
}))

beforeEach(() => vi.clearAllMocks())

vi.mock('@core/hooks/api/useMML', () => ({
  useValidateMMLScriptImport: () => ({ mutateAsync: mocks.validate, isPending: false }),
  useCreateImportedMMLScript: () => ({ mutateAsync: mocks.create, isPending: false }),
  useReplaceImportedMMLScript: () => ({ mutateAsync: mocks.replace, isPending: false }),
}))

vi.mock('@core/services/api/mmlApi', () => ({
  mmlApi: { downloadScriptImportTemplate: mocks.template, validateScriptReplacement: mocks.validateReplacement },
}))

import ScriptImportDialog from '../ScriptImportDialog'

const validation = (overrides: Record<string, unknown> = {}) => ({
  validationToken: 'token-1',
  originalFilename: 'script.txt',
  planItems: [{ lineNo: 1, deviceSn: 'SN1', order: 1, rawLine: 'LST DEVICE_INFO;SN1', command: { commandCode: 'LST DEVICE_INFO' } }],
  summary: { totalLines: 1, validLines: 1, effectiveLines: 1, deviceCount: 1, errorCount: 0, warningCount: 0 },
  issues: [],
  ...overrides,
})

describe('v3 ScriptImportDialog', () => {
  it('supports TXT upload, template download, read-only preview, re-import and warning confirmation', async () => {
    mocks.validate.mockResolvedValueOnce(validation({ summary: { totalLines: 1, validLines: 1, effectiveLines: 1, deviceCount: 1, errorCount: 0, warningCount: 1 }, issues: [{ code: 'MML_DEVICE_OFFLINE', severity: 'warning', lineNo: 1 }] }))
    mocks.create.mockResolvedValue({ id: 'script-1' })
    mocks.template.mockResolvedValue({ blob: new Blob(['template']), filename: 'MMLTemplate.txt' })
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:test')
    const user = userEvent.setup()
    render(<ScriptImportDialog open onClose={vi.fn()} />)
    expect(screen.getByRole('button', { name: '下载模板' })).toBeInTheDocument()
    await user.upload(screen.getByLabelText('选择 TXT'), new File(['LST DEVICE_INFO;SN1'], 'script.txt', { type: 'text/plain' }))
    expect(mocks.validate).toHaveBeenCalled()
    await user.click(screen.getByRole('button', { name: '下载模板' }))
    expect(mocks.template).toHaveBeenCalled()
    await user.type(screen.getByLabelText('脚本名称'), 'warning-script')
    expect(screen.queryByRole('textbox', { name: '脚本内容' })).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: '确认保存' }))
    expect(await screen.findByRole('button', { name: '继续保存' })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: '继续保存' }))
    await waitFor(() => expect(mocks.create).toHaveBeenCalled())
  })

  it('opens the hidden TXT file input from the visible chooser button', async () => {
    const inputClick = vi.spyOn(HTMLInputElement.prototype, 'click').mockImplementation(() => {})
    const user = userEvent.setup()
    render(<ScriptImportDialog open onClose={vi.fn()} />)

    try {
      await user.click(screen.getByRole('button', { name: '选择 TXT' }))

      expect(inputClick).toHaveBeenCalledTimes(1)
    } finally {
      inputClick.mockRestore()
    }
  })

  it('filters errors and disables save', async () => {
    mocks.validate.mockResolvedValue(validation({ planItems: [{ lineNo: 1, deviceSn: 'SN1', order: 1, command: { commandCode: 'BAD' } }, { lineNo: 2, deviceSn: 'SN2', order: 1, command: { commandCode: 'WARN' } }], summary: { totalLines: 2, validLines: 1, effectiveLines: 1, deviceCount: 2, errorCount: 1, warningCount: 1 }, issues: [{ code: 'MML_LINE_FORMAT_INVALID', severity: 'error', lineNo: 1 }, { code: 'MML_DEVICE_OFFLINE', severity: 'warning', lineNo: 2 }] }))
    const user = userEvent.setup()
    render(<ScriptImportDialog open onClose={vi.fn()} />)
    await user.upload(screen.getByLabelText('选择 TXT'), new File(['BAD'], 'bad.txt', { type: 'text/plain' }))
    expect(mocks.validate).toHaveBeenCalled()
    await user.click(screen.getByRole('button', { name: '仅看错误' }))
    expect(screen.getByText('SN1')).toBeInTheDocument()
    expect(screen.queryByText('SN2')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: '确认保存' })).toBeDisabled()
  })

  it('exposes accessible validation loading text while upload is pending', async () => {
    let release!: (value: ReturnType<typeof validation>) => void
    mocks.validate.mockImplementationOnce(() => new Promise((resolve) => { release = resolve }))
    const user = userEvent.setup()
    render(<ScriptImportDialog open onClose={vi.fn()} />)
    const uploadPromise = user.upload(screen.getByLabelText('选择 TXT'), new File(['PENDING'], 'pending.txt', { type: 'text/plain' }))
    expect(await screen.findByRole('status')).toHaveTextContent('正在校验脚本…')
    release(validation())
    await uploadPromise
    await waitFor(() => expect(screen.queryByText('正在校验脚本…')).not.toBeInTheDocument())
  })

  it('closes on Escape and uses replacement endpoint for re-import', async () => {
    const onClose = vi.fn()
    mocks.validateReplacement.mockResolvedValue(validation({ originalFilename: 'replacement.txt' }))
    const user = userEvent.setup()
    render(<ScriptImportDialog open script={{ id: 'script-1', scriptName: 'old', description: '', updateTime: '2026-07-10T00:00:00Z' } as never} onClose={onClose} />)
    await user.upload(screen.getByLabelText('选择 TXT'), new File(['NEW'], 'replacement.txt', { type: 'text/plain' }))
    await waitFor(() => expect(mocks.validateReplacement).toHaveBeenCalledWith('script-1', expect.any(File)))
    await user.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalled()
  })
})
