import { beforeEach, describe, expect, it } from 'vitest';

import { usePmPageStateStore } from '@core/store/pmPageStateStore';
import {
  clearBuiltinMetricDraft,
  clearCustomWizardDraft,
  getBuiltinMetricDraft,
  getCustomWizardDraft,
  PM_ADHOC_PAGE_KEY,
  restorePmAdhocDraftBucket,
  saveBuiltinMetricDraft,
  saveCustomWizardDraft,
  type PmAdhocCustomWizardDraft,
} from './pmAdhocDraftState';

const customDraft: PmAdhocCustomWizardDraft = {
  current: 2,
  name: '未完成自建任务',
  technology: 'lte',
  expireDays: 60,
  visibility: 'private',
  dimension: 'network',
  selectedSns: ['SN-001'],
  cellSel: {},
  metricPaths: ['K0001'],
  metricTypeFilter: 'kpi',
  windowStart: '2026-07-28T00:00:00.000Z',
  windowEnd: '2026-07-28T01:00:00.000Z',
  plannedEndAt: '2026-08-28T00:00:00.000Z',
  plannedEndTouched: true,
  drilldownTouched: false,
  originalObjectLdns: [],
};

describe('pmAdhocDraftState', () => {
  beforeEach(() => {
    usePmPageStateStore.setState({ pages: {} });
    sessionStorage.clear();
  });

  it('saves custom wizard drafts without keeping the old list state fields', () => {
    usePmPageStateStore.getState().savePageState(PM_ADHOC_PAGE_KEY, {
      view: {
        activeListArea: 'custom',
        builtinPagination: { current: 2, pageSize: 10 },
        customPagination: { current: 3, pageSize: 20 },
        detailTaskId: 'adhoc-001',
      },
    });

    saveCustomWizardDraft({ mode: 'new' }, customDraft);

    const snapshot = usePmPageStateStore.getState().getPageState(PM_ADHOC_PAGE_KEY);
    expect(snapshot?.view).not.toHaveProperty('activeListArea');
    expect(snapshot?.view).not.toHaveProperty('builtinPagination');
    expect(getCustomWizardDraft({ mode: 'new' })?.name).toBe('未完成自建任务');
  });

  it('saves and clears edit wizard and builtin metric drafts', () => {
    saveCustomWizardDraft({ mode: 'edit', taskId: 'custom-1' }, {
      ...customDraft,
      name: '编辑中的自建任务',
    });
    saveBuiltinMetricDraft({
      taskId: 'builtin-1',
      metricPaths: ['K0001', 'C0001'],
      metricTypeFilter: 'counter',
    });

    const bucket = restorePmAdhocDraftBucket(
      usePmPageStateStore.getState().getPageState(PM_ADHOC_PAGE_KEY),
    );
    expect(bucket.activeBuiltinMetricTaskId).toBe('builtin-1');
    expect(getCustomWizardDraft({ mode: 'edit', taskId: 'custom-1' })?.name).toBe('编辑中的自建任务');
    expect(getBuiltinMetricDraft('builtin-1')?.metricPaths).toEqual(['K0001', 'C0001']);

    clearCustomWizardDraft({ mode: 'edit', taskId: 'custom-1' });
    clearBuiltinMetricDraft('builtin-1');

    expect(getCustomWizardDraft({ mode: 'edit', taskId: 'custom-1' })).toBeUndefined();
    expect(getBuiltinMetricDraft('builtin-1')).toBeUndefined();
  });
});
