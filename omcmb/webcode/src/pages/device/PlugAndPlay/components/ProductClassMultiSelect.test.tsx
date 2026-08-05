import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import ProductClassMultiSelect from './ProductClassMultiSelect';

describe('ProductClassMultiSelect', () => {
  it('allows multiple product classes to be selected', () => {
    render(
      <ProductClassMultiSelect
        aria-label="product classes"
        value={['FAP/A', 'FAP/B']}
        options={[
          { label: 'FAP/A', value: 'FAP/A' },
          { label: 'FAP/B', value: 'FAP/B' },
        ]}
      />,
    );

    fireEvent.mouseDown(screen.getByRole('combobox'));
    expect(document.querySelectorAll('.ant-select-item-option-selected')).toHaveLength(2);
    expect(screen.getAllByTitle('FAP/A').length).toBeGreaterThan(0);
    expect(screen.getAllByTitle('FAP/B').length).toBeGreaterThan(0);
  });
});
