import { describe, it, expect } from 'vitest';
import type { PageResponse } from '@core/types/pagination';
import type { StandardParamView } from '@core/types/mmlAdmin';
import { flattenStandardParamPages } from '../standardParamPages';

function sp(id: string): StandardParamView {
  return {
    id,
    standardPath: `Device.${id}`,
    entryType: 'parameter',
    access: 'READ_WRITE',
    dataType: 'string',
    changeApplies: '',
    description: '',
  };
}

function page(ids: string[], total: number, p: number): PageResponse<StandardParamView> {
  return { items: ids.map(sp), total, page: p, pageSize: 100 };
}

describe('flattenStandardParamPages (#105 load-more 拍平 + 去重)', () => {
  it('returns [] for undefined', () => {
    expect(flattenStandardParamPages(undefined)).toEqual([]);
  });

  it('flattens pages preserving first-seen order', () => {
    const out = flattenStandardParamPages([page(['a', 'b'], 4, 1), page(['c', 'd'], 4, 2)]);
    expect(out.map((x) => x.id)).toEqual(['a', 'b', 'c', 'd']);
  });

  it('dedups by id across page boundaries (后端分页重叠/抖动时不重复)', () => {
    const out = flattenStandardParamPages([page(['a', 'b'], 3, 1), page(['b', 'c'], 3, 2)]);
    expect(out.map((x) => x.id)).toEqual(['a', 'b', 'c']);
  });
});
