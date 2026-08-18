import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { App } from 'antd';
import { useAccessDetail, useAccessStates, useReevaluateAccessDevice } from '@core/hooks/api/useDeviceAccess';
import type { AccessDetail, AccessStateItem, DecisionItem } from '@core/services/api/deviceAccessApi';
import AccessStatesPanel from './AccessStatesPanel';

vi.mock('@core/hooks/api/useDeviceAccess', () => ({
  useAccessStates: vi.fn(),
  useAccessDetail: vi.fn(),
  useReevaluateAccessDevice: vi.fn(),
}));

const t = (key: string) => key;
const state: AccessStateItem = {
  carrier: 'cmcc',
  serial_number: 'SN-1',
  product_name: 'QRTB',
  state: 'rejected',
  effective_decision: 'reject',
  reason_code: 'rule_mismatch',
  evidence_version: 12,
  decision_version: 12,
  normal_tasks_frozen: true,
  last_decided_at: '2026-08-17T03:28:10Z',
  updated_at: '2026-08-17T03:28:10Z',
};

function decision(version: number): DecisionItem {
  return {
    id: `decision-${version}`,
    trigger_type: 'evidence_updated',
    new_state: 'rejected',
    decision: 'reject',
    reason_code: 'rule_mismatch',
    evidence_version: version,
    decision_version: version,
    occurred_at: '2026-08-17T03:28:10Z',
    checks: [],
  };
}

describe('AccessStatesPanel', () => {
  beforeEach(() => {
    vi.mocked(useReevaluateAccessDevice).mockReturnValue({ isPending: false, mutateAsync: vi.fn() } as never);
  });

  it('paginates long decision history inside the detail drawer', async () => {
    const detail: AccessDetail = {
      state,
      decisions: Array.from({ length: 12 }, (_, index) => decision(12 - index)),
      evidence: [],
      actions: [],
    };
    vi.mocked(useAccessStates).mockReturnValue({
      data: { items: [state], total: 1, page: 1, pageSize: 20 },
      error: null,
      isLoading: false,
      refetch: vi.fn(),
    } as never);
    vi.mocked(useAccessDetail).mockReturnValue({ data: detail, error: null, isLoading: false } as never);

    const user = userEvent.setup();
    render(<App><AccessStatesPanel operatorCode="cmcc" t={t} canReevaluate /></App>);
    await user.click(screen.getByRole('button', { name: /common.detail/ }));

    expect(screen.getByText('v12')).toBeInTheDocument();
    expect(screen.getByText('v3')).toBeInTheDocument();
    expect(screen.queryByText('v2')).not.toBeInTheDocument();

    await user.click(screen.getByTitle('2'));
    expect(screen.getByText('v2')).toBeInTheDocument();
    expect(screen.getByText('v1')).toBeInTheDocument();
    expect(screen.queryByText('v12')).not.toBeInTheDocument();
  });

  it('submits a single-device reevaluation when permitted', async () => {
    const mutateAsync = vi.fn().mockResolvedValue(undefined);
    vi.mocked(useReevaluateAccessDevice).mockReturnValue({ isPending: false, mutateAsync } as never);
    vi.mocked(useAccessStates).mockReturnValue({
      data: { items: [state], total: 1, page: 1, pageSize: 20 },
      error: null,
      isLoading: false,
      refetch: vi.fn(),
    } as never);
    vi.mocked(useAccessDetail).mockReturnValue({ data: undefined, error: null, isLoading: false } as never);

    const user = userEvent.setup();
    render(<App><AccessStatesPanel operatorCode="cmcc" t={t} canReevaluate /></App>);
    expect(screen.getByText('QRTB')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /deviceAccess.reevaluate/ }));

    expect(mutateAsync).toHaveBeenCalledWith({ operatorCode: 'cmcc', serialNumber: 'SN-1' });
  });

  it('does not submit reevaluation while the business switch is off', async () => {
    const mutateAsync = vi.fn();
    vi.mocked(useReevaluateAccessDevice).mockReturnValue({ isPending: false, mutateAsync } as never);
    vi.mocked(useAccessStates).mockReturnValue({
      data: { items: [state], total: 1, page: 1, pageSize: 20 },
      error: null,
      isLoading: false,
      refetch: vi.fn(),
    } as never);
    vi.mocked(useAccessDetail).mockReturnValue({ data: undefined, error: null, isLoading: false } as never);

    render(<App><AccessStatesPanel operatorCode="cmcc" t={t} canReevaluate businessEnabled={false} /></App>);

    expect(screen.getByRole('button', { name: /deviceAccess.reevaluate/ })).toBeDisabled();
    expect(mutateAsync).not.toHaveBeenCalled();
  });
});
