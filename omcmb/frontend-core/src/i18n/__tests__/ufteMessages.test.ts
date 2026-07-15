import { describe, expect, it } from 'vitest';

import { messages } from '../index';

describe('UFTE i18n messages', () => {
  it('uses business category wording for the task type filter', () => {
    expect(messages['zh-CN']['ufte.filter.templateName']).toBe('按照业务类别搜索');
    expect(messages['en-US']['ufte.filter.templateName']).toBe('Search by business category');
  });

  it('does not mention task type in the task keyword search placeholder', () => {
    expect(messages['zh-CN']['ufte.search.tasks']).toBe('按任务名称搜索');
    expect(messages['en-US']['ufte.search.tasks']).toBe('Search by task name');
  });
});
