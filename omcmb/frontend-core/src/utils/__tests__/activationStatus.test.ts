import { describe, it, expect } from 'vitest';
import { activationStatusLabelOf, activationStatusOf } from '../activationStatus';

describe('activationStatusOf — 三皮肤共享的"激活状态"判定单一来源', () => {
  it('"1" → active', () => {
    expect(activationStatusOf('1')).toBe('active');
  });

  it('"true"（大小写不敏感）→ active', () => {
    expect(activationStatusOf('true')).toBe('active');
    expect(activationStatusOf('TRUE')).toBe('active');
    expect(activationStatusOf(' True ')).toBe('active');
  });

  it('严格 === "0" → inactive', () => {
    expect(activationStatusOf('0')).toBe('inactive');
  });

  it('空/null/undefined/"unknown" → null（UI 应渲染占位符 "-" 或 "—"）', () => {
    expect(activationStatusOf(undefined)).toBeNull();
    expect(activationStatusOf(null)).toBeNull();
    expect(activationStatusOf('')).toBeNull();
    expect(activationStatusOf('unknown')).toBeNull();
    expect(activationStatusOf(' Unknown ')).toBeNull();
  });

  it('其它非 "1" 字符串一律按未激活展示（避免详情页回显 raw 字符串与列表脱节）', () => {
    // 后端理论上不会返回这些值,但若出现也不应让它们以 raw 形态出现在 UI 上
    expect(activationStatusOf('2')).toBe('inactive');
    expect(activationStatusOf('active')).toBe('inactive');
    expect(activationStatusOf('inactive')).toBe('inactive');
    expect(activationStatusOf('foo')).toBe('inactive');
  });

  it('优先使用字典标签渲染激活状态展示值', () => {
    const details = [
      { value: '0', label: '未开通', status: true },
      { value: '1', label: '已开通', status: true },
    ];

    expect(activationStatusLabelOf('1', details, { active: '激活', inactive: '未激活' })).toBe('已开通');
    expect(activationStatusLabelOf('0', details, { active: '激活', inactive: '未激活' })).toBe('未开通');
  });

  it('字典缺项时回退到默认标签，unknown 仍返回空', () => {
    expect(
      activationStatusLabelOf('2', [{ value: '1', label: '已开通', status: true }], {
        active: '激活',
        inactive: '未激活',
      }),
    ).toBe('未激活');
    expect(activationStatusLabelOf('unknown', [], { active: '激活', inactive: '未激活' })).toBeNull();
  });

  it('英文 locale 优先使用 labelI18n，缺失英文翻译且 label 是中文时回退默认英文标签', () => {
    const details = [
      { value: '0', label: '未开通', labelI18n: { 'en-US': 'Not Activated' }, status: true },
      { value: '1', label: '已开通', status: true },
    ];

    expect(activationStatusLabelOf('1', details, { active: 'Active', inactive: 'Inactive' }, 'en-US')).toBe('Active');
    expect(activationStatusLabelOf('0', details, { active: 'Active', inactive: 'Inactive' }, 'en-US')).toBe('Not Activated');
  });
});
