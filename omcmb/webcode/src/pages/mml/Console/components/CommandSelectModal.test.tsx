import { fireEvent, render, screen } from '@testing-library/react';
import { App } from 'antd';
import type { ComponentProps } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type {
  useCommandSubFields,
  useCustomCommandPaths,
  useGroupTree,
  useUnsupportedPaths,
} from '@core/hooks/api/useMmlConsole';
import type { MMLCustomCommand, MMLCustomCommandPathDef } from '@core/types/mml';
import type { CommandItem } from '../types';
import CommandSelectModal from './CommandSelectModal';

const nativeGetComputedStyle = window.getComputedStyle.bind(window);
vi.spyOn(window, 'getComputedStyle').mockImplementation((element) => nativeGetComputedStyle(element));

const fixtures = vi.hoisted(() => {
  const defaultSubFields = [
    { tr069Path: 'Device.Info.Serial', label: 'Serial', accessType: 'READ_ONLY', isObject: false },
    { tr069Path: 'Device.Info.Name', label: 'Name', accessType: 'READ_WRITE', isObject: false },
    { tr069Path: 'Device.Info.Model', label: 'Model', accessType: 'READ_WRITE', isObject: false },
    { tr069Path: 'Device.Info.Alias', label: 'Alias', accessType: 'READ_WRITE', isObject: false },
  ];
  const commands = [
    {
      id: 'lst-1',
      commandCode: 'LST INFO',
      displayName: '查询设备信息',
      operationType: 'LST',
      targetObject: '',
      targetPaths: ['Device.Info.Serial', 'Device.Info.Name'],
    },
    {
      id: 'dsp-1',
      commandCode: 'DSP INFO',
      displayName: '展示设备信息',
      operationType: 'DSP',
      targetObject: '',
    },
    {
      id: 'mod-1',
      commandCode: 'MOD INFO',
      displayName: '修改设备信息',
      operationType: 'MOD',
      targetObject: '',
    },
    {
      id: 'add-1',
      commandCode: 'ADD USER',
      displayName: '新增用户',
      operationType: 'ADD',
      targetObject: 'Device.Users.User.',
    },
    {
      id: 'rmv-1',
      commandCode: 'RMV USER',
      displayName: '删除用户',
      operationType: 'RMV',
      targetObject: 'Device.Users.User.',
    },
  ];

  return {
    defaultSubFields,
    subFields: defaultSubFields,
    customCommands: [] as MMLCustomCommand[],
    customPaths: [] as MMLCustomCommandPathDef[],
    unsupportedPaths: [] as
      | Array<{
          path: string;
          readUnsupported: boolean;
          writeUnsupported: boolean;
        }>
      | undefined,
    unsupportedPathsFetching: false,
    unsupportedPathsError: false,
    groupTreeCalls: [] as unknown[][],
    subFieldCalls: [] as unknown[][],
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
  useGroupTree: (...args: unknown[]) => {
    fixtures.groupTreeCalls.push(args);
    return ({ data: fixtures.groupTree, isLoading: false }) as unknown as ReturnType<typeof useGroupTree>;
  },
  useCommandSubFields: (...args: unknown[]) => {
    fixtures.subFieldCalls.push(args);
    const id = args[0] as string | undefined;
    return ({ data: id ? fixtures.subFields : undefined, isFetching: false }) as unknown as ReturnType<
      typeof useCommandSubFields
    >;
  },
  useCustomCommandPaths: (id?: string) =>
    ({ data: id ? fixtures.customPaths : undefined, isFetching: false }) as unknown as ReturnType<
      typeof useCustomCommandPaths
    >,
  useUnsupportedPaths: () =>
    ({
      data: fixtures.unsupportedPaths,
      isFetching: fixtures.unsupportedPathsFetching,
      isError: fixtures.unsupportedPathsError,
    }) as unknown as ReturnType<typeof useUnsupportedPaths>,
}));

vi.mock('@/hooks/useI18nText', () => ({
  useI18nText: () => ({ locale: 'zh-CN' }),
}));

vi.mock('@/hooks/useT', () => ({
  useT: () => (id: string, values?: Record<string, unknown>) =>
    values ? `${id}:${JSON.stringify(values)}` : id,
}));

vi.mock('../../components/customizedSubtree', async () => {
  const actual = await vi.importActual<typeof import('../../components/customizedSubtree')>(
    '../../components/customizedSubtree',
  );
  return {
    ...actual,
    useCustomCommands: () => ({ commands: fixtures.customCommands }),
  };
});

afterEach(() => {
  fixtures.subFields = fixtures.defaultSubFields;
  fixtures.customCommands = [];
  fixtures.customPaths = [];
  fixtures.unsupportedPaths = [];
  fixtures.unsupportedPathsFetching = false;
  fixtures.unsupportedPathsError = false;
  fixtures.groupTreeCalls = [];
  fixtures.subFieldCalls = [];
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
  it('requests command tree and sub-fields with productClass context', async () => {
    renderModal({ productClass: 'FAP/BU1810' });
    fireEvent.click(document.querySelector('.ant-tree-switcher')!);
    fireEvent.click(await screen.findByText('查询设备信息'));

    expect(fixtures.groupTreeCalls).toContainEqual([undefined, 'zh-CN', 'FAP/BU1810', 'device-1']);
    expect(fixtures.subFieldCalls).toContainEqual(['lst-1', 'zh-CN', 'device-1', 'FAP/BU1810', true]);
  });

  it('filters standard commands by target path', async () => {
    renderModal();

    fireEvent.change(
      screen.getByPlaceholderText('mml.consoleV2.cmdSelect.searchPlaceholder'),
      { target: { value: 'Device.Info.Serial' } },
    );

    expect(await screen.findByText('查询设备信息')).toBeInTheDocument();
    expect(screen.queryByText('展示设备信息')).not.toBeInTheDocument();
    expect(screen.queryByText('修改设备信息')).not.toBeInTheDocument();
  });

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

  it('starts DSP with no Paths selected, requires one, and returns its key separately', async () => {
    const onConfirm = vi.fn();
    renderModal({ onConfirm, value: null, selectedPathKeys: [] });
    fireEvent.click(document.querySelector('.ant-tree-switcher')!);
    fireEvent.click(await screen.findByText('展示设备信息'));

    expect(await screen.findByRole('checkbox', { name: /Serial/ })).not.toBeChecked();
    expect(screen.getByRole('button', { name: 'mml.consoleV2.cmdSelect.okText' })).toBeDisabled();

    fireEvent.click(screen.getByRole('checkbox', { name: /Serial/ }));
    fireEvent.click(screen.getByRole('button', { name: 'mml.consoleV2.cmdSelect.okText' }));

    expect(onConfirm).toHaveBeenCalledWith(expect.objectContaining({ id: 'dsp-1' }), ['Device.Info.Serial']);
  });

  it('shows only writable Path candidates for MOD', async () => {
    fixtures.subFields = fixtures.defaultSubFields.map((subField) => ({
      ...subField,
      defaultSelected:
        subField.tr069Path === 'Device.Info.Serial' ||
        subField.tr069Path === 'Device.Info.Name',
    }));
    renderModal({ value: null, selectedPathKeys: [] });
    fireEvent.click(document.querySelector('.ant-tree-switcher')!);
    fireEvent.click(await screen.findByText('修改设备信息'));

    expect(await screen.findByRole('checkbox', { name: /Name/ })).toBeChecked();
    expect(screen.queryByRole('checkbox', { name: /Serial/ })).not.toBeInTheDocument();
    expect(
      screen.getByText('mml.consoleV2.cmdSelect.paramPathCount:{"count":3}'),
    ).toBeInTheDocument();
  });

  it('confirms enriched type and range metadata for a custom MOD command', async () => {
    fixtures.customCommands = [{
      id: 'custom-1',
      commandName: '修改名称',
      commandCode: 'MOD CUSTOM',
      operationType: 'MOD',
      commandScope: 'public',
      categoryGroup: '',
      parameters: {},
      paramPaths: ['Device.Info.Name'],
      description: '',
      creator: 'admin',
      createdAt: '',
      updatedAt: '',
    }];
    fixtures.customPaths = [{
      id: 'path-1',
      commandId: 'custom-1',
      standardPathId: 'standard-1',
      standardPath: 'Device.Info.Name',
      entryType: 'parameter',
      access: 'readWrite',
      dataType: 'string',
      description: 'Name',
      minValue: 2,
      maxValue: 32,
      defaultSelected: true,
      sortOrder: 1,
    }];
    const onConfirm = vi.fn();
    renderModal({ onConfirm });

    for (let depth = 0; depth < 4 && !screen.queryByText('修改名称'); depth += 1) {
      const switcher = Array.from(document.querySelectorAll('.ant-tree-switcher')).find((node) =>
        node.classList.contains('ant-tree-switcher_close'),
      );
      if (!switcher) break;
      fireEvent.click(switcher);
      await new Promise((resolve) => setTimeout(resolve, 0));
    }
    fireEvent.click(await screen.findByText('修改名称'));
    expect(await screen.findByRole('checkbox', { name: /Name/ })).toBeChecked();
    fireEvent.click(screen.getByRole('button', { name: 'mml.consoleV2.cmdSelect.okText' }));

    expect(onConfirm).toHaveBeenCalledWith(
      expect.objectContaining({
        isCustom: true,
        paramPaths: [expect.objectContaining({
          path: 'Device.Info.Name',
          valueType: 'string',
          minValue: 2,
          maxValue: 32,
        })],
      }),
      ['Device.Info.Name'],
    );
  });

  it('keeps enriched custom paths inside the product-filtered command paths', async () => {
    fixtures.customCommands = [{
      id: 'custom-filtered',
      commandName: '修改产品支持参数',
      commandCode: 'MOD FILTERED',
      operationType: 'MOD',
      commandScope: 'public',
      categoryGroup: '',
      parameters: {},
      paramPaths: ['Device.Info.Name'],
      description: '',
      creator: 'admin',
      createdAt: '',
      updatedAt: '',
    }];
    fixtures.customPaths = [
      {
        id: 'path-supported',
        commandId: 'custom-filtered',
        standardPathId: 'standard-supported',
        standardPath: 'Device.Info.Name',
        entryType: 'parameter',
        access: 'READ_WRITE',
        dataType: 'STRING',
        description: 'Name',
        minValue: 2,
        maxValue: 32,
        defaultSelected: false,
        sortOrder: 1,
      },
      {
        id: 'path-filtered',
        commandId: 'custom-filtered',
        standardPathId: 'standard-filtered',
        standardPath: 'Device.Info.Model',
        entryType: 'parameter',
        access: 'READ_WRITE',
        dataType: 'STRING',
        description: 'Model',
        minValue: 2,
        maxValue: 32,
        defaultSelected: false,
        sortOrder: 2,
      },
    ];
    renderModal();

    for (let depth = 0; depth < 4 && !screen.queryByText('修改产品支持参数'); depth += 1) {
      const switcher = Array.from(document.querySelectorAll('.ant-tree-switcher')).find((node) =>
        node.classList.contains('ant-tree-switcher_close'),
      );
      if (!switcher) break;
      fireEvent.click(switcher);
      await new Promise((resolve) => setTimeout(resolve, 0));
    }
    fireEvent.click(await screen.findByText('修改产品支持参数'));

    expect(await screen.findByRole('checkbox', { name: /Name/ })).toBeInTheDocument();
    expect(screen.queryByRole('checkbox', { name: /Model/ })).not.toBeInTheDocument();
    expect(
      screen.getByText('mml.consoleV2.cmdSelect.paramPathCount:{"count":1}'),
    ).toBeInTheDocument();
  });

  it('shows all writable sub-fields for ADD alongside its target object', async () => {
    const onConfirm = vi.fn();
    renderModal({ onConfirm, value: null, selectedPathKeys: [] });
    fireEvent.click(document.querySelector('.ant-tree-switcher')!);
    fireEvent.click(await screen.findByText('新增用户'));

    expect(await screen.findByText('mml.consoleV2.cmdSelect.targetObjectPath')).toBeInTheDocument();
    expect(screen.getByText('Device.Users.User.')).toBeInTheDocument();
    expect(screen.getByText('mml.consoleV2.cmdSelect.paramPathCount:{"count":3}')).toBeInTheDocument();
    expect(screen.getByText('Device.Info.Name')).toBeInTheDocument();
    expect(screen.getByText('Device.Info.Model')).toBeInTheDocument();
    expect(screen.queryByText('Device.Info.Serial')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'mml.consoleV2.cmdSelect.okText' })).toBeEnabled();

    fireEvent.click(screen.getByRole('button', { name: 'mml.consoleV2.cmdSelect.okText' }));
    expect(onConfirm).toHaveBeenCalledWith(
      expect.objectContaining({
        id: 'add-1',
        targetObject: 'Device.Users.User.',
        paramPaths: expect.arrayContaining([
          expect.objectContaining({ path: 'Device.Info.Name', writable: true }),
        ]),
      }),
      [],
    );
  });

  it('keeps target-object behavior for RMV commands', async () => {
    const onConfirm = vi.fn();
    renderModal({ onConfirm, value: null, selectedPathKeys: [] });
    fireEvent.click(document.querySelector('.ant-tree-switcher')!);
    fireEvent.click(await screen.findByText('删除用户'));

    expect(await screen.findByText('mml.consoleV2.cmdSelect.targetObjectPath')).toBeInTheDocument();
    expect(screen.getByText('Device.Users.User.')).toBeInTheDocument();
    expect(screen.getByText(/DeleteObject/)).toBeInTheDocument();
    expect(screen.queryByRole('checkbox', { name: /Name/ })).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'mml.consoleV2.cmdSelect.okText' })).toBeEnabled();

    fireEvent.click(screen.getByRole('button', { name: 'mml.consoleV2.cmdSelect.okText' }));
    expect(onConfirm).toHaveBeenCalledWith(
      expect.objectContaining({ id: 'rmv-1', targetObject: 'Device.Users.User.' }),
      [],
    );
  });

  it('keeps standard ADD target-object confirmation available when unsupported Paths fail without data', async () => {
    fixtures.unsupportedPaths = undefined;
    fixtures.unsupportedPathsFetching = false;
    fixtures.unsupportedPathsError = true;
    const onConfirm = vi.fn();
    renderModal({ onConfirm, productId: 'product-1' });
    fireEvent.click(document.querySelector('.ant-tree-switcher')!);
    fireEvent.click(await screen.findByText('新增用户'));

    const okButton = screen.getByRole('button', { name: 'mml.consoleV2.cmdSelect.okText' });
    expect(okButton).toBeEnabled();
    expect(screen.getByText('mml.consoleV2.cmdSelect.targetObjectPath')).toBeInTheDocument();
    expect(screen.getByText('Device.Users.User.')).toBeInTheDocument();

    fireEvent.click(okButton);
    expect(onConfirm).toHaveBeenCalledWith(
      expect.objectContaining({ id: 'add-1', targetObject: 'Device.Users.User.' }),
      [],
    );
  });

  it('prunes keys that become non-writable and preserves current command-definition order on confirm', async () => {
    const onConfirm = vi.fn();
    const { rerender } = renderModal({
      onConfirm,
      value: makeCommandItem('mod-1', 'MOD'),
      selectedPathKeys: [],
    });
    fireEvent.click(await screen.findByRole('checkbox', { name: /Name/ }));
    fireEvent.click(screen.getByRole('checkbox', { name: /Model/ }));
    fireEvent.click(screen.getByRole('checkbox', { name: /Alias/ }));

    fixtures.subFields = [
      { tr069Path: 'Device.Info.Serial', label: 'Serial', accessType: 'READ_ONLY', isObject: false },
      { tr069Path: 'Device.Info.Name', label: 'Name', accessType: 'READ_ONLY', isObject: false },
      { tr069Path: 'Device.Info.Model', label: 'Model', accessType: 'READ_WRITE', isObject: false },
      { tr069Path: 'Device.Info.Alias', label: 'Alias', accessType: 'READ_WRITE', isObject: false },
    ];
    rerender(renderCommandModal({ onConfirm, value: makeCommandItem('mod-1', 'MOD'), selectedPathKeys: [] }));

    expect(await screen.findByRole('checkbox', { name: /Model/ })).toBeChecked();
    expect(screen.queryByRole('checkbox', { name: /Name/ })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'mml.consoleV2.cmdSelect.okText' }));

    expect(onConfirm).toHaveBeenCalledWith(
      expect.objectContaining({ id: 'mod-1' }),
      ['Device.Info.Model', 'Device.Info.Alias'],
    );
  });

  it.each([
    ['LST', '查询设备信息'],
    ['DSP', '展示设备信息'],
    ['MOD', '修改设备信息'],
  ])('shows the zero-selection prompt for %s', async (_operation, displayName) => {
    renderModal({ value: null, selectedPathKeys: [] });
    fireEvent.click(document.querySelector('.ant-tree-switcher')!);
    fireEvent.click(await screen.findByText(displayName));

    expect(await screen.findByText('mml.consoleV2.cmdSelect.pickPathFirst')).toBeInTheDocument();
  });

  it('reapplies configured defaults on reopen and ignores confirmed keys', async () => {
    fixtures.subFields = fixtures.defaultSubFields.map((subField) => ({
      ...subField,
      defaultSelected: subField.tr069Path === 'Device.Info.Serial',
    }));
    const props = {
      value: makeCommandItem('lst-1', 'LST'),
      selectedPathKeys: ['Device.Info.Name'],
    };
    const { rerender } = renderModal(props);

    const serial = await screen.findByRole('checkbox', { name: /Serial/ });
    const name = screen.getByRole('checkbox', { name: /Name/ });
    expect(serial).toBeChecked();
    expect(name).not.toBeChecked();

    fireEvent.click(serial);
    expect(serial).not.toBeChecked();

    rerender(renderCommandModal(props));
    expect(screen.getByRole('checkbox', { name: /Serial/ })).not.toBeChecked();

    rerender(renderCommandModal({ ...props, open: false }));
    rerender(renderCommandModal({ ...props, open: true }));
    expect(await screen.findByRole('checkbox', { name: /Serial/ })).toBeChecked();
    expect(screen.getByRole('checkbox', { name: /Name/ })).not.toBeChecked();
  });

  it('reapplies defaults after switching away from and back to a command', async () => {
    fixtures.subFields = fixtures.defaultSubFields.map((subField) => ({
      ...subField,
      defaultSelected: subField.tr069Path === 'Device.Info.Name',
    }));
    renderModal({
      value: makeCommandItem('lst-1', 'LST'),
      selectedPathKeys: [],
    });

    const name = await screen.findByRole('checkbox', { name: /Name/ });
    expect(name).toBeChecked();
    fireEvent.click(name);
    expect(name).not.toBeChecked();

    fireEvent.click(document.querySelector('.ant-tree-switcher')!);
    fireEvent.click(await screen.findByText('修改设备信息'));
    expect(await screen.findByRole('checkbox', { name: /Name/ })).toBeChecked();

    fireEvent.click(screen.getByRole('checkbox', { name: /Name/ }));
    fireEvent.click(await screen.findByText('查询设备信息'));
    expect(await screen.findByRole('checkbox', { name: /Name/ })).toBeChecked();
  });

  it('does not default-select a query Path hidden by runtime support filtering', async () => {
    fixtures.subFields = fixtures.defaultSubFields.map((subField) => ({
      ...subField,
      defaultSelected:
        subField.tr069Path === 'Device.Info.Serial' ||
        subField.tr069Path === 'Device.Info.Name',
    }));
    fixtures.unsupportedPaths = [{
      path: 'Device.Info.Serial',
      readUnsupported: true,
      writeUnsupported: false,
    }];

    renderModal({ value: null, selectedPathKeys: [] });
    fireEvent.click(document.querySelector('.ant-tree-switcher')!);
    fireEvent.click(await screen.findByText('查询设备信息'));

    expect(screen.queryByRole('checkbox', { name: /Serial/ })).not.toBeInTheDocument();
    expect(await screen.findByRole('checkbox', { name: /Name/ })).toBeChecked();
  });

  it('keeps Path confirmation closed until runtime support filtering succeeds', async () => {
    fixtures.subFields = fixtures.defaultSubFields.map((subField) => ({
      ...subField,
      defaultSelected:
        subField.tr069Path === 'Device.Info.Serial' ||
        subField.tr069Path === 'Device.Info.Name',
    }));
    fixtures.unsupportedPaths = undefined;
    fixtures.unsupportedPathsFetching = true;
    const onConfirm = vi.fn();
    const props = {
      onConfirm,
      value: makeCommandItem('lst-1', 'LST'),
      selectedPathKeys: [],
    };
    const { rerender } = renderModal(props);

    const okButton = screen.getByRole('button', { name: 'mml.consoleV2.cmdSelect.okText' });
    expect(okButton).toBeDisabled();
    expect(screen.queryByRole('checkbox', { name: /Serial/ })).not.toBeInTheDocument();
    fireEvent.click(okButton);
    expect(onConfirm).not.toHaveBeenCalled();

    fixtures.unsupportedPathsFetching = false;
    fixtures.unsupportedPathsError = true;
    rerender(renderCommandModal(props));

    expect(okButton).toBeDisabled();
    expect(screen.queryByRole('checkbox', { name: /Serial/ })).not.toBeInTheDocument();
    fireEvent.click(okButton);
    expect(onConfirm).not.toHaveBeenCalled();

    fixtures.unsupportedPaths = [{
      path: 'Device.Info.Serial',
      readUnsupported: true,
      writeUnsupported: false,
    }];
    fixtures.unsupportedPathsError = false;
    rerender(renderCommandModal(props));

    expect(screen.queryByRole('checkbox', { name: /Serial/ })).not.toBeInTheDocument();
    expect(await screen.findByRole('checkbox', { name: /Name/ })).toBeChecked();
    expect(okButton).toBeEnabled();
    fireEvent.click(okButton);
    expect(onConfirm).toHaveBeenCalledWith(
      expect.objectContaining({ id: 'lst-1' }),
      ['Device.Info.Name'],
    );
  });
});
