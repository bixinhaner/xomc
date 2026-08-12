import type { OperationLog, OperationType, OperationResult, NorthboundAPIInvocationLog } from '../../types/system';
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
          l.operator.toLowerCase().includes(kw) ||
          l.content.toLowerCase().includes(kw) ||
          l.target.toLowerCase().includes(kw)
      );
    }

    filtered.sort((a, b) => b.operationTime.localeCompare(a.operationTime));
    return paginate(filtered, params.page, params.pageSize);
  },

  async getNorthboundAPIInvocationLogs(
    params: {
      apiKey?: string;
      name?: string;
      method?: string;
      path?: string;
      status?: string;
      createUser?: string;
      ipAddress?: string;
      timeRange?: [string, string];
    } & PageRequest
  ): Promise<PageResponse<NorthboundAPIInvocationLog>> {
    await delay(100, 200);
    let filtered: NorthboundAPIInvocationLog[] = mockOperationLogs.slice(0, 30).map((log, index) => ({
      id: `northbound-api-${index + 1}`,
      apiKey: index % 3 === 0 ? 'auth-login' : 'device-list',
      name: index % 3 === 0 ? '用户认证' : '设备列表查询',
      method: 'POST',
      path: index % 3 === 0 ? '/api/v1/northbound/v1/access/token' : '/api/v1/northbound/v1/device/query',
      requestParams: index % 3 === 0 ? '{"user":"oss","passwd":"<redacted>"}' : '{"sn":"SN0001"}',
      responseBody: log.result === 'success' ? '{"ret":1,"msg":"ok"}' : '{"ret":0,"msg":"failed"}',
      statusCode: log.result === 'success' ? 200 : 401,
      status: log.result === 'success' ? 'success' : 'failed',
      createUser: index % 2 === 0 ? 'oss' : 'northbound',
      ipAddress: log.clientIp,
      durationMs: 10 + index,
      createdAt: log.operationTime,
      updatedAt: log.operationTime,
    }));

    if (params.apiKey) filtered = filtered.filter((l) => l.apiKey === params.apiKey);
    if (params.name) filtered = filtered.filter((l) => l.name.includes(params.name || ''));
    if (params.method) filtered = filtered.filter((l) => l.method === params.method);
    if (params.path) filtered = filtered.filter((l) => l.path.includes(params.path || ''));
    if (params.status) filtered = filtered.filter((l) => l.status === params.status);
    if (params.createUser) filtered = filtered.filter((l) => l.createUser === params.createUser);
    if (params.ipAddress) filtered = filtered.filter((l) => l.ipAddress === params.ipAddress);
    if (params.timeRange) {
      const [start, end] = params.timeRange;
      filtered = filtered.filter((l) => l.createdAt >= start && l.createdAt <= end);
    }

    filtered.sort((a, b) => b.createdAt.localeCompare(a.createdAt));
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
