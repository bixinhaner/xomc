import { App } from 'antd';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import AgentFindingDrawer from './AgentFindingDrawer';

const mocks = vi.hoisted(() => ({
  markRead: vi.fn(),
  continueFinding: vi.fn(),
  dismiss: vi.fn(),
}));

vi.mock('@core/hooks/api/useAgentFindings', () => ({
  useAgentFinding: () => ({
    data: {
      id: 'finding-1', title: '任务失败主要由设备离线导致', summary: '设备在任务窗口内离线。',
      severity: 'high', confidence: 0.86, createdAt: '2026-08-27T09:00:00Z', read: false,
      resources: [{ type: 'device', id: 'device-1', label: 'SN001' }],
      facts: [{ id: 'fact-1', text: '任务在设备离线期间失败。', evidenceRefs: ['tool-1'] }],
      hypotheses: [{ id: 'hypothesis-1', text: '离线是主要原因。', confidence: 0.86 }],
      details: { failureCategory: 'device_offline' },
    },
    isPending: false,
    isError: false,
    refetch: vi.fn(),
  }),
  useMarkAgentFindingRead: () => ({ mutate: mocks.markRead, isPending: false }),
  useDismissAgentFinding: () => ({ mutateAsync: mocks.dismiss, isPending: false }),
  useContinueAgentFinding: () => ({ mutateAsync: mocks.continueFinding, isPending: false }),
}));
vi.mock('@/hooks/useT', () => ({
  useT: () => (key: string, values?: Record<string, string | number>) =>
    ({
      'agentFinding.generatedAt': `生成于 ${values?.time}`,
      'agentFinding.evidenceCount': `${values?.count} 条证据`,
      'agentFinding.hypothesisConfidence': `推测可信度 ${values?.value}%`,
    } as Record<string, string>)[key] ?? key,
}));
vi.mock('@core/utils/systemTime', () => ({ formatSystemTime: () => '2026-08-27 17:00:00' }));

describe('AgentFindingDrawer', () => {
  beforeEach(() => {
    mocks.markRead.mockReset();
    mocks.dismiss.mockReset();
    mocks.continueFinding.mockReset().mockResolvedValue({
      context: { agentFindingId: 'finding-1' },
      message: '请继续调查任务失败原因',
    });
  });

  it('展示证据层次，并把 Finding 上下文交给 AgentPanel', async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    const received = vi.fn();
    window.addEventListener('xomc:open-agent-finding', received);

    render(
      <MemoryRouter>
        <App><AgentFindingDrawer findingId="finding-1" open onOpenChange={onOpenChange} /></App>
      </MemoryRouter>
    );

    expect(await screen.findByText('任务失败主要由设备离线导致')).toBeInTheDocument();
    expect(screen.getByText('1 条证据')).toBeInTheDocument();
    expect(screen.getByText('推测可信度 86%')).toBeInTheDocument();
    expect(mocks.markRead).toHaveBeenCalledWith('finding-1');

    await user.click(screen.getByRole('button', { name: /agentFinding.continue/ }));
    expect(mocks.continueFinding).toHaveBeenCalledWith('finding-1');
    expect(received).toHaveBeenCalledOnce();
    expect((received.mock.calls[0][0] as CustomEvent).detail.context.agentFindingId).toBe('finding-1');
    expect(onOpenChange).toHaveBeenCalledWith(false);

    window.removeEventListener('xomc:open-agent-finding', received);
  });
});
