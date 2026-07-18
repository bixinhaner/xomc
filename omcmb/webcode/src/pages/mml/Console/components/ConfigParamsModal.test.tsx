import { fireEvent, render, screen } from '@testing-library/react';
import { App } from 'antd';
import type { ComponentProps } from 'react';
import { describe, expect, it, vi } from 'vitest';
import type { CommandItem } from '../types';
import ConfigParamsModal from './ConfigParamsModal';

const nativeGetComputedStyle = window.getComputedStyle.bind(window);
vi.spyOn(window, 'getComputedStyle').mockImplementation((element) => nativeGetComputedStyle(element));

vi.mock('@/hooks/useT', () => ({
  useT: () => (id: string) => id,
}));

vi.mock('@core/hooks/usePermission', () => ({
  usePermission: () => true,
}));

const command: CommandItem = {
  id: 'command-1',
  groupName: '设备信息',
  commandCode: 'LST INFO',
  commandName: '查询设备信息',
  operationType: 'LST',
  description: '',
  paramPaths: [
    { path: 'Device.Info.Serial', label: 'Serial', writable: false, isObject: false },
    { path: 'Device.Info.Name', label: 'Name', writable: true, isObject: false },
    { path: 'Device.Info.Mode', label: 'Mode', writable: true, isObject: false },
  ],
};

type ModalProps = ComponentProps<typeof ConfigParamsModal>;

function modal(props: Partial<ModalProps> = {}) {
  return (
    <App>
      <ConfigParamsModal
        open
        command={command}
        selectedPathKeys={[]}
        deviceCount={1}
        initialMode="standard"
        onCancel={() => undefined}
        onConfirmAndExecute={() => undefined}
        {...props}
      />
    </App>,
  );
}

function renderModal(props: Partial<ModalProps> = {}) {
  return render(modal(props));
}

describe('ConfigParamsModal', () => {
  it('shows only confirmed query Paths without second-stage checkboxes', () => {
    renderModal({
      command,
      selectedPathKeys: ['Device.Info.Name', 'Device.Info.Mode'],
    });

    expect(screen.queryByText('Serial')).not.toBeInTheDocument();
    expect(screen.getByText('Name')).toBeInTheDocument();
    expect(screen.getByText('Mode')).toBeInTheDocument();
    expect(screen.queryAllByRole('checkbox')).toHaveLength(0);
  });

  it('shows inputs only for selected writable MOD Paths', () => {
    renderModal({
      command: { ...command, operationType: 'MOD', commandCode: 'MOD INFO', commandName: '修改设备信息' },
      selectedPathKeys: ['Device.Info.Name'],
    });

    expect(screen.queryByText('Serial')).not.toBeInTheDocument();
    expect(screen.getByText('Name')).toBeInTheDocument();
    expect(screen.queryByText('Mode')).not.toBeInTheDocument();
    expect(screen.getByRole('textbox')).toBeInTheDocument();
  });

  it('requires a nonblank value for every selected MOD Path before execution', () => {
    const onConfirmAndExecute = vi.fn();
    renderModal({
      command: { ...command, operationType: 'MOD', commandCode: 'MOD INFO', commandName: '修改设备信息' },
      selectedPathKeys: ['Device.Info.Name'],
      onConfirmAndExecute,
    });

    const execute = screen.getByRole('button', { name: /mml.consoleV2.config.confirmAndExecute/ });
    expect(execute).toBeDisabled();
    fireEvent.change(screen.getByRole('textbox'), { target: { value: '   ' } });
    expect(execute).toBeDisabled();
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'cell-a' } });
    expect(execute).toBeEnabled();
    fireEvent.click(execute);

    expect(onConfirmAndExecute).toHaveBeenCalledWith(expect.objectContaining({
      mode: 'standard',
      checkedPaths: ['Device.Info.Name'],
      values: { 'Device.Info.Name': 'cell-a' },
    }));
  });

  it('executes confirmed query Paths in command-definition order', () => {
    const onConfirmAndExecute = vi.fn();
    renderModal({
      selectedPathKeys: ['Device.Info.Mode', 'Device.Info.Name'],
      onConfirmAndExecute,
    });

    fireEvent.click(screen.getByRole('button', { name: /mml.consoleV2.config.confirmAndExecute/ }));

    expect(onConfirmAndExecute).toHaveBeenCalledWith(expect.objectContaining({
      checkedPaths: ['Device.Info.Name', 'Device.Info.Mode'],
    }));
  });

  it('resets hidden MOD values when the parent confirms a different Path selection', () => {
    const modCommand = {
      ...command,
      operationType: 'MOD' as const,
      commandCode: 'MOD INFO',
      commandName: '修改设备信息',
    };
    const { rerender } = renderModal({ command: modCommand, selectedPathKeys: ['Device.Info.Name'] });
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'cell-a' } });
    expect(screen.getByRole('button', { name: /mml.consoleV2.config.confirmAndExecute/ })).toBeEnabled();

    rerender(modal({ command: modCommand, selectedPathKeys: ['Device.Info.Mode'] }));

    expect(screen.queryByText('Name')).not.toBeInTheDocument();
    expect(screen.getByText('Mode')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /mml.consoleV2.config.confirmAndExecute/ })).toBeDisabled();
  });

  it('preserves ADD target-object and value behavior without Path selection', () => {
    renderModal({
      command: {
        ...command,
        id: 'add-1',
        operationType: 'ADD',
        commandCode: 'ADD SERVICE',
        commandName: '新增服务对象',
        targetObject: 'Device.Services.FAPService.{i}.',
        paramPaths: [
          { path: 'Device.Services.FAPService.{i}.Enable', label: 'Enable', writable: true, isObject: false },
        ],
      },
      selectedPathKeys: [],
    });

    expect(screen.getAllByText(/Device\.Services\.FAPService/).length).toBeGreaterThan(0);
    expect(screen.getByText('Enable')).toBeInTheDocument();
    expect(screen.getByRole('textbox')).toBeInTheDocument();
  });

  it('preserves RMV instance input and whole-request execution mode', () => {
    renderModal({
      command: {
        ...command,
        id: 'rmv-1',
        operationType: 'RMV',
        commandCode: 'RMV SERVICE',
        commandName: '删除服务对象',
        targetObject: 'Device.Services.FAPService.{i}.',
        paramPaths: [],
      },
      selectedPathKeys: [],
    });

    expect(screen.getAllByRole('spinbutton').length).toBeGreaterThan(0);
    expect(screen.getByRole('radio', { name: 'mml.consoleV2.config.execPerPath' })).toBeDisabled();
    expect(screen.getByRole('radio', { name: 'mml.consoleV2.config.execWhole' })).toBeChecked();
  });
});
