import { describe, it, expect } from 'vitest';
import { Badge } from 'antd';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderSnWithTooltip } from '../snTooltip';

const LONG_SN = '1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZ-0987654321-VERYLONGSN';

describe('renderSnWithTooltip (issue #224)', () => {
  it('超长 SN：文本元素用 flex 收缩 + ellipsis 约束（不溢出遮挡后列）', () => {
    render(<div style={{ width: 160 }}>{renderSnWithTooltip(LONG_SN)}</div>);
    const cell = screen.getByText(LONG_SN);
    // 关键：flex:1 + minWidth:0 让 flex 子项可收缩到列宽以内，再 ellipsis 截断
    expect(cell.style.overflow).toBe('hidden');
    expect(cell.style.textOverflow).toBe('ellipsis');
    expect(cell.style.whiteSpace).toBe('nowrap');
    expect(cell.style.minWidth).toBe('0px');
    expect(cell.style.flex).toContain('1');
  });

  it('超长 SN：悬浮时 antd Tooltip 弹出完整 SN（深浅主题均完整可读）', async () => {
    const user = userEvent.setup();
    render(<div>{renderSnWithTooltip(LONG_SN)}</div>);
    const cell = screen.getByText(LONG_SN);
    await user.hover(cell);
    // Tooltip 渲染到 body portal，role=tooltip，内容为完整 SN
    const tip = await screen.findByRole('tooltip');
    expect(tip).toHaveTextContent(LONG_SN);
  });

  it('带未读小红点前缀：前缀固定占位、SN 文本仍可收缩', () => {
    render(
      <div style={{ width: 160 }}>
        {renderSnWithTooltip(LONG_SN, <Badge status="error" />)}
      </div>,
    );
    // 前缀 Badge 渲染出来
    expect(document.querySelector('.ant-badge')).not.toBeNull();
    // SN 文本仍带 ellipsis 收缩约束
    const cell = screen.getByText(LONG_SN);
    expect(cell.style.minWidth).toBe('0px');
    expect(cell.style.textOverflow).toBe('ellipsis');
  });

  it('空 SN：不挂 Tooltip，仅渲染空文本占位（不出现空气泡）', () => {
    const { container } = render(<div>{renderSnWithTooltip('')}</div>);
    const span = container.querySelector('span');
    expect(span).not.toBeNull();
    expect(span?.textContent).toBe('');
    // 空值不应触发任何 tooltip 角色节点
    expect(screen.queryByRole('tooltip')).toBeNull();
  });
});
