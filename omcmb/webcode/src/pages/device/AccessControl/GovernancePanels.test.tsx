import { App } from 'antd';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {
  useAccessEntries,
  useAccessPolicies,
  useAccessPolicy,
  useCreateAccessPolicyDraft,
  useDeleteAccessPolicyDraft,
  usePublishAccessPolicy,
  useUpsertAccessEntry,
} from '@core/hooks/api/useDeviceAccess';
import { AccessListPanel, PolicyPanel } from './GovernancePanels';

vi.mock('@core/hooks/api/useDeviceAccess', () => ({
  useAccessPolicies: vi.fn(),
  useAccessPolicy: vi.fn(),
  useCreateAccessPolicyDraft: vi.fn(),
  useDeleteAccessPolicyDraft: vi.fn(),
  usePublishAccessPolicy: vi.fn(),
  useAccessEntries: vi.fn(),
  useUpsertAccessEntry: vi.fn(),
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
    vi.mocked(useDeleteAccessPolicyDraft).mockReturnValue({ isPending: false, mutateAsync: vi.fn() } as never);
    vi.mocked(usePublishAccessPolicy).mockReturnValue({ isPending: true, mutateAsync: vi.fn(() => publishPromise) } as never);

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
});

describe('AccessListPanel', () => {
  it('lists every selected serial number and device name before batch disable', async () => {
    vi.mocked(useAccessEntries).mockReturnValue({
      data: {
        items: [
          { id: 'entry-1', carrier: 'cmcc', entry_type: 'deny', identity_type: 'serial_number', identity_value: 'SN-1', product_name: 'QRTB', reason_code: '', reason: 'test', valid_from: '2026-08-18T00:00:00Z', status: 'active', created_at: '2026-08-18T00:00:00Z', updated_at: '2026-08-18T00:00:00Z' },
          { id: 'entry-2', carrier: 'cmcc', entry_type: 'allow', identity_type: 'serial_number', identity_value: 'SN-2', product_name: 'MLN', reason_code: '', reason: 'test', valid_from: '2026-08-18T00:00:00Z', status: 'active', created_at: '2026-08-18T00:00:00Z', updated_at: '2026-08-18T00:00:00Z' },
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

    const user = userEvent.setup();
    render(<App><AccessListPanel operatorCode="cmcc" t={t} allowed /></App>);
    const checkboxes = screen.getAllByRole('checkbox');
    await user.click(checkboxes[1]);
    await user.click(checkboxes[2]);
    await user.click(screen.getByRole('button', { name: /deviceAccess.batchDisable \(2\)/ }));

    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByText('SN-1')).toBeInTheDocument();
    expect(within(dialog).getByText(/QRTB/)).toBeInTheDocument();
    expect(within(dialog).getByText('SN-2')).toBeInTheDocument();
    expect(within(dialog).getByText(/MLN/)).toBeInTheDocument();
  });
});
