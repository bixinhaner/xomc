import { describe, expect, it } from 'vitest';

import { granularityLabel, RUN_HISTORY_SCROLL_X } from './index';

describe('PmAdhoc detail drawer regression (#62)', () => {
  it('renders saved task granularity with a readable label', () => {
    const intl = {
      formatMessage: ({ id }: { id: string }) =>
        ({
          'perf.adhoc.gran.hourly': '小时',
        })[id] ?? id,
    };

    expect(granularityLabel(intl, ['hourly'])).toBe('小时');
  });

  it('keeps run history columns reachable with horizontal scroll', () => {
    expect(RUN_HISTORY_SCROLL_X).toBeGreaterThanOrEqual(1250);
  });
});
