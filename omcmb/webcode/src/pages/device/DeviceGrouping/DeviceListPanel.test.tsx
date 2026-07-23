import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
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
  it('exposes a page-level refresh action in the table toolbar', () => {
    const onRefresh = vi.fn();

    render(
      <DeviceListPanel
        devices={[]}
        total={0}
        loading={false}
        refreshing={false}
        selectedDeviceIds={[]}
        currentPage={1}
        pageSize={20}
        selectedGroupId={null}
        selectedGroupName="Group A"
        batchActions={[]}
        onSelectionChange={() => undefined}
        onPageChange={() => undefined}
        onSearch={() => undefined}
        onRefresh={onRefresh}
        onExport={() => undefined}
        onImport={() => undefined}
        onDownloadTemplate={() => undefined}
        onPreRegister={() => undefined}
        onDownloadPreRegisterTemplate={() => undefined}
        t={t}
      />,
    );

    const toolbar = lastDataTableProps?.extraToolbarAfterBatch as React.ReactNode;
    render(<>{toolbar}</>);
    fireEvent.click(screen.getByText('common.refresh'));

    expect(onRefresh).toHaveBeenCalledTimes(1);
  });

  it('lets DataTable calculate body height so pageSize=100 rows remain scrollable', () => {
    render(
      <DeviceListPanel
        devices={[]}
        total={100}
        loading={false}
        refreshing={false}
        selectedDeviceIds={[]}
        currentPage={1}
        pageSize={100}
        selectedGroupId={null}
        selectedGroupName="Group A"
        batchActions={[]}
        onSelectionChange={() => undefined}
        onPageChange={() => undefined}
        onSearch={() => undefined}
        onRefresh={() => undefined}
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
