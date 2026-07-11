import { screen, render, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Modal } from 'antd';
import { IntlProvider } from 'react-intl';
import type { ComponentProps } from 'react';
import { afterEach, describe, expect, it, vi, beforeEach } from 'vitest';
import zhCN from '@core/i18n/zh-CN';

const mocks = vi.hoisted(() => ({
  validate: vi.fn(),
  create: vi.fn(),
  replace: vi.fn(),
  template: vi.fn(),
  validateReplacement: vi.fn(),
}));

vi.mock('@core/hooks/api/useMML', () => ({
  useValidateMMLScriptImport: () => ({ mutateAsync: mocks.validate, isPending: false }),
  useCreateImportedMMLScript: () => ({ mutateAsync: mocks.create, isPending: false }),
  useReplaceImportedMMLScript: () => ({ mutateAsync: mocks.replace, isPending: false }),
}));

vi.mock('@core/services/api/mmlApi', () => ({
  mmlApi: {
    downloadScriptImportTemplate: mocks.template,
    validateScriptReplacement: mocks.validateReplacement,
  },
  MMLScriptImportApiError: class MMLScriptImportApiError extends Error {},
}));

import ScriptImportModal from '../ScriptImportModal';

const validation = (overrides: Record<string, unknown> = {}) => ({
  validationToken: 'token-1',
  originalFilename: 'script.txt',
  normalizedContent: 'LST DEVICE_INFO;SN1',
  contentSha256: 'sha',
  validationVersion: 'v1',
  validatedAt: '2026-07-10T00:00:00Z',
  planItems: [{
    lineNo: 1,
    deviceSn: 'SN1',
    order: 1,
    rawLine: 'LST DEVICE_INFO;SN1',
    command: { commandCode: 'LST DEVICE_INFO' },
  }],
  summary: { totalLines: 1, validLines: 1, effectiveLines: 1, deviceCount: 1, errorCount: 0, warningCount: 0 },
  issues: [],
  ...overrides,
});

function renderModal(props: Partial<ComponentProps<typeof ScriptImportModal>> = {}) {
  return render(
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      <ScriptImportModal open onClose={vi.fn()} {...props} />
    </IntlProvider>,
  );
}

describe('ScriptImportModal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.create.mockResolvedValue({ id: 'script-1' });
    mocks.replace.mockResolvedValue({ id: 'script-1' });
    mocks.template.mockResolvedValue({ blob: new Blob(['template']), filename: 'MMLTemplate.txt' });
    mocks.validateReplacement.mockResolvedValue(validation());
  });
  afterEach(() => {
    vi.useRealTimers();
    Modal.destroyAll();
  });

  it('leaves the script name empty for new TXT imports so users name the script themselves', () => {
    renderModal();

    const input = screen.getByLabelText('脚本名称');
    expect(input).toHaveValue('');

    fireEvent.change(input, { target: { value: '人工修改后的脚本名' } });
    expect(input).toHaveValue('人工修改后的脚本名');
  });

  it('keeps the existing script name when replacing an imported script', () => {
    renderModal({
      script: {
        id: 'script-1',
        scriptName: '巡检脚本',
        description: '已有描述',
        content: '',
        creator: 'admin',
        createTime: '2026-07-10T00:00:00Z',
        updateTime: '2026-07-10T00:00:00Z',
        tags: [],
        status: 'active',
        type: 'batch',
        progress: 0,
      },
    });

    expect(screen.getByLabelText('脚本名称')).toHaveValue('巡检脚本');
    expect(screen.getByLabelText('描述')).toHaveValue('已有描述');
  });

  it('keeps save disabled when server validation has errors', async () => {
    mocks.validate.mockResolvedValue(validation({
      summary: { totalLines: 1, validLines: 0, effectiveLines: 0, deviceCount: 0, errorCount: 1, warningCount: 0 },
      issues: [{ code: 'MML_LINE_FORMAT_INVALID', severity: 'error', lineNo: 1, rawLine: 'BAD' }],
    }));
    renderModal();

    await userEvent.upload(screen.getByLabelText('选择 TXT'), new File(['BAD'], 'bad.txt', { type: 'text/plain' }));

    expect(await screen.findByText('MML_LINE_FORMAT_INVALID')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '确认保存' })).toBeDisabled();
    expect(screen.queryByRole('textbox', { name: '脚本内容' })).not.toBeInTheDocument();
  });

  it('allows saving warnings only after an explicit confirmation', async () => {
    mocks.validate.mockResolvedValue(validation({
      summary: { totalLines: 1, validLines: 1, effectiveLines: 1, deviceCount: 1, errorCount: 0, warningCount: 1 },
      issues: [{ code: 'MML_DEVICE_OFFLINE', severity: 'warning', lineNo: 1, rawLine: 'LST DEVICE_INFO;SN1' }],
    }));
    renderModal();
    const user = userEvent.setup();
    await user.upload(screen.getByLabelText('选择 TXT'), new File(['LST DEVICE_INFO;SN1'], 'script.txt', { type: 'text/plain' }));
    await user.type(screen.getByLabelText('脚本名称'), 'warning-script');
    await user.click(screen.getByRole('button', { name: '确认保存' }));
    expect(await screen.findByRole('button', { name: '继续保存' })).toBeInTheDocument();
    expect(mocks.create).not.toHaveBeenCalled();
    await user.click(screen.getByRole('button', { name: '继续保存' }));
    await waitFor(() => expect(mocks.create).toHaveBeenCalledWith(expect.objectContaining({ validationToken: 'token-1' })));
    await waitFor(() => expect(screen.queryByRole('button', { name: '继续保存' })).not.toBeInTheDocument());
  });

  it('replaces the preview after a second upload', async () => {
    mocks.validate.mockResolvedValueOnce(validation({ originalFilename: 'first.txt' }))
      .mockResolvedValueOnce(validation({ originalFilename: 'second.txt', planItems: [] }));
    renderModal();
    const input = screen.getByLabelText('选择 TXT');
    await userEvent.upload(input, new File(['ONE'], 'first.txt', { type: 'text/plain' }));
    expect(await screen.findByText('first.txt')).toBeInTheDocument();
    await userEvent.upload(input, new File(['TWO'], 'second.txt', { type: 'text/plain' }));
    expect(await screen.findByText('second.txt')).toBeInTheDocument();
    expect(screen.queryByText('first.txt')).not.toBeInTheDocument();
  });

  it('clears the previous token and preview when a replacement upload fails', async () => {
    mocks.validate.mockResolvedValueOnce(validation({ originalFilename: 'first.txt' }))
      .mockRejectedValueOnce(new Error('bad replacement'));
    renderModal();
    const input = screen.getByLabelText('选择 TXT');
    await userEvent.upload(input, new File(['ONE'], 'first.txt', { type: 'text/plain' }));
    expect(await screen.findByText('first.txt')).toBeInTheDocument();
    await userEvent.upload(input, new File(['BAD'], 'bad.txt', { type: 'text/plain' }));
    await waitFor(() => expect(screen.getByRole('button', { name: '确认保存' })).toBeDisabled());
    expect(screen.queryByText('first.txt')).not.toBeInTheDocument();
  });

});
