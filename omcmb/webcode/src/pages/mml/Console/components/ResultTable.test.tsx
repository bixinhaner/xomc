import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import ResultTable from './ResultTable';
import type { ResultRow } from '../types';

vi.mock('@/hooks/useT', () => ({
  useT: () => (id: string) => id,
}));

const exportAllMutate = vi.fn();

vi.mock('@core/hooks/api/useMmlConsole', () => ({
  useExportTaskCSV: () => ({ mutate: exportAllMutate, isPending: false }),
  useExportTaskDeviceCSV: () => ({ mutate: vi.fn(), isPending: false }),
}));

vi.mock('@core/hooks/usePermission', () => ({
  usePermission: () => true,
}));

vi.mock('./ResultDetailModal', () => ({
  default: () => null,
}));

const row: ResultRow = {
  planLineNo: 12,
  planOrder: 3,
  planRawLine: '12,device-001,LST PERF_MGMT_CONFIG',
  commandCode: 'LST PERF_MGMT_CONFIG',
  deviceSn: 'device-001',
  deviceTaskId: 'task-001',
  status: 'success',
  cells: { 'Device.Test.Value': 'enabled' },
  raw: 'raw response',
  elapsedMs: 10,
  dispatchedAt: '10:00:00',
  respondedAt: '10:00:01',
};

describe('ResultTable', () => {
  it('downloads every command record instead of only the selected command', () => {
    render(
      <ResultTable
        execMeta={{ operationType: 'LST', read: true, label: 'LST PERF_MGMT_CONFIG' }}
        commandId="command-latest"
        commandIds={['command-latest', 'command-middle', 'command-oldest']}
        columns={[{ key: 'value', label: 'Value', path: 'Device.Test.Value' }]}
        rows={[row]}
        running={false}
        hasExecuted
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: /mml\.consoleV2\.result\.downloadAll/ }));
    expect(exportAllMutate).toHaveBeenCalledWith(
      ['command-latest', 'command-middle', 'command-oldest'],
      expect.any(Object),
    );
  });

  it('keeps only SN and action fixed while plan command scrolls with result columns', () => {
    const { container } = render(
      <ResultTable
        execMeta={{ operationType: 'LST', read: true, label: 'LST PERF_MGMT_CONFIG' }}
        commandId="command-001"
        columns={[{ key: 'value', label: 'Value', path: 'Device.Test.Value' }]}
        rows={[row]}
        running={false}
        hasExecuted
        onReexecute={vi.fn()}
      />,
    );

    const headers = Array.from(container.querySelectorAll('thead th'));
    const headerTexts = headers.map((header) => header.textContent?.trim());
    expect(headerTexts).toEqual([
      'mml.consoleV2.result.col.deviceSn',
      'mml.consoleV2.result.col.action',
      'mml.consoleV2.result.col.status',
      'mml.planCommand',
      'Value',
      'mml.consoleV2.result.col.dispatchedAt',
      'mml.consoleV2.result.col.respondedAt',
    ]);
    expect(screen.queryByText('mml.planLine')).not.toBeInTheDocument();

    expect(headers[0]).toHaveClass('ant-table-cell-fix');
    expect(headers[1]).toHaveClass('ant-table-cell-fix');
    expect(headers.slice(2).every((header) => !header.classList.contains('ant-table-cell-fix'))).toBe(true);
    expect(container.querySelectorAll('tbody button')).toHaveLength(3);
  });

  it('uses one fixed numeric table width so sticky header and body columns cannot diverge', () => {
    const { container } = render(
      <ResultTable
        execMeta={{ operationType: 'LST', read: true, label: 'LST PERF_MGMT_CONFIG' }}
        commandId="command-wide"
        columns={[
          { key: 'value-1', label: 'Very Long Parameter Header 1', path: 'Device.Test.Value1' },
          { key: 'value-2', label: 'Very Long Parameter Header 2', path: 'Device.Test.Value2' },
        ]}
        rows={[{
          ...row,
          cells: {
            'Device.Test.Value1': '1',
            'Device.Test.Value2': '2',
          },
        }]}
        running={false}
        hasExecuted
      />,
    );

    const tables = Array.from(container.querySelectorAll('.ant-table table')) as HTMLTableElement[];
    expect(tables.length).toBeGreaterThan(0);
    // 4 个基础列 570 + 2 个动态列 280 + 2 个时间列 208 = 1058px。
    expect(tables.every((table) => table.style.width === '1058px')).toBe(true);
    expect(tables.every((table) => table.style.tableLayout === 'fixed')).toBe(true);
  });

  it('shows only the first 80 oversized object result columns and prompts for full download', () => {
    const paths = Array.from(
      { length: 100 },
      (_value, index) => `DeviceGSM.Bts.${index + 1}.Band`,
    );
    const { container } = render(
      <ResultTable
        execMeta={{ operationType: 'LST', read: true, label: 'LST BTS' }}
        commandId="command-large-object"
        columns={[{ key: 'bts', label: 'BTS', path: 'DeviceGSM.Bts.' }]}
        rows={[{
          ...row,
          cells: Object.fromEntries(paths.map((path) => [path, 'GSM900'])),
        }]}
        running={false}
        hasExecuted
      />,
    );

    const headers = Array.from(container.querySelectorAll('thead th'))
      .map((header) => header.textContent?.trim());
    expect(headers).toHaveLength(86); // 4 base + 80 params + 2 timestamps
    expect(headers).toContain('Band [Bts.1]');
    expect(headers).toContain('Band [Bts.80]');
    expect(headers).not.toContain('Band [Bts.81]');
    expect(container).toHaveTextContent('mml.consoleV2.result.columnLimitHint');
    expect(screen.getByRole('button', {
      name: /mml\.consoleV2\.result\.downloadFullResult/,
    })).toBeEnabled();
    expect(container.querySelector('.ant-pagination-item-2')).toBeNull();
  });

  it('falls back to the selected command metadata when a MOD readback row has no command fields', () => {
    const { container } = render(
      <ResultTable
        execMeta={{
          operationType: 'MOD',
          read: false,
          label: 'MOD DEVICE_INFO',
          commandName: '修改 设备基本信息',
        }}
        commandId="command-mod"
        columns={[{ key: 'value', label: 'Value', path: 'Device.Test.Value' }]}
        rows={[{ ...row, planLineNo: undefined, planOrder: undefined, planRawLine: undefined, commandCode: undefined, commandName: undefined }]}
        running={false}
        hasExecuted
      />,
    );

    expect(container).toHaveTextContent('修改 设备基本信息');
  });
});
