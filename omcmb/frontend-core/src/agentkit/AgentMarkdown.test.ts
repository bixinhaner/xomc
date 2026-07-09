import { describe, expect, it } from 'vitest';

import { normalizeAgentMarkdown } from './AgentMarkdown';

describe('normalizeAgentMarkdown', () => {
  it('turns compact numbered Chinese sections into markdown lists', () => {
    expect(
      normalizeAgentMarkdown(
        '你当前能查询OMC里的这几类信息： 1.设备查询-按序列号搜索 2.设备详情-查看摘要 3.活动告警-按设备过滤'
      )
    ).toBe(
      '你当前能查询OMC里的这几类信息：\n\n1. 设备查询-按序列号搜索\n2. 设备详情-查看摘要\n3. 活动告警-按设备过滤'
    );
  });

  it('does not rewrite fenced code blocks', () => {
    expect(normalizeAgentMarkdown('```txt\n1.raw\n```\n\n1.正常')).toBe('```txt\n1.raw\n```\n\n1. 正常');
  });

  it('formats compact REST endpoint text as readable markdown', () => {
    expect(
      normalizeAgentMarkdown(
        '实际调用的接口：-GET/api/v1/dashboard/device-status：返回{}-GET/api/v1/devices?page=1&pageSize=5：返回total:0'
      )
    ).toBe(
      '实际调用的接口：\n\n- `GET /api/v1/dashboard/device-status`：返回{}\n- `GET /api/v1/devices?page=1&pageSize=5`：返回total:0'
    );
  });

  it('normalizes REST endpoints that are already inline code', () => {
    expect(normalizeAgentMarkdown('调用 `GET/api/v1/devices/stats` 返回 0')).toBe(
      '调用 `GET /api/v1/devices/stats` 返回 0'
    );
  });

  it('keeps bold markers intact while splitting compact result labels', () => {
    expect(normalizeAgentMarkdown('数量：**0台**调用的API：`GET/api/v1/devices`')).toBe(
      '数量：**0台**\n调用的API：`GET /api/v1/devices`'
    );
  });
});
