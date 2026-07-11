import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { LoadingSpinner } from './index';

describe('LoadingSpinner', () => {
  it('空区域加载提示保持单行显示', () => {
    render(<LoadingSpinner tip="正在加载任务..." />);
    expect(screen.getByText('正在加载任务...')).toHaveStyle({ whiteSpace: 'nowrap' });
  });
});
