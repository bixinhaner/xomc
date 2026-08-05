import { useState } from 'react';
import { Input, Radio } from 'antd';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import PolicyReadOnlySection from './PolicyReadOnlySection';

describe('PolicyReadOnlySection', () => {
  it('keeps detail fields visually enabled while module navigation remains interactive', async () => {
    const user = userEvent.setup();

    function Harness() {
      const [module, setModule] = useState('software');
      return (
        <>
          <PolicyReadOnlySection readOnly>
            <Input aria-label="policy name" value="enb" readOnly />
          </PolicyReadOnlySection>
          <Radio.Group value={module} onChange={(event) => setModule(event.target.value)}>
            <Radio.Button value="software">Software</Radio.Button>
            <Radio.Button value="license">License</Radio.Button>
          </Radio.Group>
        </>
      );
    }

    render(<Harness />);

    const policyName = screen.getByLabelText('policy name');
    expect(policyName).not.toBeDisabled();
    expect(policyName.closest('[inert]')).not.toBeNull();

    const licenseTab = screen.getByRole('radio', { name: 'License' });
    expect(licenseTab).not.toBeDisabled();
    await user.click(screen.getByText('License'));
    expect(licenseTab).toBeChecked();
  });
});
