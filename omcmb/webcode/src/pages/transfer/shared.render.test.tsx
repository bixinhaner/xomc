import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { renderDeviceStatus } from './shared.render';

const messages: Record<string, string> = {
  'ufte.status.completed': 'Completed',
};

const t = (id: string) => messages[id] ?? id;

describe('renderDeviceStatus', () => {
  it('keeps the status dot and text on one line', () => {
    render(<>{renderDeviceStatus('ended', t)}</>);

    const statusText = screen.getByText('Completed');
    const status = statusText.closest('[data-testid="ufte-device-status"]');

    expect(status).not.toBeNull();
    expect(status).toHaveStyle({
      display: 'inline-flex',
      alignItems: 'center',
      whiteSpace: 'nowrap',
    });
  });
});
