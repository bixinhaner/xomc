import { describe, expect, it } from 'vitest';

import {
  MML_MAX_PLAN_ITEMS,
  MML_MAX_TASK_DEVICES,
  MML_PREVIEW_PAGE_SIZE,
  MML_PREVIEW_PAGE_SIZE_OPTIONS,
  paginateMmlPreview,
  validateMmlTaskScale,
} from '../mmlTaskScale';

const sns = (count: number) => Array.from({ length: count }, (_, index) => `SN-${index + 1}`);

describe('validateMmlTaskScale', () => {
  it('accepts the supported boundary and exposes the paging contract', () => {
    expect(MML_MAX_TASK_DEVICES).toBe(200);
    expect(MML_MAX_PLAN_ITEMS).toBe(2000);
    expect(MML_PREVIEW_PAGE_SIZE).toBe(20);
    expect(MML_PREVIEW_PAGE_SIZE_OPTIONS).toEqual([20, 50, 100]);
    expect(validateMmlTaskScale(sns(200), 2000)).toBeNull();
  });

  it('counts unique trimmed device SNs and rejects more than 200 devices', () => {
    expect(validateMmlTaskScale([...sns(200), ' SN-1 ', 'SN-201'], 1)).toEqual({
      kind: 'devices',
      current: 201,
      max: 200,
    });
  });

  it('rejects more than 2000 plan rows', () => {
    expect(validateMmlTaskScale(['SN-1'], 2001)).toEqual({
      kind: 'plan_items',
      current: 2001,
      max: 2000,
    });
  });
});

describe('paginateMmlPreview', () => {
  it('returns the requested stable page without truncating the source array', () => {
    const rows = Array.from({ length: 45 }, (_, index) => index + 1);

    expect(paginateMmlPreview(rows, 2, 20)).toEqual({
      items: rows.slice(20, 40),
      page: 2,
      pageCount: 3,
      start: 21,
      end: 40,
      total: 45,
    });
    expect(rows).toHaveLength(45);
  });

  it('clamps a stale page after the data set becomes smaller', () => {
    const rows = Array.from({ length: 21 }, (_, index) => index + 1);

    expect(paginateMmlPreview(rows, 9, 20)).toMatchObject({
      items: [21],
      page: 2,
      pageCount: 2,
      start: 21,
      end: 21,
      total: 21,
    });
  });
});
