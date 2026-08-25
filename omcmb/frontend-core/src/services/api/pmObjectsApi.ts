/**
 * T-0193 「列设备小区/PLMN」只读 API。
 *
 * 后端：GET /api/v1/pm/metrics/objects?device_sns=A,B&technology=lte
 *   → { data: { items: [{ object_ldn, cell_id, plmn }], total } }
 *   - device_sns 支持逗号分隔或重复 query（后端两种都收），空设备返空清单。
 *   - http.ts 自动 snake_case↔camelCase：响应 object_ldn→objectLdn / cell_id→cellId 自动。
 *
 * 供下钻选择器填充「该设备实际出现过的小区+PLMN」清单。
 */

import http from '../http';
import type { BackendMetricObject, MetricObject } from '../../types/pmObject';
import { parseCellId, parsePlmn } from '../../types/pmObject';

// http 响应拦截器只拆信封（透出 env.data），不做 snake→camel 键转换——
// 键转换靠各 API 显式 mapBackendXxx。故这里按后端 wire 形态读 snake_case（object_ldn / cell_id）。
interface ObjectsResponse {
  items: BackendMetricObject[] | null;
  total: number;
}

export interface MetricObjectsTimeRange {
  startTime?: string;
  endTime?: string;
}

/** 把后端一项（snake_case）规整为前端 MetricObject；cell_id/plmn 缺时从 object_ldn 兜底拆解。 */
function toMetricObject(it: BackendMetricObject): MetricObject {
  return {
    objectLdn: it.object_ldn,
    cellId: it.cell_id || parseCellId(it.object_ldn),
    plmn: it.plmn || parsePlmn(it.object_ldn),
  };
}

export const pmObjectsApi = {
  /**
   * 列出一批设备在 PM 数据里实际出现过的「小区+PLMN」清单（去重）。
   * @param deviceSns 设备 SN 数组（空 → 直接返回空清单，不发请求由 hook 控制 enabled）
   * @param technology 制式 lte/nr/gsm（可选）
   */
  async listMetricObjects(
    deviceSns: string[],
    technology?: string,
    timeRange?: MetricObjectsTimeRange,
  ): Promise<MetricObject[]> {
    if (deviceSns.length === 0) return [];
    const params: Record<string, string> = {
      // 逗号拼接（后端也收重复 query；逗号更省 URL 长度）。
      device_sns: deviceSns.join(','),
    };
    if (technology) params.technology = technology;
    if (timeRange?.startTime) params.start_time = timeRange.startTime;
    if (timeRange?.endTime) params.end_time = timeRange.endTime;
    const { data } = await http.get<ObjectsResponse>('/pm/metrics/objects', { params });
    return (data.items ?? []).map(toMetricObject);
  },
};

// Mock：按设备造 2 个小区 × 1 PLMN 的清单，便于无后端时调下钻选择器。
export const pmObjectsMock: typeof pmObjectsApi = {
  async listMetricObjects(deviceSns: string[]): Promise<MetricObject[]> {
    if (deviceSns.length === 0) return [];
    const out: MetricObject[] = [];
    deviceSns.forEach((_sn, di) => {
      [1, 2].forEach((cellIdx) => {
        const cellId = `${111000000 + di * 1000 + cellIdx}`;
        const plmn = '46068';
        out.push({
          objectLdn: `Cellid=${cellId},PLMN=${plmn}`,
          cellId,
          plmn,
        });
      });
    });
    // 去重（同 objectLdn 合并），与真实接口语义一致。
    const seen = new Set<string>();
    return out.filter((o) => (seen.has(o.objectLdn) ? false : (seen.add(o.objectLdn), true)));
  },
};
