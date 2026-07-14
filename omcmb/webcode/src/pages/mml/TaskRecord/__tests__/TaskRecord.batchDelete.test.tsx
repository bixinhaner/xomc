import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { Modal } from 'antd';
import { IntlProvider } from 'react-intl';
import type { Key, ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import zhCN from '@core/i18n/zh-CN';

const mocks = vi.hoisted(() => ({
  startTasks: vi.fn(),
  cancelTasks: vi.fn(),
  deleteTasks: vi.fn(),
  getTaskResultsPage: vi.fn(),
  refetchTasks: vi.fn(),
  useMMLTasks: vi.fn(),
  taskItems: [] as Array<Record<string, unknown>>,
  taskResultItems: [] as Array<Record<string, unknown>>,
}));

vi.mock('@core/hooks/api/useMML', () => ({
  useMMLTasks: (params: Record<string, unknown>) => {
    mocks.useMMLTasks(params);
    return {
      data: {
        total: mocks.taskItems.length,
        items: mocks.taskItems,
      },
      isLoading: false,
      refetch: mocks.refetchTasks,
    };
  },
  useMMLTaskResults: () => ({
    data: {
      total: mocks.taskResultItems.length,
      items: mocks.taskResultItems,
    },
    isLoading: false,
  }),
  getMMLTaskResultsPage: mocks.getTaskResultsPage,
  useStartMMLTasks: () => ({ mutate: mocks.startTasks, isPending: false }),
  useCancelMMLTasks: () => ({ mutate: mocks.cancelTasks, isPending: false }),
  useDeleteMMLTasks: () => ({ mutate: mocks.deleteTasks, isPending: false }),
}));

vi.mock('@core/utils/saveBlob', () => ({
  saveBlob: vi.fn(),
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
                      const recordId = String(record.id);
                      const nextKeys = event.currentTarget.checked
                        ? Array.from(new Set([...selectedRowKeys.map(String), recordId]))
                        : selectedRowKeys.filter((key) => String(key) !== recordId);
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
import { saveBlob } from '@core/utils/saveBlob';

function buildTask(overrides: Record<string, unknown> = {}) {
  return {
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
    ...overrides,
  };
}

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
    mocks.taskItems = [buildTask()];
    mocks.taskResultItems = [{
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
    }];
    mocks.startTasks.mockImplementation((_ids: string[], options?: { onSuccess?: () => void }) => options?.onSuccess?.());
    mocks.cancelTasks.mockImplementation((_ids: string[], options?: { onSuccess?: () => void }) => options?.onSuccess?.());
    mocks.deleteTasks.mockImplementation((_ids: string[], options?: { onSuccess?: () => void }) => options?.onSuccess?.());
    mocks.getTaskResultsPage.mockResolvedValue({
      total: 1,
      page: 1,
      pageSize: 100,
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
    });
    confirmSpy.mockClear();
  });

  it('keeps console-origin task records visible and shows their command results', () => {
    renderPage();

    expect(screen.getByText('控制台执行')).toBeInTheDocument();

    const viewButton = screen.getByRole('button', { name: '查看' });
    expect(viewButton.textContent?.trim()).toBe('');
    fireEvent.click(viewButton);

    expect(screen.getByText('LST')).toBeInTheDocument();
    expect(screen.getByText('DEVICE_INFO')).toBeInTheDocument();
  });

  it('shows aggregate result and uses executable progress count in task list', () => {
    mocks.taskItems = [
      buildTask({
        id: 'task-partial-1',
        taskName: '脚本任务',
        taskOrigin: 'script',
        executeMode: 'device_bound',
        commandCount: 5,
        planItemCount: 4,
        successCount: 4,
        failedCount: 1,
        result: 'partial',
      }),
    ];

    const { container } = renderPage();

    expect(container.querySelector('td[data-column-key="result"]')?.textContent).toContain('部分成功');
    expect(container.querySelector('td[data-column-key="progress"]')?.textContent).toBe('5/5');
  });

  it('does not render failed device task results as pending', () => {
    mocks.taskResultItems = [{
      deviceTaskId: 'device-task-failed',
      deviceSn: 'SN002',
      deviceName: '基站 B',
      commandCode: 'RMV Device.X.9999.',
      status: 'failed',
      result: {
        success: false,
        rawOutput: '',
        parsedData: null,
        executionTime: 30,
        timestamp: '2026-07-11T00:00:01Z',
      },
      failReason: 'instance does not exist',
    }];

    renderPage();

    fireEvent.click(screen.getByRole('button', { name: '查看' }));

    expect(screen.queryByText('等待中')).not.toBeInTheDocument();
    expect(screen.getAllByText('失败').length).toBeGreaterThanOrEqual(2);
  });

  it('exports the full viewed task result list as CSV', async () => {
    mocks.taskItems = [buildTask({ id: 'task-script-1', taskName: '脚本任务', taskOrigin: 'script' })];
    mocks.getTaskResultsPage.mockResolvedValueOnce({
      total: 2,
      page: 1,
      pageSize: 100,
      items: [
        {
          deviceTaskId: 'device-task-1',
          deviceSn: 'SN,001',
          deviceName: '基站 "A"',
          planLineNo: 21,
          planOrder: 1,
          mmlScript: '=HYPERLINK("http://bad")',
          status: 'completed',
          request: { method: 'SetParameterValues', rawRequest: '<xml attr="1">x</xml>' },
          result: {
            success: true,
            rawOutput: 'line1\nline2',
            parsedData: null,
            executionTime: 30,
            timestamp: '2026-07-11T00:00:01Z',
          },
        },
        {
          deviceTaskId: 'device-task-2',
          deviceSn: 'SN002',
          deviceName: '基站 B',
          commandCode: 'LST Device.DeviceInfo.SoftwareVersion',
          status: 'completed',
          result: {
            success: true,
            rawOutput: 'OK',
            parsedData: null,
            executionTime: 20,
            timestamp: '2026-07-11T00:00:02Z',
          },
        },
      ],
    });

    renderPage();

    fireEvent.click(screen.getByRole('button', { name: '查看' }));
    fireEvent.click(screen.getByRole('button', { name: '导出 CSV' }));

    await waitFor(() => expect(saveBlob).toHaveBeenCalled());
    expect(mocks.getTaskResultsPage).toHaveBeenCalledWith('task-script-1', 1, 100);

    const [content, filename, mime] = vi.mocked(saveBlob).mock.calls[0];
    expect(filename).toMatch(/^mml-task-results-脚本任务-\d{8}-\d{6}\.csv$/);
    expect(mime).toBe('text/csv;charset=utf-8');
    expect(String(content)).toContain('\ufeff');
    expect(String(content)).toContain('"SN,001"');
    expect(String(content)).toContain('"基站 ""A"""');
    expect(String(content)).toContain('"\'=HYPERLINK(""http://bad"")"');
    expect(String(content)).toContain('"line1\nline2"');
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

  it('supports starting a suspended script task from the row actions', () => {
    mocks.taskItems = [
      buildTask({
        id: 'task-suspended-1',
        taskName: '挂起脚本任务',
        taskOrigin: 'script',
        executeType: 'suspended',
        status: 'paused',
      }),
    ];

    renderPage();

    fireEvent.click(screen.getByRole('button', { name: '执行任务' }));

    expect(confirmSpy).toHaveBeenCalledWith(expect.objectContaining({
      content: '确认执行该任务？等待中或已暂停任务会立即开始下发。',
    }));
    expect(mocks.startTasks).toHaveBeenCalledWith(
      ['task-suspended-1'],
      expect.objectContaining({ onSuccess: expect.any(Function), onError: expect.any(Function) }),
    );
  });

  it('supports terminating an active task from the row actions', () => {
    mocks.taskItems = [
      buildTask({
        id: 'task-running-row-1',
        taskName: '执行中脚本任务',
        taskOrigin: 'script',
        status: 'running',
      }),
    ];

    renderPage();

    fireEvent.click(screen.getByRole('button', { name: '终止任务' }));

    expect(confirmSpy).toHaveBeenCalledWith(expect.objectContaining({
      content: '确认终止该任务？等待中、执行中或已暂停任务会变为已终止。',
    }));
    expect(mocks.cancelTasks).toHaveBeenCalledWith(
      ['task-running-row-1'],
      expect.objectContaining({ onSuccess: expect.any(Function), onError: expect.any(Function) }),
    );
  });

  it('supports starting selected pending or paused task records in batch', () => {
    mocks.taskItems = [
      buildTask({ id: 'task-pending-1', taskName: '等待任务', status: 'pending' }),
      buildTask({ id: 'task-paused-1', taskName: '暂停任务', status: 'paused' }),
    ];

    renderPage();

    expect(screen.getByRole('button', { name: /批量执行/ })).toBeDisabled();

    fireEvent.click(screen.getByRole('checkbox', { name: 'select-task-pending-1' }));
    fireEvent.click(screen.getByRole('checkbox', { name: 'select-task-paused-1' }));
    fireEvent.click(screen.getByRole('button', { name: /批量执行/ }));

    expect(confirmSpy).toHaveBeenCalledWith(expect.objectContaining({
      content: '确认执行选中的 2 条任务记录？等待中或已暂停任务会立即开始下发。',
    }));
    expect(mocks.startTasks).toHaveBeenCalledWith(
      ['task-pending-1', 'task-paused-1'],
      expect.objectContaining({ onSuccess: expect.any(Function), onError: expect.any(Function) }),
    );
  });

  it('blocks batch start when a selected task record is already finished', () => {
    mocks.taskItems = [
      buildTask({ id: 'task-paused-1', taskName: '暂停任务', status: 'paused' }),
      buildTask({ id: 'task-completed-1', taskName: '完成任务', status: 'completed' }),
    ];

    renderPage();

    fireEvent.click(screen.getByRole('checkbox', { name: 'select-task-paused-1' }));
    fireEvent.click(screen.getByRole('checkbox', { name: 'select-task-completed-1' }));
    fireEvent.click(screen.getByRole('button', { name: /批量执行/ }));

    expect(confirmSpy).not.toHaveBeenCalled();
    expect(mocks.startTasks).not.toHaveBeenCalled();
  });

  it('supports cancelling selected active task records in batch', () => {
    mocks.taskItems = [
      buildTask({ id: 'task-running-1', taskName: '运行中任务', status: 'running' }),
      buildTask({ id: 'task-paused-1', taskName: '暂停任务', status: 'paused' }),
    ];

    renderPage();

    expect(screen.getByRole('button', { name: /批量终止/ })).toBeDisabled();

    fireEvent.click(screen.getByRole('checkbox', { name: 'select-task-running-1' }));
    fireEvent.click(screen.getByRole('checkbox', { name: 'select-task-paused-1' }));
    fireEvent.click(screen.getByRole('button', { name: /批量终止/ }));

    expect(confirmSpy).toHaveBeenCalledWith(expect.objectContaining({
      content: '确认终止选中的 2 条任务记录？等待中、执行中和已暂停任务会变为已终止。',
    }));
    expect(mocks.cancelTasks).toHaveBeenCalledWith(
      ['task-running-1', 'task-paused-1'],
      expect.objectContaining({ onSuccess: expect.any(Function), onError: expect.any(Function) }),
    );
  });

  it('blocks batch cancel when a selected task record is already finished', () => {
    mocks.taskItems = [
      buildTask({ id: 'task-running-1', taskName: '运行中任务', status: 'running' }),
      buildTask({ id: 'task-completed-1', taskName: '完成任务', status: 'completed' }),
    ];

    renderPage();

    fireEvent.click(screen.getByRole('checkbox', { name: 'select-task-running-1' }));
    fireEvent.click(screen.getByRole('checkbox', { name: 'select-task-completed-1' }));
    fireEvent.click(screen.getByRole('button', { name: /批量终止/ }));

    expect(confirmSpy).not.toHaveBeenCalled();
    expect(mocks.cancelTasks).not.toHaveBeenCalled();
  });

  it('blocks batch delete when a selected task record is still running', () => {
    mocks.taskItems = [
      buildTask(),
      buildTask({
        id: 'task-running-1',
        taskName: '运行中任务',
        status: 'running',
      }),
    ];

    renderPage();

    fireEvent.click(screen.getByRole('checkbox', { name: 'select-task-console-1' }));
    fireEvent.click(screen.getByRole('checkbox', { name: 'select-task-running-1' }));
    fireEvent.click(screen.getByRole('button', { name: /批量删除/ }));

    expect(confirmSpy).not.toHaveBeenCalled();
    expect(mocks.deleteTasks).not.toHaveBeenCalled();
  });

  it('passes console task origin through to the task list query', () => {
    renderPage();

    fireEvent.click(screen.getByRole('button', { name: 'search-console' }));

    expect(mocks.useMMLTasks).toHaveBeenLastCalledWith(expect.objectContaining({
      taskOrigin: 'console',
    }));
  });
});
