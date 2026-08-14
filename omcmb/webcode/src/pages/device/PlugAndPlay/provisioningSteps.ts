export type ProvisioningStepStatus = '0' | '1' | '2' | '3';

export interface ProvisioningStep {
  id: string;
  stepName: string;
  status: ProvisioningStepStatus;
  failureReason: string;
}

const NR_ORCHESTRATION_STEPS = [
  'match_policy',
  'prepare_context',
  'generate_xml',
  'upload_file',
  'download_xml',
  'wait_transfer_complete',
  'parameter_validation',
  'parameter_configuration',
  'startup_stage_report',
  'confirm_startup_result',
  'record_startup_success',
  'completed',
] as const;

const NR_STEP_ALIASES: Record<string, string> = {
  cell_activation: 'startup_stage_report',
  wait_startup_result: 'confirm_startup_result',
  verify_online: 'record_startup_success',
};

const LEGACY_ORCHESTRATION_STEPS = [
  'match_policy',
  'prepare_context',
  'generate_xml',
  'upload_file',
  'download_xml',
  'wait_transfer_complete',
  'reboot_device',
  'wait_device_online',
  'wait_state_stabilization',
  'wait_status_sync',
  'verify_startup_result',
  'completed',
] as const;

const LEGACY_STEP_ALIASES: Record<string, string> = {
  wait_activation_check: 'wait_state_stabilization',
  wait_activation_sync: 'wait_status_sync',
  activation_verified: 'completed',
};

const LEGACY_WORKFLOW_STATES = new Set<string>([
  ...LEGACY_ORCHESTRATION_STEPS.slice(6, -1),
  ...Object.keys(LEGACY_STEP_ALIASES),
]);

function isLegacyWorkflow(technology?: string, currentStepName?: string): boolean {
  const normalizedTechnology = technology?.trim().toLowerCase();
  return normalizedTechnology === 'lte'
    || normalizedTechnology === 'gsm'
    || LEGACY_WORKFLOW_STATES.has(currentStepName ?? '');
}

export function normalizeProvisioningStepName(input: {
  currentStepName?: string;
  technology?: string;
}): string | undefined {
  const aliases = isLegacyWorkflow(input.technology, input.currentStepName)
    ? LEGACY_STEP_ALIASES
    : NR_STEP_ALIASES;
  return aliases[input.currentStepName ?? ''] ?? input.currentStepName;
}

export function buildProvisioningSteps(input: {
  status: string;
  currentStep: number;
  currentStepName?: string;
  totalSteps: number;
  technology?: string;
  failureReason: string;
}): ProvisioningStep[] {
  const legacyWorkflow = isLegacyWorkflow(input.technology, input.currentStepName);
  const steps = legacyWorkflow ? LEGACY_ORCHESTRATION_STEPS : NR_ORCHESTRATION_STEPS;
  if (input.totalSteps !== steps.length) return [];

  const displayedSteps: readonly string[] = input.currentStepName === 'wait_startup_stage' && !legacyWorkflow
    ? steps.map((stepName, index) => index + 1 === input.currentStep ? 'wait_startup_stage' : stepName)
    : steps;
  const currentStepName = normalizeProvisioningStepName(input);
  const namedStepIndex = currentStepName
    ? displayedSteps.findIndex((stepName) => stepName === currentStepName)
    : -1;
  const currentStep = namedStepIndex >= 0
    ? namedStepIndex + 1
    : Math.min(Math.max(input.currentStep, 1), input.totalSteps);
  const taskCompleted = input.status === '0';
  const taskFailed = input.status === '1';

  return displayedSteps.map((stepName, index) => {
    const position = index + 1;
    let status: ProvisioningStepStatus = '3';
    if (taskCompleted || position < currentStep) {
      status = '0';
    } else if (position === currentStep) {
      status = taskFailed ? '1' : '2';
    }
    return {
      id: String(position),
      stepName,
      status,
      failureReason: status === '1' ? input.failureReason : '',
    };
  });
}
