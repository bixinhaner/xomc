import { fireEvent, render, screen } from '@testing-library/react';
import { App } from 'antd';
import type { ComponentProps } from 'react';
import { describe, expect, it, vi } from 'vitest';
import type { useCommandSubFields, useGroupTree, useUnsupportedPaths } from '@core/hooks/api/useMmlConsole';
import type { CommandItem } from '../types';
import CommandSelectModal from './CommandSelectModal';

const nativeGetComputedStyle = window.getComputedStyle.bind(window);
vi.spyOn(window, 'getComputedStyle').mockImplementation((element) => nativeGetComputedStyle(element));

const fixtures = vi.hoisted(() => {
  const commands = [
    {
      id: 'lst-1',
      commandCode: 'LST INFO',
      displayName: '查询设备信息',
      operationType: 'LST',
      targetObject: '',
    },
    {
      id: 'mod-1',
      commandCode: 'MOD INFO',
      displayName: '修改设备信息',
      operationType: 'MOD',
      targetObject: '',
    },
  ];

  return {
    subFields: [
      { tr069Path: 'Device.Info.Serial', label: 'Serial', accessType: 'READ_ONLY', isObject: false },
      { tr069Path: 'Device.Info.Name', label: 'Name', accessType: 'READ_WRITE', isObject: false },
    ],
    groupTree: [
      {
        id: 'group-1',
        displayName: '设备信息',
        commands,
        children: [],
      },
    ],
  };
});

vi.mock('@core/hooks/api/useMmlConsole', () => ({
  useGroupTree: () =>
    ({ data: fixtures.groupTree, isLoading: false }) as unknown as ReturnType<typeof useGroupTree>,
  useCommandSubFields: (id?: string) =>
    ({ data: id ? fixtures.subFields : undefined, isFetching: false }) as unknown as ReturnType<
      typeof useCommandSubFields
    >,
  useUnsupportedPaths: () => ({ data: [] }) as unknown as ReturnType<typeof useUnsupportedPaths>,
}));

vi.mock('@/hooks/useI18nText', () => ({
  useI18nText: () => ({ locale: 'zh-CN' }),
}));

vi.mock('@/hooks/useT', () => ({
  useT: () => (id: string) => id,
}));

vi.mock('../../components/customizedSubtree', async () => {
  const actual = await vi.importActual<typeof import('../../components/customizedSubtree')>(
    '../../components/customizedSubtree',
  );
  return {
    ...actual,
    useCustomCommands: () => ({ commands: [] }),
  };
});

function makeCommandItem(id: string, operationType: 'LST' | 'MOD'): CommandItem {
  return {
    id,
    groupName: '设备信息',
    commandCode: `${operationType} INFO`,
    commandName: operationType === 'LST' ? '查询设备信息' : '修改设备信息',
    operationType,
    description: '',
    paramPaths: [
      { path: 'Device.Info.Serial', label: 'Serial', writable: false, isObject: false },
      { path: 'Device.Info.Name', label: 'Name', writable: true, isObject: false },
    ],
  };
}

type ModalProps = ComponentProps<typeof CommandSelectModal>;

function renderCommandModal(props: Partial<ModalProps> = {}) {
  return (
    <App>
      <CommandSelectModal
        open
        value={null}
        selectedPathKeys={[]}
        deviceSn="device-1"
        productId="product-1"
        onCancel={() => undefined}
        onConfirm={() => undefined}
        onGotoRawParams={() => undefined}
        {...props}
      />
    </App>
  );
}

function renderModal(props: Partial<ModalProps> = {}) {
  return render(renderCommandModal(props));
}

describe('CommandSelectModal', () => {
  it('disables confirmation until an LST Path is selected and returns only selected keys', async () => {
    const onConfirm = vi.fn();
    renderModal({ onConfirm, value: null, selectedPathKeys: [] });
    fireEvent.click(document.querySelector('.ant-tree-switcher')!);
    fireEvent.click(await screen.findByText('查询设备信息'));

    expect(screen.getByRole('button', { name: 'mml.consoleV2.cmdSelect.okText' })).toBeDisabled();
    fireEvent.click(await screen.findByRole('checkbox', { name: /Name/ }));
    fireEvent.click(screen.getByRole('button', { name: 'mml.consoleV2.cmdSelect.okText' }));

    expect(onConfirm).toHaveBeenCalledWith(
      expect.objectContaining({
        id: 'lst-1',
        paramPaths: expect.arrayContaining([expect.objectContaining({ path: 'Device.Info.Serial' })]),
      }),
      ['Device.Info.Name'],
    );
  });

  it('shows only writable Path candidates for MOD', async () => {
    renderModal({ value: null, selectedPathKeys: [] });
    fireEvent.click(document.querySelector('.ant-tree-switcher')!);
    fireEvent.click(await screen.findByText('修改设备信息'));

    expect(await screen.findByRole('checkbox', { name: /Name/ })).toBeInTheDocument();
    expect(screen.queryByRole('checkbox', { name: /Serial/ })).not.toBeInTheDocument();
  });

  it('restores confirmed keys on reopen and clears the draft when switching commands', async () => {
    const { rerender } = renderModal({
      value: makeCommandItem('lst-1', 'LST'),
      selectedPathKeys: ['Device.Info.Name'],
    });
    expect(await screen.findByRole('checkbox', { name: /Name/ })).toBeChecked();

    rerender(
      renderCommandModal({
        open: false,
        value: makeCommandItem('lst-1', 'LST'),
        selectedPathKeys: ['Device.Info.Name'],
      }),
    );
    rerender(
      renderCommandModal({
        open: true,
        value: makeCommandItem('lst-1', 'LST'),
        selectedPathKeys: ['Device.Info.Name'],
      }),
    );
    expect(await screen.findByRole('checkbox', { name: /Name/ })).toBeChecked();

    fireEvent.click(document.querySelector('.ant-tree-switcher')!);
    fireEvent.click(await screen.findByText('修改设备信息'));
    expect(await screen.findByRole('checkbox', { name: /Name/ })).not.toBeChecked();
  });
});
