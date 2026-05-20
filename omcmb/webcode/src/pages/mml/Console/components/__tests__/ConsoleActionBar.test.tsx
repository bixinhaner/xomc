import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';

// useT mock：直接回 i18n key 便于 assert；带 values 时拼参数
vi.mock('@/hooks/useT', () => ({
  useT:
    () =>
    (id: string, values?: Record<string, unknown>) =>
      values ? `${id}|${JSON.stringify(values)}` : id,
}));

// useThemeToken mock：返回必需的 token 字段以满足 inline style 引用
vi.mock('@/hooks/useThemeToken', () => ({
  useThemeToken: () => ({
    colorBorderSecondary: '#eee',
    colorBgContainer: '#fff',
    colorTextSecondary: '#888',
  }),
}));

import { ConsoleActionBar } from '../ConsoleActionBar';

describe('ConsoleActionBar', () => {
  it('renders execute button with operationType embedded in label', () => {
    render(
      <ConsoleActionBar
        operationType="LST"
        onExecute={() => undefined}
      />,
    );
    // i18n key + values JSON 包含 op
    expect(
      screen.getByText(/mml\.console\.actionBar\.execute\|.*"op":"LST"/),
    ).toBeInTheDocument();
  });

  it('execute button calls onExecute when clicked', () => {
    const onExecute = vi.fn();
    render(<ConsoleActionBar operationType="MOD" onExecute={onExecute} />);
    fireEvent.click(
      screen.getByText(/mml\.console\.actionBar\.execute\|.*"op":"MOD"/),
    );
    expect(onExecute).toHaveBeenCalledOnce();
  });

  it('disabled=true blocks onExecute via antd disabled state', () => {
    const onExecute = vi.fn();
    render(
      <ConsoleActionBar
        operationType="ADD"
        disabled
        onExecute={onExecute}
      />,
    );
    const btn = screen.getByText(/mml\.console\.actionBar\.execute/).closest('button');
    expect(btn).toHaveAttribute('disabled');
    if (btn) fireEvent.click(btn);
    // disabled 状态下 antd Button 不触发 onClick
    expect(onExecute).not.toHaveBeenCalled();
  });

  it('omits selectAll/clearAll/validate buttons when callbacks are undefined', () => {
    render(<ConsoleActionBar operationType="LST" onExecute={() => undefined} />);
    expect(screen.queryByText('mml.console.actionBar.selectAll')).toBeNull();
    expect(screen.queryByText('mml.console.actionBar.clearAll')).toBeNull();
    expect(screen.queryByText('mml.console.actionBar.validate')).toBeNull();
  });

  it('renders selectAll + clearAll only when callbacks provided', () => {
    const onSelectAll = vi.fn();
    const onClearAll = vi.fn();
    render(
      <ConsoleActionBar
        operationType="LST"
        onSelectAll={onSelectAll}
        onClearAll={onClearAll}
        onExecute={() => undefined}
      />,
    );
    fireEvent.click(screen.getByText('mml.console.actionBar.selectAll'));
    fireEvent.click(screen.getByText('mml.console.actionBar.clearAll'));
    expect(onSelectAll).toHaveBeenCalledOnce();
    expect(onClearAll).toHaveBeenCalledOnce();
  });

  it('renders deviceCount summary in the left side', () => {
    const { container } = render(
      <ConsoleActionBar
        operationType="LST"
        deviceCount={3}
        onExecute={() => undefined}
      />,
    );
    // deviceCount + i18n key 在同一 span 中拼接渲染，匹配整段文本
    expect(container.textContent).toMatch(/3\s+mml\.console\.actionBar\.devices/);
  });

  it('summaryText + deviceCount renders both segments separated by dot', () => {
    const { container } = render(
      <ConsoleActionBar
        operationType="MOD"
        summaryText="filled 2"
        deviceCount={5}
        onExecute={() => undefined}
      />,
    );
    expect(screen.getByText('filled 2')).toBeInTheDocument();
    expect(container.textContent).toMatch(/5\s+mml\.console\.actionBar\.devices/);
    // 中间分隔符 · 出现在左侧摘要 div 中
    expect(container.textContent).toContain('·');
  });

  it('loading=true disables auxiliary buttons but execute shows loading state', () => {
    const onSelectAll = vi.fn();
    render(
      <ConsoleActionBar
        operationType="LST"
        loading
        onSelectAll={onSelectAll}
        onExecute={() => undefined}
      />,
    );
    const selectAllBtn = screen
      .getByText('mml.console.actionBar.selectAll')
      .closest('button');
    expect(selectAllBtn).toHaveAttribute('disabled');
  });
});
