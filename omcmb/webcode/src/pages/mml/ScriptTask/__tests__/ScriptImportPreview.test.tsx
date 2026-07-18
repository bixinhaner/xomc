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

  it('shows executable MML text for raw-path plan commands', async () => {
    renderPreview(<ScriptImportPreview validation={{
      originalFilename: 'raw-path.txt',
      planItems: [
        {
          lineNo: 2,
          deviceSn: 'SN001',
          order: 1,
          rawLine: 'LST Device.FAP.Ipsec.;SN001',
          command: {
            commandCode: 'RAW LST',
            operationType: 'LST',
            paramPaths: ['Device.FAP.Ipsec.'],
            parameters: {},
          },
        },
        {
          lineNo: 3,
          deviceSn: 'SN001',
          order: 2,
          rawLine: 'ADD Device.FAP.Ipsec.:TUNNEL_ENABLE=false,TUNNEL_GATEWAY=192.0.2.2;SN001',
          command: {
            commandCode: 'RAW ADD',
            operationType: 'ADD',
            paramPaths: ['Device.FAP.Ipsec.'],
            parameters: { TUNNEL_ENABLE: false, TUNNEL_GATEWAY: '192.0.2.2' },
          },
        },
      ],
      summary: { totalLines: 2, validLines: 2, effectiveLines: 2, deviceCount: 1, errorCount: 0, warningCount: 0 },
      issues: [],
    }} />);

    expect(screen.queryByText('RAW LST')).not.toBeInTheDocument();
    expect(screen.queryByText('RAW ADD')).not.toBeInTheDocument();
    expect(screen.getAllByText('LST')[0]).toBeInTheDocument();
    expect(screen.getAllByText('Device.FAP.Ipsec.')).toHaveLength(2);
    expect(screen.getByText('参数 2 项')).toBeInTheDocument();

    await userEvent.click(screen.getByRole('button', { name: '参数 2 项' }));
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByText('匹配 2 / 2 项')).toBeInTheDocument();
    expect(screen.getByText('TUNNEL_ENABLE')).toBeInTheDocument();
    expect(screen.getByText('TUNNEL_GATEWAY')).toBeInTheDocument();
  });

  it('opens a searchable parameter dialog for large command parameter sets', async () => {
    const params = Object.fromEntries(
      Array.from({ length: 16 }, (_, index) => [`PARAM_${String(index + 1).padStart(2, '0')}`, `value-${index + 1}`]),
    );
    const paramText = Object.entries(params).map(([key, value]) => `${key}=${value}`).join(',');

    renderPreview(<ScriptImportPreview validation={{
      originalFilename: 'large-params.txt',
      planItems: [{
        lineNo: 2,
        deviceSn: 'SN001',
        order: 1,
        rawLine: `ADD Device.FAP.Ipsec.:${paramText};SN001`,
        command: {
          commandCode: 'RAW ADD',
          operationType: 'ADD',
          paramPaths: ['Device.FAP.Ipsec.'],
          parameters: params,
        },
      }],
      summary: { totalLines: 1, validLines: 1, effectiveLines: 1, deviceCount: 1, errorCount: 0, warningCount: 0 },
      issues: [],
    }} />);

    await userEvent.click(screen.getByRole('button', { name: '参数 16 项' }));

    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByText('匹配 16 / 16 项')).toBeInTheDocument();
    await userEvent.type(screen.getByPlaceholderText('搜索参数名或参数值'), 'PARAM_15');
    expect(screen.getByText('匹配 1 / 16 项')).toBeInTheDocument();
    expect(screen.getByText('PARAM_15')).toBeInTheDocument();
    expect(screen.queryByText('PARAM_01')).not.toBeInTheDocument();
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
