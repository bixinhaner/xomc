import { App } from 'antd';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {
  useAccessEntries,
  useAccessCandidates,
  useAccessListImports,
  useAccessPolicies,
  useAccessPolicy,
	useAccessPolicyDifference,
  useCommitAccessListImport,
  useCreateAccessPolicyDraft,
  useDeleteAccessPolicyDraft,
  useDisableAccessEntries,
  usePreviewAccessListImport,
  usePublishAccessPolicy,
  useRollbackAccessListImport,
	useRollbackAccessPolicy,
  useReviewAccessCandidate,
  useRuleDimensionImports,
  useUpdateAccessPolicyDraft,
  useUpsertAccessEntry,
  useUpsertAccessEntries,
} from '@core/hooks/api/useDeviceAccess';
import { AccessListPanel, CandidatePanel, PolicyPanel } from './GovernancePanels';

vi.mock('@core/hooks/api/useDeviceAccess', () => ({
  useAccessPolicies: vi.fn(),
  useAccessPolicy: vi.fn(),
	useAccessPolicyDifference: vi.fn(),
  useCreateAccessPolicyDraft: vi.fn(),
  useDeleteAccessPolicyDraft: vi.fn(),
  usePublishAccessPolicy: vi.fn(),
  useAccessEntries: vi.fn(),
  useUpsertAccessEntry: vi.fn(),
  useUpsertAccessEntries: vi.fn(),
  useDisableAccessEntries: vi.fn(),
  useAccessListImports: vi.fn(),
  usePreviewAccessListImport: vi.fn(),
  useCommitAccessListImport: vi.fn(),
  useRollbackAccessListImport: vi.fn(),
	useRollbackAccessPolicy: vi.fn(),
  usePreviewRuleDimensionImport: vi.fn(() => ({ isPending: false, mutateAsync: vi.fn() })),
  useClearRuleDimension: vi.fn(() => ({ isPending: false, mutateAsync: vi.fn() })),
  useRuleDimensionImports: vi.fn(),
  useUpdateAccessPolicyDraft: vi.fn(),
  useAccessCandidates: vi.fn(),
  useReviewAccessCandidate: vi.fn(),
}));

const t = (key: string) => key;

describe('PolicyPanel', () => {
  it('shows publish loading only on the policy version being published', async () => {
    let completePublish: (() => void) | undefined;
    const publishPromise = new Promise<void>((resolve) => { completePublish = resolve; });
    vi.mocked(useAccessPolicies).mockReturnValue({
      data: {
        items: [
          { id: 'draft-1', carrier: 'cmcc', name: 'policy', version: 1, status: 'draft', default_action: 'reject', created_at: '2026-08-17T00:00:00Z' },
          { id: 'draft-2', carrier: 'cmcc', name: 'policy', version: 2, status: 'draft', default_action: 'reject', created_at: '2026-08-17T00:00:00Z' },
        ],
        total: 2,
        page: 1,
        pageSize: 20,
      },
      error: null,
      isLoading: false,
      refetch: vi.fn(),
    } as never);
    vi.mocked(useAccessPolicy).mockReturnValue({ data: undefined, error: null, isLoading: false } as never);
    vi.mocked(useCreateAccessPolicyDraft).mockReturnValue({ isPending: false, mutateAsync: vi.fn() } as never);
    vi.mocked(useUpdateAccessPolicyDraft).mockReturnValue({ isPending: false, mutateAsync: vi.fn() } as never);
    vi.mocked(useDeleteAccessPolicyDraft).mockReturnValue({ isPending: false, mutateAsync: vi.fn() } as never);
		vi.mocked(useAccessPolicyDifference).mockReturnValue({ isPending: false, mutateAsync: vi.fn() } as never);
		vi.mocked(useRollbackAccessPolicy).mockReturnValue({ isPending: false, mutateAsync: vi.fn() } as never);
    vi.mocked(usePublishAccessPolicy).mockReturnValue({ isPending: true, mutateAsync: vi.fn(() => publishPromise) } as never);
    vi.mocked(useCommitAccessListImport).mockReturnValue({ isPending: false, mutateAsync: vi.fn() } as never);
    vi.mocked(useRollbackAccessListImport).mockReturnValue({ isPending: false, mutateAsync: vi.fn() } as never);
    vi.mocked(useRuleDimensionImports).mockReturnValue({ data: { items: [], total: 0, page: 1, pageSize: 50 }, error: null, isLoading: false } as never);

    const user = userEvent.setup();
    render(<App><PolicyPanel operatorCode="cmcc" t={t} allowed /></App>);
    const rows = screen.getAllByRole('row').slice(1);
    const firstPublish = within(rows[0]).getByRole('button', { name: 'deviceAccess.publish' });
    const secondPublish = within(rows[1]).getByRole('button', { name: 'deviceAccess.publish' });

    await user.click(secondPublish);
    await user.click(await screen.findByRole('button', { name: 'common.confirm' }));

    expect(firstPublish).not.toHaveClass('ant-btn-loading');
    expect(secondPublish).toHaveClass('ant-btn-loading');
    completePublish?.();
  });

  it('clears draft save loading after server validation fails', async () => {
    const createDraft = vi.fn().mockRejectedValue(new Error('internal error'));
    const resetCreate = vi.fn();
    const resetUpdate = vi.fn();
    vi.mocked(useAccessPolicies).mockReturnValue({
      data: { items: [], total: 0, page: 1, pageSize: 20 },
      error: null,
      isLoading: false,
      refetch: vi.fn(),
    } as never);
    vi.mocked(useAccessPolicy).mockReturnValue({ data: undefined, error: null, isLoading: false } as never);
    vi.mocked(useCreateAccessPolicyDraft).mockReturnValue({ isPending: false, mutateAsync: createDraft, reset: resetCreate } as never);
    vi.mocked(useUpdateAccessPolicyDraft).mockReturnValue({ isPending: false, mutateAsync: vi.fn(), reset: resetUpdate } as never);
    vi.mocked(useDeleteAccessPolicyDraft).mockReturnValue({ isPending: false, mutateAsync: vi.fn() } as never);
    vi.mocked(useAccessPolicyDifference).mockReturnValue({ isPending: false, mutateAsync: vi.fn() } as never);
    vi.mocked(useRollbackAccessPolicy).mockReturnValue({ isPending: false, mutateAsync: vi.fn() } as never);
    vi.mocked(usePublishAccessPolicy).mockReturnValue({ isPending: false, mutateAsync: vi.fn() } as never);
    vi.mocked(useCommitAccessListImport).mockReturnValue({ isPending: false, mutateAsync: vi.fn() } as never);
    vi.mocked(useRollbackAccessListImport).mockReturnValue({ isPending: false, mutateAsync: vi.fn() } as never);
    vi.mocked(useRuleDimensionImports).mockReturnValue({ data: { items: [], total: 0, page: 1, pageSize: 50 }, error: null, isLoading: false } as never);

    const user = userEvent.setup();
    render(<App><PolicyPanel operatorCode="cmcc" t={t} allowed /></App>);
    await user.click(screen.getByRole('button', { name: 'plus deviceAccess.createPolicyDraft' }));
    await user.type(screen.getByLabelText('deviceAccess.policySetName'), 'review-policy');
    const save = screen.getByRole('button', { name: 'deviceAccess.saveDraft' });
    await user.click(save);

    await waitFor(() => expect(createDraft).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(save).not.toHaveClass('ant-btn-loading'));
    expect(resetCreate).toHaveBeenCalledTimes(1);
    expect(resetUpdate).toHaveBeenCalledTimes(1);
  });
});

describe('AccessListPanel', () => {
  it('lists every selected device and submits one atomic batch disable request', async () => {
    const disableMany = vi.fn().mockResolvedValue(undefined);
    vi.mocked(useAccessEntries).mockReturnValue({
      data: {
        items: [
          { id: 'entry-1', carrier: 'cmcc', entry_type: 'deny', identity_type: 'serial_number', identity_value: 'SN-1', product_name: 'QRTB', reason_code: '', reason: 'test', valid_from: '2026-08-18T00:00:00Z', status: 'active', created_at: '2026-08-18T00:00:00Z', updated_at: '2026-08-18T00:00:00Z' },
          { id: 'entry-2', carrier: 'cmcc', entry_type: 'deny', identity_type: 'serial_number', identity_value: 'SN-2', product_name: 'MLN', reason_code: '', reason: 'test', valid_from: '2026-08-18T00:00:00Z', status: 'active', created_at: '2026-08-18T00:00:00Z', updated_at: '2026-08-18T00:00:00Z' },
        ],
        total: 2,
        page: 1,
        pageSize: 20,
      },
      error: null,
      isLoading: false,
      refetch: vi.fn(),
    } as never);
    vi.mocked(useUpsertAccessEntry).mockReturnValue({ isPending: false, mutateAsync: vi.fn() } as never);
    vi.mocked(useUpsertAccessEntries).mockReturnValue({ isPending: false, mutateAsync: vi.fn() } as never);
    vi.mocked(useDisableAccessEntries).mockReturnValue({ isPending: false, mutateAsync: disableMany } as never);
    vi.mocked(useAccessListImports).mockReturnValue({ data: { items: [], total: 0, page: 1, pageSize: 50 }, error: null, isLoading: false } as never);
    vi.mocked(usePreviewAccessListImport).mockReturnValue({ isPending: false, mutateAsync: vi.fn() } as never);
    vi.mocked(useCommitAccessListImport).mockReturnValue({ isPending: false, mutateAsync: vi.fn() } as never);
    vi.mocked(useRollbackAccessListImport).mockReturnValue({ isPending: false, mutateAsync: vi.fn() } as never);
    vi.mocked(useRuleDimensionImports).mockReturnValue({ data: { items: [], total: 0, page: 1, pageSize: 50 }, error: null, isLoading: false } as never);

    const user = userEvent.setup();
    render(<App><AccessListPanel operatorCode="cmcc" t={t} allowed businessEnabled /></App>);
    const checkboxes = screen.getAllByRole('checkbox');
    await user.click(checkboxes[1]);
    await user.click(checkboxes[2]);
    await user.click(screen.getByRole('button', { name: /deviceAccess.batchDisable \(2\)/ }));

    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByText('SN-1')).toBeInTheDocument();
    expect(within(dialog).getByText(/QRTB/)).toBeInTheDocument();
    expect(within(dialog).getByText('SN-2')).toBeInTheDocument();
    expect(within(dialog).getByText(/MLN/)).toBeInTheDocument();
    await user.type(within(dialog).getByPlaceholderText('deviceAccess.batchDisableReason'), 'obsolete devices');
    await user.click(within(dialog).getByRole('button', { name: 'common.confirm' }));

    expect(disableMany).toHaveBeenCalledTimes(1);
    expect(disableMany).toHaveBeenCalledWith({
      operatorCode: 'cmcc',
      entryType: 'deny',
      serialNumbers: ['SN-1', 'SN-2'],
      reason: 'obsolete devices',
    });
  });
});

describe('CandidatePanel', () => {
  it('shows the unknown-device identity before submitting a review outcome', async () => {
    const candidate = {
      id: 'candidate-1', carrier: 'cmcc', serial_number: 'UNKNOWN-SN', oui: '001122',
      product_class: 'QRTB', software_version: '1.0', observed_remote_ip: '192.0.2.10',
      first_seen_at: '2026-08-21T01:00:00Z', last_seen_at: '2026-08-21T02:00:00Z',
      inform_count: 2, review_status: 'pending', expires_at: '2026-08-22T01:00:00Z',
    } as const;
    const mutateAsync = vi.fn().mockResolvedValue(undefined);
    vi.mocked(useAccessCandidates).mockReturnValue({
      data: { items: [candidate], total: 1, page: 1, pageSize: 20 }, error: null, isLoading: false, refetch: vi.fn(),
    } as never);
    vi.mocked(useReviewAccessCandidate).mockReturnValue({ isPending: false, mutateAsync } as never);

    const user = userEvent.setup();
    render(<App><CandidatePanel operatorCode="cmcc" t={t} allowed businessEnabled initialSerialNumber="UNKNOWN-SN" /></App>);

    expect(vi.mocked(useAccessCandidates).mock.calls.at(-1)?.[0].serialNumber).toBe('UNKNOWN-SN');
    await user.click(screen.getByRole('button', { name: /deviceAccess.review.allow/ }));
    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByText('UNKNOWN-SN')).toBeInTheDocument();
    expect(within(dialog).getByText('192.0.2.10')).toBeInTheDocument();
    await user.type(within(dialog).getByPlaceholderText('deviceAccess.reviewReasonPlaceholder'), 'identity confirmed');
    await user.click(within(dialog).getByRole('button', { name: 'common.confirm' }));

    expect(mutateAsync).toHaveBeenCalledWith({ operatorCode: 'cmcc', candidateId: 'candidate-1', outcome: 'allow', reason: 'identity confirmed' });
  });
});
