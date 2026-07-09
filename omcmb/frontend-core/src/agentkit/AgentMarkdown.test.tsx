import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { AgentMarkdown } from './AgentMarkdown';

describe('AgentMarkdown', () => {
  it('renders standard GFM lists and task lists', () => {
    const { container } = render(
      <AgentMarkdown content={['- 在线设备：0', '- [x] 已检查', '- [ ] 待确认'].join('\n')} />
    );

    expect(screen.getByText('在线设备：0')).toBeTruthy();
    expect(container.querySelectorAll('li')).toHaveLength(3);
    expect(container.querySelectorAll('input[type="checkbox"]')).toHaveLength(2);
  });

  it('does not rewrite compact non-standard markdown text', () => {
    const { container } = render(<AgentMarkdown content="当前设备总数：0-在线设备：0-离线设备：0结论：无在线设备。" />);

    expect(container.querySelector('li')).toBeNull();
    expect(container.textContent).toContain('当前设备总数：0-在线设备：0-离线设备：0结论：无在线设备。');
  });

  it('renders GFM tables with a scroll wrapper', () => {
    const { container } = render(
      <AgentMarkdown content={['| 指标 | 数值 |', '| --- | ---: |', '| 在线设备 | 0 |'].join('\n')} />
    );

    expect(container.querySelector('.agent-render-table-scroll')).toBeTruthy();
    expect(container.querySelector('table.agent-render-table')).toBeTruthy();
    expect(screen.getByText('在线设备')).toBeTruthy();
  });

  it('renders math formulas through katex', () => {
    const { container } = render(<AgentMarkdown content="在线率 $0/0$ 暂不可计算。" />);

    expect(container.querySelector('.katex')).toBeTruthy();
  });
});
