import { render, screen } from '@testing-library/react';
import { IntlProvider } from 'react-intl';
import { enUS } from '@core/i18n';
import ConfirmWithNoteModal from './ConfirmWithNoteModal';

describe('ConfirmWithNoteModal', () => {
  it('renders the English note copy without missing-translation errors', () => {
    const onIntlError = vi.fn();

    render(
      <IntlProvider locale="en-US" messages={enUS} onError={onIntlError}>
        <ConfirmWithNoteModal
          open
          title="Confirm alarm"
          message="Confirm the selected alarm?"
          onConfirm={() => {}}
          onCancel={() => {}}
        />
      </IntlProvider>,
    );

    expect(screen.getByText('Processing Description (Optional)')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Enter a processing description or note')).toBeInTheDocument();
    expect(onIntlError).not.toHaveBeenCalled();
  });
});
