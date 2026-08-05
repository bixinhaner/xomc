export type ProvisioningStepStatus = '0' | '1' | '2' | '3';

export interface ProvisioningStep {
  id: string;
  stepName: string;
  status: ProvisioningStepStatus;
  failureReason: string;
}

const ORCHESTRATION_STEPS = [
  'match_policy',
  'prepare_context',
  'generate_xml',
  'upload_file',
  'download_xml',
  'wait_transfer_complete',
  'parameter_validation',
  'parameter_configuration',
  'cell_activation',
  'wait_startup_result',
  'verify_online',
  'completed',
] as const;

export function buildProvisioningSteps(input: {
  status: string;
  currentStep: number;
  totalSteps: number;
  failureReason: string;
}): ProvisioningStep[] {
  if (input.totalSteps !== ORCHESTRATION_STEPS.length) return [];

  const currentStep = Math.min(Math.max(input.currentStep, 1), input.totalSteps);
  const taskCompleted = input.status === '0';
  const taskFailed = input.status === '1';

  return ORCHESTRATION_STEPS.map((stepName, index) => {
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
