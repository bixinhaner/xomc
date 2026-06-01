import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { useQueryClient } from '@tanstack/react-query';
import QueryProvider from '../QueryProvider';
import { queryClient } from '../queryClient';

function Probe() {
  const client = useQueryClient();
  const opts = client.getDefaultOptions();
  return (
    <div>
      <span data-testid="stale">{opts.queries?.staleTime}</span>
      <span data-testid="retry">{String(opts.queries?.retry)}</span>
      <span data-testid="focus">{String(opts.queries?.refetchOnWindowFocus)}</span>
      <span data-testid="mutRetry">{String(opts.mutations?.retry)}</span>
    </div>
  );
}

describe('QueryProvider', () => {
  it('exposes the shared queryClient instance', () => {
    expect(queryClient).toBeDefined();
    expect(typeof queryClient.getDefaultOptions).toBe('function');
  });

  it('provides default query options to descendants', () => {
    render(
      <QueryProvider>
        <Probe />
      </QueryProvider>,
    );

    expect(screen.getByTestId('stale').textContent).toBe('30000');
    expect(screen.getByTestId('retry').textContent).toBe('2');
    expect(screen.getByTestId('focus').textContent).toBe('false');
    expect(screen.getByTestId('mutRetry').textContent).toBe('0');
  });

  it('renders children', () => {
    render(
      <QueryProvider>
        <span data-testid="child">hi</span>
      </QueryProvider>,
    );
    expect(screen.getByTestId('child').textContent).toBe('hi');
  });
});
