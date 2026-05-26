/**
 * PM 聚合手动触发 API。
 *
 * 封装 POST /api/v1/pm/aggregation/recompute。后端 INSERT async_jobs 入队让 worker 抢。
 * 阶段 3 D3-2 决策：暂不提供 async_jobs LIST API；调用方只显示 job_id 让用户 SQL 查状态。
 */

import { http } from '../http';

export interface RecomputeAggregationInput {
  granularity: 'hourly' | 'daily' | 'weekly' | 'monthly';
  dimension: 'device' | 'device_group';
  start: string; // RFC3339
  end: string;   // RFC3339
}

export interface RecomputeAggregationResponse {
  job_id?: string;
  id?: string;
  [k: string]: unknown;
}

export const pmAggregationApi = {
  async recompute(input: RecomputeAggregationInput): Promise<RecomputeAggregationResponse> {
    // axios 拦截器拆 envelope 后 response.data 即业务体 { job_id, job_type, start, end }
    const { data } = await http.post<RecomputeAggregationResponse>('/pm/aggregation/recompute', input);
    return data;
  },
};
