import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import ProductClassSelect from './ProductClassSelect';

describe('ProductClassSelect', () => {
  it('keeps the trigger and long options readable', () => {
    const productClass = 'FAP/BSC7079B243';
    render(
      <ProductClassSelect
        aria-label="product class"
        options={[{ label: productClass, value: productClass }]}
      />,
    );

    const selector = screen.getByRole('combobox').closest('.ant-select');
    expect(selector).toHaveStyle({ width: '100%', maxWidth: '420px' });

    fireEvent.mouseDown(screen.getByRole('combobox'));
    expect(document.querySelector('.ant-select-dropdown')).toHaveStyle({ width: '420px' });
    expect(screen.getAllByTitle(productClass)).not.toHaveLength(0);
  });
});
