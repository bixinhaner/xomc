import type { ConfigApplyBatch } from '@core/types/system';

export function isApplyBatchForCategory(
  batch: ConfigApplyBatch | null | undefined,
  category: string,
): batch is ConfigApplyBatch {
  return Boolean(batch && batch.category === category);
}

export function isEventDeliveryBatch(batch: ConfigApplyBatch): boolean {
  return batch.targets.some((target) => target.successScope === 'event_delivered');
}
