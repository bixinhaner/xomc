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
  it('filters MML task records by task origin and maps the returned origin', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [{ ...taskResponse(), task_origin: 'script' }],
        total: 1,
        page: 1,
        page_size: 20,
        total_pages: 1,
      },
    });

    const result = await mmlApi.getTasks({ page: 1, pageSize: 20, taskOrigin: 'script' });

    expect(getMock.mock.calls[0]).toEqual([
      '/mml/tasks',
      { params: { page: 1, page_size: 20, task_origin: 'script' } },
    ]);
    expect(result.items[0].taskOrigin).toBe('script');
  });

  it('maps task result request and response messages separately', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [
          {
            device_sn: 'SN001',
            device_task_id: 'dt-1',
            status: 'completed',
            success: true,
            request_method: 'SetParameterValues',
            request_cwmp_id: 'ID:intrnl.unset.id.SetParameterValues1780000000.123456',
            request_command_key: 'mml-dt-1',
            request_payload: {
              values: [
                { name: 'Device.ManagementServer.URL', value: 'http://localhost:8080/smallcell/AcsService', type: 'xsd:string' },
              ],
            },
            raw_request: '<SOAP-ENV:Envelope><cwmp:SetParameterValues/></SOAP-ENV:Envelope>',
            raw_output: '<SOAP-ENV:Envelope><cwmp:SetParameterValuesResponse/></SOAP-ENV:Envelope>',
            parsed_data: { method: 'SetParameterValuesResponse' },
          },
        ],
        total: 1,
        page: 1,
        page_size: 20,
      },
    });

    const result = await mmlApi.getTaskResults('task-1', 1, 20);

    expect(getMock.mock.calls[0]).toEqual([
      '/mml/tasks/task-1/results',
      { params: { page: 1, page_size: 20 } },
    ]);
    expect(result.items[0]).toMatchObject({
      request: {
        method: 'SetParameterValues',
        cwmpId: 'ID:intrnl.unset.id.SetParameterValues1780000000.123456',
        commandKey: 'mml-dt-1',
        rawRequest: '<SOAP-ENV:Envelope><cwmp:SetParameterValues/></SOAP-ENV:Envelope>',
        payload: {
          values: [
            { name: 'Device.ManagementServer.URL', value: 'http://localhost:8080/smallcell/AcsService', type: 'xsd:string' },
          ],
        },
      },
      result: {
        rawOutput: '<SOAP-ENV:Envelope><cwmp:SetParameterValuesResponse/></SOAP-ENV:Envelope>',
      },
    });
  });

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
    expect(config).toEqual({ headers: { 'Content-Type': 'multipart/form-data' } });
    expect(result.validationToken).toBe('token');
    expect(result.planItems[0].deviceSn).toBe('SN1');
    expect(result.issues[0]).toMatchObject({ lineNo: 1, rawLine: 'LST DEVICE_INFO;SN1' });
  });

  it('uses the replacement validation endpoint for multipart files', async () => {
    postMock.mockResolvedValue({ data: { summary: {}, plan_items: [], issues: [] } });

    await mmlApi.validateScriptReplacement('script-1', new File(['LST DEVICE_INFO;SN1'], 'replacement.txt'));

    expect(postMock.mock.calls[0][0]).toBe('/mml/scripts/script-1/import/validate');
    expect(postMock.mock.calls[0][1]).toBeInstanceOf(FormData);
    expect(postMock.mock.calls[0][2]).toEqual({ headers: { 'Content-Type': 'multipart/form-data' } });
  });

  it('normalizes a 422 import validation envelope into a typed error', async () => {
    postMock.mockRejectedValue({
      response: {
        status: 422,
        data: validationFailureEnvelope('MML_SCRIPT_VALIDATION_FAILED', 'script validation failed'),
      },
    });

    await expect(mmlApi.validateScriptImport(new File(['BAD'], 'bad.txt'))).rejects.toMatchObject({
      name: 'MMLScriptImportApiError',
      status: 422,
      code: 'MML_SCRIPT_VALIDATION_FAILED',
      message: 'script validation failed',
      validation: {
        summary: { errorCount: 1, deviceCount: 1 },
        planItems: [{ lineNo: 7, deviceSn: 'SN1' }],
        issues: [{ code: 'MML_LINE_FORMAT_INVALID', lineNo: 7, rawLine: 'BAD;SN1' }],
      },
    });
  });

  it('normalizes a 422 replacement validation envelope into a typed error', async () => {
    postMock.mockRejectedValue({
      response: {
        status: 422,
        data: validationFailureEnvelope('MML_SCRIPT_VALIDATION_FAILED', 'replacement validation failed'),
      },
    });

    await expect(
      mmlApi.validateScriptReplacement('script-1', new File(['BAD'], 'replacement.txt')),
    ).rejects.toMatchObject({
      name: 'MMLScriptImportApiError',
      status: 422,
      code: 'MML_SCRIPT_VALIDATION_FAILED',
      validation: { issues: [{ code: 'MML_LINE_FORMAT_INVALID' }] },
    });
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
      requestId: 'exec-request-1',
    });

    const [url, body] = postMock.mock.calls[0];
    expect(url).toBe('/mml/scripts/script-1/executions');
    expect(body).toMatchObject({
      task_name: '执行巡检',
      execute_type: 'scheduled',
      scheduled_at: '2026-07-10T10:00:00Z',
      confirm_warnings: true,
      request_id: 'exec-request-1',
    });
    expect(body).not.toHaveProperty('commands');
    expect(body).not.toHaveProperty('device_sns');
    expect(body).not.toHaveProperty('plan_items');
  });

  it('normalizes a 409 execution warning envelope into a typed error', async () => {
    postMock.mockRejectedValue({
      response: {
        status: 409,
        data: validationFailureEnvelope('MML_SCRIPT_EXECUTION_WARNINGS', 'warnings require confirmation'),
      },
    });

    await expect(mmlApi.createScriptExecution('script-1', { taskName: '执行' })).rejects.toMatchObject({
      name: 'MMLScriptImportApiError',
      status: 409,
      code: 'MML_SCRIPT_EXECUTION_WARNINGS',
      message: 'warnings require confirmation',
      validation: {
        planItems: [{ lineNo: 7, deviceSn: 'SN1' }],
        issues: [{ severity: 'error' }],
      },
    });
  });

  it('maps persisted nested validation_summary on listed scripts', async () => {
    getMock.mockResolvedValue({
      data: { items: [importedScriptResponse()], total: 1, page: 1, page_size: 20, total_pages: 1 },
    });

    const result = await mmlApi.getScripts({ page: 1, pageSize: 20 });

    expect(result.items[0]).toMatchObject({
      validationSummary: { validLines: 2, warningCount: 1 },
      validationIssues: [{ code: 'MML_DEVICE_OFFLINE', lineNo: 2, rawLine: 'LST DEVICE_INFO;SN1' }],
      planItems: [{ lineNo: 2, deviceSn: 'SN1' }],
    });
  });

  it('maps persisted nested validation_summary on imported script creation', async () => {
    postMock.mockResolvedValue({ data: importedScriptResponse() });

    const result = await mmlApi.createImportedScript({ validationToken: 'token', scriptName: '巡检', description: '', tags: [] });

    expect(result.validationSummary).toMatchObject({ validLines: 2, warningCount: 1 });
    expect(result.validationIssues).toEqual([expect.objectContaining({ code: 'MML_DEVICE_OFFLINE' })]);
  });

  it('maps persisted nested validation_summary on imported script replacement', async () => {
    putMock.mockResolvedValue({ data: importedScriptResponse() });

    const result = await mmlApi.replaceImportedScript('script-1', {
      validationToken: 'token', scriptName: '替换', description: '', tags: [], expectedUpdatedAt: '2026-07-10T00:00:00Z',
    });

    expect(result.validationSummary).toMatchObject({ validLines: 2, warningCount: 1 });
    expect(result.validationIssues).toEqual([expect.objectContaining({ lineNo: 2 })]);
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

function validationFailureEnvelope(code: string, message: string) {
  return {
    ret: 0,
    msg: message,
    code,
    message,
    data: {
      summary: { total_lines: 2, valid_lines: 1, device_count: 1, error_count: 1, warning_count: 0 },
      plan_items: [{ line_no: 7, device_sn: 'SN1', order: 1, command: { command_code: 'LST DEVICE_INFO' } }],
      issues: [{ code: 'MML_LINE_FORMAT_INVALID', severity: 'error', line_no: 7, raw_line: 'BAD;SN1' }],
    },
  };
}

function importedScriptResponse() {
  return {
    ...scriptResponse(),
    original_filename: '巡检.txt',
    content_sha256: 'sha256',
    validation_version: 'v1',
    validated_at: '2026-07-10T00:00:00Z',
    plan_items: [{ line_no: 2, device_sn: 'SN1', order: 1, command: { command_code: 'LST DEVICE_INFO' } }],
    validation_summary: {
      summary: { total_lines: 2, valid_lines: 2, device_count: 1, error_count: 0, warning_count: 1 },
      issues: [{ code: 'MML_DEVICE_OFFLINE', severity: 'warning', line_no: 2, raw_line: 'LST DEVICE_INFO;SN1' }],
    },
  };
}
