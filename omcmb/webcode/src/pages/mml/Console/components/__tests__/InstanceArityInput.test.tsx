import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { InstanceArityInput } from '../InstanceArityInput';
import {
  deriveArityFromSubFields,
  selectorKeysForArity,
} from '../instanceArity';

vi.mock('@/hooks/useT', () => ({
  useT:
    () =>
    (id: string, values?: Record<string, unknown>) =>
      values ? `${id}|${JSON.stringify(values)}` : id,
}));

vi.mock('@/hooks/useThemeToken', () => ({
  useThemeToken: () => ({
    colorBorderSecondary: '#eee',
    colorFillTertiary: '#fafafa',
    colorTextSecondary: '#666',
    colorTextTertiary: '#999',
    colorTextQuaternary: '#bbb',
    colorError: '#f00',
  }),
}));

describe('selectorKeysForArity', () => {
  it('returns Greek selector keys up to arity', () => {
    expect(selectorKeysForArity(0)).toEqual([]);
    expect(selectorKeysForArity(1)).toEqual(['iα']);
    expect(selectorKeysForArity(2)).toEqual(['iα', 'iβ']);
    expect(selectorKeysForArity(3)).toEqual(['iα', 'iβ', 'iγ']);
  });

  it('caps at ε (arity > 5 truncated)', () => {
    expect(selectorKeysForArity(5)).toEqual(['iα', 'iβ', 'iγ', 'iδ', 'iε']);
    expect(selectorKeysForArity(99)).toEqual(['iα', 'iβ', 'iγ', 'iδ', 'iε']);
  });

  it('字典序 sorted naturally matching backend convention', () => {
    // 后端按 key 字典序左到右映射 path 中 .{i}.；Greek 字母 UTF-8 codepoint
    // 已经天然按 α<β<γ<δ<ε 排序，无需额外 sort。
    const keys = selectorKeysForArity(3);
    const sorted = [...keys].sort();
    expect(sorted).toEqual(keys);
  });
});

describe('deriveArityFromSubFields', () => {
  it('returns 0 when subFields 空', () => {
    expect(deriveArityFromSubFields([])).toBe(0);
  });

  it('returns 0 when no .{i}. placeholder', () => {
    expect(
      deriveArityFromSubFields([{ tr069Path: 'Device.DeviceInfo.SoftwareVersion' }]),
    ).toBe(0);
  });

  it('counts single-layer .{i}.', () => {
    expect(
      deriveArityFromSubFields([{ tr069Path: 'Device.IP.Interface.{i}.IPAddress' }]),
    ).toBe(1);
  });

  it('counts multi-layer .{i}. (v2.3 catalog 3-layer)', () => {
    expect(
      deriveArityFromSubFields([
        { tr069Path: 'Device.DeviceInfo.MU.{i}.Slot.{i}.EU.{i}.HardwareVersion' },
      ]),
    ).toBe(3);
  });

  it('uses max across subFields when arity varies (defensive)', () => {
    expect(
      deriveArityFromSubFields([
        { tr069Path: 'Device.Foo.{i}.Bar' },
        { tr069Path: 'Device.Foo.{i}.Baz.{i}.Qux' },
      ]),
    ).toBe(2);
  });

  it('ignores missing tr069Path', () => {
    expect(
      deriveArityFromSubFields([
        { tr069Path: undefined },
        { tr069Path: 'Device.X.{i}.Y' },
      ]),
    ).toBe(1);
  });
});

describe('InstanceArityInput render', () => {
  it('arity=0 → 返回 null（不渲染）', () => {
    const { container } = render(
      <InstanceArityInput
        arity={0}
        values={{}}
        onChange={() => undefined}
        operationType="LST"
      />,
    );
    expect(container.firstChild).toBeNull();
  });

  it('arity=2 → 显示 2 个层级输入框 + Greek key 标签', () => {
    render(
      <InstanceArityInput
        arity={2}
        values={{}}
        onChange={() => undefined}
        operationType="MOD"
      />,
    );
    expect(screen.getByText(/iα\s*=/)).toBeInTheDocument();
    expect(screen.getByText(/iβ\s*=/)).toBeInTheDocument();
    expect(screen.queryByText(/iγ/)).toBeNull();
  });

  it('LST mode 不显示必填提示', () => {
    render(
      <InstanceArityInput
        arity={2}
        values={{}}
        onChange={() => undefined}
        operationType="LST"
      />,
    );
    expect(
      screen.queryByText('mml.console.instanceArity.requiredHint'),
    ).toBeNull();
  });

  it('MOD/ADD/RMV mode 显示必填提示', () => {
    for (const op of ['MOD', 'ADD', 'RMV'] as const) {
      const { container, unmount } = render(
        <InstanceArityInput
          arity={1}
          values={{}}
          onChange={() => undefined}
          operationType={op}
        />,
      );
      // 必填星号与 i18n key 在同一 span 拼接渲染：`* <key>`
      expect(container.textContent).toContain(
        'mml.console.instanceArity.requiredHint',
      );
      expect(container.textContent).toMatch(/\*\s+mml\.console\.instanceArity\.requiredHint/);
      unmount();
    }
  });

  it('onChange 用 selector key 调用（不是 layer index / 语义名）', () => {
    const onChange = vi.fn();
    render(
      <InstanceArityInput
        arity={2}
        values={{}}
        onChange={onChange}
        operationType="MOD"
      />,
    );
    // 找第一个 input（iα 的输入框）
    const inputs = screen.getAllByRole('textbox');
    expect(inputs).toHaveLength(2);
    fireEvent.change(inputs[0], { target: { value: '5' } });
    expect(onChange).toHaveBeenCalledWith('iα', '5');
    fireEvent.change(inputs[1], { target: { value: '7' } });
    expect(onChange).toHaveBeenCalledWith('iβ', '7');
  });

  it('values prop 反映到 input value', () => {
    render(
      <InstanceArityInput
        arity={3}
        values={{ iα: '1', iβ: '2', iγ: '3' }}
        onChange={() => undefined}
        operationType="LST"
      />,
    );
    const inputs = screen.getAllByRole('textbox') as HTMLInputElement[];
    expect(inputs[0].value).toBe('1');
    expect(inputs[1].value).toBe('2');
    expect(inputs[2].value).toBe('3');
  });
});
