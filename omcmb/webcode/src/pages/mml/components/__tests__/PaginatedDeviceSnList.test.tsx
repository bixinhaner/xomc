import { fireEvent, render, screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import PaginatedDeviceSnList from '../PaginatedDeviceSnList';

const messages: Record<string, string> = {
  'mml.selectedDeviceSummary': '已选 {count}/{max} 台设备',
  'mml.clearSelectedDevices': '清空已选设备',
  'mml.removeSelectedDevice': '移除设备 {sn}',
  'mml.deviceRange': '第 {start}–{end} 台，共 {total} 台',
  'mml.taskDeviceLimitExceeded': '设备数量 {current} 超过单任务上限 {max} 台。',
};

const t = (id: string, values?: Record<string, string | number>) =>
  Object.entries(values ?? {}).reduce(
    (text, [key, value]) => text.replace(`{${key}}`, String(value)),
    messages[id] ?? id,
  );

const deviceSns = Array.from({ length: 25 }, (_, index) => `SN-${index + 1}`);

describe('PaginatedDeviceSnList', () => {
  it('shows 20 devices per page and exposes the remaining devices on page 2', () => {
    render(<PaginatedDeviceSnList deviceSns={deviceSns} onChange={() => undefined} t={t} />);

    expect(screen.getByText('已选 25/200 台设备')).toBeInTheDocument();
    expect(screen.getByText('SN-1')).toBeInTheDocument();
    expect(screen.queryByText('SN-21')).not.toBeInTheDocument();

    fireEvent.click(screen.getByTitle('2'));

    const list = screen.getByRole('list', { name: '已选 25/200 台设备' });
    expect(within(list).getAllByRole('listitem')).toHaveLength(5);
    expect(within(list).getByText('SN-21')).toBeInTheDocument();
    expect(screen.getByText('第 21–25 台，共 25 台')).toBeInTheDocument();
  });

  it('removes one device and can clear the full selection', () => {
    const onChange = vi.fn();
    render(<PaginatedDeviceSnList deviceSns={deviceSns} onChange={onChange} t={t} />);

    fireEvent.click(screen.getByRole('button', { name: '移除设备 SN-1' }));
    expect(onChange).toHaveBeenLastCalledWith(deviceSns.slice(1));

    fireEvent.click(screen.getByRole('button', { name: '清空已选设备' }));
    expect(onChange).toHaveBeenLastCalledWith([]);
  });

  it('shows an immediate warning when manual input exceeds 200 devices', () => {
    const overLimit = Array.from({ length: 201 }, (_, index) => `SN-${index + 1}`);
    render(<PaginatedDeviceSnList deviceSns={overLimit} onChange={() => undefined} t={t} />);

    expect(screen.getByText('设备数量 201 超过单任务上限 200 台。')).toBeInTheDocument();
  });
});
