import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';

vi.mock('@/hooks/useT', () => ({
  useT: () => (id: string) => id,
}));

import StepBar from '../StepBar';

describe('StepBar', () => {
  it('renders 4 step titles using i18n keys by default', () => {
    render(<StepBar current={1} />);
    expect(screen.getByText('mml.console.stepBar.step1')).toBeInTheDocument();
    expect(screen.getByText('mml.console.stepBar.step2')).toBeInTheDocument();
    expect(screen.getByText('mml.console.stepBar.step3')).toBeInTheDocument();
    expect(screen.getByText('mml.console.stepBar.step4')).toBeInTheDocument();
  });

  it('labels prop overrides i18n defaults', () => {
    render(
      <StepBar
        current={2}
        labels={{ step1: 'Pick', step2: 'Cmd', step3: 'Config', step4: 'Done' }}
      />,
    );
    expect(screen.getByText('Pick')).toBeInTheDocument();
    expect(screen.getByText('Done')).toBeInTheDocument();
    expect(screen.queryByText('mml.console.stepBar.step1')).not.toBeInTheDocument();
  });

  it('current=3 maps to antd Steps index 2 (process state on 3rd item)', () => {
    const { container } = render(<StepBar current={3} />);
    const processItem = container.querySelector('.ant-steps-item-process');
    expect(processItem).not.toBeNull();
    // antd 给当前激活项加 ant-steps-item-process；序号 1-based 第 3 项
    const title = processItem?.querySelector('.ant-steps-item-title')?.textContent;
    expect(title).toBe('mml.console.stepBar.step3');
  });
});
