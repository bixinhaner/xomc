import { App } from 'antd';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { PolicyVersion } from '@core/services/api/deviceAccessApi';
import { PolicyEditorModal } from './PolicyEditorModal';

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
      open
      source={source}
      t={t}
      onCancel={() => undefined}
      onSubmit={onSubmit}
    />);

    await waitFor(() => {
      expect(screen.getByLabelText('deviceAccess.policySetName')).toBeDisabled();
      expect(screen.getByLabelText('deviceAccess.serialList')).toHaveValue('SN-1');
      expect(screen.getByLabelText('deviceAccess.latitude')).toHaveValue('39.904200');
      expect(screen.getByLabelText('deviceAccess.longitude')).toHaveValue('116.407400');
      expect(screen.getByLabelText('deviceAccess.radiusMeters')).toHaveValue('1000');
      expect(screen.getByText('deviceAccess.gpsAllowMissing').closest('label')?.querySelector('input')).toBeChecked();
    });

    await user.click(screen.getByRole('button', { name: 'deviceAccess.saveDraft' }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalled());
    expect(onSubmit.mock.calls[0][1].rules[0].conditions[0].geo_fence?.allow_missing).toBe(true);
  });

  it('locks the existing operator policy-set name for a new version', async () => {
    render(<App>
      <PolicyEditorModal
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

  it('imports GPS planning rows from the documented CSV columns', async () => {
    const user = userEvent.setup();
    render(<App>
      <PolicyEditorModal
        open
        fixedPolicyName="CMCC policy family"
        t={t}
        onCancel={() => undefined}
        onSubmit={async () => undefined}
      />
    </App>);

    const fileInput = document.querySelector<HTMLInputElement>('input[type="file"]');
    expect(fileInput).not.toBeNull();
    await user.upload(fileInput!, new File([
      'rule_name,serial_number,latitude,longitude,radius_meters,allow_missing,enabled\n' +
      'GPS imported,SN-CSV-1,30.536879,104.065659,500,true,true\n',
    ], 'gps.csv', { type: 'text/csv' }));

    await waitFor(() => {
      expect(screen.getByLabelText('deviceAccess.ruleName')).toHaveValue('GPS imported');
      expect(screen.getByLabelText('deviceAccess.serialList')).toHaveValue('SN-CSV-1');
      expect(screen.getByLabelText('deviceAccess.latitude')).toHaveValue('30.536879');
      expect(screen.getByText('deviceAccess.gpsAllowMissing').closest('label')?.querySelector('input')).toBeChecked();
    });
  });

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
      open
      t={t}
      onCancel={() => undefined}
      onSubmit={async () => undefined}
    />);

    await user.click(screen.getByRole('button', { name: 'plus deviceAccess.addRule' }));
    await user.click(screen.getByRole('button', { name: 'plus deviceAccess.addCondition' }));
    await user.click(screen.getByLabelText('deviceAccess.conditionOperator'));

    expect(await screen.findAllByText('deviceAccess.operator.equal')).toHaveLength(2);
    expect(screen.getAllByText('deviceAccess.operator.in')).toHaveLength(1);
  });

  it('separates source loading from draft submission loading', () => {
    const { rerender } = render(<PolicyEditorModal
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
});
