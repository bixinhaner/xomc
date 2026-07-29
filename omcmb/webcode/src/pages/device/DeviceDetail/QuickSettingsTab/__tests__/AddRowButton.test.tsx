import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { AddRowButton } from '../AddRowButton';

describe('AddRowButton', () => {
  it('uses the shared dashed full-width style and forwards clicks', () => {
    const onClick = vi.fn();
    render(<AddRowButton onClick={onClick}>新增一行</AddRowButton>);

    const button = screen.getByRole('button', { name: /新增一行/ });
    expect(button).toHaveClass('ant-btn-dashed', 'ant-btn-block');

    fireEvent.click(button);
    expect(onClick).toHaveBeenCalledOnce();
  });
});
