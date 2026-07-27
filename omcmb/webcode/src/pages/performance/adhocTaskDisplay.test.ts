import { describe, expect, it } from 'vitest';

import { displayAdhocTaskName } from './adhocTaskDisplay';

const labelForTechnology = (technology?: string | null) =>
  ({ lte: 'eNB(LTE)', nr: 'gNB(NR)', gsm: 'GSM' })[technology ?? ''] ?? (technology ? technology.toUpperCase() : '—');

describe('displayAdhocTaskName', () => {
  it('内置任务展示层用字典 label 替换名称里的制式段', () => {
    expect(displayAdhocTaskName({
      name: '内置-全网-LTE',
      isBuiltin: true,
      technology: 'lte',
    }, labelForTechnology)).toBe('内置-全网-eNB(LTE)');
  });

  it('自建任务名称保持用户原文不处理', () => {
    expect(displayAdhocTaskName({
      name: '自建-LTE验收任务',
      isBuiltin: false,
      technology: 'lte',
    }, labelForTechnology)).toBe('自建-LTE验收任务');
  });
});
