import { describe, expect, it } from 'vitest';
import { buildProvisioningSteps, normalizeProvisioningStepName } from './provisioningSteps';

describe('plug-and-play orchestration steps', () => {
  it('renders the CMCC automatic-start stages while waiting for TransferComplete', () => {
    const steps = buildProvisioningSteps({
      status: '2',
      currentStep: 6,
      currentStepName: 'wait_transfer_complete',
      totalSteps: 12,
      technology: 'nr',
      failureReason: '',
    });

    expect(steps).toHaveLength(12);
    expect(steps.map((step) => step.stepName)).toEqual([
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
    ]);
    expect(steps.map((step) => step.status)).toEqual([
      '0', '0', '0', '0', '0', '2', '3', '3', '3', '3', '3', '3',
    ]);
  });

  it('renders the reported NR stage without claiming that OMC activated the cell', () => {
    const steps = buildProvisioningSteps({
      status: '2',
      currentStep: 9,
      currentStepName: 'cell_activation',
      totalSteps: 12,
      technology: 'nr',
      failureReason: '',
    });

    expect(steps[8]).toMatchObject({ stepName: 'startup_stage_report', status: '2' });
    expect(steps[9]).toMatchObject({ stepName: 'confirm_startup_result', status: '3' });
    expect(steps[10]).toMatchObject({ stepName: 'record_startup_success', status: '3' });
    expect(steps.some((step) => step.stepName === 'cell_activation')).toBe(false);
  });

  it('shows that NR is waiting for a startup stage after XML transfer completes', () => {
    const steps = buildProvisioningSteps({
      status: '2',
      currentStep: 6,
      currentStepName: 'wait_startup_stage',
      totalSteps: 12,
      technology: 'nr',
      failureReason: '',
    });

    expect(steps[5]).toMatchObject({ stepName: 'wait_startup_stage', status: '2' });
    expect(steps.some((step) => step.stepName === 'wait_transfer_complete' && step.status === '2')).toBe(false);
  });

  it('marks the current orchestration step failed', () => {
    const steps = buildProvisioningSteps({
      status: '1',
      currentStep: 6,
      currentStepName: 'wait_transfer_complete',
      totalSteps: 12,
      technology: 'nr',
      failureReason: 'TransferComplete fault 9010',
    });

    expect(steps[5]).toMatchObject({
      stepName: 'wait_transfer_complete',
      status: '1',
      failureReason: 'TransferComplete fault 9010',
    });
  });

  it.each(['lte', 'gsm'])('renders the real %s reboot verification flow for legacy tasks', (technology) => {
    const steps = buildProvisioningSteps({
      status: '2',
      currentStep: 11,
      currentStepName: 'wait_activation_check',
      totalSteps: 12,
      technology,
      failureReason: '',
    });

    expect(steps.map((step) => step.stepName)).toEqual([
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
    ]);
    expect(steps[8]).toMatchObject({ stepName: 'wait_state_stabilization', status: '2' });
    expect(steps.some((step) => step.stepName === 'cell_activation')).toBe(false);
  });

  it('recognizes an in-flight legacy reboot task when old task data has no technology', () => {
    const steps = buildProvisioningSteps({
      status: '2',
      currentStep: 7,
      currentStepName: 'wait_device_online',
      totalSteps: 12,
      failureReason: '',
    });

    expect(steps[7]).toMatchObject({ stepName: 'wait_device_online', status: '2' });
    expect(steps.some((step) => step.stepName === 'cell_activation')).toBe(false);
  });

  it('normalizes a completed legacy verification state for the detail header', () => {
    expect(normalizeProvisioningStepName({
      currentStepName: 'activation_verified',
      technology: 'lte',
    })).toBe('completed');
  });
});
