import { App } from 'antd';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { PolicyVersion } from '@core/services/api/deviceAccessApi';
import { PolicyEditorModal } from './PolicyEditorModal';

vi.mock('@core/hooks/api/useDeviceAccess', () => ({
  usePreviewRuleDimensionImport: () => ({ isPending: false, mutateAsync: vi.fn() }),
  useCommitAccessListImport: () => ({ isPending: false, mutateAsync: vi.fn() }),
  useClearRuleDimension: () => ({ isPending: false, mutateAsync: vi.fn() }),
  useRuleDimensionImports: () => ({ data: { items: [], total: 0, page: 1, pageSize: 50 }, isLoading: false, error: null }),
  useRollbackAccessListImport: () => ({ isPending: false, mutateAsync: vi.fn() }),
}));

const t = (key: string) => key;

describe('PolicyEditorModal', () => {
  it('renders fields derived from a policy rule inside Form.List', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn(async () => undefined);
    const source: PolicyVersion = {
      id: 'policy-version-1',
      carrier: 'cmcc',
      name: 'policy',
      version: 1,
      status: 'draft',
      policy: {
        default_action: 'reject',
        rules: [{
          id: 'rule-1',
          name: 'registered station',
          enabled: true,
          priority: 100,
          serial_scope: { type: 'list', values: ['SN-1'] },
          conditions: [{
            id: 'condition-1',
            type: 'gps',
            operator: 'within_radius',
            geo_fence: {
              center: { latitude: 39.9042, longitude: 116.4074 },
              radius_meters: 1000,
              allow_missing: true,
            },
            required: true,
            evidence_ttl: 900_000_000_000,
          }],
        }],
      },
    };

    render(<PolicyEditorModal
      operatorCode="cmcc"
      open
      source={source}
      t={t}
      onCancel={() => undefined}
      onSubmit={onSubmit}
    />);

    await waitFor(() => {
      expect(screen.getByLabelText('deviceAccess.policySetName')).toBeDisabled();
      expect(screen.getByLabelText('deviceAccess.serialList')).toHaveValue('SN-1');
      expect(screen.getByLabelText('deviceAccess.priority')).toHaveValue('100');
      expect(screen.getByLabelText('deviceAccess.latitude')).toHaveValue('39.904200');
      expect(screen.getByLabelText('deviceAccess.longitude')).toHaveValue('116.407400');
      expect(screen.getByLabelText('deviceAccess.radiusMeters')).toHaveValue('1000');
      expect(screen.getByText('deviceAccess.gpsAllowMissing').closest('label')?.querySelector('input')).toBeChecked();
    });

    await user.click(screen.getByRole('button', { name: 'deviceAccess.saveDraft' }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalled());
    expect(onSubmit.mock.calls[0][1].rules[0].conditions[0].geo_fence?.allow_missing).toBe(true);
    expect(onSubmit.mock.calls[0][1].rules[0].priority).toBe(100);
    expect(onSubmit.mock.calls[0][1].default_action).toBe('reject');
    expect(onSubmit.mock.calls[0][1].failure_mode).toBe('fail_closed');
  });

  it('normalizes legacy manual-review policy values to the closed core workflow', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn(async () => undefined);
    const source = {
      id: 'legacy-review', carrier: 'cmcc', name: 'policy', version: 2, status: 'retired',
      policy: { default_action: 'review', failure_mode: 'review_hold', rules: [] },
    } as PolicyVersion;

    render(<PolicyEditorModal operatorCode="cmcc" open source={source} t={t} onCancel={() => undefined} onSubmit={onSubmit} />);

    expect(screen.queryByRole('option', { name: 'deviceAccess.defaultAction.review' })).not.toBeInTheDocument();
    expect(screen.getByText('deviceAccess.defaultAction.reject')).toBeInTheDocument();
    expect(screen.getByText('deviceAccess.failureMode.fail_closed')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'deviceAccess.saveDraft' }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit.mock.calls[0][1].default_action).toBe('reject');
    expect(onSubmit.mock.calls[0][1].failure_mode).toBe('fail_closed');
  });

  it('locks the existing operator policy-set name for a new version', async () => {
    render(<App>
      <PolicyEditorModal
        operatorCode="cmcc"
        open
        fixedPolicyName="CMCC policy family"
        t={t}
        onCancel={() => undefined}
        onSubmit={async () => undefined}
      />
    </App>);

    await waitFor(() => {
      expect(screen.getByLabelText('deviceAccess.policySetName')).toHaveValue('CMCC policy family');
      expect(screen.getByLabelText('deviceAccess.policySetName')).toBeDisabled();
      expect(screen.getByText('deviceAccess.policySetNameFixed')).toBeInTheDocument();
    });
  });

  it('shows five-dimension governance only for a persisted draft rule', async () => {
    const source = {
      id: 'draft-1', carrier: 'cmcc', name: 'policy', version: 1, status: 'draft',
      policy: {
        default_action: 'reject',
        rules: [{
          id: 'rule-1', name: 'rule', enabled: true, priority: 100,
          serial_scope: { type: 'all' },
          conditions: [{ id: 'tac-1', type: 'tac', operator: 'equal', expected: '1', required: true, evidence_ttl: 0 }],
        }],
      },
    } as PolicyVersion;
    render(<App>
      <PolicyEditorModal
        operatorCode="cmcc"
        open
        source={source}
        t={t}
        onCancel={() => undefined}
        onSubmit={async () => undefined}
      />
    </App>);

    await waitFor(() => {
      expect(screen.getByText('deviceAccess.ruleDimensionGovernance')).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /deviceAccess.downloadTemplate/ })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /deviceAccess.importList/ })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /common.export/ })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /deviceAccess.clearRuleDimension/ })).toBeInTheDocument();
    });
  }, 15_000);

  it('opens an existing policy whose persisted rules are null', async () => {
    const source = {
      id: 'policy-version-empty',
      carrier: 'cmcc',
      name: 'empty policy',
      version: 1,
      status: 'published',
      policy: { default_action: 'reject', rules: null },
    } as unknown as PolicyVersion;

    render(<PolicyEditorModal
      operatorCode="cmcc"
      open
      source={source}
      t={t}
      onCancel={() => undefined}
      onSubmit={async () => undefined}
    />);

    await waitFor(() => {
      expect(screen.getByLabelText('deviceAccess.policySetName')).toHaveValue('empty policy');
      expect(screen.getByRole('button', { name: 'plus deviceAccess.addRule' })).toBeInTheDocument();
    });
  });

  it('keeps default TAC operator options after dynamically adding a rule and condition', async () => {
    const user = userEvent.setup();
    render(<PolicyEditorModal
      operatorCode="cmcc"
      open
      t={t}
      onCancel={() => undefined}
      onSubmit={async () => undefined}
    />);

    await user.click(screen.getByRole('button', { name: 'plus deviceAccess.addRule' }));
    await user.click(screen.getByRole('button', { name: /deviceAccess.addCondition/ }));
    await user.click(await screen.findByRole('menuitem', { name: 'deviceAccess.conditionType.tac' }));
    await user.click(screen.getByLabelText('deviceAccess.conditionOperator'));

    expect(await screen.findByRole('option', { name: 'deviceAccess.operator.equal' })).toBeInTheDocument();
    expect(screen.getByRole('option', { name: 'deviceAccess.operator.in' })).toBeInTheDocument();
  });

  it('separates source loading from draft submission loading', () => {
    const { rerender } = render(<PolicyEditorModal
      operatorCode="cmcc"
      open
      sourceLoading
      t={t}
      onCancel={() => undefined}
      onSubmit={async () => undefined}
    />);

    const saveButton = screen.getByText('deviceAccess.saveDraft').closest('button');
    expect(saveButton).not.toBeNull();
    expect(saveButton).toBeDisabled();
    expect(saveButton).not.toHaveClass('ant-btn-loading');

    rerender(<PolicyEditorModal
      operatorCode="cmcc"
      open
      submitting
      t={t}
      onCancel={() => undefined}
      onSubmit={async () => undefined}
    />);

    expect(screen.getByText('deviceAccess.saveDraft').closest('button')).toHaveClass('ant-btn-loading');
    expect(screen.getByRole('button', { name: 'common.cancel' })).toBeDisabled();
    expect(screen.queryByRole('button', { name: 'Close' })).not.toBeInTheDocument();
  });

  it('drills down from a persisted rule to matching access states', async () => {
    const user = userEvent.setup();
    const onCancel = vi.fn();
    const onDrilldownRule = vi.fn();
    const source = {
      id: '7aceae3e-dcbf-4f03-afbe-fd29631dd2bb', carrier: 'cmcc', name: 'policy', version: 1, status: 'published',
      policy: {
        default_action: 'reject',
        rules: [{
          id: '14418791-194f-4abd-a88c-87fba155780e', name: 'rule', enabled: true, priority: 100,
          serial_scope: { type: 'all' }, conditions: [],
        }],
      },
    } as PolicyVersion;
    render(<PolicyEditorModal operatorCode="cmcc" open source={source} readOnly t={t} onCancel={onCancel} onSubmit={async () => undefined} onDrilldownRule={onDrilldownRule} />);

    await user.click(await screen.findByRole('button', { name: 'deviceAccess.viewMatchingDevices' }));

    expect(onCancel).toHaveBeenCalledTimes(1);
    expect(onDrilldownRule).toHaveBeenCalledWith(source.id, source.policy.rules[0].id);
  });
});
