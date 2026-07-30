import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { renderDeviceStatus } from './shared.render';

const messages: Record<string, string> = {
  'ufte.status.completed': 'Completed',
  'ufte.status.rollbackChecking': 'Checking rollback availability',
  'ufte.status.rollingBack': 'Rolling back',
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

  it('renders rollback-specific running states without download wording', () => {
    render(
      <>
        {renderDeviceStatus('rollback_checking' as never, t)}
        {renderDeviceStatus('rolling_back' as never, t)}
      </>,
    );

    expect(screen.getByText('Checking rollback availability')).toBeInTheDocument();
    expect(screen.getByText('Rolling back')).toBeInTheDocument();
    expect(screen.queryByText('ufte.status.downloading')).not.toBeInTheDocument();
  });
});
