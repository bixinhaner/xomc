import { describe, expect, it } from 'vitest';
import {
  buildIpsecSubmissionPlan,
  executeIpsecSubmissionPlan,
  type IpsecSubmissionPhase,
} from '../ipsecSubmission';

describe('buildIpsecSubmissionPlan', () => {
  it('enables IPSec before applying tunnel changes', () => {
    expect(buildIpsecSubmissionPlan({
      currentEnabled: false,
      targetEnabled: true,
      hasGlobalChange: true,
      hasTunnelChanges: true,
    })).toEqual({
      ok: true,
      phases: ['enable-global', 'apply-tunnels'],
    });
  });

  it('applies tunnel changes before disabling IPSec', () => {
    expect(buildIpsecSubmissionPlan({
      currentEnabled: true,
      targetEnabled: false,
      hasGlobalChange: true,
      hasTunnelChanges: true,
    })).toEqual({
      ok: true,
      phases: ['apply-tunnels', 'disable-global'],
    });
  });

  it('creates a single phase for a global-only change', () => {
    expect(buildIpsecSubmissionPlan({
      currentEnabled: false,
      targetEnabled: true,
      hasGlobalChange: true,
      hasTunnelChanges: false,
    })).toEqual({
      ok: true,
      phases: ['enable-global'],
    });
  });

  it('creates a single phase for tunnel-only changes while enabled', () => {
    expect(buildIpsecSubmissionPlan({
      currentEnabled: true,
      targetEnabled: true,
      hasGlobalChange: false,
      hasTunnelChanges: true,
    })).toEqual({
      ok: true,
      phases: ['apply-tunnels'],
    });
  });

  it('returns no phases when nothing changed', () => {
    expect(buildIpsecSubmissionPlan({
      currentEnabled: true,
      targetEnabled: true,
      hasGlobalChange: false,
      hasTunnelChanges: false,
    })).toEqual({
      ok: true,
      phases: [],
    });
  });

  it('blocks tunnel changes when IPSec remains disabled', () => {
    expect(buildIpsecSubmissionPlan({
      currentEnabled: false,
      targetEnabled: false,
      hasGlobalChange: false,
      hasTunnelChanges: true,
    })).toEqual({
      ok: false,
      reason: 'tunnel_requires_enabled_ipsec',
    });
  });
});

describe('executeIpsecSubmissionPlan', () => {
  it('stops before the next phase when a phase fails', async () => {
    const executed: IpsecSubmissionPhase[] = [];

    await expect(executeIpsecSubmissionPlan(
      ['apply-tunnels', 'disable-global'],
      async (phase) => {
        executed.push(phase);
        throw new Error('tunnel failed');
      },
    )).rejects.toThrow('tunnel failed');

    expect(executed).toEqual(['apply-tunnels']);
  });

  it('executes successful phases in the declared order', async () => {
    const executed: IpsecSubmissionPhase[] = [];

    await executeIpsecSubmissionPlan(
      ['enable-global', 'apply-tunnels'],
      async (phase) => {
        executed.push(phase);
      },
    );

    expect(executed).toEqual(['enable-global', 'apply-tunnels']);
  });
});
