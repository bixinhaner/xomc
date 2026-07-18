import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { CommandParamPath } from '../types';
import CommandPathSelector from './CommandPathSelector';

vi.mock('@/hooks/useT', () => ({
  useT: () => (id: string, values?: Record<string, unknown>) =>
    values ? `${id}:${JSON.stringify(values)}` : id,
}));

const paths: CommandParamPath[] = [
  { path: 'Device.Info.Serial', label: 'Serial', writable: false, isObject: false },
  { path: 'Device.Info.Name', label: 'Name', writable: true, isObject: false },
];

describe('CommandPathSelector', () => {
  it('starts fully unchecked and reports selected and total counts', () => {
    render(<CommandPathSelector paths={paths} value={[]} onChange={() => undefined} />);

    expect(screen.getAllByRole('checkbox')).toHaveLength(3);
    expect(screen.getAllByRole('checkbox').every((checkbox) => !(checkbox as HTMLInputElement).checked)).toBe(true);
    expect(screen.getByText(/"selected":0/)).toBeInTheDocument();
    expect(screen.getByText(/"total":2/)).toBeInTheDocument();
  });

  it('emits selected keys in candidate order and supports select all', () => {
    const onChange = vi.fn();
    const { rerender } = render(<CommandPathSelector paths={paths} value={[]} onChange={onChange} />);

    fireEvent.click(screen.getByRole('checkbox', { name: /Name/ }));
    expect(onChange).toHaveBeenLastCalledWith(['Device.Info.Name']);

    rerender(<CommandPathSelector paths={paths} value={['Device.Info.Name']} onChange={onChange} />);
    fireEvent.click(screen.getByRole('checkbox', { name: 'mml.consoleV2.cmdSelect.selectAll' }));
    expect(onChange).toHaveBeenLastCalledWith(['Device.Info.Serial', 'Device.Info.Name']);
  });
});
