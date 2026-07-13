import { fireEvent, render, screen } from '@testing-library/react';
import { Modal } from 'antd';
import { IntlProvider } from 'react-intl';
import type { Key, ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import zhCN from '@core/i18n/zh-CN';

const mocks = vi.hoisted(() => ({
  refetch: vi.fn(),
  update: vi.fn(),
  delete: vi.fn(),
}));

vi.mock('@core/hooks/api/useMML', () => ({
  useMMLScripts: () => ({
    data: {
      total: 1,
      items: [{
        id: 'script-1',
        scriptName: '巡检脚本',
        description: 'daily check',
        content: 'LST DEVICE_INFO;SN1',
        creator: 'admin',
        createTime: '2026-07-10T00:00:00Z',
        updateTime: '2026-07-10T00:00:00Z',
        originalFilename: 'script.txt',
        status: 'script-status-active',
        tags: [],
        type: 'batch',
        progress: 0,
      }],
    },
    isLoading: false,
    refetch: mocks.refetch,
  }),
  useMMLScriptById: () => ({ data: undefined, isFetching: false }),
  useUpdateMMLScript: () => ({ mutate: mocks.update, isPending: false }),
  useDeleteMMLScripts: () => ({ mutate: mocks.delete, isPending: false }),
}));

vi.mock('../ScriptImportModal', () => ({ default: () => null }));
vi.mock('../ScriptExecutionDrawer', () => ({
  default: ({ open, script }: { open: boolean; script?: { scriptName?: string } | null }) =>
    open ? <div role="dialog">执行 {script?.scriptName}</div> : null,
}));
vi.mock('../ScriptImportPreview', () => ({ default: () => <div /> }));
vi.mock('@/components/DataTable', () => ({
  default: ({
    batchActions = [],
    columns,
    dataSource,
    hideToolbar,
    onSelectionChange,
    selectedRowKeys = [],
    selectable,
  }: {
    batchActions?: Array<{
      key: string;
      label: string;
      disabled?: boolean;
      onClick: (keys: Key[]) => void;
    }>;
    columns: Array<{
      key: string;
      dataIndex?: string;
      fixed?: 'left' | 'right';
      render?: (value: unknown, record: Record<string, unknown>, index: number) => ReactNode;
    }>;
    dataSource: Array<Record<string, unknown>>;
    hideToolbar?: boolean;
    onSelectionChange?: (keys: Key[], rows: Array<Record<string, unknown>>) => void;
    selectedRowKeys?: Key[];
    selectable?: boolean;
  }) => (
    <div>
      {!hideToolbar ? <div data-testid="datatable-toolbar" /> : null}
      {batchActions.map((action) => (
        <button
          key={action.key}
          disabled={selectedRowKeys.length === 0 || action.disabled}
          onClick={() => action.onClick(selectedRowKeys)}
          type="button"
        >
          {action.label}
        </button>
      ))}
      <table>
        <tbody>
          {dataSource.map((record, rowIndex) => (
            <tr key={String(record.id)}>
              {selectable ? (
                <td>
                  <input
                    aria-label={`select-${String(record.id)}`}
                    checked={selectedRowKeys.includes(String(record.id))}
                    onChange={(event) => {
                      const nextKeys = event.currentTarget.checked ? [String(record.id)] : [];
                      onSelectionChange?.(nextKeys, dataSource.filter((item) => nextKeys.includes(String(item.id))));
                    }}
                    type="checkbox"
                  />
                </td>
              ) : null}
              {columns.map((column) => (
                <td key={column.key} data-column-key={column.key} data-fixed={column.fixed ?? ''}>
                  {column.render
                    ? column.render(column.dataIndex ? record[column.dataIndex] : undefined, record, rowIndex)
                    : String(column.dataIndex ? record[column.dataIndex] ?? '' : '')}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  ),
}));

const confirmSpy = vi.spyOn(Modal, 'confirm').mockImplementation((config) => {
  void config.onOk?.(() => undefined);
  return {
    destroy: vi.fn(),
    update: vi.fn(),
  } as ReturnType<typeof Modal.confirm>;
});

beforeEach(() => {
  vi.clearAllMocks();
  mocks.delete.mockImplementation((_ids: string[], options?: { onSuccess?: () => void }) => options?.onSuccess?.());
  confirmSpy.mockClear();
});

import ScriptTask from '..';

function renderPage() {
  return render(
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      <ScriptTask />
    </IntlProvider>,
  );
}

describe('ScriptTask actions column', () => {
  it('keeps only execute visible and moves secondary script actions into the more menu', async () => {
    renderPage();

    const executeButton = screen.getByRole('button', { name: '执行' });
    expect(executeButton).toBeInTheDocument();
    expect(executeButton.textContent?.trim()).toBe('');
    expect(screen.queryByRole('button', { name: '查看' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '重新导入' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /下载 TXT/ })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '编辑' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '删除' })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: '更多操作' }));

    expect(await screen.findByText('查看详情')).toBeInTheDocument();
    expect(screen.getByText('重新导入')).toBeInTheDocument();
    expect(screen.getByText('下载 TXT')).toBeInTheDocument();
    expect(screen.getByText('编辑')).toBeInTheDocument();
    expect(screen.getByText('删除')).toBeInTheDocument();
  });

  it('supports selecting scripts and deleting them in batch', () => {
    renderPage();

    expect(screen.queryByTestId('datatable-toolbar')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: /批量删除/ })).toBeDisabled();

    fireEvent.click(screen.getByRole('checkbox', { name: 'select-script-1' }));
    fireEvent.click(screen.getByRole('button', { name: /批量删除/ }));

    expect(confirmSpy).toHaveBeenCalledWith(expect.objectContaining({
      content: '确认删除选中的 1 个脚本？此操作不可撤销。',
    }));
    expect(mocks.delete).toHaveBeenCalledWith(
      ['script-1'],
      expect.objectContaining({ onSuccess: expect.any(Function), onError: expect.any(Function) }),
    );
  });

  it('omits the script library status column because it is not useful on the script task list', () => {
    renderPage();

    expect(screen.queryByText('script-status-active')).not.toBeInTheDocument();
  });

  it('keeps the operation column in normal table flow instead of a fixed dark sticky rail', () => {
    renderPage();

    const operationCell = screen.getByRole('button', { name: '执行' }).closest('td');
    expect(operationCell).toHaveAttribute('data-column-key', 'operation');
    expect(operationCell).toHaveAttribute('data-fixed', '');
  });

  it('keeps the operation column as the first column', () => {
    renderPage();

    expect(document.querySelector('tbody tr td:nth-child(2)')).toHaveAttribute('data-column-key', 'operation');
  });
});
