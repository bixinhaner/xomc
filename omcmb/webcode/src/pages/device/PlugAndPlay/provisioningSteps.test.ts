import { describe, expect, it } from 'vitest';
import { buildProvisioningSteps } from './provisioningSteps';

describe('plug-and-play orchestration steps', () => {
  it('renders the CMCC automatic-start stages while waiting for TransferComplete', () => {
    const steps = buildProvisioningSteps({
      status: '2',
      currentStep: 6,
      totalSteps: 12,
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
      'cell_activation',
      'wait_startup_result',
      'verify_online',
      'completed',
    ]);
    expect(steps.map((step) => step.status)).toEqual([
      '0', '0', '0', '0', '0', '2', '3', '3', '3', '3', '3', '3',
    ]);
  });

  it('marks the current orchestration step failed', () => {
    const steps = buildProvisioningSteps({
      status: '1',
      currentStep: 6,
      totalSteps: 12,
      failureReason: 'TransferComplete fault 9010',
    });

    expect(steps[5]).toMatchObject({
      stepName: 'wait_transfer_complete',
      status: '1',
      failureReason: 'TransferComplete fault 9010',
    });
  });
});
