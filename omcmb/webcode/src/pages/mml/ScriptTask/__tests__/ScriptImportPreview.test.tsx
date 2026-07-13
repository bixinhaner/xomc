import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { ReactElement } from 'react';
import { IntlProvider } from 'react-intl';
import { afterEach, describe, expect, it, vi } from 'vitest';
import zhCN from '@core/i18n/zh-CN';
import ScriptImportPreview from '../ScriptImportPreview';

function renderPreview(ui: ReactElement) {
  return render(
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      {ui}
    </IntlProvider>,
  );
}

describe('ScriptImportPreview', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('filters the visible issue card with the selected severity', async () => {
    renderPreview(<ScriptImportPreview validation={{
      originalFilename: 'issues.txt',
      planItems: [],
      summary: { totalLines: 2, validLines: 1, effectiveLines: 1, deviceCount: 1, errorCount: 1, warningCount: 1 },
      issues: [
        { code: 'MML_LINE_FORMAT_INVALID', severity: 'error', lineNo: 1, displayMessage: '命令格式不正确' },
        { code: 'MML_DEVICE_OFFLINE', severity: 'warning', lineNo: 2, displayMessage: '设备当前离线' },
      ],
    }} />);
    expect(screen.getByText(/设备当前离线/)).toBeInTheDocument();
    expect(screen.queryByText(/MML_DEVICE_OFFLINE/)).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: '仅看错误' }));
    expect(screen.getByText(/命令格式不正确/)).toBeInTheDocument();
    expect(screen.queryByText(/设备当前离线/)).not.toBeInTheDocument();
  });

  it('shows friendly issue descriptions in the page and downloaded report', async () => {
    let reportBlob: Blob | null = null;
    Object.defineProperty(URL, 'createObjectURL', {
      configurable: true,
      value: vi.fn((blob: Blob) => {
        reportBlob = blob;
        return 'blob:mml-report';
      }),
    });
    Object.defineProperty(URL, 'revokeObjectURL', {
      configurable: true,
      value: vi.fn(),
    });

    renderPreview(<ScriptImportPreview validation={{
      originalFilename: 'issues.txt',
      planItems: [],
      summary: { totalLines: 1, validLines: 0, effectiveLines: 0, deviceCount: 0, errorCount: 1, warningCount: 0 },
      issues: [
        {
          code: 'MML_DEVICE_SN_REQUIRED',
          severity: 'error',
          lineNo: 21,
          message: 'command must end with ;SN',
          displayMessage: '命令末尾必须使用 ;设备SN',
        },
      ],
    }} />);

    expect(screen.getByText('第 21 行：命令末尾必须使用 ;设备SN')).toBeInTheDocument();
    expect(screen.queryByText(/MML_DEVICE_SN_REQUIRED/)).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole('button', { name: '下载错误报告' }));

    expect(reportBlob).not.toBeNull();
    const reportText = await reportBlob!.text();
    expect(reportText).toContain('第 21 行：命令末尾必须使用 ;设备SN');
    expect(reportText).not.toContain('MML_DEVICE_SN_REQUIRED');
  });
});
