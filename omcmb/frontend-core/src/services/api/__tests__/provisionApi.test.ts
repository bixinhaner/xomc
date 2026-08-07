import { beforeEach, describe, expect, it, vi } from 'vitest';

const { getMock, postMock, saveBlobMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
  saveBlobMock: vi.fn(),
}));

vi.mock('../../http', () => ({
  default: { get: getMock, post: postMock },
}));
vi.mock('../../../utils/saveBlob', () => ({
  filenameFromContentDisposition: vi.fn(() => 'device-config.xml'),
  saveBlob: saveBlobMock,
}));

import { mapTaskToExecuteView, provisionApi } from '../provisionApi';

beforeEach(() => {
  getMock.mockReset();
  postMock.mockReset();
  saveBlobMock.mockReset();
});

describe('provisionApi.retryPolicyTask', () => {
  it('starts a fresh execution of the original policy for the failed device', async () => {
    postMock.mockResolvedValue({ data: { items: [] } });

    await provisionApi.retryPolicyTask({
      policyId: 'policy-id',
      deviceId: 'device-id',
    });

    expect(postMock).toHaveBeenCalledWith(
      '/provisioning/policies/policy-id/execute',
      { device_ids: ['device-id'] },
    );
  });
});

describe('provisionApi.getTasks', () => {
  it('requests only plug-and-play policy tasks when policyOnly is enabled', async () => {
    getMock.mockResolvedValue({ data: { items: [], total: 0 } });

    await provisionApi.getTasks({ page: 1, pageSize: 20, policyOnly: true });

    expect(getMock).toHaveBeenCalledWith('/provisioning/tasks', {
      params: { page: 1, page_size: 20, policy_only: true },
    });
  });

  it('sends operational filters and maps task context plus status counts', async () => {
    getMock.mockResolvedValue({ data: {
      items: [{
        id: 'task-id', device_id: 'device-id', serial_number: 'SN-001',
        product_name: 'BaiBNQ', policy_name: 'NR policy', execute_type: 'manual',
        module: 'self_config', template_id: null, policy_id: 'policy-id', status: 'failed',
        current_step: 1, total_steps: 1, error_message: 'cell inactive',
        retry_count: 0, max_retries: 3, started_at: null, completed_at: null,
        created_at: '2026-08-07T00:00:00Z', updated_at: '2026-08-07T00:00:00Z',
      }],
      total: 1,
      status_counts: { completed: 7, failed: 2 },
    } });

    const result = await provisionApi.getTasks({
      page: 2, pageSize: 10, policyOnly: true, status: 'running',
      policyId: 'policy-id',
      search: 'NR policy', productName: 'BaiBNQ', module: 'self_config',
      startedAfter: '2026-08-01T00:00:00Z', startedBefore: '2026-08-07T23:59:59Z',
    });

    expect(getMock).toHaveBeenCalledWith('/provisioning/tasks', { params: {
      page: 2, page_size: 10, policy_only: true, status: 'running',
      policy_id: 'policy-id',
      search: 'NR policy', product_name: 'BaiBNQ', module: 'self_config',
      started_after: '2026-08-01T00:00:00Z', started_before: '2026-08-07T23:59:59Z',
    } });
    expect(result.statusCounts).toEqual({ completed: 7, failed: 2 });
    expect(mapTaskToExecuteView(result.items[0])).toMatchObject({
      productName: 'BaiBNQ', policyName: 'NR policy', executeType: 'manual', module: 'self_config',
    });
  });
});

describe('plug-and-play policy product classes', () => {
  it('maps multiple product classes returned by the backend', async () => {
    getMock.mockResolvedValue({ data: {
      id: 'policy-id',
      name: 'multi-product',
      enabled: true,
      product_class: 'FAP/A',
      product_classes: ['FAP/A', 'FAP/B'],
      execute_type: 'auto',
      priority: 100,
      upgrade_enabled: false,
      license_enabled: false,
      self_config_enabled: true,
      config: {},
      created_at: '2026-08-05T00:00:00Z',
      updated_at: '2026-08-05T00:00:00Z',
    } });

    const policy = await provisionApi.getPolicy('policy-id');

    expect(policy.productClasses).toEqual(['FAP/A', 'FAP/B']);
    expect(policy.productClass).toBe('FAP/A');
  });

  it('sends all selected product classes and keeps the first legacy value', async () => {
    postMock.mockResolvedValue({ data: {
      id: 'policy-id', name: 'multi-product', enabled: true,
      product_class: 'FAP/A', product_classes: ['FAP/A', 'FAP/B'],
      execute_type: 'auto', priority: 100, upgrade_enabled: false,
      license_enabled: false, self_config_enabled: true, config: {},
      created_at: '2026-08-05T00:00:00Z', updated_at: '2026-08-05T00:00:00Z',
    } });

    await provisionApi.createPolicy({
      name: 'multi-product', enabled: true,
      productClass: 'FAP/A', productClasses: ['FAP/A', 'FAP/B'],
      executeType: 'auto', priority: 100, upgradeEnabled: false,
      targetVersion: '', licenseEnabled: false, selfConfigEnabled: true, config: {},
    });

    expect(postMock).toHaveBeenCalledWith('/provisioning/policies', expect.objectContaining({
      product_class: 'FAP/A',
      product_classes: ['FAP/A', 'FAP/B'],
    }));
  });
});

describe('provisionApi.downloadXML', () => {
  it('downloads through the authenticated HTTP client and saves the XML blob', async () => {
    const blob = new Blob(['<xml />']);
    getMock.mockResolvedValue({
      data: blob,
      headers: { 'content-disposition': 'attachment; filename="device-config.xml"' },
    });

    await provisionApi.downloadXML('xml-file-id');

    expect(getMock).toHaveBeenCalledWith(
      '/provisioning/xml-files/xml-file-id/download',
      { responseType: 'blob' },
    );
    expect(saveBlobMock).toHaveBeenCalledWith(
      blob,
      'device-config.xml',
      'application/xml;charset=utf-8',
    );
  });
});

describe('mapTaskToExecuteView', () => {
  it('uses the device serial number as the execution-list identifier', () => {
    const view = mapTaskToExecuteView({
      id: 'task-id',
      deviceId: 'd02cac5a-6052-42c3-a539-5d6d80ba3085',
      serialNumber: 'SN-0001',
      templateId: null,
      policyId: 'policy-id',
      xmlFileId: null,
      deviceTaskId: null,
      status: 'failed',
      currentStep: 8,
      currentStepName: 'verify',
      totalSteps: 12,
      errorMessage: 'failed',
      retryCount: 0,
      maxRetries: 3,
      startedAt: null,
      completedAt: null,
      createdAt: '2026-08-05T00:00:00Z',
      updatedAt: '2026-08-05T00:00:00Z',
    });

    expect(view.serialNumber).toBe('SN-0001');
    expect(view.policyId).toBe('policy-id');
  });
});
