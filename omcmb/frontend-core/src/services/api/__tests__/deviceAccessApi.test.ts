import { beforeEach, describe, expect, it, vi } from 'vitest';

const { deleteMock, getMock, postMock, putMock } = vi.hoisted(() => ({
  deleteMock: vi.fn(),
  getMock: vi.fn(),
  postMock: vi.fn(),
  putMock: vi.fn(),
}));

vi.mock('../../http', () => ({
  default: { delete: deleteMock, get: getMock, post: postMock, put: putMock },
}));

import { deviceAccessApi } from '../deviceAccessApi';

beforeEach(() => {
  deleteMock.mockReset();
  getMock.mockReset();
  postMock.mockReset();
  putMock.mockReset();
});

describe('deviceAccessApi', () => {
  it('passes an exact candidate id without moving operator scope into query data', async () => {
    getMock.mockResolvedValue({ data: { items: [], total: 0, page: 1, page_size: 20 } });

    await deviceAccessApi.listCandidates({
      operatorCode: 'cucc', page: 1, pageSize: 20, reviewStatus: 'pending',
      candidateId: '892d12c0-ec1a-4fd1-8070-902d3aaf84e9',
    });

    expect(getMock).toHaveBeenCalledWith('/device-access/candidates', {
      params: {
        page: 1,
        page_size: 20,
        serial_number: undefined,
        review_status: 'pending',
        candidate_id: '892d12c0-ec1a-4fd1-8070-902d3aaf84e9',
      },
      headers: { 'X-Operator-Code': 'cucc' },
    });
  });

  it('updates the operator-scoped business switch without accepting carrier in the body', async () => {
    putMock.mockResolvedValue({ data: { carrier: 'cmcc', enabled: true } });

    const result = await deviceAccessApi.updateRuntimeSettings('cmcc', true);

    expect(putMock).toHaveBeenCalledWith('/device-access/settings', { enabled: true }, {
      headers: { 'X-Operator-Code': 'cmcc' },
    });
    expect(result.enabled).toBe(true);
  });

  it('keeps the operator scope in the trusted header and paginates state queries', async () => {
    getMock.mockResolvedValue({ data: { items: [], total: 0, page: 2, page_size: 20 } });

    const result = await deviceAccessApi.listStates({
      operatorCode: 'cmcc', page: 2, pageSize: 20, serialNumber: 'SN-1', state: 'accepted',
    });

    expect(getMock).toHaveBeenCalledWith('/device-access/states', {
      params: { page: 2, page_size: 20, serial_number: 'SN-1', state: 'accepted' },
      headers: { 'X-Operator-Code': 'cmcc' },
    });
    expect(result).toEqual({
      items: [], total: 0, page: 2, pageSize: 20,
    });
  });

  it('uses the same audit filters for state rows and server-side summary', async () => {
    getMock
      .mockResolvedValueOnce({ data: { items: [], total: 0, page: 1, page_size: 20 } })
      .mockResolvedValueOnce({ data: { accepted: 1, rejected: 2, review_required: 3, revoked: 4, total: 10 } });
    const filters = {
      operatorCode: 'cmcc', serialNumber: 'SN-1', decision: 'reject', reasonCode: 'rule_matched',
      actionStatus: 'dead' as const, dimension: 'gps' as const,
      policyVersionId: 'policy-1', matchedRuleId: 'rule-1',
      startedAt: '2026-08-19T00:00:00Z', endedAt: '2026-08-20T00:00:00Z',
    };

    await deviceAccessApi.listStates({ ...filters, page: 1, pageSize: 20 });
    await deviceAccessApi.summarizeStates(filters);

    const rowParams = getMock.mock.calls[0][1].params;
    const summaryParams = getMock.mock.calls[1][1].params;
    expect(rowParams).toMatchObject({
      serial_number: 'SN-1', decision: 'reject', reason_code: 'rule_matched', action_status: 'dead',
      dimension: 'gps', policy_version_id: 'policy-1',
      matched_rule_id: 'rule-1', started_at: '2026-08-19T00:00:00Z', ended_at: '2026-08-20T00:00:00Z',
    });
    const { page: _page, page_size: _pageSize, ...rowFilters } = rowParams;
    expect(summaryParams).toEqual(rowFilters);
  });

  it('loads an independently paged decision detail section', async () => {
    getMock.mockResolvedValue({ data: { items: [], total: 12, page: 2, page_size: 10 } });

    await deviceAccessApi.listDecisions({
      operatorCode: 'ctcc', serialNumber: 'SN/ONE', page: 2, pageSize: 10, archiveStatus: 'archived',
    });
	 expect(getMock).toHaveBeenCalledWith('/device-access/states/SN%2FONE/decisions', {
      params: { page: 2, page_size: 10, archive_status: 'archived' }, headers: { 'X-Operator-Code': 'ctcc' },
    });
  });

  it('loads notification delivery status as an independently paged detail section', async () => {
    getMock.mockResolvedValue({ data: { items: [], total: 3, page: 1, page_size: 10 } });

    const result = await deviceAccessApi.listNotifications({
      operatorCode: 'cmcc', serialNumber: 'SN/ONE', page: 1, pageSize: 10,
    });

    expect(getMock).toHaveBeenCalledWith('/device-access/states/SN%2FONE/notifications', {
      params: { page: 1, page_size: 10 }, headers: { 'X-Operator-Code': 'cmcc' },
    });
    expect(result.total).toBe(3);
  });

	it('compares and rolls back policy versions through explicit lifecycle endpoints', async () => {
		getMock.mockResolvedValue({ data: { target_version_id: 'v2', target_version: 2, no_changes: false } });
		postMock.mockResolvedValue({ data: { id: 'v3', version: 3, status: 'published' } });

		await deviceAccessApi.getPolicyDifference('cmcc', 'v/2', 'v/1');
		const published = await deviceAccessApi.rollbackPolicy('cmcc', 'v/1');

		expect(getMock).toHaveBeenCalledWith('/device-access/policies/v%2F2/difference', {
			params: { base_version_id: 'v/1' }, headers: { 'X-Operator-Code': 'cmcc' },
		});
		expect(postMock).toHaveBeenCalledWith('/device-access/policies/v%2F1/rollback', undefined, {
			headers: { 'X-Operator-Code': 'cmcc' },
		});
		expect(published).toMatchObject({ id: 'v3', status: 'published' });
	});

  it('archives with a required reason and restores through distinct endpoints', async () => {
    postMock.mockResolvedValue({ data: undefined });

    await deviceAccessApi.archiveDecision('cmcc', 'decision/1', '重复历史');
    await deviceAccessApi.restoreDecision('cmcc', 'decision/1');

    expect(postMock).toHaveBeenNthCalledWith(1, '/device-access/decisions/decision%2F1/archive', {
      reason: '重复历史',
    }, { headers: { 'X-Operator-Code': 'cmcc' } });
    expect(postMock).toHaveBeenNthCalledWith(2, '/device-access/decisions/decision%2F1/restore', undefined, {
      headers: { 'X-Operator-Code': 'cmcc' },
    });
  });

  it('updates one list entry through the permission-registered endpoint', async () => {
    postMock.mockResolvedValue({ data: undefined });

    await deviceAccessApi.upsertEntry({
      operatorCode: 'ctcc',
      entryType: 'deny',
      serialNumber: 'SN-2',
      reason: 'blocked',
      status: 'disabled',
    });

    expect(postMock).toHaveBeenCalledWith('/device-access/access-list', {
      entry: {
        type: 'deny', identity_type: 'serial_number', identity_value: 'SN-2',
        status: 'disabled', reason: 'blocked',
      },
    }, { headers: { 'X-Operator-Code': 'ctcc' } });
  });

  it('submits multiple list entries in one server-side request', async () => {
    postMock.mockResolvedValue({ data: undefined });

    await deviceAccessApi.upsertEntries({
      operatorCode: 'cmcc', entryType: 'allow', serialNumbers: ['SN-1', 'SN-2'], reason: 'approved',
    });

    expect(postMock).toHaveBeenCalledWith('/device-access/access-list', {
      entries: [
        { type: 'allow', identity_type: 'serial_number', identity_value: 'SN-1', status: 'active', reason: 'approved' },
        { type: 'allow', identity_type: 'serial_number', identity_value: 'SN-2', status: 'active', reason: 'approved' },
      ],
    }, { headers: { 'X-Operator-Code': 'cmcc' } });
  });

  it('disables a same-type list batch through the atomic endpoint', async () => {
    postMock.mockResolvedValue({ data: undefined });

    await deviceAccessApi.disableEntries({
      operatorCode: 'cmcc', entryType: 'deny', serialNumbers: ['SN-1', 'SN-2'], reason: 'expired approval',
    });

    expect(postMock).toHaveBeenCalledWith('/device-access/access-list/batch-disable', {
      entry_type: 'deny', serial_numbers: ['SN-1', 'SN-2'], reason: 'expired approval',
    }, { headers: { 'X-Operator-Code': 'cmcc' } });
  });

  it('downloads the operator-scoped CSV template as a blob', async () => {
    const blob = new Blob(['Serial Number,List Type']);
    getMock.mockResolvedValue({ data: blob });

    await expect(deviceAccessApi.downloadAccessListTemplate('ctcc')).resolves.toBe(blob);

    expect(getMock).toHaveBeenCalledWith('/device-access/access-list/template', {
      headers: { 'X-Operator-Code': 'ctcc' }, responseType: 'blob',
    });
  });

  it('previews a CSV import with explicit mode, failure policy, and idempotency key', async () => {
    const file = new File(['Serial Number,List Type\nSN-1,deny\n'], 'deny.csv', { type: 'text/csv' });
    postMock.mockResolvedValue({ data: { batch: { id: 'batch-1' }, rows: [], entries: [], disable_count: 0 } });

    await deviceAccessApi.previewAccessListImport({
      operatorCode: 'cmcc', entryType: 'deny', mode: 'replace', failurePolicy: 'strict', file, idempotencyKey: 'key-1',
    });

    const [path, body, config] = postMock.mock.calls[0];
    expect(path).toBe('/device-access/imports/preview');
    expect(body).toBeInstanceOf(FormData);
    expect((body as FormData).get('file')).toBe(file);
    expect((body as FormData).get('import_type')).toBe('access_list');
    expect((body as FormData).get('entry_type')).toBe('deny');
    expect((body as FormData).get('mode')).toBe('replace');
    expect((body as FormData).get('failure_policy')).toBe('strict');
    expect(config).toEqual({
      headers: {
        'X-Operator-Code': 'cmcc',
        'Content-Type': 'multipart/form-data',
        'Idempotency-Key': 'key-1',
      },
    });
  });

  it('previews a rule dimension import as multipart form data', async () => {
    const file = new File(['TAC\n1001\n'], 'tac.csv', { type: 'text/csv' });
    postMock.mockResolvedValue({ data: { batch: { id: 'batch-2' }, rows: [], entries: [], disable_count: 0 } });

    await deviceAccessApi.previewRuleDimensionImport({
      operatorCode: 'cmcc',
      policyVersionId: 'policy-version-1',
      ruleId: 'rule-1',
      dimension: 'tac',
      mode: 'append',
      failurePolicy: 'valid_only',
      file,
      idempotencyKey: 'key-2',
    });

    const [path, body, config] = postMock.mock.calls[0];
    expect(path).toBe('/device-access/imports/preview');
    expect(body).toBeInstanceOf(FormData);
    expect((body as FormData).get('file')).toBe(file);
    expect((body as FormData).get('import_type')).toBe('rule_dimension');
    expect((body as FormData).get('target_policy_version_id')).toBe('policy-version-1');
    expect((body as FormData).get('target_rule_id')).toBe('rule-1');
    expect((body as FormData).get('dimension')).toBe('tac');
    expect((body as FormData).get('mode')).toBe('append');
    expect((body as FormData).get('failure_policy')).toBe('valid_only');
    expect(config).toEqual({
      headers: {
        'X-Operator-Code': 'cmcc',
        'Content-Type': 'multipart/form-data',
        'Idempotency-Key': 'key-2',
      },
    });
  });

  it('paginates the operator-scoped import history', async () => {
    getMock.mockResolvedValue({ data: { items: [], total: 0, page: 2, page_size: 25 } });

    const result = await deviceAccessApi.listAccessListImports({ operatorCode: 'cmcc', page: 2, pageSize: 25 });

    expect(getMock).toHaveBeenCalledWith('/device-access/imports', {
      params: { import_type: 'access_list', page: 2, page_size: 25 }, headers: { 'X-Operator-Code': 'cmcc' },
    });
    expect(result).toEqual({ items: [], total: 0, page: 2, pageSize: 25 });
  });

  it('sends explicit confirmation for replace-with-invalid commit', async () => {
	postMock.mockResolvedValue({ data: { id: 'batch-1', status: 'committed' } });

	await deviceAccessApi.commitAccessListImport({
	  operatorCode: 'cmcc', batchId: 'batch-1', confirmReplaceWithInvalid: true,
	});

	expect(postMock).toHaveBeenCalledWith('/device-access/imports/batch-1/commit', {
	  confirm_replace_with_invalid: true,
	}, { headers: { 'X-Operator-Code': 'cmcc' } });
  });

  it('requests reevaluation for one encoded serial number', async () => {
    postMock.mockResolvedValue({ data: undefined });

    await deviceAccessApi.reevaluateDevice('cmcc', 'SN/ONE');

    expect(postMock).toHaveBeenCalledWith(
      '/device-access/devices/SN%2FONE/reevaluate',
      undefined,
      { headers: { 'X-Operator-Code': 'cmcc' } },
    );
  });

  it('deletes only the requested draft through the policy endpoint', async () => {
    deleteMock.mockResolvedValue({ data: undefined });

    await deviceAccessApi.deletePolicyDraft('cmcc', 'draft-1');

    expect(deleteMock).toHaveBeenCalledWith('/device-access/policies/draft-1', {
      headers: { 'X-Operator-Code': 'cmcc' },
    });
  });

  it('updates only the selected draft and keeps carrier scope in the header', async () => {
    const policy = { default_action: 'reject', rules: [] } as Parameters<typeof deviceAccessApi.updatePolicyDraft>[0]['policy'];
    putMock.mockResolvedValue({ data: { id: 'draft/1', policy } });

    await deviceAccessApi.updatePolicyDraft({ operatorCode: 'cmcc', versionId: 'draft/1', policy });

    expect(putMock).toHaveBeenCalledWith('/device-access/policies/draft%2F1', { policy }, {
      headers: { 'X-Operator-Code': 'cmcc' },
    });
  });

  it('requires the operator repair reason when retrying an RF action', async () => {
    postMock.mockResolvedValue({ data: undefined });

    await deviceAccessApi.retryAction('cmcc', 'action-1', '现场链路已恢复');

    expect(postMock).toHaveBeenCalledWith('/device-access/actions/action-1/retry', {
      reason: '现场链路已恢复',
    }, { headers: { 'X-Operator-Code': 'cmcc' } });
  });

  it('loads the attempt timeline for one RF action', async () => {
    getMock.mockResolvedValue({ data: { items: [{ id: 'attempt-1', attempt_no: 1 }] } });

    const result = await deviceAccessApi.listActionAttempts('ctcc', 'action-1');

    expect(getMock).toHaveBeenCalledWith('/device-access/actions/action-1/attempts', {
      headers: { 'X-Operator-Code': 'ctcc' },
    });
    expect(result).toHaveLength(1);
  });

});
