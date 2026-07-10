import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  validate: vi.fn(),
  create: vi.fn(),
  replace: vi.fn(),
  template: vi.fn(),
}))

vi.mock('@core/hooks/api/useMML', () => ({
  useValidateMMLScriptImport: () => ({ mutateAsync: mocks.validate, isPending: false }),
  useCreateImportedMMLScript: () => ({ mutateAsync: mocks.create, isPending: false }),
  useReplaceImportedMMLScript: () => ({ mutateAsync: mocks.replace, isPending: false }),
}))

vi.mock('@core/services/api/mmlApi', () => ({
  mmlApi: { downloadScriptImportTemplate: mocks.template, validateScriptReplacement: vi.fn() },
}))

import ScriptImportDialog from '../components/ScriptImportDialog'

const validation = (overrides: Record<string, unknown> = {}) => ({
  validationToken: 'token-1',
  originalFilename: 'script.txt',
  planItems: [{ lineNo: 1, deviceSn: 'SN1', order: 1, rawLine: 'LST DEVICE_INFO;SN1', command: { commandCode: 'LST DEVICE_INFO' } }],
  summary: { totalLines: 1, validLines: 1, effectiveLines: 1, deviceCount: 1, errorCount: 0, warningCount: 0 },
  issues: [],
  ...overrides,
})

describe('v2 ScriptImportDialog', () => {
  it('supports TXT upload, template download, read-only preview, re-import and warning confirmation', async () => {
    mocks.validate.mockResolvedValueOnce(validation({ summary: { totalLines: 1, validLines: 1, effectiveLines: 1, deviceCount: 1, errorCount: 0, warningCount: 1 }, issues: [{ code: 'MML_DEVICE_OFFLINE', severity: 'warning', lineNo: 1 }] }))
    mocks.create.mockResolvedValue({ id: 'script-1' })
    mocks.template.mockResolvedValue({ blob: new Blob(['template']), filename: 'MMLTemplate.txt' })
    const user = userEvent.setup()
    render(<ScriptImportDialog open onClose={vi.fn()} />)

    expect(screen.getByRole('button', { name: '下载模板' })).toBeInTheDocument()
    const input = screen.getByLabelText('选择 TXT')
    await user.upload(input, new File(['LST DEVICE_INFO;SN1'], 'script.txt', { type: 'text/plain' }))
    expect(mocks.validate).toHaveBeenCalled()
    await user.type(screen.getByLabelText('脚本名称'), 'warning-script')
    expect(screen.queryByRole('textbox', { name: '脚本内容' })).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: '确认保存' }))
    expect(await screen.findByRole('button', { name: '继续保存' })).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: '继续保存' }))
    await waitFor(() => expect(mocks.create).toHaveBeenCalled())
  })

  it('filters errors and disables save', async () => {
    mocks.validate.mockResolvedValue(validation({ summary: { totalLines: 1, validLines: 0, effectiveLines: 0, deviceCount: 0, errorCount: 1, warningCount: 0 }, issues: [{ code: 'MML_LINE_FORMAT_INVALID', severity: 'error', lineNo: 1 }] }))
    const user = userEvent.setup()
    render(<ScriptImportDialog open onClose={vi.fn()} />)
    await user.upload(screen.getByLabelText('选择 TXT'), new File(['BAD'], 'bad.txt', { type: 'text/plain' }))
    expect(mocks.validate).toHaveBeenCalled()
    expect(screen.getByRole('button', { name: '确认保存' })).toBeDisabled()
  })
})
