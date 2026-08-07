import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import ResultTable from './ResultTable';
import type { ResultRow } from '../types';

vi.mock('@/hooks/useT', () => ({
  useT: () => (id: string) => id,
}));

vi.mock('@core/hooks/api/useMmlConsole', () => ({
  useExportTaskCSV: () => ({ mutate: vi.fn(), isPending: false }),
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
