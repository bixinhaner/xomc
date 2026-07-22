import { fireEvent, render, screen } from '@testing-library/react';
import { ConfigProvider } from 'antd';
import GpsSyncTrigger from './GpsSyncTrigger';

describe('GpsSyncTrigger', () => {
  it('renders the yellow inconsistency warning and handles clicks', () => {
    const onClick = vi.fn();

    const { container } = render(
      <ConfigProvider theme={{ token: { colorWarning: '#f0a500' } }}>
        <GpsSyncTrigger
          label="Review and synchronize GPS coordinates"
          onClick={onClick}
        />
      </ConfigProvider>,
    );

    const button = screen.getByRole('button', {
      name: 'Review and synchronize GPS coordinates',
    });
    expect(container.querySelector('.anticon-warning')).toBeInTheDocument();
    expect(button).toHaveClass('device-gps-sync-trigger');
    expect(button).toHaveStyle({ width: '20px', minWidth: '20px', height: '20px' });
    expect(button).toHaveStyle({ color: '#f0a500' });

    fireEvent.click(button);
    expect(onClick).toHaveBeenCalledOnce();
  });
});
