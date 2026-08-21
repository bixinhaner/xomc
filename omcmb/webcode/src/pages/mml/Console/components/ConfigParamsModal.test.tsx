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
    </App>
  );
}

function renderModal(props: Partial<ModalProps> = {}) {
  return render(modal(props));
}

describe('ConfigParamsModal', () => {
  it('opens the command list directly when switching from raw to standard without a command', () => {
    const onGotoCommand = vi.fn();
    renderModal({
      command: null,
      initialMode: 'raw',
      onGotoCommand,
    });

    fireEvent.click(screen.getByRole('tab', { name: 'mml.consoleV2.config.tabStandard' }));

    expect(onGotoCommand).toHaveBeenCalledOnce();
  });

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

  it('shows confirmation-only query copy without the stale parameter-check instruction', () => {
    renderModal({ selectedPathKeys: ['Device.Info.Name'] });

    expect(screen.getByText('mml.consoleV2.config.confirmSelectedPaths')).toBeInTheDocument();
    expect(screen.queryByText('mml.consoleV2.config.hintRead')).not.toBeInTheDocument();
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

  it('wraps long MOD paths and keeps each value input inside its own block', () => {
    const longPath = 'Device.Services.FAPService.{i}.FAPControl.LTE.Gateway.MmePool.MmePoolListMapIpsecTunnel';
    renderModal({
      command: {
        ...command,
        operationType: 'MOD',
        commandCode: 'MOD LONG PATH',
        commandName: '修改长路径参数',
        paramPaths: [{
          path: longPath,
          label: 'MME_POOL_LIST_MAP_IPSEC_TUNNEL',
          writable: true,
          isObject: false,
        }],
      },
      selectedPathKeys: [longPath],
    });

    const path = screen.getByText(longPath, { exact: true });
    const pathContainer = path.parentElement ?? path;
    const input = screen.getByRole('textbox');

    expect(pathContainer).toHaveStyle({
      whiteSpace: 'normal',
      overflowWrap: 'anywhere',
      wordBreak: 'break-word',
    });
    expect(input).toHaveStyle({
      display: 'block',
      width: '100%',
    });
    expect(path.closest('div')).not.toBe(input.parentElement);
    expect(document.querySelectorAll('.mml-config-tab-scroll')).toHaveLength(1);
    expect(document.querySelector('.mml-config-tab-scroll')).toHaveStyle({
      overflowX: 'hidden',
      overflowY: 'auto',
    });
  });

  it('does not block MOD execution for a selected non-writable Path hidden from the config page', () => {
    renderModal({
      command: {
        ...command,
        operationType: 'MOD',
        paramPaths: [
          {
            path: 'Device.Radio.Channel',
            label: 'Channel',
            writable: true,
            isObject: false,
            valueType: 'unsignedInt',
            minValue: 1,
            maxValue: 13,
          },
          {
            path: 'Device.Radio.Serial',
            label: 'Serial',
            writable: false,
            isObject: false,
          },
        ],
      },
      selectedPathKeys: ['Device.Radio.Channel', 'Device.Radio.Serial'],
    });

    expect(screen.getByText('Channel')).toBeInTheDocument();
    expect(screen.queryByText('Serial')).not.toBeInTheDocument();
    expect(screen.getAllByRole('textbox')).toHaveLength(1);

    fireEvent.change(screen.getByRole('textbox'), { target: { value: '13' } });

    expect(screen.getByRole('button', { name: /mml.consoleV2.config.confirmAndExecute/ })).toBeEnabled();
  });

  it('shows the standard data type without treating string minLength as a default', () => {
    renderModal({
      command: {
        ...command,
        operationType: 'MOD',
        paramPaths: [{
          path: 'Device.Info.Name',
          label: 'Name',
          writable: true,
          isObject: false,
          valueType: 'string',
          minValue: 2,
          maxValue: 8,
        }],
      },
      selectedPathKeys: ['Device.Info.Name'],
    });

    expect(screen.getByText('string')).toBeInTheDocument();
    expect(screen.getByRole('textbox')).toHaveValue('');
  });

  it('blocks execution and shows a per-field error outside the configured range', () => {
    renderModal({
      command: {
        ...command,
        operationType: 'MOD',
        paramPaths: [{
          path: 'Device.Radio.Channel',
          label: 'Channel',
          writable: true,
          isObject: false,
          valueType: 'unsignedInt',
          minValue: 1,
          maxValue: 13,
        }],
      },
      selectedPathKeys: ['Device.Radio.Channel'],
    });

    const input = screen.getByRole('textbox');
    const execute = screen.getByRole('button', { name: /mml.consoleV2.config.confirmAndExecute/ });
    fireEvent.change(input, { target: { value: '14' } });

    expect(screen.queryByText('mml.consoleV2.config.validation.maxValue')).not.toBeInTheDocument();
    expect(execute).toBeEnabled();
    fireEvent.click(execute);
    expect(screen.getByText('mml.consoleV2.config.validation.maxValue')).toBeInTheDocument();
    expect(input).toHaveAttribute('aria-invalid', 'true');

    fireEvent.change(input, { target: { value: '13' } });
    expect(screen.queryByText('mml.consoleV2.config.validation.maxValue')).not.toBeInTheDocument();
    expect(execute).toBeEnabled();
  });

  it('does not type-check a nonblank MOD value when its range is not configured', () => {
    renderModal({
      command: {
        ...command,
        operationType: 'MOD',
        paramPaths: [{
          path: 'Device.Radio.Channel',
          label: 'Channel',
          writable: true,
          isObject: false,
          valueType: 'unsignedInt',
        }],
      },
      selectedPathKeys: ['Device.Radio.Channel'],
    });

    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'not-a-number' } });

    expect(screen.getByRole('button', { name: /mml.consoleV2.config.confirmAndExecute/ })).toBeEnabled();
  });

  it('defers the required MOD error until execution is submitted', () => {
    const onConfirmAndExecute = vi.fn();
    renderModal({
      command: { ...command, operationType: 'MOD', commandCode: 'MOD INFO', commandName: '修改设备信息' },
      selectedPathKeys: ['Device.Info.Name'],
      onConfirmAndExecute,
    });

    const execute = screen.getByRole('button', { name: /mml.consoleV2.config.confirmAndExecute/ });
    expect(screen.queryByText('mml.consoleV2.config.validation.required')).not.toBeInTheDocument();
    expect(execute).toBeEnabled();

    fireEvent.click(execute);

    expect(screen.getByText('mml.consoleV2.config.validation.required')).toBeInTheDocument();
    expect(onConfirmAndExecute).not.toHaveBeenCalled();
  });

  it('requires a nonblank value for every selected MOD Path before execution', () => {
    const onConfirmAndExecute = vi.fn();
    renderModal({
      command: { ...command, operationType: 'MOD', commandCode: 'MOD INFO', commandName: '修改设备信息' },
      selectedPathKeys: ['Device.Info.Name'],
      onConfirmAndExecute,
    });

    const execute = screen.getByRole('button', { name: /mml.consoleV2.config.confirmAndExecute/ });
    expect(execute).toBeEnabled();
    fireEvent.change(screen.getByRole('textbox'), { target: { value: '   ' } });
    expect(execute).toBeEnabled();
    fireEvent.click(execute);
    expect(screen.getByText('mml.consoleV2.config.validation.required')).toBeInTheDocument();
    expect(onConfirmAndExecute).not.toHaveBeenCalled();

    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'cell-a' } });
    expect(execute).toBeEnabled();
    expect(screen.queryByText('mml.consoleV2.config.validation.required')).not.toBeInTheDocument();
    fireEvent.click(execute);

    expect(onConfirmAndExecute).toHaveBeenCalledWith(expect.objectContaining({
      mode: 'standard',
      checkedPaths: ['Device.Info.Name'],
      values: { 'Device.Info.Name': 'cell-a' },
    }));
  });

  it('does not use a STRING min length as the ADD default and blocks invalid submission', () => {
    const onConfirmAndExecute = vi.fn();
    const plmnPath = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.5GCell.{i}.PLMNID';
    renderModal({
      command: {
        ...command,
        operationType: 'ADD',
        commandCode: 'ADD 5G_CELL',
        commandName: '新增 NR邻区参数管理',
        targetObject: 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.5GCell.',
        paramPaths: [{
          path: plmnPath,
          label: 'PLMNID',
          writable: true,
          isObject: false,
          isRequired: true,
          valueType: 'STRING',
          minValue: 5,
          maxValue: 6,
        }],
      },
      selectedPathKeys: [plmnPath],
      onConfirmAndExecute,
    });

    const input = screen.getByRole('textbox');
    expect(input).toHaveValue('');
    fireEvent.click(screen.getByRole('button', { name: /mml.consoleV2.config.confirmAndExecute/ }));
    expect(screen.getByText('mml.consoleV2.config.validation.required')).toBeInTheDocument();
    expect(onConfirmAndExecute).not.toHaveBeenCalled();

    fireEvent.change(input, { target: { value: '46000' } });
    expect(screen.queryByText('mml.consoleV2.config.validation.required')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /mml.consoleV2.config.confirmAndExecute/ }));
    expect(onConfirmAndExecute).toHaveBeenCalled();
  });

  it('allows optional ADD parameters such as 5GCell SSB to remain blank', () => {
    const onConfirmAndExecute = vi.fn();
    const ssbPath = 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.5GCell.{i}.SSB';
    renderModal({
      command: {
        ...command,
        operationType: 'ADD',
        commandCode: 'ADD 5G_CELL',
        commandName: '新增 NR邻区参数管理',
        targetObject: 'Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.5GCell.',
        paramPaths: [{
          path: ssbPath,
          label: 'SSB',
          writable: true,
          isObject: false,
          isRequired: false,
          valueType: 'U_INT',
          minValue: 0,
          maxValue: 3279165,
        }],
      },
      selectedPathKeys: [ssbPath],
      onConfirmAndExecute,
    });

    const input = screen.getByRole('textbox');
    expect(input).toHaveValue('0');
    fireEvent.change(input, { target: { value: '' } });
    fireEvent.click(screen.getByRole('button', { name: /mml.consoleV2.config.confirmAndExecute/ }));

    expect(screen.queryByText('mml.consoleV2.config.validation.required')).not.toBeInTheDocument();
    expect(onConfirmAndExecute).toHaveBeenCalledWith(expect.objectContaining({ values: {} }));
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

  it.each([
    ['Device.A.{i}.Value', [null]],
    ['Device.A.{i}.B.{i}.Value', [1, null]],
    ['Device.A.{i}.B.{i}.C.{i}.Value', [1, 1, null]],
  ] as const)('initializes query instance inputs for %s', (queryPath, expected) => {
    renderModal({
      command: {
        ...command,
        id: queryPath,
        operationType: 'LST',
        paramPaths: [
          { path: queryPath, label: 'Value', writable: false, isObject: false },
        ],
      },
      selectedPathKeys: [queryPath],
    });

    expect(screen.getAllByRole('spinbutton').map((input) => (
      (input as HTMLInputElement).value === '' ? null : Number((input as HTMLInputElement).value)
    ))).toEqual(expected);
    expect(screen.getByText('mml.consoleV2.config.objectInstanceRead')).toBeInTheDocument();
  });

  it('allows clearing a query instance and emits blank selectors', () => {
    const onConfirmAndExecute = vi.fn();
    const queryPath = 'Device.A.{i}.B.{i}.Value';
    renderModal({
      command: {
        ...command,
        id: 'query-optional-instance',
        operationType: 'LST',
        paramPaths: [
          { path: queryPath, label: 'Value', writable: false, isObject: false },
        ],
      },
      selectedPathKeys: [queryPath],
      onConfirmAndExecute,
    });

    const inputs = screen.getAllByRole('spinbutton');
    expect(inputs[0]).toHaveValue('1');
    expect(inputs[1]).toHaveValue('');
    fireEvent.change(inputs[0], { target: { value: '' } });
    fireEvent.click(
      screen.getByRole('button', { name: /mml.consoleV2.config.confirmAndExecute/ }),
    );

    expect(onConfirmAndExecute).toHaveBeenCalledWith(expect.objectContaining({
      checkedPaths: [queryPath],
      instanceSelectors: { i01: '', i02: '' },
    }));
  });

  it('resets MOD values and instance selectors when the parent confirms different Paths', () => {
    const onConfirmAndExecute = vi.fn();
    const modCommand = {
      ...command,
      operationType: 'MOD' as const,
      commandCode: 'MOD SERVICE',
      commandName: '修改服务信息',
      paramPaths: [
        { path: 'Device.Services.{i}.Name', label: 'Name', writable: true, isObject: false },
        {
          path: 'Device.Services.{i}.Cells.{i}.Mode',
          label: 'Mode',
          writable: true,
          isObject: false,
        },
      ],
    };
    const { rerender } = renderModal({
      command: modCommand,
      selectedPathKeys: ['Device.Services.{i}.Name'],
      onConfirmAndExecute,
    });
    fireEvent.change(screen.getAllByRole('spinbutton')[0], { target: { value: '7' } });
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'cell-a' } });

    rerender(
      modal({
        command: modCommand,
        selectedPathKeys: ['Device.Services.{i}.Cells.{i}.Mode'],
        onConfirmAndExecute,
      }),
    );

    expect(screen.queryByText('Name')).not.toBeInTheDocument();
    expect(screen.getByText('Mode')).toBeInTheDocument();
    expect(screen.getAllByRole('spinbutton')).toHaveLength(2);
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'mode-b' } });
    fireEvent.click(screen.getByRole('button', { name: /mml.consoleV2.config.confirmAndExecute/ }));

    expect(onConfirmAndExecute).toHaveBeenLastCalledWith(expect.objectContaining({
      checkedPaths: ['Device.Services.{i}.Cells.{i}.Mode'],
      values: { 'Device.Services.{i}.Cells.{i}.Mode': 'mode-b' },
      instanceSelectors: { i01: '1', i02: '1' },
    }));
  });

  it.each(['LST', 'DSP', 'MOD'] as const)(
    'derives %s instance inputs and request selectors from confirmed Paths only',
    (operationType) => {
      const onConfirmAndExecute = vi.fn();
      const selectedPath = 'Device.Selected.{i}.Name';
      renderModal({
        command: {
          ...command,
          id: `instance-${operationType}`,
          operationType,
          commandCode: `${operationType} INSTANCE`,
          commandName: `${operationType} 实例范围`,
          paramPaths: [
            {
              path: 'Device.Hidden.{i}.Cells.{i}.Serial',
              label: 'Hidden',
              writable: operationType === 'MOD',
              isObject: false,
            },
            { path: selectedPath, label: 'Name', writable: operationType === 'MOD', isObject: false },
          ],
        },
        selectedPathKeys: [selectedPath],
        onConfirmAndExecute,
      });

      const instanceInputs = screen.getAllByRole('spinbutton');
      expect(instanceInputs).toHaveLength(1);
      fireEvent.change(instanceInputs[0], { target: { value: '8' } });
      if (operationType === 'MOD') {
        fireEvent.change(screen.getByRole('textbox'), { target: { value: 'selected-value' } });
      }
      fireEvent.click(screen.getByRole('button', { name: /mml.consoleV2.config.confirmAndExecute/ }));

      expect(onConfirmAndExecute).toHaveBeenCalledWith(expect.objectContaining({
        checkedPaths: [selectedPath],
        instanceSelectors: { i01: '8' },
      }));
    },
  );

  it('emits the complete ADD target-object, value, and instance request without Path selection', () => {
    const onConfirmAndExecute = vi.fn();
    const addPath = 'Device.Services.FAPService.{i}.Enable';
    renderModal({
      command: {
        ...command,
        id: 'add-1',
        operationType: 'ADD',
        commandCode: 'ADD SERVICE',
        commandName: '新增服务对象',
        targetObject: 'Device.Services.FAPService.{i}.',
        paramPaths: [
          { path: addPath, label: 'Enable', writable: true, isObject: false },
        ],
      },
      selectedPathKeys: [],
      onConfirmAndExecute,
    });

    expect(screen.getAllByText(/Device\.Services\.FAPService/).length).toBeGreaterThan(0);
    expect(screen.getByText('Enable')).toBeInTheDocument();
    fireEvent.change(screen.getByRole('spinbutton'), { target: { value: '3' } });
    fireEvent.change(screen.getByRole('textbox'), { target: { value: 'enabled' } });
    expect(screen.getByText('Device.Services.FAPService.3.')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /mml.consoleV2.config.confirmAndExecute/ }));

    expect(onConfirmAndExecute).toHaveBeenCalledWith({
      mode: 'standard',
      checkedPaths: [addPath],
      values: { [addPath]: 'enabled' },
      instanceSelectors: { i01: '3' },
      execMode: 'whole',
    });
  });

  it('emits the complete RMV target-object and forced whole-request payload', () => {
    const onConfirmAndExecute = vi.fn();
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
      onConfirmAndExecute,
    });

    const instanceInputs = screen.getAllByRole('spinbutton');
    expect(instanceInputs).toHaveLength(2);
    fireEvent.change(instanceInputs[0], { target: { value: '4' } });
    fireEvent.change(instanceInputs[1], { target: { value: '9' } });
    expect(screen.getByText('Device.Services.FAPService.4.')).toBeInTheDocument();
    expect(screen.getByRole('radio', { name: 'mml.consoleV2.config.execPerPath' })).toBeDisabled();
    expect(screen.getByRole('radio', { name: 'mml.consoleV2.config.execWhole' })).toBeChecked();
    fireEvent.click(screen.getByRole('button', { name: /mml.consoleV2.config.confirmAndExecute/ }));

    expect(onConfirmAndExecute).toHaveBeenCalledWith({
      mode: 'standard',
      checkedPaths: [],
      instanceSelectors: { i01: '4' },
      instance: 9,
      execMode: 'whole',
    });
  });

  it('renders boolean MOD values as a true/false select', () => {
    renderModal({
      command: {
        ...command,
        operationType: 'MOD',
        paramPaths: [{
          path: 'Device.Radio.Enable',
          label: 'Enable',
          writable: true,
          isObject: false,
          valueType: 'BOOLEAN',
          defaultValue: 'true',
        }],
      },
      selectedPathKeys: ['Device.Radio.Enable'],
    });

    expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
    expect(screen.getByText('true', { exact: true })).toBeInTheDocument();

    fireEvent.mouseDown(screen.getByRole('combobox'));
    expect(screen.getAllByRole('option').map((option) => option.textContent)).toEqual(['true', 'false']);
  });

  it('submits the selected boolean value as a string', () => {
    const onConfirmAndExecute = vi.fn();
    renderModal({
      command: {
        ...command,
        operationType: 'MOD',
        paramPaths: [{
          path: 'Device.Radio.Enable',
          label: 'Enable',
          writable: true,
          isObject: false,
          valueType: 'boolean',
          defaultValue: 'true',
        }],
      },
      selectedPathKeys: ['Device.Radio.Enable'],
      onConfirmAndExecute,
    });

    fireEvent.click(screen.getByRole('button', { name: /mml.consoleV2.config.confirmAndExecute/ }));

    expect(onConfirmAndExecute).toHaveBeenCalledWith(expect.objectContaining({
      mode: 'standard',
      values: { 'Device.Radio.Enable': 'true' },
    }));
  });

  it('submits the U32 enum value for RFTxStatus', () => {
    const onConfirmAndExecute = vi.fn();
    renderModal({
      command: {
        ...command,
        operationType: 'MOD',
        paramPaths: [{
          path: 'Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus',
          label: 'RFTxStatus',
          writable: true,
          isObject: false,
          valueType: 'U_INT',
          defaultValue: '1',
          enumOptions: [
            { value: '0', label: 'Inactive' },
            { value: '1', label: 'Active' },
          ],
        }],
      },
      selectedPathKeys: ['Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus'],
      onConfirmAndExecute,
    });

    fireEvent.click(screen.getByRole('button', { name: /mml.consoleV2.config.confirmAndExecute/ }));

    expect(screen.queryByText('mml.consoleV2.config.validation.enumValue')).not.toBeInTheDocument();
    expect(onConfirmAndExecute).toHaveBeenCalledWith(expect.objectContaining({
      values: {
        'Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus': '1',
      },
    }));
  });

  it('defaults a boolean value to false when no default is provided', () => {
    renderModal({
      command: {
        ...command,
        operationType: 'MOD',
        paramPaths: [{
          path: 'Device.Radio.Enable',
          label: 'Enable',
          writable: true,
          isObject: false,
          valueType: 'boolean',
        }],
      },
      selectedPathKeys: ['Device.Radio.Enable'],
    });

    expect(screen.getByText('false', { exact: true })).toBeInTheDocument();
  });

  it('submits boolean true when the backing model enum uses 0/1 values', () => {
    const onConfirmAndExecute = vi.fn();
    const path = 'Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.NeighborList.NRCell.{i}.NoRemoveEnable';
    renderModal({
      command: {
        ...command,
        operationType: 'ADD',
        targetObject: 'Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.NeighborList.NRCell.',
        paramPaths: [{
          path,
          label: 'NoRemoveEnable',
          writable: true,
          isObject: false,
          valueType: 'BOOLEAN',
          defaultValue: 'true',
          enumOptions: [
            { value: '0', label: 'false' },
            { value: '1', label: 'true' },
          ],
        }],
      },
      selectedPathKeys: [path],
      onConfirmAndExecute,
    });

    fireEvent.click(screen.getByRole('button', { name: /mml.consoleV2.config.confirmAndExecute/ }));

    expect(screen.queryByText('mml.consoleV2.config.validation.enumValue')).not.toBeInTheDocument();
    expect(onConfirmAndExecute).toHaveBeenCalledWith(expect.objectContaining({
      values: { [path]: 'true' },
    }));
  });

  it('renders model enum values as a select and applies defaultValue', () => {
    renderModal({
      command: {
        ...command,
        operationType: 'MOD',
        paramPaths: [{
          path: 'Device.Radio.Mode',
          label: 'Mode',
          writable: true,
          isObject: false,
          valueType: 'STRING',
          defaultValue: '1',
          enumOptions: [
            { value: '0', label: 'Disabled' },
            { value: '1', label: 'Enabled' },
          ],
        }],
      },
      selectedPathKeys: ['Device.Radio.Mode'],
    });

    expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
    expect(screen.getByText('Enabled', { exact: true })).toBeInTheDocument();
  });

  it('reinitializes values when the same path receives a new parameter model', () => {
    const path = 'Device.Radio.Mode';
    const firstCommand = {
      ...command,
      operationType: 'MOD' as const,
      paramPaths: [{
        path,
        label: 'Mode',
        writable: true,
        isObject: false,
        valueType: 'STRING',
        defaultValue: '0',
        enumOptions: [
          { value: '0', label: 'Disabled' },
          { value: '1', label: 'Enabled' },
        ],
      }],
    };
    const { rerender } = renderModal({ command: firstCommand, selectedPathKeys: [path] });

    expect(screen.getByText('Disabled', { exact: true })).toBeInTheDocument();

    rerender(modal({
      command: {
        ...firstCommand,
        paramPaths: [{
          ...firstCommand.paramPaths[0],
          defaultValue: '1',
        }],
      },
      selectedPathKeys: [path],
    }));

    expect(screen.getByText('Enabled', { exact: true })).toBeInTheDocument();
  });

  it('defers validationPattern errors until submit', () => {
    const onConfirmAndExecute = vi.fn();
    renderModal({
      command: {
        ...command,
        operationType: 'MOD',
        paramPaths: [{
          path: 'Device.Radio.PLMNID',
          label: 'PLMNID',
          writable: true,
          isObject: false,
          valueType: 'STRING',
          validationPattern: '/^\\d{5,6}$/',
        }],
      },
      selectedPathKeys: ['Device.Radio.PLMNID'],
      onConfirmAndExecute,
    });

    const input = screen.getByRole('textbox');
    fireEvent.change(input, { target: { value: 'abc' } });
    expect(screen.queryByText('mml.consoleV2.config.validation.pattern')).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /mml.consoleV2.config.confirmAndExecute/ }));
    expect(screen.getByText('mml.consoleV2.config.validation.pattern')).toBeInTheDocument();
    expect(onConfirmAndExecute).not.toHaveBeenCalled();
  });

  it('keeps input and select controls at the same width with a right-aligned arrow', () => {
    renderModal({
      command: {
        ...command,
        operationType: 'MOD',
        paramPaths: [
          { path: 'Device.Param.Text', label: 'Text', writable: true, isObject: false },
          {
            path: 'Device.Param.Mode',
            label: 'Mode',
            writable: true,
            isObject: false,
            enumOptions: [{ value: 'a-very-long-enum-value', label: 'a-very-long-enum-value' }],
          },
          {
            path: 'Device.Param.Enabled',
            label: 'Enabled',
            writable: true,
            isObject: false,
            valueType: 'boolean',
          },
        ],
      },
      selectedPathKeys: ['Device.Param.Text', 'Device.Param.Mode', 'Device.Param.Enabled'],
    });

    expect(document.querySelectorAll('.mml-config-param-control')).toHaveLength(3);
    expect(document.querySelectorAll('.mml-config-param-select')).toHaveLength(2);

    const selects = [...document.querySelectorAll<HTMLElement>('.mml-config-param-select')];
    expect(selects).toHaveLength(2);
    expect(selects.every((select) => select.style.width === '100%')).toBe(true);
    expect(selects.every((select) => select.style.display !== 'block')).toBe(true);
    expect(selects.every((select) => select.querySelector('.ant-select-suffix'))).toBe(true);
  });
});
