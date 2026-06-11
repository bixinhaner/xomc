import { IntlMessageFormat } from 'intl-messageformat';
import { describe, expect, it } from 'vitest';

import { messages, SUPPORTED_LOCALES } from '../index';

/**
 * Issue #114 回归守卫：i18n 文案不得含裸 '<' 触发 formatjs FORMAT_ERROR (UNCLOSED_TAG)。
 *
 * 运行态 react-intl 用 intl-messageformat 解析每条文案；当文案里出现 ASCII 字母领头的
 * 裸 '<'（如 <platform>.xml / .bak.<ts>），formatjs 会把它当成未闭合的富文本标签而抛错，
 * UI 靠降级兜底显示原串，但 console 留下 FORMAT_ERROR 噪音。
 *
 * 本测试对两套语料逐条做解析，任何一条抛错即失败，防止裸 '<' 文案再次混入。
 */
describe('i18n message catalogs parse without FORMAT_ERROR', () => {
  for (const locale of SUPPORTED_LOCALES) {
    it(`[${locale}] every message parses cleanly`, () => {
      const failures: Array<{ key: string; value: string; error: string }> = [];
      for (const [key, value] of Object.entries(messages[locale])) {
        try {
          // eslint-disable-next-line no-new -- 仅验证解析期不抛错
          new IntlMessageFormat(value, locale);
        } catch (err) {
          failures.push({
            key,
            value,
            error: err instanceof Error ? err.message : String(err),
          });
        }
      }
      expect(failures).toEqual([]);
    });
  }

  it('detects a deliberately malformed message (failure path)', () => {
    // 含裸 '<name>' 的串应抛错，证明上面的解析检查确实有效
    expect(() => new IntlMessageFormat('saved as <name>.xml', 'en-US')).toThrow();
  });

  it('accepts the rewritten placeholders used by the fix (success path)', () => {
    expect(() => new IntlMessageFormat('文件将保存为「platform」.xml', 'zh-CN')).not.toThrow();
    expect(() => new IntlMessageFormat('saved as [platform].xml', 'en-US')).not.toThrow();
  });

  it('formats filter.* messages with a label value without error', () => {
    // alarm/rules 占位文案：以前 t() 不传值导致 {label} 缺值报错，改为传 values 后应正常输出
    for (const locale of SUPPORTED_LOCALES) {
      const enter = new IntlMessageFormat(messages[locale]['filter.enterField'], locale);
      const select = new IntlMessageFormat(messages[locale]['filter.selectField'], locale);
      const enterOut = enter.format({ label: 'X' });
      const selectOut = select.format({ label: 'Y' });
      expect(String(enterOut)).toContain('X');
      expect(String(selectOut)).toContain('Y');
    }
  });
});
