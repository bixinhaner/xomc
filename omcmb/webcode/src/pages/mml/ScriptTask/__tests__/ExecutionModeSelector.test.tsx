import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import ExecutionModeSelector from '../ExecutionModeSelector';

const copy: Record<string, string> = {
  'mml.executeModeCommon': '统一脚本批量执行',
  'mml.executeModeCommonDescription': '选择多台设备，每台设备执行同一套脚本。',
  'mml.executeModeDeviceBound': '按设备编排执行',
  'mml.executeModeDeviceBoundDescription': '每行命令绑定设备 SN；同一设备按脚本从上到下执行。',
};

const t = (id: string) => copy[id] ?? id;

describe('ExecutionModeSelector', () => {
  it('shows action-oriented mode names with always-visible explanations', () => {
    render(
      <ExecutionModeSelector
        value="device_bound"
        deviceBoundDetected={false}
        onChange={() => undefined}
        t={t}
      />,
    );

    expect(screen.getByText('统一脚本批量执行')).toBeInTheDocument();
    expect(screen.getByText('选择多台设备，每台设备执行同一套脚本。')).toBeInTheDocument();
    expect(screen.getByText('按设备编排执行')).toBeInTheDocument();
    expect(screen.getByText('每行命令绑定设备 SN；同一设备按脚本从上到下执行。')).toBeInTheDocument();
  });

  it('changes mode and prevents choosing batch execution after an SN is detected', () => {
    const onChange = vi.fn();
    const { rerender } = render(
      <ExecutionModeSelector
        value="device_bound"
        deviceBoundDetected={false}
        onChange={onChange}
        t={t}
      />,
    );

    fireEvent.click(screen.getByText('统一脚本批量执行'));
    expect(onChange).toHaveBeenCalledWith('common');

    rerender(
      <ExecutionModeSelector
        value="device_bound"
        deviceBoundDetected
        onChange={onChange}
        t={t}
      />,
    );

    expect(screen.getByRole('radio', { name: /统一脚本批量执行/ })).toBeDisabled();
  });
});
