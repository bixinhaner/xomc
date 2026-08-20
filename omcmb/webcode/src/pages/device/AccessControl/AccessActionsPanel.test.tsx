import { App } from 'antd';
import { fireEvent, render, screen } from '@testing-library/react';
import { useAccessActionAttempts, useAccessActions, useRetryAccessAction } from '@core/hooks/api/useDeviceAccess';
import AccessActionsPanel from './AccessActionsPanel';

vi.mock('@core/hooks/api/useDeviceAccess', () => ({
  useAccessActionAttempts: vi.fn(),
  useAccessActions: vi.fn(),
  useRetryAccessAction: vi.fn(),
}));

it('shows an explicit evidence gap for a quarantined historical RF action', async () => {
  vi.mocked(useAccessActions).mockReturnValue({
    data: {
      items: [{
        id: 'action-1', decision_id: 'decision-1', action_type: 'rf_off', direction: 'contain',
        status: 'dead', owned_rf_change: false, attempts: 0, max_attempts: 3,
        last_failure_code: 'evidence_missing', manual_repair_required: true,
        error_message: 'historical RF action has no attempt evidence',
        created_at: '2026-08-20T00:00:00Z', updated_at: '2026-08-20T00:00:00Z',
        serial_number: 'SN-1', product_name: 'MLN',
      }],
      total: 1,
      page: 1,
      page_size: 20,
    },
    error: null,
    isLoading: false,
    refetch: vi.fn(),
  } as never);
  vi.mocked(useAccessActionAttempts).mockReturnValue({ data: [], error: null, isLoading: false } as never);
  vi.mocked(useRetryAccessAction).mockReturnValue({ isPending: false, mutateAsync: vi.fn() } as never);

  const { container } = render(<App><AccessActionsPanel operatorCode="cmcc" t={(key) => key} allowed /></App>);
  const expand = container.querySelector<HTMLButtonElement>('.ant-table-row-expand-icon');
  expect(expand).not.toBeNull();
  fireEvent.click(expand!);

  expect(await screen.findAllByText('deviceAccess.actionEvidenceMissing')).toHaveLength(2);
  expect(screen.queryByText('deviceAccess.attemptPhase.baseline_gpv')).not.toBeInTheDocument();
});
