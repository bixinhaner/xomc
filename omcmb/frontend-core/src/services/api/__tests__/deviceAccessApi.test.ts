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

});
