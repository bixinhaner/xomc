import { describe, expect, it } from 'vitest';
import type { ConfigApplyBatch } from '@core/types/system';
import { isApplyBatchForCategory, isEventDeliveryBatch } from './applyStatus';

const batch: ConfigApplyBatch = {
  id: 'batch-1',
  category: 'acs_transfer',
  configVersion: 1,
  status: 'applied',
  createdAt: '',
  updatedAt: '',
  targets: [],
};

describe('isApplyBatchForCategory', () => {
  it('does not carry a previous tab apply result into another category', () => {
    expect(isApplyBatchForCategory(batch, 'acs_transfer')).toBe(true);
    expect(isApplyBatchForCategory(batch, 'storage')).toBe(false);
  });

  it('distinguishes event delivery from runtime application', () => {
    expect(isEventDeliveryBatch({
      ...batch,
      targets: [{
        target: 'acs_transfer_event_delivery',
        status: 'applied',
        attempts: 1,
        successScope: 'event_delivered',
      }],
    })).toBe(true);
  });
});
