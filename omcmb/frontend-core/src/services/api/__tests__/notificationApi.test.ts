import { beforeEach, describe, expect, it, vi } from 'vitest';

const { getMock, postMock, patchMock, deleteMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
  patchMock: vi.fn(),
  deleteMock: vi.fn(),
}));

vi.mock('../../http', () => ({
  default: { get: getMock, post: postMock, patch: patchMock, delete: deleteMock },
}));

import {
  alarmEmailSettingsApi,
  isNotificationRevisionConflict,
  notificationDeliveryApi,
} from '../notificationApi';

beforeEach(() => {
  getMock.mockReset();
  postMock.mockReset();
  patchMock.mockReset();
  deleteMock.mockReset();
});

describe('alarm email settings API', () => {
  it('maps only the old OMC business fields and sends If-Match', async () => {
    patchMock.mockResolvedValue({
      data: { id: 'setting-id', name: 'Major alarms', revision: 3, interval_minutes: 10 },
      headers: { etag: '"4"' },
    });

    const result = await alarmEmailSettingsApi.update('setting-id', 3, {
      name: 'Major alarms',
      enabled: true,
      alarmIdentifiers: ['11190'],
      severities: [2],
      deviceIds: [],
      deviceGroupIds: ['group-id'],
      technologies: ['lte'],
      intervalMinutes: 10,
      toleranceDurationMinutes: 0,
      recipients: ['noc@example.com'],
      includeDefaultRecipients: true,
    });

    expect(patchMock).toHaveBeenCalledWith(
      '/alarm-email-settings/setting-id',
      {
        name: 'Major alarms',
        enabled: true,
        alarm_identifiers: ['11190'],
        severities: [2],
        device_ids: [],
        device_group_ids: ['group-id'],
        technologies: ['lte'],
        interval_minutes: 10,
        tolerance_duration_minutes: 0,
        recipients: ['noc@example.com'],
        include_default_recipients: true,
      },
      { headers: { 'If-Match': '"3"' } },
    );
    expect(result.revision).toBe(4);
  });

  it('updates the dedicated default recipient list with optimistic concurrency', async () => {
    patchMock.mockResolvedValue({
      data: { recipients: ['noc@example.com'], revision: 2 },
      headers: { etag: '"2"' },
    });

    await alarmEmailSettingsApi.updateDefaults(1, ['noc@example.com']);

    expect(patchMock).toHaveBeenCalledWith(
      '/alarm-email-settings/default-recipients',
      { recipients: ['noc@example.com'] },
      { headers: { 'If-Match': '"1"' } },
    );
  });

  it('detects HTTP 412 without relying on backend text', () => {
    expect(isNotificationRevisionConflict({ response: { status: 412 } })).toBe(true);
    expect(isNotificationRevisionConflict(new Error('conflict'))).toBe(false);
  });
});

describe('notification delivery API', () => {
  it('uses the permission-filtered per-recipient endpoint and keeps masked addresses', async () => {
    getMock.mockResolvedValue({
      data: [{
        id: 'delivery-id',
        occurrence_id: 'occurrence-id',
        event_id: 'event-id',
        device_id: 'device-id',
        device_sn: 'SC-001',
        channel: 'email',
        masked_address: 'recipient-…0123abcd',
        flow_state: 'dead_letter',
        delivery_result: 'failed',
      }],
    });

    const [delivery] = await notificationDeliveryApi.list({
      occurrenceId: 'occurrence-id',
      channel: 'email',
      limit: 20,
      offset: 0,
    });

    expect(getMock).toHaveBeenCalledWith('/notification-deliveries', {
      params: {
        occurrence_id: 'occurrence-id',
        channel: 'email',
        flow_state: undefined,
        limit: 20,
        offset: 0,
      },
    });
    expect(delivery.maskedAddress).toBe('recipient-…0123abcd');
  });
});
