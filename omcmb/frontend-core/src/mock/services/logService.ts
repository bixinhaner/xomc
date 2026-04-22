import type { OperationLog, OperationType, OperationResult } from '../../types/system';
import type { PageRequest, PageResponse } from '../../types/pagination';
import { mockOperationLogs, mockSystemLogs, mockNEMessageLogs } from '../data/logs';
import type { SystemLog, NEMessageLog } from '../data/logs';
import { delay, paginate } from '../utils';

export const logService = {
  async getOperationLogs(
    params: {
      operator?: string;
      module?: string;
      operationType?: OperationType;
      result?: OperationResult;
      timeRange?: [string, string];
      keyword?: string;
    } & PageRequest
  ): Promise<PageResponse<OperationLog>> {
    await delay(100, 250);
    let filtered = [...mockOperationLogs];

    if (params.operator) filtered = filtered.filter((l) => l.operator === params.operator);
    if (params.module) filtered = filtered.filter((l) => l.module === params.module);
    if (params.operationType) filtered = filtered.filter((l) => l.operationType === params.operationType);
    if (params.result) filtered = filtered.filter((l) => l.result === params.result);
    if (params.timeRange) {
      const [start, end] = params.timeRange;
      filtered = filtered.filter((l) => l.operationTime >= start && l.operationTime <= end);
    }
    if (params.keyword) {
      const kw = params.keyword.toLowerCase();
      filtered = filtered.filter(
        (l) =>
          l.operator.includes(kw) ||
          l.content.toLowerCase().includes(kw) ||
          l.target.toLowerCase().includes(kw)
      );
    }

    filtered.sort((a, b) => b.operationTime.localeCompare(a.operationTime));
    return paginate(filtered, params.page, params.pageSize);
  },

  async getSystemLogs(
    params: { level?: SystemLog['level']; source?: string; keyword?: string } & PageRequest
  ): Promise<PageResponse<SystemLog>> {
    await delay(100, 200);
    let filtered = [...mockSystemLogs];

    if (params.level) filtered = filtered.filter((l) => l.level === params.level);
    if (params.source) filtered = filtered.filter((l) => l.source === params.source);
    if (params.keyword) {
      const kw = params.keyword.toLowerCase();
      filtered = filtered.filter(
        (l) => l.message.toLowerCase().includes(kw) || (l.details ?? '').toLowerCase().includes(kw)
      );
    }

    filtered.sort((a, b) => b.timestamp.localeCompare(a.timestamp));
    return paginate(filtered, params.page, params.pageSize);
  },

  async getNEMessageLogs(
    params: { deviceSn?: string; messageType?: string } & PageRequest
  ): Promise<PageResponse<NEMessageLog>> {
    await delay(100, 200);
    let filtered = [...mockNEMessageLogs];

    if (params.deviceSn) filtered = filtered.filter((l) => l.deviceSn === params.deviceSn);
    if (params.messageType) filtered = filtered.filter((l) => l.messageType === params.messageType);

    filtered.sort((a, b) => b.timestamp.localeCompare(a.timestamp));
    return paginate(filtered, params.page, params.pageSize);
  },

  async exportLogs(type: 'operation' | 'system' | 'ne', params: Record<string, unknown>): Promise<{ taskId: string }> {
    await delay(300, 600);
    void params;
    return { taskId: `export-${type}-${Date.now()}` };
  },
};
