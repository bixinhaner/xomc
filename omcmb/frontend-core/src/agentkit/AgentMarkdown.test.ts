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
});
