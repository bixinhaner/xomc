import { fireEvent, render, screen } from '@testing-library/react';
import GpsSyncTrigger from './GpsSyncTrigger';

describe('GpsSyncTrigger', () => {
  it('renders a discoverable GPS location action and handles clicks', () => {
    const onClick = vi.fn();

    const { container } = render(
      <GpsSyncTrigger
        label="Review and synchronize GPS coordinates"
        onClick={onClick}
      />,
    );

    const button = screen.getByRole('button', {
      name: 'Review and synchronize GPS coordinates',
    });
    expect(container.querySelector('.anticon-aim')).toBeInTheDocument();

    fireEvent.click(button);
    expect(onClick).toHaveBeenCalledOnce();
  });
});
