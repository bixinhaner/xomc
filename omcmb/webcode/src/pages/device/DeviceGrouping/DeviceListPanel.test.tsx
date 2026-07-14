import React from 'react';
import { render } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import DeviceListPanel from './DeviceListPanel';

let lastDataTableProps: Record<string, unknown> | null = null;

vi.mock('@/components/DataTable', () => ({
  default: (props: Record<string, unknown>) => {
    lastDataTableProps = props;
    return <div data-testid="device-grouping-table" />;
  },
}));

vi.mock('./BatchImportModal', () => ({
  default: () => null,
}));

vi.mock('./BatchPreRegisterModal', () => ({
  default: () => null,
}));

const t = (id: string) => id;

describe('DeviceListPanel', () => {
  it('lets DataTable calculate body height so pageSize=100 rows remain scrollable', () => {
    render(
      <DeviceListPanel
        devices={[]}
        total={100}
        loading={false}
        selectedDeviceIds={[]}
        currentPage={1}
        pageSize={100}
        selectedGroupId={null}
        selectedGroupName="Group A"
        batchActions={[]}
        onSelectionChange={() => undefined}
        onPageChange={() => undefined}
        onSearch={() => undefined}
        onExport={() => undefined}
        onImport={() => undefined}
        onDownloadTemplate={() => undefined}
        onPreRegister={() => undefined}
        onDownloadPreRegisterTemplate={() => undefined}
        t={t}
      />,
    );

    expect(lastDataTableProps?.scroll).toEqual({ x: 'max-content' });
    expect(lastDataTableProps?.autoFitHeight).toBe(true);
  });
});
