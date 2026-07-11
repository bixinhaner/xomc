import { fireEvent, render, screen } from '@testing-library/react';
import { IntlProvider } from 'react-intl';
import type { ReactNode } from 'react';
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
        status: 'active',
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
  default: ({ columns, dataSource }: {
    columns: Array<{
      key: string;
      dataIndex?: string;
      render?: (value: unknown, record: Record<string, unknown>, index: number) => ReactNode;
    }>;
    dataSource: Array<Record<string, unknown>>;
  }) => (
    <table>
      <tbody>
        {dataSource.map((record, rowIndex) => (
          <tr key={String(record.id)}>
            {columns.map((column) => (
              <td key={column.key}>
                {column.render
                  ? column.render(column.dataIndex ? record[column.dataIndex] : undefined, record, rowIndex)
                  : String(column.dataIndex ? record[column.dataIndex] ?? '' : '')}
              </td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  ),
}));

import ScriptTask from '..';

function renderPage() {
  return render(
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      <ScriptTask />
    </IntlProvider>,
  );
}

describe('ScriptTask actions column', () => {
  beforeEach(() => vi.clearAllMocks());

  it('keeps only execute visible and moves secondary script actions into the more menu', async () => {
    renderPage();

    expect(screen.getByRole('button', { name: '执行' })).toBeInTheDocument();
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
});
