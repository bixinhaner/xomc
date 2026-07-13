import { fireEvent, render, screen } from '@testing-library/react';
import { Modal } from 'antd';
import { IntlProvider } from 'react-intl';
import type { Key, ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import zhCN from '@core/i18n/zh-CN';

const mocks = vi.hoisted(() => ({
  deleteTasks: vi.fn(),
  refetchTasks: vi.fn(),
  useMMLTasks: vi.fn(),
}));

vi.mock('@core/hooks/api/useMML', () => ({
  useMMLTasks: (params: Record<string, unknown>) => {
    mocks.useMMLTasks(params);
    return {
    data: {
      total: 1,
      items: [{
        id: 'task-console-1',
        taskName: '控制台任务',
        creator: 'admin',
        taskOrigin: 'console',
        executeType: 'immediate',
        status: 'completed',
        result: 'success',
        totalDevices: 1,
        successCount: 1,
        failedCount: 0,
        createdAt: '2026-07-11T00:00:00Z',
        updatedAt: '2026-07-11T00:00:00Z',
        startedAt: '2026-07-11T00:00:00Z',
        finishedAt: '2026-07-11T00:00:01Z',
      }],
    },
    isLoading: false,
    refetch: mocks.refetchTasks,
    };
  },
  useMMLTaskResults: () => ({
    data: {
      total: 1,
      items: [{
        deviceTaskId: 'device-task-1',
        deviceSn: 'SN001',
        deviceName: '基站 A',
        commandCode: 'LST DEVICE_INFO',
        status: 'completed',
        result: {
          success: true,
          rawOutput: '<cwmp:GetParameterValuesResponse />',
          parsedData: null,
          executionTime: 30,
          timestamp: '2026-07-11T00:00:01Z',
        },
      }],
    },
    isLoading: false,
  }),
  useDeleteMMLTasks: () => ({ mutate: mocks.deleteTasks, isPending: false }),
}));

vi.mock('@/components/FilterBar', () => ({
  default: ({ onSearch, onReset }: {
    onSearch: (values: Record<string, unknown>) => void;
    onReset: () => void;
  }) => (
    <div data-testid="filter-bar">
      <button onClick={() => onSearch({ taskOrigin: 'console' })} type="button">search-console</button>
      <button onClick={() => onSearch({ taskOrigin: 'script' })} type="button">search-script</button>
      <button onClick={onReset} type="button">reset</button>
    </div>
  ),
}));

vi.mock('@/components/DataTable', () => ({
  default: ({
    batchActions = [],
    columns,
    dataSource,
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
    onSelectionChange?: (keys: Key[], rows: Array<Record<string, unknown>>) => void;
    selectedRowKeys?: Key[];
    selectable?: boolean;
  }) => (
    <div>
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

import TaskRecord from '..';

function renderPage() {
  return render(
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      <TaskRecord />
    </IntlProvider>,
  );
}

describe('TaskRecord batch delete and console task display', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.deleteTasks.mockImplementation((_ids: string[], options?: { onSuccess?: () => void }) => options?.onSuccess?.());
    confirmSpy.mockClear();
  });

  it('keeps console-origin task records visible and shows their command results', () => {
    renderPage();

    expect(screen.getByText('控制台执行')).toBeInTheDocument();

    const viewButton = screen.getByRole('button', { name: '查看' });
    fireEvent.click(viewButton);

    expect(screen.getByText('LST')).toBeInTheDocument();
    expect(screen.getByText('DEVICE_INFO')).toBeInTheDocument();
  });

  it('supports deleting selected task records in batch', () => {
    renderPage();

    expect(screen.getByRole('button', { name: /批量删除/ })).toBeDisabled();

    fireEvent.click(screen.getByRole('checkbox', { name: 'select-task-console-1' }));
    fireEvent.click(screen.getByRole('button', { name: /批量删除/ }));

    expect(confirmSpy).toHaveBeenCalledWith(expect.objectContaining({
      content: '确认删除选中的 1 条任务记录？该操作会删除对应执行结果，且不可撤销。',
    }));
    expect(mocks.deleteTasks).toHaveBeenCalledWith(
      ['task-console-1'],
      expect.objectContaining({ onSuccess: expect.any(Function), onError: expect.any(Function) }),
    );
  });

  it('passes console task origin through to the task list query', () => {
    renderPage();

    fireEvent.click(screen.getByRole('button', { name: 'search-console' }));

    expect(mocks.useMMLTasks).toHaveBeenLastCalledWith(expect.objectContaining({
      taskOrigin: 'console',
    }));
  });
});
