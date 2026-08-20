import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { App } from 'antd';
import {
  useAccessActions,
  useAccessDecisions,
  useAccessDetail,
  useAccessEvidence,
  useAccessIdentitySnapshots,
  useAccessManualOperations,
  useAccessNotifications,
  useAccessStateSummary,
  useAccessStates,
  useArchiveAccessDecision,
  useReevaluateAccessDevice,
  useRestoreAccessDecision,
  useUpsertAccessEntry,
} from '@core/hooks/api/useDeviceAccess';
import type { AccessDetail, AccessStateItem, DecisionItem } from '@core/services/api/deviceAccessApi';
import AccessStatesPanel from './AccessStatesPanel';

vi.mock('@core/hooks/api/useDeviceAccess', () => ({
  useAccessStates: vi.fn(),
  useAccessStateSummary: vi.fn(),
  useAccessDetail: vi.fn(),
  useAccessDecisions: vi.fn(),
  useAccessIdentitySnapshots: vi.fn(),
  useAccessEvidence: vi.fn(),
  useAccessActions: vi.fn(),
  useAccessManualOperations: vi.fn(),
  useAccessNotifications: vi.fn(),
  useAccessActionAttempts: vi.fn(),
  useArchiveAccessDecision: vi.fn(),
  useRestoreAccessDecision: vi.fn(),
  useReevaluateAccessDevice: vi.fn(),
  useUpsertAccessEntry: vi.fn(),
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
    const emptyPage = { data: { items: [], total: 0, page: 1, pageSize: 10 }, error: null, isLoading: false } as never;
    vi.mocked(useAccessStateSummary).mockReturnValue({ data: { accepted: 0, rejected: 1, review_required: 0, revoked: 0, total: 1 }, error: null, refetch: vi.fn() } as never);
    vi.mocked(useAccessDetail).mockReturnValue({ data: undefined, error: null, isLoading: false } as never);
    vi.mocked(useAccessDecisions).mockReturnValue(emptyPage);
    vi.mocked(useAccessIdentitySnapshots).mockReturnValue(emptyPage);
    vi.mocked(useAccessEvidence).mockReturnValue(emptyPage);
    vi.mocked(useAccessActions).mockReturnValue(emptyPage);
    vi.mocked(useAccessManualOperations).mockReturnValue(emptyPage);
    vi.mocked(useAccessNotifications).mockReturnValue(emptyPage);
    vi.mocked(useArchiveAccessDecision).mockReturnValue({ isPending: false, mutateAsync: vi.fn() } as never);
    vi.mocked(useRestoreAccessDecision).mockReturnValue({ isPending: false, mutateAsync: vi.fn() } as never);
    vi.mocked(useReevaluateAccessDevice).mockReturnValue({ isPending: false, mutateAsync: vi.fn() } as never);
    vi.mocked(useUpsertAccessEntry).mockReturnValue({ isPending: false, mutateAsync: vi.fn() } as never);
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
    vi.mocked(useAccessDecisions).mockImplementation((params) => ({
      data: {
        items: params.page === 1 ? detail.decisions.slice(0, 10) : detail.decisions.slice(10),
        total: detail.decisions.length,
        page: params.page,
        pageSize: 10,
      },
      error: null,
      isLoading: false,
    } as never));

    const user = userEvent.setup();
    render(<App><AccessStatesPanel operatorCode="cmcc" t={t} canReevaluate canManageList /></App>);
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
    render(<App><AccessStatesPanel operatorCode="cmcc" t={t} canReevaluate canManageList /></App>);
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

    render(<App><AccessStatesPanel operatorCode="cmcc" t={t} canReevaluate canManageList businessEnabled={false} /></App>);

    expect(screen.getByRole('button', { name: /deviceAccess.reevaluate/ })).toBeDisabled();
    expect(screen.getByText('deviceAccess.allowed')).toBeInTheDocument();
    expect(mutateAsync).not.toHaveBeenCalled();
  });

  it('rejects malformed policy and rule UUID filters before querying', async () => {
    vi.mocked(useAccessStates).mockReturnValue({
      data: { items: [], total: 0, page: 1, pageSize: 20 }, error: null, isLoading: false, refetch: vi.fn(),
    } as never);
    const user = userEvent.setup();
    render(<App><AccessStatesPanel operatorCode="cmcc" t={t} canReevaluate canManageList /></App>);

    await user.click(screen.getByRole('button', { name: /deviceAccess.advancedFilters/ }));
    await user.type(screen.getByPlaceholderText('deviceAccess.policyVersionId'), 'not-a-uuid');
    await user.click(screen.getByRole('button', { name: /common.search/ }));

    expect(await screen.findByText('deviceAccess.invalidUuidFilter')).toBeInTheDocument();
    expect(vi.mocked(useAccessStates).mock.calls.at(-1)?.[0].policyVersionId).toBeUndefined();
  });

  it('routes an unknown device to candidate review without exposing list shortcuts', async () => {
    const onReviewCandidate = vi.fn();
    vi.mocked(useAccessStates).mockReturnValue({
      data: { items: [{ ...state, state: 'review_required', effective_decision: 'review', reason_code: 'identity_unverified', candidate_id: 'candidate-1' }], total: 1, page: 1, pageSize: 20 },
      error: null,
      isLoading: false,
      refetch: vi.fn(),
    } as never);

    const user = userEvent.setup();
    render(<App><AccessStatesPanel operatorCode="cmcc" t={t} canReevaluate canManageList onReviewCandidate={onReviewCandidate} /></App>);

    expect(screen.queryByRole('button', { name: /deviceAccess.addToAllowlist/ })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /deviceAccess.addToDenylist/ })).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /deviceAccess.reviewCandidate/ }));
    expect(onReviewCandidate).toHaveBeenCalledWith('SN-1');
  });

  it('adds a rejected device to the allowlist only after reason confirmation', async () => {
    const mutateAsync = vi.fn().mockResolvedValue(undefined);
    vi.mocked(useUpsertAccessEntry).mockReturnValue({ isPending: false, mutateAsync } as never);
    vi.mocked(useAccessStates).mockReturnValue({
      data: { items: [state], total: 1, page: 1, pageSize: 20 },
      error: null,
      isLoading: false,
      refetch: vi.fn(),
    } as never);
    vi.mocked(useAccessDetail).mockReturnValue({ data: undefined, error: null, isLoading: false } as never);

    const user = userEvent.setup();
    render(<App><AccessStatesPanel operatorCode="cmcc" t={t} canReevaluate canManageList /></App>);
    await user.click(screen.getByRole('button', { name: /deviceAccess.addToAllowlist/ }));

    const confirm = screen.getByRole('button', { name: 'common.confirm' });
    expect(confirm).toBeDisabled();
    await user.type(screen.getByPlaceholderText('deviceAccess.quickListReasonRequired'), '现场审核放行');
    await user.click(confirm);

    expect(mutateAsync).toHaveBeenCalledWith({
      operatorCode: 'cmcc',
      entryType: 'allow',
      serialNumber: 'SN-1',
      reason: '现场审核放行',
    });
  });

  it('prevents archiving the current decision and requires a reason for historical decisions', async () => {
    const archive = vi.fn().mockResolvedValue(undefined);
    vi.mocked(useArchiveAccessDecision).mockReturnValue({ isPending: false, mutateAsync: archive } as never);
    vi.mocked(useAccessStates).mockReturnValue({
      data: { items: [state], total: 1, page: 1, pageSize: 20 }, error: null, isLoading: false, refetch: vi.fn(),
    } as never);
    vi.mocked(useAccessDetail).mockReturnValue({
      data: { state, decisions: [], evidence: [], actions: [] }, error: null, isLoading: false,
    } as never);
    vi.mocked(useAccessDecisions).mockReturnValue({
      data: { items: [decision(12), decision(11)], total: 2, page: 1, pageSize: 10 }, error: null, isLoading: false,
    } as never);

    const user = userEvent.setup();
    render(<App><AccessStatesPanel operatorCode="cmcc" t={t} canReevaluate canManageList canArchive /></App>);
    await user.click(screen.getByRole('button', { name: /common.detail/ }));

    const archiveButtons = screen.getAllByRole('button', { name: 'deviceAccess.archive.action' });
    expect(archiveButtons).toHaveLength(2);
    expect(archiveButtons[0]).toBeDisabled();
    expect(archiveButtons[1]).toBeEnabled();
    await user.click(archiveButtons[1]);

    const confirm = screen.getByRole('button', { name: 'common.confirm' });
    expect(confirm).toBeDisabled();
    await user.type(screen.getByPlaceholderText('deviceAccess.archive.reasonRequired'), '重复历史记录');
    await user.click(confirm);
    expect(archive).toHaveBeenCalledWith({ operatorCode: 'cmcc', decisionId: 'decision-11', reason: '重复历史记录' });
  });
});
