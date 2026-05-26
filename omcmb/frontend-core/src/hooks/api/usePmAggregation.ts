/**
 * PM 聚合手动触发 Hook（React Query mutation）。
 *
 * 配套 pmAggregationApi.recompute，让运维 UI 一行调用即可触发。
 */

import { useMutation } from '@tanstack/react-query';
import { pmAggregationApi, type RecomputeAggregationInput, type RecomputeAggregationResponse } from '../../services/api/pmAggregationApi';

export function useTriggerRecompute() {
  return useMutation<RecomputeAggregationResponse, Error, RecomputeAggregationInput>({
    mutationFn: pmAggregationApi.recompute,
  });
}
