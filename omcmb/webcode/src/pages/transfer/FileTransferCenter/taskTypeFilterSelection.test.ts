import { describe, expect, it } from 'vitest';
import {
  resolveTaskTypeFilterValue,
  shouldShowTaskTypeFilter,
} from './taskTypeFilterSelection';

const taskType = (typeCode: string) => ({ typeCode });

describe('task type filter selection', () => {
  it('keeps the list filter empty by default even when templates exist', () => {
    expect(resolveTaskTypeFilterValue([taskType('UPGRADE'), taskType('PATCH')], undefined)).toBeUndefined();
  });

  it('keeps a valid deep-link or user-selected template filter', () => {
    expect(resolveTaskTypeFilterValue([taskType('UPGRADE'), taskType('PATCH')], 'PATCH')).toBe('PATCH');
  });

  it('clears a stale template filter when switching to a category that does not contain it', () => {
    expect(resolveTaskTypeFilterValue([taskType('CONFIG_BACKUP')], 'PATCH')).toBeUndefined();
  });

  it('shows the template filter only when the category has more than one template', () => {
    expect(shouldShowTaskTypeFilter([taskType('ONLY')])).toBe(false);
    expect(shouldShowTaskTypeFilter([taskType('ONE'), taskType('TWO')])).toBe(true);
  });
});
