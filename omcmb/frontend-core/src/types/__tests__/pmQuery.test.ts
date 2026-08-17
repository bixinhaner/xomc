import { describe, expect, it } from 'vitest';

import { mapBackendQueryTemplate, templatePayloadToBackend } from '../pmQuery';

describe('KPI query template regular report mapping', () => {
  it('maps backend schedule fields to the frontend model', () => {
    const template = mapBackendQueryTemplate({
      id: 'template-1',
      name: 'Daily KPI',
      visibility: 'private',
      creator_id: 'user-1',
      payload: {
        device_sns: ['SN-1'],
        metric_paths: ['K-1'],
        regular_report: {
          enabled: true,
          send_time: '08:30',
          periods: ['daily', 'hourly'],
          email_enabled: true,
          recipients: ['ops@example.com'],
        },
      },
      created_at: '2026-08-17T00:00:00Z',
      updated_at: '2026-08-17T00:00:00Z',
    });

    expect(template.payload.regularReport).toEqual({
      enabled: true,
      sendTime: '08:30',
      periods: ['daily', 'hourly'],
      emailEnabled: true,
      recipients: ['ops@example.com'],
    });
  });

  it('serializes regular report fields with backend snake_case names', () => {
    expect(templatePayloadToBackend({
      deviceSns: ['SN-1'],
      metricPaths: ['K-1'],
      granularity: '15min',
      timeRangePreset: 'last_3h',
      regularReport: {
        enabled: true,
        sendTime: '09:15',
        periods: ['15min'],
        emailEnabled: true,
        recipients: ['ops@example.com'],
      },
    })).toMatchObject({
      regular_report: {
        enabled: true,
        send_time: '09:15',
        periods: ['15min'],
        email_enabled: true,
        recipients: ['ops@example.com'],
      },
    });
  });
});
