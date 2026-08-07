import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

vi.mock('@/hooks/useT', () => ({
  useT: () => (key: string) => key,
}));

vi.mock('@core/hooks/api/useNotifications', () => ({
  useNotificationChannels: () => ({
    data: [
      { id: 'email-1', name: 'SMTP', channel: 'email', enabled: true, secretConfigured: true },
      { id: 'sms-1', name: 'SMS Kafka', channel: 'sms_kafka', enabled: false, secretConfigured: false },
    ],
    isLoading: false,
    refetch: vi.fn(),
  }),
  useNotificationChannelHealth: (id: string) => ({
    data: { channelConfigId: id, circuitState: 'closed', lastVerifiedAt: '2026-08-05T00:00:00Z' },
    isLoading: false,
  }),
  useVerifyNotificationChannel: () => ({ mutateAsync: vi.fn(), isPending: false, variables: undefined }),
}));

import ChannelHealth from './ChannelHealth';

describe('ChannelHealth', () => {
  it('shows only the production email channel and its verification action', () => {
    render(<ChannelHealth />);

    expect(screen.getByText('notification.channelHealth.productionGate')).toBeInTheDocument();
    const verifyButtons = screen.getAllByRole('button', { name: 'notification.channelHealth.verify' });
    expect(verifyButtons).toHaveLength(1);
    expect(verifyButtons[0]).toBeEnabled();
  });
});
