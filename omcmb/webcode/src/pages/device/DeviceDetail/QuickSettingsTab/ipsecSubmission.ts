export type IpsecSubmissionPhase =
  | 'enable-global'
  | 'apply-tunnels'
  | 'disable-global';

export interface BuildIpsecSubmissionPlanInput {
  currentEnabled: boolean;
  targetEnabled: boolean;
  hasGlobalChange: boolean;
  hasTunnelChanges: boolean;
}

export type IpsecSubmissionPlan =
  | { ok: true; phases: IpsecSubmissionPhase[] }
  | { ok: false; reason: 'tunnel_requires_enabled_ipsec' };

export function buildIpsecSubmissionPlan({
  currentEnabled,
  targetEnabled,
  hasGlobalChange,
  hasTunnelChanges,
}: BuildIpsecSubmissionPlanInput): IpsecSubmissionPlan {
  if (hasTunnelChanges && !currentEnabled && !targetEnabled) {
    return { ok: false, reason: 'tunnel_requires_enabled_ipsec' };
  }

  const phases: IpsecSubmissionPhase[] = [];
  if (hasGlobalChange && targetEnabled) {
    phases.push('enable-global');
  }
  if (hasTunnelChanges) {
    phases.push('apply-tunnels');
  }
  if (hasGlobalChange && !targetEnabled) {
    phases.push('disable-global');
  }
  return { ok: true, phases };
}

export async function executeIpsecSubmissionPlan(
  phases: IpsecSubmissionPhase[],
  execute: (phase: IpsecSubmissionPhase) => Promise<void>,
): Promise<void> {
  for (const phase of phases) {
    await execute(phase);
  }
}
