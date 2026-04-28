import { describe, it, expect } from 'vitest';
import { renderHook } from '@testing-library/react';
import { IntlProvider } from 'react-intl';
import type { ReactNode } from 'react';
import { useT } from '../useT';

function wrap(messages: Record<string, string>) {
  return ({ children }: { children: ReactNode }) => (
    <IntlProvider locale="en" defaultLocale="en" messages={messages}>
      {children}
    </IntlProvider>
  );
}

describe('useT', () => {
  it('returns the function that resolves message ids via react-intl', () => {
    const { result } = renderHook(() => useT(), {
      wrapper: wrap({ greeting: 'Hello' }),
    });
    expect(result.current('greeting')).toBe('Hello');
  });

  it('substitutes ICU placeholders', () => {
    const { result } = renderHook(() => useT(), {
      wrapper: wrap({ welcome: 'Hello {name}' }),
    });
    expect(result.current('welcome', { name: 'OMC' })).toBe('Hello OMC');
  });

  it('returns empty string id unchanged', () => {
    const { result } = renderHook(() => useT(), {
      wrapper: wrap({}),
    });
    expect(result.current('')).toBe('');
  });

  it('returns id itself when message is missing', () => {
    const { result } = renderHook(() => useT(), {
      wrapper: wrap({}),
    });
    // react-intl falls back to id when no message is registered
    expect(result.current('missing.key')).toBe('missing.key');
  });
});
