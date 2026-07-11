import { describe, expect, it } from 'vitest';

import type { QueryTemplate, QueryTemplatePayload } from '@core/types/pmQuery';

import { synchronizeUpdatedTemplateState } from './templateUpdateState';

const currentPayload: QueryTemplatePayload = {
  deviceSns: ['SN-1'],
  metricPaths: ['KPI-1'],
  granularity: 'daily',
  timeRangePreset: 'last_7d',
};

function updatedTemplate(): QueryTemplate {
  return {
    id: 'template-36',
    name: 'Issue 36',
    visibility: 'private',
    creatorId: 'user-1',
    payload: { ...currentPayload, granularity: 'hourly' },
    createdAt: '2026-07-11T00:00:00Z',
    updatedAt: '2026-07-11T01:00:00Z',
  };
}

describe('applyUpdatedTemplateToActiveForm', () => {
  it('uses the updated payload when the edited template is active', () => {
    expect(synchronizeUpdatedTemplateState(
      'template-36',
      { formPayload: currentPayload, submittedPayload: currentPayload },
      updatedTemplate(),
      ['KPI-1'],
    ).formPayload.granularity)
      .toBe('hourly');
  });

  it('keeps the current payload when another template was edited', () => {
    expect(synchronizeUpdatedTemplateState(
      'another-template',
      { formPayload: currentPayload, submittedPayload: null },
      updatedTemplate(),
      ['KPI-1'],
    ).formPayload).toBe(currentPayload);
  });

  it('keeps the submitted snapshot and uses resolved metric IDs for an active legacy template', () => {
    const submittedPayload = { ...currentPayload, granularity: 'daily' as const };
    const legacy = updatedTemplate();
    legacy.payload = { ...legacy.payload, metricPaths: ['小区可用率'] };

    const result = synchronizeUpdatedTemplateState(
      legacy.id,
      { formPayload: currentPayload, submittedPayload },
      legacy,
      ['KPI-RESOLVED'],
    );

    expect(result.formPayload.metricPaths).toEqual(['KPI-RESOLVED']);
    expect(result.submittedPayload).toBe(submittedPayload);
  });
});
