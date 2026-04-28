import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { useIntl } from 'react-intl';

vi.mock('@core/store/appStore', () => ({
  useAppStore: <T,>(selector: (s: { locale: string }) => T) =>
    selector({ locale: 'en-US' }),
}));

vi.mock('@core/i18n', () => ({
  getMessages: (locale: string) => {
    if (locale === 'en-US') return { hello: 'Hi' };
    return { hello: '你好' };
  },
}));

import LocaleProvider from '../LocaleProvider';

function Probe() {
  const intl = useIntl();
  return <span data-testid="msg">{intl.formatMessage({ id: 'hello' })}</span>;
}

describe('LocaleProvider', () => {
  it('feeds messages from getMessages(locale) into IntlProvider', () => {
    render(
      <LocaleProvider>
        <Probe />
      </LocaleProvider>,
    );
    expect(screen.getByTestId('msg').textContent).toBe('Hi');
  });

  it('renders children', () => {
    render(
      <LocaleProvider>
        <span data-testid="child">hello world</span>
      </LocaleProvider>,
    );
    expect(screen.getByTestId('child').textContent).toBe('hello world');
  });
});
