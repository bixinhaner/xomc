import { beforeEach, describe, expect, it, vi } from 'vitest';

const { getMock, postMock, putMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
  putMock: vi.fn(),
}));

vi.mock('../../http', () => ({
  default: { get: getMock, post: postMock, put: putMock },
}));

import { mmlApi } from '../mmlApi';

beforeEach(() => {
  getMock.mockReset();
  postMock.mockReset();
  putMock.mockReset();
});

describe('mmlApi TXT script import', () => {
  it('sends multipart validate request and maps snake_case issues', async () => {
    postMock.mockResolvedValue({
      data: {
        validation_token: 'token',
        summary: { effective_lines: 1, device_count: 1, error_count: 0, warning_count: 0 },
        plan_items: [{ line_no: 1, device_sn: 'SN1', order: 1, command: { command_code: 'LST DEVICE_INFO' } }],
        issues: [{ code: 'MML_DEVICE_OFFLINE', severity: 'warning', line_no: 1, raw_line: 'LST DEVICE_INFO;SN1' }],
      },
    });

    const result = await mmlApi.validateScriptImport(new File(['LST DEVICE_INFO;SN1'], 'script.txt'));

    const [url, body, config] = postMock.mock.calls[0];
    expect(url).toBe('/mml/scripts/import/validate');
    expect(body).toBeInstanceOf(FormData);
    expect((body as FormData).get('file')).toBeInstanceOf(File);
    expect(config).toBeUndefined();
    expect(result.validationToken).toBe('token');
    expect(result.planItems[0].deviceSn).toBe('SN1');
    expect(result.issues[0]).toMatchObject({ lineNo: 1, rawLine: 'LST DEVICE_INFO;SN1' });
  });

  it('uses the replacement validation endpoint for multipart files', async () => {
    postMock.mockResolvedValue({ data: { summary: {}, plan_items: [], issues: [] } });

    await mmlApi.validateScriptReplacement('script-1', new File(['LST DEVICE_INFO;SN1'], 'replacement.txt'));

    expect(postMock.mock.calls[0][0]).toBe('/mml/scripts/script-1/import/validate');
    expect(postMock.mock.calls[0][1]).toBeInstanceOf(FormData);
    expect(postMock.mock.calls[0][2]).toBeUndefined();
  });

  it('saves imports with only the validation token and metadata', async () => {
    postMock.mockResolvedValue({ data: scriptResponse() });

    await mmlApi.createImportedScript({
      validationToken: 'token',
      scriptName: '巡检',
      description: '说明',
      tags: ['nightly'],
      requestId: 'request-1',
    });

    expect(postMock.mock.calls[0]).toEqual([
      '/mml/scripts/import',
      {
        validation_token: 'token',
        script_name: '巡检',
        description: '说明',
        tags: ['nightly'],
        request_id: 'request-1',
      },
    ]);
  });

  it('replaces imports with a validation token, metadata, and expected version only', async () => {
    putMock.mockResolvedValue({ data: scriptResponse() });

    await mmlApi.replaceImportedScript('script-1', {
      validationToken: 'token',
      scriptName: '替换后',
      description: '',
      tags: [],
      expectedUpdatedAt: '2026-07-10T00:00:00Z',
      requestId: 'request-2',
    });

    expect(putMock.mock.calls[0]).toEqual([
      '/mml/scripts/script-1/import',
      {
        validation_token: 'token',
        script_name: '替换后',
        description: '',
        tags: [],
        expected_updated_at: '2026-07-10T00:00:00Z',
        request_id: 'request-2',
      },
    ]);
  });

  it('creates script execution without browser-owned commands, devices, or plan items', async () => {
    postMock.mockResolvedValue({ data: { task: taskResponse(), validation: { summary: {}, issues: [], plan_items: [] } } });

    await mmlApi.createScriptExecution('script-1', {
      taskName: '执行巡检',
      executeType: 'scheduled',
      scheduledAt: '2026-07-10T10:00:00Z',
      offlineRetry: true,
      offlineRetryWait: 60,
      failedRetry: true,
      failedRetryCount: 2,
      failedRetryInterval: 5,
      confirmWarnings: true,
    });

    const [url, body] = postMock.mock.calls[0];
    expect(url).toBe('/mml/scripts/script-1/executions');
    expect(body).toMatchObject({
      task_name: '执行巡检',
      execute_type: 'scheduled',
      scheduled_at: '2026-07-10T10:00:00Z',
      confirm_warnings: true,
    });
    expect(body).not.toHaveProperty('commands');
    expect(body).not.toHaveProperty('device_sns');
    expect(body).not.toHaveProperty('plan_items');
  });

  it('downloads the template as a Blob and exposes the server filename', async () => {
    const blob = new Blob(['LST DEVICE_INFO;SN1\n'], { type: 'text/plain' });
    getMock.mockResolvedValue({
      data: blob,
      headers: { 'content-disposition': 'attachment; filename="MMLTemplate.txt"' },
    });

    const result = await mmlApi.downloadScriptImportTemplate();

    expect(getMock).toHaveBeenCalledWith('/mml/scripts/import/template', { responseType: 'blob' });
    expect(result).toEqual({ blob, filename: 'MMLTemplate.txt' });
  });
});

function scriptResponse() {
  return {
    id: 'script-1',
    script_name: '巡检',
    description: '',
    content: 'LST DEVICE_INFO;SN1\n',
    creator: 'admin',
    tags: [],
    created_at: '2026-07-10T00:00:00Z',
    updated_at: '2026-07-10T00:00:00Z',
  };
}

function taskResponse() {
  return {
    id: 'task-1', task_name: '执行巡检', script_id: 'script-1', device_sns: ['SN1'], commands: [],
    status: 'pending', results: [], creator: 'admin', created_at: '2026-07-10T00:00:00Z', updated_at: '2026-07-10T00:00:00Z',
    execute_type: 'scheduled', offline_retry: true, offline_retry_wait: 60, failed_retry: true,
    failed_retry_count: 2, failed_retry_interval: 5, total_devices: 1, success_count: 0, failed_count: 0,
  };
}
